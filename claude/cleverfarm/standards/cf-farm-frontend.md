# Review standards — cf-farm-frontend

Read by the independent reviewer (`/mattpocock-skills:code-review` in `/cf-ticket` phase 5) next to `CLAUDE.md` "Coding Rules" and `common.md`. Not repeated here: what `CLAUDE.md` or ESLint already enforces. Source: reviewer comments on 729 merged MRs (2026-10 pass), LEARNED.md, CLAUDE.local.md. Format: `- <rule> — evidence — lint: <rule|no>`.

## components/composition

- Composition (compound components, children) over config props/flags (`isHarvest`, `showX`) and callback/count props — app components too (`Nav({ unresolved, onNavigate })`) — evidence: "prefer component composition where possible", "why don't we use composition here?" — lint: no
- DS glitch → fix in DS + Storybook story covering the states, not an app-level override. Restyled DS components (`styled()`/`sx` overrides) where a DS default/variant would do → flag — evidence: "need to solve this in DS itself", "focus on places where we restyle some components" — lint: no
- Reuse existing DS/shared components and page building blocks instead of rebuilding (`SortableHeader`, DS Autocomplete over MUI field) — evidence: !3503, !3654 — lint: no
- Self-contained JSX chunk with its styling → own component/file, not a growing parent. `renderXxx()` helpers → real JSX components — evidence: !3503, !1476 ("Nestálo by za to udělat tohle jako samostatnou komponentu?") — lint: no
- Don't wrap trivial JSX in a local variable/helper; inline it — evidence: !3098 ("Proč to neinlinovat do toho jsx?") — lint: no
- Key mapped elements by stable id, not array index — evidence: !1463 — lint: react/no-array-index-key (off)

## state & data fetching

- Moving mutations to global invalidation → keep every `await` that was there, via `meta.awaits` (app/shared/api/MUTATION_PATTERNS.md) — evidence: !3195 (7 comments) — lint: no
- `isError` / `isPending` / `isSuccess` flags, not `status === "error"` — evidence: !2297 — lint: no-restricted-syntax (not configured)
- Query keys on the factory root (`xKeys.all()`), not separate global params — evidence: !3142 — lint: cleverfarm/no-raw-query-keys (partial)
- Async race / stale state → simplest first: block UI while in flight (`inert`/disabled), reuse loaded data. No extra fetches (`ensureQueryData`), token refs, latest-request guards — evidence: "wouldnt it be easier to disable the form fields while some async call is in flight?" — lint: no

## types

- No `unknown` + `as X` cast at use site; give the real type (`SelectChangeEvent<string>`, `ParcelTo`) — evidence: !3346 (21 comments, "Zase unknown a následný cast") — lint: partial (@typescript-eslint/consistent-type-assertions)
- Narrow prop/param types to fields actually used; no loose/wide types — evidence: !2965, !3332, !3782 ("ta parcel používá jen blockNumber, localName") — lint: no
- Fixed option sets → string-literal union, not `string` — evidence: !1723 — lint: no
- No unused generic type params — evidence: !1527 — lint: @typescript-eslint/no-unnecessary-type-parameters (needs ts-eslint v8)
- No pointless `String()` conversions — evidence: !3324 — lint: @typescript-eslint/no-unnecessary-type-conversion (needs ts-eslint v8.30+)
- Type-only imports → `import type { … }`; plain import only for real value use — evidence: "make type only imports say type" — lint: @typescript-eslint/consistent-type-imports (not configured)
- `sx` props typed `SxProps<Theme>`, not `BoxProps["sx"]` — evidence: !2172 — lint: no

## styling/DS

- Translucent colors via `alpha(theme.palette.…, n)`, `theme.palette.common.white/black`; no `rgba()` / `"white"` literals — evidence: !1459, !1720, !1721, !1723 — lint: no-restricted-syntax (not configured)
- No inline `style={{}}` / large `sx` objects; use `styled()` — evidence: !2297, !2958 ("Chceme stylovat přímo inline?") — lint: react/forbid-component-props (not configured)
- Fixed bars/footers matching page bg → `theme.palette.grey[50]` (page bg `#fafafa`), not `background.default` (white) — evidence: "white band" — lint: no
- Near-duplicate style blocks → flag with lines saved (skip < ~15 lines) — evidence: "check if we dont duplicate too many styles" — lint: no
- No `float` for layout; flexbox — evidence: !1720 — lint: no
- Zero without unit (`0`, not `0px`) — evidence: !1459 — lint: no (no stylelint)

## tests/e2e

- Page objects use shared `tests/pages/components/core/*` + `table/*` (Button, Checkbox, Select, Table, …) and extend `Sidebar.ts`; no hand-rolled locators for them — evidence: !3324, !3503, !3714, !3782 (6 comments) — lint: no
- Reuse existing step definitions (incl. common `user confirms modal dialog`) before adding steps — evidence: !3488, !3503 — lint: no
- No DOM-shape locators (`locator("../..")`) or `.Mui…` selectors; add name-based `data-testid` — evidence: !3538, "cant we use data-testid nested inside that?" — lint: no
- No `waitForTimeout`; wait for a concrete condition — evidence: !3538, !3534 — lint: playwright/no-wait-for-timeout (plugin not installed)

## i18n

- No hardcoded user-facing strings anywhere, incl. units (`kg/ha`) and labels built in `*.api.ts` / helpers — evidence: !1721, !1723, !3009, !3142 (7 comments) — lint: formatjs/no-literal-string-in-jsx (JSX only)
- Missing translation/data at runtime → never hide the feature; render fallback (raw key) + log to Sentry — evidence: "we rather show it untranslated and log the error" — lint: no

## misc

- Detail routes registered in the module's `*_URLS` constant, not ad hoc paths — evidence: !1535 — lint: no
- `window.open` external URL → `"noopener,noreferrer"` — evidence: !2734 — lint: no
- Touched `.js`/`.jsx` file → convert to TS — evidence: !1883 — lint: no
