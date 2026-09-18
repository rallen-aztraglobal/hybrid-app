package render

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hybrid-app/cli/internal/repo"
)

// SyncADIRegistration 让各 flavor 的 assets/adi-registration.properties 与本次任务一致：
//
//   - snippet 非空：写入（内容就是 Google 控制台复制的那段账号标识，原样一行）；
//   - snippet 为空：删除已有文件。构建机工作区跨任务复用，不清理的话上一个任务的注册文件会
//     混进之后所有正式包。
//
// 这是 Android 开发者验证「包名注册」的要求：APK 带上该文件，并用登记过的证书签名，Google 据此
// 确认包名归属。只是往 flavor 的 assets 目录放一个文件，走现有 sourceSet，不动 Gradle（护栏 #1）。
func SyncADIRegistration(r *repo.Repo, brand string, flavors []string, snippet string, opt Options) error {
	snippet = strings.TrimSpace(snippet)
	for _, flavor := range flavors {
		path := r.FlavorADIRegistration(brand, flavor)
		if snippet == "" {
			err := os.Remove(path)
			if err == nil {
				opt.logf("  已清理上次残留的 %s", rel(r, path))
			} else if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("清理 %s 失败: %w", rel(r, path), err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("创建 assets 目录失败: %w", err)
		}
		if err := os.WriteFile(path, []byte(snippet), 0o644); err != nil {
			return fmt.Errorf("写 %s 失败: %w", rel(r, path), err)
		}
		opt.logf("  写入 %s", rel(r, path))
	}
	return nil
}
