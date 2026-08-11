#
# Defines environment variables.
#

# Ensure that a non-login, non-interactive shell has a defined environment.
if [[ "$SHLVL" -eq 1 && ! -o LOGIN && -s "${ZDOTDIR:-$HOME}/.zprofile" ]]; then
  source "${ZDOTDIR:-$HOME}/.zprofile"
fi
. "$HOME/.cargo/env"

# >>> lean-ctx shell hook >>>
# Passthrough stubs: ensure _lc/_lc_compress exist in ALL zsh contexts
# (non-interactive subshells, eval, agent harnesses) so aliases that
# reference them degrade gracefully instead of "command not found".
# The full shell-hook.zsh overrides these when loaded via .zshrc.
_lc()          { command "$@"; }
_lc_compress() { command "$@"; }
if [[ -z "$LEAN_CTX_ACTIVE" && -z "$LEAN_CTX_NO_HOOK" && -n "$ZSH_EXECUTION_STRING" ]] \
  && [[ "$ZSH_EXECUTION_STRING" != *"__CURSOR_SANDBOX"* ]] \
  && [[ "$ZSH_EXECUTION_STRING" != *"dump_zsh_state"* ]] \
  && [[ "$ZSH_EXECUTION_STRING" != *"lean-ctx hook "* ]] \
  && command -v lean-ctx &>/dev/null; then
  if [[ -n "$LEAN_CTX_AGENT" || -n "$CLAUDECODE" || -n "$CODEBUDDY" || -n "$CODEX_CLI_SESSION" || -n "$GEMINI_SESSION" ]]; then
    export LEAN_CTX_ACTIVE=1
    exec lean-ctx -c "$ZSH_EXECUTION_STRING"
  fi
fi
# <<< lean-ctx shell hook <<<
