package yum

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// InstallRPMFromURL installs an RPM package directly from an HTTPS URL.
// Uses dnf if available (handles dependencies), otherwise falls back to rpm -i.
// This is useful for repo-setup packages like epel-release, rpmfusion-free-release, etc.
// Only https:// URLs are accepted.
func (m *YumManager) InstallRPMFromURL(rpmURL string) error {
	if err := validateRPMURL(rpmURL); err != nil {
		return err
	}
	fmt.Printf("Installing RPM from URL: %s\n", rpmURL)

	// Check if this RPM is already installed by parsing the package name from the URL
	pkgName := rpmNameFromURL(rpmURL)
	if pkgName != "" {
		installed, _ := m.IsPackageInstalled(pkgName)
		if installed {
			fmt.Printf("✓ RPM package %s already installed\n", pkgName)
			return nil
		}
	}

	// dnf can install RPMs from URLs directly and resolves deps
	var err error
	if m.IsDNF() {
		err = m.runCommand("dnf", "install", "-y", rpmURL)
	} else {
		err = m.runCommand("yum", "install", "-y", rpmURL)
	}
	if err != nil {
		return wrapCommandError(err, "install RPM from URL", rpmURL)
	}

	fmt.Printf("✓ RPM installed from: %s\n", rpmURL)
	return nil
}

// validateRPMURL ensures the URL uses https:// and ends with .rpm.
func validateRPMURL(rpmURL string) error {
	u, err := url.Parse(rpmURL)
	if err != nil {
		return fmt.Errorf("invalid RPM URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		// ok
	case "http":
		return fmt.Errorf("RPM URL must use https://, not http:// (rejected for security)")
	case "file":
		return fmt.Errorf("file:// RPM URLs are not allowed (rejected for security)")
	case "":
		return fmt.Errorf("invalid RPM URL: missing scheme (use https://)")
	default:
		return fmt.Errorf("RPM URL scheme %q not allowed; use https://", u.Scheme)
	}
	if !strings.HasSuffix(u.Path, ".rpm") {
		return fmt.Errorf("RPM URL must end with .rpm")
	}
	return nil
}

// rpmNameFromURL extracts the package name (the NEVRA name component) from an RPM
// filename URL. For example:
//
//	https://example.com/epel-release-9-7.el9.noarch.rpm → "epel-release"
//
// Returns "" if the filename cannot be parsed.
func rpmNameFromURL(rpmURL string) string {
	u, err := url.Parse(rpmURL)
	if err != nil {
		return ""
	}
	base := filepath.Base(u.Path)
	// Strip .rpm suffix
	name := strings.TrimSuffix(base, ".rpm")

	// RPM NEVRA format: name-version-release.arch
	// We need to strip the arch first: last component after the last dot
	dotIdx := strings.LastIndex(name, ".")
	if dotIdx > 0 {
		name = name[:dotIdx]
	}

	// Now name is "name-version-release"
	// Strip -release (last -NNN component with a digit)
	if idx := lastHyphenBeforeVersion(name); idx > 0 {
		name = name[:idx]
		// Strip -version (another -NNN component)
		if idx2 := lastHyphenBeforeVersion(name); idx2 > 0 {
			name = name[:idx2]
		}
	}

	return name
}
