#!/bin/bash
set -e

echo "Starting Tailscale daemon..."
# Ensure socket directory exists
mkdir -p /var/run/tailscale

# Start Tailscale daemon in userspace networking mode (no root required)
tailscaled --state=mem: --tun=userspace-networking --socket=/var/run/tailscale/tailscaled.sock &
TAILSCALED_PID=$!

# Wait for tailscaled socket to be ready
echo "Waiting for Tailscale daemon to be ready..."
for i in {1..30}; do
    if [ -S /var/run/tailscale/tailscaled.sock ]; then
        echo "✓ Tailscale daemon socket ready"
        break
    fi
    echo "  Waiting for socket... ($i/30)"
    sleep 1
done

# Verify daemon is running
if ! kill -0 $TAILSCALED_PID 2>/dev/null; then
    echo "ERROR: Tailscale daemon failed to start"
    exit 1
fi

# Authenticate with Tailscale if auth key is provided
if [ -n "$TS_AUTHKEY" ]; then
    echo "Authenticating with Tailscale..."
    
    # Remove problematic flags and retry on failure
    if ! tailscale up --authkey="$TS_AUTHKEY" --hostname=vanish-test-nas --accept-routes; then
        echo "First attempt failed, retrying without --accept-routes..."
        if ! tailscale up --authkey="$TS_AUTHKEY" --hostname=vanish-test-nas; then
            echo "ERROR: Tailscale authentication failed"
            echo "Tailscale daemon logs:"
            journalctl -u tailscaled --no-pager || echo "No journalctl available"
            exit 1
        fi
    fi
    
    # Wait for Tailscale to be fully connected
    echo "Waiting for Tailscale connection..."
    for i in {1..60}; do
        if tailscale status --json >/dev/null 2>&1; then
            IP=$(tailscale ip -4 2>/dev/null || echo "")
            if [ -n "$IP" ]; then
                echo "✓ Tailscale connected! IP: $IP"
                break
            fi
        fi
        echo "  Connecting... ($i/60)"
        sleep 2
    done
    
    # Final status check
    echo "Tailscale final status:"
    tailscale status || echo "Warning: Could not get Tailscale status"
    tailscale ip -4 || echo "Warning: Could not get Tailscale IP"
else
    echo "No TS_AUTHKEY provided, skipping Tailscale setup"
fi

# Generate host keys if they don't exist
echo "Generating SSH host keys..."
ssh-keygen -A

echo "Starting SSH server..."
# Start SSH server in foreground
/usr/sbin/sshd -D -e
