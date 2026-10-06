# cf-triage — headless Sentry → Jira → analysis → fix

Unattended pipeline on my laptop. Turns Sentry issues into DEV bugs, analyzes them with my cf-work flow, answers my Jira replies, and on my go (`ai-fix` label) implements the fix as a draft MR.

## Flow

```
Sentry (fe-monitoring, can-frontend)
  │  sentry-triage          every weekday hour, max 10 new tickets/run
  ▼
DEV Bug  [triage, sentry-<SHORTID>…, repo-<repo>]      Sentry issue assigned to me + Jira link note
  │  triage-worker          max 3/run: cf-work --no-claim → claude -p "/cf-work KEY --headless"
  ▼
[triaged]  worktree + PLAN.md + Czech Jira comment (🤖 + "AI triage" marker)
  │  I reply in Jira  →  next run answers, updates PLAN.md   (To Do only, reply ≤2h old)
  │  I add label ai-fix
  ▼  triage-worker          max 1/run: cf-work (claims) → claude -p "/cf-ticket KEY --headless"
[ai-mr]  draft MR, no reviewers, assigned to me + Jira comment with MR link
  │  I review, mark ready, add reviewers, merge, after release resolve all sentry-* issues
  ▼
done
```

Failures: `triage-failed` / `ai-fix-failed` label. Every run with activity posts a summary to Slack `#cf-triage` (private, C0C49QDLGAH).

## Labels

| label | meaning | set by |
|---|---|---|
| `triage` | new, waiting for analysis | sentry-triage |
| `sentry-<SHORTID>` | linked Sentry issue(s), full short id (dedup key) | sentry-triage |
| `repo-<dir>` | repo under `~/work/cleverfarm/repos` | sentry-triage |
| `triaged` | PLAN.md + comment done | triage-worker |
| `triage-failed` | analysis failed (no repo, worktree, crash) | triage-worker |
| `ai-fix` | **my approval** to implement PLAN.md | me |
| `ai-mr` | draft MR exists | triage-worker |
| `ai-fix-failed` | agent bailed out (see its Jira comment) or no MR | triage-worker |

## Files

| path | what |
|---|---|
| `~/dotfiles/bin/sentry-triage` | stage 1: claude -p (sonnet) + Sentry MCP + `jira` → tickets. `--dry-run`, `MAX=` |
| `~/dotfiles/bin/triage-worker` | stages 2–4: triage, reply follow-up, implement; Slack summary. `triage-worker DEV-X` = one ticket (implements if it has `ai-fix`). `MAX=`, `IMPL_MAX=` |
| `~/dotfiles/bin/cf-work` | worktree setup; `--no-launch`, `--no-claim` (skip assign + In Progress) |
| `~/dotfiles/claude/skills/cf-work` | `--headless`: root-cause analysis → PLAN.md + comment; follow-up mode on replies |
| `~/dotfiles/claude/skills/cf-ticket` | `--headless`: implement PLAN.md, verify, self-review, `/cf-mr --headless` |
| `~/dotfiles/claude/skills/cf-mr` | `--headless`: `glab mr create --draft`, no reviewers, stays In Progress, FE plays Nightly preview |
| `~/dotfiles/claude/systemd/cf-triage.{service,timer}` | symlinked to `~/.config/systemd/user/`; Mon–Fri hourly, `Persistent=true`, 110 min timeout |
| `~/.config/cf-triage.env` | `JIRA_EMAIL`, `JIRA_TOKEN` (0600, not in git) |
| `~/.local/state/cf-triage/` | `<KEY>.log`, `sentry-*.json`, `sentry-last.txt`, `notes.txt` (pending Slack lines) |
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
| `tail -f ~/.local/state/cf-triage/DEV-XXXX.log` | per-ticket detail + cost (json) |

## Test one ticket by hand

1. `MAX=1 sentry-triage --dry-run` → preview; `MAX=1 sentry-triage` → 1 real ticket
2. `triage-worker DEV-XXXX` → analysis (any DEV ticket with a `repo-*` label works)
3. comment in Jira, `triage-worker` (no args) → reply answered
4. add `ai-fix`, `triage-worker DEV-XXXX` → draft MR

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

- Slack `#cf-triage`, one message per run with activity; notes kept until a confirmed send (no permission denials).
- Posts come from my account → Slack/Jira won't ping me. Real pings = Slack app incoming webhook (not set up).

## Costs (DEV-2825 test)

create ticket ~$0.45 · analysis ~$0.65 · reply ~$0.4 · implement ~$1.6 (opus) · Slack post (haiku) cents.

## Known gaps

- reply after >2h offline is missed (window `updated >= -2h`)
- editing an old non-AI comment counts as a new reply if it's the last one
- MCP login expiry → runs fail, visible only as `*-failed` labels / Slack / logs
- no auto-resolve in Sentry after release
- Persistent catch-up can fire once on Saturday if the laptop was off at Friday's last slot
