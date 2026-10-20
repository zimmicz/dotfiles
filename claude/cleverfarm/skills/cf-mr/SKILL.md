---
name: cf-mr
description: |
  Create MR for current ticket. Use when user says "create MR",
  "merge request", "push", "cf-mr", or "ready for review".
  With `--headless`: draft MR via glab, no reviewers, no prompts.
---

# Create MR for current ticket

## Steps

0. **Multi-repo**: `TICKET.md` `## Repos` lists more than one worktree → run steps 3–7 and 10 once per worktree (`cd <worktree>` first; `git`, `glab` and repo detection are cwd-based), non-primary (BE) repos first, cf-farm-frontend last. Same branch everywhere. Every MR description lists the sibling MR URLs (`Depends on: <BE MR url>` in the FE one). Steps 8, 11, 12 run once.
1. Detect current branch name, extract ticket ID (`[A-Z]+-\d+` pattern)
2. Get ticket details — cheapest source first: if `TICKET.md` exists in worktree root, use it (summary, issue type). Otherwise fetch via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`)
3. Detect base branch: if Jira issue type is "Hotfix" → target `production`. Otherwise check `git merge-base --is-ancestor origin/<branch> HEAD` for production, staging, develop (in that order)
4. Commit all changes: run `git add -A && git commit` with a conventional commit message based on the ticket type and changes.
5. `git push -u origin <branch>`
6. Detect repo path from `git remote get-url origin`. Determine reviewers:
   - Only if repo is `cleverfarm/cleverfarm/cf-farm-frontend`:
     - Always: @vaclav.skorpil (ID 11273826), @jaroslav.tesarik (ID 3757192)
     - If E2E tests touched (`.feature`, step definitions, page objects): also add @barbora.kovarova (ID 13281885)
   - Other repos: no default reviewers
7. Create MR via `mcp__GitLab__create_merge_request`:
   - `id`: detected repo path
   - `source_branch`: current branch
   - `target_branch`: detected base branch (not always develop!)
   - `title`: `type: [DEV-XXXX] english description` (conventional commits: feat, fix, chore, refactor, test; use the actual ticket key)
   - `description`: first line = 🤖 prefix from `~/.claude/my-voice.md` "Posted by Claude" (language of the body), then ticket summary + brief list of changes from git diff
   - `assignee_ids`: [18554466] (michal.zimmermann1)
   - `reviewer_ids`: per step 6 (empty for non-cf-farm-frontend repos)
8. Transition ticket to review via `atlassian.getTransitionsForJiraIssue` then `atlassian.transitionJiraIssue` — pick the transition whose target status is "Review" (or "In Review" if that's what exists)
9. Return MR URL (all of them for multi-repo)
10. **Amplify deploy preview** — only if repo is `cleverfarm/cleverfarm/cf-farm-frontend`: read `references/amplify-preview.md` (in this skill's directory) and follow it. Other repos: skip.
11. **Log time** — prompt: "Log time on <ticket>? (duration like `2h`, or `n` to skip)". If duration provided, invoke `cf-log-time` flow with that duration.
12. **Quick retro** — ask: "Any mid-impl pivots from requirement gaps? (y/n + brief note)". Append row to `memory/workflow-retro-log.md`:
    `| {date} | {ticket} | {pivot count} | {what was caught/missed} | {notes} |`

## Headless mode (`--headless` in arguments)

No user present. Steps 1–5 as above, then:

- Step 6: no reviewers, any repo.
- Step 7: `glab mr create --draft --yes --source-branch <branch> --target-branch <base> --assignee michal.zimmermann1 --title "<title>" --description "<desc>"` (GitLab MCP not available headless). Title/description rules as in step 7; description starts with the 🤖 prefix line per `~/.claude/my-voice.md` "Posted by Claude".
- Step 8: skip — ticket stays In Progress while the MR is draft.
- Step 10: cf-farm-frontend only — play `use Nightly backend` without asking, report preview URL.
- Steps 11–12: skip.
