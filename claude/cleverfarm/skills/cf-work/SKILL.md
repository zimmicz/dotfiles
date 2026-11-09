---
name: cf-work
description: |
  Analyze a Jira ticket's requirements and surface gaps, interactive. Use when
  user says "analyze ticket", "analyze requirements", "check gaps", "cf-work",
  or "requirements". For end-to-end ticket work use cf-ticket instead. Unattended
  factory analysis (factory-worker, label triage) uses /cf-work-headless instead.
---

# Analyze ticket requirements

Ticket: $ARGUMENTS

Run inside a worktree created by the launcher script or `/cf-worktree`.

## Steps

1. If no ticket ID provided in arguments, detect it from current branch name (`[A-Z]+-\d+` pattern)
2. Read `TICKET.md` in the worktree root — description, parent chain, linked issues (blocks/relates/etc, with their descriptions) and comments are already there. Only call `atlassian.getJiraIssue` (cloudId `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`) if the file is missing or the user asks for a refresh.
3. Summarize requirements concisely, folding in the comments and linked issues — they often carry analysis, decisions, scope clarifications, or blockers
4. **Requirement analysis** — check for gaps through these lenses:
   - **Edge cases**: empty/error/boundary states, concurrent users
   - **Acceptance criteria**: is "done" testable? missing scenarios?
   - **Assumptions**: API shape, component reuse, routing, state management
   - **Dependencies**: other services, APIs, tickets, BE changes needed
   - **Scope boundaries**: what's OUT that someone might expect IN?
   - **Terms** (only if `GLOSSARY.md` exists at the repo root; read it first): Czech ticket words → EN term via `_CZ_`; concept with no entry → proposed entry as `[!]`; ticket contradicts a definition → `[?]`. No writes here, `/cf-ticket` handles the glossary.

   Format: concise bullets, max 2-3 per lens, skip empty lenses. Mark each:
   - `[?]` = needs your answer before coding
   - `[!]` = assumption, proceeding unless you disagree

5. **STOP** — present summary + checklist. Wait for confirmation or answers. Do NOT explore code or start implementation until confirmed.

> No Jira comment here — `/cf-ticket` posts one consolidated Czech comment (requirements + assumptions + plan) after the plan is confirmed. Standalone use: post only if the user asks.

> Unattended factory analysis (factory-worker): `/cf-work-headless`, nothing here applies to it.
