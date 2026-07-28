# task-133 Context — Miniroom Minigames (Omok + Match Cards)

Quick-reference for implementers. Full rationale: `design.md` (D1–D12, G1–G5). Steps: `plan.md`.

## What ships
New `atlas-mini-games` service (in-memory room registry + pure game engines + GORM `game_records`), new clientbound minigame dispatcher arms + `UPDATE_CHAR_BOX` balloon packet in libs/atlas-packet, wired `character_interaction.go` arms + status consumer + map-entry balloon spawn in atlas-channel, seed-template updates for all six versions, k8s wiring, live-tenant PATCH runbook.

## Key reference files (read before implementing)
- **Behavioral truth (game rules):** Cosmic local checkout — `<cosmic>/src/main/java/server/maps/MiniGame.java`, `net/server/channel/handlers/PlayerInteractionHandler.java`, `tools/PacketCreator.java`. Extracted semantics are in design.md §3 with line citations. **Cosmic has NO retreat feature** — retreat comes from IDA (gate G2).
- **Packet-layout truth:** `docs/tasks/task-133-miniroom-minigames/ida-notes.md` (produced by plan Task 1; G1 start byte, G2 retreat, G3 balloon, G4 visit, G5 modes).
- **Channel vertical template:** merchant — `services/atlas-channel/atlas.com/channel/merchant/{processor,producer}.go`, `kafka/consumer/merchant/consumer.go`, `kafka/consumer/map/consumer.go:666` (`spawnMerchantsForSession`), `:159` (`SpawnForSelf`).
- **Service skeleton:** `services/atlas-chalkboards/atlas.com/chalkboards/` (main.go, kafka envelopes/consumer/producer, resource.go). GORM layer: `services/atlas-buddies/atlas.com/buddies/list/`. Surrogate-PK DDL precedent: `services/atlas-gachapons/.../gachapon/entity.go`.
- **In-memory registry pattern:** `services/atlas-channel/atlas.com/channel/account/registry.go` (sync.Once + RWMutex). NOT the Redis TenantRegistry — design D1/PRD mandate in-memory, hence `replicas: 1`.
- **Dispatcher family rules:** `docs/packets/DISPATCHER_FAMILY.md` (AP-1..8, INV-1..5); existing arms `libs/atlas-packet/interaction/clientbound/{interaction.go,interaction_body.go,interaction_test.go}`; candidates `tools/packet-audit/cmd/run.go:1895-1917`; verify flow `docs/packets/audits/VERIFYING_A_PACKET.md`.

## Load-bearing decisions
- roomId = ownerId (D2). Balloon `gameId` and client VISIT `serialNumber` both carry it.
- ALL validation server-side in atlas-mini-games via REST fan-out (D3): alive = character `Hp() > 0`; map `fieldLimit & 0x80` → error 11; chalkboard open → 13; item missing → 6 (Omok `4080000 + pieceType` clamp [0,11]; Match Cards `4080100`); already-in-room → 6 (convention, not client parity); room gone → 1; full → 2; bad password → 22. Error events carry the **enterError key string** (e.g. `"NOT_WHEN_DEAD"`), resolved by the channel via the tenant `enterError` table.
- Commands keyed by characterId; room serialization via registry write lock (D5). Invalid/out-of-turn gameplay commands are silently dropped (Cosmic parity).
- Server tracks turns (Cosmic doesn't): omok alternates (+skip/retreat adjust), match cards retains on match / passes on mismatch. Initial turn from the START `firstMover` byte (G1); Cosmic wire values: initial 1, then 0 after owner win / 1 after visitor win.
- Scores are per-room session state (never persisted): +50 win (suppressed if forfeit-farm ≥4), +15/−15 loss, +10 tie with 5-min cooldown; reset when a different visitor joins.
- Records: `db.Transaction` directly — `database.ExecuteTransaction` is a no-op bug. Both rows one tx, commit before events emit.
- Balloon on map entry = standalone UPDATE_CHAR_BOX announces (D8); `spawn.go:129` mini-room byte stays 0.
- The `MiniRoom` writer name exists in channel `main.go:783` but is mapped in ZERO templates today — mapping it activates merchant personal-shop balloon sends too (verify shop branch in G3).

## Gotchas (bite in this exact task)
- `MEMORY_GAME_FIP_CARD` typo is load-bearing — keep verbatim in consts, configs, candidates.
- Every seed handler entry needs `"validator": "LoggedInValidator"` or it's silently dropped.
- Seed gaps are asymmetric: 87/95 missing the whole handler entry, 92 missing handler AND writer, jms missing MEMORY_GAME rows (verified 2026-07-04).
- Mode values are per-version; v83/v95 verify via IDA (only loaded IDBs — instance set rotates, `list_instances` + match binary name), 84 derives from 83, 87/92/jms derive + banner UNVERIFIED.
- packet-audit export splices are surgical — never regenerate/overwrite an export; audit output flag is `--output docs/packets/audits`.
- MoveStone serverbound packs x,y into one int64 (`point`): x = low DWORD.
- Readiness probe path is `/api/readyz` (base-path wedge bug). docker-bake `go_services` is hand-synced with services.json — update BOTH.
- Run `tools/redis-key-guard.sh` from repo root WITHOUT GOWORK=off; `go work sync` is banned.
- Live tenants don't pick up seed changes — rollout.md PATCH + channel restart is part of the task, not optional.

## Dependency order
Task 1 (IDA) → Tasks 2–8 (packets) → 9–16 (service; 11/12 engines independent) → 17–19 (channel) → 20 (templates) → 21 (deploy) → 22 (gates + runbook + review). Tasks 2–6 depend on ida-notes; Task 5 and 15's retreat logic are blocked if G2 is unresolved (stop-and-ask).
