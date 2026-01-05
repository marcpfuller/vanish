package network_test

import (
	"context"
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestTsnetProvider_Coverage(t *testing.T) {
	tests := []struct {
		name      string
		skipShort bool
		run       func(t *testing.T)
	}{
		{
			name:      "close after start",
			skipShort: true,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()

				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				err := provider.Start(ctx, "invalid-key", "test-host")
				assert.Error(t, err)

				err = provider.Close()
				assert.NoError(t, err)

				_, err = provider.LocalAddr()
				assert.Error(t, err)

				_, err = provider.Dial(context.Background(), "tcp", "example.com:80")
				assert.Error(t, err)
			},
		},
		{
			name:      "dial with different network",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				defer func() { _ = provider.Close() }() //nolint:errcheck

				ctx := context.Background()

				_, err := provider.Dial(ctx, "udp", "example.com:53")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not started")
			},
		},
		{
			name:      "default hostname",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				assert.NotNil(t, provider)

				err := provider.Close()
				assert.NoError(t, err)
			},
		},
		{
			name:      "concurrent close",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()

				done := make(chan bool, 3)
				for i := 0; i < 3; i++ {
					go func() {
						err := provider.Close()
						assert.NoError(t, err)
						done <- true
					}()
				}

				for i := 0; i < 3; i++ {
					<-done
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipShort && testing.Short() {
				t.Skip("Skipping Tailscale test in short mode")
			}
			tt.run(t)
		})
	}
}
