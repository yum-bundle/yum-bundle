package yum_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

var errNotFound = errors.New("not found")

func TestInstallRPMFromURL_CallsDNF(t *testing.T) {
	m, mock := dnfManager(t)
	// Make IsPackageInstalled return false (rpm -q exits 1)
	// rpmNameFromURL extracts "epel-release-latest" from this convenience URL
	mock.SetError(errNotFound, "rpm", "-q", "--quiet", "epel-release-latest")
	err := m.InstallRPMFromURL("https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertCalled(t, "dnf", "install", "-y",
		"https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm")
}

func TestInstallRPMFromURL_RejectsHTTP(t *testing.T) {
	m, _ := dnfManager(t)
	err := m.InstallRPMFromURL("http://example.com/pkg-1.0.rpm", "", "")
	if err == nil {
		t.Error("expected error for http:// URL")
	}
}

func TestInstallRPMFromURL_RequiresDotRPM(t *testing.T) {
	m, _ := dnfManager(t)
	err := m.InstallRPMFromURL("https://example.com/not-an-rpm", "", "")
	if err == nil {
		t.Error("expected error for non-.rpm URL")
	}
}

func TestInstallRPMFromURL_AlreadyInstalled(t *testing.T) {
	m, mock := dnfManager(t)
	// rpm -q exits 0 → package installed
	// (mock returns nil error by default for all commands)
	err := m.InstallRPMFromURL("https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mock.AssertNotCalled(t, "dnf", "install", "-y",
		"https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm")
}

func TestInstallRPMFromURL_WrongChecksumReturnsError(t *testing.T) {
	m, mock := dnfManager(t)
	mock.SetError(errNotFound, "rpm", "-q", "--quiet", "epel-release-latest")
	m.HTTPGet = func(_ string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("FAKERPMBYTES")),
		}, nil
	}
	err := m.InstallRPMFromURL(
		"https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm",
		"sha256",
		"0000000000000000000000000000000000000000000000000000000000000000",
	)
	if err == nil {
		t.Error("expected error for wrong checksum")
	}
}
