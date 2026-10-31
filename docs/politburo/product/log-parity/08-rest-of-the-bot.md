# Rest of the bot

The old bot’s other commands are real product surface. They should come over
as the same platform, not as ports of `magic_time`, guild JSON files, or
per-command Airtable bases.

Depends on identity and the Airtable pilot pointer. Independent of `/log`
except where stats want the same mapping.

## Strategy

Split them by data source. Live VA flights, ATIS, ATC, and IF stats read
Infinite Flight through Politburo. VA stats read one Airtable pilot row by
the stored record id. Anything that scans the PIREP table (leaderboard, WT
stats, WT leg browser) stays parked with the staff logbook decision. Link
and learn are a small VA config. Rank ladders are content, not code.
Operator commands (`sus_pilots`, `update`, `magic_time`, aircraft CSV
rebuild) are not features.

## Tasks

| Task | What to decide or build |
|---|---|
| VA live | Point Discord live at the existing VA flight index instead of the unfiltered map link. World Tour live is a route filter on that list once plans are synced. |
| IF logbook and stats | Replace `/logbook` and the game half of `/stats` stubs with the user flights and stats client that registration already uses. |
| VA stats | Read the one pilot row by `airtable_pilot_id` for the mapped stat fields. Career stats wait on the career module. |
| Discord id write-back | Optional mapped field on the pilot row, filled at register. Replaces `>sync`. |
| ATIS and ATC | Thin commands over existing Infinite Flight endpoints, session from `BASIC` preferred server. |
| Link and learn | Optional VA config map of keyword to URL. Portal later. |
| Parked reads | Leaderboard, WT leg lookup, and WT participation counts need log-table reads. Leave them until a read slice is explicitly opened. |
| Drop | Do not port `magic_time` caches, the aircraft-experiment channel, `sus_pilots` CSV, or the hardcoded rank essay unless an org asks for the ladder as content. |
