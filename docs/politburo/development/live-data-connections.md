# Live VA data connections — implementation plan and low-level design

## Delivery contract

A VA administrator can register multiple credentials, configure multiple bases for each credential, and map up to three logical entities independently. A VA has one active mapping each for pilots, routes, and PIREPs. All three can share a base or use different bases and credentials. Completing the pilot setup alone makes pilot reads available. Airtable is the first provider. A future database provider should implement the same read operations and supply its own connection settings.

The deliverables are an admin dashboard tab, a typed setup API, and an internal Go service for live mapped reads. Filing a PIREP, Discord commands, sync jobs, and a database adapter belong to later phases. Only connection settings, encrypted credentials, mappings, and edit metadata persist. Every schema inspection, preview, lookup, and page request goes to the provider. Record data stays within the requesting operation. Use neutral entity and field names throughout implementation, fixtures, help text, and examples.

## Existing integration points

| Responsibility | Existing code to extend | Design consequence |
| --- | --- | --- |
| Config storage | [VA config repository](../../../services/politburo/internal/community/virtualairlines/config/repository.go) and [table definition](../../../services/politburo/migrations/002_va_configs.sql) | Reuse the unique VA/type JSONB row and actor audit fields. Existing unconditional upsert needs a conditional sibling for concurrent edits. |
| HTTP and identity | [Config handler](../../../services/politburo/internal/transport/http/api/va/handler_config.go) and [generated route delegates](../../../services/politburo/internal/transport/http/routes_api.go) | Reuse VA context resolution, actor resolution, admin authorization, body limits, and JSON envelopes. The generic GET serializes the stored block directly. Guard both generic operations with their public config-type allowlist. |
| Construction | [Composition root](../../../services/politburo/internal/app/app.go) | Construct one shared HTTP client, provider adapter, and service. Inject them into the existing VA handler. Keep claims, cookies, generated API types, and HTTP responses in transport. |
| Dashboard | [Admin UI routing](../../../services/politburo/internal/transport/http/routes_ui.go), existing admin template and [styling guide](styling-guide.md) | Extend the admin shell, active-tab selection, embedded templates, shared form classes, and page-specific ES module pattern. |
| Contract | [VA config path](../../../openapi/politburo/paths/va/config.yaml) and [generation guide](../../development/openapi.md) | Add path and schema sources, register them in the source spec, regenerate consumers, and commit the bundled spec. |

## Configuration model

Use the internal config type `DATA_CONNECTIONS`. Keep this type out of the public generic config enum. A missing block represents an empty configuration at revision zero. The dedicated admin endpoints expose a safe projection. Example internal JSON, with illustrative identifiers and an omitted encrypted value:

```json
{
  "version": 1,
  "revision": 3,
  "connections": [{
    "id": "connection-id",
    "name": "Flight operations",
    "provider": "airtable",
    "airtable": {
      "credential": {"version": 1, "sealed": "..."},
      "bases": [{"id": "base-setup-id", "name": "Operations", "baseId": "appExample"}]
    }
  }],
  "mappings": {
    "pilots": {
      "connectionId": "connection-id",
      "baseSetupId": "base-setup-id",
      "tableId": "tblExample",
      "fields": {"callsign": "fldExample"}
    }
  }
}
```

- Generate stable internal UUIDs for connections and base setups. Connection labels are editable. Validate references within the current VA block. Keep provider-specific base and table resolution inside the adapter. Persist physical field IDs so column renames preserve mappings.
- Give the admin view its own DTO with `hasCredential`, labels, IDs, mappings, revision, and audit metadata. The write DTO accepts a replacement token. Omitting it on edit retains the existing token. An empty replacement is invalid. Deletion removes the credential with the connection.
- Seal credentials with the shared at-rest cipher (`services/politburo/internal/secretstore`), configured by `DATA_ENCRYPTION_KEY` (standard base64, 32 bytes) or `DATA_ENCRYPTION_KEY_FILE`. The root is split with HKDF into an AES-256-GCM key and a separate cursor HMAC key. Associated data binds envelope version, purpose `airtable.pat`, VA ID, and connection ID. A missing key disables this capability with a clear setup error while the rest of Politburo keeps running. Keep the root durable across replicas, backups, and redeploys, and separate from database backups. Signed dashboard tickets stay on `SIGNED_LINK_SECRET`.
- Require the observed revision on every mutation. Read and validate the candidate configuration first, including any provider calls. On first write use insert-on-conflict-do-nothing at expected revision zero. For updates, match VA, config type and expected JSON revision in the UPDATE predicate. Write the incremented revision, actor, and timestamp together and return the stored metadata. Zero affected rows means conflict. Retain an empty block after the last deletion so revisions continue increasing. Hold no database lock across upstream requests.
- Reject deletion of referenced bases or connections. Changing a mapped base's physical ID would invalidate the saved table IDs. Add a new base setup, remap the entity, then remove the original. Test a replacement credential against every configured base before committing it.

## Field catalog and mapping rules

Keep one catalog in Go that drives validation, the field-picker response, filter validation and normalized output. Each entry defines its logical key, label, value kind, and accepted provider types. Send compatible field choices with the live schema response. This keeps validation and browser options consistent.

| Entity | Supported logical keys | Minimum valid mapping |
| --- | --- | --- |
| Pilots | callsign, ifcUsername, displayName | callsign or ifcUsername |
| Routes | origin, destination, durationSeconds, aircraft | origin and destination |
| PIREPs | pilotRefs, routeRefs, ifcUsername, completedOn, flightTimeSeconds, aircraft, airline, flightMode, remarks, fuelKg, cargoKg, passengerCount | At least one supported field |

Normalize text and single-choice cells to strings, durations to seconds, quantities to numbers, and linked fields to arrays of opaque record IDs. Accept numeric mappings where the column uses the logical field's displayed unit. Preserve calendar dates as dates and normalize timestamps with a timezone to UTC. Missing optional cells are absent from the output. Numeric zero remains present. Project mapped fields only.

Validate a mapping against the current base schema at save. Check table existence, field IDs, and cell types. For mapped pilot or route references, check the linked target against the configured entity when available. Compare physical base and table, including when credentials differ. Validate affected inbound references on subsequent remaps. An independent table in another base can still be read without a linked-field mapping. Empty tables are valid because validation uses metadata.

## Service and adapter design

Place the feature under the virtual-airline bounded context in a sibling `connections` package. Put the Airtable HTTP implementation in a provider-specific subpackage. The service owns configuration validation, VA scoping, catalog rules, and logical results. The adapter owns upstream authentication, field type translation, formula construction, and pagination. Wire the supported adapter explicitly in the composition root. Add additional provider implementations when needed.

| Consumer-facing Go operation | Inputs | Result |
| --- | --- | --- |
| List | context, VA ID, entity, exact-match filters, page size, opaque cursor | Page of records with source ID and logical fields, plus optional next cursor |
| Get | context, VA ID, entity, source record ID | One mapped record or typed missing-record error |
| Preview | context, VA ID, candidate mapping | Up to five mapped sample records from one live read, including an empty result |
| Inspect | context, VA ID, connection ID, base setup ID | Live table and field descriptors, compatible logical keys, linked targets |

1. Resolve the VA block and entity mapping for each request. Decrypt only the selected credential. Pass the context deadline through database and outbound HTTP operations.
2. Bind the provider to the selected base. Resolve mapping IDs against live metadata as needed, including current field names for filters. Reuse metadata within that operation. Use a fixed Airtable HTTPS origin, encoded path/query parameters, bounded responses and a finite HTTP timeout. Disallow redirects carrying credentials.
3. Read only selected fields. Combine supported scalar equality filters with AND. Accept catalog keys and typed values, build formulas inside the adapter, and escape field names and literals. Reject unsupported types or expressions. Preserve duplicate matches as separate records. The caller can use Get for an unambiguous record ID.
4. Default to 25 records per page and cap at 100. Fetch one upstream page per call and return continuation state bound to VA, mapping revision, entity and filters. Reject a cursor used against a different query or mapping. A bounded signed cursor can carry the upstream offset without server-side storage.
5. Translate missing mapping, invalid filter, schema drift, missing record, denied upstream access, rate limiting, timeout, and provider outage into distinct domain errors. Preserve cancellation. Return sanitized messages and a valid upstream retry delay for throttling. Avoid logging authorization headers, query literals, sample records, upstream bodies, or encrypted blobs.

## HTTP contract and dashboard

Use dedicated OpenAPI operations under `/api/v1/va/data-connections`. All operations use the current VA's administrator authorization for cookie and API-key requests. Resolve the VA exclusively from authenticated context. The dashboard uses these same operations. Application features use List and Get internally in this phase.

| Method and relative path | Behavior |
| --- | --- |
| GET / | Return the redacted configuration and field catalog, including an empty state at revision zero. |
| POST /connections | Create a named connection with provider, token, and initial named base. Test schema access and read access before save. |
| PATCH or DELETE /connections/{id} | Rename or replace the token, or delete an unused connection. |
| POST /connections/{id}/bases | Add a named base after a live access test. |
| PATCH or DELETE /connections/{id}/bases/{baseId} | Rename a base setup or delete an unused setup. Path baseId denotes the internal setup ID. |
| GET /connections/{id}/bases/{baseId}/schema | Test the saved connection and return fresh metadata. |
| PUT or DELETE /mappings/{entity} | Validate and save a complete entity mapping, or clear it. |
| POST /mappings/{entity}/preview | Validate the draft mapping and return a bounded sample. Accept the mapping in the body so preview works before saving. |

Return the existing data and error envelopes. Return the revision as an ETag and require its value via If-Match on configuration writes, including revision zero. Use 412 for a stale revision, 428 for a missing precondition, 409 for an in-use resource or schema drift, 400 for malformed inputs, and 422 for incompatible mappings or rejected provider credentials. Distinguish caller authorization (401/403) from provider authorization. Use 404 for missing resources, 429 for upstream throttling, 504 for timeout, and 502 for other upstream failures. Apply Cache-Control no-store to all feature responses.

Add **Data connections** beside the existing admin tabs. First render the connection list and independent mapping statuses. The administrator adds a connection and base, chooses an entity and table, selects compatible fields, previews, then saves. Support replacing credentials without redisplaying them. Keep draft edits in page memory. On stale save, preserve the draft, show the conflict and offer reload. Show empty-table success, missing permissions, removed fields, throttling, and retry actions distinctly. Render provider strings as text and use existing busy controls, accessible labels, and status regions.

## Delivery sequence and verification

1. **Storage and secrets.** Add the internal type, typed config, redaction and conditional repository method. Test initial create races, concurrent updates, existing BASIC/WEBHOOKS behavior, tenant isolation, retained secrets, nonce uniqueness, and failed decryption. Verify generic GET and writes cannot access the restricted block even with a hand-built request.
2. **Provider and reader.** Implement against a local httptest server with synthetic credentials and schemas. Cover different connections per entity, empty tables, exact-match escaping, pagination, duplicate callsigns, missing optional fields, changed column names, incompatible types and broken links. Change an upstream record between calls and assert the second read observes it. Exercise schema/read permission failures, cancellation and 429 handling.
3. **Contract and handlers.** Add schemas, paths and delegates using the existing admin middleware, context resolution, and response helpers. Test anonymous, non-admin, and cross-VA requests plus credential retention and concurrent saves. Run `make generate` and `make openapi-bundle-check` at the root. Review the committed bundle and leave ignored generated clients to the generators.
4. **Dashboard.** Extend tab routing and rendering tests. Exercise setup from an empty VA, configuring just pilots, using separate bases, previewing before save, and replacing a mapping. Verify provider data and credentials stay out of browser storage. Run `go test ./internal/ui/... ./internal/transport/http/...` from the service, followed by `go test ./...`.
5. **Rollout.** Document the deployment key in the service and production environment templates using empty example values. Update the existing datasource product document to reflect the live flow. Provision a read-only test credential with schema and record access on an intended base. Check the dashboard with one admin VA and check a second VA cannot resolve those resources. Confirm repeated previews observe upstream edits and a disconnected provider returns an error. Use existing HTTP performance metrics and structured request logs. Rollback can remove the feature surface while retaining the typed config block and its decryption key.

Completion requires a working admin setup and successful live reads of all three entity types, including a configuration spread across multiple bases. No record tables, preview persistence, Redis keys, or background sync jobs are required.
