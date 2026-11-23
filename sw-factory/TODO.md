# sw-factory TODO

Open follow-ups. Items 1–3 come from the review-conventions change (live since 2026-11-17 20:57, master `c44ca42`, tag `sw-factory-review-conventions-2026-11-17`, rollback in README `## Rollback`); calendar reminder 2026-11-24 covers item 1.

1. 2026-11-24: `factory-compare 2026-11-17T20:57:00+01:00`, judge the effect (conventions %, review runs per ticket, findings must / should / convention, MR follow-ups per ticket vs transcript baseline 3 / p90 14, review cost vs baseline $1.17 / p90 $2.54). Needs about a week of real runs.
2. After the first real rfix cycle: confirm `<KEY>.rfix-count`, the `RFIX_MAX` cap path, and a convention finding (`- Rule:` line, `findings: … convention=<n>`) in `<KEY>.log`. Not covered by the DEV-3799 live test (real verify stage, rfix cap, non-zero findings).
3. cf-review: the test run skipped `/mattpocock-skills:code-review` (skill step 3 says run it). `glab mr note -m` is deprecated, use `glab mr note create`.
4. Holdout UI verifier leaves written data on shared nightly. Port the cleanup from the dropped branch DEV-3489 (commit `2972dd2`, 2026-10-30, written for the old `cf-verify-ui` / `cf-work`):
   - `cf-work-headless` scenario format: optional `Creates: <record the When creates and where, only if it writes data; the verifier deletes it afterwards>`.
   - `cf-holdout-ui`, after the per-scenario step: "Cleanup, shared nightly data must end as it started. Scenario with a `Creates:` line, or any When that writes (also a write you only meant to mock and that went through): before the When count the matching records (list in the UI, or the API the page calls); after the Then delete what the run created (UI first, else API) and count again. Count after ≠ before → report `Cleanup: left <record> on <farm id>`, never silently."
   - Report: per scenario `Cleanup: ok` or `Cleanup: left <record> on <farm id>`.
5. Backlog from the 2026-11-17 audit (2026-11-01 → 11-17, 57 tickets, from `<KEY>.log` + transcripts):
   - Verify = 33 % of spend, 45 % pass. Stale holdout: 19 of 32 verify-fix rounds changed no code ("scenario outdated"); pass right after a fix 4 of 26. `blocked` is treated as failed: 36 of 134 runs, about half are single scenarios with missing fixture data, ~6 Playwright MCP `CONNECT_TIMEOUT`. 68 of 134 verify runs are full re-verifies after a follow-up push.
   - Review never starts when verify ends fail/blocked (18 of 40 tickets); 21 merged MRs carry `ai-gate-skip`, 3 quiz records exist.
   - Fix variants skip skill steps 5–6 (code-review skill in 0 of 34 verify-fix, 4 of 173 follow-ups). The delta re-review after pushes covers the review part.
   - Implement: outcome inferred from "MR exists"; 2 runs ended waiting for a background eslint/test job. No retry cap after a crash or timeout (DEV-3518: 3 runs in 18 min).
   - Labels (`ai-fix`, `triage`, `ai-gate-skip`, `ai-rebase`) are acted on whoever set them; comments are owner-filtered.
   - Deny-lists are prefix patterns: `glab mr merge|approve` denied, `glab api:*` allowed; FE project has `merge_requests_author_approval: true`. Direct push to develop/staging/production is blocked server-side.
   - `factory-stats`: counts review verdicts as verify runs, `implementing` never matches.
6. DEV-3873 retro changes (2026-11-23, rollback tag `sw-factory-pre-dev3873-retro-2026-11-23`, README "Requirements don't get lost between rounds"). Not exercised on a real ticket yet, only `zsh -n` + a function test of `unrunnable` / `plan_ok`:
   - 2026-11-30: on the next 3 feature tickets check `notes/<KEY>.md` has `## Sources` + `## Acceptance criteria` with verbatim rows, `PLAN.md` carries the table with `Where`, and count MR follow-ups vs DEV-3873 v1 (2 rework rounds, 3 blocked verifies, reset).
   - In-flight tickets: a follow-up on a `PLAN.md` written before the change must add `### Acceptance criteria` or it lands in `triage-failed`.
   - Open decision: FE change whose BE lives only on a feature / dyn env (DEV-3873: `dev-3657-pasture`). The preview runs against nightly, so every scenario is `DATA: missing` and the gate refuses `ai-fix`. Option: verifier redeploys that env with `FE_BRANCH=<branch>` (touches an env the BE dev uses).
   - Unverified: `--model opusplan` in `claude -p` for `impl` (no plan mode headless, so probably sonnet throughout). The sizing change only moves the analysis to opus.
