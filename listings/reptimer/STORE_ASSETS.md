# Store assets — RepTimer

## 现成的文件

全部在 `store/` 下，可以直接传 Play Console：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（Hi-res icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic） |
| `store/screenshot-1.png` | 1200×2400 | 手机截图 —— 方案列表 |
| `store/screenshot-2.png` | 1200×2400 | 手机截图 —— 计时进行中 |
| `store/privacy-policy.html` | — | 隐私政策页，**部署用**（不是传 Console 的素材） |
| `store/adjust-events.csv` | — | Adjust 事件表，传渠道中台 Console 的「事件列表」用 |

`store/raw/` 是补边之前的原始实机截图，**不要传那些**（原因见下）。

## 重新生成

```bash
cd listings/reptimer
dart run tool/generate_icons.dart          # icon.png / icon_foreground.png / icon_background.png
dart run flutter_launcher_icons            # 渲染成 Android 各档 mipmap 资源
dart run tool/generate_store_assets.dart store/raw/shot-1.png store/raw/shot-2.png
```

两个脚本都是纯几何绘制，不依赖字体，所以在任何机器上跑出来完全一致。

## 三条硬规矩（脚本里已经处理，但改脚本时别破坏）

1. **截图必须补边到 1200 宽。** 实机是 1080×2400（9:20），比 Play 允许的最长边比例
   **2:1** 还要瘦，直接传会被拒。左右各补 60px 的 `#11151A`（与 App 背景同色，
   看不出是补的）。**不要改成裁剪。**
2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布按 4 通道建
   （超采样需要），落盘前 `convert(numChannels: 3)` 转成 24 位。
3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
   只会打架；而且 `package:image` 只有位图字体，1024 宽下排出来的字又糊又不对齐。

实测核验（读 PNG 文件头，不是假设）：特色图 1024×500、`colorType=2`（24 位无 alpha）；
两张截图 1200×2400，长边比正好 2.000。

## 图案说明

- **图标**：一圈按阶段分段的环，青绿是「做」、蓝是「休」，交替四组。

> 段间缺口的大小是算出来的：圆头端帽让每段两端各多伸约 10°，共 20°。
> 第一版只留 11° 的缺口，反而被端帽吃掉还重叠 9°，出来是个没有分段的双色环。
- **特色图**：把一次训练摊平成一条时间轴：准备，然后做/休交替。每段宽度按**真实秒数**成比例
（10 准备 / 20 做 / 10 休），所以「做比休长」是看得出来的，不是示意。

> 圆角半径要同时受段高与段宽限制。第一版只按高度取，窄段（准备、休息）的宽度
> 比半径还小，四个角一削就成了月牙。

两者都直接取 `lib/theme/app_colors.dart` 的色值。

## 重新截图

```bash
adb shell monkey -p com.quillford.reptimer7719 -c android.intent.category.LAUNCHER 1
adb shell screencap -p /sdcard/shot.png
adb pull /sdcard/shot.png store/raw/shot-1.png
```

两张截图要截的状态：

1. 两份示例方案、各自的参数与总时长
2. 大数字、阶段色、组数进度

> Git Bash 下 `adb` 会把 `/sdcard/...` 当成 Windows 路径转换掉，
> 要 `export MSYS_NO_PATHCONV=1`；但那样本地目标路径也得写成 Windows 形式。

## 还缺的

| 项 | 状态 |
| --- | --- |
| 支持邮箱 | ✅ `toolsa069@gmail.com` —— **与 UnitShift / TickPad / CheckLane / DaySpan 共用**（运营决定） |
| 隐私政策 URL | ✅ `https://splendid-frangipane-a4c517.netlify.app/` —— 本包专用 Netlify 站点 |
| 平板截图 | 未做。Play 不强制，但没有的话商店页在平板上会显示「专为手机设计」 |
