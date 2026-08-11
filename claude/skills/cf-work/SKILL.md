---
name: cf-work
description: |
  Work on a Jira ticket. Use when user says "work on", "analyze ticket",
  "cf-work", "requirements", or provides a CFD-XXXX ticket to analyze.
---

# Work on a Jira ticket

Ticket: $ARGUMENTS

Run this command inside a worktree created by `/cf-worktree`.

## Steps

1. If no ticket ID provided in arguments, detect it from current branch name (CFD-XXXX pattern)
2. Fetch ticket via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`, fields: summary, description, status, assignee, priority, comment)
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

   Also scan `.claude/coding-playbook.md` for rules relevant to this ticket's domain. Flag applicable ones as `[playbook]` items.

6. **STOP** — present summary + checklist. Wait for confirmation or answers. Do NOT explore code or start implementation until confirmed.
7. Post a Jira comment **in Czech** (via `atlassian.addCommentToJiraIssue`) summarizing: requirements, resolved assumptions, approach

## Rules

- Commit format: `type: [CFD-XXXX] english description` (conventional commits)
- Types: feat, fix, chore, refactor, test, docs
- No Czech characters in commits
- All Jira comments must be written in Czech
