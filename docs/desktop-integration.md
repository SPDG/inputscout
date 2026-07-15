# Desktop battery integration

InputScout uses a native StatusNotifierItem for its first desktop integration.
It appears in the regular KDE Plasma system tray, uses the desktop icon theme,
updates its tooltip once per minute, and sends low and critical battery
notifications.

## Why the device is not injected into UPower

UPower's public D-Bus interface is read-only for clients. It can enumerate
devices and publish change signals, but it has no method for an unrelated user
process to register a new battery. On Linux, its current backends discover
power devices from the kernel/udev `power_supply` subsystem and battery-bearing
Bluetooth devices from BlueZ.

The BlueZ Battery Provider API is not a generic battery registry. A provider
must associate every value with an existing `org.bluez.Device1` object. That is
useful for a paired Bluetooth keyboard, but cannot represent an M5 8K connected
through its proprietary USB 2.4 GHz receiver.

InputScout therefore does not impersonate UPower or create a synthetic kernel
device. Both approaches would require privileged system components and create
fragile ownership and security boundaries for a user-space HID monitor.

## Long-term system integration options

There are three technically sound paths if integration with the built-in
Power and Battery applet becomes a requirement:

1. Pair a supported device over Bluetooth and use its standard BlueZ
   `org.bluez.Battery1` interface where available.
2. Add and upstream a Linux HID driver that exposes receiver battery state as a
   real `power_supply` device, similar to kernel-supported receiver families.
3. Propose a general user-space battery provider API upstream in UPower, with
   authentication, lifecycle, and conflict handling designed by UPower.

The first two paths work because KDE's battery model consumes `Solid::Battery`
devices, and Solid's Linux backend obtains those from UPower. Logitech HID++
devices are a useful example: when supported by the kernel driver, their
battery is published through Linux's `power_supply` class and then discovered
by UPower. Solaar's own indicator is instead a separate tray item, just like
the initial InputScout integration.

Until one of those exists, StatusNotifierItem is the supported unprivileged
desktop path and works across KDE Plasma and other compatible tray hosts.

## Primary references

- [UPower D-Bus API](https://upower.freedesktop.org/docs/UPower/)
- [UPower device API](https://upower.freedesktop.org/docs/Device.html)
- [BlueZ Battery Provider API](https://kernel.googlesource.com/pub/scm/bluetooth/bluez.git/+/refs/heads/master/doc/org.bluez.BatteryProvider.rst)
- [Linux power supply class](https://docs.kernel.org/power/power_supply_class.html)
- [Linux Logitech HID++ driver](https://github.com/torvalds/linux/blob/master/drivers/hid/hid-logitech-hidpp.c)
- [StatusNotifierItem specification](https://specifications.freedesktop.org/status-notifier-item/latest/status-notifier-item.html)
