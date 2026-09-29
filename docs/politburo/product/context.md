# Product context

Infinite Experiment is a platform for **Infinite Flight virtual airlines**.
It sits between three worlds that VAs already live in:

1. Infinite Flight's live multiplayer game.
2. The VA's own records (pilots, hours, PIREPs, routes) — today usually Airtable.
3. The VA's Discord community, where pilots actually show up.

We do not replace the VA's system of record. We connect it, add live-game
awareness, and host the workflows that do not belong in Airtable: identity,
membership, Discord linkage, configuration, and features we invent (group
events, search summaries, coordination).

VAs differ. Callsign rules, PIREP fields, ranks, career-mode vs regular ops,
and event formats are configuration, not product defaults.

## Users

| Segment | Role name | Jobs |
|---|---|---|
| Pilots | proletariat | Prove who they are, join a VA, see who is flying, file a PIREP, find events, open a signed web link when Discord is too small |
| Managers | bourgeoisie | Staff work: logbook, events, change proletariat callsigns, remove proletariat. Cannot assign roles or hold the unique administrator seat |
| VA administrator | administrator | Exactly one per VA. Datasource, matching configs, role assignment, transfer of the seat |
| Platform operators | (not a VA role) | Bans, IFC occupancy reports, infra. See [identity-membership.md](identity-membership.md) |

Discord is the default surface for one-step pilot actions. The SSR portal is
the default surface for multi-step admin work and anything that needs a map,
a table, or a wizard. JSON endpoints are the shared machine plane for both.

## Repos

| Repo | Owns |
|---|---|
| `politburo/` | Backend, jobs, OpenAPI, SSR portal. Shared truth and business rules. |
| `comrade-bot/` | Discord commands, modals, buttons, notifications. HTTP client of Politburo. |
| `labour-bureau/` | Local Compose, prod Podman, deploy scripts, scrape configs. No product features. |

Comrade Bot must not call Infinite Flight or Airtable directly. Politburo
owns those integrations.

## Connection planes

### SSR UI (`/dashboard`, `/maps`, `/auth`)

Admin portal and heavier pilot views. Not in OpenAPI. Browser sessions are
Redis objects (`session:{id}`) minted from a one-time signed link.

Intended work here: VA config, datasource mapping, member management, group
event planning/coordination, live maps, staff lookup.

### JSON API (`/api/v1`)

Contract: `api/openapi/politburo.yaml`. Used by the Discord bot and by the
SSR UI whenever the browser needs JSON (maps already do this).

Two caller shapes:

- Bot: `X-API-Key` plus `X-Discord-User-Id` / `X-Discord-Server-Id`.
- Browser: session cookie on game read paths; other JSON still needs a key
  until we decide which portal APIs are cookie-authed.

Health and metrics sit outside product auth (`/health/*`, `/metrics`).

## Naming

"Vizburo" was the old name for the SSR UI, including when it was a separate
binary. The rewrite is one Politburo process. Prefer "portal" or "dashboard"
in new writing. The bot still says Vizburo in `/dashboard` copy.
