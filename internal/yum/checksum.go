package yum

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"
)

// verifyChecksum checks that data matches the expected checksum.
// algo must be "sha256" or "sha512". expected is a lowercase hex string.
// Returns nil if no checksum is specified (algo == "").
func verifyChecksum(data []byte, algo, expected string) error {
	if algo == "" {
		return nil
	}

	var computed string
	switch algo {
	case "sha256":
		sum := sha256.Sum256(data)
		computed = hex.EncodeToString(sum[:])
	case "sha512":
		sum := sha512.Sum512(data)
		computed = hex.EncodeToString(sum[:])
	default:
		return fmt.Errorf("unsupported checksum algorithm %q; use sha256 or sha512", algo)
	}

	if computed != strings.ToLower(expected) {
		return fmt.Errorf("checksum mismatch (%s): expected %s, got %s", algo, strings.ToLower(expected), computed)
	}
	return nil
}
