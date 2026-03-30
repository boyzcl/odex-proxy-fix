package platform

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/boyzcl/codex-proxy-fix/internal/detect"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/common"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/darwin"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/linux"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/types"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/windows"
	"github.com/boyzcl/codex-proxy-fix/internal/state"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform implementation")

func Install(proxyEnv common.ProxyEnv, codex detect.CodexInstall, dryRun bool) (types.InstallResult, error) {
	switch runtime.GOOS {
	case "darwin":
		return darwin.Install(proxyEnv, codex, dryRun)
	case "windows":
		return windows.Install(proxyEnv, codex, dryRun)
	case "linux":
		return linux.Install(proxyEnv, codex, dryRun)
	default:
		return types.InstallResult{}, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, runtime.GOOS)
	}
}

func Uninstall(s *state.State, dryRun bool) ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		return darwin.Uninstall(s, dryRun)
	case "windows":
		return windows.Uninstall(s, dryRun)
	case "linux":
		return linux.Uninstall(s, dryRun)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, runtime.GOOS)
	}
}
