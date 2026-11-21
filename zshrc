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
alias cr="command claude --resume"
alias tf="terraform "
alias lg="lazygit"
alias g="git "
alias nr="command npm run"
alias pnr="command pnpm run"
npm() { if [[ "$1" == "run" ]]; then echo "Use nr instead" && return 1; fi; command npm "$@"; }
alias ...="cd ../.."
alias untar='tar -xvzf'
alias ntar='tar -cf'
alias tl='tmuxloader'
alias vim='/opt/nvim/bin/nvim'
alias nvim='/opt/nvim/bin/nvim'
alias k=kubectl
export KUBE_EDITOR=/opt/nvim/bin/nvim
command -v kubectl >/dev/null && source <(kubectl completion zsh) && compdef k=kubectl
source ~/.kube-ps1/kube-ps1.sh
setopt PROMPT_SUBST
RPROMPT='$(kube_ps1)'
LS_COLORS='no=0:ex=1;41;97:fi=1;42;97:di=1;40;97:ln=04;36:pi=40;33:so=01;35:bd=40;33;01:'
export LS_COLORS
setopt PROMPT_CR
setopt PROMPT_SP
export PROMPT_EOL_MARK=""
do=(--dry-run=client -o yaml)
now=(--force --grace-period 0)

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
unalias sl 2> /dev/null

# fzf
export FZF_DEFAULT_COMMAND='fdfind --type f --hidden --exclude .git'
export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
export FZF_CTRL_T_OPTS="--preview 'batcat --color=always --line-range :200 {}'"
export FZF_ALT_C_COMMAND='fdfind --type d --hidden --exclude .git'
source /usr/share/doc/fzf/examples/key-bindings.zsh
source /usr/share/doc/fzf/examples/completion.zsh
zstyle ':fzf-tab:complete:cd:*' fzf-preview 'ls -1 --color=always $realpath'

eval "$(mise activate zsh)"
eval "$(zoxide init zsh)"
. "/home/zimmi/.deno/env"

_habit_hint() {
  local h
  case $1 in
    cd\ .*|cd\ -*) ;;
    cd\ */*/*) h="z ${${${1#cd }%/}:t}" ;;
    lg|lazygit) [[ -n $TMUX ]] && h='` g = lazygit popup' ;;
    tl|tl\ *|tmuxloader*) h='` S = výběr session přes fzf' ;;
    history*) h='ctrl+r = fzf historie' ;;
    find\ *-name*) h='ctrl+t = fzf soubor s náhledem' ;;
  esac
  [[ -n $h ]] && print -r -- $'\e[33m'"tip: $h"$'\e[0m'
}
autoload -Uz add-zsh-hook
add-zsh-hook preexec _habit_hint

() {
  local tips=(${(f)"$(<~/dotfiles/tips.txt)"})
  print -r -- $'\e[36m'"${tips[RANDOM % $#tips + 1]}"$'\e[0m'
}

# Added by flyctl installer
export FLYCTL_INSTALL="/home/zimmi/.fly"
export PATH="$FLYCTL_INSTALL/bin:$PATH"

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
