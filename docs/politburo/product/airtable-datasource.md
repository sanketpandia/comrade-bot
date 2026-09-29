# Airtable datasource

Administrator portal wizard for connecting a VA’s Airtable base. Pilots never
see table names, record ids, or link fields. Bourgeoisie do not configure
this; administrator only.

Airtable remains the system of record. We store **pointers and small
summaries we cannot function without**, not a copy of the base. Mapping JSON
is ours and is allowed to change shape without a migration.

## Wizard shape

Fixed order. Later steps depend on earlier tables so links have a target.

```text
1. Credentials   API key + base id → Test connection
2. Pilots        table → fetch preview → map fields → save
3. Routes        table → fetch preview → map fields → save
4. Logs          table → fetch preview → map fields (including links) → save
5. Sync flags    per module on/off, with the features each flag unlocks
```

Skip is allowed (a VA with no published routes can still map pilots). You
cannot map a Logs **link** to Pilots or Routes until that module has a saved
table. Career mode is a **fourth module later**; parked, not designed here.

### 1. Credentials

Ask for:

- Airtable API key (personal access token)
- Base id (the `app…` id; UI copy can say “base”, not “space/workspace”)

**Test connection** must succeed before any table step. Failure is a single
sentence (invalid key, base not found, no access). Do not dump Airtable
error bodies.

Save credentials in `va_data_provider_configs` (encrypted at rest when we
implement storage). Do not put the key in Redis.

### 2–4. Each table step

1. Administrator enters the **table name** (or picks from the base’s table
   list if the test-connection payload already has it).
2. **Test / load columns** CTA.
3. We fetch **one record** plus field metadata, infer column types, and
   cache that preview in Redis for **15 minutes**.
4. The page shows **our fields** on the left, **their columns** as dropdowns
   on the right. Compatible types only. Choosing a column shows the **sample
   value** from the cached row.
5. Save writes a JSON mapping into `config_data`. Preview cache can expire;
   the mapping does not.

If the preview cache expires mid-wizard, the CTA loads it again. Do not
block save if they already mapped from a live preview in this session;
re-fetch if they hit load again.

## Preview cache (Redis)

Wizard-only. Short TTL. Not a second copy of the VA.

Suggested key: `provider:airtable:{vaId}:preview:{module}` → JSON of
`table`, `fields[]` (name, type, linked table id if any), `sample` (one row
of display values). TTL ~15 minutes.

Jobs and live reads do **not** use this cache. They use the saved mapping
and talk to Airtable (or our summaries) directly.

### Types, internally

Inferring from a single row is enough for text and numbers. It is a bad
way to learn **links**: the sample cell may be empty, and Airtable links
are record ids, not names.

Internally, prefer the Airtable **base schema** (table list, field types,
linked-table ids) plus one record for samples. The administrator still
sees one action: “Load columns.”

## Mapping UI

Each module has a small, named set of Politburo fields. Required vs
optional is visible. Extra Airtable columns are ignored until we add a
field to this list (JSON expansion).

Starter fields. Logs is a **write** surface: map every Airtable column we
might fill when filing. Reads of the logs table are not a product in this
slice. Extra Airtable columns stay unused until we add a field name.

| Module | Our field | Required | Typical Airtable type |
|---|---|---|---|
| Pilots | IFC username | **Yes** | text |
| Pilots | Callsign | No | text |
| Pilots | Display name | No | text |
| Routes | Origin | Yes if routes enabled | text |
| Routes | Destination | Yes if routes enabled | text |
| Routes | Duration | No | number / duration |
| Routes | Type | No | text / select |
| Routes | Aircraft | No | text / select |
| Logs | Pilot | Yes | **link → Pilots table** |
| Logs | Route | Yes if routes mapped | **link → Routes table** |
| Logs | Flight time | No | number / duration |
| Logs | Remarks | No | text |
| Logs | Aircraft | No | text / select |
| Logs | Livery | No | text / select |
| Logs | Origin | No | text |
| Logs | Destination | No | text |
| Logs | Fuel | No | number |
| Logs | Cargo | No | number |
| Logs | Passengers | No | number |
| Logs | Callsign | No | text |
| Logs | Submitted at | No | date / datetime |
| Logs | Flight mode | No | text / select |

Cannot save the Pilots module without the IFC column. That is how we find
the roster row. Callsign on the pilots table is optional display; live
callsign still lives on **our** membership ([identity-membership.md](identity-membership.md)).

Logs field list will grow in the JSON. Prefer adding optional write fields
over inventing log-read screens.

Dropdowns:

- Text/number fields only list compatible columns.
- Link fields only list columns whose linked table is the one saved for
  that module. Label them as `Pilot (link)` / `Route (link)`, never `recXXX`.
- Sample value for a link: the linked record’s **primary field** (a name or
  callsign), not the raw id.

If they pick a link column that points at some other table: “This column
does not point at your Pilots table.” No mention of table ids.

## Links (keep the mess inside)

Airtable PIREPs usually store Pilot and Route as **linked records**. To
file a log we must write those record ids. To show a map we need origin
and destination from the route record.

The administrator maps “Pilot” → their link column. That is the whole UX.

Politburo then:

1. On VA join, look up the pilots table by the mapped **IFC** column using
   the user’s platform IFC username. If no row, they cannot join (“you are
   not on this VA’s roster”). When pilots sync is on, store
   `airtable_pilot_id` on the membership.
2. When routes are enabled, **Refresh now** loads route summaries (Airtable
   id, origin, destination, duration, type, aircraft). Rate limit: one
   refresh per VA per **5 minutes**. No scheduled routes job in this slice.
3. On `/log`, write a Logs row. Fill every mapped write field we have
   (links, times, remarks, fuel, …). Pilot and Route cells get stored
   record ids when those fields are links.

Nobody in Discord types a record id. If a required write field cannot be
filled, `/log` says the VA is not ready for filing, not “missing rec…”.

Linked cells may allow multiple records. For Pilot and Route we take **one**
id (the first). Document that in admin help; do not build a multi-pilot
PIREP in this slice.

## Saved JSON (`config_data`)

One document per VA, versioned (`config_version` already exists). Shape is
allowed to grow.

```json
{
  "provider": "airtable",
  "baseId": "app…",
  "modules": {
    "pilots": {
      "table": "Pilots",
      "sync": false,
      "fields": {
        "ifc": { "column": "IFC", "type": "singleLineText" }
      }
    },
    "routes": {
      "table": "Routes",
      "sync": false,
      "fields": {
        "origin": { "column": "Dep", "type": "singleLineText" },
        "destination": { "column": "Arr", "type": "singleLineText" },
        "duration": { "column": "Time", "type": "number" },
        "type": { "column": "Type", "type": "singleSelect" },
        "aircraft": { "column": "Aircraft", "type": "singleLineText" }
      }
    },
    "logs": {
      "table": "PIREPs",
      "sync": false,
      "fields": {
        "pilot": {
          "column": "Pilot",
          "type": "multipleRecordLinks",
          "linkedModule": "pilots"
        },
        "route": {
          "column": "Route",
          "type": "multipleRecordLinks",
          "linkedModule": "routes"
        }
      }
    }
  }
}
```

Credentials stay out of this blob if we can (separate secret column or
`*_FILE`). The mapping is not secret; the key is.

## Sync flags and what they unlock

Configured (mapping saved) is not the same as **sync on**. The
administrator gets a flag per module. The portal states which features
turn on, in plain language.

| Flag | What we persist | Features it unlocks |
|---|---|---|
| Pilots sync | Airtable record id on each **registered** membership, found by IFC. Not a dump of the pilots table | Roster join, `/stats` VA block, PIREP pilot link |
| Routes enabled | Small route rows we own: Airtable id, origin, destination, duration, type, aircraft. Filled by **Refresh now** (5 min rate limit), not a cron | Route maps, `/log` route picker, event legs that need a route key |
| Logs enabled | Nothing copied. Mapping is **write-only** for `/log` | Filing a PIREP. Needs pilots sync if Pilot is a link; needs routes enabled if Route is a link |

There is no inbound logs job and no staff Airtable logbook in this slice.

Turning a flag off stops using that module for new work; it does not have
to wipe route summaries immediately. Turning pilots sync off should stop
using stored ids for new PIREPs; stale ids on memberships can stay until
the next successful IFC lookup.

This **replaces** rebuilding `pilot_at_synced` / `pirep_at_synced` /
`route_at_synced` as warehouses. Routes get an explicit summary table (or
a trimmed successor of `route_at_synced` with only those columns). Pilots
get a pointer on membership. Logs stay in Airtable.

## Surfaces

| Job | Surface |
|---|---|
| Credentials, table steps, mapping, flags | Portal, administrator |
| Refresh routes (5 min rate limit) | Portal, administrator |
| Join: IFC → Airtable row → store pilot record id | Discord `/register`, JSON |
| File PIREP (write mapped log fields) | Discord `/log`, JSON |
| Route map / picker | Portal (and JSON for the bot if needed) |

## Parked

Career mode as a **fourth module**. Do not design fields, sync, or Discord
behavior for it until that discussion is opened.

## Open questions

None for this slice. New write fields for Logs are additive JSON, not a
blocker.
