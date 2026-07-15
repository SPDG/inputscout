.PHONY: build test clean install-user uninstall-user install-udev

build:
	go build -o bin/inputscout ./cmd/inputscout
	go build -o bin/inputscout-tray ./cmd/inputscout-tray

test:
	go test ./...

clean:
	rm -rf bin

install-user: build
	install -Dm0755 bin/inputscout $(HOME)/.local/bin/inputscout
	install -Dm0755 bin/inputscout-tray $(HOME)/.local/bin/inputscout-tray
	install -Dm0644 systemd/inputscout-tray.service $(HOME)/.config/systemd/user/inputscout-tray.service
	systemctl --user daemon-reload
	systemctl --user enable --now inputscout-tray.service

uninstall-user:
	-systemctl --user disable --now inputscout-tray.service
	rm -f $(HOME)/.local/bin/inputscout $(HOME)/.local/bin/inputscout-tray
	rm -f $(HOME)/.config/systemd/user/inputscout-tray.service
	systemctl --user daemon-reload

install-udev:
	sudo install -m 0644 udev/70-inputscout.rules /etc/udev/rules.d/70-inputscout.rules
	sudo udevadm control --reload-rules
	sudo udevadm trigger --subsystem-match=hidraw
