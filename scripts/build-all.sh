#!/bin/bash
set -e

VERSION="${VERSION:-v2.0.0}"
OUT="dist"

mkdir -p "$OUT"

echo "[build] building Parcel $VERSION for all platforms..."

GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-linux-amd64" ./client/main.go

GOOS=linux GOARCH=arm64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-linux-arm64" ./client/main.go

GOOS=darwin GOARCH=amd64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-darwin-amd64" ./client/main.go

GOOS=darwin GOARCH=arm64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-darwin-arm64" ./client/main.go

GOOS=windows GOARCH=amd64 go build \
    -ldflags="-X main.version=$VERSION" \
    -o "$OUT/parcel-windows-amd64.exe" ./client/main.go

echo "[build] done. binaries in $OUT/"
ls -lh "$OUT/"
