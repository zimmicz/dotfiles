#!/usr/bin/env bash
set -euo pipefail

HOOK_FILE="$HOME/.config/claude/slack-webhook"
PREVIEW="${1:-Claude nightly}"

[ -s "$HOOK_FILE" ] || { echo "slack-post: no webhook url in $HOOK_FILE" >&2; exit 1; }

jq -Rs --arg p "$PREVIEW" '{text: $p, blocks: [{type: "markdown", text: .}]}' \
  | curl -fsS -X POST -H 'Content-type: application/json' --data @- "$(cat "$HOOK_FILE")"
echo
