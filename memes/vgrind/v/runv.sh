#!/bin/bash

set -e

cat meta.txt
echo "Loading source. . ."
cat /dev/stdin > prog.v
echo "Compiling. . . "
v -o prog prog.v
echo "Running. . . "
valgrind --leak-check=full --show-leak-kinds=all -v ./prog
echo "Done!"
