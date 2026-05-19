# Offline Portable Verification

离线便携包的验收目标是证明 portable 渠道没有外部在线路径。静态审计只能证明代码和打包配置包含必要 gate；真实网络静默必须通过打包应用的运行时 smoke 观察确认。

## 静态审计

在 `Presto/` 仓库根目录运行：

```bash
bash scripts/audit-portable-offline.sh
```

期望输出：

```text
PORTABLE_OFFLINE_STATIC_AUDIT=PASS
```

## 发布资产审计

生成 release 产物和 `checksums.txt` 后运行：

```bash
bash scripts/audit-release-assets.sh dist
```

期望输出：

```text
RELEASE_ASSET_MATRIX=PASS
```

## 手工网络 Smoke

在 OS 级网络观察工具、deny proxy 或等效审计环境中运行 portable 打包应用。执行以下步骤：

1. 启动 portable 应用。
2. 打开设置页和关于窗口。
3. 列出内置模板。
4. 执行 Markdown 转换。
5. 打开预览。
6. 导入官方模板 ZIP，确认写入用户数据目录覆盖层。
7. 关闭应用。

运行期间不得出现外部域名或外部 IP 请求。

## 允许的网络

- `localhost`
- `127.0.0.1`
- `::1`

## 禁止的域名

- `presto.c-1o.top`
- `registry.presto.app`
- `api.github.com`
- `github.com`
- `raw.githubusercontent.com`
- `objects.githubusercontent.com`
