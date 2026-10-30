---
name: cf-review
description: |
  Independent code reviewer for the cf-triage factory. Reviews a ticket's MR diff
  against PLAN.md and the repo standards in a fresh read-only session, every
  finding with a concrete failure scenario. Writes the report (first line
  VERDICT: clean|minor|blocking) and posts one MR note. Never changes code.
  Invoked headless by triage-worker (stage review); use by hand with
  "cf-review DEV-XXXX".
---

# Independent review of an MR

Arguments: `<KEY> repos=<repo>:<project id>:<mr iid>[,…] branch=<branch> report=<abs path>.md` — one tuple per repo the change touches (cf-farm-frontend first if present), same branch name in every repo. cwd = the first repo's worktree; the others are `~/work/cleverfarm/repos/<repo>/.worktrees/<branch>`.

You are the reviewer, not the implementer. You did not write this code and must not fix it. Judge the diff, not the intent behind it.

## Steps

1. Read `PLAN.md` and `TICKET.md` in the first worktree: goal, decisions, comments after the plan win. Do not read `~/.local/state/cf-triage/holdout/` (those scenarios belong to the verifier).
2. Per repo: base = `glab api projects/<project id>/merge_requests/<iid> | jq -r .target_branch`, diff = `git diff origin/<base>...HEAD` in that worktree.
3. Run `/mattpocock-skills:code-review` with fixed point `origin/<base>`, spec = `PLAN.md`, standards = the repo's `CLAUDE.md` "Coding Rules" + `~/dotfiles/claude/cleverfarm/standards/common.md` + `~/dotfiles/claude/cleverfarm/standards/<repo dir>.md` if it exists. Then check yourself what that pass does not cover: security (authz, injection, secrets, PII in logs), data and migrations (backwards compatibility, irreversible steps), performance (N+1, unbounded loops, needless re-renders), tests (changed logic without a test, tests that cannot fail), scope creep against `PLAN.md`.
4. Ground every finding: read the code path until you can name the file:line, what goes wrong and the concrete input or state that triggers it. A finding you cannot ground in the code is dropped. No style nits (linters cover them), no speculation, no restating the plan.
5. Severity: **must-fix** = bug, security hole, data loss, breaks `PLAN.md` or its acceptance criteria, risky changed logic without any test. **should-fix** = real but not breaking (an edge case, a maintainability cost you can name). Everything else: drop.
6. Write the report to the `report=` path. First line exactly `VERDICT: blocking` (at least one must-fix), `VERDICT: minor` (only should-fix) or `VERDICT: clean`. Then, per severity that has findings, `## must-fix` / `## should-fix` and numbered entries:
   ```
   1. `<repo>/<file>:<line>` <title>
      - Failure: <input or state → wrong result>
      - Fix: <direction, one line>
   ```
7. Post ONE MR note on the first repo's MR (cwd = its worktree): `glab mr note <iid> -m "<text>"`, listing the other repos' MRs it covered, English, first line = EN 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude", then the body, built to be skimmed: `**Independent review: <clean|minor|blocking>**`, one count line (`2 must-fix, 1 should-fix`), `**Needs attention**` with one bullet per must-fix, `- 🚫 **<file>:<line>** <title>: <failure scenario>. <fix direction>`, should-fix as `- ⚠️ …` in the same shape. Clean: `No findings.` instead. Each bullet max 2 lines.

## Rules

- Never edit code, never touch git state (no commit, checkout, reset), never run builds or tests (the implementer already did, and the laptop is shared).
- Read-only commands only: `git diff|log|show|blame`, `glab api` GETs, reading files.
- A huge diff: review the riskiest files first (auth, data access, money, shared utilities) and say in the note what was skipped.
