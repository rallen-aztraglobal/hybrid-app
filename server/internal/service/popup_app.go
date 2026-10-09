// Package service — 马甲包弹窗模块：App 端公开配置下发、埋点上报、素材上传
// （docs/admin/12-popup.md §4）。
package service

import (
	"bytes"
	"context"
	"errors"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"  // 注册 gif 解码器（DecodeConfig 取宽高）
	_ "image/jpeg" // 注册 jpeg 解码器
	_ "image/png"  // 注册 png 解码器
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	_ "golang.org/x/image/webp" // 注册 webp 解码器

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// ---------- App 端配置 ----------

// PopupAppCard 下发给 App 的卡片。
type PopupAppCard struct {
	ID          uint64 `json:"id"`
	ImageURL    string `json:"imageUrl"`
	LinkURL     string `json:"linkUrl"`
	ButtonText  string `json:"buttonText"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// PopupAppItem 下发给 App 的弹窗（版本 / 新老用户 / 生效时间由客户端过滤）。
type PopupAppItem struct {
	ID               uint64         `json:"id"`
	Position         string         `json:"position"`
	Priority         int            `json:"priority"`
	StartAt          int64          `json:"startAt"` // 毫秒时间戳，0 = 不限
	EndAt            int64          `json:"endAt"`
	MinVersionCode   int            `json:"minVersionCode"`
	MaxVersionCode   int            `json:"maxVersionCode"`
	UserType         string         `json:"userType"`
	OpenMode         string         `json:"openMode"`
	Closable         bool           `json:"closable"`
	MaskClosable     bool           `json:"maskClosable"`
	Countdown        bool           `json:"countdown"`
	TabEnabled       bool           `json:"tabEnabled"`
	TabIconURL       string         `json:"tabIconUrl"`
	TabText          string         `json:"tabText"`
	AutoplaySeconds  int            `json:"autoplaySeconds"`
	ResumeGapMinutes int            `json:"resumeGapMinutes"`
	Badge            bool           `json:"badge"`
	Cards            []PopupAppCard `json:"cards"`
}

// PopupAppConfig GET /api/app/popups 响应（裸 JSON）。
type PopupAppConfig struct {
	AppID           string         `json:"appId"`
	ConfigVersion   string         `json:"configVersion"`
	ServerTime      int64          `json:"serverTime"`
	TzOffsetMinutes int            `json:"tzOffsetMinutes"`
	Popups          []PopupAppItem `json:"popups"`
}

func millis(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

// selectPopupsForApp 按契约 §4.1 过滤并组装下发项（纯函数）：
// 位置开 ∧ 弹窗开 ∧ endAt 未到 ∧ 品牌/包定向命中 ∧ 地区命中（country 为空而设了地区定向 → 不命中）。
// 未到 startAt 的照常下发。排序：priority 降序，平手 id 升序（保证 configVersion 稳定）。
func selectPopupsForApp(popups []model.Popup, posEnabled map[string]bool, brandCode, appID, country string, now time.Time) []PopupAppItem {
	out := make([]PopupAppItem, 0, len(popups))
	for i := range popups {
		p := &popups[i]
		if !p.Enabled || !posEnabled[p.Position] {
			continue
		}
		if p.EndAt != nil && !p.EndAt.After(now) {
			continue
		}
		if len(p.BrandCodes) > 0 && !containsStr(p.BrandCodes, brandCode) {
			continue
		}
		if len(p.AppIDs) > 0 && !containsStr(p.AppIDs, appID) {
			continue
		}
		if len(p.Countries) > 0 && (country == "" || !containsStr(p.Countries, strings.ToUpper(country))) {
			continue
		}
		cards := append([]model.PopupCard(nil), p.Cards...)
		sort.SliceStable(cards, func(a, b int) bool {
			if cards[a].Sort != cards[b].Sort {
				return cards[a].Sort < cards[b].Sort
			}
			return cards[a].ID < cards[b].ID
		})
		appCards := make([]PopupAppCard, 0, len(cards))
		for _, c := range cards {
			appCards = append(appCards, PopupAppCard{
				ID: c.ID, ImageURL: c.ImageURL, LinkURL: c.LinkURL,
				ButtonText: c.ButtonText, Title: c.Title, Description: c.Description,
			})
		}
		tabIcon := p.TabIconURL
		if tabIcon == "" && len(appCards) > 0 {
			tabIcon = appCards[0].ImageURL // 便条图标空 = 回填第一张卡片图
		}
		out = append(out, PopupAppItem{
			ID: p.ID, Position: p.Position, Priority: p.Priority,
			StartAt: millis(p.StartAt), EndAt: millis(p.EndAt),
			MinVersionCode: p.MinVersionCode, MaxVersionCode: p.MaxVersionCode, UserType: p.UserType,
			OpenMode: p.OpenMode, Closable: p.Closable, MaskClosable: p.MaskClosable, Countdown: p.Countdown,
			TabEnabled: p.TabEnabled, TabIconURL: tabIcon, TabText: p.TabText,
			AutoplaySeconds: p.AutoplaySeconds, ResumeGapMinutes: p.ResumeGapMinutes, Badge: p.Badge,
			Cards: appCards,
		})
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Priority != out[b].Priority {
			return out[a].Priority > out[b].Priority
		}
		return out[a].ID < out[b].ID
	})
	return out
}

// popupConfigVersion = popups 数组 JSON 的 sha256 前 16 位 hex（不含 serverTime）。
func popupConfigVersion(items []PopupAppItem) string {
	b, _ := json.Marshal(items)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:16]
}

// tzOffsetMinutes 返回 loc 在 now 时刻相对 UTC 的偏移（分钟）。
func tzOffsetMinutes(loc *time.Location, now time.Time) int {
	_, off := now.In(loc).Zone()
	return off / 60
}

// AssemblePopupConfig 组装某个渠道包当前应拿到的弹窗配置（App 端 API 与 runtime-preview 共用）。
// country 为空表示判不出国家。未知 / 已归档 appId → 404。
func (s *Service) AssemblePopupConfig(ctx context.Context, appID, country string) (*PopupAppConfig, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, errBadRequest("缺少 appId 参数")
	}
	ch, err := s.repo.GetChannelLite(ctx, appID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, errNotFound("applicationId 对应渠道不存在")
		}
		return nil, err
	}
	if ch.Status == model.ChannelArchived {
		return nil, errNotFound("applicationId 对应渠道不存在")
	}
	now := time.Now()
	list, err := s.cachedLivePopups(ctx, now)
	if err != nil {
		return nil, err
	}
	posEnabled, err := s.cachedPositionEnabled(ctx, now)
	if err != nil {
		return nil, err
	}
	items := selectPopupsForApp(list, posEnabled, ch.BrandCode, appID, country, now)
	return &PopupAppConfig{
		AppID:           appID,
		ConfigVersion:   popupConfigVersion(items),
		ServerTime:      now.UnixMilli(),
		TzOffsetMinutes: tzOffsetMinutes(s.popupLoc(), now),
		Popups:          items,
	}, nil
}

// PopupConfigForIP 公开端点：按请求真实 IP 解析国家后组装。
func (s *Service) PopupConfigForIP(ctx context.Context, appID string, ip net.IP) (*PopupAppConfig, error) {
	country := ""
	if s.geo != nil && ip != nil {
		if c, ok := s.geo.Country(ip); ok {
			country = c
		}
	}
	return s.AssemblePopupConfig(ctx, appID, country)
}

// PopupRuntimePreview 管理端预览：与 App 端完全相同的 payload，country 由参数指定；
// 受限账号只能预览其数据范围内的渠道包（越界 404）。
func (s *Service) PopupRuntimePreview(ctx context.Context, scope auth.Scope, appID, country string) (*PopupAppConfig, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, errBadRequest("缺少 appId 参数")
	}
	if !scopeIsFull(scope) {
		info, err := s.repo.ChannelBrandsByApplicationIDs(ctx, []string{appID})
		if err != nil {
			return nil, err
		}
		row, ok := info[appID]
		if !ok || !scope.ChannelAllowed(row.BrandCode, row.ChannelID) {
			return nil, errNotFound("applicationId 对应渠道不存在")
		}
	}
	country = strings.ToUpper(strings.TrimSpace(country))
	if country != "" && !countryRe.MatchString(country) {
		return nil, errBadRequest("country 应为 ISO-3166 二位国家码")
	}
	return s.AssemblePopupConfig(ctx, appID, country)
}

// ---------- 埋点上报 ----------

// 契约 §4.2 / PRD §7.1 的 11 个事件名。
const (
	popupEvTrigger         = "popup_trigger"
	popupEvFiltered        = "popup_filtered"
	popupEvLoadFail        = "popup_load_fail"
	popupEvImpression      = "popup_impression"
	popupEvClick           = "popup_click"
	popupEvClose           = "popup_close"
	popupEvCollapse        = "popup_collapse"
	popupEvTabImpression   = "tab_impression"
	popupEvTabClick        = "tab_click"
	popupEvTabDismiss      = "tab_dismiss"
	popupEvSlide           = "popup_slide"
	popupMaxEventsPerBatch = 100
)

var popupKnownEvents = map[string]bool{
	popupEvTrigger: true, popupEvFiltered: true, popupEvLoadFail: true, popupEvImpression: true,
	popupEvClick: true, popupEvClose: true, popupEvCollapse: true, popupEvTabImpression: true,
	popupEvTabClick: true, popupEvTabDismiss: true, popupEvSlide: true,
}

// PopupEventIn 单条埋点事件。
type PopupEventIn struct {
	Event     string `json:"event"`
	PopupID   uint64 `json:"popupId"`
	Position  string `json:"position"`
	CardID    uint64 `json:"cardId"`
	CardIndex int    `json:"cardIndex"`
	Display   string `json:"display"`
	Reason    string `json:"reason"`
	Method    string `json:"method"`
	From      int    `json:"from"`
	To        int    `json:"to"`
	TS        int64  `json:"ts"`
}

// PopupEventsRequest 批次上报请求体。
type PopupEventsRequest struct {
	BatchID     string         `json:"batchId"`
	AppID       string         `json:"appId"`
	PalCode     string         `json:"palcode"`
	AppVersion  string         `json:"appVersion"`
	VersionCode int            `json:"versionCode"`
	Events      []PopupEventIn `json:"events"`
}

// PopupEventsResult 上报结果。
type PopupEventsResult struct {
	Accepted  int  `json:"accepted"`  // 实际计入统计的事件数
	Duplicate bool `json:"duplicate"` // batchId 重复（幂等，未累加）
}

func oneOf(v string, allowed ...string) string {
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	return "other"
}

// popupStatKeyOf 把一个事件映射为日聚合唯一键（不含 application_id）。第二返回值 false = 丢弃。
// 维度规则（契约 §2.4）：filtered→reason，close→method，impression/click→display，slide→card_index=to；
// 卡片级事件（impression/click）带 card_id/card_index，其余弹窗级事件为 0。
func popupStatKeyOf(ev PopupEventIn, appVersion string, now time.Time, loc *time.Location) (repo.PopupStatKey, bool) {
	if !popupKnownEvents[ev.Event] || ev.PopupID == 0 {
		return repo.PopupStatKey{}, false
	}
	ts := now
	if ev.TS > 0 {
		t := time.UnixMilli(ev.TS)
		if !t.Before(now.Add(-7*24*time.Hour)) && !t.After(now.Add(time.Hour)) {
			ts = t
		}
	}
	k := repo.PopupStatKey{
		StatDate: ts.In(loc).Format("2006-01-02"), PopupID: ev.PopupID,
		AppVersion: appVersion, Event: ev.Event,
	}
	clampIdx := func(i int) int {
		if i < 0 {
			return 0
		}
		if i > 99 {
			return 99
		}
		return i
	}
	switch ev.Event {
	case popupEvFiltered:
		k.Dim = oneOf(ev.Reason, "frequency", "mutex", "targeting", "time")
	case popupEvClose:
		k.Dim = oneOf(ev.Method, "button", "mask", "back")
	case popupEvImpression, popupEvClick:
		k.Dim = oneOf(ev.Display, "auto", "tab")
		k.CardID = ev.CardID
		k.CardIndex = clampIdx(ev.CardIndex)
	case popupEvSlide:
		k.CardIndex = clampIdx(ev.To)
	}
	return k, true
}

// IngestPopupEvents 接收一批埋点：校验 → 内存按唯一键合并 → 事务内幂等累加。
func (s *Service) IngestPopupEvents(ctx context.Context, req PopupEventsRequest) (*PopupEventsResult, error) {
	req.BatchID = strings.TrimSpace(req.BatchID)
	req.AppID = strings.TrimSpace(req.AppID)
	if req.BatchID == "" || len(req.BatchID) > 64 {
		return nil, errBadRequest("batchId 必填且不超过 64 字符")
	}
	if req.AppID == "" {
		return nil, errBadRequest("appId 不得为空")
	}
	if len(req.Events) > popupMaxEventsPerBatch {
		return nil, errBadRequest(fmt.Sprintf("每批最多 %d 条事件", popupMaxEventsPerBatch))
	}
	if _, err := s.repo.GetChannelLite(ctx, req.AppID); err != nil {
		return nil, errBadRequest("applicationId 对应渠道不存在")
	}
	version := strings.TrimSpace(req.AppVersion)
	if len(version) > 32 {
		version = version[:32]
	}
	now := time.Now()
	loc := s.popupLoc()
	// 先按事件名 / popupId 粗筛，再用缓存校验 popupId 真实存在，防止被灌入任意 popupId。
	idSet := map[uint64]bool{}
	for _, ev := range req.Events {
		if popupKnownEvents[ev.Event] && ev.PopupID != 0 {
			idSet[ev.PopupID] = true
		}
	}
	ids := make([]uint64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	metas, err := s.popupMetas(ctx, ids, now)
	if err != nil {
		return nil, err
	}
	counts := map[repo.PopupStatKey]int64{}
	accepted := 0
	for _, ev := range req.Events {
		k, ok := popupStatKeyOf(ev, version, now, loc)
		if !ok {
			continue // 未知事件名静默丢弃（向前兼容）
		}
		m := metas[k.PopupID]
		if !m.exists {
			continue // 未知 popupId 丢弃
		}
		if k.CardID != 0 && !m.cards[k.CardID] {
			k.CardID = 0 // card_id 不属于该弹窗：置 0，card_index 保留
		}
		counts[k]++
		accepted++
	}
	dup, err := s.repo.ApplyPopupStats(ctx, req.BatchID, req.AppID, counts)
	if err != nil {
		return nil, err
	}
	if dup {
		return &PopupEventsResult{Duplicate: true}, nil
	}
	return &PopupEventsResult{Accepted: accepted}, nil
}

// PurgePopupEventBatches cron：清理 7 天前的上报批次记录。
func (s *Service) PurgePopupEventBatches(ctx context.Context) (int64, error) {
	return s.repo.PurgePopupEventBatches(ctx, time.Now().Add(-7*24*time.Hour))
}

// ---------- 素材上传 ----------

const popupImageMaxBytes = 3 << 20

// PopupImageResult upload-image 响应。
type PopupImageResult struct {
	URL    string `json:"url"`
	Key    string `json:"key"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int    `json:"size"`
}

var popupImageExt = map[string]string{
	"image/png": "png", "image/jpeg": "jpg", "image/webp": "webp", "image/gif": "gif",
}

// UploadPopupImage 校验并上传弹窗素材。对象 key 恒为 popup/images/<sha256 前 12 位>-<unix 毫秒>.<ext>，
// 每次上传都是新文件名、从不覆盖（换图必换 URL）。
func (s *Service) UploadPopupImage(ctx context.Context, data []byte) (*PopupImageResult, error) {
	if len(data) == 0 {
		return nil, errBadRequest("文件为空")
	}
	if len(data) > popupImageMaxBytes {
		return nil, errBadRequest("图片不得超过 3MB")
	}
	ct := http.DetectContentType(data)
	ext, ok := popupImageExt[ct]
	if !ok {
		return nil, errBadRequest("仅支持 png / jpeg / webp / gif 图片")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errBadRequest("无法解析图片尺寸，文件可能已损坏")
	}
	sum := sha256.Sum256(data)
	key := fmt.Sprintf("popup/images/%s-%d.%s", hex.EncodeToString(sum[:])[:12], time.Now().UnixMilli(), ext)
	u, err := s.storage.Put(ctx, key, bytes.NewReader(data), int64(len(data)), ct)
	if err != nil {
		return nil, fmt.Errorf("上传弹窗素材失败: %w", err)
	}
	return &PopupImageResult{URL: u, Key: key, Width: cfg.Width, Height: cfg.Height, Size: len(data)}, nil
}
