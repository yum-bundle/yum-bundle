package yum

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// validateRepoURL ensures the URL uses https://. Rejects http://, file://, and other schemes.
func validateRepoURL(repoURL string) error {
	u, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		return fmt.Errorf("URL must use https://, not http:// (rejected for security)")
	case "file":
		return fmt.Errorf("file:// URLs are not allowed (rejected for security)")
	case "":
		return fmt.Errorf("invalid URL: missing scheme (use https://)")
	default:
		return fmt.Errorf("URL scheme %q not allowed; use https://", u.Scheme)
	}
}

// repoFileHTTPClient is used for .repo file downloads.
var repoFileHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("redirect to non-HTTPS URL rejected: %s", req.URL)
		}
		return nil
	},
}

// AddRepoFile downloads a .repo file from the given HTTPS URL and writes it to
// /etc/yum.repos.d/ under a yum-bundle-managed filename. Returns the path of
// the written file. Idempotent: if the file already exists it is not re-downloaded.
func (m *YumManager) AddRepoFile(repoURL string) (string, error) {
	if err := validateRepoURL(repoURL); err != nil {
		return "", err
	}
	fmt.Printf("Adding repo file from: %s\n", repoURL)

	hash := sha256.Sum256([]byte(repoURL))
	filename := fmt.Sprintf("%s%x.repo", m.ReposPrefix, hash[:8])
	destPath := filepath.Join(m.ReposDir, filename)

	if _, err := os.Stat(destPath); err == nil {
		fmt.Printf("✓ Repo file already present: %s\n", destPath)
		return destPath, nil
	}

	if err := os.MkdirAll(m.ReposDir, 0755); err != nil {
		return "", fmt.Errorf("create repos directory: %w", err)
	}

	resp, err := m.HTTPGet(repoURL)
	if err != nil {
		return "", fmt.Errorf("download repo file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download repo file: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read repo file: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("write repo file: %w", err)
	}

	fmt.Printf("✓ Repo file saved to: %s\n", destPath)
	return destPath, nil
}

// RepoFileOptions configures an inline .repo file created by AddBaseurlRepo.
type RepoFileOptions struct {
	// Name is the human-readable repo name. If empty, a name is auto-generated.
	Name string
	// GPGKeyPath is an optional path to a GPG key to set in gpgkey=.
	GPGKeyPath string
	// GPGCheck enables or disables gpgcheck (default: enabled when GPGKeyPath is set).
	GPGCheck *bool
	// Enabled controls the enabled= field (default: true).
	Enabled *bool
}

// AddBaseurlRepo creates a minimal .repo file in /etc/yum.repos.d/ from a baseurl.
// If opts is nil, defaults are used (gpgcheck=0, enabled=1).
// Returns the path of the written file.
func (m *YumManager) AddBaseurlRepo(baseurlValue string, opts *RepoFileOptions) (string, error) {
	if err := validateRepoURL(baseurlValue); err != nil {
		return "", fmt.Errorf("baseurl: %w", err)
	}
	fmt.Printf("Adding baseurl repo: %s\n", baseurlValue)

	if opts == nil {
		opts = &RepoFileOptions{}
	}

	hash := sha256.Sum256([]byte(baseurlValue))
	repoID := fmt.Sprintf("%s%x", m.ReposPrefix, hash[:8])
	filename := repoID + ".repo"
	destPath := filepath.Join(m.ReposDir, filename)

	if _, err := os.Stat(destPath); err == nil {
		fmt.Printf("✓ Repo already configured: %s\n", destPath)
		return destPath, nil
	}

	if err := os.MkdirAll(m.ReposDir, 0755); err != nil {
		return "", fmt.Errorf("create repos directory: %w", err)
	}

	name := opts.Name
	if name == "" {
		name = "yum-bundle managed repo (" + repoID + ")"
	}

	enabled := true
	if opts.Enabled != nil {
		enabled = *opts.Enabled
	}
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	gpgcheck := 0
	if opts.GPGKeyPath != "" {
		gpgcheck = 1
	}
	if opts.GPGCheck != nil {
		if *opts.GPGCheck {
			gpgcheck = 1
		} else {
			gpgcheck = 0
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s]\n", repoID)
	fmt.Fprintf(&sb, "name=%s\n", name)
	fmt.Fprintf(&sb, "baseurl=%s\n", baseurlValue)
	fmt.Fprintf(&sb, "enabled=%d\n", enabledInt)
	fmt.Fprintf(&sb, "gpgcheck=%d\n", gpgcheck)
	if opts.GPGKeyPath != "" {
		fmt.Fprintf(&sb, "gpgkey=file://%s\n", opts.GPGKeyPath)
	}

	if err := os.WriteFile(destPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("write repo file: %w", err)
	}

	fmt.Printf("✓ Repo file created: %s\n", destPath)
	return destPath, nil
}

// RemoveRepoFile removes a managed .repo file.
func (m *YumManager) RemoveRepoFile(repoPath string) error {
	if err := os.Remove(repoPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove repo file: %w", err)
	}
	return nil
}

// RepoEntry represents a custom repository entry for the dump command.
type RepoEntry struct {
	// YumfileLine is the Yumfile directive line to reproduce this repo.
	YumfileLine string
	// Type is "repo" or "baseurl".
	Type string
}

// defaultRepoIDs are repo IDs that ship with common RPM distros and should be
// excluded when generating a Yumfile via dump.
var defaultRepoIDs = []string{
	"fedora", "fedora-updates", "fedora-modular", "fedora-updates-modular",
	"updates", "updates-testing",
	"baseos", "appstream", "extras", "powertools", "crb",
	"rhel-baseos", "rhel-appstream",
	"ubi-8-baseos", "ubi-8-appstream", "ubi-9-baseos", "ubi-9-appstream",
	"centos-baseos", "centos-appstream", "centos-extras",
	"rocky-baseos", "rocky-appstream", "rocky-extras",
	"almalinux-baseos", "almalinux-appstream",
	"epel", "epel-next",
}

// repoIDLine matches a [section-header] in an INI .repo file.
var repoIDLine = regexp.MustCompile(`^\[([^\]]+)\]`)

// isDefaultRepoID returns true when the given repo ID is a known distro default.
func isDefaultRepoID(id string) bool {
	lower := strings.ToLower(id)
	for _, d := range defaultRepoIDs {
		if lower == d || strings.HasPrefix(lower, d+"-") {
			return true
		}
	}
	return false
}

// ListCustomRepos reads /etc/yum.repos.d/ and returns Yumfile-style lines for
// non-default, non-yum-bundle-managed repos.
func (m *YumManager) ListCustomRepos() ([]RepoEntry, error) {
	entries, err := os.ReadDir(m.ReposDir)
	if err != nil {
		return nil, nil // directory missing is acceptable
	}

	var result []RepoEntry
	seen := make(map[string]bool)

	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".repo") {
			continue
		}
		path := filepath.Join(m.ReposDir, de.Name())
		repoEntries, err := readRepoFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		for _, e := range repoEntries {
			if !seen[e.YumfileLine] {
				seen[e.YumfileLine] = true
				result = append(result, e)
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].YumfileLine < result[j].YumfileLine
	})
	return result, nil
}

// readRepoFile parses a .repo file and returns Yumfile entries for custom repos.
func readRepoFile(path string) ([]RepoEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []RepoEntry
	var currentID string
	var baseurlVal string

	flush := func() {
		if currentID != "" && baseurlVal != "" && !isDefaultRepoID(currentID) {
			entries = append(entries, RepoEntry{
				YumfileLine: "baseurl " + baseurlVal,
				Type:        "baseurl",
			})
		}
		currentID = ""
		baseurlVal = ""
	}

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if m := repoIDLine.FindStringSubmatch(line); len(m) == 2 {
			flush()
			currentID = m[1]
			continue
		}
		if after, ok := strings.CutPrefix(line, "baseurl="); ok {
			baseurlVal = strings.TrimSpace(after)
		}
	}
	flush()

	return entries, sc.Err()
}

// RepoFilePathForURL returns the path that AddRepoFile would use for the given URL.
func (m *YumManager) RepoFilePathForURL(repoURL string) string {
	hash := sha256.Sum256([]byte(repoURL))
	filename := fmt.Sprintf("%s%x.repo", m.ReposPrefix, hash[:8])
	return filepath.Join(m.ReposDir, filename)
}
