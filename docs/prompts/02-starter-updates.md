# 提示词：更新三个模板脚手架仓库

## 背景

以下三个 GitHub 仓库是 Presto 模板开发脚手架，需要同步更新：
- `Presto-io/presto-template-starter-go`
- `Presto-io/presto-template-starter-rust`
- `Presto-io/presto-template-starter-typescript`

## 变更清单

### 1. 添加 `--version` flag（协议变更，三个仓库都要改）

在现有的 `--manifest` 和 `--example` 基础上，增加 `--version` flag：

```
./binary --version   → 输出版本号字符串（从 manifest.json 的 version 字段读取），然后退出
```

**Go (starter-go)**：

在 main.go 中添加：
```go
import "encoding/json"

// 在 flag 定义区域添加：
versionFlag := flag.Bool("version", false, "output version from manifest")

// 在 flag.Parse() 之后、manifestFlag 检查之前添加：
if *versionFlag {
    var m map[string]interface{}
    if err := json.Unmarshal(manifestData, &m); err == nil {
        if v, ok := m["version"]; ok {
            fmt.Println(v)
        }
    }
    return
}
```

**Rust (starter-rust)**：

在 src/main.rs 中，Cli struct 添加：
```rust
#[arg(long)]
version_flag: bool,    // --version
```

注意：clap 的 `--version` 可能和内置冲突，需要用 `#[arg(long = "version")]` 并重命名字段。或者解析 MANIFEST JSON 手动处理。

在 main() 中：
```rust
if cli.version_flag {
    let manifest: serde_json::Value = serde_json::from_str(MANIFEST).unwrap();
    if let Some(v) = manifest.get("version") {
        println!("{}", v.as_str().unwrap_or("unknown"));
    }
    return;
}
```

需要添加 `serde_json` 依赖到 Cargo.toml。

**TypeScript (starter-typescript)**：

在 src/index.ts 中添加：
```typescript
if (args.includes("--version")) {
  process.stdout.write((manifest as any).version + "\n");
  process.exit(0);
}
```

### 2. category 字段改为自由文本（三个仓库都要改）

**manifest.json 变更**：

将 category 从枚举改为自由文本，限制规则：
- 最大 20 个字符
- 只允许中文、英文字母、数字、空格、连字符
- 必须非空

旧的 manifest.json：
```json
"category": "other",
```

新的 manifest.json（starter 默认值）：
```json
"category": "通用",
```

**CONVENTIONS.md 变更**：

找到 category 字段的说明，将枚举列表改为：

```
| `category` | 是 | string | 模板分类标签，自由文本，最大 20 字符，只允许中文/英文/数字/空格/连字符。示例："公文"、"教育"、"简历"、"学术论文"、"商务" |
```

删除旧的枚举列表：`government`, `education`, `business`, `academic`, `legal`, `resume`, `creative`, `other`

**make test 变更**：

在 Makefile 的 test target 中，添加 manifest schema 校验步骤。在 `--manifest | python3 -m json.tool` 之后添加：

```bash
# 校验 category 字段
./$(BINARY) --manifest | python3 -c "
import json, sys, re
m = json.load(sys.stdin)
cat = m.get('category', '')
if not cat:
    print('ERROR: category is empty', file=sys.stderr); sys.exit(1)
if len(cat) > 20:
    print(f'ERROR: category too long ({len(cat)} > 20)', file=sys.stderr); sys.exit(1)
if not re.match(r'^[\u4e00-\u9fff\w\s-]+$', cat):
    print(f'ERROR: category contains invalid characters: {cat}', file=sys.stderr); sys.exit(1)
print(f'  category: {cat} ✓')
"
```

### 3. trust 字段说明（CONVENTIONS.md 变更，三个仓库都要改）

在 CONVENTIONS.md 的 manifest schema 说明之后，添加一个新小节：

```markdown
### 信任等级（trust）

trust 字段**不由模板自己声明**，而是由 Presto 的 template-registry 在索引时自动判定：

| Trust | 条件 | 含义 |
|-------|------|------|
| `official` | 仓库 owner 是 `Presto-io` 组织 | 官方出品 |
| `verified` | Release 的 SHA256SUMS 文件有有效的 GPG 签名（公钥在 registry 中注册） | 开发者身份已验证，二进制未被篡改 |
| `community` | 在 registry 中，无有效签名 | 仅收录，未审核 |
| `unrecorded` | 不在 registry 中 | 用户手动 URL 安装 |

模板开发者不需要在 manifest.json 中添加 trust 字段。如果你希望你的模板获得 `verified` 标识，需要：
1. 生成 GPG 密钥对
2. 在 template-registry 注册你的公钥
3. Release 时对 SHA256SUMS 文件进行 GPG 签名，生成 SHA256SUMS.sig
```

### 4. --version 的 make test 验证（三个仓库都要改）

在 Makefile 的 test target 中添加 --version 测试：

```makefile
test: build
	@echo "Testing manifest..."
	@./$(BINARY) --manifest | python3 -m json.tool > /dev/null
	@echo "Testing example round-trip..."
	@./$(BINARY) --example | ./$(BINARY) > /dev/null
	@echo "Testing version..."
	@./$(BINARY) --version > /dev/null
	@echo "All tests passed."
```

## 执行顺序

1. 先改 CONVENTIONS.md（category、trust、--version 文档）
2. 改 manifest.json（category 默认值）
3. 改源码（添加 --version flag）
4. 改 Makefile（添加测试）
5. 运行 `make test` 验证
6. Commit 消息：`feat: 添加 --version flag、category 自由文本、trust 说明`
