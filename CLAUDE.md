# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Self-hosted Matrix homeserver (Conduwuit) on a Hetzner VPS for `ohana-matrix.xyz`, with Element Call (LiveKit) for voice/video. A Go reverse proxy sits in front of Conduwuit to add invite-code-gated user registration. Everything runs via `docker-compose.yml`.

## Architecture

```
Internet
  |
  Caddy (:80, :443, :8448)
    |
    +-- /.well-known/matrix/client  -> static JSON (served by Caddy)
    +-- /.well-known/matrix/server  -> static JSON (served by Caddy)
    +-- /sfu/get*                   -> lk-jwt-service:8080 (JWT auth for LiveKit)
    +-- /sfu*                       -> livekit:7880 (WebSocket)
    +-- /*                          -> registration-proxy:8008
                                         +-- /register/*    -> embedded static UI
                                         +-- /api/register  -> invite-gated handler
                                         +-- /*             -> conduwuit:6167
    |
    +-- :8448 /*                    -> conduwuit:6167 (federation, TLS by Caddy)

  LiveKit direct host ports:
    7881/tcp  (ICE TCP fallback)
    50000-50200/udp (WebRTC media)
```

- **Caddy** terminates TLS for everything including federation on :8448.
- **Conduwuit** is a Rust-based Matrix homeserver using RocksDB for local storage. No external database needed.
- **Registration proxy** (`registration/main.go`): Go binary using only stdlib. Serves the registration UI, validates invite codes via Matrix UIA (m.login.registration_token), and reverse-proxies all other traffic to Conduwuit.
- **LiveKit** handles WebRTC media for Element Call voice/video.
- **lk-jwt-service** issues JWT tokens for LiveKit access, scoped to `ohana-matrix.xyz`.
- **Matrix Claude Bot** (`@claude:ohana-matrix.xyz`): standalone Go binary from [feline-dis/matrix-claude-bot](https://github.com/feline-dis/matrix-claude-bot). Responds to @-mentions via the Anthropic Claude API. Runs as a systemd service (`matrix-claude-bot.service`) outside the Docker stack. Binary at `/opt/matrix-claude-bot/matrix-claude-bot`, config at `/opt/matrix-claude-bot/config.yaml`.
- Static registration UI files (`registration/www/`) are embedded into the Go binary via `//go:embed`.

## Build and Development

### Build the Go proxy locally

```bash
cd registration && go build -o registration-proxy .
```

### Run the full stack locally

```bash
cp .env.example .env   # fill in values
docker compose up -d
```

### Build just the proxy image

```bash
docker build -t feline-matrix .
```

The Dockerfile is a two-stage build: compiles the Go proxy in `golang:1.25-alpine`, then copies it into a plain `alpine:latest` image.

### Deploy

Push to `master` triggers GitHub Actions: builds the registration proxy image on GHCR, then SSHs into the VPS to pull the image, restart containers, and update the Claude bot binary from its latest GitHub release. Manual deploy:

```bash
cd ~/feline-matrix && git pull origin master && docker compose pull && docker compose up -d --remove-orphans
```

### Create accounts

Users register through the invite-gated UI at `/register/`. The invite code doubles as the Conduwuit registration token.

## Runtime Secrets

Managed via `.env` file (not committed).

- `INVITE_CODE` - required invite code for registration (also used as Conduwuit's registration token)
- `LIVEKIT_KEY` - LiveKit API key
- `LIVEKIT_SECRET` - LiveKit API secret

## Key Files

- `registration/main.go` - the entire proxy server (single file, stdlib only)
- `docker-compose.yml` - full stack: Caddy, Conduwuit, registration proxy, LiveKit, lk-jwt-service
- `Caddyfile` - reverse proxy routing and TLS termination
- `livekit/livekit.yaml` - LiveKit server configuration
- `.env.example` - template for required environment variables

## Conventions

- The Go module (`registration/`) uses zero external dependencies -- stdlib only.
- No test suite exists yet.
- No linter or formatter is configured. Standard `gofmt` applies.
- Conduwuit stores all data in a RocksDB database on a Docker volume (`conduwuit_data`).
