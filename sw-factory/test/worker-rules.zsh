#!/bin/zsh
set -u
W=${0:A:h:h}/bin/factory-worker
source <(sed -n '/^DATA_RE=/,/^}/p' $W)
source <(grep '^POSTED_RE=' $W)
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
exit $fails
