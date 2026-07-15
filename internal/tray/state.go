package tray

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spdg/inputscout/internal/keychron"
)

// BatteryDevice is a battery exposed by a supported input device.
type BatteryDevice struct {
	ID         string
	Name       string
	Percentage int
	Charging   bool
}

// State is the desktop-facing summary shown by the tray item.
type State struct {
	Title       string
	Description string
	IconName    string
	Batteries   []BatteryDevice
}

// BuildState converts device status into a concise, deterministic tray view.
func BuildState(statuses []keychron.Status, scanErr error) State {
	state := State{
		Title:    "InputScout",
		IconName: "battery-missing",
	}

	batteryByName := make(map[string]BatteryDevice)
	unavailable := make(map[string]bool)
	var errors []string
	for _, status := range statuses {
		if status.Error != "" {
			errors = append(errors, fmt.Sprintf("%s: %s", status.Name, status.Error))
		}
		if !status.Connected {
			continue
		}
		if status.Battery == nil {
			unavailable[status.Name] = unavailable[status.Name] || status.Keyboard != nil && status.Keyboard.ExternalPower
			continue
		}
		id := status.DeviceID
		if id == "" {
			id = status.Name
		}
		batteryByName[status.Name] = BatteryDevice{
			ID:         id,
			Name:       status.Name,
			Percentage: status.Battery.Percentage,
			Charging:   status.Battery.Charging,
		}
		delete(unavailable, status.Name)
	}
	if scanErr != nil {
		errors = append(errors, scanErr.Error())
	}

	for _, battery := range batteryByName {
		state.Batteries = append(state.Batteries, battery)
	}
	sort.Slice(state.Batteries, func(i, j int) bool {
		if state.Batteries[i].Percentage == state.Batteries[j].Percentage {
			return state.Batteries[i].Name < state.Batteries[j].Name
		}
		return state.Batteries[i].Percentage < state.Batteries[j].Percentage
	})

	var lines []string
	for _, battery := range state.Batteries {
		line := fmt.Sprintf("%s: %d%%", battery.Name, battery.Percentage)
		if battery.Charging {
			line += " (charging)"
		}
		lines = append(lines, line)
	}
	var unavailableNames []string
	for name := range unavailable {
		if _, hasBattery := batteryByName[name]; !hasBattery {
			unavailableNames = append(unavailableNames, name)
		}
	}
	sort.Strings(unavailableNames)
	for _, name := range unavailableNames {
		line := name + ": battery unavailable"
		if unavailable[name] {
			line += ", USB power connected"
		}
		lines = append(lines, line)
	}
	lines = append(lines, errors...)

	if len(state.Batteries) > 0 {
		lowest := state.Batteries[0]
		if len(state.Batteries) == 1 {
			state.Title = fmt.Sprintf("InputScout — %s %d%%", lowest.Name, lowest.Percentage)
		} else {
			state.Title = fmt.Sprintf("InputScout — lowest battery %d%%", lowest.Percentage)
		}
		state.IconName = batteryIcon(lowest.Percentage, lowest.Charging)
	} else if len(errors) > 0 {
		state.Title = "InputScout — device error"
		state.IconName = "dialog-warning"
	}
	if len(lines) == 0 {
		lines = append(lines, "No supported devices connected")
	}
	state.Description = strings.Join(lines, "\n")
	return state
}

func batteryIcon(percentage int, charging bool) string {
	percentage = max(0, min(100, percentage))
	level := 0
	if percentage > 0 {
		level = ((percentage + 9) / 10) * 10
	}
	name := fmt.Sprintf("battery-%03d", level)
	if charging {
		name += "-charging"
	}
	return name
}
