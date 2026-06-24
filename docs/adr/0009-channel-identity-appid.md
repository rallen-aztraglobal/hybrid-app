# ADR-0009: 渠道身份与域名解析键——applicationId 派生且唯一；PAL_CODE 不唯一

- **状态**：已采纳(2026-06-24)，更正 ADR-0002 的「按 palcode 解析」

## 背景
ADR-0002 让 APK 用 `PAL_CODE` 作为拉取域名配置的键。但 **PAL_CODE 可能在三个大渠道之间重复**（同一营销 palcode 被不同品牌复用），并非全局唯一，作解析键会歧义。
同时历史 CSV 出现 applicationId 与 flavor 不一致的脏数据（如 `ap01035|com.arenaplus.ap01034`，包名后缀是 ap01034 却挂在 flavor ap01035 上）。

## 决策
1. **applicationId 派生且为唯一标识**：`applicationId = <品牌包前缀>.<flavor>`（ap→`com.arenaplus`、bp→`com.bingoplus`、gp→`com.gamezone`）。品牌存「包前缀」，渠道只录 `flavor`，applicationId **自动生成**（表单自动填充、不手填）。从构造上保证「applicationId 后缀 == flavor」，且全局唯一（因 `(brand, flavor)` 唯一）。
2. **域名解析键 = applicationId**：APK 用 `BuildConfig.APPLICATION_ID` 拉配置 `GET /api/app/config?appId=<applicationId>`，不再用 palcode 作键。
3. **PAL_CODE 不再全局唯一**：删除 `pal_code` 的全局 UNIQUE 约束（允许跨品牌重复）。PAL_CODE 仍编译期烧录、仍按 `/?palcode=<PAL_CODE>` 拼进加载 URL —— **用途不变，仅不再作为身份/解析键**。
4. **后台「运行时配置预览」改为下拉选渠道**（选 applicationId/渠道），不再手输 palcode。

## 后果
- ✅ 身份唯一且无歧义；从根上杜绝 appId/flavor 不一致脏数据（派生即正确）。
- ✅ 域名解析键稳定唯一。
- **数据模型变更**：`brand` 增 `package_prefix`；`channel.application_id` 改为派生（或存储但强制 == 派生值）；删除 `channel.pal_code` 的全局 UNIQUE。
- **唯一性校验**：以 applicationId（= 派生）与 `(brand, flavor)` 为准；移除 pal_code 全局查重。
- **历史脏数据**：`ap01035|com.arenaplus.ap01034`、`gzmarket062|com.gamezone.gzmarket066` 在派生规则下由 flavor 决定 appId（→ `com.arenaplus.ap01035` / `com.gamezone.gzmarket062`）。**需人工确认**这两行是「独立渠道（采用派生值）」还是「误录的重复行（删除）」。
- **代码改动（下一轮）**：后端 config 端点参数 `palcode→appId` + 删 pal_code 唯一约束 + `brand.package_prefix` + appId 派生；APK `DomainResolver` 取 `BuildConfig.APPLICATION_ID` 为键；CLI bootstrap/manifest 同步；前端「新增渠道」appId 自动填充、预览改下拉。

## 备选
- **维持 palcode 作键**：跨品牌可重复 → 歧义，被否（本 ADR 即更正它）。
- **手填 applicationId + 仅校验一致性**：仍可能录错；派生更稳，故选派生。
