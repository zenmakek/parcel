# Parcel

> No accounts. No links. Just a code.

Parcel is an OTP-based TCP file transfer system. Send files and folders between
devices using a temporary 6-digit code. No cloud storage. No accounts. No browser.

---

## Install

Download the binary for your platform from [Releases](https://github.com/zenmakek/parcel/releases):

| Platform | Binary |
|---|---|
| Linux (amd64) | `parcel-linux-amd64` |
| Linux (arm64) | `parcel-linux-arm64` |
| macOS (Intel) | `parcel-darwin-amd64` |
| macOS (Apple Silicon) | `parcel-darwin-arm64` |
| Windows | `parcel-windows-amd64.exe` |

**Linux / macOS:**
```bash
chmod +x parcel-linux-amd64
./parcel-linux-amd64
```

**Windows:**

Double-click `parcel-windows-amd64.exe` or run it in PowerShell.

---

## Usage

**Send a file:**
1. Run Parcel
2. Select Send
3. Enter the file or folder path
4. Share the 6-digit OTP with the receiver

**Receive a file:**
1. Run Parcel
2. Select Receive
3. Enter the OTP

Files land in `~/Downloads` automatically.

---

## How it works

```
Sender → Relay Server → Receiver
```

Files are streamed directly through the relay. Nothing is stored permanently.
OTPs are 6-digit, single-use, and expire after 5 minutes.

---

## Self-host the relay

```bash
VPS_IP=your.server.ip ./scripts/deploy.sh
```

Then point your client at it:

```bash
export PARCEL_RELAY=your.server.ip:8080
./parcel-linux-amd64
```

---

## Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/zenmakek/parcel.git
cd parcel
go run ./client/main.go
```

---

## Roadmap

- [ ] TLS encryption
- [ ] SHA256 integrity verification
- [ ] Resumable transfers
- [ ] LAN auto-discovery
- [ ] P2P mode

---

## Philosophy

Parcel should be lightweight, fast, open source, and simple.

Parcel avoids account requirements, complex setup, vendor lock-in, permanent cloud storage, and heavy GUI dependencies.

---

## License

MIT