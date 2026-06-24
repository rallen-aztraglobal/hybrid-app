package auth

import (
	"testing"
	"time"

	"github.com/hybrid-app/server/internal/model"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}
	if !CheckPassword(hash, "s3cret-pw") {
		t.Error("正确密码应校验通过")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("错误密码不应通过")
	}
}

func TestIssueAndParse(t *testing.T) {
	m := NewManager("test-secret", "test", time.Hour, 24*time.Hour)
	u := &model.AdminUser{ID: 7, Username: "alice", Role: model.RoleOperator}

	access, refresh, err := m.Issue(u)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	ac, err := m.Parse(access)
	if err != nil {
		t.Fatalf("解析 access 失败: %v", err)
	}
	if ac.UserID != 7 || ac.Username != "alice" || ac.Role != model.RoleOperator {
		t.Errorf("access claims 不符: %+v", ac)
	}
	if ac.Type != TokenAccess {
		t.Errorf("access token type 应为 %q，实际 %q", TokenAccess, ac.Type)
	}

	rc, err := m.Parse(refresh)
	if err != nil {
		t.Fatalf("解析 refresh 失败: %v", err)
	}
	if rc.Type != TokenRefresh {
		t.Errorf("refresh token type 应为 %q，实际 %q", TokenRefresh, rc.Type)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	m1 := NewManager("secret-a", "test", time.Hour, time.Hour)
	m2 := NewManager("secret-b", "test", time.Hour, time.Hour)
	access, _, _ := m1.Issue(&model.AdminUser{ID: 1, Username: "x", Role: model.RoleViewer})
	if _, err := m2.Parse(access); err == nil {
		t.Error("不同 secret 应解析失败")
	}
}

func TestParseRejectsExpired(t *testing.T) {
	m := NewManager("secret", "test", -time.Minute, time.Hour) // 立即过期
	access, _, _ := m.Issue(&model.AdminUser{ID: 1, Username: "x", Role: model.RoleViewer})
	if _, err := m.Parse(access); err == nil {
		t.Error("过期 token 应解析失败")
	}
}

func TestRoleRanking(t *testing.T) {
	if roleRank[model.RoleAdmin] <= roleRank[model.RoleOperator] {
		t.Error("admin 应高于 operator")
	}
	if roleRank[model.RoleOperator] <= roleRank[model.RoleViewer] {
		t.Error("operator 应高于 viewer")
	}
}
