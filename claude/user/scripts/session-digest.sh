#!/usr/bin/env bash
set -eu

for f in "$@"; do
  cwd=$(jq -rn 'first(inputs | select(.cwd) | .cwd) // ""' "$f")
  repo=""
  if [ -n "$cwd" ] && [ -d "$cwd" ]; then
    repo=$(git -C "$cwd" rev-parse --path-format=absolute --git-common-dir 2>/dev/null | xargs -r dirname || true)
  fi
  printf '\n### session %s\ncwd: %s\nrepo: %s\n\n' "$(basename "$f" .jsonl)" "${cwd:-?}" "${repo:-none}"
  jq -r '
    if .type=="user" and (.message.content|type)=="string" and ((.isMeta//false)|not) then
      "[user] " + (.message.content|.[0:1500])
    elif .type=="assistant" then
      (.message.content[]? | select(.type=="text") | "[claude] " + (.text|.[0:600]))
    else empty end
  ' "$f"
done
