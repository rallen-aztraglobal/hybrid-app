package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// CreateChannelInput 新增渠道入参。
// applicationId 不在入参里：按 ADR-0009 由 <品牌包前缀>.<flavor> 派生，不信任外部传入。
//
// AdjustAppToken/AdjustEvents（ADR-0013）：JSON 字段名 adjustAppToken / adjustEvents，
// 与本结构体其余字段一致走 camelCase。二者均可选——空 App Token 表示该渠道未绑定 Adjust（合法状态）。
type CreateChannelInput struct {
	BrandCode      string            `json:"brandCode" validate:"required"`
	FlavorName     string            `json:"flavorName" validate:"required"`
	PalCode        string            `json:"palCode" validate:"required"`
	AppName        string            `json:"appName" validate:"required"`
	Remark         string            `json:"remark"`
	LiveVersion    string            `json:"liveVersion"`
	StoreID        *uint64           `json:"storeId"`
	AdjustAppToken string            `json:"adjustAppToken"`
	AdjustEvents   map[string]string `json:"adjustEvents"`
}

// UpdateChannelInput 修改渠道入参（指针字段表示可选更新）。
// 无 applicationId：改 flavor 会自动重新派生 appId；palCode 可自由编辑（不再唯一）。
//
// AdjustAppToken/AdjustEvents 同样是指针字段（未传 = 不改动）：
//   - AdjustAppToken 传空字符串 → 显式解绑 Adjust；
//   - AdjustEvents 传 nil/空对象 → 清空事件表。
//
// LiveVersion（线上版本号，人工备忘）：传空字符串 → 清空；未传 → 不改动。
// 卡片上的就地编辑只发这一个字段，其余字段保持原值。
type UpdateChannelInput struct {
	FlavorName     *string            `json:"flavorName"`
	PalCode        *string            `json:"palCode"`
	AppName        *string            `json:"appName"`
	Status         *string            `json:"status"`
	Remark         *string            `json:"remark"`
	LiveVersion    *string            `json:"liveVersion"`
	AdjustAppToken *string            `json:"adjustAppToken"`
	AdjustEvents   *map[string]string `json:"adjustEvents"`
}

// ListChannels 列表查询，并为每条渠道填充 latestApkUrl（按 flavor 取最近成功构建产物，ADR-0008）。
func (s *Service) ListChannels(ctx context.Context, f repo.ChannelFilter) ([]model.Channel, int64, error) {
	list, total, err := s.repo.ListChannels(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	if len(list) > 0 {
		flavors := make([]string, len(list))
		for i := range list {
			flavors[i] = list[i].FlavorName
		}
		latest, err := s.repo.LatestArtifactURLsByFlavors(ctx, flavors)
		if err != nil {
			return nil, 0, err
		}
		for i := range list {
			list[i].LatestApkURL = latest[list[i].FlavorName]
		}
	}
	return list, total, nil
}

// GetChannel 详情。
func (s *Service) GetChannel(ctx context.Context, id uint64) (*model.Channel, error) {
	return s.repo.GetChannel(ctx, id)
}

// CreateChannel 新增渠道。applicationId 派生自品牌包前缀 + flavor；唯一性以 applicationId 与 (brand,flavor) 为准（ADR-0009）。
func (s *Service) CreateChannel(ctx context.Context, in CreateChannelInput) (*model.Channel, error) {
	in.normalize()
	if err := in.validate(); err != nil {
		return nil, err
	}
	adjustToken, err := normalizeAdjustAppToken(in.AdjustAppToken)
	if err != nil {
		return nil, err
	}
	if err := validateAdjustEvents(in.AdjustEvents); err != nil {
		return nil, err
	}

	brand, err := s.repo.GetBrandByCode(ctx, in.BrandCode)
	if err != nil {
		return nil, errBadRequest(fmt.Sprintf("品牌 %q 不存在", in.BrandCode))
	}

	// 若指定了应用商店，校验其存在且已启用；并要求 flavor 以 "_"+store.Code 结尾，
	// 保证派生出的 applicationId 分段与所选商店一致。
	if in.StoreID != nil {
		store, err := s.repo.GetStoreByID(ctx, *in.StoreID)
		if err != nil {
			return nil, errBadRequest(fmt.Sprintf("应用商店 id=%d 不存在", *in.StoreID))
		}
		if store.Status != model.StoreEnabled {
			return nil, errBadRequest(fmt.Sprintf("应用商店 %q 已停用，不能新建渠道", store.Name))
		}
		if !strings.HasSuffix(in.FlavorName, "_"+store.Code) {
			return nil, errBadRequest(fmt.Sprintf("flavor 后缀与所选商店不一致（应以 _%s 结尾）", store.Code))
		}
	} else if strings.Contains(in.FlavorName, "_") {
		// 下划线仅用于商店后缀分段；未选商店时不应出现，避免派生出无商店归属的点号包名。
		return nil, errBadRequest("未选择应用商店时 flavor 不能包含下划线")
	}

	// applicationId 派生（事实来源），不信任输入。
	appID := brand.DeriveApplicationID(in.FlavorName)

	// 唯一性校验：applicationId 与 (brand, flavor)；pal_code 不再查重（ADR-0009）。
	conflicts, err := s.repo.UniqueConflicts(ctx, repo.UniqueCheck{
		ApplicationID: appID,
		BrandID:       brand.ID,
		FlavorName:    in.FlavorName,
	})
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 {
		return nil, errConflict("以下字段已存在，必须唯一: " + strings.Join(conflicts, ", "))
	}

	ch := &model.Channel{
		BrandID:         brand.ID,
		FlavorName:      in.FlavorName,
		ApplicationID:   appID,
		PalCode:         in.PalCode,
		AppName:         in.AppName,
		Status:          model.ChannelEnabled,
		UseBrandDomains: true,
		Remark:          in.Remark,
		LiveVersion:     in.LiveVersion,
		StoreID:         in.StoreID,
		AdjustAppToken:  adjustToken,
		AdjustEvents:    model.AdjustEvents(in.AdjustEvents),
	}
	if err := s.repo.CreateChannel(ctx, ch); err != nil {
		return nil, err
	}
	// 新渠道默认继承品牌域名：立即生成其 CDN 配置快照，保证 /api/app/config 一上线即有抗封兜底（ADR-0002）。
	// 解析键已改为 applicationId（ADR-0009）。
	s.regenSnapshot(ctx, ch.ApplicationID)
	return s.repo.GetChannel(ctx, ch.ID)
}

// UpdateChannel 修改渠道。改 flavor 会重新派生 applicationId 并校验唯一；palCode 自由编辑。
func (s *Service) UpdateChannel(ctx context.Context, id uint64, in UpdateChannelInput) (*model.Channel, error) {
	ch, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	brand, err := s.repo.GetBrandByID(ctx, ch.BrandID)
	if err != nil {
		return nil, err
	}

	// 计算变更后的 flavor 与（派生）appId。
	newFlavor := ch.FlavorName
	if in.FlavorName != nil {
		newFlavor = strings.TrimSpace(*in.FlavorName)
	}
	if newFlavor == "" {
		return nil, errBadRequest("flavorName 不能为空")
	}
	if !looksLikeFlavor(newFlavor) {
		return nil, errBadRequest(fmt.Sprintf("flavorName %q 非法（仅允许字母、数字，可用下划线分段）", newFlavor))
	}
	newAppID := brand.DeriveApplicationID(newFlavor)

	// flavor 变更时，保持 flavor 与所属商店的一致性（前端编辑态已禁用 flavor/商店，此处兜底直连 API 的场景）。
	if newFlavor != ch.FlavorName {
		if ch.StoreID != nil {
			store, err := s.repo.GetStoreByID(ctx, *ch.StoreID)
			if err != nil {
				return nil, errBadRequest(fmt.Sprintf("渠道关联的应用商店 id=%d 不存在", *ch.StoreID))
			}
			if !strings.HasSuffix(newFlavor, "_"+store.Code) {
				return nil, errBadRequest(fmt.Sprintf("flavor 后缀与所属商店不一致（应以 _%s 结尾）", store.Code))
			}
		} else if strings.Contains(newFlavor, "_") {
			return nil, errBadRequest("未关联应用商店的渠道 flavor 不能包含下划线")
		}
	}

	// 仅当 flavor（从而 appId）实际变化时做唯一性检查（排除自身）。
	if newFlavor != ch.FlavorName {
		conflicts, err := s.repo.UniqueConflicts(ctx, repo.UniqueCheck{
			ApplicationID: newAppID,
			BrandID:       ch.BrandID,
			FlavorName:    newFlavor,
			ExcludeID:     ch.ID,
		})
		if err != nil {
			return nil, err
		}
		if len(conflicts) > 0 {
			return nil, errConflict("以下字段已存在，必须唯一: " + strings.Join(conflicts, ", "))
		}
	}

	oldAppID := ch.ApplicationID
	ch.FlavorName = newFlavor
	ch.ApplicationID = newAppID
	if in.PalCode != nil {
		pal := strings.TrimSpace(*in.PalCode)
		if pal == "" {
			return nil, errBadRequest("palCode 不能为空")
		}
		ch.PalCode = pal
	}
	if in.AppName != nil {
		ch.AppName = strings.TrimSpace(*in.AppName)
	}
	if in.Remark != nil {
		ch.Remark = *in.Remark
	}
	if in.LiveVersion != nil {
		v, err := normalizeLiveVersion(*in.LiveVersion)
		if err != nil {
			return nil, err
		}
		ch.LiveVersion = v
	}
	if in.Status != nil {
		st := *in.Status
		if st != model.ChannelEnabled && st != model.ChannelDisabled && st != model.ChannelArchived {
			return nil, errBadRequest("status 非法（enabled/disabled/archived）")
		}
		ch.Status = st
	}
	// AdjustAppToken：传空字符串表示显式解绑 Adjust（ADR-0013）。
	if in.AdjustAppToken != nil {
		tok, err := normalizeAdjustAppToken(*in.AdjustAppToken)
		if err != nil {
			return nil, err
		}
		ch.AdjustAppToken = tok
	}
	// AdjustEvents：传 nil/空对象清空事件表。
	if in.AdjustEvents != nil {
		if err := validateAdjustEvents(*in.AdjustEvents); err != nil {
			return nil, err
		}
		if len(*in.AdjustEvents) == 0 {
			ch.AdjustEvents = nil
		} else {
			ch.AdjustEvents = model.AdjustEvents(*in.AdjustEvents)
		}
	}

	if err := s.repo.UpdateChannel(ctx, ch); err != nil {
		return nil, err
	}
	// applicationId 是解析键：若它变了，旧快照置为「已归档」（避免旧 APK 继续命中），并生成新键快照。
	if oldAppID != ch.ApplicationID {
		s.archiveSnapshot(ctx, oldAppID)
	}
	s.regenSnapshot(ctx, ch.ApplicationID)
	return s.repo.GetChannel(ctx, ch.ID)
}

// ArchiveChannel 软删除（置 archived），保留归因历史。
// 归档后该 applicationId 的 /api/app/config 返回 404，CDN 快照也置为「已归档」标记，避免旧域名继续被取用。
func (s *Service) ArchiveChannel(ctx context.Context, id uint64) error {
	ch, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateChannelFields(ctx, id, map[string]any{"status": model.ChannelArchived}); err != nil {
		return err
	}
	s.archiveSnapshot(ctx, ch.ApplicationID)
	return nil
}

func (in *CreateChannelInput) normalize() {
	in.BrandCode = strings.TrimSpace(in.BrandCode)
	in.FlavorName = strings.TrimSpace(in.FlavorName)
	in.PalCode = strings.TrimSpace(in.PalCode)
	in.AppName = strings.TrimSpace(in.AppName)
	in.LiveVersion = strings.TrimSpace(in.LiveVersion)
}

func (in *CreateChannelInput) validate() error {
	if in.BrandCode == "" || in.FlavorName == "" || in.PalCode == "" || in.AppName == "" {
		return errBadRequest("brandCode / flavorName / palCode / appName 均为必填")
	}
	if _, err := normalizeLiveVersion(in.LiveVersion); err != nil {
		return err
	}
	// flavor 仅允许字母数字与下划线分段（下划线用于商店后缀，如 <base>_<storeCode>）。
	if !looksLikeFlavor(in.FlavorName) {
		return errBadRequest(fmt.Sprintf("flavorName %q 非法（仅允许字母、数字，可用下划线分段）", in.FlavorName))
	}
	return nil
}

// looksLikeFlavor 校验 flavor 命名：按 "_" 分段，每段须匹配 ^[a-zA-Z][a-zA-Z0-9]*$
// （即段首字母、其余字母数字），不允许空段（含前导/尾随/连续下划线）。
// 下划线用于承载商店后缀（<base>_<storeCode>），派生 appId 时会被转换为 "."（见 Brand.DeriveApplicationID）。
// 老数据（无下划线）天然只有一段，规则退化为原先「仅字母数字、段首非数字」，向后兼容。
func looksLikeFlavor(s string) bool {
	if s == "" {
		return false
	}
	segs := strings.Split(s, "_")
	for _, seg := range segs {
		if !looksLikeFlavorSegment(seg) {
			return false
		}
	}
	return true
}

// looksLikeFlavorSegment 校验单个分段：^[a-zA-Z][a-zA-Z0-9]*$。
func looksLikeFlavorSegment(seg string) bool {
	if seg == "" {
		return false
	}
	for i := 0; i < len(seg); i++ {
		ch := seg[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}
	if seg[0] >= '0' && seg[0] <= '9' {
		return false
	}
	return true
}

// maxLiveVersionLen 与 channel.live_version 列定义 VARCHAR(32) 对齐。
const maxLiveVersionLen = 32

// normalizeLiveVersion 规范化线上版本号：去首尾空白；空串合法（= 未记录/清空）；超长拒绝。
// 不校验格式——它只是人工备忘，运营想写 "1.2.3" 还是 "1.2.3(商店审核中)" 都行。
func normalizeLiveVersion(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if len(v) > maxLiveVersionLen {
		return "", errBadRequest(fmt.Sprintf("liveVersion 长度不能超过 %d", maxLiveVersionLen))
	}
	return v, nil
}

// maxAdjustAppTokenLen 与 channel.adjust_app_token 列定义 VARCHAR(64) 对齐（ADR-0013）。
const maxAdjustAppTokenLen = 64

// normalizeAdjustAppToken 校验并规范化 Adjust App Token：去空白；超长拒绝；
// 空字符串是合法值（表示未绑定/解绑），归一化为 nil（对应 DB NULL）。
func normalizeAdjustAppToken(raw string) (*string, error) {
	tok := strings.TrimSpace(raw)
	if tok == "" {
		return nil, nil
	}
	if len(tok) > maxAdjustAppTokenLen {
		return nil, errBadRequest(fmt.Sprintf("adjustAppToken 长度不能超过 %d", maxAdjustAppTokenLen))
	}
	return &tok, nil
}

// validateAdjustEvents 校验 adjustEvents 的形状：要么为空，要么是 string→string 的 JSON 对象
// （类型层面已由 JSON 反序列化到 map[string]string 保证；这里补业务侧校验：key/value 不允许空白）。
// server 不解析上传的 Adjust CSV，也不理解具体的事件 name/token 含义，只做这层最基础的形状校验。
func validateAdjustEvents(m map[string]string) error {
	for k, v := range m {
		if strings.TrimSpace(k) == "" {
			return errBadRequest("adjustEvents 的事件名（key）不能为空")
		}
		if strings.TrimSpace(v) == "" {
			return errBadRequest(fmt.Sprintf("adjustEvents[%q] 的 token 不能为空", k))
		}
	}
	return nil
}
