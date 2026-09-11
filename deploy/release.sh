#!/usr/bin/env bash
# ============================================================================
# deploy/release.sh —— 一键发版（all-in-one 单机部署）
#
#   同步当前工作区源码 → 服务器上 docker build 镜像 → compose up -d → 验证 healthz。
#   首次：自动装 Docker + 生成 .env（随机机密）+ build-runner 源码挂载 override。
#   之后：增量同步代码、重建镜像、滚动更新；【保留】服务器已有 .env 机密。
#
# 配置：deploy/.env.release（gitignored，从 .env.release.example 拷贝填写）。
# 红线：keystore / 口令 / SSH key 绝不进 git；本脚本不打印口令值。
#
# 用法：bash deploy/release.sh
# ============================================================================
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$REPO_ROOT"

CONF="deploy/.env.release"
[ -f "$CONF" ] || { echo "✗ 缺少 $CONF —— 拷贝 deploy/.env.release.example 填写后重试"; exit 1; }
set -a; . "$CONF"; set +a
: "${BOX_HOST:?在 $CONF 填 BOX_HOST}"
BOX_USER="${BOX_USER:-root}"; REMOTE_DIR="${REMOTE_DIR:-/opt/hybrid}"
: "${SSH_KEY:?在 $CONF 填 SSH_KEY}"
: "${KEYSTORE_PASSWORD:?在 $CONF 填 KEYSTORE_PASSWORD}"
: "${KEY_ALIAS:?}"; : "${KEY_PASSWORD:?}"
# 商店老 key（empty-app，ADR-0016）：已上架渠道必须用它重签，故与默认 key 同为必填。
: "${STORE_EMPTYAPP_KEYSTORE_PASSWORD:?在 $CONF 填 STORE_EMPTYAPP_KEYSTORE_PASSWORD（商店老 key 口令，见 .env.release.example）}"
: "${STORE_EMPTYAPP_KEY_ALIAS:?}"; : "${STORE_EMPTYAPP_KEY_PASSWORD:?}"

[ -f "$SSH_KEY" ] || { echo "✗ SSH key 不存在: $SSH_KEY"; exit 1; }
chmod 600 "$SSH_KEY" 2>/dev/null || true
[ -f deploy/secrets/release.keystore ] || { echo "✗ 缺少 deploy/secrets/release.keystore（构建机签名需要）"; exit 1; }
[ -f deploy/secrets/store-emptyapp.keystore ] || { echo "✗ 缺少 deploy/secrets/store-emptyapp.keystore（商店老 key，ADR-0016；可用 git show ap_xiaomi:app/gzmkt031-key.jks 导出）"; exit 1; }

KEYOPT="-i $SSH_KEY -o StrictHostKeyChecking=accept-new -o ConnectTimeout=20"
ssh_box(){ ssh $KEYOPT -o ServerAliveInterval=30 "${BOX_USER}@${BOX_HOST}" "$@"; }
log(){ printf '\n\033[1;36m== %s ==\033[0m\n' "$*"; }

log "1/6 连通性 + 系统"
ssh_box 'echo "  $(uname -m) / $(. /etc/os-release; echo "$PRETTY_NAME")"'

log "2/6 确保 Docker"
if ssh_box 'command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1'; then
  ssh_box 'echo "  已装: $(docker --version)"'
else
  echo "  服务器无 Docker，安装中（Aliyun docker-ce 源）..."
  ssh_box 'bash -s' <<'INSTALL'
set -e
dnf -y install dnf-plugins-core >/dev/null 2>&1 || true
dnf config-manager --add-repo https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo >/dev/null 2>&1 || true
sed -i 's/$releasever/8/g' /etc/yum.repos.d/docker-ce.repo 2>/dev/null || true
dnf -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin
mkdir -p /etc/docker
[ -f /etc/docker/daemon.json ] || printf '{"registry-mirrors":["https://registry.cn-hangzhou.aliyuncs.com"]}\n' >/etc/docker/daemon.json
systemctl enable --now docker
INSTALL
fi

log "3/6 同步源码 → ${BOX_HOST}:${REMOTE_DIR}（rsync 增量；排除机密/产物/缓存）"
# rsync 需两端都有；AliLinux 最小化默认无 rsync，确保装上。
ssh_box "mkdir -p '$REMOTE_DIR'; command -v rsync >/dev/null 2>&1 || dnf -y install rsync >/dev/null 2>&1 || true"
ssh_box 'command -v rsync >/dev/null 2>&1' || { echo "✗ 服务器装 rsync 失败"; exit 1; }

# channels/*.csv 是 hybrid-pack 从 Console 渲染出来的【产物】，后台才是 source of truth（ADR-0004），
# 本不该从开发机同步过去。但首次部署必须给一份：
#   * go-api 镜像的种子层要 COPY channels/（deploy/Dockerfile.api），目录缺失会导致 docker build 失败；
#   * csvio.WriteFile 不自建父目录，缺 channels/ 时构建机第一次 pull 也写不进去。
# 而【非首次】不能再同步：会把构建机工作区里 pull 渲染好的清单（含 _hw 等商店包）盖回开发机的旧版本，
# 直到下一次 pull 才恢复；这期间任何绕过 runner 的手工 gradlew 都看不到那些 flavor。
# 故按「服务器上是否已有该目录」区分：首次带上，之后跳过。
SYNC_PATHS="server cli web app gradle deploy"
if ssh_box "[ -d '$REMOTE_DIR/channels' ]"; then
  echo "  channels/ 已存在 → 本次不同步（渲染产物以 Console 为准，ADR-0004）"
else
  SYNC_PATHS="$SYNC_PATHS channels"
  echo "  首次部署 → 一并同步 channels/（供 go-api 种子层导入）"
fi

# shellcheck disable=SC2086  # SYNC_PATHS 需按空格分词展开为多个源路径
rsync -az -e "ssh $KEYOPT" \
  --exclude='.git' --exclude='node_modules' --exclude='.gradle' --exclude='app/build' \
  --exclude='web/dist' --exclude='*.log' \
  --exclude='deploy/certs' --exclude='deploy/.env' --exclude='deploy/.env.release' \
  $SYNC_PATHS build.gradle settings.gradle gradle.properties gradlew gradlew.bat \
  "${BOX_USER}@${BOX_HOST}:${REMOTE_DIR}/"
ssh_box "chown -R 1000:1000 '$REMOTE_DIR'"   # build-runner(uid1000) 需写权限

log "4/6 确保 .env + override（首次生成随机机密；之后保留不动）"
ssh_box "REMOTE_DIR='$REMOTE_DIR' DOMAIN_DEFAULT='$BOX_HOST' bash -s" <<'SETUP'
set -e
cd "$REMOTE_DIR/deploy"
if [ ! -f .env ]; then
  echo "  首次：生成 .env（随机机密；DOMAIN=$DOMAIN_DEFAULT，admin 初始口令 HybridInit#2026）"
  R(){ openssl rand -hex "$1"; }
  cat > .env <<EOF
TAG=v1.0.0
HTTP_PORT=80
DOMAIN=$DOMAIN_DEFAULT
MYSQL_ROOT_PASSWORD=$(R 16)
MYSQL_DATABASE=hybrid_admin
MYSQL_USER=hybrid
MYSQL_PASSWORD=$(R 16)
DB_AUTOMIGRATE=true
DB_AUTOSEED=true
STORAGE_KIND=local
STORAGE_PUBLIC_URL=/static
JWT_SECRET=$(R 32)
JWT_ISSUER=hybrid-admin
BOOTSTRAP_ADMIN=admin:HybridInit#2026
RUNNER_TOKEN=$(R 24)
BUILDER_REPO_URL=
BUILDER_REPO_BRANCH=
EOF
  chmod 600 .env
else
  echo "  .env 已存在 → 保留（不重置机密，避免掉登录/构建机失联）"
fi
[ -f allinone.override.box.yml ] || cat > allinone.override.box.yml <<EOF
name: hybrid
services:
  build-runner:
    volumes:
      - $REMOTE_DIR:/workspace/hybrid-app
EOF
SETUP

log "5/6 构建镜像（build-runner 烧 keystore；go-api/web/nginx）"
ssh_box "cd '$REMOTE_DIR' && docker build -f deploy/Dockerfile.builder \
  --build-arg KEYSTORE_PASSWORD='$KEYSTORE_PASSWORD' --build-arg KEY_ALIAS='$KEY_ALIAS' --build-arg KEY_PASSWORD='$KEY_PASSWORD' \
  --build-arg STORE_EMPTYAPP_KEYSTORE_PASSWORD='$STORE_EMPTYAPP_KEYSTORE_PASSWORD' --build-arg STORE_EMPTYAPP_KEY_ALIAS='$STORE_EMPTYAPP_KEY_ALIAS' --build-arg STORE_EMPTYAPP_KEY_PASSWORD='$STORE_EMPTYAPP_KEY_PASSWORD' \
  -t registry.example.com/hybrid/build-runner:v1.0.0 . >/dev/null && echo '  build-runner ok' && \
  docker compose -f deploy/docker-compose.allinone.yml --env-file deploy/.env build go-api web nginx >/dev/null && echo '  go-api/web/nginx ok'"

log "6/6 启动/更新 + 验证 healthz"
# 共存 override（本机可能还跑着别的系统）：这台机上 AdSystem 的发版脚本会生成
# deploy/adsystem-edge.override.yml（内容是 nginx `ports: !reset []`），把宿主 80 让给共享边缘
# 容器 adsystem-edge，由它按 Host 分流回 hybrid-nginx-1:80。本脚本**必须带上**该文件，
# 否则 up 时又会去发布 80、与 edge 撞端口，hybrid-nginx 起不来 → 线上 Console 直接 503。
# 文件由对方脚本维护、不在本仓库；存在才带，不存在时行为与以前完全一致（自己发布 80）。
EXTRA_F=""
if ssh_box "[ -f '$REMOTE_DIR/deploy/adsystem-edge.override.yml' ]"; then
  EXTRA_F="-f adsystem-edge.override.yml"
  echo "  检测到 adsystem-edge.override.yml → 宿主 80 让给共享边缘，hybrid-nginx 不发布端口"
fi
ssh_box "cd '$REMOTE_DIR/deploy' && docker compose -f docker-compose.allinone.yml -f allinone.override.box.yml $EXTRA_F --env-file .env up -d"

# healthz 在【服务器内部】探测，不从本机走公网：
#   1) 让宿主 80 给共享边缘后，hybrid-nginx 不再发布端口，本机根本连不到它；
#   2) 发版机常挂着 VPN/代理（本机 curl 会被代理拦下返回 503，响应头带 Proxy-Connection），
#      那是代理的错误页、与生产无关，却会让发版误报失败。
# 故直接在 nginx 容器里自测 127.0.0.1/healthz——只验证「本次部署的服务是否起来了」，
# 公网可达性由边缘/DNS/CDN 负责，不在本脚本职责内。
CODE=000
for i in $(seq 1 40); do
  CODE=$(ssh_box "docker exec hybrid-nginx-1 sh -c 'wget -q -O /dev/null -T 5 http://127.0.0.1/healthz' >/dev/null 2>&1 && echo 200 || echo 000" 2>/dev/null || echo 000)
  [ "$CODE" = "200" ] && break
  sleep 3
done
# 展示用的访问地址：服务器 .env 的 DOMAIN 可能已从「本机 IP」改成真实域名（多个取第一个）。
HEALTH_HOST=$(ssh_box "grep -E '^DOMAIN=' '$REMOTE_DIR/deploy/.env' | head -1 | cut -d= -f2- | tr -d '\"' | awk '{print \$1}'" 2>/dev/null || true)
[ -n "$HEALTH_HOST" ] || HEALTH_HOST="$BOX_HOST"
if [ "$CODE" = "200" ]; then
  printf '\n\033[1;32m✓ 发版完成 → http://%s/\033[0m  (admin 首登请改密；HTTP 明文，建议尽快上 TLS)\n' "$HEALTH_HOST"
else
  printf '\n\033[1;31m✗ healthz=%s（容器内自测 http://127.0.0.1/healthz），未就绪。诊断：\033[0m\n' "$CODE"
  ssh_box "cd '$REMOTE_DIR/deploy' && docker compose -f docker-compose.allinone.yml --env-file .env ps; echo '--- go-api 日志 ---'; docker logs --tail 25 hybrid-go-api-1 2>&1; echo '--- nginx 状态 ---'; docker inspect hybrid-nginx-1 --format '{{.State.Status}} / {{.State.Error}}' 2>&1"
  exit 1
fi
