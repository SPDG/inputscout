package keychron

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sstallion/go-hid"
)

const responseTimeout = time.Second

// Battery is the battery state reported by a device.
type Battery struct {
	Percentage int  `json:"percentage"`
	Charging   bool `json:"charging"`
}

// Status describes a Keychron device attached through a supported receiver.
type Status struct {
	Name       string   `json:"name"`
	Connection string   `json:"connection"`
	ReceiverID string   `json:"receiver_id"`
	DeviceID   string   `json:"device_id,omitempty"`
	Connected  bool     `json:"connected"`
	Battery    *Battery `json:"battery,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// Scan discovers supported Keychron receivers and queries their current state.
// The protocol operations are read-only; the only output reports sent are
// state queries used by Keychron Launcher itself.
func Scan(includeBattery bool) ([]Status, error) {
	if err := hid.Init(); err != nil {
		return nil, fmt.Errorf("initialize HIDAPI: %w", err)
	}
	defer hid.Exit()

	var devices []*hid.DeviceInfo
	err := hid.Enumerate(keychronVendorID, hid.ProductIDAny, func(info *hid.DeviceInfo) error {
		if isMouseConfigInterface(info) || isKeyboardConfigInterface(info) {
			copy := *info
			devices = append(devices, &copy)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("enumerate HID devices: %w", err)
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].ProductID == devices[j].ProductID {
			return devices[i].Path < devices[j].Path
		}
		return devices[i].ProductID < devices[j].ProductID
	})

	statuses := make([]Status, 0, len(devices))
	for _, info := range devices {
		switch info.ProductID {
		case mouseReceiverPID:
			statuses = append(statuses, queryMouse(info, includeBattery))
		case keyboardReceiverPID:
			statuses = append(statuses, queryKeyboard(info))
		}
	}
	return statuses, nil
}

func isMouseConfigInterface(info *hid.DeviceInfo) bool {
	return info.ProductID == mouseReceiverPID &&
		(info.InterfaceNbr == mouseConfigInterface || info.UsagePage == mouseConfigUsagePage)
}

func isKeyboardConfigInterface(info *hid.DeviceInfo) bool {
	return info.ProductID == keyboardReceiverPID &&
		(info.InterfaceNbr == keyboardConfigInterface || info.UsagePage == keyboardConfigUsage)
}

func queryMouse(info *hid.DeviceInfo, includeBattery bool) Status {
	status := Status{
		Name:       "Keychron Ultra-Link 8K receiver",
		Connection: "2.4 GHz",
		ReceiverID: usbID(info.VendorID, info.ProductID),
	}
	device, err := hid.OpenPath(info.Path)
	if err != nil {
		status.Error = accessError(err)
		return status
	}
	defer device.Close()

	report, err := exchange(device, mouseIdentityRequest(), 64, func(data []byte) bool {
		return len(data) >= 2 && data[0] == 0xb6 && data[1] == 0x03
	})
	if err != nil {
		status.Error = fmt.Sprintf("query receiver: %v", err)
		return status
	}
	identity, err := parseMouseIdentity(report)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Connected = identity.Connected
	if !identity.Connected {
		return status
	}
	status.Name = modelName(identity.VendorID, identity.ProductID)
	status.DeviceID = usbID(identity.VendorID, identity.ProductID)

	if includeBattery && identity.VendorID == 0x3434 && identity.ProductID == 0xd048 {
		report, err = exchange(device, m5BatteryRequest(), 64, func(data []byte) bool {
			return len(data) >= 2 && data[0] == 0xb4 && data[1] == 0x06
		})
		if err != nil {
			status.Error = fmt.Sprintf("query battery: %v", err)
			return status
		}
		battery, err := parseM5Battery(report)
		if err != nil {
			status.Error = err.Error()
			return status
		}
		status.Battery = &battery
	}
	return status
}

func queryKeyboard(info *hid.DeviceInfo) Status {
	status := Status{
		Name:       "Keychron Link receiver",
		Connection: "2.4 GHz",
		ReceiverID: usbID(info.VendorID, info.ProductID),
	}
	device, err := hid.OpenPath(info.Path)
	if err != nil {
		status.Error = accessError(err)
		return status
	}
	defer device.Close()

	report, err := exchange(device, keyboardIdentityRequest(), 32, func(data []byte) bool {
		return len(data) > 0 && (data[0] == 0xb2 || (data[0] == 0 && len(data) > 1 && data[1] == 0xb2))
	})
	if err != nil {
		status.Error = fmt.Sprintf("query receiver: %v", err)
		return status
	}
	identity, err := parseKeyboardIdentity(report)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Connected = identity.Connected
	if identity.Connected {
		status.Name = modelName(identity.VendorID, identity.ProductID)
		status.DeviceID = usbID(identity.VendorID, identity.ProductID)
	}
	return status
}

func exchange(device *hid.Device, request []byte, responseSize int, matches func([]byte) bool) ([]byte, error) {
	if _, err := device.Write(request); err != nil {
		return nil, fmt.Errorf("write HID report: %w", err)
	}

	deadline := time.Now().Add(responseTimeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, errors.New("timed out waiting for HID response")
		}
		buffer := make([]byte, responseSize)
		n, err := device.ReadWithTimeout(buffer, remaining)
		if err != nil {
			if errors.Is(err, hid.ErrTimeout) {
				return nil, errors.New("timed out waiting for HID response")
			}
			return nil, fmt.Errorf("read HID report: %w", err)
		}
		buffer = buffer[:n]
		if matches(buffer) {
			return buffer, nil
		}
	}
}

func accessError(err error) string {
	message := err.Error()
	if strings.Contains(strings.ToLower(message), "permission denied") {
		return "permission denied; install udev/70-inputscout.rules and reconnect the receiver"
	}
	return fmt.Sprintf("open HID interface: %v", err)
}
