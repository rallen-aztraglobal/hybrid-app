// Package service — google-services.json 按品牌存取（ADR-0012 §3 + §5 CLI 分发）。
// google-services.json 是非机密（随 APK 分发），可存对象存储、经公开端点下发。
// service account 私钥才是机密，仍只走 env/挂载（ADR-0008）。
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hybrid-app/server/internal/storage"
)

// validBrands 当前支持的存储键：三个品牌 ap/bp/gp，外加 gp 溢出项目 gp2。
// gp2 不是真品牌，是 gp 超 Firebase 每项目 30 App 上限后的第二个 Firebase 项目（hybrid-gp2），
// 其 google-services.json 既供 gp2 那批 flavor 构建分发，也供后端发送路由判定（见 fcm_routing.go）。
var validBrands = map[string]bool{"ap": true, "bp": true, "gp": true, "gp2": true}

// googleServicesKey 返回对象存储 key（fcm/<brand>/google-services.json）。
func googleServicesKey(brand string) string {
	return fmt.Sprintf("fcm/%s/google-services.json", brand)
}

// GetGoogleServices 从 Storage 读取该品牌的 google-services.json 内容。
// 对象不存在时返回 storage.ErrNotFound；调用方映射为 HTTP 404。
// brand 非法时返回业务错误（400）。
func (s *Service) GetGoogleServices(ctx context.Context, brand string) (io.ReadCloser, error) {
	if !validBrands[brand] {
		return nil, errBadRequest(fmt.Sprintf("brand 非法：%q，仅支持 ap/bp/gp", brand))
	}
	rc, err := s.storage.Get(ctx, googleServicesKey(brand))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, errNotFound(fmt.Sprintf("品牌 %s 的 google-services.json 尚未上传", brand))
		}
		return nil, fmt.Errorf("读取 google-services.json 失败: %w", err)
	}
	return rc, nil
}

// GoogleServicesUploadResult POST 上传后的响应体。
type GoogleServicesUploadResult struct {
	Brand       string `json:"brand"`
	Stored      bool   `json:"stored"`
	ClientCount int    `json:"clientCount"`
}

// UploadGoogleServices 校验并存储品牌的 google-services.json。
// 接受 io.Reader（multipart file 或 raw body），存到 Storage。
// 基本校验：合法 JSON + 含 project_info + 含 client 数组（非空）。
func (s *Service) UploadGoogleServices(ctx context.Context, brand string, r io.Reader) (*GoogleServicesUploadResult, error) {
	if !validBrands[brand] {
		return nil, errBadRequest(fmt.Sprintf("brand 非法：%q，仅支持 ap/bp/gp", brand))
	}

	// 读入全部内容（google-services.json 一般 < 50 KB）。
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取上传内容失败: %w", err)
	}

	// 校验：是否为合法 JSON。
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errBadRequest(fmt.Sprintf("不是合法的 JSON: %v", err))
	}

	// 校验：含 project_info。
	if _, ok := raw["project_info"]; !ok {
		return nil, errBadRequest("google-services.json 缺少 project_info 字段")
	}

	// 校验：含 client 数组且非空。
	clientRaw, ok := raw["client"]
	if !ok {
		return nil, errBadRequest("google-services.json 缺少 client 字段")
	}
	var clients []json.RawMessage
	if err := json.Unmarshal(clientRaw, &clients); err != nil {
		return nil, errBadRequest(fmt.Sprintf("client 字段应为数组: %v", err))
	}
	if len(clients) == 0 {
		return nil, errBadRequest("google-services.json 的 client 数组为空")
	}

	// 存入 Storage。
	key := googleServicesKey(brand)
	if _, err := s.storage.Put(ctx, key, bytes.NewReader(data), int64(len(data)),
		"application/json"); err != nil {
		return nil, fmt.Errorf("存储 google-services.json 失败: %w", err)
	}

	return &GoogleServicesUploadResult{
		Brand:       brand,
		Stored:      true,
		ClientCount: len(clients),
	}, nil
}

// validateBrand 仅供其他 service 方法内联复用。
func validateBrand(brand string) error {
	if strings.TrimSpace(brand) == "" {
		return errBadRequest("brand 参数不得为空")
	}
	if !validBrands[brand] {
		return errBadRequest(fmt.Sprintf("brand 非法：%q，仅支持 ap/bp/gp", brand))
	}
	return nil
}
