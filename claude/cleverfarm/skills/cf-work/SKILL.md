---
name: cf-work
description: |
  Analyze a Jira ticket's requirements and surface gaps. Use when user says
  "analyze ticket", "analyze requirements", "check gaps", "cf-work", or
  "requirements". For end-to-end ticket work use cf-ticket instead.
  With `--headless` (no user, e.g. triage-worker on Sentry bugs): explores code,
  writes PLAN.md with root-cause + plan, posts one Czech Jira comment, no STOP.
  Re-run after a human reply: answers it, updates PLAN.md.
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

   Format: concise bullets, max 2-3 per lens, skip empty lenses. Mark each:
   - `[?]` = needs your answer before coding
   - `[!]` = assumption, proceeding unless you disagree

5. **STOP** — present summary + checklist. Wait for confirmation or answers. Do NOT explore code or start implementation until confirmed.

> No Jira comment here — `/cf-ticket` posts one consolidated Czech comment (requirements + assumptions + plan) after the plan is confirmed. Standalone use: post only if the user asks.

## Headless mode (`--headless` in arguments)

No user present — never ask, never wait. Replaces step 5:

1. Steps 1–4 as above. Ticket is usually a Sentry bug: stack frames + culprit are in `TICKET.md`.
2. Explore code (ctx_* / codegraph / `git log|blame`) to find root cause. Sentry `get_sentry_resource` / `search_events` for more events if the ticket lacks detail. Read-only — no code changes.
3. Write `PLAN.md` in worktree root:
   - **Root cause** — hypothesis + evidence (file:line, frames, git history). Say plainly if unconfirmed.
   - **Fix plan** — stepped, files to touch, tests to add
   - **Noise?** — if it's not our bug (browser quirk, network, expected 4xx), say so + recommend ignore/filter in Sentry
   - `[?]` / `[!]` checklist from step 4
4. Post ONE Czech Jira comment via `jira add-comment <ticket-id> "<text>"` (bare `jira`, matches the permission allowlist) in the user's voice (`~/.claude/my-voice.md`): root cause, plan summary, `[?]` questions. First line EXACTLY `🤖 _AI triage (Claude, headless). Píšu za Michala. Jestli se vám to nelíbí, dejte mi prosím vědět._` (triage-worker detects replies by it). Jira wiki markup, not markdown: `h3.` headings, `*` bullets, `#` numbered, `{{code}}` inline, `{code}` blocks.

### Follow-up (headless, `PLAN.md` exists + `TICKET.md` has comments after the last `_AI triage` one)

A human replied. Don't redo the analysis from scratch:

1. Read `PLAN.md` + the new comments (after the last AI triage comment) in `TICKET.md`.
2. Apply answers / corrections: resolve `[?]` → `[!]` with the decision, adjust root cause / fix plan if challenged. Re-check code if needed.
3. Update `PLAN.md` in place.
4. Post ONE reply comment, same format + 🤖 prefix + marker: what changed in the plan, remaining `[?]` if any. Nothing to change → short confirmation.
