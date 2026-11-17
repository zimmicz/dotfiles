# sw-factory TODO

Follow-ups of the review-conventions change (live since 2026-11-17 20:57, master `c44ca42`, tag `sw-factory-review-conventions-2026-11-17`, rollback in README `## Rollback`). Calendar reminder 2026-11-24 covers item 1.

1. 2026-11-24: `factory-compare 2026-11-17T20:57:00+01:00`, judge the effect (conventions %, review runs per ticket, findings must / should / convention, MR follow-ups per ticket vs transcript baseline 3 / p90 14, review cost vs baseline $1.17 / p90 $2.54). Needs about a week of real runs.
2. After the first real rfix cycle: confirm `<KEY>.rfix-count`, the `RFIX_MAX` cap path, and a convention finding (`- Rule:` line, `findings: … convention=<n>`) in `<KEY>.log`. Not covered by the DEV-3799 live test (real verify stage, rfix cap, non-zero findings).
3. cf-review: the test run skipped `/mattpocock-skills:code-review` (skill step 3 says run it). `glab mr note -m` is deprecated, use `glab mr note create`.
4. README and map name `factory-worker KEY --review`, a flag that does not exist (only `--gate`, `--conventions`, `--stage <stage>`). Fix the text or add the flag.
5. Move the team rules to the repo, git tracked, available to everyone:
   - dotfiles `standards/cf-farm-frontend.md` (team rules mined from MR reviews) → cf-farm-frontend `.claude/skills/review-checklist/SKILL.md`, deduplicated against root `CLAUDE.md` Coding Rules. Via Jira ticket (Czech) → branch → MR, not straight on develop.
   - `/cf-feedback` writes new lessons into the repo checklist (not dotfiles), and also reads Michal's own MR notes (today filtered out as "not yours").
   - dotfiles `standards/` keeps only personal gotchas (`common.md`); `conventions()` source 4 shrinks accordingly.
   - Resolve the doc mismatches in the same MR: import rule (`../` inside a module) in root `CLAUDE.md` vs standards, checklist §12 `npm run` vs `pnpm run`, standards header "`/cf-ticket` phase 5", README grill text vs `factory-worker` grill ASK.
6. Backlog from the audit (numbers in `PLAN-review-conventions.md`): verify 33 % of spend / 45 % pass, `blocked` treated as failed, review never starts after verify fail/blocked, fix variants skip skill steps 5–6, no retry cap after a crash, grill ASK on `triaged` ticket does not set `needs-answer`, label triggers not owner-filtered, deny-list prefix patterns, `factory-stats` bugs (`sw-factory-bb`).
