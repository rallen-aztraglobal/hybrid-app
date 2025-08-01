#!/bin/bash

# 项目目录
PROJECT_DIR=$(cd "$(dirname "$0")"; pwd)
APP_NAME="bingo_plus"  # 可自定义
OUTPUT_DIR="$PROJECT_DIR/app/build/outputs/apk/release"
DESKTOP_DIR=~/Desktop

# 获取版本号
VERSION_NAME=$(grep versionName "$PROJECT_DIR/app/build.gradle" | awk -F\" '{print $2}')

# 获取时间
BUILD_TIME=$(date "+%Y%m%d_%H%M")
echo "🚧 正在清除缓存"
./gradlew clean
# 构建 APK
echo "🚧 正在构建 APK..."
./gradlew assembleRelease

if [ $? -ne 0 ]; then
  echo "❌ 构建失败，请检查错误日志"
  exit 1
fi

# 找到构建出来的 APK
APK_FILE=$(find "$OUTPUT_DIR" -name "*.apk" | head -n 1)

if [ ! -f "$APK_FILE" ]; then
  echo "❌ 未找到生成的 APK 文件"
  exit 1
fi

# 新 APK 名
NEW_APK_NAME="${APP_NAME}_release_${VERSION_NAME}.apk"
DEST_APK_PATH="${DESKTOP_DIR}/${NEW_APK_NAME}"

# 移动并重命名
cp "$APK_FILE" "$DEST_APK_PATH"

if [ $? -eq 0 ]; then
  echo "✅ APK 构建完成：$DEST_APK_PATH"
else
  echo "❌ APK 移动失败"
  exit 1
fi