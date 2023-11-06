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

alias g="git "
alias ...="cd ../.."
alias untar='tar -xvzf'
alias ntar='tar -cf'
alias tl='tmuxloader'
# needed for lazygit to open new tab in neovim
alias vim='nvim --listen /tmp/nvim-server.pipe'
LS_COLORS='no=0:ex=1;41;97:fi=1;42;97:di=1;40;97:ln=04;36:pi=40;33:so=01;35:bd=40;33;01:'
export LS_COLORS
setopt PROMPT_CR
setopt PROMPT_SP
export PROMPT_EOL_MARK=""

[ -f ~/.fzf.zsh ] && source ~/.fzf.zsh

export PATH=~/.local/bin:$PATH
export PATH="$HOME/.yarn/bin:$PATH"
export PATH="$HOME/.npm/bin:$PATH"
export PATH=$PATH:/usr/local/go/bin
export NODE_PATH="/usr/lib/node_modules/"

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
