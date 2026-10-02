# Inventory

Snapshot of what actually exists in this workspace. The previous Politburo is
not in-tree; the rewrite is a clean-slate Go binary plus an imported schema
and a Discord bot that still speaks the old API.

## Rewrite — running today

Single binary `cmd/politburo`. Jobs off unless `JOBS_ENABLED=true` and
`IF_API_KEY` is set.

### JSON (`api/openapi/politburo.yaml`)

| Path | Status |
|---|---|
| `GET /health/status` | Live |
| `GET /api/v1/game/sessions/active` | Live, Redis |
| `GET /api/v1/game/flights/active` | Live, Redis, filter + page |
| `GET /api/v1/game/flights/active/trimmed` | Live, map markers, encrypted `flightId` |
| `GET /api/v1/game/flights/active/detail` | Live, resolve encrypted id |
| `POST /api/v1/signed-link` | Live, Discord user header, mints `/auth/login?token=` |

Nothing else is on the rewrite contract. Bot calls listed below will 404
against `:8082`.

### SSR UI

| Path | Status |
|---|---|
| `GET /auth/login`, `GET /auth/logout` | Live |
| `GET /dashboard` | Stub landing page |
| `GET /maps/flights/active` | Live map: sessions, filters, Leaflet markers |
| `/static/*` | Embedded CSS/JS |

The map consumes the game JSON APIs with the session cookie. `FlightMap`
still knows how to draw altitude-colored routes; the rewrite never feeds it
waypoints because flight-plan sync is stubbed (`fplSyncRequired` is always
false).

### Jobs

Sessions (5 min), liveries (1 hour), flights (1 min). Central register:
`internal/livegame/jobs/register.go`.

### Postgres usage in code

Only `api_keys` (auth) and `users` (signed-link lookup). The rest of
`migrations/000_core_schema.sql` is the rewrite baseline; legacy tables live in `migrations/archive/`.

### Scaffold present but not mounted

| Piece | Notes |
|---|---|
| Rate limiter | Groups `registration` / `submit` / `read`, ported from legacy |
| `RequireDiscordBotContext` | Needed for register-style bot routes |
| `RequireMember` / `RequireStaff` / `RequireAdmin` | Claims have roles; UI does not gate them yet; signed-link does not resolve a VA |
| Discord context on API-key claims | Headers are copied in; role/`PbServerID` are empty until membership exists |

## Comrade Bot — still the old product surface

Deployed commands (`src/commands/registry.ts`):

| Command | Intended job | Politburo endpoint it calls |
|---|---|---|
| `/register` | Global account + optional VA callsign link | `POST /api/v1/user/register`, `GET /api/v1/user/status` |
| `/status` | Account + current-server membership | `GET /api/v1/user/status` |
| `/botstatus` | Ops health | `GET /health/status` |
| `/log` | File a PIREP for the current flight | `GET /api/v1/pireps/config`, `POST /api/v1/pireps/submit` |
| `/logbook` | Staff lookup of IF flight history | `GET /api/v1/user/{ifcId}/flights` |
| `/live` | VA members currently flying | `GET /api/v1/flights/va` |
| `/stats` | Game + VA + career-mode stats | `GET /api/v1/pilot/stats` |
| `/dashboard` | Signed portal link | `POST /api/v1/signed-link` (shape still old envelope) |
| `/initserver` | Admin: bind Discord guild to a VA code | `POST /api/v1/server/init` |
| `/events` | List active VA events | `GET /api/v1/events?active_only=true` |
| `/tour`, `/tour_leg` | Tour legs + extra data | `GET /api/v1/events/...`, `PUT .../additional-data` |
| `/help` | Local catalog, no API | — |
| `/rollout` | God-mode command redeploy | `GET /api/v1/admin/verify-god` |

`/dashboard` is the only bot command whose rewrite endpoint exists. Request
and response envelopes still differ (old `{status,result}` vs rewrite
`{data:{url,expiresIn,redirectTo}}`).

## Leftover bot code (present, not live)

| Artifact | Why leftover |
|---|---|
| `membership.ts` + join button/modal handlers | `/membership` is **not** in the registry. Join was folded into `/register`. Handlers are not in `InteractionRouter`. |
| `MEMBERSHIP_JOIN_*` constants | Dead IDs unless the command is re-registered. |
| Select-menu router branch | Comment: "pilot role configuration removed." |
| Dynamic modal router | Comment: `SyncUserModal` / `syncUserToVA` removed. |

Keep these in mind when we re-plan membership: either revive a dedicated
command or stay with `/register` as the only join path.

## Inherited schema (idle in the rewrite)

| Table | Previous job | Rewrite stance |
|---|---|---|
| `users` | Discord ↔ IF identity, leftover `otp` | Keep identity; `otp` looks unused |
| `api_keys` | Bot/service keys | Keep |
| `virtual_airlines` | Tenant: Discord guild, code, flight-modes JSON | Keep |
| `va_user_roles` | Membership, callsign, Airtable/career ids, role enum | Keep |
| `va_configs` | Key/value VA settings | Keep as config bag or replace with typed columns — discuss |
| `va_data_provider_configs` | Airtable (later RDBMS) credentials + schema maps + `features_enabled` | Keep as the mapper home |
| `va_provider_validation_history` | Wizard audit | Keep if we keep the wizard |
| `va_sync_history` | Sync bookkeeping | Likely drop if we do not warehouse |
| `va_webhooks` | Periodic Discord live-flight posts | Discuss; feature exists in old UI |
| `events`, `event_legs` | Tours / events we coordinate | Keep as *our* feature tables |
| `livery_airtable_mappings` | IF aircraft/airline names → VA labels | Keep |
| `pilot_at_synced`, `pirep_at_synced`, `route_at_synced` | Copies of Airtable rows | Default: do not re-adopt |
| `aircraft_liveries` | Postgres copy of IF liveries | Redis already owns this |
| `airports` | ICAO/IATA geo | Unused; discuss if maps/events need it |

Enums already in the baseline: `va_role` (`pilot`, `staff`, `admin`),
`pilot_type` (`regular`, `career_mode`), `validation_status`.

## Infinite Flight spec gap

Our IF OpenAPI covers sessions, session flights, and liveries. The old
product also needed user stats, user flights, and live flight plans. Those
callers are not generated yet. Adding them is a spec + client job before any
feature that reads them.

## Historical docs (not rewrite source of truth)

`TECHNICAL_STANDARDS.md` still describes old layout (`cmd/vizburo`,
`internal/runtime`, `/healthCheck`, registration OpenAPI split). Use it to
remember conventions we liked; do not implement against those paths.

`UI_FEATURE_AUDIT.md` (2026-05-13) is a complete tour of the old portal
pages. Useful as a feature checklist for what admins used to be able to do.
The rewrite dashboard is not that app.
