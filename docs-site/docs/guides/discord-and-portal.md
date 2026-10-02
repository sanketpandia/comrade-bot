# Discord & portal

Discord is the default place for **short, in-game actions**: register, status, opening a map
link, or kicking off a flow with buttons and modals.

The **Politburo portal** (web) is for **longer work**: tables, maps, configuration wizards,
and staff pages that need more space than a Discord embed.

## How you get from Discord to the web

Many commands generate a **signed link**: a time-limited URL that logs you into the portal
without a separate password. Links are tied to your Discord identity. Do not share them.

Typical flow:

1. Run a command such as `/dashboard` or `/live` (when enabled for your VA).
2. Comrade Bot asks Politburo for a signed URL.
3. You open the link in a browser; Politburo sets a session cookie for that visit.

Sessions are meant to reflect your **current Discord server’s VA** and your **role** there as
those features finish rolling out.

## What runs where

| Job | Discord | Portal |
|-----|---------|--------|
| Register / prove IFC | `/register` | — |
| Join VA on this server | `/register` (link step) | — |
| Account status | `/status` | Session details when logged in |
| Create / bind VA | `/initserver` | Further VA settings |
| Role assignment, admin transfer | — | Administrator |
| Staff logbook / events | — | Bourgeoisie or administrator |
| Datasource & matching config | — | Administrator |

## API (for bots and tools)

Integrations use JSON under `/api/v1/...` with an **API key** and, for bot calls, Discord
context headers. Full detail: [API reference](../api/reference/index.html).
