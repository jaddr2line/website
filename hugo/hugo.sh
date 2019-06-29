#!/bin/bash

set -e

cd $(dirname $0)

export HUGO_VERSION=0.55.6

rm -f hugo.tar.gz
wget https://github.com/gohugoio/hugo/releases/download/v$HUGO_VERSION/hugo_"$HUGO_VERSION"_Linux-64bit.tar.gz -O hugo.tar.gz
