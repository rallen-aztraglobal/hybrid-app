// Package service — 马甲包弹窗模块：统计口径（docs/admin/12-popup.md §5）。
package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// PopupStatsQuery 统计查询参数。
type PopupStatsQuery struct {
	From     string // YYYY-MM-DD（含），默认 to-6d
	To       string // YYYY-MM-DD（含），默认今天（POPUP_TZ）
	Brand    string
	AppID    string
	PopupID  uint64
	Position string
}

// PopupFilteredStat 拦截分布。
type PopupFilteredStat struct {
	Frequency int64 `json:"frequency"`
	Mutex     int64 `json:"mutex"`
	Targeting int64 `json:"targeting"`
	Time      int64 `json:"time"`
}

// PopupCloseStat 关闭方式分布。
type PopupCloseStat struct {
	Button int64 `json:"button"`
	Mask   int64 `json:"mask"`
	Back   int64 `json:"back"`
}

// PopupRates 比率；分母为 0 时为 0。
type PopupRates struct {
	ShowRate       float64 `json:"showRate"`
	CTR            float64 `json:"ctr"`
	CloseRate      float64 `json:"closeRate"`
	LoadFailRate   float64 `json:"loadFailRate"`
	CarouselDepth  float64 `json:"carouselDepth"`
	RecoveryRate   float64 `json:"recoveryRate"`
	TabAbandonRate float64 `json:"tabAbandonRate"`
	TotalCTR       float64 `json:"totalCtr"`
}

// PopupCardStat 卡片级曝光 / 点击。
type PopupCardStat struct {
	CardID      uint64  `json:"cardId"`
	CardIndex   int     `json:"cardIndex"`
	ImageURL    string  `json:"imageUrl"`
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
	CTR         float64 `json:"ctr"`
}

// PopupDailyStat 按日趋势。
type PopupDailyStat struct {
	Date        string `json:"date"`
	Trigger     int64  `json:"trigger"`
	Displays    int64  `json:"displays"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
}

// PopupStatsItem 单个弹窗一行。
type PopupStatsItem struct {
	PopupID        uint64            `json:"popupId"`
	Name           string            `json:"name"`
	Position       string            `json:"position"`
	Deleted        bool              `json:"deleted"`
	Trigger        int64             `json:"trigger"`
	Displays       int64             `json:"displays"`
	DisplaysAll    int64             `json:"displaysAll"` // Σ impression where card_index=0（含 auto+tab），关闭率 / 轮播深度的分母
	Impressions    int64             `json:"impressions"`
	Clicks         int64             `json:"clicks"`
	Closes         int64             `json:"closes"`
	LoadFails      int64             `json:"loadFails"`
	Collapses      int64             `json:"collapses"`
	TabImpressions int64             `json:"tabImpressions"`
	TabClicks      int64             `json:"tabClicks"`
	TabDismisses   int64             `json:"tabDismisses"`
	Slides         int64             `json:"slides"`
	Filtered       PopupFilteredStat `json:"filtered"`
	CloseByMethod  PopupCloseStat    `json:"closeByMethod"`
	Rates          PopupRates        `json:"rates"`
	Cards          []PopupCardStat   `json:"cards"`
	Daily          []PopupDailyStat  `json:"daily"`
}

// PopupStatsResult 统计响应。
type PopupStatsResult struct {
	TZ              string           `json:"tz"`              // POPUP_TZ 名
	TzOffsetMinutes int              `json:"tzOffsetMinutes"` // 当前时刻该时区相对 UTC 的偏移
	From            string           `json:"from"`            // 实际生效的起始日期
	To              string           `json:"to"`              // 实际生效的结束日期
	Popups          []PopupStatsItem `json:"popups"`
}

func ratio(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// computePopupStats 把聚合行按契约 §5 公式折算为每弹窗一行（纯函数）。
// popups 提供名称 / 位置 / 软删标记 / 卡片图；mustInclude 里的弹窗即使无数据也输出零行。
func computePopupStats(rows []repo.PopupStatRow, popups map[uint64]model.Popup, mustInclude []uint64) []PopupStatsItem {
	type cardKey struct {
		id  uint64
		idx int
	}
	type acc struct {
		item  PopupStatsItem
		idx0  int64 // Σ impression where card_index=0（含便条重开）
		cards map[cardKey]*PopupCardStat
		daily map[string]*PopupDailyStat
	}
	accs := map[uint64]*acc{}
	get := func(id uint64) *acc {
		a := accs[id]
		if a == nil {
			a = &acc{cards: map[cardKey]*PopupCardStat{}, daily: map[string]*PopupDailyStat{}}
			a.item.PopupID = id
			accs[id] = a
		}
		return a
	}
	cardOf := func(a *acc, id uint64, idx int) *PopupCardStat {
		k := cardKey{id, idx}
		c := a.cards[k]
		if c == nil {
			c = &PopupCardStat{CardID: id, CardIndex: idx}
			a.cards[k] = c
		}
		return c
	}
	for _, id := range mustInclude {
		get(id)
	}
	for _, r := range rows {
		a := get(r.PopupID)
		d := a.daily[r.StatDate]
		if d == nil {
			d = &PopupDailyStat{Date: r.StatDate}
			a.daily[r.StatDate] = d
		}
		n := r.Count
		it := &a.item
		switch r.Event {
		case popupEvTrigger:
			it.Trigger += n
			d.Trigger += n
		case popupEvLoadFail:
			it.LoadFails += n
		case popupEvImpression:
			it.Impressions += n
			d.Impressions += n
			if r.CardIndex == 0 {
				a.idx0 += n
				if r.Dim == "auto" {
					it.Displays += n
					d.Displays += n
				}
			}
			if r.CardID > 0 {
				c := cardOf(a, r.CardID, r.CardIndex)
				c.Impressions += n
			}
		case popupEvClick:
			it.Clicks += n
			d.Clicks += n
			if r.CardID > 0 {
				c := cardOf(a, r.CardID, r.CardIndex)
				c.Clicks += n
			}
		case popupEvClose:
			it.Closes += n
			switch r.Dim {
			case "button":
				it.CloseByMethod.Button += n
			case "mask":
				it.CloseByMethod.Mask += n
			case "back":
				it.CloseByMethod.Back += n
			}
		case popupEvCollapse:
			it.Collapses += n
		case popupEvTabImpression:
			it.TabImpressions += n
		case popupEvTabClick:
			it.TabClicks += n
		case popupEvTabDismiss:
			it.TabDismisses += n
		case popupEvSlide:
			it.Slides += n
		case popupEvFiltered:
			switch r.Dim {
			case "frequency":
				it.Filtered.Frequency += n
			case "mutex":
				it.Filtered.Mutex += n
			case "targeting":
				it.Filtered.Targeting += n
			case "time":
				it.Filtered.Time += n
			}
		}
	}

	ids := make([]uint64, 0, len(accs))
	for id := range accs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	out := make([]PopupStatsItem, 0, len(ids))
	for _, id := range ids {
		a := accs[id]
		it := a.item
		pop, known := popups[id]
		if !known {
			continue // 库里没有的弹窗（脏数据 / 非法 popupId）不输出
		}
		it.Name = pop.Name
		it.Position = pop.Position
		it.Deleted = pop.DeletedAt.Valid
		it.DisplaysAll = a.idx0
		imgByCard := map[uint64]string{}
		for _, c := range pop.Cards {
			imgByCard[c.ID] = c.ImageURL
		}
		it.Rates = PopupRates{
			ShowRate:       ratio(it.Displays, it.Trigger),
			CTR:            ratio(it.Clicks, it.Impressions),
			CloseRate:      ratio(it.Closes, a.idx0),
			LoadFailRate:   ratio(it.LoadFails, it.Trigger),
			CarouselDepth:  ratio(it.Impressions, a.idx0),
			RecoveryRate:   ratio(it.TabClicks, it.Collapses),
			TabAbandonRate: ratio(it.TabDismisses, it.TabImpressions),
			TotalCTR:       ratio(it.Clicks+it.TabClicks, it.Impressions),
		}
		it.Cards = make([]PopupCardStat, 0, len(a.cards))
		for _, c := range a.cards {
			c.ImageURL = imgByCard[c.CardID]
			c.CTR = ratio(c.Clicks, c.Impressions)
			it.Cards = append(it.Cards, *c)
		}
		sort.Slice(it.Cards, func(i, j int) bool {
			if it.Cards[i].CardIndex != it.Cards[j].CardIndex {
				return it.Cards[i].CardIndex < it.Cards[j].CardIndex
			}
			return it.Cards[i].CardID < it.Cards[j].CardID
		})
		it.Daily = make([]PopupDailyStat, 0, len(a.daily))
		for _, d := range a.daily {
			it.Daily = append(it.Daily, *d)
		}
		sort.Slice(it.Daily, func(i, j int) bool { return it.Daily[i].Date < it.Daily[j].Date })
		out = append(out, it)
	}
	return out
}

// PopupStats 统计查询：from/to 默认近 7 天（POPUP_TZ），跨度 ≤ 92 天；受限账号只统计其范围内 applicationId。
func (s *Service) PopupStats(ctx context.Context, scope auth.Scope, q PopupStatsQuery) (*PopupStatsResult, error) {
	loc := s.popupLoc()
	const layout = "2006-01-02"
	toT := time.Now().In(loc)
	var err error
	if q.To != "" {
		if toT, err = time.ParseInLocation(layout, q.To, loc); err != nil {
			return nil, errBadRequest("to 格式应为 YYYY-MM-DD")
		}
	}
	fromT := toT.AddDate(0, 0, -6)
	if q.From != "" {
		if fromT, err = time.ParseInLocation(layout, q.From, loc); err != nil {
			return nil, errBadRequest("from 格式应为 YYYY-MM-DD")
		}
	}
	from, to := fromT.Format(layout), toT.Format(layout)
	if from > to {
		return nil, errBadRequest("from 不得晚于 to")
	}
	if days := toT.Sub(fromT).Hours() / 24; days > 92.5 {
		return nil, errBadRequest("统计跨度不得超过 92 天")
	}

	f := repo.PopupStatFilter{From: from, To: to}

	// application_id 限定 = 数据权限 ∩ brand ∩ appId。
	var allowed map[string]bool // nil = 不限
	intersect := func(ids []string) {
		set := make(map[string]bool, len(ids))
		for _, id := range ids {
			if allowed == nil || allowed[id] {
				set[id] = true
			}
		}
		allowed = set
	}
	if !scopeIsFull(scope) {
		var ids []string
		var e error
		if !scope.AllChannels {
			ids, e = s.repo.ApplicationIDsByChannelIDs(ctx, scope.ChannelIDList())
		} else {
			ids, e = s.repo.ApplicationIDsByBrandCodes(ctx, scope.BrandCodeList())
		}
		if e != nil {
			return nil, e
		}
		intersect(ids)
	}
	if b := strings.ToLower(strings.TrimSpace(q.Brand)); b != "" {
		ids, e := s.repo.ApplicationIDsByBrandCodes(ctx, []string{b})
		if e != nil {
			return nil, e
		}
		intersect(ids)
	}
	if a := strings.TrimSpace(q.AppID); a != "" {
		intersect([]string{a})
	}
	if allowed != nil {
		f.RestrictApps = true
		for id := range allowed {
			f.AppIDs = append(f.AppIDs, id)
		}
	}

	// popup 限定 = popupId ∩ position。
	var mustInclude []uint64
	if q.PopupID != 0 {
		f.PopupIDs = []uint64{q.PopupID}
		mustInclude = []uint64{q.PopupID}
	}
	if pos := strings.ToUpper(strings.TrimSpace(q.Position)); pos != "" {
		ids, e := s.repo.PopupIDsByPositionUnscoped(ctx, pos)
		if e != nil {
			return nil, e
		}
		if q.PopupID != 0 {
			keep := []uint64{}
			for _, id := range ids {
				if id == q.PopupID {
					keep = append(keep, id)
				}
			}
			ids = keep
			if len(ids) == 0 {
				mustInclude = nil
			}
		}
		f.PopupIDs = ids
		if f.PopupIDs == nil {
			f.PopupIDs = []uint64{}
		}
	}

	rows, err := s.repo.AggregatePopupStats(ctx, f)
	if err != nil {
		return nil, err
	}
	idSet := map[uint64]bool{}
	for _, r := range rows {
		idSet[r.PopupID] = true
	}
	for _, id := range mustInclude {
		idSet[id] = true
	}
	ids := make([]uint64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	list, err := s.repo.PopupsByIDsUnscoped(ctx, ids)
	if err != nil {
		return nil, err
	}
	popups := make(map[uint64]model.Popup, len(list))
	for _, p := range list {
		popups[p.ID] = p
	}
	// 位置过滤已在 SQL 层完成；popupId 指定但库里没有该弹窗时不输出幽灵行。
	// 非全量范围：无数据的零行只在该弹窗定向落在范围内时才输出，避免借 popupId 探测越权弹窗的名称 / 位置。
	var info map[string]repo.ChannelBrandInfo
	if !scopeIsFull(scope) {
		if info, err = s.popupChannelInfo(ctx, scope, list); err != nil {
			return nil, err
		}
	}
	filteredMust := mustInclude[:0:0]
	for _, id := range mustInclude {
		p, ok := popups[id]
		if !ok {
			continue
		}
		if !scopeIsFull(scope) && !idSet2(rows)[id] && !popupInScope(scope, &p, info) {
			continue
		}
		filteredMust = append(filteredMust, id)
	}
	now := time.Now()
	return &PopupStatsResult{
		TZ: loc.String(), TzOffsetMinutes: tzOffsetMinutes(loc, now), From: from, To: to,
		Popups: computePopupStats(rows, popups, filteredMust),
	}, nil
}

// idSet2 返回聚合行里出现过的 popupId 集合。
func idSet2(rows []repo.PopupStatRow) map[uint64]bool {
	m := make(map[uint64]bool, len(rows))
	for _, r := range rows {
		m[r.PopupID] = true
	}
	return m
}
