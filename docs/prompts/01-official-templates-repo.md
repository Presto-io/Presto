# 提示词：初始化 presto-official-templates 仓库

## 背景

仓库已创建：https://github.com/Presto-io/presto-official-templates

这是 Presto（Markdown → Typst → PDF 桌面排版工具）的官方免费模板仓库。模板是独立的 Go 二进制程序，遵循 stdin/stdout 协议：
- `./binary --manifest` → 输出 manifest.json
- `./binary --example` → 输出示例 markdown
- `./binary --version` → 输出版本号（从 manifest.json 读取）
- `cat input.md | ./binary` → stdin 读 markdown，stdout 写 typst 源码

## 任务

初始化这个仓库，包含两个已有模板（gongwen、jiaoan-shicao）的完整源码。

## 仓库结构

```
presto-official-templates/
├── .github/
│   └── workflows/
│       └── release.yml              # 统一 CI（见下方详细说明）
├── gongwen/                         # 类公文模板
│   ├── main.go
│   ├── template_head.typ
│   ├── example.md
│   └── manifest.json
├── jiaoan-shicao/                   # 实操教案模板
│   ├── main.go
│   ├── example.md
│   └── manifest.json
├── go.mod                           # 共享 module
├── go.sum
├── Makefile
├── CLAUDE.md
├── CONVENTIONS.md                   # 开发者学习参考（从 starter 搬来，需更新）
├── LICENSE                          # MIT
└── README.md
```

## 源码文件

### go.mod

```
module github.com/Presto-io/presto-official-templates

go 1.23

require (
    github.com/yuin/goldmark v1.7.8
    gopkg.in/yaml.v3 v3.0.1
)
```

注意：两个模板的 `main.go` 中 import 路径不需要变，因为它们没有跨 package 引用，只用标准库 + goldmark + yaml.v3。

### gongwen/main.go

**完整复制以下代码**（这是从主仓库 cmd/gongwen/main.go 搬来的，需要做以下修改）：

1. 添加 `--version` flag 支持（见下方协议变更）
2. 其他逻辑完全不变

原始 main.go 如下，请在此基础上添加 --version：

```go
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

//go:embed template_head.typ
var templateHead string

//go:embed manifest.json
var manifestJSON string

//go:embed example.md
var exampleMD string

// ---------- YAML front-matter ----------

type frontMatter struct {
	Title     string
	Author    string // joined with "、"
	Date      string // raw string from YAML
	Signature bool
}

// parseFrontMatter splits "---" delimited YAML from body and returns metadata + body.
func parseFrontMatter(input string) (frontMatter, string) {
	var fm frontMatter
	fm.Title = "请输入文字"
	fm.Author = "请输入文字"

	// Normalise line endings
	input = strings.ReplaceAll(input, "\r\n", "\n")

	if !strings.HasPrefix(input, "---") {
		return fm, input
	}

	// Find closing ---
	rest := input[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, input
	}
	yamlBlock := rest[:idx]
	body := rest[idx+4:] // skip "\n---"
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}

	// Parse YAML into a generic map
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlBlock), &raw); err != nil {
		return fm, body
	}

	// title
	if v, ok := raw["title"]; ok {
		fm.Title = fmt.Sprintf("%v", v)
	}

	// author: string or list of strings → join with "、"
	if v, ok := raw["author"]; ok {
		switch a := v.(type) {
		case string:
			fm.Author = a
		case []interface{}:
			parts := make([]string, 0, len(a))
			for _, item := range a {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			fm.Author = strings.Join(parts, "、")
		}
	}

	// date
	if v, ok := raw["date"]; ok {
		fm.Date = fmt.Sprintf("%v", v)
	}

	// signature: bool or string
	if v, ok := raw["signature"]; ok {
		switch s := v.(type) {
		case bool:
			fm.Signature = s
		case string:
			lower := strings.ToLower(s)
			fm.Signature = lower == "true" || lower == "yes"
		}
	}

	return fm, body
}

// formatDate converts "YYYY-MM-DD" to datetime(year: N, month: N, day: N),
// otherwise returns a quoted string.
func formatDate(date string) string {
	if date == "" {
		return `""`
	}
	re := regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)
	m := re.FindStringSubmatch(date)
	if m != nil {
		// Strip leading zeros for month/day
		year := m[1]
		month := strings.TrimLeft(m[2], "0")
		day := strings.TrimLeft(m[3], "0")
		return fmt.Sprintf("datetime(\n  year: %s,\n  month: %s,\n  day: %s,\n)", year, month, day)
	}
	return fmt.Sprintf(`"%s"`, date)
}

// ---------- Punctuation conversion ----------

// urlPattern matches common URL schemes to skip
var urlPattern = regexp.MustCompile(`https?://[^\s]+|ftp://[^\s]+|mailto:[^\s]+`)

// markerPattern matches {…} markers to skip
var markerPattern = regexp.MustCompile(`\{[^}]*\}`)

// convertPunctuation converts half-width punctuation to full-width for Chinese text.
func convertPunctuation(text string) string {
	// Find all regions to skip (URLs and markers)
	type span struct{ start, end int }
	var skipSpans []span

	for _, loc := range urlPattern.FindAllStringIndex(text, -1) {
		skipSpans = append(skipSpans, span{loc[0], loc[1]})
	}
	for _, loc := range markerPattern.FindAllStringIndex(text, -1) {
		skipSpans = append(skipSpans, span{loc[0], loc[1]})
	}

	inSkip := func(pos int) bool {
		for _, s := range skipSpans {
			if pos >= s.start && pos < s.end {
				return true
			}
		}
		return false
	}

	runes := []rune(text)
	var buf strings.Builder
	buf.Grow(len(text))

	for i, r := range runes {
		bytePos := len(string(runes[:i]))
		if inSkip(bytePos) {
			buf.WriteRune(r)
			continue
		}

		switch r {
		case ',':
			buf.WriteRune('，')
		case ';':
			buf.WriteRune('；')
		case '?':
			buf.WriteRune('？')
		case '(':
			buf.WriteRune('（')
		case ')':
			buf.WriteRune('）')
		case ':':
			// Keep colon between digits (e.g. 12:30)
			if i > 0 && i < len(runes)-1 && unicode.IsDigit(runes[i-1]) && unicode.IsDigit(runes[i+1]) {
				buf.WriteRune(':')
			} else {
				buf.WriteRune('：')
			}
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// ---------- Markdown pre-processing ----------

var reNoindentOpen = regexp.MustCompile(`(?m)^::: \{\.noindent\}\s*$`)
var reNoindentClose = regexp.MustCompile(`(?m)^:::\s*$`)

func preprocessBody(body string) string {
	body = reNoindentOpen.ReplaceAllString(body, "<!-- noindent-start -->")
	body = reNoindentClose.ReplaceAllString(body, "<!-- noindent-end -->")
	return body
}

// ---------- Goldmark AST → Typst converter ----------

type converter struct {
	source        []byte
	figureCounter int
	hasSeenHeader bool
}

// nodeText extracts raw text from an inline node and its children.
func (c *converter) nodeText(n ast.Node) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindText {
			t := child.(*ast.Text)
			buf.Write(t.Segment.Value(c.source))
			if t.SoftLineBreak() {
				buf.WriteByte('\n')
			}
		} else {
			buf.WriteString(c.nodeText(child))
		}
	}
	if n.Kind() == ast.KindText {
		t := n.(*ast.Text)
		buf.Write(t.Segment.Value(c.source))
	}
	return buf.String()
}

// plainText extracts all text from a node tree (for marker detection).
func (c *converter) plainText(n ast.Node) string {
	var buf strings.Builder
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if node.Kind() == ast.KindText {
			t := node.(*ast.Text)
			buf.Write(t.Segment.Value(c.source))
			if t.SoftLineBreak() {
				buf.WriteByte(' ')
			}
		} else if node.Kind() == ast.KindCodeSpan {
			// include code span text
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if child.Kind() == ast.KindText {
					t := child.(*ast.Text)
					buf.Write(t.Segment.Value(c.source))
				}
			}
			return ast.WalkSkipChildren, nil
		} else if node.Kind() == ast.KindString {
			buf.WriteString(html.UnescapeString(string(node.(*ast.String).Value)))
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}

// renderInlines renders inline children of a node to Typst.
func (c *converter) renderInlines(n ast.Node) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		buf.WriteString(c.renderInline(child))
	}
	return buf.String()
}

// renderInline renders a single inline node to Typst.
func (c *converter) renderInline(n ast.Node) string {
	switch n.Kind() {
	case ast.KindText:
		t := n.(*ast.Text)
		raw := string(t.Segment.Value(c.source))
		result := convertPunctuation(raw)
		if t.SoftLineBreak() {
			result += "\n"
		}
		if t.HardLineBreak() {
			result += " \\\n"
		}
		return result

	case ast.KindString:
		raw := html.UnescapeString(string(n.(*ast.String).Value))
		return convertPunctuation(raw)

	case ast.KindCodeSpan:
		var code strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				code.Write(child.(*ast.Text).Segment.Value(c.source))
			}
		}
		return "`" + code.String() + "`"

	case ast.KindEmphasis:
		em := n.(*ast.Emphasis)
		inner := c.renderInlines(n)
		if em.Level == 2 {
			return "#strong[" + inner + "]"
		}
		return "#emph[" + inner + "]"

	case ast.KindLink:
		link := n.(*ast.Link)
		inner := c.renderInlines(n)
		return fmt.Sprintf(`#link("%s")[%s]`, string(link.Destination), inner)

	case ast.KindAutoLink:
		al := n.(*ast.AutoLink)
		url := string(al.URL(c.source))
		return fmt.Sprintf(`#link("%s")`, url)

	case ast.KindImage:
		return ""

	case ast.KindRawHTML:
		return ""

	default:
		return c.renderInlines(n)
	}
}

// collectImages collects all Image nodes from a paragraph's children.
func (c *converter) collectImages(para ast.Node) []*ast.Image {
	var images []*ast.Image
	for child := para.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindImage {
			images = append(images, child.(*ast.Image))
		}
	}
	return images
}

// renderSingleImage generates Typst figure code for a single image.
func (c *converter) renderSingleImage(img *ast.Image) string {
	c.figureCounter++
	path := string(img.Destination)
	filename := filepath.Base(path)
	caption := strings.TrimSuffix(filename, filepath.Ext(filename))

	return fmt.Sprintf(`#figure(
  context {
    let img = image("%s")
    let img-size = measure(img)
    let x = img-size.width
    let y = img-size.height
    let max-size = 13.4cm

    let new-x = x
    let new-y = y

    if x > max-size {
      let scale = max-size / x
      new-x = max-size
      new-y = y * scale
    }

    if new-y > max-size {
      let scale = max-size / new-y
      new-x = new-x * scale
      new-y = max-size
    }

    image("%s", width: new-x, height: new-y)
  },
  caption: [%s],
) <fig-%d>
`, path, path, caption, c.figureCounter)
}

// renderMultiImage generates Typst code for multiple images in one paragraph.
func (c *converter) renderMultiImage(images []*ast.Image) string {
	type imgInfo struct {
		path, caption, alt string
		figNum             int
	}

	var infos []imgInfo
	isSubfigure := false

	for _, img := range images {
		alt := c.plainText(img)
		if alt != "" {
			isSubfigure = true
			break
		}
	}

	if isSubfigure {
		c.figureCounter++
	}

	for _, img := range images {
		path := string(img.Destination)
		filename := filepath.Base(path)
		caption := strings.TrimSuffix(filename, filepath.Ext(filename))
		alt := c.plainText(img)
		figNum := 0
		if !isSubfigure {
			c.figureCounter++
			figNum = c.figureCounter
		}
		infos = append(infos, imgInfo{path, caption, alt, figNum})
	}

	var pathsStr, captionsStr, altsStr []string
	mainCaption := ""
	for _, info := range infos {
		pathsStr = append(pathsStr, fmt.Sprintf(`"%s"`, info.path))
		captionsStr = append(captionsStr, fmt.Sprintf(`"%s"`, info.caption))
		altsStr = append(altsStr, fmt.Sprintf(`"%s"`, info.alt))
	}
	if isSubfigure && len(infos) > 0 {
		mainCaption = infos[0].alt
	}

	return fmt.Sprintf(`
#context {
  let paths = (%s)
  let captions = (%s)
  let alts = (%s)

  let is_subfigure = %s
  let main_caption = "%s"

  let gap = 0.3cm
  let max-width = 13.4cm
  let min-height = 6cm

  let sizes = paths.zip(captions).zip(alts).map(item => {
    let p = item.at(0).at(0)
    let c = item.at(0).at(1)
    let alt = item.at(1)
    let img = image(p)
    let s = measure(img)
    (width: s.width, height: s.height, path: p, caption: c, alt: alt, ratio: s.width / s.height)
  })

  let calc-row-height(imgs, total-width) = {
    let ratio-sum = imgs.map(i => i.ratio).sum()
    total-width / ratio-sum
  }

  let rows = ()

  if is_subfigure {
    rows.push(sizes)
  } else {
    let remaining = sizes

    while remaining.len() > 0 {
      let row = ()
      let found = false

      for n in range(1, remaining.len() + 1) {
        let candidate = remaining.slice(0, n)
        let gaps = (n - 1) * gap
        let available-width = max-width - gaps
        let row-h = calc-row-height(candidate, available-width)

        if row-h < min-height and n > 1 {
          row = remaining.slice(0, n - 1)
          remaining = remaining.slice(n - 1)
          found = true
          break
        }
      }

      if not found {
        row = remaining
        remaining = ()
      }

      rows.push(row)
    }
  }

  let render-rows(rows) = {
    for row in rows {
      let n = row.len()
      let gaps = (n - 1) * gap
      let available-width = max-width - gaps
      let row-height = calc-row-height(row, available-width)

      if row-height > max-width {
        row-height = max-width
      }

      align(center, grid(
        columns: n,
        gutter: gap,
        ..row.enumerate().map(item => {
          let i = item.at(0)
          let img-data = item.at(1)
          let w = row-height * img-data.ratio

          if is_subfigure {
             let sub-label = numbering("a", i + 1)
             let sub-text = [ (#sub-label) #img-data.caption ]

             v(0.5em)
             align(center, block({
               image(img-data.path, width: w, height: row-height)
               align(center, text(font: FONT_FS, size: zh(3))[#sub-text])
             }))
          } else {
             figure(
               image(img-data.path, width: w, height: row-height),
               caption: [ #img-data.caption ]
             )
          }
        })
      ))
      if is_subfigure { v(0.5em) } else { v(0.3em) }
    }
  }

  if is_subfigure {
    figure(
      context { render-rows(rows) },
      caption: [ #main_caption ]
    )
  } else {
    render-rows(rows)
  }
}

`, strings.Join(pathsStr, ", "), strings.Join(captionsStr, ", "),
		strings.Join(altsStr, ", "), strconv.FormatBool(isSubfigure), mainCaption)
}

// vMarkerRe matches {v} or {v:N}
var vMarkerRe = regexp.MustCompile(`^\{v(?::(\d+))?\}$`)

// processMarker checks if text is a standalone marker and returns Typst code.
func processMarker(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if m := vMarkerRe.FindStringSubmatch(text); m != nil {
		count := 1
		if m[1] != "" {
			count, _ = strconv.Atoi(m[1])
		}
		var lines []string
		for i := 0; i < count; i++ {
			lines = append(lines, "#linebreak(justify: false)")
		}
		return strings.Join(lines, "\n") + "\n", true
	}
	if text == "{pagebreak}" {
		return "#pagebreak()\n", true
	}
	if text == "{pagebreak:weak}" {
		return "#pagebreak(weak: true)\n", true
	}
	return "", false
}

// stripTrailingMarker checks for {.noindent} or {indent} at end of inline text.
func stripTrailingMarker(text string) (string, string) {
	text = strings.TrimRight(text, " ")
	if strings.HasSuffix(text, "{.noindent}") {
		return strings.TrimRight(strings.TrimSuffix(text, "{.noindent}"), " "), "noindent"
	}
	if strings.HasSuffix(text, "{indent}") {
		return strings.TrimRight(strings.TrimSuffix(text, "{indent}"), " "), "indent"
	}
	return text, ""
}

// renderParagraph renders a paragraph node to Typst.
func (c *converter) renderParagraph(para *ast.Paragraph) string {
	images := c.collectImages(para)
	if len(images) == 1 {
		return c.renderSingleImage(images[0])
	}
	if len(images) > 1 {
		return c.renderMultiImage(images)
	}

	plain := c.plainText(para)
	trimmed := strings.TrimSpace(plain)

	if result, ok := processMarker(trimmed); ok {
		return result
	}

	content := c.renderInlines(para)

	_, marker := stripTrailingMarker(trimmed)
	if marker == "noindent" {
		content = strings.TrimRight(content, " \n")
		content = strings.TrimSuffix(content, "{.noindent}")
		content = strings.TrimRight(content, " ")
		return "#block[#set par(first-line-indent: 0pt)\n#block[\n" + content + "\n\n]\n]\n"
	}
	if marker == "indent" {
		content = strings.TrimRight(content, " \n")
		content = strings.TrimSuffix(content, "{indent}")
		content = strings.TrimRight(content, " ")
		return content + "\n\n"
	}

	if !c.hasSeenHeader {
		t := strings.TrimSpace(content)
		if strings.HasSuffix(t, "：") || strings.HasSuffix(t, ":") {
			return "#block[#set par(first-line-indent: 0pt)\n#block[\n" + content + "\n\n]\n]\n"
		}
	}

	return content + "\n\n"
}

// renderHeading renders a heading node to Typst.
func (c *converter) renderHeading(h *ast.Heading) string {
	c.hasSeenHeader = true

	if h.Level == 1 {
		return ""
	}

	content := c.renderInlines(h)

	_, marker := stripTrailingMarker(strings.TrimSpace(c.plainText(h)))
	if marker == "noindent" {
		content = strings.TrimRight(content, " \n")
		content = strings.TrimSuffix(content, "{.noindent}")
		content = strings.TrimRight(content, " ")
		prefix := strings.Repeat("=", h.Level)
		return "#block[#set par(first-line-indent: 0pt)\n" + prefix + " " + content + "\n]\n\n"
	}

	prefix := strings.Repeat("=", h.Level)
	return prefix + " " + content + "\n\n"
}

// renderList renders a list node to Typst.
func (c *converter) renderList(list *ast.List) string {
	var buf strings.Builder
	marker := "- "
	if list.IsOrdered() {
		marker = "+ "
	}
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindListItem {
			buf.WriteString(marker)
			buf.WriteString(c.renderListItem(child))
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

// renderListItem renders a list item's content.
func (c *converter) renderListItem(item ast.Node) string {
	var parts []string
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindParagraph:
			content := c.renderInlines(child)
			content = strings.TrimRight(content, "\n")
			parts = append(parts, content)
		case ast.KindList:
			parts = append(parts, c.renderList(child.(*ast.List)))
		default:
			content := c.renderInlines(child)
			if content == "" {
				for gc := child.FirstChild(); gc != nil; gc = gc.NextSibling() {
					content += c.renderInline(gc)
				}
			}
			content = strings.TrimRight(content, "\n")
			if content != "" {
				parts = append(parts, content)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// renderHTMLBlock checks for noindent markers.
func isNoindentStart(n ast.Node, source []byte) bool {
	if n.Kind() != ast.KindHTMLBlock {
		return false
	}
	lines := n.Lines()
	if lines.Len() == 0 {
		return false
	}
	seg := lines.At(0)
	line := string(seg.Value(source))
	return strings.Contains(line, "noindent-start")
}

func isNoindentEnd(n ast.Node, source []byte) bool {
	if n.Kind() != ast.KindHTMLBlock {
		return false
	}
	lines := n.Lines()
	if lines.Len() == 0 {
		return false
	}
	seg := lines.At(0)
	line := string(seg.Value(source))
	return strings.Contains(line, "noindent-end")
}

// renderDocument renders the full document body.
func (c *converter) renderDocument(doc ast.Node) string {
	var buf strings.Builder
	child := doc.FirstChild()

	for child != nil {
		if isNoindentStart(child, c.source) {
			child = child.NextSibling()
			var innerBuf strings.Builder
			for child != nil && !isNoindentEnd(child, c.source) {
				innerBuf.WriteString(c.renderBlock(child, true))
				child = child.NextSibling()
			}
			if child != nil {
				child = child.NextSibling()
			}
			inner := innerBuf.String()
			buf.WriteString("#block[#set par(first-line-indent: 0pt)\n#block[\n")
			buf.WriteString(inner)
			buf.WriteString("]\n]\n")
		} else {
			buf.WriteString(c.renderBlock(child, false))
			child = child.NextSibling()
		}
	}

	return buf.String()
}

// renderBlock renders a single block-level node.
func (c *converter) renderBlock(n ast.Node, inNoindent bool) string {
	switch n.Kind() {
	case ast.KindParagraph:
		return c.renderParagraph(n.(*ast.Paragraph))
	case ast.KindHeading:
		return c.renderHeading(n.(*ast.Heading))
	case ast.KindList:
		content := c.renderList(n.(*ast.List))
		if inNoindent {
			return "#block[#set par(first-line-indent: 0pt)\n" + content + "]\n"
		}
		return content
	case ast.KindFencedCodeBlock, ast.KindCodeBlock:
		return c.renderCodeBlock(n)
	case ast.KindThematicBreak:
		return "#line(length: 100%)\n\n"
	case ast.KindBlockquote:
		return c.renderBlockquote(n)
	case ast.KindHTMLBlock:
		return ""
	default:
		var buf strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			buf.WriteString(c.renderBlock(child, inNoindent))
		}
		return buf.String()
	}
}

// renderCodeBlock renders a fenced or indented code block.
func (c *converter) renderCodeBlock(n ast.Node) string {
	var buf strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(c.source))
	}
	code := buf.String()

	lang := ""
	if fcb, ok := n.(*ast.FencedCodeBlock); ok {
		if fcb.Info != nil {
			lang = string(fcb.Info.Segment.Value(c.source))
			lang = strings.TrimSpace(strings.SplitN(lang, " ", 2)[0])
		}
	}

	if lang != "" {
		return "```" + lang + "\n" + code + "```\n\n"
	}
	return "```\n" + code + "```\n\n"
}

// renderBlockquote renders a blockquote.
func (c *converter) renderBlockquote(n ast.Node) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		content := c.renderBlock(child, false)
		for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
			buf.WriteString("#quote[" + line + "]\n")
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

// convertBody parses markdown body and renders to Typst.
func convertBody(body string) string {
	body = preprocessBody(body)
	source := []byte(body)

	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	doc := md.Parser().Parse(text.NewReader(source))

	conv := &converter{source: source}
	return conv.renderDocument(doc)
}

// convert takes parsed front-matter and markdown body, returns full .typ output.
func convert(fm frontMatter, body string) string {
	var out strings.Builder

	out.WriteString(templateHead)
	out.WriteString(fmt.Sprintf("#let autoTitle = \"%s\"\n", fm.Title))
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("#let autoAuthor = \"%s\"\n", fm.Author))
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("#let autoDate = %s\n", formatDate(fm.Date)))
	out.WriteString("\n")

	out.WriteString("#set document(\n")
	out.WriteString("  title: autoTitle.replace(\"|\", \" \"),\n")
	out.WriteString("  author: autoAuthor,\n")
	out.WriteString("  keywords: \"工作总结, 年终报告\",\n")
	out.WriteString("  date: auto,\n")
	out.WriteString(")\n")
	out.WriteString("\n")

	out.WriteString("= #autoTitle.split(\"|\").map(s => s.trim()).join(linebreak())\n")
	out.WriteString("\n")

	if !fm.Signature {
		out.WriteString("#name(autoAuthor)\n")
	}
	out.WriteString("\n")

	out.WriteString(convertBody(body))

	if fm.Signature {
		out.WriteString("\n#v(18pt)\n")
		out.WriteString("#align(right, block[\n")
		out.WriteString("  #set align(center)\n")
		out.WriteString("  #autoAuthor \\\n")
		out.WriteString("  #autoDate.display(\n")
		out.WriteString("    \"[year]年[month padding:none]月[day padding:none]日\",\n")
		out.WriteString("  )\n")
		out.WriteString("])\n")
	}

	return out.String()
}

// ---------- CLI ----------

func main() {
	manifestFlag := flag.Bool("manifest", false, "output manifest JSON")
	exampleFlag := flag.Bool("example", false, "output example markdown")
	outputFile := flag.String("o", "", "output .typ file (default: stdout)")
	flag.Parse()

	if *manifestFlag {
		fmt.Print(manifestJSON)
		return
	}

	if *exampleFlag {
		fmt.Print(exampleMD)
		return
	}

	var input []byte
	var err error
	args := flag.Args()
	if len(args) > 0 {
		input, err = os.ReadFile(args[0])
	} else {
		input, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	fm, body := parseFrontMatter(string(input))
	result := convert(fm, body)

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(result), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", *outputFile, err)
			os.Exit(1)
		}
	} else {
		fmt.Print(result)
	}
}
```

### --version flag 添加方法

在所有模板的 main() 函数中，在 `flag.Parse()` 之后、`manifestFlag` 检查之前，添加：

```go
versionFlag := flag.Bool("version", false, "output version")
// ... flag.Parse() ...

if *versionFlag {
    // 从 manifestJSON 解析 version 字段
    var m map[string]interface{}
    if err := json.Unmarshal([]byte(manifestJSON), &m); err == nil {
        if v, ok := m["version"]; ok {
            fmt.Println(v)
        }
    }
    return
}
```

需要 import `encoding/json`。

### gongwen/template_head.typ

完整复制：

```typst
// 中文字号转换函数
#import "@preview/pointless-size:0.1.2": zh

// 定义常用字体名称
#let FONT_XBS = "FZXiaoBiaoSong-B05" // 方正小标宋
#let FONT_HEI = "STHeiti" // 黑体
#let FONT_FS = "STFangsong" // 仿宋
#let FONT_KAI = "STKaiti" // 楷体
#let FONT_SONG = "STSong" // 宋体

// 设置页面、页边距、页脚
#set page(
  paper: "a4",
  margin: (
    inside: 28mm,
    outside: 26mm,
    top: 37mm,
    bottom: 35mm,
  ),

  // 将页脚基线放到"版心下边缘之下 7mm"
  footer-descent: 7mm,

  // 使用更稳定的奇偶页判断和页码格式
  footer: context {
    let page-num = here().page()
    let is-even = calc.even(page-num)
    let num = str(page-num)
    let pm = text(font: FONT_SONG, size: zh(4))[— #num —] // 4 号宋体

    if is-even {
      align(left, [#h(1em) #pm]) // 偶数页：居左
    } else {
      align(right, [#pm #h(1em)]) // 奇数页：居右
    }
  },
)

// 设置文档默认语言和正文字体
#set text(
  lang: "zh",
  font: FONT_FS,
  size: zh(3),
  hyphenate: false,
  cjk-latin-spacing: auto,
)

// 设置段落样式，以满足"每行28字符，每页22行"的网格标准，首行缩进2字符
#set par(
  first-line-indent: (amount: 2em, all: true),
  justify: true,
  leading: 15.6pt, // 行间距
  spacing: 15.6pt, // 段间距
)

// 计数器设置
#let h2-counter = counter("h2")
#let h3-counter = counter("h3")
#let h4-counter = counter("h4")
#let h5-counter = counter("h5")

// 图片样式设置
#show figure: it => {
  // 居中对齐，无首行缩进
  set par(first-line-indent: 0pt)
  align(center, block({
    // 图片尺寸由 Lua filter 控制
    it.body

    // 图注样式：3号仿宋，格式为"图1 标题"
    text(
      font: FONT_FS,
      size: zh(3),
      it.caption,
    )
  }))
}

// 自定义标题函数
#let custom-heading(level, body, numbering: auto) = {
  if level == 1 {
    v(0pt)
    align(center)[
      #text(
        font: FONT_XBS,
        size: zh(2),
        weight: "bold",
      )[
        #set par(leading: 35pt - zh(2))
        #body
      ]
    ]
    v(28.7pt)
  } else if level == 2 {
    h2-counter.step()
    h3-counter.update(0)
    h4-counter.update(1)
    h5-counter.update(1)
    text(
      font: FONT_HEI,
      size: zh(3),
    )[#context h2-counter.display("一、")#body]
  } else if level == 3 {
    h3-counter.step()
    h4-counter.update(1)
    h5-counter.update(1)

    let number = h3-counter.get().first()
    text(
      font: FONT_KAI,
      size: zh(3),
    )[#context h3-counter.display("（一）")#body]
  } else if level == 4 {
    h4-counter.step()
    h5-counter.update(1)

    let number = h4-counter.get().first()
    text(
      size: zh(3),
    )[#number. #body]
  } else if level == 5 {
    h5-counter.step()

    let number = h5-counter.get().first()
    text(
      size: zh(3),
    )[（#number）#body]
  }
}

#show heading: it => {
  if it.level == 1 {
    custom-heading(it.level, it.body, numbering: it.numbering)
  } else {
    let spacing = 13.9pt
    let threshold = 3em

    block(
      sticky: true,
      above: spacing,
      below: spacing,
      {
        block(
          custom-heading(it.level, it.body, numbering: it.numbering) + v(threshold),
          breakable: false,
        )
        v(-threshold)
      },
    )
  }
}

#h2-counter.update(0)
#h3-counter.update(0)
#h4-counter.update(0)
#h5-counter.update(0)

#let list-depth = state("list-depth", 0)

#let flush-left-list(it) = {
  list-depth.update(d => d + 1)

  let is-enum = (it.func() == enum)
  let children = it.children

  context {
    let depth = list-depth.get()
    let block-indent = if depth > 1 { 2em } else { 0pt }

    pad(left: block-indent, block({
      for (count, item) in children.enumerate(start: 1) {
        if item.func() == list.item or item.func() == enum.item {
          let marker = if is-enum {
            let pattern = if it.has("numbering") and it.numbering != auto { it.numbering } else { "1." }
            numbering(pattern, count)
          } else {
            if it.has("marker") and it.marker.len() > 0 { it.marker.at(0) } else { [•] }
          }

          par(
            first-line-indent: par.first-line-indent,
            hanging-indent: 0pt,
          )[#marker#h(0.25em)#item.body]
        } else {
          item
        }
      }
    }))

    list-depth.update(d => d - 1)
  }
}

#show list: flush-left-list
#show enum: flush-left-list

#let name(name) = align(center, pad(bottom: 0.8em)[
  #text(font: FONT_KAI, size: zh(3))[#name]
])

```

### gongwen/manifest.json

```json
{
  "name": "gongwen",
  "displayName": "类公文模板",
  "description": "符合 GB/T 9704-2012 标准的类公文排版，支持标题、作者、日期、签名等元素",
  "version": "1.0.0",
  "author": "Presto-io",
  "license": "MIT",
  "category": "公文",
  "keywords": ["公文", "通知", "报告", "政府", "GB/T 9704"],
  "minPrestoVersion": "0.1.0",
  "requiredFonts": [
    { "name": "FZXiaoBiaoSong-B05", "displayName": "方正小标宋", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/164" },
    { "name": "STHeiti", "displayName": "华文黑体", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/131" },
    { "name": "STFangsong", "displayName": "华文仿宋", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/128" },
    { "name": "STKaiti", "displayName": "华文楷体", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/130" },
    { "name": "STSong", "displayName": "华文宋体", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/135" }
  ],
  "frontmatterSchema": {
    "title": { "type": "string", "default": "请输入文字" },
    "author": { "type": "string", "default": "请输入文字" },
    "date": { "type": "string", "format": "YYYY-MM-DD" },
    "signature": { "type": "boolean", "default": false }
  }
}
```

### gongwen/example.md

```markdown
---
title: "关于开展2025年度安全生产专项检查工作的通知"
author: "安全生产管理处"
date: "2025-03-15"
signature: true
template: "gongwen"
---

各部门、各单位：

为进一步加强安全生产管理，落实安全生产责任制，根据《安全生产法》和上级主管部门要求，决定在全公司范围内开展2025年度安全生产专项检查工作。现将有关事项通知如下。

## 工作目标

全面排查安全生产隐患，建立健全安全管理制度，提高全员安全意识，确保全年安全生产事故"零发生"。

## 检查范围与内容

### 检查范围

本次专项检查覆盖公司所有生产经营场所，包括：

1. 各生产车间及仓储区域
2. 办公场所及公共区域
3. 在建工程项目现场

### 重点检查内容

- 安全生产责任制落实情况
- 消防设施设备完好情况
- 特种设备检验及操作人员持证上岗情况
- 危险化学品储存、使用管理情况
- **应急预案**的制定及演练情况

## 工作安排

### 自查自纠阶段

各部门、各单位对照检查标准，全面开展自查自纠，建立问题清单，制定整改措施。

### 集中检查阶段

由安全生产管理处牵头，组织相关部门成立联合检查组，对各单位进行全面检查。

### 整改落实阶段

针对检查中发现的问题，责任单位须在规定期限内完成整改，并将整改报告报送安全生产管理处。

## 工作要求

各部门、各单位要高度重视此次专项检查工作，主要负责人要亲自部署、亲自督办。对检查中发现的重大隐患，要立即整改；对不能立即整改的，要制定切实可行的整改方案，明确整改期限和责任人。

特此通知。
```

### jiaoan-shicao/ 目录

jiaoan-shicao 的 main.go 也是从主仓库 cmd/jiaoan-shicao/main.go 完整搬过来，同样需要添加 --version flag。代码太长不在此重复，请从本地文件复制。jiaoan-shicao 不依赖 goldmark，只用标准库 + embed。

jiaoan-shicao/manifest.json 更新 category：

```json
{
  "name": "jiaoan-shicao",
  "displayName": "实操教案模板",
  "description": "将 Markdown 格式的实操教案转换为标准表格排版",
  "version": "1.0.0",
  "author": "Presto-io",
  "license": "MIT",
  "category": "教育",
  "keywords": ["教案", "实操", "教学", "表格", "教育"],
  "minPrestoVersion": "0.1.0",
  "requiredFonts": [
    { "name": "FZXiaoBiaoSong-B05", "displayName": "方正小标宋", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/164" },
    { "name": "STHeiti", "displayName": "华文黑体", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/131" },
    { "name": "STFangsong", "displayName": "华文仿宋", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/128" },
    { "name": "STKaiti", "displayName": "华文楷体", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/130" },
    { "name": "STSong", "displayName": "华文宋体", "url": "https://www.foundertype.com/index.php/FontInfo/index/id/135" }
  ]
}
```

### CI: .github/workflows/release.yml

基于 starter-go 的 release.yml 改造，增加模板名矩阵：

```yaml
name: Release

on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  build:
    runs-on: ${{ matrix.os == 'windows' && 'windows-latest' || matrix.os == 'linux' && 'ubuntu-latest' || 'macos-latest' }}
    strategy:
      matrix:
        template: [gongwen, jiaoan-shicao]
        os: [darwin, linux, windows]
        arch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build
        env:
          GOOS: ${{ matrix.os }}
          GOARCH: ${{ matrix.arch }}
        run: |
          cd ${{ matrix.template }}
          EXT=""
          if [ "${{ matrix.os }}" = "windows" ]; then EXT=".exe"; fi
          go build -trimpath -ldflags="-s -w" -o "../presto-template-${{ matrix.template }}-${{ matrix.os }}-${{ matrix.arch }}${EXT}" .

      - uses: actions/upload-artifact@v4
        with:
          name: presto-template-${{ matrix.template }}-${{ matrix.os }}-${{ matrix.arch }}
          path: presto-template-*

  release:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v4
        with:
          merge-multiple: true

      - name: Generate checksums
        run: sha256sum presto-template-* > SHA256SUMS

      - uses: softprops/action-gh-release@v2
        with:
          files: |
            presto-template-*
            SHA256SUMS
          generate_release_notes: true
```

### Makefile

```makefile
.PHONY: build build-all test clean preview

TEMPLATES := gongwen jiaoan-shicao

build:
ifndef NAME
	$(error Usage: make build NAME=gongwen)
endif
	cd $(NAME) && go build -trimpath -ldflags="-s -w" -o ../presto-template-$(NAME) .

build-all:
	@for t in $(TEMPLATES); do \
		echo "Building $$t..."; \
		cd $$t && go build -trimpath -ldflags="-s -w" -o ../presto-template-$$t . && cd ..; \
	done

test:
ifndef NAME
	@for t in $(TEMPLATES); do \
		echo "Testing $$t..."; \
		./presto-template-$$t --manifest | python3 -m json.tool > /dev/null && \
		./presto-template-$$t --example | ./presto-template-$$t > /dev/null && \
		./presto-template-$$t --version > /dev/null && \
		echo "  $$t: OK"; \
	done
else
	./presto-template-$(NAME) --manifest | python3 -m json.tool > /dev/null
	./presto-template-$(NAME) --example | ./presto-template-$(NAME) > /dev/null
	./presto-template-$(NAME) --version > /dev/null
endif

preview:
ifndef NAME
	$(error Usage: make preview NAME=gongwen)
endif
	@mkdir -p ~/.presto/templates/$(NAME)
	cp presto-template-$(NAME) ~/.presto/templates/$(NAME)/
	./presto-template-$(NAME) --manifest > ~/.presto/templates/$(NAME)/manifest.json
	@echo "Installed $(NAME) to ~/.presto/templates/$(NAME)/"

clean:
	rm -f presto-template-*
```

### CLAUDE.md

```markdown
# Presto Official Templates

请阅读并遵循 CONVENTIONS.md。

## 关键约束

- 不要修改模板二进制协议（stdin/stdout 接口）
- 不要引入新的第三方 Go 依赖（只用 goldmark + yaml.v3 + 标准库）
- Commit 消息用中文，格式 `<type>: <描述>`
- 每个模板是独立的 main package，在自己的子目录下
```

### README.md

```markdown
# Presto Official Templates

Presto 官方免费模板集合。每个模板是一个独立的 Go 程序，遵循 Presto 模板协议（stdin Markdown → stdout Typst）。

## 包含模板

| 模板 | 说明 |
|------|------|
| `gongwen` | 符合 GB/T 9704-2012 标准的类公文排版 |
| `jiaoan-shicao` | 实操教案 Markdown → 标准表格排版 |

## 快速开始

### 构建

```bash
# 构建所有模板
make build-all

# 构建单个模板
make build NAME=gongwen
```

### 测试

```bash
make test
```

### 安装到 Presto

```bash
make preview NAME=gongwen
```

## 开发者

如果你想开发自己的模板，请参考：
- [CONVENTIONS.md](CONVENTIONS.md) — 模板开发规范
- [presto-template-starter-go](https://github.com/Presto-io/presto-template-starter-go) — Go 脚手架
- [presto-template-starter-rust](https://github.com/Presto-io/presto-template-starter-rust) — Rust 脚手架
- [presto-template-starter-typescript](https://github.com/Presto-io/presto-template-starter-typescript) — TypeScript 脚手架

## 协议

MIT
```

### LICENSE

MIT License, Copyright 2026 Presto

## 注意事项

1. `go.mod` 放在仓库根目录，但每个模板是独立的 `main` package。构建时需要 `cd` 进子目录或使用 `-C` flag。
2. jiaoan-shicao 的 main.go 不依赖 goldmark，但共享 go.mod 没有问题（多余依赖不会被编译进去）。
3. CONVENTIONS.md 暂时从 starter-go 复制一份过来，后续会迁移到 Presto-Homepage。
4. 两个模板都需要添加 `--version` flag，这是协议变更（见下方脚手架提示词）。
5. 构建完成后运行 `make test` 验证所有模板。
