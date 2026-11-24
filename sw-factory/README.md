# sw-factory — my software factory

Unattended pipeline on my laptop. Turns Sentry issues and any ticket I feed it into analyzed plans, on my go (`ai-fix` label) implements them as draft MRs, verifies them on a preview against holdout scenarios, and reports to Slack.

## Operating it

My job = intent in, decisions at gates, judge results. Agents do the rest.

**Three ways in**

| work | I do | factory does |
|---|---|---|
| Sentry bug | nothing — 1/day, only when no Sentry ticket is open | ticket → analysis → waits for me |
| any ticket I'd rather not code myself | add labels `triage` + `repo-<dir>` (one per repo) | same as Sentry, from analysis on |
| new work, no ticket yet (e.g. "can-frontend npm → pnpm") | `n` in `factory-dash`: title (Czech), repo, type (Technical / Task / Bug), optional epic, description (`Ctrl+V` pastes images), `Ctrl+Enter` → confirm | DEV ticket assigned to me, active sprint, Product from the repo (`product()` in `dash/main.go`), labels `repo-<dir>` + `triage` (added last, after images are attached) → analysis next tick |
| dependabot MR (cf-farm-frontend, can-frontend) | nothing | `deps-triage` every 30 min: one ticket per bot MR, see Dependabot lane |
| ticket I want to drive | `cf-work DEV-X` → `/cf-ticket` | interactive, 2 checkpoints |
| epic / big feature | `/cf-epic DEV-X` (re-run per wayfinder decision) | 3 checkpoints, one acceptance MR, then stacked review MRs |

**My gates in the background flow** (Slack `#zimmi-claude` tells me when each is due)

1. **Plan gate** — Jira comment with PLAN + Given/When/Then scenarios. Wrong/unclear → reply in Jira (answered ≤2 min, instantly if replied via `factory-dash`, PLAN + scenarios updated). Good → add `ai-fix`. The label approves plan AND scenarios — scenarios are what "done" means, read them.
2. **MR gate** — draft MR with verify verdict note. `ai-verified` + `ai-reviewed` → **quiz**: `Q` in `factory-dash` opens `claude --model sonnet '/cf-quiz KEY'` (tmux window): 4–6 open-book questions (why / predict / blast radius / alternatives) on diff + PLAN.md, wrong answer = hint + re-ask, still wrong = explained + fresh question at the end; every question must end correct. Scenarios the factory could not run but counted as verified (`unverified/<KEY>.md`: `DATA:` fixture lacks the data, `NO PREVIEW:` can-frontend) come first: I say how I checked each by hand (or the concrete code reason it can't break); dash next step shows `hand-check S7,S8`. Pass → `ai-quizzed`, record + quizzed SHAs in `~/.local/state/sw-factory/quiz/<KEY>.md`. New commits drop `ai-quizzed`, re-quiz covers only the delta since those SHAs. Then mark ready, add reviewers, merge. Want changes → comment in Jira, agent pushes to the same MR.
   **Merge block**: every tick `gate_sync` posts commit status `cf-factory` on each open `ai-mr` MR's head pipeline (merge-ref commit + `pipeline_id`; a status on the branch head commit makes a separate pipeline and blocks nothing). "Pipelines must succeed" then holds the merge / train until the round is done:

   | labels | `cf-factory` | merge |
   |---|---|---|
   | MR label `ai-gate-skip` (`s` in `factory-dash`) | success | allowed |
   | `ai-verified` + `ai-reviewed` + `ai-quizzed` | success | allowed |
   | `ai-verified` + `ai-reviewed` | pending (`quiz Michal`) | blocked |
   | `ai-verify-failed` / `ai-review-findings` / `ai-fix-failed` | failed | blocked (pipeline red) |
   | anything else (round in flight) | pending | blocked (pipeline running, `CI_MUST_PASS` = CHECKING) |

   Only changes post (one log line `gate <old> → <new>` in `<KEY>.log`). A finished status can't go back to pending, GitLab adds a new status record on the same pipeline instead. `factory-worker DEV-X --gate` syncs one ticket and prints the state.

Conventions injection: `impl` (all variants) and `review` get the repo's conventions in the system prompt (`--append-system-prompt`, `<conventions file=…>` blocks), because nested `CLAUDE.md` only load on native `Read` and the checklist is a skill nobody invokes. Sources, in order: `.claude/skills/review-checklist/SKILL.md`, every non-root `CLAUDE.md` in an ancestor dir of a changed path (MR diff + PLAN.md `### Change map`), dotfiles `standards/common.md`. Repo files are read from the MR target branch (`origin/<base>`), not the worktree, so an MR cannot weaken its own rules. Over 100 KB → nested `CLAUDE.md` dropped deepest first. One log line `conventions: <n> files, <bytes> B: <paths>` in `<KEY>.log`. `factory-worker DEV-X --conventions` prints the block and exits.
3. **Failures** — `triage-failed` / `ai-fix-failed` / `ai-verify-failed`: read the 🤖 Jira comment or `<KEY>.verify.md`; a wrong scenario usually arrives as ✏️ `holdout proposal` (dash `A` / `X`); otherwise fix the plan/scenario and re-label (`triage` or `ai-fix`), take it over with `cf-work DEV-X`, or close the ticket.

**Merge conflicts** — every run rechecks ALL my open MRs (any repo, factory or not). Conflicting → rebase onto target in its worktree:

| MR | what happens |
|---|---|
| draft, not approved, only my commits | `pnpm-lock.yaml`-only → `--ours` + `pnpm install --lockfile-only` (no LLM); code → headless opus `/mattpocock-skills:resolving-merge-conflicts` (intent from PLAN.md + the develop commits' MRs), lint/typecheck/tests, `push --force-with-lease`, 🤖 MR note. Agent-resolved + `ai-verified` → label dropped → holdout re-verify |
| ready or approved, only my commits | clean rebase or `pnpm-lock.yaml`-only → resolved + pushed like a draft, MR note says no code changed (approval stays valid). Code conflict → rebase aborted, Slack only. Add MR label `ai-rebase` (or `b` in `factory-dash`) = my OK → agent resolves next run, label removed. Why the gate: approvals don't reset on push here, so an agent-written resolution would keep an approval nobody gave it |
| others' commits / dirty or diverged worktree / >10 files changed on both sides (`CONFLICT_MAX_FILES`) | Slack only → resolve by hand |

Failed resolve → rebase aborted, Slack line. One Slack line per MR head (also "no local repo"); retry same head = `rm ~/.local/state/sw-factory/conflicts/<project id>-<iid>`.

**Interactive `/cf-ticket`**: checkpoint 1 = approve plan + scenarios (ExitPlanMode), checkpoint 2 = verified draft MR → I say "ready" → it adds reviewers + logs time. `--step` for risky work.

**Interactive `/cf-epic`**: checkpoint 1 = SPEC.md, 2 = ticket breakdown + scenarios, 3 = acceptance on preview → I say "accepted" → stacked review MRs.

**Weekly (Monday Slack "Factory yield" + "Factory before/after")** — bail rate up → plans too vague, push harder at plan gate. Verify failures → scenarios or implementer weak. Reviewer notes per MR up → run `/cf-feedback` after merges so the reviewer learns. `$`/merged MR spikes → check `<KEY>.log`.

**After every merge with reviewer comments**: `/cf-feedback` → rule lands in the repo's `review-checklist` (commit on the ticket's MR, or a chore ticket once merged), cross-repo ones in `standards/common.md`; the reviewer agent enforces it next time. Lint-able → offer ticket.

**Debugging a run**

| want | how |
|---|---|
| all tickets at a glance + act on gates | `factory-dash` = web dashboard: the command (or launcher entry **Factory Dash**, `sw-factory/applications/factory-dash.desktop` symlinked to `~/.local/share/applications/` by `install.sh`) opens http://127.0.0.1:7777 signed in, in its own `vivaldi --app` window without tabs / panels (i3: always workspace 0, `Super+D` focuses or launches it; `i3/config`) (no `vivaldi` → `xdg-open`) (server = `factory-dash.service`, always on). **Overview** (`1`): sidebar grouped running (green ●, label = active stage from unit `sw-factory-<KEY>-<stage>`: analysing / implementing / verifying / reviewing / fixing review / rebasing …) / needs you / todo (my sprint To Do, not in factory) / in flight / parked; every ticket shows who is on turn (you / factory / reviewer / ci) + the next step; tabs (`tab`) log / PLAN.md / holdout scenarios (`<KEY>.md` + `.api.md`) / Jira description / Jira comments (AI cyan, mine yellow, others magenta; attached images inline) / verify (report + screenshots + videos: the one on the report's `Video:` line is marked `current` and gets clickable chapters coloured by scenario verdict, the playing chapter's full Given/When/Then from `holdout/<KEY>.md` shown below; media >1 h older than the report = `earlier run`) / review. **UI fix** (`2`): read-only view of the separate ui-fix runner (`~/work/cleverfarm/agents/ui-fix`), not part of the factory flow: live Jira list of `labels = ui-fix` tickets open or updated in 7 days, state from its `ui-fix-*` label, Jira status, reporter, MR + round from `~/.local/state/ui-fix/<KEY>/meta.json`, agent cost summed from its `agent-*.json` (`UI_FIX_STATE` overrides the dir), counts per state + average cost on top; `j` / `k` move, `enter` / click opens it in Overview, `J` Jira, `M` MR, `r` reload. **Backlog** (`3`): live Jira list of my tickets not Done, not On nightly / Release candidate and not in an open sprint, either in no sprint or in a future one (Tech Backlog, Příprava na sprint …; any project), future sprint name + status + labels + last update; `j` / `k` move, `enter` / click opens it in Overview, `J` Jira, `r` reload. **Board** (`B`): one column per step, todo → analysis → plan gate → implement → verify → review → quiz → merge → nightly → RC → done. **Stats** (`S`): cost per day, per-stage medians (cost, minutes, turns, peak context), tool usage, most expensive tickets, plus `factory-stats` + `factory-compare` output. `/` searches key / title incl. RC + done; enter on a key not listed looks it up in Jira. `i` (or the status pill) = factory status: last Jira / GitLab fetch + errors, dispatcher timer + last run, running stage units. Gates without Jira: `a` approve, `R` reply (textarea, `Ctrl+V` pastes the clipboard image as `!img-N.png!`, attached to Jira on send as `reply-<time>-img-N.png`; `Ctrl+Enter` sends, no confirm), `t` retry, `w` run worker, `f` feed a todo (pick repo → `triage` + `repo-<x>`), `n` new ticket into the factory (see Three ways in; Esc keeps the draft), `b` agent resolves MR conflict, `s` skip merge gate (`ai-gate-skip`) (all but `R` confirmed in a modal, `y` / `n`), `Q` quiz me (🎓 quiz rows, `/cf-quiz` in a tmux window), `A` / `X` accept / reject a holdout proposal (✏️ rows; holdout tab shows the analyzer's `.why` + `diff -u`; accept = proposal replaces `holdout/<KEY>.md`, re-attached as `holdout.md`, 🤖 Jira comment with the diff, `## Holdout changes` line in `notes/<KEY>.md`, drops `ai-verified` / `ai-verify-failed` / `ai-verify-retry` / `ai-quizzed` → full re-verify; reject = deletes it). MR rows show 🔒 gate pending / 🚫 gate failed. `e` opens `$EDITOR` on PLAN.md / holdout files in a tmux window of the last active client (plan / holdout tab; a pending holdout proposal is opened instead of the holdout file, accept it with `A` after; a changed holdout file is re-attached to Jira as `holdout.md` / `holdout.api.md`, replacing the old one). `c` resumes the session in a tmux window. `?` lists all keys. Deep link: `#view=…&key=…&tab=…` |
| watch it live | `tmux switch-client -t sw-factory` (one window per worktree while a stage runs, `tail -f` of the log, closes when the stage ends) or `tail -f ~/.local/state/sw-factory/<KEY>.log` (🔧 tool calls, 💬 agent text, 📏 context size at every 50K step, 🧹 compaction if one ever fires, ✅ exit + cost + turns + peak context) |
| ask the agent why it did X | log line `🆔 session <id> — reopen: cd <wt> && claude --resume <id> — transcript ~/.claude/projects/<cwd slug>/<id>.jsonl` → resume interactively, or dig the transcript file offline (every stage has one, a verify run ≈ 20 MB). `<KEY>.stream.jsonl` is only the last implement run, kept for the worker's fallback comment |
| verify result | `~/.local/state/sw-factory/<KEY>.verify.md` (+ screenshot paths), video: one raw recording per scenario `<KEY>.verify-media/S<n>.webm` + `narration.tsv` (Czech sentences per scenario, written by the verifier from scenarios + results, never PLAN.md) → `narrate.sh` (edge-tts `cs-CZ-AntoninNeural` via `uvx edge-tts@7.3.1`, title card per scenario (no recording, e.g. blocked → its sentences shown on the card too), waits cut by ffmpeg mpdecimate, VP9 crf 40, chapters) → `<KEY>.short.webm` (the one uploaded; narrate fails → note without video), MR note with uploaded video link |
| did the timer run / crash | `journalctl --user -u sw-factory --since today` (dispatcher); a stage: `journalctl --user -u sw-factory-<KEY>-<stage>`; running stages: `systemctl --user list-units 'sw-factory-*'` |
| re-run one stage | `factory-worker DEV-X` (analyze / implement if `ai-fix`), `factory-worker DEV-X --verify`. Spawns the stage detached; no free slot or ticket already running → exit 75 |
| why wasn't X picked up | X assigned to someone else (factory + factory-dash only take tickets assigned to me or unassigned; reassign = hand-off), or slots below: all `SLOTS` busy, or X is `impl`/`mr` and `impl-*` is full, or X is `verify` and its preview isn't ready (pipeline not green / Amplify note missing) → retried next tick |
| kill one stage | `systemctl --user stop sw-factory-<KEY>-<stage>` |

**Slots** (`sw-factory.service` = dispatcher on an every-minute timer; it picks work, spawns each stage as its own transient unit `sw-factory-<KEY>-<stage>` via `systemd-run --user` and exits within seconds, so a long stage blocks nothing but its own slot):

| limit | what | knob |
|---|---|---|
| `slots/slot-1..N` | every stage takes one; none free → rest waits for the next tick | `SLOTS` (default 3) |
| `slots/impl-1..K` | `impl` (local lint / typecheck / tests choke the laptop) | `IMPL_SLOTS` (default 1) |
| `slots/mr-1..K` | `mr` (answers to my comments on an open MR; own cap so a long `impl` can't delay an answer) | `MR_SLOTS` (default 1) |
| `slots/review-1..K` | `review` (opus, read-only; capped to bound cost). Its fix round `rfix` runs lint / tests, so it takes an `impl` slot | `REVIEW_SLOTS` (default 1) |
| `slots/verify-1..K` | holdout `verify` (browser, dyn-env create) | `VERIFY_SLOTS` (default 1) |
| `locks/<KEY>` | one ticket never gets two stages at once; busy → skipped. One exception: `verify` locks `locks/<KEY>.verify` (it only clicks the preview, never touches the worktree), so an `mr` follow-up on my new comment starts while verify runs instead of waiting (it was 31 comments, 218 min in 4 weeks). Verify then drops its result when the worktree HEAD moved meanwhile (`stale: commits moved during verify`), and on a fail it takes `locks/<KEY>` before its verify-fix; busy → no fix now, the follow-up's push re-verifies | |
| `running/<KEY>` | `mr` / `impl` in flight: start time, stage, newest human Jira comment id + GitLab note id it started with, log line count; `.heads` = worktree HEADs. My new comment within `RESTART_WINDOW` (5 min) of its start, with HEAD unchanged and nothing posted yet (no `glab mr note`, `-X POST/PUT`, `jira POST/PUT`, `git commit/push` in the log since) → dispatcher stops the unit (`restart: new comment …`), stashes uncommitted work (`git stash list`), next tick runs one follow-up for all comments (`impl` → analysis follow-up first, then impl again). Was: 30 comments, ~300 min in 4 weeks waiting behind a run started 1.7–2.5 min before | `RESTART_WINDOW` |
| held ticket | stage's class slot full (e.g. `mr` waiting for `mr-1`) → that ticket's later stages skip this tick too, so `review`/`verify` never run on code a pending answer is about to change | |

Per-tick priority (finish in-flight work before starting new): env → reply → mr → rfix → verify → review → conflict → new → impl → done → sentry. `verify` takes any `ai-mr` ticket without `ai-verified` / `ai-verify-failed` / `ai-review-retry`; `review` runs on `ai-verified` tickets without a review label (worktree + open MR needed), so the behaviour check comes first and the reviewer sees code that already works. First run reviews the whole MR; after an `mr` follow-up or `rfix` push, `unverify` also drops `ai-reviewed`, so verify runs again, then `review` re-checks only the commits pushed since the last clean / minor review (heads in `<KEY>.reviewed`; no new commits → relabel only). `rfix` is capped at `RFIX_MAX` rounds per ticket (default 2, counter `<KEY>.rfix-count`), then `ai-review-findings`. Each review logs `findings: must=<n> should=<n> convention=<n>` in `<KEY>.log`. A stage already running for a ticket (unit `sw-factory-<KEY>-*` active) is not spawned again. `verify` is spawned only when its preview is ready (not ready for `STALL_HOURS`, default 4 → one Slack line with the reason: no holdout file, no worktree / open MR, pipeline not green + MR link, Amplify note missing; repeated daily while stuck), `conflict` only when the MR head / draft / `ai-rebase` state changed since the last handled one (discovery every `CONFLICT_EVERY` min, default 5). Stage timeouts (`RuntimeMaxSec`): impl / mr / rfix / conflict / verify / env 110 min, review 45 min, new / reply / sentry 30 min, done 5 min.

Usage limit (`You've hit your 5-hour limit · resets 5pm`): `stream` sees that result line, writes the reset time to `~/.local/state/sw-factory/.limit` (unparseable → 30 min) and the stage stops at its next label / Slack / Jira write (`limit_bail`, exit 2): labels stay, no `*-failed`, no comment, `rfix-count` and `ai-verify-retry` given back, `  usage limit hit, stage abandoned` in `<KEY>.log`. Until the reset every tick (also `factory-worker KEY`) prints `usage limit until …, tick skipped` and exits; after it the same labels spawn the same stages again. Limit lifted earlier (extra usage, other plan): `rm ~/.local/state/sw-factory/.limit`.

All locks are `flock -n -o`: `-o` keeps the lock fd out of claude / pnpm / MCP children, so a leftover background process can't hold a slot forever. Each stage unit gets `EnvironmentFile=~/.config/sw-factory.env` and the dispatcher's `PATH` (transient units inherit neither). `TRIAGE_DRY=1 factory-worker` prints what it would spawn, spawns nothing.
| interactive sub-agents (`/cf-ticket` verify, `/cf-epic` slices) | inside that session only (agent panel / transcript), not in the dashboard |

**Don't**: merge anything not `ai-verified` without reading why; edit holdout files after `ai-fix` (they're the contract; a wrong scenario goes implementer flags → analyzer proposes → I accept with `A`); feed the factory tickets without a `repo-*` label.

## Flow

```
Sentry (fe-monitoring, can-frontend)
  │  sentry-triage          first run of each workday, 1 new ticket, ONLY if no pipeline ticket is open
  ▼
DEV Bug  [triage, sentry-<SHORTID>…, repo-<repo>]      assigned to me in Sentry; description ends with "Belongs in Sentry?" verdict
  │  factory-worker          every minute: cf-work --no-claim → claude -p "/cf-work-headless KEY"
  ▼  GRILL=1 only: blocking gaps in the ticket?
[needs-answer]  Czech Jira comment: Zjištěno + numbered Otázky, each with a recommended answer; no PLAN.md, blocks ai-fix
  │  I reply "ok" / "ok, 3 jinak: …"  →  analysis re-runs: more questions → next round (no cap), none ↓
  ▼  no gaps (or GRILL off): straight here
[triaged]  worktree + PLAN.md (first section: Belongs in Sentry? → filter fix if no; `## Implementation` = spec of the future diff: change map, Now/After + key-line sketch per file, data flow, tests, out of scope; binding for impl, deviations logged; repo has `GLOSSARY.md`: `## Terms` = entries to add, Czech ticket words mapped via `_CZ_`; plan, scenarios and change-map names use EN terms, touched legacy `_Avoid_` names get rename rows, self-checked) + holdout scenarios + Czech Jira comment (🤖 + "AI triage" marker), PLAN.md attached to the ticket; missing required section → triage-failed
  │  I reply in Jira  →  next run (≤2 min) answers, updates PLAN.md + scenarios   (To Do tickets any age, plus any not-Done ticket labelled `ai-fix`, so a comment can't deadlock behind `impl`)
  │  I add label ai-fix  (= approves plan AND scenarios)
  │  no holdout scenario can run (every Given `DATA: missing` / `same as` one that is) → impl refused, [ai-fix] → [triaged] + Slack line; I fix fixture / env and reply, or accept blind: `touch ~/.local/state/sw-factory/<KEY>.blind` + `ai-fix` again (verify then stays off, stall note after `STALL_HOURS`)
  ▼  factory-worker          `IMPL_SLOTS` at a time: cf-work (claims) → claude -p "/cf-ticket-headless KEY"
[ai-mr]  draft MR, no reviewers, assigned to me + Jira comment with MR link (glossary repo: `## Terms` written into `GLOSSARY.md` in the same commits, names follow its EN terms in every round, touched legacy `_Avoid_` names renamed, `_Avoid_` grep on the diff gates verify)
  │  factory-worker          every repo-* MR's latest pipeline green (FE-only: + Amplify bot note for that sha); tickets already On nightly / To test / Release candidate / Rejected (merged) are never looked at, neither by `verify` nor by `mr` follow-ups (comment on a merged ticket = nothing happens)
  ▼                         → fresh claude -p "/cf-holdout-ui" runs holdout scenarios in Playwright (label `env-<name>` and that env is up: there, the MR's code redeployed into it, no Amplify note needed; FE ticket without the label whose BE dependency (blocker link or epic sibling) is not on nightly: env-auto sets it first, see Dyn-env lane; else any BE repo / cross-repo: one throwaway dyn-env, all BE MR images + FE branch). Every `claude -p` runs with `MCP_TIMEOUT=120000` and Playwright via `npx --prefer-offline`: the default 30 s connect limit was blown under a parallel build (15 of 15 `CONNECT_TIMEOUT` runs, Nov 2026); a server that still fails to connect logs `⚠️ mcp <name> failed`
  │                         can-frontend only, no `env-` label, holdout not `NONE:` → no verifier (no reachable preview): worker writes the report itself, every scenario `blocked — NO PREVIEW`, all of them go to `unverified/<KEY>.md` → [ai-verified], hand-check in the quiz
  │                         verifier tags every blocked scenario `DATA:` (fixture lacks the data) or `ENV:` (browser, login, preview, env). Blocked + ≥1 pass + no fail / not run + every blocked one `DATA:` → counts as pass: those lines go to `unverified/<KEY>.md` → [ai-verified], log `unverified: S7,S8`, Slack line, quiz asks me how I checked each. Next verify of the same ticket (holdout file unchanged since the report) gets `skip=<KEY>.data-skip`: the `DATA:` lines are copied, not re-run. `touch holdout/<KEY>.md` = re-run them (fixture added)
  │  factory-worker          BE refactor (holdout/KEY.api.md exists): after UI verify passes → "/cf-holdout-api" runs the same requests on 2 dyn-envs (nightly vs MR images, same seed), any diff = fail
[ai-verified]  MR note with per-scenario verdicts
  │  fail → 1 retry: "/cf-ticket-headless KEY --verify-fix <report>" pushes fix [ai-verify-retry] → re-verify, failed scenarios first (`failed=S3,S5`), one still failing → rest not run. Fix round pushed nothing → [ai-verify-failed], no re-verify
  │                         scenario contradicts ticket/PLAN.md → implementer only appends `S#: reason` to holdout-suspect/KEY.md (mr follow-up too, on my comment) → amend: "/cf-work-headless KEY --holdout-amend" (sonnet, sandbox dir with copies of TICKET/PLAN/holdout/verify report, code + notes + git denied) writes holdout-proposed/KEY.md + .why, or only .why (kept) → [ai-verify-failed] ✏️ holdout proposal, verify blocked until I press A / X in dash
  │  blocked and no scenario got a pass / fail (browser down, no login, preview dead) → auto-retry, max `BLOCKED_MAX` (2) per verify, `BLOCKED_WAIT` (10) min apart (counter `<KEY>.blocked-retry`, log line `blocked-retry: N/2`); UI verify only; never when no scenario can run (`DATA: missing`, waiting does not create data)
  │  fail again / blocked with some scenarios run (other than data-only above) / still blocked after the retries / crash → [ai-verify-failed] = needs me, no review happens (`factory-worker KEY --review` by hand)
  │  factory-worker          stage review (whole MR first, later only the delta since the last review): fresh claude -p "/cf-review" (opus, read-only) reads the MR diff vs PLAN.md + the injected repo conventions (a written-rule violation in added lines = must-fix) (+ `GLOSSARY.md` where one exists: new or renamed name with an `_Avoid_` word = must-fix; prose, UI copy, unrenamed legacy name = should-fix), every finding with a failure scenario, posts ONE MR note
  │                         clean / should-fix only → [ai-reviewed]. blocking → [ai-review-retry] → stage rfix ("/cf-ticket-headless --review-fix <report>", impl slot) fixes EVERY finding once, then "/cf-review … prior=<KEY>.review.prev.md since=<heads>" re-checks only the pushed delta (fixed / open / new per prior finding)
  │                         re-check clean or minor → [ai-reviewed] + verify labels dropped (verify runs again on the fixed code). pushed nothing / still blocking / crash → [ai-review-findings] = needs me (verify still runs)
[ai-reviewed]  (+ ai-verified)  cf-factory gate pending "quiz Michal"
  │  me: Q in factory-dash → /cf-quiz (sonnet, interactive) → pass
[ai-quizzed]   gate green; new commits drop it (re-quiz = delta only)
  │  I review, mark ready, add reviewers, merge; release moves ticket to Done
  ▼  factory-worker          ticket Done → resolves every sentry-* issue (haiku + Sentry MCP)
[ai-done]  → next morning sentry-triage picks the next issue
```

**Grill phase (opt-in, `GRILL=1` in `~/.config/sw-factory.env`; unset = off, nothing above changes).** `/cf-work-headless --grill` first decides whether the ticket has blocking gaps (the answer would change the plan or the scenarios). Yes → ONE Czech Jira comment (`Zjištěno` + numbered `Otázky`, each with a recommended answer and tradeoff), label `triage` → `needs-answer`, Slack line, no PLAN.md. I reply `ok` (take every recommendation) or `ok, 3 jinak: …`; the reply (≤2 min, or `R` in `factory-dash`) re-runs the analysis. More questions → another round (no cap, rounds run until no blocking gap is left); none → PLAN.md + scenarios, `needs-answer` → `triaged`, plan gate as before. `ai-fix` is ignored while `needs-answer` is set. No answer for `GRILL_REMIND_HOURS` (24) → one Slack reminder per round. Sentry tickets are asked only when the root cause stays unconfirmed. Worker ↔ skill handshake: `<KEY>.grill` (`ASK` | `READY`) + `<KEY>.round` in the state dir. What the code revealed persists in `notes/<KEY>.md` `## Analysis`, so a round re-explores only what the answers changed.

**Requirements don't get lost between rounds (DEV-3873 retro, 2026-11-23).** Every round is a fresh session that sees `TICKET.md` + `notes/<KEY>.md`, never what an earlier round read. So:

- Sources (Confluence spec, KB refinement record, …) are read once, by the first round that meets them, and extracted lossless into the notes: `## Sources` + `## Acceptance criteria` (one row per behaviour or rule, quoted verbatim). Later rounds read the extract, not the source.
- `PLAN.md` `## Requirements` ends with `### Acceptance criteria`: that table + `Where` (plan step + scenario, or `OUT: why + who decided`). Missing heading → `triage-failed`. Binding for impl (a row without a diff line = unfinished, not a deviation), must-fix in `cf-review`; the plan comment lists the `OUT` rows.
- A reply of mine that reads two ways is a question again (grill) / `[?]`, never an `[!]`.
- Ticket says "like screen X" → `### Reuse` in `## Implementation`: X's building blocks, each `reuse` / `not needed` / `new instead` + why. Re-implementing a `reuse` row = must-fix.
- No runnable holdout scenario → blocking grill question at triage, and `impl` is refused (flow above).
- `cf-work` sizing: description under 80 chars = nothing to size by → `opusplan` (analysis on opus), not `sonnet`.

**Dependabot lane (epic DEV-3865).** Dispatcher spawns stage `deps` (`deps-triage`, no LLM) when `.deps-checked` is older than `DEPS_EVERY` min (30). Per repo (cf-farm-frontend, can-frontend): gate = an open `deps` ticket whose MR is not merged/closed (MR = key in its title, a mention in another MR's description doesn't count; none yet = blocking) → skip the repo. Else the oldest open MR labelled `dependencies` by the project bot (`project_<id>_bot…`) without a ticket (`labels = dependabot-<repo>-<iid>`) and whose title doesn't match the repo's `SKIP` regex (can-frontend: `firebase`, never bumped); pipeline still running → wait. Major = a `package.json` dependency whose major (or `0.x` minor) changes in the diff. One Technical ticket per run and repo, Czech, assigned to me, active sprint, parent `DEPS_EPIC` (DEV-3865), labels `deps` + `repo-<repo>` + `dependabot-<repo>-<iid>`, description = MR link, CI state, majors, the bot's release notes. Worktree via `cf-work KEY <bot branch>`, so the ticket branch starts at the bot's commit.

| bot MR | labels | next |
|---|---|---|
| minor/patch, pipeline green | `triaged` + `ai-fix` | no analysis, no plan gate: template `PLAN.md` (rebase onto develop, no source change) + `holdout-templates/deps-<repo>.md` copied to `holdout/<KEY>.md`, both attached, 🤖 Jira comment → `impl` next tick |
| major or pipeline red | `triage` | analysis (`/cf-work-headless`, deps section: breaking changes vs our usage, smoke of the screens using the packages) → plan gate as usual |

Then the usual flow. `impl` rebases the bot's commit onto develop and regenerates the lockfile (`cf-ticket-headless` deps section); once our draft MR exists the worker closes the bot MR with a 🤖 "superseded by" note. Shortcut fails (worktree) → labels back to `triage`. The worker tees each run's output to `deps.status`; `factory-dash` shows it as the `deps` pill in the header (last run, per-repo result in the tooltip and in status `i`; yellow = blocked / waiting, red = failed) and puts a `deps` badge on the lane's tickets.

Any other ticket (feature, chore, manual bug): I add labels `triage` + `repo-<dir>` → same flow from `[triaged]` on; PLAN.md starts with Requirements instead of "Belongs in Sentry?". On Done it just gets `ai-done`.

One Sentry issue at a time: `sentry-triage` skips the day (Slack note) while any ticket with a `sentry-*` label and labelled `triage|triaged|triage-failed|ai-fix|ai-mr|ai-fix-failed` is not Done AND its MR is not merged/closed (`glab mr list --all --search KEY` in the `repo-*` repo, only MRs with KEY in the title count; no MR at all = blocking, so failed/unfixed tickets block until closed). Jira status doesn't matter — `On nightly` with a merged MR is fine. Sentry resolve waits for status `Done` (prod release), not Release candidate; a 404 (issue deleted/merged) counts as resolved, model answer logged to `<KEY>.log`. Stamp `~/.local/state/sw-factory/sentry-today` = already ran today (delete to re-run).

Failures: `triage-failed` / `ai-fix-failed` label. Every run where I became on turn posts a summary to Slack `#zimmi-claude` via `~/.claude/scripts/slack-post.sh` (incoming webhook, no LLM); progress lines go to the ticket log only (see Notifications). Quiet timer runs → no post, but if nothing posted for 3 h a "sw-factory alive — N quiet runs since X" heartbeat goes out (state `.posted` mtime + `.quiet` in `~/.local/state/sw-factory`). No heartbeat for >3 h during 06–19 weekdays = worker dead → `systemctl --user status sw-factory`.

## Labels

| label | meaning | set by |
|---|---|---|
| `triage` | new, waiting for analysis | sentry-triage, or me on any ticket |
| `sentry-<SHORTID>` | linked Sentry issue(s), full short id (dedup key) | sentry-triage |
| `repo-<dir>` | repo under `~/work/cleverfarm/repos` | sentry-triage, deps-triage |
| `deps` | ticket from a dependabot MR: analysis / impl use their deps sections, worker closes the bot MR after impl | deps-triage |
| `dependabot-<repo>-<iid>` | the bot MR (dedup key, read by `close_bot_mr`) | deps-triage |
| `needs-answer` | grill phase (`GRILL=1`): questions posted, no plan yet. Blocks `ai-fix`; my reply re-runs the analysis | factory-worker |
| `triaged` | PLAN.md + comment done | factory-worker |
| `triage-failed` | analysis or reply follow-up failed (no repo, worktree, crash); parked, no retry loop, re-add `triage` | factory-worker |
| `ai-fix` | **my approval** to implement PLAN.md; unanswered `[?]` → its `Default:` binds impl | me |
| `ai-mr` | draft MR exists | factory-worker |
| `ai-reviewed` | independent review done (after `ai-verified`): clean, should-fix only, or its findings were fixed and the delta re-check passed. Dropped with `ai-verified` when an `mr` follow-up or `rfix` pushes new commits; the next review covers only the delta | factory-worker |
| `ai-review-retry` | reviewer found must-fix issues, `rfix` will fix every finding once, then re-check the delta | factory-worker |
| `ai-review-findings` | findings still open after the one fix round (nothing pushed, or re-check still blocking) or the reviewer crashed: read the MR note / `<KEY>.review.md`. Remove the label to re-run the review | factory-worker |
| `ai-fix-failed` | agent bailed out (see its Jira comment) or no MR. A run that ended without a Jira comment of its own (bot comments older than the run don't count) gets one posted by the worker (its final answer). Implementer runs with `BASH_MAX_TIMEOUT_MS=2700000` so tests run in the foreground; a background job dies with `claude -p`. `mr` follow-up stuck like that stays `ai-mr`, my next reply re-runs it | factory-worker |
| `ai-verify-retry` | holdout failed once, fix pushed, waiting for re-verify | factory-worker |
| `ai-verified` | holdout scenarios pass on the preview, or pass except `DATA:`-blocked ones / can-frontend (listed in `unverified/<KEY>.md`, hand-checked in the quiz); gates `review`. Dropped (with `ai-verify-failed` / `ai-verify-retry`) when an `mr` follow-up or `rfix` pushes new commits, so `verify` runs again, and `ai-reviewed` goes too (delta re-review after verify); other review labels stay | factory-worker |
| `ai-verify-failed` | holdout failed after retry, blocked by anything but data only (after 2 auto-retries when no scenario ran), or verifier crashed | factory-worker |
| `ai-quizzed` | I passed `/cf-quiz` on the current MR code; gates `cf-factory` green. Dropped with the verify labels when new commits land | /cf-quiz (me) |
| `ai-done` | ticket Done, Sentry issues resolved, dyn-env the factory created for it destroyed | factory-worker |
| `env-<name>` | verify runs on the existing dyn-env `<name>` with the MR's code redeployed into it; env gone → as if the label weren't there. Changed after a verify → verify labels dropped, verify runs again | factory-worker (`env:` comment, env-auto: FE ticket whose BE dependency is not on nightly) or me |
| `ai-gate-skip` (GitLab MR label) | override: `cf-factory` merge gate goes green whatever the Jira labels | me (`s` in factory-dash) |

## Layout & dependencies

`sw-factory/` = everything only the factory uses: `bin/` (→ `~/bin`), `skills/` (→ `~/.claude/skills`), `dash/` (`go build` → `~/bin/factory-dash`), `systemd/` (→ `~/.config/systemd/user`), `map/` (factory-map artifact source). `install.sh` does the symlinks + build. Skills resolve by name (`/cf-review`), so repo layout never reaches runtime.

Shared with interactive work, stay outside: skills `cf-mr`, `check-layout`, `cf-work`, `cf-ticket`, `cf-worktree`, `cf-epic`, `cf-feedback` + `standards/`, `my-voice.md` (`claude/cleverfarm/`); `bin/jira`, `bin/cf-work`; `claude/user/scripts/slack-post.sh`. Plugins: `cf-dyn-env`, `mattpocock-skills` (`code-review`, `resolving-merge-conflicts`).

## Files

| path | what |
|---|---|
| `~/dotfiles/sw-factory/bin/sentry-triage` | stage 1: open-ticket gate + daily stamp, then claude -p (sonnet) + Sentry MCP + `jira` → ticket. `--dry-run` (ignores gate + stamp), `MAX=` (default 1) |
| `~/dotfiles/sw-factory/bin/sentry-triage-now` | force a Sentry run today: drop `sentry-today` stamp + start `sw-factory.service` (no block; dispatcher spawns the `sentry` stage) |
| `~/dotfiles/sw-factory/bin/factory-worker` | stages 2–7: triage, reply follow-up, implement, holdout verify (+1 retry), resolve Sentry on Done, rebase my conflicting MRs (dispatcher runs only), `sentry` stage (runs `sentry-triage`); Slack summary (notes moved away atomically before posting, so stages can append meanwhile). No args = dispatcher, see Slots. `factory-worker DEV-X` = spawn one ticket's stage (implements if it has `ai-fix`; `factory-worker DEV-X --verify` = verify its MR now, `--review` = independent review now). `MAX=` (new tickets looked at per tick, default 10), `SLOTS=`, `IMPL_SLOTS=`, `MR_SLOTS=`, `REVIEW_SLOTS=`, `VERIFY_SLOTS=`, `CONFLICT_EVERY=`, `GL_FULL_EVERY=` (my GitLab MR notes: every tick one Events API call `events?action=commented` picks the tickets with a note of mine newer than `handled-mr/<KEY>`, 🤖 and notes pinging others dropped, MR → ticket from the source branch cached in `.gl-mr-keys`; only those get the thread check. Full scan of every `ai-mr` MR every 10 min and when the events call fails, log `lookup failed: gitlab events`. Before 2026-11-23 a full scan every 2 min, 13 s of a ~30 s tick). Holdout verify of a BE ticket holds its verify slot ~30 min (dyn-env create). Analyze + implement open a detached window per worktree in tmux session `sw-factory` (`tail -f <KEY>.log`) via plain `tmux new-window -d`. No tmux server = skipped |
| `~/dotfiles/sw-factory/bin/deps-triage` | dependabot MRs → tickets (Dependabot lane), no LLM; `--dry-run` prints the ticket it would create per repo, changes nothing. Run by stage `deps` |
| `~/dotfiles/sw-factory/holdout-templates/deps-<repo>.md` | smoke scenarios for green minor/patch bumps (cf-farm-frontend: `FARM-QA` screens; can-frontend: `NONE:`, no reachable preview) |
| `~/dotfiles/sw-factory/bin/factory-stats` | `factory-stats [days=7]`: yield per ticket touched in window — stages, bail rate, verify pass rate, reviewer notes per merged MR, days ticket→merge, $ total + per merged MR (from `cost=$` in `<KEY>.log`). Posted to Slack every Monday by `nightly-claude.sh` |
| `~/dotfiles/sw-factory/bin/factory-compare` | `factory-compare [cutover]` (default 2026-11-05 13:35, the context-waste commit): per-run medians before vs after per stage from the logs: cost, tool calls, peak + first-turn context (old runs: from the transcript named on the 🆔 line, cached in `peaks.tsv`), verify calls before the first screenshot, verify / impl / analyze runs per ticket, same-pipeline re-verifies, `TICKET.md` bytes, notes files present, reviews with conventions injected, review runs per ticket, findings per review (must / should / convention, from the `findings:` log line), verify runs without Playwright (`CONNECT_TIMEOUT` / `⚠️ mcp playwright`), blocked runs accepted as data-only, can-frontend verifies skipped, memory per stage from `mem.tsv` (median / p90 `memory.peak`, lowest MemAvailable at a stage start, stages ending with swap in use), dispatcher tick from `ticks.tsv` (s until stages spawn, total, events vs full GitLab check), MR follow-ups per ticket (from the `variant: --mr-notes-after` log line; transcript baseline 3, p90 14). For the review-conventions change pass its merge time as cutover. Posted to Slack every Monday next to factory-stats; read nothing before ~15 tickets after the cutover |
| `~/dotfiles/bin/cf-work` | worktree setup, one per repo (`--repos a,b` > `repo-*` labels > mapped labels `frontend`/`agroevidence`/… > cwd); same branch everywhere, claude in primary worktree + `--add-dir` siblings; `--no-launch`, `--no-claim` (skip assign + In Progress); `--comments-after <id>` (worker passes `handled/<KEY>`): `TICKET.md` puts the human comments above that id under `## New comments` right after the description, older human ones under `## Earlier comments`, 🤖 comments as one-line stubs except the last (full), so a stage reads only what is new |
| `~/dotfiles/sw-factory/skills/cf-work-headless` | root-cause analysis → PLAN.md + holdout scenarios (Given = fixture id from `cf-farm-frontend/tests/FIXTURES.md` + path with real ids, so the verifier opens it directly) + comment; follow-up mode on replies; grill mode. Interactive twin `/cf-work` carries none of it |
| `~/dotfiles/sw-factory/skills/cf-ticket-headless` | implement PLAN.md, verify, self-review, `/cf-mr --headless`; variants: MR follow-up, `--verify-fix`, `--review-fix`, duplicate close. Interactive twin `/cf-ticket` carries none of it |
| `~/dotfiles/claude/cleverfarm/skills/cf-mr` | MR body = mattpocock `pr` (`## Summary` visual, `## Evidence` with the check-layout screenshots uploaded via `glab api projects/<id>/uploads`, `## Merge Danger`) + our prefix, changed-screens list, `Closes`, footer; the reviewer checks Merge Danger against the diff (understated = should-fix). `--headless`: `glab mr create --draft --description-file`, no reviewers, stays In Progress, FE plays Nightly preview |
| `~/dotfiles/sw-factory/test/worker-rules.zsh` | `zsh test/worker-rules.zsh`: checks the worker's data-only rule (`data_only`, `DATA_RE`), the can-frontend scenario parsing (`NOPREVIEW_SED`) and the "run already posted" regex of the restart (`POSTED_RE`) on fixtures |
| `~/dotfiles/sw-factory/skills/cf-holdout-ui` | holdout verifier: Playwright login as `fe-automation@cleverfarm.cz` (agent types the secret names, MCP `--secrets ~/.config/cleverfarm/e2e.env` fills the values, the password never enters the model context), runs scenarios, rewrites `<KEY>.verify.md` after every scenario (`VERDICT: running`, final `pass/fail/blocked`; a run killed mid-way leaves `running` = crash), records a video when factory-worker passes `video=` (own Playwright MCP `@0.0.83 --headless --caps devtools --executable-path /usr/bin/chromium`, `--strict-mcp-config`; interactive runs don't record), MR note. Never edits code. BE: creates + destroys `<key>-verify` dyn-env |
| `~/dotfiles/sw-factory/skills/cf-review` | independent reviewer: fresh opus session, read-only, `/mattpocock-skills:code-review` against `PLAN.md` + standards plus its own security / data / performance / test checks, every finding grounded in code with a failure scenario; writes `<KEY>.review.md` (`VERDICT: clean/minor/blocking`) and one MR note. With `prior=<KEY>.review.prev.md since=<worktree>@<sha>,…` (after `rfix`): incremental re-check of the pushed delta only, each prior finding marked fixed / open / dropped, new ones only from the delta. Never edits code |
| `~/dotfiles/sw-factory/skills/cf-env-resolve` | resolver of the Dyn-env lane: my `env:` comment → one `RUN …` / `ASK …` line, read-only |
| `~/dotfiles/sw-factory/skills/cf-holdout-api` | differential API verifier for BE refactors: two dyn-envs (nightly vs MR images, same seed), same request sequences from `holdout/<KEY>.api.md`, writes `<KEY>.verify-api.md` (`VERDICT: pass/fail/blocked`), token from a nightly Playwright login |
| `~/dotfiles/sw-factory/systemd/sw-factory.{service,timer}` | symlinked to `~/.config/systemd/user/`; dispatcher Mon–Fri every minute 06–18 (idle run = a few Jira queries, no LLM; failed Sentry triage retries ≤ every 15 min). `factory-dash` actions start it right away, `Persistent=true`, timeout 10 min. Stages are transient `sw-factory-<KEY>-<stage>` units, not files |
| `~/dotfiles/sw-factory/systemd/factory-dash.service` | web dashboard server (`factory-dash serve`), enabled + restarted by `install.sh`; listens on 127.0.0.1:7777 only, refuses other `Host` / `Origin`, needs the cookie set from `~/.local/state/sw-factory/dash.token` (0600; `factory-dash` opens the signed-in URL). Refetches Jira + MR state every minute. After a dash change: `./install.sh`, or `go build -o ~/bin/factory-dash .` in `sw-factory/dash` + `systemctl --user restart factory-dash`. Logs: `journalctl --user -u factory-dash` |
| `~/dotfiles/sw-factory/dash/ask.go` | `q` in `factory-dash`: ask in plain words ("what's the state of the npm to pnpm migration?"), no ticket key needed. `POST /api/ask` → `claude -p --model sonnet --tools ""` (no tools, no MCP: read-only by construction) picks ≤3 tickets from the dash rows (no match → one Jira text search), then answers from their description, comments, `PLAN.md`, verify + review report and the last 80 log lines. One-shot, no memory between questions, 2-3 LLM calls per question |
| `~/.config/jira.env` | `JIRA_EMAIL`, `JIRA_TOKEN`, shared by all my tools (0600, not in git) |
| `~/.config/sw-factory.env` | factory-only settings, e.g. `GRILL` (0600, not in git) |
| `~/.config/cleverfarm/e2e.env` | CF test users `FE_AUTOMATION_*`, `FE_TEST_*`, `FE_JOHN_DEERE_*`, `FE_PTX_*` (0600, not in git), copy of those lines from cf-farm-frontend `.env` (E2E still reads `.env`; rotate both). Playwright MCP `--secrets` file (user scope + factory-worker `PW_ARGS`) and `token.mjs` source |
| `~/.local/state/sw-factory/` | `<KEY>.log`, `<KEY>.stream.jsonl` (last implement run only, source of the worker's fallback comment; full transcripts of every stage live in `~/.claude/projects/<cwd slug>/<session>.jsonl`), `<KEY>.verify.md`, `<KEY>.verify-media/` (video + MCP output), `ticks.tsv` (one line per dispatcher tick: start, s until stages spawn, total s, GitLab note check `events` / `full`, stages spawned), `.gl-mr-keys` (`<project id>:<iid> <KEY>` or `-`), `mem.tsv` (one line per stage, written by `run_stage` inside the stage unit: end time, stage, key, cgroup `memory.peak` MB (incl. page cache, same accounting `MemoryHigh` would use), MemAvailable MB at start / end, swap used MB at start / end, exit code), `unverified/<KEY>.md` (scenarios counted as verified without running: `DATA:` / `NO PREVIEW:` lines, read by the quiz + dash; deleted when verify starts or labels drop), `<KEY>.data-skip` (`DATA:` lines of the last report passed to the verifier as `skip=`), `<KEY>.review.md` (+ `.review.prev.md` = the report `rfix` worked from, input of the re-check), `holdout/<KEY>.md` (scenarios; analysis may write, implementer denied), `holdout-suspect/<KEY>.md` (scenarios the implementer flags, input of amend), `holdout-proposed/<KEY>.md` + `.api.md` + `.why` (amend output waiting for dash `A` / `X`; implementer + conflict resolver denied write, verify + review denied read), `notes/<KEY>.md` (ticket memory: `## Analysis` written by the analyzer with file:line evidence and kept across grill rounds and replies, then the implementer's change map, decisions, deviations, gotchas, round log; every analyze / impl round reads it first, `Read` denied to review + verify), `sentry-*.json`, `sentry-last.txt`, `sentry-today` (daily stamp), `notes.txt` (pending Slack lines), `conflicts/<project id>-<iid>` (last handled MR head) + `.md` (resolver report), `slots/` (slot pool, see Slots), `locks/<KEY>` (tickets), `stalled/<KEY>.since` / `.alert` (verify-not-ready clock + last Slack alert), `handled/<KEY>` (id of the newest human Jira comment the last successful `new` / `reply` / `impl` / `mr` stage had seen when it started; a comment is pending while its id is higher, so a bot comment posted later can't bury it. No file = old rule: pending while the last comment is not a 🤖 one) |
| `~/.config/claude/slack-webhook` | incoming webhook URL for `#zimmi-claude` (0600, not in git) |
| `~/dotfiles/gitignore` | ignores `PLAN.md`, `TICKET.md`, `install.log`, `.codegraph/` so MRs don't pick them up |

## Setup

```
~/dotfiles/install.sh
systemctl --user enable --now sw-factory.timer
```

Prereqs (already done):
- claude.ai connectors authenticated: Sentry (`/mcp`), Atlassian, Slack — headless `claude -p` uses them with my login
- `glab auth status` ok, SSH to gitlab works without agent (systemd has no `SSH_AUTH_SOCK`)
- no linger: runs only while I'm logged in

## Operate

| command | what |
|---|---|
| `systemctl --user list-timers 'sw-factory*'` | last/next dispatcher run |
| `systemctl --user list-units 'sw-factory-*'` | running stages |
| `journalctl --user -u sw-factory --since today` | dispatcher output (spawned / no free slot) |
| `systemctl --user start sw-factory.service` | run now |
| `systemctl --user disable --now sw-factory.timer` | turn off |
| `tail -f ~/.local/state/sw-factory/DEV-XXXX.log` | live per-ticket progress: 🔧 tool calls, 💬 agent text, 📏 context size, ✅ exit + cost + turns + peak |
| `git -C ~/dotfiles tag -a sw-factory-pre-<label>-<date> -m "…"` | snapshot before a factory change (bin + skills + docs are symlinks, so the tag is the whole factory; annotated, note the mattpocock-skills version in the message). Newest first: `git -C ~/dotfiles tag -l 'sw-factory-*' --sort=-creatordate`; tags `cf-triage-*` predate the `sw-factory/` move, roll back with `-- bin claude/cleverfarm go` |
| `git -C ~/dotfiles checkout sw-factory-<label>-<date> -- sw-factory claude/cleverfarm` | roll the factory back to that snapshot (takes effect at once; `go build` only if `sw-factory/dash` changed). Back to head: `git checkout master -- sw-factory claude/cleverfarm` |
| `git -C ~/dotfiles checkout sw-factory-pre-web-2026-11-17 -- sw-factory/dash install.sh && (cd ~/dotfiles/sw-factory/dash && go build -o ~/bin/factory-dash .) && systemctl --user disable --now factory-dash` | back to the TUI `factory-dash` (state before the web dashboard) |

Glossary (mattpocock `domain-modeling`, FE only for now): `GLOSSARY.md` at the repo root of cf-farm-frontend, format + `_CZ_` rule in its `docs/agents/domain.md`. Glossary beats code vocabulary and PLAN.md wording. Read at `new`/`reply` (PLAN.md `## Terms` + `_Avoid_` self-check), at every `impl` round incl. MR follow-up / verify-fix / review-fix (touched legacy `_Avoid_` names renamed in scope, `_Avoid_` grep on the diff gates verify), checked at `review` (new `_Avoid_` name = must-fix). Rollback of these rules: `git -C ~/dotfiles checkout sw-factory-pre-glossary-2026-11-16 -- sw-factory claude/cleverfarm`. Repos without the file: nothing changes. Interactive `/cf-ticket` phase 0 runs `grill-with-docs` (grilling + domain-modeling) instead of plain grilling.

## Test one ticket by hand

1. `sentry-triage --dry-run` → preview (shows gate verdict too); `rm ~/.local/state/sw-factory/sentry-today; sentry-triage` → 1 real ticket (only if nothing open)
2. `factory-worker DEV-XXXX` → analysis (any DEV ticket with a `repo-*` label works)
3. comment in Jira, `factory-worker` (no args) → reply answered
4. add `ai-fix`, `factory-worker DEV-XXXX` → draft MR
5. ticket Done → `factory-worker` (no args) → Sentry issues resolved, label `ai-done`
6. `deps-triage --dry-run` → which bot MR would become which ticket; `rm ~/.local/state/sw-factory/.deps-checked` → next dispatcher tick runs it for real

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
- Analysis: read-only tools (ctx_*, codegraph, `git log/show/blame/grep`, Sentry `get_sentry_resource` + `search_errors`) + write only `PLAN.md`, `notes/<KEY>.md`, `holdout/<KEY>.md` + `<KEY>.api.md`, `<KEY>.grill` + `jira add-comment`.
- Implement: denied `git push --force/-f`, push to develop/staging/production, `reset --hard`, `glab mr merge/approve`, writes to `holdout/` + `holdout-proposed/`. Protected branches enforce it server-side too.
- Holdout amend (separation of duties: the implementer judged by the scenarios can only flag them): cwd = sandbox `amend/<KEY>/` with copies, no MCP, denied `Bash`, `Agent`, web, `Edit`, `Read` of `~/work/cleverfarm/repos/**` + `notes/`, write only `holdout-proposed/<KEY>.*`. Nothing lands in `holdout/` without my `A`.
- Implementation only after I add `ai-fix`. MR always draft, no reviewers.
- Agent bails (comment, no MR) if root cause doesn't hold, scope grows, or tests can't go green.
- Only my comments are instructions: a Jira comment counts as human only when `author.accountId` = `jira GET /rest/api/3/myself` (worker exits with `lookup failed` if that call fails), MR notes only with my GitLab username. My comment or note that @mentions a colleague (Jira mention of another accountId, GitLab `@user` ≠ me) is addressed to them, not the factory. Comments by others and those never start a run and land in `TICKET.md` under `## Comments by others or to colleagues` as context only.
- Context per stage: every `claude -p` gets `--settings` disabling the plugins no stage uses (caveman, ponytail, slack, cf-planning, cf-release, cf-kb, cf-eod, cf-bug-report, cf-cli-skill, cf-investigator, cf-cc, typescript-lsp; `HEADLESS_SETTINGS` in factory-worker). impl / review / conflict / verify run `--strict-mcp-config` with lean-ctx + codegraph only (`mcp_config`; Playwright added for impl's `/check-layout` and for verify). `new` / `reply` keep every MCP server: Sentry is a claude.ai connector and strict mode drops connectors. First-turn context ≈ 50K tokens instead of 63K; headless skills are split from the interactive ones so no checkpoint / plan-mode text is loaded.

## Notifications

- Slack `#zimmi-claude` via incoming webhook (`slack-post.sh`, same as nightly digests), one message per run with a `note`; notes kept until curl succeeds.
- `note` (Slack) only where I am on turn: questions / PLAN ready, any failure / crash / block, review verdict that hands me the quiz, holdout proposal, stuck alerts, conflicts I must resolve. Same text for the same ticket within 24 h (`sent/<KEY>`) → not posted again, logged as `Slack skipped` instead.
- `info` (no Slack, `  info: …` line in `<KEY>.log`, visible in the dash log tab): reply answered, draft MR, holdout pass, fix pushed / re-verify next, must-fix auto-fix next, released, agent rebase done.
- Dash sidebar: within a group, rows sort by when they entered that group (log writes don't move them); a row whose label changed since I last opened it shows a ● after its key (per browser, `localStorage` `seenLbl`).
- Webhook posts ping like any bot message. Jira comments come from my account → no Jira ping.

## Costs (DEV-2825 test)

create ticket ~$0.45 · analysis ~$0.65 · reply ~$0.4 · implement ~$1.6 (opus) · Sentry resolve (haiku) cents · Slack post free.

## Known gaps

- editing an old non-AI comment counts as a new reply if it's the last one
- merge gate runs only with the timer (Mon–Fri 06–18): a new head pipeline outside those hours has no `cf-factory` status until the next tick, so it is mergeable meanwhile
- MCP login expiry → runs fail. sentry-triage without `TRIAGE_DONE` → one Slack note per day (`sentry-failed-noted` stamp) with the `TRIAGE_FAILED` reason; fix Sentry auth via `/mcp` → claude.ai Sentry. Worker failures → `*-failed` labels / logs
- usage limit: `sentry-triage` / `deps-triage` run their own `claude -p` without the limit detection (skipped only once a ticket stage has written `.limit`); while `.limit` holds, the whole tick is skipped, so the merge gate and `done` wait too; no Slack line, only the journal + `<KEY>.log`
- Persistent catch-up can fire once on Saturday if the laptop was off at Friday's last slot (daily stamp keeps it to one ticket)
- a `triage-failed` / `ai-fix-failed` ticket blocks new issues until I close or relabel it; same for `deps` tickets per repo
- can-frontend tickets are not verified in a browser (no reachable preview: deploy jobs manual, Amplify domain >64 chars / missing hosted zone, CAN E2E creds only in Vault). `deps`: holdout `NONE:`, verify passes on CI alone, check the preview by hand. Others: scenarios become the quiz hand-check list (`NO PREVIEW`)
- `DATA:` vs `ENV:` is the verifier's call; a UI bug that hides existing data mis-tagged `DATA:` would pass verify. Backstop: the quiz makes me say how I checked each unverified scenario
- a `deps` ticket waiting at the plan gate for a week: the bot supersedes its MR with a newer version (new iid → new ticket once the gate frees); the old ticket still ships the older version, close it if the newer one is wanted instead
- dependabot itself must run: cf-farm-frontend schedule needs `PACKAGE_MANAGER_SET=npm` (DEV-3866), can-frontend a valid `GITLAB_ACCESS_TOKEN` (DEV-3867)

## Dyn-env lane

For a ticket whose frontend needs backend versions that are not on nightly. Three inputs:

- **Jira comment starting with `env:`** (mine only), free text (dash `R`): `env: nové prostředí, agroevidence z DEV-3809, FE DEV-3785` / `env: naklonuj dev-3657-pasture` / `env: použij dev-3657-pasture` / `env: nasaď agroevidence z DEV-3830` / `env: zruš to`. Any time, also mid-`impl`: stage `env` has its own lock (`locks/env-<KEY>`, unit `sw-factory-env-<KEY>-env`), takes one general slot until its pipeline ends.
- **label `env-<name>`** = which env the ticket verifies on. Set by the lane after `create` / `use`, by env-auto, or by hand.
- **env-auto**: the worker notices by itself that Amplify against nightly is not enough (below).

**Env-auto** (`env_auto`, first step of `verify_targets` for a ticket with `cf-farm-frontend` and no `env-` label). Dependencies = tickets that block it (Jira link "is blocked by") + the other children of its epic (the team keeps FE + BE stories of a feature in one epic and rarely links them: DEV-3873 had no link, only epic DEV-3657), each not Done and not On nightly / To test / Release candidate / Rejected. Per dependency its BE MRs: group MR search for the key, source branch `<key>` / `<key>-…`, any project except `*frontend`. No BE MR anywhere → `none`, Amplify as before. Only merged ones → hold verify (`BE of DEV-X merged, not on nightly yet`). Open ones → an env, found in this order:
1. `env-<name>` label on a dependency with an open BE MR, env up;
2. an up env `dynamic/<name>` of cf-dev-kubernetes whose name is the epic's or a dependency's key, or starts with it plus `-` (`dev-3657-pasture`).

One → label `env-<name>` on the ticket, Jira comment on it, and a comment on the dependency / epic the env belongs to (name or label): its preview now shows this FE branch, backend left as is. Then the usual verify with `deploy=1` redeploys only the FE. Several → hold, Slack line, I pick with `env: použij <name>`. None → `<KEY>.env-req` with a generated request (`env: nové prostředí <epic or dep key>, agroevidence z DEV-3809, …, frontend z větve <branch>`); the next tick runs stage `env` with it (`auto` instead of a comment id): same resolver, but only a `create` is accepted. Success → label on the ticket and on every dependency without an `env-` label + a comment there (they may use it too). State in `<KEY>.env-auto` (`none` / `wait` re-checked every `ENV_AUTO_EVERY` min, default 10; `req <env> <deps>` while the create runs; `failed …` → Amplify as before, `rm` the file to retry). Every round of verify (also `factory-worker DEV-X --verify`) goes through it; with `TRIAGE_DRY` it only prints `env-auto <KEY>: …` to stderr.

Stage `env`: `/cf-env-resolve` (sonnet, read-only: GitLab reads + `cf-env.sh list/status/log`, `cf-env.sh run` and `glab api -X` denied) turns the comment into one line in `<KEY>.env-cmd`, `RUN <action> <env> KEY=VALUE…` or `ASK <question>`. The worker checks the line (action in create / redeploy / destroy / wake / sleep / set-sleep / set-db-size / use, env `^[a-z0-9-]{3,40}$`, args `KEY=[A-Za-z0-9_.:,/-]*`) and only then runs `cf-env.sh run` of the cf-dyn-env plugin itself. Jira gets: the exact command + pipeline link at start, the URL or the failure at the end; `ASK` → the question, nothing started. `create` only when no env of that name is up; `destroy` only for an env in `env-owned/` of this ticket. Comment id in `handled-env/<KEY>`, running pipeline in `<KEY>.env-pipe` (a killed stage resumes the wait next tick). `env:` comments never reach the reply / mr lanes or TICKET.md.

Verify with the label (`verify_env`): env state from GitLab environment `dynamic/<name>` of cf-dev-kubernetes (one call). Up → FE-only tickets skip the Amplify-note wait, `/cf-holdout-ui … env=<name> deploy=1` redeploys only the ticket's code into it (`FRONTEND=true FE_BRANCH=<branch>`, BE repos of the ticket at their MR image), checks the branch head is the verified sha, never creates or destroys; redeploy fails → `blocked`. Gone → today's path (Amplify / throwaway env). Lookup failed → not ready this tick (`lookup failed: state of dyn-env <name>`). The label used is saved in `<KEY>.verify-env`; a different label on a verified ticket, or a `create` / `redeploy` / `use` by the lane, drops `ai-verified` / `ai-verify-failed` / `ai-verify-retry` (review and quiz labels stay). MR note ends with `**Preview**`: one complete URL per scenario.

Redeploy after push (`env_push`, DEV-3899): MR follow-up, review fix and verify fix compare `wt_heads` before / after. Frontend worktree head moved, ticket has `env-<name>`, env up (any env, owned or not, same as verify `deploy=1`), head not queued before (`<KEY>.env-redeployed`) → `<KEY>.env-redeploy` = `RUN redeploy <name> FRONTEND=true FE_BRANCH=<branch>`. Next tick stage `env` runs it (`redeploy` instead of a comment id): no resolver, no Jira comments, success → verify labels dropped + log line, failure → Slack line. Verify of the ticket holds while it is queued or its pipeline runs (`dyn-env <name> redeploy running`). Success writes `<KEY>.env-fe` = `<pipeline> <head>`. Verify of a frontend-only ticket then passes `deployed=1` instead of `deploy=1` (holdout skips its redeploy, keeps the branch head check) when the FE worktree head is still that head and the env's latest GitLab deployment (`deployments?environment=dynamic/<name>`, newest, success) is still that pipeline. Any other deploy since (another ticket's verify, an `env:` comment, a manual `cf-env.sh`, sleep / wake), a new push, a cross-repo ticket or a failed lookup → `deploy=1` as before. So one frontend build per push. BE-only pushes and conflict rebases: no redeploy, verify `deploy=1` covers them.

Ownership: `env-owned/<env>` (content = ticket key) is written only for an env the lane created (by comment or env-auto). Owner Done (stage `done`, or the dispatcher's `env_gc` every `ENV_GC_EVERY` min, default 10, which also covers owners outside the factory, e.g. a BE dependency) → destroy, unless another not-Done ticket carries `env-<env>`: then that one becomes the owner. Lookup failed → not destroyed, Slack line, next check retries. Envs with a running `<KEY>.env-pipe` are skipped. An env I pointed at, or one env-auto reused, is never destroyed and keeps the ticket's FE branch after verify. One FE per env: with `VERIFY_SLOTS` > 1 two tickets on the same env would overwrite each other (no per-env lock yet).

By hand: `factory-worker DEV-X --env` re-runs the latest `env:` comment. Not covered: the implementer's `/check-layout` does not use the env.

## Rollback

Usage-limit handling: `git -C ~/dotfiles checkout sw-factory-pre-limit-2026-11-23 -- sw-factory/bin/factory-worker && rm -f ~/.local/state/sw-factory/.limit*`

Dyn-env lane: `git -C ~/dotfiles checkout sw-factory-pre-dyn-env-2026-11-23 -- sw-factory bin/cf-work && rm ~/.claude/skills/cf-env-resolve/SKILL.md && (cd ~/dotfiles/sw-factory/dash && go build -o ~/bin/factory-dash . && systemctl --user restart factory-dash)`. Envs it created stay: `ls ~/.local/state/sw-factory/env-owned`, destroy by hand.

Env-auto: `git -C ~/dotfiles checkout sw-factory-pre-env-auto-2026-11-24 -- sw-factory/bin/factory-worker sw-factory/test/worker-rules.zsh && rm -f ~/.local/state/sw-factory/*.env-auto ~/.local/state/sw-factory/*.env-req`. Labels and Jira comments it made stay.

Redeploy after push: `git -C ~/dotfiles checkout sw-factory-pre-env-redeploy-2026-11-24 -- sw-factory/bin/factory-worker sw-factory/test/worker-rules.zsh sw-factory/skills/cf-holdout-ui/SKILL.md && rm -f ~/.local/state/sw-factory/*.env-redeploy* ~/.local/state/sw-factory/*.env-fe`.

Review-conventions work (branch `review-conventions`): `git -C ~/dotfiles checkout sw-factory-pre-review-conventions-2026-11-17 -- sw-factory claude/cleverfarm`

DEV-3873 retro (acceptance-criteria extract, reuse table, unrunnable-holdout gate, sizing): `git -C ~/dotfiles checkout sw-factory-pre-dev3873-retro-2026-11-23 -- sw-factory bin/cf-work`. Gate alone off per ticket: `touch ~/.local/state/sw-factory/<KEY>.blind`.

Auto-retry of blocked verify: `git -C ~/dotfiles checkout sw-factory-pre-blocked-retry-2026-11-23 -- sw-factory/bin/factory-worker sw-factory/bin/factory-compare` (or `BLOCKED_MAX=0` in `~/.config/sw-factory.env` = off). Measure: `factory-compare 2026-11-23T10:00:00+01:00`, row `verify runs ending blocked` (before: 29 %, 58 of 194, all parked)

Verify unblock (`MCP_TIMEOUT`, `--prefer-offline`, data-only → unverified + quiz hand-check, `skip=`, can-frontend without verifier): `git -C ~/dotfiles checkout sw-factory-pre-verify-unblock-2026-11-23 -- sw-factory && rm -rf ~/.local/state/sw-factory/unverified ~/.local/state/sw-factory/*.data-skip && (cd ~/dotfiles/sw-factory/dash && go build -o ~/bin/factory-dash . && systemctl --user restart factory-dash)`. Measure: `factory-compare 2026-11-23T22:00:00+01:00`, rows `verify runs ending blocked`, `verify runs without Playwright`, `blocked runs accepted as data-only`, `can-frontend verifies skipped` (before, since 2026-11-16: 35 % blocked, 27 of 76, 24 parked; 9 runs without Playwright). Over 4 weeks: 61 of 204 verify runs blocked ($72), ~13 nothing ran (Playwright timeout under a parallel build), ~27 every runnable scenario passed and only data was missing

Restart of a young run on a new comment + MR follow-up parallel to verify: `git -C ~/dotfiles checkout sw-factory-pre-restart-parallel-2026-11-23 -- sw-factory/bin/factory-worker && rm -rf ~/.local/state/sw-factory/running` (restart alone off: `RESTART_WINDOW=0` in `~/.config/sw-factory.env`). Measure: grep `restart: new comment` / `stale: commits moved` in `<KEY>.log`; comment → stage delay while a stage runs (before: 66 comments, 578 min, median 5.7, p90 16 in 4 weeks)

GitLab note check via Events API + `ticks.tsv`: `git -C ~/dotfiles checkout sw-factory-pre-gl-events-2026-11-23 -- sw-factory/bin/factory-worker sw-factory/bin/factory-compare && rm -f ~/.local/state/sw-factory/.gl-mr-keys` (or `GL_FULL_EVERY=1` in `~/.config/sw-factory.env` = full scan every tick). Measure: `factory-compare 2026-11-23T22:00:00+01:00`, rows `dispatcher tick: …` (before: dry profile ~30 s until spawn, GitLab scan 13 s of it; GitLab note → stage 1.3 min median, p90 2.0, since 11-17)
