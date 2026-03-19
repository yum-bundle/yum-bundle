package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	yumfilePath string
	noUpdate    bool
	version     = "dev"
	commit      = "unknown"
)

// rootCmd is the base command for yum-bundle.
var rootCmd = &cobra.Command{
	Use:   "yum-bundle",
	Short: "A declarative package manager for yum/dnf",
	Long: `yum-bundle provides a simple, declarative, and shareable way to manage
yum/dnf packages and repositories on RPM-based systems, inspired by Homebrew's
brew bundle.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Version = version + " (" + commit + ")"
	rootCmd.PersistentFlags().StringVarP(&yumfilePath, "file", "f", "Yumfile", "Path to Yumfile")
	rootCmd.PersistentFlags().BoolVar(&noUpdate, "no-update", false, "Skip updating package metadata before installing")
}

// getEuid is the function used to get effective UID (overridable for testing)
var getEuid = os.Geteuid

func checkRoot() error {
	if getEuid() != 0 {
		return fmt.Errorf("this command requires root privileges. Please run with sudo")
	}
	return nil
}

// SetGetEuid sets the getEuid function (for testing only)
func SetGetEuid(f func() int) {
	getEuid = f
}

// ResetGetEuid resets getEuid to the default (for testing only)
func ResetGetEuid() {
	getEuid = os.Geteuid
}
