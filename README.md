# feline-matrix

Self-hosted Matrix homeserver with invite-gated registration and Element Call voice/video support.

Runs [Tuwunel](https://github.com/matrix-construct/tuwunel) (a Rust Matrix homeserver) on a Hetzner VPS behind Caddy, with a custom Go reverse proxy that adds invite-code-gated user registration.

## Architecture

```
Internet
  |
  Caddy (:80, :443, :8448)
    |
    +-- /.well-known/matrix/*      -> static JSON (served by Caddy)
    +-- /livekit/jwt*              -> lk-jwt-service:8080  (LiveKit JWT auth)
    +-- /livekit/sfu*              -> livekit:7880          (WebSocket)
    +-- /*                         -> registration-proxy:8008
                                        +-- /register/*    -> embedded static UI
                                        +-- /api/register  -> invite-gated handler
                                        +-- /*             -> tuwunel:6167
    |
    +-- :8448 /*                   -> tuwunel:6167          (federation)

  LiveKit host ports:
    7881/tcp              (ICE TCP fallback)
    50000-50200/udp       (WebRTC media)
```

**Caddy** -- TLS termination and routing for all traffic, including federation on port 8448.

**Tuwunel** -- Rust Matrix homeserver using RocksDB for storage. No external database needed.

**Registration proxy** (`registration/main.go`) -- Go binary (stdlib only, zero dependencies). Serves the registration UI, validates invite codes via Matrix UIA (`m.login.registration_token`), and reverse-proxies everything else to Tuwunel.

**LiveKit** -- WebRTC media server powering Element Call voice/video.

**lk-jwt-service** -- Issues JWT tokens so Matrix clients can authenticate with LiveKit.

**[Matrix Claude Bot](https://github.com/feline-dis/matrix-claude-bot)** -- A Go bot (`@claude:ohana-matrix.xyz`) that responds to @-mentions using the Anthropic Claude API, maintaining per-thread conversation history. Runs as a systemd service outside Docker.

## Self-hosting

### Prerequisites

- Docker and Docker Compose
- A domain with DNS pointing to your server
- Port 443 (HTTPS) and 8448 (federation) open

### Setup

1. Clone the repo and configure secrets:

   ```sh
   git clone https://github.com/felinedis/feline-matrix.git
   cd feline-matrix
   cp .env.example .env
   # Edit .env -- set INVITE_CODE, LIVEKIT_KEY, LIVEKIT_SECRET
   ```

2. Update configuration files with your domain:
   - `Caddyfile` -- replace `ohana-matrix.xyz` with your domain
   - `config/conduwuit.toml` -- set `server_name` to your domain
   - `docker-compose.yml` -- update `LIVEKIT_URL` and `LIVEKIT_FULL_ACCESS_HOMESERVERS` in the `lk-jwt-service` section

3. Generate LiveKit API keys (if you don't have them):

   ```sh
   docker run --rm livekit/generate-keys
   ```

4. Deploy:

   ```sh
   docker compose up -d --build
   ```

5. Set up DNS:
   - A record for your domain pointing to your server's IP
   - Port 8448 must be reachable for federation (Caddy handles TLS)

6. Verify federation: https://federationtester.matrix.org

## Joining the server

1. **Get an invite code** from the server admin.
2. **Register** at `https://<domain>/register/` -- enter a username, password, and the invite code.
3. **Download a Matrix client:**
   - [Element Web](https://app.element.io)
   - [Element Desktop](https://element.io/download)
   - [Element Mobile](https://element.io/download) (iOS / Android)
4. **Sign in** with your Matrix ID (`@username:<domain>`) and set the homeserver URL to `https://<domain>`.
5. **Voice/video calls** work out of the box via Element Call (built into Element clients).

## Claude bot setup

The Matrix Claude Bot runs as a standalone binary managed by systemd (not part of the Docker Compose stack). The deploy workflow automatically updates it to the latest release.

**Manual setup** (already done on the VPS):

1. Download the latest release:

   ```sh
   curl -fsSL -o /opt/matrix-claude-bot/matrix-claude-bot \
     https://github.com/feline-dis/matrix-claude-bot/releases/latest/download/matrix-claude-bot-linux-amd64
   chmod +x /opt/matrix-claude-bot/matrix-claude-bot
   ```

2. Create `/opt/matrix-claude-bot/config.yaml`:

   ```yaml
   matrix:
     homeserver_url: "https://ohana-matrix.xyz"
     user_id: "@claude:ohana-matrix.xyz"
     access_token: "<token>"

   anthropic:
     api_key: "<key>"

   claude:
     model: "claude-sonnet-4-20250514"
     max_tokens: 4096
     system_prompt: "You are a helpful assistant."
   ```

3. Enable the systemd service:

   ```sh
   systemctl enable --now matrix-claude-bot
   ```

4. Invite `@claude:ohana-matrix.xyz` to a room and mention it to interact.

## Project structure

| File | Description |
|---|---|
| `registration/main.go` | Go reverse proxy with invite-gated registration (single file, stdlib only) |
| `registration/www/` | Static HTML/CSS/JS for the registration UI (embedded into the Go binary) |
| `docker-compose.yml` | Full stack: Caddy, Tuwunel, registration proxy, LiveKit, lk-jwt-service |
| `Caddyfile` | Reverse proxy routing and TLS termination |
| `config/conduwuit.toml` | Tuwunel homeserver configuration |
| `livekit/livekit.yaml` | LiveKit server configuration |
| `Dockerfile` | Two-stage build for the Go registration proxy |
| `.env.example` | Template for required environment variables |

## Development

### Build the Go proxy locally

```sh
cd registration && go build -o registration-proxy .
```

### Run the full stack

```sh
cp .env.example .env   # fill in values
docker compose up -d --build
```

### Deploy

Push to `master` triggers automatic deployment via GitHub Actions. Manual deploy:

```sh
git pull && docker compose up -d --build
```
