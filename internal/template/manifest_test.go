package template

import "testing"

func TestParseManifest(t *testing.T) {
	data := []byte(`{
		"name": "gongwen",
		"displayName": "中国党政机关公文格式",
		"version": "1.0.0",
		"author": "mrered"
	}`)

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "gongwen" {
		t.Errorf("got name %q, want %q", m.Name, "gongwen")
	}
	if m.DisplayName != "中国党政机关公文格式" {
		t.Errorf("got displayName %q, want %q", m.DisplayName, "中国党政机关公文格式")
	}
}
