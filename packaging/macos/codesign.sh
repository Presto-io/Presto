#!/bin/bash
# codesign.sh — 对 .app 包进行由内而外的代码签名
# 用法: codesign.sh <app-path> [entitlements-path]
#
# 环境变量:
#   CODESIGN_IDENTITY  — 签名身份（如 "Developer ID Application: Name (TEAMID)"）
#                        未设置时回退到 ad-hoc 签名（-s -）

set -euo pipefail

APP_PATH="${1:?用法: codesign.sh <app-path> [entitlements-path]}"
ENTITLEMENTS="${2:-$(dirname "$0")/entitlements.plist}"
IDENTITY="${CODESIGN_IDENTITY:-}"

if [ -z "$IDENTITY" ]; then
    echo "==> CODESIGN_IDENTITY 未设置，使用 ad-hoc 签名"
    codesign --force --deep -s - "$APP_PATH"
    exit 0
fi

echo "==> 使用证书签名: $IDENTITY"

# 1. 签名嵌入的 Typst 二进制
TYPST_BIN="$APP_PATH/Contents/Resources/typst"
if [ -f "$TYPST_BIN" ]; then
    echo "  -> 签名 typst 二进制..."
    codesign --force --options runtime \
        --sign "$IDENTITY" \
        "$TYPST_BIN"
fi

# 2. 签名主程序
APP_NAME=$(defaults read "$APP_PATH/Contents/Info" CFBundleExecutable)
MAIN_BIN="$APP_PATH/Contents/MacOS/$APP_NAME"
if [ -f "$MAIN_BIN" ]; then
    echo "  -> 签名主程序 $APP_NAME..."
    codesign --force --options runtime \
        --entitlements "$ENTITLEMENTS" \
        --sign "$IDENTITY" \
        "$MAIN_BIN"
fi

# 3. 签名整个 .app 包
echo "  -> 签名 .app 包..."
codesign --force --options runtime \
    --entitlements "$ENTITLEMENTS" \
    --sign "$IDENTITY" \
    "$APP_PATH"

# 4. 验证签名
echo "  -> 验证签名..."
codesign --verify --deep --strict "$APP_PATH"
echo "==> 签名完成"
