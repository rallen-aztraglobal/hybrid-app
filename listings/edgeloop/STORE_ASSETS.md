# Store assets — EdgeLoop

## 现成的文件

全部在 `store/` 下，可以直接传 Play Console：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（Hi-res icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic） |
| `store/screenshot-1.png` | 1200×2400 | 手机截图 —— 未解的 5×5 盘面（Normal 第 1 关） |
| `store/screenshot-2.png` | 1200×2400 | 手机截图 —— 同一关解出后的样子 |
| `store/privacy-policy.html` | — | 隐私政策页，**部署用**（不是传 Console 的素材） |
| `store/adjust-events.csv` | — | Adjust 事件表，传渠道中台 Console 的「事件列表」用 |

`store/raw/` 是补边之前的原始实机截图，**不要传那些**（原因见下）。

## 重新生成

```bash
cd listings/edgeloop
dart run tool/generate_icons.dart          # icon.png / icon_foreground.png / icon_background.png
dart run flutter_launcher_icons            # 渲染成 Android 各档 mipmap 资源
dart run tool/generate_store_assets.dart store/raw/shot-1.png store/raw/shot-2.png
```

两个脚本都是纯几何绘制，不依赖字体，所以在任何机器上跑出来完全一致。

## 三条硬规矩（脚本里已经处理，但改脚本时别破坏）

1. **截图必须补边到 1200 宽。** 实机是 1080×2400（9:20），比 Play 允许的最长边比例
   **2:1** 还要瘦，直接传会被拒。左右各补 60px 的 `#F4F1EA`（与 App 背景同色，
   看不出是补的）。**不要改成裁剪。**
2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布按 4 通道建
   （超采样需要），落盘前 `convert(numChannels: 3)` 转成 24 位。
3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
   只会打架；而且 `package:image` 只有位图字体，1024 宽下排出来的字又糊又不对齐。

实测核验（读 PNG 文件头，不是假设）：特色图 1024×500、`colorType=2`（24 位无 alpha）；
两张截图 1200×2400，长边比正好 2.000。

## 图案说明

- **图标**：一条**不对称**的闭合回路走在格点之间。

> 不对称是有意的。第一版顶点序列上下左右都对称，画出来正好是一个加号 ——
> 看着像医疗十字，完全不像「玩家画出来的路径」。
- **特色图**：横向铺三块盘面，各画一条已解出的回路。中间那块最大 —— 全等大会显得像占位图。

> 尺寸和间距是算过的：白底板比格点阵每侧外扩 0.36 格。第一版取 0.62/0.78/0.62 +
> 中心 0.17/0.50/0.83，左右两块既盖住中间那块、又被画布边缘切掉一角。

两者都直接取 `lib/theme/app_colors.dart` 的色值。

## 重新截图

```bash
adb shell monkey -p com.bellcroft.edgeloop8451 -c android.intent.category.LAUNCHER 1
adb shell screencap -p /sdcard/shot.png
adb pull /sdcard/shot.png store/raw/shot-1.png
```

两张截图要截的状态：

1. 棋盘、难度档、提示数字
2. 回路闭合变绿、提示数字转灰、出现 Next level

> Git Bash 下 `adb` 会把 `/sdcard/...` 当成 Windows 路径转换掉，
> 要 `export MSYS_NO_PATHCONV=1`；但那样本地目标路径也得写成 Windows 形式。

## 还缺的

| 项 | 状态 |
| --- | --- |
| 支持邮箱 | ✅ `appaztra@outlook.com` —— **与 NonoPix / LinkFlow / MineGrid 共用**（运营决定） |
| 隐私政策 URL | ✅ `https://precious-douhua-53c7dc.netlify.app/` —— 本包专用 Netlify 站点 |
| 平板截图 | 未做。Play 不强制，但没有的话商店页在平板上会显示「专为手机设计」 |
