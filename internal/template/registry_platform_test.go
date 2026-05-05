package template

import "testing"

func TestRegistryEntryDownloadURLForPlatformPrefersCDN(t *testing.T) {
	entry := RegistryEntry{
		Name:  "jiaoan-shicao",
		Trust: "official",
		Platforms: map[string]RegistryPlatformInfo{
			"windows-amd64": {
				URL:    "https://github.com/Presto-io/presto-official-templates/releases/download/v1.1.1/presto-template-jiaoan-shicao-windows-amd64.exe",
				CdnURL: "https://presto.c-1o.top/templates/jiaoan-shicao/binaries/presto-template-jiaoan-shicao-windows-amd64.exe",
				SHA256: "abc123",
			},
		},
	}

	got, ok := entry.DownloadURLForPlatform("windows-amd64")
	if !ok {
		t.Fatal("expected windows-amd64 platform to be available")
	}

	want := "https://presto.c-1o.top/templates/jiaoan-shicao/binaries/presto-template-jiaoan-shicao-windows-amd64.exe"
	if got != want {
		t.Fatalf("DownloadURLForPlatform() = %q, want %q", got, want)
	}
}

func TestRegistryEntryDownloadURLForPlatformFallsBackToURL(t *testing.T) {
	entry := RegistryEntry{
		Name: "gongwen",
		Platforms: map[string]RegistryPlatformInfo{
			"linux-amd64": {
				URL:    "https://github.com/Presto-io/presto-official-templates/releases/download/v1.1.1/presto-template-gongwen-linux-amd64",
				SHA256: "abc123",
			},
		},
	}

	got, ok := entry.DownloadURLForPlatform("linux-amd64")
	if !ok {
		t.Fatal("expected linux-amd64 platform to be available")
	}

	want := "https://github.com/Presto-io/presto-official-templates/releases/download/v1.1.1/presto-template-gongwen-linux-amd64"
	if got != want {
		t.Fatalf("DownloadURLForPlatform() = %q, want %q", got, want)
	}
}

func TestRegistryEntryInstallOptsForPlatform(t *testing.T) {
	entry := RegistryEntry{
		Trust: "official",
		Platforms: map[string]RegistryPlatformInfo{
			"linux-arm64": {
				URL:    "https://github.com/example/linux-arm64",
				CdnURL: "https://presto.c-1o.top/templates/example/binaries/linux-arm64",
				SHA256: "abc123",
			},
		},
	}

	opts, ok := entry.InstallOptsForPlatform("linux-arm64")
	if !ok {
		t.Fatal("expected linux-arm64 install opts")
	}
	if opts.DownloadURL != "https://github.com/example/linux-arm64" {
		t.Fatalf("DownloadURL = %q", opts.DownloadURL)
	}
	if opts.CdnURL != "https://presto.c-1o.top/templates/example/binaries/linux-arm64" {
		t.Fatalf("CdnURL = %q", opts.CdnURL)
	}
	if opts.ExpectedSHA256 != "abc123" {
		t.Fatalf("ExpectedSHA256 = %q", opts.ExpectedSHA256)
	}
	if opts.Trust != "official" {
		t.Fatalf("Trust = %q", opts.Trust)
	}
}

func TestDownloadCandidatesPreferCDN(t *testing.T) {
	candidates := downloadCandidates(&InstallOpts{
		DownloadURL: "https://github.com/example/template",
		CdnURL:      "https://presto.c-1o.top/templates/example",
	})

	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}
	if candidates[0].source != "cdn" || candidates[0].url != "https://presto.c-1o.top/templates/example" {
		t.Fatalf("first candidate = %#v, want CDN first", candidates[0])
	}
	if candidates[0].filename != "example" {
		t.Fatalf("first candidate filename = %q", candidates[0].filename)
	}
	if candidates[1].source != "github" || candidates[1].url != "https://github.com/example/template" {
		t.Fatalf("second candidate = %#v, want GitHub fallback", candidates[1])
	}
	if candidates[1].filename != "template" {
		t.Fatalf("second candidate filename = %q", candidates[1].filename)
	}
}
