#!/usr/bin/env bash
# FCM 一键配置（线上拉版）：从渠道中台后端拉取每个品牌的全部渠道 applicationId，
# 注册成对应 Firebase 项目下的 Android App，再导出该品牌的「合并 google-services.json」。
#
# 数据源 = 线上后端 GET /api/channels（不读本地 channels/*.csv，避免与线上不一致）。
#
# 前置：
#   - 已建好 3 个 Firebase 项目（PROJECTS 映射对上真实 Project ID）
#   - 已装 firebase-tools，且有 FIREBASE_TOKEN（firebase login:ci 拿到）
#   - 后端可达，有一个 viewer+ 账号（拉 /api/channels 需 JWT）
#
# 用法：
#   export FIREBASE_TOKEN='1//0g...'              # firebase login:ci
#   export API_BASE='https://你的线上地址'         # 后端根，不带末尾 /
#   export HYBRID_USER='admin' HYBRID_PASS='...'   # 或直接给 HYBRID_TOKEN
#   deploy/fcm-setup.sh            # 全部品牌
#   deploy/fcm-setup.sh bp         # 单品牌
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/deploy/google-services"
mkdir -p "$OUT"

# 品牌 → Firebase Project ID（按建项目时的真实 ID 改）。
# 用 case 而非关联数组，兼容 macOS 自带 bash 3.2。
project_of() {
  case "$1" in
    ap) echo "hybrid-ap" ;;
    bp) echo "hybrid-bp" ;;
    gp) echo "hybrid-gp" ;;
    *)  echo "" ;;
  esac
}

API_BASE="${API_BASE:?请先 export API_BASE=线上后端地址}"
API_BASE="${API_BASE%/}"

command -v firebase >/dev/null 2>&1 || { echo "❌ 未装 firebase-tools"; exit 1; }
command -v jq >/dev/null 2>&1       || { echo "❌ 需要 jq：brew install jq"; exit 1; }
[ -n "${FIREBASE_TOKEN:-}" ] || { echo "❌ 缺 FIREBASE_TOKEN（firebase login:ci）"; exit 1; }

# 取 JWT：优先用现成 HYBRID_TOKEN，否则用账号密码登录换
TOKEN="${HYBRID_TOKEN:-}"
if [ -z "$TOKEN" ]; then
  : "${HYBRID_USER:?缺 HYBRID_USER 或 HYBRID_TOKEN}" "${HYBRID_PASS:?缺 HYBRID_PASS}"
  echo "🔑 登录 $API_BASE ..."
  TOKEN="$(curl -fsS -X POST "$API_BASE/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$HYBRID_USER\",\"password\":\"$HYBRID_PASS\"}" \
    | jq -r '.data.accessToken')"
  [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ] || { echo "❌ 登录失败"; exit 1; }
fi

BRANDS=(ap bp gp); [ $# -gt 0 ] && BRANDS=("$@")
GRAND=0

for brand in "${BRANDS[@]}"; do
  project="$(project_of "$brand")"
  [ -n "$project" ] || { echo "⚠️  跳过 $brand：未配 Project ID"; continue; }

  echo "==================== 品牌 $brand → 项目 $project ===================="

  # 从线上拉该品牌全部渠道的 applicationId（只取 enabled）
  appids="$(curl -fsS "$API_BASE/api/channels?brand=$brand&pageSize=1000" \
    -H "Authorization: Bearer $TOKEN" \
    | jq -r '.data.items[] | select(.status=="enabled") | .applicationId')"
  count="$(printf '%s\n' "$appids" | grep -c . || true)"
  echo "  线上 enabled 渠道：$count 个"
  [ "$count" -gt 0 ] || { echo "  ⚠️  线上无渠道，跳过"; continue; }
  GRAND=$((GRAND + count))

  # 已注册的包，避免重复 create
  existing="$(FIREBASE_TOKEN="$FIREBASE_TOKEN" firebase apps:list ANDROID --project "$project" 2>/dev/null | grep -oE 'com\.[a-zA-Z0-9._]+' || true)"

  while IFS= read -r appid; do
    [ -n "$appid" ] || continue
    if echo "$existing" | grep -qx "$appid"; then
      echo "  ✓ 已存在 $appid"
    else
      echo "  + 注册 $appid"
      # Firebase 限流：每项目每分钟 60 次 provision 请求（429 RATE_LIMIT_EXCEEDED）。
      # 失败自动退避重试，最多 6 次；命中限流则等久一点。
      ok=0
      for attempt in 1 2 3 4 5 6; do
        if FIREBASE_TOKEN="$FIREBASE_TOKEN" firebase apps:create ANDROID "$appid" \
             --project "$project" --package-name "$appid" >/dev/null 2>&1; then
          echo "    ✅ ok"; ok=1; break
        fi
        echo "    … 第 $attempt 次失败（多半限流），退避 $((attempt*10))s 后重试"
        sleep $((attempt*10))
      done
      [ "$ok" = 1 ] || echo "    ⚠️  最终失败：$appid"
      # 主动限速：每次成功后稍歇，避免触顶 60/min
      sleep 2
    fi
  done <<< "$appids"

  # 导出合并 json：项目内有多个 app 时 sdkconfig 必须带一个 app id；
  # 但导出的 google-services.json 本就含项目内全部 app 的 client[]，传任一即可。
  echo "  ⬇️  导出 $OUT/google-services-$brand.json"
  anyapp="$(FIREBASE_TOKEN="$FIREBASE_TOKEN" firebase apps:list ANDROID --project "$project" 2>/dev/null | grep -oE '1:[0-9]+:android:[a-f0-9]+' | head -1)"
  rm -f "$OUT/google-services-$brand.json"   # --out 拒绝覆盖已存在文件，先删旧的
  FIREBASE_TOKEN="$FIREBASE_TOKEN" firebase apps:sdkconfig ANDROID "$anyapp" \
    --project "$project" --out "$OUT/google-services-$brand.json"
  exported="$(jq '.client | length' "$OUT/google-services-$brand.json" 2>/dev/null || echo '?')"
  echo "  [$brand] json client[]=${exported} (expect == online ${count})"
done

echo
echo "[done] 线上渠道合计 ${GRAND}; 3 份 json 在 $OUT/"
echo "下一步："
echo "  1) 上传后台分发：POST $API_BASE/api/push/google-services（每品牌一份，operator+）"
echo "  2) 各项目设置→服务账号→生成私钥（3 把，机密）→ 后端 FIREBASE_SA_* + FIREBASE_PROJECT_*"
