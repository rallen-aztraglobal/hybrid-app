// Package auth 提供 JWT 签发/校验、密码哈希，以及基于权限点的访问控制（RBAC，见 rbac.go）。
// 权限模型：用户挂角色、角色挂一组权限点 code，唯一契约见 docs/admin/10-rbac.md（internal/perm 静态定义）。
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

// TokenType 区分 access / refresh / runner。
// TokenRunner 只由 Middleware 在校验构建机静态令牌后注入，普通签发路径（sign）永远不会产生这个
// type——机器身份的判定必须靠这个不可伪造字段，不能靠 Username（否则任何人注册一个用户名叫
// "runner" 的普通账号，其正常签发的 access token 也会被误判为机器身份，见 B1 修复）。
const (
	TokenAccess  = "access"
	TokenRefresh = "refresh"
	TokenRunner  = "runner"
	// TokenScoped 是一次性/短时效用途令牌（如设备 CSV 导出下载链接）：只携带 Scope 声明的单一
	// 用途，不是 access token，不能用于任何 RequirePerm 保护的接口（Middleware 只放行 TokenAccess）。
	TokenScoped = "scoped"
)

// Claims 是 JWT 载荷。Scope 仅 TokenScoped 类型使用，标识该令牌唯一被允许的用途
// （如 "device_export"），校验时必须与期望值逐字匹配。
type Claims struct {
	UserID   uint64 `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Type     string `json:"typ"`
	Scope    string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

// Manager 负责签发与解析 token。
type Manager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	// RunnerToken 构建机长期静态令牌（ADR-0008）：非空时，Middleware 接受等于它的 Bearer，
	// 并注入 runner 机器身份直接放行。用于 hybrid-pack runner 常驻轮询（避免 2h access token 过期）。
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

// IssueScopedToken 签发一个短时效、单一用途的令牌（不挂用户身份，不是 access token）。
// 例如设备 CSV 导出下载链接：前端先用 access token 换一个 5 分钟有效的 scoped token 拼进
// 下载 URL，避免 access token（长效、权限更广）出现在可能被日志/浏览器历史记录的 URL 里。
func (m *Manager) IssueScopedToken(scope string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Type:  TokenScoped,
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("签发 scoped token 失败: %w", err)
	}
	return s, nil
}

// VerifyScopedToken 解析并校验一个 scoped token：类型必须是 TokenScoped 且 Scope 与期望值
// 逐字匹配，否则返回错误——普通 access token 不能冒充 scoped token 使用。
func (m *Manager) VerifyScopedToken(tokenStr, wantScope string) (*Claims, error) {
	claims, err := m.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != TokenScoped {
		return nil, errors.New("需要 scoped token")
	}
	if claims.Scope != wantScope {
		return nil, fmt.Errorf("token scope 不匹配：期望 %s", wantScope)
	}
	return claims, nil
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
			// 构建机静态令牌：非空且匹配时，注入 Type=TokenRunner 的机器身份直接放行（ADR-0008 runner 长期凭证）。
			// 它无过期、不依赖登录；机器身份的判定字段是 Type（不可伪造——普通签发路径不会产生
			// TokenRunner），不是 Username，具体只放行哪些接口由 RequirePerm 判定（见 rbac.go），
			// 本中间件只负责识别身份。
			if m.RunnerToken != "" && raw == m.RunnerToken {
				c.Set(ctxClaims, &Claims{Username: RunnerUsername, Type: TokenRunner})
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

// FromContext 取出当前请求的 Claims，未鉴权返回 nil。
func FromContext(c echo.Context) *Claims {
	v, ok := c.Get(ctxClaims).(*Claims)
	if !ok {
		return nil
	}
	return v
}
