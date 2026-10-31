# Flight plans

Origin and destination have to exist before a flight select can be labeled
and before a mode can match a route. The old bot read the plan at command
time and kept only the first and last waypoint.

Depends on the Infinite Flight client, which already fetches a plan, and the
Redis flight record, which already has path-sync timestamps.

## Strategy

Finish the job that already exists. `FPLSyncDue` marks active flights every
five minutes, then `RunFPLSync` does nothing. Implement that call with the
Infinite Flight client’s `GetFlightPlan` and store a small summary on the
Redis flight record: origin, destination, route label, waypoint count,
synced time. Do not keep plan history and do not write a plan back to
Infinite Flight. If `/log` finds a candidate whose summary is empty, fetch
that flight id once. Preferred server comes from `BASIC`, so Expert is no
longer hardcoded.

## Tasks

| Task | What to decide or build |
|---|---|
| Persist summary | On a due active flight, fetch the plan and write origin, destination, and label onto the live record. |
| Stop false sync | `LastFPLSyncAt` advances only after a real fetch, including the empty-plan case. |
| On-demand gap | Expose a single-flight fetch for candidates that have never been synced, used by flight resolution. |
| Expose on the flight | Include the summary on the active-flight payload the bot and portal already read. |
| Bounds | Store the summary only while the flight is live. No flight-plan table. |
