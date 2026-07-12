// Package repo 负责定位 hybrid-app 仓库根目录，并提供其下各路径的规范拼接。
//
// CLI 可能在仓库任意子目录被调用（cli/、仓库根、CI 工作区等），故需向上查找根标记。
// 所有路径一律用 filepath 拼接以保证 Windows/macOS/Linux 跨平台（见 cli-go.md）。
package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

// Repo 表示已定位的仓库根。
type Repo struct {
	Root string
}

// rootMarkers 是判定「这是 hybrid-app 仓库根」的标记：
// 必须同时存在 channels/ 目录与 settings.gradle（避免误判任意含 channels 的目录）。
var rootMarkers = []string{"settings.gradle", "channels"}

// Find 从 start 目录向上逐级查找仓库根。start 为空则用当前工作目录。
func Find(start string) (*Repo, error) {
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("获取工作目录失败: %w", err)
		}
		start = wd
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, fmt.Errorf("解析绝对路径失败: %w", err)
	}
	for {
		if isRepoRoot(dir) {
			return &Repo{Root: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("未找到 hybrid-app 仓库根（从 %s 向上未见 settings.gradle + channels/）", start)
		}
		dir = parent
	}
}

func isRepoRoot(dir string) bool {
	for _, m := range rootMarkers {
		if _, err := os.Stat(filepath.Join(dir, m)); err != nil {
			return false
		}
	}
	return true
}

// ChannelsCSV 返回某品牌渠道清单 CSV 的绝对路径：channels/<brand>.csv。
func (r *Repo) ChannelsCSV(brand string) string {
	return filepath.Join(r.Root, "channels", brand+".csv")
}

// ChannelsDir 返回 channels/ 目录。
func (r *Repo) ChannelsDir() string {
	return filepath.Join(r.Root, "channels")
}

// FlavorResDir 返回某 flavor 的 res 目录：app/src/channels/<brand>/<flavor>/res。
// 与 app/build.gradle 的 sourceSets res.srcDirs 完全一致（ADR-0004，不可改其格式）。
func (r *Repo) FlavorResDir(brand, flavor string) string {
	return filepath.Join(r.Root, "app", "src", "channels", brand, flavor, "res")
}

// FlavorAssetsDir 返回某 flavor 的 assets 目录：app/src/channels/<brand>/<flavor>/assets。
// bootstrap.json 写入此处；Gradle 会按 flavor sourceSet 自动并入该 flavor 的 assets。
func (r *Repo) FlavorAssetsDir(brand, flavor string) string {
	return filepath.Join(r.Root, "app", "src", "channels", brand, flavor, "assets")
}

// FlavorBootstrap 返回某 flavor 的 bootstrap.json 路径。
func (r *Repo) FlavorBootstrap(brand, flavor string) string {
	return filepath.Join(r.FlavorAssetsDir(brand, flavor), "bootstrap.json")
}

// LocalProperties 返回 local.properties 路径（keystore 配置所在，绝不上传）。
func (r *Repo) LocalProperties() string {
	return filepath.Join(r.Root, "local.properties")
}

// Gradlew 按操作系统返回 gradle wrapper 的绝对路径。
// Windows -> gradlew.bat；其余 -> gradlew（POSIX 脚本）。
func (r *Repo) Gradlew(windows bool) string {
	if windows {
		return filepath.Join(r.Root, "gradlew.bat")
	}
	return filepath.Join(r.Root, "gradlew")
}

// APKOutputDir 返回某 flavor 的 release APK 输出目录：
// app/build/outputs/apk/<flavor>/release。
func (r *Repo) APKOutputDir(flavor string) string {
	return filepath.Join(r.Root, "app", "build", "outputs", "apk", flavor, "release")
}

// AppBuildGradle 返回 app/build.gradle 路径（仅读取版本号，绝不修改）。
func (r *Repo) AppBuildGradle() string {
	return filepath.Join(r.Root, "app", "build.gradle")
}

// AppGoogleServicesJSON 返回 app/google-services.json 路径。
// 该文件由 CLI pull 按品牌投放，供 google-services Gradle 插件构建期按 applicationId 匹配（ADR-0012）。
// 不进 git（已加入 app/.gitignore）；FCM 未配置时该文件不存在，插件条件式 apply。
func (r *Repo) AppGoogleServicesJSON() string {
	return filepath.Join(r.Root, "app", "google-services.json")
}

// AppAdjustTokensJSON 返回 app/adjust-tokens.json 路径。
// 该文件由 CLI pull/build 渲染（ADR-0013），供 app/build.gradle 的自包含旁路块按
// applicationId 查表注入 ADJUST_APP_TOKEN/ADJUST_EVENT_MAP。App Token 非机密（随 APK
// 分发），但仍作为渲染产物不进 git（已加入根 .gitignore）；无渠道绑定 Adjust 时该文件不存在。
func (r *Repo) AppAdjustTokensJSON() string {
	return filepath.Join(r.Root, "app", "adjust-tokens.json")
}
