# Add deno completions to search path
if [[ ":$FPATH:" != *":/home/zimmi/.zsh/completions:"* ]]; then export FPATH="/home/zimmi/.zsh/completions:$FPATH"; fi
#
# Executes commands at the start of an interactive session.
#
# Authors:
#   Sorin Ionescu <sorin.ionescu@gmail.com>
#

# Source Prezto.
if [[ -s "${ZDOTDIR:-$HOME}/.zprezto/init.zsh" ]]; then
  source "${ZDOTDIR:-$HOME}/.zprezto/init.zsh"
fi

# Customize to your needs...


alias c="command claude --enable-auto-mode"
alias xclaude="command claude --dangerously-skip-permissions"
alias dclaude='docker run --rm -it --cap-add NET_ADMIN --cap-add NET_RAW --user $(id -u):$(id -g) -v ~/.claude:/home/node/.claude -v ~/.claude.json:/home/node/.claude.json -v /home/zimmi/dotfiles:/home/zimmi/dotfiles -v $PWD:/workspace -w /workspace claude-code claude --dangerously-skip-permissions'
alias claude='echo "Use c instead" && false'
alias tf="terraform "
alias lg="lazygit"
alias g="git "
alias nr="command npm run"
alias pnr="command pnpm run"
npm() { if [[ "$1" == "run" ]]; then echo "Use nr instead" && return 1; fi; command npm "$@"; }
alias nvm="fnm"
alias ...="cd ../.."
alias untar='tar -xvzf'
alias ntar='tar -cf'
alias tl='tmuxloader'
alias vim='/opt/nvim/bin/nvim'
alias nvim='/opt/nvim/bin/nvim'
LS_COLORS='no=0:ex=1;41;97:fi=1;42;97:di=1;40;97:ln=04;36:pi=40;33:so=01;35:bd=40;33;01:'
export LS_COLORS
setopt PROMPT_CR
setopt PROMPT_SP
export PROMPT_EOL_MARK=""

[ -f ~/.fzf.zsh ] && source ~/.fzf.zsh

export PATH=~/.local/bin:$PATH
export PATH="$HOME/.yarn/bin:$PATH"
export PATH="$HOME/.npm/bin:$PATH"
export PATH=$PATH:$(go env GOPATH)/bin
export NODE_PATH="/usr/lib/node_modules/"

reclaude() {
  local n=$1
  shift

  local args=()
  local prompt=""

  # Collect all args; last one is the prompt
  while [[ $# -gt 1 ]]; do
    args+=("$1")
    shift
  done
  prompt="$1"

  for ((i=1; i<=n; i++)); do
    echo "=== Run $i/$n ==="
    claude "${args[@]}" --print "$prompt"
  done
}

unalias cdg 2> /dev/null
cdg() {
   local dest_dir=$(< ~/.dir_bookmarks | fzf )
   if [[ $dest_dir != '' ]]; then
      cd "$dest_dir"
   fi
}

unalias sl 2> /dev/null

# fzf
source /usr/share/doc/fzf/examples/key-bindings.zsh
source /usr/share/doc/fzf/examples/completion.zsh

# fnm
export PATH="/home/zimmi/.local/share/fnm:$PATH"
eval "`fnm env --use-on-cd`"
. "/home/zimmi/.deno/env"

# Added by flyctl installer
export FLYCTL_INSTALL="/home/zimmi/.fly"
export PATH="$FLYCTL_INSTALL/bin:$PATH"

if [[ -n "$TMUX" ]]; then
  precmd_tmux_title() {
    local toplevel
    toplevel=$(git rev-parse --show-toplevel 2>/dev/null)
    if [[ "$toplevel" == */.worktrees/* ]]; then
      tmux rename-window "$(basename "$toplevel")"
    fi
  }
  precmd_functions+=(precmd_tmux_title)
fi

#THIS MUST BE AT THE END OF THE FILE FOR SDKMAN TO WORK!!!
export SDKMAN_DIR="$HOME/.sdkman"
[[ -s "$HOME/.sdkman/bin/sdkman-init.sh" ]] && source "$HOME/.sdkman/bin/sdkman-init.sh"
# lean-ctx shell hook — begin
if [ -f "/home/zimmi/.config/lean-ctx/shell-hook.zsh" ]; then
. "/home/zimmi/.config/lean-ctx/shell-hook.zsh"
fi
# lean-ctx shell hook — end

# >>> lean-ctx agent aliases >>>
alias claude='LEAN_CTX_AGENT=1 BASH_ENV="$HOME/.bashenv" claude'
alias codebuddy='LEAN_CTX_AGENT=1 BASH_ENV="$HOME/.bashenv" codebuddy'
alias codex='LEAN_CTX_AGENT=1 BASH_ENV="$HOME/.bashenv" codex'
alias gemini='LEAN_CTX_AGENT=1 BASH_ENV="$HOME/.bashenv" gemini'
# <<< lean-ctx agent aliases <<<
