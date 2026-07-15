<p align="center">
  <img src="assets/logo-lockup.svg" alt="InputScout" width="760">
</p>

# InputScout

[![CI](https://github.com/spdg/inputscout/actions/workflows/ci.yml/badge.svg)](https://github.com/spdg/inputscout/actions/workflows/ci.yml)

`inputscout` is an experimental Linux command-line tool for inspecting input
devices. It communicates with supported receiver and wired configuration
interfaces directly through HID reports and does not modify onboard settings.

The project is at an early stage. Its first goal is reliable device discovery
and battery reporting, followed by a background service and desktop status
indicator.

[Roadmap](ROADMAP.md) · [Brand assets](assets/BRAND.md)

## Current support

| Device | 2.4 GHz detection | Wired telemetry | Battery | Writes |
| --- | --- | --- | --- | --- |
| Keychron M5 8K | Yes | Not yet | Yes over receiver | No |
| Keychron K8 HE | Yes | Firmware, mode, features, and active HE profile | Not exposed over USB or receiver | No |

The K8 HE firmware tracks its battery internally, but stock firmware v1.1.1
does not expose a percentage through its public wired raw-HID commands or the
known 2.4 GHz receiver protocol. The remaining stock-firmware path to test is
BlueZ's standard `Battery1` interface in Bluetooth mode. See the
[wired protocol notes](docs/protocols/k8-he-wired.md) for evidence and scope.

## Build

Building requires Go, GCC, and the `libudev` development headers.

```console
make test
make build
```

The resulting binary is written to `bin/inputscout`.

## Device permissions

Install the included udev rule once, then reconnect the receivers if the
current desktop session does not receive access immediately:

```console
make install-udev
```

The rules grant the active local session access only to the configuration
interfaces of the currently supported receivers and K8 HE wired interface 01.
They do not make normal keyboard input or all HID devices globally writable.

## Usage

```console
bin/inputscout status
bin/inputscout status --json
bin/inputscout list
```

Example:

```text
Keychron K8 HE
  Connection: USB
  Device:     3434:0e80
  Connected:  yes
  Device mode: 2.4 GHz
  Firmware:   v1.1.1
  Build:      2025-06-17 10:42:45
  Protocol:   2 (instruction set 2)
  OS mode:    Windows (layer 2)
  Features:   default layer, Bluetooth, 2.4 GHz, analog matrix, Keychron RGB
  HE profile: 1/3 (regular, 2.0 mm)
  Battery:    unavailable over this connection

Keychron M5 8K
  Connection: 2.4 GHz
  Receiver:   3434:d028
  Device:     3434:d048
  Connected:  yes
  Battery:    17%
  Charging:   no
```

## Safety

The implemented commands only send receiver identity and read-only status
queries that are also implemented by Keychron firmware and Launcher. InputScout
does not capture key events or read profile names. Device configuration writes
and firmware updates are intentionally out of scope for this first version.

## Project independence

InputScout is an independent community project. It is not affiliated with or
endorsed by Keychron. Product and company names are used only to identify
compatible hardware.

## Roadmap

The planned path includes a replayable driver architecture, diagnostics,
read-only mouse configuration, safe reversible writes, K8 HE battery research,
a D-Bus service, desktop clients, native packages, and a contributor-facing
driver model. See the complete [project roadmap](ROADMAP.md).
