package service

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/hybrid-app/server/internal/repo"
)

// minimalGoogleServices 是最小合法的 google-services.json（含 project_info + client 数组）。
const minimalGoogleServices = `{
  "project_info": {
    "project_number": "123456789",
    "project_id": "my-test-project",
    "storage_bucket": "my-test-project.appspot.com"
  },
  "client": [
    {
      "client_info": {
        "mobilesdk_app_id": "1:123456789:android:abcdef",
        "android_client_info": {
          "package_name": "com.arenaplus.ap01001"
        }
      }
    }
  ]
}`

// newTestServiceWithRepo 返回 Service 与 Repo（push_test.go 里的 newTestService 也暴露了 r）。
// 此处直接复用 newTestService，只取 svc。
func gs_newSvc(t *testing.T) *Service {
	t.Helper()
	svc, _ := newTestService(t)
	return svc
}

// TestGoogleServices_GetNotFound 验证未上传时 GET 返回 404 错误（ErrNotFound）。
func TestGoogleServices_GetNotFound(t *testing.T) {
	svc := gs_newSvc(t)
	ctx := context.Background()

	_, err := svc.GetGoogleServices(ctx, "ap")
	if err == nil {
		t.Fatal("未上传时应返回错误")
	}
	// 应是 404（errNotFound）。
	svcErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("期望 *service.Error，实际 %T: %v", err, err)
	}
	if svcErr.Code != 404 {
		t.Errorf("期望 HTTP 404，实际 %d", svcErr.Code)
	}
}

// TestGoogleServices_UploadThenGet 验证 POST 上传后 GET 能取回原始 JSON。
func TestGoogleServices_UploadThenGet(t *testing.T) {
	svc := gs_newSvc(t)
	ctx := context.Background()

	// 上传。
	result, err := svc.UploadGoogleServices(ctx, "ap", strings.NewReader(minimalGoogleServices))
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if !result.Stored {
		t.Error("stored 应为 true")
	}
	if result.Brand != "ap" {
		t.Errorf("brand 应为 ap，实际 %q", result.Brand)
	}
	if result.ClientCount != 1 {
		t.Errorf("clientCount 应为 1，实际 %d", result.ClientCount)
	}

	// GET 应取回原始 JSON 内容。
	rc, err := svc.GetGoogleServices(ctx, "ap")
	if err != nil {
		t.Fatalf("上传后 GET 失败: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("读取内容失败: %v", err)
	}
	if string(got) != minimalGoogleServices {
		t.Errorf("取回内容与上传不一致:\n上传: %s\n取回: %s", minimalGoogleServices, string(got))
	}
}

// TestGoogleServices_InvalidBrand 验证非法 brand 返回 400。
func TestGoogleServices_InvalidBrand(t *testing.T) {
	svc := gs_newSvc(t)
	ctx := context.Background()

	// GET 非法 brand。
	_, err := svc.GetGoogleServices(ctx, "xx")
	assertBadRequest(t, err, "非法 brand GET")

	// POST 非法 brand。
	_, err = svc.UploadGoogleServices(ctx, "xx", strings.NewReader(minimalGoogleServices))
	assertBadRequest(t, err, "非法 brand POST")
}

// TestGoogleServices_InvalidJSON 验证上传内容校验：非法 JSON、缺 project_info、client 为空。
func TestGoogleServices_InvalidJSON(t *testing.T) {
	svc := gs_newSvc(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		input string
	}{
		{"非法JSON", `not json at all`},
		{"缺project_info", `{"client":[{"x":1}]}`},
		{"缺client", `{"project_info":{"project_id":"x"}}`},
		{"client为空数组", `{"project_info":{"project_id":"x"},"client":[]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UploadGoogleServices(ctx, "gp", strings.NewReader(tc.input))
			if err == nil {
				t.Fatalf("%s 应被拒绝", tc.name)
			}
			assertBadRequest(t, err, tc.name)
		})
	}
}

// TestGoogleServices_BrandIsolation 验证不同品牌文件互不干扰。
func TestGoogleServices_BrandIsolation(t *testing.T) {
	svc := gs_newSvc(t)
	ctx := context.Background()

	// 上传 ap。
	ap := strings.ReplaceAll(minimalGoogleServices, "my-test-project", "ap-project")
	if _, err := svc.UploadGoogleServices(ctx, "ap", strings.NewReader(ap)); err != nil {
		t.Fatalf("上传 ap 失败: %v", err)
	}

	// bp 未上传 → 应 404。
	_, err := svc.GetGoogleServices(ctx, "bp")
	assertNotFound(t, err, "bp 未上传")

	// gp 未上传 → 应 404。
	_, err = svc.GetGoogleServices(ctx, "gp")
	assertNotFound(t, err, "gp 未上传")

	// ap 应能取到。
	rc, err := svc.GetGoogleServices(ctx, "ap")
	if err != nil {
		t.Fatalf("ap 取回失败: %v", err)
	}
	rc.Close()
}

// ---------- 辅助 ----------

func assertBadRequest(t *testing.T, err error, label string) {
	t.Helper()
	if err == nil {
		t.Errorf("[%s] 期望错误，实际 nil", label)
		return
	}
	svcErr, ok := err.(*Error)
	if !ok {
		t.Errorf("[%s] 期望 *service.Error，实际 %T: %v", label, err, err)
		return
	}
	if svcErr.Code != 400 {
		t.Errorf("[%s] 期望 HTTP 400，实际 %d: %s", label, svcErr.Code, svcErr.Message)
	}
}

func assertNotFound(t *testing.T, err error, label string) {
	t.Helper()
	if err == nil {
		t.Errorf("[%s] 期望 404，实际 nil", label)
		return
	}
	svcErr, ok := err.(*Error)
	if !ok {
		t.Errorf("[%s] 期望 *service.Error，实际 %T: %v", label, err, err)
		return
	}
	if svcErr.Code != 404 {
		t.Errorf("[%s] 期望 HTTP 404，实际 %d: %s", label, svcErr.Code, svcErr.Message)
	}
}

// 防止 repo 包 import 未使用警告（newTestService 返回了 *repo.Repo，gs_newSvc 丢弃它）。
var _ *repo.Repo
