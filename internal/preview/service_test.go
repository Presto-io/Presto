package preview

import "testing"

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
