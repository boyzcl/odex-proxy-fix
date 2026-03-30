package detect

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type CodexInstall struct {
	GUIPath string `json:"gui_path,omitempty"`
	CLIPath string `json:"cli_path,omitempty"`
}

func FindCodex(explicit string) CodexInstall {
	if explicit != "" {
		return fromExplicit(explicit)
	}

	var install CodexInstall
	switch runtime.GOOS {
	case "darwin":
		candidates := []string{
			"/Applications/Codex.app",
			"/Applications/Codex.app/Contents/MacOS/Codex",
			"/Applications/Codex.app/Contents/Resources/codex",
		}
		for _, c := range candidates {
			addCodexPath(&install, c)
		}
	case "windows":
		if exe, err := exec.LookPath("codex.exe"); err == nil {
			install.CLIPath = exe
		}
	case "linux":
		if exe, err := exec.LookPath("codex"); err == nil {
			install.CLIPath = exe
		}
	}
	if exe, err := exec.LookPath("codex"); err == nil && install.CLIPath == "" {
		install.CLIPath = exe
	}
	return install
}

func (c CodexInstall) AnyFound() bool {
	return c.GUIPath != "" || c.CLIPath != ""
}

func funcDefaultCodexCommand(ci CodexInstall) string {
	if ci.GUIPath != "" {
		return ci.GUIPath
	}
	return ci.CLIPath
}

func fromExplicit(path string) CodexInstall {
	var install CodexInstall
	addCodexPath(&install, path)
	return install
}

func addCodexPath(install *CodexInstall, path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.IsDir() && filepath.Ext(path) == ".app" {
		gui := filepath.Join(path, "Contents", "MacOS", "Codex")
		cli := filepath.Join(path, "Contents", "Resources", "codex")
		if _, err := os.Stat(gui); err == nil {
			install.GUIPath = gui
		}
		if _, err := os.Stat(cli); err == nil {
			install.CLIPath = cli
		}
		return
	}
	base := filepath.Base(path)
	switch base {
	case "Codex", "Codex.exe":
		install.GUIPath = path
	case "codex", "codex.exe":
		install.CLIPath = path
	}
}
