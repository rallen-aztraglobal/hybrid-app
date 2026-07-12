package render

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/repo"
)

// renderAdjustTokens 渲染 app/adjust-tokens.json（ADR-0013 §3 / docs/admin/08-adjust.md §3、§5）。
//
// 只收录「已在 Console 绑定 Adjust App Token」的渠道，键用派生后的 applicationId（ADR-0009）。
// CLI 不解析事件 CSV——manifest.Channel.AdjustEvents 已由后端从上传的 CSV（token,name,unique）
// 解析成 {name: token}，这里原样透传，不感知 App 内部事件命名（见 08-adjust.md §4.5）。
//
// 返回写入的已绑定渠道数（供调用方在 Result 汇报）。
func renderAdjustTokens(r *repo.Repo, brand string, channels []manifest.Channel, opt Options) (int, error) {
	tokens := make(map[string]manifest.AdjustTokenEntry)
	for _, ch := range channels {
		token := strings.TrimSpace(ch.AdjustAppToken)
		if token == "" {
			// 未绑定 → 不写入。app/build.gradle 门控块只按 adjust-tokens.json 里「存在的键」注入
			// BuildConfig，未收录的 applicationId 落回 defaultConfig 的空兜底，运行时 AdjustBootstrap
			// 探测为空后全程 no-op（不集成、不发事件），与 FCM 的 feature gate 同构。
			continue
		}
		appID := manifest.DeriveApplicationID(brand, ch.Flavor)
		if appID == "" {
			// 派生失败（未知品牌等极端情况）时回退到 manifest 给定值，与 bootstrap.json 的策略一致。
			appID = ch.ApplicationId
		}
		events := ch.AdjustEvents
		if events == nil {
			events = map[string]string{} // 兜底 nil → 空对象，避免序列化成 JSON null
		}
		tokens[appID] = manifest.AdjustTokenEntry{AppToken: token, Events: events}
	}

	dest := r.AppAdjustTokensJSON()

	if len(tokens) == 0 {
		// 该品牌本次没有任何渠道绑定 Adjust：不生成该文件。app/build.gradle 门控块用
		// `file("adjust-tokens.json").exists()` 判断，文件缺失等价于全体 flavor 走
		// defaultConfig 的空兜底（ADJUST_APP_TOKEN=""），与"写一个空 {} 对象"效果相同，
		// 但省一个空文件、且能顺带清理陈旧残留（见下）。
		if opt.DryRun {
			opt.logf("  [dry-run] 该品牌（%s）无渠道绑定 Adjust，%s 将保持缺省/被清理", brand, rel(r, dest))
			return 0, nil
		}
		// 清理工作区可能残留的旧文件：构建机工作区跨任务持久化（ADR-0008），若上一次 pull 落过
		// 绑定、这次后台已解绑（或切换到未配置 Adjust 的品牌），残留文件会让本该休眠的 flavor
		// 继续误集成 Adjust。呼应 google_services.go 对残留 google-services.json 的清理逻辑。
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			opt.logf("  警告: 清理残留 adjust-tokens.json 失败（%v）", err)
		} else {
			opt.logf("  ⏭ 该品牌（%s）无渠道绑定 Adjust，跳过 adjust-tokens.json", brand)
		}
		return 0, nil
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("序列化 adjust-tokens.json 失败: %w", err)
	}
	data = append(data, '\n')

	if opt.DryRun {
		opt.logf("  [dry-run] 将写 %s（%d 个已绑定 Adjust 的渠道）", rel(r, dest), len(tokens))
		return len(tokens), nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, fmt.Errorf("创建 app 目录失败: %w", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return 0, fmt.Errorf("写 %s 失败: %w", dest, err)
	}
	opt.logf("  adjust-tokens.json 已落地 → %s（%d 个已绑定 Adjust 的渠道）", rel(r, dest), len(tokens))
	return len(tokens), nil
}
