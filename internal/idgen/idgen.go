// Package idgen creates opaque, non-sequential identifiers.
package idgen

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// New returns prefix plus 128 bits of randomness, hex encoded.
func New(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("idgen: %w", err))
	}
	return prefix + hex.EncodeToString(b[:])
}

// Stable returns a deterministic opaque id for a natural key.
func Stable(prefix, material string) string {
	sum := sha256.Sum256([]byte(material))
	return prefix + hex.EncodeToString(sum[:12])
}

// Hash returns a hex sha256 of material.
func Hash(material string) string {
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}
