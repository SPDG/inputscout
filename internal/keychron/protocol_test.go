package keychron

import "testing"

func TestParseMouseIdentity(t *testing.T) {
	report := []byte{0xb6, 0x03, 0x01, 0x34, 0x34, 0x48, 0xd0, 0x01}
	identity, err := parseMouseIdentity(report)
	if err != nil {
		t.Fatalf("parseMouseIdentity() error = %v", err)
	}
	if !identity.Connected || identity.VendorID != 0x3434 || identity.ProductID != 0xd048 {
		t.Fatalf("parseMouseIdentity() = %#v", identity)
	}
}

func TestParseKeyboardIdentity(t *testing.T) {
	report := []byte{0xb2, 0x01, 0x34, 0x34, 0x80, 0x0e, 0x01}
	identity, err := parseKeyboardIdentity(report)
	if err != nil {
		t.Fatalf("parseKeyboardIdentity() error = %v", err)
	}
	if !identity.Connected || identity.VendorID != 0x3434 || identity.ProductID != 0x0e80 {
		t.Fatalf("parseKeyboardIdentity() = %#v", identity)
	}
}

func TestParseM5Battery(t *testing.T) {
	report := []byte{
		0xb4, 0x06, 0x00, 0x02, 0x02, 0x02, 0x90, 0x01,
		0x20, 0x03, 0x40, 0x06, 0x80, 0x0c, 0x88, 0x13,
		0x15, 0x05, 0x04, 0x0a, 0x91,
	}
	battery, err := parseM5Battery(report)
	if err != nil {
		t.Fatalf("parseM5Battery() error = %v", err)
	}
	if battery.Percentage != 17 || !battery.Charging {
		t.Fatalf("parseM5Battery() = %#v", battery)
	}
}

func TestParseM5BatteryRejectsInvalidPercentage(t *testing.T) {
	report := make([]byte, 21)
	report[0], report[1], report[20] = 0xb4, 0x06, 101
	if _, err := parseM5Battery(report); err == nil {
		t.Fatal("parseM5Battery() unexpectedly accepted 101 percent")
	}
}
