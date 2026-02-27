#!/bin/bash
# create-dmg.sh — 创建带窗口布局的 macOS DMG 安装镜像
# 两阶段挂载策略：阶段一 nobrowse 复制文件，阶段二正常挂载设置 Finder 布局
set -euo pipefail

# ─── 参数解析 ───────────────────────────────────────────
usage() {
  echo "Usage: $0 [options] <output.dmg> <source-dir>"
  echo "Options:"
  echo "  --volname NAME        Volume name (default: source dir basename)"
  echo "  --background FILE     Background image for DMG window"
  echo "  --volicon FILE        Volume icon (.icns)"
  echo "  --window-size W H     Window width and height"
  echo "  --icon-size SIZE      Icon size in pixels"
  echo "  --icon NAME X Y       Position an item"
  echo "  --app-drop-link X Y   Add Applications symlink at position"
  echo "  --hide-extension NAME Hide extension for item"
  exit 1
}

VOLNAME=""
BACKGROUND=""
VOLICON=""
WIN_W=500 WIN_H=350
ICON_SIZE=128
TEXT_SIZE=16
ICONS=()       # "name:x:y" entries
APP_LINK=""    # "x:y"
HIDE_EXT=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --volname)      VOLNAME="$2"; shift 2 ;;
    --background)   BACKGROUND="$2"; shift 2 ;;
    --volicon)      VOLICON="$2"; shift 2 ;;
    --window-size)  WIN_W="$2"; WIN_H="$3"; shift 3 ;;
    --icon-size)    ICON_SIZE="$2"; shift 2 ;;
    --icon)         ICONS+=("$2:$3:$4"); shift 4 ;;
    --app-drop-link) APP_LINK="$2:$3"; shift 3 ;;
    --hide-extension) HIDE_EXT+=("$2"); shift 2 ;;
    -*)             echo "Unknown option: $1"; usage ;;
    *)              break ;;
  esac
done

[[ $# -lt 2 ]] && usage
OUTPUT_DMG="$1"
SOURCE_DIR="$2"

[[ ! -d "$SOURCE_DIR" ]] && { echo "Error: source dir '$SOURCE_DIR' not found"; exit 1; }
[[ -z "$VOLNAME" ]] && VOLNAME="$(basename "$SOURCE_DIR")"

# ─── 准备 ──────────────────────────────────────────────
TEMP_DMG="${OUTPUT_DMG%.dmg}.rw.$$.dmg"

# 清理残留
rm -f "$OUTPUT_DMG"
rm -f "${OUTPUT_DMG%.dmg}".rw.*.dmg 2>/dev/null || true

echo "==> Creating temporary DMG..."
for attempt in 1 2 3; do
  if hdiutil create -srcfolder "$SOURCE_DIR" -volname "$VOLNAME" \
    -fs HFS+ -format UDRW -ov "$TEMP_DMG" 2>&1; then
    break
  fi
  if [[ $attempt -eq 3 ]]; then
    echo "Error: hdiutil create failed after 3 attempts"
    exit 1
  fi
  echo "  hdiutil create failed (attempt $attempt/3), retrying in 3s..."
  sleep 3
done

# 解析 hdiutil attach 输出，提取设备名和挂载路径
parse_hdiutil_output() {
  local output="$1"
  DEV_NAME=$(echo "$output" | head -1 | awk '{print $1}')
  MOUNT_DIR=$(echo "$output" | tail -1 | sed 's/.*\t//')
}

# ─── 阶段一：nobrowse 挂载，复制资源文件 ─────────────────
echo "==> Phase 1: copying resources (nobrowse)..."
ATTACH_OUT=$(hdiutil attach -nobrowse -noverify -noautoopen -readwrite "$TEMP_DMG")
parse_hdiutil_output "$ATTACH_OUT"

[[ -z "$MOUNT_DIR" || ! -d "$MOUNT_DIR" ]] && { echo "Error: failed to mount DMG (mount_dir='$MOUNT_DIR')"; exit 1; }

# 禁用 Spotlight 索引
touch "$MOUNT_DIR/.metadata_never_index"

# 背景图
if [[ -n "$BACKGROUND" && -f "$BACKGROUND" ]]; then
  mkdir -p "$MOUNT_DIR/.background"
  cp "$BACKGROUND" "$MOUNT_DIR/.background/bg.png"
fi

# 卷图标
if [[ -n "$VOLICON" && -f "$VOLICON" ]]; then
  cp "$VOLICON" "$MOUNT_DIR/.VolumeIcon.icns"
  SetFile -c icnC "$MOUNT_DIR/.VolumeIcon.icns" 2>/dev/null || true
  SetFile -a C "$MOUNT_DIR" 2>/dev/null || true
fi

# Applications 快捷方式
if [[ -n "$APP_LINK" ]]; then
  ln -sf /Applications "$MOUNT_DIR/Applications"
fi

# 隐藏扩展名
for name in "${HIDE_EXT[@]}"; do
  [[ -e "$MOUNT_DIR/$name" ]] && SetFile -a E "$MOUNT_DIR/$name" 2>/dev/null || true
done

# 阶段一卸载（nobrowse 下必定成功）
hdiutil detach "$DEV_NAME" -quiet 2>/dev/null || hdiutil detach "$DEV_NAME"
echo "==> Phase 1 complete."

# ─── 阶段二：正常挂载，AppleScript 设置窗口布局 ──────────
echo "==> Phase 2: setting Finder layout..."
ATTACH_OUT=$(hdiutil attach -noverify -noautoopen -readwrite "$TEMP_DMG")
parse_hdiutil_output "$ATTACH_OUT"

[[ -z "$MOUNT_DIR" || ! -d "$MOUNT_DIR" ]] && { echo "Error: failed to mount DMG for phase 2"; exit 1; }

# 从实际挂载路径提取卷名（可能带后缀如 "Presto 2"）
ACTUAL_VOLNAME=$(basename "$MOUNT_DIR")
echo "  Mounted at: $MOUNT_DIR (volume: $ACTUAL_VOLNAME)"

# 构建 AppleScript 图标定位语句
POSITION_CLAUSES=""
for entry in "${ICONS[@]}"; do
  IFS=: read -r name x y <<< "$entry"
  POSITION_CLAUSES+="set position of item \"$name\" of container window to {$x, $y}"$'\n'
done
if [[ -n "$APP_LINK" ]]; then
  IFS=: read -r lx ly <<< "$APP_LINK"
  POSITION_CLAUSES+="set position of item \"Applications\" of container window to {$lx, $ly}"$'\n'
fi

# 背景图子句
BG_CLAUSE=""
if [[ -n "$BACKGROUND" ]]; then
  BG_CLAUSE='set background picture of theViewOptions to file ".background:bg.png"'
fi

# 等待 Finder 识别新挂载的卷
sleep 2

/usr/bin/osascript <<EOF
tell application "Finder"
  tell disk "$ACTUAL_VOLNAME"
    open
    delay 1

    tell container window
      set current view to icon view
      set toolbar visible to false
      set statusbar visible to false
      set the bounds to {100, 100, $((100 + WIN_W)), $((100 + WIN_H))}
    end tell

    set theViewOptions to the icon view options of container window
    tell theViewOptions
      set arrangement to not arranged
      set icon size to $ICON_SIZE
      set text size to $TEXT_SIZE
    end tell
    $BG_CLAUSE

    $POSITION_CLAUSES

    -- 强制写入 .DS_Store：close → reopen → wait
    close
    open
    delay 2

    tell container window
      set statusbar visible to false
      set the bounds to {100, 100, $((100 + WIN_W - 10)), $((100 + WIN_H - 10))}
    end tell
  end tell

  delay 1

  tell disk "$ACTUAL_VOLNAME"
    tell container window
      set statusbar visible to false
      set the bounds to {100, 100, $((100 + WIN_W)), $((100 + WIN_H))}
    end tell
    close
  end tell
end tell
EOF

echo "==> AppleScript layout done."
sleep 1

# ─── 分层卸载 ──────────────────────────────────────────
safe_detach() {
  local dev="$1"
  local max_retries=3

  # 先尝试正常 detach
  if hdiutil detach "$dev" 2>/dev/null; then
    return 0
  fi

  # 重试循环
  for i in $(seq 1 $max_retries); do
    echo "  Detach failed (attempt $i/$max_retries), diagnosing..."
    lsof +D "$MOUNT_DIR" 2>/dev/null | head -20 || true
    sleep 2
    if hdiutil detach "$dev" 2>/dev/null; then
      return 0
    fi
  done

  # 强制卸载
  echo "  WARNING: force unmounting..."
  diskutil unmount force "$MOUNT_DIR" 2>/dev/null || \
    hdiutil detach "$dev" -force 2>/dev/null || true
}

safe_detach "$DEV_NAME"
echo "==> Phase 2 complete."

# ─── 转换为压缩 DMG ───────────────────────────────────
echo "==> Converting to compressed DMG..."
hdiutil convert "$TEMP_DMG" -format UDZO -o "$OUTPUT_DMG"
rm -f "$TEMP_DMG"
echo "==> $OUTPUT_DMG"
