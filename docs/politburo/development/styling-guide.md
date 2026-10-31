# Politburo UI styling guide

How the server-rendered Politburo UI (`services/politburo/internal/ui/`) should
look and how its CSS should be organised. The live flights map
(`/maps/flights/active`) is the reference implementation: its glass panels,
cards, key-value tables, and status chips are the target look for every page.

The [Migration plan](#migration-plan) at the end lists the work needed to bring
the rest of the UI in line. That work is tracked separately; this document is
the contract it should converge on.

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
5. **Glass only where something floats.** Translucent, blurred surfaces are
   for chrome layered over live content (map panels, popovers, sticky bars).
   Normal page content uses solid surfaces.

## File layout

### Today

| File | Contents |
|---|---|
| `static/css/base.css` | `:root` palette, reset, topbar, layout shells, dashboard `tile`, admin tabs/forms |
| `static/css/public.css` | Public pages: stage, buttons, contact form, loader |
| `static/css/maps.css` | Everything on the live map, including glass tokens scoped to `.maps-page` |

### Target

| File | Contents | Loaded by |
|---|---|---|
| `tokens.css` | `:root` custom properties only | every layout |
| `base.css` | Reset, typography, focus ring, `[hidden]`, app shell, topbar | every layout |
| `components.css` | Shared components listed below | every layout |
| `public.css` | Public page layout only | `public.html` |
| `maps.css` | Map layout, Leaflet overrides, map-only elements | map page |

Order in `<head>`: `tokens.css` → `base.css` → `components.css` → page CSS.
Third-party CSS (Leaflet) loads before page CSS so overrides win without
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

### Glass (promote from `maps.css`)

Currently defined as `--maps-glass-*` on `.maps-page`. Move to `:root` and drop
the `maps-` prefix.

| Token | Value | Use |
|---|---|---|
| `--glass-bg` | `color-mix(in srgb, var(--bg-surface) 74%, transparent)` | Floating panel fill |
| `--glass-bg-strong` | `color-mix(in srgb, var(--bg-surface) 86%, transparent)` | Popovers and small chips that need more contrast |
| `--glass-tile` | `color-mix(in srgb, var(--bg-surface-elevated) 42%, transparent)` | Cards, tables, and collapsibles inside a glass panel |
| `--glass-border` | `color-mix(in srgb, var(--text-primary) 11%, transparent)` | Hairline border on glass and on dividers inside it |
| `--glass-blur` | `blur(14px) saturate(140%)` | `backdrop-filter` value |

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
| `--text-xl` | `1.35rem` | Hero value such as a callsign |

Use `rem` for type and spacing. `px` is only for hairlines and `--chrome-inset`.

## Surfaces

Pick the surface by what is underneath it.

| Surface | Fill | Border | Shadow | When |
|---|---|---|---|---|
| Page | `--bg-app` | none | none | Body background |
| Solid panel | `--bg-surface` | `--border-color` | `--shadow-2` | Dashboard tiles, admin panels, topbar |
| Glass panel | `--glass-bg` + `--glass-blur` | `--glass-border` | highlight + `--shadow-3` | Anything floating over the map or other live content |
| Tile | `--glass-tile` | `--glass-border` | highlight + `--shadow-1` | Cards, tables, collapsibles inside a glass panel |
| Popover | `--glass-bg-strong` + `--glass-blur` | `--glass-border` | highlight + `--shadow-3` | Dropdown and multiselect panels |
| Control | `--bg-surface-elevated` | `--border-color` | none | Inputs, selects, icon buttons |

Inside a glass panel, dividers use `--glass-border`, never `--border-color`.

## Components

Proposed shared class names are listed first; the map class that implements it
today is in brackets. Use BEM: `block__element--modifier`.

### Panel `.panel` `.panel--glass`

[`.maps-filters`, `.maps-detail-panel`]

Container with padding `0.85rem 1rem`, `--radius-md`, and the solid or glass
surface. Header row pattern:

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

[`.maps-detail-panel__tab`, `.admin-tabs__link`]

Active state is `aria-selected="true"` (tab widgets) or
`aria-current="page"` (navigation). Underline tabs use an inset accent bar;
navigation pills use the control surface plus `--shadow-2`.

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

## Duplication audit

Found while building the map. Each line is one item of cleanup.

- Focus ring declared per component about fifteen times across the three files.
- Three input styles: `.maps-filters select/input`, `.admin-field input`,
  `.public-form__input`.
- Four icon-button implementations (close, collapse, filter toggle, profile).
- Four button looks: `.tile__link`, `.public-button`, `.maps-filters__apply`,
  `.maps-map-only`.
- Three eyebrow-label rules in `maps.css`.
- Glass tokens scoped to `.maps-page`, so other pages cannot use them.
- `--shadow-surface` and inline `rgba()` shadows instead of an elevation scale.
- `.maps-filters__header-filter` and `.maps-filters__header-collapse` use solid
  control tokens while sitting inside a glass panel.
- `.maps-map-only`, `.flight-tooltip`, and Leaflet controls still use the old
  `blur(6–8px)` and `--shadow-surface` instead of the glass tokens.
- Dead rule: the empty `.maps-shell--chrome-hidden …` block in `maps.css`.
- ID selectors used for styling (`#maps-count`, `#maps-cached`, `#maps-error`).
- `!important` on Leaflet overrides, which can be dropped once page CSS loads
  after `leaflet.css`. The map layout currently loads `maps.css` before
  Leaflet.
- Hard-coded `#c0392b` in `public.css`.
- Mixed `px` and `rem` for gaps (`6px`, `2px`, `8px`).
- `docs/standards.md` still points to `static/css/design-system.css` and
  `ui-components.mjs`, which do not exist in the rewrite.

## Migration plan

Do these in order; each phase should ship on its own with no visual
regressions except where noted.

1. **Tokens.** Create `tokens.css` holding the palette plus the glass,
   elevation, shape, motion, z-index, and type tokens above. Load it from both
   layouts. Alias the old names (`--maps-glass-*`, `--shadow-surface`) to the
   new ones so nothing breaks.
2. **Base.** Add the single focus rule, global `[hidden]`, and the
   reduced-motion block to `base.css`, then delete the per-component copies.
3. **Components.** Create `components.css` with `.panel`, `.card-grid`/`.card`,
   `.kv-table`, `.section`, `.eyebrow`, `.badge`, `.btn`, `.icon-btn`,
   `.field`/`.input`/`.select`, `.check-chip`, `.collapsible`, `.copy-key`, and
   `.multiselect`. Move the shared JS (multiselect, copy-to-clipboard) to
   `static/js/ui/`.
4. **Map.** Switch map templates to the shared classes, leaving only layout in
   `maps.css`. Put the remaining map chrome (map-only button, tooltip, Leaflet
   controls) on glass tokens. Load `maps.css` after `leaflet.css` and remove
   `!important`.
5. **Dashboard and admin.** Move `tile`, `admin-tabs`, and `admin-field` onto
   `.panel`, `.tabs`, and `.field`. This phase intentionally changes the look
   to match the map.
6. **Public pages.** Move `public-button` and `public-form__*` onto `.btn` and
   `.field`, and fix the hard-coded colour.
7. **Clean up.** Remove the token aliases from phase 1, delete dead rules, and
   update the Vizburo UI section of `docs/standards.md` to point here.

## Checklist for new UI

- [ ] No raw colours, shadows, or blur values; tokens only.
- [ ] Uses an existing component class or adds a modifier to one.
- [ ] Glass only for surfaces floating over live content.
- [ ] Missing values show `—`; times show UTC with `<time datetime>`.
- [ ] Icon-only controls have `aria-label`; `aria-expanded`/`aria-selected`
      stay in sync.
- [ ] Works with HTMX swaps (delegated listeners, no per-node init that a swap
      would drop).
- [ ] Checked at mobile width, or the desktop-only tradeoff is written down.
