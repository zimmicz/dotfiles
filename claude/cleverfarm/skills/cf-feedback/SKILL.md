---
name: cf-feedback
description: |
  Capture MR review feedback as review standards (or lint rules). Use when user says
  "feedback", "lessons", "what did reviewers say", or "cf-feedback". Lessons land in
  ~/dotfiles/claude/cleverfarm/standards/<repo>.md, which the independent reviewer in /cf-ticket checks diffs against.
---

# Capture review feedback as review standards

Ticket: $ARGUMENTS

## Steps

1. If no ticket ID in arguments, detect from current branch (`[A-Z]+-\d+` pattern)
2. Detect repo path from `git remote get-url origin`. Fetch MR + review comments in one shell call: `glab mr view <branch> --comments`. Fallback if glab fails: `mcp__GitLab__search` (scope: merge_requests, search: branch name, project: detected repo path) then `mcp__GitLab__get_workitem_notes` (type: merge_request).
3. Filter to human reviewer comments (not yours, not bots). Present a summary of what reviewers flagged.
4. If no MR found or no comments, fall back to asking: "What needed changing and why?"
5. User can add extra context or lessons beyond what's in the MR comments.
6. Extract actionable patterns (max 3). For each lesson:
   - **Rule**: what to do or avoid (imperative, one line)
   - **Evidence**: MR iid
   - **Lint**: can ESLint / TS / a test enforce it? Name the rule or plugin, else `no`
7. Read `~/dotfiles/claude/cleverfarm/standards/<repo dir>.md` (create with a `# <repo> review standards` heading if missing) and the repo `CLAUDE.md` "Coding Rules". Already covered → add the MR to that rule's evidence instead of a new line.
8. Present extracted lessons for confirmation. User can edit/reject/add.
9. Append confirmed lessons to the standards file under the matching category, format: `- <rule> — evidence: !<iid> — lint: <rule|no>`
10. Lint-enforceable lesson → offer a separate Jira ticket for the ESLint rule (don't create it unasked). Team-relevant rule → suggest adding it to the repo `CLAUDE.md` via ticket + MR.

## Rules

- Keep rules actionable and specific — "prefer X over Y" not "be careful with X"
- Max 3 lessons per feedback session — forces prioritization
- Ignore trivial comments (typos, nits) — focus on patterns that'd recur
