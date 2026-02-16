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
    // 一级标题：当作 title 方便从 Markdown 转换
    // 居中，段前 0 行段后 28.7 磅，行距固定值 35 磅，字体 FZXiaoBiaoSong-B05 字号 zh(2)，无序号，无首行缩进
    v(0pt) // 段前0行
    align(center)[
      #text(
        font: FONT_XBS,
        size: zh(2),
        weight: "bold",
      )[
        #set par(leading: 35pt - zh(2)) // 行距固定值35磅
        #body
      ]
    ]
    v(28.7pt) // 段后28.7磅
  } else if level == 2 {
    // 二级标题：首行缩进2字符，STHeiti 字号 zh(3)，使用 `一、` 作为序号
    h2-counter.step()
    h3-counter.update(0)
    h4-counter.update(1)
    h5-counter.update(1)
    text(
      font: FONT_HEI,
      size: zh(3),
    )[#context h2-counter.display("一、")#body]
  } else if level == 3 {
    // 三级标题：首行缩进2字符，STKaiti 字号 zh(3)，使用 `（一）` 作为序号
    h3-counter.step()
    h4-counter.update(1)
    h5-counter.update(1)

    let number = h3-counter.get().first()
    text(
      font: FONT_KAI,
      size: zh(3),
    )[#context h3-counter.display("（一）")#body]
  } else if level == 4 {
    // 四级标题：首行缩进2字符，STFangsong 字号 zh(3)，使用 `1.` 作为序号
    h4-counter.step()
    h5-counter.update(1)

    let number = h4-counter.get().first()
    text(
      size: zh(3),
    )[#number. #body]
  } else if level == 5 {
    // 五级标题：首行缩进2字符，STFangsong 字号 zh(3)，使用 `（1）` 作为序号
    h5-counter.step()

    let number = h5-counter.get().first()
    text(
      size: zh(3),
    )[（#number）#body]
  }
}

// 应用自定义标题样式
// 确保所有标题都与下一段内容保持在同一页（Sticky Behavior）
// 采用了 "Reservation" 技术：在标题块中包含一段不可见的高度（threshold），
// 强制排版引擎检查是否有足够空间容纳标题+后续内容。如果空间不足，整体换页。


// 应用自定义标题样式，并确保标题与下一段内容同页
#show heading: it => {
  // 一级标题（文档标题）保持原有样式，不应用 sticky 逻辑
  if it.level == 1 {
    custom-heading(it.level, it.body, numbering: it.numbering)
  } else {
    // 其他标题：应用 strict sticky 逻辑
    let spacing = 13.9pt
    let threshold = 3em // 预留给下一段的空间阈值

    block(
      sticky: true,
      above: spacing,
      below: spacing,
      {
        // "Reservation" 技术：包含标题+预留空间在不可中断块中
        block(
          custom-heading(it.level, it.body, numbering: it.numbering) + v(threshold),
          breakable: false,
        )
        v(-threshold)
      },
    )
  }
}

// 重置计数器在文档开始时
#h2-counter.update(0)
#h3-counter.update(0)
#h4-counter.update(0)
#h5-counter.update(0)

// 将列表项转换为普通段落以实现"续行顶格"
// 列表层级计数器，用于处理嵌套缩进
#let list-depth = state("list-depth", 0)

// 将列表项转换为普通段落以实现"续行顶格"
#let flush-left-list(it) = {
  // 1. 更新层级深度
  list-depth.update(d => d + 1)

  let is-enum = (it.func() == enum)
  let children = it.children

  // 2. 获取当前缩进状态（普通列表继承 2em，noindent 列表继承 0pt）
  //    并根据层级计算额外的块级缩进 (Left Padding)
  context {
    let depth = list-depth.get()
    // 第一层(depth=1)不需要额外padding，第二层(depth=2)需要 2em，以此类推
    let block-indent = if depth > 1 { 2em } else { 0pt }

    // 3. 计算枚举项数量，用于编号
    pad(left: block-indent, block({
      for (count, item) in children.enumerate(start: 1) {
        if item.func() == list.item or item.func() == enum.item {
          let marker = if is-enum {
            let pattern = if it.has("numbering") and it.numbering != auto { it.numbering } else { "1." }
            numbering(pattern, count)
          } else {
            if it.has("marker") and it.marker.len() > 0 { it.marker.at(0) } else { [•] }
          }

          // 4. 生成段落
          //    继承 first-line-indent（由外部环境决定，如 2em 或 0pt）
          //    强制 hanging-indent 为 0pt（实现续行左对齐）
          par(
            first-line-indent: par.first-line-indent,
            hanging-indent: 0pt,
          )[#marker#h(0.25em)#item.body]
        } else {
          item
        }
      }
    }))

    // 5. 恢复层级深度
    list-depth.update(d => d - 1)
  }
}

// 应用规则
#show list: flush-left-list
#show enum: flush-left-list

// 定义作者名称显示样式
#let name(name) = align(center, pad(bottom: 0.8em)[
  #text(font: FONT_KAI, size: zh(3))[#name]
])

