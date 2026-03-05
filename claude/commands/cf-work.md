# Work on a Jira ticket

Ticket: $ARGUMENTS

Run this command inside a worktree created by `/cf-worktree`.

## Steps

1. Fetch ticket via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`, fields: summary, description, status, assignee, priority, comment)
2. Read ticket comments — they often contain analysis, decisions, or scope clarifications
3. Summarize ticket requirements concisely (incorporate insights from comments)
4. Ask clarifying questions if anything is ambiguous, missing, or underspecified — do NOT assume
5. Post a Jira comment **in Czech** (via `atlassian.addCommentToJiraIssue`) summarizing key decisions, assumptions, and implementation approach taken during the session

## Rules

- Commit format: `type: [CFD-XXXX] english description` (conventional commits)
- Types: feat, fix, chore, refactor, test
- No Czech characters in commits
- All Jira comments must be written in Czech
