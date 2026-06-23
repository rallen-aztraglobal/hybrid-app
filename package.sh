#!/usr/bin/env bash
# ============================================================================
# 交互式打包脚本（兼容 macOS 自带 bash 3.2）
#   1. 选择大渠道（ap / bp / gp）
#   2. 多选该大渠道下的小渠道（空格/逗号分隔编号，或 all 全选）
#   3. 选择是否开启测试事件（首次安装一次性发送全部 AF 事件）
# 然后对选中的每个小渠道执行 assemble<Flavor>Release 打包。
#
# 用法：
#   ./package.sh                # 交互式
#   ./package.sh -b gp -c all   # 非交互：gp 全部小渠道
#   ./package.sh -b ap -c "ap01018,ap01034" -t   # 指定小渠道 + 开测试事件
# ============================================================================
set -euo pipefail
cd "$(dirname "$0")"

BRANDS="ap bp gp"
brand_name() {
  case "$1" in
    ap) echo "ArenaPlus" ;;
    bp) echo "BingoPlus" ;;
    gp) echo "GameZone" ;;
    *)  echo "$1" ;;
  esac
}

brand=""
channels_arg=""
test_events="false"
non_interactive=false

while getopts "b:c:th" opt; do
  case "$opt" in
    b) brand="$OPTARG"; non_interactive=true ;;
    c) channels_arg="$OPTARG"; non_interactive=true ;;
    t) test_events="true" ;;
    h) sed -n '2,22p' "$0"; exit 0 ;;
    *) echo "未知参数"; exit 1 ;;
  esac
done

csv_for() { echo "channels/$1.csv"; }

# 读取某品牌的小渠道名列表（过滤注释/空行）
list_channels() {
  awk -F'|' '!/^[[:space:]]*#/ && NF>=4 {gsub(/^[ \t]+|[ \t]+$/,"",$1); if($1!="") print $1}' "$(csv_for "$1")"
}
# 取某小渠道的应用名
app_name_of() {
  awk -F'|' -v n="$2" '!/^[[:space:]]*#/ && NF>=4 {gsub(/^[ \t]+|[ \t]+$/,"",$1); if($1==n){gsub(/^[ \t]+|[ \t]+$/,"",$4); print $4}}' "$(csv_for "$1")"
}
# 首字母大写（assemble 任务名需要 Flavor 首字母大写）
cap() { echo "$(tr '[:lower:]' '[:upper:]' <<< "${1:0:1}")${1:1}"; }
# 校验大渠道有效
valid_brand() { case " $BRANDS " in *" $1 "*) return 0 ;; *) return 1 ;; esac; }

# ---- 1. 选择大渠道 ----
if [ -z "$brand" ]; then
  echo "==== 选择大渠道 ===="
  i=1
  for b in $BRANDS; do echo "  $i) $b  ($(brand_name "$b"))"; i=$((i+1)); done
  read -rp "输入编号: " bidx
  j=1
  for b in $BRANDS; do [ "$j" = "$bidx" ] && brand="$b"; j=$((j+1)); done
fi
valid_brand "$brand" || { echo "无效大渠道: $brand"; exit 1; }
echo "已选大渠道: $brand ($(brand_name "$brand"))"

# ---- 2. 选择小渠道 ----
ALL=()
while IFS= read -r line; do ALL+=("$line"); done < <(list_channels "$brand")
[ "${#ALL[@]}" -gt 0 ] || { echo "该大渠道无小渠道"; exit 1; }

selected=()
if [ -n "$channels_arg" ]; then
  if [ "$channels_arg" = "all" ]; then
    selected=("${ALL[@]}")
  else
    IFS=', ' read -ra selected <<< "$channels_arg"
  fi
else
  echo ""
  echo "==== 选择小渠道（${#ALL[@]} 个，空格/逗号分隔编号，all=全选）===="
  i=1
  for c in "${ALL[@]}"; do printf "  %2d) %-14s %s\n" "$i" "$c" "$(app_name_of "$brand" "$c")"; i=$((i+1)); done
  read -rp "输入: " picks
  if [ "$picks" = "all" ]; then
    selected=("${ALL[@]}")
  else
    IFS=', ' read -ra idxs <<< "$picks"
    for x in "${idxs[@]}"; do
      [ -n "$x" ] || continue
      # 必须是纯数字且在 1..N 范围内，否则报错退出（避免静默选空导致误打全量）
      case "$x" in
        ''|*[!0-9]*) echo "无效编号: '$x'（需为数字）"; exit 1 ;;
      esac
      if [ "$x" -lt 1 ] || [ "$x" -gt "${#ALL[@]}" ]; then
        echo "编号越界: $x（有效范围 1-${#ALL[@]}）"; exit 1
      fi
      selected+=("${ALL[$((x-1))]}")
    done
  fi
fi
[ "${#selected[@]}" -gt 0 ] || { echo "未选择任何小渠道"; exit 1; }

# 校验所有选中的小渠道名确实属于该大渠道
for c in "${selected[@]}"; do
  found=false
  for a in "${ALL[@]}"; do [ "$c" = "$a" ] && found=true && break; done
  [ "$found" = true ] || { echo "未知小渠道: '$c'（不属于大渠道 $brand）"; exit 1; }
done

# ---- 3. 测试事件开关 ----
if [ "$non_interactive" = false ]; then
  read -rp $'\n是否开启测试事件？(y/N): ' yn
  case "$yn" in [Yy]*) test_events="true" ;; *) test_events="false" ;; esac
fi

# ---- 汇总确认 ----
echo ""
echo "============ 打包计划 ============"
echo "大渠道   : $brand ($(brand_name "$brand"))"
echo "小渠道   : ${selected[*]}"
echo "测试事件 : $test_events"
echo "构建类型 : Release"
echo "=================================="
if [ "$non_interactive" = false ]; then
  read -rp "确认开始打包？(Y/n): " ok
  case "$ok" in [Nn]*) echo "已取消"; exit 0 ;; esac
fi

# ---- 执行打包 ----
tasks=()
for c in "${selected[@]}"; do tasks+=("assemble$(cap "$c")Release"); done

echo ""
echo "执行: ./gradlew ${tasks[*]} -PtestEvents=$test_events"
./gradlew "${tasks[@]}" -PtestEvents="$test_events"

echo ""
echo "============ 产物 APK ============"
for c in "${selected[@]}"; do
  find "app/build/outputs/apk/$c" -name '*.apk' 2>/dev/null | while read -r apk; do echo "  $apk"; done
done
