#!/usr/bin/env bash
# ==============================================================================
# Kather Baksho - Automated Production EC2 Deployment Script
# ==============================================================================
# Usage:
#   chmod +x deploy/ec2-setup.sh
#   ./deploy/ec2-setup.sh
# ==============================================================================

set -euo pipefail

echo "========================================================"
echo "  🌱 Deploying Kather Baksho on Ubuntu / AWS EC2"
echo "========================================================"

# Check if running with sudo or root
if [ "$EUID" -ne 0 ]; then
  echo "[-] Please run as root or with sudo: sudo ./deploy/ec2-setup.sh"
  exit 1
fi

echo "[1/6] Updating system packages..."
apt-get update -y
apt-get install -y ca-certificates curl gnupg lsb-release git ufw htop jq

echo "[2/6] Verifying / Installing Docker CE & Docker Compose..."
if ! command -v docker &> /dev/null; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg

  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
    $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

systemctl enable docker
systemctl start docker

echo "[3/6] Setting kernel parameters for high-performance databases..."
sysctl -w vm.max_map_count=262144
if ! grep -q "vm.max_map_count=262144" /etc/sysctl.conf; then
  echo "vm.max_map_count=262144" >> /etc/sysctl.conf
fi

echo "[4/6] Creating production systemd service for auto-restart..."
cat << 'EOF' > /etc/systemd/system/kather_baksho.service
[Unit]
Description=Kather Baksho Production Docker Compose Ecosystem
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/kather_baksho
ExecStart=/usr/bin/docker compose -f /opt/kather_baksho/docker-compose.yml up -d
ExecStop=/usr/bin/docker compose -f /opt/kather_baksho/docker-compose.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable kather_baksho.service

echo "[5/6] Launching multi-container stack via Docker Compose..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCRIPT_DIR"

docker compose down --remove-orphans || true
docker compose up -d --build

echo "[6/6] Validating container health status..."
sleep 5
docker compose ps

echo "========================================================"
echo "  ✅ Kather Baksho Deployment Successful!"
echo "  Ingress Gateway:  http://localhost:8085"
echo "  Swagger UI Docs:  http://localhost:8085/docs"
echo "  Traefik Dashboard: http://localhost:8086/dashboard/"
echo "  Grafana Portal:   http://localhost:3000"
echo "========================================================"

