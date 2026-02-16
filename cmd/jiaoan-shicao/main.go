/*
shicaojiaoan
linux darwin windows
实操教案格式化生成器
用法: shicaojiaoan [选项] [输入文件]

选项:
  -h    显示帮助信息
  -p    生成 PDF 文件（需要安装 typst）
  -t    生成空白模板文件 template.md
  -v    显示详细输出信息
*/

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	isTemplate bool
	isPdf      bool
	verbose    bool
	help       bool
)

const preamble = `// 中文字号转换函数
#import "@preview/pointless-size:0.1.2": zh
#import "@preview/cuti:0.2.1": show-cn-fakebold
#show: show-cn-fakebold

// 定义常用字体名称
#let FONT_XBS = ("FZXiaoBiaoSong-B05") // 方正小标宋
#let FONT_HEI = ("STHeiti") // 黑体
#let FONT_FS = ("STFangsong") // 仿宋
#let FONT_KAI = ("STKaiti") // 楷体
#let FONT_SONG = ("STSong") // 宋体

#set page(
  paper: "a4",
  flipped: true,
  margin: (top: 2.54cm, bottom: 2.54cm, left: 2.58cm, right: 2.08cm)
)

#set text(
  lang: "zh",
  font: FONT_SONG,
  size: zh(5),
  hyphenate: false,
  tracking: -0.3pt,
  cjk-latin-spacing: auto
)

#show heading.where(level: 2): it => {
  align(center, par(leading: 40pt, text(font: FONT_SONG, size: zh(4), it.body)))
}
`

// H5Block 存储五级标题及其内容
type H5Block struct {
	Title   string
	Content []string
}

// H4Block 存储四级标题及其下的所有五级标题块
type H4Block struct {
	Title    string
	H5Blocks []H5Block
}

// Table 存储一个三级标题定义的表格
type Table struct {
	H3Part1  string
	H3Part2  string
	H4Blocks []H4Block
}

// DocumentSection 存储一个二级标题定义的内容区域
type DocumentSection struct {
	H2Title string
	Tables  []Table
}

const templateMd = `## 教学活动设计——任务一

### 章节标题——任务描述

#### 教学活动标题

##### 1H

学习内容

学生活动

教师活动

教学方法与手段
`

func init() {
	flag.BoolVar(&isTemplate, "t", false, "生成空白模板文件 template.md")
	flag.BoolVar(&isPdf, "p", false, "生成 PDF 文件（需要安装 typst）")
	flag.BoolVar(&verbose, "v", false, "显示详细输出信息")
	flag.BoolVar(&help, "h", false, "显示帮助信息")
}

func printHelp() {
	fmt.Println("实操教案格式化生成器")
	fmt.Println("用法: shicaojiaoan [选项] [输入文件]")
	fmt.Println()
	fmt.Println("选项:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if help {
		printHelp()
		return
	}

	// 处理生成模板的情况
	if isTemplate {
		if err := ioutil.WriteFile("template.md", []byte(templateMd), 0644); err != nil {
			log.Fatalf("failed to generate template: %v", err)
		}
		if verbose {
			log.Printf("template file generated: template.md")
		}
		return
	}

	// 检查是否提供了输入文件
	if flag.NArg() == 0 {
		printHelp()
		return
	}

	inputFile := flag.Arg(0)
	source, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	// 解析并生成 typst
	sections := parseMarkdown(string(source))
	typstOutput := generateTypst(sections)

	outputFile := strings.TrimSuffix(inputFile, ".md") + ".typ"
	if err := ioutil.WriteFile(outputFile, []byte(typstOutput), 0644); err != nil {
		log.Fatalf("failed to write file: %v", err)
	}

	if verbose {
		log.Printf("converted %s to %s", inputFile, outputFile)
	}

	// PDF 生成处理
	if isPdf {
		if _, err := exec.LookPath("typst"); err != nil {
			log.Fatal("typst command not found, please install typst first")
		}

		cmd := exec.Command("typst", "compile", outputFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("PDF generation failed: %v", err)
		}

		pdfFile := strings.TrimSuffix(outputFile, ".typ") + ".pdf"
		if verbose {
			log.Printf("PDF file generated: %s", pdfFile)
		}
	}
}

// parseMarkdown 将 markdown 字符串解析为 DocumentSection 结构体切片
func parseMarkdown(content string) []DocumentSection {
	lines := strings.Split(content, "\n")
	var sections []DocumentSection
	var currentSection *DocumentSection
	var currentTable *Table
	var currentH4 *H4Block
	var currentH5 *H5Block

	for _, line := range lines {
		line = strings.TrimRight(line, "\r") // 兼容 Windows 换行符
		if strings.HasPrefix(line, "## ") {
			sections = append(sections, DocumentSection{H2Title: strings.TrimSpace(line[3:])})
			currentSection = &sections[len(sections)-1]
			currentTable = nil
			currentH4 = nil
			currentH5 = nil
		} else if strings.HasPrefix(line, "### ") {
			if currentSection == nil {
				continue
			}
			title := strings.TrimSpace(line[4:])

			var parts []string
			// Try splitting with various separators, from longest to shortest.
			separators := []string{"——", "—", " - ", "-"}
			found := false
			for _, sep := range separators {
				parts = strings.SplitN(title, sep, 2)
				if len(parts) == 2 {
					found = true
					break
				}
			}

			if !found {
				parts = []string{title, ""}
			}

			currentSection.Tables = append(currentSection.Tables, Table{H3Part1: strings.TrimSpace(parts[0]), H3Part2: strings.TrimSpace(parts[1])})
			currentTable = &currentSection.Tables[len(currentSection.Tables)-1]
			currentH4 = nil
			currentH5 = nil
		} else if strings.HasPrefix(line, "#### ") {
			if currentTable == nil {
				continue
			}
			title := strings.TrimSpace(line[5:])
			currentTable.H4Blocks = append(currentTable.H4Blocks, H4Block{Title: title})
			currentH4 = &currentTable.H4Blocks[len(currentTable.H4Blocks)-1]
			currentH5 = nil
		} else if strings.HasPrefix(line, "##### ") {
			if currentH4 == nil {
				continue
			}
			title := strings.TrimSpace(line[6:])
			currentH4.H5Blocks = append(currentH4.H5Blocks, H5Block{Title: title})
			currentH5 = &currentH4.H5Blocks[len(currentH4.H5Blocks)-1]
			// Initialize with one empty content block, ready to be filled.
			currentH5.Content = []string{""}
		} else {
			if currentH5 != nil {
				if strings.TrimSpace(line) == "" {
					// Empty line: if the current block has content, prepare for a new block.
					lastIdx := len(currentH5.Content) - 1
					if lastIdx >= 0 && currentH5.Content[lastIdx] != "" {
						currentH5.Content = append(currentH5.Content, "")
					}
				} else {
					// Non-empty line: append to the current block.
					lastIdx := len(currentH5.Content) - 1
					if lastIdx < 0 {
						currentH5.Content = append(currentH5.Content, "")
						lastIdx = 0
					}

					if currentH5.Content[lastIdx] == "" {
						currentH5.Content[lastIdx] = line
					} else {
						// 使用真实的换行符，Typst 会在内容块中将其渲染为换行。
						currentH5.Content[lastIdx] += "\n" + line
					}
				}
			}
		}
	}

	// Clean up trailing empty content block if any
	for i := range sections {
		for j := range sections[i].Tables {
			for k := range sections[i].Tables[j].H4Blocks {
				for l := range sections[i].Tables[j].H4Blocks[k].H5Blocks {
					h5 := &sections[i].Tables[j].H4Blocks[k].H5Blocks[l]
					if len(h5.Content) > 0 && h5.Content[len(h5.Content)-1] == "" {
						h5.Content = h5.Content[:len(h5.Content)-1]
					}
				}
			}
		}
	}

	return sections
}

// generateTypst 根据解析出的结构体生成 typst 格式字符串
func generateTypst(sections []DocumentSection) string {
	var sb strings.Builder
	sb.WriteString(preamble)

	for _, section := range sections {
		sb.WriteString(fmt.Sprintf("\n== %s\n\n", section.H2Title))

		if len(section.Tables) > 0 {
			sb.WriteString("#table(\n")
			sb.WriteString("  columns: (2.3cm, 4.2cm, auto, auto, 2.2cm, 1.1cm),\n")
			sb.WriteString("  stroke: 0.5pt,\n")
			sb.WriteString("  align: center + horizon,\n")

			for _, table := range section.Tables {
				// 表格第一行
				sb.WriteString(fmt.Sprintf("  [*学习环节*], [*%s*], [*学习单元*], table.cell(colspan: 3)[*%s*],\n", table.H3Part1, table.H3Part2))

				// 表格第二行
				sb.WriteString("  [教学活动], [学习内容], [学生活动], [教师活动], [教学方法与手段], [课时分配],\n")

				h4Counter := 1 // Reset for each table (H3)

				// 内容行
				for _, h4 := range table.H4Blocks {
					if len(h4.H5Blocks) == 0 {
						continue
					}
					// 为当前 H4 构建单元格内容矩阵（每行 5 列：content0, content1, content2, teachingMethods, h5.Title）
					nRows := len(h4.H5Blocks)
					cols := 5
					cellContents := make([][]string, nRows)
					for i := 0; i < nRows; i++ {
						h5 := h4.H5Blocks[i]
						cellContents[i] = make([]string, cols)
						cellContents[i][0] = getContentLine(h5.Content, 0)
						cellContents[i][1] = getContentLine(h5.Content, 1)
						cellContents[i][2] = getContentLine(h5.Content, 2)
						cellContents[i][3] = getContentLine(h5.Content, 3) // 教学方法，渲染时会替换换行
						cellContents[i][4] = h5.Title
					}

					// 初始化 rowspan 矩阵，默认每个单元格 rowspan = 1
					rowspans := make([][]int, nRows)
					for i := 0; i < nRows; i++ {
						rowspans[i] = make([]int, cols)
						for j := 0; j < cols; j++ {
							rowspans[i][j] = 1
						}
					}

					// 处理包含 "同上" 的单元格：与正上方起始单元格合并（递归合并链）
					for col := 0; col < cols; col++ {
						for i := 0; i < nRows; i++ {
							if strings.Contains(strings.TrimSpace(cellContents[i][col]), "同上") {
								// 找到上方最近的起始单元格（rowspan != 0）
								k := i - 1
								for k >= 0 && rowspans[k][col] == 0 {
									k--
								}
								if k >= 0 {
									rowspans[k][col]++
									rowspans[i][col] = 0 // 标记为已被合并，输出时跳过
								} else {
									// 若没有上方可合并的单元格（首行），保留为空字符串，不合并
									cellContents[i][col] = ""
									rowspans[i][col] = 1
								}
							}
						}
					}

					numberedH4Title := fmt.Sprintf("%d. %s", h4Counter, h4.Title)
					h4Counter++

					// 为每列在输出时维护独立序号计数器（H4 内重置）
					counterCol0, counterCol1, counterCol2 := 1, 1, 1

					// 输出每一行，依据 rowspans 决定是否输出或输出带 rowspan 的单元格
					for i := 0; i < nRows; i++ {
						// 第一列（H4 标题）只在第一行输出，并带有整体 rowspan
						if i == 0 {
							sb.WriteString(fmt.Sprintf("  table.cell(rowspan: %d)[%s],", nRows, numberedH4Title))
						}

						// 对应三列内容 + 教学方法 + 课时分配
						for col := 0; col < cols; col++ {
							rs := rowspans[i][col]
							if rs == 0 {
								// 被上方合并，跳过输出该单元格
								continue
							}

							content := cellContents[i][col]
							// 三列学习类型内容（0..2）：按列编号，调用 formatNumberedContent
							if col == 0 {
								formatted, newc := formatNumberedContent(content, counterCol0)
								counterCol0 = newc
								content = formatted
							} else if col == 1 {
								formatted, newc := formatNumberedContent(content, counterCol1)
								counterCol1 = newc
								content = formatted
							} else if col == 2 {
								formatted, newc := formatNumberedContent(content, counterCol2)
								counterCol2 = newc
								content = formatted
							} else if col == 3 {
								// 教学方法列，替换换行为双换行
								if strings.TrimSpace(content) != "" {
									content = strings.ReplaceAll(content, "\n", "\n\n")
								}
							}

							// 仅在 rowspan > 1 时使用 table.cell
							if rs > 1 {
								var attrs []string
								attrs = append(attrs, fmt.Sprintf("rowspan: %d", rs))
								if col >= 0 && col <= 2 && strings.TrimSpace(content) != "" {
									attrs = append(attrs, "align: left")
								}
								sb.WriteString(fmt.Sprintf("  table.cell(%s)[%s],", strings.Join(attrs, ", "), content))
							} else {
								// rowspan == 1 时，不使用 table.cell，对齐通过 align() 包裹
								if col >= 0 && col <= 2 && strings.TrimSpace(content) != "" {
									sb.WriteString(fmt.Sprintf("  align(left)[%s],", content))
								} else {
									sb.WriteString(fmt.Sprintf("  [%s],", content))
								}
							}
						}
						sb.WriteString("\n")
					}
				}
			}
			sb.WriteString(")\n")
		}
	}
	return sb.String()
}

// getContentLine 安全地获取内容行，如果行不存在则返回空字符串
func getContentLine(lines []string, index int) string {
	if index < len(lines) {
		return lines[index]
	}
	return ""
}

// formatNumberedContent formats content with numbering for each line.
func formatNumberedContent(content string, startCounter int) (string, int) {
	if content == "" {
		return "", startCounter
	}
	lines := strings.Split(content, "\n")
	var formattedLines []string
	counter := startCounter
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			formattedLines = append(formattedLines, fmt.Sprintf("%d. %s；", counter, line))
			counter++
		}
	}
	return strings.Join(formattedLines, "\n"), counter
}
