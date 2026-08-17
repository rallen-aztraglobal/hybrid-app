package auth

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// newScopeTestRepo 起一个 sqlite 内存库 + 三个品牌(ap/bp/gp)，供 EffectiveScope/RoleEffectiveScope
// 测试直接摆渠道/角色数据（不经过 service 层，聚焦测 auth 包自己的算法）。
func newScopeTestRepo(t *testing.T) *repo.Repo {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	r := repo.New(db)
	ctx := context.Background()
	for _, b := range []model.Brand{
		{Code: "ap", Name: "ArenaPlus", PackagePrefix: "com.arenaplus", Scheme: "gzone"},
		{Code: "bp", Name: "BingoPlus", PackagePrefix: "com.bingoplus", Scheme: "bingo"},
		{Code: "gp", Name: "GameZone", PackagePrefix: "com.gamezone", Scheme: "gzone"},
	} {
		b := b
		if err := db.WithContext(ctx).Create(&b).Error; err != nil {
			t.Fatalf("建品牌 %s 失败: %v", b.Code, err)
		}
	}
	return r
}

func brandID(t *testing.T, r *repo.Repo, code string) uint64 {
	t.Helper()
	b, err := r.GetBrandByCode(context.Background(), code)
	if err != nil {
		t.Fatalf("取品牌 %s 失败: %v", code, err)
	}
	return b.ID
}

func mustCreateChannel(t *testing.T, r *repo.Repo, brandCode, flavor string) *model.Channel {
	t.Helper()
	ch := &model.Channel{
		BrandID: brandID(t, r, brandCode), FlavorName: flavor,
		ApplicationID: "com.test." + flavor, PalCode: "PAL", AppName: flavor,
		Status: model.ChannelEnabled, UseBrandDomains: true,
	}
	if err := r.CreateChannel(context.Background(), ch); err != nil {
		t.Fatalf("建渠道 %s 失败: %v", flavor, err)
	}
	return ch
}

func mustCreateRole(t *testing.T, r *repo.Repo, role *model.Role) *model.Role {
	t.Helper()
	if err := r.CreateRole(context.Background(), role); err != nil {
		t.Fatalf("建角色 %s 失败: %v", role.Name, err)
	}
	return role
}

// TestRoleEffectiveScopeBuiltinIsFull 验证 builtin 角色恒返回全量范围，忽略其余 scope 字段
// （即便数据库里把 scope_all_brands/scope_all_channels 存成 false，也不影响结果）。
func TestRoleEffectiveScopeBuiltinIsFull(t *testing.T) {
	r := newScopeTestRepo(t)
	rbac := NewRBAC(r)
	role := mustCreateRole(t, r, &model.Role{Name: "超管测试", Builtin: true, ScopeAllBrands: false, ScopeAllChannels: false})

	scope, err := rbac.RoleEffectiveScope(context.Background(), role.ID)
	if err != nil {
		t.Fatalf("计算范围失败: %v", err)
	}
	if !scope.AllBrands || !scope.AllChannels {
		t.Errorf("builtin 角色应恒为全量范围，实际 %+v", scope)
	}
	if !scope.BrandAllowed("ap") || !scope.BrandAllowed("zz-not-exist") {
		t.Error("builtin 全量范围应放行任意品牌 code（包括不存在的）")
	}
}

// TestRoleEffectiveScopeAllFlagsCoverNewData 验证「全部」是标志位而不是快照：
// scope_all_brands/scope_all_channels=true 时，即便 role_brand/role_channel 表里一条记录都没有
// （模拟"从未勾选过，也不需要勾选"的默认全量角色），新建的品牌/渠道也自动在范围内。
func TestRoleEffectiveScopeAllFlagsCoverNewData(t *testing.T) {
	r := newScopeTestRepo(t)
	rbac := NewRBAC(r)
	role := mustCreateRole(t, r, &model.Role{Name: "全量角色", ScopeAllBrands: true, ScopeAllChannels: true})
	ch := mustCreateChannel(t, r, "gp", "gp999") // 角色创建之后才建的渠道/品牌数据

	scope, err := rbac.RoleEffectiveScope(context.Background(), role.ID)
	if err != nil {
		t.Fatalf("计算范围失败: %v", err)
	}
	if !scope.AllBrands || !scope.AllChannels {
		t.Fatalf("应为全量范围，实际 %+v", scope)
	}
	if !scope.BrandAllowed("gp") {
		t.Error("全量品牌范围应覆盖后来新建的品牌")
	}
	if !scope.ChannelAllowed("gp", ch.ID) {
		t.Error("全量渠道范围应覆盖后来新建的渠道")
	}
}

// TestRoleEffectiveScopeBrandNarrowingInvalidatesChannels 验证品牌范围收窄会让原本勾选的渠道
// 自动失效（渠道范围与品牌范围求交），不需要额外清理 role_channel 表。
func TestRoleEffectiveScopeBrandNarrowingInvalidatesChannels(t *testing.T) {
	r := newScopeTestRepo(t)
	rbac := NewRBAC(r)
	ctx := context.Background()

	apCh := mustCreateChannel(t, r, "ap", "ap001")
	bpCh := mustCreateChannel(t, r, "bp", "bp001")

	role := mustCreateRole(t, r, &model.Role{Name: "收窄测试角色", ScopeAllBrands: false, ScopeAllChannels: false})
	// 品牌范围只有 ap，但渠道范围里混了 ap 和 bp 的渠道（模拟"先给了更宽的品牌范围，
	// 后来又把品牌范围收窄"的历史遗留数据）。
	if err := r.ReplaceRoleBrands(ctx, role.ID, []string{"ap"}); err != nil {
		t.Fatalf("写角色品牌范围失败: %v", err)
	}
	if err := r.ReplaceRoleChannels(ctx, role.ID, []uint64{apCh.ID, bpCh.ID}); err != nil {
		t.Fatalf("写角色渠道范围失败: %v", err)
	}

	scope, err := rbac.RoleEffectiveScope(ctx, role.ID)
	if err != nil {
		t.Fatalf("计算范围失败: %v", err)
	}
	if scope.AllBrands || scope.AllChannels {
		t.Fatalf("不应是全量范围，实际 %+v", scope)
	}
	if !scope.ChannelAllowed("ap", apCh.ID) {
		t.Error("ap 渠道应在范围内（品牌 ap 在允许列表里）")
	}
	if scope.ChannelAllowed("bp", bpCh.ID) {
		t.Error("bp 渠道应自动失效——品牌范围只有 ap，即便 role_channel 表里还留着这条勾选")
	}
	if contains(scope.ChannelIDList(), bpCh.ID) {
		t.Error("求交后的 ChannelIDs 集合不应包含品牌已被排除的渠道 id")
	}
	if !contains(scope.ChannelIDList(), apCh.ID) {
		t.Error("求交后的 ChannelIDs 集合应包含 ap 渠道 id")
	}
}

// TestScopeSubsetOf 覆盖 Scope.SubsetOf 的核心语义：全量不是任意非全量的子集；
// 非全量必须逐项检查；全量是全量的子集。
func TestScopeSubsetOf(t *testing.T) {
	full := FullScope()
	limited := Scope{AllBrands: false, Brands: map[string]bool{"ap": true}, AllChannels: true}
	narrower := Scope{AllBrands: false, Brands: map[string]bool{"ap": true}, AllChannels: false, ChannelIDs: map[uint64]bool{1: true}}
	wider := Scope{AllBrands: false, Brands: map[string]bool{"ap": true, "bp": true}, AllChannels: true}

	if !full.SubsetOf(full) {
		t.Error("全量应是全量的子集")
	}
	if full.SubsetOf(limited) {
		t.Error("全量不应是任何非全量范围的子集")
	}
	if !limited.SubsetOf(full) {
		t.Error("任意范围都应是全量的子集")
	}
	if !narrower.SubsetOf(limited) {
		t.Error("narrower（限定 ap 且仅 1 个渠道）应是 limited（限定 ap 全部渠道）的子集")
	}
	if limited.SubsetOf(narrower) {
		t.Error("limited（ap 全部渠道）不应是 narrower（ap 仅 1 个渠道）的子集")
	}
	if !limited.SubsetOf(wider) {
		t.Error("limited（仅 ap）应是 wider（ap+bp）的子集")
	}
	if wider.SubsetOf(limited) {
		t.Error("wider（ap+bp）不应是 limited（仅 ap）的子集")
	}
}

// TestEffectiveScopeCachedAndInvalidated 验证 EffectiveScope 的用户级缓存与 Invalidate 行为
// （与权限集缓存同一套 TTL 节奏）。
func TestEffectiveScopeCachedAndInvalidated(t *testing.T) {
	r := newScopeTestRepo(t)
	rbac := NewRBAC(r)
	ctx := context.Background()

	viewRole := mustCreateRole(t, r, &model.Role{Name: "只读范围测试", ScopeAllBrands: false, ScopeAllChannels: true})
	if err := r.ReplaceRoleBrands(ctx, viewRole.ID, []string{"ap"}); err != nil {
		t.Fatalf("写角色品牌范围失败: %v", err)
	}
	hash, _ := HashPassword("pw123456")
	u := &model.AdminUser{Username: "scope-user", PasswordHash: hash, Role: model.RoleOperator, RoleID: viewRole.ID}
	if err := r.CreateUser(ctx, u); err != nil {
		t.Fatalf("建账号失败: %v", err)
	}

	scope1, err := rbac.EffectiveScope(ctx, u.ID)
	if err != nil {
		t.Fatalf("首次解析失败: %v", err)
	}
	if scope1.AllBrands || !scope1.BrandAllowed("ap") || scope1.BrandAllowed("bp") {
		t.Fatalf("初始范围应为仅 ap，实际 %+v", scope1)
	}

	// 角色品牌范围改成 bp，但不调用 Invalidate：应仍读到缓存的旧值。
	if err := r.ReplaceRoleBrands(ctx, viewRole.ID, []string{"bp"}); err != nil {
		t.Fatalf("改角色品牌范围失败: %v", err)
	}
	scope2, err := rbac.EffectiveScope(ctx, u.ID)
	if err != nil {
		t.Fatalf("二次解析失败: %v", err)
	}
	if !scope2.BrandAllowed("ap") || scope2.BrandAllowed("bp") {
		t.Errorf("未 Invalidate 前应仍读到缓存的旧范围（仅 ap），实际 %+v", scope2)
	}

	rbac.Invalidate()
	scope3, err := rbac.EffectiveScope(ctx, u.ID)
	if err != nil {
		t.Fatalf("三次解析失败: %v", err)
	}
	if scope3.BrandAllowed("ap") || !scope3.BrandAllowed("bp") {
		t.Errorf("Invalidate 后应读到新范围（仅 bp），实际 %+v", scope3)
	}
}

func contains(list []uint64, id uint64) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}
