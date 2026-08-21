#!/usr/bin/env bash
set -euo pipefail

STATE_FILE="$HOME/.claude/scripts/.last-reflection-epoch"
LOG_DIR="$HOME/.claude/logs"
LOG_FILE="$LOG_DIR/reflection-$(date +%F).log"
LEARNED_FILE="$HOME/.claude/LEARNED.md"
CLAUDE_MD="$HOME/.claude/CLAUDE.md"
MAX_FILES=60
SLACK_SELF_ID="U063NAS9A7P"

mkdir -p "$LOG_DIR"
exec >>"$LOG_FILE" 2>&1

echo "=== reflection run $(date -Iseconds) ==="

NOW=$(date +%s)
if [ -f "$STATE_FILE" ]; then
  CUTOFF=$(cat "$STATE_FILE")
else
  CUTOFF=$((NOW - 26*3600))
fi
echo "cutoff: $(date -d "@$CUTOFF" -Iseconds)"

mapfile -t FILES < <(find "$HOME/.claude/projects" -name "*.jsonl" -newermt "@$CUTOFF" -printf '%T@ %p\n' 2>/dev/null | sort -rn | cut -d' ' -f2-)

if [ ${#FILES[@]} -eq 0 ]; then
  echo "no new sessions since cutoff, skipping"
  echo "$NOW" > "$STATE_FILE"
  exit 0
fi

TRUNCATED=0
if [ ${#FILES[@]} -gt $MAX_FILES ]; then
  TRUNCATED=$((${#FILES[@]} - MAX_FILES))
  FILES=("${FILES[@]:0:$MAX_FILES}")
fi

echo "sessions found: ${#FILES[@]} (truncated: $TRUNCATED)"

FILE_LIST_PATH="$LOG_DIR/reflection-$(date +%F)-filelist.txt"
printf '%s\n' "${FILES[@]}" > "$FILE_LIST_PATH"

TRUNC_NOTE=""
if [ "$TRUNCATED" -gt 0 ]; then
  TRUNC_NOTE=" Note: $TRUNCATED older sessions were skipped due to the daily volume cap, mention this in the briefing."
fi

PROMPT="Read $LEARNED_FILE and $CLAUDE_MD first, so you know what rules already exist. Then read the session transcript files listed in $FILE_LIST_PATH (jsonl, one JSON event per line). Look for explicit corrections the user gave Claude, or preferences the user confirmed without pushback, that are GLOBAL and cross-project (tool choice, communication style, workflow habits) - not project-specific facts or ticket details, skip those. Skip anything that duplicates what's already in $LEARNED_FILE or $CLAUDE_MD. For each new lesson, append one terse bullet (fragments OK, sacrifice grammar for concision, matching the existing style) under the '## Learned' heading in $LEARNED_FILE - append only, never delete or rewrite existing bullets, and do not touch any other file.$TRUNC_NOTE Finally send a short plain-text Slack DM to channel_id $SLACK_SELF_ID summarizing: how many sessions you reviewed, what new rules you added (or that there were none), one line why each. Keep the Slack message terse. Print the same summary as your final answer."

claude -p "$PROMPT" \
  --allowedTools "Read Edit mcp__plugin_slack_slack__slack_send_message" \
  --add-dir "$HOME/.claude"

echo "$NOW" > "$STATE_FILE"
echo "=== done $(date -Iseconds) ==="
