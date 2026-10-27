---
name: cf-ticket
description: |
  Full ticket lifecycle. Use when user says "start ticket", "work on ticket",
  "cf-ticket", or provides a ticket ID (e.g. DEV-XXXX) to work on end-to-end.
  Interactive: two checkpoints only (plan + holdout scenarios approval, draft MR
  verified on preview); `--step` adds a stop after every plan step.
  With `--headless` (triage-worker, ticket labelled ai-fix): implements PLAN.md
  unattended, verifies (tests, lint, typecheck, visual check via /verify-ui for UI
  changes), independent review vs standards + PLAN.md, opens a draft MR without reviewers. On a ticket labelled
  ai-mr: answers new human Jira comments by pushing fixes to the existing MR.
  With `--verify-fix <report>`: fixes failed holdout scenarios on the existing MR.
  Plan says already fixed elsewhere: links Duplicate, moves ticket to Done, no MR.
---

# Full ticket lifecycle

Ticket: $ARGUMENTS

Orchestrates the entire workflow for a ticket. Two checkpoints only: **plan approval** (end of phase 2) and **MR ready** (end of phase 6). Everything between runs without stopping, except the stop rules below. `--step` in arguments → also stop after every plan step for validation (risky multi-spot work).

## Phase 0: Goal

1. Run `/mattpocock-skills:grilling` — focus the grilling on: what is the actual goal, what problem does this solve, what does success look like, what are the constraints. Fold the setup questions (base branch, phase 1) into the same rounds. No separate confirmation stop — the agreed goal goes into the plan.

## Phase 1: Setup

1. If no ticket ID in arguments, ask for one
2. Check if ticket has a parent/blocking ticket that implies a non-develop base branch. If so, ask "Base on `<parent-branch>`?" as a grilling question. Also check for sibling branches off the same parent to avoid overlap.
3. If already inside a worktree with `TICKET.md` (the `cf-work` script created it), skip to Phase 2. Otherwise run `/cf-worktree` for the ticket (thin wrapper over the same `~/bin/cf-work` script), passing the base branch if identified in step 2.
3a. **Multi-repo**: `TICKET.md` `## Repos` lists more than one worktree → the session was launched with `--add-dir` for the siblings, all are on the same branch. Treat the set as one change: plan steps, verify (phase 4) and review (phase 5) per repo, in each repo's own worktree dir. BE contract first (API/DTO), FE consumes it.

## Phase 2: Requirements + plan → CHECKPOINT 1

4. Run `/cf-work` — its `[?]` items are answered in the same conversation, not a separate gate
5. If `PLAN.md` exists in worktree root, read it first — it may contain a prior plan or user notes. Incorporate, don't overwrite blindly. If it is already a confirmed full plan (goal, decisions, steps) and the session was just cleared/restarted for implementation, skip 6–9 and go to Phase 3 with `PLAN.md` as the plan.
6. Call `EnterPlanMode`, then create a detailed stepped plan of implementation, including holdout scenarios (format + rules as in `/cf-work` headless step 3a). Session runs on `opusplan`: Opus plans, Sonnet implements.
7. **CHECKPOINT 1** — `ExitPlanMode` with the full plan text + scenarios is the approval gate. User picks "clear context and implement" → Sonnet continues with the approved plan text.
8. Right after approval (plan mode blocks edits, so not before), write the confirmed plan to `PLAN.md` in worktree root **verbatim, not a summary**: goal, every grilling/`[?]`/`[!]` decision (with concrete numbers, rules, test ids), every step with its detail, verify + eyeball URLs. Fold in answers given after the plan was shown. `PLAN.md` is the single source for implementation. Write the scenarios to `~/.local/state/cf-triage/holdout/<KEY>.md`; from here on never edit that file.
9. Post ONE Czech Jira comment (via `~/bin/jira add-comment <ticket-id>`) that is a **Czech rendering of `PLAN.md`** + scenarios: same sections, same decisions, same steps, one-to-one — nothing added, dropped or merged. Before posting, diff comment vs `PLAN.md` item by item and fix gaps. Jira wiki markup (`h3.`, `#`, `*`, `{{code}}`). First line = CZ 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude". This is the only Jira comment in the flow. If `PLAN.md` changes later, edit that comment (`jira PUT /rest/api/2/issue/<id>/comment/<commentId>`), don't post a new one.

## Phase 3: Implementation

10. Implement all plan steps (with `--step`: stop after each, validate with the user). Holdout scenarios = acceptance criteria.

## Phase 4: Verify

11. Run tests, lint, typecheck (`nr test`, `nr lint`, `nr typecheck` or project equivalents). Fix failures.
11a. If diff touches UI files (`*.tsx`, `*.css`, stories), run `/verify-ui` — screenshots at desktop + mobile, all states. UI is not verified until this passes. Include the verdict table in the MR description; if something couldn't be checked (auth wall), say so there.

## Phase 5: Independent review

12. Run `/code-review` on the diff — correctness bugs
13. Run `/mattpocock-skills:code-review` with fixed point `origin/<base>`, spec = `PLAN.md`, standards = repo `CLAUDE.md` "Coding Rules" + `~/dotfiles/claude/cleverfarm/standards/common.md` + `~/dotfiles/claude/cleverfarm/standards/<repo dir>.md` if it exists. Its Standards + Spec sub-agents see only the diff and those files, not your reasoning — don't pass them your own justification.
14. Fix issues found from both passes, re-run phase 4.

## Phase 6: MR → CHECKPOINT 2

15. Run `/cf-mr --headless` — draft MR, no reviewers, FE preview on Nightly.
16. Holdout verify: wait for the preview (every repo's latest MR pipeline green; FE-only also the bot note `<!-- amplify-preview-url -->` showing `@ <its short sha>`), then run `/cf-verify <KEY> repos=<repo>:<project id>:<mr iid>:<pipeline id>:<short sha>,… branch=…` in a fresh `Agent` sub-agent (pass only those args, no context from this session). Fail → fix once (same rules as headless `--verify-fix`), push, verify again.
17. **CHECKPOINT 2** — MR link, review findings fixed, verify verdict, anything deviating from the plan. Only on the user's explicit OK: `glab mr update <iid> --ready` + reviewers per `/cf-mr` step 6, ticket to review per `/cf-mr` step 8, then `/cf-mr` steps 11–12 (time log, retro). Then: "Come back after merge and run `/cf-feedback`."

## Stop rules (between checkpoints)

Stop and ask only when: the plan / root cause doesn't hold in the code, the change needs scope beyond the plan, phase 4 can't go green, review or verify findings need a design change, or an outward/destructive action the plan doesn't cover.

## Rules

- Never skip a checkpoint
- If user wants to skip a phase (e.g. already has worktree), jump to the right phase
- Keep context concise between phases — summarize, don't repeat everything

## Headless mode (`--headless` in arguments)

No user present — never ask, never wait, no checkpoints. The user approved `PLAN.md` by adding label `ai-fix`.

1. Phases 0–1: skip. Goal = `TICKET.md` + `PLAN.md`. Worktree already exists.
2. Phase 2: no new plan, no Jira comment. `PLAN.md` is the plan; decisions in `TICKET.md` comments after it win. `~/.local/state/cf-triage/holdout/<KEY>.md` = acceptance scenarios an independent verifier runs against the preview after the MR exists. Read them, make the change satisfy them; never edit that file.
3. Phase 3: implement all plan steps in one go. Stay strictly in plan scope.
4. Phase 4: tests, lint, typecheck must all pass. Fix your own failures; pre-existing failures unrelated to the change → note them, don't fix. UI diff → `/verify-ui` against Storybook (Playwright, headless); routes needing auth can't be checked — list them in the MR description and Jira comment as unverified.
5. Phase 5: steps 12–14 on the diff.
6. Phase 6: `/cf-mr --headless`. No holdout verify here — triage-worker runs it once the preview is up.
7. Post ONE Jira comment (`jira add-comment`, Czech, my voice, Jira wiki markup, first line EXACTLY `🤖 _AI triage (Claude, headless). Píšu za Michala. Jestli se vám to nelíbí, dejte mi prosím vědět._`): MR link, what changed, how to verify (preview/screen URL for FE), anything deviating from the plan. If the ticket has `sentry-*` labels: those Sentry issues resolve automatically once the ticket is Done (list them by short id).

**No code change** — `PLAN.md` concludes the bug is already fixed elsewhere (duplicate of ticket `<FIX>`) and you verified the fix in code (commit in a released tag): skip steps 3–6. Link `jira POST /rest/api/3/issueLink '{"type":{"name":"Duplicate"},"inwardIssue":{"key":"<KEY>"},"outwardIssue":{"key":"<FIX>"}}'`, then move to Done (transition named `Done` from `jira GET /rest/api/3/issue/<KEY>/transitions`). Post ONE Jira comment, same first line as step 7: evidence (commit, tag), closed as duplicate of `<FIX>`, Sentry resolves automatically (if `sentry-*` labels). Not a bail out.

**MR follow-up** — MR for this branch already exists (ticket labelled `ai-mr`, triage-worker re-runs on a human Jira comment newer than the last 🤖 one): skip steps 3–6 above. Read every human comment in `TICKET.md` after the last 🤖 comment, plus unresolved MR threads (`glab api`). Apply what they ask on the existing branch (scope = those comments), run phases 4–5 again, commit and push to the same branch (no new MR, no force-push). Then post ONE Jira comment, same first line as step 7: what changed per comment, or why not.

**Verify fix** — `--verify-fix <report>` in arguments (MR exists, the holdout verifier failed): skip steps 3–6 above. Read the report (failed scenarios, observed vs expected, screenshots) and the scenarios file. Fix the code on the existing branch so the failed scenarios pass — never edit the scenarios. Run phases 4–5 again, commit and push to the same branch (no new MR, no force-push). Post ONE Jira comment, same first line as step 7: which scenarios failed, what you changed. Scenario itself looks wrong (contradicts `PLAN.md` / ticket) → no code change, say so in the comment.

**Bail out** — no MR, just the Jira comment explaining why — when: root cause in `PLAN.md` doesn't hold in the code, the fix needs scope beyond the plan, or phase 4 can't go green. Never push to develop/staging/production, never force-push, never merge or approve.
