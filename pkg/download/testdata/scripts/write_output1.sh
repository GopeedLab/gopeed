#!/bin/bash
set -e

if [ -z "$GOPEED_TEST_OUTPUT_FILE_1" ]; then
  echo "GOPEED_TEST_OUTPUT_FILE_1 is empty" >&2
  exit 2
fi

echo "Script 1" > "$GOPEED_TEST_OUTPUT_FILE_1"
