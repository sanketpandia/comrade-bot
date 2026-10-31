# Aircraft and airline

Each org keeps the translation from Infinite Flight aircraft and livery
names to the labels their Airtable select fields accept. Administrators can
look up a mapping, change it, add one, and see which names still fall
through to the default.

Depends on the livery catalog already in Postgres. Sightings need the VA
flight index or the submit path. Select-choice hints need the Airtable field
mapping, and the page still works without them.

## Strategy

Store rows per VA, not a shared JSON file. Two kinds, matching the old
`AircraftMappings` and `AirlineMappings`: aircraft (IF aircraft name to VA
aircraft label) and airline (IF livery name to VA airline label). Each VA
also sets a default aircraft and a default airline, which replace the old
`Other` keys. Filing looks up the live flight’s aircraft name and livery
name; a miss writes the default and records a sighting. The portal page has
three tabs: Aircraft, Airline, and Unmapped. Aircraft and Airline list saved
rows with edit and delete, plus a set-mapping form whose source field
autocompletes from the livery catalog we already sync. Unmapped lists
sightings for this VA that still have no row: source name, kind, how many
times seen, last seen, and a target field so they can set it without leaving
the tab. The full Infinite Flight catalog is not shown as unmapped. If the
logs mapping says the aircraft or airline column is a single select, keep
that choice list on the field mapping so the target control can offer those
labels. That list is configuration, not a download of rows. Edit stays with
the administrator. `/log` shows the resolved labels in the confirm embed so
the pilot can see what will be written.

## Tasks

| Task | What to decide or build |
|---|---|
| Table | `va_id`, kind (aircraft or airline), source, target. Unique on `va_id` + kind + source. Defaults stored beside it, two strings per VA. |
| Resolve | Lookup used by submit. Hit returns the target. Miss returns the default and upserts an unmapped sighting (count, last seen). |
| Mapped tabs | Aircraft and Airline: search, edit target, delete, add. Source autocomplete from the livery catalog. |
| Unmapped tab | Sightings with no mapping row. Set a target from this tab and the sighting leaves the list. |
| Select choices | When the mapped Airtable column is a single select, persist its options on the field mapping and offer them as the target dropdown. |
| Seed | Import `if_afklm_mappings.json` once for the AFKLM VA. Other orgs start empty and grow from the unmapped tab. |
