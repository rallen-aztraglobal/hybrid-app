package build

import (
	"errors"
	"os/exec"
)

// asExit 是 errors.As 针对 *exec.ExitError 的薄封装，便于在 Run 中读取退出码。
func asExit(err error, target **exec.ExitError) bool {
	return errors.As(err, target)
}
