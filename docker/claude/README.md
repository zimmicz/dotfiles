# dclaude

Claude Code with `--dangerously-skip-permissions` inside a Docker sandbox. Same skills, hooks, CLAUDE.md and plugins as on the host, mounted read-only.

## Setup

1. GitLab → User settings → Access tokens: scopes `api` + `write_repository`, expiry 90 days.
2. Create `~/.config/dclaude.env` (mode 600):

   ```
   GITLAB_TOKEN=glpat-...
   DCLAUDE_VPN_CIDRS=10.x.0.0/16
   DCLAUDE_EXTRA_DOMAINS=
   ```

   `DCLAUDE_VPN_CIDRS`: space-separated networks behind the VPN (dev DB, anything not resolvable by name). `DCLAUDE_EXTRA_DOMAINS`: extra hosts for the firewall allowlist.
3. `./install.sh` (symlinks `bin/dclaude` to `~/bin`).

## Use

```
cd ~/work/cleverfarm/repos/<repo>/.worktrees/<KEY>
dclaude                 # interactive
dclaude -p "run lint"   # headless
DCLAUDE_CMD=zsh dclaude # shell inside the sandbox, for debugging
```

First run builds the image. Rebuild: `docker rmi dclaude`.

Refuses to start outside a repo/worktree root and anywhere under `~/dotfiles`.
Do not start it in a worktree where an sw-factory stage is running; it does not see factory locks.

## What the sandbox sees

| Path | Mode |
|---|---|
| current worktree, `.git` of its main repo | rw |
| `.git/hooks`, `.git/config` | ro |
| `~/.claude/projects` | rw (resume, nightly reflection) |
| `~/.claude` config: settings, CLAUDE.md, LEARNED.md, hooks, plugins, skills, `.credentials.json` | ro |
| everything else Claude writes to `~/.claude` | volume `dclaude-state` |
| `~/dotfiles`, `~/bin`, mise node (`/opt/node`), `~/.local/bin/lean-ctx`, claude binary | ro |
| pnpm store | volume `dclaude-pnpm` |
| `~/.claude.json` | throwaway copy |

Network: allowlist in `entrypoint.sh` (Anthropic, npm, GitLab, Atlassian, Sentry, dev/staging, VPN hosts, CloudFront). Everything else is rejected. VPN must be up before start for `.internal` hosts to resolve.

## Known limits

- Reads every token in `.credentials.json`; connectors (Jira, Sentry, Drive) act as you. Slack plugin is disabled.
- All of CloudFront is reachable (Amplify previews share its IPs).
- Allowlisted IPs are resolved at start; restart after a long session if a host stops responding.
- Files in the worktree that the host later runs (`package.json` scripts, husky) can be changed by the sandbox. Read the diff first.
- Docker daemon runs as root.
- No X11 in the sandbox. Copy goes: Claude -> OSC 52 -> host tmux (`set-clipboard on` + `pane-set-clipboard` hook in `tmux.conf`) -> `xclip`. Needs `TERM=xterm-256color` inside, set by the wrapper. Outside tmux it depends on the terminal supporting OSC 52.

## Rollback

Delete `bin/dclaude` and `docker/claude/`, then `docker rmi dclaude; docker volume rm dclaude-state dclaude-pnpm`.
