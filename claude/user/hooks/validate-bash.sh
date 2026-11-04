#!/usr/bin/env bash
set -eo pipefail

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null) || exit 0
[[ -z "$COMMAND" ]] && exit 0

ask() {
  printf '{"decision": "ask", "reason": "%s"}' "$1"
  exit 0
}

# rm -rf /
if printf '%s' "$COMMAND" | grep -qE 'rm\s+(-[a-zA-Z]*r[a-zA-Z]*f|(-[a-zA-Z]*f[a-zA-Z]*r))\s+/(\s|$|\*)'; then
  ask "rm -rf / is blocked for safety"
fi

# git push --force
if printf '%s' "$COMMAND" | grep -qE 'git\s+push\s+.*(-f|--force)'; then
  ask "git push --force is blocked. Use --force-with-lease"
fi

# --no-verify (skips pre-commit hooks)
if printf '%s' "$COMMAND" | grep -qE 'git\s+(commit|push)\s+.*--no-verify'; then
  ask "--no-verify skips hooks. Fix the underlying issue instead"
fi

# git add -A / git add .
if printf '%s' "$COMMAND" | grep -qE 'git\s+add\s+(-A|--all|\.)(\s|$)'; then
  ask "git add -A is blocked. Stage specific files instead"
fi

# sudo
if printf '%s' "$COMMAND" | grep -qE '(^|[;&|]\s*)sudo\s'; then
  ask "sudo is blocked in automated mode"
fi

read -r -d '' GUARD <<'PY' || true
import json, os, re, shlex, sys

d = json.load(sys.stdin)
cmd = (d.get("tool_input") or {}).get("command") or ""
base = d.get("cwd") or os.getcwd()
root = os.path.expanduser("~/work/cleverfarm/repos")
hint = {"cat": "ctx_read", "head": "ctx_read (mode lines:N-M)", "tail": "ctx_read (mode lines:N-M)",
        "sed": "ctx_read (mode lines:N-M)", "awk": "ctx_read / ctx_search", "grep": "ctx_search",
        "egrep": "ctx_search", "rg": "ctx_search", "ls": "ctx_tree", "tree": "ctx_tree", "find": "ctx_glob"}
viewers = {"head", "tail", "grep", "egrep", "rg", "sed", "wc", "awk", "sort", "uniq", "cut", "less", "nl", "column", "tr", "cat"}
valued = {"head": {"-n", "-c"}, "tail": {"-n", "-c"}, "sed": {"-e", "-f"}, "awk": {"-F", "-v", "-f"},
          "grep": {"-e", "-f", "-A", "-B", "-C", "-m", "--max-count"}, "egrep": {"-e", "-f", "-A", "-B", "-C", "-m"},
          "rg": {"-e", "-f", "-A", "-B", "-C", "-m", "-g", "--glob", "-t", "-T"}}

kept, term = [], None
for line in cmd.split("\n"):
    if term:
        if line.strip() == term:
            term = None
        continue
    m = re.search(r"(?<!<)<<(?!<)-?\s*['\"]?(\w+)", line)
    if m:
        term = m.group(1)
        line = line[:m.start()]
    kept.append(line)
cmd = "\n".join(kept)
lex = shlex.shlex(cmd, posix=True, punctuation_chars="();<>|&\n")
lex.whitespace = " \t\r"
lex.whitespace_split = True
try:
    toks = list(lex)
except ValueError:
    sys.exit(0)

cmds, cur, stage = [], [], []
for t in toks:
    if t in ("&&", "||", ";", "\n", "&", "(", ")", ";;"):
        cur.append(stage); cmds.append(cur); cur, stage = [], []
    elif t == "|":
        cur.append(stage); stage = []
    else:
        stage.append(t)
cur.append(stage); cmds.append(cur)


def clean(argv):
    out, redir, skip = [], False, False
    for a in argv:
        if skip:
            skip = False
            continue
        if re.fullmatch(r"[<>&]+", a):
            if out and out[-1].isdigit():
                out.pop()
            redir = redir or ">" in a
            skip = True
            continue
        out.append(a)
    while out and re.match(r"^\w+=", out[0]):
        out.pop(0)
    return out, redir


def positional(name, args):
    pos, skip, has_e = [], False, False
    for a in args:
        if skip:
            skip = False
            continue
        if a in valued.get(name, ()):
            skip = True
            has_e = has_e or a == "-e"
            continue
        if a.startswith("-") and a != "-":
            continue
        pos.append(a)
    return pos, has_e


def inside(p, cwd):
    if p.startswith("$") or p == "-":
        return False
    p = os.path.normpath(os.path.join(cwd, os.path.expanduser(p)))
    if "/node_modules" in p:
        return False
    return p == root or p.startswith(root + os.sep)


cwd = base
for pipeline in cmds:
    stages = [clean(s) for s in pipeline if s]
    if not stages or not stages[0][0]:
        continue
    argv, redir = stages[0]
    name, args = os.path.basename(argv[0]), argv[1:]
    if name == "cd" and args:
        cwd = os.path.normpath(os.path.join(cwd, os.path.expanduser(args[0])))
        continue
    if name not in hint:
        continue
    if any(s[0] and os.path.basename(s[0][0]) not in viewers for s in stages[1:]):
        continue
    pos, has_e = positional(name, args)
    if name == "cat":
        if redir:
            continue
        paths = pos
    elif name in ("head", "tail"):
        paths = [p for p in pos if not re.fullmatch(r"[+-]?\d+", p)]
    elif name == "sed":
        if any(a.startswith("-i") or a == "--in-place" for a in args):
            continue
        paths = pos if has_e else pos[1:]
    elif name == "awk":
        paths = [p for p in pos[1:] if "=" not in p]
    elif name in ("grep", "egrep", "rg"):
        paths = pos if has_e else pos[1:]
        recursive = name == "rg" or any(re.fullmatch(r"-[a-zA-Z]*[rR][a-zA-Z]*|--recursive", a) for a in args)
        if not paths and recursive:
            paths = ["."]
    elif name in ("ls", "tree"):
        paths = pos or ["."]
    else:
        if "-delete" in args:
            continue
        paths = []
        for a in args:
            if a.startswith(("-", "(", "!")):
                break
            paths.append(a)
        paths = paths or ["."]
    if any(inside(p, cwd) for p in paths):
        print(f"lean-ctx guard: native `{name}` on a path inside {root} is blocked. Use {hint[name]} instead.")
        sys.exit(0)
PY

REASON=$(printf '%s' "$INPUT" | python3 -I -c "$GUARD" 2>/dev/null) || true
if [[ -n "$REASON" ]]; then
  printf '%s\n' "$REASON" >&2
  exit 2
fi

exit 0
