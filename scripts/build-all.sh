#!/bin/bash
set -e

VERSION="${VERSION:-dev}"
OUT="dist"

mkdir -p "$OUT"

echo "[build] building Parcel $VERSION for all platforms..."

# Linux amd64
GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-linux-amd64" ./client/main.go

# Linux arm64 (Raspberry Pi, ARM servers)
GOOS=linux GOARCH=arm64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-linux-arm64" ./client/main.go

# macOS amd64 (Intel)
GOOS=darwin GOARCH=amd64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-darwin-amd64" ./client/main.go

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-darwin-arm64" ./client/main.go

# Windows amd64
GOOS=windows GOARCH=amd64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-windows-amd64.exe" ./client/main.go

echo "[build] done. binaries in $OUT/"
ls -lh "$OUT/"
