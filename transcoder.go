package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
	"strconv"
	"syscall"
)

func getDBPath() string {
	var p string

	// Docker
	p = "/config/Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"
	if _, err := os.Stat(p); err == nil { return p }

	// Debian, Fedora, CentOS, Ubuntu
	p = "/var/lib/plexmediaserver/Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"
	if _, err := os.Stat(p); err == nil { return p }

	// FreeBSD
	p = "/usr/local/plexdata/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"
	if _, err := os.Stat(p); err == nil { return p }

	// ReadyNAS
	p = "/apps/plexmediaserver/MediaLibrary/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"
	if _, err := os.Stat(p); err == nil { return p }

	home, err := os.UserHomeDir()
	if err == nil {
		// Windows
		if runtime.GOOS == "windows" {
			p = home + "\\AppData\\Local\\Plex Media Server\\Plug-in Support\\Databases\\com.plexapp.plugins.library.db"
			if _, err := os.Stat(p); err == nil { return p }
		}

		// macOS
		p = home + "Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"
		if _, err := os.Stat(p); err == nil { return p }
	}

	return "com.plexapp.plugins.library.db"
}

func main() {
	f, err := os.OpenFile("/tmp/plex-custom-audio.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Args:")
	log.Println(os.Args)

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=rw", getDBPath()))
	if err != nil {
		log.Fatal(err, "You can add support for your configuration by creating link to com.plexapp.plugins.library.db in the same directory as this application")
	}
	defer db.Close()

	streams := 0
	maps := 0
	path := ""
	audio_path := ""
	audio_index := -1
	audio_codec := ""
	video_map := ""
	no_accurate_seek := false

	get_media_by_id_stmt, err := db.Prepare("SELECT `media_item_id` FROM `media_parts` WHERE `id` = ? LIMIT 1")
	if err != nil {
		log.Fatal(err)
	}
	defer get_media_by_id_stmt.Close()

	get_media_by_file_stmt, err := db.Prepare("SELECT `id`, `media_item_id` FROM `media_parts` WHERE `file` = ? LIMIT 1")
	if err != nil {
		log.Fatal(err)
	}
	defer get_media_by_file_stmt.Close()

	get_media_stream_url_stmt, err := db.Prepare("SELECT `url`, `url_index` FROM `media_streams` WHERE `media_part_id` = ? AND `media_item_id` = ? AND `index` = ? LIMIT 1")
	if err != nil {
		log.Fatal(err)
	}
	defer get_media_stream_url_stmt.Close()

	getInfo := func(path string, index int) (audio_path string, audio_index int) {
		log.Println("path:", path)
		var media_part_id int
		var media_item_id int
		if strings.HasPrefix(path, "http://127.0.0.1:32400/library/parts/") {
			media_part_id, err := strconv.Atoi(strings.Split(path, "/")[5])
			if err != nil {
				log.Fatal(err)
			}

			err = get_media_by_id_stmt.QueryRow(media_part_id).Scan(&media_item_id)
		} else {
			err = get_media_by_file_stmt.QueryRow(path).Scan(&media_part_id, &media_item_id)
		}
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("media_part_id: %d, media_item_id: %d\n", media_part_id, media_item_id)

		var url string
		var url_index int
		get_media_stream_url_stmt.QueryRow(media_part_id, media_item_id, index).Scan(&url, &url_index)
		audio_path = url[7:]
		//audio_index = index - 1000
		audio_index = url_index

		return audio_path, audio_index
	}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-no-accurate_seek" {
			no_accurate_seek = true
		} else if strings.HasPrefix(arg, "-codec:") {
			stream, err := strconv.Atoi(arg[7:])
			if err != nil {
				log.Fatal(err)
			}
			i++
			codec := os.Args[i]
			log.Println("stream", stream)
			if stream >= 1000 {
				audio_codec = codec
			}
		} else if arg == "-i" {
			i++
			if streams == 0 {
				path = os.Args[i]
			}
			streams++
		} else if arg == "-map" {
			maps++
			i++
			mapp := os.Args[i]

			if strings.IndexByte(mapp, ':') == -1 {
				continue
			}

			index, err := strconv.Atoi(strings.Split(mapp, ":")[1])
			if err != nil {
				log.Println(err)
				continue
			}

			if index < 1000 {
				log.Println("skip:", index)
				continue
			}

			log.Printf("streams: %d\n", streams)

			audio_path, audio_index = getInfo(path, index)
			log.Printf("audio_path: %s, audio_index: %d\n", audio_path, audio_index)
		} else if arg == "-filter_complex" {
			i++
			filter_complex := os.Args[i]
			log.Printf("filter_complex: %s\n", filter_complex)

			j := strings.IndexByte(filter_complex, ']')
			mapp := filter_complex[1:j]
			log.Printf("map: %s\n", mapp)

			if strings.IndexByte(filter_complex, ':') == -1 {
				continue
			}

			index, err := strconv.Atoi(strings.Split(mapp, ":")[1])
			if err != nil {
				// index = 0
				log.Fatal(err)
			}
			audio_index = index

			if index >= 1000 {
				log.Println("Path:", path)
				audio_path, audio_index = getInfo(path, index)
			}
			log.Printf("audio_path: %s, audio_index: %d\n", audio_path, audio_index)
		}
//		log.Println(arg)
	}

	if maps < 2 {
		log.Println("Probably audio streaming")
		if err := syscall.Exec(os.Args[0] + "_org", os.Args, os.Environ()); err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("audio_path: %s, audio_index: %d\n", audio_path, audio_index)

	args := []string{}
	ss := ""
	s := 0

	for i := 0; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-codec:") {
			stream, err := strconv.Atoi(arg[7:])
			if err != nil {
				log.Fatal(err)
			}
			i++
			codec := os.Args[i]

			if stream < 1000 {
				args = append(args, arg, codec)
			}
		} else if arg == "-ss" {
			i++
			ss = os.Args[i]
			args = append(args, arg, ss)
		} else if arg == "-i" {
			s++
			i++
			args = append(args, arg, os.Args[i])
			if audio_path != "" && s == streams {
				if audio_codec != "" {
					args = append(args, fmt.Sprintf("-codec:%d", audio_index))
					args = append(args, audio_codec)
				}
				if ss != "" {
					args = append(args, "-ss")
					args = append(args, ss)
				}
				if no_accurate_seek {
					// With this switch, it's faster, but sometimes at the beginning for a few seconds there is no audio
					// args = append(args, "-no_accurate_seek")
				}
				args = append(args, "-analyzeduration", "20000000", "-probesize", "20000000", "-i", audio_path)
			}
		} else if arg == "-map" {
			i++
			mapp := os.Args[i]

			log.Printf("map: %s, video_map: %s\n", mapp, video_map)
			if strings.IndexByte(mapp, ':') != -1 {
				index, err := strconv.Atoi(strings.Split(mapp, ":")[1])
				if err != nil {
					// index = 0
					log.Println(err)
				} else {
					if index >= 1000 {
						mapp = fmt.Sprintf("%d:%d", streams, audio_index)
					}
				}
			}

			args = append(args, arg, mapp)
		} else if arg == "-filter_complex" {
			i++
			filter_complex := os.Args[i]
			log.Printf("filter_complex: %s\n", filter_complex)

			j := strings.IndexByte(filter_complex, ']')
			mapp := filter_complex[1:j]
			log.Printf("map: %s\n", mapp)

			if audio_path != "" && strings.IndexByte(filter_complex, ':') != -1 {
				index, err := strconv.Atoi(strings.Split(mapp, ":")[1])
				if err != nil {
					// index = 0
					log.Fatal(err)
				}
				if index >= 1000 {
					filter_complex = fmt.Sprintf("[%d:%d]%s", streams, audio_index, filter_complex[j+1:])
				}
			}

			args = append(args, arg, filter_complex)
		} else if arg == "-loglevel" || arg == "-loglevel_plex" {
			args = append(args, arg, "verbose")
			i++
		} else {
			args = append(args, arg)
		}
	}


	args[0] = args[0] + "_org"

	log.Println("Custom audio detected. New args:")
	log.Println(args)

	if err := syscall.Exec(args[0], args, os.Environ()); err != nil {
		log.Fatal(err)
	}
}
