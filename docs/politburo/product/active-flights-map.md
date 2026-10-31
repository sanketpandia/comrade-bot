# Active flights map

Plan for the live map at `/maps/flights/active`. Not implemented.

The page today is a Leaflet map under the existing topbar, a full-width filter
slab pinned to the bottom, and a right-hand detail rail. The rail is meant to
stay hidden until a marker is clicked. In the browser it is always open,
because `.maps-detail-panel { display: flex }` overrides the `hidden`
attribute. The filter slab covers the southern map, the OSM attribution sits
on top of traffic, and every aircraft is the same gray chevron. Pilot-state
filters work on the server, but the list payload has no pilot state, so the
map cannot show it.

## Outcome

- The map page and the two cached flight reads are public.
- HTML on the page (filters, status, detail panel) is swapped by HTMX.
- Leaflet still draws markers. It does not own forms or the detail panel.
- The detail panel has three tabs. Only the first has data. Route and User
  stay placeholders.
- Motion and depth come from the existing Gruvbox tokens, CSS, and HTMX 2.0.6.
  No new UI library.
- The filter card, detail rail, and topbar each collapse, and one control
  clears all three so the map can fill the viewport.
- The same chrome is set by query flags, plus an `embed` mode: map and
  filters, no site header, for iframes and shared links.

## Out of scope

- Drawing a route, flight plan, or user profile. Those tabs are shells.
- Changing the flight cache, jobs, or track schema.
- Coloring markers by pilot state. That needs `pilotState` on the list
  payload, which is a separate API change.
- Replacing OpenStreetMap tiles, adding a flight list, or redesigning the
  topbar. On this page the topbar only hides and shows.
- Making other `/api/v1/game` routes public. Active sessions stay authenticated.

## Public reads

These are cache hits. A miss returns 503 and does not call Infinite Flight.

| Surface | Today | After |
|---|---|---|
| `GET /maps/flights/active` | `UISessionAuth` | Public |
| `GET /maps/flights/active/servers` | `UISessionAuth` | Public |
| `GET /api/v1/game/flights/active/{server}` | API key or session cookie | Public |
| `GET /api/v1/game/flight/active/{flightId}` | API key or session cookie | Public |
| `GET /api/v1/game/sessions/active` | API key or session cookie | Public |

Move the two UI routes out of the session group in
`services/politburo/internal/transport/http/server.go`. In
`AuthenticateAPI`, skip auth for those three game GETs only. Leave every other
`/api/v1` path as it is.

Update the OpenAPI sources for those three operations: drop `security` and the
401 response, then regenerate. Do not hand-edit generated Go.

The single-flight record already includes callsign, position, aircraft,
livery, username, virtual organization, and Infinite Flight user id. That is
the same class of data Infinite Flight shows on a public map, so it can be
public. The future User tab must not add Discord identity, VA membership, or
anything we store. If that tab needs private data later, it gets its own
authenticated partial. Do not widen this public record to carry it.

Signed-link login from Discord `/live` can keep working. It is no longer
required to open the map.

No new rate limiter in this pass. `RateLimit` exists and is not mounted on
these routes. Add a read limit only if public traffic becomes a problem.

## HTMX boundary

HTMX 2.0.6 is already at `/static/js/htmx.min.js`. Use it for filters, status,
polling, and the detail panel. Keep `flight-map.mjs` for markers, zoom, and
callsign tooltips.

One flights request should refresh both the status line and the markers:

1. The filter form `hx-get`s an HTML partial.
2. The partial includes the count, a human cache time, errors, and a
   `<script type="application/json" id="maps-tracks">` island with the track
   array the map already understands.
3. `active-flights.mjs` listens for `htmx:afterSettle` on that partial and
   calls `map.addFlights`. It does not `fetch` the flight list itself.
4. Poll with `hx-trigger="every 60s"` on the same form, plus `change` and
   `keyup changed delay:300ms` on the search fields. Drop the hand-rolled
   `setInterval` and debounce.
5. `hx-replace-url` keeps the current query (`serverId`, `pilotState`,
   `callSign`, `userName`, `vaCode`, `showCallsigns`, and the chrome flags
   below). The map `<div>` is `hx-preserve` so a swap never destroys the
   Leaflet instance.

Server options stay the existing `hx-get` / `hx-trigger="load"` on the
`<select>`.

Marker clicks call `htmx.ajax` against a new UI partial, not
`fetch` + client-side `textContent` fills. `flight-detail-panel.mjs` shrinks
to tab wiring that HTMX does not already cover, or goes away if the tabs are
plain `hx-get` links.

Suggested partials, all public, all HTML:

| Partial | Role |
|---|---|
| `GET /maps/flights/active/flights` | Status + JSON island for the current filters |
| `GET /maps/flights/active/flights/{flightId}` | Detail panel body for the selected flight |
| `GET /maps/flights/active/flights/{flightId}?tab=route` | Route placeholder |
| `GET /maps/flights/active/flights/{flightId}?tab=user` | User placeholder |

The list partial reads the same cache the JSON API reads. It does not add a
second code path for filtering. The JSON API stays for the bot and other
clients.

`focusFlight` already exists and is unused. After a successful detail swap,
call it so the selected chevron is highlighted. Do not `fitBounds` again
after the first load.

## Detail panel

Three tabs, in this order:

1. **Flight** — the data we already render: callsign, username, virtual
   organization, aircraft, livery, session, speed, altitude, vertical speed
   with climb / descend / level, track, pilot state, connection, FPL sync,
   last report. Drop the in-panel page 1 / page 2 pager. Identifiers (flight
   id, user id, raw coordinates, pilot-state code, session id) sit at the
   bottom of this tab in a quieter block, not on a fake second page.
2. **Route** — placeholder copy only. No map line, no flight-plan fetch.
3. **User** — placeholder copy only. No profile, logbook, or membership.

The panel starts closed. Opening adds `maps-shell--detail-open` and slides
the rail in. A close control removes it. Selecting another marker replaces
the body without closing. A 404 from a flight that left the cache swaps in
“This flight is no longer active.” Empty, loading, and error states are
server HTML, not toggled nodes in the client.

Fix the always-open rail first: `[hidden] { display: none !important; }` on
this page, and do not set `display: flex` in a way that wins over `hidden`.
Prefer an open class over the `hidden` attribute once the slide animation
needs a real box.

## Collapsible chrome

The map is the page. Filters, the detail rail, and the topbar sit on top of
it and must be able to leave.

Each piece collapses on its own. A single **Map only** control collapses all
three. Collapsing is local: a class on `body` or `#maps-shell`, not an HTMX
round trip. A link can force the same classes on first paint via query
flags, so an embed never flashes the topbar.

The page handler reads the flags and renders the matching classes in the
first HTML response. Do not hide chrome only after JavaScript runs.

### Flags

Absent means “use the default”. `1` shows the piece. `0` hides it. Any other
value is ignored and the default applies.

| Param | `1` | `0` |
|---|---|---|
| `header` | Topbar visible | Topbar omitted |
| `filters` | Filter card open | Card collapsed to the **Filters** button |
| `detail` | Rail open for the selected flight, if one is selected | Rail stays closed. Marker clicks do not open it. |

Example with no site header and no rail, filters still usable:

`/maps/flights/active?serverId=casual&header=0&detail=0`

### Embed mode

`mode=embed` is the preset for iframes and other embedded maps: the map, the
full filter card, and nothing else.

| Piece | Embed |
|---|---|
| Topbar | Off. No control in the frame restores Dashboard, Admin, or the profile button. |
| Filter card | On, with every filter. |
| Detail rail | Closed until a marker click, then available. |
| **Map only** | Hidden. The frame should not be able to grow site chrome. |

An explicit flag overrides the preset. `mode=embed&filters=0` is a bare map.
`mode=embed&detail=0` is filters plus map, and marker clicks do not open the
rail.

`hx-replace-url` and filter changes keep `mode` and any chrome flags. Dropping
them would turn an embed back into the full page on the next poll.

### What a click remembers

`localStorage` remembers a manual collapse only when the URL has no `mode`
and no chrome flag. A shared or embedded link always wins over that saved
choice. Clearing the flags returns to the saved choice, then to the default
(topbar on, filters open, rail closed).

| Piece | Expanded | Collapsed |
|---|---|---|
| Filter card | Floating card, top-left | One **Filters** button in that corner. Status count can sit on the button. |
| Detail rail | Side rail, or bottom sheet under 768px | Closed. A marker click opens it again. The rail’s own close control only closes the rail. |
| Topbar | Existing site bar | Hidden. **Map only** brings it back along with the other chrome. |

**Map only** is a small map-corner button, not a setting buried in the card.
When chrome is hidden by that button, it remains and restores the chrome. It
is omitted in `mode=embed`. On a full map the only other control is Leaflet
zoom, plus the **Filters** button when the card was collapsed on its own.

After the CSS transition ends, call `map.invalidateSize()` so tiles and
marker positions fill the freed space. `fitBounds` padding shrinks to the
zoomer only while the card and rail are collapsed.

Polling keeps running while the card is collapsed. The form stays in the
DOM, hidden, so `hx-trigger="every 60s"` still refreshes markers.

## Visual treatment

Stay on the tokens in `base.css` (`--bg-app`, `--bg-surface`,
`--bg-surface-elevated`, `--text-*`, `--accent-primary`, `--status-*`,
`--radius-*`, `--shadow-surface`). Do not add a second palette.

The bottom slab is the layout problem. Turn the filters into a floating card
anchored top-left under the topbar, with a width that fits the controls
instead of stretching across the map. Map the same fields:

- Status: `374 flights` and a short relative cache age (`2 min ago`), with
  the UTC timestamp in a `title` or `<time datetime>`.
- Server select.
- Callsign, username, VA code.
- Pilot-state checkboxes and “Show callsigns” as the same checked-chip
  treatment already in `maps.css`.

The card uses `--bg-surface` at high opacity, a 1px `--border-color`, and a
deeper shadow than `--shadow-surface` (stacked soft shadow, still black at
low alpha). The detail rail gets the same shadow on its left edge and keeps
the orange tab underline.

Lift Leaflet zoom and attribution off the card and off the open rail. Give
the map `padding` in `fitBounds` that matches the card so the first fit does
not hide markers under it.

Pilot-state colors already exist as `--status-active`,
`--status-away-in-flight`, `--status-away-parked`, and
`--status-in-background`. Use those on the filter chips. Marker coloring
waits until the list payload includes state.

Below 768px the card stacks under the topbar at full width with comfortable
padding, and the detail rail becomes a bottom sheet (max ~42vh) instead of a
side column. That breakpoint already exists; keep it.

## Motion

No animation library. HTMX 2.0.6 already supports
`hx-swap="innerHTML transition:true"`, which uses the View Transitions API
where the browser has it. Use that for tab body swaps and status updates.
Keep the duration short (about 160–200ms).

CSS only, on top of that:

- Detail rail: `transform` + `opacity`, ease-out, ~200ms.
- Filter card and topbar: the same duration, height or translate, so they
  clear the map instead of popping.
- Tab underline: the existing inset shadow can transition.
- Filter card: no entrance animation on first paint. Hover and focus use the
  same border mix already on the chips.
- `prefers-reduced-motion: reduce` disables the rail slide, chrome collapse,
  and view transitions. Content still shows or hides.

Do not animate individual aircraft. A full marker rebuild of several hundred
div icons will stutter if each one transitions.

## Work order

1. Un-break the detail rail (`hidden` vs `display`) and restyle the filter
   card, rail, attribution, and shadows. Add collapse for the card, the
   rail, the topbar, and **Map only**, including `invalidateSize`. Honor
   `mode=embed` and `header` / `filters` / `detail` in the first HTML
   response. This is the visible fix.
2. Make the four routes public and update the OpenAPI bundle plus auth tests.
3. Move status, polling, and URL sync onto HTMX. Point the map script at the
   JSON island.
4. Replace the client-rendered detail panel with the three-tab HTMX partials.
   Flight is filled. Route and User are placeholders.
5. Add the rail slide, tab transition, and reduced-motion rules.

## Checks

- Logged out: the map page, server options, flight list, and one flight
  detail all return 200 when cache is warm, and 503 when it is not.
- Logged out: `/api/v1/user/me` is still 401.
- A bot API key still works on the two flight reads.
- Filter changes update the status line, the markers, and the query string
  without reloading the map.
- The rail is closed on first paint, opens on marker click, and shows
  Flight / Route / User. Route and User do not call new APIs.
- Collapsing the card, the rail, or the topbar leaves a control that brings
  it back. **Map only** hides all three and the map fills the viewport.
  Markers still refresh. A manual collapse survives a reload only when the
  URL does not set chrome.
- `mode=embed` renders with filters and no topbar, and the topbar cannot be
  restored from inside the page. `header=0`, `filters=0`, and `detail=0`
  apply on the first response, and a filter change does not strip them.
  `mode=embed&filters=0` is a bare map.
- `prefers-reduced-motion` leaves the rail and the collapse controls usable
  with no slide.
- `go test` for UI render and API auth; the existing map render test must
  expect a public page and three tabs.
