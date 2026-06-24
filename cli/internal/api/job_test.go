package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hybrid-app/cli/internal/manifest"
)

// envelopeJSON 包一层后台统一响应 {code,message,data}。
func envelopeJSON(t *testing.T, data any) []byte {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(map[string]any{"code": 0, "message": "ok", "data": json.RawMessage(raw)})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestClaimBuildJob 验证领取任务的请求/响应契约：路径 /api/build/claim、请求体 {runner}、
// 响应是后端 BuildRecord 形状（brandCode + flavors 为 JSON 数组字符串），需正确转换为 BuildJob。
func TestClaimBuildJob(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/build/claim" || r.Method != http.MethodPost {
			t.Errorf("意外请求: %s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		// 后端真实形状：brandCode + flavors 为 JSON 数组「字符串」。
		_, _ = w.Write(envelopeJSON(t, map[string]any{
			"id":          42,
			"brandCode":   "ap",
			"flavors":     `["ap01018","ap01034"]`,
			"testEvents":  true,
			"versionName": "1.0.1",
			"name":        "ap-1.0.1-20260624-0102",
		}))
	}))
	defer srv.Close()

	cl := New(srv.URL, "tok123")
	job, err := cl.ClaimBuildJob(context.Background(), "runner-1")
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != 42 || job.Brand != "ap" || len(job.Flavors) != 2 || !job.TestEvents || job.VersionName != "1.0.1" {
		t.Errorf("领取任务解析不符: %+v", job)
	}
	if job.Flavors[0] != "ap01018" || job.Flavors[1] != "ap01034" {
		t.Errorf("flavors 解析不符: %v", job.Flavors)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("未带 Bearer: %q", gotAuth)
	}
	var sent map[string]string
	_ = json.Unmarshal([]byte(gotBody), &sent)
	if sent["runner"] != "runner-1" {
		t.Errorf("runner 未送达: %v", sent)
	}
}

// TestClaimEmptyQueue 验证空队列（空 data / null）被识别为「无任务」（ID==0）。
func TestClaimEmptyQueue(t *testing.T) {
	for _, data := range []string{`{}`, `null`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"message":"empty","data":` + data + `}`))
		}))
		cl := New(srv.URL, "t")
		job, err := cl.ClaimBuildJob(context.Background(), "r")
		srv.Close()
		if err != nil {
			t.Fatalf("data=%s: %v", data, err)
		}
		if job.ID != 0 {
			t.Errorf("空队列(data=%s)应得 ID=0，实得 %d", data, job.ID)
		}
	}
}

// TestRegisterJobArtifact 验证登记产物：路径 /api/build/records/{id}/artifacts、
// 请求体字段 {flavor,versionName,apkUrl,size}（评审 C3：size 而非 sizeBytes），且不含机密。
func TestRegisterJobArtifact(t *testing.T) {
	var gotPath string
	var gotArt struct {
		Flavor      string `json:"flavor"`
		VersionName string `json:"versionName"`
		ApkURL      string `json:"apkUrl"`
		Size        int64  `json:"size"`
	}
	gotRaw := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotRaw = string(b)
		_ = json.Unmarshal(b, &gotArt)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer srv.Close()

	cl := New(srv.URL, "t")
	art := manifest.BuildArtifact{
		Flavor: "ap01018", VersionName: "1.0.1", FileName: "app-ap01018-release.apk",
		ApkURL: "/apks/ap/ap01018/1.0.1/app-ap01018-release.apk", SizeBytes: 123, SHA256: "abc",
	}
	if err := cl.RegisterJobArtifact(context.Background(), 42, art); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/build/records/42/artifacts" {
		t.Errorf("路径不符: %q", gotPath)
	}
	if gotArt.Flavor != "ap01018" || gotArt.ApkURL == "" || gotArt.Size != 123 {
		t.Errorf("产物体不符: %+v (raw=%s)", gotArt, gotRaw)
	}
	// 安全：请求体绝不含 keystore/口令/sha 等多余机密字段（sha256 不上传）。
	if strings.Contains(gotRaw, "keystore") || strings.Contains(gotRaw, "password") {
		t.Errorf("产物体疑似含机密: %s", gotRaw)
	}
}

// TestUpdateJobStatus 验证状态回传路径与请求体：/api/build/records/{id}/status。
func TestUpdateJobStatus(t *testing.T) {
	var gotPath string
	var gotUpd struct {
		Status     string `json:"status"`
		LogExcerpt string `json:"logExcerpt"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotUpd)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer srv.Close()
	cl := New(srv.URL, "t")
	if err := cl.UpdateJobStatus(context.Background(), 7, "success", "tail log"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/build/records/7/status" {
		t.Errorf("路径不符: %q", gotPath)
	}
	if gotUpd.Status != "success" || gotUpd.LogExcerpt != "tail log" {
		t.Errorf("状态体不符: %+v", gotUpd)
	}
}

// TestAppendJobLog 验证日志以纯文本 body 发到 /api/build/records/{id}/logs。
func TestAppendJobLog(t *testing.T) {
	var gotPath, gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"appended":3}}`))
	}))
	defer srv.Close()
	cl := New(srv.URL, "t")
	if err := cl.AppendJobLog(context.Background(), 9, "→ assembling\n"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/build/records/9/logs" {
		t.Errorf("路径不符: %q", gotPath)
	}
	if !strings.HasPrefix(gotCT, "text/plain") {
		t.Errorf("Content-Type 应为 text/plain，实得 %q", gotCT)
	}
	if gotBody != "→ assembling\n" {
		t.Errorf("日志体不符: %q", gotBody)
	}
}
