//go:build linux

package appdata

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func markPlatformGenerated(paths []string, marker Marker) {
	for _, path := range paths {
		_ = unix.Setxattr(path, "user.presto.generated_by", []byte(AppID), 0)
		_ = unix.Setxattr(path, "user.xdg.tags", []byte("presto;generated;app-data"), 0)
		_ = unix.Setxattr(path, "user.xdg.comment", []byte(marker.SafeToDelete), 0)
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	supportDir := filepath.Join(dataHome, AppID)
	if err := os.MkdirAll(supportDir, 0700); err != nil {
		return
	}

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')

	locationPath := filepath.Join(supportDir, "generated-data-location.json")
	_ = writeFileIfChanged(locationPath, data, 0600)
	_ = unix.Setxattr(supportDir, "user.presto.generated_by", []byte(AppID), 0)
	_ = unix.Setxattr(locationPath, "user.presto.generated_by", []byte(AppID), 0)
}
