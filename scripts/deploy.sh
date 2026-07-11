#!/bin/bash
set -e

VPS_USER="${VPS_USER:-root}"
VPS_IP="${VPS_IP}"
VPS_PORT="${VPS_PORT:-22}"
DEPLOY_DIR="/opt/parcel"

if [ -z "$VPS_IP" ]; then
    echo "[error] VPS_IP is not set"
    echo "Usage: VPS_IP=1.2.3.4 ./scripts/deploy.sh"
    exit 1
fi

echo "[deploy] building relay server for Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o bin/parcel-server-linux ./server/main.go

echo "[deploy] uploading binary..."
ssh -p "$VPS_PORT" "$VPS_USER@$VPS_IP" "mkdir -p $DEPLOY_DIR"
rsync -avz -e "ssh -p $VPS_PORT" bin/parcel-server-linux "$VPS_USER@$VPS_IP:$DEPLOY_DIR/parcel-server"

echo "[deploy] uploading systemd service..."
rsync -avz -e "ssh -p $VPS_PORT" scripts/parcel-relay.service "$VPS_USER@$VPS_IP:/etc/systemd/system/parcel-relay.service"

echo "[deploy] restarting service..."
ssh -p "$VPS_PORT" "$VPS_USER@$VPS_IP" "
    chmod +x $DEPLOY_DIR/parcel-server
    systemctl daemon-reload
    systemctl enable parcel-relay
    systemctl restart parcel-relay
    ufw allow 8080/tcp
    ufw allow 9090/tcp
    ufw allow 3478/tcp
    ufw reload
"

echo "[deploy] done. relay+tracker+stun running at $VPS_IP"
