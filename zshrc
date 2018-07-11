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

alias ...="cd ../.."
alias untar='tar -xvzf'
alias ntar='tar -cf'
alias mc='. /usr/share/mc/bin/mc-wrapper.sh'
LS_COLORS='no=0:ex=1;41;97:fi=1;42;97:di=1;40;97:ln=04;36:pi=40;33:so=01;35:bd=40;33;01:'
export LS_COLORS
setopt PROMPT_CR
setopt PROMPT_SP
export PROMPT_EOL_MARK=""

pyenv() { source $HOME/.virtualenvs/$1/bin/activate }

color-ssh() {
    trap "$HOME/bin/colorterm.sh" INT EXIT
    if [[ "$*" =~ "prod" ]]; then
        $HOME/bin/colorterm.sh prod
    elif [[ "$*" =~ "dev" ]]; then
        $HOME/bin/colorterm.sh dev
    else
        $HOME/bin/colorterm.sh other
    fi
    ssh $*
}

compdef _ssh color-ssh=ssh
alias ssh=color-ssh

[ -f ~/.fzf.zsh ] && source ~/.fzf.zsh

export PATH=~/.local/bin:$PATH

