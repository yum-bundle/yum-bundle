package yum_test

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/testutil"
	"github.com/yum-bundle/yum-bundle/internal/yum"
)

// exitError1 returns a real *exec.ExitError with exit code 1,
// suitable for use in mock executor error injection.
func exitError1(t *testing.T) error {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit 1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected sh -c 'exit 1' to fail")
	}
	return err
}

func dnfManager(t *testing.T) (*yum.YumManager, *testutil.MockExecutor) {
	t.Helper()
	mock := testutil.NewMockExecutor()
	m := testManager(t)
	m.Executor = mock
	return m, mock
}

func TestIsPackageInstalled_Installed(t *testing.T) {
	m, mock := dnfManager(t)
	// rpm -q --quiet exits 0 → installed
	_ = mock
	installed, err := m.IsPackageInstalled("vim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Mock returns nil error by default for any command
	if !installed {
		t.Error("expected installed=true")
	}
}

func TestIsPackageInstalled_NotInstalled(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetError(exitError1(t), "rpm", "-q", "--quiet", "nosuchpkg")
	installed, err := m.IsPackageInstalled("nosuchpkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected installed=false")
	}
}

func TestIsPackageInstalled_UnexpectedError(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetError(errors.New("permission denied"), "rpm", "-q", "--quiet", "vim")
	_, err := m.IsPackageInstalled("vim")
	if err == nil {
		t.Fatal("expected error for unexpected failure, got nil")
	}
}

func TestInstallPackage_CallsDNF(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.InstallPackage("vim", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "install", "-y", "vim")
}

func TestInstallPackage_VersionPinned(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.InstallPackage("nodejs = 18.0.0", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "install", "-y", "nodejs = 18.0.0")
}

func TestInstallPackage_EmptyName(t *testing.T) {
	m, _ := dnfManager(t)
	if err := m.InstallPackage("", nil); err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestInstallPackage_InvalidName(t *testing.T) {
	m, _ := dnfManager(t)
	if err := m.InstallPackage("../etc/passwd", nil); err == nil {
		t.Error("expected error for invalid package name")
	}
}

func TestInstallPackage_WithExcludes(t *testing.T) {
	m, mock := dnfManager(t)
	excludes := []string{"kernel", "python2*"}
	if err := m.InstallPackage("vim", excludes); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "install", "-y", "--exclude=kernel", "--exclude=python2*", "vim")
}

func TestRemovePackage(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.RemovePackage("vim"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "remove", "-y", "vim")
}

func TestAutoRemove(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.AutoRemove(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "autoremove", "-y")
}

func TestMakecacheOrUpdate(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.MakecacheOrUpdate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "makecache")
}

func TestGetInstalledVersion(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetOutput([]byte("8.2.2-1.fc39"), "rpm", "-q", "--queryformat", "%{VERSION}-%{RELEASE}", "vim")
	ver, err := m.GetInstalledVersion("vim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "8.2.2-1.fc39" {
		t.Errorf("expected 8.2.2-1.fc39, got %q", ver)
	}
}

func TestGetInstalledVersion_NotInstalled(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetError(exitError1(t), "rpm", "-q", "--queryformat", "%{VERSION}-%{RELEASE}", "nosuchpkg")
	ver, err := m.GetInstalledVersion("nosuchpkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "" {
		t.Errorf("expected empty version, got %q", ver)
	}
}

func TestGetInstalledVersion_UnexpectedError(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetError(errors.New("rpm database locked"), "rpm", "-q", "--queryformat", "%{VERSION}-%{RELEASE}", "vim")
	_, err := m.GetInstalledVersion("vim")
	if err == nil {
		t.Fatal("expected error for unexpected failure, got nil")
	}
}

func TestGetAvailableVersion(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetOutput([]byte(`
Name         : vim
Version      : 9.0.0
Release      : 1.fc39
`), "dnf", "info", "--available", "vim")
	ver, err := m.GetAvailableVersion("vim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "9.0.0-1.fc39" {
		t.Errorf("expected 9.0.0-1.fc39, got %q", ver)
	}
}

func TestInstallPackage_WithProxy(t *testing.T) {
	for _, key := range []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY"} {
		os.Unsetenv(key)
	}
	os.Setenv("HTTPS_PROXY", "http://proxy.example.com:3128")
	defer os.Unsetenv("HTTPS_PROXY")

	m, mock := dnfManager(t)
	if err := m.InstallPackage("vim", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "install", "-y", "--setopt=proxy=http://proxy.example.com:3128", "vim")
}

func TestMakecacheOrUpdate_WithProxy(t *testing.T) {
	for _, key := range []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY"} {
		os.Unsetenv(key)
	}
	os.Setenv("HTTPS_PROXY", "http://proxy.example.com:3128")
	defer os.Unsetenv("HTTPS_PROXY")

	m, mock := dnfManager(t)
	if err := m.MakecacheOrUpdate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "makecache", "--setopt=proxy=http://proxy.example.com:3128")
}

func TestInstallPackage_NoProxy(t *testing.T) {
	for _, key := range []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY"} {
		os.Unsetenv(key)
	}

	m, mock := dnfManager(t)
	if err := m.InstallPackage("vim", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "install", "-y", "vim")
	mock.AssertNotCalled(t, "dnf", "install", "-y", "--setopt=proxy=", "vim")
}

func TestGetAllInstalledPackages_DNF(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetOutput([]byte("vim\ncurl\ngit\n"), "dnf", "history", "userinstalled")
	pkgs, err := m.GetAllInstalledPackages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 3 {
		t.Errorf("expected 3 packages, got %d: %v", len(pkgs), pkgs)
	}
}
