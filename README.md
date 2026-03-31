# Voting with Draw

A Go web application for exhibition visitors to vote for their favourite photo via QR code. Winners can be drawn from voters who left their contact details.

## Features

- **Mobile-friendly voting page** — visitors scan a QR code and vote on their phone
- **Photo thumbnails** — configured via YAML, served as static files
- **One vote per visitor** — cookie-based session tracking
- **Live results page** — real-time updates via WebSocket
- **Contact details collection** — optional, enters voters into a prize draw
- **Prize draw** — randomly selects a winner from voters with contact details
- **Admin dashboard** — password-protected, view results, run draws, clear votes
- **Persistent storage** — SQLite database survives container restarts

## Quick Start

### 1. Configure your exhibition

Edit `config.yaml`:

```yaml
server:
  port: 8080
  admin_password: "your-secure-password"

exhibition:
  title: "Photo Exhibition 2026"
  description: "Vote for your favourite photo!"

photos:
  - id: "photo1"
    title: "Sunset Over the Lake"
    file: "sunset.jpg"
  - id: "photo2"
    title: "Mountain Morning"
    file: "mountain.jpg"
```

### 2. Add photo thumbnails

Place your photo files in the `photos/` directory. The filenames must match the `file` field in your config. Recommended size: 800×600px JPEG for fast mobile loading.

### 3. Run with Docker Compose

```bash
docker compose up -d
```

The app will be available at `http://localhost:8080`.

### 4. Generate a QR code

Generate a QR code pointing to your server's public URL (e.g., `https://vote.your-domain.com`). You can use any QR code generator — the QR simply points to the voting page URL.

## Pages

| URL        | Description                          |
| ---------- | ------------------------------------ |
| `/`        | Voting page (share this via QR code) |
| `/results` | Live results with real-time updates  |
| `/admin`   | Admin dashboard (password-protected) |

## Admin

Access `/admin` with:

- **Username:** `admin`
- **Password:** value from `config.yaml` or `ADMIN_PASSWORD` env var

From the admin dashboard you can:

- View vote counts and percentages
- Run the prize draw (picks a random voter who left contact details)
- Clear all votes and draw results

## Development

### Prerequisites

- Go 1.22+
- [sqlc](https://sqlc.dev/) (for regenerating database code)

### Run locally

```bash
# Create a data directory
mkdir -p data

# Run the server
go run ./cmd/server/
```

### Regenerate sqlc code

If you modify the SQL schema or queries:

```bash
sqlc generate
```

## Environment Variables

| Variable         | Default         | Description                      |
| ---------------- | --------------- | -------------------------------- |
| `CONFIG_PATH`    | `config.yaml`   | Path to the configuration file   |
| `DB_PATH`        | `data/votes.db` | Path to the SQLite database file |
| `ADMIN_PASSWORD` | _(from config)_ | Override admin password          |
| `GIN_MODE`       | `debug`         | Set to `release` for production  |

## Deployment on Hetzner

1. Provision a VPS (CX22 or similar, ~€4/mo)
2. Install Docker and Docker Compose
3. Clone this repository
4. Edit `config.yaml` and add photos to `photos/`
5. Run `docker compose up -d`
6. Set up a reverse proxy (Caddy recommended) for HTTPS:

```bash
# Install Caddy
apt install caddy

# /etc/caddy/Caddyfile
vote.your-domain.com {
    reverse_proxy localhost:8080
}
```

Caddy automatically provisions TLS certificates via Let's Encrypt.

## Architecture

```
Single Docker container:
├── Gin HTTP server (templates + API + WebSocket)
├── SQLite database (WAL mode, on Docker volume)
├── Static file serving (CSS, JS, photos)
└── gorilla/websocket hub (live result broadcasts)
```

## Tech Stack

- **Go** + **Gin** web framework
- **SQLite** via mattn/go-sqlite3 (WAL mode)
- **sqlc** for type-safe SQL queries
- **gorilla/websocket** for live results
- **Docker** for deployment
