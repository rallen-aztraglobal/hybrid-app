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
type CreateChannelInput struct {
	BrandCode  string  `json:"brandCode" validate:"required"`
	FlavorName string  `json:"flavorName" validate:"required"`
	PalCode    string  `json:"palCode" validate:"required"`
	AppName    string  `json:"appName" validate:"required"`
	Remark     string  `json:"remark"`
	StoreID    *uint64 `json:"storeId"`
}

// UpdateChannelInput 修改渠道入参（指针字段表示可选更新）。
// 无 applicationId：改 flavor 会自动重新派生 appId；palCode 可自由编辑（不再唯一）。
type UpdateChannelInput struct {
	FlavorName *string `json:"flavorName"`
	PalCode    *string `json:"palCode"`
	AppName    *string `json:"appName"`
	Status     *string `json:"status"`
	Remark     *string `json:"remark"`
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
		StoreID:         in.StoreID,
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
	if in.Status != nil {
		st := *in.Status
		if st != model.ChannelEnabled && st != model.ChannelDisabled && st != model.ChannelArchived {
			return nil, errBadRequest("status 非法（enabled/disabled/archived）")
		}
		ch.Status = st
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
}

func (in *CreateChannelInput) validate() error {
	if in.BrandCode == "" || in.FlavorName == "" || in.PalCode == "" || in.AppName == "" {
		return errBadRequest("brandCode / flavorName / palCode / appName 均为必填")
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
