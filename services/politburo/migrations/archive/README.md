# Archived schema (not applied by default)

These files preserve the old Infinite Experiment database shape and incremental
upgrades. **Do not run them on a fresh Politburo rewrite database** — use
`../000_core_schema.sql` instead.

| File | Purpose |
|------|---------|
| `legacy_000_infinite_schema.sql` | Full `pg_dump` baseline (events, PIREPs, Airtable sync, etc.) |
| `legacy_001_identity_membership_upgrade.sql` | ALTERs for DBs already on the legacy `000` dump (role renames, operator tables) |

Use the archive when you need to recreate or compare against the legacy monolith
schema, or when upgrading an existing `infinite` database that was created from
the old migration set.
