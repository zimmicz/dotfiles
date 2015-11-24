#!/bin/bash

# if dotfiles doesn't exist in the home directory,
# create it and move files to that directory
if [[ ! -d ~/dotfiles ]]; then
    mkdir ~/dotfiles
    find ./ -maxdepth 1  \( ! -regex '.*/\..*' \) ! -name "*.sh" -exec cp {} ~/dotfiles \; # skip files starting with .
fi

# create symlinks to home folder
for f in $(find ~/dotfiles/* ! -name "*.sh"); do
    FILE=$(basename $f)
    echo installing $FILE...
    echo $FILE
    ln -sf $f ~/.$FILE
done

echo Done
