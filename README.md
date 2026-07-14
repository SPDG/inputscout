<p align="center">
  <img src="assets/logo-lockup.svg" alt="InputScout" width="760">
</p>

# InputScout

[![CI](https://github.com/spdg/inputscout/actions/workflows/ci.yml/badge.svg)](https://github.com/spdg/inputscout/actions/workflows/ci.yml)

`inputscout` is an experimental Linux command-line tool for inspecting wireless
input devices. It communicates with supported receivers directly through HID
reports and does not modify onboard device settings.

The project is at an early stage. Its first goal is reliable device discovery
and battery reporting, followed by a background service and desktop status
indicator.

[Roadmap](ROADMAP.md) · [Brand assets](assets/BRAND.md)

## Current support

| Device | Receiver detection | Battery | Configuration |
| --- | --- | --- | --- |
| Keychron M5 8K | Yes | Yes | Not yet |
| Keychron K8 HE | Yes | Not exposed by the 2.4 GHz protocol | Not yet |

The K8 HE firmware tracks its battery internally, but its current public raw
HID protocol does not expose that value to the host through the receiver.

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
interfaces of the currently supported receivers. They do not make all HID
devices globally writable.

## Usage

```console
bin/inputscout status
bin/inputscout status --json
bin/inputscout list
```

Example:

```text
Keychron M5 8K
  Connection: 2.4 GHz
  Receiver:   3434:d028
  Device:     3434:d048
  Connected:  yes
  Battery:    17%
  Charging:   no
```

## Safety

The implemented commands only send receiver identity and status queries that
are also used by Keychron Launcher. Device configuration writes and firmware
updates are intentionally out of scope for this first version.

## Project independence

InputScout is an independent community project. It is not affiliated with or
endorsed by Keychron. Product and company names are used only to identify
compatible hardware.

## Roadmap

The planned path includes a replayable driver architecture, diagnostics,
read-only mouse configuration, safe reversible writes, K8 HE battery research,
a D-Bus service, desktop clients, native packages, and a contributor-facing
driver model. See the complete [project roadmap](ROADMAP.md).
