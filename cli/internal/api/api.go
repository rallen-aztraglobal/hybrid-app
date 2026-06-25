// Package api 是后台 HTTP 客户端：登录、拉取打包 manifest、下载 res.zip、回传构建记录。
//
// 契约见 docs/admin/01「5. API」。除 /api/app/* 外均需 Authorization: Bearer <jwt>，
// 响应统一包络 { code, message, data }。
//
// 安全：仅发送非机密数据；构建记录回传严格使用 manifest.BuildRecord（不含 keystore/口令）。
package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hybrid-app/cli/internal/manifest"
)

// Client 封装与后台的交互。
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New 构造客户端。baseURL 末尾斜杠会被去除。
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   strings.TrimSpace(token),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// envelope 是后台统一响应包络。
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) url(p string) string { return c.BaseURL + p }

// do 执行请求并解包 envelope，把 data 解析进 out（out 可为 nil）。
func (c *Client) do(ctx context.Context, method, path string, body any, out any, auth bool) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("编码请求体失败: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), rdr)
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if auth {
		if c.Token == "" {
			return fmt.Errorf("缺少访问令牌，请先登录")
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("请求 %s 失败: %w", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("鉴权失败(401)：token 可能已过期，请重新 login")
	}

	// 尝试按 envelope 解析；非 JSON 或非包络时回退到原始报错。
	var env envelope
	if json.Unmarshal(raw, &env) == nil && (env.Code != 0 || env.Message != "" || len(env.Data) > 0) {
		if resp.StatusCode >= 400 || (env.Code != 0 && env.Code != 200) {
			return fmt.Errorf("后台返回错误(%d/%d): %s", resp.StatusCode, env.Code, env.Message)
		}
		if out != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("解析响应 data 失败: %w", err)
			}
		}
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("后台返回错误(%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}
	return nil
}

// doText 发送纯文本请求体（用于服务端按 io.ReadAll 读取的端点，如追加构建日志）。
// 只校验状态码，不解析响应体；best-effort 语义由调用方决定是否忽略错误。
func (c *Client) doText(ctx context.Context, method, path, text string, auth bool) error {
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), strings.NewReader(text))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	if auth {
		if c.Token == "" {
			return fmt.Errorf("缺少访问令牌，请先登录")
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("请求 %s 失败: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("鉴权失败(401)：token 可能已过期，请重新 login")
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("后台返回错误(%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

// LoginRequest / LoginResponse 对应 POST /api/auth/login。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	Username     string `json:"username,omitempty"`
}

// Login 账号密码登录，返回 token。
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	var out LoginResponse
	if err := c.do(ctx, http.MethodPost, "/api/auth/login", LoginRequest{Username: username, Password: password}, &out, false); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("登录响应未包含 accessToken")
	}
	if out.Username == "" {
		out.Username = username
	}
	return &out, nil
}

// serverManifest 映射后端 service.BuildManifest 的 JSON 形状。
// 后端字段名与 CLI 的 manifest.Manifest 不一致（flavorName/configBaseUrl/effectiveDomains），
// 直接解进 manifest.Manifest 会得到空 Flavor → 渲染出错。故先解进本结构再转换（评审 C6）。
type serverManifest struct {
	Brand         string   `json:"brand"`
	BrandDomains  []string `json:"brandDomains"`
	ConfigBaseURL string   `json:"configBaseUrl"`
	Channels      []struct {
		FlavorName       string   `json:"flavorName"`
		ApplicationID    string   `json:"applicationId"`
		PalCode          string   `json:"palCode"`
		AppName          string   `json:"appName"`
		EffectiveDomains []string `json:"effectiveDomains"`
		ResZipURL        string   `json:"resZipUrl"`
	} `json:"channels"`
}

// Manifest 拉取某品牌的打包清单：GET /api/build/manifest?brand=ap。
// 解析后端形状并转换为 CLI 渲染所用的 manifest.Manifest（评审 C6）。
func (c *Client) Manifest(ctx context.Context, brand string) (*manifest.Manifest, error) {
	var sm serverManifest
	p := "/api/build/manifest?brand=" + url.QueryEscape(brand)
	if err := c.do(ctx, http.MethodGet, p, nil, &sm, true); err != nil {
		return nil, err
	}
	if sm.Brand == "" {
		sm.Brand = brand
	}
	m := &manifest.Manifest{
		Brand:        sm.Brand,
		ConfigURL:    sm.ConfigBaseURL,
		BrandDomains: sm.BrandDomains,
	}
	for _, ch := range sm.Channels {
		m.Channels = append(m.Channels, manifest.Channel{
			Flavor:          ch.FlavorName,
			ApplicationId:   ch.ApplicationID,
			PalCode:         ch.PalCode,
			AppName:         ch.AppName,
			UseBrandDomains: false, // 后端已给出合并后的 effectiveDomains
			Domains:         ch.EffectiveDomains,
			ResZipURL:       ch.ResZipURL,
		})
	}
	return m, nil
}

// DownloadResZip 下载 res.zip 到内存，并按 sha256 校验（expectedSHA 为空则跳过校验）。
// rawURL 可以是绝对地址（对象存储直链），也可以是后台相对路径。
func (c *Client) DownloadResZip(ctx context.Context, rawURL, expectedSHA string) ([]byte, error) {
	full := rawURL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		full = c.url(rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, fmt.Errorf("构造下载请求失败: %w", err)
	}
	// 对象存储直链通常无需鉴权；仅当走后台相对路径时带上 token。
	if c.Token != "" && full == c.url(rawURL) {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 res.zip 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("下载 res.zip 返回 %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
	if err != nil {
		return nil, fmt.Errorf("读取 res.zip 失败: %w", err)
	}
	if expectedSHA != "" {
		sum := sha256.Sum256(data)
		got := hex.EncodeToString(sum[:])
		if !strings.EqualFold(got, strings.TrimSpace(expectedSHA)) {
			return nil, fmt.Errorf("res.zip 校验失败: 期望 %s 实得 %s", expectedSHA, got)
		}
	}
	return data, nil
}

// PostBuildRecord 回传构建记录：POST /api/build/records。
// 返回后台分配的记录 ID（若有）。
func (c *Client) PostBuildRecord(ctx context.Context, rec manifest.BuildRecord) (int64, error) {
	var out struct {
		ID int64 `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/build/records", rec, &out, true); err != nil {
		return 0, err
	}
	return out.ID, nil
}

// FetchGoogleServicesJSON 拉取指定品牌的合并 google-services.json：
// GET /api/app/google-services?brand=<ap|bp|gp>
//
// 「有则拉、无则跳过」语义：
//   - 后端返回 404 → 视为「该品牌尚未配置 FCM」，返回 (nil, nil)。
//   - 后端返回其他 4xx/5xx → 同样宽容处理，返回 (nil, nil) 而非错误，
//     以免 FCM 未就绪期间阻断 pull 流程（ADR-0012 第 6b 节）。
//   - 成功则返回 JSON 字节，由调用方写到 app/google-services.json。
func (c *Client) FetchGoogleServicesJSON(ctx context.Context, brand string) ([]byte, error) {
	path := "/api/app/google-services?brand=" + url.QueryEscape(brand)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return nil, fmt.Errorf("构造 google-services 请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		// 网络异常：宽容跳过，不阻断打包。
		return nil, nil
	}
	defer resp.Body.Close()
	// 404 或后端尚未实现该端点 → FCM 未配置，跳过。
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNotImplemented {
		return nil, nil
	}
	// 其他非 2xx → 也宽容跳过，记录在调用方日志中。
	if resp.StatusCode >= 400 {
		return nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("读取 google-services.json 失败: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	return data, nil
}

// Ping 探测后台连通性（用于 doctor）。优先打健康端点，失败再退到根路径。
func (c *Client) Ping(ctx context.Context) error {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, p := range []string{"/healthz", "/api/healthz", "/"} {
		req, err := http.NewRequestWithContext(cctx, http.MethodGet, c.url(p), nil)
		if err != nil {
			continue
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 500 {
			return nil
		}
	}
	return fmt.Errorf("无法连通后台 %s", c.BaseURL)
}
