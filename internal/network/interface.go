package network

import (
	"context"
	"net"
)

// TailscaleProvider defines the interface for managing ephemeral Tailscale connections
type TailscaleProvider interface {
	// Start initializes and starts the ephemeral Tailscale server
	Start(ctx context.Context, authKey string, hostname string) error

	// Dial creates a connection through the Tailscale network
	Dial(ctx context.Context, network, address string) (net.Conn, error)

	// Close stops the Tailscale server and cleans up resources
	Close() error

	// LocalAddr returns the Tailscale IP address of this node
	LocalAddr() (string, error)
}
