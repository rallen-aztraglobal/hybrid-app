# Play Store Listing — TileFit

> 本文是 Google Play 商店页面要填的所有文案（标题 / 短描述 / 长描述 / 分类 / 联系方式）。
> 英文部分可直接粘进 Play Console，中文引用块是内部说明，**不要**粘进去。

## App Title (max 30 chars)
TileFit

> 7 字符。备选（若 `TileFit` 在 Play 上重名或被占）：`TileFit — Block Puzzle`（22 字符）。
> 不要直接叫 `Block Puzzle`：过于通用，既搜不到也容易被判为误导性命名。

## Short Description (max 80 chars)
Fit blocks on an 8×8 grid and clear lines. No ads, no sign-in, plays offline.

> 78 字符（含空格，`×` 按 1 字符计）。改动后请重新数，Play 会硬截断。

## Full Description (max 4000 chars)
TileFit is a quiet block puzzle. Eight by eight squares, three pieces at a time,
and one decision to make: where does this one go?

How it works:
- You get three shapes per round. Drag each one onto the board.
- Fill a whole row or a whole column and it clears.
- Place all three and three more arrive.
- When none of the pieces left can fit anywhere, the round is over.

Scoring:
- Every square you place is worth a point.
- Clearing lines pays extra, and clearing several at once pays much more, so
  setting up a double or a triple is worth the wait.
- Your best score is kept on your device.

Things worth knowing:
- Shapes do not rotate. What you are dealt is what you place, so the board stays
  readable and the choice stays honest.
- Pieces that no longer fit anywhere are dimmed, so you can see the end coming
  and play around it.
- No timer. Think for as long as you like.
- Portrait layout, sized for one hand.
- Dark, low-glare colours for playing at night.

No ads. No in-app purchases. No account, no sign-in, no leaderboard, no friend
list. It opens straight into the board and works with no connection at all.

## Category
Games › Puzzle

> Play Console 里选 Application type = Game，Category = Puzzle。

## Content rating
按内容分级问卷如实作答：无暴力、无性内容、无粗俗语言、无赌博、无用户间交流、
无用户生成内容。预期落在 Everyone / PEGI 3。

## Tags / keywords
block puzzle, grid puzzle, offline puzzle, tile puzzle, brain teaser,
casual puzzle, no ads puzzle

## Contact
- Support email: `tilefit@outlook.com`（本包专用）
- Privacy policy URL: `https://moonlit-macaron-a9bd2e.netlify.app/`

> 两项都已就位，2026-09-03 实测：根路径 HTTP 200、标题 `Privacy Policy — TileFit`、
> 生效日期 1 September 2026、页内邮箱为上面这个、无占位符残留。
> 内容源是 `store/privacy-policy.html`，部署时改名为 `index.html` 上传，所以根路径
> 直接就是政策页，不需要带路径。
>
> 与 calcpad 是**两个互相独立的 Netlify 站**（calcpad 在
> `eloquent-palmier-a005c9`），URL 与邮箱都不共用 —— 符合下面这条。
> 改政策正文后要重新拖一次 Netlify Drop，仓库里改了不等于线上改了。

> 这两项是 Play 必填。`store/privacy-policy.html` 是可直接部署的隐私政策页面，
> 挂到任意静态托管（Netlify / GitHub Pages / Cloudflare Pages / Vercel）拿到 URL 即可。
>
> **两项都不要复用其余上架包的 URL 或邮箱。** 政策 URL 与支持邮箱都会公开显示在商店页面上，
> 多个包指向同一个 URL / 同一个邮箱，等于在 Play 侧把它们公开关联起来，与本包
> 单独取厂商命名空间（`emberlane`）的初衷直接冲突。各自新建一份。
> 细则见 `STORE_ASSETS.md` 末尾。
