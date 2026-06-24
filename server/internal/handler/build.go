package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

// BuildManifest godoc
// @Summary  一次拉全：某品牌启用渠道 + 域名 + palcode + 资源 zip 地址（CLI 据此重写 CSV/res）
// @Tags     build
// @Produce  json
// @Param    brand  query     string  true  "品牌 code"
// @Success  200    {object}  httpx.Envelope{data=service.BuildManifest}
// @Security BearerAuth
// @Router   /api/build/manifest [get]
func (h *Handler) BuildManifest(c echo.Context) error {
	brand := c.QueryParam("brand")
	if brand == "" {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 brand 参数")
	}
	m, err := h.svc.BuildManifestForBrand(c.Request().Context(), brand)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, m)
}

// CreateBuildJob godoc
// @Summary  入队一个构建任务（Web 打包中心触发；状态机 queued→running→success/failed）
// @Tags     build
// @Accept   json
// @Produce  json
// @Param    body  body      service.CreateBuildJobInput  true  "brand/flavors/versionName(X.Y.Z)/testEvents/name?"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/build/jobs [post]
func (h *Handler) CreateBuildJob(c echo.Context) error {
	var in service.CreateBuildJobInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	rec, err := h.svc.CreateBuildJob(c.Request().Context(), in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, rec)
}

// ListBuildRecords godoc
// @Summary  构建记录列表（按品牌/状态筛选）
// @Tags     build
// @Produce  json
// @Param    brand   query     string  false  "品牌 code"
// @Param    status  query     string  false  "状态 queued/running/success/failed"
// @Param    limit   query     int     false  "条数，默认 50"
// @Success  200     {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/build/records [get]
func (h *Handler) ListBuildRecords(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	list, err := h.svc.ListBuildRecords(c.Request().Context(), c.QueryParam("brand"), c.QueryParam("status"), limit)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, list)
}

// GetBuildRecord godoc
// @Summary  单条构建记录（含全部 APK 产物，供「构建记录页」列出下载）
// @Tags     build
// @Produce  json
// @Param    id   path      int  true  "构建记录 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/build/records/{id} [get]
func (h *Handler) GetBuildRecord(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	rec, err := h.svc.GetBuildRecord(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, rec)
}

// BuildLogs godoc
// @Summary  构建日志（分段/流式）—— 传 offset 拉增量，轮询 next 直到 done
// @Tags     build
// @Produce  json
// @Param    id      path      int  true   "构建记录 ID"
// @Param    offset  query     int  false  "起始字节偏移，默认 0"
// @Success  200     {object}  httpx.Envelope{data=service.BuildLogSegment}
// @Security BearerAuth
// @Router   /api/build/records/{id}/logs [get]
func (h *Handler) BuildLogs(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	seg, err := h.svc.BuildLog(c.Request().Context(), id, offset)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, seg)
}

// ---------- runner（构建机）领取与上报 ----------

type claimReq struct {
	Runner string `json:"runner"`
}

// ClaimBuild godoc
// @Summary  runner 领取下一条 queued 任务（原子；无任务返回 data:null）
// @Tags     build
// @Accept   json
// @Produce  json
// @Param    body  body      claimReq  false  "runner 标识"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/build/claim [post]
func (h *Handler) ClaimBuild(c echo.Context) error {
	var req claimReq
	_ = c.Bind(&req)
	rec, err := h.svc.ClaimBuild(c.Request().Context(), req.Runner)
	if err != nil {
		return fail(c, err)
	}
	// 无可领取任务时 data=null（runner 据此 sleep 后重试）。
	return httpx.OK(c, rec)
}

// ReportBuildStatus godoc
// @Summary  runner 上报构建状态（running/success/failed）
// @Tags     build
// @Accept   json
// @Produce  json
// @Param    id    path      int                              true  "构建记录 ID"
// @Param    body  body      service.ReportBuildStatusInput   true  "状态"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/build/records/{id}/status [post]
func (h *Handler) ReportBuildStatus(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.ReportBuildStatusInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	rec, err := h.svc.ReportBuildStatus(c.Request().Context(), id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, rec)
}

// AppendBuildLog godoc
// @Summary  runner 增量上报构建日志（请求体为纯文本，追加到该记录日志）
// @Tags     build
// @Accept   plain
// @Produce  json
// @Param    id    path      int     true  "构建记录 ID"
// @Param    body  body      string  true  "日志片段（纯文本）"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/build/records/{id}/logs [post]
func (h *Handler) AppendBuildLog(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "读取日志失败")
	}
	if err := h.svc.AppendBuildLog(c.Request().Context(), id, string(body)); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"appended": len(body)})
}

// AddBuildArtifact godoc
// @Summary  runner 上报一个产出的 APK（落 build_artifact，并汇总进记录）
// @Tags     build
// @Accept   json
// @Produce  json
// @Param    id    path      int                             true  "构建记录 ID"
// @Param    body  body      service.AddBuildArtifactInput   true  "flavor/versionName/apkUrl/size"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/build/records/{id}/artifacts [post]
func (h *Handler) AddBuildArtifact(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.AddBuildArtifactInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	a, err := h.svc.AddBuildArtifact(c.Request().Context(), id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, a)
}
