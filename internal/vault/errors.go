package vault

import "errors"

// ErrSecretNotFound is returned when a secret is not found in the keychain
var ErrSecretNotFound = errors.New("secret not found in keychain")
