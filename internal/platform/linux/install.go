package linux

import (
	"fmt"
	"os"
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
	envDir := filepath.Join(os.Getenv("HOME"), ".config", "environment.d")
	envPath := filepath.Join(envDir, "codex-proxy-fix.conf")
	launcherPath := filepath.Join(configDir, "launch-codex.sh")
	desktopDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "applications")
	desktopPath := filepath.Join(desktopDir, "codex-proxy-fix.desktop")
	target := preferredTarget(codex)
	if dryRun {
		return types.InstallResult{
			PersistentInstalled: true,
			FallbackInstalled:   true,
			ManagedPaths:        []string{envPath, launcherPath, desktopPath},
			Notes:               []string{"dry-run: no files were written"},
		}, nil
	}
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		return types.InstallResult{}, err
	}
	if err := os.MkdirAll(desktopDir, 0o755); err != nil {
		return types.InstallResult{}, err
	}
	if err := os.WriteFile(envPath, []byte(buildEnvFile(proxyEnv)), 0o644); err != nil {
		return types.InstallResult{}, err
	}
	if err := os.WriteFile(launcherPath, []byte(buildLauncher(proxyEnv, codex)), 0o755); err != nil {
		return types.InstallResult{}, err
	}
	if target != "" {
		if err := os.WriteFile(desktopPath, []byte(buildDesktopEntry(launcherPath, target)), 0o644); err != nil {
			return types.InstallResult{}, err
		}
	}
	return types.InstallResult{
		PersistentInstalled: true,
		PersistentVerified:  fileExists(envPath),
		FallbackInstalled:   true,
		ManagedPaths:        compactPaths(envPath, launcherPath, desktopPath),
		Notes: []string{
			"linux environment.d support installed; desktop session restart may be required",
			"desktop launcher created when the local desktop applications directory is available",
		},
	}, nil
}

func Uninstall(s *state.State, dryRun bool) ([]string, error) {
	var removed []string
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

func buildEnvFile(proxyEnv common.ProxyEnv) string {
	return fmt.Sprintf("HTTP_PROXY=%q\nHTTPS_PROXY=%q\nALL_PROXY=%q\nNO_PROXY=%q\n", proxyEnv.HTTPProxy, proxyEnv.HTTPSProxy, proxyEnv.ALLProxy, proxyEnv.NOProxy)
}

func buildLauncher(proxyEnv common.ProxyEnv, codex detect.CodexInstall) string {
	target := preferredTarget(codex)
	return fmt.Sprintf(`#!/bin/sh
export HTTP_PROXY=%q
export HTTPS_PROXY=%q
export ALL_PROXY=%q
export NO_PROXY=%q
exec %q "$@"
`, proxyEnv.HTTPProxy, proxyEnv.HTTPSProxy, proxyEnv.ALLProxy, proxyEnv.NOProxy, target)
}

func buildDesktopEntry(launcherPath string, target string) string {
	name := "Codex (Proxy)"
	execPath := launcherPath
	iconLine := ""
	if strings.HasSuffix(strings.ToLower(target), ".appimage") {
		iconLine = fmt.Sprintf("Icon=%s\n", target)
	}
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Terminal=false
Categories=Development;
StartupNotify=true
%s`, name, execPath, iconLine)
}

func preferredTarget(codex detect.CodexInstall) string {
	if codex.GUIPath != "" {
		return codex.GUIPath
	}
	return codex.CLIPath
}

func compactPaths(paths ...string) []string {
	var out []string
	for _, path := range paths {
		if path != "" {
			out = append(out, path)
		}
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
