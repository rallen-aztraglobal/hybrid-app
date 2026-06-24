package render

import (
	"context"
	"fmt"
	"sort"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/csvio"
	"github.com/hybrid-app/cli/internal/repo"
)

// Drift 描述某品牌「本地 CSV」与「后台 manifest」之间的差异（漂移检测）。
type Drift struct {
	Brand string
	// OnlyRemote 仅后台有的 flavor（本地缺，需 pull）。
	OnlyRemote []string
	// OnlyLocal 仅本地有的 flavor（后台已删/未启用，pull 后会消失）。
	OnlyLocal []string
	// Changed 两边都有但字段不同的 flavor → 字段差异描述。
	Changed map[string][]FieldDiff
	// LocalConflicts 本地 CSV 自身的唯一性冲突（脏数据）。
	LocalConflicts []csvio.Conflict
}

// FieldDiff 是单个字段的本地/后台差异。
type FieldDiff struct {
	Field  string
	Local  string
	Remote string
}

// InSync 返回是否完全一致。
func (d *Drift) InSync() bool {
	return len(d.OnlyRemote) == 0 && len(d.OnlyLocal) == 0 && len(d.Changed) == 0
}

// Status 计算某品牌的漂移。
func Status(ctx context.Context, r *repo.Repo, src api.ManifestSource, brand string) (*Drift, error) {
	m, err := src.Manifest(ctx, brand)
	if err != nil {
		return nil, fmt.Errorf("拉取 %s manifest 失败: %w", brand, err)
	}
	local, err := csvio.ReadFile(r.ChannelsCSV(brand))
	if err != nil {
		return nil, err
	}

	d := &Drift{Brand: brand, Changed: map[string][]FieldDiff{}}
	d.LocalConflicts = csvio.Validate(local.Rows)

	localByFlavor := map[string]csvio.Row{}
	for _, row := range local.Rows {
		localByFlavor[row.Flavor] = row
	}
	// 远端行用派生 applicationId（与 pull 写出的 CSV 一致，ADR-0009），
	// 否则历史 manifest 里 appId 与 flavor 不一致会被误报为字段漂移。
	remoteByFlavor := map[string]csvio.Row{}
	for _, row := range csvio.RowsFromChannelsDerived(brand, m.Channels) {
		remoteByFlavor[row.Flavor] = row
	}

	for f, rrow := range remoteByFlavor {
		lrow, ok := localByFlavor[f]
		if !ok {
			d.OnlyRemote = append(d.OnlyRemote, f)
			continue
		}
		var diffs []FieldDiff
		if lrow.ApplicationId != rrow.ApplicationId {
			diffs = append(diffs, FieldDiff{"applicationId", lrow.ApplicationId, rrow.ApplicationId})
		}
		if lrow.PalCode != rrow.PalCode {
			diffs = append(diffs, FieldDiff{"palCode", lrow.PalCode, rrow.PalCode})
		}
		if lrow.AppName != rrow.AppName {
			diffs = append(diffs, FieldDiff{"appName", lrow.AppName, rrow.AppName})
		}
		if len(diffs) > 0 {
			d.Changed[f] = diffs
		}
	}
	for f := range localByFlavor {
		if _, ok := remoteByFlavor[f]; !ok {
			d.OnlyLocal = append(d.OnlyLocal, f)
		}
	}
	sort.Strings(d.OnlyRemote)
	sort.Strings(d.OnlyLocal)
	return d, nil
}
