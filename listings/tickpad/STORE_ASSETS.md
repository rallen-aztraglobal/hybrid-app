# Store assets — TickPad

## 现成的文件

全部在 `store/` 下，可以直接传 Play Console：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（Hi-res icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic） |
| `store/screenshot-1.png` | 1200×2400 | 手机截图 —— 两个计数器，音量键已开（卡片高亮） |
| `store/screenshot-2.png` | 1200×2400 | 手机截图 —— 编辑面板：改名 / 颜色 / 步长 / 目标 |

| `store/privacy-policy.html` | — | 隐私政策页，**部署用**（不是传 Console 的素材，见下） |

`store/raw/` 是补边之前的原始实机截图，**不要传那些**（原因见下）。

## 重新生成

```bash
cd listings/tickpad
dart run tool/generate_icons.dart          # icon.png / icon_foreground.png / icon_background.png
dart run flutter_launcher_icons            # 渲染成 Android 各档 mipmap 资源
dart run tool/generate_store_assets.dart store/raw/shot-1.png store/raw/shot-2.png
```

两个脚本都是纯几何绘制，不依赖字体，所以在任何机器上跑出来完全一致。

## 三条硬规矩（脚本里已经处理，但改脚本时别破坏）

1. **截图必须补边到 1200 宽。** 实机是 1080×2400（9:20），比 Play 允许的最长边比例
   **2:1** 还要瘦，直接传会被拒。左右各补 60px 的 `#101418`（与 App 背景同色，
   看不出是补的）。**不要改成裁剪** —— 裁掉的是加减按钮，画面会失衡。
2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布是按 4 通道建的
   （超采样需要），落盘前 `convert(numChannels: 3)` 转成 24 位。
3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
   只会打架；而且 `package:image` 只有位图字体，1024 宽下排出来的字又糊又不对齐。

## 图案说明

- **图标**：正字计数的第五笔 —— 四竖 + 一斜。全世界通用的手工计数符号，
  不认字也看得懂，笔画又极简。**比画一个「+」强得多** —— 加号太泛，
  什么 App 都能用。四竖用浅色、第五笔用主色，才有「刚记上这一个」的意思。
- **特色图**：四组正字，**最后一组只划到一半**。
  「数到一半」比「数完」更说明这是个计数器 —— 静止的完成态看不出动作，
  半组的那一笔让人一眼想到「按一下就多一画」。没数到的竖笔用暗色留位置，
  让「还没数到」也看得见。

两者都直接取 `lib/theme/app_colors.dart` 的色值。

> 画斜笔时踩到过一个坑：`drawLine` 画斜线的实际宽度比 `thickness` 稍宽，
> 端头补的圆严格取一半会留下一个小缺口。圆半径要提到 0.58 倍。

## 重新截图

```bash
adb shell monkey -p com.larkspur.tickpad7358 -c android.intent.category.LAUNCHER 1
# 建几个计数器、起好名字、数出非零的值之后
adb shell screencap -p /sdcard/shot.png
adb pull /sdcard/shot.png store/raw/shot-1.png
```

**计数器要起真实的名字**（Boxes / Pallets 这类），不要留 `Counter 1` `Counter 2` ——
默认名让人看不出这东西拿来干什么。**值也不要是 0。**

第二张放编辑面板：颜色、步长、目标这些功能不打开面板根本看不见，
而它们正是这个包和一堆简陋计数器的区别。

## 隐私政策页怎么上线

`store/privacy-policy.html` 是**可以直接部署的成品**，不是给 Console 传的素材。
Play 要的是一个公网 URL，仓库里的 `PRIVACY_POLICY.md` 不算。

做法与 calcpad / tilefit 一致：**每包一个独立站点**，把这个文件改名成 `index.html`
单独拖上去（Netlify Drop / GitHub Pages / 对象存储都行），根路径就是政策页。

```bash
mkdir -p /tmp/tickpad-site && cp store/privacy-policy.html /tmp/tickpad-site/index.html
# 然后把 /tmp/tickpad-site 整个目录拖到静态托管上
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
curl -s  <你的URL> | grep -o '<title>[^<]*'   # 应为 <title>Privacy Policy — TickPad
```

> 页里的 `toolsa069@gmail.com` **要先换成真实可达的邮箱再部署** ——
> 政策里留一个假地址，比没有政策更糟。`PRIVACY_POLICY.md` 里同一处也要一起改。

## 还缺的

| 项 | 状态 |
| --- | --- |
| 支持邮箱 | ✅ `toolsa069@gmail.com` —— **与 UnitShift / CheckLane 共用**（运营决定） |
| 隐私政策 URL | ✅ `https://delicate-seahorse-a2b478.netlify.app/` —— 本包专用 Netlify 站点 |
| 平板截图 | 未做。Play 不强制，但没有的话商店页在平板上会显示「专为手机设计」 |
