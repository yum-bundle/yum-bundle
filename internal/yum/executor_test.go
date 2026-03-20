package yum_test

import (
	"errors"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/testutil"
	"github.com/yum-bundle/yum-bundle/internal/yum"
)

func TestPkgCmd_PrefersDNF(t *testing.T) {
	mgr := &yum.YumManager{
		Executor: testutil.NewMockExecutor(),
		LookPath: func(name string) (string, error) {
			if name == "dnf" {
				return "/usr/bin/dnf", nil
			}
			return "", errors.New("not found")
		},
	}
	if got := mgr.PkgCmd(); got != "dnf" {
		t.Errorf("expected dnf, got %s", got)
	}
}

func TestPkgCmd_FallsBackToYum(t *testing.T) {
	mgr := &yum.YumManager{
		Executor: testutil.NewMockExecutor(),
		LookPath: func(_ string) (string, error) {
			return "", errors.New("not found")
		},
	}
	if got := mgr.PkgCmd(); got != "yum" {
		t.Errorf("expected yum, got %s", got)
	}
}

func TestPkgCmd_Cached(t *testing.T) {
	calls := 0
	mgr := &yum.YumManager{
		Executor: testutil.NewMockExecutor(),
		LookPath: func(_ string) (string, error) {
			calls++
			return "/usr/bin/dnf", nil
		},
	}
	mgr.PkgCmd()
	mgr.PkgCmd()
	if calls != 1 {
		t.Errorf("expected LookPath called once (cached), got %d", calls)
	}
}
