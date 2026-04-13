#!/bin/bash
set -e

if [ -z "$GOPEED_TEST_DEST_DIR" ]; then
  echo "GOPEED_TEST_DEST_DIR is empty" >&2
  exit 2
fi

mkdir -p "$GOPEED_TEST_DEST_DIR"
mv "$GOPEED_TASK_PATH" "$GOPEED_TEST_DEST_DIR/"
