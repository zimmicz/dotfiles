---
name: cf-ticket
description: |
  Full ticket lifecycle. Use when user says "start ticket", "work on ticket",
  "cf-ticket", or provides a CFD-XXXX ticket ID to work on end-to-end.
---

# Full ticket lifecycle

Ticket: $ARGUMENTS

Orchestrates the entire workflow for a ticket. Each phase has a STOP gate — wait for user confirmation before proceeding.

## Phase 0: Goal

1. Run `/grill-me` — focus the grilling on: what is the actual goal, what problem does this solve, what does success look like, what are the constraints
2. **STOP** — summarize the agreed goal in 2-3 sentences, confirm with user before continuing

## Phase 1: Setup

1. If no ticket ID in arguments, ask for one
2. Run `/cf-worktree` for the ticket
3. **STOP** — confirm worktree ready

## Phase 2: Requirements

4. Run `/cf-work` (includes playbook scan)
5. **STOP** — wait for answers to `[?]` items, confirmation of `[!]` items
6. Create a detailed stepped plan of implementation
7. Post a czech JIRA comment with the plan for the ticket

## Phase 3: Implementation

7. User says "go" or gives direction — start coding
8. Stop after every plan step, validate with the user
9. When implementation complete, **STOP** — ask "Ready for self-review?"

## Phase 4: Self-review

10. Run `/simplify` on changed files
11. Fix any issues found
12. **STOP** — ask "Ready to create MR?"

## Phase 5: MR

13. Run `/cf-mr`
14. **STOP** — "MR created. Go get it reviewed. Come back after merge and run `/cf-feedback`."

## Rules

- Never skip a STOP gate
- If user wants to skip a phase (e.g. already has worktree), jump to the right phase
- Keep context concise between phases — summarize, don't repeat everything
