package windows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/boyzcl/codex-proxy-fix/internal/detect"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/common"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/types"
	"github.com/boyzcl/codex-proxy-fix/internal/state"
)

func Install(proxyEnv common.ProxyEnv, codex detect.CodexInstall, dryRun bool) (types.InstallResult, error) {
	configDir, err := state.EnsureConfigDir()
	if err != nil {
		return types.InstallResult{}, err
	}
	launcherPath := filepath.Join(configDir, "launch-codex.cmd")
	psLauncherPath := filepath.Join(configDir, "launch-codex.ps1")
	startMenuPath := filepath.Join(configDir, "Codex Proxy Launcher.url")
	originalEnv := currentUserEnvSnapshot()
	result := types.InstallResult{
		ManagedPaths: []string{launcherPath, psLauncherPath, startMenuPath},
		OriginalEnv:  originalEnv,
	}
	if dryRun {
		result.PersistentInstalled = true
		result.FallbackInstalled = true
		result.Notes = []string{"dry-run: no files were written"}
		return result, nil
	}
	if err := os.WriteFile(launcherPath, []byte(buildLauncher(proxyEnv, codex)), 0o644); err != nil {
		return types.InstallResult{}, err
	}
	if err := os.WriteFile(psLauncherPath, []byte(buildPowerShellLauncher(proxyEnv, codex)), 0o644); err != nil {
		return types.InstallResult{}, err
	}
	if err := os.WriteFile(startMenuPath, []byte(buildInternetShortcut(psLauncherPath)), 0o644); err != nil {
		return types.InstallResult{}, err
	}
	result.FallbackInstalled = true

	persistentVerified := false
	if err := setUserProxyEnv(proxyEnv); err == nil {
		result.PersistentInstalled = true
		persistentVerified = verifyUserEnv(proxyEnv)
	} else {
		result.Notes = append(result.Notes, "Windows user-level env update failed: "+err.Error())
	}

	result.PersistentVerified = persistentVerified
	result.Notes = append(result.Notes,
		"Explorer or a new login session may be required before GUI launches inherit the new values",
		"Windows support is still untested and should be treated as in-progress",
	)
	return result, nil
}

func Uninstall(s *state.State, dryRun bool) ([]string, error) {
	var removed []string
	if !dryRun {
		if err := restoreUserEnv(s.OriginalPersistentEnv); err != nil {
			return removed, err
		}
	}
	for _, path := range s.ManagedPaths {
		if path == "" {
			continue
		}
		if dryRun {
			removed = append(removed, path)
			continue
		}
		if err := os.Remove(path); err == nil || os.IsNotExist(err) {
			removed = append(removed, path)
		}
	}
	return removed, nil
}

func buildLauncher(proxyEnv common.ProxyEnv, codex detect.CodexInstall) string {
	target := preferredTarget(codex)
	return fmt.Sprintf("@echo off\r\nset HTTP_PROXY=%s\r\nset HTTPS_PROXY=%s\r\nset ALL_PROXY=%s\r\nset NO_PROXY=%s\r\nstart \"\" \"%s\"\r\n", proxyEnv.HTTPProxy, proxyEnv.HTTPSProxy, proxyEnv.ALLProxy, proxyEnv.NOProxy, target)
}

func buildPowerShellLauncher(proxyEnv common.ProxyEnv, codex detect.CodexInstall) string {
	target := preferredTarget(codex)
	return fmt.Sprintf(`$env:HTTP_PROXY = "%s"
$env:HTTPS_PROXY = "%s"
$env:ALL_PROXY = "%s"
$env:NO_PROXY = "%s"
Start-Process -FilePath "%s"
`, proxyEnv.HTTPProxy, proxyEnv.HTTPSProxy, proxyEnv.ALLProxy, proxyEnv.NOProxy, target)
}

func buildInternetShortcut(psLauncherPath string) string {
	return fmt.Sprintf("[InternetShortcut]\r\nURL=file:///%s\r\n", strings.ReplaceAll(psLauncherPath, `\`, `/`))
}

func preferredTarget(codex detect.CodexInstall) string {
	if codex.GUIPath != "" {
		return codex.GUIPath
	}
	return codex.CLIPath
}

func setUserProxyEnv(proxyEnv common.ProxyEnv) error {
	script := fmt.Sprintf(`[Environment]::SetEnvironmentVariable('HTTP_PROXY','%s','User'); `+
		`[Environment]::SetEnvironmentVariable('HTTPS_PROXY','%s','User'); `+
		`[Environment]::SetEnvironmentVariable('ALL_PROXY','%s','User'); `+
		`[Environment]::SetEnvironmentVariable('NO_PROXY','%s','User')`,
		escapePS(proxyEnv.HTTPProxy), escapePS(proxyEnv.HTTPSProxy), escapePS(proxyEnv.ALLProxy), escapePS(proxyEnv.NOProxy))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Run()
}

func verifyUserEnv(proxyEnv common.ProxyEnv) bool {
	checks := map[string]string{
		"HTTP_PROXY":  proxyEnv.HTTPProxy,
		"HTTPS_PROXY": proxyEnv.HTTPSProxy,
		"ALL_PROXY":   proxyEnv.ALLProxy,
		"NO_PROXY":    proxyEnv.NOProxy,
	}
	for key, want := range checks {
		script := fmt.Sprintf(`[Environment]::GetEnvironmentVariable('%s','User')`, key)
		out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
		if err != nil {
			return false
		}
		got := strings.TrimSpace(string(out))
		if got != want {
			return false
		}
	}
	return true
}

func escapePS(in string) string {
	return strings.ReplaceAll(in, `'`, `''`)
}

func currentUserEnvSnapshot() *state.EnvSnapshot {
	return &state.EnvSnapshot{
		HTTPProxy:  getUserEnv("HTTP_PROXY"),
		HTTPSProxy: getUserEnv("HTTPS_PROXY"),
		ALLProxy:   getUserEnv("ALL_PROXY"),
		NOProxy:    getUserEnv("NO_PROXY"),
	}
}

func getUserEnv(key string) state.EnvVarSnapshot {
	script := fmt.Sprintf(`[Environment]::GetEnvironmentVariable('%s','User')`, key)
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return state.EnvVarSnapshot{}
	}
	got := strings.TrimSpace(string(out))
	if got == "" {
		return state.EnvVarSnapshot{}
	}
	return state.EnvVarSnapshot{
		Present: true,
		Value:   got,
	}
}

func restoreUserEnv(snapshot *state.EnvSnapshot) error {
	if snapshot == nil {
		return unsetKeys("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY")
	}
	if err := restoreUserKey("HTTP_PROXY", snapshot.HTTPProxy); err != nil {
		return err
	}
	if err := restoreUserKey("HTTPS_PROXY", snapshot.HTTPSProxy); err != nil {
		return err
	}
	if err := restoreUserKey("ALL_PROXY", snapshot.ALLProxy); err != nil {
		return err
	}
	if err := restoreUserKey("NO_PROXY", snapshot.NOProxy); err != nil {
		return err
	}
	return nil
}

func restoreUserKey(key string, snap state.EnvVarSnapshot) error {
	if !snap.Present {
		return unsetKeys(key)
	}
	script := fmt.Sprintf(`[Environment]::SetEnvironmentVariable('%s','%s','User')`, key, escapePS(snap.Value))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Run()
}

func unsetKeys(keys ...string) error {
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`[Environment]::SetEnvironmentVariable('%s',$null,'User')`, key))
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", strings.Join(parts, "; "))
	return cmd.Run()
}
