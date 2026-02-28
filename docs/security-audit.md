# Presto 安全审计报告

**初审日期：** 2026-02-19
**复审日期：** 2026-02-19
**三审日期：** 2026-02-25
**四审日期：** 2026-02-28
**审计范围：** 全部 Go 后端、Svelte 前端、Wails 桌面集成、Docker/CI 部署、模板供应链、跨仓库 CI/CD、Homepage 前端
**审计方式：** 白盒代码审计

---

## 四审总览（2026-02-28）— 跨仓库安全加固

| 状态 | 数量 | 说明 |
|------|------|------|
| 已修复 | 43 | 漏洞已完全消除（含 SEC-30/39/41/NEW-01） |
| 部分修复 | 3 | 有缓解措施但仍存在残留风险（SEC-01/10/24） |
| 未修复 | 0 | 无 |

### 四审修复内容

- **SEC-30（严重→已修复）**: Install() 重构为「下载→验证→执行」三步流程，社区模板无校验时记录警告
- **SEC-39（中→已修复）**: 安装 API 不再接受客户端 URL，服务端从 registry 查询下载地址
- **SEC-41（中→已修复）**: 提供 docker-compose.tls.yml + Caddy 自动 HTTPS 反向代理配置
- **NEW-01（低-中→已修复）**: release workflow 移除 --clobber，已存在 release 时报错而非覆盖
- **域名白名单日志**: 下载/checksum URL 被白名单拦截时记录安全日志
- **Homepage postMessage**: 添加 event.origin 同源校验
- **Homepage CSP**: 添加 Content-Security-Policy 响应头
- **Homepage iframe sandbox**: 移除不必要的 allow-popups-to-escape-sandbox
- **CI/CD 权限**: 全仓库 11 个 workflow 添加 permissions 最小权限声明
- **CI/CD Action 固定**: 全部第三方 Action 从 tag 固定到 commit SHA
- **依赖审计**: govulncheck + npm audit 全仓库扫描，修复 Homepage rollup 高危漏洞
- **自动化**: 添加 security-scan.yml 周期扫描 + Dependabot 自动依赖更新

## 三审总览（2026-02-25）

| 状态 | 数量 | 说明 |
|------|------|------|
| 已修复 | 39 | 漏洞已完全消除 |
| 部分修复 | 3 | 有缓解措施但仍存在残留风险（SEC-01/10/24） |
| 未修复 | 4 | SEC-30/39/41 架构变更待定，NEW-01 策略待定 |

## 复审总览（2026-02-19）

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
| SEC-04 | **严重** | XSS (CWE-79) | **已修复** | SVG + StoreView README 均已 DOMPurify 消毒 |
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
| SEC-15 | **中** | 信息泄露 (CWE-209) | **已修复** | 所有 handler 统一使用通用错误消息 |
| SEC-16 | **中** | JSON 注入 (CWE-74) | **已修复** | json.NewEncoder 正确编码 |
| SEC-17 | **中** | 输入校验缺失 (CWE-20) | **已修复** | 正则 `^[a-zA-Z0-9][a-zA-Z0-9._-]*$` + 100 字符限制 |
| SEC-18 | **中** | HTTP 状态码未检查 (CWE-252) | **已修复** | checkHTTPStatus 统一检查 |
| SEC-19 | **中** | 无速率限制 (CWE-770) | **已修复** | 令牌桶（10 req/s, burst 30） |
| SEC-20 | **中** | HTTP 客户端无超时 (CWE-400) | **已修复** | 30s 超时自定义 Client |
| SEC-21 | **中** | Docker 以 root 运行 (CWE-250) | **已修复** | 非 root 用户 `presto` |
| SEC-22 | **中** | 供应链-编译器 (CWE-494) | **已修复** | Docker 构建传入 SHA256 参数，CI 硬编码校验，Windows CI 已补全 |
| SEC-23 | **中** | 供应链-交叉编译 (CWE-494) | **已修复** | SHA256 硬编码验证 |
| SEC-24 | **中** | CI 密钥风险 | **部分修复** | 权限文档化但 PAT 范围仍过大 |
| SEC-25 | **低** | 竞态条件 (CWE-367) | **已修复** | crypto/rand 随机后缀 |
| SEC-26 | **低** | ReDoS (CWE-1333) | **已修复** | 正则元字符转义 |
| SEC-27 | **低** | 静态文件泄露 (CWE-538) | **已修复** | dotfile 过滤中间件 |
| SEC-28 | **低** | 文件权限过宽 (CWE-732) | **已修复** | 0700 权限 |
| SEC-29 | **严重** | 供应链 RCE (CWE-494) | **已修复** | Dockerfile 模板二进制 SHA256 校验基础设施 |
| SEC-30 | **严重** | 代码注入 (CWE-94) | **未修复** | Install 执行未验证二进制提取 manifest |
| SEC-31 | **高** | 供应链 (CWE-494) | **已修复** | Windows CI Typst SHA256 硬编码校验 |
| SEC-32 | **高** | 供应链 (CWE-494) | **已修复** | Docker 构建传入 Typst SHA256 参数 |
| SEC-33 | **高** | 网络暴露 (CWE-668) | **已修复** | docker-compose 绑定 127.0.0.1 |
| SEC-34 | **高** | XSS (CWE-79) | **已修复** | StoreView README DOMPurify 消毒 |
| SEC-35 | **中** | 信息泄露 (CWE-209) | **已修复** | 统一使用通用错误消息 |
| SEC-36 | **中** | 缺少安全头 (CWE-693) | **已修复** | securityHeadersMiddleware 设置安全响应头 |
| SEC-37 | **中** | CORS 遗漏 (CWE-942) | **已修复** | Allow-Methods 添加 PATCH |
| SEC-38 | **中** | TOCTOU + symlink (CWE-367/59) | **已修复** | Uninstall 使用 os.Lstat + symlink 检测 |
| SEC-39 | **中** | SSRF (CWE-918) | **未修复** | 客户端可控下载 URL + SHA256 |
| SEC-40 | **中** | 文件读取 (CWE-552) | **已修复** | 桌面端编译器 root 改为 os.TempDir |
| SEC-41 | **中** | 明文传输 (CWE-319) | **未修复** | 服务端无 TLS 支持 |
| SEC-42 | **中** | 容器加固 (CWE-250) | **已修复** | docker-compose read_only + cap_drop + no-new-privileges |
| SEC-43 | **低** | 敏感日志 (CWE-532) | **已修复** | API Key 仅输出截断版本 |
| SEC-44 | **低** | 返回值忽略 (CWE-252) | **已修复** | os.UserHomeDir/MkdirAll 错误检查 + log.Fatal |
| SEC-45 | **低** | 文件权限 (CWE-732) | **已修复** | registry 缓存/manifest 统一 0600 |
| SEC-46 | **低** | 重定向 (CWE-295) | **已修复** | registry HTTP 客户端添加 CheckRedirect 校验 |
| SEC-47 | **低** | 资源泄露 (CWE-404) | **已修复** | defer 改为立即 Close |

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

### SEC-22: Typst 二进制供应链校验（中） — ✅ 三审已完全修复

- **原问题：** Typst 二进制下载无任何完整性校验
- **已修复：** Dockerfile 和 Makefile 支持 `TYPST_SHA256` 参数，传入时进行 SHA256 验证
- **三审修复：**
  1. SEC-31: Windows CI 添加 SHA256 硬编码校验
  2. SEC-32: `docker/build-push-action` 传入 `TYPST_SHA256_AMD64`/`TYPST_SHA256_ARM64` build-args
- **残留风险：** 无（所有平台均已校验）

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

### NEW-02: 桌面端 CheckForUpdate 无 HTTP 超时（低） — ✅ 已修复

- **文件：** `cmd/presto-desktop/main.go`
- **类型：** DoS (CWE-400)

**描述：** `CheckForUpdate` 函数仍使用 `http.Get`（即 `http.DefaultClient`，无超时），属于 SEC-20 的遗漏。慢速服务器可无限阻塞桌面应用。

**修复：** 使用 `&http.Client{Timeout: 15 * time.Second}` 替代 `http.Get`。

### NEW-03: 速率限制为全局令牌桶（低-中）

- **文件：** `internal/api/middleware.go`
- **类型：** DoS (CWE-770)

**描述：** 速率限制器为单一全局实例（非 per-IP），单个攻击者可耗尽所有客户端的配额。在桌面模式下影响较小，但服务端部署场景下构成 DoS 向量。

**建议：** 服务端模式使用 per-IP 令牌桶。

### NEW-04: API Key 比较非常量时间（低） — ✅ 已修复

- **文件：** `internal/api/middleware.go:63`
- **类型：** 时序攻击 (CWE-208)

**描述：** `auth[7:] != apiKey` 使用 Go 字符串比较（非常量时间），理论上可通过时序侧信道逐字节推导 API Key。网络环境下利用难度较高。

**修复：** 使用 `subtle.ConstantTimeCompare([]byte(auth[7:]), []byte(apiKey)) != 1` 替代字符串比较。

---

## 已修复漏洞摘要

以下 38 个漏洞已完全修复，此处仅列出修复方式：

| 编号 | 修复方式 |
|------|----------|
| SEC-02 | 服务端使用 `os.MkdirTemp` 临时目录；桌面端使用 `$HOME`；编译器警告 root 为 `/` |
| SEC-03 | `validateWorkDir()` 校验绝对路径、不含 `..`、存在且为目录 |
| SEC-04 | DOMPurify + SVG profile 消毒，窄范围 ADD_TAGS/ADD_ATTR 白名单（三审发现 StoreView README 遗漏，见 SEC-34） |
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
| SEC-29 | Dockerfile 模板二进制 SHA256 校验基础设施（ARG 传入，下载后验证） |
| SEC-31 | Windows CI Typst SHA256 硬编码校验 |
| SEC-32 | `docker/build-push-action` 传入 `TYPST_SHA256_AMD64`/`TYPST_SHA256_ARM64` |
| SEC-33 | `docker-compose.yml` 端口绑定 `127.0.0.1:8080:8080` |
| SEC-34 | StoreView README `{@html}` 添加 `DOMPurify.sanitize()` |
| SEC-35 | 统一使用通用错误消息（`"rename failed"`、`"import failed"` 等） |
| SEC-36 | `securityHeadersMiddleware` 设置 `X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy` |
| SEC-37 | CORS `Allow-Methods` 添加 `PATCH` |
| SEC-38 | `os.Lstat()` + symlink 模式检测，拒绝 symlink 目标 |
| SEC-40 | 桌面端编译器 root 改为 `os.TempDir()` |
| SEC-42 | `docker-compose.yml` 添加 `read_only`、`cap_drop`、`no-new-privileges`、`tmpfs` |
| SEC-43 | API Key 仅输出截断版本（前8后4） |
| SEC-44 | `os.UserHomeDir()`/`os.MkdirAll()` 错误检查 + `log.Fatal` |
| SEC-45 | registry 缓存/manifest/import 文件统一 `0600` 权限 |
| SEC-46 | registry HTTP 客户端添加 `CheckRedirect` 域名白名单校验 |
| SEC-47 | `defer resp.Body.Close()` 改为读取后立即 `Close()` |
| NEW-02 | `CheckForUpdate` 使用 `&http.Client{Timeout: 15 * time.Second}` |
| NEW-04 | `subtle.ConstantTimeCompare` 替代字符串比较 |

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

### 长期改进

1. **SEC-10:** 研究 OS 级沙箱方案（macOS sandbox-exec / Linux seccomp）
2. **SEC-24:** 迁移到 GitHub App Token
3. **NEW-03:** 服务端模式使用 per-IP 速率限制
4. **NEW-01:** 考虑不可变 release 策略

---

## 三审新发现（2026-02-25）

### SEC-29: Dockerfile 模板二进制零校验后执行（严重）

- **文件：** `Dockerfile:44-58`
- **类型：** 供应链 RCE (CWE-494)

**描述：** Docker 构建阶段从 GitHub Release 下载模板二进制后，**无任何 SHA256/签名校验** 即在构建过程中执行 `--manifest` 提取元数据。被污染的 Release 直接导致构建环境 RCE。

```dockerfile
curl -sSL -o "/templates/$tpl/presto-template-$tpl" \
  "https://github.com/.../presto-template-${tpl}-${SUFFIX}" && \
chmod +x "/templates/$tpl/presto-template-$tpl" && \
"/templates/$tpl/presto-template-$tpl" --manifest > "/templates/$tpl/manifest.json"
```

**建议：** 硬编码每个模板二进制的 SHA256 校验值，下载后验证通过才执行。

### SEC-30: Install 执行未验证二进制提取 manifest（严重）

- **文件：** `internal/template/github.go:236-244`
- **类型：** 代码注入 (CWE-94)

**描述：** `Install()` 下载模板二进制后，在安装完成前就执行它来获取 manifest。`lookupChecksumFromRelease()` 的校验来自同一 GitHub Release（同源校验，第 180 行注释承认 "same-source, weaker"），无法防御被攻陷的仓库。

```go
executor := NewExecutor(tmpPath)
manifestBytes, err := executor.GetManifest()  // 执行未验证的二进制
```

**建议：** 从 registry 获取独立源的校验值后再执行；或不执行二进制而通过 ZIP 内附带的 manifest.json 获取元数据。

### SEC-31: Windows CI Typst 二进制零校验（高）

- **文件：** `.github/workflows/release.yml:149-153`
- **类型：** 供应链 (CWE-494)

**描述：** Windows 构建直接 `curl` + `unzip` Typst 二进制并打包进 Release 产物，无任何 SHA256 校验。该二进制随 Release 分发给最终用户。对比同 workflow 中 llvm-mingw 下载（第 122-129 行）已正确使用 SHA256 硬编码验证（SEC-23）。

```yaml
curl -sL "https://github.com/typst/typst/releases/download/v${{ env.TYPST_VERSION }}/${{ matrix.typst_archive }}" -o typst.zip
unzip -qo typst.zip -d typst-tmp
```

**建议：** 添加 SHA256 硬编码校验，与 llvm-mingw 使用相同模式。

### SEC-32: Docker 构建未传 Typst SHA256 参数（高）

- **文件：** `.github/workflows/release.yml:273-281`
- **类型：** 供应链 (CWE-494)

**描述：** `docker/build-push-action` 未传入 `TYPST_SHA256_AMD64`/`TYPST_SHA256_ARM64` build-args，Dockerfile 中对应参数默认为空字符串，导致每次 Docker 镜像发布均跳过 Typst 校验（仅打印 WARNING）。

**建议：** 在 release.yml 中添加 `build-args` 传入已知 SHA256 值。

### SEC-33: docker-compose 暴露到所有网络接口（高）

- **文件：** `docker-compose.yml:5` + `Dockerfile:78`
- **类型：** 网络暴露 (CWE-668)

**描述：** `ports: "8080:8080"` 绑定到 `0.0.0.0`，加上 Dockerfile 中 `ENV HOST=0.0.0.0` 覆盖了服务端默认的 `127.0.0.1`（SEC-14），公网可直接访问未加密的 API 服务。

**建议：** 改为 `127.0.0.1:8080:8080`，或在文档中强制要求反向代理。

### SEC-34: StoreView README 渲染 XSS（高）

- **文件：** `frontend/src/lib/components/StoreView.svelte:438`
- **类型：** XSS (CWE-79)

**描述：** `{@html renderMarkdown(readmeContent)}` 使用 `marked` 渲染从远程 URL 拉取的 README，**未经 DOMPurify 消毒**。自定义 renderer 仅过滤了 `<a>` 和 `<img>` 标签，但无法阻止 `<script>`、事件处理器属性（如 `onerror`、`onload`）等注入。

对比：项目中所有其他 `{@html}` 使用点（`Preview.svelte:58`、`showcase/hero/+page.svelte:98`）均已正确使用 DOMPurify，属于 SEC-04 修复的遗漏。

```svelte
<!-- 当前：未消毒 -->
<div class="readme-body">{@html renderMarkdown(readmeContent)}</div>

<!-- 建议：添加 DOMPurify -->
<div class="readme-body">{@html DOMPurify.sanitize(renderMarkdown(readmeContent))}</div>
```

**建议：** 在 `renderMarkdown` 输出后添加 `DOMPurify.sanitize()` 消毒。

### SEC-35: 多处 handler 返回 err.Error()（中）

- **文件：** `internal/api/templates.go:162`、`internal/api/import.go:149`、`internal/api/batch_import.go:239`
- **类型：** 信息泄露 (CWE-209)

**描述：** 部分 API handler 通过 `writeJSONError(w, err.Error(), ...)` 直接返回内部错误信息，可能包含文件系统路径、模板目录结构、OS 级错误消息等。对比 SEC-15 已修复的模式（使用通用错误消息），这些位置属于遗漏。

```go
// templates.go:162 — 泄露重命名错误详情
writeJSONError(w, err.Error(), http.StatusBadRequest)

// import.go:149 — 泄露导入错误详情
writeJSONError(w, err.Error(), http.StatusBadRequest)
```

**建议：** 统一使用通用错误消息（如 `"rename failed"`、`"import failed"`），详细信息仅写入服务端日志。

### SEC-36: 缺少安全响应头（中）

- **文件：** `internal/api/middleware.go`、`internal/api/server.go`
- **类型：** 保护机制缺失 (CWE-693)

**描述：** 服务端未设置以下安全响应头：

- `X-Content-Type-Options: nosniff` — 防止 MIME 类型嗅探
- `X-Frame-Options: DENY` — 防止点击劫持
- `Content-Security-Policy` — 防止 XSS 和数据注入

中间件链（`loggingMiddleware` → `corsMiddleware` → `authMiddleware` → `rateLimitMiddleware`）中无安全头中间件。

**建议：** 添加 `securityHeadersMiddleware` 设置上述响应头。

### SEC-37: CORS Allow-Methods 缺少 PATCH（中）

- **文件：** `internal/api/middleware.go:26`
- **类型：** CORS 配置遗漏 (CWE-942)

**描述：** CORS 中间件设置 `Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS`，但 `server.go:65` 注册了 `PATCH /api/templates/{id}` 路由。前端跨域 PATCH 请求将在 preflight 阶段被拒绝。

```go
// middleware.go:26 — 缺少 PATCH
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")

// server.go:65 — 注册了 PATCH
s.mux.HandleFunc("PATCH /api/templates/{id}", s.handleRenameTemplate)
```

**建议：** 添加 `PATCH` 到 `Access-Control-Allow-Methods`。

### SEC-38: Uninstall TOCTOU 竞态 + symlink 跟随（中）

- **文件：** `internal/template/github.go:324-332`
- **类型：** TOCTOU (CWE-367) / symlink (CWE-59)

**描述：** `Uninstall()` 先调用 `os.Stat()`（跟随 symlink），再调用 `os.RemoveAll()`。两步之间存在竞态窗口：攻击者可将目标目录替换为指向敏感目录（如 `~/.ssh`）的 symlink，`os.Stat` 返回 `IsDir() == true`（指向目标），`RemoveAll` 删除 symlink 指向的真实目录。

```go
info, err := os.Stat(tplDir)   // 跟随 symlink
if !info.IsDir() { ... }
return os.RemoveAll(tplDir)     // 删除 symlink 目标
```

**建议：** 使用 `os.Lstat()` 检测 symlink 并拒绝；或使用 `os.Remove` 先移除 symlink 本身。

### SEC-39: 客户端可控下载 URL + SHA256（中）

- **文件：** `internal/api/templates.go:117-125`
- **类型：** SSRF / 完整性绕过 (CWE-918)

**描述：** `handleInstallTemplate` API 接受客户端请求体中的 `Platforms[].URL` 和 `Platforms[].SHA256`，作为 `InstallOpts` 传入 `Install()`。虽然 URL 域名受白名单限制（SEC-07），但攻击者可提供自签匹配的 URL + Hash 对——指向 GitHub 上任意仓库的任意 Release 产物，SHA256 校验将始终通过。

```go
opts = &template.InstallOpts{
    DownloadURL:    info.URL,       // 客户端控制
    ExpectedSHA256: info.SHA256,    // 客户端控制
}
```

**建议：** 仅从 registry 获取信任的 URL 和 SHA256，不接受客户端提供的值；或客户端提供的 SHA256 必须与 registry 中的值匹配。

### SEC-40: 桌面端编译器 root 为 $HOME（中）

- **文件：** `cmd/presto-desktop/main.go:494`
- **类型：** 任意文件读取 (CWE-552)

**描述：** 桌面端将 Typst 编译器 `--root` 设为用户主目录（`$HOME`），而服务端正确使用 `os.MkdirTemp` 临时目录。这意味着 Typst 的 `#read` 函数可读取 `~/.ssh/`、`~/.aws/credentials`、`~/.gnupg/` 等敏感文件。恶意 Markdown 文档经模板转换后的 Typst 代码可利用此路径窃取数据。

```go
// SEC-02: Use $HOME instead of "/" to restrict file access to user's home
compiler := typst.NewCompilerWithRoot(home)
```

**建议：** 将桌面端编译器 root 限制为当前工作目录或文档所在目录，而非整个 `$HOME`。

### SEC-41: 服务端无 TLS 支持（中）

- **文件：** `cmd/presto-server/main.go:75`
- **类型：** 明文传输 (CWE-319)

**描述：** 服务端仅支持 HTTP，API Key 以 `Authorization: Bearer <key>` 头明文传输。若通过 `HOST=0.0.0.0` 暴露到网络（见 SEC-33），API Key 可被网络嗅探。

**建议：** 添加 `--tls-cert` / `--tls-key` 参数支持 `http.ListenAndServeTLS`；或在文档中强制要求反向代理终结 TLS。

### SEC-42: docker-compose 无容器安全加固（中）

- **文件：** `docker-compose.yml`
- **类型：** 权限过大 (CWE-250)

**描述：** docker-compose 配置缺少以下安全加固选项：

- `read_only: true`（只读根文件系统）
- `security_opt: ["no-new-privileges:true"]`
- `cap_drop: ["ALL"]`
- `tmpfs`（可写临时目录）
- `mem_limit` / `cpus`（资源限制）

容器以默认 Linux 能力集运行。

**建议：** 添加上述安全加固选项。

### SEC-43: API Key 输出到 stdout（低）

- **文件：** `cmd/presto-server/main.go:74`
- **类型：** 敏感信息日志 (CWE-532)

**描述：** 服务端自动生成的 API Key 通过 `fmt.Printf("API Key: %s\n", apiKey)` 输出到 stdout。在容器化部署中，stdout 被 Docker logs、日志聚合系统（CloudWatch、journald 等）捕获，Key 以明文持久化在多个系统中。

**建议：** 移除 stdout 输出，改为仅在 DEBUG 级别日志中显示；或提供环境变量方式传入 Key 避免自动生成。

### SEC-44: os.UserHomeDir / os.MkdirAll 错误被忽略（低）

- **文件：** `cmd/presto-desktop/main.go:480`、`cmd/presto-server/main.go:29`
- **类型：** 返回值未检查 (CWE-252)

**描述：** 两个入口文件中 `os.UserHomeDir()` 和 `os.MkdirAll()` 的错误被 `_` 忽略。若 `UserHomeDir` 失败（如无 `$HOME` 的容器环境），返回空字符串，导致 `templatesDir` 变为相对路径 `.presto/templates`，后续操作在不可预期的目录进行。

```go
home, _ := os.UserHomeDir()         // 错误被忽略
os.MkdirAll(templatesDir, 0755)     // 错误被忽略
```

**建议：** 检查错误并在启动时 `log.Fatal` 退出。

### SEC-45: Registry 缓存 / manifest 文件权限 0644（低）

- **文件：** `internal/template/registry.go:158`、`internal/template/github.go:272`、`internal/template/manager.go:150,209`
- **类型：** 文件权限过宽 (CWE-732)

**描述：** 注册表缓存文件和模板 manifest.json 使用 `0644` 权限（所有者读写，组/其他可读），而目录和二进制使用 `0700`（SEC-28）。在多用户系统中，其他用户可读取 manifest（低风险）或修改 registry 缓存注入伪造的 SHA256 值影响校验决策。

**建议：** 统一使用 `0600` 权限。

### SEC-46: Registry HTTP 客户端无重定向校验（低）

- **文件：** `internal/template/registry.go:123`
- **类型：** 重定向验证缺失 (CWE-295)

**描述：** `fetchFromCDN()` 使用独立的 `http.Client{Timeout: fetchTimeout}`，未复用 `github.go` 中带 `CheckRedirect` 域名校验的 `httpClient`。CDN 被攻陷或 DNS 劫持时，可将注册表请求重定向到攻击者控制的服务器，注入伪造的模板信任数据和 SHA256 值。

```go
// registry.go:123 — 无重定向校验
client := &http.Client{Timeout: fetchTimeout}

// 对比 github.go:21-33 — 有 CheckRedirect 域名白名单
```

**建议：** 复用或创建带 `CheckRedirect` 校验的 HTTP 客户端。

### SEC-47: defer resp.Body.Close() 在 for 循环内（低）

- **文件：** `internal/template/github.go:290`
- **类型：** 资源泄露 (CWE-404)

**描述：** `lookupChecksumFromRelease` 中 `defer resp.Body.Close()` 位于 `for` 循环内。Go 的 `defer` 在函数返回时才执行，非循环迭代结束时。当前代码因 early return 仅执行一次迭代无实际风险，但若未来修改循环逻辑（如不再 early return），将导致多个 HTTP 连接未及时关闭。

**建议：** 将 defer 改为 `resp.Body.Close()` 在数据读取完成后立即调用，或提取为独立函数。

---

## 三审新发现攻击链

### 攻击链 4：恶意模板商店 README → XSS → Wails 全权限

```
攻击者在 GitHub 上托管含恶意 HTML 的 README
  → 用户在 StoreView 浏览模板商店
  → SEC-34: README 内容经 marked 渲染但未 DOMPurify 消毒
  → {@html} 直接注入恶意脚本到 WebView
  → 桌面端: Wails binding 暴露 SaveFile、ImportBatchZip、DeleteTemplate 等方法
  → 攻击者可读写文件、删除模板、执行编译
```

**状态：** ✅ **已阻断**（SEC-34 已修复：DOMPurify 消毒阻断 XSS 注入）

### 攻击链 5：供应链 → Docker 镜像 → 用户 RCE

```
攻击者控制模板 GitHub Release（或 MITM）
  → SEC-29: Dockerfile 下载模板二进制无校验
  → 二进制在构建时执行 --manifest
  → 构建环境 RCE，可注入后门到最终 Docker 镜像
  → 用户拉取被污染的镜像
```

**状态：** ✅ **已缓解**（SEC-29 已修复：Dockerfile 模板二进制 SHA256 校验；SEC-30 仍未修复但需攻陷 GitHub Release）

### 攻击链 6：客户端控制安装源 → 任意二进制执行

```
攻击者向 API 发送 installTemplate 请求
  → SEC-39: 请求体包含自选的 URL + 自签 SHA256
  → SEC-07: URL 域名在白名单内（指向 GitHub 上任意仓库）
  → SEC-30: 下载后立即执行获取 manifest
  → 安装并执行攻击者选择的任意 GitHub Release 二进制
```

**状态：** ⚠️ **可利用**（需 API Key 认证，桌面模式跳过认证时风险更高）

---

## 三审修复优先级

### 立即修复 — ✅ 全部完成

1. ~~**SEC-34:** StoreView `{@html}` 添加 DOMPurify 消毒~~ ✅
2. ~~**NEW-04:** `middleware.go:53` API Key 比较改用 `subtle.ConstantTimeCompare`~~ ✅
3. ~~**SEC-33:** `docker-compose.yml` 端口改为 `127.0.0.1:8080:8080`~~ ✅
4. ~~**SEC-31:** `release.yml` Windows Typst 下载添加 SHA256 校验~~ ✅
5. ~~**SEC-32:** `release.yml` Docker build 传入 Typst SHA256 build-args~~ ✅
6. ~~**SEC-37:** CORS `Allow-Methods` 添加 `PATCH`~~ ✅
7. ~~**SEC-35:** 统一错误消息，不返回 `err.Error()`~~ ✅

### 短期改进 — ✅ 全部完成

1. ~~**SEC-29:** Dockerfile 模板下载添加 SHA256 校验~~ ✅
2. **SEC-39:** 安装 API 不接受客户端提供的 URL+SHA256，仅从 registry 获取 — **未修复（架构变更）**
3. ~~**SEC-36:** 添加安全响应头中间件~~ ✅
4. ~~**SEC-38:** Uninstall 使用 `os.Lstat` 检测 symlink~~ ✅
5. ~~**NEW-02:** `CheckForUpdate` 使用带超时的 HTTP Client~~ ✅
6. ~~**SEC-45:** 文件权限统一为 `0600`~~ ✅
7. ~~**SEC-43:** 移除 stdout API Key 输出~~ ✅

### 长期加固

1. **SEC-10:** 模板二进制 OS 级沙箱（macOS sandbox-exec / Linux seccomp）
2. **NEW-03:** 服务端模式 per-IP 速率限制
3. ~~**SEC-40:** 桌面端 Typst root 限制为工作目录而非 `$HOME`~~ ✅
4. **SEC-41:** 服务端支持 TLS 或强制要求反向代理
5. **SEC-24:** 迁移 PAT 为 GitHub App Token
6. **NEW-01:** 考虑不可变 release 策略
