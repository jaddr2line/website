#!/bin/bash

set +e

cd $(dirname $0)

export binario_commit=831aa8c368c54215e8a1b4af4c0b522465c5e847

rm -rf site/themes/binario
mkdir site/themes/binario
wget https://github.com/Vimux/Binario/archive/$binario_commit.tar.gz -O binario.tar.gz
tar -xvf binario.tar.gz --strip 1 -C site/themes/binario
