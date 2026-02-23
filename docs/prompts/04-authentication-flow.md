# 提示词：模板认证与 GPG 签名验证流程设计

## 背景

Presto 是一个 Markdown → Typst → PDF 桌面排版工具。模板是独立的二进制程序，用户从模板商店安装。需要设计一套模板认证体系，防止恶意模板攻击用户。

## 信任等级体系

| Trust | 条件 | UI 标识 |
|-------|------|---------|
| `official` | 仓库 owner = `Presto-io` GitHub 组织 | 蓝色盾牌 `#3b82f6` |
| `verified` | GPG 签名有效（公钥在 registry 中注册） | 绿色对勾 `#22c55e` |
| `community` | 在 registry 中，无有效 GPG 签名 | 灰色标签 |
| `unrecorded` | 不在 registry 中（手动 URL 安装） | 警告标识 |

## 现有安全措施

1. **SEC-01**：SHA256 校验已实现但可选——如果 release 无 `SHA256SUMS` 文件，校验被跳过
2. **SEC-10**：模板执行时 env 清空、30s 超时，但无 OS 级沙箱
3. **官方模板保护**：`internal/template/builtin.go` 硬编码保护，不可删除/修改/重命名
4. **社区模板开关**：设置页有"启用社区模板"开关，默认关闭

## 需要设计的完整流程

### 1. GPG 签名流程（模板开发者侧）

```
开发者操作：
1. 生成 GPG 密钥对（或使用已有的）
2. 在 template-registry 提交 PR，注册公钥：
   - 文件路径：keys/{github-username}.asc
   - PR 内容：公钥 ASCII armor 格式
   - 需要仓库维护者审核合并
3. Release 时：
   a. CI 自动生成 SHA256SUMS（对所有 release 二进制计算）
   b. 开发者手动（或 CI 用 secret）对 SHA256SUMS 做 GPG 签名：
      gpg --armor --detach-sign SHA256SUMS
      → 生成 SHA256SUMS.sig
   c. 将 SHA256SUMS 和 SHA256SUMS.sig 都上传到 GitHub Release
```

### 2. Registry 索引时的验证（template-registry CI 侧）

```
registry CI（build_registry.py）在发现新 release 时：
1. 下载 SHA256SUMS 和 SHA256SUMS.sig
2. 从 keys/{owner-or-author}.asc 读取对应公钥
3. 使用 gpg --verify SHA256SUMS.sig SHA256SUMS 验证签名
4. 如果签名有效：
   - trust = "verified"
   - 将 SHA256SUMS 中的 hash 值存入 registry.json
5. 如果签名无效或不存在：
   - trust = "community"
   - 仍然收录，但不标记为 verified
6. 如果 owner = "Presto-io"：
   - trust = "official"（无论是否有 GPG 签名）
```

### 3. 客户端安装时的验证（Presto 软件侧）

```
用户在模板商店点击"安装"：
1. 从 registry.json 读取模板信息，包括：
   - trust 等级
   - 期望的 SHA256 hash（registry CI 已预先存储）
   - GitHub release URL
2. 下载对应平台的二进制
3. 计算下载文件的 SHA256
4. 与 registry.json 中存储的 hash 对比：
   - 匹配 → 安装成功
   - 不匹配 → 拒绝安装，提示可能被篡改
5. 安装后标记 trust 等级，在模板列表中显示对应徽章
```

**关键安全改进**：SHA256 hash 存在 registry-deploy（Cloudflare Pages），二进制存在 GitHub Release。攻击者需要同时攻破两个独立系统。

### 4. 手动 URL 安装的处理

```
用户通过 URL 手动安装模板：
1. 标记为 unrecorded
2. 弹出安全警告对话框：
   "此模板不在官方注册表中，未经过安全审核。
    模板二进制程序将在你的系统上运行，可能存在安全风险。
    确定要安装吗？"
3. 用户确认后安装
4. 不进行 SHA256 验证（因为没有可信来源的 hash）
```

### 5. registry.json 格式变更

当前格式（空的）：
```json
{
  "version": 1,
  "updatedAt": "",
  "categories": [],
  "templates": []
}
```

新格式：
```json
{
  "version": 2,
  "updatedAt": "2026-02-23T12:00:00Z",
  "templates": [
    {
      "name": "gongwen",
      "displayName": "类公文模板",
      "description": "...",
      "version": "1.0.0",
      "author": "Presto-io",
      "repo": "Presto-io/presto-official-templates",
      "license": "MIT",
      "category": "公文",
      "keywords": ["公文", "通知", "报告"],
      "trust": "official",
      "platforms": {
        "darwin-arm64": {
          "url": "https://github.com/Presto-io/presto-official-templates/releases/download/v1.0.0/presto-template-gongwen-darwin-arm64",
          "sha256": "a1b2c3d4..."
        },
        "darwin-amd64": {
          "url": "...",
          "sha256": "..."
        },
        "linux-arm64": { "url": "...", "sha256": "..." },
        "linux-amd64": { "url": "...", "sha256": "..." },
        "windows-arm64": { "url": "...", "sha256": "..." },
        "windows-amd64": { "url": "...", "sha256": "..." }
      },
      "minPrestoVersion": "0.1.0",
      "requiredFonts": [...],
      "previewImage": "previews/gongwen.svg"
    }
  ]
}
```

变更要点：
- 移除 `categories` 数组（category 现在是自由文本，客户端自行聚合）
- 每个模板的 `platforms` 对象包含各平台的下载 URL 和 SHA256
- SHA256 由 registry CI 在索引时从 release 的 SHA256SUMS 文件提取并记录
- `trust` 由 registry CI 根据 owner + GPG 签名判定

### 6. GPG 公钥注册目录结构

在 template-registry 仓库中：

```
template-registry/
├── keys/
│   ├── Presto-io.asc          # 官方组织公钥
│   ├── some-developer.asc      # 社区开发者公钥
│   └── another-dev.asc
├── scripts/
│   └── build_registry.py       # 索引构建脚本（需要添加 GPG 验证逻辑）
├── registry.json
└── ...
```

## 需要修改的文件/仓库

| 仓库 | 文件 | 变更 |
|------|------|------|
| `template-registry` | `scripts/build_registry.py` | 添加 GPG 验证逻辑、SHA256 提取、trust 判定 |
| `template-registry` | `registry.json` | 格式升级到 v2 |
| `template-registry` | `keys/` 目录 | 新建，存放开发者公钥 |
| `Presto`（主仓库） | `internal/template/github.go` | 安装时从 registry.json 读取 SHA256 校验（而非从 release 的 checksums.txt） |
| `Presto`（主仓库） | `internal/template/builtin.go` | 官方模板判定改为从 registry 读取 trust 字段（而非硬编码） |
| `Presto`（主仓库） | `frontend/src/lib/api/types.ts` | trust 类型已包含 official/verified/community，无需改 |
| 三个 starter | `CONVENTIONS.md` | 添加 trust 说明和 GPG 签名指南（见 02-starter-updates.md） |

## 实施优先级

1. **P0**：registry.json 格式升级 + SHA256 per-platform 存储（即使暂时不做 GPG，先把 hash 存到 registry 中就比现在安全很多）
2. **P1**：客户端安装时从 registry.json 读 SHA256 校验（替代当前从 release checksums.txt 读取的方式）
3. **P2**：GPG 公钥注册 + registry CI 签名验证 + verified 标识
4. **P3**：OS 级沙箱（SEC-10，独立议题）

P0+P1 就能解决 SEC-01 的核心问题（"校验文件来自同一不可信来源"）。GPG 签名是锦上添花。
