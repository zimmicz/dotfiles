- In all interactions and commit messages, be extremely concise and sacrifice grammar for the sake of concision.
- Use caveman mode for all responses.
- When drafting any message I will send to others (Slack, Jira comments, MR descriptions, emails), write in MY voice per `~/.claude/my-voice.md`. This OVERRIDES caveman mode (caveman = how you talk to me, not how I sound to others). Show me the draft to review; don't send unless I say so. Anything YOU post yourself (Jira comment, MR description/note, Slack msg to others) starts with the 🤖 prefix line from my-voice.md "Posted by Claude".
- Do not use memory as a way of storing knowledge. Use local files instead. If there is no file yet, create it.
- When writing code, do not add any comments. Just write the code.

@LEARNED.md

## Tools

- Use `glab` CLI (not `gh`) for all GitLab operations (MRs, issues, pipelines).
- Screenshots/images you take to look at yourself: ≤1280px wide, jpeg. Never Read the same image twice; reuse what you saw.
- `npm run` is aliased to `nr` for all NPM operations. When you try to run a command with `npm run` it returns non zero exit code as a hint to run `nr` instead.

## Plans

- At the end of each plan, give me a list of unresolved tasks questions to answer, if any. Make the questions extremely concise. Sacrificec grammar for the sake of concision.

<!-- lean-ctx -->
<!-- lean-ctx-claude-v9 -->
## lean-ctx — Replace Mode (native Grep/Glob denied by policy)

Native Grep/Glob are denied by policy. Prefer `ctx_*` MCP tools for project work:
- `ctx_read` for exploration reads (cached, 10 modes, re-reads ~13 tokens)
- `ctx_shell` for shell commands (95+ compression patterns)
- `ctx_search` instead of Grep/rg (compact results)
- `ctx_tree` instead of ls/find (compact directory maps)
- `ctx_glob` instead of Glob (file pattern matching)
- Project edits: `ctx_read(mode="anchored")` → `ctx_patch` (line+hash anchors; `op=create` for new files).

Native `Read` is reserved for the edit gate (read-before-write) only.
For exploration, orientation, and code understanding: ALWAYS use `ctx_read`.
Claude auto memory (`~/.claude/projects/<slug>/memory/` — MEMORY.md and topic
files) uses native Read/Edit internally; do NOT call MCP `resources/read` with
file:// URIs (lean-ctx resources are `lean-ctx://context/*` only). Native Delete is fine.

Read modes: anchored (edit), full (verbatim), map (overview), signatures (API), diff (post-edit), lines:N-M (range), auto.
Details live in the `lean-ctx` skill (loads on demand — keep this file lean).
<!-- /lean-ctx -->

<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them, including dynamic-dispatch hops grep can't follow. Name a file or symbol in the query to read its current line-numbered source. If it's listed but deferred, load it by name via tool search.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` prints the same output.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
<!-- CODEGRAPH_END -->

In CodeGraph-indexed repos, prefer `codegraph_explore` over `ctx_callgraph` for callers/callees/impact — it is pre-indexed, follows dynamic dispatch, and returns source. `ctx_callgraph` only as fallback where no `.codegraph/` exists.
