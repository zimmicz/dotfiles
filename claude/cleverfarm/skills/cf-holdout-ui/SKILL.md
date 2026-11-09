---
name: cf-holdout-ui
description: |
  Independent holdout verifier for the cf-triage factory. Runs the ticket's
  Given/When/Then scenarios from ~/.local/state/cf-triage/holdout/<KEY>.md in a
  real browser against the MR's preview (FE-only: Amplify preview; any BE repo,
  also cross-repo: a throwaway cf-dyn-env env with every BE MR image + the FE
  branch) and writes a pass/fail report (+ optional narrated Czech video). Never changes
  code. Invoked headless by triage-worker; use by hand with "cf-holdout-ui DEV-XXXX".
---

# Verify MR against holdout scenarios

Arguments: `<KEY> repos=<repo>:<project id>:<mr iid>:<pipeline id>:<short sha>[,…] branch=<branch>` — one tuple per repo the change touches (cf-farm-frontend first if present), each the latest green MR pipeline; same branch name in every repo. Optional `env=<name>`: an existing dyn-env already running these images (epic flow) — use it, never create or destroy it. Optional `video=<abs dir>`: record the run as a narrated video in that dir (needs Playwright MCP with `--caps devtools`). Optional `failed=S3,S5`: scenarios that failed the previous round, a fix was pushed since (step 5a).

You are the verifier, not the implementer. You did not write this code and must not fix it. Judge only what a user sees.

## Steps

1. Read `~/.local/state/cf-triage/holdout/<KEY>.md`. Line `NONE: <reason>` → write the report as `VERDICT: pass (no UI scenarios: <reason>)`, post the MR note, stop.
2. Target URL:
   - only `cf-farm-frontend` → its Amplify preview `https://<branch lowercased>.dev.cleverfarm.cz`
   - any other repo in `repos` → dyn-env (step 3), which also builds the FE
   That is the host only. Each scenario's Given carries a fixture id from `cf-farm-frontend/tests/FIXTURES.md` and a path with real ids: access `direct` → open `<host><path>`; `impersonate` → farm selector per that file, cancel the impersonation after the scenario. `DATA: missing …` in a Given → that scenario is `blocked` with that reason, no hunting for data.
3. With the `cf-dyn-env:dyn-env` skill. The `ai-fix` label (or the user's plan approval) is the approval for create + destroy, don't ask. Env name `<key lowercased>-verify`. `SERVICES` = every BE repo's chart at its MR image tag (chart = repo dir, or without the `cf-` prefix when only that form is in the skill's chart list) + `cf-gateway,farm-management` (+ `agroevidence` if a scenario touches farms/parcels), `DB_SEED=data`, `FRONTEND=true`, `FE_BRANCH=<branch>` if cf-farm-frontend is in `repos` (else omit → develop), `GIT_BRANCH=<branch>`. MR image tag per BE repo: the build job of its pipeline (`glab api projects/<project id>/pipelines/<pipeline id>/jobs`, then the job log). Poll `cf-env.sh status <pipeline-id>` until done (each call under 10 min). Target = `https://<env>.dev.cleverfarm.cz`. Destroy the env at the end, pass or fail.
4. Log in with Playwright MCP (`mcp__playwright__*`, fresh profile): `<target>/login`, user `fe-automation@cleverfarm.cz`, password = `FE_AUTOMATION_PSW` from `~/work/cleverfarm/repos/cf-farm-frontend/.env`, read with the native `Read` tool only (it overrides the lean-ctx "Read only for edits" policy; shell grep/sed/source and Playwright `fs` are denied headless), submit `data-testid=login-submit`. Login card re-renders on nightly; re-submit if still on `/login`. Never write the password anywhere.
4a. `video` set → one recording per scenario: right before it `browser_start_video` (`filename: <video>/S1.webm` with that scenario's id, `size: {width: 1280, height: 720}`), right after its last screenshot `browser_stop_video`, also on fail/blocked. Scenarios not run get no recording.
5. Per scenario: open `<host>` + the Given path (impersonate first if the fixture says so), set up the rest of Given, do When, check Then literally. `browser_take_screenshot` (`type: jpeg`) as evidence, `browser_console_messages` errors. Unclear step → pick the most literal reading, note it. Can't reach the Given state (data missing, env broken) → `blocked`, not `fail`.
5a. Order: `failed=` set → those scenarios first, in that order. One of them still fails → the remaining scenarios are `not run — <id> still fails` (the fix is wrong, a regression pass proves nothing), write the report, post the note, stop. All of them pass → the rest in file order. No `failed=` → file order.
6. Write `~/.local/state/cf-triage/<KEY>.verify.md` after EVERY scenario, the whole file each time (Write, not Edit): first line `VERDICT: running`, then one line per scenario done so far. The file is the record: never grade a scenario from memory, re-read the file before the final write. After the last scenario rewrite it with the final first line, EXACTLY one of:
   - `VERDICT: pass` — every scenario passed
   - `VERDICT: fail` — at least one scenario failed
   - `VERDICT: blocked` — none failed but at least one blocked

   Then per scenario: `S1 <name>: pass|fail|blocked — observed vs expected`, screenshot path, console errors; skipped ones `S4 <name>: not run — S3 still fails`. Last line `Video: <video>/<KEY>.short.webm` if recorded.
6a. `video` set → Write `<video>/narration.tsv`, one line per part, tab-separated `<id>	<card title>	<sentence>|<sentence>|…`, spoken Czech read aloud by TTS: short sentences, no markdown, URLs, `|` or tabs inside a sentence, numbers and dates as words the way a Czech speaker says them. First line id `S0`, title `<KEY> · <what the change does, short>`, 2 sentences: what the change lets the user do (from the scenarios, never PLAN.md) + how many scenarios ran and the verdict. Then one line per scenario in run order, id = its recording's id, title `S1 · <name>`: sentence 1 = what the scenario checks (plays over the title card), then 1–3 sentences on what the recording shows and the result (`Prošlo.` / `Selhalo: <why>.` / `Zablokováno: <why>.`). Not run → one sentence why. Then `bash ~/.claude/skills/cf-holdout-ui/narrate.sh <video> <KEY>` — exactly this: it builds `<video>/<KEY>.short.webm` (title card + voice per part, waits cut, chapters). It fails → note without video, say so.
7. Post ONE MR note on the first repo's MR (cwd = its worktree): `glab mr note <iid> -m "<text>"`, listing the other repos' MRs it covered, English, first line = EN 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude", then the body, built to be skimmed (no walls of text, no screenshots, no password): `**Holdout verification: <pass|fail|blocked>**` + where it ran (FE-only: Amplify preview @ `<sha8>`, else the repos); one count line (`6 pass, 1 blocked`, `1 fail, 5 not run`); if any scenario is not pass, `**Needs attention**` with one bullet each, `- 🚫 **S5 <name>**: <what happened, one sentence>. <what is needed>` (❌ for fail, 🚫 for blocked); passes below as `- ✅ **S1 <name>**: <evidence, one sentence>`, wrapped in `<details><summary>Passed (N)</summary>` + blank line + bullets + blank line + `</details>` when something needs attention, plain bullets when everything passed. Caveat on a scenario (e.g. weak coverage) = `_Note: …_` at the end of its bullet. Each bullet max 2 lines. Video built → upload it first (`curl -s --fail-with-body -H "PRIVATE-TOKEN: $(glab config get token --host gitlab.com)" -F file=@<video>/<KEY>.short.webm https://gitlab.com/api/v4/projects/<project id>/uploads | jq -r .markdown`; not `glab api -F file=@`: snap glab can't read `~/.local` and sends the file as a JSON field, not multipart → 400) and end the note with `Video: <that markdown>`. Upload fails → note without video, say so.

## Rules

- Never edit code, the scenarios file, or git state.
- A `Conversation compacted` notice mid-run → re-read this skill (`~/.claude/skills/cf-holdout-ui/SKILL.md`), the scenarios file and your `<KEY>.verify.md` so far before the next scenario.
- Don't soften: Then not met = fail, even if it "almost" works.
- FE-only: MR's `amplify-preview-url` note not showing `@ <sha>` (stale build) → `VERDICT: blocked`, say so.
- dyn-env create/redeploy fails → `VERDICT: blocked` with the failing job, destroy what was created.
