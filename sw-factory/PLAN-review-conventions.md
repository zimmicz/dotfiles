# Plan: review that catches convention drift and covers pushed code

Handoff from the audit session of 2026-11-17 (Fable). Executor: a fresh session (sonnet is enough). Everything needed is in this file; the audit's scratch scripts are gone.

## Goal

1. Reviewer and implementer always have the repo's conventions in context.
2. A convention violation is a reportable, auto-fixable finding.
3. Code pushed after the review gets reviewed too.
4. The effect is measured before/after.

## Why (measured 2026-11-01 → 11-17, 57 tickets, from `~/.local/state/sw-factory/DEV-*.log` + transcripts)

- cf-farm-frontend convention sources and how often a run loaded them (7 FE review transcripts / 120 FE implement-type transcripts):
  - root `CLAUDE.md` (23 Coding Rules + 4 migrations): 7/7, 120/120 (auto-loaded, nothing to do)
  - `app/CLAUDE.md` (nested): 0/7, 31/120
  - `.claude/skills/review-checklist/SKILL.md` (216 lines, ~90 checks, the team's "living checklist"): 0/7, 0–3/120
  - `~/dotfiles/claude/cleverfarm/standards/cf-farm-frontend.md` (42 rules): 2/7, 0/120 in the main session
- Cause: nested `CLAUDE.md` load only on native `Read`; factory agents read through `ctx_read` / `git diff`. The checklist is a skill nobody invokes. The standards path sits inside cf-review step 3, which runs `/mattpocock-skills:code-review`; that skill was called in 7 of 34 reviews.
- `cf-review` steps 4–5 drop every finding without a concrete failure scenario ("no style nits", "everything else: drop"). Only glossary drift has a carve-out. Of 51 findings in the reports ~7 are convention-type, all tied to a written rule or PLAN.md.
- Example: DEV-3532 review suggested restoring `eslint-disable react-hooks/exhaustive-deps`; the repo checklist forbids exactly that. DEV-3584 / DEV-3586 (set-state-in-effect tickets): the standards file with that rule was not opened.
- Review runs once per MR. 10 of 40 reviewed tickets got pushes after the review (≥21 pushes), never reviewed. MR follow-ups: median 3 per ticket, p90 14.
- Verdicts of 34 reviews: 8 clean, 19 minor, 7 blocking. `minor` is never auto-fixed.
- Incremental re-check (`prior=` + `since=`) already exists: median $0.66, under 1 min.

## Rules for the executor

- One step at a time. After each step: show the diff + the step's check output, then wait for Michal's "go". No batching.
- `~/bin/factory-worker` and `~/.claude/skills/cf-*` are symlinks into the master checkout and the dispatcher runs every minute: an edit on master is live at once. Work in a git worktree (`~/dotfiles/.worktrees/review-conventions`, branch `review-conventions`); master gets the change only by merge, on Michal's go.
- Another session (`sw-factory-bb`) is replacing the factory-dash TUI and touches `sw-factory/README.md`, `sw-factory/map/index.html`, `sw-factory/bin/factory-stats`, `dash/`, `install.sh`. Before editing README or map: `git log` + `git status`, rebase the worktree onto master, message that session (ListAgents → SendMessage).
- Commit only on Michal's go. Never widen `--allowedTools`. No code comments. Line numbers below are as of commit `4506c63`; find the function by name.
- Shell is zsh. Jira via `~/bin/jira`, GitLab via `glab` with `</dev/null` inside loops.
- Any behaviour change → update `sw-factory/README.md` and `sw-factory/map/index.html`, republish the map: `Artifact publish file_path=<map file> url=https://claude.ai/artifact/Y7Gr4yyNVRyrs1X37Ss955`.

## Step 0: prep

- `git -C ~/dotfiles tag sw-factory-pre-review-conventions-<date>` on master HEAD.
- `git -C ~/dotfiles worktree add .worktrees/review-conventions -b review-conventions`.
- `claude --help`: confirm `--append-system-prompt` exists. Missing → use the flag that takes a file, or prepend the block to the prompt text; say which.
- README: add the rollback line `git -C ~/dotfiles reset --hard sw-factory-pre-review-conventions-<date>` next to a short "Rollback" note (none exists yet).
- Check: tag + worktree exist, flag confirmed.

## Step 1: worker injects conventions (`bin/factory-worker`)

New function `conventions <worktree> <repo dir> <base branch>`, prints one block per file:

```
<conventions file="<repo>/<path>">
…file content…
</conventions>
```

Sources, in this order, each skipped silently when missing:
1. `.claude/skills/review-checklist/SKILL.md`
2. every non-root `CLAUDE.md` in an ancestor dir of a changed path. Changed paths = `git diff --name-only origin/<base>...HEAD` plus the backticked paths in `PLAN.md` `### Change map` (first implement run has no diff yet)
3. `~/dotfiles/claude/cleverfarm/standards/common.md`
4. `~/dotfiles/claude/cleverfarm/standards/<repo dir>.md`

Details:
- Repo files (1, 2) are read from the base branch (`git -C <wt> show origin/<base>:<path>`), not the worktree, so an MR cannot weaken the rules it is judged by.
- Base branch: `review` → MR `target_branch` (`glab api projects/<pid>/merge_requests/<iid>`); `implement` → the MR's target if one exists, else the repo's default (`git symbolic-ref --short refs/remotes/origin/HEAD`).
- Multi-repo ticket: blocks for every repo in the targets.
- Linux caps one argument at 128 KB. Total over 100 KB → drop nested `CLAUDE.md` files deepest first, log what was dropped.
- Log to `<KEY>.log`: `  conventions: <n> files, <bytes> B: <paths>`.
- Wire into `review()` (line ~403) and `implement()` (line ~194, all variants) as `--append-system-prompt "$(conventions …)"`. Not into verify, analyze, conflict.
- Debug entry: `factory-worker DEV-X --conventions` prints the block and exits (same place as `--gate`, line ~686).
- Check: `zsh -n bin/factory-worker`; `--conventions` on an FE ticket with an open MR (Michal picks) lists the checklist, `app/CLAUDE.md`, both standards files; `TRIAGE_DRY=1 factory-worker` still prints its spawn list.

## Step 2: skills use them

`skills/cf-review/SKILL.md`:
- Step 1: conventions arrive in the system prompt as `<conventions file=…>` blocks; root `CLAUDE.md` is already loaded. No blocks (run by hand) → read the four sources of Step 1 yourself.
- Step 3: the standards list for `/mattpocock-skills:code-review` becomes "root `CLAUDE.md` Coding Rules + Active Migrations + every conventions block + `GLOSSARY.md`". Add: check the diff against those rules yourself too, whether or not the sub-skill ran.
- Step 4: a convention finding is grounded by the rule (file + the rule's text) and the `file:line` in the diff that breaks it. No failure scenario needed. Only added or changed lines count; untouched legacy code is dropped.
- Step 5: violation of a written rule in added/changed lines → must-fix. A pattern used differently by 2+ analogous files with no written rule → should-fix, plus a line `Rule candidate: <rule, one line>`. Two sources contradict each other → should-fix "rule conflict", name both, never must-fix.
- Step 6: convention entries carry `- Rule: <file> "<rule text>"` in place of `- Failure:`.
- Frontmatter description: mention conventions and the re-check after later pushes (Step 3 of this plan).

`skills/cf-ticket-headless/SKILL.md`:
- Inputs, new 2c: conventions blocks in the system prompt bind every added or changed line, in every variant. Step 6 standards = those blocks.

Keep both generic: no cf-farm-frontend paths in skill text.
- Check: both files read coherently end to end; no step refers to a path the worker does not provide.

## Step 3: re-review after later pushes (`bin/factory-worker` + cf-review)

- `stage_review` (line ~616): verdict clean or minor → write `wt_heads $KEY` to `$LOG_DIR/$KEY.reviewed`. At the start: that file and `$KEY.review.md` exist → run `review $KEY $2 "$(<…reviewed)"` (incremental), else full.
- `stage_rfix` (line ~632): after a clean/minor re-check also write `$KEY.reviewed`. Count rounds in `$LOG_DIR/$KEY.rfix-count`; third round for one MR → `ai-review-findings` instead of another fix.
- `unverify()` (line ~565): also remove `ai-reviewed`. Effect: follow-up or fix push → verify again → `ai-verified` → the existing REVIEW query picks the ticket → incremental review of the delta. The merge gate stays pending meanwhile, no new label.
- After every review, log `  findings: must=<n> should=<n> convention=<n>` (count numbered entries per heading in the report; convention = entries with a `- Rule:` line).
- Not covered, leave as is: agent-resolved rebase (line ~519). `git diff <old sha>...HEAD` after a rebase contains all of develop.
- cf-review "Incremental re-check": say it also runs after an MR follow-up or verify-fix push, not only after `rfix`.
- README: flow diagram lines about "ONCE per MR", label table rows `ai-reviewed` and `ai-verified`, the per-tick priority paragraph. Map + artifact.
- Check: `zsh -n`; on a ticket Michal picks: push a trivial commit through a follow-up, confirm in `<KEY>.log` verify → `review (incremental)` → `findings:` line → `ai-reviewed` back, gate pending in between (`factory-worker DEV-X --gate`).

## Step 4: measurement (`bin/factory-compare`)

It already compares per-stage medians before/after a cutover. Add rows, cutover = the merge commit time of this branch:
- reviews with a `conventions:` log line (%)
- findings per review: must / should / convention (from the `findings:` line; before the cutover: no data, print `-`)
- review runs per ticket
- MR follow-ups per ticket: median + p90 (baseline 3 / 14)
- cost per review run: median + p90 (baseline $1.17 / $2.54)

Check: `factory-compare` prints the new rows with the baseline column filled where logs allow.

## Step 5: ship

- Michal picks a real ticket; run Steps 1–3 checks on it from the worktree copy (`<worktree>/sw-factory/bin/factory-worker DEV-X --review`).
- Go → rebase on master, fast-forward master, `git tag sw-factory-review-conventions-<date>`.
- README + map + artifact republished in the same go.

## Backlog (audit findings not in this plan, with their numbers)

- Verify, 33 % of spend, 45 % pass. Stale holdout: 19 of 32 verify-fix rounds changed no code ("scenario outdated"); pass right after a fix 4 of 26. `blocked` is treated as failed (`factory-worker:602`): 36 of 134 runs, about half are single scenarios with missing fixture data, ~6 Playwright MCP `CONNECT_TIMEOUT`. 68 of 134 verify runs are full re-verifies after a follow-up push.
- Review never starts when verify ends fail/blocked (18 of 40 tickets); 21 merged MRs carry `ai-gate-skip`, 3 quiz records exist.
- Fix variants skip skill steps 5–6 (code-review skill in 0 of 34 verify-fix, 4 of 173 follow-ups). Step 3 of this plan covers the review part.
- Implement: outcome inferred from "MR exists"; 2 runs ended waiting for a background eslint/test job. No retry cap after a crash or timeout (DEV-3518 3 runs in 18 min).
- `factory-worker:546`: grill ASK on an already `triaged` ticket does not set `needs-answer`, `ai-fix` is not blocked.
- Labels (`ai-fix`, `triage`, `ai-gate-skip`, `ai-rebase`) are acted on whoever set them; comments are owner-filtered.
- Deny-lists are prefix patterns: `glab mr merge|approve` denied, `glab api:*` allowed; FE project has `merge_requests_author_approval: true`. Direct push to develop/staging/production is blocked server-side.
- `factory-stats`: counts review verdicts as verify runs (line 37), `implementing` never matches (line 13). `sw-factory-bb` is fixing these.
- Sources of truth: rules overlap between root `CLAUDE.md`, `review-checklist` and dotfiles `standards/`. Candidate: move team rules into the repo checklist via ticket + MR; `/cf-feedback` writes there and also reads Michal's own MR notes (today filtered out as "not yours").
- Doc mismatches: checklist §12 `npm run` vs `pnpm run` in `CLAUDE.md`; import rule in root `CLAUDE.md` vs `standards/cf-farm-frontend.md` (`../` inside a module); standards header names `/cf-ticket` phase 5; README grill text vs `factory-worker:546`.

## Open questions (defaults are what the plan assumes)

1. Written-rule violation = must-fix (auto-fixed by `rfix`)? Default yes. Cost: more `rfix` rounds.
2. Re-review after every push? Default yes. Est. up to ~$45 per 17 days at current follow-up volume.
3. `rfix` cap per MR? Default 2.
4. Move dotfiles `standards/<repo>.md` into the repo checklist? Default: later, own ticket.
