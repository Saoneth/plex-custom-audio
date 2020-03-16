#!/bin/sh
if [ "$#" -eq "0" ]; then
   echo "Usage: $0 linux|windows"
   exit
fi
if [ "$1" == "linux" ]; then
    docker build -t plex-custom-audio:bins -f Dockerfile .
    docker image save plex-custom-audio:bins | tar x --no-anchored 'layer.tar' -O | tar xvf -
    docker rmi plex-custom-audio:bins
    exit
fi
if [ "$1" == "windows" ]; then
    docker build -t plex-custom-audio:bins-windows -f Dockerfile-windows .
    docker image save plex-custom-audio:bins-windows | tar x --no-anchored 'layer.tar' -O | tar xvf -
    docker rmi plex-custom-audio:bins-windows
    exit
fi
echo "Invalid system."
exit 1

