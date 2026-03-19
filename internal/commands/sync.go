package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

var syncAutoremove bool
var syncDryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Make system match Yumfile (install then cleanup)",
	Long: `Sync makes the system match the Yumfile: install any missing packages and
repositories, then remove packages that yum-bundle previously installed but are
no longer listed in the Yumfile.

This follows the "desired state" paradigm: one command to bring the system in
line with the declared Yumfile.`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncAutoremove, "autoremove", false, "Run dnf/yum autoremove after cleanup")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Only report what would be installed and removed; no changes")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if !syncDryRun {
		if err := checkRoot(); err != nil {
			return err
		}
	}

	if syncDryRun {
		return runSyncDryRun()
	}

	if err := runInstall(cmd, args); err != nil {
		return err
	}

	return doCleanup(true, false, syncAutoremove)
}

func runSyncDryRun() error {
	entries, err := yumfile.Parse(yumfilePath)
	if err != nil {
		return fmt.Errorf("failed to parse Yumfile: %w", err)
	}

	fmt.Printf("Reading Yumfile from: %s (dry-run)\n", yumfilePath)

	state, err := mgr.LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	yumfilePackages := extractPackageNames(entries)

	var wouldInstall, wouldRemove []string
	var wouldAdd []string

	for _, entry := range entries {
		switch entry.Type {
		case yumfile.EntryTypeYum:
			pkgName := yumfile.ExtractPkgName(entry.Value)
			installed, instErr := mgr.IsPackageInstalled(pkgName)
			if instErr != nil || !installed {
				wouldInstall = append(wouldInstall, entry.Value)
			}
		case yumfile.EntryTypeKey:
			wouldAdd = append(wouldAdd, "key "+entry.Value)
		case yumfile.EntryTypeRepo:
			wouldAdd = append(wouldAdd, "repo "+entry.Value)
		case yumfile.EntryTypeBaseurl:
			wouldAdd = append(wouldAdd, "baseurl "+entry.Value)
		case yumfile.EntryTypeCopr:
			wouldAdd = append(wouldAdd, "copr "+entry.Value)
		case yumfile.EntryTypeEPEL:
			wouldAdd = append(wouldAdd, "epel")
		case yumfile.EntryTypeModule:
			wouldAdd = append(wouldAdd, "module "+entry.Value)
		case yumfile.EntryTypeRPM:
			wouldInstall = append(wouldInstall, "rpm "+entry.Value)
		}
	}

	wouldRemove = state.GetPackagesNotIn(yumfilePackages)

	if len(wouldAdd) > 0 {
		fmt.Println("--- would add ---")
		for _, a := range wouldAdd {
			fmt.Printf("  %s\n", a)
		}
	}
	if len(wouldInstall) > 0 {
		fmt.Println("--- would install ---")
		for _, p := range wouldInstall {
			fmt.Printf("  %s\n", p)
		}
	}
	if len(wouldRemove) > 0 {
		fmt.Println("--- would remove ---")
		for _, p := range wouldRemove {
			fmt.Printf("  %s\n", p)
		}
	}
	if len(wouldInstall) == 0 && len(wouldRemove) == 0 && len(wouldAdd) == 0 {
		fmt.Println("Nothing to do; system matches Yumfile.")
	}
	return nil
}
