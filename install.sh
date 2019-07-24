#!/bin/bash -x
# Install python
apt update
apt install --no-install-recommends -y python3

# Install ffprobe
if [[ -z $PLEX_ARCH ]]; then
    ARCH="$(uname -m)"
else
    ARCH="$PLEX_ARCH"
fi
[[ "$ARCH" == "x86_64" ]] && ARCH="64bit"
[[ "$ARCH" == "amd64" ]] && ARCH="64bit"
[[ "$ARCH" == "i386" ]] && ARCH="32bit"

apt install --no-install-recommends -y xz-utils
wget "https://www.johnvansickle.com/ffmpeg/old-releases/ffmpeg-4.0.3-${ARCH}-static.tar.xz" -O ffmpeg.tar.xz
tar xvf ffmpeg.tar.xz
mv -v ffmpeg-*/ffprobe /usr/local/bin/
rm -rfv ffmpeg-*
apt remove --purge -y xz-utils

# Cleanup
apt-get clean
rm -rf \
	/etc/default/plexmediaserver \
	/tmp/* \
	/var/lib/apt/lists/* \
	/var/tmp/*

# Install custom transcoder
wget "https://github.com/Saoneth/plex-custom-audio/archive/master.tar.gz" -O /tmp/src.tar.gz
tar -xvf /tmp/src.tar.gz -C /opt
mv -v /opt/plex-custom-audio-master /opt/plex-custom-audio
/opt/plex-custom-audio/update-transcoder

# Change library path for docker version
[[ "$VERSION" == "docker" ]] && sed -i 's/\/var\/lib\/plexmediaserver\//\/config\//' /opt/plex-custom-audio/config.py

echo 'Run /opt/plex-custom-audio/mapper to map audio'
#/opt/plex-custom-audio/mapper
