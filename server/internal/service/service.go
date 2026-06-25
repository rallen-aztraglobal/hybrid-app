// Package service 是业务逻辑层（handler 薄、service 厚）。
// 聚合渠道 CRUD + 唯一性、域名校验/探测、运行时配置组装、图标 fan-out 编排、构建 manifest。
package service

import (
	"errors"
	"net/http"

	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/storage"
)

// Service 持有依赖，向 handler 暴露用例方法。
type Service struct {
	cfg     *config.Config
	repo    *repo.Repo
	storage storage.Storage
	fcm     *FCMManager
}

// New 创建 Service（同时初始化 FCM Manager，加载失败只警告不崩）。
func New(cfg *config.Config, r *repo.Repo, st storage.Storage) *Service {
	fcmMgr := NewFCMManager(
		map[string]string{
			"ap": cfg.FirebaseSAAP,
			"bp": cfg.FirebaseSABP,
			"gp": cfg.FirebaseSAGP,
		},
		map[string]string{
			"ap": cfg.FirebaseProjectAP,
			"bp": cfg.FirebaseProjectBP,
			"gp": cfg.FirebaseProjectGP,
		},
	)
	return &Service{cfg: cfg, repo: r, storage: st, fcm: fcmMgr}
}

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
	return &Error{Code: http.StatusInternalServerError, Message: err.Error()}
}

// repoChannelFilterAllForBrand 构造「某品牌全部非归档渠道」的查询条件。
func repoChannelFilterAllForBrand(brandCode string) repo.ChannelFilter {
	return repo.ChannelFilter{BrandCode: brandCode, PageSize: 500}
}
