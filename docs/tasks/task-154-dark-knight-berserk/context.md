# task-154 Dark Knight Berserk — Execution Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer (or reviewer) needs without re-deriving the design phase.

## What this builds

Server-side tracking + broadcast of the Dark Knight Berserk (skill 1320006) aura state. atlas-buffs owns state and the 5s-initial/3s-period tick schedule; atlas-channel translates each tick event into own + foreign `EffectSkillUse` packets. The client computes the damage itself — the server's only job is the state formula and the broadcast.

The formula (Cosmic parity, `Character.java:1843-1870`, strict less-than):

```
active := skillLevel > 0 && hp > 0 && hp*100/effectiveMaxHp < x(skillLevel)
```

`x` comes from atlas-data effect data at runtime (v83 reference: 21→50 over 30 levels). The WZ/effect `berserk` field is a dead type-marker in Atlas — never read it.

## Load-bearing decisions (design.md §3, do not re-litigate)

- **D1 — Redis registry, not in-process**: atlas-buffs runs 2 replicas; every piece of its state is already a `TenantRegistry`. Namespace `buffs-berserk`, tenants registered in the shared `buffs:_tenants` set.
- **D2 — 1s scan ticker + atomic per-entry claims**: `TenantRegistry.Update` (`libs/atlas-redis/tenant_registry.go:130`) is single-attempt WATCH/MULTI; a lost race returns an error → treat as "not claimed". At most one replica emits per deadline. The poison ticker's get-then-update is NOT atomic — do not copy it.
- **D3 — consumers only mark state**: the ONLY consumer-side REST call is the LOGIN handler's skills lookup. All other I/O happens in the ticker's claimed re-evaluation (2 REST reads: character hp/level, effective-stats maxHP).
- **D4 — one Kafka event per 3s tick** on the existing `EVENT_TOPIC_CHARACTER_BUFF_STATUS` (new type `BERSERK`); atlas-channel is stateless, no map-enter hook (periodic re-broadcast covers late joiners).
- **D5 — grace deferral (2s)** for buff-origin and MAX_HP-bearing triggers: atlas-buffs itself produces the event effective-stats consumes to recompute max HP; re-evaluating immediately would read the stale value.
- **D7 — death is `hp > 0` in the formula**: no DIED consumer; the death-accompanying STAT_CHANGED(HP) evaluates inactive.
- **D8 — channel-unknown entries** (created by skill UPDATED, which carries no channel) neither re-evaluate nor broadcast until any channel-bearing character event fills the channel in. This same refresh rule IS the transfer handling.

## Verified ground truth the plan encodes (do not re-verify unless something fails)

- `STAT_CHANGED` carries `Updates []stat.Type` but NO current-HP value → HP must be read via REST.
- atlas-character REST does not expose channel; channel comes only from events.
- Effective-stats REST route: `worlds/{w}/channels/{c}/characters/{id}/stats`, maxHP JSON tag is `maxHP` (uppercase HP).
- atlas-skills envelope carries top-level `SkillId`; UPDATED body has `Level`. CHANNEL_CHANGED's upstream body struct is named `ChangeChannelEventLoginBody` (mirror the JSON, not the name).
- Packet layer is complete: `CharacterSkillUseEffectBody(skillId, charLevel, skillLevel, darkForceEffect, createOrDeleteDragon, left)` — 4th arg is the aura on/off flag, encoded only for `skill.DarkKnightBerserkId` (gate derived inside the lib). Writers registered for every tenant version; byte fixtures exist for v83/v84/v87. Zero writer/template/k8s work.
- All needed topic env vars (`EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_SKILL_STATUS`, `EVENT_TOPIC_CHARACTER_BUFF_STATUS`) are already in the shared configmap; atlas-buffs inherits via `envFrom`.
- Max-HP-affecting buff stat types = the `MapBuffStatType` cases in atlas-effective-stats resolving to max HP: currently only `HYPER_BODY_HP` (`TemporaryStatTypeHyperBodyHP` in atlas-constants).
- `sc.Is(tenant, worldId, channelId)` exists at atlas-channel `server/model.go:49`; `kafka/consumer/monster/consumer.go:14` already imports `socket/handler` as `socketHandler` (no cycle).

## Key template files (copy the shape, not just the idea)

| Need | Template |
|---|---|
| Registry wrapper | `services/atlas-buffs/atlas.com/buffs/character/registry.go` |
| Ticker task | `services/atlas-buffs/atlas.com/buffs/tasks/poison.go` + fan-out in `character/processor.go:190-205` |
| Curried consumer | `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go` |
| Producer providers | `services/atlas-buffs/atlas.com/buffs/character/producer.go` |
| External REST client | `services/atlas-effective-stats/atlas.com/effective-stats/external/character/{requests,rest}.go` |
| Channel consumer handler | `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go` (guards) + `kafka/consumer/character/consumer.go:437-457` (`sc.Is` + own/foreign announce shape) |
| Announce helpers | `services/atlas-channel/atlas.com/channel/socket/handler/effects.go:19-39` |
| Test setup (miniredis/tenant) | `services/atlas-buffs/atlas.com/buffs/character/registry_test.go` + `testmain_test.go` (producertest noop) |

## Task order and dependencies

Tasks 1→8 are ordered within atlas-buffs (model → evaluate → registry → clients/cache → event+producer → processor/ticker/wiring → consumers → buff hook). Task 9 (atlas-channel) needs only Task 5's JSON contract; Task 10 is the full verification suite (go test/vet/build both modules, `docker buildx bake atlas-buffs atlas-channel`, `tools/redis-key-guard.sh`, PRD acceptance-criteria sweep).

## Known limits / accepted deviations

- **producertest is a no-op writer** — end-to-end emission isn't unit-testable. The contract is pinned from both sides instead: provider-output JSON test in atlas-buffs (Task 5) + golden decode test in atlas-channel (Task 9).
- **No atlas-channel session-handler unit test** — the repo has no harness for session-touching consumer handlers; the handler mirrors already-shipped shapes and the wire bytes are pinned by lib fixtures. Documented inside Task 9.
- **GM-hide is an explicit follow-up** (PRD §9.1): foreign broadcast is unconditional here. Do not add speculative hide checks.
- Consumer LOGIN handler is covered via processor tests with stubbed externals, not consumer-level tests (would need a live skills endpoint).

## After implementation

Run `superpowers:requesting-code-review` (plan-adherence-reviewer + backend-guidelines-reviewer) before opening the PR — mandatory per CLAUDE.md. Live smoke: drop a Dark Knight below the threshold, watch the aura on self + a second client in the map, heal above threshold and watch it clear within one 3s tick.
