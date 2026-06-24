package service

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/hybrid-app/server/internal/model"
)

// RunDomainHealthScan 巡检库里所有已配置域名，写入 domain_health（ADR-0003：仅监控展示）。
// 由 robfig/cron 周期触发；容错，单个域名失败不影响整体。
func (s *Service) RunDomainHealthScan(ctx context.Context) {
	urls, err := s.repo.AllConfiguredDomains(ctx)
	if err != nil {
		log.Printf("[probe] 取域名清单失败: %v", err)
		return
	}
	client := &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse // 不跟随重定向，劫持常用 302
	}}
	for _, u := range urls {
		pr := probeOne(ctx, client, u, s.cfg.DefaultProbePath)
		status := model.HealthDown
		if pr.OK {
			status = model.HealthOK
		} else if pr.HTTPCode == http.StatusOK {
			// 可达但校验不符（这里只看到 200，未做业务校验）→ 标记疑似劫持。
			status = model.HealthHijacked
		}
		_ = s.repo.SaveDomainHealth(ctx, &model.DomainHealth{
			URL: u, Status: status, HTTPCode: pr.HTTPCode, LatencyMs: pr.LatencyMs,
		})
	}
	log.Printf("[probe] 域名巡检完成，共 %d 个", len(urls))
}

// LatestDomainHealth 返回每个 URL 最近一次巡检结果（看板红绿灯用）。
func (s *Service) LatestDomainHealth(ctx context.Context) ([]model.DomainHealth, error) {
	return s.repo.LatestDomainHealth(ctx)
}
