---
name: cf-feedback
description: |
  Capture MR review feedback as coding lessons. Use when user says
  "feedback", "lessons", "what did reviewers say", or "cf-feedback".
---

# Capture review feedback as coding lessons

Ticket: $ARGUMENTS

## Steps

1. If no ticket ID in arguments, detect from current branch (CFD-XXXX pattern)
2. Detect MR for current branch via `mcp__GitLab__search` (scope: merge_requests, search: branch name, project: `cleverfarm/cleverfarm/cf-farm-frontend`)
3. If MR found, fetch review comments via `mcp__GitLab__get_workitem_notes` (type: merge_request). Filter to reviewer comments (not yours). Present a summary of what reviewers flagged.
4. If no MR found or no comments, fall back to asking: "What needed changing and why?"
5. User can add extra context or lessons beyond what's in the MR comments.
6. Extract actionable patterns (max 3). For each lesson, determine:
   - **Category**: composition | state | api | types | styling | testing | patterns | performance | accessibility
   - **Rule**: what to do or avoid (imperative, concise)
   - **Why**: the reasoning — what went wrong or what's better
7. Read `.claude/coding-playbook.md` — check for existing similar rules to deduplicate
8. Present extracted lessons for confirmation. User can edit/reject/add.
9. Append confirmed lessons to `.claude/coding-playbook.md`
10. If a lesson is general enough for all projects, suggest adding it to project CLAUDE.md

## Rules

- Keep rules actionable and specific — "prefer X over Y" not "be careful with X"
- Deduplicate: if a similar rule exists, update it (add source ticket) instead of adding a new row
- Max 3 lessons per feedback session — forces prioritization
- Ignore trivial comments (typos, nits) — focus on patterns that'd recur
