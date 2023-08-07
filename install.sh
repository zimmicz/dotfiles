#!/bin/bash

CONFIG_DIR="/home/zimmi/.config"
DOTFILES_DIR="/home/zimmi/dotfiles"

mkdir ${CONFIG_DIR}/i3 > /dev/null 2>&1
mkdir ${CONFIG_DIR}/i3blocks > /dev/null 2>&1
mkdir ${CONFIG_DIR}/nvim > /dev/null 2>&1
mkdir ${CONFIG_DIR}/vifm > /dev/null 2>&1
mkdir ${CONFIG_DIR}/dunst > /dev/null 2>&1
mkdir ${CONFIG_DIR}/bspwm > /dev/null 2>&1
mkdir ${CONFIG_DIR}/sxhkd > /dev/null 2>&1
mkdir ${CONFIG_DIR}/yabar > /dev/null 2>&1
mkdir ~/.tmuxloader-sessions > /dev/null 2>&1

ln -sf ${DOTFILES_DIR}/i3/config ${CONFIG_DIR}/i3/config
ln -sf ${DOTFILES_DIR}/i3blocks/config ${CONFIG_DIR}/i3blocks/config
ln -sf ${DOTFILES_DIR}/i3blocks/battery ${CONFIG_DIR}/i3blocks/battery
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
ln -sf ${DOTFILES_DIR}/tmuxloader-sessions/livesport ~/.tmuxloader-sessions/livesport
ln -sf ${DOTFILES_DIR}/tmuxloader-sessions/lstv-admin ~/.tmuxloader-sessions/lstv-admin
ln -sf ${DOTFILES_DIR}/tmuxloader-sessions/main ~/.tmuxloader-sessions/main



