package yum

import (
	"testing"
)

func TestKeyHTTPClientHasTransport(t *testing.T) {
	if keyHTTPClient.Transport == nil {
		t.Error("keyHTTPClient.Transport must not be nil (needed for proxy support)")
	}
}
