# Create worktree for a Jira ticket

Arguments: `<ticket-id> [base-branch]` (base-branch defaults to `develop`)

## Steps

1. Fetch ticket via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`, fields: summary, description, status, assignee, priority, comment)
2. In parallel:
   - Assign to me via `atlassian.editJiraIssue` (accountId: `712020:d8b09ea3-beb7-4afc-af06-6d7119474a4f`) — skip if already assigned to me
   - Transition to "In Progress" via `atlassian.getTransitionsForJiraIssue` then `atlassian.transitionJiraIssue` — skip if already In Progress
3. Create git worktree:
   - `git fetch origin <base-branch>`
   - Translate Czech ticket title to English
   - Generate branch name: `CFD-XXXX-kebab-case-desc` (max 46 chars total)
   - `git worktree add .worktrees/CFD-XXXX-desc origin/<base-branch> -b CFD-XXXX-desc`
4. Output for the user:
   ```
   cd .worktrees/CFD-XXXX-desc && claude
   ```
   Tell the user to run this to start working on the ticket with `/cf-work CFD-XXXX`.

## Rules

- Branch name max 46 characters
- No Czech characters in branch names
- Do NOT attempt to run commands inside the worktree — only create it
- Do NOT summarize the ticket or start implementation
