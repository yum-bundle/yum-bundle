package commands

import "github.com/yum-bundle/yum-bundle/internal/yum"

// mgr is the YumManager used by all commands.
var mgr = yum.NewYumManager()

// SetManager overrides the package-level manager singleton.
// This is intended for use in tests to inject a mock or pre-configured manager.
func SetManager(m *yum.YumManager) {
	mgr = m
}
