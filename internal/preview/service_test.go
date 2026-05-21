package preview

import (
	"errors"
	"testing"
)

func TestNextDocumentVersion(t *testing.T) {
	svc := NewService()
	identity := DocumentIdentity{TemplateID: "gongwen", WorkDir: "/tmp/doc", DocumentKey: "a.md", MainTypstPath: "/tmp/doc/main.typ"}

	v1, _ := svc.NextDocumentVersion(identity)
	v2, _ := svc.NextDocumentVersion(identity)

	if v1 != 1 || v2 != 2 {
		t.Fatalf("versions = %d, %d; want 1, 2", v1, v2)
	}
}

func TestDocumentIdentityRestart(t *testing.T) {
	base := DocumentIdentity{TemplateID: "gongwen", WorkDir: "/tmp/doc", DocumentKey: "a.md", MainTypstPath: "/tmp/doc/main.typ"}
	tests := []struct {
		name        string
		next        DocumentIdentity
		wantRestart bool
	}{
		{name: "same identity", next: base, wantRestart: false},
		{name: "template changed", next: DocumentIdentity{TemplateID: "jiaoan", WorkDir: base.WorkDir, DocumentKey: base.DocumentKey, MainTypstPath: base.MainTypstPath}, wantRestart: true},
		{name: "work dir changed", next: DocumentIdentity{TemplateID: base.TemplateID, WorkDir: "/tmp/other", DocumentKey: base.DocumentKey, MainTypstPath: base.MainTypstPath}, wantRestart: true},
		{name: "document key changed", next: DocumentIdentity{TemplateID: base.TemplateID, WorkDir: base.WorkDir, DocumentKey: "b.md", MainTypstPath: base.MainTypstPath}, wantRestart: true},
		{name: "main typst path changed", next: DocumentIdentity{TemplateID: base.TemplateID, WorkDir: base.WorkDir, DocumentKey: base.DocumentKey, MainTypstPath: "/tmp/doc/other.typ"}, wantRestart: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			svc.NextDocumentVersion(base)
			_, restart := svc.NextDocumentVersion(tt.next)
			if restart != tt.wantRestart {
				t.Fatalf("restartSession = %v, want %v", restart, tt.wantRestart)
			}
		})
	}
}

func TestApplyFallbackRejectsStaleVersion(t *testing.T) {
	svc := NewService()
	current, _ := svc.NextDocumentVersion(DocumentIdentity{TemplateID: "gongwen"})

	event, ok := svc.ApplyFallback(current-1, []Page{{Index: 1, SVG: "<svg/>", Hash: "old"}})

	if ok {
		t.Fatal("ApplyFallback returned ok=true for stale version")
	}
	if event.Kind != EventStaleIgnored {
		t.Fatalf("event.Kind = %q, want %q", event.Kind, EventStaleIgnored)
	}
	if len(svc.LastFallback()) != 0 {
		t.Fatal("stale fallback updated LastFallback")
	}
}

func TestApplySessionEventRejectsStaleSession(t *testing.T) {
	svc := NewService()
	svc.NextDocumentVersion(DocumentIdentity{TemplateID: "gongwen"})
	svc.StartSession(DocumentIdentity{TemplateID: "gongwen"})

	event, ok := svc.ApplySessionEvent("old-session", EventReady, "http://127.0.0.1:1234", nil)

	if ok {
		t.Fatal("ApplySessionEvent returned ok=true for stale session")
	}
	if event.Kind != EventStaleIgnored {
		t.Fatalf("event.Kind = %q, want %q", event.Kind, EventStaleIgnored)
	}
}

func TestReadyEventSetsEmbeddedMode(t *testing.T) {
	svc := NewService()
	svc.NextDocumentVersion(DocumentIdentity{TemplateID: "gongwen"})
	svc.StartSession(DocumentIdentity{TemplateID: "gongwen"})

	event, ok := svc.ApplySessionEvent(svc.CurrentSessionID(), EventReady, "http://127.0.0.1:1234", nil)

	if !ok {
		t.Fatal("ApplySessionEvent returned ok=false for current session")
	}
	if event.Mode != ModeEmbedded {
		t.Fatalf("event.Mode = %q, want %q", event.Mode, ModeEmbedded)
	}
	if event.DataPlaneURL != "http://127.0.0.1:1234" {
		t.Fatalf("event.DataPlaneURL = %q", event.DataPlaneURL)
	}
}

func TestTeardownForcesNextSameIdentityUpdateToRestart(t *testing.T) {
	svc := NewService()
	identity := DocumentIdentity{TemplateID: "gongwen", WorkDir: "/tmp/doc", DocumentKey: "a.md", MainTypstPath: "/tmp/doc/main.typ"}
	first := svc.BeginUpdate(identity)
	svc.StartSession(identity)
	if event, ok := svc.ApplySessionEvent(svc.CurrentSessionID(), EventReady, "http://127.0.0.1:1234", nil); !ok || event.Mode != ModeEmbedded {
		t.Fatalf("ready event = (%#v, %v), want embedded", event, ok)
	}

	if event, ok := svc.ApplySessionEvent(svc.CurrentSessionID(), EventTeardown, "", nil); !ok || event.Mode != ModeFallback {
		t.Fatalf("teardown event = (%#v, %v), want fallback", event, ok)
	}

	next := svc.BeginUpdate(identity)
	if next.Version != first.Version+1 {
		t.Fatalf("next version = %d, want %d", next.Version, first.Version+1)
	}
	if !next.RestartSession {
		t.Fatal("BeginUpdate after teardown should restart even for the same identity")
	}
}

func TestRecoverDoesNotStealPreviewUntilNextEdit(t *testing.T) {
	svc := NewService()
	identity := DocumentIdentity{TemplateID: "gongwen", WorkDir: "/tmp/doc", DocumentKey: "a.md", MainTypstPath: "/tmp/doc/main.typ"}
	update := svc.BeginUpdate(identity)
	svc.StartSession(identity)
	if event, ok := svc.ApplySessionEvent(svc.CurrentSessionID(), EventReady, "http://127.0.0.1:1234", nil); !ok || event.Mode != ModeEmbedded {
		t.Fatalf("ready event = (%#v, %v), want embedded", event, ok)
	}
	if event, ok := svc.ApplyFallback(update.Version, []Page{{Index: 1, SVG: "<svg>fallback</svg>", Hash: "fallback"}}); !ok || event.Mode != ModeFallback {
		t.Fatalf("fallback event = (%#v, %v), want fallback", event, ok)
	}

	recoverEvent, ok := svc.ApplySessionEvent(svc.CurrentSessionID(), EventRecover, "http://127.0.0.1:1234", nil)
	if !ok {
		t.Fatal("recover event for current session returned ok=false")
	}
	if recoverEvent.Mode != ModeFallback {
		t.Fatalf("recoverEvent.Mode = %q, want %q", recoverEvent.Mode, ModeFallback)
	}
	if got := svc.LastFallback(); len(got) != 1 || got[0].Hash != "fallback" {
		t.Fatalf("LastFallback = %#v, want retained fallback", got)
	}

	next := svc.BeginUpdate(identity)
	if next.Version != update.Version+1 {
		t.Fatalf("next version = %d, want %d", next.Version, update.Version+1)
	}
	readyEvent, ok := svc.ApplySessionEvent(svc.CurrentSessionID(), EventReady, "http://127.0.0.1:1234", nil)
	if !ok {
		t.Fatal("ready event after next edit returned ok=false")
	}
	if readyEvent.Mode != ModeEmbedded {
		t.Fatalf("readyEvent.Mode = %q, want %q", readyEvent.Mode, ModeEmbedded)
	}
}

func TestRapidEditsEndOnLatestDocumentVersion(t *testing.T) {
	svc := NewService()
	identity := DocumentIdentity{TemplateID: "gongwen", WorkDir: "/tmp/doc", DocumentKey: "a.md", MainTypstPath: "/tmp/doc/main.typ"}

	var latest int64
	for i := 1; i <= 8; i++ {
		latest = svc.BeginUpdate(identity).Version
	}

	for version := int64(1); version < latest; version++ {
		if event, ok := svc.ApplyFallback(version, []Page{{Index: 1, SVG: "<svg>stale</svg>", Hash: "stale"}}); ok || event.Kind != EventStaleIgnored {
			t.Fatalf("version %d fallback = (%#v, %v), want stale ignored", version, event, ok)
		}
	}

	pages := []Page{{Index: 1, SVG: "<svg>latest</svg>", Hash: "latest"}}
	if event, ok := svc.ApplyFallback(latest, pages); !ok || event.Kind != EventFallback {
		t.Fatalf("latest fallback = (%#v, %v), want applied fallback", event, ok)
	}
	if got := svc.LastFallback(); len(got) != 1 || got[0].Hash != "latest" {
		t.Fatalf("LastFallback = %#v, want latest fallback", got)
	}
}

func TestErrorRetainsLastFallback(t *testing.T) {
	svc := NewService()
	version, _ := svc.NextDocumentVersion(DocumentIdentity{TemplateID: "gongwen"})
	pages := []Page{{Index: 1, SVG: "<svg/>", Hash: "hash"}}
	if _, ok := svc.ApplyFallback(version, pages); !ok {
		t.Fatal("ApplyFallback current version failed")
	}

	if _, ok := svc.ApplySessionEvent(svc.CurrentSessionID(), EventError, "", &ErrorInfo{Code: "boom", Message: "failed", Recoverable: true}); !ok {
		t.Fatal("ApplySessionEvent current session failed")
	}

	got := svc.LastFallback()
	if len(got) != 1 || got[0].SVG != pages[0].SVG || got[0].Hash != pages[0].Hash {
		t.Fatalf("LastFallback = %#v, want %#v", got, pages)
	}
}

func TestBeginUpdate(t *testing.T) {
	svc := NewService()
	identity := DocumentIdentity{TemplateID: "gongwen", WorkDir: "/tmp/doc", DocumentKey: "a.md", MainTypstPath: "/tmp/doc/main.typ"}

	first := svc.BeginUpdate(identity)
	second := svc.BeginUpdate(identity)

	if first.Version != 1 || second.Version != 2 {
		t.Fatalf("versions = %d, %d; want 1, 2", first.Version, second.Version)
	}
	if !first.RestartSession {
		t.Fatal("first BeginUpdate should restart session for new identity")
	}
	if second.RestartSession {
		t.Fatal("second BeginUpdate should reuse session for identical identity")
	}
	if len(first.Events) != 1 || first.Events[0].Kind != EventStatus {
		t.Fatalf("first.Events = %#v, want one status event", first.Events)
	}
	if got := first.Events[0].Metadata["phase"]; got != "convert" {
		t.Fatalf("metadata.phase = %#v, want convert", got)
	}
}

func TestConversionFailed(t *testing.T) {
	svc := NewService()
	result := svc.BeginUpdate(DocumentIdentity{TemplateID: "gongwen"})
	pages := []Page{{Index: 1, SVG: "<svg/>", Hash: "hash"}}
	svc.ApplyFallback(result.Version, pages)

	event := svc.ConversionFailed(result.Version, errors.New("bad markdown"))

	if event.Kind != EventError {
		t.Fatalf("event.Kind = %q, want %q", event.Kind, EventError)
	}
	if event.Error == nil || event.Error.Code != "conversion_failed" || !event.Error.Recoverable {
		t.Fatalf("event.Error = %#v", event.Error)
	}
	if event.DocumentVersion != result.Version {
		t.Fatalf("event.DocumentVersion = %d, want %d", event.DocumentVersion, result.Version)
	}
	if got := svc.LastFallback(); len(got) != 1 || got[0].SVG != pages[0].SVG {
		t.Fatalf("LastFallback = %#v, want retained fallback", got)
	}
}

func TestFallbackFailed(t *testing.T) {
	svc := NewService()
	result := svc.BeginUpdate(DocumentIdentity{TemplateID: "gongwen"})
	pages := []Page{{Index: 1, SVG: "<svg/>", Hash: "hash"}}
	svc.ApplyFallback(result.Version, pages)

	event := svc.FallbackFailed(result.Version, errors.New("typst failed"))

	if event.Kind != EventError {
		t.Fatalf("event.Kind = %q, want %q", event.Kind, EventError)
	}
	if event.Error == nil || event.Error.Code != "fallback_compile_failed" || !event.Error.Recoverable {
		t.Fatalf("event.Error = %#v", event.Error)
	}
	if got := svc.LastFallback(); len(got) != 1 || got[0].SVG != pages[0].SVG {
		t.Fatalf("LastFallback = %#v, want retained fallback", got)
	}
}

func TestTinymistUnavailableUsesUserFacingMessage(t *testing.T) {
	svc := NewService()
	svc.BeginUpdate(DocumentIdentity{TemplateID: "gongwen"})

	event := svc.TinymistUnavailable("tinymist not found")

	if event.Kind != EventFallback {
		t.Fatalf("event.Kind = %q, want %q", event.Kind, EventFallback)
	}
	if event.Mode != ModeFallback {
		t.Fatalf("event.Mode = %q, want %q", event.Mode, ModeFallback)
	}
	if event.Error == nil {
		t.Fatal("event.Error is nil")
	}
	if event.Error.Code != "tinymist_unavailable" {
		t.Fatalf("event.Error.Code = %q", event.Error.Code)
	}
	if event.Error.Message != "兼容预览" {
		t.Fatalf("event.Error.Message = %q, want 兼容预览", event.Error.Message)
	}
	if event.Error.Detail != "tinymist not found" {
		t.Fatalf("event.Error.Detail = %q", event.Error.Detail)
	}
	if !event.Error.Recoverable {
		t.Fatal("event.Error.Recoverable = false, want true")
	}
}
