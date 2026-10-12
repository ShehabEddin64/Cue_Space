// Package secrets keeps API keys in the operating system's credential store
// (the macOS Keychain), never in plain files.
package secrets

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "space.cue.app"

// Store reads and writes secrets by name. Tests can swap in a map.
type Store interface {
	Get(name string) (string, error)
	Set(name, value string) error
	Delete(name string) error
}

type Keychain struct{}

// Get returns "" with no error when the secret is not set.
func (Keychain) Get(name string) (string, error) {
	v, err := keyring.Get(service, name)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return v, err
}

func (Keychain) Set(name, value string) error {
	return keyring.Set(service, name, value)
}

func (Keychain) Delete(name string) error {
	err := keyring.Delete(service, name)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// APIKeyName is the Keychain account used for a provider's key.
func APIKeyName(provider string) string {
	return "api-key:" + provider
}
