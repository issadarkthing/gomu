#!/bin/bash

VERSION=$(git describe --abbrev=0 --tags)
VERSION=${VERSION#v}

cd ./dist/gomu

CHECKSUM=$(makepkg -g)

sed -i "s/^pkgver.*/pkgver=$VERSION/" PKGBUILD
sed -i "s/^md5sums.*/$CHECKSUM/" PKGBUILD
makepkg --printsrcinfo > .SRCINFO
