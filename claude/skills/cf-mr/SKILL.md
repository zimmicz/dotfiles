---
name: cf-mr
description: |
  Create MR for current ticket. Use when user says "create MR",
  "merge request", "push", "cf-mr", or "ready for review".
---

# Create MR for current ticket

## Steps

1. Detect current branch name, extract ticket ID (CFD-XXXX pattern)
2. Fetch ticket details via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`)
3. Detect base branch: if Jira issue type is "Hotfix" → target `production`. Otherwise check `git merge-base --is-ancestor origin/<branch> HEAD` for production, staging, develop (in that order)
4. Commit all changes: run `git add -A && git commit` with a conventional commit message based on the ticket type and changes.
5. `git push -u origin <branch>`
6. Detect repo via `glab repo view` (or `git remote get-url origin`). Determine reviewers:
   - Only if repo is `cleverfarm/cleverfarm/cf-farm-frontend`:
     - Always: @vaclav.skorpil (ID 11273826), @jaroslav.tesarik (ID 3757192)
     - If E2E tests touched (`.feature`, step definitions, page objects): also add @barbora.kovarova (ID 13281885)
   - Other repos: no default reviewers
7. Create MR via `mcp__GitLab__create_merge_request`:
   - `id`: detected repo path
   - `source_branch`: current branch
   - `target_branch`: detected base branch (not always develop!)
   - `title`: `type: [CFD-XXXX] english description` (conventional commits: feat, fix, chore, refactor, test)
   - `description`: ticket summary + brief list of changes from git diff
   - `assignee_ids`: [18554466] (michal.zimmermann1)
   - `reviewer_ids`: per step 6 (empty for non-cf-farm-frontend repos)
8. Transition ticket to "In Review" via `atlassian.getTransitionsForJiraIssue` then `atlassian.transitionJiraIssue`
9. Return MR URL
10. **Amplify deploy preview** (only if repo is `cleverfarm/cleverfarm/cf-farm-frontend`) — needed only to **first-connect the branch** or to **switch backend**. Once a backend job has run for the branch, `enable_branch_auto_build=true` rebuilds the preview on every push automatically — no need to replay. Preview URL = `https://<branch-slug-lowercased>.dev.cleverfarm.cz/`.
    - First check if the branch is already connected: if any `amplify-deploy` job succeeded on an earlier pipeline for this branch (or the preview URL already resolves), say "preview auto-rebuilds from your push" and **skip the prompt** unless switching backend.
    - Otherwise (first MR for the branch, or switching backend) ask which backend, then trigger the matching manual MR-pipeline job (`when: manual`, stage `amplify-deploy`; runs `terraform apply` for that backend + starts the build):
      - `use Nightly backend` (default) · `use Staging backend` · `use vskorpil backend` · `use psimek backend` · `use zruzicka backend` · `use rkoslab backend` · skip
    - The MR pipeline must exist first (created on push) — if not yet present, wait a few seconds and retry the pipeline lookup.
    - Trigger the chosen job via `glab api` (project path URL-encoded):
      ```bash
      PROJECT=cleverfarm%2Fcleverfarm%2Fcf-farm-frontend
      JOBNAME="use Nightly backend"   # the picked option
      PIPE=$(glab api "projects/$PROJECT/merge_requests/<iid>/pipelines" | jq '.[0].id')
      JOB=$(glab api "projects/$PROJECT/pipelines/$PIPE/jobs" | jq --arg n "$JOBNAME" '.[] | select(.name==$n) | .id')
      glab api --method POST "projects/$PROJECT/jobs/$JOB/play"
      ```
    - Report the deploy job URL + the preview URL. Build takes a few min; commit status `amplify` + an MR comment with the URL land when done.
11. **Log time** — prompt: "Log time on CFD-XXXX? (duration like `2h`, or `n` to skip)". If duration provided, invoke `cf-log-time` flow with that duration.
12. **Quick retro** — ask: "Any mid-impl pivots from requirement gaps? (y/n + brief note)". Append row to `memory/workflow-retro-log.md`:
    `| {date} | {ticket} | {pivot count} | {what was caught/missed} | {notes} |`
