// Package handler 是 HTTP 层（Echo）。保持「薄」：解析请求 → 调 service → 统一封装响应。
package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/service"
)

// Handler 持有依赖。
type Handler struct {
	cfg     *config.Config
	svc     *service.Service
	authMgr *auth.Manager
	repo    *repo.Repo
}

// New 创建 Handler。
func New(cfg *config.Config, svc *service.Service, authMgr *auth.Manager, r *repo.Repo) *Handler {
	return &Handler{cfg: cfg, svc: svc, authMgr: authMgr, repo: r}
}

// fail 把 service 错误映射为统一响应。
func fail(c echo.Context, err error) error {
	e := service.AsError(err)
	return httpx.Fail(c, e.Code, e.Message)
}

// paramID 解析路径参数 :id。
func paramID(c echo.Context) (uint64, error) {
	return strconv.ParseUint(c.Param("id"), 10, 64)
}
