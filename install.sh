#!/bin/bash

CONFIG_DIR="/home/zimmi/.config"
DOTFILES_DIR="/home/zimmi/dotfiles"

mkdir -p ~/.claude/usage ${CONFIG_DIR}/systemd/user > /dev/null 2>&1
mkdir ${CONFIG_DIR}/i3 > /dev/null 2>&1
mkdir ${CONFIG_DIR}/i3blocks > /dev/null 2>&1
mkdir ${CONFIG_DIR}/nvim > /dev/null 2>&1
mkdir ${CONFIG_DIR}/vifm > /dev/null 2>&1
mkdir ${CONFIG_DIR}/dunst > /dev/null 2>&1
mkdir ${CONFIG_DIR}/bspwm > /dev/null 2>&1
mkdir ${CONFIG_DIR}/sxhkd > /dev/null 2>&1
mkdir ${CONFIG_DIR}/yabar > /dev/null 2>&1
mkdir -p ${CONFIG_DIR}/workmux > /dev/null 2>&1
mkdir ~/.tmuxloader-sessions > /dev/null 2>&1


# claude: user/ = personal, cleverfarm/ = work-scoped; every file symlinked to ~/.claude at same relative path,
# systemd/ units go to ~/.config/systemd/user, docs/ stays unlinked
find ~/.claude ${CONFIG_DIR}/systemd/user -xtype l -lname "${DOTFILES_DIR}/claude/*" -delete 2>/dev/null
for scope in user cleverfarm; do
  for f in $(cd ${DOTFILES_DIR}/claude/${scope} && find * -type f ! -path 'docs/*' ! -path 'systemd/*'); do
    mkdir -p ~/.claude/$(dirname ${f})
    ln -sf ${DOTFILES_DIR}/claude/${scope}/${f} ~/.claude/${f}
  done
  for u in ${DOTFILES_DIR}/claude/${scope}/systemd/*; do
    ln -sf ${u} ${CONFIG_DIR}/systemd/user/$(basename ${u})
  done
done
systemctl --user daemon-reload && systemctl --user enable claude-reflection.timer

claude mcp get playwright > /dev/null 2>&1 || claude mcp add -s user playwright -- npx -y @playwright/mcp@latest --headless

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

ln -sf ${DOTFILES_DIR}/workmux/config.yaml ${CONFIG_DIR}/workmux/config.yaml

ln -sf ${DOTFILES_DIR}/nvim/init.lua ~/.config/nvim/init.lua
ln -sf ${DOTFILES_DIR}/nvim/lua ${CONFIG_DIR}/nvim/lua

[ -d ~/.kube-ps1 ] || git clone --depth 1 https://github.com/jonmosco/kube-ps1 ~/.kube-ps1
[ -d ~/.kubectx ] || git clone --depth 1 https://github.com/ahmetb/kubectx ~/.kubectx
mkdir -p ~/.local/bin ~/.zsh/completions
ln -sf ~/.kubectx/kubectx ~/.local/bin/kubectx
ln -sf ~/.kubectx/kubens ~/.local/bin/kubens
ln -sf ~/.kubectx/completion/_kubectx.zsh ~/.zsh/completions/_kubectx
ln -sf ~/.kubectx/completion/_kubens.zsh ~/.zsh/completions/_kubens

WORKMUX_VERSION=v0.1.271
if ! ~/.local/bin/workmux --version 2>/dev/null | grep -q "${WORKMUX_VERSION#v}"; then
  WM_TMP=$(mktemp -d)
  WM_URL=https://github.com/raine/workmux/releases/download/${WORKMUX_VERSION}
  curl -fsSL -o ${WM_TMP}/wm.tar.gz ${WM_URL}/workmux-linux-amd64.tar.gz \
    && curl -fsSL -o ${WM_TMP}/wm.sha256 ${WM_URL}/workmux-linux-amd64.sha256 \
    && [ "$(sha256sum ${WM_TMP}/wm.tar.gz | cut -d' ' -f1)" = "$(cut -d' ' -f1 ${WM_TMP}/wm.sha256)" ] \
    && tar xzf ${WM_TMP}/wm.tar.gz -C ${WM_TMP} workmux \
    && install -m 755 ${WM_TMP}/workmux ~/.local/bin/workmux
  rm -rf ${WM_TMP}
fi
