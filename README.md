# Uptix

> The self-hosted status page that doesn't cost $79/month for a green dot.

**Status: Early development. Star the repo to follow along.**

## What is Uptix?

Uptix is a lightweight, self-hosted status page + uptime monitor. It combines monitoring, incident management, and subscriber notifications into a single Go binary that fits in a 30MB Docker image.

## Why

| Tool | The Problem |
|------|-------------|
| **Atlassian Statuspage** | $29–1499/mo. No built-in monitoring. Atlassian account required. |
| **Cachet** | Dead project. v3 has been "almost done" since 2021. |
| **Uptime Kuma** | No REST API. No incident lifecycle. No subscriber emails. SQLite only. |
| **OneUptime** | 4GB RAM minimum. Requires Kubernetes. Overkill for 99% of teams. |
| **Better Stack / Instatus** | Hosted only. Per-seat pricing. Unpredictable bills. |

## What Uptix Does Differently

- 🪶 **Single Go binary** — 30MB Docker image, ~10MB RAM idle
- 🔍 **Built-in monitoring** — HTTP, TCP, DNS, SSL, cron heartbeats
- 📋 **Full incident lifecycle** — Investigating → Identified → Monitoring → Resolved
- 📧 **Subscriber notifications** — Email, webhook, Slack, Discord
- 🔌 **REST API** — Full OpenAPI spec. Create incidents from CI/CD
- 💾 **SQLite or PostgreSQL** — Start simple, scale when needed
- 🎨 **Beautiful default UI** — Dark mode, custom logo, custom domain, custom CSS
- 🤖 **MCP server** — Let AI agents query your service status

## Quick Start

```bash
docker run -d -p 8080:8080 -v uptix-data:/data uptix/uptix:latest
```

Then open `http://localhost:8080`.

## Tech Stack

- **Backend:** Go 1.23 + chi router
- **Database:** SQLite (default) or PostgreSQL
- **Frontend:** HTMX + Alpine.js + Tailwind CSS
- **Deployment:** Single binary or Docker

## Development

```bash
git clone https://github.com/uptix/uptix.git
cd uptix
go run ./cmd/uptix
```

## License

MIT
