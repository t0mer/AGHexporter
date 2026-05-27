BINARY  := bin/adguardhome-exporter
LDFLAGS := -trimpath -ldflags="-s -w"

.PHONY: build test lint run release clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/adguardhome-exporter

test:
	go test ./...

lint:
	golangci-lint run

run:
	ADGUARD_URL_1=http://localhost \
	ADGUARD_USERNAME_1=admin \
	ADGUARD_PASSWORD_1=secret \
	go run ./cmd/adguardhome-exporter

release:
	./scripts/build.sh

clean:
	rm -rf bin/ dist/
