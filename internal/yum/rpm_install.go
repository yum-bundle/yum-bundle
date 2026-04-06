package yum

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

// InstallRPMFromURL installs an RPM package directly from an HTTPS URL.
// Uses dnf if available (handles dependencies), otherwise falls back to rpm -i.
// This is useful for repo-setup packages like epel-release, rpmfusion-free-release, etc.
// Only https:// URLs are accepted.
// checksumAlgo and checksum are optional: pass empty strings to skip verification.
// When a checksum is provided, the RPM is downloaded and verified before installation.
func (m *YumManager) InstallRPMFromURL(rpmURL, checksumAlgo, checksum string) error {
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

	// When a checksum is specified, download the RPM ourselves to verify it,
	// then install from the local temporary file.
	installTarget := rpmURL
	if checksumAlgo != "" {
		resp, err := m.HTTPGet(rpmURL)
		if err != nil {
			return fmt.Errorf("download RPM: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download RPM: HTTP %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read RPM data: %w", err)
		}

		if err := verifyChecksum(data, checksumAlgo, checksum); err != nil {
			return fmt.Errorf("RPM checksum verification failed: %w", err)
		}

		tmpFile, err := os.CreateTemp("", "yum-bundle-*.rpm")
		if err != nil {
			return fmt.Errorf("create temp file for RPM: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			return fmt.Errorf("write temp RPM file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("close temp RPM file: %w", err)
		}
		installTarget = tmpFile.Name()
	}

	// dnf can install RPMs from URLs or local paths directly and resolves deps
	var err error
	if m.IsDNF() {
		err = m.runCommand("dnf", "install", "-y", installTarget)
	} else {
		err = m.runCommand("yum", "install", "-y", installTarget)
	}
	if err != nil {
		return wrapCommandError(err, "install RPM from URL", rpmURL)
	}

	fmt.Printf("✓ RPM installed from: %s\n", rpmURL)
	return nil
}

// validateRPMURL ensures the URL uses https:// and ends with .rpm.
func validateRPMURL(rpmURL string) error {
	u, err := validateHTTPSURL(rpmURL, "RPM URL")
	if err != nil {
		return err
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
	if idx := yumfile.LastHyphenBeforeVersion(name); idx > 0 {
		name = name[:idx]
		// Strip -version (another -NNN component)
		if idx2 := yumfile.LastHyphenBeforeVersion(name); idx2 > 0 {
			name = name[:idx2]
		}
	}

	return name
}
