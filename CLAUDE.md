# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A standalone Go registration proxy for Matrix homeservers. Reverse-proxies all traffic to an upstream Matrix homeserver while adding invite-code-gated user registration via the Matrix UIA flow (`m.login.registration_token`).

The registration UI is embedded into the Go binary via `//go:embed`.

## Build and Development

### Build locally

```bash
cd registration && go build -o registration-proxy .
```

### Build with Nix

```bash
nix build .#registration-proxy
```

## Key Files

- `registration/main.go` -- the entire proxy server (single file, stdlib only)
- `registration/www/` -- static HTML/CSS/JS for the registration UI (embedded into binary)
- `flake.nix` -- Nix package definition

## Conventions

- The Go module (`registration/`) uses zero external dependencies -- stdlib only.
- No test suite exists yet.
- No linter or formatter is configured. Standard `gofmt` applies.
