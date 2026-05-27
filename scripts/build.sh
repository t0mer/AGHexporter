#!/usr/bin/env bash
set -euo pipefail

BIN=adguardhome-exporter
PKG=./cmd/adguardhome-exporter
LDFLAGS="-s -w"
mkdir -p dist

build() {
  local os=$1 arch=$2 arm=$3 suffix=$4
  local out="dist/${BIN}-${suffix}"
  echo "Building $out …"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" GOARM="$arm" \
    go build -trimpath -ldflags="$LDFLAGS" -o "$out" "$PKG"
}

build linux  amd64 ""  linux-amd64
build linux  arm64 ""  linux-arm64
build linux  386   ""  linux-386
build linux  arm   6   linux-armv6
build linux  arm   7   linux-armv7
build darwin amd64 ""  darwin-amd64
build darwin arm64 ""  darwin-arm64
build windows amd64 "" windows-amd64.exe
build windows arm64 "" windows-arm64.exe
