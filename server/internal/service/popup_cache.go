// Package service — 弹窗模块进程内缓存（公开热路径减负）。
package service

import (
	"context"
	"sync"
	"time"

	"github.com/hybrid-app/server/internal/model"
)

const (
	popupLiveTTL = 15 * time.Second // 生效弹窗列表 / 位置开关
	popupMetaTTL = 30 * time.Second // popupId → 卡片 id 集合（埋点校验）
	popupMetaMax = 20000            // 元数据缓存条目上限，超出整体重置
)

// popupMeta 是埋点校验用的弹窗元数据（含软删弹窗：删除前产生的埋点仍应入库）。
type popupMeta struct {
	exists bool
	cards  map[uint64]bool
	at     time.Time
}

// popupState 是 Service 内的弹窗缓存。后台写操作后调用 invalidate 主动失效本进程缓存；
// 多实例部署下其它实例最多滞后一个 TTL。
type popupState struct {
	mu     sync.Mutex
	live   []model.Popup
	liveAt time.Time
	pos    map[string]bool
	posAt  time.Time
	meta   map[uint64]popupMeta
}

func (st *popupState) invalidate() {
	st.mu.Lock()
	st.live, st.liveAt = nil, time.Time{}
	st.pos, st.posAt = nil, time.Time{}
	st.meta = nil
	st.mu.Unlock()
}

// cachedLivePopups 返回生效弹窗候选（end_at 过滤由 selectPopupsForApp 在内存里再做一遍）。
func (s *Service) cachedLivePopups(ctx context.Context, now time.Time) ([]model.Popup, error) {
	st := &s.popup
	st.mu.Lock()
	if st.live != nil && now.Sub(st.liveAt) < popupLiveTTL {
		v := st.live
		st.mu.Unlock()
		return v, nil
	}
	st.mu.Unlock()
	list, err := s.repo.ListLivePopups(ctx, now)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []model.Popup{}
	}
	st.mu.Lock()
	st.live, st.liveAt = list, now
	st.mu.Unlock()
	return list, nil
}

// cachedPositionEnabled 返回位置开关 map（只读，调用方不得修改）。
func (s *Service) cachedPositionEnabled(ctx context.Context, now time.Time) (map[string]bool, error) {
	st := &s.popup
	st.mu.Lock()
	if st.pos != nil && now.Sub(st.posAt) < popupLiveTTL {
		v := st.pos
		st.mu.Unlock()
		return v, nil
	}
	st.mu.Unlock()
	m, err := s.positionEnabledMap(ctx)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	st.pos, st.posAt = m, now
	st.mu.Unlock()
	return m, nil
}

// popupMetas 批量取弹窗元数据；过期 / 未缓存的从库里（含软删）补查并缓存（不存在也缓存，防灌爆查询）。
func (s *Service) popupMetas(ctx context.Context, ids []uint64, now time.Time) (map[uint64]popupMeta, error) {
	st := &s.popup
	out := make(map[uint64]popupMeta, len(ids))
	var miss []uint64
	st.mu.Lock()
	for _, id := range ids {
		if m, ok := st.meta[id]; ok && now.Sub(m.at) < popupMetaTTL {
			out[id] = m
		} else {
			miss = append(miss, id)
		}
	}
	st.mu.Unlock()
	if len(miss) == 0 {
		return out, nil
	}
	list, err := s.repo.PopupsByIDsUnscoped(ctx, miss)
	if err != nil {
		return nil, err
	}
	found := make(map[uint64]popupMeta, len(list))
	for _, p := range list {
		cards := make(map[uint64]bool, len(p.Cards))
		for _, c := range p.Cards {
			cards[c.ID] = true
		}
		found[p.ID] = popupMeta{exists: true, cards: cards, at: now}
	}
	st.mu.Lock()
	if st.meta == nil || len(st.meta) > popupMetaMax {
		st.meta = map[uint64]popupMeta{}
	}
	for _, id := range miss {
		m, ok := found[id]
		if !ok {
			m = popupMeta{at: now}
		}
		st.meta[id] = m
		out[id] = m
	}
	st.mu.Unlock()
	return out, nil
}
