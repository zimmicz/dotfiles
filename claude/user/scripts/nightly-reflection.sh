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
STANDARDS_DIR="$HOME/dotfiles/claude/cleverfarm/standards"
ADD_DIRS=(--add-dir "$HOME/.claude" --add-dir "$STANDARDS_DIR")
for r in "${REPOS[@]}"; do ADD_DIRS+=(--add-dir "$r"); done

TRUNC_NOTE=""
if [ "$TRUNCATED" -gt 0 ]; then
  TRUNC_NOTE=" Note: $TRUNCATED older sessions were skipped due to the daily volume cap, mention this in the briefing."
fi

COMPACT_NOTE=""
if [ "$(date +%u)" = 7 ]; then
  COMPACT_NOTE=" Today is compaction day: after appending, rewrite the '## Learned' list in $LEARNED_FILE in place - merge duplicate or overlapping bullets into one, drop bullets that are obsolete or already covered by $CLAUDE_MD, keep the terse style and the 'User: ...' quotes, cap at $MAX_LEARNED bullets (keep the most recurring / most costly-to-get-wrong). Do not touch anything above the '## Learned' heading. Report how many bullets before/after."
fi

PROMPT="Read $LEARNED_FILE, $CLAUDE_MD and every file in $STANDARDS_DIR first, so you know what rules already exist. Then read $DIGEST_FILE - a digest of yesterday's Claude Code sessions ('### session' blocks with cwd + repo, then [user] prompts and [claude] replies, tool output stripped). Look for explicit corrections the user gave Claude, or preferences the user confirmed without pushback. Classify each lesson:
- CODE STANDARD (how code itself should be written, i.e. something a reviewer reading only the diff would flag: structure, naming, types, components, styling, tests): append to $STANDARDS_DIR/<repo dir name>.md, or $STANDARDS_DIR/common.md if it holds across repos, under the matching category heading (create file/heading if missing), format '- <rule> — evidence: session <date> — lint: <eslint rule|no>'. Skip duplicates of those files and of <repo>/CLAUDE.md. Never put these in $LEARNED_FILE.
- GLOBAL (tool choice, communication style, workflow habits, cross-project): append one terse bullet (fragments OK, sacrifice grammar for concision, matching existing style) under '## Learned' in $LEARNED_FILE. Skip duplicates of what's already there or in $CLAUDE_MD.
- REPO-SPECIFIC (conventions, gotchas, commands, structure of one codebase): append to <repo>/CLAUDE.local.md under a '## Learned' heading (create the file with that heading if missing), where <repo> is the 'repo:' path of the session. Skip duplicates of what's already in that file or in <repo>/CLAUDE.md. Skip if repo is 'none'.
- Ticket details, one-off facts, things Claude got right: skip.
Append only, never delete or rewrite existing bullets, do not touch any other file.$TRUNC_NOTE$COMPACT_NOTE Finally print a terse summary as your final answer: how many sessions you reviewed, what new rules you added and where (or that there were none), one line why each. Do not send it anywhere."

SUMMARY=$(claude -p "$PROMPT" --allowedTools "Read Edit Write" "${ADD_DIRS[@]}")
echo "$SUMMARY"
printf '%s' "$SUMMARY" | "$HOME/.claude/scripts/slack-post.sh" "Claude reflection" || echo "slack post failed"

echo "$NOW" > "$STATE_FILE"
echo "=== done $(date -Iseconds) ==="
