---
name: cf-work
description: |
  Work on a Jira ticket. Use when user says "work on", "analyze ticket",
  "cf-work", "requirements", or provides a ticket ID (e.g. DEV-XXXX) to analyze.
---

# Work on a Jira ticket

Ticket: $ARGUMENTS

Run this command inside a worktree created by `cf-work` (script) or `/cf-worktree`.

## Steps

1. If no ticket ID provided in arguments, detect it from current branch name (`[A-Z]+-\d+` pattern)
2. Get ticket content — cheapest source first:
   - If `TICKET.md` exists in worktree root, read it. Do NOT call Jira MCP unless the user asks for a refresh.
   - Otherwise fetch via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`, fields: summary, description, status, assignee, priority, comment)
3. Read ticket comments — they often contain analysis, decisions, or scope clarifications
4. Summarize ticket requirements concisely (incorporate insights from comments)
5. **Requirement analysis** — check for gaps through these lenses:
   - **Edge cases**: empty/error/boundary states, concurrent users
   - **Acceptance criteria**: is "done" testable? missing scenarios?
   - **Assumptions**: API shape, component reuse, routing, state management
   - **Dependencies**: other services, APIs, tickets, BE changes needed
   - **Scope boundaries**: what's OUT that someone might expect IN?

   Format: concise bullets, max 2-3 per lens, skip empty lenses. Mark each:
   - `[?]` = needs your answer before coding
   - `[!]` = assumption, proceeding unless you disagree

6. **STOP** — present summary + checklist. Wait for confirmation or answers. Do NOT explore code or start implementation until confirmed.

> No Jira comment here — `/cf-ticket` posts one consolidated Czech comment (requirements + assumptions + plan) after the plan is confirmed. Standalone use: post only if the user asks.

## Rules

- Commit format: `type: [DEV-XXXX] english description` (conventional commits, use the actual ticket key)
- Types: feat, fix, chore, refactor, test, docs
- No Czech characters in commits
- All Jira comments must be written in Czech
