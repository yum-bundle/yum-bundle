package yum_test

import (
	"os"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/testutil"
)

func TestEnableCOPR_CallsDNF(t *testing.T) {
	m, mock := dnfManager(t)
	if err := m.EnableCOPR("atim/lazygit"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "copr", "enable", "-y", "atim/lazygit")
}

func TestEnableCOPR_Idempotent(t *testing.T) {
	m, mock := dnfManager(t)
	// Pre-create the expected .repo file
	if err := os.MkdirAll(m.ReposDir, 0755); err != nil {
		t.Fatal(err)
	}
	coprPath := m.CoprRepoPathFor("atim/lazygit")
	if err := os.WriteFile(coprPath, []byte("[copr]\n"), 0644); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	if err := m.EnableCOPR("atim/lazygit"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertNotCalled(t, "dnf", "copr", "enable", "-y", "atim/lazygit")
}

func TestEnableCOPR_RejectsInvalidFormat(t *testing.T) {
	m, _ := dnfManager(t)
	if err := m.EnableCOPR("notacopr"); err == nil {
		t.Error("expected error for invalid COPR format")
	}
}

func TestEnableCOPR_RequiresDNF(t *testing.T) {
	mock := testutil.NewMockExecutor()
	m := testManager(t)
	m.Executor = mock
	m.LookPath = func(name string) (string, error) {
		return "", os.ErrNotExist
	}
	if err := m.EnableCOPR("user/project"); err == nil {
		t.Error("expected error when dnf not available")
	}
}

func TestCoprRepoPathFor(t *testing.T) {
	m := testManager(t)
	path := m.CoprRepoPathFor("atim/lazygit")
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if !contains(path, "atim") || !contains(path, "lazygit") {
		t.Errorf("expected path to contain user and project, got %s", path)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
