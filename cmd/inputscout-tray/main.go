package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spdg/inputscout/internal/keychron"
	"github.com/spdg/inputscout/internal/statusnotifier"
	"github.com/spdg/inputscout/internal/tray"
)

const (
	defaultPollInterval = time.Minute
	minimumPollInterval = 10 * time.Second
)

func main() {
	interval := flag.Duration("interval", defaultPollInterval, "device polling interval")
	notifications := flag.Bool("notifications", true, "send low-battery notifications")
	flag.Parse()
	if *interval < minimumPollInterval {
		fmt.Fprintf(os.Stderr, "inputscout-tray: interval must be at least %s\n", minimumPollInterval)
		os.Exit(2)
	}

	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("inputscout-tray: ")

	initial := scanState()
	item, err := statusnotifier.New(initial.Title, initial.IconName, initial.Description)
	if err != nil {
		log.Fatal(err)
	}
	defer item.Close()

	previousTiers := make(map[string]int)
	if *notifications {
		notifyLowBatteries(item, initial, previousTiers)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state := scanState()
			if err := item.Update(state.Title, state.IconName, state.Description); err != nil {
				log.Printf("update tray item: %v", err)
			}
			if *notifications {
				notifyLowBatteries(item, state, previousTiers)
			}
		}
	}
}

func scanState() tray.State {
	statuses, err := keychron.Scan(true)
	return tray.BuildState(statuses, err)
}

func notifyLowBatteries(item *statusnotifier.Item, state tray.State, previous map[string]int) {
	current := make(map[string]struct{}, len(state.Batteries))
	for _, battery := range state.Batteries {
		current[battery.ID] = struct{}{}
		tier := batteryTier(battery.Percentage)
		previousTier, seen := previous[battery.ID]
		previous[battery.ID] = tier
		if tier == 2 || seen && tier >= previousTier {
			continue
		}
		summary := "Low battery"
		urgency := byte(1)
		if tier == 0 {
			summary = "Critical battery"
			urgency = 2
		}
		body := fmt.Sprintf("%s: %d%%", battery.Name, battery.Percentage)
		if err := item.Notify(summary, body, state.IconName, urgency); err != nil {
			log.Printf("send low-battery notification: %v", err)
		}
	}
	for id := range previous {
		if _, present := current[id]; !present {
			delete(previous, id)
		}
	}
}

func batteryTier(percentage int) int {
	switch {
	case percentage <= 10:
		return 0
	case percentage <= 20:
		return 1
	default:
		return 2
	}
}
