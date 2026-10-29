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

1. **Plan gate** — Jira comment with PLAN + Given/When/Then scenarios. Wrong/unclear → reply in Jira (answered ≤5 min, PLAN + scenarios updated). Good → add `ai-fix`. The label approves plan AND scenarios — scenarios are what "done" means, read them.
2. **MR gate** — draft MR with verify verdict note. `ai-verified` → review diff, mark ready, add reviewers, merge. Want changes → comment in Jira, agent pushes to the same MR.
3. **Failures** — `triage-failed` / `ai-fix-failed` / `ai-verify-failed`: read the 🤖 Jira comment or `<KEY>.verify.md`; fix the plan/scenario and re-label (`triage` or `ai-fix`), take it over with `cf-work DEV-X`, or close the ticket.

**Merge conflicts** — every run rechecks ALL my open MRs (any repo, factory or not). Conflicting → rebase onto target in its worktree:

| MR | what happens |
|---|---|
| draft, not approved, only my commits | `pnpm-lock.yaml`-only → `--ours` + `pnpm install --lockfile-only` (no LLM); code → headless opus `/mattpocock-skills:resolving-merge-conflicts` (intent from PLAN.md + the develop commits' MRs), lint/typecheck/tests, `push --force-with-lease`, 🤖 MR note. Agent-resolved + `ai-verified` → label dropped → holdout re-verify |
| ready or approved | Slack only. Add MR label `ai-rebase` = my OK → resolved next run, label removed (approvals don't reset on push here) |
| others' commits / dirty or diverged worktree / >10 files changed on both sides (`CONFLICT_MAX_FILES`) | Slack only → resolve by hand |

Failed resolve → rebase aborted, Slack line. One Slack line per MR head (also "no local repo"); retry same head = `rm ~/.local/state/cf-triage/conflicts/<project id>-<iid>`.

**Interactive `/cf-ticket`**: checkpoint 1 = approve plan + scenarios (ExitPlanMode), checkpoint 2 = verified draft MR → I say "ready" → it adds reviewers + logs time. `--step` for risky work.

**Interactive `/cf-epic`**: checkpoint 1 = SPEC.md, 2 = ticket breakdown + scenarios, 3 = acceptance on preview → I say "accepted" → stacked review MRs.

**Weekly (Monday Slack "Factory yield")** — bail rate up → plans too vague, push harder at plan gate. Verify failures → scenarios or implementer weak. Reviewer notes per MR up → run `/cf-feedback` after merges so the reviewer learns. `$`/merged MR spikes → check `<KEY>.log`.

**After every merge with reviewer comments**: `/cf-feedback` → rule lands in `standards/<repo>.md`, the reviewer agent enforces it next time. Lint-able → offer ticket.

**Debugging a run**

| want | how |
|---|---|
| all tickets at a glance (stage, cost, MR, live last tool call) | `triage-dash` TUI: sidebar of tickets + pane with details and `tab` log / PLAN.md / Jira comments (`^u`/`^d` or wheel scrolls); gates without Jira: `a` approve (add `ai-fix`), `R` reply (vim → Jira comment), `t` retry failed label, `w` run worker now, all y/n confirmed; j/k, enter opens MR (else Jira), `J`/`m` Jira/MR, `l` log in less, `c` resume session interactively (new tmux window named `<KEY>`); click KEY/MR (tmux mouse on: `prefix m`) |
| is something running now | `workmux dashboard` — headless runs show as their `cf-triage` window (working → done) while active; finished ones drop off |
| watch it live | `tmux switch-client -t cf-triage` (one window per worktree, `tail -f` of the log) or `tail -f ~/.local/state/cf-triage/<KEY>.log` (🔧 tool calls, 💬 agent text, ✅ exit + cost) |
| ask the agent why it did X | log line `🆔 session <id> — reopen: cd <wt> && claude --resume <id>` → full transcript, interactive, can continue |
| verify result | `~/.local/state/cf-triage/<KEY>.verify.md` (+ screenshot paths), MR note |
| did the timer run / crash | `journalctl --user -u cf-triage --since today` |
| re-run one stage | `triage-worker DEV-X` (analyze / implement if `ai-fix`), `triage-worker DEV-X --verify` |
| interactive sub-agents (`/cf-ticket` verify, `/cf-epic` slices) | inside that session only (agent panel / transcript), not in the dashboard |

**Don't**: merge anything not `ai-verified` without reading why; edit holdout files after `ai-fix` (they're the contract); feed the factory tickets without a `repo-*` label.

## Flow

```
Sentry (fe-monitoring, can-frontend)
  │  sentry-triage          first run of each workday, 1 new ticket, ONLY if no pipeline ticket is open
  ▼
DEV Bug  [triage, sentry-<SHORTID>…, repo-<repo>]      assigned to me in Sentry; description ends with "Belongs in Sentry?" verdict
  │  triage-worker          every 5 min: cf-work --no-claim → claude -p "/cf-work KEY --headless"
  ▼
[triaged]  worktree + PLAN.md (first section: Belongs in Sentry? → filter fix if no) + holdout scenarios + Czech Jira comment (🤖 + "AI triage" marker)
  │  I reply in Jira  →  next run (≤5 min) answers, updates PLAN.md + scenarios   (To Do only, any age)
  │  I add label ai-fix  (= approves plan AND scenarios)
  ▼  triage-worker          max 1/run: cf-work (claims) → claude -p "/cf-ticket KEY --headless"
[ai-mr]  draft MR, no reviewers, assigned to me + Jira comment with MR link
  │  triage-worker          every repo-* MR's latest pipeline green (FE-only: + Amplify bot note for that sha)
  ▼                         → fresh claude -p "/cf-verify" runs holdout scenarios in Playwright (any BE repo / cross-repo: one throwaway dyn-env, all BE MR images + FE branch)
[ai-verified]  MR note with per-scenario verdicts
  │  fail → 1 retry: "/cf-ticket KEY --headless --verify-fix <report>" pushes fix [ai-verify-retry] → re-verify
  │  fail again / blocked / crash → [ai-verify-failed]
  │  I review, mark ready, add reviewers, merge; release moves ticket to Done
  ▼  triage-worker          ticket Done → resolves every sentry-* issue (haiku + Sentry MCP)
[ai-done]  → next morning sentry-triage picks the next issue
```

Any other ticket (feature, chore, manual bug): I add labels `triage` + `repo-<dir>` → same flow from `[triaged]` on; PLAN.md starts with Requirements instead of "Belongs in Sentry?". On Done it just gets `ai-done`.

One Sentry issue at a time: `sentry-triage` skips the day (Slack note) while any ticket with a `sentry-*` label and labelled `triage|triaged|triage-failed|ai-fix|ai-mr|ai-fix-failed` is not Done AND its MR is not merged/closed (`glab mr list --all --search KEY` in the `repo-*` repo; no MR at all = blocking, so failed/unfixed tickets block until closed). Jira status doesn't matter — `On nightly` with a merged MR is fine. Sentry resolve waits for status `Done` (prod release), not Release candidate. Stamp `~/.local/state/cf-triage/sentry-today` = already ran today (delete to re-run).

Failures: `triage-failed` / `ai-fix-failed` label. Every run with activity posts a summary to Slack `#zimmi-claude` via `~/.claude/scripts/slack-post.sh` (incoming webhook, no LLM). Quiet timer runs → no post, but if nothing posted for 3 h a "cf-triage alive — N quiet runs since X" heartbeat goes out (state `.posted` mtime + `.quiet` in `~/.local/state/cf-triage`). No heartbeat for >3 h during 07–19 weekdays = worker dead → `systemctl --user status cf-triage`.

## Labels

| label | meaning | set by |
|---|---|---|
| `triage` | new, waiting for analysis | sentry-triage, or me on any ticket |
| `sentry-<SHORTID>` | linked Sentry issue(s), full short id (dedup key) | sentry-triage |
| `repo-<dir>` | repo under `~/work/cleverfarm/repos` | sentry-triage |
| `triaged` | PLAN.md + comment done | triage-worker |
| `triage-failed` | analysis failed (no repo, worktree, crash) | triage-worker |
| `ai-fix` | **my approval** to implement PLAN.md | me |
| `ai-mr` | draft MR exists | triage-worker |
| `ai-fix-failed` | agent bailed out (see its Jira comment) or no MR | triage-worker |
| `ai-verify-retry` | holdout failed once, fix pushed, waiting for re-verify | triage-worker |
| `ai-verified` | holdout scenarios pass on the preview | triage-worker |
| `ai-verify-failed` | holdout failed after retry, blocked, or verifier crashed | triage-worker |
| `ai-done` | ticket Done, Sentry issues resolved | triage-worker |

## Files

| path | what |
|---|---|
| `~/dotfiles/bin/sentry-triage` | stage 1: open-ticket gate + daily stamp, then claude -p (sonnet) + Sentry MCP + `jira` → ticket. `--dry-run` (ignores gate + stamp), `MAX=` (default 1) |
| `~/dotfiles/bin/sentry-triage-now` | force a Sentry run today: drop `sentry-today` stamp + start `cf-triage.service` (no block) |
| `~/dotfiles/bin/triage-worker` | stages 2–7: triage, reply follow-up, implement, holdout verify (+1 retry), resolve Sentry on Done, rebase my conflicting MRs (timer runs only); Slack summary. flock'd (manual run while timer runs = exits). `triage-worker DEV-X` = one ticket (implements if it has `ai-fix`; `triage-worker DEV-X --verify` = verify its MR now). `MAX=`, `IMPL_MAX=`, `VERIFY_MAX=` (default 1; not-ready previews don't count). Holdout verify of a BE ticket blocks the run ~30 min (dyn-env create). Analyze + implement open a detached window per worktree in tmux session `cf-triage` (`tail -f <KEY>.log`) and run claude with that `TMUX_PANE`, so workmux status hooks show 🤖/✅ in the dashboard. Plain `tmux new-window -d`, not `workmux open` (that one `switch-client`s your attached client). No tmux server = skipped |
| `~/dotfiles/bin/factory-stats` | `factory-stats [days=7]`: yield per ticket touched in window — stages, bail rate, verify pass rate, reviewer notes per merged MR, days ticket→merge, $ total + per merged MR (from `cost=$` in `<KEY>.log`). Posted to Slack every Monday by `nightly-claude.sh` |
| `~/dotfiles/bin/cf-work` | worktree setup, one per repo (`--repos a,b` > `repo-*` labels > mapped labels `frontend`/`agroevidence`/… > cwd); same branch everywhere, claude in primary worktree + `--add-dir` siblings; `--no-launch`, `--no-claim` (skip assign + In Progress) |
| `~/dotfiles/claude/skills/cf-work` | `--headless`: root-cause analysis → PLAN.md + comment; follow-up mode on replies |
| `~/dotfiles/claude/skills/cf-ticket` | `--headless`: implement PLAN.md, verify, self-review, `/cf-mr --headless` |
| `~/dotfiles/claude/skills/cf-mr` | `--headless`: `glab mr create --draft`, no reviewers, stays In Progress, FE plays Nightly preview |
| `~/dotfiles/claude/cleverfarm/skills/cf-verify` | holdout verifier: Playwright login as `fe-automation@cleverfarm.cz` (password from cf-farm-frontend `.env`), runs scenarios, writes `<KEY>.verify.md` (`VERDICT: pass/fail/blocked`), MR note. Never edits code. BE: creates + destroys `<key>-verify` dyn-env |
| `~/dotfiles/claude/systemd/cf-triage.{service,timer}` | symlinked to `~/.config/systemd/user/`; Mon–Fri every 5 min 07–18 (idle run = 4 Jira queries, no LLM), `Persistent=true`, 110 min timeout |
| `~/.config/cf-triage.env` | `JIRA_EMAIL`, `JIRA_TOKEN` (0600, not in git) |
| `~/.local/state/cf-triage/` | `<KEY>.log`, `<KEY>.verify.md`, `holdout/<KEY>.md` (scenarios; analysis may write, implementer denied), `sentry-*.json`, `sentry-last.txt`, `sentry-today` (daily stamp), `notes.txt` (pending Slack lines), `conflicts/<project id>-<iid>` (last handled MR head) + `.md` (resolver report), `.lock` |
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
| `systemctl --user list-timers cf-triage.timer` | last/next run |
| `systemctl --user status cf-triage.service` | running? last exit |
| `journalctl --user -u cf-triage --since today` | run output |
| `systemctl --user start cf-triage.service` | run now |
| `systemctl --user disable --now cf-triage.timer` | turn off |
| `tail -f ~/.local/state/cf-triage/DEV-XXXX.log` | live per-ticket progress: 🔧 tool calls, 💬 agent text, ✅ exit + cost |

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

## Notifications

- Slack `#zimmi-claude` via incoming webhook (`slack-post.sh`, same as nightly digests), one message per run with activity; notes kept until curl succeeds.
- Webhook posts ping like any bot message. Jira comments come from my account → no Jira ping.

## Costs (DEV-2825 test)

create ticket ~$0.45 · analysis ~$0.65 · reply ~$0.4 · implement ~$1.6 (opus) · Sentry resolve (haiku) cents · Slack post free.

## Known gaps

- editing an old non-AI comment counts as a new reply if it's the last one
- MCP login expiry → runs fail. sentry-triage without `TRIAGE_DONE` → one Slack note per day (`sentry-failed-noted` stamp) with the `TRIAGE_FAILED` reason; fix Sentry auth via `/mcp` → claude.ai Sentry. Worker failures → `*-failed` labels / logs
- Persistent catch-up can fire once on Saturday if the laptop was off at Friday's last slot (daily stamp keeps it to one ticket)
- a `triage-failed` / `ai-fix-failed` ticket blocks new issues until I close or relabel it
