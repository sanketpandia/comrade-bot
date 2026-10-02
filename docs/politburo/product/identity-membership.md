# Identity and membership

Decisions from the identity discussion. Implementation details (migrations,
OpenAPI) wait until this page is stable.

Roles use the Politburo theme in the product. Spelling: **proletariat**,
**bourgeoisie**, **administrator**.

## Two layers

| Layer | What it is | Uniqueness |
|---|---|---|
| Platform user | One person: Discord account ↔ Infinite Flight Community (IFC) username | One Discord ID, one IFC ID, globally |
| VA membership | That person in a specific VA, with a role | Many VAs per user; one role per VA |

Authentication is the Discord link. The bot sends `X-Discord-User-Id` (and
server id). The portal session is minted from that user. IFC is the game
identity we attach, not a second login.

A user can be proletariat in one VA and bourgeoisie in another. There is no
platform-wide VA role. Platform operator (bans, deletes) is a separate
capability, not a VA role.

```text
Discord user ──1:1── Politburo user ──1:1── IFC username
                         │
                         └──< memberships >── Virtual airline (1:1 Discord guild)
```

## Platform user

### Registration

Surface: Discord (`/register`). Must happen before `/initserver` or a VA
join.

1. Discord identity is already known from the interaction.
2. The user enters their **IFC username**.
3. We prove they own it: they enter an origin–destination (ICAO–ICAO) that
   matches **any of their last three** Infinite Flight logbook flights. We
   never ask for IF password or email.
4. We store the link. Discord ID and IFC ID are both unique.

`/register` stays the only join command. If they are already a platform user
and the current guild is a VA they have not joined, the same command links
them (default proletariat, our callsign for live matching). Roster match
against Airtable is by **IFC**, not callsign — see
[airtable-datasource.md](airtable-datasource.md). The leftover `/membership`
command stays dropped.

If the IFC username is already linked to another Discord account, registration
does not steal it. See [Occupied IFC and reports](#occupied-ifc-and-reports).

### What we store on the user

Ours: Discord ID, IFC username, optional IF API user id once resolved, active
flag. Drop `otp` when we touch the table. Banned Discord IDs live on a
separate list so a deleted user cannot immediately re-register.

## VA / Discord server

One virtual airline is one Discord guild, forever. We do not support extra
guilds on the same VA. If the community moves servers, that is a **migrate
Discord server** report for the platform operator, who rebinds
`discord_server_id`. It is not self-service.

The person who runs `/initserver` must already be a platform user and a
Discord administrator of that guild. They become the **sole administrator**
of the VA.

After init, Politburo administrator is independent of Discord’s Administrator
permission. Losing Discord admin does not dethrone them; transferring
administrator inside Politburo does.

### Exactly one administrator

- At most one membership with role `administrator` per VA.
- The administrator assigns bourgeoisie and proletariat roles.
- The administrator may **transfer** administrator to another member of that
  VA. The transfer is one step: the target becomes administrator and the
  previous administrator becomes **bourgeoisie**. They must not remain
  administrator.
- There is no “second admin.” There is no way to have zero admins except
  deleting the VA (not in scope yet).

### Roles

| Role | Theme | Job |
|---|---|---|
| `proletariat` | Pilots | Fly, file PIREPs, use pilot Discord commands, open the portal as a member |
| `bourgeoisie` | Managers | Staff work: logbook, event ops, troubleshooting. May change proletariat callsigns and remove proletariat. Cannot assign roles, transfer administrator, or change VA settings |
| `administrator` | The one admin | Datasource, matching configs, role assignment, admin transfer, destructive VA settings. Can also do everything bourgeoisie can |

Default role on join: **proletariat**. We collect **our** callsign at join
for live-flight matching. The Airtable roster row is found by IFC. Callsign
changes are not self-service.

Store these strings as the canonical enum. Do not keep a parallel
`pilot` / `staff` / `admin` mapping in the rewrite.

## VA settings (ours, in Postgres)

Not Airtable. These exist so *we* can recognize the VA in the live game and
talk to the VA’s database.

| Setting | Why |
|---|---|
| Display name, code | Identity of the VA |
| Datasource | Airtable mapper first; RDBMS later. Credentials and field maps |
| Callsign prefixes | Live flights whose callsign **starts with** these belong to the VA |
| Callsign suffixes | Live flights whose callsign **ends with** these belong to the VA |
| Preferred Infinite Flight server | Default world for `/live` and the VA map (e.g. expert / casual) |

Prefix/suffix matching is how we spot people flying the VA callsign even
before (or without) a Politburo membership. Membership matching (this user’s
IFC username is in the air) is a second signal. Both should be available;
settings decide which the VA wants.

Other similar configs (flight modes, livery maps) stay VA-scoped and will be
listed when we plan those features. They are not identity.

## Reports (platform operator queue)

Reports are not a VA bourgeoisie action. Types:

| Kind | Who files | What the operator may do |
|---|---|---|
| Occupied IFC | Anyone blocked at `/register` | Ban occupant Discord ID, delete that user, free the IFC |
| Migrate Discord server | Administrator (or later a general “report issue”) | Rebind the VA to a new guild id. Still one guild per VA |

### Occupied IFC, at registration

If IFC `X` is already linked:

- Tell the user it is taken. Do not reveal the occupying Discord account.
- Warn that claiming someone else’s IFC is grounds for a ban.
- Offer **Report this username**. The report records: reporter Discord ID,
  claimed IFC, optional note, timestamp. It does not change the link.

After a takedown, the reporter proves ownership again with **any of their
last three** logbook flights, then registers.

Do **not** ban the IFC username itself. It belongs to the Infinite Flight
player, who may be the reporter. A banned Discord ID cannot register again.
A new Discord account stealing the same IFC is a new report.

### Ban, export, then vanish

When the operator deletes a banned user:

1. Export a copy of that user’s Politburo record (identity + memberships /
   callsigns / roles) to an operator-only archive.
2. Delete the user. Memberships and callsigns **vanish** from every VA. The
   IFC id is free. Nothing remains on the VA member list for that person.

The archive is for disputes and forensics. It is not VA-visible history.

### Guardrails

- Rate-limit reports per Discord user.
- Last-three-flights proof on the reporter when they retry register after a
  takedown.
- A VA cannot seize an IFC or rebind its own guild.

## Surfaces

| Job | Discord | Portal |
|---|---|---|
| Register / prove IFC | `/register` | No |
| Join current VA | `/register` (link step) | No |
| Status | `/status` | Session shows current VA + role |
| Init VA | `/initserver` | Rest of VA settings |
| Assign roles, transfer administrator | No (too easy to get wrong in a modal) | Administrator |
| Change proletariat callsign, remove proletariat | No | Bourgeoisie or administrator |
| Datasource, prefixes/suffixes, preferred server | No | Administrator |
| Report occupied IFC | From the register failure | Operator queue at `/operator/reports` (SSR); optional JSON `GET/POST /api/v1/operator/reports*` for API-key or session operators—no Discord bot client yet |
| Report guild migration | Report issue | Operator rebinds guild |
| Ban + delete (export, then vanish) | No | Operator only |

Signed-link sessions must eventually carry `user id`, Discord id, current
guild’s VA, and **role in that VA**. Today they store Discord ids and do not
resolve a VA.
