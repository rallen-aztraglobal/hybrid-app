package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/hybrid-app/server/internal/auth"
	_ "github.com/hybrid-app/server/internal/docs" // 注册 swag 生成的 OpenAPI spec
	"github.com/hybrid-app/server/internal/model"
)

// Register 挂载全部路由到 Echo 实例。
// 鉴权分层：/api/app/* 与 /healthz 公开；其余需 JWT；写操作要求 operator+，账号管理要求 admin。
func (h *Handler) Register(e *echo.Echo) {
	// 全局中间件。
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: h.cfg.CORSAllowOrigin,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
	}))

	// OpenAPI / Swagger UI（供前端生成 TS 客户端；spec 在 /swagger/doc.json）。
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	// 注：本地存储的 /static 已由 cmd/server/main.go 注册（e.Static("/static", local.Root())），此处不再重复。

	// 公开端点（APK / 探针）。
	e.GET("/healthz", h.Healthz)
	e.GET("/api/app/config", h.AppConfig)
	// APK 推送 token 注册（公开，校验 appId 对应渠道存在，ADR-0012）。
	e.POST("/api/app/push/register-token", h.RegisterPushToken)
	// google-services.json 按品牌下发（公开，非机密；CLI/构建机消费，ADR-0012 §3/§5）。
	e.GET("/api/app/google-services", h.GetGoogleServices)
	// 上架包 AB 面判定（公开，客户端 App 启动调用，服务端按真实 IP 判国家，不缓存）。
	e.POST("/api/app/listing/gate", h.ListingGate)
	// 上架包设备 push token 注册（公开，带 AB 面判定结果；上架包推送强制只发 B 面设备）。
	e.POST("/api/app/listing/register-token", h.RegisterListingToken)

	// 鉴权端点（登录/刷新公开）。
	e.POST("/api/auth/login", h.Login)
	e.POST("/api/auth/refresh", h.Refresh)

	// 需要 JWT 的管理面。
	api := e.Group("/api")
	api.Use(h.authMgr.Middleware())

	viewer := auth.RequireRole(model.RoleViewer)   // 只读
	operator := auth.RequireRole(model.RoleOperator) // 写

	// 大渠道。
	api.GET("/brands", h.ListBrands, viewer)
	api.GET("/brands/:code/domains", h.GetBrandDomains, viewer)
	api.PUT("/brands/:code/domains", h.SetBrandDomains, operator)

	// 应用商店（渠道 store 后缀，见 CLAUDE.md 商店后缀功能）。
	api.GET("/stores", h.ListStores, viewer)
	api.POST("/stores", h.CreateStore, operator)
	api.PUT("/stores/:id", h.UpdateStore, operator)
	api.DELETE("/stores/:id", h.DeleteStore, operator)

	// 上架包（Flutter/原生 App，独立于小渠道 APK 产线）。
	api.GET("/listings", h.ListListings, viewer)
	api.POST("/listings", h.CreateListing, operator)
	api.GET("/listings/:id", h.GetListing, viewer)
	api.PUT("/listings/:id", h.UpdateListing, operator)
	api.DELETE("/listings/:id", h.DeleteListing, operator)
	api.PUT("/listings/:id/domains", h.SetListingDomains, operator)
	api.PUT("/listings/:id/gate", h.SetListingGate, operator)
	api.POST("/listings/:id/gate/test", h.TestListingGate, operator) // 后台试算判定
	api.GET("/listings/:id/gate/logs", h.ListGateLogs, viewer)       // 判定流水排查

	// 上架包推送（复用推送管线，但强制只发 B 面设备；独立 Firebase 项目）。
	api.GET("/push/listing-campaigns", h.ListListingCampaigns, viewer)
	api.POST("/push/listing-campaigns", h.CreateListingCampaign, operator)
	api.POST("/push/listing-campaigns/:id/send", h.SendListingCampaign, operator)

	// 小渠道 CRUD。
	api.GET("/channels", h.ListChannels, viewer)
	api.POST("/channels", h.CreateChannel, operator)
	api.GET("/channels/:id", h.GetChannel, viewer)
	api.PUT("/channels/:id", h.UpdateChannel, operator)
	api.DELETE("/channels/:id", h.DeleteChannel, operator)
	api.PUT("/channels/:id/domains", h.SetChannelDomains, operator)
	api.GET("/channels/:id/latest-apk", h.LatestApk, viewer) // 渠道卡片「下载最新包」（ADR-0008）

	// 图标管线。
	api.POST("/channels/:id/icon", h.UploadIcon, operator)
	api.POST("/channels/:id/splash", h.UploadSplash, operator)
	api.GET("/channels/:id/res.zip", h.GetResZip, viewer)

	// 打包：manifest 供 CLI 拉全量；jobs 队列与记录供 Web 打包中心（ADR-0008）。
	api.GET("/build/manifest", h.BuildManifest, viewer)
	api.POST("/build/jobs", h.CreateBuildJob, operator)
	api.GET("/build/records", h.ListBuildRecords, viewer)
	api.GET("/build/records/:id", h.GetBuildRecord, viewer)
	api.GET("/build/records/:id/logs", h.BuildLogs, viewer)

	// 构建机（runner）领取与上报：机器对机器，要求 operator+ 的服务账号。
	api.POST("/build/claim", h.ClaimBuild, operator)
	api.POST("/build/records/:id/status", h.ReportBuildStatus, operator)
	api.POST("/build/records/:id/logs", h.AppendBuildLog, operator)
	api.POST("/build/records/:id/artifacts", h.AddBuildArtifact, operator)

	// 推送管理（ADR-0012）。
	api.GET("/push/status", h.GetPushStatus, viewer)
	api.GET("/push/campaigns", h.ListPushCampaigns, viewer)
	api.POST("/push/campaigns", h.CreatePushCampaign, operator)
	api.GET("/push/campaigns/:id", h.GetPushCampaign, viewer)
	api.PUT("/push/campaigns/:id", h.UpdatePushCampaign, operator)
	api.POST("/push/campaigns/:id/send", h.SendPushCampaign, operator)
	api.POST("/push/campaigns/:id/schedule", h.SchedulePushCampaign, operator)
	api.POST("/push/upload-image", h.UploadPushImage, operator)
	api.GET("/push/audience", h.GetPushAudience, viewer)
	// google-services.json 上传（operator+；GET 公开已在上方注册）。
	api.POST("/push/google-services", h.UploadGoogleServices, operator)
}
