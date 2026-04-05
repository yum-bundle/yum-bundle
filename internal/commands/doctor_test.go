package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/testutil"
	"github.com/yum-bundle/yum-bundle/internal/yum"
)

// newDoctorCmd creates a minimal cobra.Command for testing runDoctor.
func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{}
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	return cmd
}

// writeTempDoctorYumfile writes a Yumfile with the given content to a temp dir
// and returns its path.
func writeTempDoctorYumfile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "Yumfile")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write temp Yumfile: %v", err)
	}
	return path
}

// makePathWith creates a temp dir containing fake executables for the given
// binary names (zero-byte files with execute permission) and returns the dir.
// The caller should prepend this dir to PATH.
func makePathWith(t *testing.T, binaries ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range binaries {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0755); err != nil { //nolint:gosec
			t.Fatalf("create fake binary %s: %v", name, err)
		}
	}
	return dir
}

// setupDoctorTest sets up a doctor test environment: writes a Yumfile, sets
// yumfilePath, configures mgr with a temp state dir, and manipulates PATH so
// that only the named binaries (from binaries) are found via exec.LookPath.
// It restores all globals on cleanup.
func setupDoctorTest(t *testing.T, yumfileContent string, binaries []string) *cobra.Command {
	t.Helper()

	// Write the Yumfile
	path := writeTempDoctorYumfile(t, yumfileContent)

	// Save and restore package-level globals
	origYumfilePath := yumfilePath
	origMgr := mgr
	origDoctorYumfileOnly := doctorYumfileOnly

	t.Cleanup(func() {
		yumfilePath = origYumfilePath
		mgr = origMgr
		doctorYumfileOnly = origDoctorYumfileOnly
	})

	yumfilePath = path
	doctorYumfileOnly = false

	// Set up a fake mgr with temp state dir
	stateDir := t.TempDir()
	mgr = &yum.YumManager{
		Executor:      testutil.NewMockExecutor(),
		ReposDir:      filepath.Join(stateDir, "yum.repos.d"),
		ReposPrefix:   "yum-bundle-",
		KeyDir:        filepath.Join(stateDir, "rpm-gpg"),
		KeyPrefix:     "yum-bundle-",
		OsReleasePath: "/dev/null",
		StatePath:     func() string { return filepath.Join(stateDir, "state.json") },
		LookPath:      func(_ string) (string, error) { return "", os.ErrNotExist }, // unused; exec.LookPath is used in doctor.go
	}

	// Manipulate PATH to control which binaries are visible to exec.LookPath
	fakeDir := makePathWith(t, binaries...)
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", origPath) }) //nolint:errcheck
	// Prepend fakeDir and use only it (no real /usr/bin etc.) so we have
	// deterministic control.  We keep /dev/null as the rest so no real
	// binaries leak through.
	os.Setenv("PATH", fakeDir) //nolint:errcheck

	cmd := newDoctorCmd()
	return cmd
}

func doctorOutput(cmd *cobra.Command) (stdout, stderr string) {
	if ob, ok := cmd.OutOrStdout().(*bytes.Buffer); ok {
		stdout = ob.String()
	}
	if eb, ok := cmd.ErrOrStderr().(*bytes.Buffer); ok {
		stderr = eb.String()
	}
	return
}

// TestDoctor_CoprRequiresDNF checks that doctor fails when the Yumfile has a
// copr directive and only yum (not dnf) is available.
func TestDoctor_CoprRequiresDNF(t *testing.T) {
	content := "yum vim\ncopr atim/lazygit\n"
	// Provide yum and rpm but NOT dnf
	cmd := setupDoctorTest(t, content, []string{"yum", "rpm"})

	err := runDoctor(cmd, nil)
	if err == nil {
		t.Fatal("expected error when copr used without dnf")
	}

	_, stderr := doctorOutput(cmd)
	if !strings.Contains(stderr, "copr") {
		t.Errorf("expected stderr to mention 'copr', got: %s", stderr)
	}
	if !strings.Contains(stderr, "requires dnf") {
		t.Errorf("expected stderr to mention 'requires dnf', got: %s", stderr)
	}
	if !strings.Contains(stderr, "line 2") {
		t.Errorf("expected stderr to mention 'line 2', got: %s", stderr)
	}
}

// TestDoctor_ModuleRequiresDNF checks that doctor fails when the Yumfile has a
// module directive and only yum (not dnf) is available.
func TestDoctor_ModuleRequiresDNF(t *testing.T) {
	content := "yum vim\nmodule nodejs:18\n"
	// Provide yum and rpm but NOT dnf
	cmd := setupDoctorTest(t, content, []string{"yum", "rpm"})

	err := runDoctor(cmd, nil)
	if err == nil {
		t.Fatal("expected error when module used without dnf")
	}

	_, stderr := doctorOutput(cmd)
	if !strings.Contains(stderr, "module") {
		t.Errorf("expected stderr to mention 'module', got: %s", stderr)
	}
	if !strings.Contains(stderr, "requires dnf") {
		t.Errorf("expected stderr to mention 'requires dnf', got: %s", stderr)
	}
	if !strings.Contains(stderr, "line 2") {
		t.Errorf("expected stderr to mention 'line 2', got: %s", stderr)
	}
}

// TestDoctor_CoprAndModuleOKWithDNF checks that doctor passes when the Yumfile
// has copr/module directives and dnf IS available.
func TestDoctor_CoprAndModuleOKWithDNF(t *testing.T) {
	content := "copr atim/lazygit\nmodule nodejs:18\n"
	// Provide dnf and rpm
	cmd := setupDoctorTest(t, content, []string{"dnf", "rpm"})

	err := runDoctor(cmd, nil)
	if err != nil {
		t.Fatalf("expected no error when copr/module used with dnf: %v", err)
	}

	_, stderr := doctorOutput(cmd)
	if strings.Contains(stderr, "requires dnf") {
		t.Errorf("unexpected 'requires dnf' in stderr: %s", stderr)
	}
}

// TestDoctor_NoCoprOrModulePassesWithYumOnly checks that doctor does not report
// dnf-only directive errors when the Yumfile has no copr or module entries.
func TestDoctor_NoCoprOrModulePassesWithYumOnly(t *testing.T) {
	content := "yum vim\nyum curl\n"
	// Provide yum and rpm but NOT dnf
	cmd := setupDoctorTest(t, content, []string{"yum", "rpm"})

	err := runDoctor(cmd, nil)
	if err != nil {
		t.Fatalf("expected no error for yum-only Yumfile without copr/module: %v", err)
	}

	_, stderr := doctorOutput(cmd)
	if strings.Contains(stderr, "requires dnf") {
		t.Errorf("unexpected 'requires dnf' in stderr: %s", stderr)
	}
}
