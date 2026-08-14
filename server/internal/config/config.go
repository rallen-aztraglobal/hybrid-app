// Package config 负责从环境变量加载后端运行配置。
// 约定：所有配置项都有合理默认值，本地零配置即可 go run 起来（用 SQLite 内存/文件 + 本地磁盘对象存储）。
// 生产用环境变量覆盖（MySQL DSN、MinIO、JWT 秘钥等）。
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 是后端全部运行期配置。
type Config struct {
	// HTTP
	Addr            string        // 监听地址，默认 :8080
	CORSAllowOrigin []string      // 允许的前端来源
	ShutdownTimeout time.Duration // 优雅关闭超时

	// 数据库
	// Driver: "mysql" 走真实 MySQL；"sqlite" 走本地文件，方便本机零依赖启动与测试。
	DBDriver string
	DBDSN    string // mysql: user:pass@tcp(host:port)/db?... ; sqlite: 文件路径或 :memory:

	// 是否在启动时自动建表 + seed（开发便利；生产建议关掉走 golang-migrate）。
	AutoMigrate bool
	AutoSeed    bool

	// SeedDir 是首次部署资产初始化（图标/启动页/res.zip）读取源图的根目录。
	// 镜像里由 Dockerfile.api 构建期 COPY 进 /app/seed（含 channels/*.csv 与各渠道 res/）。
	// 本机开发可指向仓库的 app/src/channels。布局二选一（自动探测）：
	//   ① <SeedDir>/res/<brand>/<flavor>/res/...    + <SeedDir>/channels/*.csv  （镜像精简布局）
	//   ② <SeedDir>/<brand>/<flavor>/res/...                                    （直接指向 app/src/channels）
	SeedDir string

	// 对象存储
	// Kind: "local"（本地磁盘，开发默认）| "minio"（MinIO/S3）。
	StorageKind      string
	StorageLocalDir  string // local: 资源根目录
	StoragePublicURL string // 对外可访问的资源前缀，用于拼 icon_master_url 等

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool

	// JWT
	JWTSecret         string
	JWTAccessTTL      time.Duration
	JWTRefreshTTL     time.Duration
	JWTIssuer         string
	RunnerToken       string // 构建机(runner)长期静态令牌：非空时 /build/* 机器接口接受该 Bearer（机器身份=user，ADR-0008）
	BootstrapAdmin    string // 启动时若无任何用户则建一个 admin，格式 user:password
	DefaultProbePath  string // /api/app/config 返回的 probePath，默认 /healthz
	AppConfigTTLSecs  int    // /api/app/config 的 ttlSeconds
	AppConfigBaseURL  string // APK 烧录的运行时配置端点（APK 用 GET ${base}?appId= 拉域名列表）；空则回退对象存储前缀
	DomainProbeEnable bool   // 是否启用 cron 域名巡检

	// 推送功能（ADR-0012）。
	// PUSH_ENABLED=true 才真正发送；缺少 service account 时门控拦截，campaign 留 draft。
	PushEnabled     bool   // PUSH_ENABLED，默认 false
	PushCronEnable  bool   // PUSH_CRON_ENABLE，默认 false；启用 @every 1m 定时扫描
	FirebaseSAAP    string // FIREBASE_SA_AP：service account JSON 文件路径或 JSON 内容
	FirebaseSABP    string // FIREBASE_SA_BP
	FirebaseSAGP    string // FIREBASE_SA_GP
	FirebaseSAGP2   string // FIREBASE_SA_GP2：gp 溢出项目（超 Firebase 每项目 30 App 上限，gp 拆 hybrid-gp + hybrid-gp2）
	FirebaseProjectAP string // FIREBASE_PROJECT_AP：Firebase 项目 ID
	FirebaseProjectBP string // FIREBASE_PROJECT_BP
	FirebaseProjectGP string // FIREBASE_PROJECT_GP
	FirebaseProjectGP2 string // FIREBASE_PROJECT_GP2：gp 溢出项目 ID（缺失则 gp2 包跳过不发，no-op）
	// 上架包推送用独立 Firebase 项目（装 ColorStack android/ios + DeckTallyPro ios 三个 App）。
	// 缺失则上架包推送整体 no-op（Send 返回 Skipped，不算失败），待运维配好私钥即可发。
	FirebaseSAListings      string // FIREBASE_SA_LISTINGS：service account JSON 路径或内容
	FirebaseProjectListings string // FIREBASE_PROJECT_LISTINGS：项目 ID

	// 上架包 AB 面网关（listing gate）。
	// GeoIPPath：DB-IP 国家库落盘路径。镜像构建期烤一份保底；GEOIP_REFRESH_ENABLE=true 时
	// cron 每月拉新覆盖（DB-IP 免费库无需账号/凭据，可全自动更新，见 internal/geoip）。
	GeoIPPath          string // GEOIP_PATH，默认 /app/geoip/dbip-country-lite.mmdb
	GeoIPRefreshEnable bool   // GEOIP_REFRESH_ENABLE，默认 true：启用 cron 月度自动更新
	// TrustedProxyCIDRs：可信反向代理网段（用于从 X-Forwarded-For 提取真实客户端 IP）。
	// 留空 → 用内置私有网段集合（回环 + RFC1918 + IPv6 ULA），适配同机 nginx+go-api 部署。
	TrustedProxyCIDRs []string // TRUSTED_PROXY_CIDRS，逗号分隔
	// GateLogEnable：是否把每次网关判定落 listing_gate_log（排查用；量大时可关）。
	GateLogEnable bool // GATE_LOG_ENABLE，默认 true
}

// Load 从环境变量装配 Config，缺省值保证本机可零配置启动。
func Load() *Config {
	c := &Config{
		Addr:            env("SERVER_ADDR", ":8080"),
		CORSAllowOrigin: splitCSV(env("CORS_ALLOW_ORIGIN", "http://localhost:5173,http://localhost:3000")),
		ShutdownTimeout: 10 * time.Second,

		DBDriver: env("DB_DRIVER", "sqlite"),
		DBDSN:    env("DB_DSN", "file:hybrid_admin.db?cache=shared&_pragma=foreign_keys(1)"),

		AutoMigrate: envBool("DB_AUTOMIGRATE", true),
		AutoSeed:    envBool("DB_AUTOSEED", true),

		SeedDir: env("SEED_DIR", "/app/seed"),

		StorageKind:      env("STORAGE_KIND", "local"),
		StorageLocalDir:  env("STORAGE_LOCAL_DIR", "./data/objects"),
		StoragePublicURL: env("STORAGE_PUBLIC_URL", "http://localhost:8080/static"),

		MinIOEndpoint:  env("MINIO_ENDPOINT", ""),
		MinIOAccessKey: env("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey: env("MINIO_SECRET_KEY", ""),
		MinIOBucket:    env("MINIO_BUCKET", "hybrid-admin"),
		MinIOUseSSL:    envBool("MINIO_USE_SSL", false),

		JWTSecret:        env("JWT_SECRET", ""),
		JWTAccessTTL:     envDuration("JWT_ACCESS_TTL", 24*time.Hour),
		JWTRefreshTTL:    envDuration("JWT_REFRESH_TTL", 720*time.Hour),
		JWTIssuer:        env("JWT_ISSUER", "hybrid-admin"),
		RunnerToken:      env("RUNNER_TOKEN", ""),
		BootstrapAdmin:   env("BOOTSTRAP_ADMIN", "admin:admin12345"),
		DefaultProbePath: env("APP_PROBE_PATH", "/healthz"),
		AppConfigTTLSecs: envInt("APP_CONFIG_TTL_SECONDS", 600),
		// APK 运行时配置端点基址（烧进 bootstrap.json 的 configUrl）。
		// 形如 https://api.example.com/api/app/config；APK 实际请求 ${base}?appId=<applicationId>。
		// 留空 → 回退旧静态快照前缀（storage.PublicURL），但那样 APK 拉不到实时配置、无法热更域名。
		AppConfigBaseURL: env("APP_CONFIG_BASE_URL", ""),

		DomainProbeEnable: envBool("DOMAIN_PROBE_ENABLE", true),

		// 推送功能（ADR-0012）。
		PushEnabled:       envBool("PUSH_ENABLED", false),
		PushCronEnable:    envBool("PUSH_CRON_ENABLE", false),
		FirebaseSAAP:      env("FIREBASE_SA_AP", ""),
		FirebaseSABP:      env("FIREBASE_SA_BP", ""),
		FirebaseSAGP:      env("FIREBASE_SA_GP", ""),
		FirebaseSAGP2:     env("FIREBASE_SA_GP2", ""),
		FirebaseProjectAP: env("FIREBASE_PROJECT_AP", ""),
		FirebaseProjectBP: env("FIREBASE_PROJECT_BP", ""),
		FirebaseProjectGP: env("FIREBASE_PROJECT_GP", ""),
		FirebaseProjectGP2: env("FIREBASE_PROJECT_GP2", ""),
		FirebaseSAListings:      env("FIREBASE_SA_LISTINGS", ""),
		FirebaseProjectListings: env("FIREBASE_PROJECT_LISTINGS", ""),

		// 上架包 AB 面网关。
		GeoIPPath:          env("GEOIP_PATH", "/app/geoip/dbip-country-lite.mmdb"),
		GeoIPRefreshEnable: envBool("GEOIP_REFRESH_ENABLE", true),
		TrustedProxyCIDRs:  splitCSV(env("TRUSTED_PROXY_CIDRS", "")),
		GateLogEnable:      envBool("GATE_LOG_ENABLE", true),
	}
	// JWT_SECRET 未设置时自动生成随机密钥：兑现 compose「留空→自动生成」的承诺，
	// 同时去掉可被伪造的硬编码弱默认值（评审 S2）。注意：随机密钥重启后失效 → 需重新登录；
	// 生产应在 .env 固定 JWT_SECRET 以保证重启后令牌仍有效。
	if strings.TrimSpace(c.JWTSecret) == "" {
		c.JWTSecret = randomSecret()
		fmt.Fprintln(os.Stderr, "[config] 警告：未设置 JWT_SECRET，已生成随机密钥（重启后所有登录失效）。生产请在 .env 固定 JWT_SECRET。")
	}
	return c
}

// randomSecret 生成 32 字节随机密钥的十六进制串（JWT_SECRET 缺省时兜底）。
func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "insecure-fallback-please-set-JWT_SECRET"
	}
	return hex.EncodeToString(b)
}

// Validate 做一些基本一致性检查。
func (c *Config) Validate() error {
	switch c.DBDriver {
	case "mysql", "sqlite":
	default:
		return fmt.Errorf("不支持的 DB_DRIVER: %s（仅 mysql/sqlite）", c.DBDriver)
	}
	switch c.StorageKind {
	case "local", "minio":
	default:
		return fmt.Errorf("不支持的 STORAGE_KIND: %s（仅 local/minio）", c.StorageKind)
	}
	if c.StorageKind == "minio" && c.MinIOEndpoint == "" {
		return fmt.Errorf("STORAGE_KIND=minio 时必须设置 MINIO_ENDPOINT")
	}
	if c.AppConfigTTLSecs <= 0 {
		return fmt.Errorf("APP_CONFIG_TTL_SECONDS 必须为正")
	}
	return nil
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(k string, def int) int {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(k string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
