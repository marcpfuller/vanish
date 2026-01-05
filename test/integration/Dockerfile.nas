FROM ubuntu:24.04

# Install OpenSSH Server and Tailscale
RUN apt-get update && apt-get install -y \
    openssh-server \
    curl \
    ca-certificates \
    && curl -fsSL https://tailscale.com/install.sh | sh \
    && mkdir -p /var/run/sshd \
    && rm -rf /var/lib/apt/lists/*

# Create test user for SFTP
RUN useradd -m -s /bin/bash nasuser \
    && echo "nasuser:naspass123" | chpasswd \
    && mkdir -p /home/nasuser/backups \
    && chown -R nasuser:nasuser /home/nasuser

# Configure SSH for password authentication
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config \
    && sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config \
    && sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config \
    && echo "Subsystem sftp /usr/lib/openssh/sftp-server" >> /etc/ssh/sshd_config

# Create directory for Tailscale socket
RUN mkdir -p /var/run/tailscale

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 22

HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=3 \
    CMD pgrep sshd || exit 1

ENTRYPOINT ["/entrypoint.sh"]
