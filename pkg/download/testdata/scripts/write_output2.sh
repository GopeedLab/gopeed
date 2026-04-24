#!/bin/bash
set -e

if [ -z "$GOPEED_TEST_OUTPUT_FILE_2" ]; then
  echo "GOPEED_TEST_OUTPUT_FILE_2 is empty" >&2
  exit 2
fi

echo "Script 2" > "$GOPEED_TEST_OUTPUT_FILE_2"
