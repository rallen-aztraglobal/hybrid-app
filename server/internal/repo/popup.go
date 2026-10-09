// Package repo — 马甲包弹窗模块数据访问（ADR-0019 / docs/admin/12-popup.md）。
package repo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hybrid-app/server/internal/model"
)

// ---------- 位置开关 ----------

// ListPopupPositions 返回全部位置开关（按 code 升序）。
func (r *Repo) ListPopupPositions(ctx context.Context) ([]model.PopupPosition, error) {
	var list []model.PopupPosition
	if err := r.db.WithContext(ctx).Order("code asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询弹窗位置开关失败: %w", err)
	}
	return list, nil
}

// SetPopupPosition 更新位置开关；位置不存在返回 ErrNotFound。
func (r *Repo) SetPopupPosition(ctx context.Context, code string, enabled bool, by string) (*model.PopupPosition, error) {
	var p model.PopupPosition
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code = ?", code).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		p.Enabled = enabled
		p.UpdatedBy = by
		return tx.Save(&p).Error
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("更新弹窗位置开关失败: %w", err)
	}
	return &p, nil
}

// ---------- 弹窗 ----------

func orderedCards(d *gorm.DB) *gorm.DB { return d.Order("sort asc, id asc") }

// ListPopups 按位置（SQL 过滤）取弹窗，预加载卡片；其余过滤（品牌/状态/关键字/数据权限）由 service 在内存里做。
func (r *Repo) ListPopups(ctx context.Context, position, keyword string) ([]model.Popup, error) {
	q := r.db.WithContext(ctx).Preload("Cards", orderedCards).Order("id desc")
	if position != "" {
		q = q.Where("position = ?", position)
	}
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	var list []model.Popup
	if err := q.Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询弹窗列表失败: %w", err)
	}
	return list, nil
}

// ListLivePopups 取 App 端候选弹窗：弹窗开关开 ∧ 未删 ∧ （end_at 为空或晚于 now）。预加载卡片。
func (r *Repo) ListLivePopups(ctx context.Context, now time.Time) ([]model.Popup, error) {
	var list []model.Popup
	if err := r.db.WithContext(ctx).Preload("Cards", orderedCards).
		Where("enabled = ? AND (end_at IS NULL OR end_at > ?)", true, now.UTC()).
		Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询生效弹窗失败: %w", err)
	}
	return list, nil
}

// GetPopup 按 id 取弹窗（含卡片）；已软删返回 ErrNotFound。
func (r *Repo) GetPopup(ctx context.Context, id uint64) (*model.Popup, error) {
	var p model.Popup
	err := r.db.WithContext(ctx).Preload("Cards", orderedCards).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询弹窗失败: %w", err)
	}
	return &p, nil
}

// PopupsByIDsUnscoped 含软删地批量取弹窗（统计要显示已删弹窗的名称），预加载卡片（含卡片全量，不按软删过滤）。
func (r *Repo) PopupsByIDsUnscoped(ctx context.Context, ids []uint64) ([]model.Popup, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.Popup
	if err := r.db.WithContext(ctx).Unscoped().Preload("Cards", orderedCards).
		Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询弹窗（含已删）失败: %w", err)
	}
	return list, nil
}

// PopupIDsByPositionUnscoped 含软删地按位置取弹窗 id（统计的 position 过滤）。
func (r *Repo) PopupIDsByPositionUnscoped(ctx context.Context, position string) ([]uint64, error) {
	var ids []uint64
	if err := r.db.WithContext(ctx).Unscoped().Model(&model.Popup{}).
		Where("position = ?", position).Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("按位置查询弹窗 id 失败: %w", err)
	}
	return ids, nil
}

// SavePopup 在一个事务内保存弹窗主体与卡片。p.ID==0 为新建，否则全量更新。
// buildCards 接收库里现有卡片（新建时为空），返回要 upsert 的卡片（ID!=0 原地更新、ID==0 新建）
// 与要删除的卡片 id；PopupID 由这里统一回填。
func (r *Repo) SavePopup(ctx context.Context, p *model.Popup,
	buildCards func(existing []model.PopupCard) (upsert []model.PopupCard, deleteIDs []uint64)) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if p.ID == 0 {
			if err := tx.Omit("Cards").Create(p).Error; err != nil {
				return err
			}
		} else {
			// 不用 Save：并发软删后 Save 会把 deleted_at 写回 NULL 复活弹窗。
			// Updates 自带 deleted_at IS NULL 条件；created_at / deleted_at 不参与更新。
			res := tx.Model(p).Omit("Cards", "created_at", "deleted_at").Select("*").Updates(p)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				// MySQL 只统计「实际变化」的行，0 不一定是不存在，再确认一次。
				var n int64
				if err := tx.Model(&model.Popup{}).Where("id = ?", p.ID).Count(&n).Error; err != nil {
					return err
				}
				if n == 0 {
					return ErrNotFound
				}
			}
		}
		var existing []model.PopupCard
		if err := tx.Where("popup_id = ?", p.ID).Order("sort asc, id asc").Find(&existing).Error; err != nil {
			return err
		}
		upsert, del := buildCards(existing)
		if len(del) > 0 {
			if err := tx.Where("popup_id = ? AND id IN ?", p.ID, del).Delete(&model.PopupCard{}).Error; err != nil {
				return err
			}
		}
		for i := range upsert {
			upsert[i].PopupID = p.ID
			if upsert[i].ID == 0 {
				if err := tx.Create(&upsert[i]).Error; err != nil {
					return err
				}
			} else if err := tx.Save(&upsert[i]).Error; err != nil {
				return err
			}
		}
		p.Cards = upsert
		return nil
	})
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("保存弹窗失败: %w", err)
	}
	return nil
}

// SetPopupEnabled 单独改弹窗开关（不动其它列）。
func (r *Repo) SetPopupEnabled(ctx context.Context, id uint64, enabled bool) error {
	res := r.db.WithContext(ctx).Model(&model.Popup{}).Where("id = ?", id).Update("enabled", enabled)
	if res.Error != nil {
		return fmt.Errorf("更新弹窗开关失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeletePopup 软删弹窗（卡片保留，统计仍可显示图片）。
func (r *Repo) DeletePopup(ctx context.Context, id uint64) error {
	res := r.db.WithContext(ctx).Delete(&model.Popup{}, id)
	if res.Error != nil {
		return fmt.Errorf("删除弹窗失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- 埋点 ----------

// PopupStatKey 是日聚合的唯一键（application_id 在批次级固定）。
type PopupStatKey struct {
	StatDate   string
	PopupID    uint64
	CardID     uint64
	CardIndex  int
	AppVersion string
	Event      string
	Dim        string
}

// ApplyPopupStats 在同一事务内：先插 popup_event_batch（重复 batch_id → 返回 dup=true、不累加），
// 再把内存已合并好的计数 upsert 累加进 popup_stat_daily（count = count + ?）。
func (r *Repo) ApplyPopupStats(ctx context.Context, batchID, applicationID string, counts map[PopupStatKey]int64) (dup bool, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		b := model.PopupEventBatch{BatchID: batchID}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&b)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			dup = true
			return nil
		}
		rows := make([]model.PopupStatDaily, 0, len(counts))
		// 逐行 upsert 的加锁顺序必须确定（按唯一键排序），否则 MySQL 并发批次互相交叉加锁会死锁。
		keys := make([]PopupStatKey, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return lessPopupStatKey(keys[i], keys[j]) })
		for _, k := range keys {
			n := counts[k]
			rows = append(rows, model.PopupStatDaily{
				StatDate: k.StatDate, PopupID: k.PopupID, CardID: k.CardID, CardIndex: k.CardIndex,
				ApplicationID: applicationID, AppVersion: k.AppVersion, Event: k.Event, Dim: k.Dim, Count: n,
			})
		}
		// 每行累加量不同，逐行 upsert：count = count + ?（单批 ≤100 事件，行数有限）。
		// 反引号包裹的 `表`.`列` 在 sqlite 与 mysql 下写法一致。
		for i := range rows {
			row := rows[i]
			if e := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "stat_date"}, {Name: "popup_id"}, {Name: "card_id"}, {Name: "card_index"},
					{Name: "application_id"}, {Name: "app_version"}, {Name: "event"}, {Name: "dim"},
				},
				DoUpdates: clause.Assignments(map[string]any{
					"count": gorm.Expr("`popup_stat_daily`.`count` + ?", row.Count),
				}),
			}).Create(&row).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("写入弹窗埋点失败: %w", err)
	}
	return dup, nil
}

// lessPopupStatKey 按唯一键列顺序比较。
func lessPopupStatKey(a, b PopupStatKey) bool {
	if a.StatDate != b.StatDate {
		return a.StatDate < b.StatDate
	}
	if a.PopupID != b.PopupID {
		return a.PopupID < b.PopupID
	}
	if a.CardID != b.CardID {
		return a.CardID < b.CardID
	}
	if a.CardIndex != b.CardIndex {
		return a.CardIndex < b.CardIndex
	}
	if a.AppVersion != b.AppVersion {
		return a.AppVersion < b.AppVersion
	}
	if a.Event != b.Event {
		return a.Event < b.Event
	}
	return a.Dim < b.Dim
}

// purgeChunk 每次删除的批次行数上限（避免一条 DELETE 锁住一整天的行）。
const purgeChunk = 5000

// PurgePopupEventBatches 分块清理早于 before 的上报批次记录。
func (r *Repo) PurgePopupEventBatches(ctx context.Context, before time.Time) (int64, error) {
	var total int64
	for {
		var ids []uint64
		if err := r.db.WithContext(ctx).Model(&model.PopupEventBatch{}).
			Where("created_at < ?", before).Limit(purgeChunk).Pluck("id", &ids).Error; err != nil {
			return total, fmt.Errorf("清理弹窗上报批次失败: %w", err)
		}
		if len(ids) == 0 {
			return total, nil
		}
		res := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.PopupEventBatch{})
		if res.Error != nil {
			return total, fmt.Errorf("清理弹窗上报批次失败: %w", res.Error)
		}
		total += res.RowsAffected
		if len(ids) < purgeChunk {
			return total, nil
		}
	}
}

// ChannelLite 是公开热路径用的轻量渠道信息（不预载品牌域名）。
type ChannelLite struct {
	ApplicationID string
	BrandCode     string
	Status        string
}

// GetChannelLite 只取 application_id / 品牌 code / 状态；不存在返回 ErrNotFound。
func (r *Repo) GetChannelLite(ctx context.Context, appID string) (*ChannelLite, error) {
	var rows []ChannelLite
	if err := r.db.WithContext(ctx).Table("channel ch").
		Select("ch.application_id AS application_id, b.code AS brand_code, ch.status AS status").
		Joins("JOIN brand b ON b.id = ch.brand_id").
		Where("ch.application_id = ?", appID).Limit(1).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询渠道失败: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

// PopupStatFilter 统计聚合过滤。AppIDs 非 nil 表示按 application_id 限定（含数据权限与 brand/appId 筛选）。
type PopupStatFilter struct {
	From, To string // YYYY-MM-DD，含
	PopupIDs []uint64
	AppIDs   []string
	// RestrictApps 为 true 时 AppIDs 即使为空也生效（空 = 什么都看不到）。
	RestrictApps bool
}

// PopupStatRow 是聚合结果的一行。
type PopupStatRow struct {
	StatDate  string
	PopupID   uint64
	CardID    uint64
	CardIndex int
	Event     string
	Dim       string
	Count     int64
}

// AggregatePopupStats 按 (日期, 弹窗, 卡片, card_index, event, dim) 聚合（合并 app/version 维度）。
func (r *Repo) AggregatePopupStats(ctx context.Context, f PopupStatFilter) ([]PopupStatRow, error) {
	if f.RestrictApps && len(f.AppIDs) == 0 {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Model(&model.PopupStatDaily{}).
		Select("stat_date, popup_id, card_id, card_index, event, dim, SUM(`count`) AS count").
		Where("stat_date >= ? AND stat_date <= ?", f.From, f.To).
		Group("stat_date, popup_id, card_id, card_index, event, dim")
	if f.PopupIDs != nil {
		if len(f.PopupIDs) == 0 {
			return nil, nil
		}
		q = q.Where("popup_id IN ?", f.PopupIDs)
	}
	if f.RestrictApps {
		q = q.Where("application_id IN ?", f.AppIDs)
	}
	var rows []PopupStatRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("聚合弹窗统计失败: %w", err)
	}
	return rows, nil
}

// ApplicationIDsByBrandCodes 取若干品牌下的全部渠道 applicationId（含已归档，统计历史数据仍要能看）。
func (r *Repo) ApplicationIDsByBrandCodes(ctx context.Context, codes []string) ([]string, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	var ids []string
	if err := r.db.WithContext(ctx).Table("channel ch").
		Joins("JOIN brand b ON b.id = ch.brand_id").
		Where("b.code IN ?", codes).
		Pluck("ch.application_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("按品牌查询 applicationId 失败: %w", err)
	}
	return ids, nil
}
