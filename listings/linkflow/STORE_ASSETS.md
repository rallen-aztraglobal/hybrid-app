# Store assets — LinkFlow

## 现成的文件

全部在 `store/` 下，可以直接传 Play Console：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（Hi-res icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic） |
| `store/screenshot-1.png` | 1200×2400 | 手机截图 —— Normal 档 5×5 |
| `store/screenshot-2.png` | 1200×2400 | 手机截图 —— Master 档 9×9，12 条线 |

| `store/privacy-policy.html` | — | 隐私政策页，**部署用**（不是传 Console 的素材，见下） |

`store/raw/` 是补边之前的原始实机截图，**不要传那些**（原因见下）。

## 重新生成

```bash
cd listings/linkflow
dart run tool/generate_icons.dart          # icon.png / icon_foreground.png / icon_background.png
dart run flutter_launcher_icons            # 渲染成 Android 各档 mipmap 资源
dart run tool/generate_store_assets.dart store/raw/shot-1.png store/raw/shot-2.png
```

两个脚本都是纯几何绘制，不依赖字体，所以在任何机器上跑出来完全一致。

## 三条硬规矩（脚本里已经处理，但改脚本时别破坏）

1. **截图必须补边到 1200 宽。** 实机是 1080×2400（9:20），比 Play 允许的最长边比例
   **2:1** 还要瘦，直接传会被拒。左右各补 60px 的 `#F2EFE9`（与 App 背景同色，
   看不出是补的）。**不要改成裁剪** —— 裁掉的是棋盘两侧，画面会失衡。
2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布是按 4 通道建的
   （超采样需要），落盘前 `convert(numChannels: 3)` 转成 24 位。
3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
   只会打架；而且 `package:image` 只有位图字体，1024 宽下排出来的字又糊又不对齐。

## 图案说明

- **图标**：一条蓝色折线把两个端点连起来，旁边还有一对**没连上**的红点。
  有那对红点才看得出「这是道题」，而不只是个箭头。
- **特色图**：一块横向棋盘，四条已连通的彩色通路。线画成圆角粗管、端点是更粗的圆点，
  和游戏里完全一样 —— 商店图和实际画面对得上，用户点进来不会有落差。

两者都直接取 `lib/theme/app_colors.dart` 的色值。

## 重新截图

```bash
adb shell monkey -p com.wrenfield.linkflow6142 -c android.intent.category.LAUNCHER 1
# 切到想要的难度档、画出想要的画面后
adb shell screencap -p /sdcard/shot.png
adb pull /sdcard/shot.png store/raw/shot-1.png
```

**两张截图要挑不同难度档。** 只放 5×5 会让人以为这游戏就那么大点；
Master 档 9×9 摆着 24 个端点，一眼就能看出上限在哪。

> 拖线截图的一个坑：`adb shell input swipe` 只在两点之间走直线，画不出直角折线。
> 要截「已画好的线」，得用 `input motionevent DOWN/MOVE/.../UP` 一格一格喂坐标，
> 或者干脆手动在模拟器窗口里拖。

## 隐私政策页怎么上线

`store/privacy-policy.html` 是**可以直接部署的成品**，不是给 Console 传的素材。
Play 要的是一个公网 URL，仓库里的 `PRIVACY_POLICY.md` 不算。

做法与 calcpad / tilefit 一致：**每包一个独立站点**，把这个文件改名成 `index.html`
单独拖上去（Netlify Drop / GitHub Pages / 对象存储都行），根路径就是政策页。

```bash
mkdir -p /tmp/linkflow-site && cp store/privacy-policy.html /tmp/linkflow-site/index.html
# 然后把 /tmp/linkflow-site 整个目录拖到静态托管上
```

三条别踩：

1. **不能和其他上架包共用一个站点或一个 URL。** 政策 URL 会公开显示在商店页，
   六个包指向同一个地址，等于自己把它们关联起来 —— 和各取独立厂商命名空间的初衷相反。
2. **改了仓库里的文件不等于线上改了。** 静态托管是把文件传上去，之后再改
   `store/privacy-policy.html` 必须重新部署一次，否则线上还是旧版。
3. **URL 要能匿名访问**（不登录、不跳转、不是 403），且长期不变 —— 换 URL 要回 Console 改。

上线后自查：

```bash
curl -sI <你的URL> | head -1          # 应为 HTTP/2 200
curl -s  <你的URL> | grep -o '<title>[^<]*'   # 应为 <title>Privacy Policy — LinkFlow
```

> 页里的 `appaztra@outlook.com` **要先换成真实可达的邮箱再部署** ——
> 政策里留一个假地址，比没有政策更糟。`PRIVACY_POLICY.md` 里同一处也要一起改。

## 还缺的

| 项 | 状态 |
| --- | --- |
| 支持邮箱 | ✅ `appaztra@outlook.com` —— **与 NonoPix / MineGrid 共用**（运营决定） |
| 隐私政策 URL | ✅ `https://frolicking-capybara-46647a.netlify.app/` —— 本包专用 Netlify 站点 |
| 平板截图 | 未做。Play 不强制，但没有的话商店页在平板上会显示「专为手机设计」 |
