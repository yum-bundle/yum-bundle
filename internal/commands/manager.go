package commands

import "github.com/yum-bundle/yum-bundle/internal/yum"

// mgr is the YumManager used by all commands.
var mgr = yum.NewYumManager()
