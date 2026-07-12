// Package model — Adjust 归因集成的自定义列类型（ADR-0013 / docs/admin/08-adjust.md）。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// AdjustEvents 是 channel.adjust_events 列在 Go 侧的类型：{事件 name: token} 的映射。
//
// 跨层契约（server 不解析 CSV，只存储/校验/原样读出）：
//   - Web 前端把运营上传的 Adjust 事件 CSV（token,name,unique）解析成本类型对应的 JSON 对象；
//   - server 只做「空 或 string→string JSON 对象」的形状校验，不理解/不改写具体的 name、token 值；
//   - DB：NULL/空 = 该渠道未配置 Adjust 事件；非空按 JSON 对象存储；
//   - API（渠道 CRUD、CLI 配置下发 /api/build/manifest）：原样以 JSON 对象读出，键名固定 adjustEvents。
//
// 底层就是 map[string]string，encoding/json 对具名 map 类型的默认行为已经是「按 JSON 对象序列化/反序列化」，
// 故无需自定义 MarshalJSON/UnmarshalJSON；这里只需补 database/sql 的 Scanner/Valuer 用于 DB 读写。
type AdjustEvents map[string]string

// Value 实现 driver.Valuer：空 map 存 NULL，非空序列化为 JSON 字符串（供 gorm 写入 JSON/TEXT 列）。
func (a AdjustEvents) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(map[string]string(a))
	if err != nil {
		return nil, fmt.Errorf("序列化 adjust_events 失败: %w", err)
	}
	return string(b), nil
}

// Scan 实现 sql.Scanner：NULL/空 → nil map；否则按 JSON 对象解析。
func (a *AdjustEvents) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("adjust_events 列不支持的底层类型: %T", value)
	}
	if len(raw) == 0 {
		*a = nil
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("解析 adjust_events 失败: %w", err)
	}
	*a = AdjustEvents(m)
	return nil
}
