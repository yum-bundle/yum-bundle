package yum_test

import (
	"errors"
	"testing"
)

func TestInstallGroup_CallsDNF(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.InstallGroup("Development Tools", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "groupinstall", "-y", "Development Tools")
}

func TestInstallGroup_GroupID(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.InstallGroup("development", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "groupinstall", "-y", "development")
}

func TestInstallGroup_EmptyName(t *testing.T) {
	m, _ := dnfManager(t)
	if err := m.InstallGroup("", nil); err == nil {
		t.Error("expected error for empty group name")
	}
}

func TestInstallGroup_InvalidName(t *testing.T) {
	m, _ := dnfManager(t)
	if err := m.InstallGroup("../etc/passwd", nil); err == nil {
		t.Error("expected error for invalid group name")
	}
}

func TestInstallGroup_WithExcludes(t *testing.T) {
	m, mock := dnfManager(t)
	excludes := []string{"kernel", "kernel-devel"}
	if err := m.InstallGroup("Development Tools", excludes); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "groupinstall", "-y", "--exclude=kernel", "--exclude=kernel-devel", "Development Tools")
}

func TestRemoveGroup_CallsDNF(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.RemoveGroup("Development Tools"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "groupremove", "-y", "Development Tools")
}

func TestIsGroupInstalled_Installed(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetOutput([]byte("Installed Groups:\n   Development Tools\n   Server with GUI\n"),
		"dnf", "grouplist", "installed")
	installed, err := m.IsGroupInstalled("Development Tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected installed=true")
	}
}

func TestIsGroupInstalled_CaseInsensitive(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetOutput([]byte("Installed Groups:\n   Development Tools\n"),
		"dnf", "grouplist", "installed")
	installed, err := m.IsGroupInstalled("development tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected installed=true (case-insensitive match)")
	}
}

func TestIsGroupInstalled_NotInstalled(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetOutput([]byte("Installed Groups:\n   Server with GUI\n"),
		"dnf", "grouplist", "installed")
	installed, err := m.IsGroupInstalled("Development Tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected installed=false")
	}
}

func TestIsGroupInstalled_CommandError(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetError(errors.New("exit status 1"), "dnf", "grouplist", "installed")
	installed, err := m.IsGroupInstalled("Development Tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected installed=false on command error")
	}
}
