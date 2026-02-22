# 模板名称

这是一个示例模板的 README 文件，用于本地开发测试。

## 功能特点

- 支持 Markdown 转 Typst 自动排版
- 符合标准格式规范
- 支持自定义 frontmatter 字段

## 使用方式

在 Markdown 文件顶部添加 YAML frontmatter：

```yaml
---
title: 文档标题
author: 作者姓名
date: 2026-02-22
---
```

然后编写正文内容即可。

## 配置选项

| 字段       | 类型     | 默认值   | 说明         |
| ---------- | -------- | -------- | ------------ |
| title      | string   | 无       | 文档标题     |
| author     | string   | 无       | 作者姓名     |
| date       | date     | 今天     | 文档日期     |
| lang       | string   | zh-CN    | 文档语言     |

## 许可证

MIT License
