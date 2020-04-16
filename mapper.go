package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/saoneth/goav/avcodec"
	"github.com/saoneth/goav/avformat"
	"github.com/saoneth/goav/avutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func encodeUriParams(m map[string]string) string {
	values := url.Values{}
	for k, v := range m {
		values.Set(k, v)
	}
	return values.Encode()
}

func isAudioFile(file string) bool {
	// aac,ac3,alac,dts,flac,matroska,mp2,mp3,ogg,wav
	return strings.HasSuffix(file, ".aac") || strings.HasSuffix(file, ".ac3") || strings.HasSuffix(file, ".alac") || strings.HasSuffix(file, ".dtshd") || strings.HasSuffix(file, ".dts") || strings.HasSuffix(file, ".thd") || strings.HasSuffix(file, ".flac") || strings.HasSuffix(file, ".mka") || strings.HasSuffix(file, ".mp2") || strings.HasSuffix(file, ".mp3") || strings.HasSuffix(file, ".ogg") || strings.HasSuffix(file, ".wav")
}

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

func main() {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=rw", getDBPath()))
	if err != nil {
		log.Fatal(err, "You can add support for your configuration by creating link to com.plexapp.plugins.library.db in the same directory as this application")
	}
	defer db.Close()

	args := os.Args
	skip_cleaning := false
	fmt.Println(len(os.Args))
	if len(os.Args) > 1 && os.Args[1] == "-s" {
		skip_cleaning = true
		args = args[1:]
	}

	if !skip_cleaning {
		// To-do: Add our own datatabase, so custom audio metadata isn't completly lost when plex analyses original file
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("DELETE FROM `media_streams` WHERE `id` = ?")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		fmt.Println("Cleaning up old audio records")
		rows, err := db.Query("SELECT `id`, `url` FROM `media_streams` WHERE `index` >= 1000 AND `url` != ''")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			var streamId int
			var streamUrl string
			err = rows.Scan(&streamId, &streamUrl)
			if err != nil {
				log.Fatal(err)
			}
			path := string(streamUrl[7:])
			fmt.Println("Scanning:", path)
			if _, err := os.Stat(path); err == nil {
				continue
			}
			fmt.Println("Custom audio file deleted:", path)
			_, err = stmt.Exec(streamId)
			if err != nil {
				log.Fatal(err)
			}
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		tx.Commit()
	}

	pathQuery := ""
	if len(args) > 1 {
		fmt.Println("Selected directories for scanning:")
		for i := 1; i < len(args); i++ {
			fmt.Printf(" - %s\n", args[i])
			pathQuery = pathQuery + " OR `file` LIKE " + strconv.Quote(args[i] + "%")
		}
		pathQuery = " AND (" + pathQuery[4:] + ")"
	}

	// To-do: Make option to scan individual files
	fmt.Println("Scanning library for new audio files...")
	rows, err := db.Query("SELECT `id`, `media_item_id`, `file` FROM `media_parts` WHERE `file` != \"\" " + pathQuery + " AND (`file` NOT LIKE \"%/Trailers/%\") ORDER BY `file` ASC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Change container to some gibberish, so it always gets remuxed
	force_transcoding_stmt, err := db.Prepare("UPDATE `media_items` SET `container` = \"force_transcode\" WHERE `id` = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer force_transcoding_stmt.Close()

	last_index_stmt, err := db.Prepare("SELECT MAX(`index`) FROM `media_streams` WHERE `media_item_id` = ? LIMIT 1")
	if err != nil {
		log.Fatal(err)
	}
	defer last_index_stmt.Close()

	check_stmt, err := db.Prepare("SELECT `id` FROM `media_streams` WHERE `url` = ? LIMIT 1")
	if err != nil {
		log.Fatal(err)
	}
	defer check_stmt.Close()

	ins_stmt, err := db.Prepare("INSERT INTO `media_streams` (`id`, `stream_type_id`, `media_item_id`, `url`, `codec`, `language`, `created_at`, `updated_at`, `index`, `media_part_id`, `channels`, `bitrate`, `url_index`, `default`, `forced`, `extra_data`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer ins_stmt.Close()

	upd_stmt, err := db.Prepare("UPDATE `metadata_items` SET `added_at`=? WHERE `id` = (SELECT `metadata_item_id` FROM `media_items` WHERE `id` = ? LIMIT 1)")
	if err != nil {
		log.Fatal(err)
	}
	defer upd_stmt.Close()

	avformat.AvRegisterAll()

	for rows.Next() {
		var media_part_id int
		var media_item_id int
		var file string
		err = rows.Scan(&media_part_id, &media_item_id, &file)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Processing:", file)

		ext := filepath.Ext(file)
		if ext != ".mp4" && ext != ".mkv" && ext != ".avi" {
			fmt.Println(" ! file is not a video")
			continue
		}

		var last_index int
		_ = last_index_stmt.QueryRow(media_item_id).Scan(&last_index)

		if last_index == 0 {
			fmt.Println(" ! file is not yet analysed")
			continue
		}
		if last_index < 1000 {
			last_index = 1000
		}

		file_dir := filepath.Dir(file) + "/"
		filename := filepath.Base(file)
		base_filename := filename[0:len(filename) - len(ext)] + "."
		err = filepath.Walk(file_dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				return err
			}
			if info.IsDir() {
				return nil //filepath.SkipDir
			}
			name := info.Name()
			if name == filename || !strings.HasPrefix(name, base_filename) || !isAudioFile(name) {
				return nil
			}
			ext := filepath.Ext(name)
			fmt.Printf(" - found audio file: %s\n", path)

			// File needs to be in format: BASENAME.LANG.mka
			if len(base_filename) > len(name)-len(ext) {
				return nil
			}
			s := strings.Split(name[len(base_filename):len(name)-len(ext)], ".")
			fTitle := ""
			fLanguage := s[0]
			if len(fLanguage) != 3 {
				fmt.Printf("	! invalid language: %s\n", fLanguage)
				return nil
			}

			if len(s) > 1 {
				fTitle = strings.Join(s[1:], ".")
			}
			if strings.HasPrefix(fTitle, "track-") {
				fTitle = ""
			}

			fileUrl := "file://" + path

			var res int
			_ = check_stmt.QueryRow(fileUrl).Scan(&res)
			if res > 0 {
				fmt.Println("	@ file already in database")
				return nil
			}

			_, err = force_transcoding_stmt.Exec(media_item_id)
			if err != nil {
				log.Fatal(err)
			}

			pFormatCtx := avformat.AvformatAllocContext()
			if avformat.AvformatOpenInput(&pFormatCtx, path, nil, nil) != 0 {
				log.Println("	! avformat failed to open file")
				return nil
			}

			if pFormatCtx.AvformatFindStreamInfo(nil) < 0 {
				log.Println("	! couldn't find stream information")

				// Close input file and free context
				pFormatCtx.AvformatCloseInput()
				return nil
			}

			streams := int(pFormatCtx.NbStreams())
			multiple_streams := streams > 1
			for i := 0; i < streams; i++ {
				pStream := pFormatCtx.Streams()[i]
				pCodecParametersCtx := pStream.CodecParameters()
				if pCodecParametersCtx.AvCodecGetType() != avformat.AVMEDIA_TYPE_AUDIO {
					continue
				}
				// Get a pointer to the codec context for the video stream
				pCodecCtxOrg := pStream.Codec()

				codecId := avcodec.CodecId(pCodecCtxOrg.GetCodecId())
				pCodec := avcodec.AvcodecFindDecoder(codecId)
				if pCodec == nil {
					fmt.Printf("	! unsupported codec in stream: %d\n", i)
					continue
				}
				// Copy context
				pCodecCtx := pCodec.AvcodecAllocContext3()
				if pCodecCtx.AvcodecCopyContext((*avcodec.Context)(unsafe.Pointer(pCodecCtxOrg))) != 0 {
					fmt.Println("	! couldn't copy codec context")
					continue
				}

				// Open codec
				if pCodecCtx.AvcodecOpen2(pCodec, nil) < 0 {
					fmt.Println("	! could not open codec")
					continue
				}

				codec := avcodec.AvcodecGetName(codecId)
				if codec == "dts" {
					codec = "dca"
				}

				extra_data := make(map[string]string)

				switch pCodecCtx.ChannelLayout() {
					case avutil.AV_CH_LAYOUT_MONO:
						extra_data["ma:audioChannelLayout"] = "mono"
					case avutil.AV_CH_LAYOUT_STEREO:
						extra_data["ma:audioChannelLayout"] = "stereo"
					case avutil.AV_CH_LAYOUT_2POINT1:
						extra_data["ma:audioChannelLayout"] = "2.1"
					case avutil.AV_CH_LAYOUT_SURROUND:
						extra_data["ma:audioChannelLayout"] = "3.0"
					case avutil.AV_CH_LAYOUT_2_1:
						extra_data["ma:audioChannelLayout"] = "3.0(back)"
					case avutil.AV_CH_LAYOUT_4POINT0:
						extra_data["ma:audioChannelLayout"] = "4.0"
					case avutil.AV_CH_LAYOUT_QUAD:
						extra_data["ma:audioChannelLayout"] = "quad"
					case avutil.AV_CH_LAYOUT_2_2:
						extra_data["ma:audioChannelLayout"] = "quad(side)"
					case avutil.AV_CH_LAYOUT_3POINT1:
						extra_data["ma:audioChannelLayout"] = "3.1"
					case avutil.AV_CH_LAYOUT_5POINT0_BACK:
						extra_data["ma:audioChannelLayout"] = "5.0"
					case avutil.AV_CH_LAYOUT_5POINT0:
						extra_data["ma:audioChannelLayout"] = "5.0(side)"
					case avutil.AV_CH_LAYOUT_4POINT1:
						extra_data["ma:audioChannelLayout"] = "4.1"
					case avutil.AV_CH_LAYOUT_5POINT1_BACK:
						extra_data["ma:audioChannelLayout"] = "5.1"
					case avutil.AV_CH_LAYOUT_5POINT1:
						extra_data["ma:audioChannelLayout"] = "5.1(side)"
					case avutil.AV_CH_LAYOUT_6POINT0:
						extra_data["ma:audioChannelLayout"] = "6.0"
					case avutil.AV_CH_LAYOUT_6POINT0_FRONT:
						extra_data["ma:audioChannelLayout"] = "6.0(front)"
					case avutil.AV_CH_LAYOUT_HEXAGONAL:
						extra_data["ma:audioChannelLayout"] = "hexagonal"
					case avutil.AV_CH_LAYOUT_6POINT1:
						extra_data["ma:audioChannelLayout"] = "6.1"
					case avutil.AV_CH_LAYOUT_6POINT1_BACK:
						extra_data["ma:audioChannelLayout"] = "6.1(back)"
					case avutil.AV_CH_LAYOUT_6POINT1_FRONT:
						extra_data["ma:audioChannelLayout"] = "6.1(front)"
					case avutil.AV_CH_LAYOUT_7POINT0:
						extra_data["ma:audioChannelLayout"] = "7.0"
					case avutil.AV_CH_LAYOUT_7POINT0_FRONT:
						extra_data["ma:audioChannelLayout"] = "7.0(front)"
					case avutil.AV_CH_LAYOUT_7POINT1:
						extra_data["ma:audioChannelLayout"] = "7.1"
					case avutil.AV_CH_LAYOUT_7POINT1_WIDE_BACK:
						extra_data["ma:audioChannelLayout"] = "7.1(wide)"
					case avutil.AV_CH_LAYOUT_7POINT1_WIDE:
						extra_data["ma:audioChannelLayout"] = "7.1(wide-side)"
					case avutil.AV_CH_LAYOUT_OCTAGONAL:
						extra_data["ma:audioChannelLayout"] = "octagonal"
					case avutil.AV_CH_LAYOUT_HEXADECAGONAL:
						extra_data["ma:audioChannelLayout"] = "hexadecagonal"
					case avutil.AV_CH_LAYOUT_STEREO_DOWNMIX:
						extra_data["ma:audioChannelLayout"] = "downmix"
					default:
						fmt.Println("	# could not identify channel layout.")
				}

				extra_data["ma:samplingRate"] = strconv.Itoa(pCodecCtx.SampleRate())

				if codec == "dca" {
					extra_data["ma:bitDepth"] = strconv.Itoa(pCodecCtx.BitsPerRawSample())
					if extra_data["ma:bitDepth"] == "0" {
						extra_data["ma:bitDepth"] = "24"
					}
				}
				profile := pCodec.AvGetProfileName(pCodecCtx.Profile())
				switch profile {
					case "":
						// ignore
					case "DTS":
						extra_data["ma:profile"] = "dts"
					case "DTS-HD MA":
						extra_data["ma:profile"] = "ma"
					default:
						fmt.Printf("Unknown profile: %s\n", profile)
						extra_data["ma:profile"] = profile
				}

				var de *avutil.DictionaryEntry

				language := ""
				de = pStream.Metadata().AvDictGet("language", nil, 0)
				if de != nil {
					language = de.Value()
					fmt.Printf("	- language: %s\n", language)
				}

				title := ""
				de = pStream.Metadata().AvDictGet("title", nil, 0)
				if de != nil {
					title = de.Value()
					fmt.Printf("	- tile: %s\n", title)
				}

				if fLanguage != language && (language == "" || multiple_streams) {
					language = fLanguage
					fmt.Printf("	- forcing language: %s\n", language)
				}

				if fTitle != "" && (title == "" || multiple_streams) {
					title = fTitle
					fmt.Printf("	- forcing title: %s\n", title)
				}

				extra_data_encoded := encodeUriParams(extra_data)

				bitrate := pCodecCtx.BitRate()

				fmt.Println("	@ bitrate:", bitrate)

				date := time.Now().Format("2006-01-02 15:04:05")

				fmt.Println("	@", nil, 2, media_item_id, fileUrl, codec, language, date, date, last_index, media_part_id, pCodecCtx.Channels(), bitrate, pStream.Index(), 0, 0, extra_data_encoded)
				_, err := ins_stmt.Exec(nil, 2, media_item_id, "file://" + path, codec, language, date, date, last_index, media_part_id, pCodecCtx.Channels(), bitrate, pStream.Index(), 0, 0, extra_data_encoded)
				if err != nil {
					log.Fatal(err)
				}

				_, err = upd_stmt.Exec(date, media_item_id)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("	- added successfully")
				last_index++

				pCodecCtx.AvcodecClose()
			}

			// Close container
			pFormatCtx.AvformatCloseInput()

			return nil
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", file_dir, err)
			return
		}
	}
}
