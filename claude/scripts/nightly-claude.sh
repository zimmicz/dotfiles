#!/usr/bin/env bash
set -uo pipefail
export TZ=Europe/Prague
export LC_NUMERIC=C

SCRIPTS="$HOME/.claude/scripts"
LOG_DIR="$HOME/.claude/logs"
USAGE_DIR="$HOME/.claude/usage"
SLACK_SELF_ID="U063NAS9A7P"
DATE="$(date -d yesterday +%F)"
LOG_FILE="$LOG_DIR/nightly-$(date +%F).log"
USAGE_TXT="$LOG_DIR/usage-$DATE.txt"
DRY_RUN=0

[ "${1:-}" = "--dry-run" ] && DRY_RUN=1

mkdir -p "$LOG_DIR" "$USAGE_DIR"
[ "$DRY_RUN" = 1 ] || exec >>"$LOG_FILE" 2>&1

echo "=== nightly wrapper $(date -Iseconds) ==="

echo "--- usage report for $DATE"
if "$SCRIPTS/usage-report.sh" --date "$DATE" --format slack > "$USAGE_TXT.tmp"; then
  mv "$USAGE_TXT.tmp" "$USAGE_TXT"
  "$SCRIPTS/usage-report.sh" --date "$DATE" --format md --no-ledger > /dev/null
  "$SCRIPTS/usage-report.sh" --date "$DATE" --format html --no-ledger > /dev/null
  cat "$USAGE_TXT"
else
  echo "usage-report.sh failed (exit $?)"
  rm -f "$USAGE_TXT.tmp" "$USAGE_TXT"
fi

if [ "$DRY_RUN" = 1 ]; then
  echo "--- dry run: skipping reflection and slack dm"
else
  echo "--- reflection"
  "$SCRIPTS/nightly-reflection.sh"
  echo "reflection exit: $?"
fi

if [ "$DRY_RUN" = 0 ] && [ -s "$USAGE_TXT" ]; then
  echo "--- usage slack dm"
  claude -p "Read $USAGE_TXT and send its contents VERBATIM as a Slack DM to channel_id $SLACK_SELF_ID. It contains a markdown table - pass it through exactly as written, keeping every pipe and separator row. Do not add commentary, do not reformat, do not recompute any number. Reply with 'sent' when done." \
    --allowedTools "Read mcp__plugin_slack_slack__slack_send_message" \
    --add-dir "$HOME/.claude"
  echo "dm exit: $?"
fi

find "$USAGE_DIR" -name 'report-*.md' -mtime +30 -delete 2>/dev/null
find "$LOG_DIR" -name 'usage-*.txt' -mtime +30 -delete 2>/dev/null
find "$LOG_DIR" -name 'nightly-*.log' -mtime +30 -delete 2>/dev/null

echo "=== done $(date -Iseconds) ==="
