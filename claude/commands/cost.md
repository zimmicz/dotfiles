---
description: Token usage and notional cost per session (default yesterday)
allowed-tools: Bash(~/.claude/scripts/usage-report.sh:*)
---

Run the usage report and show its output as-is. Do not recompute or reword any number.

Arguments given by the user: `$ARGUMENTS` (empty = yesterday).

- no arguments → `~/.claude/scripts/usage-report.sh --format slack --no-ledger`
- `today` → `--today`
- `week` → `--week`
- `month` → `--month`
- a date `YYYY-MM-DD` → `--date <date>`
- `since YYYY-MM-DD` → `--since <date>`

Always add `--format slack --no-ledger` unless the user explicitly asks for markdown
(`--format md`, which also writes `~/.claude/usage/report-<window>.md`) or for the ledger to be
updated.

After printing the report, add at most one short line of comment only if something stands out
(a single session dominating the day, an unknown-model pricing warning).
