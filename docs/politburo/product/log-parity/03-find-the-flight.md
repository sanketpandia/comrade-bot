# Find the flight

When someone runs `/log`, return the live flights that belong to them inside
their current VA. Zero flights is an error. One flight continues. Several
flights go back to Discord as a route select.

Depends on the VA index and username/callsign filters, which exist, and on
membership IFC and callsign, which exist. Plan labels need
[flight plans](02-flight-plans.md) for a good select; the endpoint can ship
with a callsign label first.

## Strategy

Use the VA flight index the jobs already write from prefix and suffix.
Intersect it with the user’s Infinite Flight username from the users table.
Fall back to the membership callsign when the username misses. Do not parse
the Discord display name. Apply `preferred_servers` from `BASIC`. Return
each hit with flight id, callsign, aircraft, livery, and the plan’s
origin–destination. The bot renders that list; Politburo does not pick the
lowest `pilotState` the way the old helper did.

## Tasks

| Task | What to decide or build |
|---|---|
| My-flights query | New JSON route: current user + current VA → zero or more live flights. |
| Match rules | Username first, membership callsign second, VA index required, preferred server applied. |
| Candidate shape | Flight id, callsign, aircraft, livery, origin, destination, plan-synced flag. |
| Empty and many | Empty returns a stable error the bot already phrases as not in flight. Many returns the list with no server-side pick. |
| Auth | Discord bot context headers, scoped to the guild’s VA. A user with no membership cannot file. |
