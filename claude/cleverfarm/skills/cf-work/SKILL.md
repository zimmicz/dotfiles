---
name: cf-work
description: |
  Analyze a Jira ticket's requirements and surface gaps. Use when user says
  "analyze ticket", "analyze requirements", "check gaps", "cf-work", or
  "requirements". For end-to-end ticket work use cf-ticket instead.
  With `--headless` (no user, triage-worker on any ticket labelled triage): explores
  code, writes PLAN.md (root cause for bugs) + plan + holdout scenarios, posts one
  Czech Jira comment, no STOP.
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

1. Steps 1–4 as above. Sentry ticket = has a `sentry-*` label: stack frames + culprit are in `TICKET.md`. Any other ticket (feature, chore, manual bug): requirements from step 3–4 drive the plan.
2. Explore code (ctx_* / codegraph / `git log|blame`) to find root cause (bugs) or where the change lands. Sentry tools are deferred: `ToolSearch select:mcp__claude_ai_Sentry__get_sentry_resource,mcp__claude_ai_Sentry__search_errors` first, then call. Never claim Sentry "not authorized" without a real tool call returning an auth error. Use it for event counts / more events if the ticket lacks detail. Read-only — no code changes.
3. Write `PLAN.md` in worktree root:
   - **Requirements** — non-Sentry tickets only, first: summary from step 3
   - **Belongs in Sentry?** — Sentry tickets only, ALWAYS first, mandatory, yes/no + evidence. "No" = not our bug: browser extension, bot/crawler, user offline / cancelled request, ChunkLoadError after deploy, third-party script, unsupported browser, expected 4xx, dev/localhost env. Then the fix plan is a client-side filter at the Sentry init — `app/shared/services/SentryLogger.service.ts` (cf-farm-frontend) or `src/common/services/logger.ts` (can-frontend): `ignoreErrors`, `denyUrls` or `beforeSend`. Read the existing rules first; if one should already match, explain why it slipped through and fix that rule instead of adding another.
   - **Root cause** — bugs only: hypothesis + evidence (file:line, frames, git history). Say plainly if unconfirmed.
   - **Plan** — stepped, files to touch, tests to add
   - `[?]` / `[!]` checklist from step 4
3a. Write holdout scenarios to `~/.local/state/cf-triage/holdout/<KEY>.md` (absolute path, outside the worktree). An independent verifier runs them in a browser against the MR's preview after the MR exists; the implementer can read but never edit them. Format, 3–8 scenarios:
   ```
   ## S1: <name>
   Given: <route, farm/entity, state>   (user fe-automation@cleverfarm.cz, farm "Cucumber Automation s.r.o."; data per cf-farm-frontend tests/TEST-DATA-AUDIT.md)
   When: <user actions>
   Then: <observable outcome in the UI>
   ```
   Derive them from the ticket's acceptance criteria / the bug's repro, not from the planned code. Each must be checkable by clicking through the UI. Change not observable in the UI (Sentry filter, pure refactor, BE change no screen shows) → file contains one line `NONE: <reason>`.
4. Post ONE Czech Jira comment via `jira add-comment <ticket-id> "<text>"` (bare `jira`, matches the permission allowlist) in the user's voice (`~/.claude/my-voice.md`): root cause (bugs), plan summary, scenarios (Given/When/Then, so the human approves them with `ai-fix`), `[?]` questions. First line EXACTLY `🤖 _AI triage (Claude, headless). Píšu za Michala. Jestli se vám to nelíbí, dejte mi prosím vědět._` (triage-worker detects replies by it). Jira wiki markup, not markdown: `h3.` headings, `*` bullets, `#` numbered, `{{code}}` inline, `{code}` blocks.

### Follow-up (headless, `PLAN.md` exists + `TICKET.md` has comments after the last `_AI triage` one)

A human replied. Don't redo the analysis from scratch:

1. Read `PLAN.md` + the new comments (after the last AI triage comment) in `TICKET.md`.
2. Apply answers / corrections: resolve `[?]` → `[!]` with the decision, adjust root cause / fix plan if challenged. Re-check code if needed.
3. Update `PLAN.md` in place, and the holdout scenarios file if replies change behaviour or acceptance.
4. Post ONE reply comment, same format + 🤖 prefix + marker: what changed in the plan, remaining `[?]` if any. Nothing to change → short confirmation.
