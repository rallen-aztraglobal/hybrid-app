---
name: release
description: 发版/部署「渠道中台」到服务器（all-in-one 单机：mysql+go-api+build-runner+web+nginx）。同步源码→服务器 docker build→compose up -d→验证 healthz。当用户说"发版""部署""上线""上线新版本""更新服务器""发到线上"时使用。
---

# 发版 · 渠道中台 all-in-one 单机部署

一键把当前工作区代码发布到目标服务器。底层是幂等脚本 `deploy/release.sh`：
rsync 源码 → 服务器上构建镜像 → `compose up -d` → 轮询 healthz。
首次自动装 Docker + 生成 `.env`（随机机密）；之后增量更新、**保留**服务器机密。

## 红线（务必遵守）
- keystore / 签名口令 / SSH 私钥 **绝不**提交 git、**绝不**打印到对话。
- 服务器 `.env` 首次生成后**不要重置**（会换 JWT/RUNNER_TOKEN → 掉登录、构建机失联）。脚本已做幂等保护，不要绕过。

## 步骤
1. **预检**：确认 `deploy/.env.release`（gitignored，含 BOX_HOST/SSH_KEY/签名口令）与 `deploy/secrets/release.keystore` 都在。
   - 缺 `.env.release` → 引导用户 `cp deploy/.env.release.example deploy/.env.release` 并填写。
   - 缺 keystore → 让用户把签名 keystore 放到 `deploy/secrets/release.keystore`。
2. **执行**：用 Bash 的 `run_in_background` 跑 `bash deploy/release.sh`（构建数分钟）。不要在前台 sleep 等待。
3. **收尾**：任务结束读输出——
   - 出现 `✓ 发版完成 → http://<IP>/` 即成功，把地址回报用户。
   - 出现 `✗ healthz=...` → 把脚本贴出的 `compose ps` + go-api 日志拿来定位（常见：端口被占、磁盘满、镜像构建失败、安全组没放行）。
4. 若用户本次改了 `cli/` 或 `deploy/Dockerfile.builder`，提示 build-runner 镜像会重建（Android SDK 走层缓存，仍需数分钟）。

## 每次发版后视情况提醒
- 当前对外是 **HTTP/IP**：admin 口令明文传输；建议尽快上域名 + TLS（见 `docs/admin/05` 文末「升级 TLS」），或安全组把 80 端口限到办公 IP。
- 发版**不影响**已入队/在跑的打包任务；构建机随容器重启会自动重连后端继续轮询。
- 改了 Android `app/` 代码后，发版只是更新了「打包用的源码」；要产出新 APK 仍需在 Console「打包中心」触发构建（或调 `POST /api/build/jobs`）。
