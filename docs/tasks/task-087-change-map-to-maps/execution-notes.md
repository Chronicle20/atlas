# Task-087 — Execution Notes & Plan Deviations

Companion to `plan.md`. Records where execution diverged from the plan and why.
Read this alongside `plan.md` when auditing plan adherence — several deviations
are **additive corrections** to a plan whose consumer audit was incomplete.

## Summary

All 12 planned tasks were implemented as written, plus **three additional
consumer migrations** the plan's audit (`context.md` "Consumer migration audit",
`plan.md` Task 10 table) misclassified or omitted. Every migration follows the
same established pattern (per-service `location` client → `location.GetField`, or
the in-scope live `field.Model`). All Go modules build/vet/race clean; atlas-ui
builds and 740/740 tests pass.

## The plan's consumer audit was wrong about three services

The plan classified the character `mapId` mirror consumers into ACTIVE (migrate)
and PASSIVE (strip dead field). The "Task 11 last; shim removal after all
consumers" ordering is correct, but the audit **undercounted active consumers**.
Independent verification during execution found three more services that
**read** the mirror's `MapId()` and would have silently received `mapId=0` after
Task 11 removed it from atlas-character's GET:

| Service | Plan said | Reality | What was read | Fix |
|---|---|---|---|---|
| **atlas-channel** | PASSIVE ("declared, never read") | **ACTIVE** | `socket/writer/character_data.go` CharacterData packet (field-entry / cash-shop-open) + `socket/handler/character_chat_whisper.go` whisper-find result + `kafka/consumer/character/consumer.go` party-HP field build | Reused the existing `atlas-channel/maps/location` client; threaded `mapId` into `BuildCharacterData`; whisper-find + HP-propagation use `location.GetField`. Extracted `location.ResolveMapId` helper. (commits `c16e6b1b5`, `05c40d956`) |
| **atlas-messages** | not in the audit at all | **ACTIVE** | `@query map` (WhereAmI, player-visible) + `@query rates` + ~30 other command sites building `field.NewBuilder(ch…, c.MapId())` | All command readers switched to the live inbound `field.Model f` the dispatcher already passes (no network call needed). Mirror stripped. (commits `b61c944b5`, `f05191d5`, `0b1ba2c26`) |
| **atlas-pets** | misclassified PASSIVE | **ACTIVE** | `pet/processor.go` `GetBelow(c.MapId(), c.X(), c.Y())` for pet-spawn foothold | Added `atlas-pets/location` client; spawn map from `location.GetField` (no live field in scope there); mirror stripped + dropped `mapId` from the sparse-fieldset request. (commits `b3af77854`, `4191e3153`) |

**Verification that the set is now complete:** after these, the only remaining
character-package `MapId()` getters in the repo are `atlas-parties` (the live
registry `Model`, field-backed since Task 7), `atlas-maps/character/location`
(the location **owner**), and atlas-character's own create-input. No service
reads a character-resource `mapId` mirror anymore.

Two services declared the mirror field but genuinely never read its map and were
stripped as passive cleanups: **atlas-pets**'s sibling **atlas-maps**
`character/rest.go` (reads only X/Y — `bb658c1a3`) and the five plan-listed
passive services (atlas-login, atlas-npc-shops, atlas-cashshop, atlas-messengers
in Task 10; atlas-channel was reclassified ACTIVE).

## Notable design decisions during execution

- **atlas-maps import cycle (Task 2):** `location` cannot import `warp` (warp
  imports location). Broke it with a `WarpProvider` DI seam injected from
  `main.go` — the REST handler and the Kafka consumer still call the *same*
  `warp.ChangeMap`.
- **atlas-messages used the live field, not a location client:** the command
  dispatcher already passes the inbound `field.Model` (carrying the player's
  current map) to every command, so a network lookup was unnecessary and was
  removed.
- **Error-fallback policy:** location-lookup failures log (Warn on infra,
  silent on `ErrNotFound`) and fall back to map 0 / skip, consistent across
  atlas-parties, atlas-channel (`ResolveMapId`), atlas-pets, atlas-query-aggregator.

## Verification gate results

- **Go build/vet/race:** clean across all 12 changed modules
  (atlas-maps, atlas-parties, atlas-consumables, atlas-query-aggregator,
  atlas-channel, atlas-login, atlas-npc-shops, atlas-cashshop, atlas-messengers,
  atlas-messages, atlas-pets, atlas-character).
  - **Pre-existing exception:** `atlas-login` `go vet ./...` reports
    `socket/init.go:39: WaitGroup.Add called from inside new goroutine`. This
    file is **not** touched by task-087 and the warning exists on `main`
    (unrelated pre-existing bug). The task-087-changed `character` package vets
    clean.
- **atlas-ui:** `npm run build` clean (chunk-size advisories only); `vitest run`
  **740/740 pass**.
- **docker buildx bake atlas-maps:** image built and exported successfully.
- **redis-key-guard:** no `go.mod`/`go.sum` changed and **zero** redis code was
  added by task-087, so the invariant cannot regress. The local
  `GOWORK=off` run reports a `matched no packages` artifact that affects
  **untouched** services (e.g. atlas-buffs) identically — an environment issue,
  not a task-087 violation. CI (provisioned for standalone module builds) is the
  authoritative check.
- **No go.mod/go.sum changes:** every new `location` package imports only
  existing shared libs, so no Dockerfile `COPY`/`go.work` edits were needed.

## Commit map (high level)

- Tasks 1–2 atlas-maps: `e19ca8af2`, `b0e3f5dd8`, `35dc7808e`, `a3e3e0320`
- Tasks 3–6 atlas-ui: `11751ac3e`, `15fdb9edd`, `3d431ee8f`, `829b2d131`, `d625cdcb8`
- Task 7 atlas-parties: `2a15a2ada`, `97ecc2e06`
- Task 8 atlas-consumables: `3a4581cc5`
- Task 9 atlas-query-aggregator: `2acb9fb0f`, `f4d7b2996`
- Task 10 passive strips: `333e740f6`, `4737cfc7c`, `3dace1dd5`, `703abc1cc`
- atlas-channel ACTIVE: `c16e6b1b5`, `05c40d956`
- atlas-messages ACTIVE: `b61c944b5`, `f05191d5`, `0b1ba2c26`
- atlas-pets/maps: `bb658c1a3`, `b3af77854`, `4191e3153`
- Task 11 atlas-character shim removal: `dfcc53946`, `fbce37a46`
