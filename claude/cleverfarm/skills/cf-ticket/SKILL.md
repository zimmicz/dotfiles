---
name: cf-ticket
description: |
  Full ticket lifecycle. Use when user says "start ticket", "work on ticket",
  "cf-ticket", or provides a ticket ID (e.g. DEV-XXXX) to work on end-to-end.
  With `--headless` (triage-worker, ticket labelled ai-fix): implements PLAN.md
  unattended, verifies (tests, lint, typecheck, visual check via /verify-ui for UI
  changes), self-reviews, opens a draft MR without reviewers. On a ticket labelled
  ai-mr: answers new human Jira comments by pushing fixes to the existing MR.
  Plan says already fixed elsewhere: links Duplicate, moves ticket to Done, no MR.
---

# Full ticket lifecycle

Ticket: $ARGUMENTS

Orchestrates the entire workflow for a ticket. Each phase has a STOP gate — wait for user confirmation before proceeding.

## Phase 0: Goal

1. Run `/mattpocock-skills:grilling` — focus the grilling on: what is the actual goal, what problem does this solve, what does success look like, what are the constraints
2. **STOP** — summarize the agreed goal in 2-3 sentences, confirm with user before continuing

## Phase 1: Setup

1. If no ticket ID in arguments, ask for one
2. Check if ticket has a parent/blocking ticket that implies a non-develop base branch. If so, surface it: "Base on `<parent-branch>`?" Also check for sibling branches off the same parent to avoid overlap.
3. If already inside a worktree with `TICKET.md` (the `cf-work` script created it), skip to Phase 2. Otherwise run `/cf-worktree` for the ticket (thin wrapper over the same `~/bin/cf-work` script), passing the base branch if identified in step 2.
3a. **Multi-repo**: `TICKET.md` `## Repos` lists more than one worktree → the session was launched with `--add-dir` for the siblings, all are on the same branch. Treat the set as one change: plan steps, verify (phase 4) and self-review (phase 5) per repo, in each repo's own worktree dir. BE contract first (API/DTO), FE consumes it.
4. **STOP** — confirm worktree ready

## Phase 2: Requirements

4. Run `/cf-work`
5. **STOP** — wait for answers to `[?]` items, confirmation of `[!]` items
6. If `PLAN.md` exists in worktree root, read it first — it may contain a prior plan or user notes. Incorporate, don't overwrite blindly. If it is already a confirmed full plan (goal, decisions, steps) and the session was just cleared/restarted for implementation, skip 7–9 and go to Phase 3 with `PLAN.md` as the plan.
7. Call `EnterPlanMode`, then create a detailed stepped plan of implementation. Session runs on `opusplan`: Opus plans, Sonnet implements.
8. **STOP** — `ExitPlanMode` with the full plan text is the approval gate. User picks "clear context and implement" → Sonnet continues with the approved plan text.
9. Right after approval (plan mode blocks edits, so not before), write the confirmed plan to `PLAN.md` in worktree root **verbatim, not a summary**: goal, every grilling/`[?]`/`[!]` decision (with concrete numbers, rules, test ids), every step with its detail, verify + eyeball URLs. Fold in answers given after the plan was shown. `PLAN.md` is the single source for implementation.
10. Post ONE Czech Jira comment (via `~/bin/jira add-comment <ticket-id>`) that is a **Czech rendering of `PLAN.md`**: same sections, same decisions, same steps, one-to-one — nothing added, dropped or merged. Before posting, diff comment vs `PLAN.md` item by item and fix gaps. Jira wiki markup (`h3.`, `#`, `*`, `{{code}}`). First line = CZ 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude". This is the only Jira comment in the flow. If `PLAN.md` changes later, edit that comment (`jira PUT /rest/api/2/issue/<id>/comment/<commentId>`), don't post a new one.

## Phase 3: Implementation

11. User says "go" or gives direction, or plan was just approved via ExitPlanMode — start coding
12. Stop after every plan step, validate with the user
13. When implementation complete, **STOP** — ask "Ready to verify?"

## Phase 4: Verify

14. Run tests, lint, typecheck (`nr test`, `nr lint`, `nr typecheck` or project equivalents). Fix failures.
14a. If diff touches UI files (`*.tsx`, `*.css`, stories), run `/verify-ui` — screenshots at desktop + mobile, all states. UI is not verified until this passes. Include the verdict table in the MR description; if something couldn't be checked (auth wall), say so there.
15. **STOP** if any test/lint failures or visual fixes required non-trivial changes — show what changed

## Phase 5: Self-review

16. Run `/code-review` on changed files — correctness bugs
17. Run `/simplify` on changed files — cleanup/reuse
18. Fix issues found from both passes
19. **STOP** — ask "Ready to create MR?"

## Phase 6: MR

20. Run `/cf-mr`
21. **STOP** — "MR created. Go get it reviewed. Come back after merge and run `/cf-feedback`."

## Rules

- Never skip a STOP gate
- If user wants to skip a phase (e.g. already has worktree), jump to the right phase
- Keep context concise between phases — summarize, don't repeat everything

## Headless mode (`--headless` in arguments)

No user present — never ask, never wait, no STOP gates. The user approved `PLAN.md` by adding label `ai-fix`.

1. Phases 0–1: skip. Goal = `TICKET.md` + `PLAN.md`. Worktree already exists.
2. Phase 2: no new plan, no Jira comment. `PLAN.md` is the plan; decisions in `TICKET.md` comments after it win.
3. Phase 3: implement all plan steps in one go. Stay strictly in plan scope.
4. Phase 4: tests, lint, typecheck must all pass. Fix your own failures; pre-existing failures unrelated to the change → note them, don't fix. UI diff → `/verify-ui` against Storybook (Playwright, headless); routes needing auth can't be checked — list them in the MR description and Jira comment as unverified.
5. Phase 5: `/code-review` + `/simplify` on the diff, apply fixes, re-run phase 4.
6. Phase 6: `/cf-mr --headless`.
7. Post ONE Jira comment (`jira add-comment`, Czech, my voice, Jira wiki markup, first line EXACTLY `🤖 _AI triage (Claude, headless). Píšu za Michala. Jestli se vám to nelíbí, dejte mi prosím vědět._`): MR link, what changed, how to verify (preview/screen URL for FE), anything deviating from the plan. If the ticket has `sentry-*` labels: those Sentry issues resolve automatically once the ticket is Done (list them by short id).

**No code change** — `PLAN.md` concludes the bug is already fixed elsewhere (duplicate of ticket `<FIX>`) and you verified the fix in code (commit in a released tag): skip phases 3–6. Link `jira POST /rest/api/3/issueLink '{"type":{"name":"Duplicate"},"inwardIssue":{"key":"<KEY>"},"outwardIssue":{"key":"<FIX>"}}'`, then move to Done (transition named `Done` from `jira GET /rest/api/3/issue/<KEY>/transitions`). Post ONE Jira comment, same first line as step 7: evidence (commit, tag), closed as duplicate of `<FIX>`, Sentry resolves automatically (if `sentry-*` labels). Not a bail out.

**MR follow-up** — MR for this branch already exists (ticket labelled `ai-mr`, triage-worker re-runs on a human Jira comment newer than the last 🤖 one): skip phases 3–6 above. Read every human comment in `TICKET.md` after the last 🤖 comment, plus unresolved MR threads (`glab api`). Apply what they ask on the existing branch (scope = those comments), run phases 4–5 again, commit and push to the same branch (no new MR, no force-push). Then post ONE Jira comment, same first line as step 7: what changed per comment, or why not.

**Bail out** — no MR, just the Jira comment explaining why — when: root cause in `PLAN.md` doesn't hold in the code, the fix needs scope beyond the plan, or phase 4 can't go green. Never push to develop/staging/production, never force-push, never merge or approve.
