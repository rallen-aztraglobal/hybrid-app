#!/usr/bin/env bash
# 用「商店老 key」（gzmkt031-key.jks，证书 CN=empty-app）重签 Console 打出的渠道包（v1+v2）。
# 【手工兜底】正常路径是 Console 渠道选 signingKey=emptyapp 后由构建机自动重签（ADR-0016）；
# 本脚本用于构建机不可用、或需要对历史产物补签的场合。
#
# 背景：2025-09~10 上小米/OPPO 的一批渠道（ap01013~ap01022、gzmkt031）是在 ap_xiaomi / gzmkt031
# 分支上用 gzmkt031-key.jks 签的；商店按包名绑定首次上传的证书，而 Console 构建机只有
# release-key.jks（CN=bingo），直接上传会被判「签名无效」。这些渠道上商店前需用本脚本重签。
#
# key 与口令不落在工作区：运行时从 git 老分支临时导出到内存盘/临时目录，结束即删。
#
# 用法：scripts/resign-store-key.sh <输入.apk> [输出.apk]
#   输出默认为 <输入名>-emptyapp-key.apk（放在输入同目录）。
set -euo pipefail

IN="${1:?用法: $0 <输入.apk> [输出.apk]}"
OUT="${2:-${IN%.apk}-emptyapp-key.apk}"
KEY_REF="${STORE_KEY_REF:-ap_xiaomi}"           # 存放老 key 的 git 引用（分支/提交）
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$HOME/Library/Android/sdk}}"
BT="$(ls -d "$SDK"/build-tools/* 2>/dev/null | sort -V | tail -1)"
[ -x "$BT/apksigner" ] || { echo "✗ 找不到 apksigner（$SDK/build-tools）"; exit 1; }

TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
git -C "$REPO_ROOT" show "${KEY_REF}:app/gzmkt031-key.jks" > "$TMP/store.jks"
PROPS="$(git -C "$REPO_ROOT" show "${KEY_REF}:local.properties")"
prop() { printf '%s\n' "$PROPS" | grep "^$1=" | cut -d= -f2- | tr -d '\r'; }
export STORE_KS_PASS="$(prop KEYSTORE_PASSWORD)" STORE_KEY_PASS="$(prop KEY_PASSWORD)"
ALIAS="$(prop KEY_ALIAS)"
[ -s "$TMP/store.jks" ] && [ -n "$STORE_KS_PASS" ] || { echo "✗ 从 ${KEY_REF} 取 key/口令失败"; exit 1; }

"$BT/apksigner" sign --ks "$TMP/store.jks" --ks-key-alias "$ALIAS" \
  --ks-pass env:STORE_KS_PASS --key-pass env:STORE_KEY_PASS \
  --v1-signing-enabled true --v2-signing-enabled true \
  --v3-signing-enabled false --v4-signing-enabled false \
  --out "$OUT" "$IN"

# 校验：--min-sdk-version 21 才会真正走 v1 校验（minSdk≥24 时 apksigner 默认跳过 v1）
"$BT/apksigner" verify -v --min-sdk-version 21 --print-certs "$OUT" 2>/dev/null \
  | grep -E "v1 scheme|v2 scheme|certificate DN|certificate SHA-1" | sed 's/^/  /'
"$BT/aapt2" dump badging "$OUT" 2>/dev/null | grep -o -E "package: name='[^']+' versionCode='[^']+' versionName='[^']+'" | sed 's/^/  /'
echo "✓ 已输出: $OUT"
