package commands

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "List Yumfile packages that have available upgrades",
	Long: `Outdated compares installed versions of Yumfile packages to the available
versions and lists packages that have upgrades available.
Exit code is 0 only when no packages are outdated (suitable for CI).`,
	RunE: runOutdated,
}

func init() {
	rootCmd.AddCommand(outdatedCmd)
}

// OutdatedEntry holds one outdated package line.
type OutdatedEntry struct {
	Name      string
	Installed string
	Available string
}

// collectOutdated returns Yumfile packages that have a newer available version.
func collectOutdated(yumfilePath string) (outdated []OutdatedEntry, numYum int, err error) {
	entries, err := yumfile.Parse(yumfilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, fmt.Errorf("Yumfile not found: %s", yumfilePath)
		}
		return nil, 0, fmt.Errorf("parse Yumfile: %w", err)
	}

	var packages []string
	for _, e := range entries {
		if e.Type != yumfile.EntryTypeYum {
			continue
		}
		numYum++
		packages = append(packages, yumfile.ExtractPkgName(e.Value))
	}

	for _, pkg := range packages {
		installed, err := mgr.GetInstalledVersion(pkg)
		if err != nil {
			return nil, numYum, fmt.Errorf("get installed version of %s: %w", pkg, err)
		}
		if installed == "" {
			continue
		}
		available, err := mgr.GetAvailableVersion(pkg)
		if err != nil {
			return nil, numYum, fmt.Errorf("get available version of %s: %w", pkg, err)
		}
		if available == "" || available == installed {
			continue
		}
		outdated = append(outdated, OutdatedEntry{pkg, installed, available})
	}

	sort.Slice(outdated, func(i, j int) bool { return outdated[i].Name < outdated[j].Name })
	return outdated, numYum, nil
}

func runOutdated(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()

	outdated, numYum, err := collectOutdated(yumfilePath)
	if err != nil {
		return err
	}

	if len(outdated) == 0 {
		if numYum == 0 {
			fmt.Fprintln(w, "No yum packages in Yumfile.")
		}
		return nil
	}

	for _, e := range outdated {
		fmt.Fprintf(w, "%s (installed: %s, available: %s)\n", e.Name, e.Installed, e.Available)
	}
	return fmt.Errorf("%d package(s) have upgrades available", len(outdated))
}
