# ohana-matrix-server-registration

A standalone Go registration proxy for Matrix homeservers. Sits in front of a Matrix homeserver and adds invite-code-gated user registration via the Matrix User-Interactive Authentication (UIA) flow.

## How it works

The proxy reverse-proxies all traffic to the upstream Matrix homeserver, except:

- **`/register/`** -- serves an embedded registration UI (HTML/CSS/JS)
- **`/api/register`** -- handles registration requests, validating invite codes as `m.login.registration_token` via the Matrix UIA spec

Users visit `/register/`, enter a username, password, and invite code. The proxy validates the invite code against the homeserver's registration token, then completes the registration flow.

All other Matrix client and federation traffic passes through transparently.

## Building

### Local Go build

```sh
cd registration && go build -o registration-proxy .
```

### Nix build

```sh
nix build .#registration-proxy
```

The binary will be at `./result/bin/registration-proxy`.

## Configuration

The proxy reads its configuration from environment variables:

- `MATRIX_HOST` -- upstream homeserver address (default: `localhost:6167`)
- `INVITE_CODE` -- required invite code for registration
- `PORT` -- listen port (default: `8008`)

## Server deployment

For full server deployment (homeserver, TLS, LiveKit, etc.), see [matrix-server-nix](https://github.com/feline-dis/matrix-server-nix).

## Project structure

| File | Description |
|---|---|
| `registration/main.go` | Go reverse proxy with invite-gated registration (single file, stdlib only) |
| `registration/www/` | Static HTML/CSS/JS for the registration UI (embedded into the binary) |
| `flake.nix` | Nix package definition |
