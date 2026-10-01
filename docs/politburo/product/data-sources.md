# Data sources

Four sources. Politburo is the only process that talks to all of them.

## 1. VA database (currently Airtable)

The VA's system of record: pilots, logged flights / PIREPs, hours, ranks,
routes, career-mode assignments, and whatever else that VA invented.

**Mapper plus named summaries.** Airtable adapter first; common RDBMS later.
We do **not** warehouse the VA base. We do persist:

- Mapping JSON and credentials (`va_data_provider_configs`)
- Pilot **record ids** on memberships when pilots sync is on
- A small **route summary** (id, origin, destination, duration, type,
  aircraft) when routes are enabled, filled by administrator **Refresh now**
  (5 min rate limit)

Logs/PIREPs stay in Airtable; we write through the mapper. Wizard previews
(one sample row + column types) live in Redis for ~15 minutes only.

See [airtable-datasource.md](airtable-datasource.md).

Schema details of a VA's Airtable must not leak into pilot-facing UX.

## 2. Our Postgres (`politburo_next`)

Things that are ours, not the VA's:

| Kind | Examples |
|---|---|
| Identity | `users` (Discord ↔ IF community / IF API id) |
| Membership | `virtual_airlines`, `va_user_roles` (callsign, role, Airtable record id as a *pointer*) |
| Auth | `api_keys` |
| Connections | `va_data_provider_configs`, validation history, `va_configs`, `va_webhooks` |
| Our features | `events`, `event_legs` (group events / tours we coordinate) |
| Mappings we own | `livery_airtable_mappings` (IF name → VA label) |
| Optional summaries | Search indexes or rollups **we** define, not a dump of Airtable rows |

Inherited tables that look like Airtable copies (`pilot_at_synced`,
`pirep_at_synced`, `route_at_synced`) are from the previous product. Do not
re-adopt them as a warehouse. Routes sync gets an explicit summary (see
[airtable-datasource.md](airtable-datasource.md)); pilots get a pointer on
the membership; PIREPs are not copied.

`aircraft_liveries` and `airports` also exist in the baseline. The rewrite
already caches liveries in Redis from the live API. Airports are unused.
Keep or drop is a catalog decision, not a data-source decision.

## 3. In-cluster Redis

Operational cache and session store. Not a system of record.

| Prefix | Contents |
|---|---|
| `game:sessions:*` | Active Infinite Flight sessions |
| `game:flights:active:*` | Per-server live flights |
| `game:flights:history:*` | Per-flight recent snapshots |
| `game:livery:*` | Aircraft / livery names |
| `auth:api_key:*` | API-key status (1 minute) |
| `auth:login_ticket:*` | One-time signed-link tickets |
| `provider:airtable:*` | Wizard preview (schema + one sample row), ~15 min TTL |

Jobs populate game keys. Cache-backed handlers **do not** hit Infinite Flight
or Postgres on a miss; they return unavailable. See `../conventions.md`.

Precompiled payloads (trimmed map snapshots, filter metadata) belong here.

## 4. Infinite Flight Live API

Upstream contract: `api/openapi/infinite-flight/`. Generated client under
`internal/api/generated/infiniteflight/`; handwritten wrapper
`internal/livegame/infiniteflight`.

Currently specified and used:

| Operation | Job cadence | Used for |
|---|---|---|
| `GET /sessions` | 5 minutes | Server list, map server picker |
| `GET /sessions/{id}/flights` | 1 minute | Live map / live list |
| `GET /aircraft/liveries` | 1 hour | Aircraft and livery names |

Not yet in our IF spec, but the previous product used them (and the map still
has route-drawing code):

- User lookup / user stats (registration proof, `/stats` game block, logbook)
- User flights / flight detail / waypoints — registration now needs **the
  last three** flights so any of those routes can prove IFC ownership
- Session flight plan / route for a live flight
- Possibly ATC / world status later

Live API data is temporary operational cache. Do not warehouse raw responses.
Do not present it as real-world aviation data.

## Ownership rule

When designing a feature, name the source of each field:

- IF live → Redis, then JSON/UI
- VA record → mapper, then response (no copy unless we explicitly add a summary)
- Our identity / config / events → Postgres
- Session / ticket → Redis

If a field does not have a source, it is not a field yet.
