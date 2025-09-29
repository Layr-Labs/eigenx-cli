#!/usr/bin/env bash

VERSION=$1

if [[ -z $VERSION ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi


for i in $(ls release); do
    if [[ $i == windows-* ]]; then
        fileName="eigenx-cli-${i}-${VERSION}.zip"
        (cd "./release/${i}/" && zip -r "../${fileName}" eigenx.exe)
    else
        fileName="eigenx-cli-${i}-${VERSION}.tar.gz"
        tar -czvf "./release/${fileName}" -C "./release/${i}/" eigenx
    fi
done
