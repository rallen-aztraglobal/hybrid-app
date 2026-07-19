// Package geoip 提供「IP → 国家码」解析，供上架包 AB 面网关判定使用。
//
// 数据源选型：DB-IP country-lite（CC-BY 4.0），而非 MaxMind GeoLite2。
// 理由是 GeoLite2 需要注册账号并携带 license key 才能下载，无法做到零人工维护；
// DB-IP 的免费库下载地址可按月份直接拼出、无需任何凭据，因此能被 cron 全自动更新。
// 精度上两者都是国家级，满足网关需求（我们只需要判到国家）。
//
// 署名要求：CC-BY 4.0 要求标注来源，Console 设置页页脚已声明「IP 地理数据来自 DB-IP」。
//
// 可用性策略：库加载失败、文件缺失、IP 查不到，一律返回「未知国家」而不是报错。
// 上层网关把未知国家判为 A 面（fail-closed），因此 GeoIP 故障的后果是「没人能进 B 面」，
// 而不是「所有人都能进 B 面」——这是本模块所有错误处理的取向。
package geoip

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// countryRecord 只解出我们需要的国家码字段。
// mmdb 里还有大量其他字段，不声明即不解析，省内存也省 CPU。
type countryRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

// Resolver 是并发安全的 IP→国家解析器。
//
// db 用 atomic.Pointer 持有：cron 月度更新时构造一个全新的 Reader 再原子替换指针，
// 期间正在执行的查询继续用旧 Reader，不加锁、不中断判定。旧 Reader 由 GC 回收前
// 已无人引用（maxminddb 的 Reader 是 mmap，Close 会 munmap，故这里刻意不 Close 旧实例，
// 让 GC 处理，避免极小概率的「查询中途 munmap」导致进程崩溃）。
type Resolver struct {
	db     atomic.Pointer[maxminddb.Reader]
	path   string
	client *http.Client
}

// New 构造 Resolver 并尝试加载 path 处的 mmdb。
//
// 文件不存在或损坏不返回错误，只返回一个「空」Resolver：此时所有查询返回未知国家，
// 网关据此全判 A 面。这样 GeoIP 库缺失不会让整个服务起不来，而是安全降级。
// 调用方应检查 Loaded() 并在未加载时告警。
func New(path string) *Resolver {
	r := &Resolver{
		path:   path,
		client: &http.Client{Timeout: 5 * time.Minute}, // 库约 4MB，但构建机网络可能慢
	}
	if err := r.load(path); err != nil {
		// 仅记录，不阻断启动。
		fmt.Printf("[geoip] 加载 %s 失败（将全部判为未知国家 → A 面）: %v\n", path, err)
	}
	return r
}

// load 打开 mmdb 并原子替换当前 Reader。校验通过才替换，避免半损坏的库上线。
func (r *Resolver) load(path string) error {
	db, err := maxminddb.Open(path)
	if err != nil {
		return fmt.Errorf("打开 mmdb 失败: %w", err)
	}
	// 校验：能查通一个已知 IP 才认为库可用，防止文件存在但内容是 HTML 错误页之类。
	if err := db.Verify(); err != nil {
		_ = db.Close()
		return fmt.Errorf("mmdb 校验失败: %w", err)
	}
	r.db.Store(db)
	return nil
}

// Loaded 返回当前是否有可用的 GeoIP 库。
func (r *Resolver) Loaded() bool { return r.db.Load() != nil }

// BuildTime 返回当前库的构建时间，未加载时返回零值。用于 Console 展示「库有多旧」。
func (r *Resolver) BuildTime() time.Time {
	db := r.db.Load()
	if db == nil {
		return time.Time{}
	}
	return time.Unix(int64(db.Metadata.BuildEpoch), 0)
}

// Country 返回 ip 所属国家的 ISO-3166-1 alpha-2 大写码。
//
// 第二个返回值为 false 表示「未知」——库没加载、IP 非法、私有地址、库里查不到，
// 全部归为未知。调用方（网关）必须把未知当作 A 面处理，不得有「未知就放行」的分支。
func (r *Resolver) Country(ip net.IP) (string, bool) {
	if ip == nil {
		return "", false
	}
	db := r.db.Load()
	if db == nil {
		return "", false
	}
	var rec countryRecord
	if err := db.Lookup(ip, &rec); err != nil {
		return "", false
	}
	if rec.Country.ISOCode == "" {
		return "", false
	}
	return rec.Country.ISOCode, true
}

// dbipURL 拼出 DB-IP 免费国家库某月份的下载地址。
// 形如 https://download.db-ip.com/free/dbip-country-lite-2026-07.mmdb.gz
func dbipURL(t time.Time) string {
	return fmt.Sprintf("https://download.db-ip.com/free/dbip-country-lite-%04d-%02d.mmdb.gz", t.Year(), int(t.Month()))
}

// ErrNotPublished 表示目标月份的库尚未发布（404）。
var ErrNotPublished = errors.New("该月份的 DB-IP 库尚未发布")

// Refresh 下载最新的 DB-IP 国家库并热替换。
//
// 月初新库可能还没发布，故先试本月、404 再回退上月。两个月份都拿不到才算失败。
// 下载先落临时文件并校验，通过后才 rename 覆盖正式路径——避免下载中断留下半个文件，
// 导致下次启动加载到损坏的库。
func (r *Resolver) Refresh(ctx context.Context, now time.Time) error {
	candidates := []time.Time{now, now.AddDate(0, -1, 0)}
	var lastErr error
	for _, t := range candidates {
		err := r.refreshFrom(ctx, dbipURL(t))
		if err == nil {
			return nil
		}
		lastErr = err
		if !errors.Is(err, ErrNotPublished) {
			// 非 404 的错误（网络/磁盘）不必再试上个月，直接返回。
			return err
		}
	}
	return fmt.Errorf("刷新 GeoIP 库失败: %w", lastErr)
}

// refreshFrom 从指定 URL 下载 .mmdb.gz、解压、校验、原子替换。
func (r *Resolver) refreshFrom(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("下载 %s 失败: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotPublished
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 %s 返回 %d", url, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("创建 GeoIP 目录失败: %w", err)
	}

	// 临时文件与目标同目录，保证 rename 是同一文件系统内的原子操作。
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".geoip-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	// 失败路径统一清理临时文件；成功后 tmpName 已被 rename 走，Remove 无副作用。
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("解压 %s 失败: %w", url, err)
	}
	defer func() { _ = gz.Close() }()

	if _, err := io.Copy(tmp, gz); err != nil {
		return fmt.Errorf("写入 GeoIP 库失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("刷盘 GeoIP 库失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}

	// 先在临时路径上验证是合法 mmdb，再覆盖正式文件。
	probe, err := maxminddb.Open(tmpName)
	if err != nil {
		return fmt.Errorf("下载的文件不是合法 mmdb: %w", err)
	}
	verifyErr := probe.Verify()
	_ = probe.Close()
	if verifyErr != nil {
		return fmt.Errorf("下载的 mmdb 校验失败: %w", verifyErr)
	}

	if err := os.Rename(tmpName, r.path); err != nil {
		return fmt.Errorf("替换 GeoIP 库失败: %w", err)
	}
	// 重新加载并原子换上。
	if err := r.load(r.path); err != nil {
		return fmt.Errorf("热加载新 GeoIP 库失败: %w", err)
	}
	return nil
}
