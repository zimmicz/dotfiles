---
name: cf-worktree
description: |
  Create worktree for a Jira ticket. Use when user says "worktree",
  "cf-worktree", "setup ticket", or "new branch for ticket".
---

# Create worktree for a Jira ticket

Arguments: `<ticket-id> [base-branch]` (base-branch defaults to `develop`; auto-switches to `production` for hotfix tickets)

> **Branch name limit: 45 chars max** — branch name + `.dev.cleverfarm.cz` (18 chars) must stay ≤ 63 chars total (Amplify subdomain limit).

## Steps

1. Fetch ticket via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`, fields: summary, description, status, assignee, priority, comment, issuetype, labels)
2. Determine base branch (if not explicitly provided):
   - If issue type name contains "hotfix" (case-insensitive) OR labels contain "hotfix": use `production`
   - Otherwise: use `develop`
3. In parallel:
   - Assign to me via `atlassian.editJiraIssue` (accountId: `712020:d8b09ea3-beb7-4afc-af06-6d7119474a4f`) — skip if already assigned to me
   - Transition to "In Progress" via `atlassian.getTransitionsForJiraIssue` then `atlassian.transitionJiraIssue` — skip if already In Progress
4. Create git worktree:
   - `git fetch origin <base-branch>`
   - Translate Czech ticket title to English
   - Generate branch name: `DEV-XXXX-kebab-case-desc` (max 45 chars total)
   - `git worktree add .worktrees/DEV-XXXX-desc origin/<base-branch> -b DEV-XXXX-desc`
5. Copy files into the new worktree:
   - `cp .env .worktrees/DEV-XXXX-desc/.env` (if `.env` exists in repo root)
6. Install dependencies in the worktree:
   - `cd .worktrees/DEV-XXXX-desc && npm ci` (if `package.json` exists)
   - `codegraph init .worktrees/DEV-XXXX-desc` (if `codegraph` CLI is installed; run from repo root, don't fail the flow if it errors)
7. **Size the model** from the `description` already fetched in step 1:
   - Count acceptance criteria / checklist items / distinct sub-tasks.
   - Note description length (short = a few lines; long = multiple paragraphs or a checklist).
   - Map to a tier:
     - **`haiku`** (trivial): purely mechanical, no real logic — rename, typo fix, config/copy value change, one-liner.
     - **`sonnet`** (small): short description AND ≤ 2 acceptance criteria — single-component fix.
     - **`opus`** (large): long description OR ≥ 3 acceptance criteria OR clearly multi-part (several distinct sub-tasks / cross-cutting changes).
     - On the fence → prefer `opus` and say so.
   - Do NOT auto-launch. Print the recommendation with a one-line reason (e.g. `Sizing: 4 acceptance criteria, multi-part → opus`) and let the user override the tier.
8. Output for the user (fill `<tier>` from step 7):
   ```
   cd .worktrees/DEV-XXXX-desc && claude --model <tier>
   ```
   Tell the user to run this to start working on the ticket with `/cf-work DEV-XXXX`.

## Rules

- Branch name max 45 characters (Amplify subdomain: branch + `.dev.cleverfarm.cz` ≤ 63)
- No Czech characters in branch names
- Do NOT run commands inside the worktree beyond `npm ci`
- Do NOT summarize the ticket or start implementation
- Model tier for the worktree session is sized from the ticket description (length + acceptance-criteria count): `haiku` for trivial mechanical tickets, `sonnet` for small, `opus` for large. On the fence → `opus`. Recommend it in the launch command; the user may override.
