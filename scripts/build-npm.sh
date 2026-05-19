#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
DIST_DIR="$ROOT_DIR/dist"

mkdir -p "$DIST_DIR"
rm -f "$DIST_DIR"/gitid-*

build() {
  goos="$1"
  goarch="$2"
  out="$DIST_DIR/gitid-$goos-$goarch"
  echo "building $out"
  (cd "$ROOT_DIR" && CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$out" ./cmd/gitid)
  chmod 755 "$out"
}

build darwin arm64
build darwin amd64
build linux arm64
build linux amd64
