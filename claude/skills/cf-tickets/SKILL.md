---
name: cf-tickets
description: |
  Show active sprint tickets. Use when user says "my tickets",
  "sprint", "what's open", "cf-tickets", or "backlog".
---

# Show my active sprint tickets

## Steps

1. Query Jira via `atlassian.searchJiraIssuesUsingJql`:
   - cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`
   - JQL: `assignee = "712020:d8b09ea3-beb7-4afc-af06-6d7119474a4f" AND sprint in openSprints() AND status in ("To Do", "Open", "In Progress", "Review") ORDER BY priority DESC, status ASC`
   - fields: summary, status, priority, issuetype
2. Display as compact table: ticket ID | status | priority | title
