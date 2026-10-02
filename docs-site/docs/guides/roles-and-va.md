# Roles & your VA

Politburo separates **who you are on the platform** from **how you participate in each virtual
airline (VA)**.

## Two layers

| Layer | Meaning |
|-------|---------|
| **Platform user** | One person: Discord ↔ IFC username (globally unique) |
| **VA membership** | That person in one VA, with one role in that VA |

You can be a pilot in one VA and staff in another. There is no single “global VA rank.”

```text
Discord user ──1:1── Politburo user ──1:1── IFC username
                         │
                         └── memberships ── Virtual airline (1:1 Discord guild)
```

## VA and Discord

- **One VA = one Discord guild** (server). Extra guilds for the same VA are not supported.
- **`/initserver`** binds the guild to a VA code. You must already be registered and be a
  Discord **Administrator** on that guild. You become the VA’s sole **administrator** in
  Politburo (independent of Discord permissions later).
- If your community moves to a new Discord server, contact the platform operator — rebinding is
  not self-service.

## Roles (per VA)

| Role | Who | Typical duties |
|------|-----|----------------|
| **Proletariat** | Pilots | Fly, use pilot commands, open the portal as a member |
| **Bourgeoisie** | Managers | Staff work, logbook/events, change pilot callsigns, remove pilots — not role assignment |
| **Administrator** | Exactly one per VA | Datasource, matching rules, roles, transfer admin seat, destructive VA settings |

Default on join: **proletariat**. The administrator may promote members to bourgeoisie or
transfer the administrator seat (the previous admin becomes bourgeoisie).

## VA settings (in Politburo)

Examples of what administrators configure (not stored in your Airtable):

- Display name and VA code
- Datasource connection (e.g. Airtable) and field maps
- Callsign **prefixes** and **suffixes** for live-flight matching
- Preferred Infinite Flight server (expert, casual, etc.)

## Reports

| Situation | Who acts |
|-----------|----------|
| IFC username already taken at registration | Reporter can file an **occupied IFC** report; platform operator investigates |
| Discord guild migration | VA **administrator** requests rebind; operator updates the guild link |

VA staff cannot seize someone else’s IFC or rebind the guild without operator involvement.

## Related

- [Registration](registration.md)
- [Discord & portal](discord-and-portal.md)
