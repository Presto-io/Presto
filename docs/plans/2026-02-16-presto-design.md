# Presto 设计文档

Markdown → Typst → PDF 转换平台，支持可插拔模板生态。

## 架构概览

```
用户访问 → Svelte 前端（浏览器/Wails WebView）
         → Go API 服务（Docker 容器 / Wails 本地）
         → 模板二进制执行（stdin markdown → stdout typst）
         → typst compile → PDF
```

三个部署形态：
- **Docker 服务端**：公网部署，浏览器访问，手机/电脑通用
- **Wails 桌面版**：本地运行，复用同一套 Svelte 前端
- 桌面版可连远程服务器 API，也可用本地 Go 后端

## 模板系统

### 核心协议

模板是独立的可执行文件，语言无关（Go/Rust/Python/JS 等均可），遵循：

- **stdin** 接收 Markdown 内容
- **stdout** 输出 Typst 源码
- `--manifest` 参数输出模板元信息
- 退出码：0 成功，1 输入错误，2 内部错误

### manifest.json

```json
{
  "name": "gongwen",
  "displayName": "中国党政机关公文格式",
  "description": "符合 GB/T 9704-2012 标准的公文排版",
  "version": "1.0.0",
  "author": "mrered",
  "license": "MIT",
  "minPrestoVersion": "0.1.0",
  "frontmatterSchema": {
    "title": { "type": "string", "default": "请输入文字" },
    "author": { "type": "string", "default": "请输入文字" },
    "date": { "type": "string", "format": "YYYY-MM-DD" },
    "signature": { "type": "boolean", "default": false }
  }
}
```

`frontmatterSchema` 让前端能根据模板动态生成表单，用户不需要手写 YAML front-matter。

### 模板仓库规范

- 命名约定：`presto-template-{name}`
- GitHub topic 标签：`presto-template`
- 仓库包含源码 + GitHub Release 发布多平台二进制
- 官方模板在 `presto` 组织下，第三方模板在任意用户/组织下

### 模板发现与安装

通过 GitHub Search API 按 topic 搜索全网模板：

```
GET https://api.github.com/search/repositories?q=topic:presto-template
```

对每个仓库获取最新 Release，按服务端 GOOS/GOARCH 下载对应二进制。

### 模板自动更新

应用启动时并发检查已安装模板的 GitHub Release 最新版本，发现新版本自动下载替换。

### 安全机制

- 默认开启安全模式，仅显示官方模板
- 设置中提供"启用社区模板"开关
- 开启时弹出警告："社区模板由第三方开发者提供，未经官方审核，可能存在安全风险。请仅安装你信任的模板。"
- 确认后才可浏览和安装第三方模板

### CI/CD（goreleaser）

每个模板仓库使用 goreleaser + GitHub Actions，tag push 时自动构建多平台二进制：

```yaml
version: 2
project_name: presto-template-gongwen

builds:
  - id: template
    dir: .
    binary: presto-template-gongwen
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0

archives:
  - id: template
    builds: [template]
    format: tar.gz
    name_template: "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - manifest.json
      - LICENSE
```

## Go API 服务

### 端点

```
POST   /api/convert                     Markdown → Typst
POST   /api/compile                     Typst → PDF
POST   /api/convert-and-compile         Markdown → PDF
POST   /api/batch                       批量转换

GET    /api/templates                   已安装模板列表
GET    /api/templates/discover          从 GitHub 搜索可用模板
POST   /api/templates/{id}/install      安装模板
DELETE /api/templates/{id}              卸载模板
GET    /api/templates/{id}/manifest     获取模板元信息

GET    /api/health                      健康检查
```

### 本地文件结构

```
~/.presto/
├── config.json
├── templates/
│   ├── gongwen/
│   │   ├── presto-template-gongwen
│   │   └── manifest.json
│   └── jiaoan/
│       ├── presto-template-jiaoan
│       └── manifest.json
└── cache/
```

## Svelte 前端

### 页面

1. **编辑器页**：左右分栏，左 Markdown 编辑器（CodeMirror），右 PDF 预览。顶部模板选择器和下载按钮
2. **模板商店页**：浏览官方/社区模板，安装/卸载/更新
3. **批量转换页**：拖入多个 MD 文件，选模板，一键批量生成 PDF
4. **设置页**：通用设置、模板开发入口、关于、开源协议

### 预览方案

编辑时用 typst.ts（Typst 官方 WASM 渲染库）在前端直接渲染 Typst 为 SVG 预览，不走后端。下载 PDF 时才调后端 typst compile。

```
编辑时：Markdown → 后端模板转换 → Typst 源码 → 前端 typst.ts 渲染预览
下载时：Markdown → 后端模板转换 → 后端 typst compile → PDF
```

### 前端 API 抽象

```typescript
interface PrestoAPI {
  convert(markdown: string, templateId: string): Promise<string>
  compile(typst: string): Promise<Blob>
  listTemplates(): Promise<Template[]>
  installTemplate(repo: string): Promise<void>
}

class RemoteAPI implements PrestoAPI { /* HTTP 请求 */ }
class WailsAPI implements PrestoAPI { /* Go binding 调用 */ }
```

Web 版和 Wails 版通过此接口切换后端，前端代码完全复用。

### 设置页结构

```
设置
├── 通用
│   └── 社区模板开关（默认关闭 + 警告弹窗）
├── 模板开发
│   ├── 开发文档链接
│   ├── 模板协议说明（stdin/stdout、manifest.json 规范）
│   ├── 快速开始（各语言脚手架仓库链接）
│   └── 发布指南
├── 关于 Presto
│   ├── 版本号
│   ├── GitHub 仓库链接
│   └── Presto 开源协议
└── 开源协议声明
    └── 依赖列表（构建时自动生成）
```

开源协议通过 `go-licenses`（Go 依赖）和 `license-checker`（前端依赖）在构建时自动扫描生成。

## Docker 部署

```dockerfile
FROM golang:1.24-alpine AS builder
# 构建 Presto API 服务

FROM alpine:latest
# 安装 typst CLI
# 复制 API 服务二进制
# 复制官方模板二进制
EXPOSE 8080
CMD ["presto-server"]
```

后期扩容使用 docker-compose 或 Kubernetes，API 服务无状态，可水平扩展。

## Wails 桌面版（后期）

- `wails init -n presto-desktop -t svelte`
- 复用 Web 版 Svelte 前端代码
- Go 后端实现同样的 API 接口，模板在本地执行
- 可选连远程服务器或用本地后端
- 不受社区模板开关限制（本地运行，用户自负）

## 技术栈总结

| 组件 | 技术 |
|------|------|
| 前端 | Svelte + CodeMirror + typst.ts |
| 后端 | Go HTTP 服务 |
| 模板 | 独立可执行文件（语言无关） |
| 部署 | Docker |
| CI/CD | goreleaser + GitHub Actions |
| 桌面版 | Wails（后期） |
| 模板发现 | GitHub Search API + topic 标签 |
| 模板分发 | GitHub Releases |
