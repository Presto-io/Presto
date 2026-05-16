package typst

import (
	"strings"
	"testing"
)

func TestNormalizeTypstFontFamiliesChoosesAvailableTupleFont(t *testing.T) {
	source := `#let FONT_SONG = ("Songti SC", "STSong")
#set text(font: FONT_SONG)
#set text(font: ("STSong", "Noto Serif CJK SC", "Songti SC", "SimSun"))`
	available := map[string]bool{"STSong": true}

	got := normalizeTypstFontFamilies(source, available)
	if strings.Contains(got, "Songti SC") {
		t.Fatalf("normalized source still contains missing Songti SC:\n%s", got)
	}
	if strings.Count(got, `"STSong"`) != 2 {
		t.Fatalf("normalized source = %q, want two STSong literals", got)
	}
}

func TestNormalizeTypstFontFamiliesUsesPlatformAlias(t *testing.T) {
	source := `#let FONT_FS = "STFangsong"
#let FONT_KAI = "STKaiti"
#let FONT_SONG = ("STSong", "Songti SC")`
	available := map[string]bool{
		"FangSong": true,
		"KaiTi":    true,
		"SimSun":   true,
	}

	got := normalizeTypstFontFamilies(source, available)
	for _, want := range []string{`#let FONT_FS = "FangSong"`, `#let FONT_KAI = "KaiTi"`, `#let FONT_SONG = "SimSun"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("normalized source missing %q:\n%s", want, got)
		}
	}
}

func TestNormalizeTypstFontFamiliesLeavesUnknownFonts(t *testing.T) {
	source := `#let FONT_CUSTOM = "Definitely Missing"`

	got := normalizeTypstFontFamilies(source, map[string]bool{"STSong": true})
	if got != source {
		t.Fatalf("normalize changed unknown font source: %q", got)
	}
}
