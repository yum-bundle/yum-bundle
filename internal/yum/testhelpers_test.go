package yum_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/testutil"
	"github.com/yum-bundle/yum-bundle/internal/yum"
)

// testManager creates a YumManager wired for testing: real filesystem under
// t.TempDir(), a MockExecutor, and a LookPath stub that finds dnf but not yum.
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
