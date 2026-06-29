// Package service — FCM HTTP v1 REST 发送（不引 Admin SDK，ADR-0012 / ADR-0001）。
// 发送路径由 PUSH_ENABLED + service account 双门控；缺失账号不报错退出，仅标记「未配置」。
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2/google"
)

// fcmScopes FCM HTTP v1 需要的 OAuth2 scope。
var fcmScopes = []string{"https://www.googleapis.com/auth/firebase.messaging"}

// BrandFCMClient 持有单个品牌的 FCM 凭证与项目 ID。
type BrandFCMClient struct {
	creds     *google.Credentials
	projectID string
}

// FCMManager 持有三个品牌的 FCM 客户端（ap/bp/gp）。
// 启动时尝试加载；加载失败不崩，只标记「未配置」。
type FCMManager struct {
	clients    map[string]*BrandFCMClient // brand_code → client
	httpClient *http.Client
}

// NewFCMManager 从配置文件路径或 JSON 内容加载 3 个品牌的 service account。
// 品牌名 → (saJSON, projectID)。加载失败的品牌标记为「未配置」但不终止启动（ADR-0012 §6b）。
func NewFCMManager(brandSA map[string]string, brandProject map[string]string) *FCMManager {
	m := &FCMManager{
		clients:    make(map[string]*BrandFCMClient),
		httpClient: &http.Client{},
	}
	for brand, sa := range brandSA {
		if sa == "" {
			continue
		}
		projectID := brandProject[brand]
		if projectID == "" {
			continue
		}
		saJSON, err := resolveSAJSON(sa)
		if err != nil {
			// 加载失败只警告，不崩（ADR-0012 §6b 特性门控）。
			fmt.Fprintf(os.Stderr, "[fcm] 品牌 %s service account 加载失败（暂不可发送）: %v\n", brand, err)
			continue
		}
		creds, err := google.CredentialsFromJSON(context.Background(), saJSON, fcmScopes...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[fcm] 品牌 %s google.Credentials 解析失败: %v\n", brand, err)
			continue
		}
		m.clients[brand] = &BrandFCMClient{creds: creds, projectID: projectID}
		fmt.Fprintf(os.Stderr, "[fcm] 品牌 %s FCM 已配置（project=%s）\n", brand, projectID)
	}
	return m
}

// resolveSAJSON 把 FIREBASE_SA_* 的值（文件路径或 JSON 字符串）解析为 JSON 字节。
func resolveSAJSON(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "{") {
		// 内容直接就是 JSON。
		return []byte(raw), nil
	}
	// 视为文件路径。
	data, err := os.ReadFile(raw)
	if err != nil {
		return nil, fmt.Errorf("读取 service account 文件 %q 失败: %w", raw, err)
	}
	return data, nil
}

// IsConfigured 返回指定品牌是否已配置 FCM service account。
func (m *FCMManager) IsConfigured(brand string) bool {
	_, ok := m.clients[brand]
	return ok
}

// ConfiguredBrands 返回每个已知品牌是否已配置的 map（ap/bp/gp）。
func (m *FCMManager) ConfiguredBrands() map[string]bool {
	out := map[string]bool{
		"ap": m.IsConfigured("ap"),
		"bp": m.IsConfigured("bp"),
		"gp": m.IsConfigured("gp"),
	}
	return out
}

// fcmMessage 是 FCM HTTP v1 单条消息体。
type fcmMessage struct {
	Message struct {
		Token        string         `json:"token"`
		Notification *fcmNotif      `json:"notification,omitempty"`
		Data         map[string]string `json:"data,omitempty"`
	} `json:"message"`
}

type fcmNotif struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	ImageURL string `json:"image,omitempty"`
}

// SendResult 单条发送结果。
type SendResult struct {
	Token string
	Err   error
	// Unregistered=true 时 APK 端 token 已失效，应置 is_active=false。
	Unregistered bool
	// Skipped=true 表示该 token 所属路由项目未配置 FCM（如 gp 溢出项目 gp2 暂无私钥）：
	// 不发送、不算失败、不下线 token——「在线但惰性」（ADR-0012 §6b）。
	Skipped bool
}

// Send 向单个设备 token 发送 FCM 消息（HTTP v1）。
// routeKey 为路由键（ap/bp/gp/gp2 等），通常等于品牌，gp 拆分后溢出包路由到 gp2。
// 返回 SendResult；调用方在 worker pool 中汇总结果。
func (m *FCMManager) Send(ctx context.Context, routeKey, token, title, body, imageURL string, data map[string]string) SendResult {
	cli, ok := m.clients[routeKey]
	if !ok {
		// 该路由项目未配置私钥（典型：gp2 尚无私钥）→ 跳过，不当失败处理。
		return SendResult{Token: token, Skipped: true}
	}

	// 取 OAuth2 access token（自动缓存/刷新）。
	ts := cli.creds.TokenSource
	t, err := ts.Token()
	if err != nil {
		return SendResult{Token: token, Err: fmt.Errorf("获取 FCM access token 失败: %w", err)}
	}

	msg := fcmMessage{}
	msg.Message.Token = token
	if title != "" || body != "" {
		msg.Message.Notification = &fcmNotif{
			Title:    title,
			Body:     body,
			ImageURL: imageURL,
		}
	}
	if len(data) > 0 {
		msg.Message.Data = data
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return SendResult{Token: token, Err: fmt.Errorf("序列化 FCM 消息失败: %w", err)}
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", cli.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return SendResult{Token: token, Err: fmt.Errorf("构造 FCM 请求失败: %w", err)}
	}
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return SendResult{Token: token, Err: fmt.Errorf("FCM HTTP 请求失败: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return SendResult{Token: token}
	}

	// 解析 FCM 错误体。
	bodyBytes, _ := io.ReadAll(resp.Body)
	errBody := string(bodyBytes)

	// UNREGISTERED / INVALID_ARGUMENT → token 已失效，标记下线。
	if strings.Contains(errBody, "UNREGISTERED") || strings.Contains(errBody, "INVALID_ARGUMENT") {
		return SendResult{Token: token, Err: fmt.Errorf("FCM token 失效: %s", errBody), Unregistered: true}
	}

	return SendResult{Token: token, Err: fmt.Errorf("FCM 发送失败(status=%d): %s", resp.StatusCode, errBody)}
}
