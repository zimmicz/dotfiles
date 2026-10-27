#!/usr/bin/env bash
set -uo pipefail
export TZ=Europe/Prague
export LC_NUMERIC=C

SCRIPTS="$HOME/.claude/scripts"
LOG_DIR="$HOME/.claude/logs"
USAGE_DIR="$HOME/.claude/usage"
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
  echo "--- dry run: skipping reflection and slack post"
else
  echo "--- reflection"
  "$SCRIPTS/nightly-reflection.sh"
  echo "reflection exit: $?"
fi

if [ "$DRY_RUN" = 0 ] && [ -s "$USAGE_TXT" ]; then
  echo "--- usage slack post"
  "$SCRIPTS/slack-post.sh" "Claude usage $DATE" < "$USAGE_TXT"
  echo "slack exit: $?"
fi

if [ "$DRY_RUN" = 0 ] && [ "$(date +%u)" = 1 ]; then
  echo "--- factory stats"
  if STATS=$("$HOME/bin/factory-stats" 7); then
    printf '%s\n' "$STATS"
    printf '%s\n' "$STATS" | "$SCRIPTS/slack-post.sh" "Factory yield"
    echo "slack exit: $?"
  else
    echo "factory-stats failed"
  fi
fi

find "$USAGE_DIR" -name 'report-*.md' -mtime +30 -delete 2>/dev/null
find "$LOG_DIR" -name 'usage-*.txt' -mtime +30 -delete 2>/dev/null
find "$LOG_DIR" -name 'nightly-*.log' -mtime +30 -delete 2>/dev/null

echo "=== done $(date -Iseconds) ==="
