# Politburo UI styling guide

How the server-rendered Politburo UI (`services/politburo/internal/ui/`) should
look and how its CSS should be organised. The live flights map
(`/maps/flights/active`) established the glass panel look; Dashboard, Live Mode,
Admin, and the map now share the same tokens and components.

Run `go test ./internal/ui/...` after CSS or template changes; `TestCSSUsesTokens`
and `TestTemplatesAvoidRemovedUIClasses` guard the contract.

## Principles

1. **Tokens first.** Colours, shadows, radii, blur, motion, and z-index come
   from custom properties in `:root`. A raw hex or `rgba()` outside the token
   block is a bug.
2. **Components before page CSS.** Reach for a shared component class
   (panel, card, button, field, badge, table) before writing a page-local rule.
   Page CSS should mostly hold layout: positioning, grids, and page-specific
   sizing.
3. **One way to do each thing.** One focus ring, one icon button, one input
   style, one badge. Variants are modifiers on the component, not new blocks.
4. **Server-rendered and HTMX-friendly.** Styling must not depend on JS state
   beyond classes that the server can also render (`hidden`, `aria-expanded`,
   `aria-selected`, `aria-current`, shell modifier classes).
5. **Glass is the default surface.** App pages use `--app-backdrop` (CDN image
   `/static/brand/background.jpg` with a scrim). Panels, side nav, tables, and
   the sticky topbar are glass; inputs, selects, and icon buttons stay on the
   solid control surface for legibility. The map page keeps the Leaflet canvas as
   its backdrop (`.maps-page`).

## Assets

Brand images live under `static/brand/` and are served at `/static/brand/*` with
long cache headers (see `Brand()` in the HTTP UI handler). Reference them with
root-relative URLs in `tokens.css` (for example `--app-backdrop-image`), never
inline `style=` in templates.

The backdrop photo is from Unsplash and needs attribution wherever it shows.
The `photo-credit` partial renders it in the app footer (`base.html`, every page
except the map) and in `public-footer`. If you replace the image, update that
partial; the dashboard and contact handler tests assert the credit is present.

## File layout

| File | Contents | Loaded by |
|---|---|---|
| `tokens.css` | `:root` custom properties only | every layout |
| `base.css` | Reset, app backdrop, topbar, page shell, `.page-header`, `.section-layout` | every layout |
| `components.css` | Shared components listed below | every layout |
| `public.css` | Public page layout only | `public.html` |
| `maps.css` | Map layout, Leaflet overrides, map-only elements | map page |

Order in `<head>`: `tokens.css` → `base.css` → `components.css` → page CSS.
Third-party CSS (Leaflet) loads before `maps.css` so overrides win without
`!important`.

## Tokens

### Palette (existing)

Gruvbox dark. Keep these names; they are already used everywhere.

| Token | Value | Use |
|---|---|---|
| `--bg-app` | `#282828` | Page background |
| `--bg-surface` | `#1d2021` | Solid panels, topbar |
| `--bg-surface-elevated` | `#3c3836` | Inputs, icon buttons, raised controls |
| `--text-primary` | `#ebdbb2` | Body and values |
| `--text-secondary` | `#d5c4a1` | Secondary values, section titles |
| `--text-muted` | `#a89984` | Labels, hints, timestamps |
| `--border-color` | `#3c3836` | Borders on solid surfaces |
| `--accent-primary` | `#fe8019` | Primary action, focus, links, selection |
| `--accent-text` | `#282828` | Text on solid accent fills |
| `--status-active` | `#8ec07c` | Active pilot, success, high confidence, climbing |
| `--status-away-in-flight` | `#d3869b` | Away in flight |
| `--status-away-parked` | `#a89984` | Away parked |
| `--status-in-background` | `#665c54` | In background |
| `--status-error` | `#fb4934` | Errors, low confidence, descending |

Add `--status-warn: var(--accent-primary)` so "warning" reads as intent rather
than reusing the accent name. Replace the hard-coded `#c0392b` in
`public.css` with `--status-error`.

### Glass and backdrop

| Token | Use |
|---|---|
| `--app-backdrop` | Full-page background on authenticated pages (scrim + photo + fallback) |
| `--app-backdrop-image` | `image-set()` of `/static/brand/background.webp` and `.jpg` |
| `--glass-bg` | Default panel fill |
| `--glass-bg-strong` | Topbar, popovers, row menus |
| `--glass-tile` | Cards, tables, notices inside a panel |
| `--glass-head` | Table header row |
| `--glass-row-hover` / `--glass-row-selected` | Interactive table rows |
| `--glass-border` | Hairlines on glass |
| `--glass-blur` | `backdrop-filter` value |

Always pair `backdrop-filter` with `-webkit-backdrop-filter` (Safari).

### Elevation

One shadow scale for solid and glass surfaces. Replaces `--shadow-surface` and
the ad-hoc `rgba()` shadows in `maps.css`.

| Token | Value | Use |
|---|---|---|
| `--shadow-highlight` | `inset 0 1px 0 rgba(255, 255, 255, 0.05)` | Top-edge sheen on glass and tiles |
| `--shadow-1` | `0 1px 3px rgba(0, 0, 0, 0.2)` | Cards and tiles inside a panel |
| `--shadow-2` | `0 1px 2px rgba(0,0,0,.22), 0 6px 16px rgba(0,0,0,.26)` | Raised controls, active tabs, tooltips |
| `--shadow-3` | `0 1px 2px rgba(0,0,0,.22), 0 6px 16px rgba(0,0,0,.26), 0 18px 44px rgba(0,0,0,.34)` | Floating panels, popovers |

A glass panel uses `box-shadow: var(--shadow-highlight), var(--shadow-3)`.

### Shape, motion, layering

| Token | Value | Notes |
|---|---|---|
| `--radius-sm` | `0.5rem` | Controls, cards, tables (existing) |
| `--radius-md` | `0.75rem` | Panels (existing) |
| `--radius-pill` | `999px` | Badges, avatar, count bubbles (replaces literal `999px`) |
| `--duration-fast` | `0.15s` | Hover and colour changes |
| `--duration-base` | `0.22s` | Panel slide, expand, collapse |
| `--ease-out` | `ease-out` | Default easing |
| `--focus-ring` | `2px solid var(--accent-primary)` | Used by the single focus rule |
| `--z-panel` | `450` | Detail rail |
| `--z-chrome` | `500` | Filters and map chrome |
| `--z-popover` | `600` | Dropdown panels |
| `--chrome-inset` | `12px` | Gap between floating chrome and viewport edge |

### Type scale

The map settled on a small set of sizes. Name them and stop inventing new
ones (`0.58rem`, `0.82rem`, `0.78rem`, and similar should collapse into these).

| Token | Size | Use |
|---|---|---|
| `--text-2xs` | `0.6rem` | Badges, source tags |
| `--text-xs` | `0.68rem` | Eyebrow labels, section titles, tabs |
| `--text-sm` | `0.8rem` | Table cells, field labels, panel body |
| `--text-md` | `0.85rem` | Inputs, buttons, card values |
| `--text-lg` | `1.15rem` | Tile and panel headings |
| `--text-xl` | `1.35rem` | Page titles, hero values |
| `--text-2xl` | `1.75rem` | Public marketing headings |

Use `rem` for type and spacing. `px` is only for hairlines and `--chrome-inset`.

## Surfaces

Pick the surface by what is underneath it.

| Surface | Fill | Border | Shadow | When |
|---|---|---|---|---|
| App backdrop | `--app-backdrop` | none | none | Body on Dashboard, Live, Admin |
| Map page | Leaflet tiles | none | none | `.maps-page` only |
| Glass panel | `--glass-bg` + blur | `--glass-border` | `--shadow-glass-panel` | `.panel`, side nav |
| Glass topbar | `--glass-bg-strong` + blur | `--glass-border` | highlight | Sticky `.topbar` |
| Tile | `--glass-tile` | `--glass-border` | highlight + `--shadow-1` | Cards, `.data-table`, `.notice` |
| Popover | `--glass-bg-strong` + blur | `--glass-border` | `--shadow-glass-panel` | `.row-menu`, multiselect panel |
| Control | `--bg-surface-elevated` | `--border-color` | none | `.input`, `.select`, `.btn`, `.icon-btn` |

Inside a glass panel, dividers use `--glass-border`, never `--border-color`.

### App page skeleton

Authenticated pages (Dashboard, Live Mode, Admin) follow this structure:

```html
<header class="page-header">
  <h1 class="page-header__title">…</h1>
  <!-- optional: <div class="field page-header__aside">…</div> -->
</header>
<div class="section-layout">
  <nav class="tabs tabs--nav">…</nav>
  <section class="panel stack">…</section>
</div>
```

Use `.stack` instead of `.section-layout` when there is no side nav (for example
the Dashboard live-flights tile).

## Components

Shared class names use BEM: `block__element--modifier`.

### Panel `.panel`

[`.maps-filters`, `.maps-detail-panel`, Live fleet panels]

Glass by default: padding `0.85rem 1rem`, `--radius-md`. Header row pattern:

```
<icon> <title / count> .................. <icon button>
```

Use `display: flex; align-items: center; gap: 0.5rem` and push the trailing
control with `margin-left: auto`.

### Card grid `.card-grid` and `.card`

[`.maps-detail-info__cards`, `.maps-detail-card`]

Two equal columns of equal-height cards for headline facts:

```css
grid-template-columns: repeat(2, minmax(0, 1fr));
grid-auto-rows: 1fr;
gap: 0.5rem;
```

A card is an eyebrow label over a single-line value that truncates with an
ellipsis. Missing values render as `—`, never an empty cell.

### Key-value table `.kv-table`

[`.maps-detail-table`]

A real `<table>` with `<th scope="row">` labels (40% width, muted) and `<td>`
values. Tile surface, rounded via `border-collapse: separate` plus
`overflow: hidden`, `--glass-border` between rows, no border on the last row.
Use it for any list of metrics; do not build these from `<ul>` or divs.

### Data table `.data-table`

[Live Mode fleet, schedules, flights]

Columnar lists inside a `.panel`. Tile surface (`--glass-tile`), glass borders,
`--glass-head` on `thead`, row hover/selection via `--glass-row-hover` and
`--glass-row-selected`. Wrap in `.data-table-wrap` for horizontal scroll.

### Notice `.notice` `.notice--error`

Standalone callouts (especially HTMX/JS errors). Use instead of bare
`.panel-message--error` outside a panel.

### Panel header modifiers

- `.panel__header--wrap` — title and filter on one row (schedules, flights).
- `.panel__filter` — min-width for aircraft `<select>` filters.

### Section `.section` with `.section__title`

[`.maps-detail-section`]

Uppercase `--text-xs` title in `--text-secondary` above a card grid or table,
with `0.4rem` gap. Stack sections with `0.9rem` gap.

### Eyebrow label `.eyebrow`

[`.maps-detail-info__label`, `.maps-detail-debug__label`]

`--text-xs`, uppercase, `letter-spacing: 0.05em`, `--text-muted`. One class;
the map currently has three copies.

### Badge `.badge` `.badge--{intent}`

[`.maps-detail-badge`]

Pill, `--text-2xs`, uppercase, tinted border. Intents map to tokens, not to
features:

| Modifier | Colour |
|---|---|
| `--success` | `--status-active` |
| `--warn` | `--status-warn` |
| `--error` | `--status-error` |
| `--muted` | `--text-muted` |

Pilot state badges (`--pilot-active`, `--pilot-away-flight`,
`--pilot-away-parked`, `--pilot-background`) stay as domain modifiers because
they carry their own palette. Climb/level/descend should become
`--success`/`--muted`/`--error` instead of separate classes.

### Confidence indicator `.confidence--{high|medium|low}`

[`.maps-confidence`]

Icon-only signal bars in a tinted circle. Colour and the number of filled bars
both encode the level, so it does not rely on colour alone. Always include
`role="img"`, an `aria-label` with level and percentage, and a `title` tooltip.
Thresholds: high ≥ 0.7, medium ≥ 0.4, low below that.

### Buttons

| Class | Today | Look |
|---|---|---|
| `.btn` | `.tile__link`, `.public-button--ghost` | Control surface, border, `--text-md` |
| `.btn--primary` | `.maps-filters__apply` | Accent-tinted fill and border on dark surfaces |
| `.btn--solid` | `.public-button` | Solid accent fill, `--accent-text` |
| `.btn--block` | `.maps-filters__apply` | Full width |
| `.icon-btn` | `.maps-detail-panel__close`, `.maps-filters__header-collapse`, `.maps-filters__icon-toggle`, `.topbar__profile` | `1.75rem` square, control surface, centred `1rem` SVG |
| `.icon-btn--round` | `.topbar__profile` | `--radius-pill` |
| `.icon-btn--glass` | `.maps-filters__icon-toggle` | Glass-strong fill when floating alone over the map |

Hover on any button: text to `--text-primary`, border mixed 45% toward
`--accent-primary`.

### Form fields `.field`, `.input`, `.select`

[`.maps-filters__field`, `.admin-field`, `.public-form__input`]

Label above control, `0.35rem` gap, label in `--text-sm` muted. Inputs and
selects share one rule: control surface, `--radius-sm`, padding
`0.5rem 0.65rem`, `--text-md`. There are three copies of this today.

### Multiselect `.multiselect`

[`.maps-multiselect`, behaviour in `static/js/maps/maps-multiselect.mjs`]

Use instead of `<select multiple>`, which renders as a list box. The trigger
looks like a `.select`; the panel is a popover of checkbox rows. The summary
reads "Any …", the single label, "N selected", or "All …". Closes on outside
click and Escape. Checkboxes keep the field `name` so HTMX and plain form
submission both work. When this becomes shared, move the JS to
`static/js/ui/multiselect.mjs` and read labels from the DOM instead of a
hard-coded map.

### Toggle chip `.check-chip`

[`.maps-filters__check`]

Checkbox inside a bordered label. When checked (`:has(input:checked)`) the
border and fill tint toward the accent.

### Collapsible `.collapsible`

[`.maps-detail-debug`]

Native `<details>`/`<summary>` on a tile surface, styled like a section title
with a chevron that rotates when open. Hide the default marker
(`::-webkit-details-marker`). Use it for secondary information such as
debugging keys.

### Copyable value `.copy-key`

[`.maps-detail-debug__key`]

A `<button data-copy="…">` showing an eyebrow label over a monospace value.
Cursor is `copy`. One delegated click handler writes to the clipboard and adds
`--copied` or `--failed` for 1.2s, which shows a "Copied" or "Copy failed" tag.
The handler should live in a shared module so any page can use it.

**Source tags.** Prefix identifiers with their origin so nobody pastes an
Infinite Flight ID into a database query or the other way round:

| Tag | Meaning | Colour |
|---|---|---|
| `IF` | Infinite Flight identifier (flight, user, session) | `--text-secondary` |
| `CB` | Comrade Bot database identifier (VA ID, user ID, …) | `--accent-primary` |

Copy the raw value only; the tag is display-only.

### Tabs `.tabs`

[`.maps-detail-panel__tab`, `.tabs--nav` on Admin and Live]

Active state is `aria-selected="true"` (tab widgets) or
`aria-current="page"` (navigation). Side nav (`.tabs--nav`) is a glass panel;
the current item uses a tile pill and accent colour.

### Row menu `.row-menu`

Context menu for selectable table rows. Popover glass surface; position with JS.

### Links

Inline data links (coordinates, external references) use `--accent-primary`
with a dashed underline that turns solid on hover, and `tabular-nums` for
numbers. External links open in a new tab with `rel="noopener noreferrer"`.

## Content conventions

- **Missing values:** `—`.
- **Times:** `HH:MM:SS UTC` for display; put the RFC 3339 value in
  `<time datetime>` and `title`. Add a relative hint ("12 min ago") next to
  times people scan.
- **Enums:** humanise snake_case on the server ("Awaiting AP+ resumption"),
  not in templates or CSS.
- **Coordinates:** four decimal places, linking to OpenStreetMap
  (`https://www.openstreetmap.org/?mlat={lat}&mlon={lon}#map=10/{lat}/{lon}`).
- **Counts:** show the number in the panel header ("12 flights"); show only
  the number in a count bubble on collapsed controls.

## Interaction and accessibility

- **One focus rule.** In `base.css`:
  `:where(a, button, input, select, textarea, summary, [tabindex]):focus-visible { outline: var(--focus-ring); outline-offset: 2px; }`.
  Remove the per-component copies (there are about fifteen today).
- **Reduced motion.** One block under `prefers-reduced-motion: reduce` that
  disables transitions on panels, popovers, and chips. Do not repeat it per
  component.
- **Icon-only controls** need `aria-label`; decorative SVGs get
  `aria-hidden="true"`.
- **Expand/collapse controls** keep `aria-expanded` in sync, and the label
  says what the click will do ("Collapse filters").
- **Floating panels** must not cover map controls. When a rail opens, shift
  map controls with a shell modifier class rather than raising z-index.

## Status

The token split (`tokens.css`, `components.css`), map shared components, glass
Dashboard/Live/Admin shell, CDN backdrop, and CSS guard tests are implemented.
Remaining cleanup: migrate public pages fully onto `.btn`/`.field`, and trim
legacy map-specific class names where templates already use shared components.

## Checklist for new UI

- [ ] No raw colours, shadows, or blur values outside `tokens.css`; `go test ./internal/ui/...` passes.
- [ ] Uses an existing component class or adds a BEM modifier to one.
- [ ] Follows the app page skeleton (`.page-header`, `.section-layout` or `.stack`, `.panel`).
- [ ] Errors use `.notice notice--error` when rendered outside a panel.
- [ ] Missing values show `—`; times show UTC with `<time datetime>`.
- [ ] Icon-only controls have `aria-label`; `aria-expanded`/`aria-selected`
      stay in sync.
- [ ] Works with HTMX swaps (delegated listeners, no per-node init that a swap
      would drop).
- [ ] Brand images referenced via `/static/brand/...` in tokens, not inline styles.
- [ ] Checked at mobile width, or the desktop-only tradeoff is written down.
