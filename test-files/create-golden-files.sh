#!/bin/bash

set -euo pipefail

TEST_DIR=$(pwd)
GOSH_DIR="$TEST_DIR/.."

cd "$GOSH_DIR"
go build
cd "$TEST_DIR"

for f in $(ls -1 ./*.gosh); do
	"$GOSH_DIR/gosh" "$f" 2>/dev/null > "golden/${f%.gosh}.golden"
done
