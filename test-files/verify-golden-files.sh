#!/bin/bash
set -uo pipefail
VERBOSE=1
CONTINUE_WITH_ERROR=0

while getopts ":vi" opt; do
	case $opt in
		v)
			VERBOSE=0
			;;
		i)
			CONTINUE_WITH_ERROR=1
			;;
		\?)
			echo "Invalid option: -$OPTARG" >&2
			;;
	esac
done

TEST_DIR=$(pwd)
GOSH_DIR="$TEST_DIR/.."

cd "$GOSH_DIR"
go build
if [ "$?" -ne 0 ]; then
	echo "Building Gosh failed"
	exit 1
fi
cd "$TEST_DIR"

for f in $(ls -1 *.gosh); do
	GOLDEN="./golden/${f%.gosh}.golden"
	if [ ! -e "$GOLDEN" ]; then
		echo "Missing Golden file: $GOLDEN"
		[[ CONTINUE_WITH_ERROR -eq 1 ]] || exit 1
	fi
	if [ $VERBOSE -eq 0 ]; then
		echo "Testing $f"
	fi
	diff <(cat "$GOLDEN") <("$GOSH_DIR/gosh" "$f" 2>/dev/null) &>/dev/null
	if [ "$?" -ne 0 ]; then
		echo "Gosh running '$f' differs from the golden file."
		echo "Run the below commands to see how"
		echo "cat $GOLDEN"
		echo "echo '======='"
		echo "../gosh ./$f"
		[[ CONTINUE_WITH_ERROR -eq 1 ]] || exit 1
	fi
done

echo "Passed Golden Files Test"
