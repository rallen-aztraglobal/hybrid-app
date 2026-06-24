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
	BrandCode  string `json:"brandCode" validate:"required"`
	FlavorName string `json:"flavorName" validate:"required"`
	PalCode    string `json:"palCode" validate:"required"`
	AppName    string `json:"appName" validate:"required"`
	Remark     string `json:"remark"`
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
		return nil, errBadRequest(fmt.Sprintf("flavorName %q 非法（仅允许字母和数字）", newFlavor))
	}
	newAppID := brand.DeriveApplicationID(newFlavor)

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
	// flavor 仅允许字母数字（与 Gradle flavor 命名一致，避免生成非法 task 名 / 非法包名后缀）。
	if !looksLikeFlavor(in.FlavorName) {
		return errBadRequest(fmt.Sprintf("flavorName %q 非法（仅允许字母和数字）", in.FlavorName))
	}
	return nil
}

func looksLikeFlavor(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}
	// 段首不能是数字（Java 包名规则：appId 末段 == flavor，需可作合法包名片段）。
	if s[0] >= '0' && s[0] <= '9' {
		return false
	}
	return true
}
