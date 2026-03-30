package detect

import (
	"runtime"
)

type PlatformInfo struct {
	OS      string `json:"os"`
	Version string `json:"version,omitempty"`
	Arch    string `json:"arch"`
}

func CurrentPlatform() PlatformInfo {
	return PlatformInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}
