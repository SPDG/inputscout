package keychron

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/sstallion/go-hid"
)

const (
	commandProtocolVersion = 0xa0
	commandFirmwareVersion = 0xa1
	commandFeatures        = 0xa2
	commandDefaultLayer    = 0xa3
	commandAnalogMatrix    = 0xa9
	commandFactory         = 0xab

	analogGetVersion       = 0x01
	analogGetProfileInfo   = 0x10
	analogGetGlobalProfile = 0x12
	factoryGetTransport    = 0x05
)

type wiredProtocol struct {
	Version        int
	InstructionSet int
}

type wiredFeatures struct {
	Mask  uint16
	Names []string
}

type analogProfileInfo struct {
	CurrentProfile int
	ProfileCount   int
	ProfileSize    int
	OKMCSlots      int
	SOCDSlots      int
}

type factoryTransportStatus struct {
	Mode          string
	ExternalPower bool
}

func queryWiredK8Telemetry(device *hid.Device) (KeyboardTelemetry, error) {
	protocolReport, err := wiredExchange(device, commandProtocolVersion)
	if err != nil {
		return KeyboardTelemetry{}, fmt.Errorf("protocol version: %w", err)
	}
	protocol, err := parseWiredProtocol(protocolReport)
	if err != nil {
		return KeyboardTelemetry{}, err
	}

	firmwareReport, err := wiredExchange(device, commandFirmwareVersion)
	if err != nil {
		return KeyboardTelemetry{}, fmt.Errorf("firmware version: %w", err)
	}
	firmwareVersion, firmwareBuild, err := parseWiredFirmware(firmwareReport)
	if err != nil {
		return KeyboardTelemetry{}, err
	}

	featuresReport, err := wiredExchange(device, commandFeatures)
	if err != nil {
		return KeyboardTelemetry{}, fmt.Errorf("features: %w", err)
	}
	features, err := parseWiredFeatures(featuresReport, protocol.InstructionSet)
	if err != nil {
		return KeyboardTelemetry{}, err
	}

	layerReport, err := wiredExchange(device, commandDefaultLayer)
	if err != nil {
		return KeyboardTelemetry{}, fmt.Errorf("default layer: %w", err)
	}
	layer, osMode, err := parseDefaultLayer(layerReport)
	if err != nil {
		return KeyboardTelemetry{}, err
	}

	telemetry := KeyboardTelemetry{
		FirmwareVersion: firmwareVersion,
		FirmwareBuild:   firmwareBuild,
		ProtocolVersion: protocol.Version,
		InstructionSet:  protocol.InstructionSet,
		OSMode:          osMode,
		DefaultLayer:    layer,
		Features:        features.Names,
	}

	if features.Mask&(1<<3) != 0 {
		analog, err := queryAnalogTelemetry(device)
		if err != nil {
			return KeyboardTelemetry{}, err
		}
		telemetry.Analog = &analog
	}

	transportReport, err := factoryTransportExchange(device)
	if err != nil {
		return KeyboardTelemetry{}, fmt.Errorf("device mode: %w", err)
	}
	transport, err := parseFactoryTransport(transportReport)
	if err != nil {
		return KeyboardTelemetry{}, err
	}
	telemetry.DeviceMode = transport.Mode
	telemetry.ExternalPower = transport.ExternalPower

	return telemetry, nil
}

func queryAnalogTelemetry(device *hid.Device) (AnalogTelemetry, error) {
	versionReport, err := wiredExchange(device, commandAnalogMatrix, analogGetVersion)
	if err != nil {
		return AnalogTelemetry{}, fmt.Errorf("analog protocol version: %w", err)
	}
	version, err := parseAnalogVersion(versionReport)
	if err != nil {
		return AnalogTelemetry{}, err
	}

	infoReport, err := wiredExchange(device, commandAnalogMatrix, analogGetProfileInfo)
	if err != nil {
		return AnalogTelemetry{}, fmt.Errorf("analog profile info: %w", err)
	}
	info, err := parseAnalogProfileInfo(infoReport)
	if err != nil {
		return AnalogTelemetry{}, err
	}

	// Read only the four-byte global config at profile offset zero. The firmware
	// returns profile bytes starting at response byte six.
	profileReport, err := wiredExchange(device, commandAnalogMatrix, analogGetGlobalProfile, byte(info.CurrentProfile), 0, 0, 4)
	if err != nil {
		return AnalogTelemetry{}, fmt.Errorf("analog profile: %w", err)
	}
	profile, err := parseAnalogGlobalProfile(profileReport)
	if err != nil {
		return AnalogTelemetry{}, err
	}

	return AnalogTelemetry{
		ProtocolVersion:     version,
		CurrentProfile:      info.CurrentProfile,
		ProfileCount:        info.ProfileCount,
		ProfileSize:         info.ProfileSize,
		OKMCSlots:           info.OKMCSlots,
		SOCDSlots:           info.SOCDSlots,
		CurrentProfileState: &profile,
	}, nil
}

func wiredExchange(device *hid.Device, command byte, data ...byte) ([]byte, error) {
	request := make([]byte, 33)
	request[1] = command
	copy(request[2:], data)
	return exchange(device, request, 32, func(response []byte) bool {
		if len(response) == 0 || response[0] != command {
			return false
		}
		return len(data) == 0 || len(response) > 1 && response[1] == data[0]
	})
}

func factoryTransportExchange(device *hid.Device) ([]byte, error) {
	request := make([]byte, 33)
	request[1] = commandFactory
	request[2] = factoryGetTransport
	binary.LittleEndian.PutUint16(request[31:33], uint16(factoryGetTransport))
	return exchange(device, request, 32, func(response []byte) bool {
		return len(response) >= 3 && response[0] == commandFactory && response[1] == factoryGetTransport
	})
}

func parseWiredProtocol(report []byte) (wiredProtocol, error) {
	if len(report) < 4 || report[0] != commandProtocolVersion {
		return wiredProtocol{}, invalidWiredResponse("protocol", report, 4)
	}
	return wiredProtocol{Version: int(report[1]), InstructionSet: int(report[3])}, nil
}

func parseWiredFirmware(report []byte) (string, string, error) {
	if len(report) < 2 || report[0] != commandFirmwareVersion {
		return "", "", invalidWiredResponse("firmware", report, 2)
	}
	value := report[1:]
	if index := bytesIndexByte(value, 0); index >= 0 {
		value = value[:index]
	}
	fields := strings.Fields(string(value))
	if len(fields) == 0 {
		return "", "", fmt.Errorf("empty wired firmware response")
	}
	build := ""
	if len(fields) > 1 {
		build = strings.Join(fields[1:], " ")
		if len(build) > 10 && build[10] == '-' {
			build = build[:10] + " " + build[11:]
		}
	}
	return fields[0], build, nil
}

func parseWiredFeatures(report []byte, instructionSet int) (wiredFeatures, error) {
	offset := 1
	if instructionSet == 2 || instructionSet == 4 {
		offset = 2
	}
	if len(report) < offset+2 || report[0] != commandFeatures {
		return wiredFeatures{}, invalidWiredResponse("features", report, offset+2)
	}
	mask := binary.LittleEndian.Uint16(report[offset : offset+2])
	labels := []struct {
		bit  uint
		name string
	}{
		{0, "default layer"},
		{1, "Bluetooth"},
		{2, "2.4 GHz"},
		{3, "analog matrix"},
		{4, "default layer upload"},
		{5, "debounce"},
		{6, "snap action"},
		{7, "Keychron RGB"},
	}
	names := make([]string, 0, len(labels)+1)
	knownMask := uint16(0)
	for _, label := range labels {
		knownMask |= 1 << label.bit
		if mask&(1<<label.bit) != 0 {
			names = append(names, label.name)
		}
	}
	if unknown := mask &^ knownMask; unknown != 0 {
		names = append(names, fmt.Sprintf("unknown (0x%04x)", unknown))
	}
	return wiredFeatures{Mask: mask, Names: names}, nil
}

func parseDefaultLayer(report []byte) (int, string, error) {
	if len(report) < 2 || report[0] != commandDefaultLayer {
		return 0, "", invalidWiredResponse("default layer", report, 2)
	}
	layer := int(report[1])
	mode := "unknown"
	switch layer {
	case 0, 1:
		mode = "macOS"
	case 2, 3:
		mode = "Windows"
	}
	return layer, mode, nil
}

func parseAnalogVersion(report []byte) (int, error) {
	if len(report) < 3 || report[0] != commandAnalogMatrix || report[1] != analogGetVersion {
		return 0, invalidWiredResponse("analog protocol version", report, 3)
	}
	return int(report[2]), nil
}

func parseAnalogProfileInfo(report []byte) (analogProfileInfo, error) {
	if len(report) < 8 || report[0] != commandAnalogMatrix || report[1] != analogGetProfileInfo {
		return analogProfileInfo{}, invalidWiredResponse("analog profile info", report, 8)
	}
	return analogProfileInfo{
		CurrentProfile: int(report[2]),
		ProfileCount:   int(report[3]),
		ProfileSize:    int(binary.LittleEndian.Uint16(report[4:6])),
		OKMCSlots:      int(report[6]),
		SOCDSlots:      int(report[7]),
	}, nil
}

func parseAnalogGlobalProfile(report []byte) (AnalogProfileTelemetry, error) {
	if len(report) < 10 || report[0] != commandAnalogMatrix || report[1] != analogGetGlobalProfile {
		return AnalogProfileTelemetry{}, invalidWiredResponse("analog profile", report, 10)
	}
	mode := "unknown"
	switch report[6] & 0x03 {
	case 1:
		mode = "regular"
	case 2:
		mode = "rapid trigger"
	}
	sensitivity := binary.LittleEndian.Uint16(report[7:9])
	return AnalogProfileTelemetry{
		Mode:                 mode,
		ActuationMM:          float64(report[6]>>2) / 10,
		PressSensitivityMM:   float64(sensitivity&0x3f) / 10,
		ReleaseSensitivityMM: float64((sensitivity>>6)&0x3f) / 10,
	}, nil
}

func parseFactoryTransport(report []byte) (factoryTransportStatus, error) {
	if len(report) < 32 || report[0] != commandFactory || report[1] != factoryGetTransport {
		return factoryTransportStatus{}, invalidWiredResponse("device mode", report, 32)
	}
	want := binary.LittleEndian.Uint16(report[30:32])
	var got uint16
	for _, value := range report[1:29] {
		got += uint16(value)
	}
	if got != want {
		return factoryTransportStatus{}, fmt.Errorf("invalid device mode checksum: got 0x%04x, want 0x%04x", want, got)
	}
	status := factoryTransportStatus{
		// K8 HE defines USB_POWER_CONNECTED_LEVEL as active-low. This proves
		// external power presence, not that the cell is still charging.
		ExternalPower: report[3] == 0,
	}
	switch report[2] {
	case 1:
		status.Mode = "USB"
	case 2:
		status.Mode = "Bluetooth"
	case 4:
		status.Mode = "2.4 GHz"
	default:
		status.Mode = fmt.Sprintf("unknown (%d)", report[2])
	}
	return status, nil
}

func invalidWiredResponse(name string, report []byte, minimum int) error {
	if len(report) < minimum {
		return fmt.Errorf("short wired %s response: got %d bytes", name, len(report))
	}
	return fmt.Errorf("unexpected wired %s response header: %02x", name, report[0])
}

func bytesIndexByte(value []byte, target byte) int {
	for index, current := range value {
		if current == target {
			return index
		}
	}
	return -1
}
