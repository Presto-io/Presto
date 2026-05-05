# Registry 协议规范

**版本**: 1.0
**最后更新**: 2026-03-17

本文档定义 Presto 模板 Registry 的数据格式、安全策略和验证规则。

## 1. Registry JSON Schema

### 1.1 完整格式

```json
{
  "version": "1.0",
  "templates": [
    {
      "id": "gongwen",
      "name": "公文模板",
      "description": "政府公文格式模板",
      "owner": "Presto-io",
      "repo": "presto-template-gongwen",
      "trust": "official",
      "platforms": [
        {
          "os": "darwin",
          "arch": "arm64",
          "url": "https://github.com/Presto-io/presto-template-gongwen/releases/download/v1.0.0/presto-template-gongwen-darwin-arm64",
          "cdn_url": "https://cdn.presto.c-1o.top/templates/binaries/v1.0.0/gongwen/darwin-arm64",
          "sha256": "abc123def456..."
        },
        {
          "os": "linux",
          "arch": "amd64",
          "url": "https://github.com/Presto-io/presto-template-gongwen/releases/download/v1.0.0/presto-template-gongwen-linux-amd64",
          "cdn_url": "https://cdn.presto.c-1o.top/templates/binaries/v1.0.0/gongwen/linux-amd64",
          "sha256": "def789abc012..."
        }
      ]
    }
  ]
}
```

### 1.2 字段说明

**顶层字段:**
- `version` (string, 必需): Registry 版本,当前为 "1.0"
- `templates` (array, 必需): 模板列表

**模板对象字段:**
- `id` (string, 必需): 模板唯一标识符
- `name` (string, 必需): 模板显示名称
- `description` (string, 可选): 模板描述
- `owner` (string, 必需): GitHub 仓库所有者
- `repo` (string, 必需): GitHub 仓库名称
- `trust` (string, 必需): 信任级别,见第 5 节
- `platforms` (array, 必需): 支持的平台列表

**平台对象字段:**
- `os` (string, 必需): 操作系统 (darwin, linux, windows)
- `arch` (string, 必需): 架构 (amd64, arm64)
- `url` (string, 必需): GitHub 下载 URL
- `cdn_url` (string, 可选): CDN 镜像 URL
- `sha256` (string, 可选/必需): SHA256 校验和 (十六进制编码)

## 2. SHA256 校验策略

### 2.1 校验规则

根据模板的信任级别,`sha256` 字段的必需性不同:

| 信任级别 | SHA256 字段 | 行为 |
|---------|------------|------|
| `official` | **必需** | 缺失时拒绝安装 |
| `verified` | **必需** | 缺失时拒绝安装 |
| `community` | **可选** | 缺失时记录警告,允许安装 |

### 2.2 验证流程

```
下载二进制 → 计算 SHA256 → 对比 ExpectedSHA256
                ↓
        [匹配?]
         ↓    ↓
        是    否 → [信任级别?]
         ↓         ↓      ↓
      安装成功  official/  community
                verified   ↓
                  ↓      记录警告
               拒绝安装  允许安装
```

### 2.3 错误处理

- **SHA256 不匹配**: 拒绝安装,返回 `ErrChecksumMismatch`
- **official/verified 缺失 SHA256**: 拒绝安装,返回错误
- **community 缺失 SHA256**: 记录日志警告,继续安装

**日志示例:**
```
[security] WARNING: installing owner/repo without SHA256 verification (hash: abc123...)
```

## 3. 下载 URL 域名白名单

### 3.1 允许的域名

所有 `url` 和 `cdn_url` 字段的域名必须在以下白名单中:

**GitHub 域名:**
- `github.com`
- `api.github.com`
- `objects.githubusercontent.com`
- `github-releases.githubusercontent.com`
- `github.githubassets.com`
- `codeload.github.com`
- `release-assets.githubusercontent.com`

**CDN 域名:**
- `presto.c-1o.top`
- `cdn.presto.c-1o.top`

### 3.2 验证实现

域名白名单验证在以下位置强制执行:

1. **二进制下载** (`downloadWithRetry` 函数):
   ```go
   parsedURL, err := url.Parse(downloadURL)
   if !isAllowedDownloadHost(parsedURL.Host) {
       return ErrNotFound
   }
   ```

2. **Checksum 文件下载** (`lookupChecksumFromRelease` 函数):
   ```go
   checksumURL, err := url.Parse(asset.BrowserDownloadURL)
   if !isAllowedDownloadHost(checksumURL.Host) {
       log.Printf("[security] BLOCKED: checksum URL host not in whitelist")
       return ""
   }
   ```

3. **HTTP 重定向验证** (所有 HTTP Client 的 `CheckRedirect`):
   ```go
   CheckRedirect: func(req *http.Request, via []*http.Request) error {
       if !isAllowedDownloadHost(req.URL.Host) {
           return fmt.Errorf("redirect to disallowed host: %s", req.URL.Host)
       }
       return nil
   }
   ```

### 3.3 错误处理

非白名单域名的下载请求将被拒绝,返回 `ErrNotFound` 错误:

```
[security] BLOCKED: download URL host not in whitelist: evil.com
```

## 4. 社区模板安全警告

### 4.1 当前实现 (Phase 2)

当用户安装缺少 SHA256 的社区模板时:

**后端行为:**
```
[security] WARNING: installing {owner}/{repo} without SHA256 verification
```

**前端行为:**
- 当前版本: 不显示警告
- 未来版本 (Phase 3+): 显示确认对话框

### 4.2 未来计划 (Phase 3+)

前端将实现确认对话框:

```
┌─────────────────────────────────────┐
│  ⚠️  安全警告                       │
├─────────────────────────────────────┤
│  该模板缺少安全校验 (SHA256)。      │
│  安装未经校验的二进制可能存在风险。 │
│                                     │
│  是否继续安装?                      │
│                                     │
│  [取消]  [继续安装]                 │
└─────────────────────────────────────┘
```

**设计理由:**
- 功能完整性和用户体验的权衡
- Phase 2 优先核心安全功能 (域名白名单、SHA256 校验)
- UI 改进推迟到后续版本

## 5. 信任级别定义

### 5.1 信任级别说明

| 级别 | 含义 | SHA256 | 验证流程 |
|------|------|--------|---------|
| `official` | Presto 官方维护的模板 | 必需 | 完整安全验证,CI/CD 自动发布 |
| `verified` | 第三方验证的模板 | 必需 | 代码审查,签名验证 |
| `community` | 社区贡献的模板 | 可选 | 无强制验证,用户自行评估 |

### 5.2 信任级别分配

**official:**
- 由 `Presto-io` 组织维护
- 代码审查和自动化测试
- CI/CD 自动构建和发布
- 示例: `presto-template-gongwen`, `presto-template-official`

**verified:**
- 第三方开发者提交
- 经过 Presto 团队代码审查
- 可能需要签名验证
- 示例: `partner-template-verified`

**community:**
- 任何开发者发布到 GitHub
- 包含 `presto-template` topic
- 无强制审查流程
- 用户自行评估风险

## 6. 下载优先级策略

### 6.1 CDN-first 下载顺序

```
1. registry 提供 cdn_url?
   ↓
2. [cdn_url 可用?]
   ↓        ↓
  是        否
   ↓        ↓
CDN下载   GitHub URL下载
   ↓        ↓
3. [下载成功?]
   ↓    ↓
  是    否
   ↓    ↓
验证SHA256  尝试GitHub URL
   ↓
安装成功
```

### 6.2 超时和重试

- **连接超时**: 10s
- **响应头超时**: 15s
- **总体超时**: 120s (大文件)
- **重试次数**: 3 次
- **退避策略**: 指数退避 (1s, 2s, 4s)

## 7. 安全考虑

### 7.1 防护措施

**SECU-01**: SHA256 校验在执行前完成
- 下载 → 验证 → 执行 (三步流程)
- 校验失败则拒绝安装

**SECU-05**: 域名白名单覆盖所有下载路径
- 二进制下载验证
- Checksum 文件下载验证
- HTTP 重定向验证

**SECU-06**: 路径遍历防护
- 模板名称验证 (正则表达式)
- 路径解析后验证在 TemplatesDir 内

**SECU-28**: 临时文件权限控制
- 二进制文件权限: 0700
- Manifest 文件权限: 0600
- 临时目录权限: 0700

### 7.2 已知限制

- **社区模板 SHA256 可选**: 后端记录警告,前端不显示 (Phase 2)
- **HTTP 重定向限制**: 最多 10 次重定向
- **文件大小限制**: Checksum 文件最大 1MB

## 8. Registry 维护指南

### 8.1 添加新模板

1. 确定信任级别 (official/verified/community)
2. 生成平台二进制文件
3. 计算 SHA256 校验和
4. 上传到 GitHub Release
5. 更新 Registry JSON
6. 验证 URL 在白名单中

### 8.2 更新模板版本

1. 构建新版本二进制
2. 计算新的 SHA256
3. 发布 GitHub Release
4. 更新 Registry JSON 中的 URL 和 SHA256
5. 测试安装流程

### 8.3 安全审计

定期检查:
- Registry JSON 完整性
- SHA256 校验和有效性
- 域名白名单更新
- 信任级别分配合理性

---

**相关文档:**
- `security-audit.md`: 安全审计报告
- `ai-reference.md`: Presto 技术架构参考
- Phase 2 计划文档: `.planning/phases/02-security-hardening/`

**版本历史:**
- v1.0 (2026-03-17): 初始版本,Phase 2 安全加固完成
