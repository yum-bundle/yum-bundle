package yum

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// keyDownloadHTTPClient is used for GPG key downloads.
var keyDownloadHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("redirect to non-HTTPS URL rejected for security: %s", req.URL)
		}
		return nil
	},
}

// validateKeyURL ensures the key URL uses https://.
func validateKeyURL(keyURL string) error {
	u, err := url.Parse(keyURL)
	if err != nil {
		return fmt.Errorf("invalid key URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		return fmt.Errorf("key URL must use https://, not http:// (rejected for security)")
	case "file":
		return fmt.Errorf("file:// key URLs are not allowed (rejected for security)")
	case "":
		return fmt.Errorf("invalid key URL: missing scheme (use https://)")
	default:
		return fmt.Errorf("key URL scheme %q not allowed; use https://", u.Scheme)
	}
}

// KeyPathForURL returns the path that ImportGPGKey would store the key for the given URL.
func (m *YumManager) KeyPathForURL(keyURL string) string {
	hash := sha256.Sum256([]byte(keyURL))
	filename := fmt.Sprintf("%s%x.key", m.KeyPrefix, hash[:8])
	return filepath.Join(m.KeyDir, filename)
}

// ImportGPGKey downloads a GPG key from an HTTPS URL, saves it to KeyDir, and
// imports it with rpm --import. Idempotent: if the key file already exists the
// download is skipped but rpm --import is still run (rpm --import is idempotent).
// Returns the path of the saved key file.
func (m *YumManager) ImportGPGKey(keyURL string) (string, error) {
	if err := validateKeyURL(keyURL); err != nil {
		return "", err
	}
	fmt.Printf("Importing GPG key from: %s\n", keyURL)

	keyPath := m.KeyPathForURL(keyURL)

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Need to download
		if err := os.MkdirAll(m.KeyDir, 0755); err != nil {
			return "", fmt.Errorf("create key directory: %w", err)
		}

		resp, err := m.HTTPGet(keyURL)
		if err != nil {
			return "", fmt.Errorf("download GPG key: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("download GPG key: HTTP %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read GPG key data: %w", err)
		}

		if err := os.WriteFile(keyPath, data, 0644); err != nil { //nolint:gosec // GPG key files in /etc/pki/rpm-gpg/ are conventionally world-readable
			return "", fmt.Errorf("write GPG key: %w", err)
		}
		fmt.Printf("✓ GPG key saved to: %s\n", keyPath)
	} else {
		fmt.Printf("✓ GPG key already downloaded: %s\n", keyPath)
	}

	// Always run rpm --import; it is idempotent.
	if err := m.runCommand("rpm", "--import", keyPath); err != nil {
		return keyPath, wrapCommandError(err, "rpm --import", keyPath)
	}

	fmt.Printf("✓ GPG key imported: %s\n", keyPath)
	return keyPath, nil
}

// RemoveGPGKey removes a saved GPG key file.
func (m *YumManager) RemoveGPGKey(keyPath string) error {
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove GPG key: %w", err)
	}
	return nil
}
