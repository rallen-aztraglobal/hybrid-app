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

// EnsureBootstrapAdmin 在无任何账号时按 "user:password" 建一个 admin。
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
	u := &model.AdminUser{Username: parts[0], PasswordHash: hash, Role: model.RoleAdmin}
	if err := r.CreateUser(ctx, u); err != nil {
		return err
	}
	log.Printf("[seed] 已创建初始管理员账号: %s（角色 admin）", parts[0])
	return nil
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
//   - applicationId **不信任 CSV 值**：一律按 brand.PackagePrefix + "." + flavor 派生。
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
