#!/usr/bin/env bash
# build-win.sh — Windows-only local debug build
# Usage: bash build-win.sh
set -e

VERSION="dev"

echo "==> [1/4] Building dashboard..."
cd dashboard
npm run build
echo "==> [2/4] Injecting dashboard into relay embed..."
rm -rf ../relay/internal/web/dist
cp -r dist ../relay/internal/web/dist
cd ..

echo "==> [3/4] Building relay (windows/amd64)..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -tags embed \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -o relay/renode-relay.exe \
  ./relay

echo "==> [4/4] Building client (windows/amd64)..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -tags embed \
  -ldflags="-s -w -X main.clientVersion=${VERSION}" \
  -o client/renode.exe \
  ./client

echo ""
echo "Done!"
echo "  relay/renode-relay.exe"
echo "  client/renode.exe"
