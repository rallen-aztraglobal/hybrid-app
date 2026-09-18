// Package handler — 推送 HTTP 端点（ADR-0012）。
package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

// registerTokenReq APK 上报 token 请求体。
type registerTokenReq struct {
	AppID    string `json:"appId"`
	Token    string `json:"token"`
	PalCode  string `json:"palcode"`
	Platform string `json:"platform"`
	Model    string `json:"model"`
}

// RegisterPushToken godoc
// @Summary  APK 上报 FCM 设备 token（公开端点）
// @Tags     push
// @Accept   json
// @Produce  json
// @Param    body  body      registerTokenReq  true  "token 注册请求"
// @Success  200   {object}  httpx.Envelope
// @Router   /api/app/push/register-token [post]
func (h *Handler) RegisterPushToken(c echo.Context) error {
	var req registerTokenReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	if err := h.svc.RegisterDeviceToken(c.Request().Context(),
		req.AppID, req.Token, req.PalCode, req.Platform, req.Model); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"registered": true})
}

// GetPushStatus godoc
// @Summary  FCM 配置状态（viewer）
// @Tags     push
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/status [get]
func (h *Handler) GetPushStatus(c echo.Context) error {
	return httpx.OK(c, h.svc.PushStatus())
}

// ListPushCampaigns godoc
// @Summary  推送活动列表（viewer）
// @Tags     push
// @Produce  json
// @Param    brand  query  string  false  "按品牌 code 筛选"
// @Success  200    {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns [get]
func (h *Handler) ListPushCampaigns(c echo.Context) error {
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	list, err := h.svc.ListCampaigns(c.Request().Context(), c.QueryParam("brand"), scope)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, list)
}

// CreatePushCampaign godoc
// @Summary  创建推送活动草稿（operator）
// @Tags     push
// @Accept   json
// @Produce  json
// @Param    body  body      service.PushCampaignInput  true  "活动内容"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns [post]
func (h *Handler) CreatePushCampaign(c echo.Context) error {
	var in service.PushCampaignInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	createdBy := ""
	if claims := auth.FromContext(c); claims != nil {
		createdBy = claims.Username
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.CreateCampaign(c.Request().Context(), scope, in, createdBy)
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, v)
}

// GetPushCampaign godoc
// @Summary  推送活动详情（viewer）
// @Tags     push
// @Produce  json
// @Param    id  path  int  true  "活动 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns/{id} [get]
func (h *Handler) GetPushCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	detail, err := h.svc.GetCampaign(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, detail)
}

// UpdatePushCampaign godoc
// @Summary  修改推送活动草稿（operator，仅 draft 可改）
// @Tags     push
// @Accept   json
// @Produce  json
// @Param    id    path  int                        true  "活动 ID"
// @Param    body  body  service.PushCampaignInput  true  "更新内容"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns/{id} [put]
func (h *Handler) UpdatePushCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.PushCampaignInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.UpdateCampaign(c.Request().Context(), scope, id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// sendCampaignReq 立即发送请求体。
type sendCampaignReq struct {
	DryRun bool `json:"dryRun"`
}

// SendPushCampaign godoc
// @Summary  立即发送推送活动（operator）；dryRun=true 走完统计但不真发
// @Tags     push
// @Accept   json
// @Produce  json
// @Param    id    path  int              true  "活动 ID"
// @Param    body  body  sendCampaignReq  true  "dryRun 标志"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns/{id}/send [post]
func (h *Handler) SendPushCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req sendCampaignReq
	_ = c.Bind(&req) // 可选 body，解析失败不阻断
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.SendCampaign(c.Request().Context(), scope, id, req.DryRun)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// scheduleCampaignReq 定时发送请求体。RepeatEveryDays 0=单次；1=每天；N=每 N 天（1..365）。
// RepeatEndAt 为空串或不传表示不限；单次任务传了 repeatEndAt/repeatMaxRuns 会被服务端忽略清零。
type scheduleCampaignReq struct {
	ScheduledAt     string `json:"scheduledAt"` // ISO8601
	RepeatEveryDays int    `json:"repeatEveryDays"`
	RepeatEndAt     string `json:"repeatEndAt"` // ISO8601，可选/空串=不限
	RepeatMaxRuns   int    `json:"repeatMaxRuns"`
}

// parseScheduleReq 把请求体解析为 service.ScheduleCampaignInput（channel/listing 两个
// schedule 端点共用），只做格式解析，取值范围等业务校验交给 service 层。
func parseScheduleReq(req scheduleCampaignReq) (service.ScheduleCampaignInput, error) {
	var in service.ScheduleCampaignInput
	if req.ScheduledAt == "" {
		return in, fmt.Errorf("scheduledAt (ISO8601) 不得为空")
	}
	t, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return in, fmt.Errorf("scheduledAt 格式应为 ISO8601，例如 2026-07-01T10:00:00Z")
	}
	in.ScheduledAt = t
	in.RepeatEveryDays = req.RepeatEveryDays
	in.RepeatMaxRuns = req.RepeatMaxRuns
	if strings.TrimSpace(req.RepeatEndAt) != "" {
		endAt, err := time.Parse(time.RFC3339, req.RepeatEndAt)
		if err != nil {
			return in, fmt.Errorf("repeatEndAt 格式应为 ISO8601，例如 2026-07-01T10:00:00Z")
		}
		in.RepeatEndAt = &endAt
	}
	return in, nil
}

// SchedulePushCampaign godoc
// @Summary  设置渠道推送活动定时发送（operator），支持单次/周期（每天/每 N 天）
// @Tags     push
// @Accept   json
// @Produce  json
// @Param    id    path  int                  true  "活动 ID"
// @Param    body  body  scheduleCampaignReq  true  "定时时间 + 周期参数"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns/{id}/schedule [post]
func (h *Handler) SchedulePushCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req scheduleCampaignReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	in, err := parseScheduleReq(req)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, err.Error())
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.ScheduleCampaign(c.Request().Context(), scope, id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// PausePushCampaign godoc
// @Summary  暂停周期推送活动（operator），仅 scheduled 且为周期任务可暂停；对 channel/listing 两种 kind 都生效
// @Tags     push
// @Produce  json
// @Param    id  path  int  true  "活动 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns/{id}/pause [post]
func (h *Handler) PausePushCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.PauseCampaign(c.Request().Context(), scope, id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// ResumePushCampaign godoc
// @Summary  恢复已暂停的周期推送活动（operator）；若原定时间已过去，按周期步进到下一个未来时间点
// @Tags     push
// @Produce  json
// @Param    id  path  int  true  "活动 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns/{id}/resume [post]
func (h *Handler) ResumePushCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.ResumeCampaign(c.Request().Context(), scope, id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// CancelPushCampaign godoc
// @Summary  取消定时推送活动（operator），scheduled/paused 均可取消，单次与周期任务都适用
// @Tags     push
// @Produce  json
// @Param    id  path  int  true  "活动 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/campaigns/{id}/cancel [post]
func (h *Handler) CancelPushCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.CancelCampaign(c.Request().Context(), scope, id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// UploadPushImage godoc
// @Summary  上传推送图片（operator）
// @Tags     push
// @Accept   multipart/form-data
// @Produce  json
// @Param    file  formData  file  true  "图片文件"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/upload-image [post]
func (h *Handler) UploadPushImage(c echo.Context) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 file 字段")
	}
	f, err := fh.Open()
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "打开上传文件失败")
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "读取文件内容失败")
	}

	// 从文件名或 Content-Type 猜测类型。
	ct := fh.Header.Get("Content-Type")
	if ct == "" {
		if strings.HasSuffix(strings.ToLower(fh.Filename), ".png") {
			ct = "image/png"
		} else {
			ct = "image/jpeg"
		}
	}

	url, err := h.svc.UploadPushImageRaw(c.Request().Context(), bytes.NewReader(data), int64(len(data)), ct, fh.Filename)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"url": url})
}

// GetPushAudience godoc
// @Summary  预估目标活跃设备数（viewer）
// @Tags     push
// @Produce  json
// @Param    appIds  query  string  true  "逗号分隔的 applicationId 列表"
// @Success  200     {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/audience [get]
func (h *Handler) GetPushAudience(c echo.Context) error {
	raw := c.QueryParam("appIds")
	var appIDs []string
	for _, s := range strings.Split(raw, ",") {
		if t := strings.TrimSpace(s); t != "" {
			appIDs = append(appIDs, t)
		}
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	result, err := h.svc.PushAudience(c.Request().Context(), scope, appIDs)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, result)
}
