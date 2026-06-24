package build

import (
	"bufio"
	"os"
	"regexp"

	"github.com/hybrid-app/cli/internal/repo"
)

// versionNameRe 匹配 app/build.gradle 里的 `versionName "1.0.3"`（双引号）。
// 与 build.sh 的 `grep versionName ... | awk -F\" '{print $2}'` 等价，只读不改。
var versionNameRe = regexp.MustCompile(`versionName\s+"([^"]+)"`)

// semverRe 与 app/build.gradle 的版本号校验正则等价（X.Y.Z）。
// build.gradle: if (!(appVersionName ==~ /\d+\.\d+\.\d+/)) throw ...
// CLI 在透传 -PversionName 前先本地校验，给出更友好的报错（而非等 Gradle 抛异常）。
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// ValidVersionName 判断版本号是否符合 build.gradle 要求的 X.Y.Z 格式。
func ValidVersionName(v string) bool {
	return semverRe.MatchString(v)
}

// ReadVersionName 从 app/build.gradle 读取 versionName；读取失败或未命中返回空串。
func ReadVersionName(r *repo.Repo) string {
	f, err := os.Open(r.AppBuildGradle())
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := versionNameRe.FindStringSubmatch(sc.Text()); m != nil {
			return m[1]
		}
	}
	return ""
}
