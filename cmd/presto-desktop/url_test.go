package main

import "testing"

func TestParsePrestoTemplateURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{
			name: "resource template open URL",
			raw:  "presto://open?resource=template&id=gongwen",
			want: "gongwen",
			ok:   true,
		},
		{
			name: "hyphenated template id",
			raw:  "presto://open?resource=template&id=jiaoan-shicao",
			want: "jiaoan-shicao",
			ok:   true,
		},
		{
			name: "legacy install URL remains supported",
			raw:  "presto://install/gongwen",
			want: "gongwen",
			ok:   true,
		},
		{
			name: "reject website URL as protocol payload",
			raw:  "presto://open?resource=template&id=https%3A%2F%2Fpresto.mre.red%2Ftemplates%2Fgongwen",
			ok:   false,
		},
		{
			name: "reject non-template resource",
			raw:  "presto://open?resource=skill&id=gongwen",
			ok:   false,
		},
		{
			name: "reject extra open query parameters",
			raw:  "presto://open?resource=template&id=gongwen&url=https%3A%2F%2Fpresto.mre.red%2Ftemplates%2Fgongwen",
			ok:   false,
		},
		{
			name: "reject open URL fragment",
			raw:  "presto://open?resource=template&id=gongwen#install",
			ok:   false,
		},
		{
			name: "reject path traversal",
			raw:  "presto://open?resource=template&id=../gongwen",
			ok:   false,
		},
		{
			name: "reject invalid id",
			raw:  "presto://open?resource=template&id=Gongwen",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parsePrestoTemplateURL(tt.raw)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("template id = %q, want %q", got, tt.want)
			}
		})
	}
}
