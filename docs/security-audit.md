# Presto 安全审计报告

**初审日期：** 2026-02-19
**复审日期：** 2026-02-19
**审计范围：** 全部 Go 后端、Svelte 前端、Wails 桌面集成、Docker/CI 部署、模板供应链
**审计方式：** 白盒代码审计

---

## 复审总览

| 状态 | 数量 | 说明 |
|------|------|------|
| 已修复 | 22 | 漏洞已完全消除 |
| 部分修复 | 4 | 有缓解措施但仍存在残留风险 |
| 新发现 | 4 | 复审中新发现的问题 |

---

## 漏洞状态一览

| 编号 | 原严重性 | 类型 | 状态 | 简述 |
|------|----------|------|------|------|
| SEC-01 | **严重** | RCE (CWE-494) | **部分修复** | SHA256 校验可选，无签名验证 |
| SEC-02 | **严重** | 任意文件读取 (CWE-552) | **已修复** | Typst root 限制为临时目录 |
| SEC-03 | **严重** | 任意文件写入 (CWE-22/73) | **已修复** | workDir 校验为绝对路径且不含 `..` |
| SEC-04 | **严重** | XSS (CWE-79) | **已修复** | DOMPurify SVG 消毒 |
| SEC-05 | **高** | 目录穿越删除 (CWE-22) | **已修复** | filepath.Base + validateName + 路径边界检查 |
| SEC-06 | **高** | 目录穿越安装 (CWE-22) | **已修复** | 名称正则校验 + 绝对路径边界检查 |
| SEC-07 | **高** | SSRF (CWE-918) | **已修复** | GitHub 域名白名单 + 重定向拦截 |
| SEC-08 | **高** | CORS 配置错误 (CWE-942) | **已修复** | 来源白名单（localhost + Wails） |
| SEC-09 | **高** | 缺少认证 (CWE-306) | **已修复** | Bearer Token 认证（服务端自动生成） |
| SEC-10 | **高** | 无进程沙箱 (CWE-250) | **部分修复** | 环境变量清空，但无 OS 级沙箱 |
| SEC-11 | **中** | DoS (CWE-400) | **已修复** | 10MB 请求体限制 |
| SEC-12 | **中** | DoS (CWE-400) | **已修复** | 编译 60s / 执行 30s 超时 |
| SEC-13 | **中** | DoS (CWE-400) | **已修复** | 100MB 下载限制 |
| SEC-14 | **中** | 网络暴露 (CWE-668) | **已修复** | 默认 127.0.0.1 |
| SEC-15 | **中** | 信息泄露 (CWE-209) | **已修复** | 通用错误消息 + 服务端日志 |
| SEC-16 | **中** | JSON 注入 (CWE-74) | **已修复** | json.NewEncoder 正确编码 |
| SEC-17 | **中** | 输入校验缺失 (CWE-20) | **已修复** | 正则 `^[a-zA-Z0-9][a-zA-Z0-9._-]*$` + 100 字符限制 |
| SEC-18 | **中** | HTTP 状态码未检查 (CWE-252) | **已修复** | checkHTTPStatus 统一检查 |
| SEC-19 | **中** | 无速率限制 (CWE-770) | **已修复** | 令牌桶（10 req/s, burst 30） |
| SEC-20 | **中** | HTTP 客户端无超时 (CWE-400) | **已修复** | 30s 超时自定义 Client |
| SEC-21 | **中** | Docker 以 root 运行 (CWE-250) | **已修复** | 非 root 用户 `presto` |
| SEC-22 | **中** | 供应链-编译器 (CWE-494) | **部分修复** | 校验基础设施存在但默认为空 |
| SEC-23 | **中** | 供应链-交叉编译 (CWE-494) | **已修复** | SHA256 硬编码验证 |
| SEC-24 | **中** | CI 密钥风险 | **部分修复** | 权限文档化但 PAT 范围仍过大 |
| SEC-25 | **低** | 竞态条件 (CWE-367) | **已修复** | crypto/rand 随机后缀 |
| SEC-26 | **低** | ReDoS (CWE-1333) | **已修复** | 正则元字符转义 |
| SEC-27 | **低** | 静态文件泄露 (CWE-538) | **已修复** | dotfile 过滤中间件 |
| SEC-28 | **低** | 文件权限过宽 (CWE-732) | **已修复** | 0700 权限 |

---

## 部分修复漏洞详情

### SEC-01: 模板二进制完整性验证（严重 → 中）

- **原问题：** 下载并执行未经任何验证的二进制文件
- **已修复：** 从 release 的 `checksums.txt` / `SHA256SUMS` 获取哈希值进行 SHA256 校验
- **残留风险：**
  1. 校验为可选——若 release 不包含校验文件则静默跳过
  2. 无 GPG/cosign 签名验证
  3. 校验文件来自同一不可信来源（攻击者控制的仓库可同时伪造二进制和校验文件）
- **建议：** 要么强制要求校验文件存在，要么实现独立的签名验证机制

### SEC-10: 模板二进制沙箱（高 → 中）

- **原问题：** 以完整用户权限执行模板二进制
- **已修复：** 清空环境变量，仅保留最小 `PATH=/usr/local/bin:/usr/bin:/bin`；30s 执行超时
- **残留风险：**
  1. 无 OS 级沙箱（seccomp-bpf / macOS sandbox-exec / 容器隔离）
  2. 仍有完整文件系统读写权限
  3. 仍有完整网络访问权限
  4. 可生成不受限子进程
- **建议：** macOS 使用 `sandbox-exec`，Linux 使用 seccomp-bpf 或 namespace 隔离

### SEC-22: Typst 二进制供应链校验（中）

- **原问题：** Typst 二进制下载无任何完整性校验
- **已修复：** Dockerfile 和 Makefile 支持 `TYPST_SHA256` 参数，传入时进行 SHA256 验证
- **残留风险：**
  1. 默认值为空字符串，不传参时仅打印警告不阻断构建
  2. CI workflow 的 `docker/build-push-action` 未传入校验参数
  3. Windows CI 构建完全无校验基础设施（直接 `curl` + `unzip`）
- **建议：** 将已知 SHA256 硬编码为默认值，构建时强制验证

### SEC-24: CI 密钥范围（中）

- **原问题：** `BREWFORGE_PAT` 权限范围可能过大
- **已修复：** 添加了最小权限范围文档注释；顶层 `permissions` 已限定为 `contents: write` + `packages: write`
- **残留风险：**
  1. `BREWFORGE_PAT` 仍使用经典 PAT 的 `repo` scope（授予所有仓库完整读写权限）
  2. 建议迁移到 fine-grained PAT 或 GitHub App Token，仅限 `brewforge/homebrew-more` 仓库
- **建议：** 使用 GitHub App Installation Token 替代 PAT

---

## 复审新发现

### NEW-01: Release 产物可被覆盖（低-中）

- **文件：** `.github/workflows/release.yml`
- **类型：** 供应链 (CWE-494)

**描述：** `gh release upload --clobber` 允许在 re-run 时静默覆盖已发布的二进制和校验文件。若 CI 被攻陷或 workflow 被恶意重触发，可替换合法产物。

**建议：** 考虑使用不可变 release 或在覆盖前添加审批步骤。

### NEW-02: 桌面端 CheckForUpdate 无 HTTP 超时（低）

- **文件：** `cmd/presto-desktop/main.go`
- **类型：** DoS (CWE-400)

**描述：** `CheckForUpdate` 函数仍使用 `http.Get`（即 `http.DefaultClient`，无超时），属于 SEC-20 的遗漏。慢速服务器可无限阻塞桌面应用。

**建议：** 使用带超时的自定义 HTTP Client。

### NEW-03: 速率限制为全局令牌桶（低-中）

- **文件：** `internal/api/middleware.go`
- **类型：** DoS (CWE-770)

**描述：** 速率限制器为单一全局实例（非 per-IP），单个攻击者可耗尽所有客户端的配额。在桌面模式下影响较小，但服务端部署场景下构成 DoS 向量。

**建议：** 服务端模式使用 per-IP 令牌桶。

### NEW-04: API Key 比较非常量时间（低）

- **文件：** `internal/api/middleware.go:63`
- **类型：** 时序攻击 (CWE-208)

**描述：** `auth[7:] != apiKey` 使用 Go 字符串比较（非常量时间），理论上可通过时序侧信道逐字节推导 API Key。网络环境下利用难度较高。

```go
// 当前
if !strings.HasPrefix(auth, "Bearer ") || auth[7:] != apiKey {
// 建议
if !strings.HasPrefix(auth, "Bearer ") || subtle.ConstantTimeCompare([]byte(auth[7:]), []byte(apiKey)) != 1 {
```

---

## 已修复漏洞摘要

以下 22 个漏洞已完全修复，此处仅列出修复方式：

| 编号 | 修复方式 |
|------|----------|
| SEC-02 | 服务端使用 `os.MkdirTemp` 临时目录；桌面端使用 `$HOME`；编译器警告 root 为 `/` |
| SEC-03 | `validateWorkDir()` 校验绝对路径、不含 `..`、存在且为目录 |
| SEC-04 | DOMPurify + SVG profile 消毒，窄范围 ADD_TAGS/ADD_ATTR 白名单 |
| SEC-05 | `filepath.Base()` + `validateName()` + 绝对路径边界检查 + 目录存在性验证 |
| SEC-06 | 名称正则校验 + `filepath.Base()` + 绝对路径不超出 TemplatesDir |
| SEC-07 | `allowedDownloadHosts` 域名白名单 + `CheckRedirect` 拦截非白名单重定向 |
| SEC-08 | CORS 来源白名单（localhost:8080/5173, 127.0.0.1, wails://wails）+ `Vary: Origin` |
| SEC-09 | Bearer Token 认证；服务端自动生成 32 字节随机密钥；桌面模式跳过认证 |
| SEC-11 | `http.MaxBytesReader(w, r.Body, 10<<20)` 限制所有请求体 |
| SEC-12 | `exec.CommandContext` + `context.WithTimeout`：编译 60s，执行 30s |
| SEC-13 | `io.LimitReader(binResp.Body, 100<<20)` 限制下载 |
| SEC-14 | 默认绑定 `127.0.0.1:8080`，可通过 `HOST` 环境变量覆盖 |
| SEC-15 | `writeJSONError` 返回通用错误消息，`log.Printf` 记录详细信息 |
| SEC-16 | `json.NewEncoder(w).Encode(map[string]string{"error": msg})` |
| SEC-17 | 双层校验：API 层 `isValidGitHubName()` + 模板层 `validateName()` |
| SEC-18 | `checkHTTPStatus()` 统一检查所有 HTTP 响应 2xx |
| SEC-19 | 令牌桶速率限制器（10 req/s, burst 30） |
| SEC-20 | 自定义 `httpClient` 30s 超时 |
| SEC-21 | Dockerfile 创建非 root 用户 `presto`，`COPY --chown`，`USER presto` |
| SEC-23 | SHA256 硬编码验证 + 失败时 `exit 1` |
| SEC-25 | `crypto/rand` 生成 16 hex 字符随机后缀：`.presto-temp-<random>.typ` |
| SEC-26 | 正则元字符转义后再构造 RegExp |
| SEC-27 | `dotfileFilterHandler` 中间件过滤 `.` 开头的路径段 |
| SEC-28 | 目录和二进制均使用 `0700` 权限 |

---

## 原始攻击链状态

### 攻击链 1：从网页访问到远程代码执行 — **已阻断**

```
用户访问恶意网站
  → SEC-08 (CORS) ✅ 来源白名单阻断跨域请求
  → SEC-09 (认证) ✅ Bearer Token 阻断未授权调用
  → 攻击链终止
```

### 攻击链 2：从网页访问到文件系统窃取 — **已阻断**

```
用户访问恶意网站
  → SEC-08 (CORS) ✅ 来源白名单阻断
  → SEC-02 (--root) ✅ 编译器 root 限制为临时目录
  → 攻击链终止
```

### 攻击链 3：从网页访问到任意文件删除 — **已阻断**

```
用户访问恶意网站
  → SEC-08 (CORS) ✅ 来源白名单阻断
  → SEC-05 (路径校验) ✅ validateName + 边界检查
  → 攻击链终止
```

---

## 剩余风险优先级

### 建议修复（加固纵深防御）

1. **SEC-01:** 强制要求校验文件或实现签名验证
2. **SEC-22:** 硬编码 Typst SHA256 默认值，Windows CI 添加校验
3. **NEW-02:** 桌面端 CheckForUpdate 使用带超时的 HTTP Client

### 长期改进

1. **SEC-10:** 研究 OS 级沙箱方案（macOS sandbox-exec / Linux seccomp）
2. **SEC-24:** 迁移到 GitHub App Token
3. **NEW-03:** 服务端模式使用 per-IP 速率限制
4. **NEW-04:** API Key 比较使用 `subtle.ConstantTimeCompare`
5. **NEW-01:** 考虑不可变 release 策略
