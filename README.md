# flare-cli

**Expose any local app to the internet in one command — secured with Cloudflare Tunnel + Access.**

flare-cli is a CLI tool built for AI agents and developers that serves local or containerized applications through [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/) and protects them with [Cloudflare Access](https://developers.cloudflare.com/cloudflare-one/policies/access/). One command creates the tunnel, DNS record, and access policy — and tears it all down when you're done.

```
flare serve builtin:static --path ./dist --subdomain myapp --auth otp
```

```
✓ Origin ready at http://127.0.0.1:49152
✓ Tunnel created: flare-a1b2c3d4
✓ DNS record created for myapp.example.com
✓ Access application created (auth: otp)
✓ cloudflared running (PID 12345)

Session Active
  Hostname: https://myapp.example.com
  Auth:     otp
```

## Why flare-cli?

### Built for AI agents

flare-cli is designed to be called by AI agents that need to expose services programmatically. Every command supports `--json` output, session IDs are stable UUIDs, and the `--idempotency-key` flag lets agents safely retry operations.

```bash
# Agent creates a preview environment
SESSION=$(flare serve builtin:static --path ./build --subdomain pr-42 --json | jq -r '.id')

# Agent checks status
flare status $SESSION --json

# Agent tears it down
flare close $SESSION
```

### Built for developers

No YAML configs, no dashboards, no manual DNS entries. One command to go from local to live with authentication.

```bash
# Serve a React app
flare serve builtin:static --path ./build --subdomain myapp --spa

# Expose a local dev server
flare serve local:http --url http://localhost:3000 --subdomain api

# Share a directory with file upload
flare serve builtin:file-browser --path ~/shared --allow-upload --subdomain files

# Run a Docker container publicly
flare serve docker:container --image nginx:alpine --container-port 80 --subdomain demo
```

## Install

**One-liner (Linux / macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/Gladium-AI/flare-cli/main/install.sh | sh
```

Downloads the latest release for your OS/arch and installs to `~/.local/bin`.

**Install the Claude + Codex skill:**

```bash
curl -fsSL https://raw.githubusercontent.com/Gladium-AI/flare-cli/main/install-skill.sh | sh
```

By default this installs `flare-cli` into `~/.claude/skills` and `${CODEX_HOME:-~/.codex}/skills`. Set `SKILLS_DIR=/path/to/skills` to install to a single custom skill directory instead.

**Build from source:**

```bash
git clone https://github.com/Gladium-AI/flare-cli.git
cd flare-cli
make install   # installs to ~/.local/bin
```

### Prerequisites

- **[cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/)** — Cloudflare's tunnel connector
- **Docker** (optional) — only needed for `docker:*` origin types
- A **Cloudflare account** with a domain (free plan works)
- **Cloudflare Zero Trust** enabled ([one.dash.cloudflare.com](https://one.dash.cloudflare.com/) — free for up to 50 users)

## Quick start

### 1. Authenticate

```bash
flare auth login
```

This opens your browser to the Cloudflare dashboard where you create an API token with these permissions:

| Scope | Permission | Access |
|-------|-----------|--------|
| Account | Cloudflare Tunnel | Edit |
| Account | Access: Apps and Policies | Edit |
| Zone | DNS | Edit |

After you paste the token, flare-cli validates it and auto-discovers your accounts and zones.

### 2. Verify setup

```bash
flare doctor
```

```
✓ cloudflared: cloudflared version 2024.12.0
✓ docker: 27.4.0
✓ API token: present (40 chars)
✓ account_id: e14b8f06...
✓ zone_id: 27aae3bb...
✓ domain: example.com
✓ state dir: ~/.config/flare-cli
```

### 3. Serve something

```bash
# Create a quick demo page
mkdir demo && echo '<h1>Hello from flare!</h1>' > demo/index.html

# Serve it
flare serve builtin:static --path ./demo --subdomain hello --auth otp
```

Visit `https://hello.example.com` — you'll be prompted for an email OTP before seeing your page.

### 4. Clean up

```bash
flare close <session-id>
```

This stops cloudflared and deletes the tunnel, DNS record, and Access application from Cloudflare.

## Origin types

### `local:http` — Proxy to an existing service

Expose an already-running local HTTP service.

```bash
flare serve local:http --url http://localhost:3000 --subdomain api
```

| Flag | Description |
|------|-------------|
| `--url` | Local origin URL (required) |
| `--health-url` | Override health check URL |
| `--health-path` | HTTP health check path |
| `--wait-for-ready` | Startup timeout (e.g., `60s`) |

### `local:command` — Launch and expose a command

Start a process and expose its HTTP port.

```bash
flare serve local:command --cmd "python -m http.server 8000" --port 8000 --subdomain pyserver
```

| Flag | Description |
|------|-------------|
| `--cmd` | Command to execute (required) |
| `--port` | Port the app listens on (required) |
| `--cwd` | Working directory |
| `--env` | Environment variables (`KEY=VALUE`, repeatable) |

### `builtin:static` — Static file server

Serve static files from a directory with optional SPA routing.

```bash
flare serve builtin:static --path ./dist --subdomain app --spa --cache-control "public, max-age=3600"
```

| Flag | Description |
|------|-------------|
| `--path` | Root directory (required) |
| `--index` | Index file (default: `index.html`) |
| `--spa` | SPA fallback — serves index.html for missing routes |
| `--cache-control` | Cache-Control header value |

### `builtin:file-browser` — File browser

A built-in file browser with upload/download/delete support.

```bash
flare serve builtin:file-browser --path ~/documents --allow-upload --allow-delete --subdomain files
```

| Flag | Description |
|------|-------------|
| `--path` | Root directory (required) |
| `--read-only` | Disable all writes |
| `--allow-upload` | Enable file uploads |
| `--allow-delete` | Enable file deletion |
| `--allow-rename` | Enable file renaming |
| `--show-hidden` | Show dotfiles |
| `--download` | Allow downloads (default: true) |

### `docker:container` — Docker container

Run a Docker container and expose it.

```bash
flare serve docker:container --image nginx:alpine --container-port 80 --subdomain nginx
```

| Flag | Description |
|------|-------------|
| `--image` | Docker image (required) |
| `--container-port` | Container port (required) |
| `--publish-port` | Local bind port (e.g., `127.0.0.1:38080`) |
| `--entrypoint` | Container entrypoint override |
| `--network` | Docker network |
| `--mount` | Bind mounts (`src:dst:mode`, repeatable) |
| `--env` | Environment variables (`KEY=VALUE`, repeatable) |
| `--rm` | Remove container on stop (default: true) |

### `docker:compose-service` — Docker Compose service

Start a Compose service and expose it.

```bash
flare serve docker:compose-service --file docker-compose.yml --service web --container-port 3000 --subdomain app
```

| Flag | Description |
|------|-------------|
| `--file` | Compose file path (required) |
| `--service` | Compose service name (required) |
| `--container-port` | Service port (required) |
| `--project-name` | Compose project name |
| `--build` | Build images before starting |

## Authentication

Every session is protected by Cloudflare Access. Choose an auth mode with `--auth`:

| Mode | Description |
|------|-------------|
| `otp` | Email one-time password (default) |
| `idp` | Identity provider (OIDC/SAML configured in Zero Trust dashboard) |
| `idp+mtls` | Identity provider + mutual TLS client certificate |
| `service-token` | Machine-to-machine via Cloudflare service token |

### Restrict access

```bash
# Only allow specific emails
flare serve ... --auth otp --allow-email alice@company.com --allow-email bob@company.com

# Allow an entire domain
flare serve ... --auth otp --allow-domain company.com

# Update policy on a running session
flare update <session-id> --allow-email add:charlie@company.com
flare update <session-id> --allow-domain remove:contractor.com
```

## Session management

### List sessions

```bash
flare list              # Active sessions only
flare list --all        # Include stopped sessions
flare list --stale      # Sessions that may be leaked
flare list --json       # JSON output for scripting
```

### Check status

```bash
flare status <session-id>
flare status <session-id> --json
```

Session IDs support prefix matching — `flare status a1b2` works if the prefix is unambiguous.

### Update a running session

```bash
flare update <session-id> --ttl 2h                         # Extend lifetime
flare update <session-id> --allow-email add:new@user.com   # Add user
flare update <session-id> --session-duration 4h             # Extend Access session
```

### View logs

```bash
flare logs <session-id>
flare logs <session-id> --source cloudflared --follow
```

### Close a session

```bash
flare close <session-id>                    # Graceful teardown
flare close <session-id> --force            # Force close errored sessions
flare close <session-id> --delete-tunnel=false  # Keep tunnel for reuse
```

## Configuration

### `flare auth login`

The recommended way to configure flare-cli. Opens a browser to create an API token, validates it, and auto-discovers your account and zone:

```bash
flare auth login
```

Or pass a token directly:

```bash
flare auth login --token <your-token>
```

### `flare init` (manual)

For manual configuration or CI environments:

```bash
flare init \
  --account-id YOUR_ACCOUNT_ID \
  --zone-id YOUR_ZONE_ID \
  --domain example.com
```

### Config file

Located at `~/.config/flare-cli/config.yaml`:

```yaml
cloudflare:
  account_id: "your-account-id"
  zone_id: "your-zone-id"
  domain: "example.com"
  api_token_env: "CLOUDFLARE_API_TOKEN"   # env var name to read token from
  team_domain: "yourteam.cloudflareaccess.com"
defaults:
  auth: "otp"
  session_duration: "30m"
  hostname_template: "{app}-{id}.{domain}"
  reuse_tunnel: true
log_level: "info"
paths:
  cloudflared_bin: "cloudflared"
```

### API token priority

flare-cli looks for the API token in this order:

1. `CLOUDFLARE_API_TOKEN` environment variable
2. `cloudflare.api_token` in config file
3. `~/.config/flare-cli/credentials` (written by `flare auth login`)

### Read/write config values

```bash
flare config get cloudflare.account_id
flare config set defaults.auth idp
flare config list
```

## JSON output for agents

All read commands support `--json` for structured output:

```bash
# List sessions as JSON array
flare list --json

# Session details as JSON object
flare status abc123 --json

# Version info
flare version --json

# Login URL for a session
flare auth print-login-url abc123 --json
```

### Agent workflow example

```bash
#!/bin/bash
set -e

# Deploy a preview environment
RESULT=$(flare serve builtin:static \
  --path ./build \
  --subdomain "pr-${PR_NUMBER}" \
  --auth otp \
  --allow-domain company.com \
  --ttl 4h \
  --json)

SESSION_ID=$(echo "$RESULT" | jq -r '.id')
HOSTNAME=$(echo "$RESULT" | jq -r '.hostname')
LOGIN_URL=$(echo "$RESULT" | jq -r '.login_url')

echo "Preview: https://${HOSTNAME}"
echo "Session: ${SESSION_ID}"

# Later: tear down
flare close "$SESSION_ID"
```

## How it works

When you run `flare serve`, the pipeline executes these steps:

```
1. Start origin          → Launch local server / container / proxy
2. Create tunnel         → Cloudflare API: create remotely-managed tunnel
3. Configure ingress     → Set tunnel routing: hostname → origin
4. Create DNS record     → CNAME: subdomain.domain → tunnel
5. Create Access app     → Cloudflare Access: app + allow policy
6. Start cloudflared     → Connect tunnel to Cloudflare edge
7. Session active        → Wait for Ctrl+C or TTL expiry
```

On `flare close` or Ctrl+C, everything is torn down in reverse:

```
1. Stop cloudflared      → SIGTERM → SIGKILL
2. Cleanup connections   → Drain tunnel connections on Cloudflare
3. Delete Access app     → Remove app + policy
4. Delete DNS record     → Remove CNAME
5. Delete tunnel         → Remove tunnel from Cloudflare
```

If any step fails during creation, all prior resources are rolled back automatically.

## Session states

```
initializing → provisioning → active → stopping → stopped
                                  ↓
                                error
```

| State | Description |
|-------|-------------|
| `initializing` | Session created, origin starting |
| `provisioning` | Origin ready, creating tunnel/DNS/Access |
| `active` | Fully operational, serving traffic |
| `stopping` | Teardown in progress |
| `stopped` | All resources cleaned up |
| `error` | Something failed (use `--force` to close) |

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint
```

### Project structure

```
main.go                    → Entrypoint
cmd/                       → Cobra command tree
internal/
  origin/                  → Origin implementations (6 types)
  tunnel/                  → Cloudflare Tunnel API + cloudflared process
  access/                  → Cloudflare Access API
  dns/                     → Cloudflare DNS API
  pipeline/                → Serve orchestration (10-step pipeline)
  session/                 → Session state machine + file store
  config/                  → Viper config + credential storage
  exec/                    → Subprocess helpers
  ui/                      → Terminal formatting (lipgloss)
  testutil/                → Test mocks and helpers
```

## License

MIT
