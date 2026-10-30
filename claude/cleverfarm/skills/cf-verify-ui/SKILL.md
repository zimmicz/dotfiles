---
name: cf-verify-ui
description: |
  Independent holdout verifier for the cf-triage factory. Runs the ticket's
  Given/When/Then scenarios from ~/.local/state/cf-triage/holdout/<KEY>.md in a
  real browser against the MR's preview (FE-only: Amplify preview; any BE repo,
  also cross-repo: a throwaway cf-dyn-env env with every BE MR image + the FE
  branch) and writes a pass/fail report (+ optional video). Never changes
  code. Invoked headless by triage-worker; use by hand with "cf-verify-ui DEV-XXXX".
---

# Verify MR against holdout scenarios

Arguments: `<KEY> repos=<repo>:<project id>:<mr iid>:<pipeline id>:<short sha>[,…] branch=<branch>` — one tuple per repo the change touches (cf-farm-frontend first if present), each the latest green MR pipeline; same branch name in every repo. Optional `env=<name>`: an existing dyn-env already running these images (epic flow) — use it, never create or destroy it. Optional `video=<abs path>.webm`: record the run (needs Playwright MCP with `--caps devtools`).

You are the verifier, not the implementer. You did not write this code and must not fix it. Judge only what a user sees.

## Steps

1. Read `~/.local/state/cf-triage/holdout/<KEY>.md`. Line `NONE: <reason>` → write the report as `VERDICT: pass (no UI scenarios: <reason>)`, post the MR note, stop.
2. Target URL:
   - only `cf-farm-frontend` → its Amplify preview `https://<branch lowercased>.dev.cleverfarm.cz`
   - any other repo in `repos` → dyn-env (step 3), which also builds the FE
3. With the `cf-dyn-env:dyn-env` skill. The `ai-fix` label (or the user's plan approval) is the approval for create + destroy, don't ask. Env name `<key lowercased>-verify`. `SERVICES` = every BE repo's chart at its MR image tag (chart = repo dir, or without the `cf-` prefix when only that form is in the skill's chart list) + `cf-gateway,farm-management` (+ `agroevidence` if a scenario touches farms/parcels), `DB_SEED=data`, `FRONTEND=true`, `FE_BRANCH=<branch>` if cf-farm-frontend is in `repos` (else omit → develop), `GIT_BRANCH=<branch>`. MR image tag per BE repo: the build job of its pipeline (`glab api projects/<project id>/pipelines/<pipeline id>/jobs`, then the job log). Poll `cf-env.sh status <pipeline-id>` until done (each call under 10 min). Target = `https://<env>.dev.cleverfarm.cz`. Destroy the env at the end, pass or fail.
4. Log in with Playwright MCP (`mcp__playwright__*`, fresh profile): `<target>/login`, user `fe-automation@cleverfarm.cz`, password = `FE_AUTOMATION_PSW` from `~/work/cleverfarm/repos/cf-farm-frontend/.env`, submit `data-testid=login-submit`. Login card re-renders on nightly; re-submit if still on `/login`. Never write the password anywhere.
4a. `video` set → after login `browser_start_video` (`filename: <video>`, `size: {width: 1280, height: 720}`). Stop it with `browser_stop_video` after the last scenario, also on fail/blocked.
5. Per scenario: `video` set → `browser_video_chapter` (`title: "S1 <name>"`) first. Set up Given, do When, check Then literally. `browser_take_screenshot` (`type: jpeg`) as evidence, `browser_console_messages` errors. Unclear step → pick the most literal reading, note it. Can't reach the Given state (data missing, env broken) → `blocked`, not `fail`.
6. Write `~/.local/state/cf-triage/<KEY>.verify.md`. First line EXACTLY one of:
   - `VERDICT: pass` — every scenario passed
   - `VERDICT: fail` — at least one scenario failed
   - `VERDICT: blocked` — none failed but at least one blocked

   Then per scenario: `S1 <name>: pass|fail|blocked — observed vs expected`, screenshot path, console errors. Last line `Video: <video>` if recorded.
7. Post ONE MR note on the first repo's MR (cwd = its worktree): `glab mr note <iid> -m "<text>"`, listing the other repos' MRs it covered, English, first line = EN 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude", then the body, built to be skimmed (no walls of text, no screenshots, no password): `**Holdout verification: <pass|fail|blocked>**` + where it ran (FE-only: Amplify preview @ `<sha8>`, else the repos); one count line (`6 pass, 1 blocked`); if any scenario is not pass, `**Needs attention**` with one bullet each, `- 🚫 **S5 <name>**: <what happened, one sentence>. <what is needed>` (❌ for fail, 🚫 for blocked); passes below as `- ✅ **S1 <name>**: <evidence, one sentence>`, wrapped in `<details><summary>Passed (N)</summary>` + blank line + bullets + blank line + `</details>` when something needs attention, plain bullets when everything passed. Caveat on a scenario (e.g. weak coverage) = `_Note: …_` at the end of its bullet. Each bullet max 2 lines. Video recorded → upload it first (`curl -s --fail-with-body -H "PRIVATE-TOKEN: $(glab config get token --host gitlab.com)" -F file=@<video> https://gitlab.com/api/v4/projects/<project id>/uploads | jq -r .markdown`; not `glab api -F file=@`: snap glab can't read `~/.local` and sends the file as a JSON field, not multipart → 400) and end the note with `Video: <that markdown>`. Upload fails → note without video, say so.

## Rules

- Never edit code, the scenarios file, or git state.
- Don't soften: Then not met = fail, even if it "almost" works.
- FE-only: MR's `amplify-preview-url` note not showing `@ <sha>` (stale build) → `VERDICT: blocked`, say so.
- dyn-env create/redeploy fails → `VERDICT: blocked` with the failing job, destroy what was created.
