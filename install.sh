#!/bin/bash

CONFIG_DIR="/home/zimmi/.config"
DOTFILES_DIR="/home/zimmi/dotfiles"

mkdir ~/.claude > /dev/null 2>&1
mkdir ~/.claude/hooks > /dev/null 2>&1
mkdir -p ~/.claude/skills/{cf-feedback,cf-mr,cf-ticket,cf-tickets,cf-work,cf-worktree} 2>/dev/null
mkdir ${CONFIG_DIR}/i3 > /dev/null 2>&1
mkdir ${CONFIG_DIR}/i3blocks > /dev/null 2>&1
mkdir ${CONFIG_DIR}/nvim > /dev/null 2>&1
mkdir ${CONFIG_DIR}/vifm > /dev/null 2>&1
mkdir ${CONFIG_DIR}/dunst > /dev/null 2>&1
mkdir ${CONFIG_DIR}/bspwm > /dev/null 2>&1
mkdir ${CONFIG_DIR}/sxhkd > /dev/null 2>&1
mkdir ${CONFIG_DIR}/yabar > /dev/null 2>&1
mkdir ~/.tmuxloader-sessions > /dev/null 2>&1


ln -sf ${DOTFILES_DIR}/claude/CLAUDE.md ~/.claude/CLAUDE.md
ln -sf ${DOTFILES_DIR}/claude/settings.json ~/.claude/settings.json
ln -sf ${DOTFILES_DIR}/claude/hooks/validate-bash.sh ~/.claude/hooks/validate-bash.sh
ln -sf ${DOTFILES_DIR}/claude/hooks/plan-mode-guard.sh ~/.claude/hooks/plan-mode-guard.sh
ln -sf ${DOTFILES_DIR}/claude/simple-statusline.sh ~/.claude/simple-statusline.sh

# claude skills
ln -sf ${DOTFILES_DIR}/claude/skills/cf-feedback/SKILL.md ~/.claude/skills/cf-feedback/SKILL.md
ln -sf ${DOTFILES_DIR}/claude/skills/cf-mr/SKILL.md ~/.claude/skills/cf-mr/SKILL.md
ln -sf ${DOTFILES_DIR}/claude/skills/cf-ticket/SKILL.md ~/.claude/skills/cf-ticket/SKILL.md
ln -sf ${DOTFILES_DIR}/claude/skills/cf-tickets/SKILL.md ~/.claude/skills/cf-tickets/SKILL.md
ln -sf ${DOTFILES_DIR}/claude/skills/cf-work/SKILL.md ~/.claude/skills/cf-work/SKILL.md
ln -sf ${DOTFILES_DIR}/claude/skills/cf-worktree/SKILL.md ~/.claude/skills/cf-worktree/SKILL.md

ln -sf ${DOTFILES_DIR}/i3/config ${CONFIG_DIR}/i3/config
ln -sf ${DOTFILES_DIR}/i3blocks/config ${CONFIG_DIR}/i3blocks/config
ln -sf ${DOTFILES_DIR}/i3blocks/battery ${CONFIG_DIR}/i3blocks/battery
ln -sf ${DOTFILES_DIR}/i3blocks/keyboard-layout ${CONFIG_DIR}/i3blocks/keyboard-layout
ln -sf ${DOTFILES_DIR}/vifm/vifmrc ${CONFIG_DIR}/vifm/vifmrc
ln -sf ${DOTFILES_DIR}/dunst/dunstrc ${CONFIG_DIR}/dunst/dunstrc

ln -sf ${DOTFILES_DIR}/redshift.conf ${CONFIG_DIR}/redshift.conf

ln -sf ${DOTFILES_DIR}/user-dirs.dirs ${CONFIG_DIR}/user-dirs.dirs

ln -sf ${DOTFILES_DIR}/bspwm/bspwmrc ${CONFIG_DIR}/bspwm/bspwmrc
ln -sf ${DOTFILES_DIR}/sxhkd/sxhkdrc ${CONFIG_DIR}/sxhkd/sxhkdrc
ln -sf ${DOTFILES_DIR}/yabar/yabar.conf ${CONFIG_DIR}/yabar/yabar.conf

ln -sf ${DOTFILES_DIR}/ackrc ~/.ackrc
ln -sf ${DOTFILES_DIR}/dir_bookmarks ~/.dir_bookmarks
ln -sf ${DOTFILES_DIR}/gitconfig ~/.gitconfig
ln -sf ${DOTFILES_DIR}/gitignore ~/.gitignore
ln -sf ${DOTFILES_DIR}/psqlrc ~/.psqlrc
ln -sf ${DOTFILES_DIR}/tmux.conf ~/.tmux.conf
ln -sf ${DOTFILES_DIR}/tmuxloader ~/.tmuxloader
ln -sf ${DOTFILES_DIR}/xbindkeysrc ~/.xbindkeysrc
ln -sf ${DOTFILES_DIR}/xsessionrc ~/.xsessionrc
ln -sf ${DOTFILES_DIR}/zlogin ~/.zlogin
ln -sf ${DOTFILES_DIR}/zlogout ~/.zlogout
ln -sf ${DOTFILES_DIR}/zpreztorc ~/.zpreztorc
ln -sf ${DOTFILES_DIR}/zprofile ~/.zprofile
ln -sf ${DOTFILES_DIR}/zshenv ~/.zshenv
ln -sf ${DOTFILES_DIR}/zshrc ~/.zshrc

ln -sf ${DOTFILES_DIR}/tmuxloader-sessions/cartostory ~/.tmuxloader-sessions/cartostory
ln -sf ${DOTFILES_DIR}/tmuxloader-sessions/main ~/.tmuxloader-sessions/main

ln -sf ${DOTFILES_DIR}/npmrc ~/.npmrc

ln -sf ${DOTFILES_DIR}/nvim/init.lua ~/.config/nvim/init.lua
ln -sf ${DOTFILES_DIR}/nvim/lua ${CONFIG_DIR}/nvim/lua
