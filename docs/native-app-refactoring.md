# Presto 原生应用重构方案

> Markdown → Typst → PDF 一站式文档转换平台
> 从 Go + SvelteKit + Wails 迁移至 Swift + SwiftUI 原生多平台应用

---

## 目录

1. [项目现状分析](#1-项目现状分析)
2. [重构目标与范围](#2-重构目标与范围)
3. [技术选型](#3-技术选型)
4. [系统架构设计](#4-系统架构设计)
5. [项目结构](#5-项目结构)
6. [核心模块详细设计](#6-核心模块详细设计)
7. [平台适配策略](#7-平台适配策略)
8. [数据模型与持久化](#8-数据模型与持久化)
9. [模板系统重构](#9-模板系统重构)
10. [Typst 引擎集成](#10-typst-引擎集成)
11. [编辑器实现方案](#11-编辑器实现方案)
12. [预览系统实现](#12-预览系统实现)
13. [文件管理与文档架构](#13-文件管理与文档架构)
14. [iCloud 同步方案](#14-icloud-同步方案)
15. [网络层与 API 兼容](#15-网络层与-api-兼容)
16. [UI/UX 设计规范](#16-uiux-设计规范)
17. [键盘快捷键与菜单系统](#17-键盘快捷键与菜单系统)
18. [无障碍访问](#18-无障碍访问)
19. [性能优化策略](#19-性能优化策略)
20. [测试策略](#20-测试策略)
21. [分发与部署](#21-分发与部署)
22. [迁移路线图](#22-迁移路线图)
23. [风险评估与缓解](#23-风险评估与缓解)
24. [附录](#附录)

---

## 1. 项目现状分析

### 1.1 当前技术栈

| 层级 | 技术 | 版本 |
| ------ | ------ | ------ |
| 前端框架 | SvelteKit 2 + Svelte 5 (runes) | 5.49+ |
| 编辑器 | CodeMirror 6 + Markdown 扩展 | 6.0 |
| 图标 | Lucide Svelte | 0.564 |
| 后端 | Go 标准库 `net/http` | 1.25 |
| 桌面框架 | Wails v2 | 2.11 |
| 排版引擎 | Typst CLI | 0.14.2 |
| Markdown 解析 | Goldmark | - |
| 构建工具 | Vite 7 | 7.3 |
| 容器化 | Docker 多阶段构建 | - |

### 1.2 当前架构概览

```text
┌─────────────────────────────────────────────────┐
│             前端 (SvelteKit 2 + Svelte 5)        │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Editor   │  │ Preview  │  │ TemplateStore │  │
│  │(CodeMirror)│ │ (SVG)   │  │  (GitHub API) │  │
│  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │
│       └──────────────┴────────────────┘          │
│                      │ HTTP / Wails Binding       │
├──────────────────────┼───────────────────────────┤
│              Go HTTP API / Wails Backend          │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Template │  │  Typst   │  │   Template    │  │
│  │ Executor │  │ Compiler │  │   Manager     │  │
│  └──────────┘  └──────────┘  └───────────────┘  │
└─────────────────────────────────────────────────┘
```

### 1.3 核心数据流

1. 用户在 CodeMirror 编辑器中编写 Markdown
2. 前端调用 `POST /api/convert` → Go 后端调用模板二进制 (stdin/stdout) → 返回 Typst 源码
3. 前端调用 `POST /api/compile-svg` → Go 后端调用 Typst CLI → 返回 SVG 页面数组
4. 前端渲染 SVG 到预览区域
5. 导出时调用 `POST /api/compile` → 获取 PDF 二进制

### 1.4 现有功能清单

| 功能类别 | 具体功能 | 实现状态 |
| --------- | ---------- | --------- |
| **编辑器** | Markdown 语法高亮 | ✅ |
| | CodeMirror 6 基础编辑 | ✅ |
| | 中文本地化搜索面板 (查找/替换) | ✅ |
| | 自动换行 | ✅ |
| | Placeholder 提示 | ✅ |
| | 编辑器↔预览双向滚动同步 | ✅ |
| **预览** | SVG 多页渲染 | ✅ |
| | 实时预览 (500ms debounce) | ✅ |
| **模板系统** | 本地模板列表 | ✅ |
| | 模板切换 + 确认对话框 | ✅ |
| | 示例内容自动加载 | ✅ |
| | GitHub 模板商店发现 | ✅ |
| | 一键安装/卸载 | ✅ |
| | Manifest 元数据查看 | ✅ |
| **内置模板** | 公文格式 (gongwen) — GB/T 9704-2012 | ✅ |
| | 教案试操 (jiaoan-shicao) | ✅ |
| **导出** | PDF 导出 (Web 下载 / 桌面原生保存) | ✅ |
| | 智能文件名提取 (Typst 标题) | ✅ |
| **桌面端** | 原生菜单栏 (中文) | ✅ |
| | 原生文件打开对话框 | ✅ |
| | 原生 PDF 保存对话框 | ✅ |
| | 键盘快捷键 (⌘O, ⌘E, ⌘,) | ✅ |
| | macOS 隐藏标题栏 (Inset) | ✅ |
| **Web 端** | Docker 一键部署 | ✅ |
| | 浏览器文件上传回退 | ✅ |
| **设置** | 设置页面 | ✅ |
| | 模板管理页面 | ✅ |
| **其他** | 批量转换 | ❌ (501 未实现) |

### 1.5 现有 API 接口

| 方法 | 路径 | 功能 |
| ------ | ------ | ------ |
| `GET` | `/api/health` | 健康检查 |
| `POST` | `/api/convert` | Markdown → Typst |
| `POST` | `/api/compile` | Typst → PDF |
| `POST` | `/api/compile-svg` | Typst → SVG 页面 |
| `POST` | `/api/convert-and-compile` | Markdown → PDF 一步到位 |
| `GET` | `/api/templates` | 已安装模板列表 |
| `GET` | `/api/templates/discover` | GitHub 模板发现 |
| `POST` | `/api/templates/{id}/install` | 安装模板 |
| `DELETE` | `/api/templates/{id}` | 卸载模板 |
| `GET` | `/api/templates/{id}/manifest` | 模板元数据 |
| `GET` | `/api/templates/{id}/example` | 模板示例内容 |

### 1.6 现有数据模型

```text
Manifest {
    name: String              // 模板标识符
    displayName: String       // 显示名称
    description: String       // 描述
    version: String           // 版本号
    author: String            // 作者
    license: String           // 许可证
    minPrestoVersion: String  // 最低兼容版本
    frontmatterSchema: Map<String, FieldSchema>  // YAML Front Matter 字段定义
}

FieldSchema {
    type: String              // 字段类型
    default: Any?             // 默认值
    format: String?           // 格式
}

InstalledTemplate {
    manifest: Manifest        // 模板元数据
    binaryPath: String        // 可执行文件路径
    dir: String               // 模板目录路径
}
```

### 1.7 当前设计系统

- **配色**：深色主题为主 (Background: `#0F172A`, Primary: `#1E293B`, Accent: `#22C55E`)
- **字体**：JetBrains Mono (标题/代码) + IBM Plex Sans (正文)
- **风格**：Vibrant & Block-based，开发者工具/IDE 风格
- **图标**：Lucide icon set

---

## 2. 重构目标与范围

### 2.1 核心目标

1. **原生体验**：利用 SwiftUI 构建真正的原生 macOS/iPadOS/iOS 应用，而非 WebView 包装
2. **多平台统一代码库**：一套 Swift 代码，适配三大 Apple 平台
3. **功能对等**：保留现有所有功能，并利用原生能力增强
4. **性能提升**：消除 WebView 层的性能损耗，原生渲染更快更流畅
5. **生态融入**：深度集成 Apple 生态（iCloud、Shortcuts、Share Extension 等）

### 2.2 目标平台与最低版本

| 平台 | 最低版本 | 目标版本 | 说明 |
| ------ | --------- | --------- | ------ |
| macOS | 14.0 (Sonoma) | 15.0+ (Sequoia) | 主力开发平台 |
| iPadOS | 17.0 | 18.0+ | 触控优化，支持外接键盘 |
| iOS | 17.0 | 18.0+ | 移动端轻量使用 |

> **说明**：选择 macOS 14 / iOS 17 作为最低版本，以充分利用 SwiftData、新 Observable 宏、改进的 SwiftUI 导航 API 等现代特性。

### 2.3 重构范围

#### 必须重构的部分

- [ ] 前端 UI 层：SvelteKit → SwiftUI
- [ ] 编辑器：CodeMirror 6 → 原生文本编辑方案
- [ ] 预览渲染：HTML SVG → 原生 PDF/SVG 渲染
- [ ] 桌面框架：Wails → 原生 SwiftUI App
- [ ] 文件管理：浏览器 File API → 原生 DocumentGroup / FileDocument
- [ ] 菜单/快捷键：Wails menu → 原生 SwiftUI Commands
- [ ] 数据持久化：文件系统 → SwiftData + 文件系统

#### 需要重写/适配的部分

- [ ] 模板管理器：Go → Swift（管理逻辑重写）
- [ ] Typst 编译器封装：Go exec → Swift Process（macOS）/ 编译为库（iOS/iPadOS）
- [ ] API 客户端：Fetch API → URLSession / 本地调用
- [ ] Markdown → Typst 转换：Go Goldmark → Swift Markdown 解析

#### 可以复用的部分

- [x] 模板二进制协议（stdin/stdout）：macOS 上可直接复用
- [x] 模板 manifest.json 格式：JSON 结构不变
- [x] Typst 源码格式：排版引擎不变
- [x] YAML Front Matter 规范：格式不变
- [x] 公文/教案模板逻辑：可编译为 macOS 原生二进制或移植为 Swift

#### 新增功能（利用原生能力）

- [ ] iCloud 文档同步
- [ ] Shortcuts / Automator 集成
- [ ] Share Extension（从其他 App 分享文本到 Presto）
- [ ] Quick Look 预览扩展
- [ ] Spotlight 索引
- [ ] iPadOS 分屏多任务
- [ ] Apple Pencil 手写标注（iPadOS）
- [ ] Live Activity / Widget（显示最近文档）
- [ ] 拖拽支持（macOS/iPadOS）

---

## 3. 技术选型

### 3.1 总体技术栈

| 层级 | 选型 | 理由 |
| ------ | ------ | ------ |
| UI 框架 | **SwiftUI** | Apple 官方声明式 UI 框架，多平台统一，持续增强 |
| 编程语言 | **Swift 6+** | 完整并发安全、宏系统、性能优秀 |
| 数据持久化 | **SwiftData** | 现代化 ORM，与 SwiftUI 深度集成，替代 Core Data |
| 包管理 | **Swift Package Manager** | 官方原生方案，Xcode 深度集成 |
| 网络 | **URLSession + async/await** | 原生异步网络，无需第三方依赖 |
| Markdown 解析 | **swift-markdown (Apple)** | Apple 官方 Markdown 解析库，Swift 原生 |
| 排版引擎 | **Typst** | 保持不变，通过 Process / 编译为库集成 |
| 文本编辑 | **TextKit 2 + NSTextView/UITextView** | 原生文本排版引擎，完整的语法高亮能力 |
| PDF 渲染 | **PDFKit** | 原生 PDF 渲染，性能优异 |
| 测试 | **Swift Testing + XCTest** | 现代测试框架 |

### 3.2 关键依赖

| 依赖 | 用途 | SPM 包 |
| ------ | ------ | -------- |
| swift-markdown | Markdown AST 解析 | `apple/swift-markdown` |
| swift-syntax-highlight | 语法高亮 | 自建或社区方案 |
| Yams | YAML Front Matter 解析 | `jpsim/Yams` |
| typst-swift (自建) | Typst CLI 封装 | 项目内部 Package |
| KeyboardShortcuts | 全局快捷键 (macOS) | `sindresorhus/KeyboardShortcuts` |

### 3.3 为什么不选择其他方案

| 备选方案 | 不选择的理由 |
| --------- | ------------- |
| **AppKit/UIKit** | SwiftUI 已足够成熟 (2026)，声明式编程效率更高，多平台共享代码更方便 |
| **Catalyst** | iPadOS app 在 macOS 上体验不佳，不如原生 SwiftUI |
| **Electron / Tauri** | 不满足"原生应用"需求，依然是 WebView 方案 |
| **React Native** | 不支持 macOS（有限支持），非 Apple 原生 |
| **Flutter** | macOS 支持尚不成熟，非 Apple 原生体验 |
| **Core Data** | SwiftData 是其现代替代，API 更简洁，SwiftUI 集成更好 |

---

## 4. 系统架构设计

### 4.1 整体架构

采用 **Clean Architecture + MVVM** 混合架构，分层清晰、可测试性强：

```text
┌─────────────────────────────────────────────────────────┐
│                    Presentation Layer                     │
│           (SwiftUI Views + ViewModels)                   │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐  │
│  │ Editor   │  │ Preview  │  │ Template │  │Settings│  │
│  │  View    │  │  View    │  │  Store   │  │  View  │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───┬────┘  │
│       └──────────────┴─────────────┴────────────┘       │
│                          │                               │
├──────────────────────────┼───────────────────────────────┤
│                    Domain Layer                           │
│              (Models + Use Cases)                         │
│                                                          │
│  ┌──────────────┐  ┌───────────────┐  ┌──────────────┐  │
│  │ Document     │  │ Conversion    │  │  Template    │  │
│  │ Model        │  │ Pipeline      │  │  Model       │  │
│  └──────────────┘  └───────────────┘  └──────────────┘  │
│                          │                               │
├──────────────────────────┼───────────────────────────────┤
│                 Infrastructure Layer                      │
│           (Services + External Integrations)              │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐  │
│  │ Typst    │  │ Template │  │ GitHub   │  │  File  │  │
│  │ Engine   │  │ Executor │  │ Client   │  │ System │  │
│  └──────────┘  └──────────┘  └──────────┘  └────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 4.2 模块依赖关系

```text
PrestoApp (Application Target)
├── PrestoUI          // SwiftUI Views, Platform-specific UI
├── PrestoCore        // Domain Models, Use Cases, ViewModels
├── PrestoEngine      // Typst 编译、模板执行、Markdown 解析
├── PrestoTemplates   // 模板管理、GitHub 集成
└── PrestoCommon      // 共享工具、扩展、常量
```

### 4.3 数据流架构

采用**单向数据流**，结合 SwiftUI 的 `@Observable` 宏：

```text
User Action → ViewModel → Service → Engine → Result → ViewModel → View Update

具体到编辑-预览流程：
1. 用户输入 Markdown         → EditorViewModel.markdown 更新
2. Debounce (500ms)          → 触发转换 Pipeline
3. MarkdownToTypst 转换      → TemplateExecutor.convert(markdown)
4. TypstToSVG/PDF 编译       → TypstEngine.compile(typstSource)
5. 更新预览数据              → PreviewViewModel.pages 更新
6. SwiftUI 自动重渲染        → PreviewView 显示新内容
```

### 4.4 并发模型

使用 Swift Structured Concurrency：

```swift
// 转换 Pipeline 使用 Actor 隔离
@Observable
final class ConversionPipeline {
    private let executor: TemplateExecutor
    private let compiler: TypstCompiler

    // 使用 Task 管理异步转换，支持取消
    private var conversionTask: Task<Void, Never>?

    func convert(markdown: String, templateId: String) {
        conversionTask?.cancel()
        conversionTask = Task {
            try await Task.sleep(for: .milliseconds(500)) // debounce
            guard !Task.isCancelled else { return }

            let typst = try await executor.convert(markdown, template: templateId)
            guard !Task.isCancelled else { return }

            let pages = try await compiler.compileToSVG(typst)
            await MainActor.run {
                self.svgPages = pages
            }
        }
    }
}
```

---

## 5. 项目结构

### 5.1 Xcode 项目组织

```text
Presto/
├── Presto.xcodeproj              # 或 Package.swift (SPM-based)
├── App/
│   ├── PrestoApp.swift           # @main 入口
│   ├── ContentView.swift         # 根视图
│   ├── Info.plist
│   └── Assets.xcassets
│       ├── AppIcon.appiconset
│       └── Colors/
├── Sources/
│   ├── PrestoCore/               # 核心业务逻辑 (Platform-independent)
│   │   ├── Models/
│   │   │   ├── PrestoDocument.swift
│   │   │   ├── Template.swift
│   │   │   ├── Manifest.swift
│   │   │   └── FieldSchema.swift
│   │   ├── ViewModels/
│   │   │   ├── EditorViewModel.swift
│   │   │   ├── PreviewViewModel.swift
│   │   │   ├── TemplateStoreViewModel.swift
│   │   │   └── SettingsViewModel.swift
│   │   ├── Services/
│   │   │   ├── ConversionPipeline.swift
│   │   │   ├── TemplateManager.swift
│   │   │   └── DocumentManager.swift
│   │   └── Extensions/
│   │       ├── String+Extensions.swift
│   │       └── URL+Extensions.swift
│   │
│   ├── PrestoEngine/             # 编译引擎 (Platform-aware)
│   │   ├── TypstCompiler.swift
│   │   ├── TypstCompiler+macOS.swift
│   │   ├── TypstCompiler+iOS.swift
│   │   ├── TemplateExecutor.swift
│   │   ├── MarkdownParser.swift
│   │   └── MarkdownToTypst/
│   │       ├── GongwenConverter.swift
│   │       └── JiaoanShicaoConverter.swift
│   │
│   ├── PrestoTemplates/          # 模板管理
│   │   ├── TemplateStore.swift
│   │   ├── GitHubClient.swift
│   │   ├── TemplateInstaller.swift
│   │   └── ManifestParser.swift
│   │
│   ├── PrestoUI/                 # 共享 SwiftUI 视图
│   │   ├── Editor/
│   │   │   ├── MarkdownEditorView.swift
│   │   │   ├── MarkdownTextView.swift
│   │   │   ├── SyntaxHighlighter.swift
│   │   │   └── SearchPanel.swift
│   │   ├── Preview/
│   │   │   ├── DocumentPreview.swift
│   │   │   ├── SVGPageView.swift
│   │   │   └── PDFPreviewView.swift
│   │   ├── Templates/
│   │   │   ├── TemplateSelectorView.swift
│   │   │   ├── TemplateStoreView.swift
│   │   │   └── TemplateCard.swift
│   │   ├── Settings/
│   │   │   └── SettingsView.swift
│   │   ├── Shared/
│   │   │   ├── ToolbarView.swift
│   │   │   ├── SplitEditorView.swift
│   │   │   └── ErrorBanner.swift
│   │   └── Styles/
│   │       ├── PrestoTheme.swift
│   │       ├── Colors.swift
│   │       └── Typography.swift
│   │
│   ├── PrestoPlatform/           # 平台特定代码
│   │   ├── macOS/
│   │   │   ├── MacEditorView.swift
│   │   │   ├── MacMenuCommands.swift
│   │   │   └── MacWindowManager.swift
│   │   ├── iPadOS/
│   │   │   ├── iPadEditorView.swift
│   │   │   ├── iPadToolbar.swift
│   │   │   └── iPadSplitView.swift
│   │   └── iOS/
│   │       ├── iPhoneEditorView.swift
│   │       └── iPhoneCompactLayout.swift
│   │
│   └── PrestoCommon/             # 共享工具
│       ├── Logging.swift
│       ├── Constants.swift
│       └── FileUtilities.swift
│
├── Tests/
│   ├── PrestoCoreTests/
│   │   ├── ConversionPipelineTests.swift
│   │   ├── TemplateManagerTests.swift
│   │   └── ManifestParserTests.swift
│   ├── PrestoEngineTests/
│   │   ├── TypstCompilerTests.swift
│   │   ├── MarkdownParserTests.swift
│   │   └── GongwenConverterTests.swift
│   └── PrestoUITests/
│       └── SnapshotTests/
│
├── Resources/
│   ├── Templates/                # 内置模板
│   │   ├── gongwen/
│   │   │   ├── manifest.json
│   │   │   └── template_head.typ
│   │   └── jiaoan-shicao/
│   │       └── manifest.json
│   ├── Typst/                    # 捆绑的 Typst 二进制
│   │   └── typst                 # (macOS only, iOS 需要编译为库)
│   └── Localizable.xcstrings     # 本地化
│
└── Extensions/                   # App Extensions
    ├── ShareExtension/           # 分享扩展
    ├── QuickLookPreview/         # Quick Look 预览
    └── ShortcutsProvider/        # Shortcuts 集成
```

### 5.2 SPM Package 结构 (替代方案)

如果选择纯 SPM 驱动的项目结构：

```swift
// Package.swift
let package = Package(
    name: "Presto",
    platforms: [
        .macOS(.v14),
        .iOS(.v17)
    ],
    products: [
        .library(name: "PrestoCore", targets: ["PrestoCore"]),
        .library(name: "PrestoEngine", targets: ["PrestoEngine"]),
        .library(name: "PrestoTemplates", targets: ["PrestoTemplates"]),
        .library(name: "PrestoUI", targets: ["PrestoUI"]),
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-markdown", from: "0.5.0"),
        .package(url: "https://github.com/jpsim/Yams", from: "5.0.0"),
    ],
    targets: [
        .target(name: "PrestoCommon"),
        .target(name: "PrestoCore", dependencies: ["PrestoCommon"]),
        .target(name: "PrestoEngine", dependencies: [
            "PrestoCore",
            .product(name: "Markdown", package: "swift-markdown"),
            .product(name: "Yams", package: "Yams"),
        ]),
        .target(name: "PrestoTemplates", dependencies: ["PrestoCore"]),
        .target(name: "PrestoUI", dependencies: ["PrestoCore", "PrestoEngine", "PrestoTemplates"]),
        // Tests
        .testTarget(name: "PrestoCoreTests", dependencies: ["PrestoCore"]),
        .testTarget(name: "PrestoEngineTests", dependencies: ["PrestoEngine"]),
    ]
)
```

---

## 6. 核心模块详细设计

### 6.1 PrestoDocument — 文档模型

采用 SwiftUI 的 `ReferenceFileDocument` 协议，支持增量保存和 undo/redo：

```swift
import SwiftUI
import UniformTypeIdentifiers

/// Presto 文档类型定义
extension UTType {
    static let prestoDocument = UTType(exportedAs: "com.mrered.presto.document")
    static let markdownText = UTType("net.daringfireball.markdown")!
}

/// 核心文档模型
@Observable
final class PrestoDocument: ReferenceFileDocument {
    // MARK: - Document Content
    var markdown: String
    var selectedTemplateId: String
    var documentDirectory: URL?

    // MARK: - Derived State (不持久化)
    var typstSource: String = ""
    var previewPages: [PreviewPage] = []
    var isConverting: Bool = false
    var lastError: String?

    // MARK: - ReferenceFileDocument
    static var readableContentTypes: [UTType] { [.markdownText, .plainText] }
    static var writableContentTypes: [UTType] { [.markdownText] }

    required init(configuration: ReadConfiguration) throws {
        guard let data = configuration.file.regularFileContents,
              let text = String(data: data, encoding: .utf8) else {
            throw CocoaError(.fileReadCorruptFile)
        }
        self.markdown = text
        self.selectedTemplateId = ""
    }

    init() {
        self.markdown = ""
        self.selectedTemplateId = ""
    }

    func snapshot(contentType: UTType) throws -> Data {
        guard let data = markdown.data(using: .utf8) else {
            throw CocoaError(.fileWriteInapplicableStringEncoding)
        }
        return data
    }

    func fileWrapper(snapshot: Data, configuration: WriteConfiguration) throws -> FileWrapper {
        FileWrapper(regularFileWithContents: snapshot)
    }
}
```

### 6.2 ConversionPipeline — 转换管线

核心业务逻辑，管理 Markdown → Typst → PDF/SVG 的完整流程：

```swift
/// 转换管线 — 管理 Markdown → Typst → Preview 的异步流程
@Observable
final class ConversionPipeline {
    // MARK: - Dependencies
    private let templateManager: TemplateManager
    private let typstCompiler: TypstCompiler

    // MARK: - State
    var isConverting = false
    var lastError: String?
    var typstSource: String = ""
    var previewPages: [PreviewPage] = []

    // MARK: - Internal
    private var currentTask: Task<Void, Never>?
    private let debounceInterval: Duration = .milliseconds(500)

    init(templateManager: TemplateManager, typstCompiler: TypstCompiler) {
        self.templateManager = templateManager
        self.typstCompiler = typstCompiler
    }

    /// 触发转换（带 debounce）
    func convert(markdown: String, templateId: String, workDir: URL? = nil) {
        currentTask?.cancel()
        currentTask = Task { @MainActor in
            do {
                try await Task.sleep(for: debounceInterval)
                guard !Task.isCancelled else { return }

                isConverting = true
                lastError = nil

                // Step 1: Markdown → Typst
                let typst = try await templateManager.convert(
                    markdown: markdown,
                    templateId: templateId
                )
                guard !Task.isCancelled else { return }
                self.typstSource = typst

                // Step 2: Typst → SVG/PDF pages
                let pages = try await typstCompiler.compileToPreview(
                    source: typst,
                    workDirectory: workDir
                )
                guard !Task.isCancelled else { return }
                self.previewPages = pages

            } catch is CancellationError {
                // 正常取消，忽略
            } catch {
                self.lastError = error.localizedDescription
            }

            self.isConverting = false
        }
    }

    /// 导出 PDF
    func exportPDF(markdown: String, templateId: String, workDir: URL? = nil) async throws -> Data {
        let typst = try await templateManager.convert(
            markdown: markdown,
            templateId: templateId
        )
        return try await typstCompiler.compileToPDF(source: typst, workDirectory: workDir)
    }
}
```

### 6.3 TemplateManager — 模板管理器

```swift
/// 模板管理器
actor TemplateManager {
    private let templatesDirectory: URL
    private var installedTemplates: [String: InstalledTemplate] = [:]

    init() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        self.templatesDirectory = home
            .appendingPathComponent(".presto")
            .appendingPathComponent("templates")

        try? FileManager.default.createDirectory(
            at: templatesDirectory,
            withIntermediateDirectories: true
        )
    }

    /// 扫描并列出已安装模板
    func listTemplates() throws -> [InstalledTemplate] {
        let contents = try FileManager.default.contentsOfDirectory(
            at: templatesDirectory,
            includingPropertiesForKeys: nil
        )

        return try contents.compactMap { dir in
            let manifestURL = dir.appendingPathComponent("manifest.json")
            guard FileManager.default.fileExists(atPath: manifestURL.path) else { return nil }

            let data = try Data(contentsOf: manifestURL)
            let manifest = try JSONDecoder().decode(Manifest.self, from: data)

            let binaryName = "presto-template-\(manifest.name)"
            let binaryURL = dir.appendingPathComponent(binaryName)
            guard FileManager.default.fileExists(atPath: binaryURL.path) else { return nil }

            return InstalledTemplate(
                manifest: manifest,
                binaryURL: binaryURL,
                directory: dir
            )
        }
    }

    /// 通过模板执行器转换 Markdown
    func convert(markdown: String, templateId: String) async throws -> String {
        guard let template = try listTemplates().first(where: { $0.manifest.name == templateId }) else {
            throw PrestoError.templateNotFound(templateId)
        }

        let executor = TemplateExecutor(binaryURL: template.binaryURL)
        return try await executor.convert(markdown: markdown)
    }

    /// 获取模板示例内容
    func getExample(templateId: String) async throws -> String {
        guard let template = try listTemplates().first(where: { $0.manifest.name == templateId }) else {
            throw PrestoError.templateNotFound(templateId)
        }

        let executor = TemplateExecutor(binaryURL: template.binaryURL)
        return try await executor.getExample()
    }
}
```

---

## 7. 平台适配策略

### 7.1 条件编译

使用编译条件区分平台特定代码：

```swift
#if os(macOS)
    // macOS 特有代码
#elseif os(iOS)
    // iOS/iPadOS 共享代码
    #if targetEnvironment(macCatalyst)
        // Mac Catalyst 特定 (如有需要)
    #endif
#endif
```

### 7.2 各平台 UI 布局策略

#### macOS — 三栏式布局

```text
┌──────────────────────────────────────────────────┐
│  ← → ● ● ●        Presto        [▼ 公文]  [导出] │  ← 标题栏 + 工具栏
├───────────┬─────────────────┬────────────────────┤
│           │                 │                    │
│  文件导航  │   Markdown 编辑器 │   实时预览          │
│  Sidebar  │                 │   (PDF/SVG)        │
│           │                 │                    │
│  📄 文件1  │  ---            │   ┌─────────────┐  │
│  📄 文件2  │  title: 通知    │   │   Page 1    │  │
│  📄 文件3  │  ---            │   │             │  │
│           │                 │   │             │  │
│  模板      │  正文内容...     │   └─────────────┘  │
│  ⚙️ 设置   │                 │   ┌─────────────┐  │
│           │                 │   │   Page 2    │  │
├───────────┴─────────────────┴────────────────────┤
│  状态栏：字数 1234 | 模板: 公文 | Typst 0.14       │
└──────────────────────────────────────────────────┘
```

#### iPadOS — 自适应分栏

> **重要变化**：iPadOS 26 移除了 Split View 和 Slide Over，改为全新的窗口系统（支持 2-4 个窗口自由调整大小并排排列）。Stage Manager 保留。开发者需要适配新的多任务模型。

```text
横屏 (Split View):
┌──────────────────────┬────────────────────────┐
│    Markdown 编辑器     │      实时预览            │
│    (键盘输入)          │      (PDF/SVG)          │
│                      │                        │
└──────────────────────┴────────────────────────┘

竖屏 (Tab 切换):
┌─────────────────────────────┐
│  [编辑器]  [预览]            │  ← Segmented Control
│                             │
│    当前选中的视图内容         │
│                             │
└─────────────────────────────┘
```

#### iOS — 紧凑布局

```text
┌─────────────────────┐
│  ← Presto    [导出]  │
├─────────────────────┤
│  [编辑]  [预览]       │  ← Tab 切换
│                     │
│   编辑器或预览内容    │
│   (全屏，一次一个)    │
│                     │
│                     │
│                     │
├─────────────────────┤
│  ▼ 公文模板           │  ← Bottom sheet 选择模板
└─────────────────────┘
```

### 7.3 平台特性矩阵

| 特性 | macOS | iPadOS | iOS |
| ------ | ------- | -------- | ----- |
| 三栏布局 (Sidebar + Editor + Preview) | ✅ | ✅ (横屏) | ❌ |
| 双栏布局 (Editor + Preview) | ✅ | ✅ | ❌ |
| Tab 切换布局 | ❌ | ✅ (竖屏) | ✅ |
| 原生菜单栏 | ✅ | ❌ | ❌ |
| 键盘快捷键 | ✅ | ✅ (外接键盘) | ⚠️ 有限 |
| 文件拖拽打开 | ✅ | ✅ | ❌ |
| 模板二进制执行 (Process) | ✅ | ❌ | ❌ |
| 内置模板 (Swift 原生) | ✅ | ✅ | ✅ |
| Typst CLI 调用 | ✅ | ❌ | ❌ |
| Typst 库调用 (编译) | ✅ | ✅ | ✅ |
| Apple Pencil 标注 | ❌ | ✅ | ❌ |
| Split View 多任务 | ✅ | ✅ | ❌ |
| Stage Manager | ✅ | ✅ (M 系列) | ❌ |
| iCloud 同步 | ✅ | ✅ | ✅ |
| Share Extension | ✅ | ✅ | ✅ |
| Quick Look | ✅ | ✅ | ✅ |
| Spotlight | ✅ | ✅ | ✅ |
| Widget | ✅ | ✅ | ✅ |

---

## 8. 数据模型与持久化

### 8.1 SwiftData 模型

```swift
import SwiftData

@Model
final class RecentDocument {
    var fileURL: URL
    var lastOpened: Date
    var templateId: String
    var title: String  // 从文档中提取
    var wordCount: Int

    init(fileURL: URL, templateId: String = "", title: String = "", wordCount: Int = 0) {
        self.fileURL = fileURL
        self.lastOpened = .now
        self.templateId = templateId
        self.title = title
        self.wordCount = wordCount
    }
}

@Model
final class InstalledTemplateRecord {
    @Attribute(.unique) var name: String
    var displayName: String
    var version: String
    var author: String
    var installedDate: Date
    var source: TemplateSource  // .builtin, .github(owner, repo), .local

    init(name: String, displayName: String, version: String, author: String, source: TemplateSource) {
        self.name = name
        self.displayName = displayName
        self.version = version
        self.author = author
        self.installedDate = .now
        self.source = source
    }
}

@Model
final class UserPreferences {
    var defaultTemplateId: String
    var editorFontSize: Double
    var editorTheme: EditorTheme
    var autoSaveEnabled: Bool
    var previewZoomLevel: Double

    init() {
        self.defaultTemplateId = "gongwen"
        self.editorFontSize = 14
        self.editorTheme = .dark
        self.autoSaveEnabled = true
        self.previewZoomLevel = 1.0
    }
}
```

### 8.2 值类型模型 (非持久化)

```swift
/// 模板 Manifest — 从 JSON 解析
struct Manifest: Codable, Identifiable {
    var id: String { name }

    let name: String
    let displayName: String
    let description: String
    let version: String
    let author: String
    let license: String?
    let minPrestoVersion: String?
    let frontmatterSchema: [String: FieldSchema]?
}

struct FieldSchema: Codable {
    let type: String
    let defaultValue: AnyCodable?
    let format: String?

    enum CodingKeys: String, CodingKey {
        case type
        case defaultValue = "default"
        case format
    }
}

/// 已安装模板 (运行时)
struct InstalledTemplate: Identifiable {
    var id: String { manifest.name }
    let manifest: Manifest
    let binaryURL: URL
    let directory: URL
}

/// GitHub 仓库
struct GitHubRepo: Codable, Identifiable {
    var id: String { fullName }

    let fullName: String
    let description: String?
    let htmlURL: String
    let owner: GitHubOwner
    let name: String

    enum CodingKeys: String, CodingKey {
        case fullName = "full_name"
        case description
        case htmlURL = "html_url"
        case owner
        case name
    }
}

struct GitHubOwner: Codable {
    let login: String
}

/// 预览页面
struct PreviewPage: Identifiable {
    let id: Int       // 页码
    let content: PageContent

    enum PageContent {
        case svg(String)       // SVG 字符串
        case pdf(CGPDFPage)    // PDF 页
        case image(CGImage)    // 渲染后的图像
    }
}
```

### 8.3 ModelContainer 配置

```swift
@main
struct PrestoApp: App {
    let container: ModelContainer

    init() {
        let schema = Schema([
            RecentDocument.self,
            InstalledTemplateRecord.self,
            UserPreferences.self,
        ])
        let configuration = ModelConfiguration(
            "Presto",
            schema: schema,
            cloudKitDatabase: .automatic  // 自动 iCloud 同步
        )
        self.container = try! ModelContainer(
            for: schema,
            configurations: [configuration]
        )
    }

    var body: some Scene {
        DocumentGroup(newDocument: PrestoDocument()) { file in
            ContentView(document: file.$document)
        }
        .modelContainer(container)

        #if os(macOS)
        Settings {
            SettingsView()
        }
        .modelContainer(container)
        #endif
    }
}
```

---

## 9. 模板系统重构

### 9.1 现有模板架构

当前模板系统基于**外部可执行文件**的插件架构：

```text
输入: Markdown (stdin) → 模板二进制 → 输出: Typst (stdout)
参数: --manifest → 输出 manifest.json
参数: --example  → 输出示例 Markdown
```

### 9.2 重构策略：混合架构

由于 iOS/iPadOS 不允许执行外部二进制，需要采用**混合策略**：

| 策略 | macOS | iPadOS/iOS | 说明 |
| ------ | ------- | ----------- | ------ |
| **内置模板** | Swift 原生实现 | Swift 原生实现 | 公文、教案等核心模板直接用 Swift 重写 |
| **外部模板** | Process 执行 | ❌ 不支持 | 仅 macOS 保留现有 stdin/stdout 协议 |
| **ExtensionKit 插件** | ✅ macOS 13+ | ❌ | Apple 推荐的现代插件架构，独立进程，XPC 通信 |
| **WASM 模板** | WKWebView 执行 | WKWebView 执行 | 未来方向：模板编译为 WASM |

> **ExtensionKit 说明**：macOS 13 引入的 [ExtensionKit](https://developer.apple.com/documentation/extensionkit) 允许定义自定义扩展点。扩展在独立沙箱进程中运行，通过 XPC 通信，可呈现 UI。这是 Apple 推荐的现代插件系统方案，比直接 `Process` 调用更安全、更符合 App Store 要求。

### 9.3 模板协议 (Swift Protocol)

```swift
/// 模板协议 — 所有模板必须遵循
protocol PrestoTemplate: Sendable {
    /// 模板元数据
    var manifest: Manifest { get }

    /// 转换 Markdown 为 Typst
    func convert(markdown: String) async throws -> String

    /// 获取示例 Markdown
    func exampleMarkdown() -> String
}
```

### 9.4 内置模板实现 (以公文为例)

```swift
/// 公文模板 — GB/T 9704-2012
struct GongwenTemplate: PrestoTemplate {
    let manifest = Manifest(
        name: "gongwen",
        displayName: "中国党政机关公文格式",
        description: "符合 GB/T 9704-2012 标准的公文排版",
        version: "1.0.0",
        author: "mrered",
        license: "MIT",
        minPrestoVersion: "1.0.0",
        frontmatterSchema: [
            "title": FieldSchema(type: "string", defaultValue: nil, format: nil),
            "author": FieldSchema(type: "string", defaultValue: nil, format: nil),
            "date": FieldSchema(type: "string", defaultValue: nil, format: "date"),
            "signature": FieldSchema(type: "boolean", defaultValue: nil, format: nil),
        ]
    )

    func convert(markdown: String) async throws -> String {
        // 1. 解析 YAML Front Matter
        let (frontMatter, body) = try parseFrontMatter(markdown)

        // 2. 生成 Typst 头部 (页面设置、字体等)
        var typst = generateTypstHeader(frontMatter: frontMatter)

        // 3. 解析 Markdown AST 并转换为 Typst
        let document = try Document(parsing: body)
        typst += convertToTypst(document: document, frontMatter: frontMatter)

        // 4. 处理签名
        if frontMatter.signature {
            typst += generateSignature(author: frontMatter.author, date: frontMatter.date)
        }

        return typst
    }

    func exampleMarkdown() -> String {
        // 返回内嵌的示例文本
        return """
        ---
        title: "关于开展2025年度安全生产专项检查工作的通知"
        author: "安全生产管理处"
        date: "2025-03-15"
        signature: true
        ---

        各部门、各单位：

        为进一步加强安全生产管理...
        """
    }
}
```

### 9.5 外部模板执行器 (仅 macOS)

```swift
#if os(macOS)
/// 外部模板执行器 — 通过 Process 调用外部二进制
actor ExternalTemplateExecutor {
    let binaryURL: URL

    init(binaryURL: URL) {
        self.binaryURL = binaryURL
    }

    func convert(markdown: String) async throws -> String {
        let process = Process()
        process.executableURL = binaryURL

        let inputPipe = Pipe()
        let outputPipe = Pipe()
        let errorPipe = Pipe()

        process.standardInput = inputPipe
        process.standardOutput = outputPipe
        process.standardError = errorPipe

        try process.run()

        // Write markdown to stdin
        if let data = markdown.data(using: .utf8) {
            inputPipe.fileHandleForWriting.write(data)
        }
        inputPipe.fileHandleForWriting.closeFile()

        // Read typst from stdout
        let outputData = outputPipe.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()

        guard process.terminationStatus == 0 else {
            let errorData = errorPipe.fileHandleForReading.readDataToEndOfFile()
            let errorMessage = String(data: errorData, encoding: .utf8) ?? "Unknown error"
            throw PrestoError.templateExecutionFailed(errorMessage)
        }

        guard let result = String(data: outputData, encoding: .utf8) else {
            throw PrestoError.invalidOutput
        }

        return result
    }

    func getManifest() async throws -> Manifest {
        let process = Process()
        process.executableURL = binaryURL
        process.arguments = ["--manifest"]

        let outputPipe = Pipe()
        process.standardOutput = outputPipe

        try process.run()
        let data = outputPipe.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()

        return try JSONDecoder().decode(Manifest.self, from: data)
    }

    func getExample() async throws -> String {
        let process = Process()
        process.executableURL = binaryURL
        process.arguments = ["--example"]

        let outputPipe = Pipe()
        process.standardOutput = outputPipe

        try process.run()
        let data = outputPipe.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()

        return String(data: data, encoding: .utf8) ?? ""
    }
}
#endif
```

### 9.6 GitHub 模板商店

```swift
/// GitHub 模板发现与安装
actor GitHubTemplateStore {
    private let session = URLSession.shared
    private let searchURL = "https://api.github.com/search/repositories"

    func discover() async throws -> [GitHubRepo] {
        var components = URLComponents(string: searchURL)!
        components.queryItems = [
            URLQueryItem(name: "q", value: "topic:presto-template"),
            URLQueryItem(name: "sort", value: "stars"),
            URLQueryItem(name: "order", value: "desc"),
        ]

        let (data, _) = try await session.data(from: components.url!)
        let result = try JSONDecoder().decode(GitHubSearchResult.self, from: data)
        return result.items
    }

    #if os(macOS)
    func install(owner: String, repo: String) async throws {
        // 1. 获取最新 Release
        let releaseURL = URL(string: "https://api.github.com/repos/\(owner)/\(repo)/releases/latest")!
        let (data, _) = try await session.data(from: releaseURL)
        let release = try JSONDecoder().decode(GitHubRelease.self, from: data)

        // 2. 查找匹配当前平台的 Asset
        let arch = ProcessInfo.processInfo.machineHardwareName  // arm64 / x86_64
        let platform = "darwin"
        let assetName = "presto-template-\(repo)-\(platform)-\(arch)"

        guard let asset = release.assets.first(where: { $0.name.contains(assetName) }) else {
            throw PrestoError.noCompatibleAsset
        }

        // 3. 下载并安装
        let (downloadedURL, _) = try await session.download(from: URL(string: asset.browserDownloadURL)!)

        let templateDir = templatesDirectory.appendingPathComponent(repo)
        try FileManager.default.createDirectory(at: templateDir, withIntermediateDirectories: true)

        let binaryDest = templateDir.appendingPathComponent("presto-template-\(repo)")
        try FileManager.default.moveItem(at: downloadedURL, to: binaryDest)

        // 设置可执行权限
        try FileManager.default.setAttributes(
            [.posixPermissions: 0o755],
            ofItemAtPath: binaryDest.path
        )
    }
    #endif
}
```

---

## 10. Typst 引擎集成

### 10.1 集成方案对比

| 方案 | macOS | iOS/iPadOS | 优势 | 劣势 |
| ------ | ------- | ----------- | ------ | ------ |
| **A: CLI 调用** | ✅ | ❌ | 简单直接，与现有方案一致 | iOS 不支持；沙箱中需嵌入二进制 |
| **B: typst-rs 编译为库** | ✅ | ✅ | 全平台统一，无沙箱问题，性能最佳 | 需要 Rust → C → Swift FFI |
| **C: WASM + WKWebView** | ✅ | ✅ | 跨平台，沙箱友好 | 性能略差，复杂度高 |
| **D: 混合方案 (推荐)** | CLI | 编译库 | 各取所长 | 维护两套代码 |

> **沙箱注意事项**：Mac App Store 要求 App Sandbox。沙箱应用的子进程继承父进程的沙箱限制，无法访问 Homebrew 安装的 `typst`。必须将 Typst 二进制嵌入 App Bundle (参考 [Apple: Embedding a helper tool in a sandboxed app](https://developer.apple.com/documentation/xcode/embedding-a-helper-tool-in-a-sandboxed-app))，或直接编译为静态库链接。
>
> **推荐生产方案**：Typst 是 Apache 2.0 协议的 Rust 项目，`typst` crate 可从 crates.io 获取。通过 [UniFFI](https://mozilla.github.io/uniffi-rs/) 自动生成 Swift 绑定，或使用 `cbindgen` 生成 C 头文件后通过 Swift C interop 调用。这消除了子进程开销、沙箱限制和用户安装 Typst 的依赖。参考项目：[Ghostty 的 Rust + SwiftUI 集成](https://dfrojas.com/software/integrating-Rust-and-SwiftUI.html)。

### 10.2 推荐方案：混合集成

#### macOS — Process 调用 Typst CLI

```swift
#if os(macOS)
/// macOS: 通过 Process 调用 Typst CLI
final class TypstCLICompiler: TypstCompiling {
    let binaryPath: String

    init(binaryPath: String? = nil) {
        if let path = binaryPath {
            self.binaryPath = path
        } else {
            self.binaryPath = Self.findTypstBinary()
        }
    }

    func compileToPDF(source: String, workDirectory: URL?) async throws -> Data {
        let tempDir = workDirectory ?? FileManager.default.temporaryDirectory
        let inputFile = tempDir.appendingPathComponent(".presto-temp-input.typ")
        let outputFile = tempDir.appendingPathComponent(".presto-temp-output.pdf")

        try source.write(to: inputFile, atomically: true, encoding: .utf8)
        defer {
            try? FileManager.default.removeItem(at: inputFile)
            try? FileManager.default.removeItem(at: outputFile)
        }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: binaryPath)
        process.arguments = ["compile", inputFile.path, outputFile.path]
        if let workDir = workDirectory {
            process.arguments! += ["--root", workDir.path]
        }

        let errorPipe = Pipe()
        process.standardError = errorPipe

        try process.run()
        process.waitUntilExit()

        guard process.terminationStatus == 0 else {
            let errorData = errorPipe.fileHandleForReading.readDataToEndOfFile()
            let errorMsg = String(data: errorData, encoding: .utf8) ?? "Typst compilation failed"
            throw PrestoError.compilationFailed(errorMsg)
        }

        return try Data(contentsOf: outputFile)
    }

    func compileToSVG(source: String, workDirectory: URL?) async throws -> [String] {
        let tempDir = workDirectory ?? FileManager.default.temporaryDirectory
        let inputFile = tempDir.appendingPathComponent(".presto-temp-input.typ")
        let outputPattern = tempDir.appendingPathComponent(".presto-temp-output-{n}.svg")

        try source.write(to: inputFile, atomically: true, encoding: .utf8)
        defer { try? FileManager.default.removeItem(at: inputFile) }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: binaryPath)
        process.arguments = [
            "compile", inputFile.path,
            outputPattern.path.replacingOccurrences(of: "{n}", with: "{0}"),
            "--format", "svg"
        ]
        if let workDir = workDirectory {
            process.arguments! += ["--root", workDir.path]
        }

        try process.run()
        process.waitUntilExit()

        // Collect SVG files
        var pages: [String] = []
        for i in 1... {
            let svgFile = tempDir.appendingPathComponent(
                ".presto-temp-output-\(i).svg"
            )
            guard FileManager.default.fileExists(atPath: svgFile.path) else { break }
            let svg = try String(contentsOf: svgFile, encoding: .utf8)
            pages.append(svg)
            try? FileManager.default.removeItem(at: svgFile)
        }

        // Fallback: single page without number suffix
        if pages.isEmpty {
            let svgFile = tempDir.appendingPathComponent(".presto-temp-output.svg")
            if FileManager.default.fileExists(atPath: svgFile.path) {
                let svg = try String(contentsOf: svgFile, encoding: .utf8)
                pages.append(svg)
                try? FileManager.default.removeItem(at: svgFile)
            }
        }

        return pages
    }

    /// 查找 Typst 二进制路径
    private static func findTypstBinary() -> String {
        // 1. App Bundle 内
        if let bundlePath = Bundle.main.path(forResource: "typst", ofType: nil) {
            return bundlePath
        }

        // 2. 可执行文件旁边
        let executableDir = Bundle.main.bundleURL
            .deletingLastPathComponent()
        let beside = executableDir.appendingPathComponent("typst")
        if FileManager.default.fileExists(atPath: beside.path) {
            return beside.path
        }

        // 3. Homebrew 常见路径
        for path in ["/opt/homebrew/bin/typst", "/usr/local/bin/typst"] {
            if FileManager.default.fileExists(atPath: path) {
                return path
            }
        }

        // 4. 系统 PATH
        if let path = try? Process.run(
            URL(fileURLWithPath: "/usr/bin/which"),
            arguments: ["typst"]
        ) {
            return path.trimmingCharacters(in: .whitespacesAndNewlines)
        }

        return "typst"
    }
}
#endif
```

#### iOS/iPadOS — Typst 编译为 Swift 库

```swift
#if os(iOS)
/// iOS/iPadOS: 通过 typst-swift 库直接编译
/// 需要将 typst (Rust) 通过 UniFFI / cbindgen 编译为 xcframework
final class TypstLibCompiler: TypstCompiling {

    func compileToPDF(source: String, workDirectory: URL?) async throws -> Data {
        // 调用 typst-swift 绑定
        // 注意：需要预先将 Typst Rust 代码编译为 iOS static library
        return try await withCheckedThrowingContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                do {
                    let result = typst_compile_to_pdf(source)  // C FFI 调用
                    continuation.resume(returning: result)
                } catch {
                    continuation.resume(throwing: error)
                }
            }
        }
    }

    func compileToSVG(source: String, workDirectory: URL?) async throws -> [String] {
        return try await withCheckedThrowingContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                do {
                    let result = typst_compile_to_svg(source)  // C FFI 调用
                    continuation.resume(returning: result)
                } catch {
                    continuation.resume(throwing: error)
                }
            }
        }
    }
}
#endif
```

### 10.3 Typst 编译协议

```swift
/// Typst 编译器协议
protocol TypstCompiling: Sendable {
    func compileToPDF(source: String, workDirectory: URL?) async throws -> Data
    func compileToSVG(source: String, workDirectory: URL?) async throws -> [String]
}

extension TypstCompiling {
    /// 编译为预览页面
    func compileToPreview(source: String, workDirectory: URL?) async throws -> [PreviewPage] {
        let svgStrings = try await compileToSVG(source: source, workDirectory: workDirectory)
        return svgStrings.enumerated().map { index, svg in
            PreviewPage(id: index, content: .svg(svg))
        }
    }
}
```

### 10.4 Typst Rust → Swift 桥接方案

要在 iOS 上使用 Typst，需要将 Typst 的 Rust 代码编译为 C 静态库：

```text
Typst (Rust)
    ↓ cargo build --target aarch64-apple-ios --release
C Static Library (.a)
    ↓ cbindgen / UniFFI
C Header (typst.h)
    ↓ Xcode Framework / xcframework
Swift Package (typst-swift)
    ↓ import
Presto App
```

**步骤概要**：

1. Fork typst/typst，添加 C FFI 接口层
2. 使用 `cargo-lipo` 或手动交叉编译为 iOS/macOS static library
3. 创建 `typst.xcframework` 包含所有架构
4. 包装为 Swift Package，暴露类型安全的 Swift API

---

## 11. 编辑器实现方案

### 11.1 方案对比

| 方案 | 复杂度 | 功能完整度 | 性能 | 推荐度 |
| ------ | -------- | ----------- | ------ | -------- |
| **SwiftUI TextEditor** | 低 | ❌ 极有限 | 好 | ❌ 不推荐 |
| **UITextView/NSTextView + TextKit 2** | 高 | ✅ 完整 | 优秀 | ⚠️ 可选 |
| **CodeEditorView (开源库)** | 中 | ✅ 较完整 | 好 | ⚠️ 可选 |
| **STTextView (开源库)** | 中 | ✅ 较完整 | 中 | ⚠️ 可选 |
| **WKWebView + CodeMirror 6** | 中 | ✅ 最完整 | 中等 | ✅ **推荐 (务实选择)** |
| **自定义 Canvas 渲染** | 极高 | ✅ 完整 | 最佳 | ❌ 过度工程 |

> **重要发现**：开源项目 [MarkEdit](https://github.com/MarkEdit-app/MarkEdit) (macOS Markdown 编辑器，MIT 协议) 明确论证了 "TextKit is not better than contentEditable"，其选择 CodeMirror 6 + WKWebView 作为编辑器方案，仅 4 MB 体积即可打开 10 MB 文件。这是目前最务实的生产级方案。

### 11.1.1 开源编辑器库参考

| 库 | 底层技术 | 特点 | 许可证 |
| --- | --- | --- | --- |
| [CodeEditorView](https://github.com/mchakravarty/CodeEditorView) | TextKit 2 | SwiftUI 原生，语法高亮，括号匹配，minimap (macOS) | MIT-like |
| [STTextView](https://github.com/krzyzanowskim/STTextView) | TextKit 2 | 模块化插件架构，Neon 语法高亮插件 | GPL v3 / 商业 |
| [MarkEdit](https://github.com/MarkEdit-app/MarkEdit) | CodeMirror 6 + WKWebView | 最成熟方案，JavaScript 桥接 | MIT |

> **注意**：CodeEdit 项目曾使用 STTextView 但因大文件性能问题而放弃，转向自建 CoreText 渲染器。TextKit 2 虽然是苹果的方向，但截至 2025 年仍有不少已知 bug。

### 11.2 推荐方案 A：WKWebView + CodeMirror 6 (务实首选)

复用现有项目的 CodeMirror 6 编辑器代码，通过 WKWebView 嵌入 SwiftUI。参考 MarkEdit 项目的成熟实现：

```swift
/// WKWebView 嵌入 CodeMirror 6 编辑器 (推荐方案)
/// 参考: https://github.com/MarkEdit-app/MarkEdit
struct CodeMirrorEditorView: NSViewRepresentable {  // 或 UIViewRepresentable
    @Binding var text: String
    var onTextChange: ((String) -> Void)?
    var onScrollChange: ((CGFloat) -> Void)?

    func makeNSView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        let handler = context.coordinator
        config.userContentController.add(handler, name: "textChanged")
        config.userContentController.add(handler, name: "scrollChanged")

        let webView = WKWebView(frame: .zero, configuration: config)
        // 加载内嵌的 CodeMirror HTML
        if let htmlURL = Bundle.main.url(forResource: "editor", withExtension: "html") {
            webView.loadFileURL(htmlURL, allowingReadAccessTo: htmlURL.deletingLastPathComponent())
        }
        return webView
    }

    // JavaScript ↔ Swift 桥接通信
    class Coordinator: NSObject, WKScriptMessageHandler {
        var parent: CodeMirrorEditorView

        init(_ parent: CodeMirrorEditorView) {
            self.parent = parent
        }

        func userContentController(_ controller: WKUserContentController,
                                   didReceive message: WKScriptMessage) {
            switch message.name {
            case "textChanged":
                if let text = message.body as? String {
                    parent.text = text
                    parent.onTextChange?(text)
                }
            case "scrollChanged":
                if let ratio = message.body as? Double {
                    parent.onScrollChange?(CGFloat(ratio))
                }
            default: break
            }
        }
    }
}
```

**优势**：

- 完全复用现有 CodeMirror 6 代码 (语法高亮、搜索替换、中文本地化)
- 功能最完整 (多光标、代码折叠、括号匹配等)
- MarkEdit 已验证此方案可用于生产级 macOS 应用
- 编辑器 HTML/JS/CSS 打包在 App Bundle 中，无网络依赖

**劣势**：

- 非完全原生体验 (无法直接使用 NSTextView 的所有 API)
- 需要 JavaScript 桥接，调试略复杂

### 11.3 推荐方案 B：TextKit 2 原生编辑器 (纯原生)

#### 架构概述

```text
MarkdownEditorView (SwiftUI)
    └── MarkdownTextViewRepresentable (UIViewRepresentable / NSViewRepresentable)
        └── MarkdownTextView (NSTextView / UITextView)
            ├── NSTextLayoutManager (TextKit 2)
            ├── NSTextContentManager
            ├── MarkdownSyntaxHighlighter
            └── MarkdownCompletionProvider
```

#### SwiftUI 包装

```swift
/// SwiftUI 包装的 Markdown 编辑器
struct MarkdownEditorView: View {
    @Binding var text: String
    var onTextChange: ((String) -> Void)?
    var onScrollChange: ((CGFloat) -> Void)?

    @State private var scrollRatio: CGFloat = 0

    var body: some View {
        #if os(macOS)
        MacMarkdownTextView(
            text: $text,
            onTextChange: onTextChange,
            onScrollChange: onScrollChange
        )
        #else
        iOSMarkdownTextView(
            text: $text,
            onTextChange: onTextChange,
            onScrollChange: onScrollChange
        )
        #endif
    }
}

#if os(macOS)
struct MacMarkdownTextView: NSViewRepresentable {
    @Binding var text: String
    var onTextChange: ((String) -> Void)?
    var onScrollChange: ((CGFloat) -> Void)?

    func makeNSView(context: Context) -> NSScrollView {
        let scrollView = NSTextView.scrollableTextView()
        let textView = scrollView.documentView as! NSTextView

        // 基本配置
        textView.font = NSFont.monospacedSystemFont(ofSize: 14, weight: .regular)
        textView.isAutomaticQuoteSubstitutionEnabled = false
        textView.isAutomaticDashSubstitutionEnabled = false
        textView.isRichText = false
        textView.allowsUndo = true
        textView.usesFindBar = true

        // TextKit 2 语法高亮
        let highlighter = MarkdownSyntaxHighlighter()
        textView.textContentStorage?.delegate = highlighter

        textView.delegate = context.coordinator
        return scrollView
    }

    func updateNSView(_ scrollView: NSScrollView, context: Context) {
        let textView = scrollView.documentView as! NSTextView
        if textView.string != text {
            textView.string = text
        }
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(text: $text, onTextChange: onTextChange, onScrollChange: onScrollChange)
    }

    class Coordinator: NSObject, NSTextViewDelegate {
        @Binding var text: String
        var onTextChange: ((String) -> Void)?
        var onScrollChange: ((CGFloat) -> Void)?

        init(text: Binding<String>, onTextChange: ((String) -> Void)?, onScrollChange: ((CGFloat) -> Void)?) {
            self._text = text
            self.onTextChange = onTextChange
            self.onScrollChange = onScrollChange
        }

        func textDidChange(_ notification: Notification) {
            guard let textView = notification.object as? NSTextView else { return }
            text = textView.string
            onTextChange?(text)
        }
    }
}
#endif
```

#### 语法高亮

```swift
/// Markdown 语法高亮器 — 基于 TextKit 2
final class MarkdownSyntaxHighlighter: NSObject, NSTextContentStorageDelegate {

    // 高亮规则
    private let rules: [(pattern: NSRegularExpression, style: HighlightStyle)] = [
        // 标题
        (try! NSRegularExpression(pattern: "^#{1,6}\\s.+$", options: .anchorsMatchLines),
         .heading),
        // 粗体
        (try! NSRegularExpression(pattern: "\\*\\*[^*]+\\*\\*"),
         .bold),
        // 斜体
        (try! NSRegularExpression(pattern: "(?<![*])\\*[^*]+\\*(?![*])"),
         .italic),
        // 代码块
        (try! NSRegularExpression(pattern: "```[\\s\\S]*?```"),
         .codeBlock),
        // 行内代码
        (try! NSRegularExpression(pattern: "`[^`]+`"),
         .inlineCode),
        // 链接
        (try! NSRegularExpression(pattern: "\\[([^\\]]+)\\]\\(([^)]+)\\)"),
         .link),
        // YAML Front Matter
        (try! NSRegularExpression(pattern: "^---[\\s\\S]*?---", options: .anchorsMatchLines),
         .frontMatter),
    ]

    enum HighlightStyle {
        case heading, bold, italic, codeBlock, inlineCode, link, frontMatter

        var attributes: [NSAttributedString.Key: Any] {
            switch self {
            case .heading:
                return [.foregroundColor: NSColor.systemBlue, .font: NSFont.boldSystemFont(ofSize: 16)]
            case .bold:
                return [.foregroundColor: NSColor.labelColor, .font: NSFont.boldSystemFont(ofSize: 14)]
            case .italic:
                return [.foregroundColor: NSColor.labelColor, .obliqueness: 0.2]
            case .codeBlock:
                return [.foregroundColor: NSColor.systemGreen, .backgroundColor: NSColor.systemGray.withAlphaComponent(0.1)]
            case .inlineCode:
                return [.foregroundColor: NSColor.systemOrange, .backgroundColor: NSColor.systemGray.withAlphaComponent(0.1)]
            case .link:
                return [.foregroundColor: NSColor.systemCyan, .underlineStyle: NSUnderlineStyle.single.rawValue]
            case .frontMatter:
                return [.foregroundColor: NSColor.systemPurple]
            }
        }
    }
}
```

### 11.3 备选方案：WKWebView + CodeMirror

如果原生 TextKit 2 方案过于复杂，可以在 SwiftUI 中嵌入 WKWebView 运行 CodeMirror 6：

```swift
/// WKWebView 嵌入 CodeMirror 6 编辑器 (备选方案)
struct CodeMirrorWebView: UIViewRepresentable {
    @Binding var text: String
    var onTextChange: ((String) -> Void)?

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "textChanged")

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.loadHTMLString(codeMirrorHTML, baseURL: Bundle.main.bundleURL)
        return webView
    }

    // ... JavaScript 桥接通信
}
```

> **注意**：此方案虽然功能最完整（完全复用现有 CodeMirror 6 代码），但违背了"原生体验"的重构目标。建议仅作为过渡方案。

---

## 12. 预览系统实现

### 12.1 SVG 渲染方案

> **重要限制**：Apple 没有公开的运行时 SVG 渲染 API（仅 Asset Catalog 中的 SVG 支持通过 CGSVGDocument 私有 API 实现）。社区方案包括 [SVGView (Exyte)](https://github.com/exyte/SVGView) 纯 SwiftUI 渲染和 WKWebView 渲染。对于 Typst 输出的复杂 SVG（含渐变、非系统字体），**推荐直接编译为 PDF 使用 PDFKit 渲染**，或使用 WKWebView 作为 SVG 渲染后端。

```swift
/// SVG 页面渲染视图
struct SVGPageView: View {
    let svgContent: String
    let pageNumber: Int

    var body: some View {
        #if os(macOS)
        // macOS: 使用 NSImage 渲染 SVG
        if let image = NSImage(data: svgContent.data(using: .utf8)!) {
            Image(nsImage: image)
                .resizable()
                .aspectRatio(contentMode: .fit)
                .background(.white)
                .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: 2)
                .padding(8)
        }
        #else
        // iOS: 使用 WKWebView 渲染 SVG (最可靠的方案)
        SVGWebView(svgContent: svgContent)
            .aspectRatio(210.0/297.0, contentMode: .fit)  // A4 比例
            .background(.white)
            .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: 2)
            .padding(8)
        #endif
    }
}
```

### 12.2 PDF 预览方案 (更推荐)

直接将 Typst 编译为 PDF 而非 SVG，使用 PDFKit 渲染：

```swift
import PDFKit

/// PDF 预览视图 — 使用 PDFKit
struct PDFPreviewView: View {
    let pdfData: Data

    var body: some View {
        #if os(macOS)
        PDFKitView(data: pdfData)
        #else
        PDFKitView(data: pdfData)
        #endif
    }
}

#if os(macOS)
struct PDFKitView: NSViewRepresentable {
    let data: Data

    func makeNSView(context: Context) -> PDFView {
        let pdfView = PDFView()
        pdfView.autoScales = true
        pdfView.displayMode = .singlePageContinuous
        pdfView.displaysPageBreaks = true
        pdfView.backgroundColor = .gray.withAlphaComponent(0.3)
        return pdfView
    }

    func updateNSView(_ pdfView: PDFView, context: Context) {
        if let document = PDFDocument(data: data) {
            pdfView.document = document
        }
    }
}
#else
struct PDFKitView: UIViewRepresentable {
    let data: Data

    func makeUIView(context: Context) -> PDFView {
        let pdfView = PDFView()
        pdfView.autoScales = true
        pdfView.displayMode = .singlePageContinuous
        pdfView.displaysPageBreaks = true
        return pdfView
    }

    func updateUIView(_ pdfView: PDFView, context: Context) {
        if let document = PDFDocument(data: data) {
            pdfView.document = document
        }
    }
}
#endif
```

### 12.3 双向滚动同步

```swift
/// 编辑器-预览 滚动同步管理
@Observable
final class ScrollSyncManager {
    enum Source { case editor, preview }

    var editorScrollRatio: CGFloat = 0
    var previewScrollRatio: CGFloat = 0
    private var activeSource: Source?

    func editorDidScroll(to ratio: CGFloat) {
        guard activeSource != .preview else { return }
        activeSource = .editor
        editorScrollRatio = ratio
        previewScrollRatio = ratio

        // 防止回弹
        Task { @MainActor in
            try? await Task.sleep(for: .milliseconds(100))
            activeSource = nil
        }
    }

    func previewDidScroll(to ratio: CGFloat) {
        guard activeSource != .editor else { return }
        activeSource = .preview
        previewScrollRatio = ratio
        editorScrollRatio = ratio

        Task { @MainActor in
            try? await Task.sleep(for: .milliseconds(100))
            activeSource = nil
        }
    }
}
```

---

## 13. 文件管理与文档架构

### 13.1 Document-Based App

使用 SwiftUI 的 `DocumentGroup` 构建文档型应用：

```swift
@main
struct PrestoApp: App {
    @Environment(\.openDocument) private var openDocument

    var body: some Scene {
        // 文档场景 — 主编辑界面
        DocumentGroup(newDocument: PrestoDocument()) { config in
            ContentView(document: config.$document)
        }
        .commands {
            PrestoMenuCommands()
        }

        #if os(macOS)
        // 设置窗口
        Settings {
            SettingsView()
        }

        // 模板商店窗口
        Window("模板商店", id: "template-store") {
            TemplateStoreView()
        }
        .defaultSize(width: 800, height: 600)
        #endif
    }
}
```

### 13.2 文件类型注册

在 `Info.plist` 中注册支持的文件类型：

```xml
<key>CFBundleDocumentTypes</key>
<array>
    <dict>
        <key>CFBundleTypeName</key>
        <string>Markdown Document</string>
        <key>CFBundleTypeRole</key>
        <string>Editor</string>
        <key>LSItemContentTypes</key>
        <array>
            <string>net.daringfireball.markdown</string>
        </array>
    </dict>
    <dict>
        <key>CFBundleTypeName</key>
        <string>Text Document</string>
        <key>CFBundleTypeRole</key>
        <string>Editor</string>
        <key>LSItemContentTypes</key>
        <array>
            <string>public.plain-text</string>
        </array>
    </dict>
</array>

<key>UTExportedTypeDeclarations</key>
<array>
    <dict>
        <key>UTTypeIdentifier</key>
        <string>com.mrered.presto.document</string>
        <key>UTTypeDescription</key>
        <string>Presto Document</string>
        <key>UTTypeConformsTo</key>
        <array>
            <string>net.daringfireball.markdown</string>
        </array>
        <key>UTTypeTagSpecification</key>
        <dict>
            <key>public.filename-extension</key>
            <array>
                <string>md</string>
                <string>markdown</string>
            </array>
        </dict>
    </dict>
</array>
```

### 13.3 文件打开与保存

```swift
// macOS 使用 NSOpenPanel / NSSavePanel (通过 DocumentGroup 自动处理)
// iOS 使用 UIDocumentPickerViewController (通过 DocumentGroup 自动处理)

// 自定义文件导出 (PDF)
struct ContentView: View {
    @Binding var document: PrestoDocument
    @State private var showExporter = false
    @State private var pdfData: Data?

    var body: some View {
        // ... 编辑器和预览视图 ...
        .fileExporter(
            isPresented: $showExporter,
            document: PDFExportDocument(data: pdfData ?? Data()),
            contentType: .pdf,
            defaultFilename: extractTitle()
        ) { result in
            switch result {
            case .success(let url):
                print("PDF exported to \(url)")
            case .failure(let error):
                print("Export failed: \(error)")
            }
        }
    }
}
```

---

## 14. iCloud 同步方案

### 14.1 同步策略

| 数据类型 | 同步方案 | 说明 |
| --------- | --------- | ------ |
| 文档文件 (.md) | **iCloud Drive / DocumentGroup** | 自动同步，用户可选保存位置 |
| 用户偏好设置 | **NSUbiquitousKeyValueStore** | 轻量 KV 同步，自动 |
| 最近文档记录 | **SwiftData + CloudKit** | 通过 ModelConfiguration 自动同步 |
| 模板安装记录 | **SwiftData + CloudKit** | 仅同步记录，二进制不同步 |

### 14.2 iCloud Drive 文档同步

```swift
// DocumentGroup 自动支持 iCloud Drive
// 用户在 "文件" App 中可以看到 Presto 的文档目录
// 需要在 Entitlements 中启用：
// com.apple.developer.icloud-container-identifiers = ["iCloud.com.mrered.presto"]
// com.apple.developer.ubiquity-container-identifiers = ["iCloud.com.mrered.presto"]
```

### 14.3 设置同步

```swift
/// 跨设备设置同步
final class CloudSettings {
    static let shared = CloudSettings()
    private let store = NSUbiquitousKeyValueStore.default

    var defaultTemplateId: String {
        get { store.string(forKey: "defaultTemplateId") ?? "gongwen" }
        set { store.set(newValue, forKey: "defaultTemplateId") }
    }

    var editorFontSize: Double {
        get { store.double(forKey: "editorFontSize").nonZero ?? 14.0 }
        set { store.set(newValue, forKey: "editorFontSize") }
    }

    func synchronize() {
        store.synchronize()
    }
}
```

---

## 15. 网络层与 API 兼容

### 15.1 保留 Web API (可选)

如果需要保留 Web 端部署能力或远程服务器模式：

```swift
/// API 客户端 — 连接远程 Presto 服务器
actor PrestoAPIClient {
    let baseURL: URL

    init(baseURL: URL) {
        self.baseURL = baseURL
    }

    func convert(markdown: String, templateId: String) async throws -> String {
        var request = URLRequest(url: baseURL.appendingPathComponent("/api/convert"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(
            ConvertRequest(markdown: markdown, templateId: templateId)
        )

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw PrestoError.apiError
        }

        let result = try JSONDecoder().decode(ConvertResponse.self, from: data)
        return result.typst
    }

    // ... 其他 API 方法 ...
}
```

### 15.2 本地 vs 远程模式

```swift
/// 统一的服务层 — 根据配置选择本地或远程执行
protocol PrestoService {
    func convert(markdown: String, templateId: String) async throws -> String
    func compile(typstSource: String) async throws -> Data
    func listTemplates() async throws -> [InstalledTemplate]
}

/// 本地模式（原生执行）
final class LocalPrestoService: PrestoService { /* ... */ }

/// 远程模式（HTTP API 调用）
final class RemotePrestoService: PrestoService { /* ... */ }
```

---

## 16. UI/UX 设计规范

### 16.1 Apple 原生设计原则迁移

| 当前设计 (Web) | 原生设计 (SwiftUI) |
| --------------- | ------------------- |
| 自定义暗色主题 (#0F172A) | 系统暗色模式 (自适应) |
| JetBrains Mono 字体 | SF Mono (系统等宽) / 可配置 |
| IBM Plex Sans 正文 | SF Pro (系统字体) |
| Lucide 图标 | SF Symbols |
| 自定义 CSS 变量 | SwiftUI Color.accentColor |
| 自定义按钮样式 | .buttonStyle(.bordered) 等系统样式 |

### 16.2 配色方案

```swift
/// Presto 配色 — 适配系统明暗模式
extension Color {
    static let prestoAccent = Color("AccentColor")  // Assets 中定义，明暗两套
    static let prestoBackground = Color(nsColor: .windowBackgroundColor)  // 自动适配
    static let prestoSurface = Color(nsColor: .controlBackgroundColor)
    static let prestoText = Color(nsColor: .labelColor)
    static let prestoMuted = Color(nsColor: .secondaryLabelColor)

    // 编辑器特定颜色
    static let editorBackground = Color("EditorBackground")
    static let editorGutter = Color("EditorGutter")
}
```

### 16.3 Typography

```swift
/// 排版规范
struct PrestoTypography {
    // 编辑器字体
    static let editorFont: Font = .system(size: 14, design: .monospaced)
    static let editorLineHeight: CGFloat = 1.6

    // UI 字体
    static let toolbarTitle: Font = .headline
    static let sidebarItem: Font = .body
    static let statusBar: Font = .caption

    // 预览标注字体
    static let pageNumber: Font = .caption2.monospacedDigit()
}
```

### 16.4 主界面实现 (macOS)

```swift
struct ContentView: View {
    @Binding var document: PrestoDocument
    @StateObject private var pipeline = ConversionPipeline(...)
    @StateObject private var scrollSync = ScrollSyncManager()
    @State private var showSidebar = true

    var body: some View {
        NavigationSplitView {
            // Sidebar
            SidebarView()
        } detail: {
            HSplitView {
                // 编辑器面板
                MarkdownEditorView(
                    text: $document.markdown,
                    onTextChange: { text in
                        pipeline.convert(
                            markdown: text,
                            templateId: document.selectedTemplateId
                        )
                    },
                    onScrollChange: { ratio in
                        scrollSync.editorDidScroll(to: ratio)
                    }
                )
                .frame(minWidth: 300)

                // 预览面板
                DocumentPreview(
                    pages: pipeline.previewPages,
                    scrollRatio: scrollSync.previewScrollRatio,
                    onScrollChange: { ratio in
                        scrollSync.previewDidScroll(to: ratio)
                    }
                )
                .frame(minWidth: 300)
            }
        }
        .toolbar {
            PrestoToolbar(
                document: $document,
                pipeline: pipeline
            )
        }
        .navigationTitle(document.displayTitle)
        .navigationSubtitle(document.selectedTemplateId.isEmpty ? "" : "模板: \(document.selectedTemplateName)")
    }
}
```

---

## 17. 键盘快捷键与菜单系统

### 17.1 SwiftUI Commands

```swift
/// Presto 菜单命令
struct PrestoMenuCommands: Commands {
    @FocusedBinding(\.document) var document

    var body: some Commands {
        // 替换默认的 "文件" 菜单项
        CommandGroup(after: .newItem) {
            Button("打开 Markdown…") {
                // 通过 FocusedValue 传递动作
            }
            .keyboardShortcut("o")
        }

        // 导出菜单
        CommandGroup(after: .saveItem) {
            Button("导出 PDF…") {
                document?.exportPDF()
            }
            .keyboardShortcut("e")
        }

        // 编辑菜单扩展
        CommandMenu("排版") {
            Button("切换模板…") { }
                .keyboardShortcut("t")

            Divider()

            Button("编译并预览") { }
                .keyboardShortcut("r")
        }

        // 视图菜单
        CommandGroup(after: .sidebar) {
            Button("显示预览") { }
                .keyboardShortcut("p", modifiers: [.command, .shift])

            Button("仅编辑器") { }
                .keyboardShortcut("1", modifiers: [.command, .control])

            Button("仅预览") { }
                .keyboardShortcut("2", modifiers: [.command, .control])

            Button("分栏视图") { }
                .keyboardShortcut("3", modifiers: [.command, .control])
        }
    }
}
```

### 17.2 快捷键映射

| 快捷键 | 功能 | 平台 |
| -------- | ------ | ------ |
| `⌘O` | 打开 Markdown 文件 | 全平台 |
| `⌘S` | 保存文档 | 全平台 (DocumentGroup 自动) |
| `⌘E` | 导出 PDF | 全平台 |
| `⌘,` | 打开设置 | macOS (系统自动) |
| `⌘F` | 编辑器内搜索 | 全平台 |
| `⌘T` | 切换模板 | 全平台 |
| `⌘R` | 编译并预览 | 全平台 |
| `⌘⇧P` | 显示/隐藏预览 | macOS |
| `⌃⌘1` | 仅编辑器视图 | macOS |
| `⌃⌘2` | 仅预览视图 | macOS |
| `⌃⌘3` | 分栏视图 | macOS |
| `⌘Z` | 撤销 | 全平台 (系统自动) |
| `⌘⇧Z` | 重做 | 全平台 (系统自动) |

---

## 18. 无障碍访问

### 18.1 VoiceOver 支持

```swift
// 所有视图添加无障碍标签
MarkdownEditorView(text: $document.markdown)
    .accessibilityLabel("Markdown 编辑器")
    .accessibilityHint("在此输入 Markdown 内容")

DocumentPreview(pages: pipeline.previewPages)
    .accessibilityLabel("文档预览")
    .accessibilityHint("显示排版后的文档，共 \(pipeline.previewPages.count) 页")

// 模板选择器
TemplateSelectorView(selected: $document.selectedTemplateId)
    .accessibilityLabel("模板选择器")
    .accessibilityValue(document.selectedTemplateName)
```

### 18.2 Dynamic Type

```swift
// 编辑器字体跟随系统设置
@Environment(\.sizeCategory) var sizeCategory

var editorFont: Font {
    .system(
        size: UIFontMetrics.default.scaledValue(for: 14),
        design: .monospaced
    )
}
```

### 18.3 Reduce Motion

```swift
@Environment(\.accessibilityReduceMotion) var reduceMotion

// 条件动画
.animation(reduceMotion ? nil : .easeInOut(duration: 0.2), value: isConverting)
```

---

## 19. 性能优化策略

### 19.1 编辑器性能

- **增量更新**：仅高亮变化的文本区域，而非全文重新解析
- **异步语法高亮**：在后台线程执行正则匹配，结果回到主线程应用
- **懒加载**：长文档只渲染可见区域

### 19.2 预览性能

- **增量编译**：Typst 支持增量编译，缓存未变化的页面
- **分页加载**：使用 `LazyVStack` 仅渲染可见页面
- **图片缓存**：SVG 渲染结果缓存为位图

```swift
/// 懒加载预览
struct DocumentPreview: View {
    let pages: [PreviewPage]

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 12) {
                ForEach(pages) { page in
                    SVGPageView(page: page)
                        .id(page.id)
                }
            }
            .padding()
        }
    }
}
```

### 19.3 内存管理

- **弱引用**：ViewModel 不强持有 View
- **任务取消**：用户停止输入时取消进行中的编译任务
- **PDF 数据流**：导出大型 PDF 时使用流式写入，避免整个文件驻留内存

### 19.4 编译性能基准

| 操作 | 当前 (Go + Typst CLI) | 预期 (Swift + Typst) |
| ------ | ---------------------- | --------------------- |
| Markdown → Typst | ~10ms | ~5ms (Swift 原生更快) |
| Typst → SVG (1 页) | ~50ms | ~50ms (同为 CLI) |
| Typst → PDF (1 页) | ~30ms | ~30ms |
| 整体预览延迟 | ~560ms (含 500ms debounce) | ~555ms |

---

## 20. 测试策略

### 20.1 单元测试

```swift
import Testing
@testable import PrestoCore

@Suite("Manifest 解析")
struct ManifestTests {
    @Test("解析完整 manifest.json")
    func parseFullManifest() throws {
        let json = """
        {
            "name": "gongwen",
            "displayName": "中国党政机关公文格式",
            "version": "1.0.0",
            "author": "mrered"
        }
        """
        let data = json.data(using: .utf8)!
        let manifest = try JSONDecoder().decode(Manifest.self, from: data)

        #expect(manifest.name == "gongwen")
        #expect(manifest.displayName == "中国党政机关公文格式")
    }

    @Test("缺少必填字段应抛出错误")
    func parseMissingRequiredField() {
        let json = """
        { "name": "test" }
        """
        let data = json.data(using: .utf8)!
        #expect(throws: DecodingError.self) {
            try JSONDecoder().decode(Manifest.self, from: data)
        }
    }
}

@Suite("转换管线")
struct ConversionPipelineTests {
    @Test("Markdown 转换为 Typst")
    func convertMarkdownToTypst() async throws {
        let template = GongwenTemplate()
        let markdown = """
        ---
        title: 测试标题
        author: 测试
        ---

        ## 第一节

        正文内容。
        """
        let typst = try await template.convert(markdown: markdown)
        #expect(typst.contains("测试标题"))
    }
}
```

### 20.2 UI 测试

```swift
import XCTest

final class PrestoUITests: XCTestCase {
    func testEditorCanType() throws {
        let app = XCUIApplication()
        app.launch()

        let editor = app.textViews["Markdown 编辑器"]
        editor.tap()
        editor.typeText("# 测试标题")

        XCTAssertTrue(editor.value as? String == "# 测试标题")
    }

    func testExportPDF() throws {
        let app = XCUIApplication()
        app.launch()

        // 输入内容
        let editor = app.textViews["Markdown 编辑器"]
        editor.tap()
        editor.typeText("# Hello\n\n正文内容")

        // 选择模板
        app.popUpButtons["模板选择器"].tap()
        app.menuItems["公文"].tap()

        // 导出
        app.buttons["导出 PDF"].tap()
        // 验证保存对话框出现
        XCTAssertTrue(app.sheets.firstMatch.waitForExistence(timeout: 5))
    }
}
```

### 20.3 测试覆盖目标

| 模块 | 覆盖率目标 | 重点 |
| ------ | ----------- | ------ |
| PrestoCore | 90%+ | 数据模型、业务逻辑 |
| PrestoEngine | 85%+ | 转换管线、编译器封装 |
| PrestoTemplates | 80%+ | 模板解析、安装/卸载 |
| PrestoUI | 70%+ | 关键交互流程 |

---

## 21. 分发与部署

### 21.1 分发渠道

| 渠道 | 平台 | 优势 | 劣势 |
| ------ | ------ | ------ | ------ |
| **Mac App Store** | macOS | 广泛触达、自动更新 | 审核限制、沙箱约束 |
| **TestFlight** | 全平台 | Beta 测试分发 | 仅测试用途 |
| **直接分发 (DMG)** | macOS | 无审核、完整权限 | 需公证 (Notarization) |
| **App Store** | iOS/iPadOS | 唯一分发渠道 | 审核限制 |

### 21.2 沙箱适配

Mac App Store 要求 App Sandbox，需注意：

```xml
<!-- Entitlements -->
<key>com.apple.security.app-sandbox</key>
<true/>
<key>com.apple.security.files.user-selected.read-write</key>
<true/>
<key>com.apple.security.network.client</key>
<true/>
<!-- 如需执行外部二进制 (模板) -->
<key>com.apple.security.temporary-exception.mach-lookup.global-name</key>
<array>
    <string>com.mrered.presto.template-service</string>
</array>
```

> **重要**：App Sandbox 限制了 `Process` 的使用。如果要在沙箱中执行外部模板二进制，需要使用 **XPC Service** 架构。

### 21.3 XPC Service 方案 (沙箱兼容的模板执行)

```swift
/// XPC Service: 在沙箱外执行模板二进制
// PrestoTemplateService (XPC Service Target)
class TemplateServiceDelegate: NSObject, NSXPCListenerDelegate {
    func listener(_ listener: NSXPCListener,
                  shouldAcceptNewConnection connection: NSXPCConnection) -> Bool {
        connection.exportedInterface = NSXPCInterface(with: TemplateServiceProtocol.self)
        connection.exportedObject = TemplateService()
        connection.resume()
        return true
    }
}

@objc protocol TemplateServiceProtocol {
    func convert(markdown: String, templateBinaryPath: String,
                 reply: @escaping (String?, Error?) -> Void)
}
```

### 21.4 CI/CD Pipeline

```yaml
# .github/workflows/build-native.yml
name: Build Native Apps

on:
  push:
    tags: ['v*']

jobs:
  build-macos:
    runs-on: macos-15
    steps:
      - uses: actions/checkout@v4
      - name: Build macOS App
        run: |
          xcodebuild -scheme Presto -configuration Release \
            -destination "platform=macOS" \
            -archivePath build/Presto-macOS.xcarchive \
            archive

      - name: Export for App Store
        run: |
          xcodebuild -exportArchive \
            -archivePath build/Presto-macOS.xcarchive \
            -exportOptionsPlist ExportOptions-AppStore.plist \
            -exportPath build/AppStore/

      - name: Export for Direct Distribution
        run: |
          xcodebuild -exportArchive \
            -archivePath build/Presto-macOS.xcarchive \
            -exportOptionsPlist ExportOptions-Developer.plist \
            -exportPath build/Direct/
          # 创建 DMG
          create-dmg build/Direct/Presto.app build/Presto.dmg

  build-ios:
    runs-on: macos-15
    steps:
      - uses: actions/checkout@v4
      - name: Build iOS App
        run: |
          xcodebuild -scheme Presto -configuration Release \
            -destination "generic/platform=iOS" \
            -archivePath build/Presto-iOS.xcarchive \
            archive

      - name: Upload to TestFlight
        run: |
          xcrun altool --upload-app \
            -f build/Presto-iOS.xcarchive/Products/Applications/Presto.ipa \
            -u "${{ secrets.APPLE_ID }}" \
            -p "${{ secrets.APP_SPECIFIC_PASSWORD }}"
```

---

## 22. 迁移路线图

### Phase 0: 准备工作 (2 周)

- [ ] 创建 Xcode 多平台项目骨架
- [ ] 配置 SPM 依赖
- [ ] 设置 CI/CD Pipeline
- [ ] 研究 Typst Rust → Swift FFI 可行性
- [ ] 确定编辑器方案 (TextKit 2 vs WKWebView+CodeMirror)

### Phase 1: 核心引擎 (3-4 周)

- [ ] 实现 `PrestoCore` 数据模型
- [ ] 实现 `TypstCompiler` (macOS CLI 版本)
- [ ] 移植公文模板 (`GongwenConverter`) 为 Swift
- [ ] 移植教案模板 (`JiaoanShicaoConverter`) 为 Swift
- [ ] 实现 `TemplateManager` (本地模板管理)
- [ ] 实现 `ConversionPipeline`
- [ ] 单元测试覆盖

### Phase 2: macOS 应用 (4-5 周)

- [ ] 实现 Markdown 编辑器 (TextKit 2)
- [ ] 实现语法高亮
- [ ] 实现搜索/替换面板
- [ ] 实现 PDF/SVG 预览
- [ ] 实现双向滚动同步
- [ ] 实现 DocumentGroup 文档架构
- [ ] 实现原生菜单和快捷键
- [ ] 实现模板选择器 UI
- [ ] 实现设置页面
- [ ] 实现 PDF 导出
- [ ] macOS 窗口管理 (Sidebar + Split)

### Phase 3: iPadOS / iOS 适配 (3 周)

- [ ] 适配 iPadOS 分屏布局
- [ ] 适配 iOS 紧凑布局 (Tab 切换)
- [ ] 触控优化 (大点击区域、手势)
- [ ] 键盘快捷键 (iPadOS 外接键盘)
- [ ] 图片选择器适配 (PHPicker)
- [ ] Typst 库集成 (Rust FFI)

### Phase 4: 生态集成 (2-3 周)

- [ ] iCloud 文档同步
- [ ] GitHub 模板商店
- [ ] Share Extension
- [ ] Quick Look 扩展
- [ ] Spotlight 索引
- [ ] Shortcuts 集成

### Phase 5: 打磨与发布 (2-3 周)

- [ ] 无障碍审计
- [ ] 性能优化
- [ ] UI 测试
- [ ] TestFlight Beta 测试
- [ ] App Store 审核准备
- [ ] 用户文档/帮助

### 预估总周期

| 阶段 | 时长 | 人力 |
| ------ | ------ | ------ |
| Phase 0 | 2 周 | 1 人 |
| Phase 1 | 3-4 周 | 1-2 人 |
| Phase 2 | 4-5 周 | 1-2 人 |
| Phase 3 | 3 周 | 1 人 |
| Phase 4 | 2-3 周 | 1 人 |
| Phase 5 | 2-3 周 | 1-2 人 |
| **总计** | **16-20 周** | **1-2 人** |

---

## 23. 风险评估与缓解

### 23.1 技术风险

| 风险 | 可能性 | 影响 | 缓解措施 |
| ------ | -------- | ------ | --------- |
| Typst Rust → Swift FFI 复杂度高 | 高 | 高 | Phase 0 先做可行性验证；备选方案：iOS 上仅支持内置模板 |
| TextKit 2 编辑器功能不足 | 中 | 中 | 备选方案：WKWebView + CodeMirror 6 嵌入 |
| 语法高亮性能问题 | 中 | 低 | 增量高亮 + 可见区域优先 |
| App Store 审核拒绝 | 低 | 中 | 同时准备直接分发渠道 (DMG + Notarization) |
| SwiftUI 布局在某些场景下表现异常 | 中 | 低 | 关键组件使用 AppKit/UIKit 桥接 |

### 23.2 产品风险

| 风险 | 可能性 | 影响 | 缓解措施 |
| ------ | -------- | ------ | --------- |
| 用户不愿迁移到新应用 | 中 | 高 | 保持功能对等，支持导入旧配置 |
| Web 端用户流失 | 中 | 中 | 保留 Go 服务器 + Docker 部署 |
| 开发周期过长 | 中 | 高 | 分阶段发布，Phase 2 后即可 macOS Alpha |

### 23.3 关键决策点

1. **Phase 0 末**: Typst FFI 可行性评估 → 决定 iOS 上的 Typst 集成方案
2. **Phase 1 末**: 核心引擎验证 → 确认转换精度与性能满足要求
3. **Phase 2 末**: macOS App Alpha → 决定是否继续 iOS 开发或先打磨 macOS

---

## 附录

### A. 功能映射表

| 现有功能 | 现有实现 | 原生实现方案 |
| --------- | --------- | ------------ |
| Markdown 编辑 | CodeMirror 6 | TextKit 2 + NSTextView |
| 语法高亮 | CodeMirror markdown() | 自定义 NSTextContentStorageDelegate |
| 搜索替换 | CodeMirror search() | NSTextView usesFindBar / UIFindInteraction |
| SVG 预览 | HTML `{@html svg}` | NSImage / WKWebView / Core Graphics |
| PDF 导出 | Fetch → Blob → download | PDFKit → NSSavePanel / UIDocumentPickerViewController |
| 文件打开 | Wails OpenFileDialog | NSOpenPanel / UIDocumentPickerViewController / DocumentGroup |
| 模板选择 | Svelte Select component | SwiftUI Picker / Menu |
| 模板商店 | Svelte TemplateStore page | SwiftUI NavigationStack + List |
| 设置页面 | Svelte Settings route | SwiftUI Settings scene / Form |
| 滚动同步 | JS scrollTop ratio sync | ScrollViewReader + coordinate spaces |
| 菜单栏 | Wails menu.NewMenu() | SwiftUI Commands |
| 快捷键 | Wails keys.CmdOrCtrl() | SwiftUI .keyboardShortcut() |
| CORS 中间件 | Go corsMiddleware | 原生不需要（无 HTTP） |
| 日志中间件 | Go loggingMiddleware | os.Logger |
| 状态管理 | Svelte $state runes | @Observable / @State / @Environment |

### B. Go → Swift 类型映射

| Go 类型 | Swift 类型 |
| --------- | ----------- |
| `string` | `String` |
| `[]byte` | `Data` |
| `error` | `Error` (protocol) / `throws` |
| `context.Context` | `Task` / `withTaskCancellationHandler` |
| `map[string]any` | `[String: Any]` / `Codable struct` |
| `*os.File` | `FileHandle` / `URL` |
| `exec.Command` | `Process` (macOS) |
| `http.Handler` | 不需要（原生直接调用） |
| `embed.FS` | `Bundle.main` / `NSDataAsset` |
| `sync.Mutex` | `actor` / `@Sendable` |
| `chan` | `AsyncStream` / `AsyncChannel` |
| `json.Marshal/Unmarshal` | `JSONEncoder/JSONDecoder` |

### C. 参考资源

#### Apple 官方文档

- [Apple Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/)
- [SwiftUI Documentation](https://developer.apple.com/documentation/swiftui)
- [SwiftData Documentation](https://developer.apple.com/documentation/swiftdata)
- [TextKit 2 Overview](https://developer.apple.com/documentation/appkit/textkit)
- [PDFKit Documentation](https://developer.apple.com/documentation/pdfkit)
- [Document-Based App Programming Guide](https://developer.apple.com/documentation/swiftui/documents)
- [App Sandbox Design Guide](https://developer.apple.com/documentation/security/app-sandbox)
- [Embedding a helper tool in a sandboxed app](https://developer.apple.com/documentation/xcode/embedding-a-helper-tool-in-a-sandboxed-app)
- [ExtensionKit Documentation](https://developer.apple.com/documentation/extensionkit)
- [Notarizing macOS software](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)

#### 开源项目参考

- [MarkEdit](https://github.com/MarkEdit-app/MarkEdit) — macOS Markdown 编辑器，CodeMirror 6 + WKWebView 方案
- [CodeEditorView](https://github.com/mchakravarty/CodeEditorView) — SwiftUI 原生代码编辑器
- [STTextView](https://github.com/krzyzanowskim/STTextView) — TextKit 2 文本视图
- [SVGView](https://github.com/exyte/SVGView) — 纯 SwiftUI SVG 渲染
- [SwiftyXPC](https://github.com/CharlesJS/SwiftyXPC) — Swift 友好的 XPC 封装

#### 依赖库

- [swift-markdown](https://github.com/apple/swift-markdown) — Apple 官方 Markdown 解析器
- [Yams](https://github.com/jpsim/Yams) — Swift YAML 解析器
- [Typst](https://typst.app/docs/) — 排版引擎文档
- [Typst Open Source](https://typst.app/open-source/) — Typst 源码 (Apache 2.0)

#### 技术文章

- [Calling Rust code from Swift](https://www.strathweb.com/2023/07/calling-rust-code-from-swift/)
- [Integrating Rust and SwiftUI (Ghostty)](https://dfrojas.com/software/integrating-Rust-and-SwiftUI.html)
- [Creating custom extension points with ExtensionKit](https://rambo.codes/posts/2022-06-27-creating-custom-extension-points-with-extensionkit)
- [SwiftData vs Core Data in 2025](https://www.hashstudioz.com/blog/swiftdata-vs-core-data-which-should-you-choose-in-2025/)
- [Food Truck: Building a SwiftUI Multiplatform App](https://developer.apple.com/documentation/swiftui/food-truck-building-a-swiftui-multiplatform-app)

### D. 术语表

| 术语 | 说明 |
| ------ | ------ |
| Typst | 现代排版引擎，类似 LaTeX 但更简单 |
| Goldmark | Go 语言 Markdown 解析库 |
| Wails | Go 语言桌面应用框架（类似 Electron，使用 WebView） |
| SwiftUI | Apple 声明式 UI 框架 |
| SwiftData | Apple 数据持久化框架，基于 Core Data |
| TextKit 2 | Apple 文本排版引擎（NSTextView/UITextView 底层） |
| PDFKit | Apple PDF 渲染和操作框架 |
| XPC Service | macOS 进程间通信机制 |
| DocumentGroup | SwiftUI 文档型应用场景 |
| SF Symbols | Apple 系统图标库 |
| UniFFI | Mozilla 的跨语言 FFI 工具 |
