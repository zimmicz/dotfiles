---
name: cf-epic
description: |
  Epic-sized work, too big for one ticket session: wayfind decisions (local map),
  spec, slice into Jira child tickets with blocking links + holdout scenarios,
  build all slices into ONE draft acceptance MR via parallel worktree sub-agents,
  verify on preview, then after acceptance cut stacked review MRs per ticket.
  Use when user says "cf-epic", "epic", "big feature", or gives an epic key.
---

# Epic → acceptance MR → stacked review MRs

Epic: $ARGUMENTS (Jira epic key; none → ask, or create one after phase 1 with the user's OK)

State dir: `~/.local/state/cf-epics/<EPIC>/` (`map.md`, `issues/`, `SPEC.md`, `tickets.md`). Holdout scenarios: `~/.local/state/cf-triage/holdout/<KEY>.md`, same format and rules as `/cf-work` headless step 3a.

Three checkpoints: **spec**, **ticket breakdown**, **acceptance MR**. Between them, run without stopping except on `/cf-ticket` stop rules.

## Phase 1: Wayfind (skip if the way is already clear)

1. Read the epic (description, comments, linked issues) via `jira GET /rest/api/2/issue/<EPIC>`.
2. Run `/mattpocock-skills:wayfinder` with the **local** tracker: map = `<state dir>/map.md`, tickets = `<state dir>/issues/NN-<slug>.md`, ops as in the plugin's `setup-matt-pocock-skills/issue-tracker-local.md` "Wayfinding operations". Decision tickets stay out of Jira (DEV is company-wide). One ticket per session, as wayfinder says; the user re-invokes `/cf-epic <EPIC>` to continue — load `map.md` first and resume.
3. Map has no open tickets and no fog → phase 2.

## Phase 2: Spec → CHECKPOINT 1

4. `/mattpocock-skills:to-spec` from the map's Decisions-so-far + conversation, but write it to `<state dir>/SPEC.md` instead of publishing an issue.
5. **CHECKPOINT 1** — user approves `SPEC.md`. Then post ONE Czech Jira comment on the epic: spec summary + link-free decisions list, first line = CZ 🤖 prefix from `~/.claude/my-voice.md`.

## Phase 3: Tickets → CHECKPOINT 2

6. `/mattpocock-skills:to-tickets` on `SPEC.md`. Vertical slices, prefactors first, blocking edges. For each slice also draft its holdout scenarios.
7. **CHECKPOINT 2** — to-tickets' quiz: titles, blocked-by, what it delivers, scenarios. Iterate until approved.
8. Publish in dependency order via `~/bin/jira` (not MCP), English summaries: DEV issues with `parent` = `<EPIC>`, Product + Component per the repo's `docs/agents/issue-tracker.md`, labels `repo-<dir>` + `cf-epic`, assignee me, current sprint. Second pass: `Blocks` links (`jira POST /rest/api/3/issueLink '{"type":{"name":"Blocks"},"inwardIssue":{"key":"<BLOCKER>"},"outwardIssue":{"key":"<BLOCKED>"}}'`). Write each ticket's scenarios to its holdout file. Record keys + edges in `<state dir>/tickets.md`.

## Phase 4: Build the acceptance MR

9. `cf-work <EPIC> --no-launch` → acceptance worktree + branch `<EPIC>-<slug>`. Push it, open the draft MR now (`/cf-mr --headless`, title `feat: [<EPIC>] <epic summary>`), so the preview exists early.
10. Work the **frontier** (open tickets whose blockers are all merged into the acceptance branch), all frontier tickets in parallel: one `Agent` sub-agent per ticket with `isolation: "worktree"`, started from the acceptance branch HEAD. Brief = pointers only: ticket key, `SPEC.md`, its holdout file, `tickets.md`. Each implements its ticket per `/cf-ticket` headless phases 3–5 (scope = that ticket, no MR, no Jira comment), commits on its own branch, reports the branch.
11. Each finished slice: squash-merge into the acceptance branch as ONE commit `feat: [<KEY>] <summary>` (that commit = the future review MR), run phase 4 checks of `/cf-ticket` on the acceptance worktree, push. Conflicts → `/mattpocock-skills:resolving-merge-conflicts`. Recompute the frontier, start newly unblocked slices. Sub-agent bails → note it, keep going with unaffected slices.
12. All slices in: `/code-review` + `/mattpocock-skills:code-review` on the whole acceptance diff (spec = `SPEC.md`, standards as in `/cf-ticket` step 13), fix in one more commit per affected ticket (`fix: [<KEY>] …`), push.
13. Holdout verify: once the preview shows the latest pipeline sha, run `/cf-verify <KEY> …` in a fresh sub-agent **per child ticket** against the acceptance MR. Fail → one fix round, re-verify.

## Phase 5: Acceptance → CHECKPOINT 3

14. **CHECKPOINT 3** — acceptance MR + preview URL, per-ticket verify verdicts, bailed/deviating slices. Stakeholders accept on the preview. Never mark the acceptance MR ready — it is never merged.
15. After the user says accepted: cut stacked review MRs in dependency order. Per ticket: branch `<KEY>-<slug>` off the previous one (first off `develop`), cherry-pick its commits from the acceptance branch, push, `glab mr create --draft` targeting the previous branch, description lists the stack. Ask before marking any ready / adding reviewers. Close the acceptance MR only when the user says so.

## Rules

- Never push to develop/staging/production, never force-push shared branches, never merge or approve.
- Scope change mid-build → stop, update `SPEC.md` + affected tickets with the user, then continue.
