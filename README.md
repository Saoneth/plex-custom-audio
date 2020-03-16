# plex-custom-audio

## How to install
1. If you are running in docker run: `docker exec -it plex bash`. For Windows you can download binaries from Releases tab and DIY.
2. Get into plex installation directory (eg. `cd /usr/lib/plexmediaplayer`).
2. Run `curl https://raw.githubusercontent.com/Saoneth/plex-custom-audio/master/install.sh | sudo bash`
4. To map audio tracks run `./Plex\ Custom\ Audio\ Mapper` as plex user (or `docker exec plex "/usr/lib/plexmediaplayer/Plex Custom Audio Mapper"` on docker)
5. You can speed up maping process by limiting scaned directories. You can provide them as arguments for mapper. `./Plex\ Custom\ Audio\ Mapper "/Movies/" "/TV Shows/Some very specific movie (2020)/"

## Caveats
1. It remuxes everything which has custom audio (even when you select build in track).
2. It probably will require video transcoding for codecs not supported by hls or dash (eg. xvid).
