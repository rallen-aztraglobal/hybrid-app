// Package auth 提供 JWT 签发/校验、密码哈希与基于角色的访问控制（RBAC）。
// 角色：admin（全部）> operator（写渠道/图标/域名、构建）> viewer（只读）。
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/hybrid-app/server/internal/model"
)

// contextKey 用于在 echo.Context 里存放鉴权信息。
const (
	ctxClaims = "auth.claims"
)

// TokenType 区分 access / refresh。
const (
	TokenAccess  = "access"
	TokenRefresh = "refresh"
)

// Claims 是 JWT 载荷。
type Claims struct {
	UserID   uint64 `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Type     string `json:"typ"`
	jwt.RegisteredClaims
}

// Manager 负责签发与解析 token。
type Manager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	// RunnerToken 构建机长期静态令牌（ADR-0008）：非空时，Middleware 接受等于它的 Bearer，
	// 并注入「机器 operator」身份直接放行。用于 hybrid-pack runner 常驻轮询（避免 2h access token 过期）。
	RunnerToken string
}

// NewManager 创建鉴权管理器。
func NewManager(secret, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), issuer: issuer, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// HashPassword 用 bcrypt 哈希密码。
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("密码哈希失败: %w", err)
	}
	return string(b), nil
}

// CheckPassword 校验明文与哈希是否匹配。
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Issue 为用户签发一对 access + refresh token。
func (m *Manager) Issue(u *model.AdminUser) (access, refresh string, err error) {
	access, err = m.sign(u, TokenAccess, m.accessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = m.sign(u, TokenRefresh, m.refreshTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (m *Manager) sign(u *model.AdminUser, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		Type:     typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   u.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("签发 token 失败: %w", err)
	}
	return s, nil
}

// Parse 解析并校验 token。
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析 token 失败: %w", err)
	}
	if !tok.Valid {
		return nil, errors.New("token 无效")
	}
	return claims, nil
}

// AccessTTLSeconds 返回 access token 的有效期秒数（供登录响应里告知前端）。
func (m *Manager) AccessTTLSeconds() int { return int(m.accessTTL.Seconds()) }

// Middleware 解析 Bearer token，校验为 access 类型，把 Claims 放入 context。
func (m *Manager) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authz := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authz, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "缺少 Bearer token")
			}
			raw := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
			// 构建机静态令牌：非空且匹配时，注入机器 operator 身份直接放行（ADR-0008 runner 长期凭证）。
			// 它无过期、不依赖登录，专供 /build/* 机器接口；其余路由仍受 RequireRole 约束（机器=operator）。
			if m.RunnerToken != "" && raw == m.RunnerToken {
				c.Set(ctxClaims, &Claims{Username: "runner", Role: model.RoleOperator, Type: TokenAccess})
				return next(c)
			}
			claims, err := m.Parse(raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "token 校验失败")
			}
			if claims.Type != TokenAccess {
				return echo.NewHTTPError(http.StatusUnauthorized, "需要 access token")
			}
			c.Set(ctxClaims, claims)
			return next(c)
		}
	}
}

// roleRank 用于角色比较。数值越大权限越高。
var roleRank = map[string]int{
	model.RoleViewer:   1,
	model.RoleOperator: 2,
	model.RoleAdmin:    3,
}

// RequireRole 返回一个中间件，要求当前用户角色 >= minRole。
func RequireRole(minRole string) echo.MiddlewareFunc {
	need := roleRank[minRole]
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := FromContext(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "未鉴权")
			}
			if roleRank[claims.Role] < need {
				return echo.NewHTTPError(http.StatusForbidden, "权限不足")
			}
			return next(c)
		}
	}
}

// FromContext 取出当前请求的 Claims，未鉴权返回 nil。
func FromContext(c echo.Context) *Claims {
	v, ok := c.Get(ctxClaims).(*Claims)
	if !ok {
		return nil
	}
	return v
}
