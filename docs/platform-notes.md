# Presto 平台说明

## 支持的平台

Presto 支持以下平台：
- **Windows**: Windows 10/11 (x64)
- **macOS**: macOS 10.15+ (x64, ARM64)
- **Linux**: Ubuntu 20.04+ (x64)

## 平台差异

### Windows

#### 路径长度限制
- Windows 路径长度限制为 **260 字符**（MAX_PATH）
- 如果模板路径超过限制，安装会失败
- **解决方案**: 将 Presto 安装到浅层目录（如 `C:\Presto`）

#### 文件权限
- 某些操作可能需要管理员权限
- **解决方案**: 右键 Presto → 以管理员身份运行

#### 防火墙和杀毒软件
- Windows Defender 可能阻止模板下载
- **解决方案**:
  1. 允许 Presto 通过防火墙
  2. 暂时禁用杀毒软件
  3. 将 Presto 添加到杀毒软件白名单

### macOS

#### 权限问题
- 首次运行可能被阻止（未签名应用）
- **解决方案**: 系统偏好设置 → 安全性与隐私 → 允许 Presto

#### Apple Silicon (M1/M2)
- 支持 ARM64 原生运行
- 无需 Rosetta 2

### Linux

#### 依赖项
- 需要 `webkit2gtk` 库（Wails 依赖）
- **Ubuntu/Debian**: `sudo apt install libwebkit2gtk-4.0-dev`
- **Fedora/RHEL**: `sudo dnf install webkit2gtk3-devel`

#### 文件权限
- 确保 `~/.presto/templates` 目录可写
- **解决方案**: `chmod 755 ~/.presto/templates`

## 已知问题

### Windows
1. 路径长度超过 260 字符会导致安装失败
2. 某些杀毒软件可能误报（未签名）
3. 网络超时错误提示不够友好（03-03 改进）

### macOS
1. 首次运行需要手动允许（Gatekeeper）

### Linux
1. Wayland 支持有限（推荐 X11）

## 跨平台路径处理

Presto 使用 Go 的 `filepath` 包处理所有文件路径，自动适配平台分隔符：
- **Windows**: 反斜杠 `\`
- **macOS/Linux**: 正斜杠 `/`

**示例:**
```go
// ✅ Good: 使用 filepath.Join
path := filepath.Join(".presto", "templates", "official")

// ❌ Bad: 硬编码分隔符
path := ".presto/templates/official"  // Windows 会失败
```

## 字体加载位置

Presto 在转换和预览 PDF 时会把字体目录传给 Typst。默认字体目录如下：

- **本机服务端**: 当前用户的 `~/.presto/fonts`
- **Docker Web 端**: 容器内的 `/home/presto/.presto/fonts`

Docker Compose 部署默认把本地目录 `./.presto-data/fonts` 挂载到 `/home/presto/.presto/fonts`。需要安装模板依赖字体时，把 `.ttf`、`.otf`、`.ttc` 或 `.otc` 字体文件放进 `./.presto-data/fonts`，然后重启 Presto 服务让字体列表重新扫描。

如果目录曾由 root 或其他用户创建，先修正权限，确保当前用户可以复制字体文件：

```bash
sudo chown -R "$(id -u):$(id -g)" .presto-data
```

桌面端当前主要依赖 Typst 自身可发现的系统字体目录。

如果需要加载额外目录，可以设置 `FONT_PATHS` 环境变量覆盖默认值，多个目录用冒号分隔：

```bash
FONT_PATHS="$HOME/.presto/fonts:/usr/share/fonts" presto-server
```

## 日志文件位置

默认日志输出到 stderr，使用 `--log-file` 参数指定日志文件：
```bash
# Windows
.\Presto.exe --log-file presto.log

# macOS/Linux
./presto-desktop --log-file presto.log
```

### 日志轮转

Presto 自动轮转日志文件以避免占用过多磁盘空间：

- **最大文件大小**: 10MB
- **保留历史文件**: 最多 5 个
- **轮转策略**: 当日志文件达到 10MB 时，自动轮转并创建新文件
- **文件命名**:
  - 当前日志: `presto.log`
  - 轮转后: `presto-{timestamp}.log` (由 lumberjack 自动管理)

**示例：**

```text
~/.presto/logs/
├── presto.log              (当前日志，10MB 以内)
├── presto-2026-04-03T10-00-00.log  (轮转文件 1)
├── presto-2026-04-03T11-30-00.log  (轮转文件 2)
├── ...
└── presto-2026-04-03T15-00-00.log  (轮转文件 5)
```

超过 5 个历史文件后，最旧的文件会被自动删除。

### 断点续传

Presto 支持模板下载的断点续传功能：

**工作原理：**
1. 下载时，Presto 将部分下载的文件保存在临时目录中
   - **Windows**: `%TEMP%\presto-downloads\`
   - **macOS/Linux**: `/tmp/presto-downloads/`

2. 如果下载中断（网络错误、超时、手动停止），临时文件会被保留

3. 下次下载相同模板时，Presto 会：
   - 检测临时文件的存在
   - 使用 HTTP Range 请求从断点继续
   - 如果服务器不支持 Range，则从头开始（优雅降级）

**临时文件管理：**
- 文件名：URL hash + `.tmp`（例如：`a1b2c3d4e5f6.tmp`）
- 清理时机：
  - 下载成功后立即清理
  - 应用启动时自动清理所有临时下载文件
  - 手动清理：可以安全删除 `presto-downloads/` 目录

**注意事项：**
- 断点续传依赖服务器支持 HTTP Range 请求
- GitHub Releases 和大多数 CDN 都支持 Range 请求
- 如果服务器不支持，Presto 会自动降级到普通下载（不影响功能）

**日志示例：**
```
INFO [download] resuming from partial download offset_bytes=5242880
INFO [download] completed with resume support bytes=10485760
```

## 报告问题

如果您遇到平台特定问题，请提供：
1. 操作系统版本
2. Presto 版本（`Presto --version`）
3. 详细错误消息（使用 `--verbose` 参数）
4. 日志文件（使用 `--log-file` 参数）

GitHub Issues: https://github.com/Presto-io/Presto/issues
