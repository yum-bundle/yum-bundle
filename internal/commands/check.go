package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

var checkJSON bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if packages and repositories from Yumfile are present",
	Long: `Read the Yumfile and check if all specified packages are present on the system.
Exit 0 only if all entries are satisfied; non-zero otherwise.
Use --json for machine-friendly output.`,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Output result as JSON (ok, missing list)")
}

// CheckResult is the structure for --json output.
type CheckResult struct {
	OK      bool     `json:"ok"`
	Missing []string `json:"missing"`
}

// doCheck runs the check and returns ok, missing list, and entries.
func doCheck(yumfilePath string) (ok bool, missing []string, entries []yumfile.Entry, err error) {
	entries, err = yumfile.Parse(yumfilePath)
	if err != nil {
		return false, nil, nil, fmt.Errorf("parse Yumfile: %w", err)
	}

	for _, entry := range entries {
		switch entry.Type {
		case yumfile.EntryTypeYum:
			pkgName := yumfile.ExtractPkgName(entry.Value)
			installed, err := mgr.IsPackageInstalled(pkgName)
			if err != nil || !installed {
				missing = append(missing, pkgName)
			}

		case yumfile.EntryTypeKey:
			keyPath := mgr.KeyPathForURL(entry.Value)
			if _, statErr := os.Stat(keyPath); statErr != nil {
				if errors.Is(statErr, os.ErrNotExist) {
					missing = append(missing, "key "+entry.Value)
				} else {
					return false, nil, nil, fmt.Errorf("checking key %s: %w", entry.Value, statErr)
				}
			}

		case yumfile.EntryTypeGroup:
			installed, err := mgr.IsGroupInstalled(entry.Value)
			if err != nil || !installed {
				missing = append(missing, "group "+entry.Value)
			}
		}
	}

	sort.Strings(missing)
	return len(missing) == 0, missing, entries, nil
}

func runCheck(_ *cobra.Command, _ []string) error {
	ok, missing, entries, err := doCheck(yumfilePath)
	if err != nil {
		return err
	}

	if checkJSON {
		out := CheckResult{OK: ok, Missing: missing}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%d entries missing", len(missing))
		}
		return nil
	}

	fmt.Printf("Checking Yumfile: %s\n", yumfilePath)
	fmt.Printf("Checking %d entries...\n\n", len(entries))
	if ok {
		fmt.Println("✓ All entries present.")
		return nil
	}
	return fmt.Errorf("%d missing: %v", len(missing), missing)
}
