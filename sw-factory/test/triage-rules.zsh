#!/bin/zsh
set -u
D=${0:A:h:h}/bin
fails=0
check() { [[ "$2" == "$3" ]] && echo "ok   $1" || { echo "FAIL $1: got '$2' want '$3'"; fails=1 } }

MRS='[
 {"title":"chore(deps): [DEV-3879] bump sentry group","description":"","state":"merged"},
 {"title":"fix: [DEV-3881] Sentry filter keeps our own string messages","description":"spin-off of DEV-3879","state":"opened"},
 {"title":"fix: [DEV-38790] other","description":"","state":"opened"},
 {"title":"Draft: DEV-3879 retry","description":"","state":"closed"}
]'
for F in deps-triage sentry-triage; do
  source <(grep '^MR_STATES_JQ=' $D/$F)
  check "$F: key in title only" "$(jq -r --arg k DEV-3879 "$MR_STATES_JQ" <<<$MRS)" "closed,merged"
  check "$F: mention in description ignored" "$(jq -r --arg k DEV-3881 "$MR_STATES_JQ" <<<$MRS)" "opened"
  check "$F: no MR" "$(jq -r --arg k DEV-1 "$MR_STATES_JQ" <<<$MRS)" ""
  unset MR_STATES_JQ
done
exit $fails
