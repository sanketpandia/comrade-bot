# Submit

One request files the PIREP. The bot sends the chosen flight, the chosen
mode, and the typed fields. Politburo applies the mode, fills computed
fields, and writes Airtable.

Depends on [flight modes](01-flight-modes.md),
[find the flight](03-find-the-flight.md),
[Airtable connection](04-airtable-connection.md), and
[aircraft and airline](05-aircraft-and-airline.md).
[Flight plans](02-flight-plans.md) are required for any strategy that
matches origin and destination.

## Strategy

`POST /api/v1/pireps/submit` (name can match the bot’s existing type).
Server re-checks that the flight still belongs to the user and VA,
re-resolves the route with the mode’s strategy, rejects ineligible modes,
applies multiplier to flight time, resolves aircraft and airline through the
mapping table, renders the remarks template, and creates the log row. The
bot never sees column names or record ids. Idempotency can be a short-lived
key of user + flight id + mode so a double submit does not create two rows;
dedupe against Airtable is out of scope because we do not read logs.

## Tasks

| Task | What to decide or build |
|---|---|
| OpenAPI | Config read and submit write. Generate Go and TypeScript. Retire the bot stubs. |
| Eligibility | Evaluate the selected mode against the flight: plan present, route match, pilot pointer present. |
| Compute | Date, mapped aircraft, mapped airline, IFC, callsign, multiplier, remarks template. A mapping miss uses the VA default and records an unmapped sighting. |
| Write | Call the Airtable create from the connection stream. Map our field keys through `config_data`. |
| Errors | Stable codes: not in flight, mode ineligible, route miss, VA not ready, Airtable rejected. |
| Config read | GET returns candidate flights plus modes with eligibility and the field list for the selected flight, so Discord can render without local rules. |
