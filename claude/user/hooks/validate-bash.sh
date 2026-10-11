#!/usr/bin/env bash
set -eo pipefail

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null) || exit 0
[[ -z "$COMMAND" ]] && exit 0

ask() {
  printf '{"decision": "ask", "reason": "%s"}' "$1"
  exit 0
}

# rm -rf /
if printf '%s' "$COMMAND" | grep -qE 'rm\s+(-[a-zA-Z]*r[a-zA-Z]*f|(-[a-zA-Z]*f[a-zA-Z]*r))\s+/(\s|$|\*)'; then
  ask "rm -rf / is blocked for safety"
fi

# git push --force
if printf '%s' "$COMMAND" | grep -qE 'git\s+push\s+.*(-f|--force)'; then
  ask "git push --force is blocked. Use --force-with-lease"
fi

# --no-verify (skips pre-commit hooks)
if printf '%s' "$COMMAND" | grep -qE 'git\s+(commit|push)\s+.*--no-verify'; then
  ask "--no-verify skips hooks. Fix the underlying issue instead"
fi

# git add -A / git add .
if printf '%s' "$COMMAND" | grep -qE 'git\s+add\s+(-A|--all|\.)(\s|$)'; then
  ask "git add -A is blocked. Stage specific files instead"
fi

# sudo
if printf '%s' "$COMMAND" | grep -qE '(^|[;&|]\s*)sudo\s'; then
  ask "sudo is blocked in automated mode"
fi

exit 0
