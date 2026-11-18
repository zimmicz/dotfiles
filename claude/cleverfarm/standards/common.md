# Review standards — all CleverFarm repos

Cross-repo rules, read next to the repo's `CLAUDE.md` and `.claude/skills/review-checklist/SKILL.md` by `/cf-ticket` step 13 and the factory (`conv_collect`, `cf-review`). Format: `- <rule> — evidence — lint: <rule|no>`.

## misc

- Migration/refactor → delete leftovers in the SAME change: dead wrappers, unused components/HOCs/hooks/mutations/styles, empty dirs — evidence: !1617, !1618, !1721, !2080, !3258, "why do we need LegacyFieldErrors?", "delete useCustomStyles too" — lint: knip (installed in cf-farm-frontend, no script/CI job)
- No `@ts-ignore` / `@ts-expect-error` — solve the type — evidence: "drop ts-ignore and solve that" — lint: @typescript-eslint/ban-ts-comment
- Don't replace existing named constants with string literals — evidence: !1527 ("proč jsi prosím zrušil ty konstanty? V apce je to takhle všude.") — lint: no
- No speculative wrappers, props, API widening; justify each new one. Dedup worth it only above ~15 lines saved — evidence: "why do we need styled Box?", "how many rows saved by this?" — lint: no

## tests

- No new unit tests for thin env/config gates or fetch wrappers (Mixpanel init gating, `*.api.ts` calls); behaviour covered by E2E/holdout. Add unit tests only for real logic or when PLAN/user asks — evidence: session 2026-11-11, can-frontend !1415 + cf-farm-frontend !4477, !4472 ("Drop this file" ×3) — lint: no
