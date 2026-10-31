# Work to reach /log parity

One ephemeral `/log` replaces acars, world tour, and career filing. Rules,
Airtable columns, and aircraft or airline labels are three separate per-VA
stores. The bot never downloads a base.

Seven streams come before Discord wiring. Three per-VA stores, including
mappings. Zero Airtable tables are copied locally.

Source: the-afklm-experiment commands and helpers, comrade-bot product docs,
and the current `/log` stubs. October 2026.

## Storage split

`FLIGHT_MODES` on `va_configs` is behavior: route strategy, prompts,
multiplier, remarks template. The provider config is the base: token, table
names, and which column each of our fields writes. Aircraft and airline
labels are their own rows: Infinite Flight name to the VA’s label, with
defaults for anything still unmapped. A mode says write aircraft. The
provider says which column. The mapping row says which label. None of the
three is a copy of Airtable rows.

## Streams

| Doc | Stream |
|---|---|
| [01-flight-modes.md](01-flight-modes.md) | Replace hardcoded acars, world tour, and career branches with one per-VA document |
| [02-flight-plans.md](02-flight-plans.md) | Persist origin and destination on the live flight so a mode can match a route |
| [03-find-the-flight.md](03-find-the-flight.md) | Return the user’s live flights in the current VA when they run `/log` |
| [04-airtable-connection.md](04-airtable-connection.md) | Per-org base, column map, one pilot pointer, one log write |
| [05-aircraft-and-airline.md](05-aircraft-and-airline.md) | Per-VA labels from Infinite Flight names, plus an unmapped tab |
| [06-submit.md](06-submit.md) | One request files the PIREP |
| [07-discord-log.md](07-discord-log.md) | One slash command: flight select, mode select, modal, submit |
| [08-rest-of-the-bot.md](08-rest-of-the-bot.md) | Other old commands, split by data source |

## Order to go deeper

Flight modes first, because every later stream consumes that contract.
Flight plans next, because route labels and `match_od` need origin and
destination. Flight resolution, the Airtable write path, and aircraft or
airline mappings can proceed together after that. Submit applies all three.
Discord is last and should not grow mode-specific code. The mapping page can
be designed on its own before submit exists: tabs, edit, and the unmapped
list.

## Old commands

| Old command | New home | How it comes over |
|---|---|---|
| `acars_pirep` | Streams 1–7 | `/log` mode select |
| `wt_pirep` | Streams 1–7 | Mode with `match_od`, reject on miss |
| `cm_pirep` | Streams 1–7 | Mode with extra fields and a pilot-module pointer |
| `if_afklm_mappings.json` | Stream 5 | Per-VA aircraft and airline rows, plus an Unmapped tab |
| `log` | Parity | IF logbook, already sketched as `/logbook` |
| `live` / `live_mini` / `live_wt` | Parity | VA flight index; WT is a route filter |
| `sync` | Parity | Register writes optional Discord id onto the pilot row |
| `pilot_stats` / `cm_stats` | Parity | One-row Airtable read by stored id; career module later |
| `if_stats` | Parity | IF user stats |
| `atis` / `ifatc` | Parity | IF endpoints, preferred server from `BASIC` |
| `link` / `learn` | Parity | VA config keyword map |
| `wt` / `wt_stats` / `leaderboard` | Parked | They scan PIREP rows and views |
| `prep_my_flight` / `aircraft` / `ranks` | Later | Local assets, not VA data |
| `magic_time` / `update` / `sus_pilots` | Drop | Cache and operator hacks |
