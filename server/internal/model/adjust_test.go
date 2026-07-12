package model

import (
	"reflect"
	"testing"
)

// TestAdjustEventsValueEmptyIsNull 验证空/nil map 写入 DB 时落为 NULL（Value 返回 nil），
// 对应「未配置 Adjust 事件」的合法状态（ADR-0013）。
func TestAdjustEventsValueEmptyIsNull(t *testing.T) {
	cases := []AdjustEvents{nil, {}}
	for i, ev := range cases {
		v, err := ev.Value()
		if err != nil {
			t.Fatalf("用例 %d: Value() 报错: %v", i, err)
		}
		if v != nil {
			t.Errorf("用例 %d: 空 map 应序列化为 NULL，实际 %#v", i, v)
		}
	}
}

// TestAdjustEventsValueNonEmpty 验证非空 map 序列化为 JSON 字符串（供 JSON/TEXT 列写入）。
func TestAdjustEventsValueNonEmpty(t *testing.T) {
	ev := AdjustEvents{"Login": "wzb3fb", "Purchase": "gyuu2f"}
	v, err := ev.Value()
	if err != nil {
		t.Fatalf("Value() 报错: %v", err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value() 应返回 string，实际 %T", v)
	}
	// 反过来 Scan 回来验证内容一致（不假定 JSON 序列化的 key 顺序）。
	var back AdjustEvents
	if err := back.Scan(s); err != nil {
		t.Fatalf("Scan() 报错: %v", err)
	}
	if !reflect.DeepEqual(map[string]string(back), map[string]string(ev)) {
		t.Errorf("Scan 回填结果 = %#v，期望 %#v", back, ev)
	}
}

// TestAdjustEventsScanNull 验证 DB NULL 值 Scan 出 nil map（未配置事件）。
func TestAdjustEventsScanNull(t *testing.T) {
	var ev AdjustEvents
	if err := ev.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) 报错: %v", err)
	}
	if ev != nil {
		t.Errorf("Scan(nil) 应得到 nil map，实际 %#v", ev)
	}
}

// TestAdjustEventsScanBytesAndString 验证 Scan 同时兼容 []byte（多数 driver 返回类型）与 string。
func TestAdjustEventsScanBytesAndString(t *testing.T) {
	want := map[string]string{"Login": "wzb3fb"}

	var fromBytes AdjustEvents
	if err := fromBytes.Scan([]byte(`{"Login":"wzb3fb"}`)); err != nil {
		t.Fatalf("Scan([]byte) 报错: %v", err)
	}
	if !reflect.DeepEqual(map[string]string(fromBytes), want) {
		t.Errorf("Scan([]byte) = %#v，期望 %#v", fromBytes, want)
	}

	var fromString AdjustEvents
	if err := fromString.Scan(`{"Login":"wzb3fb"}`); err != nil {
		t.Fatalf("Scan(string) 报错: %v", err)
	}
	if !reflect.DeepEqual(map[string]string(fromString), want) {
		t.Errorf("Scan(string) = %#v，期望 %#v", fromString, want)
	}
}

// TestAdjustEventsScanInvalidJSON 验证非法 JSON 报错，而不是静默丢数据。
func TestAdjustEventsScanInvalidJSON(t *testing.T) {
	var ev AdjustEvents
	if err := ev.Scan("not-json"); err == nil {
		t.Error("非法 JSON 应报错")
	}
}
