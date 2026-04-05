package yum

import (
	"fmt"
	"regexp"
	"strings"
)

// modulePattern validates "name:stream" format for DNF modules.
var modulePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+:[a-zA-Z0-9_.-]+$`)

// EnableModule enables a DNF module stream.
// Modules are a dnf-only feature available on RHEL8+, CentOS Stream 8+, Fedora 28+.
// The module argument must be in "name:stream" format (e.g. "nodejs:18").
func (m *YumManager) EnableModule(module string) error {
	if !modulePattern.MatchString(module) {
		return fmt.Errorf("invalid module argument %q: expected \"name:stream\" format (e.g. \"nodejs:18\")", module)
	}

	if !m.IsDNF() {
		return fmt.Errorf("DNF modules require dnf (dnf not found on this system)")
	}

	fmt.Printf("Enabling DNF module: %s\n", module)

	if m.isModuleEnabled(module) {
		fmt.Printf("✓ Module %s already enabled\n", module)
		return nil
	}

	args := append([]string{"module", "enable", "-y"}, m.ProxySetopt()...)
	args = append(args, module)
	if err := m.runCommand("dnf", args...); err != nil {
		return wrapCommandError(err, "enable module", module)
	}

	fmt.Printf("✓ Module %s enabled\n", module)
	return nil
}

// isModuleEnabled checks whether the module stream is already enabled via
// "dnf module list --enabled <name>". Returns false on any error.
func (m *YumManager) isModuleEnabled(module string) bool {
	name := moduleNameOnly(module)
	stream := moduleStreamOnly(module)

	output, err := m.runCommandWithOutput("dnf", "module", "list", "--enabled", name)
	if err != nil {
		return false
	}

	// Look for a line containing the stream name marked as [e] (enabled)
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, stream) && strings.Contains(line, "[e]") {
			return true
		}
	}
	return false
}

func moduleNameOnly(module string) string {
	name, _, _ := strings.Cut(module, ":")
	return name
}

func moduleStreamOnly(module string) string {
	_, stream, _ := strings.Cut(module, ":")
	return stream
}
