---
name: cf-verify
description: |
  Independent holdout verifier for the cf-triage factory. Runs the ticket's
  Given/When/Then scenarios from ~/.local/state/cf-triage/holdout/<KEY>.md in a
  real browser against the MR's preview (cf-farm-frontend: Amplify preview; BE
  repos: a throwaway cf-dyn-env env) and writes a pass/fail report. Never changes
  code. Invoked headless by triage-worker; use by hand with "cf-verify DEV-XXXX".
---

# Verify MR against holdout scenarios

Arguments: `<KEY> repo=<repo> mr=<iid> pipeline=<id> sha=<short sha> branch=<branch>` (latest green MR pipeline; FE preview note says `@ <sha>`)

You are the verifier, not the implementer. You did not write this code and must not fix it. Judge only what a user sees.

## Steps

1. Read `~/.local/state/cf-triage/holdout/<KEY>.md`. Line `NONE: <reason>` → write the report as `VERDICT: pass (no UI scenarios: <reason>)`, post the MR note, stop.
2. Target URL:
   - `repo=cf-farm-frontend` → `https://<branch lowercased>.dev.cleverfarm.cz`
   - any other repo → dyn-env (step 3)
3. BE only, with the `cf-dyn-env:dyn-env` skill. The `ai-fix` label is the human's approval for create + destroy, don't ask. Env name `<key lowercased>-verify`. `SERVICES=<repo's chart>:<MR image tag>,cf-gateway,farm-management` (+ `agroevidence` if a scenario touches farms/parcels), `DB_SEED=data`, `FRONTEND=true`, `GIT_BRANCH=<branch>`. MR image tag: the build job of `pipeline` (`glab api projects/<id>/pipelines/<pipeline>/jobs`, then its log). Poll `cf-env.sh status <pipeline-id>` until done (each call under 10 min). Target = `https://<env>.dev.cleverfarm.cz`. Destroy the env at the end, pass or fail.
4. Log in with Playwright MCP (`mcp__playwright__*`, fresh profile): `<target>/login`, user `fe-automation@cleverfarm.cz`, password = `FE_AUTOMATION_PSW` from `~/work/cleverfarm/repos/cf-farm-frontend/.env`, submit `data-testid=login-submit`. Login card re-renders on nightly; re-submit if still on `/login`. Never write the password anywhere.
5. Per scenario: set up Given, do When, check Then literally. `browser_take_screenshot` (`type: jpeg`) as evidence, `browser_console_messages` errors. Unclear step → pick the most literal reading, note it. Can't reach the Given state (data missing, env broken) → `blocked`, not `fail`.
6. Write `~/.local/state/cf-triage/<KEY>.verify.md`. First line EXACTLY one of:
   - `VERDICT: pass` — every scenario passed
   - `VERDICT: fail` — at least one scenario failed
   - `VERDICT: blocked` — none failed but at least one blocked

   Then per scenario: `S1 <name>: pass|fail|blocked — observed vs expected`, screenshot path, console errors.
7. Post ONE MR note: `glab mr note <iid> -m "<text>"`, English, first line = EN 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude", then the verdict + per-scenario lines (no screenshots, no password).

## Rules

- Never edit code, the scenarios file, or git state.
- Don't soften: Then not met = fail, even if it "almost" works.
- FE: MR's `amplify-preview-url` note not showing `@ <sha>` (stale build) → `VERDICT: blocked`, say so.
