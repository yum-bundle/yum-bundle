package yum

import (
	"os"
	"testing"
)

func TestProxyFromEnv_NoEnv(t *testing.T) {
	for _, key := range []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY"} {
		os.Unsetenv(key)
	}
	if got := proxyFromEnv(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestProxyFromEnv_HTTPSProxy_Upper(t *testing.T) {
	for _, key := range []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY"} {
		os.Unsetenv(key)
	}
	os.Setenv("HTTPS_PROXY", "http://proxy.example.com:3128")
	defer os.Unsetenv("HTTPS_PROXY")

	if got := proxyFromEnv(); got != "http://proxy.example.com:3128" {
		t.Errorf("expected http://proxy.example.com:3128, got %q", got)
	}
}

func TestProxyFromEnv_HTTPSProxy_Lower(t *testing.T) {
	for _, key := range []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY"} {
		os.Unsetenv(key)
	}
	os.Setenv("https_proxy", "http://proxy.example.com:3128")
	defer os.Unsetenv("https_proxy")

	if got := proxyFromEnv(); got != "http://proxy.example.com:3128" {
		t.Errorf("expected http://proxy.example.com:3128, got %q", got)
	}
}

func TestProxyFromEnv_BothSet_LowerFirst(t *testing.T) {
	for _, key := range []string{"https_proxy", "HTTPS_PROXY", "http_proxy", "HTTP_PROXY"} {
		os.Unsetenv(key)
	}
	// https_proxy (lowercase) is checked before HTTPS_PROXY (uppercase)
	os.Setenv("https_proxy", "http://lower.example.com:3128")
	os.Setenv("HTTPS_PROXY", "http://upper.example.com:3128")
	defer func() {
		os.Unsetenv("https_proxy")
		os.Unsetenv("HTTPS_PROXY")
	}()

	if got := proxyFromEnv(); got != "http://lower.example.com:3128" {
		t.Errorf("expected http://lower.example.com:3128 (first match wins), got %q", got)
	}
}
