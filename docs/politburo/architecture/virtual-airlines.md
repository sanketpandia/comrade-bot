# Virtual airlines: config, live matching, flight index

Everything virtual-airline (VA) owned lives in `internal/community/virtualairlines`
(`Repository`, init `Service`, live matching catalog) and its `config` subpackage
(typed `va_configs` blocks). Neither package imports `livegame` or `webhooks`;
consumers depend on them, never the reverse.

## Config blocks (`va_configs`)

One row per `(va_id, config_type)`, JSON payload (`migrations/002_va_configs.sql`).
Read and upsert through `GET`/`PUT /api/v1/va/config?configType=…` (VA
`administrator` only). The transport layer composes per-type validators in
`transport/http/api/va/handler_config.go`; an unknown type is
`400 UNKNOWN_CONFIG_TYPE`, a payload that fails validation is `400 INVALID_CONFIG`.

| Type | Owner | Parser / validator |
|---|---|---|
| `BASIC` | `virtualairlines/config` | `ParseBasic` / `ValidateBasic` |
| `WEBHOOKS` | `webhooks/config` | `Parse` / `Validate` (see [webhooks.md](webhooks.md)) |

`BASIC` shape:

```json
{
  "game_callsign": { "prefix": "TST", "suffix": "VA" },
  "preferred_servers": ["expert", "training"]
}
```

`ParseBasic` is the single typed reader used by validation and every loader:
affixes are trimmed; `preferred_servers` are lowercased, trimmed, de-duplicated,
and sorted, and an empty entry is rejected.

Loaders are repository queries on `vaconfig.Repository`, joined to active VAs:

- `ListActiveBasic` — BASIC rows for every active VA (invalid rows are logged and skipped).
- `ListActiveWebhooks` / `GetActiveWebhooks(vaID)` — WEBHOOKS rows plus the VA's
  BASIC `preferred_servers` (`ErrNotFound` when the VA has no WEBHOOKS row or is inactive).

## Live callsign matching

The flights job (`livegame/jobs/flights`) loads `ListActiveBasic` once per run
and builds a `virtualairlines.LiveVACatalog`. For each flight it passes the raw
callsign, the flights-side normalized callsign
(`flights.NormalizedCallsignForAffix`), and the session's normalized server name
to `LiveVACatalog.Match`, and stores the resulting
`[]virtualairlines.VirtualAirlineMatch` on the cached flight record.

Matching rules:

- Prefix: case-sensitive `HasPrefix` on the raw callsign.
- Suffix: case-sensitive `HasSuffix` on the raw **or** normalized callsign.
- A VA with no `preferred_servers` matches on every server; otherwise only on
  those servers.
- If the catalog cannot be loaded the run continues with an empty catalog
  (no matches) rather than failing the flights sync.

## VA flight index (`livegame/jobs/vaflightindex`)

Job `infinite-flight-va-flight-index`, schedule `cache.ScheduleVAFlightIndexSync`
(second 30 of every minute, 30s after the flights sync). It reads only Redis
snapshots/records plus `virtualairlines.Repository.ListActive`, and writes:

| Key | Value |
|---|---|
| `cache.KeyVACodeIndex` | `{ "<UPPERCASE CODE>": "<va id>" }` for every active VA |
| `cache.KeyVAActiveFlights(vaID)` | sorted active flight IDs (empty list when none) for every active VA |

Both use `cache.ActiveFlightsTTL`. Because every active VA is rewritten each run,
lists never go stale when a VA stops matching. The gauge
`politburo_livegame_va_active_flights{va_id}` is reset and set for every active VA
each run (0 when none), so deactivated VAs drop out.

## `vaCode` filter on `/api/v1/game/flights/active`

Cache-only, like every game endpoint: the reader uppercases the code, resolves it
through `KeyVACodeIndex`, then intersects with `KeyVAActiveFlights(vaID)`.

| Situation | Response |
|---|---|
| Code index or VA list not cached yet | `503` |
| Code is not an active VA | `400` |
| Redis read error | `500` |
