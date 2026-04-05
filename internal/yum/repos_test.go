package yum_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yum-bundle/yum-bundle/internal/yum"
)

func repoManager(t *testing.T) *yum.YumManager {
	t.Helper()
	m := testManager(t)
	m.HTTPGet = func(_ string) (*http.Response, error) {
		body := "[docker-ce-stable]\nname=Docker CE\nbaseurl=https://download.docker.com/linux/centos/$releasever/$basearch/stable\nenabled=1\ngpgcheck=1\n"
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}
	return m
}

func TestAddRepoFile_Downloads(t *testing.T) {
	m := repoManager(t)
	path, err := m.AddRepoFile("https://download.docker.com/linux/centos/docker-ce.repo", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("repo file not created at %s: %v", path, statErr)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "docker-ce") {
		t.Error("expected repo file content to contain docker-ce")
	}
}

func TestAddRepoFile_Idempotent(t *testing.T) {
	m := repoManager(t)
	path1, err := m.AddRepoFile("https://download.docker.com/linux/centos/docker-ce.repo", "", "")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	path2, err := m.AddRepoFile("https://download.docker.com/linux/centos/docker-ce.repo", "", "")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if path1 != path2 {
		t.Errorf("expected same path on second call, got %s vs %s", path1, path2)
	}
}

func TestAddRepoFile_RejectsHTTP(t *testing.T) {
	m := testManager(t)
	_, err := m.AddRepoFile("http://example.com/my.repo", "", "")
	if err == nil {
		t.Error("expected error for http:// URL")
	}
}

func TestAddRepoFile_WrongChecksumReturnsError(t *testing.T) {
	m := repoManager(t)
	_, err := m.AddRepoFile("https://download.docker.com/linux/centos/docker-ce.repo", "sha256", "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for wrong checksum")
	}
}

func TestAddBaseurlRepo_CreatesRepoFile(t *testing.T) {
	m := testManager(t)
	path, err := m.AddBaseurlRepo("https://packages.example.com/stable/x86_64/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("repo file not created: %v", statErr)
	}
	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "baseurl=https://packages.example.com/stable/x86_64/") {
		t.Errorf("baseurl not in repo file: %s", content)
	}
	if !strings.Contains(content, "enabled=1") {
		t.Errorf("enabled=1 not in repo file: %s", content)
	}
}

func TestAddBaseurlRepo_WithGPGKey(t *testing.T) {
	m := testManager(t)
	keyPath := "/etc/pki/rpm-gpg/yum-bundle-testkey.key"
	path, err := m.AddBaseurlRepo("https://packages.example.com/stable/x86_64/", &yum.RepoFileOptions{
		Name:       "Example Repo",
		GPGKeyPath: keyPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "gpgcheck=1") {
		t.Errorf("gpgcheck=1 not set when key path provided: %s", content)
	}
	if !strings.Contains(content, "gpgkey=file://"+keyPath) {
		t.Errorf("gpgkey not set: %s", content)
	}
}

func TestAddBaseurlRepo_Idempotent(t *testing.T) {
	m := testManager(t)
	p1, err := m.AddBaseurlRepo("https://packages.example.com/stable/x86_64/", nil)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := m.AddBaseurlRepo("https://packages.example.com/stable/x86_64/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Errorf("expected same path, got %s and %s", p1, p2)
	}
}

func TestListCustomRepos(t *testing.T) {
	m := testManager(t)
	repoContent := "[docker-ce-stable]\nname=Docker CE\nbaseurl=https://download.docker.com/linux/centos/$releasever/$basearch/stable\nenabled=1\n"
	if err := os.MkdirAll(m.ReposDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.ReposDir, "docker-ce.repo"), []byte(repoContent), 0644); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	repos, err := m.ListCustomRepos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d: %v", len(repos), repos)
	}
	if !strings.Contains(repos[0].YumfileLine, "baseurl") {
		t.Errorf("expected baseurl line, got %q", repos[0].YumfileLine)
	}
}

func TestListCustomRepos_SkipsDefaultRepos(t *testing.T) {
	m := testManager(t)
	if err := os.MkdirAll(m.ReposDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a default fedora repo
	fedoraRepo := "[fedora]\nname=Fedora\nbaseurl=https://mirrors.fedoraproject.org/...\nenabled=1\n"
	if err := os.WriteFile(filepath.Join(m.ReposDir, "fedora.repo"), []byte(fedoraRepo), 0644); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	repos, err := m.ListCustomRepos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected default repos to be skipped, got %v", repos)
	}
}

func TestListCustomRepos_EmptyDir(t *testing.T) {
	m := testManager(t)
	// Don't create ReposDir at all
	repos, err := m.ListCustomRepos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repos != nil {
		t.Errorf("expected nil for missing repos dir, got %v", repos)
	}
}
