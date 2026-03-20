package yum

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// coprRepoPattern validates "user/project" format.
var coprRepoPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

// EnableCOPR enables a COPR repository using dnf copr enable.
// COPR is a Fedora community package repository service, analogous to Ubuntu PPAs.
// Requires dnf and the dnf-plugins-core package (provides the copr subcommand).
// The copr argument must be in "user/project" format.
func (m *YumManager) EnableCOPR(copr string) error {
	if !coprRepoPattern.MatchString(copr) {
		return fmt.Errorf("invalid COPR argument %q: expected \"user/project\" format", copr)
	}

	if !m.IsDNF() {
		return fmt.Errorf("COPR requires dnf (dnf not found on this system)")
	}

	fmt.Printf("Enabling COPR repo: %s\n", copr)

	// Idempotency: check if the .repo file already exists before calling dnf.
	if m.isCOPREnabled(copr) {
		fmt.Printf("✓ COPR repo %s already enabled\n", copr)
		return nil
	}

	if err := m.runCommand("dnf", "copr", "enable", "-y", copr); err != nil {
		return wrapCommandError(err, "enable COPR", copr)
	}

	fmt.Printf("✓ COPR repo %s enabled\n", copr)
	return nil
}

// CoprRepoPathFor returns the expected .repo file path dnf copr creates.
// dnf copr enable writes: _copr:copr.fedorainfracloud.org:<user>:<project>.repo
func (m *YumManager) CoprRepoPathFor(copr string) string {
	parts := strings.SplitN(copr, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return filepath.Join(m.ReposDir,
		fmt.Sprintf("_copr:copr.fedorainfracloud.org:%s:%s.repo", parts[0], parts[1]))
}

// isCOPREnabled checks whether the expected COPR .repo file exists.
func (m *YumManager) isCOPREnabled(copr string) bool {
	path := m.CoprRepoPathFor(copr)
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
