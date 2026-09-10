# Store assets — CheckLane

## 现成的文件

全部在 `store/` 下，可以直接传 Play Console：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（Hi-res icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic） |
| `store/screenshot-1.png` | 1200×2400 | 手机截图 —— 清单详情，勾了 3/4 |
| `store/screenshot-2.png` | 1200×2400 | 手机截图 —— 总览，卡片带进度条 |

| `store/privacy-policy.html` | — | 隐私政策页，**部署用**（不是传 Console 的素材，见下） |

`store/raw/` 是补边之前的原始实机截图，**不要传那些**（原因见下）。

## 重新生成

```bash
cd listings/checklane
dart run tool/generate_icons.dart          # icon.png / icon_foreground.png / icon_background.png
dart run flutter_launcher_icons            # 渲染成 Android 各档 mipmap 资源
dart run tool/generate_store_assets.dart store/raw/shot-1.png store/raw/shot-2.png
```

两个脚本都是纯几何绘制，不依赖字体，所以在任何机器上跑出来完全一致。

## 三条硬规矩（脚本里已经处理，但改脚本时别破坏）

1. **截图必须补边到 1200 宽。** 实机是 1080×2400（9:20），比 Play 允许的最长边比例
   **2:1** 还要瘦，直接传会被拒。左右各补 60px 的 `#131218`（与 App 背景同色，
   看不出是补的）。**不要改成裁剪** —— 裁掉的是勾选框和条目文字。
2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布是按 4 通道建的
   （超采样需要），落盘前 `convert(numChannels: 3)` 转成 24 位。
3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
   只会打架；而且 `package:image` 只有位图字体，1024 宽下排出来的字又糊又不对齐。

## 图案说明

- **图标**：两行清单 —— 上面一行勾了、下面一行没勾。
  只画一个对勾会跟一堆待办 App 撞脸；**「一勾一空」才说得出这个包的重点**：
  一条条核对，而且勾完可以清掉重来。
  只画两行不画三四行：行数一多，48dp 下每一行就细成一根发丝，全糊在一起。
- **特色图**：一份**走到一半**的核对表 —— 上面三条勾了（压暗 + 删除线），
  下面两条还空着。「走到一半」是有意的：全勾完看不出这是个能反复用的表。
  每行的横条长度不一样，看着才像一份真的清单而不是色卡。

两者都直接取 `lib/theme/app_colors.dart` 的色值。

> 空勾选框用「外圈实心 + 内圈填底色」两层叠出来，**不用 `drawRect` 描边** ——
> 那个库带圆角时描出来的线极细，深底上几乎看不见。

## 重新截图

```bash
adb shell monkey -p com.thornbury.checklane2894 -c android.intent.category.LAUNCHER 1
# 建一份有真实条目的清单、勾掉一部分之后
adb shell screencap -p /sdcard/shot.png
adb pull /sdcard/shot.png store/raw/shot-1.png
```

**一定要截「勾了一部分」的状态**，不要全空也不要全勾。
全空看不出这 App 在干什么；全勾看不出还能继续 —— 而**半勾正是这个包的卖点**
（勾完可以清掉重来）。

清单和条目要用真实文字（Keys / Wallet / Lock the windows 这类），
不要留 `Item 1` `Item 2`。

## 隐私政策页怎么上线

`store/privacy-policy.html` 是**可以直接部署的成品**，不是给 Console 传的素材。
Play 要的是一个公网 URL，仓库里的 `PRIVACY_POLICY.md` 不算。

做法与 calcpad / tilefit 一致：**每包一个独立站点**，把这个文件改名成 `index.html`
单独拖上去（Netlify Drop / GitHub Pages / 对象存储都行），根路径就是政策页。

```bash
mkdir -p /tmp/checklane-site && cp store/privacy-policy.html /tmp/checklane-site/index.html
# 然后把 /tmp/checklane-site 整个目录拖到静态托管上
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
curl -s  <你的URL> | grep -o '<title>[^<]*'   # 应为 <title>Privacy Policy — CheckLane
```

> 页里的 `toolsa069@gmail.com` **要先换成真实可达的邮箱再部署** ——
> 政策里留一个假地址，比没有政策更糟。`PRIVACY_POLICY.md` 里同一处也要一起改。

## 还缺的

| 项 | 状态 |
| --- | --- |
| 支持邮箱 | ✅ `toolsa069@gmail.com` —— **与 UnitShift / TickPad 共用**（运营决定） |
| 隐私政策 URL | ✅ `https://sensational-peony-d09cf4.netlify.app/` —— 本包专用 Netlify 站点 |
| 平板截图 | 未做。Play 不强制，但没有的话商店页在平板上会显示「专为手机设计」 |
