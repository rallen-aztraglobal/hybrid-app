// Package service — FCM 发送的 applicationId → 路由键映射（gp 拆分专用）。
//
// 背景：Firebase 每个项目最多 30 个 App，gp 品牌 42 个渠道装不下，被拆成 hybrid-gp(30)+hybrid-gp2(12)。
// 发送时不能再只按品牌路由，需按 applicationId 判定其所属 Firebase 项目：
//   - 命中 gp2 项目的 google-services.json（fcm/gp2/...）→ 路由键 "gp2"；
//   - 其余 → 退回品牌 code（ap/bp/gp，行为不变）。
// 数据源是已上传到 Storage 的 google-services.json（package_name 即 applicationId），自维护、无需改库表。
package service

import (
	"context"
	"encoding/json"
	"io"
)

// splitOverflowKeys 列出「溢出 Firebase 项目」的路由键。其 fcm/<key>/google-services.json
// 里出现的 package_name 都路由到该键。当前仅 gp2；未来再溢出（gp3…）追加即可。
var splitOverflowKeys = []string{"gp2"}

// gsClientName 仅解析 google-services.json 里我们需要的 package_name。
type gsClientName struct {
	Client []struct {
		ClientInfo struct {
			AndroidClientInfo struct {
				PackageName string `json:"package_name"`
			} `json:"android_client_info"`
		} `json:"client_info"`
	} `json:"client"`
}

// buildFCMAppIndex 扫描各溢出项目的 google-services.json，构建 applicationId → 路由键 映射。
// 某溢出项目的 json 未上传（404）时跳过（其包届时退回品牌路由）；不因缺失报错。
func (s *Service) buildFCMAppIndex(ctx context.Context) map[string]string {
	index := map[string]string{}
	for _, key := range splitOverflowKeys {
		rc, err := s.storage.Get(ctx, googleServicesKey(key))
		if err != nil {
			// 未上传或读失败：该溢出项目暂不参与路由（其包退回品牌，命中已配置品牌则正常发）。
			continue
		}
		pkgs, perr := parseGoogleServicesPackages(rc)
		_ = rc.Close()
		if perr != nil {
			continue
		}
		for _, pkg := range pkgs {
			if pkg != "" {
				index[pkg] = key
			}
		}
	}
	return index
}

// parseGoogleServicesPackages 从 google-services.json 提取全部 package_name。
func parseGoogleServicesPackages(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var g gsClientName
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(g.Client))
	for _, c := range g.Client {
		out = append(out, c.ClientInfo.AndroidClientInfo.PackageName)
	}
	return out, nil
}

// resolveFCMKey 计算某 token 的 FCM 路由键：命中溢出索引则用之，否则退回品牌 code。
func resolveFCMKey(appIndex map[string]string, applicationID, brandFallback string) string {
	if k, ok := appIndex[applicationID]; ok {
		return k
	}
	return brandFallback
}
