package yum_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/yum"
)

func TestEnableEPEL_InstallsEpelRelease(t *testing.T) {
	m, mock := dnfManager(t)
	// Write a fake os-release for a RHEL-like distro
	writeOsRelease(t, m, `ID="rhel"
NAME="Red Hat Enterprise Linux"
`)
	if err := m.EnableEPEL(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "install", "-y", "epel-release")
}

func TestEnableEPEL_SkipsOnFedora(t *testing.T) {
	m, mock := dnfManager(t)
	writeOsRelease(t, m, `ID="fedora"
NAME="Fedora Linux"
`)
	if err := m.EnableEPEL(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertNotCalled(t, "dnf", "install", "-y", "epel-release")
}

func TestEnableEPEL_Idempotent(t *testing.T) {
	m, mock := dnfManager(t)
	writeOsRelease(t, m, `ID="rocky"`)
	// Pre-create epel.repo to simulate EPEL already enabled
	if err := os.MkdirAll(m.ReposDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.ReposDir, "epel.repo"), []byte("[epel]\n"), 0644); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	if err := m.EnableEPEL(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertNotCalled(t, "dnf", "install", "-y", "epel-release")
}

func writeOsRelease(t *testing.T, m *yum.YumManager, content string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "os-release")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write os-release: %v", err)
	}
	m.OsReleasePath = path
}
