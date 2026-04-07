package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/yum"
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
7. Install all specified package groups
8. Install RPMs from URLs if specified`,
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
		return fmt.Errorf("parse Yumfile: %w", err)
	}
	fmt.Printf("Found %d entries in Yumfile\n", len(entries))
	if installDryRun {
		return runInstallDryRun(entries)
	}
	return doInstall(entries)
}

// repoEntry groups a repo/baseurl/copr/epel/module entry with its associated
// GPG key path (non-empty only for baseurl entries that are immediately preceded
// by a key directive in the Yumfile).
//
// Ordering contract: a key directive must appear directly before the baseurl
// it protects. This is enforced by convention in the Yumfile format. If a key
// entry is not immediately followed by a baseurl, a warning is printed during
// categorization and the key is still imported but not wired to any repo.
type repoEntry struct {
	entry      yumfile.Entry
	gpgKeyPath string // non-empty only for baseurl entries with a preceding key
}

// categorizedEntries holds entries grouped by directive type, produced by a
// single categorization pass over the raw entry list.
type categorizedEntries struct {
	keys     []yumfile.Entry
	repos    []repoEntry // repo, baseurl, copr, epel, module (in original order)
	packages []string    // yum package specs (in original order)
	groups   []yumfile.Entry
	rpms     []yumfile.Entry
	excludes []string
}

// categorize performs a single pass over entries and returns grouped slices.
// The key→baseurl pairing is resolved here by index: if entry[i] is a key and
// entry[i+1] is a baseurl, the key URL is stored alongside the baseurl entry so
// it can be looked up after the keys are imported. If a key is not immediately
// followed by a baseurl, a warning is emitted.
func categorize(entries []yumfile.Entry) *categorizedEntries {
	c := &categorizedEntries{}
	for i, entry := range entries {
		switch entry.Type {
		case yumfile.EntryTypeKey:
			c.keys = append(c.keys, entry)
			// Validate ordering: a key should be immediately followed by a baseurl.
			nextIsBaseurl := i+1 < len(entries) && entries[i+1].Type == yumfile.EntryTypeBaseurl
			if !nextIsBaseurl {
				fmt.Fprintf(os.Stderr, "warning: key directive on line %d is not immediately followed by a baseurl; it will be imported but not associated with any repo\n", entry.LineNum)
			}
		case yumfile.EntryTypeBaseurl:
			// Pair with the immediately preceding key entry, if any.
			var keyURL string
			if i > 0 && entries[i-1].Type == yumfile.EntryTypeKey {
				keyURL = entries[i-1].Value
			}
			c.repos = append(c.repos, repoEntry{entry: entry, gpgKeyPath: keyURL})
		case yumfile.EntryTypeRepo, yumfile.EntryTypeCopr, yumfile.EntryTypeEPEL, yumfile.EntryTypeModule:
			c.repos = append(c.repos, repoEntry{entry: entry})
		case yumfile.EntryTypeYum:
			c.packages = append(c.packages, entry.Value)
		case yumfile.EntryTypeGroup:
			c.groups = append(c.groups, entry)
		case yumfile.EntryTypeRPM:
			c.rpms = append(c.rpms, entry)
		case yumfile.EntryTypeExclude:
			c.excludes = append(c.excludes, entry.Value)
		}
	}
	return c
}

// doInstall performs the non-dry-run install workflow for the given entries.
// Callers are responsible for root checks and dry-run branching.
func doInstall(entries []yumfile.Entry) error {
	state, err := mgr.LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	stateFilePath := mgr.StatePath()
	defer func() {
		if saveErr := mgr.SaveState(state); saveErr != nil {
			fmt.Fprintf(os.Stderr, "warning: save state to %s: %v\n", stateFilePath, saveErr)
		}
	}()

	// Single categorization pass — avoids repeated scans of entries.
	cat := categorize(entries)

	// 1. Import GPG keys; build a map from key URL → installed key path so that
	// baseurl entries can resolve their associated key in step 2.
	importedKeyPaths := make(map[string]string) // key URL -> key path on disk
	for _, entry := range cat.keys {
		keyPath, err := mgr.ImportGPGKey(entry.Value, entry.ChecksumAlgo, entry.Checksum)
		if err != nil {
			return fmt.Errorf("import GPG key: %w", err)
		}
		state.AddKey(keyPath)
		importedKeyPaths[entry.Value] = keyPath
	}

	// 2. Add repositories (repo files, baseurls, COPRs, EPEL, modules).
	var reposAdded bool
	for _, re := range cat.repos {
		entry := re.entry
		switch entry.Type {
		case yumfile.EntryTypeRepo:
			repoPath, err := mgr.AddRepoFile(entry.Value, entry.ChecksumAlgo, entry.Checksum)
			if err != nil {
				return fmt.Errorf("add repo: %w", err)
			}
			state.AddRepo(repoPath)
			reposAdded = true

		case yumfile.EntryTypeBaseurl:
			opts := &yum.RepoFileOptions{}
			if re.gpgKeyPath != "" {
				opts.GPGKeyPath = importedKeyPaths[re.gpgKeyPath]
			}
			repoPath, err := mgr.AddBaseurlRepo(entry.Value, opts)
			if err != nil {
				return fmt.Errorf("add baseurl repo: %w", err)
			}
			state.AddRepo(repoPath)
			reposAdded = true

		case yumfile.EntryTypeCopr:
			if err := mgr.EnableCOPR(entry.Value); err != nil {
				return fmt.Errorf("enable COPR %s: %w", entry.Value, err)
			}
			coprPath := mgr.CoprRepoPathFor(entry.Value)
			state.AddRepo(coprPath)
			reposAdded = true

		case yumfile.EntryTypeEPEL:
			if err := mgr.EnableEPEL(); err != nil {
				return fmt.Errorf("enable EPEL: %w", err)
			}
			reposAdded = true

		case yumfile.EntryTypeModule:
			if err := mgr.EnableModule(entry.Value); err != nil {
				return fmt.Errorf("enable module %s: %w", entry.Value, err)
			}
		}
	}

	if !noUpdate {
		if err := mgr.MakecacheOrUpdate(); err != nil {
			return fmt.Errorf("update package metadata: %w", err)
		}
	} else if reposAdded {
		fmt.Println("Warning: Repositories were added; run without --no-update to fetch package metadata.")
	}

	// 3. Install packages.
	packagesToInstall := cat.packages
	if installLocked {
		specs, err := ReadLockFile()
		if err != nil {
			return err
		}
		packagesToInstall = specs
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
			if err := mgr.InstallPackage(pkg, cat.excludes); err != nil {
				return fmt.Errorf("install package %s: %w", pkg, err)
			}
			state.AddPackage(pkgName)
		}
		fmt.Println("✓ All packages installed successfully")
	} else {
		fmt.Println("No packages to install")
	}

	// 4. Install package groups.
	for _, entry := range cat.groups {
		installed, err := mgr.IsGroupInstalled(entry.Value)
		if err != nil {
			fmt.Printf("Warning: could not check if group %s is installed: %v\n", entry.Value, err)
		}
		if installed {
			fmt.Printf("✓ Group %s is already installed\n", entry.Value)
			state.AddGroup(entry.Value)
			continue
		}
		if err := mgr.InstallGroup(entry.Value, cat.excludes); err != nil {
			return fmt.Errorf("install group %s: %w", entry.Value, err)
		}
		state.AddGroup(entry.Value)
	}

	// 5. Install RPMs from URLs.
	for _, entry := range cat.rpms {
		if err := mgr.InstallRPMFromURL(entry.Value, entry.ChecksumAlgo, entry.Checksum); err != nil {
			return fmt.Errorf("install RPM from URL %s: %w", entry.Value, err)
		}
	}

	if installLock {
		if len(packagesToInstall) > 0 {
			if err := writeLockFileFromPackages(packagesToInstall); err != nil {
				return fmt.Errorf("write lock file: %w", err)
			}
		} else {
			fmt.Fprintln(os.Stderr, "warning: --lock specified but no packages to lock (Yumfile may only contain groups or RPMs); no lock file written")
		}
	}

	return nil
}

func writeLockFileFromPackages(packages []string) error {
	locked, skipped := resolveInstalledVersions(packages)
	for _, name := range skipped {
		fmt.Fprintf(os.Stderr, "warning: %s not installed or version unavailable; skipping in lock file\n", name)
	}
	if len(locked) == 0 {
		return nil
	}
	return writeLockFileEntries(locked)
}

func runInstallDryRun(entries []yumfile.Entry) error {
	repos, err := mgr.ListCustomRepos()
	if err != nil {
		return fmt.Errorf("list repos: %w", err)
	}
	repoSet := make(map[string]bool)
	for _, r := range repos {
		repoSet[r.YumfileLine] = true
	}

	var wouldAddKeys, wouldAddRepos, wouldInstall, wouldInstallGroups, wouldInstallRPM []string

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
		case yumfile.EntryTypeGroup:
			installed, err := mgr.IsGroupInstalled(entry.Value)
			if err != nil || !installed {
				wouldInstallGroups = append(wouldInstallGroups, entry.Value)
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
	if len(wouldInstall) > 0 || len(wouldInstallGroups) > 0 {
		fmt.Printf("Would run dnf/yum makecache\n")
		for _, p := range wouldInstall {
			fmt.Printf("Would install: %s\n", p)
		}
	}
	for _, g := range wouldInstallGroups {
		fmt.Printf("Would install group: %s\n", g)
	}
	for _, r := range wouldInstallRPM {
		fmt.Printf("Would install RPM: %s\n", r)
	}
	if len(wouldAddKeys) == 0 && len(wouldAddRepos) == 0 && len(wouldInstall) == 0 && len(wouldInstallGroups) == 0 && len(wouldInstallRPM) == 0 {
		fmt.Println("Nothing to do; all entries already present.")
	}
	return nil
}
