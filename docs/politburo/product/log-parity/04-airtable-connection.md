# Airtable connection

Each org points at its own base and column names. Filing writes one log row.
Joining stores one pilot record id. Nothing downloads a table into JSON the
way `magic_time` did.

Depends on the product doc
[airtable-datasource.md](../airtable-datasource.md). The membership column
`airtable_pilot_id` already exists. The provider table is not in the rewrite
migrations yet.

## Strategy

Follow the decided datasource wizard, narrowed to what `/log` needs.
Credentials and mapping live in `va_data_provider_configs`, separate from
`FLIGHT_MODES`. The mode says write `flight_time`; the mapping says
`flight_time` is a column on a table. At register, one filtered lookup by
the IFC column stores `airtable_pilot_id` on the membership. At file time,
one filtered lookup by origin and destination resolves a route record id
when the mode’s strategy needs a link. Special routes in the mode document
may store a record id directly so ROTW does not scan a base. Logs are
create-only. Preview for the wizard is one schema read plus one sample row
in Redis for 15 minutes. Route summary refresh stays a later portal feature
and is not a prerequisite for filing.

## Tasks

| Task | What to decide or build |
|---|---|
| Provider table | Migration for encrypted token, base id, and `config_data` JSON. One provider row per VA. |
| Test connection | Validate token and base. Cache schema preview in Redis. Do not persist rows. |
| Field map | Pilots: IFC required. Logs: pilot link, route link, flight time, remarks, aircraft, livery, fuel, cargo, passengers, callsign, submitted at, flight mode. All optional except the pilot link when logs are links. |
| Pilot pointer | On join, filter the pilots table by IFC and save the record id. Missing row blocks join when pilots sync is on. |
| Route lookup | Filter formula on origin and destination at submit time. Return one id or none. No `route_database.json`. |
| Create log | POST one Airtable record from mapped fields. Failure is a VA-not-ready or Airtable error sentence, never a record id in Discord. |
