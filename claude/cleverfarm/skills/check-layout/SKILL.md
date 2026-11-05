---
name: check-layout
description: |
  Visually verify changed UI components in a real browser before calling a UI task done.
  Use when user says "check layout", "check-layout", "check visually", "screenshot it", or after
  any frontend change (component, styling, Storybook story) before reporting success.
  Typecheck/lint/unit green is NOT done for UI work.
---

# Verify UI

Arguments: `[base-branch]` (default `develop`)

Goal: catch stretched flex children, overflowing widths, double borders, wrong icons, mobile breakage BEFORE user does.

## Steps

1. Changed UI files: `git diff --name-only <base>...HEAD -- '*.tsx' '*.css' '*.ts' '*.stories.*'`. Nothing UI-related → say so, stop.
2. Map each changed component to its stories (`*.stories.tsx` next to it, or grep for imports). No story → find the app route that renders it.
3. Storybook running? `curl -sf localhost:6006/index.json`. If not: start `pnpm run storybook` in background, wait for `/index.json`. Story ids come from `/index.json` (`entries[*].id`, filter by `importPath`).
4. Per story, with Playwright MCP (`mcp__playwright__*`):
   - `browser_navigate` to `http://localhost:6006/iframe.html?id=<story-id>&viewMode=story`
   - `browser_resize` 1280x800 → `browser_take_screenshot` (`type: jpeg`, `scale: css`)
   - `browser_resize` 375x800 → `browser_take_screenshot` (same)
   - `browser_console_messages` (errors only)
   - Cover every state: default, hover/focus/open (click via `browser_snapshot` refs), disabled, error, long text, empty.
5. App route needing auth (not Storybook): log in with Playwright MCP yourself, exactly as `/cf-holdout-ui` step 4 (`fe-automation@cleverfarm.cz`, `FE_AUTOMATION_PSW` from cf-farm-frontend `.env`), works on localhost and `https://<branch>.dev.cleverfarm.cz`. Never claim you can't log in or ask the user to; only a failed attempt (state the error) counts.
6. Compare against Figma frame / Jira designer comments (TICKET.md, PLAN.md) if present. Check: widths, spacing, borders (doubles), icon, alignment, flex stretching, overflow, truncation.
7. Issues found → fix, re-screenshot, repeat. Shared/DS component glitch → fix in DS + story covering the state, not app override.
8. Report: table `story/route | viewport | verdict | note`, plus screenshot paths. Skip only after an actual attempt failed: quote the exact error (command + output) in the report and MR description. "Headless", "no logged-in browser", "will run in CI/preview" are never valid reasons — Playwright MCP runs headless and logs in per step 5.

## Rules

- Never claim UI done without screenshots actually taken and looked at.
- Screenshots go to scratchpad, not repo.
- Run lint + format after any fix.
- Stop storybook you started when done.
