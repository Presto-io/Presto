//go:build darwin

package appdata

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func markPlatformGenerated(paths []string, marker Marker) {
	for _, path := range paths {
		_ = unix.Setxattr(path, "com.mrered.presto.generated-by", []byte(AppID), 0)
		_ = unix.Setxattr(path, "com.mrered.presto.cleanup-hint", []byte(marker.SafeToDelete), 0)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	supportDir := filepath.Join(home, "Library", "Application Support", AppID)
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
	_ = unix.Setxattr(supportDir, "com.mrered.presto.generated-by", []byte(AppID), 0)
	_ = unix.Setxattr(locationPath, "com.mrered.presto.generated-by", []byte(AppID), 0)
}
