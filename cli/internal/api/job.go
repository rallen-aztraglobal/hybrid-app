package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hybrid-app/cli/internal/manifest"
)

// 本文件实现 ADR-0008「build-job 队列」的 CLI 侧 HTTP 契约。
//
// runner（构建机）通过这些端点：领取任务 → 回传状态/日志 → 登记产物 APK。
// 路径与请求/响应体严格对齐后端实际路由（server/internal/handler/routes.go + build.go）：
//
//	领取  POST /api/build/claim                    body {runner}                     → data: BuildRecord
//	状态  POST /api/build/records/{id}/status      body {status,logExcerpt}
//	日志  POST /api/build/records/{id}/logs        body 纯文本（服务端 io.ReadAll）
//	产物  POST /api/build/records/{id}/artifacts   body {flavor,versionName,apkUrl,size}
//
// 安全红线（CLAUDE.md 护栏 4 / ADR-0008）：keystore 路径与口令绝不出现在任何请求体里。
// 登记产物只含 flavor / versionName / 下载 URL / 大小等非机密信息。

// claimedRecord 是后端 BuildRecord 在「领取」响应里的精简映射。
// 后端 flavors 是 JSON 数组「字符串」、品牌字段名为 brandCode —— 不能直接解进 BuildJob，
// 先解进本结构再转换（评审 C2：避免 []string 解析失败、brand 取空）。
type claimedRecord struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	BrandCode   string `json:"brandCode"`
	Flavors     string `json:"flavors"` // JSON 数组字符串，如 "[\"ap01018\"]"
	TestEvents  bool   `json:"testEvents"`
	VersionName string `json:"versionName"`
}

// ClaimBuildJob 向队列领取一个待构建任务：POST /api/build/claim。
//
// 后端无任务时返回 data:null（或空对象），此时 ID==0，调用方据此判定「暂无任务」。
// runnerID 用于审计与任务归属（非机密）。
func (c *Client) ClaimBuildJob(ctx context.Context, runnerID string) (*manifest.BuildJob, error) {
	var rec claimedRecord
	body := map[string]string{"runner": runnerID}
	if err := c.do(ctx, http.MethodPost, "/api/build/claim", body, &rec, true); err != nil {
		return nil, err
	}
	job := &manifest.BuildJob{
		ID:          rec.ID,
		Brand:       rec.BrandCode,
		TestEvents:  rec.TestEvents,
		VersionName: rec.VersionName,
		TaskName:    rec.Name,
	}
	if strings.TrimSpace(rec.Flavors) != "" {
		if err := json.Unmarshal([]byte(rec.Flavors), &job.Flavors); err != nil {
			return nil, fmt.Errorf("解析任务 flavors 失败: %w", err)
		}
	}
	return job, nil
}

// UpdateJobStatus 回传任务状态：POST /api/build/records/{id}/status。
// status ∈ running/success/failed（manifest.StatusRunning 等）。logExcerpt 为日志尾摘要（可空）。
func (c *Client) UpdateJobStatus(ctx context.Context, jobID int64, status, logExcerpt string) error {
	body := manifest.JobStatusUpdate{Status: status, LogExcerpt: logExcerpt}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/build/records/%d/status", jobID), body, nil, true)
}

// AppendJobLog 追加一段构建日志：POST /api/build/records/{id}/logs（best-effort 流式回传）。
// 服务端把整个请求体按纯文本读取（io.ReadAll），故发送纯文本而非 JSON。失败不应中断构建。
func (c *Client) AppendJobLog(ctx context.Context, jobID int64, chunk string) error {
	return c.doText(ctx, http.MethodPost, fmt.Sprintf("/api/build/records/%d/logs", jobID), chunk, true)
}

// RegisterJobArtifact 登记一个产物 APK：POST /api/build/records/{id}/artifacts。
// 字段对齐后端 AddBuildArtifactInput{flavor,versionName,apkUrl,size}（评审 C3：size 而非 sizeBytes）。
func (c *Client) RegisterJobArtifact(ctx context.Context, jobID int64, art manifest.BuildArtifact) error {
	body := map[string]any{
		"flavor":      art.Flavor,
		"versionName": art.VersionName,
		"apkUrl":      art.ApkURL,
		"size":        art.SizeBytes,
	}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/build/records/%d/artifacts", jobID), body, nil, true)
}
