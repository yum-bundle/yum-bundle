package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yum-bundle/yum-bundle/internal/yum"
	"github.com/yum-bundle/yum-bundle/internal/yumfile"
)

var doctorYumfileOnly bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate Yumfile and check environment",
	Long: `Doctor runs Yumfile validation (parse, unknown directives, syntax) and
environment checks (PATH, dnf/yum availability, state file). Use --yumfile-only
to run only Yumfile validation. Exit non-zero if any check fails.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorYumfileOnly, "yumfile-only", false, "Only validate Yumfile; skip environment checks")
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()
	ew := cmd.ErrOrStderr()
	var failed bool

	// Yumfile validation
	entries, err := yumfile.Parse(yumfilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(ew, "Yumfile not found: %s (skipping validation)\n", yumfilePath)
		} else {
			fmt.Fprintf(ew, "✗ Yumfile validation failed: %v\n", err)
			failed = true
		}
	} else {
		fmt.Fprintf(w, "✓ Yumfile valid (%d entries)\n", len(entries))
	}

	if doctorYumfileOnly {
		if failed {
			return fmt.Errorf("Yumfile validation failed")
		}
		return nil
	}

	// Check dnf availability
	hasDNF := false
	if _, err := exec.LookPath("dnf"); err == nil {
		hasDNF = true
		fmt.Fprintln(w, "✓ dnf available")
	} else if _, err := exec.LookPath("yum"); err == nil {
		fmt.Fprintln(w, "✓ yum available (dnf not found)")
	} else {
		fmt.Fprintf(ew, "✗ neither dnf nor yum found on PATH\n")
		failed = true
	}

	// Check dnf-only directives compatibility
	if entries != nil && !hasDNF {
		for _, entry := range entries {
			if entry.Type == yumfile.EntryTypeCopr || entry.Type == yumfile.EntryTypeModule {
				fmt.Fprintf(ew, "✗ line %d: %s %s requires dnf (not available)\n",
					entry.LineNum, entry.Type, entry.Value)
				failed = true
			}
		}
	}

	// Check rpm availability (needed for key import, package checks)
	if _, err := exec.LookPath("rpm"); err != nil {
		fmt.Fprintf(ew, "✗ rpm not found on PATH\n")
		failed = true
	} else {
		fmt.Fprintln(w, "✓ rpm available")
	}

	// Check state file
	if _, err := mgr.LoadState(); err != nil {
		fmt.Fprintf(ew, "✗ state file: %v (path: %s)\n", err, yum.StateDir)
		failed = true
	} else {
		statePath := filepath.Join(yum.StateDir, yum.StateFile)
		if _, err := os.Stat(statePath); err == nil {
			fmt.Fprintln(w, "✓ state file readable")
		} else {
			fmt.Fprintln(w, "✓ state file OK (will be created on first install)")
		}
	}

	if failed {
		return fmt.Errorf("environment check failed")
	}
	return nil
}
