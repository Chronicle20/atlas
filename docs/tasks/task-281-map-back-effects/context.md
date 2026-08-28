# task-281-map-back-effects — Planning Context

Companion to `plan.md`. Records the key files, the decisions the plan encodes, the
things it deliberately does not do, and the facts an implementer should not
re-derive.

## Key files by layer

| Layer | Module root | Files |
|---|---|---|
| Codecs | `libs/atlas-packet` | `field/clientbound/{set,clear}_back_effect.go` + tests (new) |
| Matrix | repo root | `docs/packets/registry/*.yaml`, `docs/packets/evidence/`, `docs/packets/audits/`, `docs/packets/feature-{families,na-evidence}.yaml`, 8 seed templates |
| State | `services/atlas-maps/atlas.com/maps` | `map/backeffect/` (new, 6 files), `kafka/message/map/{command,kafka}.go`, `kafka/consumer/map/consumer.go`, `main.go` |
| Emit + replay | `services/atlas-channel/atlas.com/channel` | `backeffect/` (new, 4 files), `kafka/message/map/kafka.go`, `kafka/consumer/map/consumer.go`, `socket/writer/`, `main.go` |
| Saga | `libs/atlas-saga` + `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | `model.go`, `payloads.go`, `unmarshal.go`; `saga/{model,handler,event_acceptance}.go`, `map_command/`, `kafka/message/map/kafka.go` |
| GM command | `services/atlas-messages/atlas.com/messages` | `command/map/back_effect.go` (new), `kafka/message/map/command.go`, `main.go` |

The whole feature is modelled on **jukebox**, which is the closest existing
end-to-end analogue: `map/jukebox/` in atlas-maps, `jukebox/` + the
`handleStatusEventJukeboxStart` / `announceActiveJukebox` pair in atlas-channel,
`handlePlayJukebox` + `map_command` in the orchestrator, `weather.go` in
atlas-messages. Every `Patterns to copy:` line in `plan.md` points into that set.

## Decisions carried from design.md

- **Registry is `map[FieldKey][]BackEffectEntry`** — N entries per field, one per
  `pageId`, first-set-first order, replace-in-place on a repeat page. Forced by
  the wire: set is per-page, clear is whole-field (`ReloadBack`).
- **No expiry and no reaper.** `tDuration` is a client-side fade length, not a
  lifetime. Reading it as an expiry would be a misreading of the decompile.
- **No teardown hook.** `atlas-maps` has no instance-destroyed event and no
  per-field occupancy state; building it would be larger than the feature.
  Accepted leak: a few 10-byte entries per destroyed instance until restart.
- **Replay with `Duration = 0`.** The fade already ran for everyone else; a late
  joiner lands on the end state instead of re-running the tween.
- **REST is a collection (`/backEffects`, 200-empty), not the PRD's singular
  `/backEffect` with a 404.** PRD FR-4 conditioned that shape on the decompile,
  and the decompile proved N-per-field.
- **Reject `effect` outside {0,1} at the atlas-maps consumer**; do not clamp
  `duration`. The client ignores an out-of-range effect, so broadcasting it is a
  guaranteed no-op; duration has no DoS shape comparable to pinning a field's BGM.
- **No `MajorAtLeast` gate in the first cut.** v72/v84/v95 are proven identical;
  an unexercised gate is a liability.

## Decisions this plan adds

- **`docs/packets/feature-families.yaml` gains a `back_effect` family** (Task 4
  Step 4). The design asked for "fresh VERSION-ABSENT records" on the six ⬜
  cells but did not name the mechanism that keeps that claim honest. Declaring
  the family makes `matrix --check` hold every ⬜ cell to positive absence
  evidence once a sibling verifies
  (`tools/packet-audit/cmd/na_consistency.go:120-205`). The obligation is real
  but cheap: Task 2 produces exactly the prose the ledger wants. If the family
  is *not* declared, the six ⬜ cells carry no enforced proof at all.
- **`server.MarshalResponse[[]RestModel]` for the collection endpoint**, not
  `MarshalPaginatedResponse`. Precedent: `services/atlas-ban/.../report/resource.go:62`,
  `services/atlas-rankings/.../ranking/resource.go:140`. The paginated form
  (used by the `event-visuals` collection) buys nothing for a result bounded by
  the number of back-layer pages on one field, and the channel-side
  `requests.SliceProvider` decodes the plain array form.
- **`socket/writer/{set,clear}_back_effect.go` are written even though they have
  no call site.** The existing `play_jukebox.go` / `field_obstacle_*.go` writer
  bodies in that package likewise have zero production callers — the consumer
  calls `fieldcb.New...().Encode` directly. Design §4.1 asks for them and the
  file-per-writer convention is uniform, so the plan matches it; an implementer
  should not be surprised to find the functions unreferenced.

## Facts verified during planning (do not re-derive)

- Packet id convention is `<domain>/<direction>/<Domain><TypeName>` — e.g.
  `field/clientbound/FieldPlayJukebox` for type `PlayJukebox`. Ours are
  `FieldSetBackEffect` and `FieldClearBackEffect`.
- `MajorAtLeast` is a **method on `tenant.Model`** (`libs/atlas-tenant/tenant.go:93`),
  used compound: `t.IsRegion("GMS") && t.MajorAtLeast(N)` — see
  `libs/atlas-packet/field/clientbound/set_field.go:46`.
- `rest.RegisterHandler` in atlas-maps is a var alias of `server.RegisterHandler`
  (`services/atlas-maps/atlas.com/maps/rest/handler.go:28`), not a local function.
- `atlas-saga-orchestrator` and `atlas-messages` both have **no**
  `kafka/message/map/command.go`; `Command[E]` and every body live in the single
  `kafka/message/map/kafka.go`. Only `atlas-maps` splits `command.go`/`kafka.go`.
  (`plan-lint` F1 caught this after the first draft assumed the atlas-maps split
  held everywhere.)
- atlas-messages registers command producers in **`main.go:93`**, not in
  `commands.go`.
- `--ida` on `packet-audit evidence pin` takes the fname string, not an address.
- The `Command[E]` and `StatusEvent[E]` envelopes are already identical across
  atlas-maps, atlas-channel and the orchestrator — no envelope change is needed.
- `test.CreateContext` / `test.Encode` / `test.RoundTrip` / `test.Variants` live
  in `libs/atlas-packet/test/`.
- `weather.go` has no executor-level test today. That is a pre-existing coverage
  gap; Task 12 tests the new commands at the same regex-and-gate level the
  sibling `TestWarpCommandProducer_RegexPatterns` uses, and does not backfill it.

## Task sizing notes

- **Task 4 is deliberately large** — 8 registry files, 8 seed templates, 20
  evidence pins, 2 ledger files. It is one mechanical edit repeated, which
  batches fine, and splitting it would leave the matrix red at a task boundary
  (`matrix --check` only goes green once every cell and route lands together).
- **Task 11 touches 8 files in one service.** The orchestrator's PlayJukebox
  wiring is spread across `saga/model.go`, `saga/handler.go`,
  `saga/event_acceptance.go`, `kafka/message/map/kafka.go` and `map_command/`;
  every one of those is a two-to-six-line copy of the PlayJukebox arm, and a
  partial landing does not compile. Kept whole for that reason.
- Every other task is ≤6 files and one module.
- Tasks 5–7 (atlas-maps), 8–9 (atlas-channel), 10–11 (saga), 12 (messages) are
  independent of each other once Task 3's field names are fixed, and can run in
  parallel if the executor wants. Tasks 1–2 gate Task 3; Task 3 gates Task 4.

## Open risks

- **A version diverges from the four-field layout.** Tasks 1–2 stop for a
  controller ruling rather than guessing a gate. jms_v185 is the likeliest.
- **v48/v61 absence may be harder to prove than expected.** The exports say
  "function not found in IDB", which is a lead, not evidence. If the router read
  is inconclusive, that is a genuine blocker to surface, not something to paper
  over with a `feature-na-evidence.yaml` entry that says less than it claims.
