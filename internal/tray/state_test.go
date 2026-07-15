package tray

import (
	"errors"
	"testing"

	"github.com/spdg/inputscout/internal/keychron"
)

func TestBuildState(t *testing.T) {
	statuses := []keychron.Status{
		{Name: "Keychron K8 HE", Connection: "USB", DeviceID: "3434:0e80", Connected: true, Keyboard: &keychron.KeyboardTelemetry{ExternalPower: true}},
		{
			Name:       "Keychron M5 8K",
			Connection: "2.4 GHz",
			DeviceID:   "3434:d048",
			Connected:  true,
			Battery:    &keychron.Battery{Percentage: 16},
		},
		{Name: "Keychron K8 HE", Connection: "2.4 GHz", DeviceID: "3434:0e80", Connected: true},
	}
	state := BuildState(statuses, nil)
	if state.Title != "InputScout — Keychron M5 8K 16%" {
		t.Fatalf("BuildState().Title = %q", state.Title)
	}
	if state.IconName != "battery-020" {
		t.Fatalf("BuildState().IconName = %q", state.IconName)
	}
	wantDescription := "Keychron M5 8K: 16%\nKeychron K8 HE: battery unavailable, USB power connected"
	if state.Description != wantDescription {
		t.Fatalf("BuildState().Description = %q, want %q", state.Description, wantDescription)
	}
	if len(state.Batteries) != 1 || state.Batteries[0].ID != "3434:d048" {
		t.Fatalf("BuildState().Batteries = %#v", state.Batteries)
	}
}

func TestBuildStateCharging(t *testing.T) {
	state := BuildState([]keychron.Status{{
		Name:      "Mouse",
		Connected: true,
		Battery:   &keychron.Battery{Percentage: 91, Charging: true},
	}}, nil)
	if state.IconName != "battery-100-charging" {
		t.Fatalf("BuildState().IconName = %q", state.IconName)
	}
	if state.Description != "Mouse: 91% (charging)" {
		t.Fatalf("BuildState().Description = %q", state.Description)
	}
}

func TestBuildStateError(t *testing.T) {
	state := BuildState(nil, errors.New("HID unavailable"))
	if state.Title != "InputScout — device error" || state.IconName != "dialog-warning" {
		t.Fatalf("BuildState() = %#v", state)
	}
}

func TestBatteryIconClampsValues(t *testing.T) {
	for _, test := range []struct {
		percentage int
		want       string
	}{
		{-1, "battery-000"},
		{0, "battery-000"},
		{1, "battery-010"},
		{10, "battery-010"},
		{11, "battery-020"},
		{101, "battery-100"},
	} {
		if got := batteryIcon(test.percentage, false); got != test.want {
			t.Errorf("batteryIcon(%d) = %q, want %q", test.percentage, got, test.want)
		}
	}
}
