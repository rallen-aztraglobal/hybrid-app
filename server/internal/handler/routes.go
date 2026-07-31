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
// 鉴权分层：/api/app/* 与 /healthz 公开；其余需 JWT；日常业务读写要求 user+；
// 系统设置（商店管理）与渠道归档/删除要求 admin。
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

	// 需要 JWT 的管理面。RequireEnabled 必须紧跟鉴权中间件之后：
	// 禁用账号已签发的 token 要在这里就被拦下，而不是先跑到某个业务 handler 里才发现。
	api := e.Group("/api")
	api.Use(h.authMgr.Middleware(), h.RequireEnabled)

	user := auth.RequireRole(model.RoleUser)   // 日常业务操作（读 + 大多数写）
	admin := auth.RequireRole(model.RoleAdmin) // 系统设置（商店）、渠道归档/删除、账号管理

	// 大渠道。
	api.GET("/brands", h.ListBrands, user)
	api.GET("/brands/:code/domains", h.GetBrandDomains, user)
	api.PUT("/brands/:code/domains", h.SetBrandDomains, user)

	// 应用商店（系统设置模块；渠道 store 后缀见 CLAUDE.md 商店后缀功能）：admin-only。
	api.GET("/stores", h.ListStores, admin)
	api.POST("/stores", h.CreateStore, admin)
	api.PUT("/stores/:id", h.UpdateStore, admin)
	api.DELETE("/stores/:id", h.DeleteStore, admin)

	// 账号管理（Admin-only User Management）：admin-only。无硬删除端点——
	// audit_log.user_id / channel.created_by / listing_app.created_by 均引用 admin_user.id
	// 且无级联，硬删会破坏审计与归属追溯，故只提供启停用（见 service.DeleteUser 相关说明）。
	api.GET("/users", h.ListUsers, admin)
	api.POST("/users", h.CreateUser, admin)
	api.PUT("/users/:id", h.UpdateUser, admin)
	api.POST("/users/:id/reset-password", h.ResetUserPassword, admin)

	// 上架包（Flutter/原生 App，独立于小渠道 APK 产线）。
	api.GET("/listings", h.ListListings, user)
	api.POST("/listings", h.CreateListing, user)
	api.GET("/listings/:id", h.GetListing, user)
	api.PUT("/listings/:id", h.UpdateListing, user)
	api.DELETE("/listings/:id", h.DeleteListing, user)
	api.PUT("/listings/:id/domains", h.SetListingDomains, user)
	api.PUT("/listings/:id/gate", h.SetListingGate, user)
	api.POST("/listings/:id/gate/test", h.TestListingGate, user) // 后台试算判定
	api.GET("/listings/:id/gate/logs", h.ListGateLogs, user)     // 判定流水排查

	// 上架包推送（复用推送管线，但强制只发 B 面设备；独立 Firebase 项目）。
	api.GET("/push/listing-campaigns", h.ListListingCampaigns, user)
	api.POST("/push/listing-campaigns", h.CreateListingCampaign, user)
	api.POST("/push/listing-campaigns/:id/send", h.SendListingCampaign, user)

	// 小渠道 CRUD。
	api.GET("/channels", h.ListChannels, user)
	api.POST("/channels", h.CreateChannel, user)
	api.GET("/channels/:id", h.GetChannel, user)
	api.PUT("/channels/:id", h.UpdateChannel, user)
	api.DELETE("/channels/:id", h.DeleteChannel, admin) // 归档=破坏性操作，admin-only
	api.PUT("/channels/:id/domains", h.SetChannelDomains, user)
	api.GET("/channels/:id/latest-apk", h.LatestApk, user) // 渠道卡片「下载最新包」（ADR-0008）

	// 图标管线。
	api.POST("/channels/:id/icon", h.UploadIcon, user)
	api.POST("/channels/:id/splash", h.UploadSplash, user)
	api.GET("/channels/:id/res.zip", h.GetResZip, user)

	// 打包：manifest 供 CLI 拉全量；jobs 队列与记录供 Web 打包中心（ADR-0008）。
	api.GET("/build/manifest", h.BuildManifest, user)
	// 当前版本：与 CreateBuildJob 的版本校验共用 Service.CurrentVersion，全量扫描 success 记录
	// （不像 /build/records 那样受分页/上限约束），保证前端展示与后端强制校验永远一致。
	api.GET("/build/current-version", h.GetCurrentVersion, user)
	api.POST("/build/jobs", h.CreateBuildJob, user)
	api.GET("/build/records", h.ListBuildRecords, user)
	api.GET("/build/records/:id", h.GetBuildRecord, user)
	api.GET("/build/records/:id/logs", h.BuildLogs, user)

	// 构建机（runner）领取与上报：机器对机器，静态令牌注入的是 user 身份（见 auth.Manager.Middleware）。
	api.POST("/build/claim", h.ClaimBuild, user)
	api.POST("/build/records/:id/status", h.ReportBuildStatus, user)
	api.POST("/build/records/:id/logs", h.AppendBuildLog, user)
	api.POST("/build/records/:id/artifacts", h.AddBuildArtifact, user)

	// 推送管理（ADR-0012）。
	api.GET("/push/status", h.GetPushStatus, user)
	api.GET("/push/campaigns", h.ListPushCampaigns, user)
	api.POST("/push/campaigns", h.CreatePushCampaign, user)
	api.GET("/push/campaigns/:id", h.GetPushCampaign, user)
	api.PUT("/push/campaigns/:id", h.UpdatePushCampaign, user)
	api.POST("/push/campaigns/:id/send", h.SendPushCampaign, user)
	api.POST("/push/campaigns/:id/schedule", h.SchedulePushCampaign, user)
	api.POST("/push/upload-image", h.UploadPushImage, user)
	api.GET("/push/audience", h.GetPushAudience, user)
	// google-services.json 上传（user+；GET 公开已在上方注册）。
	api.POST("/push/google-services", h.UploadGoogleServices, user)
}
