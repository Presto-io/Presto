# Presto 鸿蒙全平台原生应用重构方案

> Markdown → Typst → PDF 文档转换平台 — HarmonyOS NEXT 全设备适配

---

## 目录

1. [项目现状分析](#1-项目现状分析)
2. [重构目标与范围](#2-重构目标与范围)
3. [技术选型](#3-技术选型)
4. [系统架构设计](#4-系统架构设计)
5. [项目工程结构](#5-项目工程结构)
6. [核心模块详细设计](#6-核心模块详细设计)
7. [多设备适配策略](#7-多设备适配策略)
8. [数据模型与持久化](#8-数据模型与持久化)
9. [模板系统重构](#9-模板系统重构)
10. [Typst 引擎集成](#10-typst-引擎集成)
11. [编辑器实现方案](#11-编辑器实现方案)
12. [预览系统](#12-预览系统)
13. [文件管理与文档架构](#13-文件管理与文档架构)
14. [分布式能力](#14-分布式能力)
15. [网络层设计](#15-网络层设计)
16. [UI/UX 设计规范](#16-uiux-设计规范)
17. [快捷键与菜单系统](#17-快捷键与菜单系统)
18. [无障碍访问](#18-无障碍访问)
19. [性能优化策略](#19-性能优化策略)
20. [测试策略](#20-测试策略)
21. [分发与部署](#21-分发与部署)
22. [迁移路线图](#22-迁移路线图)
23. [风险评估与应对](#23-风险评估与应对)
24. [附录](#附录)

---

## 1. 项目现状分析

### 1.1 技术栈概览

| 层级 | 当前技术 | 版本 |
| --- | --- | --- |
| 后端语言 | Go | 1.25 |
| 前端框架 | SvelteKit 2 + Svelte 5 (runes) | 2.50+ / 5.49+ |
| 编辑器 | CodeMirror 6 | — |
| 桌面框架 | Wails v2 | 2.11 |
| 排版引擎 | Typst CLI | 0.14 |
| Markdown 解析 | Goldmark（模板内部） | 1.7.16 |
| 图标库 | Lucide Svelte | — |
| 构建工具 | Vite 7.3 + Go Makefile | — |
| 容器化 | Docker 多阶段构建 | — |

### 1.2 当前架构概览

```text
┌─────────────────────────────────────────────────┐
│             前端 (SvelteKit 2 + Svelte 5)        │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Editor   │  │ Preview  │  │ TemplateStore │  │
│  │(CodeMirror)│ │ (SVG)   │  │  (GitHub API) │  │
│  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │
│       └──────────────┴────────────────┘          │
│                      │ fetch / Wails binding     │
├──────────────────────┼───────────────────────────┤
│                Go HTTP API / Wails                │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Template │  │  Typst   │  │   Template    │  │
│  │ Executor │  │ Compiler │  │   Manager     │  │
│  └──────────┘  └──────────┘  └───────────────┘  │
└─────────────────────────────────────────────────┘
```

### 1.3 核心功能清单

| 功能 | 描述 | 状态 |
| --- | --- | --- |
| Markdown 编辑 | CodeMirror 6，语法高亮，中文搜索，自动换行 | ✅ 已实现 |
| 实时预览 | SVG 多页渲染，500ms 防抖 | ✅ 已实现 |
| 双向滚动同步 | 编辑器 ↔ 预览面板滚动联动 | ✅ 已实现 |
| 模板系统 | 插件化可执行文件架构，stdin/stdout 协议 | ✅ 已实现 |
| PDF 导出 | Typst CLI 编译为 PDF | ✅ 已实现 |
| 文件打开 | 原生文件对话框（桌面端）/ 浏览器文件输入 | ✅ 已实现 |
| 模板商店 | GitHub Search API 发现并安装社区模板 | ✅ 已实现 |
| 模板切换确认 | 切换模板时选择保留内容或加载示例 | ✅ 已实现 |
| 设置页面 | 社区模板开关、关于信息、开源协议声明 | ✅ 已实现 |
| 桌面端菜单 | macOS 原生菜单栏（文件/编辑/窗口） | ✅ 已实现 |
| 快捷键 | ⌘O 打开、⌘E 导出、⌘, 设置、⌘F 搜索 | ✅ 已实现 |
| 批量转换 | 一次处理多个文件 | 🔲 API 已注册 |

### 1.4 现有 API 端点

| 方法 | 路径 | 功能 |
| --- | --- | --- |
| GET | `/api/health` | 健康检查 |
| POST | `/api/convert` | Markdown → Typst |
| POST | `/api/compile` | Typst → PDF |
| POST | `/api/compile-svg` | Typst → SVG（多页） |
| POST | `/api/convert-and-compile` | Markdown → PDF（一步完成） |
| POST | `/api/batch` | 批量转换（待实现） |
| GET | `/api/templates` | 列出已安装模板 |
| GET | `/api/templates/discover` | GitHub 搜索可用模板 |
| POST | `/api/templates/{id}/install` | 安装模板 |
| DELETE | `/api/templates/{id}` | 卸载模板 |
| GET | `/api/templates/{id}/manifest` | 获取模板元数据 |
| GET | `/api/templates/{id}/example` | 获取模板示例内容 |

### 1.5 现有数据模型

```typescript
// TypeScript 类型定义（当前前端）
interface Template {
  name: string;        // 模板标识符
  displayName: string; // 显示名称
  description: string; // 描述
  version: string;     // 版本号
  author: string;      // 作者
}

interface Manifest extends Template {
  license: string;
  minPrestoVersion: string;
  frontmatterSchema?: Record<string, FieldSchema>;
}

interface FieldSchema {
  type: string;        // "string" | "boolean"
  default?: unknown;
  format?: string;     // 如 "YYYY-MM-DD"
}
```

### 1.6 内置模板详情

| 模板 | ID | 描述 | 前置字段 |
| --- | --- | --- | --- |
| 类公文模板 | `gongwen` | 符合 GB/T 9704-2012 标准 | title, author, date, signature |
| 实操教案模板 | `jiaoan-shicao` | 教学计划表格格式 | （使用 Markdown 结构化标题） |

### 1.7 模板系统工作原理

```text
输入: Markdown (stdin) → 模板可执行文件 → 输出: Typst (stdout)
参数: --manifest → 输出 manifest.json
参数: --example  → 输出示例 Markdown
```

模板是独立的可执行文件（可用任何语言编写），通过 stdin/stdout 协议与 Presto 主程序通信。模板二进制文件存放在 `~/.presto/templates/{name}/` 目录下。

---

## 2. 重构目标与范围

### 2.1 核心目标

1. **全平台原生体验**：支持 HarmonyOS 手机、平板、PC（2-in-1 笔记本）三种设备形态
2. **一次开发多端部署**：利用 ArkUI 的自适应布局和断点系统，单一代码库适配多端
3. **分布式协同**：利用鸿蒙分布式能力，实现跨设备文档流转和协同编辑
4. **原生性能**：Typst 引擎通过 N-API/ohos-rs 原生集成，避免 CLI 进程调用开销
5. **鸿蒙生态融合**：接入华为云服务（AGC）、应用市场分发、HarmonyOS 设计规范

### 2.2 目标平台

| 平台 | 设备类型 | 最低 API 版本 | 布局模式 |
| --- | --- | --- | --- |
| HarmonyOS NEXT | 手机 | API 12 (5.0) | 单栏紧凑型 |
| HarmonyOS NEXT | 平板 | API 12 (5.0) | 双栏分屏型 |
| HarmonyOS NEXT | PC / 2-in-1 | API 12 (5.0) | 三栏桌面型 |

### 2.3 非目标（明确排除）

- 不支持 HarmonyOS 4.x 及更早版本（仅支持 NEXT 纯鸿蒙内核）
- 不保留 Web 端部署能力（鸿蒙版为独立原生应用）
- 不兼容 Android 应用包（不使用兼容层）
- 不实现 iOS/macOS 支持（另有独立方案）

---

## 3. 技术选型

### 3.1 开发语言与框架

| 选项 | 技术 | 说明 |
| --- | --- | --- |
| 开发语言 | **ArkTS** | HarmonyOS NEXT 唯一官方应用开发语言，TypeScript 超集 |
| UI 框架 | **ArkUI (声明式)** | 鸿蒙原生声明式 UI 框架，支持多设备自适应 |
| 应用模型 | **Stage 模型** | HarmonyOS NEXT 推荐的应用模型，取代 FA 模型 |
| 构建系统 | **hvigor** | 鸿蒙官方构建工具 |
| 包管理器 | **ohpm** | OpenHarmony Package Manager |
| IDE | **DevEco Studio** | 基于 IntelliJ 的鸿蒙专用 IDE |

### 3.2 ArkTS 语言特性

ArkTS 是 TypeScript 的超集，在此基础上增加了以下关键能力：

```typescript
// 组件装饰器
@Entry          // 入口组件
@Component      // 自定义组件（V1）
@ComponentV2    // 自定义组件（V2，支持深度观测）

// 状态管理装饰器（V1）
@State          // 组件内部状态
@Prop           // 父→子单向数据传递
@Link           // 父→子双向绑定
@Provide/@Consume  // 跨组件层级传递

// 状态管理装饰器（V2）
@Local          // 组件内部状态（替代 @State）
@Param          // 父→子单向传递（替代 @Prop）
@Provider/@Consumer  // 真正双向通信
@Event          // 子→父事件传递
@Monitor        // 深度变化监听（替代 @Watch）
@ObservedV2/@Trace  // 多层深度观测

// UI 构建装饰器
@Builder        // 可复用 UI 构建函数
@Styles         // 通用样式复用
@Extend         // 组件专属样式扩展（支持参数）
@Reusable       // 组件复用优化
```

### 3.3 V1 与 V2 装饰器对比

| 能力 | V1 | V2 |
| --- | --- | --- |
| 深度观测 | `@Observed` + `@ObjectLink`（单层） | `@ObservedV2` + `@Trace`（多层） |
| 变化监听 | `@Watch` | `@Monitor`（深度监听） |
| 组件装饰器 | `@Component` | `@ComponentV2` |
| 内部状态 | `@State` | `@Local` |
| 父→子 | `@Prop` | `@Param` |
| 双向绑定 | `@Link` | `@Provider/@Consumer` |
| 子→父事件 | 回调函数 | `@Event` |

> **决策**：本项目采用 **V2 装饰器体系**（`@ComponentV2`），以获得更好的深度观测和状态管理能力。V2 在 API 12+ 中可用。

### 3.4 核心依赖

| 依赖 | 用途 | 集成方式 |
| --- | --- | --- |
| Typst | 排版引擎 | Rust → .so（ohos-rs / N-API） |
| CodeMirror 6 | Markdown 编辑器 | Web 组件嵌入 |
| FluidMarkdown | Markdown 原生渲染 | ohpm（蚂蚁集团开源，Apache 2.0） |
| Goldmark 等效 | Markdown 解析（模板内部） | Rust 桥接（pulldown-cmark / comrak） |
| `@ohos.web.webview` | WebView 容器 | 系统 API |
| `@ohos.data.relationalStore` | 关系型数据库 | 系统 API |
| `@ohos.data.preferences` | 轻量偏好存储 | 系统 API |
| `@ohos.net.http` | HTTP 网络请求 | 系统 API |
| `@ohos.file.picker` | 文件选择器 | 系统 API |

---

## 4. 系统架构设计

### 4.1 整体架构

采用 **分层架构 + MVVM** 模式：

```text
┌─────────────────────────────────────────────────────────┐
│                    Presentation Layer                     │
│   ┌─────────────┐  ┌──────────┐  ┌──────────────────┐  │
│   │  EditorPage │  │ Settings │  │  TemplateStore   │  │
│   │  (ArkUI)    │  │  (ArkUI) │  │     (ArkUI)      │  │
│   └──────┬──────┘  └────┬─────┘  └────────┬─────────┘  │
│          └───────────────┴─────────────────┘            │
│                          │ ViewModel                     │
├──────────────────────────┼───────────────────────────────┤
│                    Business Layer                         │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│   │ EditorVM     │  │ TemplateVM   │  │ SettingsVM   │  │
│   │ @ObservedV2  │  │ @ObservedV2  │  │ @ObservedV2  │  │
│   └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
│          └──────────────────┼─────────────────┘          │
│                             │ Service                     │
├─────────────────────────────┼────────────────────────────┤
│                    Service Layer                          │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│   │ ConvertSvc   │  │ TemplateSvc  │  │  CompileSvc  │  │
│   │ (MD→Typst)   │  │ (管理/安装)   │  │ (Typst→PDF)  │  │
│   └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
│          └──────────────────┼─────────────────┘          │
│                             │ Engine                      │
├─────────────────────────────┼────────────────────────────┤
│                    Engine Layer (Native)                   │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│   │ TypstEngine  │  │ MarkdownParser│  │ FileManager  │  │
│   │ (Rust/.so)   │  │ (ArkTS/Rust)  │  │ (ohos.file)  │  │
│   └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 4.2 模块依赖关系

```text
PrestoApp (Entry HAP)
├── feature_editor (HSP)      // 编辑器功能模块
│   ├── TypstEngine (HAR)     // Typst 原生引擎封装
│   └── MarkdownParser (HAR)  // Markdown 解析
├── feature_template (HSP)    // 模板管理功能模块
├── feature_settings (HSP)    // 设置页面
├── common_ui (HAR)           // 通用 UI 组件库
├── common_model (HAR)        // 数据模型定义
└── common_utils (HAR)        // 工具函数
```

### 4.3 HAP / HSP / HAR 模块说明

| 模块类型 | 全称 | 说明 |
| --- | --- | --- |
| **HAP** | HarmonyOS Ability Package | 应用入口包，包含 UIAbility |
| **HSP** | HarmonyOS Shared Package | 动态共享包，可按需加载，支持独立页面和资源 |
| **HAR** | HarmonyOS Archive | 静态共享包，编译时链接，适合工具库和模型定义 |

### 4.4 数据流

```text
用户输入 Markdown
    ↓
EditorPage (ArkUI / Web 组件)
    ↓ onchange (500ms 防抖)
EditorViewModel
    ↓ convert(markdown, templateId)
ConvertService
    ↓ 调用模板转换逻辑
TypstEngine (Native .so)
    ↓ compile(typstSource) → SVG / PDF
    ↓
EditorViewModel.svgPages 更新
    ↓ @Trace 驱动 UI 更新
PreviewComponent 渲染 SVG
```

---

## 5. 项目工程结构

### 5.1 DevEco Studio 工程组织

```text
Presto/
├── AppScope/                          # 应用全局配置
│   ├── app.json5                      # 应用配置（bundleName, versionCode 等）
│   └── resources/                     # 全局资源
│       ├── base/
│       │   ├── element/
│       │   │   └── string.json        # 全局字符串
│       │   ├── media/
│       │   │   └── app_icon.png       # 应用图标
│       │   └── profile/
│       │       └── main_pages.json    # 页面路由表
│       ├── zh_CN/                     # 中文资源
│       └── en_US/                     # 英文资源
│
├── entry/                             # 主 HAP 模块
│   ├── src/main/
│   │   ├── ets/
│   │   │   ├── entryability/
│   │   │   │   └── EntryAbility.ets   # UIAbility 入口
│   │   │   ├── pages/
│   │   │   │   ├── Index.ets          # 主页面（导航容器）
│   │   │   │   ├── EditorPage.ets     # 编辑器页面
│   │   │   │   ├── SettingsPage.ets   # 设置页面
│   │   │   │   └── TemplateStorePage.ets # 模板商店
│   │   │   ├── viewmodel/
│   │   │   │   ├── EditorViewModel.ets
│   │   │   │   ├── TemplateViewModel.ets
│   │   │   │   └── SettingsViewModel.ets
│   │   │   └── components/
│   │   │       ├── EditorComponent.ets
│   │   │       ├── PreviewComponent.ets
│   │   │       ├── ToolbarComponent.ets
│   │   │       └── TemplateSelectorComponent.ets
│   │   ├── resources/
│   │   │   ├── base/
│   │   │   ├── phone/                 # 手机专用资源
│   │   │   ├── tablet/                # 平板专用资源
│   │   │   └── rawfile/               # 原始文件
│   │   │       └── editor/            # CodeMirror HTML/JS/CSS
│   │   │           ├── index.html
│   │   │           ├── editor.js
│   │   │           └── editor.css
│   │   └── module.json5               # 模块配置
│   ├── oh-package.json5               # 模块依赖
│   └── build-profile.json5            # 构建配置
│
├── feature_editor/                    # 编辑器 HSP 模块
│   └── src/main/ets/
│       ├── service/
│       │   ├── ConvertService.ets     # Markdown → Typst 转换服务
│       │   └── CompileService.ets     # Typst → PDF/SVG 编译服务
│       └── bridge/
│           └── EditorBridge.ets       # Web ↔ ArkTS 通信桥
│
├── feature_template/                  # 模板管理 HSP 模块
│   └── src/main/ets/
│       ├── service/
│       │   ├── TemplateManager.ets    # 模板管理服务
│       │   ├── TemplateInstaller.ets  # 模板安装逻辑
│       │   └── GitHubService.ets      # GitHub API 客户端
│       └── model/
│           └── TemplateModel.ets      # 模板数据模型
│
├── common_ui/                         # 通用 UI HAR 模块
│   └── src/main/ets/
│       ├── theme/
│       │   └── PrestoTheme.ets        # 设计令牌（颜色、字体、间距）
│       └── components/
│           ├── PrestoButton.ets
│           ├── PrestoCard.ets
│           └── PrestoDialog.ets
│
├── common_model/                      # 数据模型 HAR
│   └── src/main/ets/
│       ├── Template.ets
│       ├── Manifest.ets
│       ├── EditorState.ets
│       └── Settings.ets
│
├── native_typst/                      # Typst 原生模块
│   ├── src/
│   │   ├── lib.rs                     # Rust 入口
│   │   └── typst_bridge.rs            # Typst API 封装
│   ├── Cargo.toml
│   └── build.rs                       # ohos-rs 构建脚本
│
├── oh-package.json5                   # 根依赖配置
├── build-profile.json5                # 全局构建配置
└── hvigorfile.ts                      # hvigor 构建脚本
```

### 5.2 构建配置示例

**app.json5**:

```json
{
  "app": {
    "bundleName": "com.mrered.presto",
    "vendor": "mrered",
    "versionCode": 1000000,
    "versionName": "1.0.0",
    "icon": "$media:app_icon",
    "label": "$string:app_name",
    "minAPIVersion": 12,
    "targetAPIVersion": 14
  }
}
```

**module.json5** (entry):

```json
{
  "module": {
    "name": "entry",
    "type": "entry",
    "description": "$string:module_desc",
    "mainElement": "EntryAbility",
    "deviceTypes": ["phone", "tablet", "2in1"],
    "deliveryWithInstall": true,
    "installationFree": false,
    "pages": "$profile:main_pages",
    "abilities": [
      {
        "name": "EntryAbility",
        "srcEntry": "./ets/entryability/EntryAbility.ets",
        "description": "$string:ability_desc",
        "icon": "$media:app_icon",
        "label": "$string:app_name",
        "startWindowIcon": "$media:app_icon",
        "startWindowBackground": "$color:start_window_background",
        "exported": true,
        "skills": [
          {
            "entities": ["entity.system.home"],
            "actions": ["action.system.home"]
          }
        ]
      }
    ],
    "requestPermissions": [
      {
        "name": "ohos.permission.INTERNET",
        "reason": "$string:permission_internet_reason",
        "usedScene": {
          "abilities": ["EntryAbility"],
          "when": "always"
        }
      }
    ]
  }
}
```

---

## 6. 核心模块详细设计

### 6.1 应用入口 — EntryAbility

```typescript
// entry/src/main/ets/entryability/EntryAbility.ets
import { UIAbility, AbilityConstant, Want } from '@kit.AbilityKit';
import { window } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';

const TAG = 'EntryAbility';
const DOMAIN = 0xFF00;

export default class EntryAbility extends UIAbility {
  onCreate(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    hilog.info(DOMAIN, TAG, 'onCreate');
    // 初始化全局服务：模板管理器、Typst 引擎等
    globalThis.templateManager = new TemplateManager(this.context);
    globalThis.typstEngine = new TypstEngine();
  }

  onDestroy(): void {
    hilog.info(DOMAIN, TAG, 'onDestroy');
  }

  onWindowStageCreate(windowStage: window.WindowStage): void {
    hilog.info(DOMAIN, TAG, 'onWindowStageCreate');

    // 设置沉浸式状态栏
    windowStage.getMainWindow().then((win) => {
      win.setWindowLayoutFullScreen(true);
      win.setWindowSystemBarProperties({
        statusBarColor: '#00000000',
        navigationBarColor: '#00000000',
        statusBarContentColor: '#FFFFFF',
      });
    });

    // 加载主页面
    windowStage.loadContent('pages/Index', (err) => {
      if (err.code) {
        hilog.error(DOMAIN, TAG, 'Failed to load content: %{public}s', JSON.stringify(err));
        return;
      }
      hilog.info(DOMAIN, TAG, 'Content loaded successfully');
    });
  }
}
```

### 6.2 主页面 — 导航容器

```typescript
// entry/src/main/ets/pages/Index.ets
import { BreakpointSystem, BreakpointState } from '../utils/BreakpointSystem';

@Entry
@ComponentV2
struct Index {
  @Local currentBreakpoint: string = 'sm';
  @Local navPathStack: NavPathStack = new NavPathStack();
  private breakpointSystem: BreakpointSystem = new BreakpointSystem();

  aboutToAppear(): void {
    this.breakpointSystem.register();
    this.breakpointSystem.onChange((bp: string) => {
      this.currentBreakpoint = bp;
    });
  }

  aboutToDisappear(): void {
    this.breakpointSystem.unregister();
  }

  build() {
    Navigation(this.navPathStack) {
      // 主内容区域
      EditorPage()
    }
    .title($r('app.string.app_name'))
    .mode(this.currentBreakpoint === 'sm'
      ? NavigationMode.Stack
      : NavigationMode.Split)
    .navBarWidth(this.currentBreakpoint === 'lg' ? '30%' : '40%')
    .navBarWidthRange([240, 400])
    .minContentWidth(360)
    .hideToolBar(this.currentBreakpoint === 'sm')
  }
}
```

### 6.3 编辑器 ViewModel

```typescript
// entry/src/main/ets/viewmodel/EditorViewModel.ets
import { ConvertService } from '../../feature_editor/service/ConvertService';
import { CompileService } from '../../feature_editor/service/CompileService';

@ObservedV2
export class EditorViewModel {
  @Trace markdown: string = '';
  @Trace typstSource: string = '';
  @Trace svgPages: string[] = [];
  @Trace selectedTemplate: string = '';
  @Trace documentDir: string = '';
  @Trace isConverting: boolean = false;
  @Trace errorMessage: string = '';

  private convertService: ConvertService = new ConvertService();
  private compileService: CompileService = new CompileService();
  private debounceTimer: number = -1;

  async onMarkdownChange(newValue: string): Promise<void> {
    this.markdown = newValue;
    if (!this.selectedTemplate || !newValue.trim()) return;

    // 500ms 防抖
    clearTimeout(this.debounceTimer);
    this.debounceTimer = setTimeout(async () => {
      await this.convert();
    }, 500);
  }

  async convert(): Promise<void> {
    if (!this.selectedTemplate || !this.markdown.trim()) return;
    this.isConverting = true;
    this.errorMessage = '';

    try {
      // 1. Markdown → Typst（通过模板转换）
      this.typstSource = await this.convertService.convert(
        this.markdown, this.selectedTemplate
      );

      // 2. Typst → SVG（通过原生 Typst 引擎）
      this.svgPages = await this.compileService.compileSvg(
        this.typstSource, this.documentDir
      );
    } catch (e) {
      this.errorMessage = e instanceof Error ? e.message : String(e);
    } finally {
      this.isConverting = false;
    }
  }

  async exportPdf(): Promise<Uint8Array> {
    return this.compileService.compilePdf(
      this.typstSource, this.documentDir
    );
  }

  async selectTemplate(templateId: string): Promise<void> {
    this.selectedTemplate = templateId;
    if (this.markdown.trim()) {
      await this.convert();
    }
  }
}
```

### 6.4 转换服务

```typescript
// feature_editor/src/main/ets/service/ConvertService.ets
import { TemplateManager } from '../../feature_template/service/TemplateManager';

export class ConvertService {
  private templateManager: TemplateManager;

  constructor() {
    this.templateManager = globalThis.templateManager;
  }

  /**
   * 将 Markdown 通过指定模板转换为 Typst 源码。
   * 在鸿蒙端，模板逻辑需要重新实现：
   * - 方案 A：将模板逻辑用 ArkTS 重写
   * - 方案 B：通过 childProcessManager 启动原生子进程
   * - 方案 C：将模板编译为 .so 通过 N-API 调用
   */
  async convert(markdown: string, templateId: string): Promise<string> {
    const template = this.templateManager.get(templateId);
    if (!template) {
      throw new Error(`Template not found: ${templateId}`);
    }
    return template.executor.convert(markdown);
  }
}
```

---

## 7. 多设备适配策略

### 7.1 断点系统

鸿蒙使用 vp（虚拟像素）作为断点判断依据：

| 断点 | 范围 | 设备类型 | 典型宽度 |
| --- | --- | --- | --- |
| `sm` | < 600vp | 手机 | 360-414vp |
| `md` | 600vp - 840vp | 折叠屏/小平板 | 600-768vp |
| `lg` | ≥ 840vp | 平板 / PC | 840-1440vp |

### 7.2 断点系统工具类

```typescript
// entry/src/main/ets/utils/BreakpointSystem.ets
import { mediaquery } from '@kit.ArkUI';

type BreakpointCallback = (breakpoint: string) => void;

export class BreakpointSystem {
  private smListener?: mediaquery.MediaQueryListener;
  private mdListener?: mediaquery.MediaQueryListener;
  private lgListener?: mediaquery.MediaQueryListener;
  private callback?: BreakpointCallback;

  register(): void {
    this.smListener = mediaquery.matchMediaSync('(width < 600vp)');
    this.mdListener = mediaquery.matchMediaSync('(600vp <= width < 840vp)');
    this.lgListener = mediaquery.matchMediaSync('(width >= 840vp)');

    this.smListener.on('change', (result) => {
      if (result.matches) this.callback?.('sm');
    });
    this.mdListener.on('change', (result) => {
      if (result.matches) this.callback?.('md');
    });
    this.lgListener.on('change', (result) => {
      if (result.matches) this.callback?.('lg');
    });
  }

  onChange(callback: BreakpointCallback): void {
    this.callback = callback;
  }

  unregister(): void {
    this.smListener?.off('change');
    this.mdListener?.off('change');
    this.lgListener?.off('change');
  }
}
```

### 7.3 各平台布局方案

#### 手机（sm）— 标签页切换

```text
┌─────────────────────┐
│  Presto    [▼模板]   │ ← 顶部工具栏
├─────────────────────┤
│                     │
│   编辑器 / 预览      │ ← 全屏单面板
│   (滑动或标签切换)    │
│                     │
│                     │
├─────────────────────┤
│  [编辑] [预览] [设置] │ ← 底部标签栏
└─────────────────────┘
```

#### 平板（md）— 左右分栏

```text
┌──────────────────────────────────────────┐
│  Presto     [▼ 公文模板]      [导出 PDF]  │ ← 工具栏
├────────────────────┬─────────────────────┤
│                    │                     │
│    Markdown        │    实时预览          │
│    编辑器          │    (SVG 渲染)        │
│                    │                     │
│                    │                     │
│                    │                     │
│                    │                     │
└────────────────────┴─────────────────────┘
```

#### PC / 2-in-1（lg）— 三栏布局

```text
┌──────────────────────────────────────────────────────┐
│  Presto        [▼ 模板]  [打开] [导出]        [设置]  │ ← 工具栏
├──────────┬─────────────────────┬─────────────────────┤
│ 文件列表  │                     │                     │
│          │    Markdown         │    实时预览           │
│ doc1.md  │    编辑器            │    (SVG 渲染)         │
│ doc2.md  │                     │                     │
│ doc3.md  │                     │                     │
│          │                     │                     │
│ [新建]    │                     │                     │
└──────────┴─────────────────────┴─────────────────────┘
```

### 7.4 Navigation 组件适配

```typescript
@Entry
@ComponentV2
struct EditorPage {
  @Local currentBreakpoint: string = 'sm';
  @Local showPreview: boolean = true;

  build() {
    Column() {
      // 工具栏（始终显示）
      ToolbarComponent({
        breakpoint: this.currentBreakpoint,
      })

      if (this.currentBreakpoint === 'sm') {
        // 手机：Tabs 切换编辑器和预览
        Tabs({ barPosition: BarPosition.End }) {
          TabContent() {
            EditorComponent()
          }.tabBar('编辑')

          TabContent() {
            PreviewComponent()
          }.tabBar('预览')
        }
        .barMode(BarMode.Fixed)
        .height('100%')

      } else {
        // 平板/PC：并排布局
        Row() {
          // PC 模式下显示文件列表
          if (this.currentBreakpoint === 'lg') {
            FileListPanel()
              .width(200)
          }

          EditorComponent()
            .layoutWeight(1)

          PreviewComponent()
            .layoutWeight(1)
        }
        .height('100%')
      }
    }
    .width('100%')
    .height('100%')
  }
}
```

### 7.5 GridRow/GridCol 响应式栅格

```typescript
// 使用栅格系统实现更精细的响应式布局
GridRow({
  columns: { sm: 4, md: 8, lg: 12 },
  gutter: { x: 8, y: 8 }
}) {
  GridCol({ span: { sm: 4, md: 4, lg: 6 } }) {
    EditorComponent()
  }
  GridCol({ span: { sm: 4, md: 4, lg: 6 } }) {
    PreviewComponent()
  }
}
```

---

## 8. 数据模型与持久化

### 8.1 数据模型定义

```typescript
// common_model/src/main/ets/Template.ets
@ObservedV2
export class TemplateModel {
  @Trace id: string = '';
  @Trace name: string = '';
  @Trace displayName: string = '';
  @Trace description: string = '';
  @Trace version: string = '';
  @Trace author: string = '';
  @Trace license: string = '';
  @Trace minPrestoVersion: string = '';
  @Trace installedPath: string = '';
  @Trace isBuiltIn: boolean = false;
}

@ObservedV2
export class ManifestModel extends TemplateModel {
  @Trace frontmatterSchema: Map<string, FieldSchema> = new Map();
}

export interface FieldSchema {
  type: string;
  default?: string | boolean | number;
  format?: string;
}
```

```typescript
// common_model/src/main/ets/EditorState.ets
@ObservedV2
export class EditorState {
  @Trace markdown: string = '';
  @Trace typstSource: string = '';
  @Trace svgPages: string[] = [];
  @Trace selectedTemplate: string = '';
  @Trace documentDir: string = '';
  @Trace documentPath: string = '';
  @Trace isModified: boolean = false;
}
```

### 8.2 持久化策略

| 数据类型 | 存储方式 | API |
| --- | --- | --- |
| 用户偏好设置 | Preferences | `@ohos.data.preferences` |
| 已安装模板元数据 | 关系型数据库 (RDB) | `@ohos.data.relationalStore` |
| 模板二进制文件 | 应用沙箱文件系统 | `@ohos.file.fs` |
| 编辑器临时状态 | AppStorage | `AppStorage` / `PersistentStorage` |
| 文档文件 | 用户文件系统 | `@ohos.file.picker` |
| 跨设备同步数据 | 分布式 KV 存储 | `@ohos.data.distributedKVStore` |

### 8.3 Preferences 使用示例

```typescript
// 设置偏好存储
import { preferences } from '@ohos.data.preferences';

const PREFS_NAME = 'presto_settings';

export class SettingsStore {
  private prefs?: preferences.Preferences;

  async init(context: Context): Promise<void> {
    this.prefs = await preferences.getPreferences(context, PREFS_NAME);
  }

  async getCommunityEnabled(): Promise<boolean> {
    return (await this.prefs?.get('communityTemplates', false)) as boolean;
  }

  async setCommunityEnabled(enabled: boolean): Promise<void> {
    await this.prefs?.put('communityTemplates', enabled);
    await this.prefs?.flush();
  }

  async getLastTemplate(): Promise<string> {
    return (await this.prefs?.get('lastTemplate', '')) as string;
  }

  async setLastTemplate(templateId: string): Promise<void> {
    await this.prefs?.put('lastTemplate', templateId);
    await this.prefs?.flush();
  }
}
```

### 8.4 关系型数据库 (RDB) 使用

```typescript
// 模板元数据存储
import { relationalStore } from '@ohos.data.relationalStore';

const DB_CONFIG: relationalStore.StoreConfig = {
  name: 'presto.db',
  securityLevel: relationalStore.SecurityLevel.S1,
};

const SQL_CREATE_TEMPLATES = `
  CREATE TABLE IF NOT EXISTS templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT,
    description TEXT,
    version TEXT,
    author TEXT,
    license TEXT,
    installed_path TEXT,
    is_built_in INTEGER DEFAULT 0,
    installed_at INTEGER
  )`;

export class TemplateDatabase {
  private store?: relationalStore.RdbStore;

  async init(context: Context): Promise<void> {
    this.store = await relationalStore.getRdbStore(context, DB_CONFIG);
    await this.store.executeSql(SQL_CREATE_TEMPLATES);
  }

  async insertTemplate(template: TemplateModel): Promise<void> {
    const valueBucket: relationalStore.ValuesBucket = {
      'id': template.id,
      'name': template.name,
      'display_name': template.displayName,
      'description': template.description,
      'version': template.version,
      'author': template.author,
      'license': template.license,
      'installed_path': template.installedPath,
      'is_built_in': template.isBuiltIn ? 1 : 0,
      'installed_at': Date.now(),
    };
    await this.store!.insert('templates', valueBucket);
  }

  async listTemplates(): Promise<TemplateModel[]> {
    const predicates = new relationalStore.RdbPredicates('templates');
    const resultSet = await this.store!.query(predicates);
    const templates: TemplateModel[] = [];

    while (resultSet.goToNextRow()) {
      const t = new TemplateModel();
      t.id = resultSet.getString(resultSet.getColumnIndex('id'));
      t.name = resultSet.getString(resultSet.getColumnIndex('name'));
      t.displayName = resultSet.getString(resultSet.getColumnIndex('display_name'));
      t.description = resultSet.getString(resultSet.getColumnIndex('description'));
      t.version = resultSet.getString(resultSet.getColumnIndex('version'));
      t.author = resultSet.getString(resultSet.getColumnIndex('author'));
      templates.push(t);
    }
    resultSet.close();
    return templates;
  }

  async deleteTemplate(id: string): Promise<void> {
    const predicates = new relationalStore.RdbPredicates('templates');
    predicates.equalTo('id', id);
    await this.store!.delete(predicates);
  }
}
```

---

## 9. 模板系统重构

### 9.1 当前问题

当前模板系统基于**外部可执行文件**的插件架构，这在鸿蒙平台面临严重限制：

| 问题 | 影响 | 严重度 |
| --- | --- | --- |
| HarmonyOS 不支持传统进程 fork/exec | 无法直接运行模板二进制 | 🔴 致命 |
| 应用沙箱限制 | 无法执行沙箱外的二进制文件 | 🔴 致命 |
| 模板使用不同语言编写 | 需要为每个目标平台交叉编译 | 🟡 高 |

### 9.2 重构方案

#### 方案 A（推荐）：内置模板 + ArkTS 重写

将内置模板（gongwen、jiaoan-shicao）直接用 ArkTS 重写，作为应用内置模块：

```typescript
// feature_template/src/main/ets/builtin/GongwenTemplate.ets
import { MarkdownParser } from '../../../native_typst/MarkdownParser';

export class GongwenTemplate implements TemplateExecutor {
  readonly id = 'gongwen';
  readonly displayName = '类公文模板';

  convert(markdown: string): string {
    // 1. 解析 YAML front matter
    const { frontMatter, body } = this.parseFrontMatter(markdown);

    // 2. 生成 Typst 头部
    let output = GONGWEN_PREAMBLE;

    // 3. 变量定义
    output += `#let autoTitle = "${frontMatter.title ?? '请输入文字'}"\n`;
    output += `#let autoAuthor = "${frontMatter.author ?? '请输入文字'}"\n`;

    // 4. 转换 Markdown body → Typst body
    output += this.convertBody(body);

    return output;
  }

  private parseFrontMatter(md: string): {
    frontMatter: Record<string, string>;
    body: string;
  } {
    // YAML front matter 解析逻辑
    const fmRegex = /^---\n([\s\S]*?)\n---\n([\s\S]*)$/;
    const match = md.match(fmRegex);
    if (!match) return { frontMatter: {}, body: md };

    const fm: Record<string, string> = {};
    match[1].split('\n').forEach(line => {
      const [key, ...vals] = line.split(':');
      if (key && vals.length) {
        fm[key.trim()] = vals.join(':').trim();
      }
    });
    return { frontMatter: fm, body: match[2] };
  }

  private convertBody(body: string): string {
    // Markdown → Typst 转换（等效于当前 Go 实现）
    // 推荐使用 FluidMarkdown（蚂蚁集团开源）进行原生渲染
    // 或通过 ohos-rs 桥接 Rust 的 pulldown-cmark / comrak
    return MarkdownParser.toTypst(body);
  }
}
```

#### 方案 B：Native Child Process

利用 HarmonyOS API 13+ 的 `childProcessManager.startNativeChildProcess` 运行编译为 .so 的模板逻辑：

```typescript
import { childProcessManager } from '@kit.AbilityKit';

async function runNativeTemplate(
  libName: string,
  entryPoint: string,
  markdown: string
): Promise<string> {
  // 启动原生子进程执行模板转换
  const pid = await childProcessManager.startNativeChildProcess(
    libName,     // 如 'libtemplate_gongwen.so'
    entryPoint,  // 如 'NativeTemplateConvert'
  );
  // 通过 IPC 传递数据
  // 注意：每个应用最多 512 个子进程
}
```

#### 方案 C：WASM 运行时（未来方向）

将模板编译为 WebAssembly，通过 Web 组件或 WASM 运行时执行：

```typescript
// 在 Web 组件中加载 WASM 模板
webController.loadUrl('resource://rawfile/wasm_runner.html');
webController.runJavaScript(
  `runWasmTemplate('${templateId}', '${btoa(markdown)}')`
);
```

### 9.3 推荐策略

```text
阶段 1（MVP）：内置模板用 ArkTS 重写，不支持第三方模板
阶段 2（扩展）：通过 N-API 子进程支持 Rust/C 编译的模板 .so
阶段 3（生态）：支持 WASM 格式的社区模板分发
```

---

## 10. Typst 引擎集成

### 10.1 集成方案对比

| 方案 | 优势 | 劣势 | 推荐度 |
| --- | --- | --- | --- |
| Rust → .so (ohos-rs) | 原生性能，直接链接 Typst crate | 需要 Rust 交叉编译工具链 | ⭐⭐⭐⭐⭐ |
| CLI 子进程调用 | 与现有架构一致 | HarmonyOS 不支持 exec | ❌ 不可行 |
| WASM 编译 | 跨平台一致性 | 性能较低，内存受限 | ⭐⭐⭐ |
| 云端编译 | 无本地依赖 | 需网络，延迟高 | ⭐⭐ |

### 10.2 ohos-rs 集成方案（推荐）

#### Rust 项目配置

```toml
# native_typst/Cargo.toml
[package]
name = "presto-typst"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
napi-ohos = "1.0"
napi-derive-ohos = "1.0"
typst = "0.14"
typst-pdf = "0.14"
typst-svg = "0.14"
comemo = "0.4"

[build-dependencies]
napi-build-ohos = "1.1"
```

```rust
// native_typst/build.rs
extern crate napi_build_ohos;

fn main() {
    napi_build_ohos::setup();
}
```

#### Rust 桥接代码

```rust
// native_typst/src/lib.rs
use napi_derive_ohos::napi;
use napi_ohos::bindgen_prelude::*;

/// 将 Typst 源码编译为 PDF 字节数组
#[napi]
pub fn compile_pdf(source: String, root_dir: String) -> Result<Vec<u8>> {
    let world = PrestoWorld::new(&source, &root_dir)
        .map_err(|e| Error::from_reason(format!("World init failed: {}", e)))?;

    let document = typst::compile(&world)
        .output
        .map_err(|diags| {
            let msgs: Vec<String> = diags.iter()
                .map(|d| d.message.to_string())
                .collect();
            Error::from_reason(msgs.join("\n"))
        })?;

    let pdf = typst_pdf::pdf(&document, &typst_pdf::PdfOptions::default())
        .map_err(|e| Error::from_reason(format!("PDF export failed: {}", e)))?;

    Ok(pdf)
}

/// 将 Typst 源码编译为 SVG 页面数组
#[napi]
pub fn compile_svg(source: String, root_dir: String) -> Result<Vec<String>> {
    let world = PrestoWorld::new(&source, &root_dir)
        .map_err(|e| Error::from_reason(format!("World init failed: {}", e)))?;

    let document = typst::compile(&world)
        .output
        .map_err(|diags| {
            let msgs: Vec<String> = diags.iter()
                .map(|d| d.message.to_string())
                .collect();
            Error::from_reason(msgs.join("\n"))
        })?;

    let pages: Vec<String> = document.pages.iter()
        .map(|page| typst_svg::svg(page))
        .collect();

    Ok(pages)
}
```

#### 构建与集成

```bash
# 安装 Rust OpenHarmony 目标
rustup target add aarch64-unknown-linux-ohos
rustup target add x86_64-unknown-linux-ohos

# 设置 NDK 环境变量
export OHOS_NDK_HOME=/Applications/DevEco-Studio.app/Contents/sdk/default/openharmony

# 构建
ohrs build

# 产出: libpresto_typst.so
# 复制到 entry/libs/arm64-v8a/ 或对应架构目录
```

#### ArkTS 调用

```typescript
// feature_editor/src/main/ets/service/CompileService.ets
import typstEngine from 'libpresto_typst.so';

export class CompileService {
  async compilePdf(typstSource: string, workDir: string): Promise<Uint8Array> {
    return typstEngine.compilePdf(typstSource, workDir || '/');
  }

  async compileSvg(typstSource: string, workDir: string): Promise<string[]> {
    return typstEngine.compileSvg(typstSource, workDir || '/');
  }
}
```

---

## 11. 编辑器实现方案

### 11.1 方案对比

| 方案 | 优势 | 劣势 | 推荐度 |
| --- | --- | --- | --- |
| Web 组件 + CodeMirror 6 | 复用现有编辑器代码，功能完整 | WebView 开销，通信延迟 | ⭐⭐⭐⭐⭐ |
| RichEditor 原生组件 | 原生性能，无 WebView 开销 | 功能有限，无语法高亮 | ⭐⭐ |
| TextArea + 自定义渲染 | 原生控件，简单 | 无法实现代码编辑器功能 | ⭐ |

### 11.2 推荐方案：Web 组件 + CodeMirror 6

#### 架构概述

```text
┌─────────────────────────────────────┐
│          ArkUI 宿主层               │
│  ┌─────────────────────────────┐    │
│  │     Web 组件 (WebView)       │    │
│  │  ┌───────────────────────┐  │    │
│  │  │   CodeMirror 6        │  │    │
│  │  │   (HTML/JS/CSS)       │  │    │
│  │  └───────────────────────┘  │    │
│  └──────────┬──────────────────┘    │
│             │ javaScriptProxy       │
│  ┌──────────┴──────────────────┐    │
│  │     EditorBridge (ArkTS)     │    │
│  │  - onContentChange()         │    │
│  │  - setContent()              │    │
│  │  - getContent()              │    │
│  └──────────────────────────────┘    │
└─────────────────────────────────────┘
```

#### Web 组件宿主

```typescript
// entry/src/main/ets/components/EditorComponent.ets
import { webview } from '@kit.ArkUI';

@ComponentV2
export struct EditorComponent {
  @Param onContentChange: (content: string) => void = () => {};
  @Param initialContent: string = '';

  private webController: webview.WebviewController = new webview.WebviewController();
  private bridge: EditorBridge = new EditorBridge();

  aboutToAppear(): void {
    this.bridge.onContentChange = (content: string) => {
      this.onContentChange(content);
    };
  }

  build() {
    Web({
      src: $rawfile('editor/index.html'),
      controller: this.webController,
    })
    .javaScriptAccess(true)
    .domStorageAccess(true)
    .javaScriptProxy({
      object: this.bridge,
      name: 'nativeBridge',
      methodList: ['onEditorReady', 'onContentChanged', 'onScrollChanged'],
      controller: this.webController,
    })
    .onPageEnd(() => {
      // 页面加载完成后设置初始内容
      if (this.initialContent) {
        this.webController.runJavaScript(
          `editor.setContent(${JSON.stringify(this.initialContent)})`
        );
      }
    })
    .width('100%')
    .height('100%')
  }

  // 外部调用：设置编辑器内容
  setContent(content: string): void {
    this.webController.runJavaScript(
      `editor.setContent(${JSON.stringify(content)})`
    );
  }
}
```

#### JavaScript 桥接对象

```typescript
// entry/src/main/ets/bridge/EditorBridge.ets
export class EditorBridge {
  onContentChange: (content: string) => void = () => {};
  onScrollChange: (ratio: number) => void = () => {};

  // 由 Web 端 CodeMirror 调用
  onEditorReady(): void {
    // 编辑器就绪
  }

  // 内容变化回调
  onContentChanged(content: string): void {
    this.onContentChange(content);
  }

  // 滚动变化回调
  onScrollChanged(ratio: number): void {
    this.onScrollChange(ratio);
  }
}
```

#### CodeMirror HTML（rawfile）

```html
<!-- entry/src/main/resources/rawfile/editor/index.html -->
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="stylesheet" href="editor.css">
</head>
<body>
  <div id="editor"></div>
  <script type="module">
    import { EditorView, basicSetup } from './codemirror.bundle.js';
    import { markdown } from './markdown.bundle.js';
    import { oneDark } from './one-dark.bundle.js';

    const view = new EditorView({
      parent: document.getElementById('editor'),
      extensions: [
        basicSetup,
        markdown(),
        oneDark,
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            window.nativeBridge.onContentChanged(
              update.state.doc.toString()
            );
          }
        }),
        EditorView.domEventHandlers({
          scroll(event) {
            const el = event.target;
            const ratio = el.scrollTop / (el.scrollHeight - el.clientHeight);
            window.nativeBridge.onScrollChanged(ratio);
          }
        }),
      ],
    });

    // 暴露给原生层调用的接口
    window.editor = {
      setContent(text) {
        const current = view.state.doc.toString();
        if (text !== current) {
          view.dispatch({
            changes: { from: 0, to: current.length, insert: text }
          });
        }
      },
      getContent() {
        return view.state.doc.toString();
      },
      setScrollRatio(ratio) {
        const scroller = view.scrollDOM;
        const maxScroll = scroller.scrollHeight - scroller.clientHeight;
        scroller.scrollTop = ratio * maxScroll;
      }
    };

    window.nativeBridge.onEditorReady();
  </script>
</body>
</html>
```

---

## 12. 预览系统

### 12.1 SVG 渲染方案

鸿蒙平台的 SVG 渲染可通过以下方式实现：

| 方案 | 说明 | 推荐度 |
| --- | --- | --- |
| Web 组件渲染 SVG | 在 WebView 中渲染 SVG HTML | ⭐⭐⭐⭐⭐ |
| Image 组件 + SVG 数据 | `Image($rawfile(...))` 加载 SVG | ⭐⭐⭐ |
| Canvas 自绘 | 解析 SVG 路径手动绘制 | ⭐ |

### 12.2 Web 组件预览实现

```typescript
// entry/src/main/ets/components/PreviewComponent.ets
import { webview } from '@kit.ArkUI';

@ComponentV2
export struct PreviewComponent {
  @Param svgPages: string[] = [];
  @Param scrollRatio: number = 0;
  @Param onScrollChange: (ratio: number) => void = () => {};

  private webController: webview.WebviewController = new webview.WebviewController();
  private isReady: boolean = false;

  @Monitor('svgPages')
  onSvgPagesChange(): void {
    if (this.isReady && this.svgPages.length > 0) {
      this.updatePreview();
    }
  }

  private updatePreview(): void {
    const pagesJson = JSON.stringify(this.svgPages);
    this.webController.runJavaScript(`updatePages(${pagesJson})`);
  }

  build() {
    Web({
      src: $rawfile('preview/index.html'),
      controller: this.webController,
    })
    .javaScriptAccess(true)
    .javaScriptProxy({
      object: {
        onScrollChanged: (ratio: number) => {
          this.onScrollChange(ratio);
        }
      },
      name: 'previewBridge',
      methodList: ['onScrollChanged'],
      controller: this.webController,
    })
    .onPageEnd(() => {
      this.isReady = true;
      if (this.svgPages.length > 0) {
        this.updatePreview();
      }
    })
    .width('100%')
    .height('100%')
    .backgroundColor('#2A2A2A')
  }
}
```

### 12.3 PDF 导出与保存

```typescript
// 使用文件保存对话框导出 PDF
import { picker } from '@kit.CoreFileKit';
import { fileIo } from '@ohos.file.fs';

async function exportPdf(pdfData: Uint8Array, defaultName: string): Promise<void> {
  const documentSaveOptions = new picker.DocumentSaveOptions();
  documentSaveOptions.newFileNames = [defaultName];
  documentSaveOptions.fileSuffixChoices = ['.pdf'];

  const documentPicker = new picker.DocumentViewPicker();
  const saveResult = await documentPicker.save(documentSaveOptions);

  if (saveResult && saveResult.length > 0) {
    const uri = saveResult[0];
    const file = fileIo.openSync(uri, fileIo.OpenMode.WRITE_ONLY | fileIo.OpenMode.CREATE);
    fileIo.writeSync(file.fd, pdfData.buffer);
    fileIo.closeSync(file.fd);
  }
}
```

---

## 13. 文件管理与文档架构

### 13.1 文件系统路径

| 路径 | 用途 | API |
| --- | --- | --- |
| `context.filesDir` | 应用持久化文件 | `@ohos.file.fs` |
| `context.cacheDir` | 缓存文件（可被系统清理） | `@ohos.file.fs` |
| `context.tempDir` | 临时文件 | `@ohos.file.fs` |
| 用户文件 | 文档、图片等 | `@ohos.file.picker` |

### 13.2 文件操作

```typescript
// 打开 Markdown 文件
import { picker } from '@kit.CoreFileKit';
import { fileIo } from '@ohos.file.fs';

async function openMarkdownFile(): Promise<{ content: string; dir: string } | null> {
  const documentSelectOptions = new picker.DocumentSelectOptions();
  documentSelectOptions.fileSuffixFilters = ['.md', '.markdown', '.txt'];
  documentSelectOptions.maxSelectNumber = 1;

  const documentPicker = new picker.DocumentViewPicker();
  const selectResult = await documentPicker.select(documentSelectOptions);

  if (!selectResult || selectResult.length === 0) return null;

  const uri = selectResult[0];
  const file = fileIo.openSync(uri, fileIo.OpenMode.READ_ONLY);
  const stat = fileIo.statSync(file.fd);
  const buffer = new ArrayBuffer(stat.size);
  fileIo.readSync(file.fd, buffer);
  fileIo.closeSync(file.fd);

  const content = new TextDecoder().decode(new Uint8Array(buffer));
  const dir = uri.substring(0, uri.lastIndexOf('/'));

  return { content, dir };
}
```

### 13.3 模板文件管理

```typescript
// 模板存储在应用沙箱目录
// context.filesDir + '/templates/{templateId}/'
// ├── manifest.json
// └── libtemplate_{id}.so (或 ArkTS 内置)

export class TemplateFileManager {
  private templatesDir: string;

  constructor(context: Context) {
    this.templatesDir = context.filesDir + '/templates';
  }

  getTemplatePath(templateId: string): string {
    return `${this.templatesDir}/${templateId}`;
  }

  async ensureDirectory(): Promise<void> {
    const fs = fileIo;
    if (!fs.accessSync(this.templatesDir)) {
      fs.mkdirSync(this.templatesDir, true);
    }
  }
}
```

---

## 14. 分布式能力

### 14.1 跨设备场景

| 场景 | 描述 | 技术方案 |
| --- | --- | --- |
| 文档流转 | 手机编辑→平板预览→PC 导出 | 分布式数据同步 |
| 多屏协同 | 手机编辑器 + 平板预览（两个屏幕） | 跨设备 UIAbility 调用 |
| 模板同步 | 在一台设备安装模板，其他设备自动同步 | 分布式 KV 存储 |

### 14.2 分布式 KV 存储

```typescript
import { distributedKVStore } from '@ohos.data.distributedKVStore';

const KV_OPTIONS: distributedKVStore.Options = {
  createIfMissing: true,
  encrypt: false,
  backup: false,
  autoSync: true,
  kvStoreType: distributedKVStore.KVStoreType.SINGLE_VERSION,
  securityLevel: distributedKVStore.SecurityLevel.S2,
};

export class DistributedSync {
  private kvManager?: distributedKVStore.KVManager;
  private kvStore?: distributedKVStore.SingleKVStore;

  async init(context: Context): Promise<void> {
    this.kvManager = distributedKVStore.createKVManager({
      context,
      bundleName: 'com.mrered.presto',
    });
    this.kvStore = await this.kvManager.getKVStore<distributedKVStore.SingleKVStore>(
      'presto_sync', KV_OPTIONS
    );
  }

  // 同步编辑器状态
  async syncEditorState(state: {
    markdown: string;
    templateId: string;
  }): Promise<void> {
    await this.kvStore?.put('editor_markdown', state.markdown);
    await this.kvStore?.put('editor_template', state.templateId);
  }

  // 监听远端变更
  onRemoteChange(callback: (key: string, value: string) => void): void {
    this.kvStore?.on('dataChange', distributedKVStore.SubscribeType.SUBSCRIBE_TYPE_REMOTE,
      (data) => {
        for (const entry of data.insertEntries.concat(data.updateEntries)) {
          callback(entry.key, entry.value.value as string);
        }
      }
    );
  }
}
```

---

## 15. 网络层设计

### 15.1 HTTP 客户端

```typescript
// common_utils/src/main/ets/HttpClient.ets
import { http } from '@ohos.net.http';

export class HttpClient {
  private static instance: HttpClient;

  static getInstance(): HttpClient {
    if (!HttpClient.instance) {
      HttpClient.instance = new HttpClient();
    }
    return HttpClient.instance;
  }

  async get<T>(url: string): Promise<T> {
    const httpRequest = http.createHttp();
    try {
      const response = await httpRequest.request(url, {
        method: http.RequestMethod.GET,
        header: { 'Content-Type': 'application/json' },
      });
      if (response.responseCode !== 200) {
        throw new Error(`HTTP ${response.responseCode}`);
      }
      return JSON.parse(response.result as string) as T;
    } finally {
      httpRequest.destroy();
    }
  }

  async post<T>(url: string, body: object): Promise<T> {
    const httpRequest = http.createHttp();
    try {
      const response = await httpRequest.request(url, {
        method: http.RequestMethod.POST,
        header: { 'Content-Type': 'application/json' },
        extraData: JSON.stringify(body),
      });
      if (response.responseCode !== 200) {
        throw new Error(`HTTP ${response.responseCode}`);
      }
      return JSON.parse(response.result as string) as T;
    } finally {
      httpRequest.destroy();
    }
  }

  async download(url: string): Promise<ArrayBuffer> {
    const httpRequest = http.createHttp();
    try {
      const response = await httpRequest.request(url, {
        method: http.RequestMethod.GET,
        expectDataType: http.HttpDataType.ARRAY_BUFFER,
      });
      if (response.responseCode !== 200) {
        throw new Error(`HTTP ${response.responseCode}`);
      }
      return response.result as ArrayBuffer;
    } finally {
      httpRequest.destroy();
    }
  }
}
```

### 15.2 GitHub 模板发现服务

```typescript
// feature_template/src/main/ets/service/GitHubService.ets
interface GitHubRepo {
  full_name: string;
  description: string;
  html_url: string;
  owner: { login: string };
  name: string;
}

interface SearchResult {
  items: GitHubRepo[];
}

export class GitHubService {
  private static readonly API_BASE = 'https://api.github.com';
  private client = HttpClient.getInstance();

  async discoverTemplates(): Promise<GitHubRepo[]> {
    const result = await this.client.get<SearchResult>(
      `${GitHubService.API_BASE}/search/repositories?q=topic:presto-template&sort=stars`
    );
    return result.items;
  }

  async downloadRelease(owner: string, repo: string, platform: string): Promise<ArrayBuffer> {
    // 获取 release 信息并下载对应平台的二进制
    const releases = await this.client.get<any[]>(
      `${GitHubService.API_BASE}/repos/${owner}/${repo}/releases/latest`
    );
    // 找到对应 ohos-aarch64 的资产下载
    // ...
    return new ArrayBuffer(0); // placeholder
  }
}
```

---

## 16. UI/UX 设计规范

### 16.1 设计令牌（Design Tokens）

```typescript
// common_ui/src/main/ets/theme/PrestoTheme.ets

// 颜色系统（深色主题，与现有设计一致）
export const PrestoColors = {
  // 主色
  primary: '#1E293B',
  secondary: '#334155',
  accent: '#22C55E',      // CTA 绿色

  // 背景
  background: '#0F172A',
  surface: '#1E293B',
  surfaceHover: '#2D3A4F',
  bgElevated: '#252630',

  // 文本
  text: '#F8FAFC',
  textBright: '#FFFFFF',
  textMuted: '#94A3B8',

  // 边框
  border: 'rgba(255, 255, 255, 0.08)',

  // 功能色
  danger: '#EF4444',
  warning: '#F59E0B',
  success: '#22C55E',
  info: '#3B82F6',
};

// 字体
export const PrestoFonts = {
  mono: 'JetBrains Mono, HarmonyOS Sans, monospace',
  ui: 'HarmonyOS Sans, IBM Plex Sans, sans-serif',
};

// 间距
export const PrestoSpacing = {
  xs: 4,   // vp
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48,
};

// 圆角
export const PrestoRadius = {
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
};

// 阴影
export const PrestoShadow = {
  sm: { radius: 2, offsetY: 1, color: 'rgba(0,0,0,0.05)' },
  md: { radius: 6, offsetY: 4, color: 'rgba(0,0,0,0.1)' },
  lg: { radius: 15, offsetY: 10, color: 'rgba(0,0,0,0.1)' },
};
```

### 16.2 通用按钮组件

```typescript
// common_ui/src/main/ets/components/PrestoButton.ets
@ComponentV2
export struct PrestoButton {
  @Param label: string = '';
  @Param variant: 'primary' | 'secondary' | 'danger' = 'primary';
  @Param icon?: Resource;
  @Param onClick: () => void = () => {};

  build() {
    Button() {
      Row({ space: 4 }) {
        if (this.icon) {
          Image(this.icon)
            .width(14)
            .height(14)
            .fillColor(this.variant === 'primary' ? PrestoColors.background : PrestoColors.text)
        }
        Text(this.label)
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.variant === 'primary' ? PrestoColors.background : PrestoColors.text)
      }
    }
    .padding({ left: 10, right: 10, top: 4, bottom: 4 })
    .borderRadius(PrestoRadius.sm)
    .backgroundColor(this.getBackgroundColor())
    .onClick(() => this.onClick())
  }

  private getBackgroundColor(): string {
    switch (this.variant) {
      case 'primary': return PrestoColors.accent;
      case 'secondary': return PrestoColors.surface;
      case 'danger': return PrestoColors.danger;
    }
  }
}
```

### 16.3 确认对话框

```typescript
// 模板切换确认对话框
@CustomDialog
struct TemplateSwitchDialog {
  controller: CustomDialogController;
  onUseExample: () => void = () => {};
  onKeepContent: () => void = () => {};

  build() {
    Column({ space: PrestoSpacing.md }) {
      Text('切换模板')
        .fontSize(16)
        .fontWeight(FontWeight.Bold)
        .fontColor(PrestoColors.text)

      Text('当前编辑器中有内容，切换模板后如何处理？')
        .fontSize(13)
        .fontColor(PrestoColors.textMuted)
        .lineHeight(20)

      Row({ space: PrestoSpacing.sm }) {
        PrestoButton({
          label: '使用示例内容',
          variant: 'primary',
          onClick: () => {
            this.onUseExample();
            this.controller.close();
          }
        })
        PrestoButton({
          label: '保留当前内容',
          variant: 'secondary',
          onClick: () => {
            this.onKeepContent();
            this.controller.close();
          }
        })
        PrestoButton({
          label: '取消',
          variant: 'secondary',
          onClick: () => this.controller.close()
        })
      }
      .justifyContent(FlexAlign.End)
      .width('100%')
    }
    .padding(PrestoSpacing.lg)
    .backgroundColor(PrestoColors.bgElevated)
    .borderRadius(PrestoRadius.lg)
    .border({ width: 1, color: PrestoColors.border })
    .width('90%')
    .constraintSize({ maxWidth: 400 })
  }
}
```

---

## 17. 快捷键与菜单系统

### 17.1 快捷键映射

| 当前快捷键 | 鸿蒙映射 | 功能 |
| --- | --- | --- |
| ⌘O / Ctrl+O | Ctrl+O (外接键盘) | 打开文件 |
| ⌘E / Ctrl+E | Ctrl+E (外接键盘) | 导出 PDF |
| ⌘, / Ctrl+, | Ctrl+, (外接键盘) | 打开设置 |
| ⌘F / Ctrl+F | Ctrl+F (内置于 CodeMirror) | 搜索 |
| ⌘Z / Ctrl+Z | Ctrl+Z (内置于 CodeMirror) | 撤销 |

### 17.2 键盘事件处理

```typescript
// 在页面级别监听外接键盘快捷键
@Entry
@ComponentV2
struct EditorPage {
  build() {
    Column() {
      // ... 页面内容
    }
    .onKeyEvent((event: KeyEvent) => {
      if (event.type === KeyType.Down && event.keyCode === KeyCode.KEYCODE_O
          && (event.metaKey || event.ctrlKey)) {
        this.handleOpen();
        return;
      }
      if (event.type === KeyType.Down && event.keyCode === KeyCode.KEYCODE_E
          && (event.metaKey || event.ctrlKey)) {
        this.handleExport();
        return;
      }
      if (event.type === KeyType.Down && event.keyCode === KeyCode.KEYCODE_COMMA
          && (event.metaKey || event.ctrlKey)) {
        this.navPathStack.pushPathByName('SettingsPage', null);
        return;
      }
    })
  }
}
```

---

## 18. 无障碍访问

### 18.1 无障碍支持要点

```typescript
// 所有交互组件添加无障碍标签
Button('导出 PDF')
  .accessibilityText('导出当前文档为 PDF 文件')
  .accessibilityDescription('点击后选择保存位置')

// 预览区域标记
Web({ src: $rawfile('preview/index.html') })
  .accessibilityText('文档预览')
  .accessibilityDescription('显示当前文档的排版预览效果')

// 动态内容更新通知
Text(this.errorMessage)
  .accessibilityLevel('yes')
  .accessibilityText(`错误提示: ${this.errorMessage}`)
```

---

## 19. 性能优化策略

### 19.1 关键优化点

| 优化项 | 策略 | 预期效果 |
| --- | --- | --- |
| 编辑器防抖 | 500ms debounce 转换请求 | 减少无效编译 |
| SVG 分页加载 | LazyForEach 按需渲染可见页 | 减少内存占用 |
| Typst 编译缓存 | 增量编译 + comemo 缓存 | 大幅降低编译时间 |
| Web 组件复用 | @Reusable 装饰器 | 避免 WebView 重复创建 |
| 图片懒加载 | Markdown 中图片按需加载 | 减少初始渲染时间 |

### 19.2 LazyForEach 优化预览

```typescript
// 使用 LazyForEach 按需加载 SVG 页面
class SvgPagesDataSource implements IDataSource {
  private pages: string[] = [];

  totalCount(): number { return this.pages.length; }

  getData(index: number): string { return this.pages[index]; }

  updatePages(newPages: string[]): void {
    this.pages = newPages;
    this.listeners.forEach(l => l.onDataReloaded());
  }

  // IDataSource 必须实现的监听器方法
  private listeners: DataChangeListener[] = [];
  registerDataChangeListener(listener: DataChangeListener): void {
    this.listeners.push(listener);
  }
  unregisterDataChangeListener(listener: DataChangeListener): void {
    const idx = this.listeners.indexOf(listener);
    if (idx >= 0) this.listeners.splice(idx, 1);
  }
}
```

### 19.3 @Reusable 组件复用

```typescript
@Reusable
@ComponentV2
struct SvgPageItem {
  @Param svgHtml: string = '';

  aboutToReuse(params: Record<string, Object>): void {
    this.svgHtml = params['svgHtml'] as string;
  }

  build() {
    // 渲染单页 SVG
  }
}
```

---

## 20. 测试策略

### 20.1 测试层次

| 层次 | 工具 | 覆盖范围 |
| --- | --- | --- |
| 单元测试 | ArkTS 测试框架 | ViewModel、Service、Utils |
| 组件测试 | UiTest | ArkUI 组件行为 |
| 集成测试 | ArkTS + N-API Mock | 模板转换 + Typst 编译链路 |
| E2E 测试 | UiTest (自动化) | 完整用户流程 |

### 20.2 单元测试示例

```typescript
// 测试模板转换逻辑
import { describe, it, expect } from '@ohos/hypium';
import { GongwenTemplate } from '../builtin/GongwenTemplate';

export default function gongwenTemplateTest() {
  describe('GongwenTemplate', () => {
    it('should parse front matter correctly', () => {
      const template = new GongwenTemplate();
      const input = `---
title: 测试标题
author: 测试作者
---

正文内容`;

      const output = template.convert(input);
      expect(output).assertContain('#let autoTitle = "测试标题"');
      expect(output).assertContain('#let autoAuthor = "测试作者"');
    });

    it('should handle empty input', () => {
      const template = new GongwenTemplate();
      const output = template.convert('');
      expect(output).assertContain('#let autoTitle');
    });
  });
}
```

---

## 21. 分发与部署

### 21.1 应用分发渠道

| 渠道 | 说明 | 要求 |
| --- | --- | --- |
| 华为应用市场 (AppGallery) | 官方应用商店 | AGC 注册 + 签名 + 审核 |
| 企业签名分发 | 企业内部分发 | 企业开发者证书 |

### 21.2 HAP 打包流程

```text
1. DevEco Studio 中配置签名证书（AGC）
2. Build → Build HAP(s)/APP(s)
3. 生成 .hap 文件（或 .app bundle）
4. 上传至 AppGallery Connect
5. 配置应用信息、截图、描述
6. 提交审核
```

### 21.3 签名配置

```json
// build-profile.json5
{
  "app": {
    "signingConfigs": [
      {
        "name": "default",
        "type": "HarmonyOS",
        "material": {
          "certpath": "~/.ohos/certificates/presto.cer",
          "storePassword": "****",
          "keyAlias": "presto",
          "keyPassword": "****",
          "profile": "~/.ohos/profiles/presto.p7b",
          "signAlg": "SHA256withECDSA",
          "storeFile": "~/.ohos/keystore/presto.p12"
        }
      }
    ]
  }
}
```

---

## 22. 迁移路线图

### 22.1 阶段规划

```text
Phase 1: 基础框架（2-3 周）
├── DevEco Studio 工程搭建
├── 应用入口 + 导航框架
├── 断点系统 + 多设备布局骨架
├── 设计令牌系统
└── 基础 UI 组件库

Phase 2: 核心功能（3-4 周）
├── Web 组件 + CodeMirror 编辑器集成
├── SVG 预览组件
├── ArkTS ↔ Web 双向通信桥
├── 编辑器状态管理（ViewModel）
└── 编辑器 ↔ 预览滚动同步

Phase 3: Typst 引擎（3-4 周）
├── ohos-rs 环境搭建
├── Typst Rust 封装 → .so
├── N-API 桥接层
├── PDF / SVG 编译集成
└── 内置模板 ArkTS 重写（gongwen / jiaoan-shicao）

Phase 4: 完整功能（2-3 周）
├── 文件打开/保存（Picker API）
├── PDF 导出功能
├── 设置页面
├── 模板选择器
├── 模板切换确认对话框
├── 快捷键支持
└── 错误处理和用户反馈

Phase 5: 分布式与优化（2-3 周）
├── 分布式数据同步
├── 跨设备文档流转
├── 性能优化（LazyForEach, @Reusable）
├── 无障碍适配
├── 多设备测试
└── AppGallery 上架准备
```

### 22.2 时间线概览

| 阶段 | 时长 | 里程碑 |
| --- | --- | --- |
| Phase 1 | 2-3 周 | 可运行的空壳应用，多设备布局 |
| Phase 2 | 3-4 周 | 编辑器 + 预览功能可用 |
| Phase 3 | 3-4 周 | 完整编辑→转换→预览链路 |
| Phase 4 | 2-3 周 | 功能对标现有 Web/桌面版 |
| Phase 5 | 2-3 周 | 鸿蒙特色功能 + 上架 |
| **总计** | **12-17 周** | **生产就绪版本** |

---

## 23. 风险评估与应对

| 风险 | 概率 | 影响 | 应对策略 |
| --- | --- | --- | --- |
| Typst Rust 编译到 OHOS 目标失败 | 中 | 致命 | 备选方案：WASM 编译 + Web 组件运行 |
| CodeMirror Web 组件性能不佳 | 低 | 高 | 优化 WebView 配置；备选 RichEditor |
| 模板系统无法复用现有二进制 | 高 | 高 | ArkTS 重写内置模板；.so 原生子进程 |
| ohos-rs 生态不成熟 | 中 | 中 | 回退至原生 N-API C 接口 |
| 分布式能力受限于同华为账号 | 低 | 低 | 分布式功能作为增强特性 |
| AppGallery 审核延迟 | 中 | 低 | 提前提交审核，准备企业签名分发 |
| 平板/PC 布局复杂度高 | 中 | 中 | 分阶段实现，先手机后桌面 |

---

## 附录

### A. 功能映射表

| 现有功能 | 鸿蒙实现方式 | 复杂度 |
| --- | --- | --- |
| CodeMirror 编辑器 | Web 组件 + javaScriptProxy | 中 |
| SVG 预览渲染 | Web 组件渲染 | 低 |
| Typst CLI 编译 | Rust .so + N-API (ohos-rs) | 高 |
| 文件打开对话框 | `@ohos.file.picker` | 低 |
| PDF 保存对话框 | `@ohos.file.picker` | 低 |
| 模板可执行文件 | ArkTS 重写 / .so 子进程 | 高 |
| GitHub 模板发现 | `@ohos.net.http` | 低 |
| 模板安装 | HTTP 下载 + 文件系统 | 中 |
| 设置持久化 | `@ohos.data.preferences` | 低 |
| 双向滚动同步 | Web ↔ ArkTS 双向通信 | 中 |
| 菜单栏 | 工具栏组件替代 | 低 |
| 快捷键 | `onKeyEvent` + 外接键盘 | 低 |

### B. ArkTS ↔ TypeScript 类型映射

| TypeScript | ArkTS | 说明 |
| --- | --- | --- |
| `string` | `string` | 完全一致 |
| `number` | `number` | 完全一致 |
| `boolean` | `boolean` | 完全一致 |
| `string[]` | `string[]` | 完全一致 |
| `Record<string, T>` | `Map<string, T>` / `Record<string, T>` | ArkTS 支持两种 |
| `interface` | `interface` / `class` | ArkTS 更倾向 class |
| `Promise<T>` | `Promise<T>` | 完全一致 |
| `Blob` | `ArrayBuffer` | ArkTS 使用 ArrayBuffer |
| `$state()` (Svelte) | `@Trace` / `@Local` | V2 状态装饰器 |
| `$props()` (Svelte) | `@Param` | 组件参数 |
| `$bindable()` (Svelte) | `@Provider/@Consumer` | 双向绑定 |

### C. 参考资源

#### 华为官方文档

- [HarmonyOS 开发者文档](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides-V5/application-dev-guide-V5)
- [ArkTS 语言介绍](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/arkts-overview)
- [ArkUI 组件参考](https://developer.huawei.com/consumer/cn/doc/harmonyos-references/navigation-and-switching)
- [Stage 模型开发指南](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/application-model-description)
- [一次开发多端部署](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/multi-device-app-dev)
- [分布式数据管理](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/data-sync-of-kv-store)

#### ohos-rs (Rust for HarmonyOS)

- [ohos-rs GitHub](https://github.com/ohos-rs/ohos-rs)
- [ohos-rs 官方文档](https://ohos.rs/en/docs/basic/quick-start)
- [Rust OpenHarmony 目标平台](https://doc.rust-lang.org/rustc/platform-support/openharmony.html)

#### Typst

- [Typst 官方文档](https://typst.app/docs/)
- [Typst 源码 (Apache 2.0)](https://typst.app/open-source/)

#### 开发工具

- [DevEco Studio 下载](https://developer.huawei.com/consumer/cn/deveco-studio/)
- [ohpm 包管理器](https://ohpm.openharmony.cn/)

#### Markdown 解析

- [FluidMarkdown (蚂蚁集团)](https://github.com/AntGroup/FluidMarkdown) — 原生流式 Markdown 渲染，支持 HarmonyOS，Apache 2.0
- [pulldown-cmark](https://github.com/raphlinus/pulldown-cmark) — Rust CommonMark 解析器，可通过 ohos-rs 桥接
- [comrak](https://github.com/kivikakk/comrak) — Rust GFM 兼容 Markdown 解析器

### D. 术语表

| 术语 | 全称 | 说明 |
| --- | --- | --- |
| ArkTS | Ark TypeScript | 鸿蒙应用开发语言 |
| ArkUI | Ark User Interface | 鸿蒙声明式 UI 框架 |
| HAP | HarmonyOS Ability Package | 应用入口模块包 |
| HSP | HarmonyOS Shared Package | 动态共享包 |
| HAR | HarmonyOS Archive | 静态共享包 |
| Stage 模型 | — | HarmonyOS NEXT 应用模型 |
| UIAbility | — | 包含 UI 的 Ability 类型 |
| ohos-rs | — | Rust for OpenHarmony 框架 |
| N-API | Node-API | 原生模块接口标准 |
| AGC | AppGallery Connect | 华为应用服务平台 |
| vp | virtual pixel | 鸿蒙虚拟像素单位 |
| KV Store | Key-Value Store | 分布式键值存储 |
| RDB | Relational Database | 关系型数据库 |
