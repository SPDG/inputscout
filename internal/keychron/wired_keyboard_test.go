package keychron

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestParseWiredProtocol(t *testing.T) {
	protocol, err := parseWiredProtocol([]byte{0xa0, 0x02, 0x00, 0x02})
	if err != nil {
		t.Fatalf("parseWiredProtocol() error = %v", err)
	}
	if protocol.Version != 2 || protocol.InstructionSet != 2 {
		t.Fatalf("parseWiredProtocol() = %#v", protocol)
	}
}

func TestParseWiredFirmware(t *testing.T) {
	report := make([]byte, 32)
	report[0] = 0xa1
	copy(report[1:], "v1.1.1 2025-06-17-10:42:45")
	version, build, err := parseWiredFirmware(report)
	if err != nil {
		t.Fatalf("parseWiredFirmware() error = %v", err)
	}
	if version != "v1.1.1" || build != "2025-06-17 10:42:45" {
		t.Fatalf("parseWiredFirmware() = %q, %q", version, build)
	}
}

func TestParseWiredFeatures(t *testing.T) {
	features, err := parseWiredFeatures([]byte{0xa2, 0x00, 0x8f, 0x00}, 2)
	if err != nil {
		t.Fatalf("parseWiredFeatures() error = %v", err)
	}
	wantNames := []string{"default layer", "Bluetooth", "2.4 GHz", "analog matrix", "Keychron RGB"}
	if features.Mask != 0x008f || !reflect.DeepEqual(features.Names, wantNames) {
		t.Fatalf("parseWiredFeatures() = %#v", features)
	}
}

func TestParseDefaultLayer(t *testing.T) {
	layer, mode, err := parseDefaultLayer([]byte{0xa3, 0x02})
	if err != nil {
		t.Fatalf("parseDefaultLayer() error = %v", err)
	}
	if layer != 2 || mode != "Windows" {
		t.Fatalf("parseDefaultLayer() = %d, %q", layer, mode)
	}
}

func TestParseAnalogTelemetry(t *testing.T) {
	version, err := parseAnalogVersion([]byte{0xa9, 0x01, 0x04})
	if err != nil || version != 4 {
		t.Fatalf("parseAnalogVersion() = %d, %v", version, err)
	}

	info, err := parseAnalogProfileInfo([]byte{0xa9, 0x10, 0x00, 0x03, 0x5c, 0x03, 0x14, 0x14})
	if err != nil {
		t.Fatalf("parseAnalogProfileInfo() error = %v", err)
	}
	wantInfo := analogProfileInfo{CurrentProfile: 0, ProfileCount: 3, ProfileSize: 860, OKMCSlots: 20, SOCDSlots: 20}
	if info != wantInfo {
		t.Fatalf("parseAnalogProfileInfo() = %#v, want %#v", info, wantInfo)
	}

	profile, err := parseAnalogGlobalProfile([]byte{0xa9, 0x12, 0x00, 0x00, 0x00, 0x00, 0x51, 0x04, 0x01, 0x00})
	if err != nil {
		t.Fatalf("parseAnalogGlobalProfile() error = %v", err)
	}
	wantProfile := AnalogProfileTelemetry{
		Mode:                 "regular",
		ActuationMM:          2.0,
		PressSensitivityMM:   0.4,
		ReleaseSensitivityMM: 0.4,
	}
	if profile != wantProfile {
		t.Fatalf("parseAnalogGlobalProfile() = %#v, want %#v", profile, wantProfile)
	}
}

func TestParseFactoryTransport(t *testing.T) {
	report := make([]byte, 32)
	report[0], report[1], report[2] = 0xab, 0x05, 0x04
	binary.LittleEndian.PutUint16(report[30:32], 0x0009)
	status, err := parseFactoryTransport(report)
	if err != nil {
		t.Fatalf("parseFactoryTransport() error = %v", err)
	}
	if status.Mode != "2.4 GHz" || !status.ExternalPower {
		t.Fatalf("parseFactoryTransport() = %#v", status)
	}
}

func TestParseFactoryTransportRejectsInvalidChecksum(t *testing.T) {
	report := make([]byte, 32)
	report[0], report[1], report[2] = 0xab, 0x05, 0x04
	if _, err := parseFactoryTransport(report); err == nil {
		t.Fatal("parseFactoryTransport() unexpectedly accepted an invalid checksum")
	}
}

func TestParseFactoryTransportWithoutExternalPower(t *testing.T) {
	report := make([]byte, 32)
	report[0], report[1], report[2], report[3] = 0xab, 0x05, 0x02, 0x01
	binary.LittleEndian.PutUint16(report[30:32], 0x0008)
	status, err := parseFactoryTransport(report)
	if err != nil {
		t.Fatalf("parseFactoryTransport() error = %v", err)
	}
	if status.Mode != "Bluetooth" || status.ExternalPower {
		t.Fatalf("parseFactoryTransport() = %#v", status)
	}
}

func TestWiredParsersRejectShortReports(t *testing.T) {
	if _, err := parseWiredProtocol([]byte{0xa0}); err == nil {
		t.Fatal("parseWiredProtocol() unexpectedly accepted a short report")
	}
	if _, _, err := parseWiredFirmware([]byte{0xa1}); err == nil {
		t.Fatal("parseWiredFirmware() unexpectedly accepted a short report")
	}
	if _, err := parseAnalogProfileInfo([]byte{0xa9, 0x10}); err == nil {
		t.Fatal("parseAnalogProfileInfo() unexpectedly accepted a short report")
	}
}
