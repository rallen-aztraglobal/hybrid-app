package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"

	"github.com/hybrid-app/server/internal/auth"
)

// ---------- XLSX 导出（美化版，默认导出格式；原始 CSV 通道保留） ----------

// deviceXLSXColumn 描述 XLSX 导出的一列：中文表头 + 列宽（Excel 字符宽度）。
type deviceXLSXColumn struct {
	Title string
	Width float64
}

// deviceXLSXColumns 是 XLSX 导出的固定列。与 CSV 相比去掉了恒空的 oaid 保留列，
// 表头用中文并标注用途（Meta 直接上传原文 / TikTok 用 SHA256）。
var deviceXLSXColumns = []deviceXLSXColumn{
	{Title: "设备名称", Width: 22},
	{Title: "GAID（Meta 直接上传）", Width: 40},
	{Title: "GAID SHA256（TikTok）", Width: 68},
	{Title: "Adjust ADID", Width: 26},
	{Title: "应用名", Width: 30},
	{Title: "PAL_CODE", Width: 14},
	{Title: "包名（applicationId）", Width: 34},
	{Title: "品牌", Width: 8},
	{Title: "注册时间", Width: 20},
	{Title: "最后活跃时间", Width: 20},
}

// xlsxSheetMaxRows 是单 sheet 数据行上限：Excel 硬上限 1,048,576 行，留出表头与余量，
// 超出后自动开新 sheet（设备明细2、设备明细3…），百万级导出不会炸文件。
const xlsxSheetMaxRows = 1_000_000

// deviceXLSXWriter 封装「多 sheet 自动分页」的流式写入：每个 sheet 先写列宽/冻结/表头，
// 数据行写满 xlsxSheetMaxRows 后 Flush 当前 StreamWriter 并开下一个 sheet。
type deviceXLSXWriter struct {
	f           *excelize.File
	sw          *excelize.StreamWriter
	headerStyle int
	cellStyle   int
	sheetIdx    int // 已开的 sheet 序号（1 起）
	rowInSheet  int // 当前 sheet 已写的数据行数
}

func newDeviceXLSXWriter() (*deviceXLSXWriter, error) {
	f := excelize.NewFile()
	// 表头：加粗白字 + 深蓝底 + 居中；数据行：默认样式 + 垂直居中。
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"305496"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("创建表头样式失败: %w", err)
	}
	cellStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("创建单元格样式失败: %w", err)
	}
	w := &deviceXLSXWriter{f: f, headerStyle: headerStyle, cellStyle: cellStyle}
	if err := w.openSheet(); err != nil {
		return nil, err
	}
	return w, nil
}

// sheetName 第 1 个 sheet 叫「设备明细」，之后「设备明细2」「设备明细3」…
func (w *deviceXLSXWriter) sheetName(idx int) string {
	if idx == 1 {
		return "设备明细"
	}
	return fmt.Sprintf("设备明细%d", idx)
}

// openSheet 开一个新 sheet 并写入列宽、冻结首行与表头。
func (w *deviceXLSXWriter) openSheet() error {
	w.sheetIdx++
	name := w.sheetName(w.sheetIdx)
	if w.sheetIdx == 1 {
		// excelize.NewFile 自带默认 sheet "Sheet1"，重命名复用。
		if err := w.f.SetSheetName("Sheet1", name); err != nil {
			return fmt.Errorf("重命名 sheet 失败: %w", err)
		}
	} else if _, err := w.f.NewSheet(name); err != nil {
		return fmt.Errorf("新建 sheet 失败: %w", err)
	}

	sw, err := w.f.NewStreamWriter(name)
	if err != nil {
		return fmt.Errorf("创建流式写入器失败: %w", err)
	}
	w.sw = sw
	w.rowInSheet = 0

	// 列宽与冻结首行都必须在写入任何行之前设置（StreamWriter 的硬性约束）。
	for i, col := range deviceXLSXColumns {
		if err := sw.SetColWidth(i+1, i+1, col.Width); err != nil {
			return fmt.Errorf("设置列宽失败: %w", err)
		}
	}
	if err := sw.SetPanes(&excelize.Panes{
		Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft",
	}); err != nil {
		return fmt.Errorf("冻结表头失败: %w", err)
	}

	header := make([]interface{}, len(deviceXLSXColumns))
	for i, col := range deviceXLSXColumns {
		header[i] = excelize.Cell{Value: col.Title, StyleID: w.headerStyle}
	}
	if err := sw.SetRow("A1", header, excelize.RowOpts{Height: 24}); err != nil {
		return fmt.Errorf("写入表头失败: %w", err)
	}
	return nil
}

// writeRow 写一条设备数据；当前 sheet 写满自动切下一个 sheet。
func (w *deviceXLSXWriter) writeRow(row deviceCSVRow) error {
	if w.rowInSheet >= xlsxSheetMaxRows {
		if err := w.sw.Flush(); err != nil {
			return fmt.Errorf("flush sheet 失败: %w", err)
		}
		if err := w.openSheet(); err != nil {
			return err
		}
	}
	gaidSha256 := ""
	if row.GAID != "" {
		sum := sha256.Sum256([]byte(row.GAID))
		gaidSha256 = hex.EncodeToString(sum[:])
	}
	w.rowInSheet++
	cell, err := excelize.CoordinatesToCellName(1, w.rowInSheet+1)
	if err != nil {
		return fmt.Errorf("计算单元格坐标失败: %w", err)
	}
	values := []interface{}{
		excelize.Cell{Value: row.DeviceName, StyleID: w.cellStyle},
		excelize.Cell{Value: row.GAID, StyleID: w.cellStyle},
		excelize.Cell{Value: gaidSha256, StyleID: w.cellStyle},
		excelize.Cell{Value: row.ADID, StyleID: w.cellStyle},
		excelize.Cell{Value: row.AppName, StyleID: w.cellStyle},
		excelize.Cell{Value: row.PalCode, StyleID: w.cellStyle},
		excelize.Cell{Value: row.ApplicationID, StyleID: w.cellStyle},
		excelize.Cell{Value: row.BrandCode, StyleID: w.cellStyle},
		excelize.Cell{Value: row.CreatedAt.Format("2006-01-02 15:04:05"), StyleID: w.cellStyle},
		excelize.Cell{Value: row.UpdatedAt.Format("2006-01-02 15:04:05"), StyleID: w.cellStyle},
	}
	return w.sw.SetRow(cell, values)
}

// finish 收尾：flush 最后一个 sheet 并把整个工作簿写到 w（HTTP 响应体）。
// XLSX 是 zip 容器，无法像 CSV 那样边生成边发——excelize 内部用临时文件缓冲行数据，
// 内存占用可控，组装完成后一次性写出。
func (w *deviceXLSXWriter) finish(out io.Writer) error {
	if err := w.sw.Flush(); err != nil {
		return fmt.Errorf("flush sheet 失败: %w", err)
	}
	if err := w.f.Write(out); err != nil {
		return fmt.Errorf("写出 XLSX 失败: %w", err)
	}
	return w.f.Close()
}

// streamDeviceXLSX 与 streamDeviceCSV 对应的 XLSX 内核，复用同一 deviceCSVSource 游标。
func streamDeviceXLSX(out io.Writer, src deviceCSVSource) error {
	defer src.Close()

	w, err := newDeviceXLSXWriter()
	if err != nil {
		return err
	}
	for src.Next() {
		row, err := src.Scan()
		if err != nil {
			return err
		}
		if err := w.writeRow(row); err != nil {
			return fmt.Errorf("写入 XLSX 行失败: %w", err)
		}
	}
	if err := src.Err(); err != nil {
		return fmt.Errorf("读取设备导出游标失败: %w", err)
	}
	return w.finish(out)
}

// ExportDevicesXLSX 按筛选条件导出美化 XLSX（GET /api/devices/export.xlsx）。
// in.Scope* 见 ListDevicesInput——导出通道与列表同一套数据权限过滤（docs/admin/10-rbac.md）。
func (s *Service) ExportDevicesXLSX(ctx context.Context, w io.Writer, in ListDevicesInput) error {
	f, err := buildDeviceFilter(in)
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
	return streamDeviceXLSX(w, &sqlRowsDeviceSource{rows: rows})
}

// ExportDevicesXLSXByIDs 按勾选 id 列表导出美化 XLSX（POST /api/devices/export 默认格式）。
// scope 过滤见 ExportDevicesCSVByIDs 同款注释。
func (s *Service) ExportDevicesXLSXByIDs(ctx context.Context, w io.Writer, scope auth.Scope, ids []uint64) error {
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
	return streamDeviceXLSX(w, &sliceDeviceSource{list: list})
}
