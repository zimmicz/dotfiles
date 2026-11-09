---
name: cf-ticket
description: |
  Full ticket lifecycle, interactive. Use when user says "start ticket", "work on
  ticket", "cf-ticket", or provides a ticket ID (e.g. DEV-XXXX) to work on
  end-to-end. Two checkpoints only (plan + holdout scenarios approval, draft MR
  verified on preview); `--step` adds a stop after every plan step. Unattended
  factory runs (factory-worker, label ai-fix) use /cf-ticket-headless instead.
---

# Full ticket lifecycle

Ticket: $ARGUMENTS

Orchestrates the entire workflow for a ticket. Two checkpoints only: **plan approval** (end of phase 2) and **MR ready** (end of phase 6). Everything between runs without stopping, except the stop rules below. `--step` in arguments → also stop after every plan step for validation (risky multi-spot work).

## Phase 0: Goal

1. Run `/mattpocock-skills:grill-with-docs` (grilling + domain-modeling: challenges terms against the repo's `GLOSSARY.md`, writes resolved terms into it inline, `_CZ_` line per the repo's `docs/agents/domain.md`; no `GLOSSARY.md` in the repo → behaves as plain grilling, don't create one) — focus the grilling on: what is the actual goal, what problem does this solve, what does success look like, what are the constraints. Fold the setup questions (base branch, phase 1) into the same rounds. No separate confirmation stop — the agreed goal goes into the plan.

## Phase 1: Setup

1. If no ticket ID in arguments, ask for one
2. Check if ticket has a parent/blocking ticket that implies a non-develop base branch. If so, ask "Base on `<parent-branch>`?" as a grilling question. Also check for sibling branches off the same parent to avoid overlap.
3. If already inside a worktree with `TICKET.md` (the `cf-work` script created it), skip to Phase 2. Otherwise run `/cf-worktree` for the ticket (thin wrapper over the same `~/bin/cf-work` script), passing the base branch if identified in step 2.
3a. **Multi-repo**: `TICKET.md` `## Repos` lists more than one worktree → the session was launched with `--add-dir` for the siblings, all are on the same branch. Treat the set as one change: plan steps, verify (phase 4) and review (phase 5) per repo, in each repo's own worktree dir. BE contract first (API/DTO), FE consumes it.

## Phase 2: Requirements + plan → CHECKPOINT 1

4. Run `/cf-work` — its `[?]` items are answered in the same conversation, not a separate gate
5. If `PLAN.md` exists in worktree root, read it first — it may contain a prior plan or user notes. Incorporate, don't overwrite blindly. If it is already a confirmed full plan (goal, decisions, steps) and the session was just cleared/restarted for implementation, skip 6–9 and go to Phase 3 with `PLAN.md` as the plan.
6. Call `EnterPlanMode`, then create a detailed stepped plan of implementation, including holdout scenarios (format + rules as in `/cf-work-headless` step 4a) and, in a glossary repo, `## Terms` (as in `/cf-work-headless` step 4; implemented per `/cf-ticket-headless` step 4a). Session runs on `opusplan`: Opus plans, Sonnet implements.
7. **CHECKPOINT 1** — `ExitPlanMode` with the full plan text + scenarios is the approval gate. User picks "clear context and implement" → Sonnet continues with the approved plan text.
8. Right after approval (plan mode blocks edits, so not before), write the confirmed plan to `PLAN.md` in worktree root **verbatim, not a summary**: goal, every grilling/`[?]`/`[!]` decision (with concrete numbers, rules, test ids), every step with its detail, verify + eyeball URLs. Fold in answers given after the plan was shown. `PLAN.md` is the single source for implementation. Write the scenarios to `~/.local/state/sw-factory/holdout/<KEY>.md`; from here on never edit that file. Then attach them to the ticket: `jira attach <KEY> ~/.local/state/sw-factory/holdout/<KEY>.md holdout.md` (and `<KEY>.api.md` as `holdout.api.md` if it exists); it replaces an older attachment of the same name.
9. Post ONE Czech Jira comment (via `~/bin/jira add-comment <ticket-id>`) that is a **Czech rendering of `PLAN.md`** + scenarios: same sections, same decisions, same steps, one-to-one — nothing added, dropped or merged. Before posting, diff comment vs `PLAN.md` item by item and fix gaps. Jira wiki markup (`h3.`, `#`, `*`, `{{code}}`). First line = CZ 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude". This is the only Jira comment in the flow. If `PLAN.md` changes later, edit that comment (`jira PUT /rest/api/2/issue/<id>/comment/<commentId>`), don't post a new one.

## Phase 3: Implementation

10. Implement all plan steps (with `--step`: stop after each, validate with the user). Holdout scenarios = acceptance criteria.

## Phase 4: Verify

11. Run tests, lint, typecheck (`nr test`, `nr lint`, `nr typecheck` or project equivalents). Fix failures.
11a. If diff touches UI files (`*.tsx`, `*.css`, stories), run `/check-layout` — screenshots at desktop + mobile, all states. UI is not verified until this passes. The verdict table and screenshot paths go to `/cf-mr` (`## MR body`, Evidence); if something couldn't be checked, say so there with the exact error of the failed attempt.
11b. New or changed E2E scenarios → run them locally (repo's E2E command, filtered to those scenarios) with `CI=true TARGET=http://localhost:<this worktree's dev server port>`. E2E logs in by itself from `.env` — no logged-in browser needed. Skip only with the failing command + output quoted in the MR description; "headless" / "will run in CI/preview" is never a reason.

## Phase 5: Independent review

12. Run `/code-review` on the diff — correctness bugs
13. Run `/mattpocock-skills:code-review` with fixed point `origin/<base>`, spec = `PLAN.md`, standards = repo `CLAUDE.md` "Coding Rules" + `~/dotfiles/claude/cleverfarm/standards/common.md` + `~/dotfiles/claude/cleverfarm/standards/<repo dir>.md` if it exists + the repo's `GLOSSARY.md` if it exists. Its Standards + Spec sub-agents see only the diff and those files, not your reasoning — don't pass them your own justification.
14. Fix issues found from both passes, re-run phase 4.

## Phase 6: MR → CHECKPOINT 2

15. Run `/cf-mr --headless` — draft MR, no reviewers, FE preview on Nightly.
16. Holdout verify: wait for the preview (every repo's latest MR pipeline green; FE-only also the bot note `<!-- amplify-preview-url -->` showing `@ <its short sha>`), then run `/cf-holdout-ui <KEY> repos=<repo>:<project id>:<mr iid>:<pipeline id>:<short sha>,… branch=…` in a fresh `Agent` sub-agent (pass only those args, no context from this session). `holdout/<KEY>.api.md` exists (BE refactor) → after it passes, same for `/cf-holdout-api <KEY> repos=… branch=…`. Fail → fix once (same rules as headless `--verify-fix`), push, verify again.
16a. Rewrite `~/.local/state/sw-factory/notes/<KEY>.md` as in `/cf-ticket-headless` step 9 (change map, decisions, deviations, gotchas, rounds), so a later factory round, or you after a break, starts from what was actually built.
17. **CHECKPOINT 2** — MR link, review findings fixed, verify verdict, anything deviating from the plan. Only on the user's explicit OK: `glab mr update <iid> --ready` + reviewers per `/cf-mr` step 6, ticket to review per `/cf-mr` step 8, then `/cf-mr` steps 11–12 (time log, retro). Then: "Come back after merge and run `/cf-feedback`."

## Stop rules (between checkpoints)

Stop and ask only when: the plan / root cause doesn't hold in the code, the change needs scope beyond the plan, phase 4 can't go green, review or verify findings need a design change, or an outward/destructive action the plan doesn't cover.

## Rules

- Never skip a checkpoint
- If user wants to skip a phase (e.g. already has worktree), jump to the right phase
- Keep context concise between phases — summarize, don't repeat everything
- Unattended factory runs (factory-worker): `/cf-ticket-headless`, nothing here applies to them
