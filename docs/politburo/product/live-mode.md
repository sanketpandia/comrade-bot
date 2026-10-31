# Live mode (Infinite Flight organizations)

Product plan for supporting Infinite Flight **Live organizations** inside a
VA: fleet visibility, flight claiming, joining, and fleet upkeep. It replaces
the manual Discord process VAs run today. Builds on
[identity-membership.md](identity-membership.md); data rules follow
[data-sources.md](data-sources.md).

Upstream capability: Infinite Flight Public API v3 (Live preview), described
in `openapi/infinite-flight/v3/`. It is a **preview** contract and may change.

## The problem today

Using AFKLM as the reference VA, Live mode is run by hand in Discord:

| Manual step | Pain |
|---|---|
| Two in-game organizations because of the per-org aircraft cap | Pilots and staff track two fleets in their heads |
| Join: follow the CEO/COO in game, wait for an invite | No record of who asked; staff chase usernames |
| "Lift one of the two orgs" rule | Staff check lifts by eye |
| Claim a leg by posting `PH-BVA EHAM-KJFK VA 1` in `#live-claiming` | Free-text, typos, no validation |
| "Search the channel for the registration and read the last claim" | Slow, error-prone, nobody knows if a claim is stale |
| Claims are only valid if you depart within 6 hours | Nobody enforces it |
| Staff react ✅ after copying the claim into the Live dashboard | Duplicate work; claims without ✅ are still valid, so ✅ means little |
| Rank restrictions still apply | Checked by memory |
| Park at a green/orange gate, don't leave an aircraft stopped for 3 days | No one notices until someone complains |

Goal: each of these becomes a validated Discord action or an automatic check,
with a few portal views for staff. Day-to-day pilot work stays in Discord.

## What exists already

- **Platform user** with Discord ↔ IFC link, proven by logbook flights, and
  an optional Infinite Flight user id. That id is what Live organizations use
  for members, crew, and flight logs, so it is the join key for everything
  below.
- **VA membership** with roles proletariat / bourgeoisie / administrator,
  exactly one administrator, one Discord guild per VA.
- **Connect Infinite Flight** tile on the portal dashboard. It runs the
  official Infinite Flight sign-in with read-only access. Today the result
  lasts only as long as the browser session, isn't kept up to date in the
  background, and can't write anything.
- Signed links from Discord to the portal, used whenever a flow needs a
  browser (OAuth sign-in is one of them).
- The live-flight and livery caches, which already give aircraft and livery
  names for display.

## Concepts

| Concept | Meaning |
|---|---|
| **Live connection** | A platform user's Infinite Flight account, linked by signing in with Infinite Flight. Belongs to the person, not the VA. Kept alive in the background until revoked. |
| **Live organization** | An in-game org (e.g. "AFKLM VA 1"). Owned in game by one IF user; members have Member / Admin / Owner roles in game. |
| **Linked organization** | A Live organization the VA administrator has attached to the VA. A VA can link **several**. |
| **Managing connection** | For each linked org, the Live connection we act through for staff-level changes (schedules, crew). Must be an in-game Owner or Admin of that org. |
| **Claim** | Our record that a pilot intends to fly a leg on a specific aircraft soon. Lives in Politburo. Can optionally be mirrored into the in-game schedule. |

### Two kinds of connection

| | Pilot connection | Staff connection |
|---|---|---|
| Who | Any registered pilot | VA administrator, or a staff member the administrator nominates |
| Access asked | Read: orgs, members, aircraft, schedules | Read **plus** schedule write |
| Used for | Prove which orgs the pilot is in; personal views ("my claims", "my flights") | Everything a pilot connection does, plus acting for the VA on its linked orgs |
| If it lapses | Pilot sees "reconnect" in Discord; personal features pause | Write-back features for those orgs pause; administrator is alerted |

**Identity rule:** a Live connection must be the **same Infinite Flight
account** as the user's registered IFC. If someone signs in with a different
IF account, we refuse the link. This keeps the IFC proof from registration
meaningful and stops one person acting through another's account.

Writes only ever happen through a staff connection whose owner holds
Owner/Admin in that org. That's Infinite Flight's own rule, and we also
gate it on the Politburo role.

## Linking organizations to a VA

Administrator, in the portal:

1. Connect their own Infinite Flight account (staff connection).
2. See every active Live organization that account belongs to, with its
   in-game role.
3. Pick one or more to link to the VA. Each gets a short label the bot uses
   (`VA 1`, `VA 2`, or a custom name).
4. For each linked org, choose the managing connection. Default: the
   administrator, if they are Owner/Admin there. Otherwise pick a VA staff
   member who has connected and is Owner/Admin of that org. This covers
   AFKLM, where the CEO and COO may each own a different org.

Rules:

- An org can be linked to **one VA only**. If someone tries to link an org
  that's already taken, it becomes a platform operator report, the same way
  occupied IFCs work.
- Unlinking an org stops all features for it and closes its open claims.
- If the managing connection's owner leaves the VA, loses in-game admin, or
  lets the connection lapse, that org drops to **read-only** and the
  administrator is told to pick another manager. Nothing is written silently
  through a stale account.
- Transferring the Politburo administrator seat doesn't move connections. The
  new administrator keeps existing managers or replaces them.

## VA feature toggles

Live mode is a VA module. Off by default.

| Toggle | Default | What it turns on |
|---|---|---|
| Live mode | Off | Fleet board, aircraft lookup, member view |
| Claims | On when Live mode is on | `/claim` and the claims feed |
| Mirror claims to Live schedules | Off | Claims create or update the in-game schedule and assign the pilot as crew. Needs a managing connection on the org. |
| Fleet upkeep alerts | On | Idle-aircraft and stale-claim notices to a staff channel |
| Join assistant | On | `/fleet join` request flow and the lift check |
| Rank rules | Off | Block claims on aircraft above the pilot's rank. Needs the VA datasource. |

Settings that go with them:

- Claim window (default **6 hours** to depart)
- Idle threshold (default **3 days** stopped)
- Claims feed channel and staff alert channel
- Lift rule: "lift at least one linked org" (AFKLM) or "lift each org you fly in"
- A short VA rules text shown by `/fleet rules`

## Pilot experience (Discord)

Command names are proposals. `/live` stays as the existing VA live-flights
view, so Live organization features sit under `/fleet`.

| Command | What it does |
|---|---|
| `/fleet connect` | Signed link to the portal → Infinite Flight sign-in → back to Discord with a confirmation |
| `/fleet join` | Checks whether you're already in each linked org. If not, files a join request to staff with your IFC username. Shows whether you meet the lift rule. |
| `/fleet list` | Fleet board: aircraft across all linked orgs, filterable by org, type, airport, and "available now" |
| `/fleet aircraft <registration>` | One aircraft: org, type/livery, where it is, state, last pilot, how long it's been idle, current claim, upcoming schedule, recent flights |
| `/claim <registration> <origin-destination>` | Claim a leg (autocomplete on registration and current airport) |
| `/unclaim` | Release your claim |
| `/claims` | Open claims, newest first; filter by org or registration |
| `/fleet rules` | The VA's Live rules text |

### Claiming a leg

`/claim PH-BVA EHAM-KJFK` replaces the free-text post. The org is implied
by the registration; no more `VA 1/2`.

Checks before the claim is accepted:

1. The registration belongs to a linked org and is in an active fleet slot,
   not hangared.
2. The origin matches where the aircraft is now (its last landing airport).
   If we can't tell, we warn instead of blocking.
3. Nobody else holds an open claim on that aircraft.
4. The pilot is a member of that org in game. If not, we point them to
   `/fleet join`.
5. The lift rule is satisfied.
6. Rank rules pass, if enabled.
7. One open claim per pilot (VA-configurable).

On success:

- The bot posts a claim card in the claims feed channel, so the channel stays
  the human-readable history. People no longer have to search it, though.
- If **mirror to Live schedules** is on, Politburo adds or updates the leg in
  that aircraft's in-game schedule and assigns the pilot as captain through
  the managing connection. The card then shows a "synced to Live" badge,
  which replaces the staff ✅.

How a claim ends:

| Ending | Trigger |
|---|---|
| **Flown** | A matching flight on that aircraft (same origin, departed after the claim) shows up in the aircraft's flight log. The flight log isn't linked to the schedule, so we match on airports and time. |
| **Expired** | No departure within the claim window. The pilot is told, staff see it in the feed, and a mirrored schedule assignment is removed. |
| **Released** | `/unclaim` by the pilot, or by staff with a reason |
| **Overridden** | Staff reassign the leg. Both pilots are told. |

"You don't have to claim before flying" still holds. If someone flies an
unclaimed aircraft, the flight just appears in its history. A flight on an
aircraft someone else had claimed shows up as a conflict for staff, and is
not blocked.

## Staff experience

### Discord (bourgeoisie and administrator)

- Release or reassign any claim, with a reason.
- Join requests land in the staff channel with the pilot's IFC username and a
  "Mark invited" button. Infinite Flight has no API for sending invites, so
  the CEO/COO still invites in game. Once the pilot appears in the org's
  member list, the request closes on its own and the pilot is told.
- Upkeep alerts:
  - Aircraft stopped longer than the idle threshold, naming the last pilot.
  - Aircraft left stopped mid-trip, or not parked after the session ended.
  - Claims that are about to expire.
- A daily or weekly fleet digest, if the VA wants one.

### Portal views

| View | Who | Contents |
|---|---|---|
| Live connections | Everyone | Your connection status, which orgs we see you in, reconnect |
| Linked organizations | Administrator | Link/unlink orgs, labels, managing connection per org, toggles and settings |
| Fleet board | Pilots and staff | Every aircraft across linked orgs on one table and a map: position, state, idle time, claim, next schedule. The map reuses the existing live map. |
| Claims | Staff | Open, expired, flown, and conflicts, with filters and an audit trail |
| Members audit | Staff | Cross-check of the VA roster against org members: in the VA but not in any org, in an org but not registered with us, missing the lift rule, in-game role vs Politburo role |
| Schedules (later) | Staff with a managing connection | Per-aircraft in-game schedule: add legs, reorder, delete, edit briefings and flight plans. Option to build legs from the VA's route list. |

Assigning roles, linking orgs, and changing toggles stay in the portal only,
the same as other admin settings.

## Roles

| Action | Proletariat | Bourgeoisie | Administrator |
|---|---|---|---|
| Connect own IF account | Yes | Yes | Yes |
| View fleet, aircraft, claims | Yes | Yes | Yes |
| Claim / release own claim | Yes | Yes | Yes |
| Release or reassign others' claims | | Yes | Yes |
| Handle join requests, see members audit | | Yes | Yes |
| Receive upkeep alerts | | Yes | Yes |
| Edit in-game schedules (later) | | If they are the org's managing connection | Yes, if they are the managing connection |
| Link orgs, pick managing connections, toggles | | | Yes |

Being Owner/Admin of an org in game doesn't grant a Politburo role. Holding
a Politburo staff role doesn't grant in-game rights either. Writes need both.

## Data handling

Infinite Flight's terms say Live data must not be stored, accumulated,
shared, or used for AI training. So:

- **We store** only what's ours: connections (tokens kept secret), linked
  orgs (id, label, manager), VA settings, claims and their outcomes, join
  requests, and alert history.
- **We don't store** fleets, positions, schedules, member lists, or flight
  logs. They're read through a short-lived cache that a background refresh
  keeps warm, the same pattern as the live map. A claim records its
  registration and airports because the claim is our record, but we don't
  keep a copy of the aircraft.
- Disconnecting an account deletes its tokens immediately. Deleting a user,
  including through ban-and-delete, deletes their connections, and any org
  they managed drops to read-only.

## Constraints and risks

- **Preview API.** Paths, scopes, and request bodies may change. The
  schedule-create and crew request formats aren't documented yet, so
  mirroring claims depends on that settling.
- **App review.** Until Infinite Flight approves our app, only the owner plus
  five invited testers can sign in. Pilot connections are blocked at scale
  until then. Staff connections (a few people per VA) can launch first.
- **No invite API.** Joining stays an in-game action. We only take over the
  tracking.
- **Flight logs aren't linked to schedules.** Detecting "flown" is a match
  on aircraft, airports, and time, so edge cases go to staff instead of being
  decided automatically.
- **Tokens rotate on every refresh.** A failed refresh means someone has to
  reconnect. We alert early instead of failing quietly.
- **Rate limits are unknown.** The fleet board and checks share one cached
  read per org, rather than one call per Discord command.

## Phasing

1. **Connections that last.** Keep Live connections between sessions and
   refresh them in the background. Enforce the identity rule. Add a
   reconnect prompt. Request write access only for staff connections.
2. **Link orgs and the fleet board.** Administrator links multiple orgs and
   picks managers. `/fleet list`, `/fleet aircraft`, and the portal fleet
   board run off the cached read.
3. **Claims.** `/claim`, `/unclaim`, `/claims`, the feed channel, the claim
   window, and automatic "flown" and "expired" outcomes. Data stays in
   Politburo only.
4. **Join and members.** `/fleet join`, the staff join queue that closes
   itself, lift checks, and the members audit.
5. **Upkeep.** Idle-aircraft and stale-claim alerts, plus the optional
   digest.
6. **Write-back.** Mirror claims into in-game schedules and crew, then the
   staff schedule editor, legs from VA routes, and rank rules from the
   datasource.

Phases 1–5 need only read access, so they can ship before the schedule-write
details settle.

## Open questions

- Should pilot connections be **required** for `/claim` (strong proof of org
  membership), or optional, with staff connections covering reads? This
  matters a lot while app review limits testers.
- One open claim per pilot, or per org?
- Should a claim be a single leg, or a short chain on the same aircraft
  (EHAM-KJFK, then KJFK-EHAM)?
- Rank rules: map rank to aircraft type in our settings, or read it from the
  VA's route list in the datasource?
- Where does the "current airport" come from when there's no recent landing?
  Options: nearest airport to the last position, or the destination of the
  last schedule.
- Should a flight on an unclaimed aircraft auto-create a claim, so the feed
  and stats stay complete?
- Should flown Live legs prefill `/log`, so a PIREP comes straight from the
  Live flight log?
