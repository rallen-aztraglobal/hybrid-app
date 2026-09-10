# Store assets — UnitShift

## 现成的文件

全部在 `store/` 下，可以直接传 Play Console：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（Hi-res icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic） |
| `store/screenshot-1.png` | 1200×2400 | 手机截图 —— Length，223 m 换成全部单位 |
| `store/screenshot-2.png` | 1200×2400 | 手机截图 —— Volume，11 个单位同屏 |

| `store/privacy-policy.html` | — | 隐私政策页，**部署用**（不是传 Console 的素材，见下） |

`store/raw/` 是补边之前的原始实机截图，**不要传那些**（原因见下）。

## 重新生成

```bash
cd listings/unitshift
dart run tool/generate_icons.dart          # icon.png / icon_foreground.png / icon_background.png
dart run flutter_launcher_icons            # 渲染成 Android 各档 mipmap 资源
dart run tool/generate_store_assets.dart store/raw/shot-1.png store/raw/shot-2.png
```

两个脚本都是纯几何绘制，不依赖字体，所以在任何机器上跑出来完全一致。

## 三条硬规矩（脚本里已经处理，但改脚本时别破坏）

1. **截图必须补边到 1200 宽。** 实机是 1080×2400（9:20），比 Play 允许的最长边比例
   **2:1** 还要瘦，直接传会被拒。左右各补 60px 的 `#F4F5F7`（与 App 背景同色，
   看不出是补的）。**不要改成裁剪** —— 裁掉的是数值列，那正是要给人看的东西。
2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布是按 4 通道建的
   （超采样需要），落盘前 `convert(numChannels: 3)` 转成 24 位。
3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
   只会打架；而且 `package:image` 只有位图字体，1024 宽下排出来的字又糊又不对齐。

## 图案说明

- **图标**：一对反向的箭头（⇄）。换算类工具的通用符号，不认字也看得懂，
  笔画少、48dp 下不会糊。两支用不同深浅的蓝，**避免看成一个双头箭头**。
- **特色图**：左边一张「单位列表」卡片（第一行是选中的源单位，淡蓝底），
  右边一对反向箭头。画的就是这个 App 的核心交互 —— 输入一次、整列跟着变。
  行里的名字与数字用**灰条代替**：位图字体排出来的字又糊又不对齐，
  灰条反而更像一张排版整齐的表。

两者都直接取 `lib/theme/app_colors.dart` 的色值。

## 重新截图

```bash
adb shell monkey -p com.mossgate.unitshift4610 -c android.intent.category.LAUNCHER 1
# 敲一个有代表性的数、切到想要的分类之后
adb shell screencap -p /sdcard/shot.png
adb pull /sdcard/shot.png store/raw/shot-1.png
```

**截图里的数字不要用 0 或 1。** 整屏都是 `0` 看不出这 App 在干什么；
`1` 又会让一半的行显示成整数，看不出小数精度。挑个 200 上下的数，
各行长短不一，「一次全变」这件事才看得出来。

**两张挑单位数不同的分类。** 一张 10 个单位（Length）、一张 11 个（Volume），
能看出列表会按行数自适应；只放同一类的两张就是重复。

## 隐私政策页怎么上线

`store/privacy-policy.html` 是**可以直接部署的成品**，不是给 Console 传的素材。
Play 要的是一个公网 URL，仓库里的 `PRIVACY_POLICY.md` 不算。

做法与 calcpad / tilefit 一致：**每包一个独立站点**，把这个文件改名成 `index.html`
单独拖上去（Netlify Drop / GitHub Pages / 对象存储都行），根路径就是政策页。

```bash
mkdir -p /tmp/unitshift-site && cp store/privacy-policy.html /tmp/unitshift-site/index.html
# 然后把 /tmp/unitshift-site 整个目录拖到静态托管上
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
curl -s  <你的URL> | grep -o '<title>[^<]*'   # 应为 <title>Privacy Policy — UnitShift
```

> 页里的 `toolsa069@gmail.com` **要先换成真实可达的邮箱再部署** ——
> 政策里留一个假地址，比没有政策更糟。`PRIVACY_POLICY.md` 里同一处也要一起改。

## 还缺的

| 项 | 状态 |
| --- | --- |
| 支持邮箱 | ✅ `toolsa069@gmail.com` —— **与 TickPad / CheckLane 共用**（运营决定） |
| 隐私政策 URL | ✅ `https://thriving-banoffee-396732.netlify.app/` —— 本包专用 Netlify 站点 |
| 平板截图 | 未做。Play 不强制，但没有的话商店页在平板上会显示「专为手机设计」 |
