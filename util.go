package util

import (
	"os"
	"runtime"
	"fmt"
	"path/filepath"
)

func GetDBPath() string {
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
	p = filepath.Dir(ex) + "/com.plexapp.plugins.library.db"
    return p
}

func GetDSN() string {
    path := GetDBPath()
    fmt.Printf("Database path: %s\n", path)
    return fmt.Sprintf("file:%s?mode=rw", path)
}

func GetLogPath() string {
    return os.TempDir() + "/plex-custom-audio.log"
}
