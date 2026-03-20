package yum_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/testutil"
	"github.com/yum-bundle/yum-bundle/internal/yum"
)

func testManager(t *testing.T) *yum.YumManager {
	t.Helper()
	dir := t.TempDir()
	return &yum.YumManager{
		Executor:      testutil.NewMockExecutor(),
		ReposDir:      filepath.Join(dir, "yum.repos.d"),
		ReposPrefix:   "yum-bundle-",
		KeyDir:        filepath.Join(dir, "rpm-gpg"),
		KeyPrefix:     "yum-bundle-",
		OsReleasePath: "/dev/null",
		StatePath:     func() string { return filepath.Join(dir, "state.json") },
		LookPath: func(name string) (string, error) {
			if name == "dnf" {
				return "/usr/bin/dnf", nil
			}
			return "", os.ErrNotExist
		},
	}
}

func TestLoadState_NewWhenMissing(t *testing.T) {
	m := testManager(t)
	state, err := m.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.Packages) != 0 || len(state.Repos) != 0 || len(state.Keys) != 0 {
		t.Error("expected empty state for missing file")
	}
}

func TestSaveAndLoadState(t *testing.T) {
	m := testManager(t)

	state := yum.NewState()
	state.AddPackage("vim")
	state.AddPackage("curl")
	state.AddRepo("/etc/yum.repos.d/yum-bundle-abc.repo")
	state.AddKey("/etc/pki/rpm-gpg/yum-bundle-def.key")

	if err := m.SaveState(state); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := m.LoadState()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if !loaded.HasPackage("vim") || !loaded.HasPackage("curl") {
		t.Error("packages not persisted")
	}
	if !loaded.HasRepo("/etc/yum.repos.d/yum-bundle-abc.repo") {
		t.Error("repo not persisted")
	}
	if !loaded.HasKey("/etc/pki/rpm-gpg/yum-bundle-def.key") {
		t.Error("key not persisted")
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	os.WriteFile(path, []byte("{invalid"), 0600)
	m := &yum.YumManager{
		Executor:  testutil.NewMockExecutor(),
		StatePath: func() string { return path },
	}
	_, err := m.LoadState()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestState_AddRemovePackage(t *testing.T) {
	s := yum.NewState()
	if s.AddPackage("vim") != true {
		t.Error("first add should return true")
	}
	if s.AddPackage("vim") != false {
		t.Error("duplicate add should return false")
	}
	if !s.HasPackage("vim") {
		t.Error("vim should be present")
	}
	if s.RemovePackage("vim") != true {
		t.Error("remove should return true")
	}
	if s.HasPackage("vim") {
		t.Error("vim should be gone")
	}
	if s.RemovePackage("vim") != false {
		t.Error("second remove should return false")
	}
}

func TestState_GetPackagesNotIn(t *testing.T) {
	s := yum.NewState()
	s.AddPackage("vim")
	s.AddPackage("curl")
	s.AddPackage("git")

	notIn := s.GetPackagesNotIn([]string{"vim", "git"})
	if len(notIn) != 1 || notIn[0] != "curl" {
		t.Errorf("expected [curl], got %v", notIn)
	}
}

func TestSaveState_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "sub", "deep", "state.json")
	m := &yum.YumManager{
		Executor:  testutil.NewMockExecutor(),
		StatePath: func() string { return nestedPath },
	}
	if err := m.SaveState(yum.NewState()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(nestedPath); err != nil {
		t.Errorf("state file not created: %v", err)
	}
}

func TestSaveState_IsMode0600(t *testing.T) {
	m := testManager(t)
	if err := m.SaveState(yum.NewState()); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(m.StatePath())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected mode 0600, got %o", info.Mode().Perm())
	}
}

func TestSaveState_Version(t *testing.T) {
	m := testManager(t)
	s := yum.NewState()
	if err := m.SaveState(s); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(m.StatePath())
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if v, _ := raw["version"].(float64); int(v) != yum.StateVersion {
		t.Errorf("expected version %d, got %v", yum.StateVersion, raw["version"])
	}
}
