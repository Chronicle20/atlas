# Fix round B report — `atlas-player-npcs` client packages (EXT-01/EXT-02) + compose scaffolding

## Summary

Fixed all three blocking findings from `audit-backend-player-npcs.md` scoped to this round
(EXT-01, EXT-02, SCAFFOLD-06), plus closed the Task 18 nested-`equipment` decode gap. Stayed out
of `playernpc/` and `services/atlas-channel/` per the constraints.

## EXT-01 — reference-ID stubs

Added the standard `libs/atlas-rest/CLAUDE.md` boilerplate (`SetToOneReferenceID` /
`SetToManyReferenceIDs`, both no-ops) to every `RestModel` that is an `api2go/jsonapi.Unmarshal`
target and previously lacked them, copying the pattern already present at `character/rest.go:48,52`
and `inventory/rest.go:95,99,185,189`:

- `data/map/rest.go` — `RestModel`, `GroundRequestRestModel`, `GroundResultRestModel` (all three,
  matching the audit's exact finding — `GroundRequestRestModel` is a POST body but the pattern is
  applied uniformly with its siblings).
- `data/npc/rest.go` — `RestModel`.
- `ranking/rest.go` — `RestModel`.
- `configuration/rest.go` — `RestModel`.

## EXT-02 — httptest-backed integration tests

Added a `httptest.NewServer`-backed test to each of the four packages the audit flagged as
lacking one, each asserting the path the client's own `requests.go` builds and decoding a
representative captured JSON:API fixture through the real `Processor` call (not a hand-built
`RestModel`):

- `character/rest_test.go` — `TestGetById_HttpDecode`, serves `/characters/1001`, asserts via
  `Processor.GetById`.
- `data/map/rest_test.go` — `TestGetById_HttpDecode` (`/data/maps/100000000`) and
  `TestGround_HttpDecode` (`/data/maps/100000000/ground`, the POST path), both via `Processor`.
- `data/npc/rest_test.go` — `TestGetById_HttpDecode`, serves `/data/npcs/9010000`.
- `inventory/rest_test.go` — `TestGetByCharacterId_HttpDecode`, serves
  `/characters/1001/inventory` with the full `included` compartments→assets chain, asserting
  `Processor.GetByCharacterId` decodes `CharacterId()` and `Equipment()`.

Each test sets the package's actual per-domain env var (`CHARACTERS_SERVICE_URL`,
`DATA_SERVICE_URL`, `INVENTORY_SERVICE_URL`) via `t.Setenv`, matching the existing PASS pattern in
`configuration/rest_test.go` / `ranking/rest_test.go`.

## The Task 18 nested-`equipment` decode gap

Added `services/atlas-player-npcs/atlas.com/player-npcs/playernpc_equipment_decode_test.go`
(package `main`, at the module root — outside `playernpc/`, per the constraint) with
`TestPlayerNpcRestModel_EquipmentRoundTrips`. It imports the real, exported
`playernpc.RestModel`/`playernpc.EquipmentRestModel` types and round-trips a populated instance
with a non-empty two-element `Equipment` slice through real `jsonapi.Marshal` then
`jsonapi.Unmarshal` — the actual wire mechanism, not a hand-built decode target.

**Finding: the decode is NOT broken.** `Equipment` is a plain JSON attribute (`json:"equipment"`,
no `GetReferences`/relationship methods on `playernpc.RestModel`), so api2go serializes/deserializes
it as an ordinary nested-struct-slice attribute via the standard JSON attributes path, not the
relationship-walk path that `SetToOneReferenceID`/`SetToManyReferenceIDs` guard. The round-trip
test passed on the first run with no changes needed to `playernpc/`. This closes the Task 18 gap
as a verified-safe finding, not a "documented but unfixed" one — the previously-missing coverage
now exists and is green.

## SCAFFOLD-06 — docker-compose entry

Added `atlas-player-npcs:` to `deploy/compose/docker-compose.core.yml`, alphabetically between
`atlas-pets` and `atlas-portal-actions`, matching the `atlas-pets` shape (`*atlas-defaults`, no
`seed-catalog`, `LOG_LEVEL: debug`, `DB_NAME: atlas-player-npcs` — matches
`deploy/k8s/base/atlas-player-npcs.yaml`'s env). Also added explicit
`COMMAND_TOPIC_PLAYER_NPC`/`EVENT_TOPIC_PLAYER_NPC_STATUS` passthrough overrides, because (unlike
`EVENT_TOPIC_CHARACTER_STATUS`, already present) these two topic names are absent from
`deploy/compose/.env.example`; this follows the same explicit-override pattern already used by
`atlas-storage` (`COMMAND_TOPIC_STORAGE`/`EVENT_TOPIC_STORAGE_STATUS`) and `atlas-configurations`
(`EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`/`_TENANT_STATUS`) for topics not yet in the shared
`.env.example`. Validated the resulting file is well-formed YAML via `python3 -c "import yaml;
yaml.safe_load(...)"` (`tools/verify.sh`/docker itself are out of scope for this round).

## Testing

```
cd services/atlas-player-npcs/atlas.com/player-npcs && go build ./... && go test -count=1 ./...
```

Output: `Go test: 213 passed in 18 packages` — all packages pass, including the new tests, with no
skips or flags.

Also ran the new equipment round-trip test in isolation before the full suite, to confirm it was
exercising real marshal/unmarshal and not silently passing:

```
go test -run TestPlayerNpcRestModel_EquipmentRoundTrips -v .
=== RUN   TestPlayerNpcRestModel_EquipmentRoundTrips
--- PASS: TestPlayerNpcRestModel_EquipmentRoundTrips (0.00s)
PASS
ok  	atlas-player-npcs	0.013s
```

## Files changed

- `services/atlas-player-npcs/atlas.com/player-npcs/data/map/rest.go` — EXT-01 stubs (3 structs)
- `services/atlas-player-npcs/atlas.com/player-npcs/data/npc/rest.go` — EXT-01 stubs
- `services/atlas-player-npcs/atlas.com/player-npcs/ranking/rest.go` — EXT-01 stubs
- `services/atlas-player-npcs/atlas.com/player-npcs/configuration/rest.go` — EXT-01 stubs
- `services/atlas-player-npcs/atlas.com/player-npcs/character/rest_test.go` — EXT-02 httptest test
- `services/atlas-player-npcs/atlas.com/player-npcs/data/map/rest_test.go` — EXT-02 httptest tests (2)
- `services/atlas-player-npcs/atlas.com/player-npcs/data/npc/rest_test.go` — EXT-02 httptest test
- `services/atlas-player-npcs/atlas.com/player-npcs/inventory/rest_test.go` — EXT-02 httptest test
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc_equipment_decode_test.go` — new file,
  closes the Task 18 equipment-decode gap
- `deploy/compose/docker-compose.core.yml` — SCAFFOLD-06 new service entry

## Self-review

- EXT-01: applied uniformly to every `RestModel` the audit named, matching the exact existing
  repo pattern (comment + no-op stub pair) rather than inventing a new shape.
- EXT-02: each new test exercises the actual `Processor` method and the actual per-domain env var
  the package's own `requests.go` reads — not a shortcut that bypasses `RootUrlFor`.
- Did not touch `playernpc/` or `services/atlas-channel/`.
- Did not weaken any existing test.
- `.env.example` was left alone — out of this round's `### Files` scope; the two missing topic
  vars are instead passed explicitly in the compose service block, following an established
  precedent already in the file.

## Concerns

- None blocking. One judgment call: EXT-01 was applied to `GroundRequestRestModel` even though it
  is only ever used as a POST request *body* (never an `Unmarshal` target in this codebase today),
  because the audit's finding explicitly named it alongside `GroundResultRestModel`/`RestModel`
  and the stub is harmless boilerplate — consistent with "the same interface pair, every target
  struct," not a functional risk.
- The equipment-decode test's location (module root, package `main`) was chosen specifically to
  stay outside `playernpc/` per the constraint while still exercising the *real* production
  `playernpc.RestModel` type (imported, not re-declared) through the real `jsonapi` package. If a
  future reviewer would prefer this live inside `playernpc/` itself, that is a one-file move for a
  later round — the test's substance (and its "not broken" finding) would not change.
