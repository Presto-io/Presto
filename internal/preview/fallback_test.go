package preview

import (
	"context"
	"errors"
	"testing"
)

type fakeCompiler struct {
	called bool
	pages  []string
	err    error
}

func (f *fakeCompiler) CompileToSVG(typstSource string, workDir string) ([]string, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return f.pages, nil
}

func TestHashSVGStable(t *testing.T) {
	first := HashSVG("<svg>1</svg>")
	second := HashSVG("<svg>1</svg>")
	if first != second {
		t.Fatalf("HashSVG not stable: %q != %q", first, second)
	}
}

func TestPagesFromSVGAssignsOneBasedIndexes(t *testing.T) {
	pages := PagesFromSVG([]string{"<svg>1</svg>", "<svg>2</svg>"})

	if len(pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(pages))
	}
	if pages[0].Index != 1 || pages[1].Index != 2 {
		t.Fatalf("indexes = %d, %d; want 1, 2", pages[0].Index, pages[1].Index)
	}
}

func TestDiffPagesOnlyChangedPage(t *testing.T) {
	previous := PagesFromSVG([]string{"<svg>1</svg>", "<svg>2</svg>", "<svg>3</svg>"})
	next := PagesFromSVG([]string{"<svg>1</svg>", "<svg>changed</svg>", "<svg>3</svg>"})

	diff := DiffPages(previous, next)

	if len(diff) != 1 {
		t.Fatalf("len(diff) = %d, want 1: %#v", len(diff), diff)
	}
	if diff[0].Index != 2 || diff[0].SVG != "<svg>changed</svg>" {
		t.Fatalf("diff[0] = %#v, want changed page 2", diff[0])
	}
}

func TestDiffPagesMarksRemovedPages(t *testing.T) {
	previous := PagesFromSVG([]string{"<svg>1</svg>", "<svg>2</svg>"})
	next := PagesFromSVG([]string{"<svg>1</svg>"})

	diff := DiffPages(previous, next)

	if len(diff) != 1 {
		t.Fatalf("len(diff) = %d, want 1: %#v", len(diff), diff)
	}
	if diff[0].Index != 2 {
		t.Fatalf("removed index = %d, want 2", diff[0].Index)
	}
	if diff[0].SVG != "" {
		t.Fatalf("removed SVG = %q, want empty", diff[0].SVG)
	}
	if diff[0].Hash != HashSVG("") {
		t.Fatalf("removed hash = %q, want empty SVG hash", diff[0].Hash)
	}
}

func TestCompileFallbackCallsCompilerAndHashesPages(t *testing.T) {
	compiler := &fakeCompiler{pages: []string{"<svg>1</svg>", "<svg>2</svg>"}}

	pages, err := CompileFallback(context.Background(), compiler, "= Hello", "/tmp/doc")

	if err != nil {
		t.Fatalf("CompileFallback failed: %v", err)
	}
	if !compiler.called {
		t.Fatal("CompileFallback did not call compiler")
	}
	if len(pages) != 2 || pages[0].Hash == "" || pages[1].Hash == "" {
		t.Fatalf("pages missing hashes: %#v", pages)
	}
}

func TestCompileFallbackReturnsCanceledContextBeforeCompile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	compiler := &fakeCompiler{pages: []string{"<svg/>"}}

	_, err := CompileFallback(ctx, compiler, "= Hello", "/tmp/doc")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if compiler.called {
		t.Fatal("compiler was called after context cancellation")
	}
}
