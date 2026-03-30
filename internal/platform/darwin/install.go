package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/boyzcl/codex-proxy-fix/internal/detect"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/common"
	"github.com/boyzcl/codex-proxy-fix/internal/platform/types"
	"github.com/boyzcl/codex-proxy-fix/internal/state"
)

const label = "com.codexproxyfix.env"

func Install(proxyEnv common.ProxyEnv, codex detect.CodexInstall, dryRun bool) (types.InstallResult, error) {
	configDir, err := state.EnsureConfigDir()
	if err != nil {
		return types.InstallResult{}, err
	}
	launchAgentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0o755); err != nil {
		return types.InstallResult{}, err
	}

	scriptPath := filepath.Join(configDir, "setenv.sh")
	plistPath := filepath.Join(launchAgentsDir, label+".plist")
	launcherPath := filepath.Join(configDir, "launch-codex.sh")
	originalEnv := currentEnvSnapshot()

	script := buildSetEnvScript(proxyEnv)
	plist := buildLaunchAgentPlist(scriptPath)
	launcher := buildLauncherScript(proxyEnv, codex)
	result := types.InstallResult{
		ManagedPaths: []string{scriptPath, plistPath, launcherPath},
		OriginalEnv:  originalEnv,
	}

	if dryRun {
		result.PersistentInstalled = true
		result.FallbackInstalled = true
		result.Notes = []string{"dry-run: no files were written"}
		return result, nil
	}

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return types.InstallResult{}, err
	}
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return types.InstallResult{}, err
	}
	if err := os.WriteFile(launcherPath, []byte(launcher), 0o755); err != nil {
		return types.InstallResult{}, err
	}
	result.FallbackInstalled = true

	_ = exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid()), plistPath).Run()
	if err := exec.Command("launchctl", "bootstrap", "gui/"+strconv.Itoa(os.Getuid()), plistPath).Run(); err != nil {
		result.Notes = append(result.Notes, "LaunchAgent bootstrap failed: "+err.Error())
	} else {
		result.PersistentInstalled = true
	}
	if result.PersistentInstalled {
		if err := exec.Command("launchctl", "kickstart", "-k", "gui/"+strconv.Itoa(os.Getuid())+"/"+label).Run(); err != nil {
			result.Notes = append(result.Notes, "LaunchAgent kickstart failed: "+err.Error())
		}
	}
	if err := exec.Command(scriptPath).Run(); err != nil {
		result.Notes = append(result.Notes, "Session proxy env apply failed: "+err.Error())
	}

	verified := false
	if value, err := currentLaunchCtlValue("HTTP_PROXY"); err == nil && value == proxyEnv.HTTPProxy {
		verified = true
	}
	if verified && !result.PersistentInstalled {
		result.Notes = append(result.Notes, "Session env was applied but persistent LaunchAgent registration did not complete")
	}
	if result.PersistentInstalled {
		result.Notes = append(result.Notes, "macOS user-session proxy injection installed")
	}
	if result.FallbackInstalled {
		result.Notes = append(result.Notes, "Fallback launcher written to "+launcherPath)
	}

	result.PersistentVerified = verified
	return result, nil
}

func Uninstall(s *state.State, dryRun bool) ([]string, error) {
	var removed []string
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", label+".plist")
	if !dryRun {
		_ = exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid()), plistPath).Run()
		if err := restoreEnvSnapshot(s.OriginalPersistentEnv); err != nil {
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

func buildSetEnvScript(proxyEnv common.ProxyEnv) string {
	return fmt.Sprintf(`#!/bin/zsh
launchctl setenv HTTP_PROXY %q
launchctl setenv HTTPS_PROXY %q
launchctl setenv ALL_PROXY %q
launchctl setenv NO_PROXY %q
`, proxyEnv.HTTPProxy, proxyEnv.HTTPSProxy, proxyEnv.ALLProxy, proxyEnv.NOProxy)
}

func buildLaunchAgentPlist(scriptPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
`, label, scriptPath)
}

func buildLauncherScript(proxyEnv common.ProxyEnv, codex detect.CodexInstall) string {
	target := codex.GUIPath
	if target == "" {
		target = codex.CLIPath
	}
	return fmt.Sprintf(`#!/bin/zsh
export HTTP_PROXY=%q
export HTTPS_PROXY=%q
export ALL_PROXY=%q
export NO_PROXY=%q
exec %q "$@"
`, proxyEnv.HTTPProxy, proxyEnv.HTTPSProxy, proxyEnv.ALLProxy, proxyEnv.NOProxy, target)
}

func currentLaunchCtlValue(key string) (string, error) {
	out, err := exec.Command("launchctl", "getenv", key).Output()
	if err != nil {
		return "", err
	}
	return string(bytesTrimSpace(out)), nil
}

func currentEnvSnapshot() *state.EnvSnapshot {
	return &state.EnvSnapshot{
		HTTPProxy:  snapshotKey("HTTP_PROXY"),
		HTTPSProxy: snapshotKey("HTTPS_PROXY"),
		ALLProxy:   snapshotKey("ALL_PROXY"),
		NOProxy:    snapshotKey("NO_PROXY"),
	}
}

func snapshotKey(key string) state.EnvVarSnapshot {
	value, err := currentLaunchCtlValue(key)
	if err != nil {
		return state.EnvVarSnapshot{}
	}
	return state.EnvVarSnapshot{
		Present: true,
		Value:   value,
	}
}

func restoreEnvSnapshot(snapshot *state.EnvSnapshot) error {
	if snapshot == nil {
		return unsetKeys("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY")
	}
	if err := restoreKey("HTTP_PROXY", snapshot.HTTPProxy); err != nil {
		return err
	}
	if err := restoreKey("HTTPS_PROXY", snapshot.HTTPSProxy); err != nil {
		return err
	}
	if err := restoreKey("ALL_PROXY", snapshot.ALLProxy); err != nil {
		return err
	}
	if err := restoreKey("NO_PROXY", snapshot.NOProxy); err != nil {
		return err
	}
	return nil
}

func restoreKey(key string, snap state.EnvVarSnapshot) error {
	if !snap.Present {
		return exec.Command("launchctl", "unsetenv", key).Run()
	}
	return exec.Command("launchctl", "setenv", key, snap.Value).Run()
}

func unsetKeys(keys ...string) error {
	for _, key := range keys {
		if err := exec.Command("launchctl", "unsetenv", key).Run(); err != nil {
			return err
		}
	}
	return nil
}

func bytesTrimSpace(in []byte) []byte {
	start, end := 0, len(in)
	for start < end && (in[start] == '\n' || in[start] == '\r' || in[start] == '\t' || in[start] == ' ') {
		start++
	}
	for end > start && (in[end-1] == '\n' || in[end-1] == '\r' || in[end-1] == '\t' || in[end-1] == ' ') {
		end--
	}
	return in[start:end]
}
