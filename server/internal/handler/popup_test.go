package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hybrid-app/server/internal/seed"
)

// popupBody 一份合法的 P1 弹窗请求体。
func popupBody(name string, brands []string) map[string]any {
	return map[string]any{
		"name": name, "position": "P1", "enabled": true, "brandCodes": brands,
		"cards": []map[string]any{{"imageUrl": "https://cdn.example.com/a.png"}},
	}
}

// TestPopupEndpointPermsAndScope 验证弹窗端点的权限点映射与数据权限：
// 只读仅 page:popups；运营含 popup:edit；受限账号建弹窗越界 403、位置开关 403；公开端点无需鉴权。
func TestPopupEndpointPermsAndScope(t *testing.T) {
	f := newScopeFixture(t)
	app := f.app
	if err := seed.EnsurePopupPositions(t.Context(), app.repo.DB()); err != nil {
		t.Fatal(err)
	}
	_, viewerTok := app.createUser(t, "pv", "只读")
	_, opTok := app.createUser(t, "po", "运营")

	// 只读：能看，不能写。
	if rec := app.do(http.MethodGet, "/api/popups", viewerTok, nil); rec.Code != http.StatusOK {
		t.Errorf("只读 GET /popups 应放行: %d %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodGet, "/api/popups/positions", viewerTok, nil); rec.Code != http.StatusOK {
		t.Errorf("只读 GET /popups/positions 应放行: %d", rec.Code)
	}
	if rec := app.do(http.MethodGet, "/api/popups/stats", viewerTok, nil); rec.Code != http.StatusOK {
		t.Errorf("只读 GET /popups/stats 应放行: %d %s", rec.Code, rec.Body.String())
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/popups"},
		{http.MethodPut, "/api/popups/1"},
		{http.MethodPut, "/api/popups/1/enabled"},
		{http.MethodDelete, "/api/popups/1"},
		{http.MethodPost, "/api/popups/upload-image"},
		{http.MethodPut, "/api/popups/positions/P3"},
	} {
		if rec := app.do(tc.method, tc.path, viewerTok, map[string]any{}); rec.Code != http.StatusForbidden {
			t.Errorf("只读 %s %s 应 403: %d", tc.method, tc.path, rec.Code)
		}
	}

	// 运营：可建、可改位置开关（全量范围）。
	rec := app.do(http.MethodPost, "/api/popups", opTok, popupBody("运营弹窗", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("运营建弹窗应 201: %d %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodPut, "/api/popups/positions/P3", opTok, map[string]any{"enabled": true}); rec.Code != http.StatusOK {
		t.Errorf("运营（全量范围）改位置开关应 200: %d %s", rec.Code, rec.Body.String())
	}

	// ap 品牌受限账号：越界 / 空品牌新建 403；范围内 201；位置开关 403；列表看不到全品牌弹窗。
	if rec := app.do(http.MethodPost, "/api/popups", f.apAllTok, popupBody("空品牌", nil)); rec.Code != http.StatusForbidden {
		t.Errorf("受限账号空品牌新建应 403: %d %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodPost, "/api/popups", f.apAllTok, popupBody("越界", []string{"bp"})); rec.Code != http.StatusForbidden {
		t.Errorf("受限账号越界品牌新建应 403: %d", rec.Code)
	}
	if rec := app.do(http.MethodPost, "/api/popups", f.apAllTok, popupBody("范围内", []string{"ap"})); rec.Code != http.StatusCreated {
		t.Errorf("受限账号范围内新建应 201: %d %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodPut, "/api/popups/positions/P3", f.apAllTok, map[string]any{"enabled": true}); rec.Code != http.StatusForbidden {
		t.Errorf("受限账号改位置开关应 403: %d", rec.Code)
	}
	rec = app.do(http.MethodGet, "/api/popups", f.apAllTok, nil)
	var env struct {
		Data []struct{ Name string } `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if len(env.Data) != 1 || env.Data[0].Name != "范围内" {
		t.Errorf("受限账号列表只应看到范围内弹窗: %s", rec.Body.String())
	}

	// 公开端点：无需鉴权；App 端配置 no-store 且为裸 JSON；未知 appId 404 / 400。
	appID := f.ap001.ApplicationID
	rec = app.do(http.MethodGet, "/api/app/popups?appId="+appID, "", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET /app/popups 应 200 + no-store: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	var cfg struct {
		AppID         string           `json:"appId"`
		ConfigVersion string           `json:"configVersion"`
		Popups        []map[string]any `json:"popups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil || cfg.AppID != appID || len(cfg.ConfigVersion) != 16 {
		t.Errorf("响应应为裸 JSON: %s", rec.Body.String())
	}
	if rec := app.do(http.MethodGet, "/api/app/popups?appId=com.no.such", "", nil); rec.Code != http.StatusNotFound {
		t.Errorf("未知 appId 应 404: %d", rec.Code)
	}
	if rec := app.do(http.MethodGet, "/api/app/popups", "", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("缺 appId 应 400: %d", rec.Code)
	}
	ev := map[string]any{"batchId": "h-1", "appId": appID, "appVersion": "1.0.0",
		"events": []map[string]any{{"event": "popup_trigger", "popupId": 1}}}
	if rec := app.do(http.MethodPost, "/api/app/popups/events", "", ev); rec.Code != http.StatusOK {
		t.Errorf("上报应 200: %d %s", rec.Code, rec.Body.String())
	}
	if rec := app.do(http.MethodPost, "/api/app/popups/events", "", ev); rec.Code != http.StatusOK {
		t.Errorf("重复批次应 200: %d", rec.Code)
	}
	ev["appId"] = "com.no.such"
	ev["batchId"] = "h-2"
	if rec := app.do(http.MethodPost, "/api/app/popups/events", "", ev); rec.Code != http.StatusBadRequest {
		t.Errorf("未知 appId 上报应 400: %d", rec.Code)
	}
}
