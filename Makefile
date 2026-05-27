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
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/adguardhome-exporter-linux-amd64   ./cmd/adguardhome-exporter
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/adguardhome-exporter-linux-arm64   ./cmd/adguardhome-exporter
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/adguardhome-exporter-darwin-arm64  ./cmd/adguardhome-exporter
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/adguardhome-exporter-windows-amd64.exe ./cmd/adguardhome-exporter

clean:
	rm -rf bin/ dist/
