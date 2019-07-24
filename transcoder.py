#!/usr/bin/python3

import os
import sys
import sqlite3
from sqlite3 import Error
from pathlib import Path
import config

def main():
    env = os.environ.copy()

    PLEX_TRANSCODER = config.PLEX_PATH + os.sep + "Plex Transcoder_org"
    PLEX_DATABASE_PATH = config.PLEX_LIBRARY_PATH + "/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"

    try:
        conn = sqlite3.connect(PLEX_DATABASE_PATH)
        cur = conn.cursor()

        streams = 0
        maps = 0
        path = None
        audio_path = None
        audio_index = None
        audio_codec = None
        args_iter = iter(sys.argv)
        video_map = None
        no_accurate_seek = False
        for arg in args_iter:
            if arg == "-no_accurate_seek":
                no_accurate_seek = True
            elif arg.find("-codec:") == 0:
                codec = next(args_iter)
                stream = arg[arg.find(":")+1:]
                print("stream", stream)
                try:
                    stream = int(stream)
                except ValueError:
                    stream = 0

                if stream >= 1000:
                    audio_codec = codec

            elif arg == "-i":
                if streams == 0:
                    path = next(args_iter)
                streams = streams + 1

            elif arg == "-map":
                maps = maps + 1
                map = next(args_iter)

                if map.find(":") == -1:
                    continue

                tmp = map.split(":")
                try:
                    index = int(tmp[1])
                except ValueError:
                    index = 0

                if index < 1000:
                    continue

                print(streams)
                print("path: %s" % path)
                if path.startswith("http://127.0.0.1:32400/library/parts/"):
                    media_part_id = path.split('/')[5]
                    print(media_part_id)
                    cur.execute('SELECT `media_item_id` FROM `media_parts` WHERE `id` = ? LIMIT 1', (media_part_id,))
                    (media_item_id,) = cur.fetchone()
                else:
                    cur.execute('SELECT `id`, `media_item_id` FROM `media_parts` WHERE `file` = ? LIMIT 1', (path,))
                    (media_part_id, media_item_id) = cur.fetchone()

                print("media_part_id: %s, media_item_id: %s" % (media_part_id, media_item_id))

                cur.execute('SELECT `url`, `url_index` FROM `media_streams` WHERE `media_part_id` = ? AND `media_item_id` = ? AND `index` = ? LIMIT 1', (media_part_id, media_item_id, index,))
                (url, url_index) = cur.fetchone()
                audio_path = url[7:]
                audio_index = index - 1000

                if url_index != None:
                    audio_index = url_index

            elif arg == "-filter_complex":
                filter_complex = next(args_iter)
                print("filter_complex: %s" % filter_complex)

                j = filter_complex.find(']', 3)

                map = filter_complex[1:j]
                print("map: %s" % map)
                if map == "0:0" and filter_complex.find('scale=') > 0:
                    k = filter_complex.rfind('[', j)
                    video_map = filter_complex[k:filter_complex.find(']', k+1)+1]
                    print("skip: %s" % video_map)
                    continue

                if map.find(":") == -1:
                    continue

                tmp = map.split(":")
                try:
                    index = int(tmp[1])
                except ValueError:
                    index = 0

                audio_index = index

                if index >= 1000:
                    print(streams)

                    print("path: %s" % path)
                    if path.startswith("http://127.0.0.1:32400/library/parts/"):
                        media_part_id = path.split('/')[5]
                        print(media_part_id)
                        cur.execute('SELECT `media_item_id` FROM `media_parts` WHERE `id` = ? LIMIT 1', (media_part_id,))
                        (media_item_id,) = cur.fetchone()
                    else:
                        cur.execute('SELECT `id`, `media_item_id` FROM `media_parts` WHERE `file` = ? LIMIT 1', (path,))
                        (media_part_id, media_item_id) = cur.fetchone()

                    print("media_part_id: %s, media_item_id: %s" % (media_part_id, media_item_id))

                    cur.execute('SELECT `url`, `url_index` FROM `media_streams` WHERE `media_part_id` = ? AND `media_item_id` = ? AND `index` = ? LIMIT 1', (media_part_id, media_item_id, index,))
                    (url, url_index) = cur.fetchone()
                    audio_path = url[7:]
                    audio_index = index - 1000

                    if url_index != None:
                        audio_index = url_index

        print("maps: %d" % maps)
        if maps < 2:
            print("Probably audio streaming")
            print(sys.argv)
            os.execve(PLEX_TRANSCODER, sys.argv, env)

        #if audio_path == None:
        #    print("Not custom audio")
        #    print(sys.argv)
        #    os.execve(PLEX_TRANSCODER, sys.argv, env)

        args = []
        args_iter = iter(sys.argv)
        i = 0
        R = False

        ss = None

        for arg in args_iter:
            if arg.find("-codec:") == 0:
               codec = next(args_iter)
               stream = arg[arg.find(":")+1:]
               try:
                   stream = int(stream)
               except ValueError:
                   stream = -1

               if config.DISABLE_TRANSCODING == True and R == True and stream == 0:
                   codec = "copy"

               if stream < 1000:
                   args.extend([arg, codec])

            elif arg == "-ss":
                ss = next(args_iter)
                args.extend([arg, ss])

            elif arg == "-i":
                if video_map != None:
                    R = True
                args.extend(["-i", next(args_iter)])
                i = i + 1
                if audio_path != None and i == streams:
                    if audio_codec != None:
                        args.extend(["-codec:%d" % audio_index, audio_codec])
                    if ss != None:
                        args.extend(["-ss", ss])
                    #if no_accurate_seek == True:
                    #    args.append("-no_accurate_seek")
                    args.extend(["-probesize", "10000000", "-i", audio_path])

            elif arg == "-map":
                map = next(args_iter)

                print("map: %s, video_map: %s" % (map, video_map,))
                if config.DISABLE_TRANSCODING == True and map == video_map:
                    map = "0:0"

                if map.find(":") != -1:
                    tmp = map.split(":")
                    try:
                        index = int(tmp[1])
                    except ValueError:
                        index = 0
                    if index >= 1000:
                        map = "%d:%d" % (streams, audio_index,)

                args.append(arg)
                args.append(map)

            elif arg == "-filter_complex":
                filter_complex = next(args_iter)
                j = filter_complex.find(']', 3)

                map = filter_complex[1:j]
                print("map: %s" % map)
                if config.DISABLE_TRANSCODING == True and map == "0:0" and filter_complex.find('scale=') > 0:
                    print("skip: %s" % filter_complex)
                    continue

                print(audio_path)

                if audio_path != None and map.find(":") != -1:
                    tmp = map.split(":")
                    try:
                        index = int(tmp[1])
                    except ValueError:
                        index = 0

                    print("tmp: %s" % tmp)
                    if index >= 1000:
                        filter_complex = "[%d:%d]%s" % (streams, audio_index, filter_complex[j+1:],)

                args.extend([arg, filter_complex])

            elif config.DISABLE_TRANSCODING == True and ( arg == "-crf:0" or arg == "-maxrate:0" or arg == "-bufsize:0" or arg == "-r:0" or arg == "-preset:0" or arg == "-level:0" or arg == "-x264opts:0" or arg == "-force_key_frames:0" ):
                next(args_iter)
                continue

            elif arg == "-loglevel": #or arg == "-loglevel_plex":
                args.extend([arg, "verbose"])
                next(args_iter)
                continue

            else:
                args.append(arg)
                continue

        args[0] = 'Plex Transcoder_org'

        print("Custom audio detected")
        print(sys.argv)
        print("new:")
        print(args)

        conn.commit()
        conn.close()

        if config.DEBUG == False:
            os.execve(PLEX_TRANSCODER, args, env)
    except Error as e:
        print(e)
        sys.exit(1)


if __name__ == '__main__':
    main()
