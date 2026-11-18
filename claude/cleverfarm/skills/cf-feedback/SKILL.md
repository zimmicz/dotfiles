---
name: cf-feedback
description: |
  Capture MR review feedback as review standards (or lint rules). Use when user says
  "feedback", "lessons", "what did reviewers say", or "cf-feedback". Lessons land in
  the repo's .claude/skills/review-checklist/SKILL.md (cross-repo ones in ~/dotfiles/claude/cleverfarm/standards/common.md), which the independent reviewer checks diffs against.
---

# Capture review feedback as review standards

Ticket: $ARGUMENTS

## Steps

1. If no ticket ID in arguments, detect from current branch (`[A-Z]+-\d+` pattern)
2. Detect repo path from `git remote get-url origin`. Fetch MR + review comments in one shell call: `glab mr view <branch> --comments`. Fallback if glab fails: `mcp__GitLab__search` (scope: merge_requests, search: branch name, project: detected repo path) then `mcp__GitLab__get_workitem_notes` (type: merge_request).
3. Keep human notes incl. mine, drop bots and factory notes (body starts with `🤖` or contains "writing on Michal"). Present a summary of what reviewers flagged.
4. If no MR found or no comments, fall back to asking: "What needed changing and why?"
5. User can add extra context or lessons beyond what's in the MR comments.
6. Extract actionable patterns (max 3). For each lesson:
   - **Rule**: what to do or avoid (imperative, one line)
   - **Evidence**: MR iid
   - **Lint**: can ESLint / TS / a test enforce it? Name the rule or plugin, else `no`
7. Read the repo `.claude/skills/review-checklist/SKILL.md`, root `CLAUDE.md` Coding Rules + Active Migrations and `~/dotfiles/claude/cleverfarm/standards/common.md`. In `CLAUDE.md` → skip; in the checklist → add `!<iid>` to that line.
8. Present extracted lessons for confirmation. User can edit/reject/add.
9. Write confirmed lessons as `- [ ] **Title** — text (!<iid>)` into the matching checklist section (cross-repo → `common.md`).
10. Ticket MR open → commit `docs: [<KEY>] review lesson: <title>` on its branch, push. MR merged, or repo has no checklist → propose a chore ticket, write nothing until answered.
11. Lint-enforceable lesson → offer a separate Jira ticket for the ESLint rule (don't create it unasked). Team-relevant rule → suggest adding it to the repo `CLAUDE.md` via ticket + MR. A naming / vocabulary lesson in a repo with `GLOSSARY.md` → a glossary entry or `_Avoid_` word there (via ticket + MR), not a checklist line.

## Rules

- Keep rules actionable and specific — "prefer X over Y" not "be careful with X"
- Max 3 lessons per feedback session — forces prioritization
- Ignore trivial comments (typos, nits) — focus on patterns that'd recur
