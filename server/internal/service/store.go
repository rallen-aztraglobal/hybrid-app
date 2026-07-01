package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hybrid-app/server/internal/model"
)

// storeCodeRe 应用商店 code 规范：小写字母开头，其余小写字母数字，最长 16（与列宽一致）。
var storeCodeRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// CreateStoreInput 新增应用商店入参。
type CreateStoreInput struct {
	Code string `json:"code" validate:"required"`
	Name string `json:"name" validate:"required"`
	Sort int    `json:"sort"`
}

// UpdateStoreInput 修改应用商店入参（指针字段表示可选更新）。
// 不含 Code：code 一旦创建即不可改（已进入 applicationId 分段，改动会破坏存量包的解析）。
type UpdateStoreInput struct {
	Name   *string `json:"name"`
	Sort   *int    `json:"sort"`
	Status *string `json:"status"`
}

// ListStores 返回全部应用商店（含 disabled），按 sort 升序。
func (s *Service) ListStores(ctx context.Context) ([]model.Store, error) {
	return s.repo.ListStores(ctx)
}

// CreateStore 新增应用商店。code 规范化为小写，需匹配 ^[a-z][a-z0-9]*$，且全局唯一。
func (s *Service) CreateStore(ctx context.Context, in CreateStoreInput) (*model.Store, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return nil, errBadRequest("code / name 均为必填")
	}
	if !storeCodeRe.MatchString(code) {
		return nil, errBadRequest(fmt.Sprintf("code %q 非法（仅允许小写字母开头的小写字母数字）", code))
	}
	if len(code) > 16 {
		return nil, errBadRequest("code 长度不能超过 16")
	}
	if _, err := s.repo.GetStoreByCode(ctx, code); err == nil {
		return nil, errConflict(fmt.Sprintf("应用商店 code %q 已存在", code))
	}

	st := &model.Store{Code: code, Name: name, Sort: in.Sort, Status: model.StoreEnabled}
	if err := s.repo.CreateStore(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// UpdateStore 修改应用商店。仅允许改 name/sort/status；code 不可改。
func (s *Service) UpdateStore(ctx context.Context, id uint64, in UpdateStoreInput) (*model.Store, error) {
	st, err := s.repo.GetStoreByID(ctx, id)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, errBadRequest("name 不能为空")
		}
		fields["name"] = name
	}
	if in.Sort != nil {
		fields["sort"] = *in.Sort
	}
	if in.Status != nil {
		if *in.Status != model.StoreEnabled && *in.Status != model.StoreDisabled {
			return nil, errBadRequest("status 非法（enabled/disabled）")
		}
		fields["status"] = *in.Status
	}
	if len(fields) == 0 {
		return st, nil
	}
	if err := s.repo.UpdateStoreFields(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetStoreByID(ctx, id)
}

// DeleteStore 删除应用商店。若已被渠道引用则拒绝（应改为停用）。
func (s *Service) DeleteStore(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetStoreByID(ctx, id); err != nil {
		return err
	}
	n, err := s.repo.CountChannelsByStore(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errConflict("该商店已被渠道使用，请改为停用")
	}
	return s.repo.DeleteStore(ctx, id)
}
