# plex-custom-audio

## How to install
1. Extract everything to Plex directory: `/usr/lib/plexmediaserver` (the one with `Plex Transcoder`, etc.)
2. Run `update-transcoder.sh`
3. Add path to your media in `Plex Custom Audio Mapper`
4. Run it as plex, eg. `sudo -u plex './Plex Custom Audio Mapper'`

## Caveats
1. It transcodes everything (Direct Stream) which has custom audio.
2. It might not work with anything which isn't h264 or hevc.
3. If your client doesn't have option to disable direct play and has support for rmvb container it won't work.
4. There is option to disable video transcoding in `Plex Custom Audio Transcoder` but it's currently broken.
5. It's my second script in python ever so code is one big spagetti.

## Requirements
1. python3
2. ffprobe (It's part of ffmpeg. You can download static build.)
3. Some python packages, probably?
