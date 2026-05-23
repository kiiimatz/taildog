# taildog

Open-source tunneling service — expose local ports through a relay server, with a visual canvas dashboard.

[![Release](https://img.shields.io/github/v/release/kiiimatz/taildog)](https://github.com/kiiimatz/taildog/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Install

**Linux / macOS**
```bash
curl -fsSL https://raw.githubusercontent.com/kiiimatz/taildog/main/install/install.sh | bash
```

**Windows (PowerShell, run as Administrator)**
```powershell
iwr https://raw.githubusercontent.com/kiiimatz/taildog/main/install/install.ps1 | iex
```

## Quick start

```bash
# 1. Edit ~/.taildog/config.yaml — set relay_host
taildog status

# 2. Connect to relay
taildog up

# 3. Add your first tunnel
taildog add http 3000          # forward local :3000 → relay :3000
taildog add tcp 5432 15432     # forward local :5432 → relay :15432

# 4. List tunnels
taildog status

# 5. Remove a tunnel
taildog remove <id>

# 6. Stop daemon
taildog down
```

## Dashboard

Open `http://<relay-host>:7701` in your browser.
- Drag clients from the sidebar onto the canvas
- Drag a line from a client's handle to the server handle to create a tunnel
- Real-time status updates via WebSocket

## Protocol reference

| Protocol | Description                        | Default port |
|----------|------------------------------------|--------------|
| `tcp`    | Raw TCP forwarding                 | any          |
| `udp`    | UDP forwarding                     | any          |
| `http`   | HTTP reverse proxy                 | 80           |
| `https`  | HTTPS with TLS termination         | 443          |
| `socks5` | SOCKS5 proxy                       | 1080         |
| `quic`   | QUIC (UDP-based, quic-go)          | any          |
| `ws`     | WebSocket (ws://)                  | any          |
| `scp`    | SCP / SSH port forward (port 22)   | 22           |
| `smtp`   | SMTP forwarding (port 25 / 587)    | 25           |

## Config file

`~/.taildog/config.yaml` (Linux/macOS) · `%APPDATA%\taildog\config.yaml` (Windows)

```yaml
relay_host: relay.example.com
relay_port: 7700
auth_token: ""
ca_cert_path: ""   # path to relay CA cert (for self-signed TLS)
tunnels:
  - id: abc123
    protocol: http
    local_port: 3000
    remote_port: 3000
```

## Relay API

The relay exposes a REST + WebSocket API on port **7701** (TLS).

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/login` | Obtain access + refresh tokens |
| `POST` | `/api/auth/refresh` | Refresh access token |
| `POST` | `/api/auth/logout` | Revoke session |
| `GET`  | `/api/auth/me` | Current user info |
| `GET`  | `/api/clients` | List connected clients |
| `GET`  | `/api/clients/:id/tunnels` | Tunnels for a client |
| `POST` | `/api/tunnels` | Create tunnel |
| `DELETE` | `/api/tunnels/:id` | Delete tunnel |
| `GET`  | `/api/server/info` | Relay version + uptime |
| `WS`   | `/api/events` | Real-time event stream |
| `GET`  | `/api/admin/audit-log` | Audit log (admin only) |

## Security

- mTLS between relay and clients
- JWT auth (access 15 min, refresh 7 days) for the dashboard API
- bcrypt (cost 14) password hashing
- httpOnly + SameSite=Strict cookie for refresh token
- Rate limiting: 5 failures per IP → 10-minute lockout
- Audit log for all auth and tunnel events

## Dashboard screenshot

<!-- TODO: add screenshot -->

## Repository structure

```
taildog/
├── relay/        # Relay daemon (Go) — mTLS server, REST API, SQLite
├── client/       # Client daemon (Go) — CLI, tunnel management
├── dashboard/    # Web UI (React + Vite + TypeScript)
├── install/      # Install scripts
│   ├── install.sh
│   └── install.ps1
└── .github/
    └── workflows/release.yml
```

## License

MIT © taildog contributors
