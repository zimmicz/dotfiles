---
name: cf-verify-api
description: |
  Independent differential API verifier for the cf-triage factory, for BE refactors
  (no UI effect). Runs the ticket's request sequences from
  ~/.local/state/cf-triage/holdout/<KEY>.api.md against two throwaway cf-dyn-env envs
  with the same seeded data, one on the baseline (nightly) images and one on the MR
  images, and diffs status + body. Any difference = fail. Never changes code.
  Invoked headless by triage-worker; use by hand with "cf-verify-api DEV-XXXX".
---

# Verify MR against baseline, API level

Arguments: `<KEY> repos=<repo>:<project id>:<mr iid>:<pipeline id>:<short sha>[,…] branch=<branch>` — one tuple per BE repo the change touches, each the latest green MR pipeline.

You are the verifier, not the implementer. A refactor must not change behaviour: the baseline env is the oracle, you never judge what the "right" answer is, only whether the two envs answer the same.

## Scenario file

`~/.local/state/cf-triage/holdout/<KEY>.api.md`, per scenario:

```
## A1: <name>
Service: <chart, e.g. agroevidence>
Given: <farm / entity, resolved by name via GET, never by hardcoded id>
1. GET /farms/{farmId}/parcels
2. POST /farms/{farmId}/parcels  {"name": "x", ...}   -> $p = .id
3. GET /farms/{farmId}/parcels/$p
4. DELETE /farms/{farmId}/parcels/$p
Headers: X-Total-Count
Expect: 200
Expect-nonempty: true
Ignore: id, createdAt, updatedAt
```

`Expect` and `Expect-nonempty` make a scenario meaningful: a diff of two empty or two unexpected answers proves nothing.

Steps run in order, the same on both envs. `-> $v = <jq path>` captures a value from that step's response per env, so later steps use each env's own id. Compared per step: HTTP status, the response headers named in `Headers` (optional), and JSON body minus `Ignore` keys (anywhere in the body), arrays compared in order. A body that isn't JSON is compared as text. A step may carry a raw body (`{not json`) to exercise error paths.

## Steps

1. Read the scenario file. Missing → write the report as `VERDICT: blocked`, stop.
2. Two envs via the `cf-dyn-env:dyn-env` skill. The `ai-fix` label (or the user's plan approval) is the approval for create + destroy, don't ask. Names `<key lowercased>-base` and `<key lowercased>-mr`. Both: `DB_SEED=data`, `GIT_BRANCH=<branch>`, `JWT_SOURCE=nightly`, no frontend, `SERVICES`:
   - `-mr`: every BE repo's chart at its MR image tag (build job of its pipeline: `glab api projects/<project id>/pipelines/<pipeline id>/jobs`, then the `publish-manifest` job log; not an `-amd64` tag) + `cf-gateway,farm-management,agroevidence` (+ `earth-data` when the chart is `cf-satellite`, nightly tag on both envs).
   - `-base`: the same charts without tags (= nightly).
   `cf-env.sh` needs `export TMPDIR=$HOME/cf-env-tmp` (mkdir it; a non-hidden dir under `$HOME`, glab is a snap and can't see `/tmp`), and runs as `bash "$CF_ENV" …` (a bare `"$CF_ENV"` is refused by the lean-ctx allowlist). Start both creates; the pipelines serialize on a resource group, so expect ~2x one create (~15 min each, destroys the same) and poll both `status <pipeline-id>` until done (each call under 10 min). Either fails → `VERDICT: blocked` with the failing job, destroy what was created.
3. Service URL per env: `https://<chart>.<env>.dev.cleverfarm.cz` for service charts (cf-satellite, agroevidence, farm-management); the gateway is `https://api.<env>.dev.cleverfarm.cz`. Poll `curl -s -o /dev/null -w '%{http_code}' <url>/api-docs` on both envs' service under test until 200 (a pod can answer 502/503 for a few minutes after the create reports success), up to 10 min, else `blocked`. Quote URLs with `?` (zsh).
4. Token: `TOKEN=$(node ~/.claude/skills/cf-verify-api/token.mjs)` (Playwright login on `https://nightly.dev.cleverfarm.cz/login` as `fe-automation@cleverfarm.cz`, prints the Bearer token). Run it inside the same command that uses the token, never write the token or the password to disk or the report. `JWT_SOURCE=nightly` makes it valid in both envs.
5. Given farm by name: `GET https://agroevidence.<env>.dev.cleverfarm.cz/api/farms` (farm-management `/api/farms` answers 401, the gateway 404), match by name, per env. Per scenario, run the steps against `-base` then `-mr` with `curl -s -D - -w '\n%{http_code}'` (headers needed for `Headers`), resolving `$v` per env. Normalise with `jq` (drop `Ignore` keys, `-S` sort keys) and compare step by step. Any step differs (status or body) → scenario `fail` with the first differing step, both bodies trimmed to the differing part. Can't reach Given (data missing, 401, env broken) on either env → `blocked`, not `fail`. Run each scenario's DELETE/cleanup steps even after a diff.
   - Weak-pass guard: scenario line `Expect: <status>` (and `Expect-nonempty: true` for list bodies) is checked on BOTH envs besides the diff. An env answering something else, or an empty list where non-empty was expected, makes the scenario `blocked` (the scenario can't tell anything), not `pass`. Report the observed status.
6. Write `~/.local/state/cf-triage/<KEY>.verify-api.md` (absolute path; confirm with `head -1` that it exists before posting the note). First line EXACTLY one of:
   - `VERDICT: pass`
   - `VERDICT: fail`
   - `VERDICT: blocked` — none failed but at least one blocked

   Then per scenario `A1 <name>: pass|fail|blocked — first differing step, base vs mr`.
7. Destroy both envs, pass or fail (`cf-env.sh run destroy`, then `wait`).
8. Post ONE MR note on the first repo's MR (cwd = its worktree): `glab mr note <iid> -m "<text>"`, English, first line = EN 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude", then the body, built to be skimmed (no walls of text, no tokens, no passwords): `**API verification: <pass|fail|blocked>**` + the two envs compared; one count line; if anything differs or is blocked, `**Needs attention**` with one bullet per sequence, `- ❌ **<sequence>**: <request, nightly vs MR status/body difference, one sentence>` (🚫 for blocked); matching sequences as `- ✅ **<sequence>**: <what matched>`, wrapped in `<details><summary>Matching (N)</summary>` + blank line + bullets + blank line + `</details>` when something needs attention. Each bullet max 2 lines.

## Rules

- Never edit code, the scenario file, or git state.
- Don't soften: any status or body difference is a fail, even if it looks harmless.
- Don't "fix" a flaky diff by editing `Ignore`; report it, the human decides.
- Read-only on envs beyond the scenario's own requests; never touch nightly, staging or someone else's env.
