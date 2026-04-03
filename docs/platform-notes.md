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

## 日志文件位置

默认日志输出到 stderr，使用 `--log-file` 参数指定日志文件：
```bash
# Windows
.\Presto.exe --log-file presto.log

# macOS/Linux
./presto-desktop --log-file presto.log
```

## 报告问题

如果您遇到平台特定问题，请提供：
1. 操作系统版本
2. Presto 版本（`Presto --version`）
3. 详细错误消息（使用 `--verbose` 参数）
4. 日志文件（使用 `--log-file` 参数）

GitHub Issues: https://github.com/Presto-io/Presto/issues
