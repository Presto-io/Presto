package appdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMarkGeneratedWritesCleanupMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "templates"), 0700); err != nil {
		t.Fatal(err)
	}

	if err := MarkGenerated(dir); err != nil {
		t.Fatalf("MarkGenerated returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, markerFile))
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}

	var marker Marker
	if err := json.Unmarshal(data, &marker); err != nil {
		t.Fatalf("parse marker: %v", err)
	}
	if marker.AppID != AppID {
		t.Fatalf("AppID = %q, want %q", marker.AppID, AppID)
	}
	if marker.DataDir != dir {
		t.Fatalf("DataDir = %q, want %q", marker.DataDir, dir)
	}
	if !marker.Removable {
		t.Fatal("marker should identify the data directory as removable")
	}

	readme, err := os.ReadFile(filepath.Join(dir, readmeFile))
	if err != nil {
		t.Fatalf("read cleanup readme: %v", err)
	}
	if len(readme) == 0 {
		t.Fatal("cleanup readme should not be empty")
	}
}
