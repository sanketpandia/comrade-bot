# Discord /log

One slash command replaces `acars_pirep`, `wt_pirep`, and `cm_pirep`. The
pilot picks a flight if needed, picks a mode from a select, sees that mode’s
form update in the same message, fills it, and submits.

Depends on [submit](06-submit.md)’s config and submit contracts. Can be
sketched against fixtures before the API exists.

## Strategy

Keep the reply ephemeral. If the config read returns multiple flights, the
first row is a select labeled by origin–destination (callsign and aircraft
when two flights share a route). The second row is a flight-mode select of
every configured mode. Changing either select edits the embed: route
resolution, whether the mode is eligible, and the fields that mode will ask.
A Continue button opens the modal for that mode’s inputs. Discord modals
cannot be edited in place, so the dynamic part is the message, and the modal
is only the typed fields. Wire the existing pirep modal handler to the new
submit body. Select menus are currently ignored by the interaction router;
this command is what turns that router on.

## Tasks

| Task | What to decide or build |
|---|---|
| Select router | Handle flight and mode select custom ids. Stop dropping select interactions. |
| Flight select | Shown only when the API returns more than one flight. Choice is stored on the message and sent back on submit. |
| Mode select | Always shown. Edit the embed on change. Disable Continue when the mode is ineligible and say why. |
| Modal | Build fields from the mode. `flight_time` stays `HH:MM`. Career fields appear only for modes that list them. |
| Submit | Send flight id, mode id, and inputs. The embed shows the mapped aircraft and airline before submit, and the success line repeats them. No Airtable ids. |
| Retire buttons | Replace `mode_{id}` buttons. Leave tour’s separate button until events are in scope; do not special-case World Tour in this command. |
