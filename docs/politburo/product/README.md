# Product planning

This folder is the working set for **what Infinite Experiment is for users**,
which data it may own, and which features we will re-implement, drop, or add
on the rewrite.

It is not an architecture spec and not a production-migration guide.

| Document | Purpose |
|---|---|
| [context.md](context.md) | Product thesis, users, repos, and the two connection planes |
| [data-sources.md](data-sources.md) | Where data lives and what we refuse to copy |
| [inventory.md](inventory.md) | What the rewrite actually runs vs leftover bot/schema/code |
| [identity-membership.md](identity-membership.md) | Users, VA roles, single administrator, IFC reports |
| [airtable-datasource.md](airtable-datasource.md) | Airtable wizard, field maps, links, per-module sync |
| [feature-catalog.md](feature-catalog.md) | Feature board: live, inherited, leftover, and open for planning |

Related docs:

- `../architecture/overview.md` — current rewrite process and HTTP surfaces
- `../conventions.md` — cache-backed API and auth boundaries
- `../future/` — infra plans not yet executed (Kubernetes)
- Workspace root `TECHNICAL_STANDARDS.md` and `UI_FEATURE_AUDIT.md` describe
  the **previous** Politburo/Vizburo. Treat them as historical product memory,
  not as the rewrite's current shape.

## How to use this folder

1. Agree data ownership in `data-sources.md` before designing a feature.
2. Check `inventory.md` before assuming an old command, table, or UI page
   still exists in the rewrite.
3. Add or move rows in `feature-catalog.md` as we decide keep / reshape / drop.
4. Implementation details belong in architecture docs or a later plan, not here.
