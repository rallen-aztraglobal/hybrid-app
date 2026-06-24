# APK 域名容灾机制（核心）

> 对应需求 ④：APK 配置「1 个主域名 + 最多 3 个备用域名」，启动先加载主域名，不通则换备用；全挂则显示「网络异常 + 刷新」。
> **最高优先级约束**：只有「域名本身出问题」才换域名；「本机网络问题」绝不乱换 —— 这是本文的设计灵魂。

---

## 1. 为什么「乱换」是个真问题

朴素实现「主域名失败就试下一个」会在**本机断网/弱网**时表现得很糟：

- 用户在地铁里没信号 → 主域名超时 → 立刻换备用 1 → 也超时 → 换备用 2 → … → 把 4 个域名全试一遍、全失败 → 显示「服务不可用」。
- 真实原因是**用户没网**，却把 4 个好域名挨个「判了死刑」，既浪费十几秒，又给出误导文案，甚至可能因此误把正常域名标记为故障。

**核心洞察**：「我的域名连不上」有两种本质不同的原因，必须先分清，再决定要不要换域名：

| 真因 | 现象 | 正确动作 |
| --- | --- | --- |
| **A. 本机网络问题**（没网/弱网/连了 WiFi 但无外网/门户认证页） | 任何域名都连不上，连 Google 都连不上 | **不换域名**，显示「网络异常·检查网络」+ 刷新 |
| **B. 域名问题**（域名被封/DNS 污染/被劫持/服务挂了） | 我的域名连不上，但**中立的公网端点能连上** | **换备用域名**；全换完仍不行才显示「服务不可用」 |

**判别 A / B 的关键工具 = 中立连通性探针**：去请求一个与业务无关、极高可用、不会被针对性封锁的端点（如 Google/Cloudflare 的 `generate_204`）。
- 它通 → 设备有真实公网 → 我域名还不通 = **B（域名问题）** → 该换。
- 它也不通 → 设备根本上不了网 = **A（本机问题）** → 别换，提示用户。

---

## 2. 完整状态机

```
                      ┌─────────────────────────────┐
   启动 / 点刷新  ───▶ │ STEP 0  本机有「已验证」网络? │
                      └──────────────┬──────────────┘
                            否 │             │ 是
                ┌──────────────▼──┐          │
                │  网络异常页(A)    │          ▼
                │ 「检查网络」+刷新 │   ┌────────────────────────────────┐
                └─────────────────┘   │ STEP 1  取候选域名清单           │
                       ▲              │  实时拉取→缓存→兜底(成功即更新)  │
            网络恢复广播自动重试        │  + 上次可用域名提到队首           │
                                      └───────────────┬────────────────┘
                                                      ▼
                                      ┌────────────────────────────────┐
                                      │ STEP 2  按序探测+校验「是我们站点」│◀─┐
                                      │  probe(d, 2.5s) 并发/顺序        │  │ 下一个
                                      └───────┬───────────────┬─────────┘  │
                                        命中  │               │ 失败        │
                                              ▼               └────────────┘
                                      ┌───────────────┐   清单耗尽 │
                                      │ 加载该域名 WebView│           ▼
                                      │ 记为 lastGood   │   ┌────────────────────────────┐
                                      └───────┬────────┘   │ STEP 3  中立连通性探针       │
                                  运行中主框架 │             │  gstatic/cloudflare 204 ?   │
                                  报错(域名中途挂)            └──────┬──────────────┬───────┘
                                              └──────────▶ 防抖重走  通 │(B)        │ 不通(A)
                                                STEP 1            ▼               ▼
                                                        ┌──────────────┐ ┌──────────────┐
                                                        │ 服务不可用页(B)│ │ 网络异常页(A)  │
                                                        │「稍后重试」+刷新│ │「检查网络」+刷新│
                                                        │ + 后台上报告警 │ └──────────────┘
                                                        └──────────────┘
```

### STEP 0 — 本机网络闸门（最先、最便宜）
用 `ConnectivityManager` 判断是否存在「**已验证可上网**」的网络（`NET_CAPABILITY_INTERNET` + `NET_CAPABILITY_VALIDATED`）。
- 无 → 直接进**网络异常页(A)**，**完全不碰任何域名**。这一步就挡掉了绝大多数「乱换」的源头（飞行模式、没信号、没连 WiFi）。
- 有 → 进 STEP 1。

> 注意：`VALIDATED` 只表示系统层面认为「这网能上」，不代表「能访问公网/能访问我们」。所以它不是最终裁决，只是第一道便宜的闸门。真正裁决在 STEP 3。

### STEP 1 — 候选域名清单（实时拉取 + 自更新缓存 + 编译期兜底）
APK **每次启动调用一次** `GET /api/app/config?palcode=…`，域名清单 `[主, 备用1..3]` 按以下优先级确定：
1. **实时接口成功** → 用返回的清单，并**立即写入本地缓存**（覆盖旧值）。这样域名在后台随时改、随时生效。
2. **接口失败** → 用**本地缓存**（上一次成功返回的配置，持久化在 `SharedPreferences`/文件）。
3. **从未成功过**（首次安装即无网）→ 用编译期烧录的 `assets/bootstrap.json` 默认清单。

> 即「**兜底数据 = 最近一次成功的配置**」，随每次成功拉取自更新，不会停留在打包时的陈旧快照。
>
> **首屏不被接口拖慢**：拉取设短超时（~800ms–1.5s）。超时未回则先用缓存/兜底清单继续走 STEP 2，同时让请求在后台跑完、回来再更新缓存供下次启动用——启动一次接口，但绝不为它白等。
>
> **再优化**：把缓存里「上次真正加载成功的域名 `lastGood`」提到队首，减少首屏 failover 时延（不改变"主域名优先"语义，仅加速）。

### STEP 2 — 探测 + 校验「确实是我们的站点」
对每个域名做轻量探测（`HEAD`/`GET` 到约定的 `probePath`，超时 ~2.5s）：
- **可达 ≠ 是我们**。ISP 劫持 / 门户认证 / 运营商广告页会返回 `200` 但内容是别的。因此探测必须**校验响应特征**才算「命中」：
  - 约定健康端点 `GET https://<域名>/healthz` 返回固定 JSON，如 `{"ok":true,"brand":"ap","v":1}`；校验 `ok==true` 且 `brand` 匹配；
  - 或校验 TLS 证书的 CN/SAN 是否为预期域名（防 DNS 污染到劫持服务器）；
  - 或校验某个约定响应头。
- 命中 → `WebView.loadUrl("https://<域名>/?palcode=<PAL_CODE>")`，记 `lastGood`，结束。
- 未命中（超时/DNS失败/连接拒绝/TLS错/5xx/校验不符）→ 记录错误类型，试下一个。
- **并发优化**：可同时探测全部域名，取「最先命中且校验通过」者（首屏更快）；全不命中再进 STEP 3。

### STEP 3 — 裁决 A 还是 B（决定文案，绝不再瞎换）
清单全部未命中后，做**一次**中立连通性探针（多个端点取「任一成功即有网」）：
- 候选端点（互为备份，避免单点被封）：
  - `https://www.gstatic.com/generate_204`（期望 204）
  - `https://cp.cloudflare.com/generate_204`（期望 204）
  - `https://captive.apple.com`（期望含 `Success`）
- **任一通过 → 判定 B（域名问题）**：设备能上公网，是我们的域名/服务出问题。已穷尽备用，显示**「服务暂时不可用·稍后重试」**+ 刷新，并**上报后台告警**（这条上报本身也要容错，失败不影响 UI）。
- **全部不通 → 判定 A（本机问题）**：设备根本没有真实公网（假 WiFi/弱网/门户认证）。显示**「网络异常·请检查网络连接」**+ 刷新。**没有冤枉任何域名。**

> 这一步是「不乱换」的最终保险：换域名只发生在 STEP 2（且 STEP 0 已确保有网才会走到这里），而「全失败」的归因由 STEP 3 的中立探针决定文案，不会把本机问题甩锅给域名。

### 运行中容灾（加载成功之后）
域名可能在使用中途被封。监听 `WebViewClient.onReceivedError` / `onReceivedHttpError`：
- **仅对主框架**（`request.isForMainFrame == true`）生效，忽略图片/JS 等子资源错误（否则会被无关错误触发误切）；
- 触发时带**防抖 + 次数上限**（如 30s 内最多重走一次 STEP 1），避免抖动网络下「狂换」；
- 重走 STEP 1 时 STEP 0 仍先闸一道。

---

## 3. 错误类型与动作映射

| 探测结果 | 归类 | STEP 2 动作 | 备注 |
| --- | --- | --- | --- |
| DNS 解析失败 / `UnknownHostException` | 疑似域名问题（也可能没网） | 试下一个 | 最终由 STEP 3 区分是没网还是被污染 |
| 连接超时 / `SocketTimeout` | 疑似域名问题 | 试下一个 | 短超时(2.5s)避免久等 |
| 连接被拒 / `ConnectException` | 域名问题 | 试下一个 | 服务没起 |
| TLS 握手失败 / 证书域名不符 | 域名问题(疑劫持) | 试下一个 | 强信号：被中间人 |
| HTTP 5xx | 域名问题 | 试下一个 | 服务异常 |
| HTTP 200 但校验不符 | 劫持/门户页 | 试下一个 | **关键**：可达但不是我们 |
| HTTP 200 且校验通过 | 命中 | 加载 | ✅ |

---

## 4. Kotlin 实现草案

> 仅核心骨架，体现机制；落地时可用 OkHttp 替代 `HttpURLConnection`、用协程 `async` 做并发探测。与现有 `BrandHost`/`BrandStrategy` 架构兼容。

### 4.1 域名解析器
```kotlin
sealed class ResolveResult {
    data class Loadable(val url: String) : ResolveResult()      // 命中，去加载
    object ServiceDown : ResolveResult()                        // B：域名/服务问题
    object NoNetwork  : ResolveResult()                         // A：本机网络问题
}

class DomainResolver(
    private val context: Context,
    private val palCode: String,
    private val probePath: String = "/healthz",
    private val perDomainTimeoutMs: Int = 2500,
) {
    private val prefs = context.getSharedPreferences("domain_resolver", Context.MODE_PRIVATE)

    suspend fun resolve(): ResolveResult = withContext(Dispatchers.IO) {
        // STEP 0：本机网络闸门
        if (!hasValidatedNetwork()) return@withContext ResolveResult.NoNetwork

        // STEP 1：实时拉取(成功即更新缓存) → 缓存 → 编译期兜底；再把 lastGood 提到队首
        val domains = orderByLastGood(loadDomainList())

        // STEP 2：并发探测，取最先命中
        val hit = probeAllAndPickFirst(domains)
        if (hit != null) {
            prefs.edit { putString("last_good", hit) }
            return@withContext ResolveResult.Loadable("$hit/?palcode=$palCode")
        }

        // STEP 3：中立探针裁决 A / B
        return@withContext if (neutralInternetReachable()) {
            reportAllDomainsDown(domains)          // 上报告警(容错)
            ResolveResult.ServiceDown              // B
        } else {
            ResolveResult.NoNetwork                // A
        }
    }

    /** STEP 0：存在「已验证可上网」的网络 */
    private fun hasValidatedNetwork(): Boolean {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        val net = cm.activeNetwork ?: return false
        val cap = cm.getNetworkCapabilities(net) ?: return false
        return cap.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
               cap.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
    }

    /** STEP 2：探测单个域名，校验「确实是我们的站点」 */
    private fun probeOne(domain: String): Boolean = runCatching {
        val conn = (URL("$domain$probePath").openConnection() as HttpsURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = perDomainTimeoutMs
            readTimeout = perDomainTimeoutMs
            instanceFollowRedirects = false        // 劫持常用 302 跳广告页
        }
        if (conn.responseCode != 200) return false
        // 证书域名校验（防 DNS 污染到劫持服务器）
        val sanOk = conn.serverCertificates
            .firstOrNull()?.let { certMatchesDomain(it, domain) } ?: false
        if (!sanOk) return false
        // 业务特征校验
        val body = conn.inputStream.bufferedReader().use { it.readText() }
        val json = JSONObject(body)
        json.optBoolean("ok", false)               // 约定 {"ok":true,"brand":"ap"}
    }.getOrDefault(false)

    /** STEP 3：中立连通性，多端点取「任一成功」 */
    private fun neutralInternetReachable(): Boolean {
        val checks = listOf(
            "https://www.gstatic.com/generate_204" to 204,
            "https://cp.cloudflare.com/generate_204" to 204,
        )
        return checks.any { (url, expect) ->
            runCatching {
                val c = (URL(url).openConnection() as HttpsURLConnection).apply {
                    connectTimeout = 2000; readTimeout = 2000; requestMethod = "GET"
                }
                c.responseCode == expect
            }.getOrDefault(false)
        }
    }

    /** STEP 1：三级取用域名清单，接口成功则自更新本地缓存（你确认的机制） */
    private suspend fun loadDomainList(): List<String> {
        // ① 实时拉取（短超时，不拖慢首屏）
        fetchRuntimeDomains(timeoutMs = 1200)?.let { fresh ->
            prefs.edit { putString("cached_domains", fresh.joinToString(",")) } // ② 成功即更新兜底
            return fresh
        }
        // ② 本地缓存（上一次成功返回的配置）
        prefs.getString("cached_domains", null)?.takeIf { it.isNotBlank() }
            ?.let { return it.split(",") }
        // ③ 编译期兜底（仅首次安装即无网、从未成功拉取过时）
        return bakedDefaultDomains()
    }

    // fetchRuntimeDomains() / bakedDefaultDomains() / orderByLastGood() /
    // probeAllAndPickFirst() / certMatchesDomain() / reportAllDomainsDown() 略
}
```

### 4.2 接入 WebViewActivity
把 [WebViewActivity.kt:187](../../app/src/main/java/com/hybrid/android/WebViewActivity.kt#L187) 现在这句：
```kotlin
_webView.loadUrl("${domain}/?palcode=${BuildConfig.PAL_CODE}")
```
替换为：
```kotlin
lifecycleScope.launch {
    when (val r = DomainResolver(this@WebViewActivity, BuildConfig.PAL_CODE).resolve()) {
        is ResolveResult.Loadable  -> _webView.loadUrl(r.url)
        ResolveResult.ServiceDown  -> showErrorView(ErrorKind.SERVICE_DOWN)  // 服务不可用
        ResolveResult.NoNetwork    -> showErrorView(ErrorKind.NO_NETWORK)    // 网络异常
    }
}
```
错误页 `showErrorView` 是**原生 View**（非网页，因为网页都加载不出来），含图标 + 文案 + 刷新按钮：
```kotlin
private fun showErrorView(kind: ErrorKind) {
    splashImageView.visibility = View.GONE
    errorView.bind(kind) { retry() }          // retry() 重走 resolve()
    errorView.visibility = View.VISIBLE
    // 监听网络恢复，自动重试
    registerNetworkCallbackOnce { runOnUiThread { retry() } }
}
```
文案：
- `NO_NETWORK`（A）→「网络异常，请检查你的网络连接」
- `SERVICE_DOWN`（B）→「服务暂时不可用，请稍后重试」

> 域名清单与 `probePath` 来自运行时配置；`BuildConfig.PAL_CODE` 不变。整段逻辑也可下沉进 `BrandStrategy`，让不同品牌定制健康端点/中立探针策略。

---

## 5. 时序与超时预算（首屏体验）

| 阶段 | 预算 | 说明 |
| --- | --- | --- |
| STEP 0 网络闸门 | <10ms | 本地系统调用 |
| STEP 1 拉运行时配置 | ~300ms（命中缓存 ~0） | 失败立刻用兜底，不阻塞 |
| STEP 2 并发探测 | ≤2.5s | 并发，取最快命中；`lastGood` 命中通常 <500ms |
| STEP 3 中立探针 | ≤2s | 仅在全失败时才走 |

正常情况（`lastGood` 可用）：**< 0.6s** 进站。最坏情况（全挂 + 要裁决）：~4.5s 出错误页。splash 在此期间持续展示，无白屏。

---

## 6. 这套机制如何逐条满足你的要求

| 你的要求 | 本方案 |
| --- | --- |
| 配 1 主 + 最多 3 备用 | `channel_domain`/`brand_domain` 的 `position 0..3`，后台校验数量 |
| 启动先加载主域名 | STEP 1 清单首位即主域名（`lastGood` 优化不改变「主优先」的语义，仅加速） |
| 不通就加载备用 | STEP 2 按序/并发 failover |
| 全部无法加载 → 网络异常 + 刷新 | STEP 3 后显示原生错误页含刷新；并细分「服务不可用 / 网络异常」两种文案 |
| **域名问题才换，本机网络问题别乱换** | STEP 0 闸门 + STEP 3 中立探针双保险；运行中容灾仅对主框架且带防抖 |
| APK 自动拼接 PAL_CODE | 命中后 `https://<域名>/?palcode=<PAL_CODE>`，与现状 URL 格式一致 |
