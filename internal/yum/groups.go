package yum

import (
	"fmt"
	"regexp"
	"strings"
)

// groupNameRE validates package group names: must start with alphanumeric,
// followed by alphanumerics, spaces, hyphens, dots, underscores, plus signs,
// or parentheses. Covers both group IDs ("development") and display names
// ("Development Tools", "GNOME Desktop Environment").
var groupNameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 ._+()\-]*$`)

func validateGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	if !groupNameRE.MatchString(name) {
		return fmt.Errorf("invalid group name %q", name)
	}
	return nil
}

// IsGroupInstalled checks if a package group is installed on the system.
// Uses "dnf/yum grouplist installed" and checks for the group name in the output.
func (m *YumManager) IsGroupInstalled(groupName string) (bool, error) {
	output, err := m.runCommandWithOutput(m.PkgCmd(), "grouplist", "installed")
	if err != nil {
		return false, nil
	}
	lines, err := splitLines(string(output))
	if err != nil {
		return false, err
	}
	lower := strings.ToLower(groupName)
	for _, line := range lines {
		if strings.ToLower(strings.TrimSpace(line)) == lower {
			return true, nil
		}
	}
	return false, nil
}

// InstallGroup installs a package group using dnf/yum groupinstall.
// excludes is an optional list of package patterns to pass as --exclude=<pkg>
// to the dnf/yum groupinstall command.
func (m *YumManager) InstallGroup(groupName string, excludes []string) error {
	if err := validateGroupName(groupName); err != nil {
		return err
	}
	fmt.Printf("Installing group: %s\n", groupName)

	args := append([]string{"groupinstall", "-y"}, m.ProxySetopt()...)
	for _, ex := range excludes {
		args = append(args, "--exclude="+ex)
	}
	args = append(args, groupName)
	if err := m.runCommand(m.PkgCmd(), args...); err != nil {
		return wrapCommandError(err, "install group", groupName)
	}

	fmt.Printf("✓ Group %s installed successfully\n", groupName)
	return nil
}

// RemoveGroup removes a package group using dnf/yum groupremove.
func (m *YumManager) RemoveGroup(groupName string) error {
	fmt.Printf("Removing group: %s\n", groupName)

	args := append([]string{"groupremove", "-y"}, m.ProxySetopt()...)
	args = append(args, groupName)
	if err := m.runCommand(m.PkgCmd(), args...); err != nil {
		return wrapCommandError(err, "remove group", groupName)
	}

	fmt.Printf("✓ Group %s removed successfully\n", groupName)
	return nil
}
