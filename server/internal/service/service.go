// Package service 是业务逻辑层（handler 薄、service 厚）。
// 聚合渠道 CRUD + 唯一性、域名校验/探测、运行时配置组装、图标 fan-out 编排、构建 manifest。
package service

import (
	"errors"
	"net"
	"net/http"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/storage"
)

// CountryResolver 把 IP 解析为国家码。service 只依赖这个最小接口（geoip.Resolver 实现之），
// 便于单测替换，也让「未注入 resolver」成为合法状态——此时一律解析不出国家 → 网关判 A 面。
type CountryResolver interface {
	Country(ip net.IP) (string, bool)
}

// Service 持有依赖，向 handler 暴露用例方法。
type Service struct {
	cfg     *config.Config
	repo    *repo.Repo
	storage storage.Storage
	fcm     *FCMManager
	geo     CountryResolver // 可空：nil 时上架包网关把所有请求判为未知国家 → A 面（fail-closed）
	// rbac 供角色/用户管理的最小权限校验（B2）复用 ResolveFresh/RolePermCodes 这套「角色 → 权限集」
	// 算法，与 handler 层 RequirePerm 用的是同一份实现，避免两处判定逻辑各写一遍、随时间漂移。
	// 这里只用它的无缓存方法（ResolveFresh/RolePermCodes），不共享 handler 那份带缓存的实例——
	// 角色/用户管理写操作低频，值得每次现查现算，不依赖 30s 缓存的及时失效。
	rbac *auth.RBAC
}

// New 创建 Service（同时初始化 FCM Manager，加载失败只警告不崩）。
func New(cfg *config.Config, r *repo.Repo, st storage.Storage) *Service {
	// gp 超 Firebase 每项目 30 App 上限被拆成 hybrid-gp + hybrid-gp2，故 FCM 客户端按
	// 「路由键」而非纯品牌组织：ap/bp/gp 同名，外加 gp2 溢出项目。gp2 未配置则其包跳过不发。
	fcmMgr := NewFCMManager(
		map[string]string{
			"ap":  cfg.FirebaseSAAP,
			"bp":  cfg.FirebaseSABP,
			"gp":  cfg.FirebaseSAGP,
			"gp2": cfg.FirebaseSAGP2,
			// 上架包推送独立项目（ColorStack/DeckTallyPro）；未配则其推送整体 Skipped。
			fcmRouteKeyListings: cfg.FirebaseSAListings,
		},
		map[string]string{
			"ap":  cfg.FirebaseProjectAP,
			"bp":  cfg.FirebaseProjectBP,
			"gp":  cfg.FirebaseProjectGP,
			"gp2": cfg.FirebaseProjectGP2,
			fcmRouteKeyListings: cfg.FirebaseProjectListings,
		},
	)
	return &Service{cfg: cfg, repo: r, storage: st, fcm: fcmMgr, rbac: auth.NewRBAC(r)}
}

// SetCountryResolver 注入 GeoIP 解析器（在 main 里 resolver 构造完成后调用）。
// 分开设置而非进 New：resolver 是可选依赖，且 New 的签名被测试复用，保持稳定。
func (s *Service) SetCountryResolver(g CountryResolver) { s.geo = g }

// Error 是带 HTTP 状态码的业务错误，便于 handler 直接映射。
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string { return e.Message }

// NewError 构造业务错误。
func NewError(code int, msg string) *Error { return &Error{Code: code, Message: msg} }

// 常用业务错误构造。
func errBadRequest(msg string) *Error { return &Error{Code: http.StatusBadRequest, Message: msg} }
func errConflict(msg string) *Error   { return &Error{Code: http.StatusConflict, Message: msg} }
func errNotFound(msg string) *Error   { return &Error{Code: http.StatusNotFound, Message: msg} }
func errForbidden(msg string) *Error  { return &Error{Code: http.StatusForbidden, Message: msg} }

// AsError 把任意 error 归一化为 *Error（repo.ErrNotFound → 404，其余 → 500）。
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var se *Error
	if errors.As(err, &se) {
		return se
	}
	if errors.Is(err, repo.ErrNotFound) {
		return errNotFound("资源不存在")
	}
	if errors.Is(err, auth.ErrRoleMissing) {
		return errForbidden("账号未挂角色或角色已被删除，请联系管理员处理")
	}
	return &Error{Code: http.StatusInternalServerError, Message: err.Error()}
}

// repoChannelFilterAllForBrand 构造「某品牌全部非归档渠道」的查询条件。
func repoChannelFilterAllForBrand(brandCode string) repo.ChannelFilter {
	return repo.ChannelFilter{BrandCode: brandCode, PageSize: 500}
}
