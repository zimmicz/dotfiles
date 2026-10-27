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
8. Publish in dependency order via `~/bin/jira` (not MCP), English summaries: DEV issues with `parent` = `<EPIC>`, Product + Component per the repo's `docs/agents/issue-tracker.md`, labels `repo-<dir>` for EVERY repo the slice touches (component BE/FE accordingly) + `cf-epic`, assignee me, current sprint. Second pass: `Blocks` links (`jira POST /rest/api/3/issueLink '{"type":{"name":"Blocks"},"inwardIssue":{"key":"<BLOCKER>"},"outwardIssue":{"key":"<BLOCKED>"}}'`). Write each ticket's scenarios to its holdout file. Record keys + edges in `<state dir>/tickets.md`.

## Phase 4: Build the acceptance MR

9. Epic carries a `repo-<dir>` label for every repo any ticket touches (add missing ones). `cf-work <EPIC> --no-launch` → one acceptance worktree per repo, same branch `<EPIC>-<slug>` everywhere (`TICKET.md` `## Repos` lists them). Run the session from the primary (cf-farm-frontend if present) with `--add-dir` for the others. Push every repo, open the draft MRs now (`/cf-mr --headless`, multi-repo rules: BE first, FE lists `Depends on: <BE MR>`, title `feat: [<EPIC>] <epic summary>`).
10. Work the **frontier** (open tickets whose blockers are all merged into the acceptance branches), all frontier tickets in parallel. Per ticket, for EACH repo in its `repo-*` labels, create a slice worktree yourself: `git -C <repo> worktree add <repo>/.worktrees/<KEY>-slice <EPIC branch> -b <KEY>-slice` (no `Agent` `isolation` — it only covers this session's repo). One `Agent` sub-agent per ticket, brief = pointers only: ticket key, its slice worktree paths, `SPEC.md`, its holdout file, `tickets.md`. It implements per `/cf-ticket` headless phases 3–5 across those worktrees (scope = that ticket, BE contract first, no MR, no Jira comment), commits in each, reports done.
11. Each finished slice, per repo it touched: squash-merge `<KEY>-slice` into that repo's acceptance branch as ONE commit `feat: [<KEY>] <summary>` (same message in every repo; that commit = the future review MR), run `/cf-ticket` phase 4 checks in that acceptance worktree, push, remove the slice worktree + branch. Conflicts → `/mattpocock-skills:resolving-merge-conflicts`. Recompute the frontier, start newly unblocked slices. Sub-agent bails → note it, keep going with unaffected slices.
12. All slices in: `/code-review` + `/mattpocock-skills:code-review` on each repo's acceptance diff (spec = `SPEC.md`, standards as in `/cf-ticket` step 13), fix in one more commit per affected ticket and repo (`fix: [<KEY>] …`), push.
13. Holdout verify, **per child ticket**, in a fresh sub-agent: `/cf-verify <KEY> repos=<tuples of the acceptance MRs in the repos that ticket touches> branch=<EPIC branch>` once those pipelines are green. Any BE repo involved → it runs on one dyn-env with all BE MR images + the FE acceptance branch; create that env once yourself (`cf-dyn-env:dyn-env`, name `<epic lowercased>-accept`, same request as `/cf-verify` step 3) and pass `env=<name>` to every verify; redeploy it after fix rounds. Fail → one fix round, re-verify.

## Phase 5: Acceptance → CHECKPOINT 3

14. **CHECKPOINT 3** — acceptance MRs + preview URL (Amplify, or the dyn-env FE when BE is involved — keep that env up for stakeholders until acceptance), per-ticket verify verdicts, bailed/deviating slices. Stakeholders accept on the preview. Never mark acceptance MRs ready — they are never merged.
15. After the user says accepted: cut stacked review MRs **per repo**, in dependency order. Per repo, per ticket that touched it: branch `<KEY>-<slug>` off that repo's previous stack branch (first off `develop`), cherry-pick its commits from that repo's acceptance branch, push, `glab mr create --draft` targeting the previous branch, description lists the stack. A ticket in several repos → its FE MR says `Depends on: <BE MR>` of the same ticket. Ask before marking any ready / adding reviewers. Close the acceptance MRs and destroy the dyn-env only when the user says so.

## Rules

- Never push to develop/staging/production, never force-push shared branches, never merge or approve.
- Scope change mid-build → stop, update `SPEC.md` + affected tickets with the user, then continue.
