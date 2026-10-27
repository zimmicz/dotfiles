# Review standards — all CleverFarm repos

Read by the independent reviewer (`/mattpocock-skills:code-review` in `/cf-ticket` phase 5) next to the repo's own `CLAUDE.md` and `<repo>.md`. Format: `- <rule> — evidence — lint: <rule|no>`.

## misc

- Migration/refactor → delete leftovers in the SAME change: dead wrappers, unused components/HOCs/hooks/mutations/styles, empty dirs — evidence: !1617, !1618, !1721, !2080, !3258, "why do we need LegacyFieldErrors?", "delete useCustomStyles too" — lint: knip (installed in cf-farm-frontend, no script/CI job)
- No `@ts-ignore` / `@ts-expect-error` — solve the type — evidence: "drop ts-ignore and solve that" — lint: @typescript-eslint/ban-ts-comment
- Don't replace existing named constants with string literals — evidence: !1527 ("proč jsi prosím zrušil ty konstanty? V apce je to takhle všude.") — lint: no
- No speculative wrappers, props, API widening; justify each new one. Dedup worth it only above ~15 lines saved — evidence: "why do we need styled Box?", "how many rows saved by this?" — lint: no
