<p align="center">
  <img src="https://img.shields.io/github/license/xieyi138-star/uptix?color=blue" alt="License: MIT">
  <img src="https://img.shields.io/github/stars/xieyi138-star/uptix?style=social" alt="GitHub stars">
  <img src="https://img.shields.io/badge/Docker-30MB-blue?logo=docker" alt="Docker 30MB">
  <img src="https://img.shields.io/badge/RAM-10MB-green" alt="RAM 10MB">
</p>

<h1 align="center">Uptix</h1>

<p align="center">
  <strong>The self-hosted status page that doesn't cost $79/month for a green dot.</strong><br>
  One Go binary. One Docker container. Monitoring + incidents + notifications. MIT license.
</p>

<p align="center">
  <a href="https://uptix-mu.vercel.app/"><strong>Live Demo →</strong></a>
</p>

---

## 😤 The Problem

Here's what every small team hits when they need a simple status page:

| Tool | What you pay | What you actually get |
|------|:-----------:|------|
| **Atlassian Statuspage** | $29–$1,499/mo | A green dot. No monitoring built in. Must use an Atlassian account. Small teams pay enterprise prices. |
| **Cachet** | Free (in theory) | Dead since 2020. v3 has been "coming soon" for three years. PHP 7.4. No commits. |
| **Uptime Kuma** | Free | Great monitoring. No REST API. No incident lifecycle. No subscriber emails. SQLite only. |
| **Gatus** | Free | Super lightweight. No admin UI. No incident management. No notifications. |
| **OneUptime** | Free (open source) | You wanted a status page. You got Kubernetes + APM + traces + logs. 4 GB RAM minimum. |
| **Better Stack** | $29/responder/mo | Nice UI. Pricing grows with your team. 5 people = ~$145/mo. Hosted only — if they're down, your status page is too. |
| **Instatus** | $20 → $300/mo | Pro → Business is a 15× price jump. No middle ground. |

**Every option forces a tradeoff you shouldn't have to make.**

---

## 💡 Uptix: No Tradeoffs

```bash
docker run -d -p 8080:8080 -v uptix-data:/data ghcr.io/xieyi138-star/uptix:latest
```

| | |
|---|---|
| 🪶 **Footprint** | 30 MB Docker image. ~10 MB RAM at idle. Runs on a $5 VPS. |
| 🔍 **Monitoring** | HTTP, HTTPS, TCP, DNS, SSL expiry. 30-second intervals. No external service needed. |
| 📋 **Incidents** | Full lifecycle — Investigating → Identified → Monitoring → Resolved. Auto-created when a check fails. Auto-resolved when it recovers. |
| 📧 **Notifications** | Email, Slack, Discord, webhook. Subscribers get pushed updates. No manual check-ins. |
| 🔌 **REST API** | Full OpenAPI spec. Create incidents from CI/CD. Update status from Terraform. Automate everything. |
| 🎨 **Status page** | Dark mode. Custom logo, domain, CSS. Looks like you paid for it. |
| 💾 **Database** | SQLite by default. Switch to PostgreSQL when you need HA. |
| 🔄 **Maintenance windows** | Schedule recurring windows with cron expressions. Auto-created. Auto-resolved. |
| 🤖 **MCP server** | Let Claude, ChatGPT, or Cursor query your service status directly. Built for the agent era. |
| 📦 **Deployment** | Single binary. Single container. No Kubernetes. No YAML hell. No PhD required. |

---

## 🆚 The "Why Not Just Use..." FAQ

<details>
<summary><strong>Why not Uptime Kuma?</strong></summary>

Uptime Kuma is excellent for monitoring. It's what I used before building Uptix. But it has no REST API (everything is UI-only), no incident lifecycle workflow, no way for subscribers to sign up and get emails when things break, and it's SQLite-only so you can't scale it.

Uptix adds the missing pieces: API, incident management, and subscriber notifications — while staying just as lightweight.
</details>

<details>
<summary><strong>Why not just pay for Atlassian Statuspage?</strong></summary>

If you're a 50+ person team fully on Atlassian, Statuspage makes sense. For everyone else — indie hackers, small startups, hobbyists — $29/mo for a Hobby plan that can't even notify subscribers via webhook is absurd. The Startup plan is $99/mo. For a status page.

Uptix Cloud will cost $12/mo with all features included. Or self-host it for free forever.
</details>

<details>
<summary><strong>Who monitors the monitor?</strong></summary>

Good question. Don't run Uptix on the same server it's monitoring. Put it on a $5 Hetzner/DigitalOcean VPS separate from your main infrastructure. If your app server dies, Uptix keeps running and tells your users what's happening.

That's one reason Uptix is designed to be so lightweight — it runs on the cheapest VPS money can buy.
</details>

---

## 🏗️ Tech Stack

| Layer | Choice |
|------|------|
| **Language** | Go 1.22 |
| **Router** | chi |
| **Database** | SQLite (default) / PostgreSQL |
| **Frontend** | Server-rendered HTML + minimal JS |
| **Container** | Alpine-based, multi-stage build |
| **Deployment** | Single binary or Docker |

No React. No Node.js. No 500 MB `node_modules`. Deliberately boring, deliberately fast.

---

## 📍 Roadmap

- [x] HTTP/TCP/DNS/SSL monitoring
- [x] Incident lifecycle management
- [x] Auto-create incidents on downtime
- [x] Auto-resolve incidents on recovery
- [x] Public status page (dark mode)
- [x] Admin dashboard
- [x] REST API
- [x] Subscriber signup
- [x] Maintenance windows
- [ ] Email notification delivery (SMTP)
- [ ] Slack/Discord webhook delivery
- [ ] PostgreSQL support
- [ ] Custom CSS / custom domain
- [ ] MCP server for AI agents
- [ ] Terraform provider
- [ ] Managed cloud hosting ($12/mo)

---

## ⭐ Star History

If this project resonates, a star helps more people discover it.

---

## 📄 License

MIT — use it, fork it, sell it, deploy it anywhere. No strings attached.

---

<p align="center">
  <a href="https://uptix-mu.vercel.app/"><strong>Live Demo</strong></a> ·
  <a href="https://github.com/xieyi138-star/uptix/issues"><strong>Request a Feature</strong></a> ·
  <a href="https://github.com/xieyi138-star/uptix/discussions"><strong>Discussions</strong></a>
</p>
