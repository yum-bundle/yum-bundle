package yum_test

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/testutil"
	"github.com/yum-bundle/yum-bundle/internal/yum"
)

func keyManager(t *testing.T, keyContent string) *yum.YumManager {
	t.Helper()
	m := testManager(t)
	m.HTTPGet = func(_ string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(keyContent)),
		}, nil
	}
	return m
}

func TestImportGPGKey_DownloadsAndImports(t *testing.T) {
	mock := testutil.NewMockExecutor()
	m := keyManager(t, "-----BEGIN PGP PUBLIC KEY BLOCK-----\nFAKEKEY\n-----END PGP PUBLIC KEY BLOCK-----\n")
	m.Executor = mock

	keyPath, err := m.ImportGPGKey("https://example.com/RPM-GPG-KEY-example", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(keyPath); statErr != nil {
		t.Errorf("key file not created: %v", statErr)
	}

	mock.AssertCalled(t, "rpm", "--import", keyPath)
}

func TestImportGPGKey_Idempotent(t *testing.T) {
	mock := testutil.NewMockExecutor()
	m := keyManager(t, "FAKEKEYDATA")
	m.Executor = mock

	url := "https://example.com/RPM-GPG-KEY-example"
	path1, err := m.ImportGPGKey(url, "", "")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	path2, err := m.ImportGPGKey(url, "", "")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if path1 != path2 {
		t.Errorf("expected same path, got %s vs %s", path1, path2)
	}
	// rpm --import should be called twice (idempotent on the rpm side)
	count := 0
	for _, call := range mock.Calls {
		if len(call) >= 2 && call[0] == "rpm" && call[1] == "--import" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected rpm --import called twice, got %d", count)
	}
}

func TestImportGPGKey_RejectsHTTP(t *testing.T) {
	m := testManager(t)
	_, err := m.ImportGPGKey("http://example.com/key.pub", "", "")
	if err == nil {
		t.Error("expected error for http:// URL")
	}
}

func TestImportGPGKey_RejectsFileURL(t *testing.T) {
	m := testManager(t)
	_, err := m.ImportGPGKey("file:///etc/pki/rpm-gpg/key", "", "")
	if err == nil {
		t.Error("expected error for file:// URL")
	}
}

func TestKeyPathForURL_Deterministic(t *testing.T) {
	m := testManager(t)
	url := "https://example.com/key.pub"
	p1 := m.KeyPathForURL(url)
	p2 := m.KeyPathForURL(url)
	if p1 != p2 {
		t.Errorf("KeyPathForURL not deterministic: %s vs %s", p1, p2)
	}
}

func TestKeyPathForURL_PrefixedWithYumBundle(t *testing.T) {
	m := testManager(t)
	path := m.KeyPathForURL("https://example.com/key.pub")
	base := path[strings.LastIndex(path, "/")+1:]
	if !strings.HasPrefix(base, "yum-bundle-") {
		t.Errorf("expected filename to start with yum-bundle-, got %q", base)
	}
}

func TestImportGPGKey_CreatesKeyDir(t *testing.T) {
	m := keyManager(t, "FAKEKEY")
	// Ensure key dir doesn't exist yet
	os.RemoveAll(m.KeyDir)

	_, err := m.ImportGPGKey("https://example.com/key.pub", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(m.KeyDir); statErr != nil {
		t.Errorf("key directory not created: %v", statErr)
	}
}

func TestImportGPGKey_WrongChecksumReturnsError(t *testing.T) {
	m := keyManager(t, "FAKEKEYDATA")

	_, err := m.ImportGPGKey("https://example.com/key.pub", "sha256", "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for wrong checksum")
	}
}
