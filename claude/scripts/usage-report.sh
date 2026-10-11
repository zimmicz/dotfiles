#!/usr/bin/env bash
set -uo pipefail
export TZ=Europe/Prague
export LC_NUMERIC=C
export LC_COLLATE=C

SELF="$(readlink -f "$0")"
SCRIPT_DIR="$(dirname "$SELF")"
CLAUDE_DIR="$HOME/.claude"
PROJECTS_DIR="$CLAUDE_DIR/projects"
USAGE_DIR="$CLAUDE_DIR/usage"
PRICES="${USAGE_PRICES:-$SCRIPT_DIR/model-prices.tsv}"
LCTX_EVENTS="${LCTX_EVENTS:-$HOME/.local/state/lean-ctx/events.jsonl}"
SESS_LEDGER="$USAGE_DIR/sessions.tsv"
DAY_LEDGER="$USAGE_DIR/daily.tsv"

CW5M=1.25
CW1H=2.0
CR=0.1

FORMAT=md
WRITE_LEDGER=1
SINCE=""
UNTIL=""

usage() {
  cat <<EOF
usage: $(basename "$0") [--date YYYY-MM-DD | --since YYYY-MM-DD | --week | --month] [--format md|slack] [--no-ledger]

  --date D     single day (default: yesterday)
  --since D    D .. today
  --week       last 7 days including today
  --month      last 30 days including today
  --format     md (default, also written to $USAGE_DIR/report-<window>.md), slack (plain text)
               or html (30-day dashboard from the ledgers -> $USAGE_DIR/index.html)
  --no-ledger  do not update $USAGE_DIR/*.tsv
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --date) SINCE="$2"; UNTIL="$2"; shift 2 ;;
    --since) SINCE="$2"; UNTIL="$(date +%F)"; shift 2 ;;
    --week) SINCE="$(date -d '6 days ago' +%F)"; UNTIL="$(date +%F)"; shift ;;
    --month) SINCE="$(date -d '29 days ago' +%F)"; UNTIL="$(date +%F)"; shift ;;
    --today) SINCE="$(date +%F)"; UNTIL="$SINCE"; shift ;;
    --format) FORMAT="$2"; shift 2 ;;
    --no-ledger) WRITE_LEDGER=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown arg: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[ -n "$SINCE" ] || { SINCE="$(date -d yesterday +%F)"; UNTIL="$SINCE"; }
case "$FORMAT" in md|slack|html) ;; *) echo "bad --format: $FORMAT" >&2; exit 2 ;; esac

date -d "$SINCE" +%F >/dev/null 2>&1 || { echo "bad date: $SINCE" >&2; exit 2; }
date -d "$UNTIL" +%F >/dev/null 2>&1 || { echo "bad date: $UNTIL" >&2; exit 2; }

WSTART=$(date -d "$SINCE 00:00:00" +%s)
WEND=$(date -d "$UNTIL 00:00:00 + 1 day" +%s)
WINDOW="$SINCE"
[ "$SINCE" = "$UNTIL" ] || WINDOW="$SINCE..$UNTIL"
D7="$(date -d "$UNTIL -6 days" +%F)"
D30="$(date -d "$UNTIL -29 days" +%F)"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$USAGE_DIR"

JQ_PROG='
(try fromjson catch empty) as $d
| select($d.timestamp != null)
| ($d.timestamp[0:19] + "Z" | fromdateiso8601) as $t
| select($t >= $ws and $t < $we)
| ($t | strflocaltime("%Y-%m-%d")) as $day
| (if ($d.type == "assistant") and ($d.requestId != null) and ($d.message.usage != null)
     and (($d.message.model // "") | startswith("claude"))
  then ["U", $d.requestId, $day, ($d.sessionId // "unknown"), $d.message.model,
        ($d.message.usage.input_tokens // 0),
        ($d.message.usage.cache_creation.ephemeral_5m_input_tokens
          // $d.message.usage.cache_creation_input_tokens // 0),
        ($d.message.usage.cache_creation.ephemeral_1h_input_tokens // 0),
        ($d.message.usage.cache_read_input_tokens // 0),
        ($d.message.usage.output_tokens // 0),
        ($d.cwd // ""), ($d.gitBranch // ""),
        ($d.message.usage.output_tokens_details.thinking_tokens // 0),
        ($d.agentId // "")] | @tsv
  elif ($d.type == "user") and ($d.isCompactSummary == true)
  then ["C", ($d.sessionId // "unknown"), $day] | @tsv
  elif ($d.type == "user") and ($d.isSidechain != true) and ($d.toolUseResult == null)
       and (($d.message.content | type) == "string")
  then ["L", ($d.sessionId // "unknown"), $day, ($t | tostring),
        ($d.cwd // ""), ($d.gitBranch // ""),
        ($d.message.content | gsub("[\n\r\t]"; " ") | .[0:80])] | @tsv
  elif ($d.type == "user") and ($d.isSidechain != true) and (($d.message.content | type) == "array")
  then ($d.message.content[] | select(.type == "tool_result")
        | ["R", ($d.sessionId // "unknown"), $day, .tool_use_id, (.content | tostring | length)] | @tsv)
  else empty end),
  (if ($d.type == "assistant") and ($d.isSidechain != true)
   then ($d.message.content[]? | select(.type == "tool_use") | ["N", .id, .name] | @tsv)
   else empty end)
'

: > "$TMP/raw.tsv"
if [ -d "$PROJECTS_DIR" ]; then
  find "$PROJECTS_DIR" -name '*.jsonl' -newermt "@$((WSTART - 86400))" -print0 2>/dev/null \
    | xargs -0 -r -n 40 jq -rR --argjson ws "$WSTART" --argjson we "$WEND" "$JQ_PROG" \
      >> "$TMP/raw.tsv" 2>/dev/null
fi

grep '^U' "$TMP/raw.tsv" 2>/dev/null | sort -t$'\t' -u -k2,2 > "$TMP/u.tsv"
grep '^L' "$TMP/raw.tsv" 2>/dev/null | sort -t$'\t' -k2,2 -k4,4n > "$TMP/l.tsv"
grep '^N' "$TMP/raw.tsv" 2>/dev/null | sort -u > "$TMP/n.tsv"
grep '^R' "$TMP/raw.tsv" 2>/dev/null > "$TMP/r.tsv"
grep '^C' "$TMP/raw.tsv" 2>/dev/null > "$TMP/c.tsv"

: > "$TMP/lctx.tsv"
if [ -f "$LCTX_EVENTS" ]; then
  jq -rR --arg s "$SINCE" --arg u "$UNTIL" '
    (try fromjson catch empty) as $d
    | select($d.kind.type == "ToolCall")
    | ($d.timestamp[0:10]) as $day
    | select($day >= $s and $day <= $u)
    | [$day, ($d.kind.tokens_original // 0), ($d.kind.tokens_saved // 0)] | @tsv
  ' "$LCTX_EVENTS" 2>/dev/null \
    | awk -F'\t' -v OFS='\t' '{o[$1]+=$2; s[$1]+=$3} END{for (d in o) print d, o[d], s[d]}' \
    | sort > "$TMP/lctx.tsv"
fi

cat > "$TMP/agg.awk" <<'AWK'
function ratekey(m,   i, p, best, bl) {
  best = ""; bl = -1
  for (i = 1; i <= np; i++) {
    p = pfx[i]
    if (index(m, p) == 1 && length(p) > bl) { best = p; bl = length(p) }
  }
  if (best == "") { best = "DEFAULT"; unknown[m] = 1 }
  return best
}
function shortmodel(m,   s) {
  s = m
  sub(/^claude-/, "", s)
  sub(/-[0-9]{8}$/, "", s)
  return s
}
function mix(prefix, cost,   i, n, arr, out, k, best, bl, bk, used) {
  n = 0
  for (k in mcost) {
    if (index(k, prefix SUBSEP) != 1) continue
    n++
    names[n] = substr(k, length(prefix SUBSEP) + 1)
    vals[n] = mcost[k]
  }
  out = ""
  for (i = 1; i <= n; i++) {
    bl = -1; bk = 0
    for (k = 1; k <= n; k++) if (!(k in used) && vals[k] > bl) { bl = vals[k]; bk = k }
    if (!bk) break
    used[bk] = 1
    pct = (cost > 0) ? (vals[bk] / cost) * 100 : 0
    if (n > 1 && pct < 0.5) continue
    out = out (out == "" ? "" : "/") names[bk]
    if (n > 1 && cost > 0) out = out sprintf(" %d%%", pct + 0.5)
  }
  delete names; delete vals; delete used
  return out
}
function ht(n) {
  if (n >= 1000000) return sprintf("%.1fM", n / 1000000)
  if (n >= 1000) return sprintf("%.0fk", n / 1000)
  return sprintf("%d", n)
}
function tools(key,   k, n, i, bl, bk, out) {
  n = 0
  for (k in tt) {
    if (index(k, key SUBSEP) != 1) continue
    n++; tn[n] = substr(k, length(key SUBSEP) + 1); tv[n] = tt[k]
  }
  out = ""
  for (i = 1; i <= n; i++) {
    bl = -1; bk = 0
    for (k = 1; k <= n; k++) if (!(k in tu) && tv[k] > bl) { bl = tv[k]; bk = k }
    if (!bk) break
    tu[bk] = 1
    out = out (out == "" ? "" : ";") tn[bk] ":" int(tv[bk])
  }
  delete tn; delete tv; delete tu
  return (out == "" ? "-" : out)
}
BEGIN { FS = OFS = "\t" }
FILENAME == NFILE { tname[$2] = $3; next }
FILENAME == RFILE {
  if (!($4 in tname)) next
  k = $3 SUBSEP $2
  tt[k SUBSEP tname[$4]] += $5 / 4; ttot[k] += $5 / 4
  next
}
FILENAME == CFILE { comp[$3 SUBSEP $2]++; next }
FILENAME == PRICES {
  if ($0 ~ /^#/ || NF < 3) next
  if ($1 == "DEFAULT") { pin["DEFAULT"] = $2; pout["DEFAULT"] = $3; next }
  pfx[++np] = $1; pin[$1] = $2; pout[$1] = $3
  next
}
FILENAME == LCTX { lo[$1] = $2; ls[$1] = $3; next }
FILENAME == LFILE {
  sid = $2; txt = $7
  bad = (substr(txt, 1, 1) == "<" || txt == "")
  if (!(sid in lab) || (labbad[sid] && !bad)) { lab[sid] = txt; labbad[sid] = bad }
  next
}
{
  day = $3; sid = $4; model = $5
  rk = ratekey(model); ri = pin[rk] + 0; ro = pout[rk] + 0
  incost = ($6 * ri + $7 * ri * CW5M + $8 * ri * CW1H + $9 * ri * CR) / 1000000
  cost = incost + ($10 * ro) / 1000000
  key = day SUBSEP sid
  if (!(key in sreq)) { sday[key] = day; ssid[key] = sid; keys[++nk] = key }
  sreq[key]++; s_in[key] += $6; s5[key] += $7; s1[key] += $8; scr[key] += $9; sout[key] += $10
  scost[key] += cost
  if ($11 != "") sproj[key] = $11
  if ($12 != "") sbr[key] = $12
  ctx = $6 + $7 + $8 + $9
  sctx[key] += ctx; if (ctx > speak[key]) speak[key] = ctx
  sth[key] += $13
  if ($14 != "" && !(key SUBSEP $14 in agents)) { agents[key SUBSEP $14] = 1; nsub[key]++ }
  mcost["S" SUBSEP key SUBSEP shortmodel(model)] += cost
  mcost["D" SUBSEP day SUBSEP shortmodel(model)] += cost
  if (!(day in dreq)) days[++nd] = day
  dreq[day]++; din[day] += $6; d5[day] += $7; d1[day] += $8; dcr[day] += $9; dout[day] += $10
  dcost[day] += cost; dincost[day] += incost; dsess[day SUBSEP sid] = 1
  if (rk == "DEFAULT") dunknown[day] = 1
}
END {
  for (i = 1; i <= nk; i++) {
    key = keys[i]
    proj = sproj[key]
    sub(/\/\.worktrees\/.*$/, "", proj)
    sub(/.*\//, "", proj)
    br = sbr[key]; if (br == "") br = "-"
    l = lab[ssid[key]]; if (l == "") l = "-"
    print sday[key], ssid[key], (proj == "" ? "-" : proj), br, mix("S" SUBSEP key, scost[key]),
          sreq[key], s_in[key], s5[key], s1[key], scr[key], sout[key],
          sprintf("%.6f", scost[key]), l,
          int(sctx[key] / sreq[key]), speak[key], sth[key], nsub[key] + 0, comp[key] + 0,
          int(ttot[key]), tools(key) > SESSOUT
  }
  for (i = 1; i <= nd; i++) {
    day = days[i]
    ns = 0
    for (k in dsess) { split(k, a, SUBSEP); if (a[1] == day) ns++ }
    rate = (din[day] + d5[day] + d1[day] + dcr[day]) > 0 \
      ? dincost[day] / (din[day] + d5[day] + d1[day] + dcr[day]) : 0
    print day, ns, dreq[day], din[day], d5[day], d1[day], dcr[day], dout[day],
          sprintf("%.6f", dcost[day]), mix("D" SUBSEP day, dcost[day]),
          (lo[day] + 0), (ls[day] + 0), sprintf("%.6f", (ls[day] + 0) * rate),
          (dunknown[day] ? "unknown-model-priced-as-opus" : "-") > DAYOUT
  }
}
AWK

: > "$TMP/sessions.new"
: > "$TMP/days.new"
awk -f "$TMP/agg.awk" \
  -v PRICES="$PRICES" -v LFILE="$TMP/l.tsv" -v LCTX="$TMP/lctx.tsv" \
  -v CW5M="$CW5M" -v CW1H="$CW1H" -v CR="$CR" \
  -v SESSOUT="$TMP/sessions.new" -v DAYOUT="$TMP/days.new" \
  -v NFILE="$TMP/n.tsv" -v RFILE="$TMP/r.tsv" -v CFILE="$TMP/c.tsv" \
  "$PRICES" "$TMP/lctx.tsv" "$TMP/l.tsv" "$TMP/n.tsv" "$TMP/r.tsv" "$TMP/c.tsv" "$TMP/u.tsv"

sort -t$'\t' -k1,1 -k12,12gr "$TMP/sessions.new" -o "$TMP/sessions.new"
sort -t$'\t' -k1,1 "$TMP/days.new" -o "$TMP/days.new"

merge_ledger() {
  local ledger="$1" fresh="$2"
  touch "$ledger"
  awk -F'\t' -v s="$SINCE" -v u="$UNTIL" '$1 < s || $1 > u' "$ledger" > "$TMP/keep"
  cat "$TMP/keep" "$fresh" | sort -t$'\t' -k1,1 -s > "$TMP/merged"
  mv "$TMP/merged" "$ledger"
}

if [ "$WRITE_LEDGER" = 1 ]; then
  merge_ledger "$SESS_LEDGER" "$TMP/sessions.new"
  merge_ledger "$DAY_LEDGER" "$TMP/days.new"
fi

touch "$DAY_LEDGER"
ROLL="$(awk -F'\t' -v d7="$D7" -v d30="$D30" -v u="$UNTIL" -v OFS='\t' '
  $1 <= u && $1 >= d30 { c30 += $9; s30 += $13; if ($1 >= d7) { c7 += $9; s7 += $13 } }
  END { printf "%.6f\t%.6f\t%.6f\t%.6f\n", c7 + 0, c30 + 0, s7 + 0, s30 + 0 }
' "$DAY_LEDGER")"

cat > "$TMP/render.awk" <<'AWK'
function ht(n) {
  if (n >= 1000000000) return sprintf("%.2fB", n / 1000000000)
  if (n >= 1000000) return sprintf("%.2fM", n / 1000000)
  if (n >= 1000) return sprintf("%.0fk", n / 1000)
  return sprintf("%d", n)
}
function usd(c) { return sprintf("$%.2f", c) }
function clean(s) {
  gsub(/https?:\/\/[^ ]+/, "<link>", s)
  gsub(/\/home\/[^ ]+/, "<path>", s)
  gsub(/\|/, "/", s)
  gsub(/  +/, " ", s)
  sub(/^ +/, "", s)
  sub(/ +$/, "", s)
  if (s == "") s = "-"
  return s
}
BEGIN {
  FS = OFS = "\t"
  split(ROLL, r, "\t")
}
FILENAME == DAYS {
  nd++
  sessions += $2; req += $3; in_t += $4; c5 += $5; c1 += $6; cr += $7; out += $8
  cost += $9; lo += $11; ls += $12; lsusd += $13
  if ($10 != "") mixday[$1] = $10
  if ($14 != "-") warn = $14
  next
}
{ ns++; srow[ns] = $0 }
END {
  inside = in_t + c5 + c1 + cr
  crshare = inside > 0 ? (cr / inside) * 100 : 0
  mixall = ""
  for (d in mixday) { mixall = mixday[d]; break }
  if (nd > 1) mixall = ""
  if (md) {
    print "# Token usage " WINDOW
    print ""
    printf "**%s notional**%s · %d sessions · %d requests\n\n", usd(cost),
      (mixall == "" ? "" : " (" mixall ")"), sessions, req
    printf "- input side %s (cache read %.0f%%, cache write 5m %s / 1h %s, fresh %s)\n",
      ht(inside), crshare, ht(c5), ht(c1), ht(in_t)
    printf "- output %s\n", ht(out)
    printf "- rolling: 7d %s · 30d %s\n", usd(r[1]), usd(r[2])
    if (lo > 0)
      printf "- lean-ctx: %s of %s tool tokens compressed away (~%s saved, lower bound; 7d %s · 30d %s)\n",
        ht(ls), ht(lo), usd(lsusd), usd(r[3]), usd(r[4])
    if (warn != "" && warn != "-") printf "- note: %s\n", warn
    print ""
    print "| cost | day | project | branch | models | requests | in | out | session |"
    print "| --- | --- | --- | --- | --- | --- | --- | --- | --- |"
    for (i = 1; i <= ns; i++) {
      split(srow[i], f, "\t")
      printf "| %s | %s | %s | %s | %s | %d | %s | %s | %s |\n",
        usd(f[12]), f[1], f[3], f[4], f[5], f[6],
        ht(f[7] + f[8] + f[9] + f[10]), ht(f[11]), f[13]
    }
    print ""
    print "notional = list API prices, not plan billing. lean-ctx savings come from lean-ctx's own"
    print "compression accounting, day-global (not per session), incl. subagent tool calls; they"
    print "are a lower bound - a token kept out of context is also never re-sent as a cache read"
    print "on every later request of that session."
    print "Sessions spanning midnight are split by message timestamp (Europe/Prague)."
  } else {
    printf "Token usage %s — *%s notional*%s\n", WINDOW, usd(cost),
      (mixall == "" ? "" : " (" mixall ")")
    printf "in %s (cache read %.0f%%) · out %s · %d requests · %d sessions · 7d %s · 30d %s\n\n",
      ht(inside), crshare, ht(out), req, sessions, usd(r[1]), usd(r[2])
    if (nd > 1) {
      print "| cost | day | project | branch | session |"
      print "| --- | --- | --- | --- | --- |"
    } else {
      print "| cost | project | branch | session |"
      print "| --- | --- | --- | --- |"
    }
    for (i = 1; i <= ns; i++) {
      split(srow[i], f, "\t")
      if (nd > 1)
        printf "| %s | %s | %s | %s | %s |\n", usd(f[12]), f[1], f[3], f[4], clean(f[13])
      else
        printf "| %s | %s | %s | %s |\n", usd(f[12]), f[3], f[4], clean(f[13])
    }
    print ""
    if (lo > 0)
      printf "lean-ctx: %s of %s tool tokens compressed away (~%s saved, lower bound)\n",
        ht(ls), ht(lo), usd(lsusd)
    if (warn != "" && warn != "-") printf "note: %s\n", warn
    print "notional = list API prices, not plan billing"
  }
}
AWK

if [ "$FORMAT" = html ]; then
  touch "$SESS_LEDGER"
  {
    sed -e 's|@@WINDOW@@|'"$D30..$UNTIL"'|' -e '/@@SESSIONS@@/q' "$SCRIPT_DIR/usage-dashboard.html" | sed '$d'
    awk -F'\t' -v s="$D30" -v u="$UNTIL" '$1 >= s && $1 <= u' "$SESS_LEDGER" | sed 's|</script|<\\/script|g'
    sed -n '/@@SESSIONS@@/,/@@DAYS@@/p' "$SCRIPT_DIR/usage-dashboard.html" | sed '1d;$d'
    awk -F'\t' -v s="$D30" -v u="$UNTIL" '$1 >= s && $1 <= u' "$DAY_LEDGER"
    sed '1,/@@DAYS@@/d' "$SCRIPT_DIR/usage-dashboard.html"
  } > "$USAGE_DIR/index.html"
  echo "$USAGE_DIR/index.html"
  exit 0
fi

if [ ! -s "$TMP/days.new" ]; then
  if [ "$FORMAT" = slack ]; then
    echo "Token usage $WINDOW - no sessions recorded"
  else
    printf '# Token usage %s\n\nNo sessions recorded.\n' "$WINDOW"
  fi
  exit 0
fi

render() {
  awk -f "$TMP/render.awk" -v DAYS="$TMP/days.new" -v WINDOW="$WINDOW" \
    -v ROLL="$ROLL" -v md="$1" "$TMP/days.new" "$TMP/sessions.new"
}

if [ "$FORMAT" = slack ]; then
  render 0
else
  render 1 | tee "$USAGE_DIR/report-$WINDOW.md"
fi
