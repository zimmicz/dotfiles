# cf-triage — my software factory

Unattended pipeline on my laptop. Turns Sentry issues and any ticket I feed it into analyzed plans, on my go (`ai-fix` label) implements them as draft MRs, verifies them on a preview against holdout scenarios, and reports to Slack.

## Operating it

My job = intent in, decisions at gates, judge results. Agents do the rest.

**Three ways in**

| work | I do | factory does |
|---|---|---|
| Sentry bug | nothing — 1/day, only when no Sentry ticket is open | ticket → analysis → waits for me |
| any ticket I'd rather not code myself | add labels `triage` + `repo-<dir>` (one per repo) | same as Sentry, from analysis on |
| ticket I want to drive | `cf-work DEV-X` → `/cf-ticket` | interactive, 2 checkpoints |
| epic / big feature | `/cf-epic DEV-X` (re-run per wayfinder decision) | 3 checkpoints, one acceptance MR, then stacked review MRs |

**My gates in the background flow** (Slack `#zimmi-claude` tells me when each is due)

1. **Plan gate** — Jira comment with PLAN + Given/When/Then scenarios. Wrong/unclear → reply in Jira (answered ≤2 min, instantly if replied via `triage-dash`, PLAN + scenarios updated). Good → add `ai-fix`. The label approves plan AND scenarios — scenarios are what "done" means, read them.
2. **MR gate** — draft MR with verify verdict note. `ai-verified` → review diff, mark ready, add reviewers, merge. Want changes → comment in Jira, agent pushes to the same MR.
   **Merge block**: every tick `gate_sync` posts commit status `cf-factory` on each open `ai-mr` MR's head pipeline (merge-ref commit + `pipeline_id`; a status on the branch head commit makes a separate pipeline and blocks nothing). "Pipelines must succeed" then holds the merge / train until the round is done:

   | labels | `cf-factory` | merge |
   |---|---|---|
   | MR label `ai-gate-skip` (`s` in `triage-dash`) | success | allowed |
   | `ai-verified` + `ai-reviewed` | success | allowed |
   | `ai-verify-failed` / `ai-review-findings` / `ai-fix-failed` | failed | blocked (pipeline red) |
   | anything else (round in flight) | pending | blocked (pipeline running, `CI_MUST_PASS` = CHECKING) |

   Only changes post (one log line `gate <old> → <new>` in `<KEY>.log`). A finished status can't go back to pending, GitLab adds a new status record on the same pipeline instead. `triage-worker DEV-X --gate` syncs one ticket and prints the state.
3. **Failures** — `triage-failed` / `ai-fix-failed` / `ai-verify-failed`: read the 🤖 Jira comment or `<KEY>.verify.md`; fix the plan/scenario and re-label (`triage` or `ai-fix`), take it over with `cf-work DEV-X`, or close the ticket.

**Merge conflicts** — every run rechecks ALL my open MRs (any repo, factory or not). Conflicting → rebase onto target in its worktree:

| MR | what happens |
|---|---|
| draft, not approved, only my commits | `pnpm-lock.yaml`-only → `--ours` + `pnpm install --lockfile-only` (no LLM); code → headless opus `/mattpocock-skills:resolving-merge-conflicts` (intent from PLAN.md + the develop commits' MRs), lint/typecheck/tests, `push --force-with-lease`, 🤖 MR note. Agent-resolved + `ai-verified` → label dropped → holdout re-verify |
| ready or approved, only my commits | clean rebase or `pnpm-lock.yaml`-only → resolved + pushed like a draft, MR note says no code changed (approval stays valid). Code conflict → rebase aborted, Slack only. Add MR label `ai-rebase` (or `b` in `triage-dash`) = my OK → agent resolves next run, label removed. Why the gate: approvals don't reset on push here, so an agent-written resolution would keep an approval nobody gave it |
| others' commits / dirty or diverged worktree / >10 files changed on both sides (`CONFLICT_MAX_FILES`) | Slack only → resolve by hand |

Failed resolve → rebase aborted, Slack line. One Slack line per MR head (also "no local repo"); retry same head = `rm ~/.local/state/cf-triage/conflicts/<project id>-<iid>`.

**Interactive `/cf-ticket`**: checkpoint 1 = approve plan + scenarios (ExitPlanMode), checkpoint 2 = verified draft MR → I say "ready" → it adds reviewers + logs time. `--step` for risky work.

**Interactive `/cf-epic`**: checkpoint 1 = SPEC.md, 2 = ticket breakdown + scenarios, 3 = acceptance on preview → I say "accepted" → stacked review MRs.

**Weekly (Monday Slack "Factory yield" + "Factory before/after")** — bail rate up → plans too vague, push harder at plan gate. Verify failures → scenarios or implementer weak. Reviewer notes per MR up → run `/cf-feedback` after merges so the reviewer learns. `$`/merged MR spikes → check `<KEY>.log`.

**After every merge with reviewer comments**: `/cf-feedback` → rule lands in `standards/<repo>.md`, the reviewer agent enforces it next time. Lint-able → offer ticket.

**Debugging a run**

| want | how |
|---|---|
| all tickets at a glance + act on gates | `triage-dash` TUI: sidebar grouped running (green ●, label = active stage from unit `cf-triage-<KEY>-<stage>`: analysing / implementing / verifying / reviewing / fixing review / rebasing …) / needs you / todo (my sprint To Do, not in factory) / in flight / parked, pane with log / PLAN.md / holdout scenarios (`<KEY>.md` + `.api.md`) / Jira description / Jira comments (`tab`, markdown rendered). Gates without Jira: `a` approve, `R` reply, `t` retry, `w` run worker, `f` feed a todo (pick repo → `triage` + `repo-<x>`), `s` skip merge gate (`ai-gate-skip`) (all y/n). MR rows show 🔒 gate pending / 🚫 gate failed. `e` opens `$EDITOR` on PLAN.md / holdout files (plan / holdout tab; a changed holdout file is re-attached to Jira as `holdout.md` / `holdout.api.md`, replacing the old one). `c` resumes the session in a tmux window. `?` lists all keys. Mouse: tmux `prefix m` |
| is something running now | `workmux dashboard` — headless runs show as their `cf-triage` window (working → done) while active; finished ones drop off |
| watch it live | `tmux switch-client -t cf-triage` (one window per worktree, `tail -f` of the log) or `tail -f ~/.local/state/cf-triage/<KEY>.log` (🔧 tool calls, 💬 agent text, 📏 context size at every 50K step, 🧹 compaction if one ever fires, ✅ exit + cost + turns + peak context) |
| ask the agent why it did X | log line `🆔 session <id> — reopen: cd <wt> && claude --resume <id> — transcript ~/.claude/projects/<cwd slug>/<id>.jsonl` → resume interactively, or dig the transcript file offline (every stage has one, a verify run ≈ 20 MB). `<KEY>.stream.jsonl` is only the last implement run, kept for the worker's fallback comment |
| verify result | `~/.local/state/cf-triage/<KEY>.verify.md` (+ screenshot paths), video: one raw recording per scenario `<KEY>.verify-media/S<n>.webm` + `narration.tsv` (Czech sentences per scenario, written by the verifier from scenarios + results, never PLAN.md) → `narrate.sh` (edge-tts `cs-CZ-AntoninNeural` via `uvx edge-tts@7.3.1`, title card per scenario, waits cut by ffmpeg mpdecimate, VP9 crf 40, chapters) → `<KEY>.short.webm` (the one uploaded; narrate fails → note without video), MR note with uploaded video link |
| did the timer run / crash | `journalctl --user -u cf-triage --since today` (dispatcher); a stage: `journalctl --user -u cf-triage-<KEY>-<stage>`; running stages: `systemctl --user list-units 'cf-triage-*'` |
| re-run one stage | `triage-worker DEV-X` (analyze / implement if `ai-fix`), `triage-worker DEV-X --verify`. Spawns the stage detached; no free slot or ticket already running → exit 75 |
| why wasn't X picked up | X assigned to someone else (factory + triage-dash only take tickets assigned to me or unassigned; reassign = hand-off), or slots below: all `SLOTS` busy, or X is `impl`/`mr` and `impl-*` is full, or X is `verify` and its preview isn't ready (pipeline not green / Amplify note missing) → retried next tick |
| kill one stage | `systemctl --user stop cf-triage-<KEY>-<stage>` |

**Slots** (`cf-triage.service` = dispatcher on an every-minute timer; it picks work, spawns each stage as its own transient unit `cf-triage-<KEY>-<stage>` via `systemd-run --user` and exits within seconds, so a long stage blocks nothing but its own slot):

| limit | what | knob |
|---|---|---|
| `slots/slot-1..N` | every stage takes one; none free → rest waits for the next tick | `SLOTS` (default 3) |
| `slots/impl-1..K` | `impl` (local lint / typecheck / tests choke the laptop) | `IMPL_SLOTS` (default 1) |
| `slots/mr-1..K` | `mr` (answers to my comments on an open MR; own cap so a long `impl` can't delay an answer) | `MR_SLOTS` (default 1) |
| `slots/review-1..K` | `review` (opus, read-only; capped to bound cost). Its fix round `rfix` runs lint / tests, so it takes an `impl` slot | `REVIEW_SLOTS` (default 1) |
| `slots/verify-1..K` | holdout `verify` (browser, dyn-env create) | `VERIFY_SLOTS` (default 1) |
| `locks/<KEY>` | one ticket never gets two stages at once; busy → skipped | |
| held ticket | stage's class slot full (e.g. `mr` waiting for `mr-1`) → that ticket's later stages skip this tick too, so `review`/`verify` never run on code a pending answer is about to change | |

Per-tick priority (finish in-flight work before starting new): reply → mr → rfix → verify → review → conflict → new → impl → done → sentry. `verify` takes any `ai-mr` ticket without `ai-verified` / `ai-verify-failed` / `ai-review-retry`; `review` runs ONCE per MR, only on `ai-verified` tickets without a review label (worktree + open MR needed), so the behaviour check comes first and the reviewer sees code that already works. An `mr` follow-up push re-runs verify only, never the review. A stage already running for a ticket (unit `cf-triage-<KEY>-*` active) is not spawned again. `verify` is spawned only when its preview is ready (not ready for `STALL_HOURS`, default 4 → one Slack line with the reason: no holdout file, no worktree / open MR, pipeline not green + MR link, Amplify note missing; repeated daily while stuck), `conflict` only when the MR head / draft / `ai-rebase` state changed since the last handled one (discovery every `CONFLICT_EVERY` min, default 5). Stage timeouts (`RuntimeMaxSec`): impl / mr / rfix / conflict / verify 110 min, review 45 min, new / reply / sentry 30 min, done 5 min.

All locks are `flock -n -o`: `-o` keeps the lock fd out of claude / pnpm / MCP children, so a leftover background process can't hold a slot forever. Each stage unit gets `EnvironmentFile=~/.config/cf-triage.env` and the dispatcher's `PATH` (transient units inherit neither). `TRIAGE_DRY=1 triage-worker` prints what it would spawn, spawns nothing.
| interactive sub-agents (`/cf-ticket` verify, `/cf-epic` slices) | inside that session only (agent panel / transcript), not in the dashboard |

**Don't**: merge anything not `ai-verified` without reading why; edit holdout files after `ai-fix` (they're the contract); feed the factory tickets without a `repo-*` label.

## Flow

```
Sentry (fe-monitoring, can-frontend)
  │  sentry-triage          first run of each workday, 1 new ticket, ONLY if no pipeline ticket is open
  ▼
DEV Bug  [triage, sentry-<SHORTID>…, repo-<repo>]      assigned to me in Sentry; description ends with "Belongs in Sentry?" verdict
  │  triage-worker          every minute: cf-work --no-claim → claude -p "/cf-work-headless KEY"
  ▼  GRILL=1 only: blocking gaps in the ticket?
[needs-answer]  Czech Jira comment: Zjištěno + numbered Otázky, each with a recommended answer; no PLAN.md, blocks ai-fix
  │  I reply "ok" / "ok, 3 jinak: …"  →  analysis re-runs: more questions → next round (≤ GRILL_MAX_ROUNDS), none ↓
  ▼  no gaps (or GRILL off): straight here
[triaged]  worktree + PLAN.md (first section: Belongs in Sentry? → filter fix if no; repo has `GLOSSARY.md`: `## Terms` = entries to add, Czech ticket words mapped via `_CZ_`) + holdout scenarios + Czech Jira comment (🤖 + "AI triage" marker)
  │  I reply in Jira  →  next run (≤2 min) answers, updates PLAN.md + scenarios   (To Do tickets any age, plus any not-Done ticket labelled `ai-fix`, so a comment can't deadlock behind `impl`)
  │  I add label ai-fix  (= approves plan AND scenarios)
  ▼  triage-worker          `IMPL_SLOTS` at a time: cf-work (claims) → claude -p "/cf-ticket-headless KEY"
[ai-mr]  draft MR, no reviewers, assigned to me + Jira comment with MR link (glossary repo: `## Terms` written into `GLOSSARY.md` in the same commits, names follow its EN terms)
  │  triage-worker          every repo-* MR's latest pipeline green (FE-only: + Amplify bot note for that sha); tickets already On nightly / To test / Release candidate / Rejected (merged) are never looked at, neither by `verify` nor by `mr` follow-ups (comment on a merged ticket = nothing happens)
  ▼                         → fresh claude -p "/cf-holdout-ui" runs holdout scenarios in Playwright (any BE repo / cross-repo: one throwaway dyn-env, all BE MR images + FE branch)
  │  triage-worker          BE refactor (holdout/KEY.api.md exists): after UI verify passes → "/cf-holdout-api" runs the same requests on 2 dyn-envs (nightly vs MR images, same seed), any diff = fail
[ai-verified]  MR note with per-scenario verdicts
  │  fail → 1 retry: "/cf-ticket-headless KEY --verify-fix <report>" pushes fix [ai-verify-retry] → re-verify, failed scenarios first (`failed=S3,S5`), one still failing → rest not run. Fix round pushed nothing → [ai-verify-failed], no re-verify
  │  fail again / blocked / crash → [ai-verify-failed] = needs me, no review happens (`triage-worker KEY --review` by hand)
  │  triage-worker          stage review, ONCE per MR: fresh claude -p "/cf-review" (opus, read-only) reads the MR diff vs PLAN.md + repo standards (+ `GLOSSARY.md` where one exists: `_Avoid_` words / synonym drift = should-fix, never blocking), every finding with a failure scenario, posts ONE MR note
  │                         clean / should-fix only → [ai-reviewed]. blocking → [ai-review-retry] → stage rfix ("/cf-ticket-headless --review-fix <report>", impl slot) fixes EVERY finding once, then "/cf-review … prior=<KEY>.review.prev.md since=<heads>" re-checks only the pushed delta (fixed / open / new per prior finding)
  │                         re-check clean or minor → [ai-reviewed] + verify labels dropped (verify runs again on the fixed code). pushed nothing / still blocking / crash → [ai-review-findings] = needs me (verify still runs)
[ai-reviewed]  (+ ai-verified)
  │  I review, mark ready, add reviewers, merge; release moves ticket to Done
  ▼  triage-worker          ticket Done → resolves every sentry-* issue (haiku + Sentry MCP)
[ai-done]  → next morning sentry-triage picks the next issue
```

**Grill phase (opt-in, `GRILL=1` in `~/.config/cf-triage.env`; unset = off, nothing above changes).** `/cf-work-headless --grill` first decides whether the ticket has blocking gaps (the answer would change the plan or the scenarios). Yes → ONE Czech Jira comment (`Zjištěno` + numbered `Otázky`, each with a recommended answer and tradeoff), label `triage` → `needs-answer`, Slack line, no PLAN.md. I reply `ok` (take every recommendation) or `ok, 3 jinak: …`; the reply (≤2 min, or `R` in `triage-dash`) re-runs the analysis. More questions → another round (cap `GRILL_MAX_ROUNDS`, default 3, then plan with leftovers as `[!]`); none → PLAN.md + scenarios, `needs-answer` → `triaged`, plan gate as before. `ai-fix` is ignored while `needs-answer` is set. No answer for `GRILL_REMIND_HOURS` (24) → one Slack reminder per round. Sentry tickets are asked only when the root cause stays unconfirmed. Worker ↔ skill handshake: `<KEY>.grill` (`ASK` | `READY`) + `<KEY>.round` in the state dir. What the code revealed persists in `notes/<KEY>.md` `## Analysis`, so a round re-explores only what the answers changed.

Any other ticket (feature, chore, manual bug): I add labels `triage` + `repo-<dir>` → same flow from `[triaged]` on; PLAN.md starts with Requirements instead of "Belongs in Sentry?". On Done it just gets `ai-done`.

One Sentry issue at a time: `sentry-triage` skips the day (Slack note) while any ticket with a `sentry-*` label and labelled `triage|triaged|triage-failed|ai-fix|ai-mr|ai-fix-failed` is not Done AND its MR is not merged/closed (`glab mr list --all --search KEY` in the `repo-*` repo; no MR at all = blocking, so failed/unfixed tickets block until closed). Jira status doesn't matter — `On nightly` with a merged MR is fine. Sentry resolve waits for status `Done` (prod release), not Release candidate; a 404 (issue deleted/merged) counts as resolved, model answer logged to `<KEY>.log`. Stamp `~/.local/state/cf-triage/sentry-today` = already ran today (delete to re-run).

Failures: `triage-failed` / `ai-fix-failed` label. Every run with activity posts a summary to Slack `#zimmi-claude` via `~/.claude/scripts/slack-post.sh` (incoming webhook, no LLM). Quiet timer runs → no post, but if nothing posted for 3 h a "cf-triage alive — N quiet runs since X" heartbeat goes out (state `.posted` mtime + `.quiet` in `~/.local/state/cf-triage`). No heartbeat for >3 h during 07–19 weekdays = worker dead → `systemctl --user status cf-triage`.

## Labels

| label | meaning | set by |
|---|---|---|
| `triage` | new, waiting for analysis | sentry-triage, or me on any ticket |
| `sentry-<SHORTID>` | linked Sentry issue(s), full short id (dedup key) | sentry-triage |
| `repo-<dir>` | repo under `~/work/cleverfarm/repos` | sentry-triage |
| `needs-answer` | grill phase (`GRILL=1`): questions posted, no plan yet. Blocks `ai-fix`; my reply re-runs the analysis | triage-worker |
| `triaged` | PLAN.md + comment done | triage-worker |
| `triage-failed` | analysis or reply follow-up failed (no repo, worktree, crash); parked, no retry loop, re-add `triage` | triage-worker |
| `ai-fix` | **my approval** to implement PLAN.md | me |
| `ai-mr` | draft MR exists | triage-worker |
| `ai-reviewed` | independent review done (runs once per MR, after `ai-verified`): clean, should-fix only, or its findings were fixed once and the delta re-check passed. Survives `mr` follow-up pushes (no re-review) | triage-worker |
| `ai-review-retry` | reviewer found must-fix issues, `rfix` will fix every finding once, then re-check the delta | triage-worker |
| `ai-review-findings` | findings still open after the one fix round (nothing pushed, or re-check still blocking) or the reviewer crashed: read the MR note / `<KEY>.review.md`. Remove the label to re-run the review | triage-worker |
| `ai-fix-failed` | agent bailed out (see its Jira comment) or no MR. A run that ended without a Jira comment gets one posted by the worker (its final answer). `mr` follow-up stuck like that stays `ai-mr`, my next reply re-runs it | triage-worker |
| `ai-verify-retry` | holdout failed once, fix pushed, waiting for re-verify | triage-worker |
| `ai-verified` | holdout scenarios pass on the preview; gates `review`. Dropped (with `ai-verify-failed` / `ai-verify-retry`) when an `mr` follow-up or `rfix` pushes new commits, so `verify` runs again; review labels stay | triage-worker |
| `ai-verify-failed` | holdout failed after retry, blocked, or verifier crashed | triage-worker |
| `ai-done` | ticket Done, Sentry issues resolved | triage-worker |
| `ai-gate-skip` (GitLab MR label) | override: `cf-factory` merge gate goes green whatever the Jira labels | me (`s` in triage-dash) |

## Files

| path | what |
|---|---|
| `~/dotfiles/bin/sentry-triage` | stage 1: open-ticket gate + daily stamp, then claude -p (sonnet) + Sentry MCP + `jira` → ticket. `--dry-run` (ignores gate + stamp), `MAX=` (default 1) |
| `~/dotfiles/bin/sentry-triage-now` | force a Sentry run today: drop `sentry-today` stamp + start `cf-triage.service` (no block; dispatcher spawns the `sentry` stage) |
| `~/dotfiles/bin/triage-worker` | stages 2–7: triage, reply follow-up, implement, holdout verify (+1 retry), resolve Sentry on Done, rebase my conflicting MRs (dispatcher runs only), `sentry` stage (runs `sentry-triage`); Slack summary (notes moved away atomically before posting, so stages can append meanwhile). No args = dispatcher, see Slots. `triage-worker DEV-X` = spawn one ticket's stage (implements if it has `ai-fix`; `triage-worker DEV-X --verify` = verify its MR now, `--review` = independent review now). `MAX=` (new tickets looked at per tick, default 10), `SLOTS=`, `IMPL_SLOTS=`, `MR_SLOTS=`, `REVIEW_SLOTS=`, `VERIFY_SLOTS=`, `CONFLICT_EVERY=`. Holdout verify of a BE ticket holds its verify slot ~30 min (dyn-env create). Analyze + implement open a detached window per worktree in tmux session `cf-triage` (`tail -f <KEY>.log`) and run claude with that `TMUX_PANE`, so workmux status hooks show 🤖/✅ in the dashboard. Plain `tmux new-window -d`, not `workmux open` (that one `switch-client`s your attached client). No tmux server = skipped |
| `~/dotfiles/bin/factory-stats` | `factory-stats [days=7]`: yield per ticket touched in window — stages, bail rate, verify pass rate, reviewer notes per merged MR, days ticket→merge, $ total + per merged MR (from `cost=$` in `<KEY>.log`). Posted to Slack every Monday by `nightly-claude.sh` |
| `~/dotfiles/bin/factory-compare` | `factory-compare [cutover]` (default 2026-11-05 13:35, the context-waste commit): per-run medians before vs after per stage from the logs: cost, tool calls, peak + first-turn context (old runs: from the transcript named on the 🆔 line, cached in `peaks.tsv`), verify calls before the first screenshot, verify / impl / analyze runs per ticket, same-pipeline re-verifies, `TICKET.md` bytes, notes files present. Posted to Slack every Monday next to factory-stats; read nothing before ~15 tickets after the cutover |
| `~/dotfiles/bin/cf-work` | worktree setup, one per repo (`--repos a,b` > `repo-*` labels > mapped labels `frontend`/`agroevidence`/… > cwd); same branch everywhere, claude in primary worktree + `--add-dir` siblings; `--no-launch`, `--no-claim` (skip assign + In Progress); `--comments-after <id>` (worker passes `handled/<KEY>`): `TICKET.md` puts the human comments above that id under `## New comments` right after the description, older human ones under `## Earlier comments`, 🤖 comments as one-line stubs except the last (full), so a stage reads only what is new |
| `~/dotfiles/claude/cleverfarm/skills/cf-work-headless` | root-cause analysis → PLAN.md + holdout scenarios (Given = fixture id from `cf-farm-frontend/tests/FIXTURES.md` + path with real ids, so the verifier opens it directly) + comment; follow-up mode on replies; grill mode. Interactive twin `/cf-work` carries none of it |
| `~/dotfiles/claude/cleverfarm/skills/cf-ticket-headless` | implement PLAN.md, verify, self-review, `/cf-mr --headless`; variants: MR follow-up, `--verify-fix`, `--review-fix`, duplicate close. Interactive twin `/cf-ticket` carries none of it |
| `~/dotfiles/claude/skills/cf-mr` | `--headless`: `glab mr create --draft`, no reviewers, stays In Progress, FE plays Nightly preview |
| `~/dotfiles/claude/cleverfarm/skills/cf-holdout-ui` | holdout verifier: Playwright login as `fe-automation@cleverfarm.cz` (password from cf-farm-frontend `.env`), runs scenarios, rewrites `<KEY>.verify.md` after every scenario (`VERDICT: running`, final `pass/fail/blocked`; a run killed mid-way leaves `running` = crash), records a video when triage-worker passes `video=` (own Playwright MCP `@0.0.83 --headless --caps devtools --executable-path /usr/bin/chromium`, `--strict-mcp-config`; interactive runs don't record), MR note. Never edits code. BE: creates + destroys `<key>-verify` dyn-env |
| `~/dotfiles/claude/cleverfarm/skills/cf-review` | independent reviewer: fresh opus session, read-only, `/mattpocock-skills:code-review` against `PLAN.md` + standards plus its own security / data / performance / test checks, every finding grounded in code with a failure scenario; writes `<KEY>.review.md` (`VERDICT: clean/minor/blocking`) and one MR note. With `prior=<KEY>.review.prev.md since=<worktree>@<sha>,…` (after `rfix`): incremental re-check of the pushed delta only, each prior finding marked fixed / open / dropped, new ones only from the delta. Never edits code |
| `~/dotfiles/claude/cleverfarm/skills/cf-holdout-api` | differential API verifier for BE refactors: two dyn-envs (nightly vs MR images, same seed), same request sequences from `holdout/<KEY>.api.md`, writes `<KEY>.verify-api.md` (`VERDICT: pass/fail/blocked`), token from a nightly Playwright login |
| `~/dotfiles/claude/cleverfarm/systemd/cf-triage.{service,timer}` | symlinked to `~/.config/systemd/user/`; dispatcher Mon–Fri every minute 07–18 (idle run = a few Jira queries, no LLM; failed Sentry triage retries ≤ every 15 min). `triage-dash` actions start it right away, `Persistent=true`, timeout 10 min. Stages are transient `cf-triage-<KEY>-<stage>` units, not files |
| `~/.config/cf-triage.env` | `JIRA_EMAIL`, `JIRA_TOKEN` (0600, not in git) |
| `~/.local/state/cf-triage/` | `<KEY>.log`, `<KEY>.stream.jsonl` (last implement run only, source of the worker's fallback comment; full transcripts of every stage live in `~/.claude/projects/<cwd slug>/<session>.jsonl`), `<KEY>.verify.md`, `<KEY>.verify-media/` (video + MCP output), `<KEY>.review.md` (+ `.review.prev.md` = the report `rfix` worked from, input of the re-check), `holdout/<KEY>.md` (scenarios; analysis may write, implementer denied), `notes/<KEY>.md` (ticket memory: `## Analysis` written by the analyzer with file:line evidence and kept across grill rounds and replies, then the implementer's change map, decisions, deviations, gotchas, round log; every analyze / impl round reads it first, `Read` denied to review + verify), `sentry-*.json`, `sentry-last.txt`, `sentry-today` (daily stamp), `notes.txt` (pending Slack lines), `conflicts/<project id>-<iid>` (last handled MR head) + `.md` (resolver report), `slots/` (slot pool, see Slots), `locks/<KEY>` (tickets), `stalled/<KEY>.since` / `.alert` (verify-not-ready clock + last Slack alert), `handled/<KEY>` (id of the newest human Jira comment the last successful `new` / `reply` / `impl` / `mr` stage had seen when it started; a comment is pending while its id is higher, so a bot comment posted later can't bury it. No file = old rule: pending while the last comment is not a 🤖 one) |
| `~/.config/claude/slack-webhook` | incoming webhook URL for `#zimmi-claude` (0600, not in git) |
| `~/dotfiles/gitignore` | ignores `PLAN.md`, `TICKET.md`, `install.log`, `.codegraph/` so MRs don't pick them up |

## Setup

```
systemctl --user daemon-reload
systemctl --user enable --now cf-triage.timer
```

Prereqs (already done):
- claude.ai connectors authenticated: Sentry (`/mcp`), Atlassian, Slack — headless `claude -p` uses them with my login
- `glab auth status` ok, SSH to gitlab works without agent (systemd has no `SSH_AUTH_SOCK`)
- no linger: runs only while I'm logged in

## Operate

| command | what |
|---|---|
| `systemctl --user list-timers 'cf-triage*'` | last/next dispatcher run |
| `systemctl --user list-units 'cf-triage-*'` | running stages |
| `journalctl --user -u cf-triage --since today` | dispatcher output (spawned / no free slot) |
| `systemctl --user start cf-triage.service` | run now |
| `systemctl --user disable --now cf-triage.timer` | turn off |
| `tail -f ~/.local/state/cf-triage/DEV-XXXX.log` | live per-ticket progress: 🔧 tool calls, 💬 agent text, 📏 context size, ✅ exit + cost + turns + peak |
| `git -C ~/dotfiles tag -a cf-triage-<label>-<date> -m "…"` | snapshot before a factory change (bin + skills + docs are symlinks, so the tag is the whole factory; note the mattpocock-skills version in the message). Last: `cf-triage-pre-glossary-2026-11-06` |
| `git -C ~/dotfiles checkout cf-triage-<label>-<date> -- bin claude/cleverfarm` | roll the factory back to that snapshot (takes effect at once; `go build` only if `go/triage-dash` changed). Back to head: `git checkout master -- bin claude/cleverfarm` |

Glossary (mattpocock `domain-modeling`, FE only for now): `GLOSSARY.md` at the repo root of cf-farm-frontend, format + `_CZ_` rule in its `docs/agents/domain.md`. Read at `new`/`reply` (PLAN.md `## Terms`), written at `impl`, checked at `review`. Repos without the file: nothing changes. Interactive `/cf-ticket` phase 0 runs `grill-with-docs` (grilling + domain-modeling) instead of plain grilling.

## Test one ticket by hand

1. `sentry-triage --dry-run` → preview (shows gate verdict too); `rm ~/.local/state/cf-triage/sentry-today; sentry-triage` → 1 real ticket (only if nothing open)
2. `triage-worker DEV-XXXX` → analysis (any DEV ticket with a `repo-*` label works)
3. comment in Jira, `triage-worker` (no args) → reply answered
4. add `ai-fix`, `triage-worker DEV-XXXX` → draft MR
5. ticket Done → `triage-worker` (no args) → Sentry issues resolved, label `ai-done`

Pick up manually instead: `cd ~/work/cleverfarm/repos/<repo> && cf-work DEV-XXXX` → claims ticket, reuses worktree + PLAN.md, launches interactive `/cf-ticket`.

## Scope & mapping (in `sentry-triage` prompt)

- Sentry org `cleverfarm-sro` (region us), my user id `3546621`
- query: `is:unresolved is:unassigned`, 90d, sort freq, all environments, no noise filter
- `fe-monitoring` → `cf-farm-frontend`, Product by route (telematics 11714, sensors/irrigation 11712, actions/parcels/catalogues/stores 11711, satellite 11713, else 11710)
- `can-frontend` → `can-frontend`, Product 11707
- same title + project within one run → one ticket with several `sentry-*` labels (across runs: add label manually + assign in Sentry)
- dedup: JQL `labels = sentry-<SHORTID>`; assigning in Sentry also drops it from `is:unassigned`

## Guardrails

- Everything runs as me (Jira, GitLab, Sentry, Slack tokens are mine). Posts start with the 🤖 prefix from `~/.claude/my-voice.md`.
- Analysis: read-only tools + write `PLAN.md` + `jira add-comment`.
- Implement: denied `git push --force/-f`, push to develop/staging/production, `reset --hard`, `glab mr merge/approve`. Protected branches enforce it server-side too.
- Implementation only after I add `ai-fix`. MR always draft, no reviewers.
- Agent bails (comment, no MR) if root cause doesn't hold, scope grows, or tests can't go green.
- Only my comments are instructions: a Jira comment counts as human only when `author.accountId` = `jira GET /rest/api/3/myself` (worker exits with `lookup failed` if that call fails), MR notes only with my GitLab username. Comments by others never start a run and land in `TICKET.md` under `## Comments by others` as context only.
- Context per stage: every `claude -p` gets `--settings` disabling the plugins no stage uses (caveman, ponytail, slack, cf-planning, cf-release, cf-kb, cf-eod, cf-bug-report, cf-cli-skill, cf-investigator, cf-cc, typescript-lsp; `HEADLESS_SETTINGS` in triage-worker). impl / review / conflict / verify run `--strict-mcp-config` with lean-ctx + codegraph only (`mcp_config`; Playwright added for impl's `/check-layout` and for verify). `new` / `reply` keep every MCP server: Sentry is a claude.ai connector and strict mode drops connectors. First-turn context ≈ 50K tokens instead of 63K; headless skills are split from the interactive ones so no checkpoint / plan-mode text is loaded.

## Notifications

- Slack `#zimmi-claude` via incoming webhook (`slack-post.sh`, same as nightly digests), one message per run with activity; notes kept until curl succeeds.
- Webhook posts ping like any bot message. Jira comments come from my account → no Jira ping.

## Costs (DEV-2825 test)

create ticket ~$0.45 · analysis ~$0.65 · reply ~$0.4 · implement ~$1.6 (opus) · Sentry resolve (haiku) cents · Slack post free.

## Known gaps

- editing an old non-AI comment counts as a new reply if it's the last one
- merge gate runs only with the timer (Mon–Fri 07–18): a new head pipeline outside those hours has no `cf-factory` status until the next tick, so it is mergeable meanwhile
- MCP login expiry → runs fail. sentry-triage without `TRIAGE_DONE` → one Slack note per day (`sentry-failed-noted` stamp) with the `TRIAGE_FAILED` reason; fix Sentry auth via `/mcp` → claude.ai Sentry. Worker failures → `*-failed` labels / logs
- Persistent catch-up can fire once on Saturday if the laptop was off at Friday's last slot (daily stamp keeps it to one ticket)
- a `triage-failed` / `ai-fix-failed` ticket blocks new issues until I close or relabel it
