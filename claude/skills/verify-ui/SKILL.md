---
name: verify-ui
description: |
  Visually verify changed UI components in a real browser before calling a UI task done.
  Use when user says "verify ui", "verify-ui", "check visually", "screenshot it", or after
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
5. App route needing auth/VPN (not Storybook): use claude-in-chrome (`mcp__claude-in-chrome__*`, real logged-in session) or give user the URL (`https://<branch>.dev.cleverfarm.cz/...` or localhost path). Playwright MCP has fresh profile, no Auth0 session.
6. Compare against Figma frame / Jira designer comments (TICKET.md, PLAN.md) if present. Check: widths, spacing, borders (doubles), icon, alignment, flex stretching, overflow, truncation.
7. Issues found → fix, re-screenshot, repeat. Shared/DS component glitch → fix in DS + story covering the state, not app override.
8. Report: table `story/route | viewport | verdict | note`, plus screenshot paths. Cannot verify (no browser, auth wall) → say so explicitly, put it in MR description.

## Rules

- Never claim UI done without screenshots actually taken and looked at.
- Screenshots go to scratchpad, not repo.
- Run lint + format after any fix.
- Stop storybook you started when done.
