// Package seed 负责初始化基础数据：三个大渠道（品牌）+ 品牌默认域名 + bootstrap admin，
// 以及把现有 channels/*.csv 一次性导入并清洗脏数据（包名重复 ap01035、gzmarket062）。
package seed

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"gorm.io/gorm"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/perm"
	"github.com/hybrid-app/server/internal/repo"
)

// brandSeed 与 app/build.gradle 的 brandConfig 保持一致（域名为编译期兜底默认，运行期以后台为准）。
// PackagePrefix 是 applicationId 的包前缀（ADR-0009），渠道 appId 据此派生。
type brandSeed struct {
	Code          string
	Name          string
	PackagePrefix string
	Scheme        string
	HMS           bool
	Accent        string
	Sort          int
	Domain        string // position 0 主域名
}

var brands = []brandSeed{
	{Code: "ap", Name: "ArenaPlus", PackagePrefix: "com.arenaplus", Scheme: "gzone", HMS: false, Accent: "#2563eb", Sort: 0, Domain: "https://arenaplus.ph"},
	{Code: "bp", Name: "BingoPlus", PackagePrefix: "com.bingoplus", Scheme: "bingo", HMS: true, Accent: "#dc2626", Sort: 1, Domain: "https://www.bingoplus.com"},
	{Code: "gp", Name: "GameZone", PackagePrefix: "com.gamezone", Scheme: "gzone", HMS: false, Accent: "#16a34a", Sort: 2, Domain: "https://gzone.ph"},
}

// EnsureBrands 幂等地建好三个品牌与各自的主域名；已存在的品牌回填 package_prefix（首次升级到 ADR-0009 时）。
func EnsureBrands(ctx context.Context, db *gorm.DB) error {
	for _, b := range brands {
		var existing model.Brand
		err := db.WithContext(ctx).Where("code = ?", b.Code).First(&existing).Error
		if err == nil {
			// 已存在：若旧库 package_prefix 为空（升级前建的），回填之，保证 appId 派生可用。
			if existing.PackagePrefix == "" {
				if err := db.WithContext(ctx).Model(&existing).
					Update("package_prefix", b.PackagePrefix).Error; err != nil {
					return fmt.Errorf("回填品牌 %s package_prefix 失败: %w", b.Code, err)
				}
				log.Printf("[seed] 已回填品牌 %s package_prefix=%s", b.Code, b.PackagePrefix)
			}
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("查询品牌 %s 失败: %w", b.Code, err)
		}
		brand := model.Brand{
			Code: b.Code, Name: b.Name, PackagePrefix: b.PackagePrefix, Scheme: b.Scheme,
			HMSEnabled: b.HMS, AccentColor: b.Accent, Sort: b.Sort,
		}
		if err := db.WithContext(ctx).Create(&brand).Error; err != nil {
			return fmt.Errorf("创建品牌 %s 失败: %w", b.Code, err)
		}
		bd := model.BrandDomain{BrandID: brand.ID, Position: 0, URL: b.Domain, Enabled: true}
		if err := db.WithContext(ctx).Create(&bd).Error; err != nil {
			return fmt.Errorf("创建品牌 %s 主域名失败: %w", b.Code, err)
		}
		log.Printf("[seed] 已创建品牌 %s (%s)", b.Code, b.Name)
	}
	return nil
}

// storeSeed 默认预置的应用商店清单（幂等：按 code 判重，已存在则跳过）。
type storeSeed struct {
	Code string
	Name string
	Sort int
}

var stores = []storeSeed{
	{Code: "hw", Name: "华为", Sort: 1},
	{Code: "xm", Name: "小米", Sort: 2},
	{Code: "op", Name: "Oppo", Sort: 3},
}

// EnsureStores 幂等地建好默认的几个应用商店（按 code 判重，已存在则跳过）。
func EnsureStores(ctx context.Context, db *gorm.DB) error {
	for _, s := range stores {
		var existing model.Store
		err := db.WithContext(ctx).Where("code = ?", s.Code).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("查询商店 %s 失败: %w", s.Code, err)
		}
		store := model.Store{Code: s.Code, Name: s.Name, Sort: s.Sort, Status: model.StoreEnabled}
		if err := db.WithContext(ctx).Create(&store).Error; err != nil {
			return fmt.Errorf("创建商店 %s 失败: %w", s.Code, err)
		}
		log.Printf("[seed] 已创建应用商店 %s (%s)", s.Code, s.Name)
	}
	return nil
}

// EnsureBootstrapAdmin 在无任何账号时按 "user:password" 建一个账号，挂超级管理员角色（RBAC）。
// 依赖 EnsureRBAC 已先执行（main.go 按此顺序调用）：超级管理员角色须已存在。
func EnsureBootstrapAdmin(ctx context.Context, r *repo.Repo, spec string) error {
	n, err := r.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("BOOTSTRAP_ADMIN 格式应为 user:password")
	}
	hash, err := auth.HashPassword(parts[1])
	if err != nil {
		return err
	}
	superAdmin, err := r.GetRoleByName(ctx, superAdminRoleName)
	if err != nil {
		return fmt.Errorf("bootstrap 账号需要超级管理员角色（请确认 EnsureRBAC 已先执行）: %w", err)
	}
	u := &model.AdminUser{Username: parts[0], PasswordHash: hash, Role: model.RoleAdmin, RoleID: superAdmin.ID}
	if err := r.CreateUser(ctx, u); err != nil {
		return err
	}
	log.Printf("[seed] 已创建初始管理员账号: %s（角色 %s）", parts[0], superAdminRoleName)
	return nil
}

// ---------- RBAC（docs/admin/10-rbac.md）----------

// 三个内置初始角色名。
const (
	superAdminRoleName = "超级管理员"
	operatorRoleName   = "运营"
	viewerRoleName     = "只读"
)

// EnsureRBAC 幂等地初始化 RBAC 基础数据——这是生产唯一会跑的数据迁移路径（SQL migration 只建表加列），
// main.go 无条件调用（不受 DB_AUTOSEED 开关影响，见该处注释 / B3）：
//  1. 建三个初始角色（仅按名称不存在时才创建；已存在则不覆盖其权限，尊重管理员后续的自定义调整——
//     超级管理员是例外：其权限恒由 auth.RBAC 动态解析为 perm.AllCodes()，不依赖 role_permission 落库，
//     因此“不存在才建”对它而言不存在“catalog 演进后权限过期”的问题）；
//  2. 存量账号按旧 role 字符串映射回填 role_id：admin→超级管理员、operator→运营、viewer→只读，
//     仅当 role_id 为 0（未回填过）时才动；
//  3. 兜底：回填后仍 role_id=0 的账号（脏数据——role 字符串不在 admin/operator/viewer 三值内，
//     比如手工改过库）统一挂「只读」，避免永久锁死（M2：以前这类账号会被 Resolve 判定角色缺失，
//     报「账号不存在」这种误导排查的 401，现在直接给个最低权限兜底 + 打日志，运维可再手动升权）；
//  4. 清理 role_permission 中已不在当前 catalog 内的权限点 code（catalog 演进时防悬挂）。
func EnsureRBAC(ctx context.Context, db *gorm.DB) error {
	superID, err := ensureRole(ctx, db, superAdminRoleName, "拥有全部权限，内置角色，不可编辑或删除", true, nil)
	if err != nil {
		return err
	}
	opID, err := ensureRole(ctx, db, operatorRoleName,
		"渠道/域名/打包/推送/上架包的查看与写操作，不含商店管理、角色管理、用户管理", false, operatorDefaultPerms())
	if err != nil {
		return err
	}
	viewID, err := ensureRole(ctx, db, viewerRoleName,
		"仅可查看各页面，无写操作；不含角色管理、用户管理这类敏感管理页面", false, viewerDefaultPerms())
	if err != nil {
		return err
	}

	roleByOldString := map[string]uint64{
		model.RoleAdmin:    superID,
		model.RoleOperator: opID,
		model.RoleViewer:   viewID,
	}
	for oldRole, roleID := range roleByOldString {
		res := db.WithContext(ctx).Model(&model.AdminUser{}).
			Where("role = ? AND role_id = 0", oldRole).Update("role_id", roleID)
		if res.Error != nil {
			return fmt.Errorf("按旧角色 %s 回填账号 role_id 失败: %w", oldRole, res.Error)
		}
		if res.RowsAffected > 0 {
			log.Printf("[seed] 已为 %d 个旧角色=%s 的账号回填 role_id=%d", res.RowsAffected, oldRole, roleID)
		}
	}

	// M2 兜底：role 字符串不在 admin/operator/viewer 三值内的脏数据账号，上面的映射回填不会碰到它们，
	// role_id 会一直卡在 0（角色表里不存在 id=0 的角色）。挂「只读」兜底，不放大权限、只求不锁死。
	fallback := db.WithContext(ctx).Model(&model.AdminUser{}).
		Where("role_id = 0").Update("role_id", viewID)
	if fallback.Error != nil {
		return fmt.Errorf("兜底回填脏数据账号 role_id 失败: %w", fallback.Error)
	}
	if fallback.RowsAffected > 0 {
		log.Printf("[seed] 警告：%d 个账号的旧 role 字符串无法识别，已兜底挂「只读」角色（role_id=%d），"+
			"请核查后按需手动升权", fallback.RowsAffected, viewID)
	}

	if err := repo.New(db).DeleteStaleRolePermissions(ctx, perm.AllCodes()); err != nil {
		return err
	}
	return nil
}

// systemManageSet 把 perm.SystemManageCodes()（商店/角色/用户管理三个敏感管理权限）转成
// O(1) 查找的集合，供 operatorDefaultPerms/viewerDefaultPerms 共用同一份排除名单——
// catalog 演进时只要维护 perm.SystemManageCodes() 一处，不必在这两个函数里各写一份字面量。
func systemManageSet() map[string]bool {
	set := make(map[string]bool, 4)
	for _, c := range perm.SystemManageCodes() {
		set[c] = true
	}
	return set
}

// operatorDefaultPerms 「运营」角色的默认权限：全部 route + 全部 button，减去
// perm.SystemManageCodes()（商店管理、角色管理、用户管理）。
// 注意：角色管理/用户管理现在 kind=route（各自独立菜单的进入权），会被 perm.RouteCodes() 收进去，
// 所以 RouteCodes() 这半也必须过一遍排除名单，不能像以前那样只过滤 ButtonCodes() 那半。
func operatorDefaultPerms() []string {
	excluded := systemManageSet()
	out := make([]string, 0, len(perm.RouteCodes())+len(perm.ButtonCodes()))
	for _, code := range perm.RouteCodes() {
		if !excluded[code] {
			out = append(out, code)
		}
	}
	for _, code := range perm.ButtonCodes() {
		if !excluded[code] {
			out = append(out, code)
		}
	}
	return out
}

// viewerDefaultPerms 「只读」角色的默认权限：全部 route，减去 perm.SystemManageCodes()。
//
// 陷阱（见 perm.SystemManageCodes 注释）：角色管理/用户管理的 kind 从 button 改成 route 后，
// 会被 perm.RouteCodes() 一并收进去——如果只读角色的默认权限直接等于 perm.RouteCodes()，
// 就会自动白捡这两个本该只有超管才能授予的敏感管理页面权限。必须显式减去
// perm.SystemManageCodes() 才能保持「只读=只看业务页面，不含系统级管理页面」的原意。
func viewerDefaultPerms() []string {
	excluded := systemManageSet()
	out := make([]string, 0, len(perm.RouteCodes()))
	for _, code := range perm.RouteCodes() {
		if !excluded[code] {
			out = append(out, code)
		}
	}
	return out
}

// ensureRole 按名称幂等取或建角色；仅在角色不存在时创建并写入 perms（builtin 角色权限恒由
// auth.RBAC 动态取 perm.AllCodes()，不落 role_permission 行，perms 参数对其被忽略）。
//
// 数据范围（ScopeAllBrands/ScopeAllChannels）显式写 true（= 全部）——不依赖 GORM 对
// gorm:"default:true" 零值字段的省略插入行为，避免这个关键路径受 GORM/驱动版本细节影响；
// 三个内置角色默认都是「全部品牌 + 全部渠道」，符合「已有角色默认全部范围」的契约。
func ensureRole(ctx context.Context, db *gorm.DB, name, desc string, builtin bool, perms []string) (uint64, error) {
	var existing model.Role
	err := db.WithContext(ctx).Where("name = ?", name).First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("查询角色 %s 失败: %w", name, err)
	}
	role := model.Role{Name: name, Description: desc, Builtin: builtin, ScopeAllBrands: true, ScopeAllChannels: true}
	if err := db.WithContext(ctx).Create(&role).Error; err != nil {
		return 0, fmt.Errorf("创建角色 %s 失败: %w", name, err)
	}
	if !builtin && len(perms) > 0 {
		seen := map[string]bool{}
		rows := make([]model.RolePermission, 0, len(perms))
		for _, p := range perms {
			if seen[p] {
				continue
			}
			seen[p] = true
			rows = append(rows, model.RolePermission{RoleID: role.ID, PermCode: p})
		}
		if err := db.WithContext(ctx).Create(&rows).Error; err != nil {
			return 0, fmt.Errorf("写入角色 %s 权限失败: %w", name, err)
		}
	}
	log.Printf("[seed] 已创建角色 %s (builtin=%v)", name, builtin)
	return role.ID, nil
}

// ImportReport 汇总一次 CSV 导入的结果。
type ImportReport struct {
	Brand     string
	Inserted  int
	Skipped   []string // 真正被跳过的行（字段不足 / 与已入库的 (brand,flavor) 重复 / 插入失败）
	Corrected []string // CSV applicationId 列与派生值不一致、已按派生值修正导入的行（ADR-0009）
}

// ImportCSV 把某品牌的 channels/<brand>.csv 内容导入库（ADR-0009：appId 按 flavor 派生）。
// 字段：flavorName|applicationId|palCode|appName。# 开头与空行跳过。
//   - applicationId **不信任 CSV 值**：一律按 brand.DeriveApplicationID(flavor) 派生
//     （PackagePrefix + "." + flavor，flavor 中的 "_" 会转成 "."，用于商店后缀分段）。
//     CSV 里的 applicationId 列仅用于「与派生值比对」，不一致则记为 Corrected（如历史脏数据
//     ap01035|com.arenaplus.ap01034 → 派生 com.arenaplus.ap01035，作独立渠道导入，不再跳过）。
//   - pal_code 不再全局唯一（ADR-0009），允许跨品牌重复，不参与查重。
//   - 唯一性只校验 (brand, flavor)（appId 由 flavor 派生，故二者等价）；重复才跳过。
func ImportCSV(ctx context.Context, r *repo.Repo, brandCode string, csv io.Reader) (*ImportReport, error) {
	brand, err := r.GetBrandByCode(ctx, brandCode)
	if err != nil {
		return nil, fmt.Errorf("品牌 %s 不存在（请先 EnsureBrands）: %w", brandCode, err)
	}
	if brand.PackagePrefix == "" {
		return nil, fmt.Errorf("品牌 %s 缺少 package_prefix（无法派生 applicationId）", brandCode)
	}
	rep := &ImportReport{Brand: brandCode}

	sc := bufio.NewScanner(csv)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Split(line, "|")
		if len(p) < 4 {
			rep.Skipped = append(rep.Skipped, fmt.Sprintf("第%d行 字段不足: %q", lineNo, line))
			continue
		}
		flavor := strings.TrimSpace(p[0])
		csvAppID := strings.TrimSpace(p[1])
		pal := strings.TrimSpace(p[2])
		appName := strings.TrimSpace(p[3])

		// 解析键 = 派生 appId（事实来源），CSV 的 applicationId 列仅作比对参考。
		appID := brand.DeriveApplicationID(flavor)
		if csvAppID != "" && csvAppID != appID {
			rep.Corrected = append(rep.Corrected,
				fmt.Sprintf("第%d行 %s: CSV applicationId=%q 与派生值不符，按派生值 %q 导入",
					lineNo, flavor, csvAppID, appID))
		}

		// 唯一性只看 (brand, flavor)（=> appId）。pal_code 不参与查重。
		conflicts, err := r.UniqueConflicts(ctx, repo.UniqueCheck{
			ApplicationID: appID, BrandID: brand.ID, FlavorName: flavor,
		})
		if err != nil {
			return nil, err
		}
		if len(conflicts) > 0 {
			rep.Skipped = append(rep.Skipped,
				fmt.Sprintf("第%d行 %s 冲突字段[%s] 已跳过(重复行): %q",
					lineNo, flavor, strings.Join(conflicts, ","), line))
			continue
		}

		ch := &model.Channel{
			BrandID: brand.ID, FlavorName: flavor, ApplicationID: appID,
			PalCode: pal, AppName: appName, Status: model.ChannelEnabled, UseBrandDomains: true,
		}
		if err := r.CreateChannel(ctx, ch); err != nil {
			// 并发或残留唯一约束兜底：记为跳过而非整体失败。
			rep.Skipped = append(rep.Skipped, fmt.Sprintf("第%d行 %s 插入失败: %v", lineNo, flavor, err))
			continue
		}
		rep.Inserted++
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("读取 CSV 失败: %w", err)
	}
	return rep, nil
}
