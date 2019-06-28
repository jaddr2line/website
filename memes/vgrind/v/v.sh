#!/bin/bash

set +e

cd $(dirname $0)

export v_commit=165dfe5fe0045642f1168045448a3533107a706f
export vc_commit=26629422db43c556bcdf71badf8be542a15bd54a

rm -f v.tar.gz v.c v.tar
wget https://github.com/vlang/v/archive/$v_commit.tar.gz -O v.tar.gz
wget https://raw.githubusercontent.com/vlang/vc/$vc_commit/v.c -O v-bootstrap.c
gunzip v.tar.gz
tar -rf v.tar v-bootstrap.c --transform "s,^,v-$v_commit/compiler/,"
gzip v.tar

echo "V Commit: $v_commit" > meta.txt
echo "V Bootstrap Commit: $vc_commit" >> meta.txt
