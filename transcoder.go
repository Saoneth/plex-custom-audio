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
	"runtime"
	"path/filepath"
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
		p = home + "/Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db"
		if _, err := os.Stat(p); err == nil { return p }
	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath + "/com.plexapp.plugins.library.db"
}

func runTranscoder(args string[]) {
	err := syscall.Exec(args[0] + "_org", args, os.Environ())
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	f, err := os.OpenFile(os.TempDir() + "/plex-custom-audio.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
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

	getInfo := func(path string, index int) (audioPath string, audioIndex int) {
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
		audioPath = url[7:]
		//audioIndex = index - 1000
		audioIndex = url_index

		return audioPath, audioIndex
	}

	inputCounter := 0
	mapCounter := 0
	path := ""
	audioPath := ""
	audioIndex := -1
	audioCodec := ""
	videoMap := ""
	hasNoAccurateSeek := false
	seek := ""

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-no-accurate_seek" {
			hasNoAccurateSeek = true
			continue
		}
		if arg == "-ss" {
			i++
			seek = os.Args[i]
			continue
		}
		if strings.HasPrefix(arg, "-codec:") {
			i++
			streamIndex, err := strconv.Atoi(arg[7:])
			if err == nil && streamIndex >= 1000 {
				continue
			}
			audioCodec = os.Args[i]
			log.Printf("audioCodec: %s\n", audioCodec)
			continue
		}
		if arg == "-i" {
			i++
			if inputCounter == 0 {
				path = os.Args[i]
			}
			inputCounter++
			continue
		}
		if arg == "-map" {
			i++
			streamMap := os.Args[i]
			mapCounter++

			if strings.IndexByte(streamMap, ':') == -1 {
				continue
			}

			streamIndex, err := strconv.Atoi(strings.Split(streamMap, ":")[1])
			if err != nil {
				continue
			}

			if streamIndex < 1000 {
				log.Printf("skip: inputCounter: %d, path: %s, index: %s\n", inputCounter, path, streamIndex)
				continue
			}

			audioPath, audioIndex = getInfo(path, streamIndex)
			log.Printf("inputCounter: %d, audioPath: %s, audioIndex: %d\n", inputCounter, audioPath, audioIndex)
			continue
		}
		if arg == "-filter_complex" {
			i++
			filter_complex := os.Args[i]
			log.Printf("filter_complex: %s\n", filter_complex)

			j := strings.IndexByte(filter_complex, ']')
			streamMap := filter_complex[1:j]
			log.Printf("filter_complex map: %s\n", streamMap)

			if strings.IndexByte(filter_complex, ':') == -1 {
				continue
			}

			streamIndex, err := strconv.Atoi(strings.Split(streamMap, ":")[1])
			if err != nil {
				// index = 0
				log.Fatal(err)
			}
			audioIndex = streamIndex

			if streamIndex >= 1000 {
				log.Println("Path:", path)
				audioPath, audioIndex = getInfo(path, streamIndex)
			}
			log.Printf("audioPath: %s, audioIndex: %d\n", audioPath, audioIndex)
			continue
		}
	}

	if mapCounter < 2 {
		log.Println("Probably audio streaming")
		runTranscoder(os.Args)
	}

	log.Printf("audioPath: %s, audioIndex: %d\n", audioPath, audioIndex)

	args := []string{os.Args[0] + "_org"}
	currentInputIdx := 0

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		
		// Add -codec only if it's for valid input
		if strings.HasPrefix(arg, "-codec:") {
			i++
			index, err := strconv.Atoi(arg[7:])
			if err != nil || index < 1000 {
				args = append(args, arg, os.Args[i])
			}
			continue
		}

		// Input flag
		if arg == "-i" {
			i++
			currentInputIdx++

			// Append flags as they are
			args = append(args, arg, os.Args[i])
			
			// If it's last not last input - skip
			if currentInputIdx != inputCounter {
				continue
			}
			
			// Add codec flag
			if audioCodec != "" {
				args = append(args, fmt.Sprintf("-codec:%d", audioIndex), audioCodec)
			}

			// Add seeking flag
			if seek != "" {
				args = append(args, "-ss", seek)
			}

			// Add -no_accurate_seek
			if hasNoAccurateSeek {
				// With this switch, it's faster, but sometimes at the beginning for a few seconds there is no audio
				// args = append(args, "-no_accurate_seek")
			}
			
			// Add new input with audio file
			args = append(args, "-analyzeduration", "20000000", "-probesize", "20000000", "-i", audioPath)
			continue
		}

		// Audio identifier could be specified in this flag
		if arg == "-map" {
			i++
			streamMap := os.Args[i]

			log.Printf("streamMap: %s, videoMap: %s\n", streamMap, videoMap)
			if strings.IndexByte(streamMap, ':') != -1 {
				index, err := strconv.Atoi(strings.Split(streamMap, ":")[1])
				// Replace invalid stream identifier for audio with working one
				if err == nil && index >= 1000 {
					streamMap = fmt.Sprintf("%d:%d", inputCounter, audioIndex)
				}
			}

			args = append(args, arg, streamMap)
			continue
		}

		// Audio identifier could be specified in this flag
		if arg == "-filter_complex" {
			i++
			filter_complex := os.Args[i]
			log.Printf("filter_complex: %s\n", filter_complex)

			j := strings.IndexByte(filter_complex, ']')
			streamMap := filter_complex[1:j]
			log.Printf("filter_complex map: %s\n", streamMap)

			if audioPath != "" && strings.IndexByte(filter_complex, ':') != -1 {
				index, err := strconv.Atoi(strings.Split(streamMap, ":")[1])
				if err != nil {
					// index = 0
					log.Fatal(err)
				}
				if index >= 1000 {
					// Replace invalid stream identifier for audio with working one
					filter_complex = fmt.Sprintf("[%d:%d]%s", inputCounter, audioIndex, filter_complex[j+1:])
				}
			}

			args = append(args, arg, filter_complex)
			continue
		}
		// default
		args = append(args, arg)
	}

	log.Println("Custom audio detected. New args:")
	log.Println(args)

	runTranscoder(args)
}
