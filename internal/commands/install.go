package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

var (
	installLock   bool
	installLocked bool
	installDryRun bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install packages and repositories from Yumfile",
	Long: `Read the Yumfile and perform the following operations:
1. Import any specified GPG keys
2. Add any specified repositories (repo, baseurl, copr)
3. Enable EPEL if requested
4. Enable DNF modules if specified
5. Run dnf/yum makecache (unless --no-update is specified)
6. Install all specified packages
7. Install RPMs from URLs if specified`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().BoolVar(&installLock, "lock", false, "After install, write Yumfile.lock with current package versions")
	installCmd.Flags().BoolVar(&installLocked, "locked", false, "Install only versions from Yumfile.lock (fail if lock missing)")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Only report what would be installed/added; do not run dnf/yum or change state")
	rootCmd.AddCommand(installCmd)
	rootCmd.RunE = runInstall
}

func runInstall(_ *cobra.Command, _ []string) error {
	if !installDryRun {
		if err := checkRoot(); err != nil {
			return err
		}
	}

	fmt.Printf("Reading Yumfile from: %s\n", yumfilePath)

	entries, err := yumfile.Parse(yumfilePath)
	if err != nil {
		return fmt.Errorf("failed to parse Yumfile: %w", err)
	}

	fmt.Printf("Found %d entries in Yumfile\n", len(entries))

	if installDryRun {
		return runInstallDryRun(entries)
	}

	state, err := mgr.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	defer func() {
		if saveErr := mgr.SaveState(state); saveErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save state: %v\n", saveErr)
		}
	}()

	var pendingKeyPath string
	var reposAdded bool

	for _, entry := range entries {
		switch entry.Type {
		case yumfile.EntryTypeKey:
			keyPath, err := mgr.ImportGPGKey(entry.Value)
			if err != nil {
				return fmt.Errorf("failed to import GPG key: %w", err)
			}
			pendingKeyPath = keyPath
			state.AddKey(keyPath)

		case yumfile.EntryTypeRepo:
			repoPath, err := mgr.AddRepoFile(entry.Value)
			if err != nil {
				return fmt.Errorf("failed to add repo: %w", err)
			}
			state.AddRepo(repoPath)
			pendingKeyPath = ""
			reposAdded = true

		case yumfile.EntryTypeBaseurl:
			repoPath, err := mgr.AddBaseurlRepo(entry.Value, nil)
			if err != nil {
				return fmt.Errorf("failed to add baseurl repo: %w", err)
			}
			_ = pendingKeyPath // TODO: associate key with baseurl repo
			state.AddRepo(repoPath)
			pendingKeyPath = ""
			reposAdded = true

		case yumfile.EntryTypeCopr:
			if err := mgr.EnableCOPR(entry.Value); err != nil {
				return fmt.Errorf("failed to enable COPR %s: %w", entry.Value, err)
			}
			coprPath := mgr.CoprRepoPathFor(entry.Value)
			state.AddRepo(coprPath)
			reposAdded = true

		case yumfile.EntryTypeEPEL:
			if err := mgr.EnableEPEL(); err != nil {
				return fmt.Errorf("failed to enable EPEL: %w", err)
			}
			reposAdded = true

		case yumfile.EntryTypeModule:
			if err := mgr.EnableModule(entry.Value); err != nil {
				return fmt.Errorf("failed to enable module %s: %w", entry.Value, err)
			}
		}
	}

	if !noUpdate {
		if err := mgr.MakecacheOrUpdate(); err != nil {
			return fmt.Errorf("failed to update package metadata: %w", err)
		}
	} else if reposAdded {
		fmt.Println("Warning: Repositories were added; run without --no-update to fetch package metadata.")
	}

	packagesToInstall := []string{}
	if installLocked {
		specs, err := ReadLockFile()
		if err != nil {
			return err
		}
		packagesToInstall = specs
	} else {
		for _, entry := range entries {
			if entry.Type == yumfile.EntryTypeYum {
				packagesToInstall = append(packagesToInstall, entry.Value)
			}
		}
	}

	if len(packagesToInstall) > 0 {
		fmt.Printf("Installing %d packages...\n", len(packagesToInstall))
		for _, pkg := range packagesToInstall {
			pkgName := yumfile.ExtractPkgName(pkg)
			installed, err := mgr.IsPackageInstalled(pkgName)
			if err != nil {
				fmt.Printf("Warning: could not check if %s is installed: %v\n", pkgName, err)
			}
			if installed {
				fmt.Printf("✓ Package %s is already installed\n", pkgName)
				state.AddPackage(pkgName)
				continue
			}
			if err := mgr.InstallPackage(pkg); err != nil {
				return fmt.Errorf("failed to install package %s: %w", pkg, err)
			}
			state.AddPackage(pkgName)
		}
		fmt.Println("✓ All packages installed successfully")
	} else {
		fmt.Println("No packages to install")
	}

	// Install RPMs from URLs
	for _, entry := range entries {
		if entry.Type == yumfile.EntryTypeRPM {
			if err := mgr.InstallRPMFromURL(entry.Value); err != nil {
				return fmt.Errorf("failed to install RPM from URL %s: %w", entry.Value, err)
			}
		}
	}

	if installLock && len(packagesToInstall) > 0 {
		if err := writeLockFileFromPackages(packagesToInstall); err != nil {
			return fmt.Errorf("failed to write lock file: %w", err)
		}
	}

	return nil
}

func writeLockFileFromPackages(packages []string) error {
	locked, _ := resolveInstalledVersions(packages)
	if len(locked) == 0 {
		return nil
	}
	return writeLockFileEntries(locked)
}

func runInstallDryRun(entries []yumfile.Entry) error {
	repos, err := mgr.ListCustomRepos()
	if err != nil {
		return fmt.Errorf("failed to list repos: %w", err)
	}
	repoSet := make(map[string]bool)
	for _, r := range repos {
		repoSet[r.YumfileLine] = true
	}

	var wouldAddKeys, wouldAddRepos, wouldInstall, wouldInstallRPM []string

	for _, entry := range entries {
		switch entry.Type {
		case yumfile.EntryTypeKey:
			keyPath := mgr.KeyPathForURL(entry.Value)
			if _, err := os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
				wouldAddKeys = append(wouldAddKeys, entry.Value)
			}
		case yumfile.EntryTypeRepo, yumfile.EntryTypeBaseurl, yumfile.EntryTypeCopr:
			wouldAddRepos = append(wouldAddRepos, string(entry.Type)+" "+entry.Value)
		case yumfile.EntryTypeEPEL:
			wouldAddRepos = append(wouldAddRepos, "epel")
		case yumfile.EntryTypeModule:
			wouldAddRepos = append(wouldAddRepos, "module "+entry.Value)
		case yumfile.EntryTypeYum:
			pkgName := yumfile.ExtractPkgName(entry.Value)
			installed, err := mgr.IsPackageInstalled(pkgName)
			if err != nil || !installed {
				wouldInstall = append(wouldInstall, entry.Value)
			}
		case yumfile.EntryTypeRPM:
			wouldInstallRPM = append(wouldInstallRPM, entry.Value)
		}
	}

	fmt.Println("--- dry-run: would perform the following ---")
	if len(wouldAddKeys) > 0 {
		for _, u := range wouldAddKeys {
			fmt.Printf("Would import GPG key: %s\n", u)
		}
	}
	for _, r := range wouldAddRepos {
		fmt.Printf("Would add: %s\n", r)
	}
	if len(wouldInstall) > 0 {
		fmt.Printf("Would run dnf/yum makecache\n")
		for _, p := range wouldInstall {
			fmt.Printf("Would install: %s\n", p)
		}
	}
	for _, r := range wouldInstallRPM {
		fmt.Printf("Would install RPM: %s\n", r)
	}
	if len(wouldAddKeys) == 0 && len(wouldAddRepos) == 0 && len(wouldInstall) == 0 && len(wouldInstallRPM) == 0 {
		fmt.Println("Nothing to do; all entries already present.")
	}
	return nil
}
