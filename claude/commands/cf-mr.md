# Create MR for current ticket

## Steps

1. Detect current branch name, extract ticket ID (CFD-XXXX pattern)
2. Fetch ticket details via `atlassian.getJiraIssue` (cloudId: `1f10ad4f-84e7-4939-b19a-bc8e7bf0553d`)
3. Detect base branch: check `git merge-base --is-ancestor origin/<branch> HEAD` for production, staging, develop (in that order) to determine target branch
4. Commit all changes: run `git add -A && git commit` with a conventional commit message based on the ticket type and changes.
5. `git push -u origin <branch>`
6. Determine reviewers:
   - Always: @vaclav.skorpil (ID 11273826), @jaroslav.tesarik (ID 3757192)
   - If E2E tests touched (`.feature`, step definitions, page objects): also add @barbora.kovarova (ID 13281885)
7. Create MR via `glab mr create`:
   - `--source-branch`: current branch
   - `--target-branch`: detected base branch (not always develop!)
   - `--title`: `type: [CFD-XXXX] english description` (conventional commits: feat, fix, chore, refactor, test)
   - `--description`: ticket summary + brief list of changes from git diff
   - `--reviewer vaclav.skorpil,jaroslav.tesarik` (add `barbora.kovarova` if E2E touched)
   - `--no-editor`
8. Transition ticket to "In Review" via `atlassian.getTransitionsForJiraIssue` then `atlassian.transitionJiraIssue`
9. Return MR URL
