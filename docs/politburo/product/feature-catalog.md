# Feature catalog

Planning board. Status means **this workspace**, not production of the old
app.

| Status | Meaning |
|---|---|
| **Rewrite** | Implemented in the current Politburo binary |
| **Bot-only** | Discord still implements it; rewrite has no matching API |
| **Leftover** | Code or tables exist; not wired or we do not want the old shape |
| **Open** | To decide in this discussion |

Surfaces: **Discord**, **Portal** (SSR), **JSON** (shared), **Jobs**.

---

## Platform / identity

Decided in [identity-membership.md](identity-membership.md). Old
`pilot` / `staff` / `admin` names are replaced by **proletariat** /
**bourgeoisie** / **administrator**. One administrator per VA, transferable.

| Feature | Surfaces | Status | Notes |
|---|---|---|---|
| API-key auth for machines | JSON | Rewrite | Redis-cached 1 minute |
| Discord context headers | JSON | Rewrite (copied, unused except signed-link) | `RequireDiscordBotContext` not mounted |
| Browser session from signed link | Discord → JSON → Portal | Rewrite | Must later carry current VA + role |
| Global user: Discord ↔ IFC | Discord, JSON | Decided / Bot-only | `/register`; any of last 3 logbook flights as proof |
| Link existing user to current VA | Discord | Decided / Bot-only | Same `/register` command; leftover `/membership` stays dropped |
| Account / membership status | Discord | Bot-only | `GET /api/v1/user/status` |
| Bind Discord guild to VA | Discord, then Portal | Decided / Bot-only | One guild per VA; creator becomes the sole administrator |
| Guild migration | Report → operator | Decided | Operator rebinds `discord_server_id`; not self-service |
| Role model | JSON, Portal | Decided | proletariat (default), bourgeoisie (callsigns + remove proletariat), unique administrator |
| Administrator transfer | Portal | Decided | Atomic; outgoing admin becomes bourgeoisie |
| VA matching configs | Portal | Decided | Prefixes, suffixes, preferred IF server, datasource |
| Occupied-IFC report | Discord → operator | Decided | Warn, report, operator bans Discord ID and deletes the user |
| Ban delete | Operator | Decided | Export archive, then vanish memberships/callsigns from every VA |
| Banned Discord list | Postgres, operator | Decided | Blocks re-registration |
| God-mode command rollout | Discord | Bot-only | Operator tool, not a VA feature |
| Rate limits on register/submit | JSON | Leftover | Implemented, unwired; still needed for register + reports |
| Users.`otp` | Postgres | Leftover | Drop when the user table is touched |

---

## Live Infinite Flight

| Feature | Surfaces | Status | Notes |
|---|---|---|---|
| Active sessions list | JSON, Portal | Rewrite | Cache-backed, optional history |
| Active flights (filter, page) | JSON | Rewrite | `serverId`, pilot state, username, callsign |
| Trimmed map snapshot + detail token | JSON, Portal | Rewrite | `/maps/flights/active` |
| Per-flight Redis history | Jobs / Redis | Rewrite (stored) | Not exposed as its own API |
| Livery name cache | Jobs / Redis | Rewrite | Feeds flight payloads |
| VA-filtered live flights (`/live`) | Discord | Bot-only | Old `GET /api/v1/flights/va` — filter by VA callsign / VO |
| Live flight plans / waypoints | Jobs, Portal | Leftover | `PathSync` stub; map can draw routes; IF route API not in spec |
| Flight phase (climb/cruise/…) | Discord, old Portal | Bot-only | Rewrite exposes IF `pilotState`, not derived phase |
| Origin / destination on live flights | Discord, old Portal | Bot-only | Needs FPL or world/route data |
| Periodic Discord webhook snapshots | Jobs, Portal | Leftover | `va_webhooks` table idle |

**Open:** VA live view is a product feature, not a new data source — it is
our membership list applied to the Redis flight snapshot. Likely an early
JSON endpoint the bot can share with the portal.

**Open:** Flight plans: worth a dedicated job, or only fetch on map click?

---

## VA records (Airtable mapper)

Decided in [airtable-datasource.md](airtable-datasource.md). Wizard:
credentials → pilots → routes → logs. Mapping JSON in Postgres. Per-module
sync flags. Links resolved inside Politburo.

| Feature | Surfaces | Status | Notes |
|---|---|---|---|
| Credentials + test connection | Portal | Decided | API key + base id |
| Table preview (one row, types, 15 min Redis) | Portal | Decided | Load-columns CTA per module |
| Field mapping dropdowns + sample value | Portal | Decided | Our fields → their columns |
| Link fields (logs → pilots/routes) | Portal, mapper | Decided | Dropdowns filtered to the linked table; ids never shown |
| Mapping JSON in `config_data` | Postgres | Decided | Versioned, expandable |
| Pilots sync | Membership | Decided | Required IFC column; lookup on join; store Airtable id |
| Routes enabled | Portal Refresh now | Decided | 5 min rate limit; arr/dep/duration/type/aircraft + id |
| Logs enabled | Discord `/log` | Decided | Write-only mappings; no inbound log job or staff AT logbook |
| Roster join against VA DB | Discord | Decided | IFC column only; not in roster → cannot join |
| Pilot stats from VA + IF | Discord, old Portal | Bot-only | Needs pilots sync pointer |
| PIREP modes and form fields | Discord, old Portal admin | Bot-only | Separate from this mapper slice |
| File PIREP into VA DB | Discord | Bot-only | Writes every mapped log field + stored ids |
| Staff VA logbook (Airtable PIREPs) | — | Parked | No log reads in this slice |
| Livery / airline name mappings | Portal | Leftover | Separate from Airtable tables |
| `*_at_synced` warehouses | Postgres | Leftover | Do not rebuild |
| RDBMS VA providers | Mapper | Open | After Airtable |
| Career-mode table | Portal | Parked | Fourth module later; not designed |

---

## Our features (not the VA's database)

| Feature | Surfaces | Status | Notes |
|---|---|---|---|
| Group events / tours | Discord, Portal, JSON | Bot-only + schema | Tables `events` / `event_legs` are ours. Portal CRUD is gone. Bot list/tour still calls old API. |
| Event coordination (planning, sign-up, who is flying the leg) | Portal, Discord | Open | Stated direction for the new portal; old product was admin CRUD + Discord list |
| Career-mode as a PIREP mode | Discord | Parked | Fourth Airtable module later; not designed |
| Search over our summaries | JSON, Portal | Open | Only if we define summaries |
| Airports reference | Postgres | Leftover | Unused `airports` table |

---

## Old portal pages (from `UI_FEATURE_AUDIT.md`)

None of these routes exist on the rewrite except the stub dashboard and the
new live map.

| Old page | Role | Bring back? |
|---|---|---|
| Dashboard home (IF + VA + career stats, event leaderboard) | Pilot+ | Open |
| Live flights split-pane | Pilot+ | Partially replaced by `/maps/flights/active` (all IF traffic, not VA-filtered) |
| IF logbook map | Staff | Open — this is IF user flights, not Airtable |
| VA admin hub | Admin | Open |
| Manage pilots (callsign, role, remove) | Admin | Open — our membership table |
| Flight-mode editor | Admin | Open — needed before `/log` can return |
| Webhooks | Admin | Open |
| Events CRUD + route autocomplete | Admin | Open — core of "group events" |
| Datasource wizard | Admin | Decided — see [airtable-datasource.md](airtable-datasource.md) |
| Livery mappings | Admin | Open |

---

## Suggested discussion order

Not a commitment — a way to walk the board:

1. **Identity and membership** — without this, signed-link is a dead end and
   no VA feature can authorize.
2. **Airtable mapper (read + PIREP write)** — without this, `/stats` and
   `/log` cannot return.
3. **VA live flights** — cheap once membership exists; uses data we already
   cache.
4. **Portal admin: datasource + members + flight modes** — unblocks Discord.
5. **Group events** — our feature, already has tables and bot commands.
6. **IF enrichment** — user stats, user flights/logbook, live FPL.
7. **What we explicitly drop** — Airtable copies, `/membership`, webhooks,
   airports, Postgres livery table, `otp`.

Add decisions as status changes in this file.
