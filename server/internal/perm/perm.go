// Package perm 静态定义 RBAC 权限点清单（唯一契约见 docs/admin/10-rbac.md）。
// code 常量与顺序必须与该文档「权限点清单」表逐字一致——前端从 GET /api/perms/catalog
// 拉取本包 Catalog() 渲染勾选树，双方以此为唯一事实来源。
package perm

// Kind 标识权限点类型。
const (
	KindRoute  = "route"  // 路由权限：控制页面/菜单可见、其读接口可调
	KindButton = "button" // 按钮权限：控制页面内某写操作 + 对应按钮显隐
)

// 权限点 code 常量。分组、顺序与 docs/admin/10-rbac.md 权限点清单表一致。
const (
	// 渠道管理
	PageChannels   = "page:channels"
	ChannelCreate  = "channel:create"
	ChannelEdit    = "channel:edit"
	ChannelArchive = "channel:archive"

	// 域名管理
	PageDomains = "page:domains"
	DomainEdit  = "domain:edit"

	// 打包中心
	PagePack    = "page:pack"
	BuildSubmit = "build:submit"

	// 构建记录
	PageBuilds = "page:builds"

	// 推送管理
	PagePush   = "page:push"
	PushCreate = "push:create"
	PushSend   = "push:send"
	PushConfig = "push:config"

	// 上架包
	PageListings = "page:listings"
	ListingEdit  = "listing:edit"
	ListingGate  = "listing:gate"

	// 系统设置
	PageSettings = "page:settings"
	StoreManage  = "store:manage"
	UserManage   = "user:manage"
	RoleManage   = "role:manage"

	// 设备管理
	PageDevices  = "page:devices"
	DeviceExport = "device:export"
)

// RunnerPerm 是构建机静态 token 隐式持有的机器权限：不进 catalog、不可分配给任何角色
// （校验时会被 IsValid 拒绝），仅用于放行 4 条机器接口
// （/build/claim、/build/records/:id/{status,logs,artifacts}）。
const RunnerPerm = "build:runner"

// Perm 是 catalog 里的一个权限点。
type Perm struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

// Module 是 catalog 里的一个分组，对应前端设置页「角色管理」勾选树的一个模块。
type Module struct {
	Module string `json:"module"`
	Label  string `json:"label"`
	Perms  []Perm `json:"perms"`
}

// catalog 静态定义，字段与顺序对齐 docs/admin/10-rbac.md 的权限点清单表。
var catalog = []Module{
	{Module: "channels", Label: "渠道管理", Perms: []Perm{
		{Code: PageChannels, Label: "渠道列表/详情/res.zip/最新包下载", Kind: KindRoute},
		{Code: ChannelCreate, Label: "新增渠道", Kind: KindButton},
		{Code: ChannelEdit, Label: "编辑渠道、上传图标/启动图", Kind: KindButton},
		{Code: ChannelArchive, Label: "归档(删除)渠道", Kind: KindButton},
	}},
	{Module: "domains", Label: "域名管理", Perms: []Perm{
		{Code: PageDomains, Label: "品牌/渠道域名查看", Kind: KindRoute},
		{Code: DomainEdit, Label: "修改品牌域名、渠道域名", Kind: KindButton},
	}},
	{Module: "pack", Label: "打包中心", Perms: []Perm{
		{Code: PagePack, Label: "打包提交页(manifest/当前版本查询)", Kind: KindRoute},
		{Code: BuildSubmit, Label: "提交打包任务", Kind: KindButton},
	}},
	{Module: "builds", Label: "构建记录", Perms: []Perm{
		{Code: PageBuilds, Label: "构建记录/日志查看", Kind: KindRoute},
	}},
	{Module: "push", Label: "推送管理", Perms: []Perm{
		{Code: PagePush, Label: "推送状态/活动/受众查看(含上架包推送活动)", Kind: KindRoute},
		{Code: PushCreate, Label: "新建/编辑活动、上传图片", Kind: KindButton},
		{Code: PushSend, Label: "发送/定时发送(含上架包活动发送)", Kind: KindButton},
		{Code: PushConfig, Label: "上传 google-services.json", Kind: KindButton},
	}},
	{Module: "listings", Label: "上架包", Perms: []Perm{
		{Code: PageListings, Label: "上架包列表/详情/判定流水查看", Kind: KindRoute},
		{Code: ListingEdit, Label: "新建/编辑/删除上架包、改域名", Kind: KindButton},
		{Code: ListingGate, Label: "AB 面网关配置与试算", Kind: KindButton},
	}},
	{Module: "settings", Label: "系统设置", Perms: []Perm{
		{Code: PageSettings, Label: "设置页(商店/运行时预览等查看)", Kind: KindRoute},
		{Code: StoreManage, Label: "商店管理写操作", Kind: KindButton},
	}},
	// 角色管理 / 用户管理已从「系统设置」拆成侧边栏独立菜单：role:manage / user:manage
	// 现在是各自页面的路由权限（kind=route，控制菜单可见与进入权），而不是某个页面内的按钮权限。
	// code 字符串保持不变——它们已写进存量 role_permission 数据，改字符串会让已有授权失效。
	{Module: "roles", Label: "角色管理", Perms: []Perm{
		{Code: RoleManage, Label: "角色管理(查看/增删改)", Kind: KindRoute},
	}},
	{Module: "users", Label: "用户管理", Perms: []Perm{
		{Code: UserManage, Label: "用户管理(查看/增删改/重置密码)", Kind: KindRoute},
	}},
	{Module: "devices", Label: "设备管理", Perms: []Perm{
		{Code: PageDevices, Label: "设备列表查看", Kind: KindRoute},
		{Code: DeviceExport, Label: "导出设备 CSV", Kind: KindButton},
	}},
}

var (
	allCodes    []string
	routeCodes  []string
	buttonCodes []string
	codeSet     map[string]bool
)

func init() {
	codeSet = map[string]bool{}
	for _, m := range catalog {
		for _, p := range m.Perms {
			allCodes = append(allCodes, p.Code)
			codeSet[p.Code] = true
			if p.Kind == KindRoute {
				routeCodes = append(routeCodes, p.Code)
			} else {
				buttonCodes = append(buttonCodes, p.Code)
			}
		}
	}
}

// Catalog 返回全量权限点清单（登录即可访问，供前端渲染勾选树）。
func Catalog() []Module { return catalog }

// AllCodes 返回全部可分配权限点 code（route+button，不含机器权限 RunnerPerm）。
func AllCodes() []string { return append([]string(nil), allCodes...) }

// RouteCodes 返回全部 kind=route 的 code。多数形如 page:*，但不是所有 route code 都是
// page:*——role:manage/user:manage 也是 route（各自一个独立菜单的进入权，不是 page:* 命名），
// 调用方（尤其是给默认角色算权限集时）注意：这里面混了 SystemManageCodes() 里的敏感权限，
// 通常要再减去 SystemManageCodes() 才能当作「普通业务页面」的权限集使用，见该函数注释。
func RouteCodes() []string { return append([]string(nil), routeCodes...) }

// ButtonCodes 返回全部 button code。
func ButtonCodes() []string { return append([]string(nil), buttonCodes...) }

// IsValid 判断 code 是否在 catalog 内（可分配）。机器权限 RunnerPerm 不算合法可分配 code。
func IsValid(code string) bool { return codeSet[code] }

// SystemManageCodes 返回「敏感管理权限」的具名集合：商店管理、角色管理、用户管理。
//
// 陷阱说明：RoleManage/UserManage 的 kind 是 route（各自一个独立菜单的进入权），会被
// RouteCodes() 收进去；而 seed 里「只读」角色的默认权限集恰好就是 RouteCodes()——如果不
// 特殊处理，只读角色会自动白捡角色管理/用户管理这两个本该只有超管才能授予的敏感权限。
// 因此默认角色（只读、运营）的权限集一律要在 RouteCodes()/ButtonCodes() 基础上减去这个
// 集合，只有超级管理员能显式把它们挂给某个角色。catalog 演进时新增的同类敏感权限点也应
// 收进这里，而不是在各处散落地手写排除名单（避免以后再加漏一处）。
func SystemManageCodes() []string {
	return []string{StoreManage, RoleManage, UserManage}
}
