package yum_test

import (
	"testing"
)

func TestEnableModule_CallsDNF(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.EnableModule("nodejs:18"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "module", "enable", "-y", "nodejs:18")
}

func TestEnableModule_RejectsInvalidFormat(t *testing.T) {
	m, _ := dnfManager(t)
	if err := m.EnableModule("nodejs"); err == nil {
		t.Error("expected error for missing stream")
	}
}

func TestEnableModule_RequiresDNF(t *testing.T) {
	m := testManager(t)
	m.LookPath = func(name string) (string, error) {
		return "", errNotFound
	}
	if err := m.EnableModule("nodejs:18"); err == nil {
		t.Error("expected error when dnf not available")
	}
}

func TestEnableModule_Idempotent(t *testing.T) {
	m, mock := dnfManager(t)
	// Pre-program dnf module list to return enabled output
	mock.SetOutput(
		[]byte("nodejs   18   [e]  common\n"),
		"dnf", "module", "list", "--enabled", "nodejs",
	)
	if err := m.EnableModule("nodejs:18"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertNotCalled(t, "dnf", "module", "enable", "-y", "nodejs:18")
}
