#!/bin/zsh
set -u
W=${0:A:h:h}/bin/factory-worker
source <(sed -n '/^DATA_RE=/,/^}/p' $W)
source <(grep '^POSTED_RE=' $W)
source <(grep -E '^(BLOCKERS_JQ|BE_MRS_JQ|ENV_PICK_JQ)=' $W)
T=$(mktemp -d)
trap 'rm -rf $T' EXIT
fails=0
check() { [[ "$2" == "$3" ]] && echo "ok   $1" || { echo "FAIL $1: got '$2' want '$3'"; fails=1 } }

print -l 'VERDICT: blocked' 'S1 Map: pass — ok' 'S7 Group: blocked — DATA: no group fixture' 'S8 Telematics: blocked — DATA: missing machine' >$T/data
check "data-only → ids" "$(data_only $T/data)" "S7,S8"

print -l 'VERDICT: blocked' 'S1 Map: pass — ok' 'S2 X: blocked — ENV: Playwright CONNECT_TIMEOUT' 'S3 Y: blocked — DATA: none' >$T/env
check "ENV block → not data-only" "$(data_only $T/env; echo $?)" "1"

print -l 'VERDICT: blocked' 'S1 Map: blocked — DATA: none' >$T/nopass
check "no pass → not data-only" "$(data_only $T/nopass; echo $?)" "1"

print -l 'VERDICT: fail' 'S1 Map: pass — ok' 'S2 X: fail — wrong colour' 'S3 Y: blocked — DATA: none' >$T/fail
check "fail → not data-only" "$(data_only $T/fail; echo $?)" "1"

print -l 'VERDICT: blocked' 'S1 Map: pass — ok' 'S2 X: blocked — DATA: none' 'S3 Y: not run — verifier stopped' >$T/stopped
check "not run → not data-only" "$(data_only $T/stopped; echo $?)" "1"

print -l '## S1: Multi-company user loads users' 'Given: x' '' '## S2: Dropdown lists the other company' 'Given: y' >$T/h1
print -l 'S1: Klik na parcelu' 'Given: x' '' 'S2: Změna plodiny' >$T/h2
check "can-frontend ## headings" "$(sed -nE "$NOPREVIEW_SED" $T/h1 | head -1)" "S1 Multi-company user loads users: blocked — NO PREVIEW: can-frontend has no reachable preview, check by hand"
check "can-frontend plain headings" "$(sed -nE "$NOPREVIEW_SED" $T/h2 | tail -1)" "S2 Změna plodiny: blocked — NO PREVIEW: can-frontend has no reachable preview, check by hand"
posted() { print -r -- "$1" | grep -qE "$POSTED_RE" && echo yes || echo no }
check "posted: MR note" "$(posted '10:01:02 🔧 Bash {"command":"glab mr note 4526 -m \"🤖 …"}')" yes
check "posted: thread reply" "$(posted '10:01:02 🔧 Bash {"command":"glab api -X POST projects/1/merge_requests/2/discussions/abc/notes -f body=x"}')" yes
check "posted: jira comment" "$(posted '10:01:02 🔧 Bash {"command":"jira POST /rest/api/2/issue/DEV-1/comment x"}')" yes
check "posted: push" "$(posted '10:01:02 🔧 Bash {"command":"git push origin DEV-1-x"}')" yes
check "not posted: read threads" "$(posted '10:01:02 🔧 Bash {"command":"glab api projects/1/merge_requests/2/discussions?per_page=100"}')" no
check "not posted: list notes" "$(posted '10:01:02 🔧 Bash {"command":"glab mr note list 4526"}')" no
check "not posted: read issue" "$(posted '10:01:02 🔧 Bash {"command":"jira GET /rest/api/2/issue/DEV-1"}')" no
LINKS='{"fields":{"issuelinks":[
 {"type":{"inward":"is blocked by"},"inwardIssue":{"key":"DEV-1","fields":{"status":{"name":"Review","statusCategory":{"key":"indeterminate"}}}}},
 {"type":{"inward":"is blocked by"},"inwardIssue":{"key":"DEV-2","fields":{"status":{"name":"On nightly","statusCategory":{"key":"indeterminate"}}}}},
 {"type":{"inward":"is blocked by"},"inwardIssue":{"key":"DEV-3","fields":{"status":{"name":"Done","statusCategory":{"key":"done"}}}}},
 {"type":{"inward":"is blocked by"},"outwardIssue":{"key":"DEV-4","fields":{"status":{"name":"Review","statusCategory":{"key":"indeterminate"}}}}},
 {"type":{"inward":"relates to"},"inwardIssue":{"key":"DEV-5","fields":{"status":{"name":"Review","statusCategory":{"key":"indeterminate"}}}}}]}}'
check "blockers: only open is-blocked-by" "$(jq -r "$BLOCKERS_JQ" <<<$LINKS | paste -sd, -)" "DEV-1"
MRS='[{"state":"opened","source_branch":"DEV-3809-pasture-action","references":{"full":"cleverfarm/cleverfarm/agroevidence!1826"}},
 {"state":"opened","source_branch":"DEV-3873-fe-pasture-form","references":{"full":"cleverfarm/cleverfarm/cf-farm-frontend!4557"}},
 {"state":"merged","source_branch":"DEV-38099-other","references":{"full":"cleverfarm/cleverfarm/agroevidence!1"}},
 {"state":"merged","source_branch":"dev-3809-fix","references":{"full":"cleverfarm/cleverfarm/cf-gateway!7"}}]'
check "BE MRs: branch prefix, no FE" "$(jq -r --arg k dev-3809 "$BE_MRS_JQ" <<<$MRS | paste -sd, -)" "opened agroevidence,merged cf-gateway"
check "BE MRs: FE only → none" "$(jq -r --arg k dev-3873 "$BE_MRS_JQ" <<<$MRS)" ""
ENVS='[{"name":"dynamic/dev-3657-pasture"},{"name":"dynamic/dev-36570-x"},{"name":"dynamic/dev-3809"},{"name":"dynamic/dev-3830-catalogue"}]'
check "env pick: epic prefix" "$(jq -r --args "$ENV_PICK_JQ" dev-3657 <<<$ENVS | paste -sd, -)" "dev-3657-pasture"
check "env pick: epic + dep" "$(jq -r --args "$ENV_PICK_JQ" dev-3657 dev-3809 <<<$ENVS | paste -sd, -)" "dev-3657-pasture,dev-3809"
check "env pick: none" "$(jq -r --args "$ENV_PICK_JQ" dev-4000 <<<$ENVS)" ""

source <(sed -n '/^env_push() {/,/^}/p;/^env_fe_fresh() {/,/^}/p' $W)
REPOS=$T/repos LOG_DIR=$T/state STATE=up
mkdir -p $LOG_DIR
FW=$REPOS/cf-farm-frontend/.worktrees/DEV-1-fe
git init -q $FW && git -C $FW -c user.name=t -c user.email=t@t commit -q --allow-empty -m a
HD=$(git -C $FW rev-parse HEAD)
ticket_env() { echo envx }
env_state() { echo $STATE }
info() { : }
env_push DEV-1 "DEV-1-fe $HD"
check "env push: FE head unchanged → nothing" "$(cat $LOG_DIR/DEV-1.env-redeploy 2>/dev/null)" ""
STATE=gone env_push DEV-1 "DEV-1-fe old"
check "env push: env not up → nothing" "$(cat $LOG_DIR/DEV-1.env-redeploy 2>/dev/null)" ""
env_push DEV-1 "DEV-1-fe old"
check "env push: FE moved → redeploy line + head" "$(cat $LOG_DIR/DEV-1.env-redeploy)" "RUN redeploy envx FRONTEND=true FE_BRANCH=DEV-1-fe"$'\n'$HD
rm $LOG_DIR/DEV-1.env-redeploy
env_push DEV-1 "DEV-1-fe old"
check "env push: same head twice → once" "$(cat $LOG_DIR/DEV-1.env-redeploy 2>/dev/null)" ""

env_last_pipe() { echo $LAST }
LAST=77
check "env fresh: no marker → deploy" "$(env_fe_fresh DEV-1 envx; echo $?)" "1"
echo "77 $HD" >$LOG_DIR/DEV-1.env-fe
check "env fresh: our pipeline, same head → skip deploy" "$(env_fe_fresh DEV-1 envx; echo $?)" "0"
LAST=78
check "env fresh: other deploy since → deploy" "$(env_fe_fresh DEV-1 envx; echo $?)" "1"
LAST=77
git -C $FW -c user.name=t -c user.email=t@t commit -q --allow-empty -m b
check "env fresh: head moved → deploy" "$(env_fe_fresh DEV-1 envx; echo $?)" "1"
LAST=""
echo "77 $(git -C $FW rev-parse HEAD)" >$LOG_DIR/DEV-1.env-fe
check "env fresh: lookup failed → deploy" "$(env_fe_fresh DEV-1 envx; echo $?)" "1"
source <(sed -n '/^unreview() {/,/^}/p;/^unverify() {/,/^}/p' $W)
limit_bail() { : }
jira() { print -r -- "$3" >>$T/jira }
UNVERIFIED=$T/unv; mkdir -p $UNVERIFIED; touch $UNVERIFIED/DEV-1.md
unreview DEV-1
check "unreview: only review + quiz labels" "$(jq -r '.update.labels[].remove' $T/jira | sort | paste -sd, -)" "ai-quizzed,ai-reviewed"
rm $T/jira
unverify DEV-1
check "unverify: all five labels" "$(jq -r '.update.labels[].remove' $T/jira | sort | paste -sd, -)" "ai-quizzed,ai-reviewed,ai-verified,ai-verify-failed,ai-verify-retry"
check "unverify: file removed" "$(test -e $UNVERIFIED/DEV-1.md; echo $?)" "1"
exit $fails
