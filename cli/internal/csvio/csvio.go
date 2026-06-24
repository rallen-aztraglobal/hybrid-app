// Package csvio 负责 channels/<brand>.csv 的读取与「字节级兼容」渲染。
//
// 现有 app/build.gradle 的 loadChannels 用 '|' 分隔、跳过以 # 开头与空行、要求 >=4 列。
// CLI 是这些文件的渲染器（ADR-0004：后台是上游，Gradle 一行不动），因此输出格式必须
// 与现状完全一致：
//   - 两行注释头（# 开头）原样保留；
//   - 数据行 `flavor|applicationId|palCode|appName`，单管道分隔、无多余空格；
//   - LF 行尾、文件以单个换行结尾（与现有 channels/ap.csv 一致）。
package csvio

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hybrid-app/cli/internal/manifest"
)

// 默认注释头：当目标 CSV 不存在（首次 pull）时使用，与现有三份 CSV 完全一致。
var defaultHeader = []string{
	"# 渠道数据表  字段: flavorName|applicationId|palCode|appName",
	"# 新增小渠道 = 在此加一行 + 在 app/src/<flavorName>/ 放 icon 与 splash 资源",
}

// Row 是一条小渠道记录（CSV 的一行）。
type Row struct {
	Flavor        string
	ApplicationId string
	PalCode       string
	AppName       string
}

// File 表示一份解析后的渠道 CSV：保留原始注释头，便于回写时不丢失。
type File struct {
	Header []string // 文件顶部的注释行（含 '#' 前缀，已去除行尾换行）
	Rows   []Row
}

// Parse 解析 CSV 字节内容。规则与 app/build.gradle 的 loadChannels 对齐：
// trim 后空行与 '#' 开头视为注释/空白；数据行按 '|' 切分需 >=4 段，逐段 trim。
//
// 注释头的判定：在「首个数据行」之前出现的所有 '#' 开头行计入 Header；
// 数据行之间或之后的注释行不会丢，但会被规整到 Header 之后输出（现有文件无此情况）。
func Parse(data []byte) *File {
	f := &File{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	seenRow := false
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if !seenRow {
				f.Header = append(f.Header, line)
			}
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			// 与 Gradle 一致：不足 4 列直接忽略。
			continue
		}
		seenRow = true
		f.Rows = append(f.Rows, Row{
			Flavor:        strings.TrimSpace(parts[0]),
			ApplicationId: strings.TrimSpace(parts[1]),
			PalCode:       strings.TrimSpace(parts[2]),
			AppName:       strings.TrimSpace(parts[3]),
		})
	}
	return f
}

// ReadFile 从磁盘读取并解析 CSV；文件不存在返回空 File（非错误，供首次 pull 用）。
func ReadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Header: append([]string(nil), defaultHeader...)}, nil
		}
		return nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	return Parse(data), nil
}

// Render 把渠道行渲染成与现状字节级兼容的 CSV 文本。
//
// header 为 nil 时使用默认注释头。每行 `flavor|applicationId|palCode|appName`，
// 末尾保证恰好一个 '\n'（与现有文件一致；Gradle 的 eachLine 对结尾换行不敏感，
// 但我们追求与人工维护结果一致，避免 git 噪声）。
func Render(header []string, rows []Row) []byte {
	if len(header) == 0 {
		header = defaultHeader
	}
	var b bytes.Buffer
	for _, h := range header {
		b.WriteString(h)
		b.WriteByte('\n')
	}
	for _, r := range rows {
		b.WriteString(r.Flavor)
		b.WriteByte('|')
		b.WriteString(r.ApplicationId)
		b.WriteByte('|')
		b.WriteString(r.PalCode)
		b.WriteByte('|')
		b.WriteString(r.AppName)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// RowsFromChannels 把 manifest 渠道转为 CSV 行，顺序保持 manifest 给定顺序
// （后台决定顺序；不重排，避免无谓 diff）。直接采用 manifest 给定的 applicationId。
func RowsFromChannels(chs []manifest.Channel) []Row {
	rows := make([]Row, 0, len(chs))
	for _, c := range chs {
		rows = append(rows, Row{
			Flavor:        c.Flavor,
			ApplicationId: c.ApplicationId,
			PalCode:       c.PalCode,
			AppName:       c.AppName,
		})
	}
	return rows
}

// RowsFromChannelsDerived 与 RowsFromChannels 类似，但 applicationId 统一用 ADR-0009
// 的派生值（<品牌包前缀>.<flavor>），从构造上保证「appId 后缀 == flavor」且全局唯一，
// 杜绝历史 appId/flavor 不一致脏数据。品牌未知无法派生时回退到 manifest 给定的 applicationId。
func RowsFromChannelsDerived(brand string, chs []manifest.Channel) []Row {
	rows := make([]Row, 0, len(chs))
	for _, c := range chs {
		appID := manifest.DeriveApplicationID(brand, c.Flavor)
		if appID == "" {
			appID = c.ApplicationId
		}
		rows = append(rows, Row{
			Flavor:        c.Flavor,
			ApplicationId: appID,
			PalCode:       c.PalCode,
			AppName:       c.AppName,
		})
	}
	return rows
}

// WriteFile 原子写出 CSV（先写临时文件再 rename），避免中断导致半截文件。
func WriteFile(path string, header []string, rows []Row) error {
	data := Render(header, rows)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("替换 %s 失败: %w", path, err)
	}
	return nil
}

// Flavors 返回所有 flavor 名（有序，按出现顺序）。
func (f *File) Flavors() []string {
	out := make([]string, 0, len(f.Rows))
	for _, r := range f.Rows {
		out = append(out, r.Flavor)
	}
	return out
}

// FindByFlavor 按 flavor 名查找一行。
func (f *File) FindByFlavor(flavor string) (Row, bool) {
	for _, r := range f.Rows {
		if r.Flavor == flavor {
			return r, true
		}
	}
	return Row{}, false
}

// Conflict 描述一处唯一性冲突。
type Conflict struct {
	Field   string // applicationId / palCode / flavor
	Value   string
	Flavors []string // 命中该值的所有 flavor
}

func (c Conflict) String() string {
	return fmt.Sprintf("%s 重复: %q 被 %s 同时使用", c.Field, c.Value, strings.Join(c.Flavors, ", "))
}

// Validate 校验渠道行的唯一性约束（CLAUDE.md 护栏 5 / ADR-0009）：
// flavor 与 applicationId 必须全局唯一；**palCode 不再参与查重**（ADR-0009：
// PAL_CODE 跨品牌可重复，仅作 URL 参数与编译期烧录，不再是身份/解析键）。
// 现有 CSV 含脏数据（ap01035 与 ap01034 共用 applicationId）——本函数据此拦截。
// 返回的冲突列表为空即通过。
func Validate(rows []Row) []Conflict {
	var conflicts []Conflict
	conflicts = append(conflicts, dupCheck("flavor", rows, func(r Row) string { return r.Flavor })...)
	conflicts = append(conflicts, dupCheck("applicationId", rows, func(r Row) string { return r.ApplicationId })...)
	return conflicts
}

func dupCheck(field string, rows []Row, key func(Row) string) []Conflict {
	seen := map[string][]string{}
	order := []string{}
	for _, r := range rows {
		k := key(r)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; !ok {
			order = append(order, k)
		}
		seen[k] = append(seen[k], r.Flavor)
	}
	var out []Conflict
	for _, k := range order {
		if len(seen[k]) > 1 {
			fl := append([]string(nil), seen[k]...)
			sort.Strings(fl)
			out = append(out, Conflict{Field: field, Value: k, Flavors: fl})
		}
	}
	return out
}
