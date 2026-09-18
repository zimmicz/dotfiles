---
name: cf-ticket
description: |
  Full ticket lifecycle. Use when user says "start ticket", "work on ticket",
  "cf-ticket", or provides a ticket ID (e.g. DEV-XXXX) to work on end-to-end.
---

# Full ticket lifecycle

Ticket: $ARGUMENTS

Orchestrates the entire workflow for a ticket. Each phase has a STOP gate — wait for user confirmation before proceeding.

## Phase 0: Goal

1. Run `/mattpocock-skills:grilling` — focus the grilling on: what is the actual goal, what problem does this solve, what does success look like, what are the constraints
2. **STOP** — summarize the agreed goal in 2-3 sentences, confirm with user before continuing

## Phase 1: Setup

1. If no ticket ID in arguments, ask for one
2. Check if ticket has a parent/blocking ticket that implies a non-develop base branch. If so, surface it: "Base on `<parent-branch>`?" Also check for sibling branches off the same parent to avoid overlap.
3. If already inside a worktree with `TICKET.md` (the `cf-work` script created it), skip to Phase 2. Otherwise run `/cf-worktree` for the ticket (thin wrapper over the same `~/bin/cf-work` script), passing the base branch if identified in step 2.
4. **STOP** — confirm worktree ready

## Phase 2: Requirements

4. Run `/cf-work`
5. **STOP** — wait for answers to `[?]` items, confirmation of `[!]` items
6. If `PLAN.md` exists in worktree root, read it first — it may contain a prior plan or user notes. Incorporate, don't overwrite blindly.
7. Create a detailed stepped plan of implementation
8. **STOP** — confirm plan with user
9. Write confirmed plan to `PLAN.md` in worktree root
10. Post ONE Czech Jira comment (via `~/bin/jira add-comment <ticket-id>`) covering: requirements, resolved assumptions, and the confirmed plan. This is the only Jira comment in the flow.

## Phase 3: Implementation

11. User says "go" or gives direction — start coding
12. Stop after every plan step, validate with the user
13. When implementation complete, **STOP** — ask "Ready to verify?"

## Phase 4: Verify

14. Run tests, lint, typecheck (`nr test`, `nr lint`, `nr typecheck` or project equivalents). Fix failures.
15. **STOP** if any test/lint failures required non-trivial fixes — show what changed

## Phase 5: Self-review

16. Run `/code-review` on changed files — correctness bugs
17. Run `/simplify` on changed files — cleanup/reuse
18. Fix issues found from both passes
19. **STOP** — ask "Ready to create MR?"

## Phase 6: MR

20. Run `/cf-mr`
21. **STOP** — "MR created. Go get it reviewed. Come back after merge and run `/cf-feedback`."

## Rules

- Never skip a STOP gate
- If user wants to skip a phase (e.g. already has worktree), jump to the right phase
- Keep context concise between phases — summarize, don't repeat everything
