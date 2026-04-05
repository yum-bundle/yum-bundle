package yum

import (
	"fmt"
	"os"
	"strings"
)

// EnableEPEL enables EPEL (Extra Packages for Enterprise Linux).
// On RHEL/CentOS/Rocky/AlmaLinux: installs epel-release via dnf/yum.
// On Fedora: EPEL is for RHEL-family only; a warning is printed and the
// directive is skipped since Fedora provides its packages directly.
// Idempotent: checks /etc/yum.repos.d/epel.repo before installing.
func (m *YumManager) EnableEPEL() error {
	fmt.Println("Enabling EPEL...")

	if m.isFedora() {
		fmt.Println("⚠️  Warning: EPEL is designed for RHEL-family distros. Skipping on Fedora.")
		return nil
	}

	if m.isEPELEnabled() {
		fmt.Println("✓ EPEL already enabled")
		return nil
	}

	args := append([]string{"install", "-y"}, m.ProxySetopt()...)
	args = append(args, "epel-release")
	if err := m.runCommand(m.PkgCmd(), args...); err != nil {
		return wrapCommandError(err, "enable EPEL", "epel-release")
	}

	fmt.Println("✓ EPEL enabled")
	return nil
}

// isEPELEnabled checks whether EPEL is already configured by looking for
// /etc/yum.repos.d/epel.repo.
func (m *YumManager) isEPELEnabled() bool {
	_, err := os.Stat(m.ReposDir + "/epel.repo")
	return err == nil
}

// isFedora reads /etc/os-release and returns true when the distro is Fedora.
func (m *YumManager) isFedora() bool {
	return m.isDistroID("fedora")
}

// IsRHELFamily returns true when the distro is a RHEL-family system
// (RHEL, CentOS, Rocky, AlmaLinux, Oracle, etc.).
func (m *YumManager) IsRHELFamily() bool {
	data, err := os.ReadFile(m.OsReleasePath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, "\"")
		if key == "ID_LIKE" {
			for _, word := range strings.Fields(val) {
				if word == "rhel" || word == "centos" || word == "fedora" {
					return true
				}
			}
		}
		if key == "ID" {
			switch val {
			case "rhel", "centos", "rocky", "almalinux", "ol":
				return true
			}
		}
	}
	return false
}

// isDistroID checks if the current distro's ID field matches the given value.
func (m *YumManager) isDistroID(id string) bool {
	data, err := os.ReadFile(m.OsReleasePath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, "\"")
		if key == "ID" && val == id {
			return true
		}
	}
	return false
}
