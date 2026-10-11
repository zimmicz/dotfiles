#!/usr/bin/env bash
set -euo pipefail

STATE_FILE="$HOME/.claude/scripts/.last-reflection-epoch"
LOG_DIR="$HOME/.claude/logs"
LOG_FILE="$LOG_DIR/reflection-$(date +%F).log"
DIGEST_FILE="$LOG_DIR/reflection-$(date +%F)-digest.md"
LEARNED_FILE="$HOME/.claude/LEARNED.md"
CLAUDE_MD="$HOME/.claude/CLAUDE.md"
DIGEST="$HOME/.claude/scripts/session-digest.sh"
MAX_FILES=60
MAX_LEARNED=50
SLACK_CHANNEL_ID="C0C97UE8F2L"

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

"$DIGEST" "${FILES[@]}" > "$DIGEST_FILE"
echo "digest: $DIGEST_FILE ($(wc -c < "$DIGEST_FILE") bytes)"

mapfile -t REPOS < <(grep '^repo: /' "$DIGEST_FILE" | cut -d' ' -f2- | sort -u)
ADD_DIRS=(--add-dir "$HOME/.claude")
for r in "${REPOS[@]}"; do ADD_DIRS+=(--add-dir "$r"); done

TRUNC_NOTE=""
if [ "$TRUNCATED" -gt 0 ]; then
  TRUNC_NOTE=" Note: $TRUNCATED older sessions were skipped due to the daily volume cap, mention this in the briefing."
fi

COMPACT_NOTE=""
if [ "$(date +%u)" = 7 ]; then
  COMPACT_NOTE=" Today is compaction day: after appending, rewrite the '## Learned' list in $LEARNED_FILE in place - merge duplicate or overlapping bullets into one, drop bullets that are obsolete or already covered by $CLAUDE_MD, keep the terse style and the 'User: ...' quotes, cap at $MAX_LEARNED bullets (keep the most recurring / most costly-to-get-wrong). Do not touch anything above the '## Learned' heading. Report how many bullets before/after."
fi

PROMPT="Read $LEARNED_FILE and $CLAUDE_MD first, so you know what rules already exist. Then read $DIGEST_FILE - a digest of yesterday's Claude Code sessions ('### session' blocks with cwd + repo, then [user] prompts and [claude] replies, tool output stripped). Look for explicit corrections the user gave Claude, or preferences the user confirmed without pushback. Classify each lesson:
- GLOBAL (tool choice, communication style, workflow habits, cross-project): append one terse bullet (fragments OK, sacrifice grammar for concision, matching existing style) under '## Learned' in $LEARNED_FILE. Skip duplicates of what's already there or in $CLAUDE_MD.
- REPO-SPECIFIC (conventions, gotchas, commands, structure of one codebase): append to <repo>/CLAUDE.local.md under a '## Learned' heading (create the file with that heading if missing), where <repo> is the 'repo:' path of the session. Skip duplicates of what's already in that file or in <repo>/CLAUDE.md. Skip if repo is 'none'.
- Ticket details, one-off facts, things Claude got right: skip.
Append only, never delete or rewrite existing bullets, do not touch any other file.$TRUNC_NOTE$COMPACT_NOTE Finally send a short plain-text Slack message to channel_id $SLACK_CHANNEL_ID summarizing: how many sessions you reviewed, what new rules you added and where (or that there were none), one line why each. Keep the Slack message terse. Print the same summary as your final answer."

claude -p "$PROMPT" \
  --allowedTools "Read Edit Write mcp__plugin_slack_slack__slack_send_message" \
  "${ADD_DIRS[@]}"

echo "$NOW" > "$STATE_FILE"
echo "=== done $(date -Iseconds) ==="
