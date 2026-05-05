package main

import (
	"errors"
	"testing"

	"github.com/mrered/presto/internal/template"
)

type fakeOfficialTemplateInstaller struct {
	existing  map[string]bool
	failNames map[string]bool
	installed []string
	opts      map[string]*template.InstallOpts
}

func (f *fakeOfficialTemplateInstaller) Exists(name string) bool {
	return f.existing[name]
}

func (f *fakeOfficialTemplateInstaller) Install(owner, repo string, opts *template.InstallOpts) error {
	name := repo
	if opts != nil && opts.CdnURL != "" {
		name = opts.CdnURL
	}
	if f.failNames[name] {
		return errors.New("install failed")
	}
	f.installed = append(f.installed, owner+"/"+repo)
	if f.opts == nil {
		f.opts = make(map[string]*template.InstallOpts)
	}
	f.opts[owner+"/"+repo] = opts
	return nil
}

func TestInstallOfficialTemplatesFromRegistryInstallsMissingOnly(t *testing.T) {
	reg := &template.Registry{
		Templates: []template.RegistryEntry{
			{
				Name:  "gongwen",
				Trust: "official",
				Repo:  "Presto-io/presto-official-templates",
				Platforms: map[string]template.RegistryPlatformInfo{
					"linux-amd64": {
						URL:    "https://github.com/Presto-io/presto-official-templates/releases/download/v1/presto-template-gongwen-linux-amd64",
						CdnURL: "https://presto.c-1o.top/templates/gongwen/binaries/presto-template-gongwen-linux-amd64",
						SHA256: "hash-gongwen",
					},
				},
			},
			{
				Name:  "jiaoan-shicao",
				Trust: "official",
				Repo:  "Presto-io/presto-official-templates",
				Platforms: map[string]template.RegistryPlatformInfo{
					"linux-amd64": {
						URL:    "https://github.com/Presto-io/presto-official-templates/releases/download/v1/presto-template-jiaoan-shicao-linux-amd64",
						CdnURL: "https://presto.c-1o.top/templates/jiaoan-shicao/binaries/presto-template-jiaoan-shicao-linux-amd64",
						SHA256: "hash-jiaoan",
					},
				},
			},
			{
				Name:  "community",
				Trust: "community",
				Repo:  "example/community",
			},
		},
	}
	installer := &fakeOfficialTemplateInstaller{
		existing: map[string]bool{"gongwen": true},
	}

	summary := installOfficialTemplatesFromRegistry(installer, reg, "linux-amd64")

	if summary.Found != 2 || summary.Installed != 1 || summary.Skipped != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if len(installer.installed) != 1 || installer.installed[0] != "Presto-io/presto-official-templates" {
		t.Fatalf("installed = %#v", installer.installed)
	}
	opts := installer.opts["Presto-io/presto-official-templates"]
	if opts == nil {
		t.Fatal("expected install opts")
	}
	if opts.CdnURL != "https://presto.c-1o.top/templates/jiaoan-shicao/binaries/presto-template-jiaoan-shicao-linux-amd64" {
		t.Fatalf("CdnURL = %q", opts.CdnURL)
	}
	if opts.ExpectedSHA256 != "hash-jiaoan" {
		t.Fatalf("ExpectedSHA256 = %q", opts.ExpectedSHA256)
	}
}

func TestInstallOfficialTemplatesFromRegistryHandlesUnavailableInputs(t *testing.T) {
	installer := &fakeOfficialTemplateInstaller{}

	summary := installOfficialTemplatesFromRegistry(installer, nil, "linux-amd64")
	if summary != (officialTemplateInstallSummary{}) {
		t.Fatalf("nil registry summary = %+v", summary)
	}

	summary = installOfficialTemplatesFromRegistry(nil, &template.Registry{}, "linux-amd64")
	if summary != (officialTemplateInstallSummary{}) {
		t.Fatalf("nil installer summary = %+v", summary)
	}
}

func TestInstallOfficialTemplatesFromRegistryCountsFailures(t *testing.T) {
	reg := &template.Registry{
		Templates: []template.RegistryEntry{
			{
				Name:  "broken-repo",
				Trust: "official",
				Repo:  "invalid",
			},
			{
				Name:  "missing-platform",
				Trust: "official",
				Repo:  "Presto-io/presto-official-templates",
				Platforms: map[string]template.RegistryPlatformInfo{
					"linux-arm64": {URL: "https://github.com/example/arm64"},
				},
			},
		},
	}

	summary := installOfficialTemplatesFromRegistry(&fakeOfficialTemplateInstaller{}, reg, "linux-amd64")
	if summary.Found != 2 || summary.Failed != 2 || summary.Installed != 0 {
		t.Fatalf("summary = %+v", summary)
	}
}
