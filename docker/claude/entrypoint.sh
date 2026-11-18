#!/usr/bin/env bash
set -euo pipefail

DOMAINS="
api.anthropic.com claude.ai platform.claude.com console.anthropic.com mcp-proxy.anthropic.com
registry.npmjs.org
gitlab.com registry.gitlab.com
cleverfarm.atlassian.net api.atlassian.com auth.atlassian.com mcp.atlassian.com
sentry.io mcp.sentry.dev
nightly.dev.cleverfarm.cz staging.dev.cleverfarm.cz
vault.dev.cleverfarm.internal loki.dev.cleverfarm.internal prometheus.dev.cleverfarm.internal grafana.dev.cleverfarm.internal
${DCLAUDE_EXTRA_DOMAINS:-}
"

NS=$(awk '/^nameserver/ && $2 ~ /^[0-9.]+$/ {print $2}' /etc/resolv.conf)

resolve() {
  local ns r
  for ns in $NS; do
    r=$(dig +short +time=2 +tries=1 A "$1" "@$ns" | grep -E '^[0-9.]+$' || true)
    [ -n "$r" ] && { echo "$r"; return; }
  done
  return 0
}

ipset create allowed hash:net
for d in $DOMAINS; do
  ips=$(resolve "$d")
  [ -n "$ips" ] || { echo "dclaude: cannot resolve $d, skipped" >&2; continue; }
  for ip in $ips; do ipset add -exist allowed "$ip"; done
  getent hosts "$d" >/dev/null || echo "${ips%%$'\n'*} $d" >>/etc/hosts
done

cf=$(curl -fsS -m 10 https://ip-ranges.amazonaws.com/ip-ranges.json | jq -r '.prefixes[] | select(.service=="CLOUDFRONT") | .ip_prefix' || true)
[ -n "$cf" ] || echo "dclaude: CloudFront ranges not loaded, previews unreachable" >&2
for c in $cf ${DCLAUDE_VPN_CIDRS:-}; do ipset add -exist allowed "$c"; done

iptables -A OUTPUT -o lo -j ACCEPT
iptables -A INPUT -i lo -j ACCEPT
for ns in $NS; do
  iptables -A OUTPUT -p udp -d "$ns" --dport 53 -j ACCEPT
  iptables -A OUTPUT -p tcp -d "$ns" --dport 53 -j ACCEPT
done
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m set --match-set allowed dst -j ACCEPT
iptables -A OUTPUT -j REJECT --reject-with icmp-admin-prohibited
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT DROP
ip6tables -P INPUT DROP 2>/dev/null || true
ip6tables -P OUTPUT DROP 2>/dev/null || true

if curl -fsS -m 3 -o /dev/null https://example.com 2>/dev/null; then
  echo "dclaude: firewall check failed, example.com reachable" >&2
  exit 1
fi

if [ -n "${GITLAB_TOKEN:-}" ]; then
  export GIT_CONFIG_COUNT=1
  export GIT_CONFIG_KEY_0="url.https://oauth2:${GITLAB_TOKEN}@gitlab.com/.insteadOf"
  export GIT_CONFIG_VALUE_0="git@gitlab.com:"
fi

exec setpriv --reuid="$DCLAUDE_UID" --regid="$DCLAUDE_UID" --init-groups \
  --inh-caps=-all --bounding-set=-all --no-new-privs "$@"
