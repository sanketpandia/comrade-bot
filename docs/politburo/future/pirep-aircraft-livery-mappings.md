# Per-server aircraft and livery mappings for PIREP filing

**Proposed module.** Give each virtual airline (VA) an editor under **Admin → PIREP mapping → Aircraft and airline values**. An admin can pick a game aircraft or livery, enter the value their form accepts, and see the resulting PIREP. Store both dictionaries and their defaults in one dedicated Postgres row per VA. Use the same resolver for the admin preview and the eventual bot filing flow.

## What the old form actually does

The AF/KLM project loads `assets_contents/if_afklm_mappings.json` in `helper_services/utils.js#mapAircraftAirline`. It looks up two raw game names independently and uses each dictionary’s `Other` entry on a miss. The ACARS, world tour and career mode commands call that helper. For example, `commands/acars_pirep.js` writes the results into the form’s **Aircraft** and **Airline** fields. A livery is the input to the airline field. Its paint scheme can be more specific than the value the VA wants to file.

| Legacy input | AF/KLM filing value | Implication for the editor |
|---|---|---|
| Aircraft `Airbus A318` | `A318-100` | Show the game model beside the filing value. |
| Livery `Air France` | `Air France` | Offer “Use game name” for an explicit identity mapping. |
| Liveries `KLM Cityhopper`, `KLM - Orange Pride` | `KLM` | Allow multiple selected names to receive one filing value. |
| Aircraft or livery with no exact key | `Other` for aircraft, `Generic` for airline | Configure each fallback independently. |

### Cross-check against the supplied files

Parsed the CSV by column name and excluded its extra header-like record (`id,aircraftID,aircraftName,liveryName`). The legacy snapshot contains **1,167 distinct livery IDs, 61 aircraft IDs, 61 aircraft names and 576 livery names**. The counts below compare original strings exactly, including case and punctuation. They describe the two files supplied for this plan. A deployed catalog can differ.

| Measure | Aircraft | Livery → airline |
|---|---:|---:|
| JSON keys including the legacy fallback | 54 | 270 |
| Distinct catalog names with an explicit key | 48 of 61 | 207 of 576 |
| Catalog rows with an explicit key | 1,085 of 1,167 | 655 of 1,167 |
| Non-fallback keys absent from this CSV | 5 | 62 |

Only **649 of 1,167** rows match both dictionaries explicitly. That measures catalog coverage, not flight frequency. For example, `Generic` occurs on 47 aircraft/livery rows and `Air France` on 14. A dictionary keyed by livery UUID would make admins repeat a common decision and would miss future aircraft using the same livery name. Conversely, unmatched aliases such as `ERJ-190` and `Air France (2019)` are valid legacy data. Keep them on import and mark them “Absent from catalog.” Never silently drop them or fuzzy-match them to another name.

## Where it fits in Politburo

| Existing component | Integration |
|---|---|
| [Core schema](../../../services/politburo/migrations/000_core_schema.sql) | The unique Discord server ID on `virtual_airlines` resolves to an internal VA UUID. Derive the owner from authenticated VA context on every read and write. |
| [Catalog schema](../../../services/politburo/migrations/001_game_aircraft_liveries.sql) and [repository](../../../services/politburo/internal/livegame/liveries/repository.go) | The sync job upserts upstream IDs and raw names. On conflict it refreshes raw names while preserving display names. Join flight IDs to raw names for lookup. Use display names as optional UI context. The sync currently retains older rows, so catalog presence cannot certify current in-game availability. |
| [Existing PIREP admin form](../../../services/politburo/internal/ui/static/js/admin/feature-pirep.mjs) | The current panel at `/admin/feature-pirep` selects writable fields in `MAP_PIREP`. Add a separately saved panel on the same page. Its catalog and import/export should work before an Airtable connection exists. Label each save/remove action with the settings it affects. |
| [Connection field catalog](../../../services/politburo/internal/community/virtualairlines/connections/catalog.go) | Data connections select the destination table and columns. This module supplies values for logical `pireps.aircraft` and `pireps.airline`, which accept text or single-select fields. Apply the filing projection to determine which resolved values are needed. |
| [Flight observation](../../../services/politburo/internal/livegame/flights/observation.go) and [Discord API client](../../../services/comrade-bot-discord/src/services/apiService.ts) | Observations carry both IDs. The bot’s PIREP config and submission methods currently reject as stubs. Admin editing and local preview can ship independently. Actual filing requires wiring the Politburo filing flow and bot methods. |

## Admin experience

1. **Browse and edit.** Show the active VA name. Provide Aircraft and Liveries tabs with searchable game names, filing value, affected catalog row count and status (explicit, uses default, needs a value, or absent from catalog). Group identical raw names. Expanding a livery lists the aircraft it affects. Filter to unmapped names and select several for a shared value. A separate defaults control shows exactly what a miss will file.
2. **Suggest and preview.** Offer existing target values as completions and “Use game name” for exact copies. If the connected single-select choices are available, suggest exact matches first. Case-insensitive or punctuation-similar matches can be suggestions the admin accepts individually. Preserve spelling, accents and case on save. A sample-flight picker shows both raw inputs, both proposed outputs, and the rule or default used for each. Show a known invalid select choice as an error for that destination. When choices are unavailable, label destination validation as unavailable.
3. **Review and save.** Display the diff and impact before applying bulk changes. For example, a change to the `Air France` rule affects 14 catalog rows in the supplied snapshot. Explain that removing a rule restores fallback behavior, and show the resulting value. Keep the draft visible after validation errors or a stale-revision conflict. An empty catalog should offer a sync-status message and still allow export of existing rules.

### Three-step guide to display beside the download buttons

1. **Download your files.** Choose **Export mappings JSON** for this server and **Download aircraft and liveries CSV** for the game names.
2. **Edit the JSON.** Use the exact names in the CSV as keys and your form’s accepted values as values. If using an AI editor, give it both files and your accepted values. Ask it to preserve existing rules and defaults and leave uncertain matches for review.
3. **Import and review.** Choose **Import mappings JSON**, check additions, changed values, removals and sample PIREPs, then **Apply to this server**. Exporting the previous settings first gives you a file you can restore through the same review.

## Persistence and matching

Proposed migration, following the manual migration sequence in `services/politburo/migrations/`:

| Column in `public.va_pirep_value_mappings` | Purpose |
|---|---|
| `va_id uuid PRIMARY KEY REFERENCES public.virtual_airlines(id) ON DELETE CASCADE` | One independently editable mapping set per VA. |
| `aircraft_mappings jsonb NOT NULL`, `airline_mappings jsonb NOT NULL` | Exact source-name → target-string objects. Default each to an empty object and check JSON object type. Validate entries in the domain service. |
| `default_aircraft text NULL`, `default_airline text NULL` | A null default leaves an unmatched field unresolved. |
| `revision bigint NOT NULL`, `updated_at timestamptz NOT NULL`, `updated_by uuid NOT NULL` | Positive monotonic revision, save time and authenticated internal user ID. |

One VA-level read retrieves both small dictionaries. One conditional write commits an entire reviewed edit, including defaults. This suits the measured file size and JSON workflow. Keep the global catalog independent so the next game sync can update names without modifying VA settings. A missing VA row reads as revision zero with empty dictionaries and null defaults. Never auto-seed an unrelated VA with AF/KLM values.

**Resolution order.** For each enabled filing field, resolve the raw source from the catalog, look up its exact key, then try the VA default, then report “needs a value.” For an unknown livery ID with a known aircraft ID, still resolve aircraft. On inconsistent aircraft/livery IDs, surface a catalog mismatch and withhold the livery match. Treat missing source names as misses, never as synthetic lookup keys such as “Unrecognized.” An explicit identity mapping is valid. Passthrough on every miss would change the legacy filing behavior. Show an actionable validation error before submission if a required output is unresolved. Omit fields disabled in the projection. On provider submission, surface rejected select values with the field and offending value.

A game rename appears as a new unmapped name and an old alias. The editor can copy the old target to the new key after review. The first release gives the same livery name one airline target across the VA, matching the legacy helper. An aircraft-specific exception would need an additional rule type and explicit precedence if a VA demonstrates that need.

## File contracts

**Canonical UTF-8 JSON (v1).** Export stable, sorted keys, indentation and a trailing newline. Keep the document portable between VAs. Scope, credentials, connection IDs, pilot data and server-managed revision metadata belong outside the editable payload.

```json
{
  "version": 1,
  "defaults": { "aircraft": "Other", "airline": "Generic" },
  "AircraftMappings": {
    "Airbus A318": "A318-100"
  },
  "AirlineMappings": {
    "Air France": "Air France",
    "KLM Cityhopper": "KLM"
  }
}
```

Require all four top-level fields in v1. Defaults accept a nonblank string or null. Rules accept nonblank strings on both sides. Reject duplicate object keys before normal JSON decoding collapses them, unknown fields, unsupported versions and malformed or oversized uploads. Return a key path and reason for each validation issue. Preserve exact strings and flag suspicious surrounding whitespace for review. Start with a 1 MiB upload limit and verify it against a full-catalog fixture.

**Legacy import.** Recognize the original unversioned document by its two dictionaries. Extract each reserved `Other` entry into its corresponding default, using null if absent. Preserve every other pair, including aliases absent from the local catalog. In v1 the explicit defaults free up the literal key `Other` for a real catalog name. Import is a full replacement. Highlight removed rules and cleared defaults, including replacement by empty dictionaries. Preview has no side effects. Importing an unchanged export produces an empty diff.

**Downloadable CSV.** Generate a UTF-8 attachment from Postgres on demand, one row per livery ID. Use a single header and the first four legacy column names in their original order, followed by the optional display context:

```csv
LiveryId,AircraftId,AircraftName,LiveryName,DisplayAircraftName,DisplayLiveryName
```

Use Go’s CSV writer for quotes, commas and newlines. Order by raw aircraft name, raw livery name and livery ID for repeatable output. Export the full shared catalog regardless of the active UI filter. The guide should point to `AircraftName` and `LiveryName` as the exact JSON keys. CSV import is unnecessary for this workflow.

## HTTP contract and write safety

Proposed base path `/api/v1/va/pirep-value-mappings`. Document operations in the split OpenAPI sources, regenerate the committed bundle and consumers with `make generate`, and mount through the same VA admin stack used for data connections. The browser uses the current VA session. Reject requests lacking the active VA’s admin authority, including download and preview requests. Ignore no ownership overrides silently: reject unexpected identifiers in the document.

| Operation | Behavior |
|---|---|
| `GET /` (base resource) | Return canonical document, revision and update metadata. |
| `GET /export` | Download just the canonical JSON as an attachment. |
| `GET /catalog` and `GET /catalog.csv` | Provide searchable, paginated catalog names and coverage for the editor, and a full CSV attachment for offline edits. |
| `POST /preview` | Accept an edited or legacy document, return normalized v1, current revision, additions/changes/removals, unmatched aliases and coverage. Optionally resolve a selected aircraft/livery pair against the draft. |
| `PUT /` (base resource) | Accept reviewed v1 and the expected revision via `If-Match`. Authenticate, validate again and atomically replace the VA row. Return the committed revision. |

Match the existing PIREP settings convention for `If-Match` and HTTP 412. For revision zero use a conditional insert and map a concurrent first insert to 412. For an existing row update only where VA and revision both match. Reject a missing precondition. After conflict, reload the server version and show a fresh diff against the retained draft before another apply. Retain the row and advance revision when clearing a saved configuration. Handle a matching no-op without a revision bump. A repeated request carrying a stale revision must never overwrite newer data.

## Delivery and acceptance

| Stage | Deliverable and verification |
|---|---|
| 1. Domain and storage | Add migration, repository and resolver under the VA domain, wired in `internal/app/app.go`. Exercise identical strings, many-to-one rules, unknown IDs, independent defaults and empty configuration. Configure two VAs differently for the same game pair and verify each gets its own output. |
| 2. Contract and files | Add admin handlers and OpenAPI definitions. Verify CSV quoting, single header, full catalog and deterministic ordering. Import the AF/KLM file into an explicitly selected test VA, export v1 and re-import it. All 53 aircraft and 269 airline non-fallback rules plus both defaults must survive. Compare resolver outputs with the legacy helper over all 1,167 snapshot rows. Use the old files as local migration evidence, without introducing runtime dependencies on the old project. |
| 3. Admin UI | Add the editor, file controls and three-step guide to the existing panel. Verify keyboard-only editing, readable validation, empty catalog behavior, edit without a provider, preview cancellation, bulk removals, cross-VA denial and conflicting admin saves. Run `go test ./internal/ui/...` and transport-handler tests from the service directory. |
| 4. Filing integration | When implementing the bot’s config/submit endpoints, resolve on the server using the authenticated VA and flight IDs. Show the proposed values in the form. Carry the mapping revision into submission and ask for a refreshed preview if the rules changed. Verify the submitted provider payload uses the same values the pilot reviewed and test one aircraft-only filing projection. Run root `make check` and the bot build/tests when those paths change. |

Ship stages 1–3 as a usable admin module. Enable bot filing once stage 4 passes an end-to-end test against a configured test VA. A fresh installation can export a blank template immediately. AF/KLM can explicitly import its existing document and inspect the live catalog diff before applying it.
