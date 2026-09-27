package secrets

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type Secret struct {
	Name   string
	EnvVar string
}

// known lists every secret managed by Ensure/Health. Adding a secret is a
// one-line change here plus the matching *_FILE support in config.
var known = []Secret{
	{Name: "postgres_password", EnvVar: "POSTGRES_PASSWORD"},
	{Name: "jwt_secret", EnvVar: "PECULIUM_JWT_SECRET"},
}

func Known() []Secret {
	return known
}

// Ensure materializes every known secret under dir: an existing non-empty
// file is kept untouched, otherwise the env override is written, otherwise a
// random value is generated. It returns the names of the secrets that were
// generated (never their values).
func Ensure(dir string) (generated []string, err error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create secrets dir: %w", err)
	}
	for _, s := range known {
		path := filepath.Join(dir, s.Name)
		if existingNonEmpty(path) {
			continue
		}
		var value string
		if v := os.Getenv(s.EnvVar); v != "" {
			value = v
		} else {
			value, err = randomHex(32)
			if err != nil {
				return nil, fmt.Errorf("generate %s: %w", s.Name, err)
			}
			generated = append(generated, s.Name)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", s.Name, err)
		}
	}
	return generated, nil
}

// Health returns nil only when every known secret file exists and is non-empty.
func Health(dir string) error {
	for _, s := range known {
		path := filepath.Join(dir, s.Name)
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", s.Name, err)
		}
		if len(bytes.TrimSpace(b)) == 0 {
			return fmt.Errorf("secret %s is empty", s.Name)
		}
	}
	return nil
}

func existingNonEmpty(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(b)) > 0
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
