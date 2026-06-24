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

[ -f "$SSH_KEY" ] || { echo "✗ SSH key 不存在: $SSH_KEY"; exit 1; }
chmod 600 "$SSH_KEY" 2>/dev/null || true
[ -f deploy/secrets/release.keystore ] || { echo "✗ 缺少 deploy/secrets/release.keystore（构建机签名需要）"; exit 1; }

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
rsync -az -e "ssh $KEYOPT" \
  --exclude='.git' --exclude='node_modules' --exclude='.gradle' --exclude='app/build' \
  --exclude='web/dist' --exclude='*.log' \
  --exclude='deploy/certs' --exclude='deploy/.env' --exclude='deploy/.env.release' \
  server cli web app channels gradle deploy build.gradle settings.gradle gradle.properties gradlew gradlew.bat \
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
  -t registry.example.com/hybrid/build-runner:v1.0.0 . >/dev/null && echo '  build-runner ok' && \
  docker compose -f deploy/docker-compose.allinone.yml --env-file deploy/.env build go-api web nginx >/dev/null && echo '  go-api/web/nginx ok'"

log "6/6 启动/更新 + 验证 healthz"
ssh_box "cd '$REMOTE_DIR/deploy' && docker compose -f docker-compose.allinone.yml -f allinone.override.box.yml --env-file .env up -d"
CODE=000
for i in $(seq 1 40); do
  CODE=$(curl -s -m 8 -o /dev/null -w '%{http_code}' "http://${BOX_HOST}/healthz" || echo 000)
  [ "$CODE" = "200" ] && break
  sleep 3
done
if [ "$CODE" = "200" ]; then
  printf '\n\033[1;32m✓ 发版完成 → http://%s/\033[0m  (admin 首登请改密；HTTP 明文，建议尽快上 TLS)\n' "$BOX_HOST"
else
  printf '\n\033[1;31m✗ healthz=%s，未就绪。诊断：\033[0m\n' "$CODE"
  ssh_box "cd '$REMOTE_DIR/deploy' && docker compose -f docker-compose.allinone.yml --env-file .env ps; echo '--- go-api 日志 ---'; docker logs --tail 25 hybrid-go-api-1 2>&1"
  exit 1
fi
