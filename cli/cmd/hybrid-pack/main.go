// 命令 hybrid-pack 是渠道中台的跨平台打包 CLI。
//
// 它把后台配置渲染回现有 Gradle 输入（channels/*.csv + res + bootstrap.json），
// 再跨平台调用 ./gradlew 打包。设计与护栏见 docs/admin/03、ADR-0002/0004 与根 CLAUDE.md。
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// cobra 已打印用法类错误；这里只确保退出码非零。
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}
