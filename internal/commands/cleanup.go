package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/yum"
	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

var (
	cleanupForce      bool
	cleanupZap        bool
	cleanupAutoremove bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove packages not listed in Yumfile",
	Long: `Remove packages that were previously installed by yum-bundle but are no longer
listed in the Yumfile.

By default, cleanup only removes packages that yum-bundle itself installed (tracked
in the state file). This is safe to use with container base images or systems where
packages were installed manually.

Use --zap to remove ALL packages not in the Yumfile (dangerous - may break your system).`,
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().BoolVar(&cleanupForce, "force", false, "Actually remove packages (default is dry-run)")
	cleanupCmd.Flags().BoolVar(&cleanupZap, "zap", false, "Remove ALL packages not in Yumfile (dangerous)")
	cleanupCmd.Flags().BoolVar(&cleanupAutoremove, "autoremove", false, "Also run dnf/yum autoremove after cleanup")
	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(_ *cobra.Command, _ []string) error {
	return doCleanup(cleanupForce, cleanupZap, cleanupAutoremove)
}

// doCleanup performs cleanup with explicit parameters.
func doCleanup(force, zap, autoremove bool) error {
	if force {
		if err := checkRoot(); err != nil {
			return err
		}
	}

	fmt.Printf("Reading Yumfile from: %s\n", yumfilePath)

	entries, err := yumfile.Parse(yumfilePath)
	if err != nil {
		return fmt.Errorf("failed to parse Yumfile: %w", err)
	}

	aptfilePackages := extractPackageNames(entries)
	aptfileGroups := extractGroupNames(entries)

	var packagesToRemove []string
	var cachedState *yum.State

	if zap {
		packagesToRemove, err = getPackagesToZap(aptfilePackages)
		if err != nil {
			return err
		}
	} else {
		packagesToRemove, cachedState, err = getPackagesToCleanup(aptfilePackages)
		if err != nil {
			return err
		}
	}

	// Groups are only cleaned up via state (not --zap mode).
	var groupsToRemove []string
	if !zap {
		var state *yum.State
		if cachedState != nil {
			state = cachedState
		} else {
			state, err = mgr.LoadState()
			if err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}
			cachedState = state
		}
		groupsToRemove = state.GetGroupsNotIn(aptfileGroups)
	}

	if len(packagesToRemove) == 0 && len(groupsToRemove) == 0 {
		fmt.Println("✓ Nothing to clean up")
		return nil
	}

	if zap {
		fmt.Printf("\nZAP MODE: The following %d packages are NOT in your Yumfile and will be removed:\n", len(packagesToRemove))
	} else {
		if len(packagesToRemove) > 0 {
			fmt.Printf("\nThe following %d packages were installed by yum-bundle but are no longer in your Yumfile:\n", len(packagesToRemove))
		}
	}

	for _, pkg := range packagesToRemove {
		fmt.Printf("  - %s\n", pkg)
	}

	if len(groupsToRemove) > 0 {
		fmt.Printf("\nThe following %d groups were installed by yum-bundle but are no longer in your Yumfile:\n", len(groupsToRemove))
		for _, g := range groupsToRemove {
			fmt.Printf("  - %s\n", g)
		}
	}
	fmt.Println()

	if !force {
		fmt.Println("Run with --force to actually remove these packages/groups")
		return nil
	}

	if zap {
		fmt.Print("WARNING: This will remove packages that may be critical to your system.\n")
		fmt.Print("Type 'yes' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if strings.TrimSpace(confirmation) != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	var state *yum.State
	if cachedState != nil {
		state = cachedState
	} else {
		state, err = mgr.LoadState()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}
	}

	if len(packagesToRemove) > 0 {
		fmt.Printf("Removing %d packages...\n", len(packagesToRemove))
		for _, pkg := range packagesToRemove {
			if err := mgr.RemovePackage(pkg); err != nil {
				return fmt.Errorf("failed to remove package %s: %w", pkg, err)
			}
			state.RemovePackage(pkg)
		}
		fmt.Printf("✓ Removed %d packages\n", len(packagesToRemove))
	}

	if len(groupsToRemove) > 0 {
		fmt.Printf("Removing %d groups...\n", len(groupsToRemove))
		for _, g := range groupsToRemove {
			if err := mgr.RemoveGroup(g); err != nil {
				return fmt.Errorf("failed to remove group %s: %w", g, err)
			}
			state.RemoveGroup(g)
		}
		fmt.Printf("✓ Removed %d groups\n", len(groupsToRemove))
	}

	if err := mgr.SaveState(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if autoremove {
		if err := mgr.AutoRemove(); err != nil {
			return fmt.Errorf("failed to autoremove: %w", err)
		}
	}

	return nil
}

// extractGroupNames returns deduplicated group names from group entries.
func extractGroupNames(entries []yumfile.Entry) []string {
	seen := make(map[string]bool)
	var names []string
	for _, entry := range entries {
		if entry.Type != yumfile.EntryTypeGroup {
			continue
		}
		if !seen[entry.Value] {
			seen[entry.Value] = true
			names = append(names, entry.Value)
		}
	}
	return names
}

// extractPackageNames returns deduplicated package names from yum entries.
func extractPackageNames(entries []yumfile.Entry) []string {
	seen := make(map[string]bool)
	var names []string
	for _, entry := range entries {
		if entry.Type != yumfile.EntryTypeYum {
			continue
		}
		pkgName := yumfile.ExtractPkgName(entry.Value)
		if !seen[pkgName] {
			seen[pkgName] = true
			names = append(names, pkgName)
		}
	}
	return names
}

// getPackagesToCleanup returns packages tracked by yum-bundle but no longer in Yumfile.
func getPackagesToCleanup(yumfilePackages []string) ([]string, *yum.State, error) {
	state, err := mgr.LoadState()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load state: %w", err)
	}
	return state.GetPackagesNotIn(yumfilePackages), state, nil
}

// getPackagesToZap returns ALL installed packages not in Yumfile.
func getPackagesToZap(yumfilePackages []string) ([]string, error) {
	allInstalled, err := mgr.GetAllInstalledPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to get installed packages: %w", err)
	}

	yumfileSet := make(map[string]bool)
	for _, pkg := range yumfilePackages {
		yumfileSet[pkg] = true
	}

	var toRemove []string
	for _, pkg := range allInstalled {
		if !yumfileSet[pkg] {
			toRemove = append(toRemove, pkg)
		}
	}
	return toRemove, nil
}
