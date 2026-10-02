# Prometheus metrics

Politburo exports **performance-oriented** application metrics from `/metrics`.
Metric labels are limited to HTTP routes, known cache operations/outcomes, and
registered job names; cache keys, error messages, and feature-specific business
values are deliberately not labels.

Feature-level gauges (session counts, user totals, and similar) are added only
when explicitly requested. Job freshness is observable via
`jobs_last_success_timestamp_seconds` and cache miss/error rates.

## HTTP

- `politburo_http_requests_total{method,route,status}` counts requests.
- `politburo_http_request_duration_seconds{method,route}` measures latency.

## Cache

- `politburo_cache_operations_total{operation,outcome}` counts `get`, `set`,
  and `ping` outcomes. Get outcomes are `hit`, `miss`, or `error`; set and ping
  outcomes are `success` or `error`.
- `politburo_cache_operation_duration_seconds{operation}` measures Redis and
  JSON processing latency.
- `politburo_cache_payload_bytes{operation}` measures encoded payload sizes.
- `politburo_cache_inserts_total` counts successful cache writes.

Prometheus calculates rates from counters. Successful inserts during each
one-minute window:

```promql
increase(politburo_cache_inserts_total[1m])
```

Cache miss ratio over five minutes:

```promql
sum(rate(politburo_cache_operations_total{operation="get",outcome="miss"}[5m]))
/
sum(rate(politburo_cache_operations_total{operation="get"}[5m]))
```

## Scheduled jobs

- `politburo_jobs_runs_total{job,outcome}` counts successful and failed runs.
- `politburo_jobs_run_duration_seconds{job}` measures execution time.
- `politburo_jobs_running{job}` is `1` while the job runs and `0` otherwise.
- `politburo_jobs_last_success_timestamp_seconds{job}` records the most recent
  successful completion time.

Runs per minute for the Infinite Flight sessions job:

```promql
increase(politburo_jobs_runs_total{job="infinite-flight-sessions"}[1m])
```

Seconds since its last success:

```promql
time() - politburo_jobs_last_success_timestamp_seconds{job="infinite-flight-sessions"}
```

## Live game (Infinite Flight)

Grafana dashboard **Infinite Live metrics** (`politburo-livegame`).

- `politburo_livegame_flights_active{server}` — active flights in the latest
  cached snapshot per normalized server name.
- `politburo_livegame_flights_by_pilot_state{server,pilot_state}` — active
  flights in the latest upstream poll grouped by pilot state.
- `politburo_livegame_livery_resolve_total{outcome}` — livery catalog match
  outcomes while mapping flights (`livery`, `aircraft_only`, `miss`).
- `politburo_livegame_flights_filtered_total{server,endpoint}` — flights
  returned after HTTP query filters (`endpoint` is `active` or `trimmed`).

Livery resolves per 15 minutes:

```promql
sum by (outcome) (increase(politburo_livegame_livery_resolve_total[15m]))
```
