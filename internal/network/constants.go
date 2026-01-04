// Package network provides Tailscale VPN networking functionality.
package network

const (
	// TailscaleEphemeralMode ensures the VPN connection doesn't persist
	TailscaleEphemeralMode = true

	// TailscaleStateDir is the directory for Tailscale state (empty for ephemeral)
	TailscaleStateDir = ""

	// DefaultTailscaleHostname is the default hostname for the tsnet server
	DefaultTailscaleHostname = "vanish-backup"
)
