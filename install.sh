#!/bin/bash

# if dotfiles doesn't exist in the home directory,
# create it and move files to that directory
if [[ ! -d ~/dotfiles ]]; then
    mkdir ~/dotfiles
    find ./ -maxdepth 1  \( ! -regex '.*/\..*' \) ! -name "*.sh" -exec cp {} ~/dotfiles \; # skip files starting with .
fi

# create symlinks to home folder
for f in $(find ~/dotfiles/* -type f ! -name "*.sh"); do
    FILE=$(basename $f)
    echo installing $FILE...
    echo $FILE

    if [[ $FILE == "init.vim" ]]
    then
        mkdir ~/.config/nvim/
        ln -sf $f ~/.config/nvim/$FILE
    elif [[ $FILE == "vifmrc" ]]
    then
        mkdir ~/.config/vifm/
        rm -f ~/.config/vifm/$FILE
        ln -sf $f ~/.config/vifm/$FILE
    else
        ln -sf $f ~/.$FILE
    fi

done

for d in $(find ~/dotfiles/* -type d)
do
    BASE_DIR=~/.$(basename $d)
    echo $BASE_DIR
    mkdir $BASE_DIR

    for f in $(find ${d}/* -type f)
    do
        FILE=$(basename $f)
        echo $FILE
        ln -sf $d/$FILE ~/.$(basename $d)/$FILE
    done
done

echo Done
