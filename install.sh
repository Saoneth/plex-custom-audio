#!/bin/sh -e
TAG="bins"

[ ! -e "Plex Transcoder_org" ] && mv "Plex Transcoder" "Plex Transcoder_org"

TOKEN="$(curl -s "https://auth.docker.io/token?service=registry.docker.io&scope=repository:saoneth/plex-custom-audio:pull" | cut -d'"' -f4)"
BLOB="$(curl -s -H "Authorization: Bearer ${TOKEN}" "https://registry.hub.docker.com/v2/saoneth/plex-custom-audio/manifests/${TAG}" | grep '"blobSum":' | cut -d'"' -f4)"
curl -s -L -H "Authorization: Bearer ${TOKEN}" "https://registry.hub.docker.com/v2/saoneth/plex-custom-audio/blobs/${BLOB}" | tar xzv
