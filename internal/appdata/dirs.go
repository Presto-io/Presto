package appdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	legacyMigrationFile = ".presto-legacy-migration.json"
)

type Dirs struct {
	ConfigDir string
	DataDir   string
	CacheDir  string
	LogDir    string
	LegacyDir string
}

type LegacyMigrationResult struct {
	Attempted bool
	Skipped   bool
	Migrated  []string
	Conflicts []string
	Message   string
}

type legacyMigrationMarker struct {
	SchemaVersion int       `json:"schemaVersion"`
	AppID         string    `json:"appId"`
	LegacyDir     string    `json:"legacyDir"`
	DataDir       string    `json:"dataDir"`
	CacheDir      string    `json:"cacheDir"`
	LogDir        string    `json:"logDir"`
	MigratedAt    time.Time `json:"migratedAt"`
	Message       string    `json:"message"`
}

func ResolveDirs() (Dirs, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Dirs{}, fmt.Errorf("get home directory: %w", err)
	}

	configBase, err := userConfigBase(home)
	if err != nil {
		return Dirs{}, err
	}
	dataBase, err := userDataBase(home)
	if err != nil {
		return Dirs{}, err
	}
	cacheBase, err := userCacheBase(home)
	if err != nil {
		return Dirs{}, err
	}
	logBase, err := userLogBase(home)
	if err != nil {
		return Dirs{}, err
	}

	dirs := Dirs{
		ConfigDir: filepath.Join(configBase, AppID),
		DataDir:   filepath.Join(dataBase, AppID),
		CacheDir:  filepath.Join(cacheBase, AppID),
		LogDir:    filepath.Join(logBase, AppID, "logs"),
		LegacyDir: filepath.Join(home, ".presto"),
	}

	if value := os.Getenv("PRESTO_CONFIG_DIR"); value != "" {
		dirs.ConfigDir = value
	}
	if value := os.Getenv("PRESTO_DATA_DIR"); value != "" {
		dirs.DataDir = value
	}
	if value := os.Getenv("PRESTO_CACHE_DIR"); value != "" {
		dirs.CacheDir = value
	}
	if value := os.Getenv("PRESTO_LOG_DIR"); value != "" {
		dirs.LogDir = value
	}
	if value := os.Getenv("PRESTO_LEGACY_DIR"); value != "" {
		dirs.LegacyDir = value
	}

	return dirs, nil
}

func (d Dirs) TemplatesDir() string {
	return filepath.Join(d.DataDir, "templates")
}

func (d Dirs) FontsDir() string {
	return filepath.Join(d.DataDir, "fonts")
}

func (d Dirs) WebViewDataDir() string {
	return filepath.Join(d.CacheDir, "WebView2")
}

func (d Dirs) Ensure() error {
	for _, dir := range []string{d.ConfigDir, d.DataDir, d.CacheDir, d.LogDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}

func MigrateLegacyOnce(dirs Dirs) (LegacyMigrationResult, error) {
	result := LegacyMigrationResult{}
	if samePath(dirs.LegacyDir, dirs.DataDir) {
		result.Skipped = true
		result.Message = "legacy directory is already the active data directory"
		return result, nil
	}

	if err := os.MkdirAll(dirs.ConfigDir, 0700); err != nil {
		return result, err
	}
	markerPath := filepath.Join(dirs.ConfigDir, legacyMigrationFile)
	if _, err := os.Stat(markerPath); err == nil {
		result.Skipped = true
		result.Message = "legacy migration already recorded"
		return result, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}

	result.Attempted = true
	if info, err := os.Stat(dirs.LegacyDir); errors.Is(err, os.ErrNotExist) {
		result.Message = "legacy directory not found"
		return result, writeLegacyMigrationMarker(markerPath, dirs, result.Message)
	} else if err != nil {
		return result, err
	} else if !info.IsDir() {
		result.Message = "legacy path is not a directory"
		return result, writeLegacyMigrationMarker(markerPath, dirs, result.Message)
	}

	if err := dirs.Ensure(); err != nil {
		return result, err
	}

	steps := []struct {
		name string
		src  string
		dst  string
	}{
		{"templates", filepath.Join(dirs.LegacyDir, "templates"), dirs.TemplatesDir()},
		{"fonts", filepath.Join(dirs.LegacyDir, "fonts"), dirs.FontsDir()},
		{"registry-cache.json", filepath.Join(dirs.LegacyDir, "registry-cache.json"), filepath.Join(dirs.CacheDir, "registry-cache.json")},
		{"logs", filepath.Join(dirs.LegacyDir, "logs"), dirs.LogDir},
	}

	for _, step := range steps {
		outcome, err := migratePath(step.src, step.dst)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return result, err
		}
		switch outcome {
		case migrationCopied:
			result.Migrated = append(result.Migrated, step.name)
		case migrationConflict:
			result.Conflicts = append(result.Conflicts, step.name)
		}
	}

	result.Message = "legacy migration completed"
	return result, writeLegacyMigrationMarker(markerPath, dirs, result.Message)
}

func userConfigBase(home string) (string, error) {
	if runtime.GOOS == "windows" {
		if value := os.Getenv("APPDATA"); value != "" {
			return value, nil
		}
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return dir, nil
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support"), nil
	}
	return filepath.Join(home, ".config"), nil
}

func userDataBase(home string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		if value := os.Getenv("LOCALAPPDATA"); value != "" {
			return value, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil
	default:
		if value := os.Getenv("XDG_DATA_HOME"); value != "" {
			return value, nil
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}

func userCacheBase(home string) (string, error) {
	if runtime.GOOS == "windows" {
		if value := os.Getenv("LOCALAPPDATA"); value != "" {
			return value, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	}
	if dir, err := os.UserCacheDir(); err == nil {
		return dir, nil
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches"), nil
	}
	return filepath.Join(home, ".cache"), nil
}

func userLogBase(home string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		if value := os.Getenv("LOCALAPPDATA"); value != "" {
			return value, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Logs"), nil
	default:
		if value := os.Getenv("XDG_STATE_HOME"); value != "" {
			return value, nil
		}
		return filepath.Join(home, ".local", "state"), nil
	}
}

type migrationOutcome int

const (
	migrationNone migrationOutcome = iota
	migrationCopied
	migrationConflict
)

func migratePath(src, dst string) (migrationOutcome, error) {
	info, err := os.Stat(src)
	if err != nil {
		return migrationNone, err
	}
	if info.IsDir() {
		return migrateDir(src, dst)
	}
	return migrateFile(src, dst, info.Mode())
}

func migrateDir(src, dst string) (migrationOutcome, error) {
	outcome := migrationNone
	if err := filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		fileOutcome, err := migrateFile(path, target, info.Mode())
		if err != nil {
			return err
		}
		if fileOutcome == migrationConflict {
			outcome = migrationConflict
		} else if fileOutcome == migrationCopied && outcome != migrationConflict {
			outcome = migrationCopied
		}
		return nil
	}); err != nil {
		return outcome, err
	}
	return outcome, nil
}

func migrateFile(src, dst string, mode os.FileMode) (migrationOutcome, error) {
	if info, err := os.Stat(dst); err == nil {
		if info.IsDir() {
			return migrationConflict, nil
		}
		same, err := sameFileContents(src, dst)
		if err != nil {
			return migrationConflict, nil
		}
		if same {
			return migrationCopied, nil
		}
		return migrationConflict, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return migrationNone, err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return migrationNone, err
	}
	if err := copyFile(src, dst, mode); err != nil {
		return migrationNone, err
	}
	return migrationCopied, nil
}

func sameFileContents(left, right string) (bool, error) {
	leftData, err := os.ReadFile(left)
	if err != nil {
		return false, err
	}
	rightData, err := os.ReadFile(right)
	if err != nil {
		return false, err
	}
	return string(leftData) == string(rightData), nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	return nil
}

func writeLegacyMigrationMarker(path string, dirs Dirs, message string) error {
	marker := legacyMigrationMarker{
		SchemaVersion: 1,
		AppID:         AppID,
		LegacyDir:     dirs.LegacyDir,
		DataDir:       dirs.DataDir,
		CacheDir:      dirs.CacheDir,
		LogDir:        dirs.LogDir,
		MigratedAt:    time.Now(),
		Message:       message,
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileIfChanged(path, data, 0600)
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil {
		left = leftAbs
	}
	if rightErr == nil {
		right = rightAbs
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}
