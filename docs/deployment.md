# Parcel Relay — Deployment Guide

## Requirements

- Ubuntu 24.04 LTS VPS
- Port 8080 open (TCP)
- SSH access

## Deploy

```bash
VPS_IP=YOUR_SERVER_IP ./scripts/deploy.sh
```

## Verify

SSH into the server and check the service:

```bash
systemctl status parcel-relay
journalctl -u parcel-relay -f
```

## Use the public relay

Set the relay address on both sender and receiver machines:

```bash
export PARCEL_RELAY=YOUR_SERVER_IP:8080
go run ./client/main.go
```

## Firewall

If your VPS has ufw enabled:

```bash
ufw allow 8080/tcp
ufw reload
```

## Update

Re-run the deploy script to push a new binary. The service restarts automatically.
