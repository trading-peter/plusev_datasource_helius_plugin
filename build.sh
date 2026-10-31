#!/bin/bash
set -e

GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm -buildmode=c-shared .
echo "Built plugin.wasm"
