package keychron

import (
	"encoding/binary"
	"fmt"
)

const (
	keychronVendorID             = 0x3434
	mouseReceiverPID             = 0xd028
	keyboardReceiverPID          = 0xd030
	k8HEPID                      = 0x0e80
	mouseConfigUsagePage         = 0xffc1
	keyboardConfigUsage          = 0xff60
	mouseConfigInterface         = 4
	keyboardConfigInterface      = 3
	wiredKeyboardConfigInterface = 1
)

type deviceIdentity struct {
	VendorID  uint16
	ProductID uint16
	Connected bool
}

func mouseIdentityRequest() []byte {
	report := make([]byte, 21)
	report[0] = 0xb5
	report[1] = 0x03
	return report
}

func parseMouseIdentity(report []byte) (deviceIdentity, error) {
	if len(report) < 7 {
		return deviceIdentity{}, fmt.Errorf("short mouse receiver response: got %d bytes", len(report))
	}
	if report[0] != 0xb6 || report[1] != 0x03 {
		return deviceIdentity{}, fmt.Errorf("unexpected mouse receiver response header: %02x %02x", report[0], report[1])
	}
	return deviceIdentity{
		Connected: report[2] != 0,
		VendorID:  binary.LittleEndian.Uint16(report[3:5]),
		ProductID: binary.LittleEndian.Uint16(report[5:7]),
	}, nil
}

func keyboardIdentityRequest() []byte {
	// HIDAPI requires a leading zero report ID for unnumbered reports.
	report := make([]byte, 33)
	report[1] = 0xb2
	return report
}

func parseKeyboardIdentity(report []byte) (deviceIdentity, error) {
	if len(report) > 0 && report[0] == 0x00 {
		report = report[1:]
	}
	if len(report) < 6 {
		return deviceIdentity{}, fmt.Errorf("short keyboard receiver response: got %d bytes", len(report))
	}
	if report[0] != 0xb2 {
		return deviceIdentity{}, fmt.Errorf("unexpected keyboard receiver response header: %02x", report[0])
	}
	return deviceIdentity{
		Connected: report[1] != 0,
		VendorID:  binary.LittleEndian.Uint16(report[2:4]),
		ProductID: binary.LittleEndian.Uint16(report[4:6]),
	}, nil
}

func m5BatteryRequest() []byte {
	report := make([]byte, 64)
	report[0] = 0xb3
	report[1] = 0x06
	return report
}

func parseM5Battery(report []byte) (Battery, error) {
	if len(report) < 21 {
		return Battery{}, fmt.Errorf("short mouse status response: got %d bytes", len(report))
	}
	if report[0] != 0xb4 || report[1] != 0x06 {
		return Battery{}, fmt.Errorf("unexpected mouse status response header: %02x %02x", report[0], report[1])
	}
	value := report[20]
	percentage := int(value & 0x7f)
	if percentage > 100 {
		return Battery{}, fmt.Errorf("invalid battery percentage: %d", percentage)
	}
	return Battery{
		Percentage: percentage,
		Charging:   value&0x80 != 0,
	}, nil
}

func modelName(vendorID, productID uint16) string {
	switch {
	case vendorID == 0x3434 && productID == 0xd048:
		return "Keychron M5 8K"
	case vendorID == 0x3434 && productID == 0x0e80:
		return "Keychron K8 HE"
	default:
		return fmt.Sprintf("Keychron device %04x:%04x", vendorID, productID)
	}
}

func usbID(vendorID, productID uint16) string {
	return fmt.Sprintf("%04x:%04x", vendorID, productID)
}
