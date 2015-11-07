#!/bin/bash


for f in $(find ~/dotfiles/* ! -name "*.sh"); do
    FILE=$(basename $f)
    echo installing $FILE...
    echo $FILE
    ln -sf $f ~/.$FILE
done

echo Done
