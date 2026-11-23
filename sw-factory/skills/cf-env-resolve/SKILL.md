---
name: cf-env-resolve
description: |
  Resolver of the sw-factory env lane. Turns Michal's free-text Jira comment
  that starts with "env:" (create / clone / reuse / redeploy / destroy a
  cf-dyn-env dynamic environment for a ticket) into ONE exact `cf-env.sh run`
  argument line, or one question. Read-only: never starts a pipeline, the
  worker validates the line and runs it. Invoked headless by factory-worker
  (stage env); by hand with "cf-env-resolve DEV-XXXX request=<file> out=<file>".
---

# Resolve an env request into one command

Arguments: `<KEY> request=<abs file> out=<abs file> env=<env the ticket uses now, may be empty> repos=<repo,…> branch=<ticket branch, may be empty>`.

You only translate. You never run `cf-env.sh run`, never POST to GitLab, never edit anything but `out`. No human is present: never ask in chat, an unclear request becomes an `ASK` line.

## Steps

1. Read `request`: Michal's comment, Czech or English, free text after `env:`. It is the whole instruction; the ticket itself is not.
2. Load the `cf-dyn-env:dyn-env` skill and follow its rules for names, services, hard rules (`cf-gateway` needs `farm-management`, `FRONTEND=true` needs `cf-gateway`, `agroevidence` needs `DB_SEED=data`, services that must not be deployed). Use only its read commands (`list`, `status`, `log`).
3. Pick the action:

   | request means | line |
   |---|---|
   | a new env for this ticket | `create <env>` |
   | an env like / cloned from `<src>` | `create <env>` with the recipe of `<src>` (step 5) |
   | verify this ticket on an existing env, nothing to build | `use <env>` |
   | another tag of a service, another service, another FE branch in an existing env | `redeploy <env>` |
   | cancel, remove, "zruš to" | `destroy <env>` |
   | wake / sleep / keep up at night / bigger database | `wake`, `sleep`, `set-sleep`, `set-db-size` |

   `<env>`: the name in the request; else for `create` the key lowercased (`dev-4000`), for everything else the `env` argument. Neither → `ASK`.
4. Image tags. A service named with a ticket or MR ("agroevidence z DEV-3809") → that MR's image, never a guess:
   - the MR: `glab api "groups/cleverfarm/merge_requests?state=opened&search=<KEY>&per_page=20"` (no open one → also `state=merged`), pick the one in that service's repo. Two candidates in the same repo → `ASK`, naming both.
   - its latest green pipeline: `glab api "projects/<project id>/merge_requests/<iid>/pipelines"`, then `glab api "projects/<project id>/pipelines/<pipeline id>/jobs?per_page=100"`, the image build job, its log `glab api "projects/<project id>/jobs/<job id>/trace"`: the pushed tag (`<branch lowercased>-<sha8>-amd64`). No green build → `ASK`, say which MR has none.
   - a service named without a version → untagged (the tag running on nightly). A full tag given in the request → as given.
   - chart name = repo dir, or without the `cf-` prefix when only that form is in the skill's chart list.
5. Recipe of an existing env `<src>` (clone): `glab api "projects/cleverfarm%2Fcleverfarm%2Finfrastructure%2Fcf-dev-kubernetes/deployments?environment=dynamic/<src>&per_page=100"` lists every pipeline that deployed it (`.deployable.pipeline.id`), oldest first; per pipeline `glab api "projects/cleverfarm%2Fcleverfarm%2Finfrastructure%2Fcf-dev-kubernetes/pipelines/<id>/variables"`. Start from the create, apply each successful redeploy in order: a chart's later tag replaces the earlier one, a chart added later is added, `FE_BRANCH` of the last pipeline that set it. Charts never given a tag stay untagged. No deployments → `ASK` (`<src>` does not exist).
6. Defaults for `create`, each only when the request doesn't say otherwise: `DB_SEED=data`, `FRONTEND=true`, `cf-gateway,farm-management` added to `SERVICES`, `FE_BRANCH` = the branch named in the request, else the `branch` argument when `glab api "projects/cleverfarm%2Fcleverfarm%2Fcf-farm-frontend/repository/branches/<branch>"` finds it, else left out (develop). `GIT_BRANCH=<branch>` when `branch` is set. Never `OWNER`, never `SLEEP` unless the request talks about nights or weekends.
7. `redeploy`: only what the request changes. FE only → `FRONTEND=true FE_BRANCH=<branch>`, no `SERVICES`. A service that the preview should call in the env → in `SERVICES` together with `FRONTEND=true`.
8. Write `out` (Write, the whole file):
   - line 1, EXACTLY one of
     - `RUN <action> <env> [KEY=VALUE …]` — single spaces, values without spaces or quotes, only the characters `A-Za-z0-9_.:,/-`
     - `ASK <one question in Czech, with the options you found>`
   - lines 2–6 (only after `RUN`): Czech, plain text, one line per decision Michal can't see in the command: where each tag came from (MR reference), what you defaulted, what a clone could not carry over (data created in the source env; untagged services get today's nightly).

## Rules

- One line, one action. A request that needs two (create and then something else) → the first, and say in the notes what is left.
- Never invent a tag, a branch or an env name that you did not read from GitLab or the request.
- Anything you can't resolve with a read → `ASK`. A wrong env costs 20 minutes and money, a question costs one comment.
- `destroy` of an env other than the one in `env` or named in the request: never.
- A `Conversation compacted` notice mid-run → re-read this skill and `request` before writing `out`.
