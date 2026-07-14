.PHONY: build test clean install-udev

build:
	go build -o bin/inputscout ./cmd/inputscout

test:
	go test ./...

clean:
	rm -rf bin

install-udev:
	sudo install -m 0644 udev/70-inputscout.rules /etc/udev/rules.d/70-inputscout.rules
	sudo udevadm control --reload-rules
	sudo udevadm trigger --subsystem-match=hidraw
