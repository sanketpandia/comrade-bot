# Features

Comrade Bot and Politburo are being rebuilt as a single platform. This page describes what
you can rely on today versus what is still on the roadmap.

## Available today

| Area | What you get |
|------|----------------|
| **Identity** | Link your Discord account to your Infinite Flight Community (IFC) username with logbook proof — no IF password. |
| **Registration** | `/register` creates your global account and can link you to the current VA with a callsign. |
| **Live game data** | Cached active sessions and flights from Infinite Flight (JSON API and portal views where wired). |
| **Signed links** | Short-lived browser links from Discord into the Politburo portal. |
| **API access** | Machine clients use an API key; see the [API reference](api/reference/index.html). |

## Discord commands (high level)

| Command | Purpose |
|---------|---------|
| `/register` | Create your account or link to the VA on this server |
| `/initserver` | Bind this Discord guild to a VA (administrators only, after you are registered) |
| `/status` | Check your registration and membership |
| `/help` | Command catalog |

Other commands (live map, stats, dashboard, events) depend on your VA setup and what the
rewrite has wired for your server — ask your VA staff if something is missing.

## Portal (web)

The Politburo portal is for longer workflows: maps, tables, VA configuration, and staff
actions that do not fit a Discord modal. Access is usually through a **signed link** from
Discord. See [Discord & portal](guides/discord-and-portal.md).

## Roadmap (not promised on a date)

- Full role management and administrator transfer in the portal
- VA-filtered live views and events in Discord matching the old app
- Datasource wizards and deeper Airtable integration
- Operator tools for occupied-IFC reports and guild migration

Your VA’s **system of record** (hours, PIREPs, ranks) stays in your own tools; we connect
and orchestrate — we do not replace your spreadsheet or database.
