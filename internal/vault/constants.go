// Package vault provides secure secret storage using system keyrings and environment variables.
package vault

const (
	// KeychainServiceName is the identifier for our application in the system keychain
	KeychainServiceName = "vanish-cli"

	// TailscaleAuthKeyItem is the keyring key for Tailscale auth key
	TailscaleAuthKeyItem = "TS_AUTHKEY"

	// NASCredsItem is the keyring key for NAS credentials (username:password format)
	NASCredsItem = "NAS_CREDS"
)
