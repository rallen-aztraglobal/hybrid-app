package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "github.com/hybrid-app/server/internal/docs" // 注册 swag 生成的 OpenAPI spec
	"github.com/hybrid-app/server/internal/perm"
)

// Register 挂载全部路由到 Echo 实例。
// 鉴权分层：/api/app/* 与 /healthz 公开；其余需 JWT；具体读写权限按 h.rbac.RequirePerm 映射到
// perm 包的权限点 code（唯一契约 docs/admin/10-rbac.md 「鉴权中间件」节）。
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
	// APK 设备信息上报（公开，校验 appId 对应渠道存在）。
	e.POST("/api/app/device/register", h.RegisterDevice)
	// 设备 CSV 导出下载（公开路由，但用 export-token 换来的 scoped token 鉴权，不接受 access
	// token；不进 JWT 组是因为下载链接要能直接被浏览器/新标签页打开，不便带 Authorization 头）。
	e.GET("/api/devices/export.csv", h.ExportDevicesCSV)
	e.GET("/api/devices/export.xlsx", h.ExportDevicesXLSX)
	// google-services.json 按品牌下发（公开，非机密；CLI/构建机消费，ADR-0012 §3/§5）。
	e.GET("/api/app/google-services", h.GetGoogleServices)
	// 上架包 AB 面判定（公开，客户端 App 启动调用，服务端按请求真实 IP 判国家，不缓存）。
	e.POST("/api/app/listing/gate", h.ListingGate)
	// 上架包设备 push token 注册（公开，带 AB 面判定结果；上架包推送强制只发 B 面设备）。
	e.POST("/api/app/listing/register-token", h.RegisterListingToken)

	// 鉴权端点（登录/刷新公开）。
	e.POST("/api/auth/login", h.Login)
	e.POST("/api/auth/refresh", h.Refresh)

	// 需要 JWT 的管理面。
	api := e.Group("/api")
	api.Use(h.authMgr.Middleware())

	// need(codes...) 是 h.rbac.RequirePerm 的简写：any-of 语义的权限中间件构造器。
	need := h.rbac.RequirePerm

	// 登录即可（无需具体权限点）：当前用户信息、权限点 catalog、渠道抽屉的品牌/商店基础数据。
	// 这几条不进 RequirePerm，额外挂 RequireActiveAccount 确认账号仍存在——否则被删账号的旧
	// token 与 runner 静态 token 都能一直读到底（应修4）；目标是删号后这四条也立刻 401。
	active := h.rbac.RequireActiveAccount()
	api.GET("/auth/me", h.Me, active)
	api.GET("/perms/catalog", h.PermsCatalog, active)
	api.GET("/brands", h.ListBrands, active)
	api.GET("/stores", h.ListStores, active)
	// 签名 key 注册表（固定清单）：任何已登录账号可读，供渠道编辑抽屉的下拉框选择。
	api.GET("/signing-keys", h.ListSigningKeys, active)

	// 大渠道。
	api.GET("/brands/:code/domains", h.GetBrandDomains, need(perm.PageDomains))
	api.PUT("/brands/:code/domains", h.SetBrandDomains, need(perm.DomainEdit))

	// 应用商店（渠道 store 后缀，见 CLAUDE.md 商店后缀功能）；GET 登录即可，写操作需 store:manage。
	api.POST("/stores", h.CreateStore, need(perm.StoreManage))
	api.PUT("/stores/:id", h.UpdateStore, need(perm.StoreManage))
	api.DELETE("/stores/:id", h.DeleteStore, need(perm.StoreManage))

	// 角色管理 / 用户管理：各自独立的侧边栏菜单（已从「系统设置」拆出，role:manage/user:manage
	// 的 kind 也已改成 route，见 perm 包），路由映射本身不变。
	// 角色列表放宽为 any-of：用户管理的「角色下拉框」也要读它，仅有 user:manage 的账号不能没有选项。
	api.GET("/roles", h.ListRoles, need(perm.RoleManage, perm.UserManage))
	api.POST("/roles", h.CreateRole, need(perm.RoleManage))
	api.PUT("/roles/:id", h.UpdateRole, need(perm.RoleManage))
	api.DELETE("/roles/:id", h.DeleteRole, need(perm.RoleManage))

	api.GET("/users", h.ListUsers, need(perm.UserManage))
	api.POST("/users", h.CreateUser, need(perm.UserManage))
	api.PUT("/users/:id", h.UpdateUser, need(perm.UserManage))
	api.POST("/users/:id/reset-password", h.ResetUserPassword, need(perm.UserManage))
	api.DELETE("/users/:id", h.DeleteUser, need(perm.UserManage))

	// 上架包（Flutter/原生 App，独立于小渠道 APK 产线）。
	// 上架包列表放宽为 any-of：推送页的「上架包活动」面板也要读它。
	api.GET("/listings", h.ListListings, need(perm.PageListings, perm.PagePush))
	api.POST("/listings", h.CreateListing, need(perm.ListingEdit))
	api.GET("/listings/:id", h.GetListing, need(perm.PageListings))
	api.PUT("/listings/:id", h.UpdateListing, need(perm.ListingEdit))
	api.DELETE("/listings/:id", h.DeleteListing, need(perm.ListingEdit))
	api.PUT("/listings/:id/domains", h.SetListingDomains, need(perm.ListingEdit))
	api.PUT("/listings/:id/gate", h.SetListingGate, need(perm.ListingGate))
	api.POST("/listings/:id/gate/test", h.TestListingGate, need(perm.ListingGate)) // 后台试算判定
	api.GET("/listings/:id/gate/logs", h.ListGateLogs, need(perm.PageListings))    // 判定流水排查

	// 上架包推送（复用推送管线，但强制只发 B 面设备；独立 Firebase 项目）。
	api.GET("/push/listing-campaigns", h.ListListingCampaigns, need(perm.PagePush))
	api.POST("/push/listing-campaigns", h.CreateListingCampaign, need(perm.PushCreate))
	api.POST("/push/listing-campaigns/:id/send", h.SendListingCampaign, need(perm.PushSend))

	// 小渠道 CRUD。
	// 渠道列表放宽为 any-of：打包中心选渠道、推送页受众、设置页运行时预览都要读它,
	// 否则「打包专员」这类自定义角色(仅 page:pack)一个渠道都选不到,页面完全不可用。
	api.GET("/channels", h.ListChannels, need(perm.PageChannels, perm.PagePack, perm.PagePush, perm.PageSettings))
	api.POST("/channels", h.CreateChannel, need(perm.ChannelCreate))
	api.GET("/channels/:id", h.GetChannel, need(perm.PageChannels))
	api.PUT("/channels/:id", h.UpdateChannel, need(perm.ChannelEdit))
	api.DELETE("/channels/:id", h.DeleteChannel, need(perm.ChannelArchive))
	api.PUT("/channels/:id/domains", h.SetChannelDomains, need(perm.DomainEdit))
	api.GET("/channels/:id/latest-apk", h.LatestApk, need(perm.PageChannels)) // 渠道卡片「下载最新包」（ADR-0008）

	// 图标管线。
	api.POST("/channels/:id/icon", h.UploadIcon, need(perm.ChannelEdit))
	api.POST("/channels/:id/splash", h.UploadSplash, need(perm.ChannelEdit))
	// res.zip 例外地也放行 runner：构建机 claim 任务后走 render.Pull 拉 manifest + 各渠道 res.zip
	// 来渲染 flavor 资源，这条是服务端打包流水线必经的读接口（渠道的其他写接口不放 runner）。
	api.GET("/channels/:id/res.zip", h.GetResZip, need(perm.PageChannels, perm.RunnerPerm))

	// 打包：manifest 供 CLI 拉全量；jobs 队列与记录供 Web 打包中心（ADR-0008）。
	// GET 类接口横跨「打包中心」「构建记录」两个模块，用 any-of；同时也放行 runner——
	// 构建机 claim 任务后要拉 manifest 才能渲染 channels.csv + res（不放行会让整条服务端
	// 打包流水线 403 断掉）。
	buildRead := need(perm.PagePack, perm.PageBuilds, perm.RunnerPerm)
	api.GET("/build/manifest", h.BuildManifest, buildRead)
	api.POST("/build/jobs", h.CreateBuildJob, need(perm.BuildSubmit))
	api.GET("/build/records", h.ListBuildRecords, buildRead)
	api.GET("/build/records/:id", h.GetBuildRecord, buildRead)
	api.GET("/build/records/:id/logs", h.BuildLogs, buildRead)

	// 构建机（runner）领取与上报：机器对机器，静态 token 隐式持有 perm.RunnerPerm，
	// 仅放行这 4 条接口，不再是可碰任意写接口的泛化角色。
	buildRunner := need(perm.RunnerPerm)
	api.POST("/build/claim", h.ClaimBuild, buildRunner)
	api.POST("/build/records/:id/status", h.ReportBuildStatus, buildRunner)
	api.POST("/build/records/:id/logs", h.AppendBuildLog, buildRunner)
	api.POST("/build/records/:id/artifacts", h.AddBuildArtifact, buildRunner)

	// 推送管理（ADR-0012）。
	api.GET("/push/status", h.GetPushStatus, need(perm.PagePush))
	api.GET("/push/campaigns", h.ListPushCampaigns, need(perm.PagePush))
	api.POST("/push/campaigns", h.CreatePushCampaign, need(perm.PushCreate))
	api.GET("/push/campaigns/:id", h.GetPushCampaign, need(perm.PagePush))
	api.PUT("/push/campaigns/:id", h.UpdatePushCampaign, need(perm.PushCreate))
	api.POST("/push/campaigns/:id/send", h.SendPushCampaign, need(perm.PushSend))
	api.POST("/push/campaigns/:id/schedule", h.SchedulePushCampaign, need(perm.PushSend))
	api.POST("/push/upload-image", h.UploadPushImage, need(perm.PushCreate))
	api.GET("/push/audience", h.GetPushAudience, need(perm.PagePush))
	// google-services.json 上传（push:config；GET 公开已在上方注册）。
	api.POST("/push/google-services", h.UploadGoogleServices, need(perm.PushConfig))

	// 设备管理（渠道设备上报列表 + CSV 导出；导出下载端点 export.csv 走 scoped token，
	// 不进 JWT 组，已在上方公开区注册）。
	api.GET("/devices", h.ListDevices, need(perm.PageDevices))
	api.GET("/devices/export-token", h.GetDeviceExportToken, need(perm.DeviceExport))
	api.POST("/devices/export", h.ExportDevicesByIDs, need(perm.DeviceExport))
}
