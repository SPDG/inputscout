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

// KeyboardTelemetry is read-only state reported by a directly connected
// keyboard configuration interface.
type KeyboardTelemetry struct {
	FirmwareVersion string           `json:"firmware_version"`
	FirmwareBuild   string           `json:"firmware_build,omitempty"`
	ProtocolVersion int              `json:"protocol_version"`
	InstructionSet  int              `json:"instruction_set"`
	DeviceMode      string           `json:"device_mode,omitempty"`
	OSMode          string           `json:"os_mode"`
	DefaultLayer    int              `json:"default_layer"`
	Features        []string         `json:"features"`
	Analog          *AnalogTelemetry `json:"analog,omitempty"`
}

// AnalogTelemetry describes the active Hall-effect keyboard profile.
type AnalogTelemetry struct {
	ProtocolVersion     int                     `json:"protocol_version"`
	CurrentProfile      int                     `json:"current_profile"`
	ProfileCount        int                     `json:"profile_count"`
	ProfileSize         int                     `json:"profile_size"`
	OKMCSlots           int                     `json:"okmc_slots"`
	SOCDSlots           int                     `json:"socd_slots"`
	CurrentProfileState *AnalogProfileTelemetry `json:"current_profile_state,omitempty"`
}

// AnalogProfileTelemetry is the global configuration for one HE profile.
type AnalogProfileTelemetry struct {
	Mode                 string  `json:"mode"`
	ActuationMM          float64 `json:"actuation_mm"`
	PressSensitivityMM   float64 `json:"press_sensitivity_mm"`
	ReleaseSensitivityMM float64 `json:"release_sensitivity_mm"`
}

// Status describes a supported Keychron device connection.
type Status struct {
	Name       string             `json:"name"`
	Connection string             `json:"connection"`
	ReceiverID string             `json:"receiver_id,omitempty"`
	DeviceID   string             `json:"device_id,omitempty"`
	Connected  bool               `json:"connected"`
	Battery    *Battery           `json:"battery,omitempty"`
	Keyboard   *KeyboardTelemetry `json:"keyboard,omitempty"`
	Error      string             `json:"error,omitempty"`
}

// Scan discovers supported Keychron receivers and queries their current state.
// The protocol operations are read-only; the only output reports sent are
// state queries used by Keychron Launcher itself.
func Scan(includeTelemetry bool) ([]Status, error) {
	if err := hid.Init(); err != nil {
		return nil, fmt.Errorf("initialize HIDAPI: %w", err)
	}
	defer hid.Exit()

	var devices []*hid.DeviceInfo
	err := hid.Enumerate(keychronVendorID, hid.ProductIDAny, func(info *hid.DeviceInfo) error {
		if isMouseConfigInterface(info) || isKeyboardConfigInterface(info) || isWiredK8ConfigInterface(info) {
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
			statuses = append(statuses, queryMouse(info, includeTelemetry))
		case keyboardReceiverPID:
			statuses = append(statuses, queryKeyboard(info))
		case k8HEPID:
			statuses = append(statuses, queryWiredK8(info, includeTelemetry))
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

func isWiredK8ConfigInterface(info *hid.DeviceInfo) bool {
	return info.ProductID == k8HEPID &&
		(info.InterfaceNbr == wiredKeyboardConfigInterface || info.UsagePage == keyboardConfigUsage)
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

func queryWiredK8(info *hid.DeviceInfo, includeTelemetry bool) Status {
	status := Status{
		Name:       "Keychron K8 HE",
		Connection: "USB",
		DeviceID:   usbID(info.VendorID, info.ProductID),
		Connected:  true,
	}
	if !includeTelemetry {
		return status
	}

	device, err := hid.OpenPath(info.Path)
	if err != nil {
		status.Error = accessError(err)
		return status
	}
	defer device.Close()

	telemetry, err := queryWiredK8Telemetry(device)
	if err != nil {
		status.Error = fmt.Sprintf("query keyboard: %v", err)
		return status
	}
	status.Keyboard = &telemetry
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
		return "permission denied; install udev/70-inputscout.rules and reconnect the device"
	}
	return fmt.Sprintf("open HID interface: %v", err)
}
