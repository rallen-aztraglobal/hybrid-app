# 马甲包UserAgent 设置说明

## WebView UserAgent 设置说明

### 适用范围

仅开启「BP 原始事件」模式的 bp 马甲包（App 内 `bpRawMode`）。未开启的包 UA 保持系统默认，不追加后缀。

### 格式

App 会在 WebView 系统默认 UA 的末尾追加一段后缀，中间用一个空格隔开：

```
<系统默认 WebView UA> <当前包名>/<versionName>
```

例如 com.bingoplus.bpom3410 包（1.0.3 版本，真机实测）：

```
Mozilla/5.0 (Linux; Android 16; 2211133C Build/BP2A.250605.031.A3; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/156.0.8078.47 Mobile Safari/537.36 com.bingoplus.bpom3410/1.0.3
```

### 规则

1. **包名在运行时获取**
   - 用的是 `context.packageName`，也就是当前 APK 的 applicationId，照实传，不做映射。

2. **版本号取 `BuildConfig.VERSION_NAME`**
   - 和 APK 的 versionName 一致（打包时在后台填写），H5 可以按版本做兼容判断。

3. **和 URL 参数的关系**
   - 加载 URL 上的 `appSource` 固定为 `mktApp`（H5 白名单要求），**不是**包名，用来识别「在壳内」。
   - 要区分具体是哪个包，只能看 UA 末尾的包名。壳不提供 `package=` URL 参数，也不提供 JS Bridge 的 `getPackageName()`。

4. **防止重复追加**
   - 设置前，会先从当前 UA 末尾去掉已有的 `<包名>/<版本>` 或 `<包名>`，并去掉尾部空格，然后再追加。
   - 所以同一个 WebView 多次设置，后缀也只会出现一次。

5. **只追加，不改动原 UA**
   - 系统 UA 里的 Chrome 内核版本、Android 版本、`wv`、`Mobile` 等字段都保留，不影响 H5 的机型和浏览器识别。

### H5 侧注意

- 本功能上线前打的包 UA 没有后缀，H5 读不到包名时请按「未知马甲包」兼容处理。
- 建议从 UA 末尾按 `/\s(com\.bingoplus\.[\w.]+)\/(\d+\.\d+\.\d+)$/` 取包名与版本；用 `includes('com.bingoplus.xxx')` 判断时注意子串误命中（如 `com.bingoplus.lite` 会命中 `com.bingoplus.lite01`）。
