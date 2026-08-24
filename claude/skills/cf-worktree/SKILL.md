---
name: cf-worktree
description: |
  Create worktree for a Jira ticket. Use when user says "worktree",
  "cf-worktree", "setup ticket", or "new branch for ticket".
---

# Create worktree for a Jira ticket

Arguments: `<ticket-id> [base-branch]` (base defaults to `develop`; script auto-switches to `production` for hotfix tickets)

Thin wrapper around `~/bin/cf-work` — the script does everything: Jira fetch, assign, transition to In Progress, branch name (≤45 chars, Amplify subdomain limit), worktree, `.env` copy, `TICKET.md`, deps install (lockfile-detected, background), codegraph init, model-tier sizing.

## Steps

1. Check `JIRA_EMAIL` and `JIRA_TOKEN` are set (`[[ -n "$JIRA_EMAIL" && -n "$JIRA_TOKEN" ]]`). If missing, tell user to export them and stop.
2. Run: `~/bin/cf-work <ticket-id> [base-branch] --no-launch`
3. Relay script output to user: worktree path, model tier + reason, and the printed launch command (`cd .worktrees/... && claude --model <tier> "/cf-ticket <ticket>"`).
4. If the script fails, show its error output and stop — do NOT re-implement the flow via Jira MCP calls.

## Rules

- Do NOT summarize the ticket or start implementation.
- User may override model tier via `CF_MODEL=tier` env or by editing the launch command.
