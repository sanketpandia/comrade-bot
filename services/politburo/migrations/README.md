# Database migrations

Plain PostgreSQL files applied **manually** in filename order. Politburo does
not run migrations at startup.

## Active baseline

`000_core_schema.sql` defines only what the rewrite uses today:

- **Auth:** `api_keys`
- **Identity / membership:** `users`, `virtual_airlines`, `va_user_roles`, `banned_discord_ids`
- **Operator:** `platform_reports`, `user_deletion_archives`
- **Livegame catalog:** `game_aircraft_liveries` (see `001_game_aircraft_liveries.sql`)

Active session and flight snapshots live in **Redis** (jobs). The aircraft livery catalog is in Postgres and synced from the Infinite Flight bulk liveries API.

Apply to an **empty** database:

```sh
# from infra/dev/ (Compose Postgres)
docker compose -f docker-compose.dev.yml exec -T db \
  psql -v ON_ERROR_STOP=1 -1 -U ieuser -d politburo_next \
  < ../../services/politburo/migrations/000_core_schema.sql
```

Create `politburo_next` first if needed:

```sh
docker compose -f docker-compose.dev.yml exec -T db \
  psql -U ieuser -d postgres -c "CREATE DATABASE politburo_next;"
```

Seed an API key after migrate:

```sql
INSERT INTO api_keys (id, status) VALUES ('<uuid>', true);
```

Do **not** apply `000_core_schema.sql` to a database that already has these
objects. Drop and recreate the database when iterating locally.

## Legacy full schema

The old full dump and follow-up ALTER migration are under `archive/`. See
`archive/README.md`. They are optional and not part of the default dev path.

## Adding changes

Add `001_<slug>.sql`, `002_<slug>.sql`, … as transactional deltas.

`003_va_webhook_runs.sql` adds append-only `va_webhook_runs` (dispatch outcome and timing for each feature that entered the webhook worker pool). It does not store Discord webhook URLs, raw config, or feature payloads. There is no purge job in this phase; the table grows with each dispatched run.

`004_contact_submissions.sql` adds append-only `contact_submissions` (public `/contact` form: IFC username + message).

Test on:

1. A fresh DB from `000_core_schema.sql` plus all new files.
2. A copy of an environment that already ran previous migrations.
