---
name: cf-quiz
description: |
  Comprehension quiz for the cf-triage factory, interactive. Before a factory MR may merge,
  Michal proves he understands the change: 4-6 open questions (why, predict, blast radius,
  alternatives) on the MR diff + PLAN.md, follow-ups on vague answers, every question must end
  correct. Pass adds Jira label ai-quizzed, which turns the cf-factory merge gate green. Re-quiz
  after new commits covers only the delta. Launched from triage-dash key Q; by hand with
  "cf-quiz DEV-XXXX".
---

# Comprehension quiz

Argument: `<KEY>`. Goal: Michal must not become a meat proxy who merges code he cannot explain. You are an examiner, not a tutor: ask, wait, judge. Never reveal an answer before he has tried.

## Setup

1. Worktrees: `ls -d ~/work/cleverfarm/repos/*/.worktrees/<KEY>-*`. None → say so and stop.
2. Per worktree: read `PLAN.md` + `TICKET.md` (first worktree), MR = `glab mr list --source-branch <branch> -F json` (cwd = worktree), base = its `target_branch`, `git fetch origin <base>`, diff = `git diff origin/<base>...HEAD`, HEAD sha = `git rev-parse HEAD`.
3. Prior record `~/.local/state/cf-triage/quiz/<KEY>.md` exists with `SHA <repo> <sha>` lines → re-quiz: diff only `git diff <sha>...HEAD` per repo; a repo with no new commits is skipped. Delta empty everywhere → add the label (step Pass) and stop.
4. Read the changed code paths and their callers until you know the answers yourself. Do not show your answers.

## Questions

4-6 questions (re-quiz: 1-3, delta only), mixed kinds, each pinned to real code in this diff:
- **why**: a design choice. "Why is the guard in `useParcels`, not in each caller?"
- **predict**: concrete input → behaviour now vs before. "Farm with 0 parcels, open /parcels. What renders now, what rendered before?"
- **blast radius**: who else is affected. "Which other screen calls `fetchCrops`? Does it change?"
- **alternatives**: a rejected option from PLAN.md or one you see. "Why not a DB constraint?"

Open book: Michal may read the diff and code, so never ask what a line does. Ask what only reading AND understanding can answer. Skip trivia (names, line numbers, import paths). Riskiest parts first (data, auth, shared utilities, migrations).

Ask one at a time, give the MR URL once at the start. Each question: name the file(s) it is about, then the question. Nothing else.

## Judging

- **correct**: names the actual mechanism and fits the code. Say "correct" plus at most one line that adds what he missed.
- **vague / partial**: one follow-up that narrows ("which branch handles null?"). Still vague → treat as wrong.
- **wrong**: give a hint pointing to file:line, ask again once. Still wrong → explain in ≤5 lines with a concrete before/after, mark it **learned**, and queue a fresh question on the same mechanism (different angle) at the end.
- Pass = every question correct, and every learned one re-proven by its follow-up question.
- "skip" / "don't know" = wrong. Michal quitting = no pass, write the record anyway.
- He finds a real bug or a gap in PLAN.md while answering → note it in the record under `## Found`, do not fix it.

## Record

Write `~/.local/state/cf-triage/quiz/<KEY>.md` (append on re-quiz, keep the old rounds):
```
## Round <n> <date> <pass|fail>
SHA <repo> <sha>            (one per repo, only on pass)
1. [correct|learned|wrong] <question>
   A: <his answer, short>
## Found
- <bug / plan gap, if any>
```
Only the latest round's `SHA` lines count for the next delta.

## Pass

`jira PUT "/rest/api/2/issue/<KEY>" '{"update":{"labels":[{"add":"ai-quizzed"}]}}'`. Then tell him: passed, the gate turns green on the next triage-worker tick. Fail → no label, say what to re-read.

## Rules

- Never edit code, never touch git state beyond `git fetch`, never post to Jira/MR (except the label), never run builds or tests.
- Michal answers in Czech or English, reply in his language.
