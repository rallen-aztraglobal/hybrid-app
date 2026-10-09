// Package service — 马甲包弹窗模块：管理端 CRUD、保存校验与能力矩阵归一化
// （ADR-0019 / docs/admin/12-popup.md §1~§3）。
package service

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// ---------- 能力矩阵（契约 §1） ----------

// popupCap 描述一个位置的能力；归一化时据此强制回默认。
type popupCap struct {
	Name          string
	MaxCards      int  // 卡片数上限（P1/P3/P7 = 5，其余 = 1）
	ImageRequired bool // 图片必填
	ImageUnused   bool // 不用图片（P5），传了也清空
	TitleRequired bool // 卡片文案必填
	TitleIfNoImg  bool // 无图时文案必填（P4）
	Tab           bool // 可开便条态
	Countdown     bool // 倒计时（P1/P7）
	Force         bool // 可设强制模式（P4）
	MaskClosable  bool // 遮罩点击关闭（P1/P4/P6）
	Badge         bool // 红点角标（P2）
	Autoplay      bool // 自动轮播间隔（P3）
	Resume        bool // 回前台触发间隔（P1）
	TitleMaxRunes int  // 卡片 title 长度上限（按 rune），P8 为 8
}

var popupCaps = map[string]popupCap{
	model.PopupP1: {Name: "启动弹窗", MaxCards: 5, ImageRequired: true, Tab: true, Countdown: true, MaskClosable: true, Resume: true, TitleMaxRunes: 64},
	model.PopupP2: {Name: "悬浮球", MaxCards: 1, ImageRequired: true, Badge: true, TitleMaxRunes: 64},
	model.PopupP3: {Name: "底部横幅", MaxCards: 5, TitleRequired: true, Tab: true, Autoplay: true, TitleMaxRunes: 64},
	model.PopupP4: {Name: "更新 / 公告", MaxCards: 1, TitleIfNoImg: true, Tab: true, Force: true, MaskClosable: true, TitleMaxRunes: 64},
	model.PopupP5: {Name: "顶部通栏", MaxCards: 1, ImageUnused: true, TitleRequired: true, TitleMaxRunes: 64},
	model.PopupP6: {Name: "退出挽留", MaxCards: 1, ImageRequired: true, MaskClosable: true, TitleMaxRunes: 64},
	model.PopupP7: {Name: "全屏插屏", MaxCards: 5, ImageRequired: true, Tab: true, Countdown: true, TitleMaxRunes: 64},
	model.PopupP8: {Name: "便条", MaxCards: 1, ImageRequired: true, TitleRequired: true, TitleMaxRunes: 8},
}

// PopupPositionCodes 按 P1..P8 顺序返回全部位置码。
func PopupPositionCodes() []string {
	return []string{model.PopupP1, model.PopupP2, model.PopupP3, model.PopupP4, model.PopupP5, model.PopupP6, model.PopupP7, model.PopupP8}
}

// PopupPositionName 返回位置中文名（未知返回空串）。
func PopupPositionName(code string) string { return popupCaps[code].Name }

// ---------- 输入 / 输出类型 ----------

// PopupCardInput 卡片入参；带 id 且属于该弹窗 = 原地更新。
type PopupCardInput struct {
	ID          uint64 `json:"id"`
	Sort        int    `json:"sort"`
	ImageURL    string `json:"imageUrl"`
	LinkURL     string `json:"linkUrl"`
	ButtonText  string `json:"buttonText"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// PopupInput 新建 / 全量更新弹窗入参（契约 §3 弹窗 JSON）。
type PopupInput struct {
	Name             string           `json:"name"`
	Position         string           `json:"position"`
	Enabled          bool             `json:"enabled"`
	Priority         int              `json:"priority"`
	StartAt          *time.Time       `json:"startAt"`
	EndAt            *time.Time       `json:"endAt"`
	BrandCodes       []string         `json:"brandCodes"`
	AppIDs           []string         `json:"appIds"`
	MinVersion       string           `json:"minVersion"`
	MaxVersion       string           `json:"maxVersion"`
	UserType         string           `json:"userType"`
	Countries        []string         `json:"countries"`
	TabEnabled       bool             `json:"tabEnabled"`
	TabIconURL       string           `json:"tabIconUrl"`
	TabText          string           `json:"tabText"`
	Countdown        bool             `json:"countdown"`
	MaskClosable     bool             `json:"maskClosable"`
	Closable         *bool            `json:"closable"` // 缺省 = true
	OpenMode         string           `json:"openMode"`
	AutoplaySeconds  int              `json:"autoplaySeconds"`
	ResumeGapMinutes int              `json:"resumeGapMinutes"`
	Badge            bool             `json:"badge"`
	Cards            []PopupCardInput `json:"cards"`
}

// PopupCardView 卡片出参。
type PopupCardView struct {
	ID          uint64 `json:"id"`
	Sort        int    `json:"sort"`
	ImageURL    string `json:"imageUrl"`
	LinkURL     string `json:"linkUrl"`
	ButtonText  string `json:"buttonText"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// PopupView 管理端弹窗出参。
type PopupView struct {
	ID               uint64          `json:"id"`
	Name             string          `json:"name"`
	Position         string          `json:"position"`
	Enabled          bool            `json:"enabled"`
	Priority         int             `json:"priority"`
	StartAt          *time.Time      `json:"startAt"`
	EndAt            *time.Time      `json:"endAt"`
	BrandCodes       []string        `json:"brandCodes"`
	AppIDs           []string        `json:"appIds"`
	MinVersion       string          `json:"minVersion"`
	MaxVersion       string          `json:"maxVersion"`
	UserType         string          `json:"userType"`
	Countries        []string        `json:"countries"`
	TabEnabled       bool            `json:"tabEnabled"`
	TabIconURL       string          `json:"tabIconUrl"`
	TabText          string          `json:"tabText"`
	Countdown        bool            `json:"countdown"`
	MaskClosable     bool            `json:"maskClosable"`
	Closable         bool            `json:"closable"`
	OpenMode         string          `json:"openMode"`
	AutoplaySeconds  int             `json:"autoplaySeconds"`
	ResumeGapMinutes int             `json:"resumeGapMinutes"`
	Badge            bool            `json:"badge"`
	Cards            []PopupCardView `json:"cards"`
	Status           string          `json:"status"`
	PositionEnabled  bool            `json:"positionEnabled"`
	CreatedBy        string          `json:"createdBy"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// PopupPositionView 位置开关出参。
type PopupPositionView struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

// PopupListQuery 列表过滤。
type PopupListQuery struct {
	Position string
	Brand    string
	Status   string
	Keyword  string
}

// 弹窗状态（服务端计算）。
const (
	PopupStatusActive    = "active"
	PopupStatusScheduled = "scheduled"
	PopupStatusEnded     = "ended"
	PopupStatusDisabled  = "disabled"
)

// ---------- 版本号 ----------

var popupVersionRe = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.(\d{1,2})$`)

// ParsePopupVersion 把 "X.Y.Z"（各段 0–99）换算为 major*10000+minor*100+patch；空串 = 0（不限）。
func ParsePopupVersion(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	m := popupVersionRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("版本号 %q 格式应为 X.Y.Z（各段 0-99）", s)
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return maj*10000 + min*100 + pat, nil
}

// FormatPopupVersion 是 ParsePopupVersion 的逆运算；0 → 空串。
func FormatPopupVersion(code int) string {
	if code <= 0 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", code/10000, code/100%100, code%100)
}

// ---------- 归一化 / 校验（纯函数，便于表驱动单测） ----------

var countryRe = regexp.MustCompile(`^[A-Z]{2}$`)

// normalizedPopup 是通过校验并按能力矩阵归一化后的结果。
type normalizedPopup struct {
	Popup model.Popup
	Cards []PopupCardInput // 已按数组下标重写 sort
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// validHTTPURL 判断是否 http(s) 绝对地址。
func validHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// validatePopupLink 校验 link_url：允许空；/path 站内相对路径（仅 webview）；http(s)；market://（仅 store）。
func validatePopupLink(link, openMode string) error {
	if link == "" {
		return nil
	}
	if len(link) > 512 {
		return fmt.Errorf("链接长度不得超过 512")
	}
	switch {
	case strings.HasPrefix(link, "//"):
		return fmt.Errorf("链接 %q 非法", link)
	case strings.HasPrefix(link, "/"):
		if openMode != model.PopupOpenWebview {
			return fmt.Errorf("站内相对路径仅 webview 跳转方式可用")
		}
		return nil
	case strings.HasPrefix(link, "market://"):
		if openMode != model.PopupOpenStore {
			return fmt.Errorf("market:// 链接仅 store 跳转方式可用")
		}
		if len(link) == len("market://") {
			return fmt.Errorf("market:// 链接不完整")
		}
		return nil
	case validHTTPURL(link):
		return nil
	}
	return fmt.Errorf("链接 %q 非法：仅支持 /path、http(s)://、market://", link)
}

func dedupe(in []string, f func(string) string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = f(strings.TrimSpace(v))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func identity(s string) string { return s }

// normalizePopupInput 校验并按能力矩阵归一化入参；不碰数据库。
func normalizePopupInput(in PopupInput) (*normalizedPopup, error) {
	pos := strings.ToUpper(strings.TrimSpace(in.Position))
	caps, ok := popupCaps[pos]
	if !ok {
		return nil, errBadRequest(fmt.Sprintf("位置 %q 非法，应为 P1~P8", in.Position))
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errBadRequest("名称不得为空")
	}
	if runeLen(name) > 64 {
		return nil, errBadRequest("名称不得超过 64 字符")
	}
	if in.Priority < 0 || in.Priority > 999 {
		return nil, errBadRequest("优先级取值范围为 0~999")
	}
	if in.StartAt != nil && in.EndAt != nil && !in.EndAt.After(*in.StartAt) {
		return nil, errBadRequest("结束时间必须晚于开始时间")
	}

	minV, err := ParsePopupVersion(in.MinVersion)
	if err != nil {
		return nil, errBadRequest("最低版本: " + err.Error())
	}
	maxV, err := ParsePopupVersion(in.MaxVersion)
	if err != nil {
		return nil, errBadRequest("最高版本: " + err.Error())
	}
	if minV > 0 && maxV > 0 && minV >= maxV {
		return nil, errBadRequest("最低版本必须小于最高版本")
	}

	userType := strings.ToLower(strings.TrimSpace(in.UserType))
	if userType == "" {
		userType = model.PopupUserAll
	}
	switch userType {
	case model.PopupUserAll, model.PopupUserNew, model.PopupUserOld:
	default:
		return nil, errBadRequest("userType 仅支持 all / new / old")
	}

	openMode := strings.ToLower(strings.TrimSpace(in.OpenMode))
	if openMode == "" {
		openMode = model.PopupOpenWebview
	}
	switch openMode {
	case model.PopupOpenWebview, model.PopupOpenBrowser, model.PopupOpenStore:
	default:
		return nil, errBadRequest("openMode 仅支持 webview / browser / store")
	}

	countries := dedupe(in.Countries, strings.ToUpper)
	for _, c := range countries {
		if !countryRe.MatchString(c) {
			return nil, errBadRequest(fmt.Sprintf("国家码 %q 非法，应为 ISO-3166 二位大写字母", c))
		}
	}
	brandCodes := dedupe(in.BrandCodes, strings.ToLower)
	appIDs := dedupe(in.AppIDs, identity)

	// 行为字段归一化。
	closable := true
	if in.Closable != nil {
		closable = *in.Closable
	}
	if !caps.Force {
		closable = true
	}
	tabEnabled := in.TabEnabled && caps.Tab
	countdown := in.Countdown && caps.Countdown
	maskClosable := in.MaskClosable && caps.MaskClosable
	if caps.Force && !closable {
		// 强制模式：无 X、遮罩无效、返回键不关弹窗；不可收起为便条、无倒计时。
		tabEnabled, maskClosable, countdown = false, false, false
	}
	badge := in.Badge && caps.Badge

	autoplay := 5
	if caps.Autoplay {
		if in.AutoplaySeconds != 0 {
			if in.AutoplaySeconds < 2 || in.AutoplaySeconds > 30 {
				return nil, errBadRequest("轮播间隔 autoplaySeconds 取值范围为 2~30")
			}
			autoplay = in.AutoplaySeconds
		}
	}
	resume := 30
	if caps.Resume {
		if in.ResumeGapMinutes != 0 {
			if in.ResumeGapMinutes < 5 || in.ResumeGapMinutes > 1440 {
				return nil, errBadRequest("回前台触发间隔 resumeGapMinutes 取值范围为 5~1440")
			}
			resume = in.ResumeGapMinutes
		}
	}

	// 便条。
	tabText := strings.TrimSpace(in.TabText)
	tabIcon := strings.TrimSpace(in.TabIconURL)
	if tabEnabled {
		if tabText == "" {
			return nil, errBadRequest("开启便条态时便条文案必填")
		}
		if runeLen(tabText) > 8 {
			return nil, errBadRequest("便条文案不得超过 8 个字符")
		}
		if tabIcon != "" && !validHTTPURL(tabIcon) {
			return nil, errBadRequest("便条图标必须是 http(s) 地址")
		}
		if len(tabIcon) > 512 {
			return nil, errBadRequest("便条图标地址过长（≤512）")
		}
	} else {
		tabText, tabIcon = "", ""
	}

	// 卡片。
	if len(in.Cards) == 0 {
		return nil, errBadRequest("至少需要 1 张卡片")
	}
	if len(in.Cards) > caps.MaxCards {
		return nil, errBadRequest(fmt.Sprintf("最多 %d 张", caps.MaxCards))
	}
	cards := make([]PopupCardInput, 0, len(in.Cards))
	for i, c := range in.Cards {
		n := i + 1
		c.ImageURL = strings.TrimSpace(c.ImageURL)
		c.LinkURL = strings.TrimSpace(c.LinkURL)
		c.ButtonText = strings.TrimSpace(c.ButtonText)
		c.Title = strings.TrimSpace(c.Title)
		c.Description = strings.TrimSpace(c.Description)
		if caps.ImageUnused {
			c.ImageURL = ""
		}
		if c.ImageURL == "" {
			if caps.ImageRequired {
				return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片图片必填", n))
			}
		} else if !validHTTPURL(c.ImageURL) {
			return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片图片必须是 http(s) 地址", n))
		} else if len(c.ImageURL) > 512 {
			return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片图片地址过长（≤512）", n))
		}
		if c.Title == "" && (caps.TitleRequired || (caps.TitleIfNoImg && c.ImageURL == "")) {
			if caps.TitleIfNoImg {
				return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片无图时文案必填", n))
			}
			return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片文案必填", n))
		}
		if runeLen(c.Title) > caps.TitleMaxRunes {
			return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片文案不得超过 %d 个字符", n, caps.TitleMaxRunes))
		}
		if runeLen(c.Description) > 255 {
			return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片描述不得超过 255 字符", n))
		}
		if runeLen(c.ButtonText) > 32 {
			return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片按钮文案不得超过 32 字符", n))
		}
		if err := validatePopupLink(c.LinkURL, openMode); err != nil {
			return nil, errBadRequest(fmt.Sprintf("第 %d 张卡片链接: %s", n, err.Error()))
		}
		c.Sort = i
		cards = append(cards, c)
	}

	p := model.Popup{
		Name: name, Position: pos, Enabled: in.Enabled, Priority: in.Priority,
		BrandCodes: brandCodes, AppIDs: appIDs,
		MinVersionCode: minV, MaxVersionCode: maxV, UserType: userType, Countries: countries,
		TabEnabled: tabEnabled, TabIconURL: tabIcon, TabText: tabText,
		Countdown: countdown, MaskClosable: maskClosable, Closable: closable, OpenMode: openMode,
		AutoplaySeconds: autoplay, ResumeGapMinutes: resume, Badge: badge,
	}
	if in.StartAt != nil {
		t := in.StartAt.UTC()
		p.StartAt = &t
	}
	if in.EndAt != nil {
		t := in.EndAt.UTC()
		p.EndAt = &t
	}
	return &normalizedPopup{Popup: p, Cards: cards}, nil
}

// mergePopupCards 按契约 §2.2 合并卡片：带 id 且属于该弹窗 → 原地更新（ID 不变）；
// 不带 id（或 id 不属于该弹窗）→ 新建；库里存在而请求缺席 → 删除；sort 已由归一化按下标重写。
func mergePopupCards(existing []model.PopupCard, in []PopupCardInput) (upsert []model.PopupCard, deleteIDs []uint64) {
	have := make(map[uint64]bool, len(existing))
	for _, c := range existing {
		have[c.ID] = true
	}
	kept := map[uint64]bool{}
	for i, c := range in {
		card := model.PopupCard{
			Sort: i, ImageURL: c.ImageURL, LinkURL: c.LinkURL, ButtonText: c.ButtonText,
			Title: c.Title, Description: c.Description,
		}
		if c.ID != 0 && have[c.ID] && !kept[c.ID] {
			card.ID = c.ID
			kept[c.ID] = true
		}
		upsert = append(upsert, card)
	}
	for _, c := range existing {
		if !kept[c.ID] {
			deleteIDs = append(deleteIDs, c.ID)
		}
	}
	return upsert, deleteIDs
}

// popupStatus 计算状态：disabled（弹窗关）> ended（end 已过）> scheduled（start 未到）> active。
func popupStatus(p *model.Popup, now time.Time) string {
	switch {
	case !p.Enabled:
		return PopupStatusDisabled
	case p.EndAt != nil && !p.EndAt.After(now):
		return PopupStatusEnded
	case p.StartAt != nil && p.StartAt.After(now):
		return PopupStatusScheduled
	}
	return PopupStatusActive
}

func popupView(p *model.Popup, posEnabled map[string]bool, now time.Time) PopupView {
	cards := make([]PopupCardView, 0, len(p.Cards))
	for _, c := range p.Cards {
		cards = append(cards, PopupCardView{
			ID: c.ID, Sort: c.Sort, ImageURL: c.ImageURL, LinkURL: c.LinkURL,
			ButtonText: c.ButtonText, Title: c.Title, Description: c.Description,
		})
	}
	nz := func(l model.StringList) []string {
		if len(l) == 0 {
			return []string{}
		}
		return []string(l)
	}
	return PopupView{
		ID: p.ID, Name: p.Name, Position: p.Position, Enabled: p.Enabled, Priority: p.Priority,
		StartAt: p.StartAt, EndAt: p.EndAt,
		BrandCodes: nz(p.BrandCodes), AppIDs: nz(p.AppIDs),
		MinVersion: FormatPopupVersion(p.MinVersionCode), MaxVersion: FormatPopupVersion(p.MaxVersionCode),
		UserType: p.UserType, Countries: nz(p.Countries),
		TabEnabled: p.TabEnabled, TabIconURL: p.TabIconURL, TabText: p.TabText,
		Countdown: p.Countdown, MaskClosable: p.MaskClosable, Closable: p.Closable, OpenMode: p.OpenMode,
		AutoplaySeconds: p.AutoplaySeconds, ResumeGapMinutes: p.ResumeGapMinutes, Badge: p.Badge,
		Cards: cards, Status: popupStatus(p, now), PositionEnabled: posEnabled[p.Position],
		CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// ---------- 数据权限（ALL-match，与推送活动同口径） ----------

// popupInScope 判断弹窗的定向是否完全落在 scope 内（列表可见性 / 单体读写闸门）：
//   - 品牌受限：brandCodes 必须非空且全在范围内（空 = 全品牌，只有全量账号可见）；
//   - 渠道受限：appIds 必须非空；
//   - appIds 非空时，每个包都必须 ChannelAllowed（解析不出渠道 = 越界，fail-closed）。
func popupInScope(scope auth.Scope, p *model.Popup, info map[string]repo.ChannelBrandInfo) bool {
	if !scope.AllBrands {
		if len(p.BrandCodes) == 0 {
			return false
		}
		for _, b := range p.BrandCodes {
			if !scope.BrandAllowed(b) {
				return false
			}
		}
	}
	if !scope.AllChannels && len(p.AppIDs) == 0 {
		return false
	}
	for _, id := range p.AppIDs {
		row, ok := info[id]
		if !ok || !scope.ChannelAllowed(row.BrandCode, row.ChannelID) {
			return false
		}
	}
	return true
}

func scopeIsFull(scope auth.Scope) bool { return scope.AllBrands && scope.AllChannels }

// popupChannelInfo 批量解析若干弹窗定向里出现的 appId（scope 全量时跳过查询）。
func (s *Service) popupChannelInfo(ctx context.Context, scope auth.Scope, list []model.Popup) (map[string]repo.ChannelBrandInfo, error) {
	if scopeIsFull(scope) {
		return nil, nil
	}
	set := map[string]bool{}
	for i := range list {
		for _, id := range list[i].AppIDs {
			set[id] = true
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return s.repo.ChannelBrandsByApplicationIDs(ctx, ids)
}

// assertPopupTargetInScope 保存时校验定向写入权限，并校验品牌 / 渠道真实存在。
func (s *Service) assertPopupTargetInScope(ctx context.Context, scope auth.Scope, brandCodes, appIDs []string) error {
	if !scope.AllBrands {
		if len(brandCodes) == 0 {
			return errForbidden("你的数据范围受限，必须选择定向品牌")
		}
		for _, b := range brandCodes {
			if !scope.BrandAllowed(b) {
				return errForbidden(fmt.Sprintf("品牌 %q 不在你的数据范围内", b))
			}
		}
	}
	if !scope.AllChannels && len(appIDs) == 0 {
		return errForbidden("你的数据范围限定了渠道，必须选择定向渠道包")
	}
	if len(appIDs) > 0 {
		if err := s.assertAppIDsInScope(ctx, scope, appIDs); err != nil {
			return err
		}
	}
	// 存在性校验（越界已在上面拒绝，到这里只可能是全量范围或范围内的合法 id）。
	if len(brandCodes) > 0 {
		brands, err := s.repo.ListBrands(ctx)
		if err != nil {
			return err
		}
		known := map[string]bool{}
		for _, b := range brands {
			if b.SupportsChannels {
				known[b.Code] = true
			}
		}
		for _, b := range brandCodes {
			if !known[b] {
				return errBadRequest(fmt.Sprintf("品牌 %q 不存在或不支持渠道包", b))
			}
		}
	}
	if len(appIDs) > 0 {
		info, err := s.repo.ChannelBrandsByApplicationIDs(ctx, appIDs)
		if err != nil {
			return err
		}
		for _, id := range appIDs {
			if _, ok := info[id]; !ok {
				return errBadRequest(fmt.Sprintf("渠道包 %q 不存在", id))
			}
		}
	}
	return nil
}

// ---------- Service 方法 ----------

func (s *Service) positionEnabledMap(ctx context.Context) (map[string]bool, error) {
	list, err := s.repo.ListPopupPositions(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(list))
	for _, p := range list {
		m[p.Code] = p.Enabled
	}
	return m, nil
}

// ListPopupPositions 8 个位置开关。
func (s *Service) ListPopupPositions(ctx context.Context) ([]PopupPositionView, error) {
	list, err := s.repo.ListPopupPositions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PopupPositionView, 0, len(list))
	for _, p := range list {
		out = append(out, PopupPositionView{Code: p.Code, Name: PopupPositionName(p.Code), Enabled: p.Enabled, UpdatedAt: p.UpdatedAt, UpdatedBy: p.UpdatedBy})
	}
	return out, nil
}

// SetPopupPosition 开关位置。位置开关是全局的，调用者必须是全量数据范围，否则 403。
func (s *Service) SetPopupPosition(ctx context.Context, scope auth.Scope, code string, enabled bool, by string) (*PopupPositionView, error) {
	if !scopeIsFull(scope) {
		return nil, errForbidden("位置开关是全局设置，仅全量数据范围的账号可操作")
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if _, ok := popupCaps[code]; !ok {
		return nil, errNotFound("位置不存在")
	}
	p, err := s.repo.SetPopupPosition(ctx, code, enabled, by)
	if err != nil {
		return nil, err
	}
	s.popup.invalidate()
	return &PopupPositionView{Code: p.Code, Name: PopupPositionName(p.Code), Enabled: p.Enabled, UpdatedAt: p.UpdatedAt, UpdatedBy: p.UpdatedBy}, nil
}

// ListPopups 弹窗列表：position/keyword 走 SQL，brand/status 与数据权限在内存过滤。
// brand 过滤口径：定向包含该品牌，或未限定品牌（= 全品牌）的弹窗都算命中。
func (s *Service) ListPopups(ctx context.Context, scope auth.Scope, q PopupListQuery) ([]PopupView, error) {
	pos := strings.ToUpper(strings.TrimSpace(q.Position))
	list, err := s.repo.ListPopups(ctx, pos, strings.TrimSpace(q.Keyword))
	if err != nil {
		return nil, err
	}
	info, err := s.popupChannelInfo(ctx, scope, list)
	if err != nil {
		return nil, err
	}
	posEnabled, err := s.positionEnabledMap(ctx)
	if err != nil {
		return nil, err
	}
	brand := strings.ToLower(strings.TrimSpace(q.Brand))
	status := strings.TrimSpace(q.Status)
	now := time.Now()
	out := make([]PopupView, 0, len(list))
	for i := range list {
		p := &list[i]
		if !scopeIsFull(scope) && !popupInScope(scope, p, info) {
			continue
		}
		if brand != "" && len(p.BrandCodes) > 0 && !containsStr(p.BrandCodes, brand) {
			continue
		}
		v := popupView(p, posEnabled, now)
		if status != "" && v.Status != status {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

func containsStr(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// loadScopedPopup 取弹窗并做数据权限闸门；越界与不存在一样返回 404（不泄露存在性）。
func (s *Service) loadScopedPopup(ctx context.Context, scope auth.Scope, id uint64) (*model.Popup, error) {
	p, err := s.repo.GetPopup(ctx, id)
	if err != nil {
		return nil, errNotFound("弹窗不存在")
	}
	if !scopeIsFull(scope) {
		info, err := s.popupChannelInfo(ctx, scope, []model.Popup{*p})
		if err != nil {
			return nil, err
		}
		if !popupInScope(scope, p, info) {
			return nil, errNotFound("弹窗不存在")
		}
	}
	return p, nil
}

// GetPopup 弹窗详情。
func (s *Service) GetPopup(ctx context.Context, scope auth.Scope, id uint64) (*PopupView, error) {
	p, err := s.loadScopedPopup(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	posEnabled, err := s.positionEnabledMap(ctx)
	if err != nil {
		return nil, err
	}
	v := popupView(p, posEnabled, time.Now())
	return &v, nil
}

// CreatePopup 新建弹窗。
func (s *Service) CreatePopup(ctx context.Context, scope auth.Scope, in PopupInput, createdBy string) (*PopupView, error) {
	n, err := normalizePopupInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.assertPopupTargetInScope(ctx, scope, n.Popup.BrandCodes, n.Popup.AppIDs); err != nil {
		return nil, err
	}
	p := n.Popup
	p.CreatedBy = createdBy
	if err := s.repo.SavePopup(ctx, &p, func(existing []model.PopupCard) ([]model.PopupCard, []uint64) {
		return mergePopupCards(existing, n.Cards)
	}); err != nil {
		return nil, err
	}
	s.popup.invalidate()
	return s.GetPopup(ctx, scope, p.ID)
}

// UpdatePopup 全量更新弹窗（卡片按 id 合并）。
func (s *Service) UpdatePopup(ctx context.Context, scope auth.Scope, id uint64, in PopupInput) (*PopupView, error) {
	old, err := s.loadScopedPopup(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	n, err := normalizePopupInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.assertPopupTargetInScope(ctx, scope, n.Popup.BrandCodes, n.Popup.AppIDs); err != nil {
		return nil, err
	}
	p := n.Popup
	p.ID = old.ID
	p.CreatedBy = old.CreatedBy
	p.CreatedAt = old.CreatedAt
	if err := s.repo.SavePopup(ctx, &p, func(existing []model.PopupCard) ([]model.PopupCard, []uint64) {
		return mergePopupCards(existing, n.Cards)
	}); err != nil {
		return nil, err
	}
	s.popup.invalidate()
	return s.GetPopup(ctx, scope, p.ID)
}

// SetPopupEnabled 行内快速开关。
func (s *Service) SetPopupEnabled(ctx context.Context, scope auth.Scope, id uint64, enabled bool) (*PopupView, error) {
	if _, err := s.loadScopedPopup(ctx, scope, id); err != nil {
		return nil, err
	}
	if err := s.repo.SetPopupEnabled(ctx, id, enabled); err != nil {
		return nil, err
	}
	s.popup.invalidate()
	return s.GetPopup(ctx, scope, id)
}

// DeletePopup 软删。
func (s *Service) DeletePopup(ctx context.Context, scope auth.Scope, id uint64) error {
	if _, err := s.loadScopedPopup(ctx, scope, id); err != nil {
		return err
	}
	if err := s.repo.DeletePopup(ctx, id); err != nil {
		return err
	}
	s.popup.invalidate()
	return nil
}

// ---------- 时区 ----------

var popupLocCache sync.Map // name → *time.Location

// popupLocation 解析 POPUP_TZ；加载失败时 Asia/Manila 回落固定 +8，其余回落 UTC。
func popupLocation(name string) *time.Location {
	if name == "" {
		name = "Asia/Manila"
	}
	if v, ok := popupLocCache.Load(name); ok {
		return v.(*time.Location)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		if name == "Asia/Manila" {
			loc = time.FixedZone("Asia/Manila", 8*3600)
		} else {
			loc = time.UTC
		}
	}
	popupLocCache.Store(name, loc)
	return loc
}

func (s *Service) popupLoc() *time.Location { return popupLocation(s.cfg.PopupTZ) }
