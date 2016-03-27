#!/bin/bash

set -euo pipefail

TEST_DIR=$(pwd)
GOSH_DIR="$TEST_DIR/.."

for f in $(ls -1 ./*.gosh); do
	GOLDEN="${f%.gosh}.golden"
	if [ ! -e "$GOLDEN" ]; then
		echo "Missing Golden file: $GOLDEN"
		exit 1
	fi
	diff <(cat "$GOLDEN") <("$GOSH_DIR/gosh" "$f" 2>/dev/null)
	if [ "$?" -eq 1 ]; then
		echo "Gosh running '$f' differs from the golden file"
		exit 1
	fi
done

echo "Passed Golden Files Test"
