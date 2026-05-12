package preview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

type Page struct {
	Index int    `json:"index"`
	SVG   string `json:"svg"`
	Hash  string `json:"hash"`
}

func HashSVG(svg string) string {
	sum := sha256.Sum256([]byte(svg))
	return hex.EncodeToString(sum[:])
}

func PagesFromSVG(svgPages []string) []Page {
	pages := make([]Page, len(svgPages))
	for i, svg := range svgPages {
		pages[i] = Page{
			Index: i + 1,
			SVG:   svg,
			Hash:  HashSVG(svg),
		}
	}
	return pages
}

func DiffPages(previous, next []Page) []Page {
	previousByIndex := make(map[int]Page, len(previous))
	nextByIndex := make(map[int]Page, len(next))
	for _, page := range previous {
		previousByIndex[page.Index] = page
	}
	for _, page := range next {
		nextByIndex[page.Index] = page
	}

	var diff []Page
	for _, page := range next {
		if previousPage, ok := previousByIndex[page.Index]; !ok || previousPage.Hash != page.Hash {
			diff = append(diff, page)
		}
	}

	emptyHash := HashSVG("")
	for _, page := range previous {
		if _, ok := nextByIndex[page.Index]; !ok {
			diff = append(diff, Page{Index: page.Index, Hash: emptyHash})
		}
	}

	sort.Slice(diff, func(i, j int) bool {
		return diff[i].Index < diff[j].Index
	})
	return diff
}

type SVGCompiler interface {
	CompileToSVG(typstSource string, workDir string) ([]string, error)
}

func CompileFallback(ctx context.Context, compiler SVGCompiler, typstSource string, workDir string) ([]Page, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	svgPages, err := compiler.CompileToSVG(typstSource, workDir)
	if err != nil {
		return nil, err
	}
	return PagesFromSVG(svgPages), nil
}
