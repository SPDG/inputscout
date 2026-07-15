# Keychron K8 HE wired protocol notes

These notes document the read-only subset used by InputScout with a K8 HE
running stock firmware v1.1.1. The implementation was validated on physical
hardware through USB ID `3434:0e80` and vendor configuration interface 01.

InputScout does not open the boot-keyboard interface, capture keypresses, read
profile names, change keyboard settings, or invoke firmware update commands.

## Transport

The configuration interface uses vendor usage page `0xff60`, usage `0x61`, and
unnumbered 32-byte input/output reports. HIDAPI output buffers therefore contain
a leading zero report ID followed by the 32-byte protocol payload.

The keyboard can expose its USB configuration interface while its physical
connection selector still reports 2.4 GHz. InputScout presents these as two
separate facts: `connection: USB` describes the host path used for the query,
while `device_mode: 2.4 GHz` describes the selector state returned by firmware.

## Implemented reads

| Command | Meaning | Fields currently decoded |
| --- | --- | --- |
| `A0` | Protocol information | Protocol version and instruction-set version |
| `A1` | Firmware information | Firmware version and build timestamp |
| `A2` | Feature flags | Default layer, Bluetooth, 2.4 GHz, analog matrix, Keychron RGB |
| `A3` | Default layer | macOS/Windows mode and layer index |
| `A9 01` | Analog protocol version | Version |
| `A9 10` | Analog profile information | Current/count/size plus OKMC and SOCD slot counts |
| `A9 12` | Raw profile range | Only the four-byte global config at offset zero |
| `AB 05` | Factory transport read | Physical connection-selector mode and USB power sense with checksum validation |

The `A9 12` request deliberately asks for only four bytes. That is enough to
decode the active profile's regular/rapid-trigger mode, actuation point, and
press/release sensitivity without collecting per-key mappings or its name.

For K8 HE, the third `AB 05` payload byte is the active-low
`USB_POWER_SENSE_PIN`. It confirms that external USB power is present, but it
cannot distinguish a battery that is actively charging from one that is
already full. InputScout therefore labels this field `USB power` rather than
`Charging`.

## Battery result

No battery percentage is available through the stock wired path currently
known to InputScout:

- the USB report descriptors do not expose a standard HID battery usage;
- Ubuntu UPower does not create a battery device for the wired K8 HE;
- the public raw-HID dispatcher implements protocol, firmware, feature, layer,
  and analog-matrix reads but no keyboard battery read;
- the firmware has an internal battery percentage function, while the
  `BAT_LVL` keycode displays charge on the keyboard LEDs rather than returning
  it to the host;
- the factory charging test is not a percentage query and is intentionally not
  used.

The next non-invasive experiment is Bluetooth mode: after pairing, check
whether BlueZ exports `org.bluez.Battery1`. If it does not, a new receiver
command or a small opt-in firmware extension would be required for host battery
telemetry.

## Primary sources

- [Keychron raw-HID dispatcher](https://github.com/Keychron/qmk_firmware/blob/hall_effect_playground/keyboards/keychron/common/keychron_raw_hid.c)
- [Analog-matrix command dispatcher](https://github.com/Keychron/qmk_firmware/blob/hall_effect_playground/keyboards/keychron/common/analog_matrix/analog_matrix.c)
- [Analog profile data types](https://github.com/Keychron/qmk_firmware/blob/hall_effect_playground/keyboards/keychron/common/analog_matrix/analog_matrix_type.h)
- [Factory test commands](https://github.com/Keychron/qmk_firmware/blob/hall_effect_playground/keyboards/keychron/common/factory_test.c)
- [Wireless battery implementation](https://github.com/Keychron/qmk_firmware/blob/hall_effect_playground/keyboards/keychron/common/wireless/battery.c)
- [Keychron Launcher](https://launcher.keychron.com/)
