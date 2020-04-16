#!/bin/bash -e
if [[ "$#" -eq "0" ]]; then
   echo "Usage: $0 os"
   exit
fi
D="Dockerfile.${1}"
if [[ ! -e "$D" ]]; then
    echo "Invalid os"
    exit 1
fi
N="plex-custom-audio:${1}-${RANDOM}"
docker build -t "$N" -f "$D" .
docker image save "$N" | tar x --no-anchored 'layer.tar' -O | tar xvf -
docker rmi "$N"
exit
