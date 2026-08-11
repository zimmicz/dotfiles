#!/usr/bin/env bash
set -eo pipefail

INPUT=$(cat)
MODE=$(printf '%s' "$INPUT" | jq -r '.permission_mode // empty' 2>/dev/null) || exit 0

if [[ "$MODE" == "plan" ]]; then
  FILE_PATH=$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null)
  [[ "$FILE_PATH" == *.md ]] && exit 0
  echo "Write/Edit blocked in plan mode" >&2
  exit 2
fi

exit 0
