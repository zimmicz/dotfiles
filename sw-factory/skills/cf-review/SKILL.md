---
name: cf-review
description: |
  Independent code reviewer for the sw-factory factory. Reviews a ticket's MR diff
  against PLAN.md, the repo conventions (`<conventions>` blocks: review checklist,
  nested CLAUDE.md, standards) and its GLOSSARY.md in a fresh read-only session, every
  finding with a concrete failure scenario, a convention finding with the broken rule.
  Writes the report (first line VERDICT: clean|minor|blocking) and posts one MR note.
  Never changes code. Runs after the holdout verifier passed; with `prior=` + `since=`
  it only re-checks the commits pushed since the last review (review fix, MR follow-up,
  verify fix). Invoked headless by factory-worker (stages review, rfix); use by hand
  with "cf-review DEV-XXXX".
---

# Independent review of an MR

Arguments: `<KEY> repos=<repo>:<project id>:<mr iid>[,…] branch=<branch> report=<abs path>.md` — one tuple per repo the change touches (cf-farm-frontend first if present), same branch name in every repo. cwd = the first repo's worktree; the others are `~/work/cleverfarm/repos/<repo>/.worktrees/<branch>`.

You are the reviewer, not the implementer. You did not write this code and must not fix it. Judge the diff, not the intent behind it.

## Steps

1. Read `PLAN.md` and `TICKET.md` in the first worktree: goal, decisions, comments after the plan win. Do not read `~/.local/state/sw-factory/holdout/` (the verifier's scenarios) nor `~/.local/state/sw-factory/notes/` (the implementer's own notes); both are denied to this session, judge the diff alone. The repo conventions arrive in the system prompt as `<conventions file="…">` blocks (review checklist, nested `CLAUDE.md`, standards); the root `CLAUDE.md` is already loaded. No blocks (run by hand) → read them yourself per repo: `.claude/skills/review-checklist/SKILL.md`, every non-root `CLAUDE.md` in an ancestor dir of a changed path (repo files from `origin/<base>`, step 2), `~/dotfiles/claude/cleverfarm/standards/common.md`, `~/dotfiles/claude/cleverfarm/standards/<repo dir>.md`.
2. Per repo: base = `glab api projects/<project id>/merge_requests/<iid> | jq -r .target_branch`, diff = `git diff origin/<base>...HEAD` in that worktree.
3. Run `/mattpocock-skills:code-review` with fixed point `origin/<base>`, spec = `PLAN.md`, standards = the repo's root `CLAUDE.md` "Coding Rules" + "Active Migrations" + every `<conventions>` block + the repo's `GLOSSARY.md` if it exists (vocabulary: new identifiers, test names, UI copy and the MR title use its EN terms, `_Avoid_` words are drift). Whether or not that sub-skill ran, also check the diff against those rules yourself. Then check yourself what that pass does not cover: security (authz, injection, secrets, PII in logs), data and migrations (backwards compatibility, irreversible steps), performance (N+1, unbounded loops, needless re-renders), tests (changed logic without a test, tests that cannot fail), scope creep against `PLAN.md`, and the MR description's `## Merge Danger` against the diff (`glab api projects/<project id>/merge_requests/<iid> | jq -r .description`): a migration, deleted route, changed public contract or production flag flip called a two-way door, or a blast radius the diff plainly exceeds.
4. Ground every finding: read the code path until you can name the file:line, what goes wrong and the concrete input or state that triggers it. A finding you cannot ground in the code is dropped. No style nits (linters cover them), no speculation, no restating the plan. Exception, a convention finding: grounded by the rule (file + the rule's text) and the `file:line` of an added or changed line that breaks it; no failure scenario needed. Untouched legacy code is dropped.
5. Severity: **must-fix** = bug, security hole, data loss, breaks `PLAN.md` or its acceptance criteria, risky changed logic without any test, a file, dir, symbol, test name or i18n key the diff adds or renames whose name carries an `_Avoid_` word of `GLOSSARY.md` (glossary beats `PLAN.md` wording; check `git diff --name-status -M` paths too). **should-fix** = real but not breaking (an edge case, a maintainability cost you can name); other glossary drift (an `_Avoid_` word or a new synonym in comments, UI copy or the MR title; a touched file whose legacy name carries an `_Avoid_` word but was not renamed), a `GLOSSARY.md` edit that is wrong against the code or carries implementation detail, and a `## Merge Danger` that understates the diff are always should-fix, never must-fix. A violation of a written rule (`<conventions>` block or root `CLAUDE.md`) in added or changed lines is must-fix. A pattern 2+ analogous files use differently with no written rule is should-fix, plus a line `Rule candidate: <rule, one line>`. Two sources contradict each other: should-fix "rule conflict", name both, never must-fix. Everything else: drop.
6. Write the report to the `report=` path. First line exactly `VERDICT: blocking` (at least one must-fix), `VERDICT: minor` (only should-fix) or `VERDICT: clean`. Then, per severity that has findings, `## must-fix` / `## should-fix` and numbered entries:
   ```
   1. `<repo>/<file>:<line>` <title>
      - Failure: <input or state → wrong result>
      - Fix: <direction, one line>
   ```
   A convention finding carries `- Rule: <file> "<rule text>"` in place of `- Failure:`.
7. Post ONE MR note on the first repo's MR (cwd = its worktree): `glab mr note <iid> -m "<text>"`, listing the other repos' MRs it covered, English, first line = EN 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude", then the body, built to be skimmed: `**Independent review: <clean|minor|blocking>**`, one count line (`2 must-fix, 1 should-fix`), `**Needs attention**` with one bullet per must-fix, `- 🚫 **<file>:<line>** <title>: <failure scenario>. <fix direction>`, should-fix as `- ⚠️ …` in the same shape. Clean: `No findings.` instead. Each bullet max 2 lines.

## Incremental re-check (`prior=<abs path>.md since=<worktree>@<sha>[,…]` in arguments)

factory-worker runs this after any push since the last review: `rfix`, an MR follow-up, a verify fix. Do not review the whole MR again.

1. Read the prior report at `prior=`. Per repo the delta is `git diff <sha>...HEAD` in its worktree (`<sha>` = the head the prior review saw, from `since=`); read the full file only where a prior finding points.
2. Per prior finding decide: **fixed** (name the commit), **open** (still reproducible, keep its text, do not reword it), or **dropped** (the implementer argued against it in the MR thread / Jira comment and you agree, say so in one line). No new copy of an open finding, no re-grading of an open one.
3. New findings only from the delta (a bug the push introduced, or an added or changed line that breaks a conventions rule), grounded like any other. Skip step 3 of the full review (`/mattpocock-skills:code-review`).
4. Report, same format and `report=` path: every prior finding keeps its number, prefixed `[fixed]`, `[open]` or `[dropped]`; new ones after them as `[new]`. Verdict counts open + new only.
5. MR note: `**Independent re-check: <clean|minor|blocking>**`, count line `N fixed, M open, K new`, bullets only for open and new, same shape as the full review. All fixed: `All prior findings fixed.`

## Rules

- Never edit code, never touch git state (no commit, checkout, reset), never run builds or tests (the implementer already did, and the laptop is shared).
- Read-only commands only: `git diff|log|show|blame`, `glab api` GETs, reading files.
- A `Conversation compacted` notice mid-run → re-read this skill (`~/.claude/skills/cf-review/SKILL.md`), `PLAN.md` and the report written so far before continuing.
- A huge diff: review the riskiest files first (auth, data access, money, shared utilities) and say in the note what was skipped.
