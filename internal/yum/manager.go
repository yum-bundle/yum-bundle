package yum

import (
	"net/http"
	"os/exec"
	"path/filepath"
)

// YumManager provides dependency-injected access to yum/dnf operations.
// Construct with NewYumManager for production defaults, or create
// directly with custom fields for testing.
type YumManager struct {
	Executor      Executor
	HTTPGet       func(string) (*http.Response, error)
	ReposDir      string
	ReposPrefix   string
	KeyDir        string
	KeyPrefix     string
	LookPath      func(string) (string, error)
	OsReleasePath string
	StatePath     func() string
	// pkgCmd is the resolved package manager binary ("dnf" or "yum").
	// Populated lazily by PkgCmd().
	pkgCmd string
}

// NewYumManager creates a YumManager with production defaults.
func NewYumManager() *YumManager {
	return &YumManager{
		Executor:      &realExecutor{},
		HTTPGet:       keyHTTPClient.Get,
		ReposDir:      ReposDir,
		ReposPrefix:   ReposPrefix,
		KeyDir:        KeyDir,
		KeyPrefix:     KeyPrefix,
		LookPath:      exec.LookPath,
		OsReleasePath: "/etc/os-release",
		StatePath:     func() string { return filepath.Join(StateDir, StateFile) },
	}
}

// PkgCmd returns the package manager binary to use: "dnf" if available, else "yum".
// The result is cached after the first call.
func (m *YumManager) PkgCmd() string {
	if m.pkgCmd != "" {
		return m.pkgCmd
	}
	if _, err := m.LookPath("dnf"); err == nil {
		m.pkgCmd = "dnf"
	} else {
		m.pkgCmd = "yum"
	}
	return m.pkgCmd
}

// IsDNF returns true when the resolved package manager is dnf.
func (m *YumManager) IsDNF() bool {
	return m.PkgCmd() == "dnf"
}
