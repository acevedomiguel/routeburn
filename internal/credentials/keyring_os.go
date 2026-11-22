package credentials

import (
	"errors"

	keyring "github.com/zalando/go-keyring"
)

// osKeyring adapts github.com/zalando/go-keyring to the Keyring interface.
type osKeyring struct{}

// NewOSKeyring returns a keyring backed by the operating system secret store.
func NewOSKeyring() Keyring { return osKeyring{} }

func (osKeyring) Get(service, user string) (string, error) {
	v, err := keyring.Get(service, user)
	if err != nil {
		return "", mapKeyringError(err)
	}
	return v, nil
}

func (osKeyring) Set(service, user, secret string) error {
	if err := keyring.Set(service, user, secret); err != nil {
		return mapKeyringError(err)
	}
	return nil
}

func (osKeyring) Delete(service, user string) error {
	if err := keyring.Delete(service, user); err != nil {
		return mapKeyringError(err)
	}
	return nil
}

// mapKeyringError normalises the library's sentinels into our own so callers
// can branch on a stable set of errors regardless of platform.
func mapKeyringError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrKeyringUnavailable
	}
	return errors.Join(ErrKeyringUnavailable, err)
}
