package yum

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// packageNameRE validates RPM package names: must start with an alphanumeric or
// underscore, followed by alphanumerics, hyphens, underscores, dots, or plus signs.
var packageNameRE = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._+\-]*$`)

// validatePackageName checks that a package spec has a valid RPM package name.
// Version-pinned specs ("name = version", "name-version") are split and only
// the name portion is validated.
func validatePackageName(spec string) error {
	name := extractNameForValidation(spec)
	if !packageNameRE.MatchString(name) {
		return fmt.Errorf("invalid package name %q: must match RPM naming convention", name)
	}
	return nil
}

func extractNameForValidation(spec string) string {
	// "name = version" format
	if idx := strings.Index(spec, " = "); idx > 0 {
		return strings.TrimSpace(spec[:idx])
	}
	// "name=version" compact form
	if idx := strings.Index(spec, "="); idx > 0 {
		return spec[:idx]
	}
	// "name-version" where version starts with digit
	if idx := lastHyphenBeforeVersion(spec); idx > 0 {
		return spec[:idx]
	}
	return spec
}

// lastHyphenBeforeVersion finds the last hyphen followed by a digit.
func lastHyphenBeforeVersion(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return i
		}
	}
	return -1
}

// IsPackageInstalled checks if a package is installed on the system via rpm -q.
// Returns (false, nil) when the package is not installed (rpm exits non-zero).
// Returns (false, err) only on unexpected errors (not exit code 1 from rpm -q).
func (m *YumManager) IsPackageInstalled(packageName string) (bool, error) {
	err := m.runCommand("rpm", "-q", "--quiet", packageName)
	if err != nil {
		// rpm -q exits 1 when the package is not installed; treat that as "not installed"
		return false, nil
	}
	return true, nil
}

// InstallPackage installs a package using dnf/yum.
// excludes is an optional list of package patterns to pass as --exclude=<pkg>
// to the dnf/yum install command.
func (m *YumManager) InstallPackage(spec string, excludes []string) error {
	if spec == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	if err := validatePackageName(extractNameForValidation(spec)); err != nil {
		return err
	}
	fmt.Printf("Installing package: %s\n", spec)

	args := append([]string{"install", "-y"}, m.ProxySetopt()...)
	for _, ex := range excludes {
		args = append(args, "--exclude="+ex)
	}
	args = append(args, spec)
	if err := m.runCommand(m.PkgCmd(), args...); err != nil {
		return wrapCommandError(err, "install package", spec)
	}

	fmt.Printf("✓ Package %s installed successfully\n", spec)
	return nil
}

// MakecacheOrUpdate runs dnf makecache (or yum makecache) to refresh metadata.
// With dnf this is equivalent to apt-get update; yum equivalent is yum makecache.
func (m *YumManager) MakecacheOrUpdate() error {
	fmt.Println("Refreshing package metadata...")

	args := append([]string{"makecache"}, m.ProxySetopt()...)
	if err := m.runCommand(m.PkgCmd(), args...); err != nil {
		return wrapCommandError(err, "refresh package metadata", "")
	}

	fmt.Println("✓ Package metadata refreshed")
	return nil
}

// RemovePackage removes a package using dnf/yum.
func (m *YumManager) RemovePackage(packageName string) error {
	fmt.Printf("Removing package: %s\n", packageName)

	args := append([]string{"remove", "-y"}, m.ProxySetopt()...)
	args = append(args, packageName)
	if err := m.runCommand(m.PkgCmd(), args...); err != nil {
		return wrapCommandError(err, "remove package", packageName)
	}

	fmt.Printf("✓ Package %s removed successfully\n", packageName)
	return nil
}

// AutoRemove runs dnf/yum autoremove to remove orphaned dependencies.
func (m *YumManager) AutoRemove() error {
	fmt.Println("Removing orphaned dependencies...")

	args := append([]string{"autoremove", "-y"}, m.ProxySetopt()...)
	if err := m.runCommand(m.PkgCmd(), args...); err != nil {
		return wrapCommandError(err, "autoremove packages", "")
	}

	fmt.Println("✓ Orphaned dependencies removed")
	return nil
}

// GetInstalledVersion returns the installed version string for a package.
// Returns ("", nil) when the package is not installed.
func (m *YumManager) GetInstalledVersion(packageName string) (string, error) {
	output, err := m.runCommandWithOutput(
		"rpm", "-q", "--queryformat", "%{VERSION}-%{RELEASE}", packageName,
	)
	if err != nil {
		// rpm -q exits 1 when not installed
		return "", nil
	}
	ver := strings.TrimSpace(string(output))
	// rpm outputs "package not installed" text when missing; treat as empty
	if strings.Contains(ver, "not installed") {
		return "", nil
	}
	return ver, nil
}

// availableVersionRE extracts the version from "dnf info" or "dnf repoquery" output.
var availableVersionRE = regexp.MustCompile(`(?m)^Version\s*:\s*(.+)$`)
var availableReleaseRE = regexp.MustCompile(`(?m)^Release\s*:\s*(.+)$`)

// GetAvailableVersion returns the latest available version for a package from
// the configured repositories. Returns ("", nil) when the package is not found.
func (m *YumManager) GetAvailableVersion(packageName string) (string, error) {
	output, err := m.runCommandWithOutput(m.PkgCmd(), "info", "--available", packageName)
	if err != nil {
		return "", nil
	}
	text := string(output)

	verMatches := availableVersionRE.FindStringSubmatch(text)
	relMatches := availableReleaseRE.FindStringSubmatch(text)
	if len(verMatches) < 2 {
		return "", nil
	}

	ver := strings.TrimSpace(verMatches[1])
	if len(relMatches) >= 2 {
		ver += "-" + strings.TrimSpace(relMatches[1])
	}
	return ver, nil
}

// GetAllInstalledPackages returns names of all explicitly installed packages
// (i.e. packages not installed as dependencies).
// Uses "dnf history userinstalled" when dnf is available, else falls back to
// "rpm -qa --queryformat '%{NAME}\n'" which includes all packages.
func (m *YumManager) GetAllInstalledPackages() ([]string, error) {
	var output []byte
	var err error

	if m.IsDNF() {
		output, err = m.runCommandWithOutput("dnf", "history", "userinstalled")
	} else {
		output, err = m.runCommandWithOutput("rpm", "-qa", "--queryformat", "%{NAME}\\n")
	}
	if err != nil {
		return nil, wrapCommandError(err, "list installed packages", "")
	}

	lines, err := splitLines(string(output))
	if err != nil {
		return nil, fmt.Errorf("parsing installed packages output: %w", err)
	}

	var packages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Last metadata") || strings.HasPrefix(line, "Installed Packages") {
			continue
		}
		packages = append(packages, line)
	}
	return packages, nil
}

// splitLines splits a string by newlines.
func splitLines(s string) ([]string, error) {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}
