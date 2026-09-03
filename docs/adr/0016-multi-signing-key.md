# ADR-0016: 多签名 key——按渠道选择，构建机打包后用 apksigner 重签

- **状态**：已采纳（2026-09-01）
- **背景**：2025-09~10 上小米/OPPO 的一批渠道（ap01013~ap01022、gzmkt031）是在老分支 `ap_xiaomi` /
  `gzmkt031` 上用 `gzmkt031-key.jks`（证书 CN=empty-app，SHA-1 `afeaec41…`）签名的；而现在的 Console
  构建机只烧了 `release-key.jks`（CN=bingo，SHA-1 `c52c6e05…`）。应用商店按包名绑定首次上传的证书，
  同包名换证书即判「签名无效」（2026-09-01 OPPO 上传 `com.arenaplus.ap01020` 实测），且已上架不能改。
  仍在 Console 清单里的受影响渠道：ap01018、ap01019、ap01020、ap01021、ap01022、gzmkt031。
- **决策**：
  1. **Console 只存 key 的 ID**：`channel.signing_key VARCHAR(32)`，空 = 默认 key；`GET /api/signing-keys`
     下发固定注册表（id / 名称 / 证书指纹）供前端下拉与校验；manifest 的每个渠道带 `signingKey`。
     密钥材料与口令**仍不进** git / DB / API / 对象存储 / 前端（护栏 #4 不变）。
  2. **Gradle 一行不动**（护栏 #1）：Gradle 仍用默认 key 出包；runner 在 assemble 成功后、投递前，
     对 `signingKey != ""` 的 flavor 用 `apksigner sign`（v1+v2）重签，再 `apksigner verify
     --min-sdk-version 21` 确认 v1/v2 与证书后才投递。
  3. **非默认 key 全部烧进 build-runner 镜像**（与默认 key 同一策略，ADR-0008）：
     `deploy/secrets/store-emptyapp.keystore` → `/opt/hybrid/store-emptyapp.keystore`，并在镜像内写
     `/opt/hybrid/signing-keys.properties`（`<id>.file/.alias/.storePassword/.keyPassword`），路径经
     `HYBRID_PACK_SIGNING_KEYS` 告知 runner。
  4. **fail-closed**：渠道要求的 key 在构建机注册表里不存在 → 整个任务失败并指明缺哪把，绝不把默认
     签名的包当商店包投递。
  5. 存量回填：迁移 `000014` 把上述 6 个 applicationId 置为 `emptyapp`；生产走 AutoMigrate 只加列，
     回填由运维执行同一条 UPDATE。ap01018 / gzmkt031 也出现在过 bingo 分支，若商店实际登记的是 bingo，
     在 Console 把该渠道改回默认即可。
- **理由**：
  - 商店证书不可变是硬约束，只能让构建机迁就历史；重签比「按渠道切 Gradle signingConfig」侵入小、
    不碰 flavor 生成逻辑，也天然覆盖未来更多把 key。
  - 密钥仍只存在于镜像，与既有 keystore 策略一致；Console 侧只是一个可审计的 ID。
  - fail-closed 防止最坏情况：商店拒收还好，若被商店收下则是同包名双证书，用户无法覆盖升级。
- **后果**：
  - 新增一把 key 需要三处同步：server 注册表（`model.SigningKeys`）+ `deploy/Dockerfile.builder`
    （COPY + 注册表段 + build args）+ `deploy/.env.release`。
  - 同一包名在网页渠道（bingo）与商店（empty-app）是两张证书，同一手机不能互相覆盖安装——这是既定历史
    状态，本决策不改变它。
  - 本地手工兜底：`scripts/resign-store-key.sh <apk>`（运行时从 `ap_xiaomi` 分支临时导出 key）。
