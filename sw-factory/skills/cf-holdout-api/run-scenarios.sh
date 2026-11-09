#!/usr/bin/env bash
set -u
[[ $# -eq 3 ]] || { echo "usage: run-scenarios.sh <scenario-file> <base-env> <mr-env>" >&2; exit 2; }
FILE=$1 BASE=$2 MR=$3
HERE=$(cd "$(dirname "$0")" && pwd)
DOMAIN=dev.cleverfarm.cz
WORK=$(mktemp -d "${TMPDIR:-$HOME}/cf-holdout-api.XXXXXX")
trap 'rm -rf "$WORK"' EXIT
TOKEN=$(node "$HERE/token.mjs") || { echo "VERDICT: blocked"; echo "token: login failed"; exit 3; }

declare -A VAR CODE SIG NONEMPTY
NAME= CHART= GIVEN_NAME= GIVEN_VAR= HEADERS= EXPECT= WANT_NONEMPTY= IGNORE=
STEPS=()
RESULTS=()
ANY_FAIL=0 ANY_BLOCKED=0

url() { echo "https://$1.$2.$DOMAIN$3"; }

farm_id() {
  curl -s -H "Authorization: Bearer $TOKEN" "$(url agroevidence "$1" /api/farms)" \
    | jq -r --arg n "$2" '(if type=="array" then . else (.data // .items // []) end) | map(select(.name==$n)) | .[0].id // empty' 2>/dev/null
}

subst() {
  local s=$1 env=$2 k
  for k in "${!VAR[@]}"; do
    [[ $k == "$env:"* ]] && s=${s//\$${k#"$env":}/${VAR[$k]}}
  done
  echo "$s"
}

norm() {
  local f=$1 ign
  ign=$(jq -Rc 'split(",") | map(gsub("^\\s+|\\s+$";"")) | map(select(length>0))' <<<"$IGNORE")
  if jq -e . "$f" >/dev/null 2>&1; then
    jq -S --argjson ign "$ign" 'walk(if type=="object" then with_entries(select(.key as $k | ($ign | index($k)) | not)) else . end)' "$f"
  else
    cat "$f"
  fi
}

run_env() {
  local env=$1 i=0 line method path body cap capvar capexpr hdr h val id
  if [[ -n $GIVEN_NAME ]]; then
    id=$(farm_id "$env" "$GIVEN_NAME")
    if [[ -z $id ]]; then CODE["$env:given"]=missing; return; fi
    VAR["$env:$GIVEN_VAR"]=$id
  fi
  for line in "${STEPS[@]}"; do
    i=$((i + 1))
    cap=
    if [[ $line =~ ^(.*)\ -\>\ \$([A-Za-z_]+)\ =\ (.*)$ ]]; then
      line=${BASH_REMATCH[1]} capvar=${BASH_REMATCH[2]} capexpr=${BASH_REMATCH[3]} cap=1
    fi
    [[ $line =~ ^([A-Z]+)\ ([^ ]+)(\ \ (.*))?$ ]] || { CODE["$env:$i"]=unparsed; continue; }
    method=${BASH_REMATCH[1]} path=$(subst "${BASH_REMATCH[2]}" "$env") body=$(subst "${BASH_REMATCH[4]:-}" "$env")
    local args=(-s -D "$WORK/hdr" -o "$WORK/body" -w '%{http_code}' -X "$method" -H "Authorization: Bearer $TOKEN")
    [[ -n $body ]] && args+=(-H 'Content-Type: application/json' --data "$body")
    CODE["$env:$i"]=$(curl "${args[@]}" "$(url "$CHART" "$env" "$path")")
    hdr=
    for h in $HEADERS; do
      val=$(grep -i "^$h:" "$WORK/hdr" | tail -1 | tr -d '\r' | sed 's/^[^:]*: *//')
      hdr+="$h=$val;"
    done
    SIG["$env:$i"]="${CODE["$env:$i"]}|$hdr
$(norm "$WORK/body")"
    NONEMPTY["$env:$i"]=$(jq -r '(if type=="array" then . else (.data // .items // []) end) | length > 0' "$WORK/body" 2>/dev/null || echo false)
    [[ -n $cap ]] && VAR["$env:$capvar"]=$(jq -r "$capexpr" "$WORK/body" 2>/dev/null)
  done
}

finish_scenario() {
  [[ -z $NAME ]] && return
  local status=pass note= i n=${#STEPS[@]} d
  VAR=() CODE=() SIG=() NONEMPTY=()
  run_env "$BASE"
  run_env "$MR"
  if [[ ${CODE["$BASE:given"]:-} == missing || ${CODE["$MR:given"]:-} == missing ]]; then
    status=blocked note="Given farm \"$GIVEN_NAME\" not found on ${CODE["$BASE:given"]:+base }${CODE["$MR:given"]:+mr}"
  else
    for ((i = 1; i <= n; i++)); do
      if [[ ${SIG["$BASE:$i"]} != "${SIG["$MR:$i"]}" ]]; then
        d=$(diff <(echo "${SIG["$BASE:$i"]}") <(echo "${SIG["$MR:$i"]}") | head -20)
        status=fail note="step $i differs (< base, > mr): $d"
        break
      fi
    done
    if [[ $status == pass ]]; then
      for ((i = 1; i <= n; i++)); do
        if [[ -n $EXPECT && ( ${CODE["$BASE:$i"]} != "$EXPECT" || ${CODE["$MR:$i"]} != "$EXPECT" ) ]]; then
          status=blocked note="step $i expected $EXPECT, got base ${CODE["$BASE:$i"]} / mr ${CODE["$MR:$i"]}"
          break
        fi
        if [[ $WANT_NONEMPTY == true && ( ${NONEMPTY["$BASE:$i"]} != true || ${NONEMPTY["$MR:$i"]} != true ) ]]; then
          status=blocked note="step $i expected a non-empty list, got base ${NONEMPTY["$BASE:$i"]} / mr ${NONEMPTY["$MR:$i"]}"
          break
        fi
      done
    fi
    [[ $status == pass ]] && note="$n step(s) identical, statuses $(for ((i = 1; i <= n; i++)); do printf '%s ' "${CODE["$BASE:$i"]}"; done)"
  fi
  [[ $status == fail ]] && ANY_FAIL=1
  [[ $status == blocked ]] && ANY_BLOCKED=1
  RESULTS+=("$NAME: $status — $note")
}

reset_scenario() {
  NAME=$1 CHART= GIVEN_NAME= GIVEN_VAR= HEADERS= EXPECT= WANT_NONEMPTY= IGNORE=
  STEPS=()
}

while IFS= read -r line || [[ -n $line ]]; do
  case $line in
    "## "*) finish_scenario; reset_scenario "${line#\#\# }" ;;
    "Service: "*) CHART=${line#Service: } ;;
    "Given: "*) [[ $line =~ farm\ \"([^\"]+)\".*→\ \$([A-Za-z_]+) ]] && GIVEN_NAME=${BASH_REMATCH[1]} GIVEN_VAR=${BASH_REMATCH[2]} ;;
    "Headers: "*) HEADERS=${line#Headers: } ;;
    "Expect: "*) EXPECT=${line#Expect: } ;;
    "Expect-nonempty: "*) WANT_NONEMPTY=${line#Expect-nonempty: } ;;
    "Ignore: "*) IGNORE=${line#Ignore: } ;;
    [0-9]*". "*) STEPS+=("${line#*. }") ;;
  esac
done <"$FILE"
finish_scenario

if ((ANY_FAIL)); then echo "VERDICT: fail"; elif ((ANY_BLOCKED)); then echo "VERDICT: blocked"; else echo "VERDICT: pass"; fi
printf '%s\n' "${RESULTS[@]}"
exit $((ANY_FAIL ? 1 : ANY_BLOCKED ? 4 : 0))
