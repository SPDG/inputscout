package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/spdg/inputscout/internal/keychron"
)

const usageText = `Usage: inputscout [status|list] [--json]

Commands:
  status  Show connected Keychron devices and available battery state (default)
  list    List connected Keychron devices without querying battery state
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	command := "status"
	if len(args) > 0 && args[0] != "--json" {
		command = args[0]
		args = args[1:]
	}

	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	flags.Usage = func() { fmt.Fprint(stderr, usageText) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected argument: %s\n\n", flags.Arg(0))
		flags.Usage()
		return 2
	}

	includeBattery := false
	switch command {
	case "status":
		includeBattery = true
	case "list":
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", command)
		fmt.Fprint(stderr, usageText)
		return 2
	}

	statuses, err := keychron.Scan(includeBattery)
	if err != nil {
		fmt.Fprintf(stderr, "inputscout: %v\n", err)
		return 1
	}
	if len(statuses) == 0 {
		fmt.Fprintln(stderr, "inputscout: no supported receiver found")
		return 1
	}

	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(statuses); err != nil {
			fmt.Fprintf(stderr, "inputscout: encode JSON: %v\n", err)
			return 1
		}
	} else {
		printHuman(stdout, statuses, includeBattery)
	}

	for _, status := range statuses {
		if status.Error != "" {
			return 1
		}
	}
	return 0
}

func printHuman(w io.Writer, statuses []keychron.Status, includeBattery bool) {
	for i, status := range statuses {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, status.Name)
		fmt.Fprintf(w, "  Connection: %s\n", status.Connection)
		fmt.Fprintf(w, "  Receiver:   %s\n", status.ReceiverID)
		if status.DeviceID != "" {
			fmt.Fprintf(w, "  Device:     %s\n", status.DeviceID)
		}
		fmt.Fprintf(w, "  Connected:  %s\n", yesNo(status.Connected))
		if status.Error != "" {
			fmt.Fprintf(w, "  Error:      %s\n", status.Error)
			continue
		}
		if !includeBattery || !status.Connected {
			continue
		}
		if status.Battery != nil {
			fmt.Fprintf(w, "  Battery:    %d%%\n", status.Battery.Percentage)
			fmt.Fprintf(w, "  Charging:   %s\n", yesNo(status.Battery.Charging))
		} else {
			fmt.Fprintln(w, "  Battery:    unavailable over this connection")
		}
	}
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
