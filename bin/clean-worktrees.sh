#!/usr/bin/env bash
set -euo pipefail

FORCE=false
for arg in "$@"; do
  case "$arg" in
    --force) FORCE=true ;;
    -h|--help)
      echo "Usage: clean-worktrees.sh [--force]"
      echo "  Removes git worktrees whose remote branch is gone and that had an MR (never-pushed branches are kept)."
      echo "  Dry-run by default. Pass --force to actually delete."
      exit 0
      ;;
  esac
done

if ! git rev-parse --git-dir &>/dev/null; then
  echo "Not a git repository." >&2
  exit 1
fi

main_worktree=$(git worktree list --porcelain | sed -n '1s/^worktree //p')
current_dir=$(pwd -P)

echo "Fetching and pruning remote refs..."
git fetch --prune --quiet

echo "Pruning stale worktree metadata..."
git worktree prune

remote_branches=$(git ls-remote --heads origin 2>/dev/null | awk '{print $2}' | sed 's|refs/heads/||')

to_remove=()
skipped_unpushed=()
skipped_dirty=()
skipped_current=()

while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  wt_path=$(echo "$line" | awk '{print $1}')
  wt_branch=$(echo "$line" | awk '{print $3}' | sed 's|^\[||;s|\]$||')

  [[ "$wt_path" == "$main_worktree" ]] && continue

  real_wt=$(realpath "$wt_path" 2>/dev/null || echo "$wt_path")
  real_cur=$(realpath "$current_dir" 2>/dev/null || echo "$current_dir")
  if [[ "$real_wt" == "$real_cur" ]]; then
    skipped_current+=("$wt_path ($wt_branch)")
    continue
  fi

  if echo "$remote_branches" | grep -qx "$wt_branch" 2>/dev/null; then
    continue
  fi

  if [[ "$(glab mr list --source-branch "$wt_branch" --all -F json 2>/dev/null | jq 'length' 2>/dev/null)" =~ ^0?$ ]]; then
    skipped_unpushed+=("$wt_path ($wt_branch)")
    continue
  fi

  if [[ -d "$wt_path" ]] && [[ -n "$(git -C "$wt_path" status --porcelain 2>/dev/null)" ]]; then
    skipped_dirty+=("$wt_path ($wt_branch)")
    continue
  fi

  to_remove+=("$wt_path|$wt_branch")
done < <(git worktree list | tail -n +2)

echo ""
if [[ ${#skipped_current[@]} -gt 0 ]]; then
  echo "Skipped (current dir):"
  for s in "${skipped_current[@]}"; do echo "  $s"; done
  echo ""
fi

if [[ ${#skipped_unpushed[@]} -gt 0 ]]; then
  echo "Skipped (no MR, never pushed):"
  for s in "${skipped_unpushed[@]}"; do echo "  $s"; done
  echo ""
fi

if [[ ${#skipped_dirty[@]} -gt 0 ]]; then
  echo "Skipped (dirty working tree):"
  for s in "${skipped_dirty[@]}"; do echo "  $s"; done
  echo ""
fi

if [[ ${#to_remove[@]} -eq 0 ]]; then
  echo "Nothing to clean up."
  exit 0
fi

if [[ "$FORCE" == false ]]; then
  echo "Would remove ${#to_remove[@]} worktree(s):"
  for entry in "${to_remove[@]}"; do
    IFS='|' read -r path branch <<< "$entry"
    echo "  $path ($branch)"
  done
  echo ""
  echo "Run with --force to actually delete."
else
  echo "Removing ${#to_remove[@]} worktree(s):"
  for entry in "${to_remove[@]}"; do
    IFS='|' read -r path branch <<< "$entry"
    echo "  $path ($branch)"
    git worktree remove --force "$path" 2>/dev/null || rm -rf "$path"
    git branch -D "$branch" 2>/dev/null || true
  done
  echo ""
  echo "Done."
fi
