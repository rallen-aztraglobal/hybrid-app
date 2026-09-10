# CheckLane

可复用的核对表，Flutter，**仅 Android**，包名 `com.thornbury.checklane2894`。
是 `listings/` 下的上架包之一 —— A 面是工具本体，外层套 AB 面网关（见 `README_GATE.md`）。
深色主题。

## 和待办 App 的区别

**待办勾完就删了；核对表要的是「勾完 → 清一遍勾 → 下次照原样再走一遍」。**

所以 `Checklist.reset()` 只清勾选、**保留条目** —— 这是整个 App 最要紧的一个动作，
也是唯一值得单开一个包的理由。围绕它：

- **复制成模板** —— 副本一律未勾选，且每条都换新 id
  （沿用旧 id 两份会在存档里互相串，`checklist_test.dart` 钉了这条）
- **撤销覆盖重置** —— 核对走到一半误按重置，没有撤销就得从头勾一遍
- **空清单不算「已完成」** —— 一条都没有却显示 done 是这类 App 的经典尴尬
- 首次打开给的是「出门前检查」这种**明显要反复用**的示例清单，一眼就懂它不是待办

## 撤销用快照，不用反向操作

反向操作要给每种动作单独写逆操作，删除一条的逆操作还得记住它原来的位置，
漏一个就会出现「撤销之后数据变了样」这种最难查的 bug。
这里数据量很小（几份清单、几十条），一份快照不值几个字节。撤销栈深 30 层。

**空操作不进撤销栈**：本来就没勾还点重置、越界的拖动 —— 都直接返回 false。
否则用户之后再撤销会连撤好几次才回到真正的上一步。

## 存档里有用户输入的文字

`checklane_lists_v1` 一个 JSON 字符串，含 `title` 与每条的 `text` ——
**都是用户自由输入的**。这是本包与另外几个的实质差别（那几个只存整数），
所以 `DATA_SAFETY.md` / `PRIVACY_POLICY.md` / `SECURITY_NOTES.md` 里都单独写明了，
**别照抄「只存整数」的说法**。

只存在本机 SharedPreferences 里，不上报、不随任何请求外发，卸载即删。

> 读那一侧全程兜底：键不存在、不是合法 JSON、顶层不是数组、某条不是对象、
> `done` 不是布尔、文字是空串 —— 各自兜住，能救几条救几条，绝不抛异常。
> 但**空数组是合法的** —— 用户可能真把清单都删了，那时不该硬塞回示例清单，
> 否则删了又冒出来，像 App 不听话。

## 目录

```
lib/
  logic/
    checklist.dart   一份清单与一条待勾项；reset() / duplicateAs() 在这
    book.dart        所有清单的集合 + 快照式撤销
  screens/lists_screen.dart · list_detail_screen.dart
  storage/checklist_store.dart
  theme/app_colors.dart
  gate/              AB 面网关，工具本体对它零感知
tool/
  generate_icons.dart         图标源图
  generate_store_assets.dart  商店素材
```

> 详情页直接操作总览页传下来的**同一个** `ChecklistBook`，改完回调让上层写盘 ——
> 两页各存一份状态再想办法同步，是这类界面最常见的错误来源。

## 常用命令

```bash
cd listings/checklane
flutter pub get
flutter analyze
flutter test                    # 38 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，38 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/checklane2894.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `m4r7coxl34sg` + `OpenBLanding` 事件 `h759bt` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `toolsa069@gmail.com`（与另两个工具包共用）|
| 隐私政策 URL | ✅ `https://sensational-peony-d09cf4.netlify.app/`（本包专用站点）|
| 挂哪个品牌 | ✅ `gp`（GameZone），外开 |

上架前的完整清单见 `PLAY_CONSOLE_FORM_ANSWERS.md` 末尾与 `SUBMISSION_NOTES.md`。

## 相关文档

| 文件 | 内容 |
| --- | --- |
| `README_GATE.md` | AB 面网关怎么接的、怎么验 |
| `PRIVACY_POLICY.md` | 隐私政策正文（需托管成网页） |
| `DATA_SAFETY.md` | Play 的 Data safety 表单逐项答案 |
| `PLAY_CONSOLE_FORM_ANSWERS.md` | 其余各表单答案 + 提交前清单 |
| `PLAY_STORE_LISTING.md` | 商店文案 |
| `STORE_ASSETS.md` | 素材清单与重新生成方法 |
| `RELEASE_SIGNING.md` | keystore 生成与签名核对 |
| `SECURITY_NOTES.md` | 什么绝不进 git、APK 自查 |
| `SUBMISSION_NOTES.md` | **内部**：口径、待定项、取证依据（不要粘进 Console） |
