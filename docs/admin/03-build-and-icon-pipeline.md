# 打包工具 & 图标管线

> 对应需求 ③（本地打包脚本拉后台配置、跨平台、做得完善）与需求 ②（Xcode 式图标：拖入、展示各尺寸位置、裁剪到最佳尺寸）。

---

## 第一部分：跨平台打包 CLI

### 1. 为什么用 Go CLI 替代 `package.sh`

现有 [package.sh](../../package.sh) 是交互式 Bash，只能在 macOS/Linux 跑，Windows 同事无法使用，且渠道清单靠手工维护 CSV。新方案做一个 **Go CLI（`hybrid-pack`）**，与后端同语言、共享类型：

| 诉求 | Bash 现状 | Go CLI |
| --- | --- | --- |
| 跨平台 Win/macOS | ❌ Windows 不可用 | ✅ `GOOS/GOARCH` 交叉编译三平台 |
| 从后台拉配置 | ❌ 无 | ✅ `pull` 命令 |
| 体积 / 分发 | 需复制脚本 | **每平台 ~5–15MB 单文件、零依赖**，目标机无需装任何运行时 |
| 与后端同语言 | ❌ | ✅ `go.work` 共享 manifest / 域名结构体，定义不漂移 |
| 交互体验 | 有 | 更好（`charmbracelet/huh` 多选表单 + `pterm` 进度/spinner） |

> 也可选 Python（PyInstaller）等，但 Go 在「单文件小体积 + 跨平台交叉编译 + 与后端同语言」三点上最契合，故选定 Go。

### 2. 设计原则：后台是上游，Gradle 不动

CLI 的本质是把后台数据**渲染成现有 Gradle 构建已经认识的输入**，从而**完全不改** `app/build.gradle` 的 `loadChannels` / `productFlavors` 逻辑：

```
后台 (MySQL/对象存储)
   │  hybrid-pack pull
   ▼
渲染产物（与现状格式 100% 兼容）：
   ├── channels/ap.csv  channels/bp.csv  channels/gp.csv      ← 重写
   ├── app/src/channels/<brand>/<flavor>/res/...              ← 解压图标资源
   └── app/src/channels/<brand>/<flavor>/assets/bootstrap.json ← 域名兜底+配置端点
   │  hybrid-pack build
   ▼
./gradlew assemble<Flavor>Release   ← 原样调用，行为不变
```

### 3. 命令设计

```bash
# 首次：登录并保存 token 到 ~/.hybrid-pack/config.json
hybrid-pack login --server https://admin.example.com

# 拉取最新配置 + 资源（重写 CSV / res / bootstrap.json）
hybrid-pack pull                      # 全部品牌
hybrid-pack pull --brand ap           # 仅某品牌

# 打包（交互式，沿用 package.sh 的体验：选大渠道→多选小渠道→测试事件）
hybrid-pack build
# 非交互（CI 用）
hybrid-pack build --brand ap --channels all --test-events
hybrid-pack build --brand ap --channels ap01018,ap01034

# 一条龙：pull → build → 收集产物 → 回传构建记录
hybrid-pack release --brand ap --channels all

# 辅助
hybrid-pack status                    # 显示本地 CSV 与后台的差异（漂移检测）
hybrid-pack doctor                    # 检查 JDK/Android SDK/keystore/网络
```

### 4. 关键实现点

- **拉取**：`GET /api/build/manifest?brand=ap` 一次拿全（渠道 + 域名 + PAL_CODE + 资源 zip 地址），逐个下载 `res.zip` 解压到对应 flavor 目录。
- **CSV 渲染**：保留现有注释头，逐行输出 `flavor|applicationId|palCode|appName`，与 [channels/ap.csv](../../channels/ap.csv) 字节级兼容。
- **bootstrap.json**（写入每个 flavor 的 assets，供 APK 离线兜底）：
  ```json
  { "configUrl": "https://cdn.example.com/app/config",
    "palcode": "1053259232660520961",
    "defaultDomains": ["https://arenaplus.ph", "https://ap-backup.net"] }
  ```
- **调用 Gradle**：用 `os/exec` 跨平台执行；Windows 调 `gradlew.bat`，macOS/Linux 调 `./gradlew`，task 名 `assemble<Cap(flavor)>Release`（复刻 package.sh 的 `cap` 逻辑）。
- **产物收集**：扫描 `app/build/outputs/apk/<flavor>/release/*.apk`，可选重命名（沿用 build.sh 的 `应用名_release_版本.apk`）、上传后台、写 `build_record`。
- **健壮性**：`doctor` 预检 JDK 版本、`ANDROID_HOME`、keystore（`local.properties`）、与后台连通性；`pull` 做漂移检测，提示「本地 CSV 与后台不一致」。
- **签名**：保持现状，密钥仍在本地 `local.properties` 的 keystore，**绝不进后台**（安全红线）。

### 5. 交互示意

```
$ hybrid-pack release
◆  选择大渠道
│  ● ap  ArenaPlus   (18 个渠道)
│  ○ bp  BingoPlus   (20 个渠道)
│  ○ gp  GameZone    (41 个渠道)
◇  从后台同步配置… ✓ 18 渠道，3 个域名更新
◆  选择小渠道  (空格多选 / a 全选)
│  ◼ ap01018  ArenaPlus:USA Basketball Live
│  ◼ ap01034  Arena Plus
│  ◻ ap01035  Arena Plus
◇  开启测试事件? › 否
◇  打包中  assembleAp01018Release  [▰▰▰▰▱] 3/5
✓  完成 5 个 APK → 已上传后台，构建记录 #128
```

---

## 第二部分：Xcode 式图标管线

### 1. 需求拆解
你要的是 Xcode AppIcon 那种体验：
1. **拖一张主图进去**；
2. 自动**展示不同尺寸该放哪**（各 density 的槽位 + 像素尺寸标注）；
3. 可**裁剪到最佳尺寸**（方形裁剪）。

### 2. Android 图标到底要哪些尺寸

当前工程每个渠道的 res 里有：`mipmap-{m,h,xh,xxh,xxxh}dpi/ic_launcher.png`，BP 还多了 `ic_launcher_round.png` 与 `ic_launcher_foreground.png`（自适应图标）。完整目标矩阵：

| 槽位 | 文件 | mdpi | hdpi | xhdpi | xxhdpi | xxxhdpi | 说明 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 方形 | `ic_launcher.png` | 48 | 72 | 96 | 144 | 192 | 传统启动图标 |
| 圆形 | `ic_launcher_round.png` | 48 | 72 | 96 | 144 | 192 | 圆形遮罩 |
| 自适应前景 | `ic_launcher_foreground.png` | 108 | 162 | 216 | 324 | 432 | 内容居中于内 ~66%「安全区」 |
| 自适应配置 | `mipmap-anydpi-v26/ic_launcher.xml` | — | — | — | — | — | 指向前景+背景 |
| 启动图 | `drawable/splash_fullscreen.png` | 单图 | | | | | 全屏 CENTER_CROP，单独上传 |

> 推荐**主图统一传 1024×1024 PNG**（与 App Store 一致，留足下采样空间），服务端用 `imaging`（纯 Go）下采样出上表全部尺寸。这样「人肉切 5~6 张图」变成「传 1 张」。

### 3. 前后端分工

```
前端（React）                          后端（Go + imaging）
─────────────                         ──────────────────────
拖入主图(任意尺寸)                       ① 接收裁剪后的 1024² 主图
  ↓ react-easy-crop 方形裁剪            ② imaging.Resize() 生成方形 5 档
预览九宫格（各槽位+尺寸标注）  ──上传──▶   ③ 圆形遮罩合成 round 5 档
  ↓                                    ④ 安全区内缩放生成 foreground 5 档
单槽位可单独覆盖(高级)                    ⑤ 生成 anydpi-v26/ic_launcher.xml
                                       ⑥ 打 res.zip 存对象存储，返回各槽位预览URL
```

- **客户端裁剪**：`react-easy-crop` 提供方形裁剪框 + 缩放/拖动，所见即所得，导出 1024² 主图再上传。满足你说的「剪切到最佳尺寸」。
- **服务端 fan-out**（`imaging` 伪代码，纯 Go、全静态无 cgo）：
  ```go
  square := map[string]int{"mdpi":48,"hdpi":72,"xhdpi":96,"xxhdpi":144,"xxxhdpi":192}
  for dpi, px := range square {
      sq := imaging.Resize(master, px, px, imaging.Lanczos)
      imaging.Save(sq, fmt.Sprintf("mipmap-%s/ic_launcher.png", dpi))
      round := applyCircleMask(sq, px)                       // image/draw + 圆形 alpha 遮罩
      imaging.Save(round, fmt.Sprintf("mipmap-%s/ic_launcher_round.png", dpi))
  }
  fg := map[string]int{"mdpi":108,"hdpi":162,"xhdpi":216,"xxhdpi":324,"xxxhdpi":432}
  for dpi, px := range fg {
      inner := int(float64(px) * 0.66)                       // 安全区
      icon := imaging.Resize(master, inner, inner, imaging.Lanczos)
      canvas := imaging.New(px, px, color.Transparent)
      canvas = imaging.PasteCenter(canvas, icon)             // 居中留边
      imaging.Save(canvas, fmt.Sprintf("mipmap-%s/ic_launcher_foreground.png", dpi))
  }
  ```

### 4. 「九宫格」UI（对应 Xcode 那种位置展示）

后台新增/编辑渠道时，图标区是一个**槽位网格**，每格标注用途与像素尺寸，主图生成后自动填入预览；任意一格支持**单独拖入**覆盖该档（高级用户精修）：

```
┌── App 图标 ────────────────────────────────────────────────┐
│  [ 拖入主图 1024×1024  或  点击上传 ]   ← 大 dropzone        │
│                                                             │
│  方形 ic_launcher                                           │
│  ┌────┐ ┌────┐ ┌────┐ ┌─────┐ ┌─────┐                       │
│  │ 48 │ │ 72 │ │ 96 │ │ 144 │ │ 192 │  ← 每格可单独覆盖      │
│  │mdpi│ │hdpi│ │xhdpi│ │xxhdpi│ │xxxhdpi│                   │
│  └────┘ └────┘ └────┘ └─────┘ └─────┘                       │
│  圆形 / 自适应前景  同上两行（可折叠）                         │
└─────────────────────────────────────────────────────────────┘
```

> 这一屏的精美版见 [UI 原型](./ui/index.html) 的「新增渠道」抽屉。

### 5. 校验
- 主图建议 ≥512²，否则放大模糊 → 前端拦截 + 提示；
- 非正方形 → 强制进入裁剪；
- 透明通道：方形图标若全透明边会显丑 → 提示补背景；
- 生成后给「真机预览」缩略（圆形/方形/自适应三种桌面形态）。
