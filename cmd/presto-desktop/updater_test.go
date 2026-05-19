package main

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func TestParseUpdateAssetURLAllowsExpectedReleaseAsset(t *testing.T) {
	filename := testUpdateFilename()
	rawURL := fmt.Sprintf("https://github.com/Presto-io/Presto/releases/download/v1.2.3/%s", filename)

	repo, tag, gotFilename, err := parseUpdateAssetURL(rawURL)
	if err != nil {
		t.Fatalf("parseUpdateAssetURL returned error: %v", err)
	}
	if repo != "Presto" || tag != "v1.2.3" || gotFilename != filename {
		t.Fatalf("unexpected parse result: repo=%q tag=%q filename=%q", repo, tag, gotFilename)
	}
}

func TestParseUpdateAssetURLRejectsUntrustedSources(t *testing.T) {
	filename := testUpdateFilename()
	tests := []string{
		fmt.Sprintf("http://github.com/Presto-io/Presto/releases/download/v1.2.3/%s", filename),
		fmt.Sprintf("https://example.com/Presto-io/Presto/releases/download/v1.2.3/%s", filename),
		fmt.Sprintf("https://github.com/Presto-io/Other/releases/download/v1.2.3/%s", filename),
		fmt.Sprintf("https://github.com/Presto-io/Presto-Homepage/releases/download/v1.2.3/%s", filename),
		"https://github.com/Presto-io/Presto/releases/tag/v1.2.3",
		"https://github.com/Presto-io/Presto/releases/download/v1.2.3/..",
	}

	for _, rawURL := range tests {
		if _, _, _, err := parseUpdateAssetURL(rawURL); err == nil {
			t.Fatalf("expected %q to be rejected", rawURL)
		}
	}
}

func TestIsExpectedUpdateAsset(t *testing.T) {
	if !isExpectedUpdateAsset(testUpdateFilename()) {
		t.Fatalf("expected platform asset to be accepted")
	}
	if isExpectedUpdateAsset(strings.Replace(testUpdateFilename(), "Presto-1.2.3-", "Presto-1.2.3-portable-", 1)) {
		t.Fatalf("portable update asset should not be accepted by default updater")
	}
	if isExpectedUpdateAsset("Presto-1.2.3-plan9-amd64.tar.gz") {
		t.Fatalf("unexpected platform asset accepted")
	}
	if isExpectedUpdateAsset("notes.txt") {
		t.Fatalf("unexpected non-update asset accepted")
	}
}

func TestIsExpectedUpdateAssetForChannel(t *testing.T) {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && !isExpectedUpdateAssetForChannel("Presto-1.2.3-macOS-arm64.dmg", "slim") {
		t.Fatalf("default macOS arm64 asset should be accepted on darwin/arm64")
	}
	if isExpectedUpdateAssetForChannel("Presto-1.2.3-portable-macOS-arm64.dmg", "slim") {
		t.Fatalf("default updater should reject portable-macOS assets")
	}
	if isExpectedUpdateAssetForChannel("Presto-1.2.3-portable-windows-amd64.exe", "slim") {
		t.Fatalf("default updater should reject portable Windows assets")
	}
	if isExpectedUpdateAssetForChannel(testUpdateFilename(), "portable") {
		t.Fatalf("portable rejects all update assets")
	}
}

func TestParseUpdateChecksums(t *testing.T) {
	filename := testUpdateFilename()
	hash := strings.Repeat("a", 64)
	data := []byte(fmt.Sprintf("%s  %s\n%s  ignored.txt\nnot-a-hash  %s\n", hash, filename, strings.Repeat("b", 64), filename))

	checksums := parseUpdateChecksums(data)
	if checksums[filename] != hash {
		t.Fatalf("expected checksum %q, got %q", hash, checksums[filename])
	}
	if _, ok := checksums["ignored.txt"]; !ok {
		t.Fatalf("expected sha256sum-formatted line to be parsed")
	}
}

func TestPortableCapabilitiesDisableUpdateChecks(t *testing.T) {
	app := &App{
		capabilities: ReleaseCapabilities{
			ReleaseChannel: "portable",
			AppUpdateCheck: false,
		},
	}

	info, err := app.CheckForUpdate()
	if err != nil {
		t.Fatalf("disabled update check returned error: %v", err)
	}
	if info.HasUpdate {
		t.Fatalf("disabled update check should not report an update: %+v", info)
	}
	if info.CurrentVersion != info.LatestVersion {
		t.Fatalf("disabled update check should return current as latest: %+v", info)
	}

	err = app.DownloadAndInstallUpdate("https://github.com/Presto-io/Presto/releases/download/v1.2.3/" + testUpdateFilename())
	if err == nil || !strings.Contains(err.Error(), "online updates are disabled in this release channel") {
		t.Fatalf("expected disabled update install error, got %v", err)
	}
}

func testUpdateFilename() string {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macOS"
	}
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("Presto-1.2.3-%s-%s.dmg", osName, runtime.GOARCH)
	case "windows":
		return fmt.Sprintf("Presto-1.2.3-%s-%s-installer.exe", osName, runtime.GOARCH)
	default:
		return fmt.Sprintf("Presto-1.2.3-%s-%s.tar.gz", osName, runtime.GOARCH)
	}
}
