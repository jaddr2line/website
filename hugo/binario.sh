#!/bin/bash

set -e

cd $(dirname $0)

export binario_commit=ec26d2e882963d500f9ddba8a9e87ae70024542b

rm -rf site/themes/binario
mkdir site/themes/binario
wget https://github.com/Vimux/Binario/archive/$binario_commit.tar.gz -O binario.tar.gz
tar -xvf binario.tar.gz --strip 1 -C site/themes/binario
