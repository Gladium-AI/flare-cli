---
name: flare-cli
description: Use this skill when the task is to authenticate, configure, expose, inspect, update, log, or close Cloudflare Tunnel plus Cloudflare Access sessions with `flare`. Trigger on requests about sharing localhost securely, creating preview URLs for local apps, exposing static files or Docker services, or cleaning up Cloudflare resources created by flare-cli.
license: MIT
metadata:
  author: Gladium AI
  version: 1.0.0
  category: developer-tools
  tags:
    - cloudflare
    - tunnel
    - access
    - preview
    - localhost
    - docker
---

# Flare CLI

Use this skill to operate `flare` safely and consistently.

## Use This Skill For

- Exposing an existing local HTTP service with `local:http`
- Launching and exposing a command with `local:command`
- Sharing static files or a directory with `builtin:static` or `builtin:file-browser`
- Exposing Docker containers or Compose services
- Authenticating or configuring Cloudflare account, zone, and domain settings
- Inspecting, updating, logging, or closing active sessions

## Core Rules

- Prefer `flare` over raw `cloudflared` or manual Cloudflare dashboard changes when the task fits the CLI surface.
- Start with `flare doctor` for first-run setup and troubleshooting.
- Use `flare auth login` for interactive authentication instead of editing credential files manually.
- Prefer `--json` when another agent or script will consume the result.
- Match the origin type to the workload instead of wrapping everything in `local:command`.
- Close disposable sessions with `flare close` so the tunnel, DNS record, and Access app are removed.
- Do not use forceful teardown or preserve remote resources unless the user explicitly asks.

## Quick Workflow

1. Verify prerequisites with `flare doctor`.
2. Authenticate with `flare auth login`, or inspect current auth with `flare auth whoami`.
3. Expose the target with `flare serve <origin-type> ...`.
4. Inspect or update the session with `flare status`, `flare logs`, or `flare update`.
5. Clean up with `flare close <session-id>`.

## Prerequisites

- `flare` installed and available on `PATH`, or use the repo-local binary while developing
- `cloudflared` installed
- Docker installed for `docker:*` origin types
- A Cloudflare account with a managed domain
- Cloudflare Zero Trust enabled for Access policies

## Standard Command Path

```bash
flare doctor
flare auth login
flare serve local:http --url http://localhost:3000 --subdomain preview --auth otp --json
flare status <session-id> --json
flare close <session-id>
```

For static sites:

```bash
flare serve builtin:static --path ./dist --subdomain preview --spa --auth otp
```

For containerized services:

```bash
flare serve docker:container --image nginx:alpine --container-port 80 --subdomain demo
```

## Origin Selection

- `local:http`: the service is already running on a local URL
- `local:command`: you need `flare` to launch the process that serves HTTP
- `builtin:static`: serve a static directory or SPA build output
- `builtin:file-browser`: share a directory with upload/download controls
- `docker:container`: run and expose a single container
- `docker:compose-service`: start and expose one Compose service

## Troubleshooting

- Run `flare doctor` if setup looks wrong.
- Use `flare auth whoami` to confirm the current token and discovered account context.
- Use `flare list --stale` to spot leaked or unhealthy sessions.
- Use `flare logs <session-id>` and `flare status <session-id> --json` before rerunning a failed session.
- If `serve` fails during provisioning, verify `cloudflare.account_id`, `cloudflare.zone_id`, and `cloudflare.domain` in config.
