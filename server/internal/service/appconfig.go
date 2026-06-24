package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hybrid-app/server/internal/model"
)

// AppConfig 是 GET /api/app/config?appId= 的响应体（docs/admin/01 §5.6）。
// APK 据此做「实时拉取 + 自更新缓存 + 编译期兜底」三级取用（ADR-0002）。
// 解析键为 applicationId（ADR-0009 更正 0002 的 palcode）；palcode 仍回显，但不作为身份键。
type AppConfig struct {
	AppID         string   `json:"appId"`         // = applicationId（解析键）
	Palcode       string   `json:"palcode"`       // 渠道 PAL_CODE（回显，APK 拼 /?palcode= 用，可跨品牌重复）
	Domains       []string `json:"domains"`       // [主, 备用1..3]，按 position 升序
	ProbePath     string   `json:"probePath"`     // APK 探测「确实是我们站点」用（ADR-0003）
	ConfigVersion int64    `json:"configVersion"` // 渠道 updated_at 时间戳，单调递增便于客户端判新
	TTLSeconds    int      `json:"ttlSeconds"`    // 建议缓存秒数
}

// AppConfigForAppId 组装某渠道的运行时配置（以 applicationId 为键，ADR-0009）。
// 域名按 use_brand_domains 合并继承（ADR-0006）。这是公开端点的核心逻辑。
func (s *Service) AppConfigForAppId(ctx context.Context, appID string) (*AppConfig, error) {
	ch, err := s.repo.GetChannelByApplicationID(ctx, appID)
	if err != nil {
		return nil, errNotFound("未知 appId")
	}
	if ch.Status == model.ChannelArchived {
		return nil, errNotFound("渠道已归档")
	}

	domains, err := s.effectiveDomains(ctx, ch.ID)
	if err != nil {
		return nil, err
	}

	cfg := &AppConfig{
		AppID:         ch.ApplicationID,
		Palcode:       ch.PalCode,
		Domains:       domains,
		ProbePath:     s.cfg.DefaultProbePath,
		ConfigVersion: ch.UpdatedAt.Unix(),
		TTLSeconds:    s.cfg.AppConfigTTLSecs,
	}
	return cfg, nil
}

// snapshotKey 返回某 applicationId 的 CDN 静态快照对象 key。
func snapshotKey(appID string) string {
	return fmt.Sprintf("app-config/config-%s.json", appID)
}

// SnapshotURL 返回某 applicationId 的 CDN 快照公开地址。
func (s *Service) SnapshotURL(appID string) string {
	return s.storage.PublicURL(snapshotKey(appID))
}

// regenSnapshot 重新生成某 applicationId 的 CDN 静态快照（ADR-0002：配置端点部署在抗封基础设施）。
// 容错：失败仅记日志，不阻断域名保存主流程。
func (s *Service) regenSnapshot(ctx context.Context, appID string) {
	cfg, err := s.AppConfigForAppId(ctx, appID)
	if err != nil {
		log.Printf("[snapshot] 组装 appId=%s 配置失败: %v", appID, err)
		return
	}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("[snapshot] 序列化 appId=%s 失败: %v", appID, err)
		return
	}
	if _, err := s.storage.Put(ctx, snapshotKey(appID), bytes.NewReader(body), int64(len(body)), "application/json"); err != nil {
		log.Printf("[snapshot] 上传 appId=%s 快照失败: %v", appID, err)
		return
	}
}

// archiveSnapshot 渠道归档（或 appId 变更）后，把其 CDN 快照内容替换为「已归档」标记（容错）。
// 不直接删除，避免旧 APK 拉到 404 后无法区分「网络问题/已下线」；返回明确状态更友好。
func (s *Service) archiveSnapshot(ctx context.Context, appID string) {
	body := []byte(fmt.Sprintf(`{"appId":%q,"archived":true}`, appID))
	if _, err := s.storage.Put(ctx, snapshotKey(appID), bytes.NewReader(body), int64(len(body)), "application/json"); err != nil {
		log.Printf("[snapshot] 归档 appId=%s 快照失败: %v", appID, err)
	}
}

// regenSnapshotsForBrand 品牌域名变更后，刷新该品牌下所有「继承品牌域名」渠道的快照。
func (s *Service) regenSnapshotsForBrand(ctx context.Context, brandID uint64) {
	brand, err := s.repo.GetBrandByID(ctx, brandID)
	if err != nil {
		log.Printf("[snapshot] 取品牌失败 brandID=%d: %v", brandID, err)
		return
	}
	// 拉取该品牌所有非归档渠道。
	list, _, err := s.repo.ListChannels(ctx, repoChannelFilterAllForBrand(brand.Code))
	if err != nil {
		log.Printf("[snapshot] 列渠道失败 brand=%s: %v", brand.Code, err)
		return
	}
	for i := range list {
		ch := &list[i]
		if ch.Status == model.ChannelArchived {
			continue
		}
		// 仅继承品牌域名的渠道才受品牌域名变更影响；覆盖了的渠道快照不变。
		if ch.UseBrandDomains {
			s.regenSnapshot(ctx, ch.ApplicationID)
		}
	}
}
