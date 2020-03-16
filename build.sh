#!/bin/bash -e
if [[ "$#" -eq "0" ]]; then
   echo "Usage: $0 os"
   exit
fi
N="Dockerfile.${1}"
if [[ -e "$D" ]]; then
    N="${D}-${RANDOM}"
    docker build -t "$N" -f "$D".linux .
    docker image save "$N" | tar x --no-anchored 'layer.tar' -O | tar xvf -
    docker rmi "$N"
    exit
fi
