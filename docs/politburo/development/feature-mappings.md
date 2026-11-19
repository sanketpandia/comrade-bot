# VA feature mappings for Stats and PIREPs

**Delivery target.** A VA administrator can use an established Airtable connection to configure pilot stats, add career stats, preview the combined result for one pilot, save it, and load the same configuration in a new session. A second feature, PIREPs, can reuse the pilot and route bindings and preview a filing. This release delivers configuration and read-only previews. The subsequent Discord release consumes the saved configuration.

A small VA can finish Stats with one table, a pilot identifier and one statistic. A VA with separate roster, career, report and route tables can configure all four. Administrators choose the information they want to use. Politburo owns the variable names, types, join paths and formatting.

## Ownership and persistence

Use the existing uniqueness constraint on `va_configs(va_id, config_type)`. The organization boundary is the VA resolved by the existing request context. A missing feature row means setup has not started. No database migration, record mirror or scheduled sync is required.

| Row | Responsibility | Written when |
| --- | --- | --- |
| `DATA_CONNECTIONS` | Existing encrypted credentials, connection/base setups, and one source mapping per logical entity. Each mapping contains connection ID, base setup ID, table ID and logical-key-to-field-ID bindings. | A source or binding changes. |
| `MAP_STATS` | Selected Pilot and Career Mode values and the identity key to use for each configured section. The presence of a section enables it. | The administrator saves Stats. |
| `MAP_PIREP` | The recognised fields the VA wants a future filing operation to supply. A mapped read-only route label can support Stats without becoming a filing field. | The administrator saves PIREPs. |

The shared document already supports independently situated pilots, routes and PIREPs. Add one entity, `careerPilots`. Two entities may point to the same physical table. A native Airtable record-link traversal requires the link and its mapped target to belong to the same physical base. Matching a pilot and career row by callsign or IFC username works across bases and connections.

Example feature documents below use application keys. The keys within each section resolve against its entity. Render in catalog order so administrators need no ordering controls.

```json
{
  "version": 1,
  "revision": 3,
  "pilot": {
    "matchBy": "callsign",
    "fields": ["displayName", "flightTimeSeconds", "category", "region", "rank"]
  },
  "career": {
    "matchBy": "callsign",
    "fields": ["flightTimeSeconds", "timeToNextAircraftSeconds", "lastActivity",
               "assignedRoutes", "aircraft", "airline", "lastFlownRoute"]
  }
}
```

```json
{
  "version": 1,
  "revision": 1,
  "fields": ["pilotRefs", "routeRefs", "flightTimeSeconds", "remarks"]
}
```

For example, the shared roster binding might be `{"connectionId":"conn-demo","baseSetupId":"base-demo","tableId":"tblRoster","fields":{"callsign":"fldIdentifier","flightTimeSeconds":"fldBlockTime"}}`. These identifiers are illustrative. Store provider IDs returned by inspection. Resolve current names from schema when building an Airtable query. Existing string field bindings continue to decode. For a plain numeric duration only, store its confirmed unit in an optional sibling map such as `fieldUnits: {"flightTimeSeconds":"hours"}`; a native duration already carries seconds. Preserve the current seconds interpretation for existing numeric duration bindings without unit metadata.

## Typed field catalog

Extend the existing Go catalog in `internal/community/virtualairlines/connections`. Each entry declares its entity, key, label, normalized kind, compatible provider types and supported usage (read, write or both). Keep a small static feature definition for section order, selectable fields, lookup keys and required combinations. New features can reuse these definitions and add a config type when they have independent choices to persist.

| Entity | Recognised application keys | Behavior |
| --- | --- | --- |
| Pilots | `callsign`, `ifcUsername`, `displayName`, `flightTimeSeconds`, `category`, `region`, `rank`, `status` | Match on one identifier and show selected roster values. |
| Career pilots | `callsign`, `ifcUsername`, `flightTimeSeconds`, `timeToNextAircraftSeconds`, `lastActivity`, `assignedRoutes`, `aircraft`, `airline`, `lastLogRefs` | Match independently on the configured identifier. Last-log references target PIREPs. |
| PIREPs | Existing pilot/route references, IFC username, completion date, duration, aircraft, airline, flight mode, remarks, fuel, cargo and passenger fields. Add `routeLabel` for a readable route on the report. | Validate write access by field type for the filing projection. A route label can be read to display a career pilot's last flight. |
| Routes | Existing `origin`, `destination`, duration and aircraft; add optional `routeLabel` for display. | Render a route label when mapped, otherwise format origin and destination. |
| Derived Stats value | `career.lastFlownRoute` | Follow the single last-log reference. Read the report's route label, or follow its route reference and format the mapped route. |

**Normalization rules**

- Use the base schema as the source of field identity and compatibility. Extend the Airtable field descriptor with linked table ID, computed result kind and list element kind when available. A sample may be empty. Give a field-specific explanation if its type cannot be safely determined.
- Text and single-select values yield strings. Multi-selects and textual lookups can yield a list of strings. Support scalar fields such as aircraft or airline supplied by a single-valued lookup; surface multiple distinct values as a cardinality issue for that scalar. Keep assigned routes as a list. For native route links, resolve display labels from the configured Routes table in one bounded batch, preserving source order.
- Accept formulas, lookups and rollups for reads when their result type and cardinality match. Exclude computed fields from the PIREP write picker. Validate link destinations using metadata even if no linked cells are populated. Writability of a column describes its schema; actual write permission is checked when filing is implemented.
- Normalize durations to whole seconds. For new plain number columns, ask once for seconds, minutes or hours. Convert on read and use the inverse conversion in a filing preview. Distinguish a calendar date from a timestamp and preserve date-only values without shifting time zones. Render zero and empty/missing cells distinctly.
- Treat the last-log relationship as singular. An empty link omits that value. Multiple last-log links produce a field-specific preview warning because their order cannot establish which flight is latest. Never show raw record IDs as pilot-facing values.

## Administrator workflow

1. **Choose a feature.** In the existing configuration area, show Connections, Stats and PIREPs with a saved/setup-needed status and an action. Render the overview from Postgres. Reuse an established connection, default the base when only one is available, and offer the existing connection setup inline when needed.
2. **Map the roster.** Ask “Where is your pilot roster?” Show table names. Display the pilot lookup key first, then common statistics. Suggest unambiguous type-compatible matches using normalized labels and a short catalog alias list. Keep ambiguous matches unselected. An existing saved choice always wins. Selecting a supported statistic adds it to the preview.
3. **Add career stats if wanted.** Offer the same roster source or a different table/base. Ask for the career lookup column. Show last-log and route settings only when the admin selects Last flown route. Reuse existing compatible bindings. Indicate dependencies with a short instruction such as “Choose the report column that contains the route.”
4. **Preview and save.** A table sample shows example values while mapping. A separate “Test a pilot” action exercises lookup and joins using the entered identifier, or the chosen member's identities when the sections use different keys. Show both sections, highlight unresolved fields, and retain the draft after errors. Preview is optional for an empty table with valid schema. Save revalidates current schema and returns the saved revisions.
5. **Configure PIREPs independently.** Select the report table, a pilot reference or writable IFC username, and at least one writable flight detail. Suggest existing bindings and allow optional columns to be skipped. Preview a synthetic filing against the destination columns. This validates the mapping; it does not submit a record or certify a token's write scope.

Use shared glass panels, solid controls, badges, buttons, validation messages and focus tokens throughout configuration, including the existing connection form. Follow the [UI styling guide](styling-guide.md). On desktop, align application field, source column and sample in one row. On narrow screens stack those three elements together. Use visible labels, focus the first invalid control after submit, and announce save/preview status. Show a friendly connection/base/table name where available. Avoid exposing technical IDs as the main setup instruction.

**Illustrative preview** for pilot NX042:

| Section | Application field | Organization's column | Source sample | Display |
| --- | --- | --- | --- | --- |
| Pilot | Flight time | Block time | 45300 seconds | 12h 35m |
| Pilot | Region / rank | Division / Grade | North / First Officer | North · First Officer |
| Career | Time to next aircraft | Time remaining | 0 seconds | 0h 00m |
| Career | Assigned routes | Rotation | Two text values | ABC–DEF, DEF–GHI |
| Career | Last flown route | Latest report → Sector | Linked report with route label | ABC–DEF |

For a synthetic PIREP, show Pilot **NX042**, Route **ABC–DEF**, Flight time **1h 05m**, and the destination column beside each. If the destination stores numeric minutes, show the value **65** as well. Label samples taken from Airtable and synthetic filing values distinctly.

## Service and API implementation

1. **Build on the current connection service.** Keep credentials, schema inspection, query construction, raw provider errors and redaction there. Extend the existing entity enums in the OpenAPI sources and the Go validators together. Put feature-specific draft validation, field selection and composition in a small VA feature-config service. The HTTP handler resolves the VA and actor and enforces existing admin authorization on every operation.
2. **Expose a dedicated feature resource.** Use `GET /api/v1/va/feature-configs/{feature}` for the redacted catalog, saved selection, relevant bindings, revisions and dependencies. Use `POST .../{feature}/preview` for a draft plus an optional test identity, and `PUT .../{feature}` to validate and save. Limit the path enum to `stats` and `pirep`. Define typed request/response schemas with the existing data/error envelope. Keep these config types off the generic config API. Generate from OpenAPI sources and commit the bundle.
3. **Represent edits as a feature selection plus changed bindings.** Include the observed feature revision using the existing `If-Match` convention and the observed connections revision in the body. Submit only edited shared keys. Removing a displayed stat deselects it for this feature; it does not erase the shared binding. Identify other features using a changed binding before save. Reject changes that would invalidate another saved feature and point to the dependency. Apply the same check to existing mapping, connection and base edit/delete entry points.
4. **Save consistently.** Perform provider validation outside the DB transaction. In a short transaction serialize config changes for the VA, reload affected documents, recheck revisions and dependencies, then conditionally update the changed documents with actor and timestamp. A small transaction-scoped VA lock shared by these mutation paths covers first inserts as well as edits. Refactor the revision-aware repository method to run against the transaction. On conflict, roll back and return the existing 412 response pattern. Preserve the browser draft and offer reload/review. First save uses revision zero; an unchanged shared document keeps its revision.
5. **Use bounded reads.** Load schema once per distinct connection/base per operation and pass the index into validation, formula construction and normalization. In the browser, reuse inspected schema and a five-row maximum sample for the current edit session. Request only the fields needed; get a fresh sample on explicit refresh or when a newly chosen column lacks a sample. Normal dropdown changes use the available sample or show “Sample unavailable”. No background requests on keystrokes.
6. **Match a member deterministically.** Use the configured membership callsign or verified IFC identity. Build equality filters using the existing field-reference and literal-escaping helpers. Compare callsigns case-insensitively and ignore spacing, preserving significant characters and leading zeroes. Limit to two matches and do not follow pagination. Zero roster matches gives a pilot-not-found result; two gives an ambiguous match. A missing career row still yields roster stats. A provider failure on an optional join yields a visibly incomplete preview and a retry action.
7. **Bound optional work.** Resolve at most one last report and one route for that report. Display the first six assigned routes and the total count; batch-fetch up to those six labels if stored as links. Reuse results for repeated table/record references within the operation. If the roster and career roles share a table and lookup, request their combined fields in a single filtered lookup.
8. **Preserve privacy.** Enforce VA ownership on every connection/base reference and derive secrets server-side. Keep tokens, raw provider payloads, sample rows, formulas and test identities out of logs and metric labels. Keep samples in the active request or browser view and return admin previews with `Cache-Control: no-store`.

## Metrics and operational checks

Instrument at two boundaries. The Airtable client already counts completed HTTP attempts and records their durations. The service counts user-visible operations. A successful preview with three upstream calls should increase the corresponding logical preview count by one and the provider counter by three.

| Question | Signal |
| --- | --- |
| How much Airtable traffic are we generating? | `politburo_dataconnections_provider_requests_total{provider="airtable"}` by operation and outcome. Current provider operations are `schema` and `records`. |
| Are permissions, throttling or latency hurting setup? | Chart `denied` and `throttled` outcomes and p95 from `politburo_dataconnections_provider_request_duration_seconds_bucket`. Surface retry-after to the UI on 429. |
| Which setup action failed or became slow? | Reuse `politburo_dataconnections_reads_total` and `politburo_dataconnections_read_duration_seconds` with bounded operation/entity values for feature previews. Track one terminal outcome per operation, including partial results. |

Fix the current duplicate observation in `Service.Inspect` before using its counts as a baseline. Use a single deferred observation with the actual returned error. Verify that validation and preview reuse one schema fetch for the same base. Keep dashboards compatible with existing labels. Use a fake provider and request counters for exact per-preview budgets; aggregate production counters also include connection setup and other reads.

Example Grafana expressions:

```promql
sum by (operation, outcome) (
  increase(politburo_dataconnections_provider_requests_total{provider="airtable"}[1h])
)
```

```promql
histogram_quantile(0.95, sum by (le, operation) (
  rate(politburo_dataconnections_provider_request_duration_seconds_bucket{provider="airtable"}[5m])
))
```

## Release acceptance

**Compatibility gate.** Demonstrate that the complete representative organization setup can be loaded, saved, reopened and used to resolve every selected value. Use synthetic names, IDs and records in version control. Run the same check against the authorized organization's supplied configuration as a release acceptance step. Use its existing registered connection and resolve its table/column names to inspected IDs. For a name-based input, use a thin acceptance loader to translate per-section table, callsign column and typed field selections into the same draft accepted by the API. Report unresolved or ambiguous fields individually. Review the mapping and supply relationship bindings through the ordinary workflow. This gate exercises the actual configuration shape without making a general-purpose import UI part of the product.

| Scenario | Pass condition |
| --- | --- |
| Complete setup | Load roster name, callsign, total flight time, category, region and rank; career total time, time remaining, last activity, assigned routes, aircraft and airline; follow the last report to its route. Assert every expected key, value and type after save/reload and in the combined preview. Test both direct report route labels and linked routes. |
| Minimal setup and isolation | Roster plus one stat can save on its own. Adding Career Mode or PIREPs later preserves prior choices. A second VA with different columns gets its own result; foreign connection/base references fail. |
| Diverse tables and types | Cover the same table serving two roles, separate bases for identity lookups, native durations and numeric hours, scalar and list lookups, computed results, empty cells, zero, and empty tables. Reject incompatible link targets and computed write fields. |
| Identity and drift | Cover spaced and mixed-case callsigns, leading zeroes, duplicate identities, missing career row, missing linked record, renamed columns, deleted fields and changed result types. A renamed field resolves by ID; broken bindings identify the field to repair. |
| Consistent edits | Race two first saves, two edits, and a connection deletion against a feature save. Assert that the winner persists a usable configuration and the losing draft cannot partially update either row. Existing connection documents and API readers continue to work. |
| Call budget | Overview and local edits generate zero upstream traffic. A cold roster-only pilot preview uses one schema call and one filtered record call. A shared-base preview with independent roster/career lookups and one last report containing route text uses one schema call and three record calls. Verify a 429 preserves the draft and leaves a single throttled HTTP observation. |
| Usability and contract | Complete setup by keyboard and on a narrow screen. Verify sample labels, inline errors and draft recovery. Run OpenAPI generation/bundle checks, Politburo tests and UI token/template tests. Exercise the dashboard through its normal admin entry point. |

Deliver the catalog and persistence first, then the API and wizard, then the compatibility and call-budget checks. Once this gate passes, the Discord milestone can connect the existing self-service `/stats` command to a typed Politburo endpoint using the same resolver as the preview. A later PIREP writer can consume the validated write projection and handle submission permissions and failures at filing time.
