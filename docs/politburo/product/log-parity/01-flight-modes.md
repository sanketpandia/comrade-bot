# Flight modes

Replace the hardcoded branches in `acars_pirep`, `wt_pirep`, and `cm_pirep`
with one per-VA document. Classic, ROTW, World Tour, Career, and any future
mode are data, so a second org can file without a new command.

Depends on nothing. This contract is what the other streams implement.

## Strategy

Add a `FLIGHT_MODES` row on `va_configs`, same storage as `BASIC`. The
document lists modes. Each mode names a route strategy, a multiplier, which
form fields the pilot types, a remarks template, the string written as
flight mode, and which pilot module the link uses. It never names Airtable
columns. Closed route strategies cover the old branches:

- `match_od` — World Tour rejects a miss; Classic binds a hit
- `match_od_else_select` — offer specials when the plan misses
- `fixed` — ROTW and Live always use one configured route
- `select` — pilot picks a special
- `none` — no route link

Multiplier lives on the mode and can be overridden by the chosen special.
Asked fields are a closed set: `flight_time`, `remarks`, `fuel_kg`,
`cargo_kg`, `passengers`. Computed values (aircraft, airline, date, IFC,
callsign, origin, destination) are never asked. Aircraft and airline strings
written to Airtable come from the livery-mapping table, not from this
document.

## Tasks

| Task | What to decide or build |
|---|---|
| Schema | Define the `FLIGHT_MODES` JSON, validation, and a config type next to `BASIC`. Unknown keys ignored so the document can grow. |
| Route strategies | Specify `match_od`, `match_od_else_select`, `fixed`, `select`, `none`, including on-miss behavior and where the multiplier is read. |
| Form contract | List allowed input fields, required flags, and the five-field Discord modal cap. Career’s four prompts fit; a sixth field is a later multi-step, not a new command. |
| Pilot module pointer | A mode references pilots by default. A career mode may reference a later `career_pilots` module. Do not design that module in this task; only the pointer. |
| Seed AFKLM | Translate today’s `flight_modes`, `special_routes`, World Tour 9, and Career Mode into one example document so parity is checkable. |
| Admin API | GET/PUT the block the same way as `BASIC`. Portal editor can follow after `/log` works against the JSON. |
