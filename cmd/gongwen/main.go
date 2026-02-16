package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

//go:embed template_head.typ
var templateHead string

//go:embed manifest.json
var manifestJSON string

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
			buf.Write(node.(*ast.String).Value)
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
		return convertPunctuation(string(n.(*ast.String).Value))

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
		// Images are handled at block level in renderParagraph
		return ""

	case ast.KindRawHTML:
		return ""

	default:
		// For unknown inline nodes, try to render children
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

	// Check if any image has alt text → subfigure mode
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

	// Build paths, captions, alts strings
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
  // 图片路径列表
  let paths = (%s)
  // 图片标题列表（对应 paths）
  let captions = (%s)
  // Alt Text 列表
  let alts = (%s)

  let is_subfigure = %s
  let main_caption = "%s"

  let gap = 0.3cm  // 图片间隙
  let max-width = 13.4cm
  let min-height = 6cm

  // 测量所有图片的原始尺寸
  let sizes = paths.zip(captions).zip(alts).map(item => {
    let p = item.at(0).at(0)
    let c = item.at(0).at(1)
    let alt = item.at(1)
    let img = image(p)
    let s = measure(img)
    (width: s.width, height: s.height, path: p, caption: c, alt: alt, ratio: s.width / s.height)
  })

  // 函数：计算一组图片等高排列时的高度
  let calc-row-height(imgs, total-width) = {
    // 计算所有图片宽高比之和
    let ratio-sum = imgs.map(i => i.ratio).sum()
    // 等高时的高度 = 总宽度 / 宽高比之和
    total-width / ratio-sum
  }

  // 分行算法
  let rows = ()

  if is_subfigure {
    // 方案一（子图模式）：强制所有图片在同一行
    rows.push(sizes)
  } else {
    // 方案二（独立模式）：自动换行逻辑
    let remaining = sizes

    while remaining.len() > 0 {
      let row = ()
      let found = false

      // 尝试放入尽可能多的图片
      for n in range(1, remaining.len() + 1) {
        let candidate = remaining.slice(0, n)
        let gaps = (n - 1) * gap
        let available-width = max-width - gaps
        let row-h = calc-row-height(candidate, available-width)

        if row-h < min-height and n > 1 {
          // 高度不够，使用前一个数量
          row = remaining.slice(0, n - 1)
          remaining = remaining.slice(n - 1)
          found = true
          break
        }
      }

      if not found {
        // 所有图片都能放，或者只有一张图片
        row = remaining
        remaining = ()
      }

      rows.push(row)
    }
  }

  // 渲染函数
  let render-rows(rows) = {
    for row in rows {
      let n = row.len()
      let gaps = (n - 1) * gap
      let available-width = max-width - gaps
      let row-height = calc-row-height(row, available-width)

      // 限制最大高度
      if row-height > max-width {
        row-height = max-width
      }

      align(center, grid(
        columns: n,
        gutter: gap,
        ..row.enumerate().map(item => {
          let i = item.at(0)
          let img-data = item.at(1)
          // 使用比例计算宽度：w = row-height * ratio
          let w = row-height * img-data.ratio

          if is_subfigure {
             // 子图模式：上面是图，下面是 (a) 文件名
             // 使用文件名(img-data.caption)作为子图注，忽略其他 Alt
             let sub-label = numbering("a", i + 1)
             let sub-text = [ (#sub-label) #img-data.caption ]

             v(0.5em)
             align(center, block({
               image(img-data.path, width: w, height: row-height)
               // 子图注样式：与大图注一致 (FONT_FS, zh(3))
               align(center, text(font: FONT_FS, size: zh(3))[#sub-text])
             }))
          } else {
             // 独立模式：完整的 figure
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

  // 根据模式输出
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
// Returns: cleaned text, "noindent"/"indent"/"" marker type.
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
	// Check for images
	images := c.collectImages(para)
	if len(images) == 1 {
		return c.renderSingleImage(images[0])
	}
	if len(images) > 1 {
		return c.renderMultiImage(images)
	}

	// Get plain text for marker detection
	plain := c.plainText(para)
	trimmed := strings.TrimSpace(plain)

	// Check standalone markers
	if result, ok := processMarker(trimmed); ok {
		return result
	}

	// Render inline content
	content := c.renderInlines(para)

	// Check trailing markers
	_, marker := stripTrailingMarker(trimmed)
	if marker == "noindent" {
		// Strip the marker from rendered content too
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

	// Greeting detection: before any header, paragraph ending with : or ：
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

	// Level 1: skip (title from metadata)
	if h.Level == 1 {
		return ""
	}

	content := c.renderInlines(h)

	// Check for trailing {.noindent} marker
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
			// For tight lists, goldmark may use TextBlock or other node types
			content := c.renderInlines(child)
			if content == "" {
				// Try rendering as block children
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
			// Collect children until noindent-end
			child = child.NextSibling()
			var innerBuf strings.Builder
			for child != nil && !isNoindentEnd(child, c.source) {
				innerBuf.WriteString(c.renderBlock(child, true))
				child = child.NextSibling()
			}
			if child != nil {
				child = child.NextSibling() // skip noindent-end
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
		// Skip HTML blocks that aren't our markers
		return ""
	default:
		// Try to render children for unknown block types
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
		// Prefix each line with >
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
		goldmark.WithExtensions(extension.Typographer),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	doc := md.Parser().Parse(text.NewReader(source))

	conv := &converter{source: source}
	return conv.renderDocument(doc)
}

// ---------- Convert ----------

// convert takes parsed front-matter and markdown body, returns full .typ output.
func convert(fm frontMatter, body string) string {
	var out strings.Builder

	// 1. Template head
	out.WriteString(templateHead)

	// 2. Variable section
	out.WriteString(fmt.Sprintf("#let autoTitle = \"%s\"\n", fm.Title))
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("#let autoAuthor = \"%s\"\n", fm.Author))
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("#let autoDate = %s\n", formatDate(fm.Date)))
	out.WriteString("\n")

	// Document metadata
	out.WriteString("#set document(\n")
	out.WriteString("  title: autoTitle.replace(\"|\", \" \"),\n")
	out.WriteString("  author: autoAuthor,\n")
	out.WriteString("  keywords: \"工作总结, 年终报告\",\n")
	out.WriteString("  date: auto,\n")
	out.WriteString(")\n")
	out.WriteString("\n")

	// Title heading
	out.WriteString("= #autoTitle.split(\"|\").map(s => s.trim()).join(linebreak())\n")
	out.WriteString("\n")

	// Author name (only if NOT signature mode)
	if !fm.Signature {
		out.WriteString("#name(autoAuthor)\n")
	}
	out.WriteString("\n")

	// Body: convert markdown to Typst
	out.WriteString(convertBody(body))

	// Signature block at end
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
	outputFile := flag.String("o", "", "output .typ file (default: stdout)")
	flag.Parse()

	if *manifestFlag {
		fmt.Print(manifestJSON)
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