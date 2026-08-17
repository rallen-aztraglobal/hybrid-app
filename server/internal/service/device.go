// Package service — 渠道设备上报 + 设备管理列表 + CSV 导出。
package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// DeviceExportScope 是设备 CSV 导出 scoped token 的唯一用途标识（auth.Manager.IssueScopedToken /
// VerifyScopedToken 的 scope 参数），前后端约定的下载链接令牌只能干这一件事。
const DeviceExportScope = "device_export"

// deviceDateLayout 是 API 契约里 from/to 查询参数的日期格式。
const deviceDateLayout = "2006-01-02"

// deviceMaxOffset 是分页深度上限：超出后拒绝翻页，引导使用导出接口（CSV 导出不分页，走游标）。
const deviceMaxOffset = 100000

// DeviceExportMaxIDs 是「勾选 id 导出」单次请求允许的最大 id 数量（handler 层校验请求体时复用）。
const DeviceExportMaxIDs = 1000

// ---------- 类型定义（与 API 契约对齐） ----------

// RegisterDeviceInput POST /api/app/device/register 请求体。
type RegisterDeviceInput struct {
	AppID      string `json:"appId"`
	PalCode    string `json:"palcode"`
	DeviceKey  string `json:"deviceKey"`
	DeviceName string `json:"deviceName"`
	GAID       string `json:"gaid"`
	ADID       string `json:"adid"`
	OAID       string `json:"oaid"`
}

// DeviceView GET /api/devices 单条设备记录（camelCase 与其余 handler 风格一致）。
type DeviceView struct {
	ID            uint64    `json:"id"`
	DeviceKey     string    `json:"deviceKey"`
	DeviceName    string    `json:"deviceName"`
	GAID          string    `json:"gaid"`
	ADID          string    `json:"adid"`
	OAID          string    `json:"oaid"`
	ApplicationID string    `json:"applicationId"`
	AppName       string    `json:"appName"`
	PalCode       string    `json:"palCode"`
	BrandCode     string    `json:"brandCode"`
	CreatedAt     time.Time `json:"createdAt"`
}

// DeviceListResult GET /api/devices 返回。
type DeviceListResult struct {
	Items []DeviceView `json:"items"`
	Total int64        `json:"total"`
}

// ListDevicesInput GET /api/devices 查询参数 / 导出接口共用的筛选条件。
//
// Scope* 是数据权限过滤（docs/admin/10-rbac.md「数据权限」节）：opt-in，零值不生效
// （ScopeRestricted=false，等价不限），兼容不涉及数据权限的既有调用点。handler 层用
// applyDeviceInputScope 从调用者的 auth.Scope populate 这几个字段。
type ListDevicesInput struct {
	ApplicationID string
	From          string // YYYY-MM-DD，含
	To            string // YYYY-MM-DD，含（内部转为 [from, to+1d) 半开区间）
	Page          int
	PageSize      int

	ScopeRestricted  bool
	ScopeAllBrands   bool
	ScopeBrandCodes  []string
	ScopeAllChannels bool
	ScopeChannelIDs  []uint64
}

// ---------- 上报 ----------

// RegisterDevice APK 上报设备信息（公开端点）。
// 归一化：gaid/adid/oaid 统一小写；各字段按列宽截断。
// 只接受带有效 GAID 的上报：GAID 是受众导出的唯一有效标识，空值/全 0（opt-out）直接拒收，
// 客户端同样约定「拿不到 GAID 整条不上报」，这里是防御性双保险。
// applicationId 必须对应一个存在的渠道（防滥用，与 push 上报同风格）。
func (s *Service) RegisterDevice(ctx context.Context, in RegisterDeviceInput) error {
	appID := strings.TrimSpace(in.AppID)
	deviceKey := strings.TrimSpace(in.DeviceKey)
	if appID == "" || deviceKey == "" {
		return errBadRequest("appId 与 deviceKey 不得为空")
	}
	if len([]rune(deviceKey)) > 64 {
		return errBadRequest("deviceKey 过长（最长 64 字符）")
	}
	if normalizeAdID(in.GAID) == "" {
		return errBadRequest("gaid 为空或无效（全 0），不接受上报")
	}

	ch, err := s.repo.GetChannelByApplicationID(ctx, appID)
	if err != nil {
		return errBadRequest("applicationId 对应渠道不存在")
	}
	brandCode := ""
	if ch.Brand != nil {
		brandCode = ch.Brand.Code
	}

	d := &model.ChannelDevice{
		DeviceKey:     deviceKey,
		ApplicationID: appID,
		BrandCode:     truncateRunes(brandCode, 16),
		PalCode:       truncateRunes(strings.TrimSpace(in.PalCode), 64),
		AppName:       truncateRunes(ch.AppName, 128),
		DeviceName:    truncateRunes(strings.TrimSpace(in.DeviceName), 128),
		GAID:          normalizeAdID(in.GAID),
		ADID:          normalizeAdID(in.ADID),
		OAID:          normalizeAdID(in.OAID),
	}
	if err := s.repo.UpsertChannelDevice(ctx, d); err != nil {
		return fmt.Errorf("注册设备失败: %w", err)
	}
	return nil
}

// zeroGAID 是 Google Play Services 在用户关闭广告个性化（opt-out）时返回的全 0 GAID。
const zeroGAID = "00000000-0000-0000-0000-000000000000"

// normalizeAdID 统一小写、trim、按列宽截断；全 0 GAID 归一化为空串。
func normalizeAdID(raw string) string {
	v := truncateRunes(strings.ToLower(strings.TrimSpace(raw)), 64)
	if v == zeroGAID {
		return ""
	}
	return v
}

// truncateRunes 按 rune（而非字节）截断，避免切碎多字节 UTF-8 字符。
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// ---------- 列表 ----------

// ListDevices 分页/筛选设备列表。offset 过深（page 翻太远）拒绝，引导使用导出接口。
func (s *Service) ListDevices(ctx context.Context, in ListDevicesInput) (*DeviceListResult, error) {
	f, err := buildDeviceFilter(in.ApplicationID, in.From, in.To)
	if err != nil {
		return nil, err
	}
	if err := s.applyListDevicesScope(ctx, &f, in); err != nil {
		return nil, err
	}

	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	if (page-1)*pageSize > deviceMaxOffset {
		return nil, errBadRequest("翻页过深，请缩小筛选范围或使用导出")
	}
	f.Page = page
	f.PageSize = pageSize

	list, total, err := s.repo.ListChannelDevices(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]DeviceView, 0, len(list))
	for _, d := range list {
		items = append(items, deviceView(d))
	}
	return &DeviceListResult{Items: items, Total: total}, nil
}

func deviceView(d model.ChannelDevice) DeviceView {
	return DeviceView{
		ID:            d.ID,
		DeviceKey:     d.DeviceKey,
		DeviceName:    d.DeviceName,
		GAID:          d.GAID,
		ADID:          d.ADID,
		OAID:          d.OAID,
		ApplicationID: d.ApplicationID,
		AppName:       d.AppName,
		PalCode:       d.PalCode,
		BrandCode:     d.BrandCode,
		CreatedAt:     d.CreatedAt,
	}
}

// applyListDevicesScope 把 in 的 Scope* 字段写入 f。渠道维度需要把角色范围里的 channel id
// 解析成 application_id（channel_device 表按 application_id 存储，不是 channel_id），
// 故需要 ctx + repo 访问，不能像 buildDeviceFilter 那样是纯函数。
func (s *Service) applyListDevicesScope(ctx context.Context, f *repo.DeviceFilter, in ListDevicesInput) error {
	if !in.ScopeRestricted {
		return nil
	}
	f.ScopeRestricted = true
	f.ScopeAllBrands = in.ScopeAllBrands
	f.ScopeBrandCodes = in.ScopeBrandCodes
	f.ScopeAllChannels = in.ScopeAllChannels
	if !in.ScopeAllChannels {
		appIDs, err := s.repo.ApplicationIDsByChannelIDs(ctx, in.ScopeChannelIDs)
		if err != nil {
			return err
		}
		f.ScopeAppIDs = appIDs
	}
	return nil
}

// buildDeviceFilter 把 API 层的字符串日期参数解析为 repo.DeviceFilter 的半开区间
// [from 00:00:00, to+1d 00:00:00)。
func buildDeviceFilter(appID, from, to string) (repo.DeviceFilter, error) {
	f := repo.DeviceFilter{ApplicationID: strings.TrimSpace(appID)}
	if from != "" {
		t, err := time.Parse(deviceDateLayout, from)
		if err != nil {
			return f, errBadRequest("from 格式应为 YYYY-MM-DD")
		}
		f.From = &t
	}
	if to != "" {
		t, err := time.Parse(deviceDateLayout, to)
		if err != nil {
			return f, errBadRequest("to 格式应为 YYYY-MM-DD")
		}
		exclusive := t.Add(24 * time.Hour)
		f.To = &exclusive
	}
	return f, nil
}

// ---------- CSV 导出 ----------

// deviceCSVHeader 是导出 CSV 的固定列顺序，两个导出端点共用。
var deviceCSVHeader = []string{
	"device_name", "gaid", "gaid_sha256", "adid", "oaid",
	"app_name", "palcode", "application_id", "brand", "created_at",
}

// csvBOM 是 UTF-8 BOM，写在 CSV 最前面，确保 Excel 打开时不会把中文渠道名/设备名识别成乱码。
var csvBOM = []byte{0xEF, 0xBB, 0xBF}

// deviceCSVRow 是单行导出数据（与 deviceCSVHeader 顺序对齐，created_at 单独格式化）。
type deviceCSVRow struct {
	DeviceName    string
	GAID          string
	ADID          string
	OAID          string
	AppName       string
	PalCode       string
	ApplicationID string
	BrandCode     string
	CreatedAt     time.Time
}

// deviceCSVSource 是导出行游标的最小接口：筛选导出用 sql.Rows 包一层，勾选 id 导出用内存
// 切片包一层，streamDeviceCSV 对两者一视同仁。
type deviceCSVSource interface {
	Next() bool
	Scan() (deviceCSVRow, error)
	Err() error
	Close() error
}

// sqlRowsDeviceSource 包装 repo.ExportDeviceRows 返回的 *sql.Rows（列顺序见
// repo.deviceExportColumns：device_name, gaid, adid, oaid, app_name, pal_code, application_id,
// brand_code, created_at）。
type sqlRowsDeviceSource struct {
	rows *sql.Rows
}

func (s *sqlRowsDeviceSource) Next() bool   { return s.rows.Next() }
func (s *sqlRowsDeviceSource) Err() error   { return s.rows.Err() }
func (s *sqlRowsDeviceSource) Close() error { return s.rows.Close() }

func (s *sqlRowsDeviceSource) Scan() (deviceCSVRow, error) {
	var row deviceCSVRow
	var createdRaw any
	if err := s.rows.Scan(
		&row.DeviceName, &row.GAID, &row.ADID, &row.OAID,
		&row.AppName, &row.PalCode, &row.ApplicationID, &row.BrandCode,
		&createdRaw,
	); err != nil {
		return row, fmt.Errorf("扫描设备导出行失败: %w", err)
	}
	t, err := scanAnyTime(createdRaw)
	if err != nil {
		return row, fmt.Errorf("解析 created_at 失败: %w", err)
	}
	row.CreatedAt = t
	return row, nil
}

// scanAnyTime 兼容不同 SQL 驱动对 DATETIME 列的原生 Scan 类型（mysql 视 parseTime 而定返回
// time.Time/[]byte，sqlite 驱动多返回字符串），统一解析为 time.Time。
func scanAnyTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case []byte:
		return parseTimeFlexible(string(t))
	case string:
		return parseTimeFlexible(t)
	case nil:
		return time.Time{}, nil
	default:
		return time.Time{}, fmt.Errorf("无法识别的 created_at 类型 %T", v)
	}
}

// parseTimeFlexible 依次尝试常见时间格式（含带/不带小数秒、RFC3339）。
func parseTimeFlexible(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999Z07:00",
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// sliceDeviceSource 包装一批已加载进内存的 model.ChannelDevice（DevicesByIDs 结果）。
type sliceDeviceSource struct {
	list []model.ChannelDevice
	idx  int
}

func (s *sliceDeviceSource) Next() bool {
	if s.idx >= len(s.list) {
		return false
	}
	s.idx++
	return true
}

func (s *sliceDeviceSource) Scan() (deviceCSVRow, error) {
	d := s.list[s.idx-1]
	return deviceCSVRow{
		DeviceName: d.DeviceName, GAID: d.GAID, ADID: d.ADID, OAID: d.OAID,
		AppName: d.AppName, PalCode: d.PalCode, ApplicationID: d.ApplicationID, BrandCode: d.BrandCode,
		CreatedAt: d.CreatedAt,
	}, nil
}

func (s *sliceDeviceSource) Err() error   { return nil }
func (s *sliceDeviceSource) Close() error { return nil }

// writeDeviceCSVRow 写一行；gaid 非空时附加 gaid_sha256（hex(sha256(gaid))，gaid 已小写）。
func writeDeviceCSVRow(w *csv.Writer, row deviceCSVRow) error {
	gaidSha256 := ""
	if row.GAID != "" {
		sum := sha256.Sum256([]byte(row.GAID))
		gaidSha256 = hex.EncodeToString(sum[:])
	}
	return w.Write([]string{
		row.DeviceName, row.GAID, gaidSha256, row.ADID, row.OAID,
		row.AppName, row.PalCode, row.ApplicationID, row.BrandCode,
		row.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// streamDeviceCSV 是两个导出端点共用的 CSV 生成内核：写 BOM + 表头，逐行从 src 拉取写入，
// 每 500 行 flush 一次底层 csv.Writer 并调用 flush 回调（handler 传入的 flush 负责把已写内容
// 真正推给 HTTP 客户端；service 层不感知 HTTP，测试等场景可传 nil 跳过）。
func streamDeviceCSV(w io.Writer, src deviceCSVSource, flush func()) error {
	defer src.Close()

	if _, err := w.Write(csvBOM); err != nil {
		return fmt.Errorf("写入 CSV BOM 失败: %w", err)
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(deviceCSVHeader); err != nil {
		return fmt.Errorf("写入 CSV 表头失败: %w", err)
	}

	n := 0
	for src.Next() {
		row, err := src.Scan()
		if err != nil {
			return err
		}
		if err := writeDeviceCSVRow(cw, row); err != nil {
			return fmt.Errorf("写入 CSV 行失败: %w", err)
		}
		n++
		if n%500 == 0 {
			cw.Flush()
			if err := cw.Error(); err != nil {
				return fmt.Errorf("flush CSV 失败: %w", err)
			}
			if flush != nil {
				flush()
			}
		}
	}
	if err := src.Err(); err != nil {
		return fmt.Errorf("读取设备导出游标失败: %w", err)
	}
	cw.Flush()
	return cw.Error()
}

// ExportDevicesCSV 按筛选条件流式导出 CSV（GET /api/devices/export.csv）。
// w 通常是 HTTP 响应体，flush 由调用方注入用于分批推送（可为 nil）。in.Scope* 见 ListDevicesInput
// ——导出通道与列表同一套数据权限过滤，不能「列表被过滤、导出却漏了」（docs/admin/10-rbac.md）。
func (s *Service) ExportDevicesCSV(ctx context.Context, w io.Writer, in ListDevicesInput, flush func()) error {
	f, err := buildDeviceFilter(in.ApplicationID, in.From, in.To)
	if err != nil {
		return err
	}
	if err := s.applyListDevicesScope(ctx, &f, in); err != nil {
		return err
	}
	rows, err := s.repo.ExportDeviceRows(ctx, f)
	if err != nil {
		return err
	}
	return streamDeviceCSV(w, &sqlRowsDeviceSource{rows: rows}, flush)
}

// ExportDevicesCSVByIDs 按勾选 id 列表流式导出 CSV（POST /api/devices/export）。
// 勾选导出绕不开数据权限：按 id 查出的记录仍要过 scope 关，越界的 id 静默丢弃
// （不报错，视为「这些 id 里你能导出的部分」，与批量操作的一般处理方式一致）。
func (s *Service) ExportDevicesCSVByIDs(ctx context.Context, w io.Writer, scope auth.Scope, ids []uint64, flush func()) error {
	if len(ids) == 0 {
		return errBadRequest("ids 不得为空")
	}
	if len(ids) > DeviceExportMaxIDs {
		return errBadRequest(fmt.Sprintf("ids 数量超出上限（最多 %d 个）", DeviceExportMaxIDs))
	}
	list, err := s.repo.DevicesByIDs(ctx, ids)
	if err != nil {
		return err
	}
	list = s.filterDevicesByScope(ctx, scope, list)
	return streamDeviceCSV(w, &sliceDeviceSource{list: list}, flush)
}

// filterDevicesByScope 按数据权限过滤一批已加载进内存的设备记录（用于「勾选 id 导出」这类
// 先按 id 查出再需要二次过滤的场景）。channel_device 表没有 channel_id 列，渠道维度的判定要
// 先按 applicationId 批量解出 (channelID, brandCode) 再比对。scope 全量或列表为空时原样返回。
func (s *Service) filterDevicesByScope(ctx context.Context, scope auth.Scope, list []model.ChannelDevice) []model.ChannelDevice {
	if (scope.AllBrands && scope.AllChannels) || len(list) == 0 {
		return list
	}
	seen := map[string]bool{}
	appIDs := make([]string, 0, len(list))
	for _, d := range list {
		if !seen[d.ApplicationID] {
			seen[d.ApplicationID] = true
			appIDs = append(appIDs, d.ApplicationID)
		}
	}
	info, err := s.repo.ChannelBrandsByApplicationIDs(ctx, appIDs)
	if err != nil {
		return nil // fail-closed：查询出错时不返回未经过滤的数据
	}
	out := make([]model.ChannelDevice, 0, len(list))
	for _, d := range list {
		if row, ok := info[d.ApplicationID]; ok && scope.ChannelAllowed(row.BrandCode, row.ChannelID) {
			out = append(out, d)
		}
	}
	return out
}
