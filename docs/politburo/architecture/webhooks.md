# VA Discord webhooks

Periodic Discord posts configured per virtual airline through the `WEBHOOKS`
config block. Code lives in `internal/webhooks`; it consumes
`virtualairlines/config` (loading) and `livegame` readers (data), and nothing in
`virtualairlines` depends on it.

| Package | Role |
|---|---|
| `webhooks` | `NewJob` (single constructor used by `jobs.Register` and `app.New`), `NewRegistry` (feature handlers) |
| `webhooks/config` | `Validate` / `Parse` the WEBHOOKS document, `Due` frequency gate |
| `webhooks/jobs` | `Job.Run` (scheduled), `Job.Trigger` (manual), `Job.DispatchOne` |
| `webhooks/dispatch` | Feature registry, bounded worker pool (`DefaultWorkers = 8`), outcomes |
| `webhooks/features/activeflights` | `active_flights` feature |
| `webhooks/discord` | Embed poster; never logs or returns the webhook URL |
| `webhooks/runlog` | `va_webhook_runs` audit rows (`migrations/003_va_webhook_runs.sql`) |

## Config

```json
{
  "active_flights": {
    "url": "https://discord.com/api/webhooks/{id}/{token}",
    "frequency": "30m",
    "filters": { "pilotState": ["active"], "vaCode": "TST" }
  }
}
```

- Keys are feature ids. Unknown keys must be objects and are stored unvalidated,
  so a config can be saved before a build knows the feature; dispatch skips them
  with `unknown_feature`.
- `url` must be an HTTPS `discord.com`/`discordapp.com` `/api/webhooks/{id}/{token}` URL.
- `frequency` is `30m`, `60m`, or `120m`.
- `active_flights.filters`: optional `pilotState` (known pilot state names) and
  `vaCode` (any VA code; resolved through the VA flight index at dispatch, see
  [virtual-airlines.md](virtual-airlines.md)).

## Scheduling

Job `infinite-flight-webhooks`, schedule `cache.ScheduleWebhooksSync` (minute 0
and 30). Each tick loads `ListActiveWebhooks`, keeps entries whose frequency is
`Due` at the tick's UTC time (`30m` every tick, `60m` at :00, `120m` at :00 of
even hours), and dispatches them through the worker pool. Invalid stored
documents are logged and skipped.

## Manual trigger

`POST /api/v1/va/webhook/trigger` with `{"webhookType":"active_flights"}`
(VA `administrator`). Calls `Job.Trigger`, which ignores frequency but otherwise
shares the handler, metrics, and run log with scheduled runs. `404` when the VA
has no WEBHOOKS config or no entry for the type; `500` if dispatch or run
logging fails. A `200` body carries the outcome (`success`, `skipped`, `error`).

## `active_flights`

Searches the VA's BASIC `preferred_servers` that currently have an active
session (skipped with `no_preferred_servers` when none are set), applies the
filters, and posts a summary embed plus flight pages (25 per embed, 100 flights
total; the summary notes when the cap applies). Data comes from the flights and
sessions cache readers only.

## Observability

- Job metrics: `politburo_jobs_*{job="infinite-flight-webhooks"}`.
- `politburo_webhooks_dispatch_total{webhook_type,outcome}`,
  `politburo_webhooks_dispatch_duration_seconds{webhook_type}`,
  `politburo_webhooks_discord_posts_total{webhook_type,outcome}`.
- `active_flights`: `politburo_webhooks_active_flights_servers_searched`,
  `_matched_total`, `_reported_total`, `_capped_total` (all `{webhook_type}`).
- Grafana dashboard `politburo-webhooks` (dev and prod copies are identical).
- `va_webhook_runs` stores one row per dispatch (outcome, skip reason, message
  count, error, timing); URLs and payloads are never persisted. No purge job yet.
