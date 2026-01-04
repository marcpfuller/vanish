package network

import (
	"context"
	"fmt"
	"net"
	"sync"

	"tailscale.com/tsnet"
)

// TsnetProvider implements TailscaleProvider using tsnet for ephemeral connections
type TsnetProvider struct {
	server   *tsnet.Server
	started  bool
	hostname string
	mu       sync.RWMutex
}

// NewTsnetProvider creates a new Tailscale provider with ephemeral configuration
func NewTsnetProvider() *TsnetProvider {
	return &TsnetProvider{
		started: false,
	}
}

// Start initializes and starts the ephemeral Tailscale server
func (t *TsnetProvider) Start(ctx context.Context, authKey string, hostname string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("tailscale server already started")
	}

	// Create ephemeral tsnet server
	srv := &tsnet.Server{
		Hostname:  hostname,
		AuthKey:   authKey,
		Ephemeral: TailscaleEphemeralMode,
		Dir:       TailscaleStateDir,                   // Empty for ephemeral
		Logf:      func(format string, args ...any) {}, // Silent logging
	}

	// Start the server (this will authenticate and join the tailnet)
	if _, err := srv.Up(ctx); err != nil {
		return fmt.Errorf("failed to start Tailscale server: %w", err)
	}

	t.server = srv
	t.hostname = hostname
	t.started = true

	return nil
}

// Dial creates a connection through the Tailscale network
func (t *TsnetProvider) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.started || t.server == nil {
		return nil, fmt.Errorf("tailscale server not started")
	}

	// Use tsnet's Dial which routes through the Tailscale network
	conn, err := t.server.Dial(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial through Tailscale: %w", err)
	}

	return conn, nil
}

// Close stops the Tailscale server and cleans up resources
func (t *TsnetProvider) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return nil // Already closed
	}

	if t.server != nil {
		if err := t.server.Close(); err != nil {
			return fmt.Errorf("failed to close Tailscale server: %w", err)
		}
		t.server = nil
	}

	t.started = false
	return nil
}

// LocalAddr returns the Tailscale IP address of this node
func (t *TsnetProvider) LocalAddr() (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.started || t.server == nil {
		return "", fmt.Errorf("tailscale server not started")
	}

	lc, err := t.server.LocalClient()
	if err != nil {
		return "", fmt.Errorf("failed to get local client: %w", err)
	}

	status, err := lc.Status(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	if len(status.TailscaleIPs) == 0 {
		return "", fmt.Errorf("no Tailscale IP addresses found")
	}

	return status.TailscaleIPs[0].String(), nil
}
