// Package vapid reads the VAPID key that signs notify's Web Pushes.
package vapid

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// Key is a VAPID key pair, each half base64url-encoded without padding, as Web Push wants it.
type Key struct {
	Private string
	Public  string
}

// Parse derives the key pair from stored, the private key in base64url without padding; the
// whitespace around it, such as a file's last line end, doesn't count.
func Parse(stored string) (Key, error) {
	private := strings.TrimSpace(stored)

	raw, err := base64.RawURLEncoding.DecodeString(private)
	if err != nil {
		return Key{}, fmt.Errorf("the VAPID key isn't base64url without padding: %w", err)
	}

	key, err := ecdh.P256().NewPrivateKey(raw)
	if err != nil {
		return Key{}, fmt.Errorf("the VAPID key isn't a P-256 private key: %w", err)
	}

	public := base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())

	return Key{Private: private, Public: public}, nil
}

// Generate makes a new private key in base64url without padding, for the user to store as the
// secret.
func Generate() (string, error) {
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("make a VAPID key: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(key.Bytes()), nil
}
