# Amplify deploy preview (cf-farm-frontend only)

Needed only to **first-connect the branch** or to **switch backend**. Once a backend job has run for the branch, `enable_branch_auto_build=true` rebuilds the preview on every push automatically — no need to replay. Preview URL = `https://<branch-slug-lowercased>.dev.cleverfarm.cz/`.

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
