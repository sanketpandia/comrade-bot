# Temporal flight-state estimation

## Record contract

Active flights are stored as `flights.Record` with immutable upstream `raw` (`Observation`) plus persisted sections `analytics`, `actions`, `links`, and reserved `history`. Resolution effects may only mutate those sections and job `Outcome` flags. Upstream values are rebuilt each tick via `NewObservation`; display strings and livery names are built at serve time via `flights.Normalize`. Rounding for API output happens in the HTTP mapper.

## Redis keys

| Key | Purpose |
|-----|---------|
| `game:flights:record:v3:{flightId}` | Sectioned record (`raw`, `analytics`, `actions`, `links`, `history`), 3-day TTL |
| `game:flights:history:{flightId}` | Estimator sample window + full internal state, 30-minute TTL |

History is written every refresh tick. The record is written only when persisted sections change (motion, new flight, or `Record.StoredSectionsEqual` mismatch). The API adds a serve-time `normalized` block and joins flight-state window features from the history key on read.

## Phases

`spawned_in`, `taxiing`, `takeoff`, `climbing`, `cruise`, `descending`, `landing`, `parked`, `awaiting_ap_plus_resumption`, `unknown`.

Transitions use soft expectations, hysteresis (`Pending` confirmations), and strong-evidence shortcuts. Unexpected transitions require extra confirmations (`UnexpectedTransitionConfirmations`).

## Configuration

Defaults live in `flightstate.DefaultConfig()` in `services/politburo/internal/livegame/flights/flightstate/config.go`. Each field is a tuning hypothesis (movement tolerance, climb/descent FPM thresholds, AP+ stasis window, staleness windows, etc.).

Inject overrides through `flightresolution.ComputeDeps.FlightState` for tests or experiments.

## Limitations

- No terrain elevation: “ground” uses a rolling `GroundRefAltFt` heuristic and `FallbackGroundMaxFt`.
- AP+ detection is a **candidate** flag, not confirmation of autopilot state.
- `confidence` is evidence strength (0–1), not a calibrated probability. Values are quantized to 0.05 on the stored record summary to avoid spurious cache writes.

## Livery enrichment

Aircraft and livery display names come from the in-memory livery catalog when serving (`Normalize`). The sync job only increments `politburo_livegame_livery_resolve_total` when the flight is new or the livery/aircraft identity changes.

## Tuning loop

1. Capture `game:flights:history:{id}` from Redis during a flight of interest.
2. Add a JSONL fixture under `flightstate/testdata/` and extend `replay_test.go`.
3. Adjust `Config` fields and re-run `go test ./internal/livegame/flights/flightstate/...`.
