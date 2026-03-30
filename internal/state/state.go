package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const fileName = "state.json"

type State struct {
	Version                   int          `json:"version"`
	Platform                  string       `json:"platform"`
	SelectedProxy             string       `json:"selected_proxy"`
	CodexGUIPath              string       `json:"codex_gui_path,omitempty"`
	CodexCLIPath              string       `json:"codex_cli_path,omitempty"`
	OriginalPersistentEnv     *EnvSnapshot `json:"original_persistent_env,omitempty"`
	PersistentFixInstalled    bool         `json:"persistent_fix_installed"`
	PersistentFixVerified     bool         `json:"persistent_fix_verified"`
	FallbackLauncherInstalled bool         `json:"fallback_launcher_installed"`
	ManagedPaths              []string     `json:"managed_paths,omitempty"`
	LastFixTime               string       `json:"last_fix_time,omitempty"`
}

type EnvVarSnapshot struct {
	Present bool   `json:"present"`
	Value   string `json:"value,omitempty"`
}

type EnvSnapshot struct {
	HTTPProxy  EnvVarSnapshot `json:"http_proxy"`
	HTTPSProxy EnvVarSnapshot `json:"https_proxy"`
	ALLProxy   EnvVarSnapshot `json:"all_proxy"`
	NOProxy    EnvVarSnapshot `json:"no_proxy"`
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "codex-proxy-fix"), nil
	case "windows":
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, "codex-proxy-fix"), nil
		}
		return filepath.Join(home, "AppData", "Roaming", "codex-proxy-fix"), nil
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "codex-proxy-fix"), nil
		}
		return filepath.Join(home, ".config", "codex-proxy-fix"), nil
	}
}

func EnsureConfigDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func Path() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func Load() (*State, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Version == 0 {
		s.Version = 1
	}
	return &s, nil
}

func Save(s *State) error {
	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}
	if s.Version == 0 {
		s.Version = 1
	}
	if s.Platform == "" {
		s.Platform = runtime.GOOS
	}
	s.LastFixTime = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fileName), data, 0o644)
}

func Delete() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
