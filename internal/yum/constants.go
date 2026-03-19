package yum

import (
	"net/http"
	"time"
)

const (
	// ReposDir is where yum/dnf .repo files are stored.
	ReposDir = "/etc/yum.repos.d"
	// ReposPrefix is the prefix for yum-bundle managed .repo files.
	ReposPrefix = "yum-bundle-"

	// KeyDir is the directory where yum-bundle stores downloaded GPG keys.
	KeyDir = "/etc/pki/rpm-gpg"
	// KeyPrefix is the prefix for yum-bundle managed key files.
	KeyPrefix = "yum-bundle-"

	// StateDir is the directory where yum-bundle stores its state.
	StateDir = "/var/lib/yum-bundle"
	// StateFile is the filename for the state file.
	StateFile = "state.json"
)

// keyHTTPClient is used for key/repo downloads; has timeout and enforces HTTPS on redirects.
var keyHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return &redirectError{url: req.URL.String()}
		}
		return nil
	},
}

type redirectError struct {
	url string
}

func (e *redirectError) Error() string {
	return "redirect to non-HTTPS URL rejected for security: " + e.url
}
