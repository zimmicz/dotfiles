#!/bin/zsh
set -u
W=${0:A:h:h}/bin/factory-worker
source <(sed -n '/^DATA_RE=/,/^}/p' $W)
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
exit $fails
