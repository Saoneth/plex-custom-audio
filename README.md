# plex-custom-audio

## How to install
1. If you are running in docker run: `docker exec -it plex bash`
2. Run `curl https://raw.githubusercontent.com/Saoneth/plex-custom-audio/master/install.sh | bash`
3. It will install ffprobe python3 and this repository in /opt/plex-custom-audio
4. To map audio tracks run `/opt/plex-custom-audio/mapper` as plex user (or `docker exec plex /opt/plex-custom-audio/mapper` on docker)
5. You can speed up maping process by limiting scaned directories in /opt/plex-custom-audio/config.py
6. After plex update run `/opt/plex-custom-audio/update-transcoder`

## Caveats
1. It transcodes everything (Direct Stream) which has custom audio.
2. It might not work with anything which isn't h264 or hevc.
3. If your client doesn't have option to disable direct play and has support for rmvb container it won't work.
4. There is option to disable video transcoding in `Plex Custom Audio Transcoder` but it's currently broken.
5. It's my second script in python ever so code is one big spagetti.

## Requirements
1. python3 with sqlite3 support
2. ffprobe
