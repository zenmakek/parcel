# Parcel

> No accounts. No links. Just a code.

Parcel is a P2P file transfer system built on content-addressed TCP streaming.  
Share files using a SHA256 hash — no cloud storage, no accounts, no browser.

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/zenmakek/parcel/main/scripts/install.sh | bash
```

Or download a binary from **[Releases](https://github.com/zenmakek/parcel/releases)**.

| Platform | Binary |
| :-------- | :----- |
| Linux (amd64) | `parcel-linux-amd64` |
| Linux (arm64) | `parcel-linux-arm64` |
| macOS (Intel) | `parcel-darwin-amd64` |
| macOS (Apple Silicon) | `parcel-darwin-arm64` |
| Windows | `parcel-windows-amd64.exe` |

---

## Usage

### Send

1. Run `parcel`
2. Select **Send**
3. Enter file or folder path
4. Share the 64-character SHA256 hash with the receiver

### Receive

1. Run `parcel`
2. Select **Receive**
3. Enter the hash

Files land in `~/Downloads` automatically.

---

## How it works

```text
Sender hashes file → announces hash to DHT
        ↓
Receiver enters hash → DHT returns peer list
        ↓
Receiver connects directly to peers
        ↓
Chunks downloaded in parallel, each verified by hash
        ↓
Receiver becomes a seeder automatically
```

### Features

- Content addressing — files identified by SHA256 hash
- Chunked transfer — 256KB chunks, each independently verified
- Parallel download — fetch chunks from multiple peers simultaneously
- NAT traversal — direct connections via TCP hole punching
- Noise encryption — ChaCha20-Poly1305 on all peer connections
- Kademlia DHT — decentralized peer discovery, no central server required
- Relay fallback — DigitalOcean relay for cases where direct connection fails

---

## Architecture

```text
v1.0: Sender → Relay → Receiver (OTP)

v2.0: Sender ↔ DHT ↔ Receiver (hash, direct P2P)
```

---

## Self-host the server

```bash
VPS_IP=your.server.ip ./scripts/deploy.sh
export PARCEL_RELAY=your.server.ip:8080
```

---

## Build from source

Requires **Go 1.21+**.

```bash
git clone https://github.com/zenmakek/parcel.git
cd parcel
go run ./client/main.go
```

---

## Roadmap

- [x] OTP-based relay transfer (v1.0)
- [x] Content addressing (SHA256)
- [x] Chunk engine + resumable transfers
- [x] Peer identity (Ed25519)
- [x] Tracker server
- [x] Direct peer connections
- [x] NAT traversal (STUN + hole punching)
- [x] Parallel chunk download
- [x] Seeding
- [x] Noise encryption
- [x] Kademlia DHT
- [ ] LAN auto-discovery
- [ ] Mobile clients