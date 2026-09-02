# CalcPad

Flutter 计算器，**仅 Android**，作为上架包接入了 AB 面网关。

- 包名 `com.northglade.calcpad5170`
- 商店名 `CalcPad`
- A 面 = 计算器本体（`lib/screens/calculator_screen.dart`），B 面 = 服务端下发的 web

## 从哪份文档开始看

| 想做的事 | 看这份 |
| --- | --- |
| 搞懂网关怎么接的、还差什么没配 | **`README_GATE.md`**（先看这份） |
| 生成 release keystore | `RELEASE_SIGNING.md` |
| 填 Play Console 的问卷 | `PLAY_CONSOLE_FORM_ANSWERS.md` |
| 填 Data safety 表单 | `DATA_SAFETY.md` |
| 上传商店文案 | `PLAY_STORE_LISTING.md` |
| 上传截图 / 特色图 / 托管隐私政策 | `STORE_ASSETS.md` |
| 隐私政策正文 | `PRIVACY_POLICY.md`（可托管版：`store/privacy-policy.html`） |
| 密钥与机密的边界 | `SECURITY_NOTES.md` |
| 内部口径、待定项、教训 | `SUBMISSION_NOTES.md`（**不要粘进 Play Console**） |

## 当前状态

跑得通的：

```bash
flutter pub get
flutter analyze               # No issues found
flutter test                  # 23 passed（20 原有 + 3 网关）
flutter build apk --release   # 47.7MB，已在 Android 35 模拟器上实跑验过 A 面
```

上架前还差的（都在 `README_GATE.md` 里有步骤）：

- [x] Adjust 后台建 App，填 `adjustAppToken` 与 `adjustOpenBLandingToken`
- [x] Firebase 注册本包，放入**裁剪过的** `android/app/google-services.json`
- [ ] 生成 release keystore（现在缺 `key.properties`，release 回退 debug 签名，Play 拒收）
- [ ] 渠道中台建 listing 条目（建好之前判定恒为 A 面，属预期的 fail-closed）
- [ ] 托管隐私政策、定支持邮箱（都**不要**复用其他上架包的）

## 目录

```
lib/
  gate/          AB 面网关（新增，与 colorstack / hexacolorsort 同构）
  push/          FCM token 注册（新增）
  tracking/      AppsFlyer / Adjust 上报（新增）
  logic/         计算器求值逻辑（原有，未改）
  models/        按键配置（原有，未改）
  screens/       计算器主界面（原有，未改）
  theme/         配色（原有，未改）
  widgets/       按键 widget（原有，未改）
store/           Play 上架素材（截图、特色图、图标、可托管的政策页）
icon*.png        应用图标源图，由 flutter_launcher_icons 生成 Android 资源
```
