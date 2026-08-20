# Store assets — Hexa Color Sort

素材都在 `store/`，全部由本仓库的构建产物生成（模拟器实机截图 / 自绘），可直接上传。

| 文件 | 尺寸 | Play 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（App icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic，必填） |
| `store/screenshot-1-home.png` | 1200×2400 | 手机截图 — 首页 |
| `store/screenshot-2-how-to-play.png` | 1200×2400 | 手机截图 — 玩法说明 |
| `store/screenshot-3-gameplay.png` | 1200×2400 | 手机截图 — 对局中 |
| `store/screenshot-4-selection.png` | 1200×2400 | 手机截图 — 选中方块 |

Play 要求手机截图至少 2 张，这里给了 4 张。

## 为什么截图是 1200×2400 而不是 1080×2400

Play 规定截图**最长边不能超过最短边的 2 倍**。设备原生截图是 1080×2400 = 2.22 倍，
会被拒。这里按行取左右边缘像素向两侧各延展 60px 补到 1200 宽，比例正好 2.0。
游戏背景是纯竖向渐变（`AppTheme.backgroundGradient` 是 topCenter→bottomCenter，
同一行水平方向同色），所以延展出来的边缘与原图完全连续，看不出接缝。

重新出截图时记得做同样的处理，别直接上传 1080×2400。

## 截图是怎么拍的

Android 35 模拟器（Pixel 6 / x86_64，`-gpu host`），装 release APK，
并开了系统 UI demo 模式让状态栏干净（只剩 9:00 与满电池，没有调试图标）：

```bash
adb shell settings put global sysui_demo_allowed 1
adb shell am broadcast -a com.android.systemui.demo -e command enter
adb shell am broadcast -a com.android.systemui.demo -e command clock -e hhmm 0900
adb shell am broadcast -a com.android.systemui.demo -e command network -e wifi show -e level 4 -e mobile hide
adb shell am broadcast -a com.android.systemui.demo -e command notifications -e visible false
adb shell am broadcast -a com.android.systemui.demo -e command battery -e level 100 -e plugged false
```

另外先 `adb shell pm grant com.slatecove.hexasort4173 android.permission.POST_NOTIFICATIONS`，
否则每次冷启动都会弹通知授权框挡住画面（这也顺带验证了 `requestPermission()` 确实
会触发 Android 13+ 的运行时弹窗）。

## 特色图

`store/feature-graphic.png` 是自绘的：左侧四色叠块（与应用图标同一套画法、
用游戏 `kColorPalette` 的原色），右侧标题 + tagline，底色沿用游戏深紫渐变。
24 位 PNG 无 alpha（Play 不接受带透明通道的特色图）。文字与图形都留在中间区域，
避免被各展示位裁切。
