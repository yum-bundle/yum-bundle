package yum

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"testing"
)

func TestVerifyChecksum_ValidSHA256(t *testing.T) {
	data := []byte("hello world")
	sum := sha256.Sum256(data)
	expected := hex.EncodeToString(sum[:])

	if err := verifyChecksum(data, "sha256", expected); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyChecksum_ValidSHA512(t *testing.T) {
	data := []byte("hello world")
	sum := sha512.Sum512(data)
	expected := hex.EncodeToString(sum[:])

	if err := verifyChecksum(data, "sha512", expected); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyChecksum_WrongChecksum(t *testing.T) {
	data := []byte("hello world")
	wrong := "0000000000000000000000000000000000000000000000000000000000000000"

	err := verifyChecksum(data, "sha256", wrong)
	if err == nil {
		t.Error("expected error for wrong sha256 checksum")
	}
}

func TestVerifyChecksum_EmptyAlgo(t *testing.T) {
	data := []byte("hello world")

	if err := verifyChecksum(data, "", ""); err != nil {
		t.Errorf("expected nil for empty algo, got: %v", err)
	}
}

func TestVerifyChecksum_InvalidAlgo(t *testing.T) {
	data := []byte("hello world")

	err := verifyChecksum(data, "md5", "d41d8cd98f00b204e9800998ecf8427e")
	if err == nil {
		t.Error("expected error for unsupported algorithm md5")
	}
}
