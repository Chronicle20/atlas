# Task-065 Post-Phase-B — Combat-Domain Audit Closeout (monster-only)

## Final state

- **Packets audited**: 9 (monster clientbound only — `MonsterSpawn`, `MonsterControl`, `MonsterDestroy`, `MonsterDamage`, `MonsterHealth`, `MonsterMovement`, `MonsterMovementAck`, `MonsterStatSet`, `MonsterStatReset`).
- **Verdicts (GMS v95)**: ✅ 3 / ❌ 5 / 🔍 1.
- **Cross-version passes**: none in this PR. v83/v87/JMS-v185 passes (Phase 3 in plan.md) deferred to follow-up tasks.
- **Combined SUMMARY.md (login + character + combat-monster, GMS v95)**: 87 packets total.
- **Total commits on branch above task-028 baseline**: 7.

## Scope deviation from plan.md

This PR ships a **monster-only Phase 2a** plus the tooling extension required to support combat-domain auditing. The plan's Phase 2b (pet, 14 packets), Phase 2c (drop, 3 packets), Phase 2d (reactor, 4 packets), Phase 3 (cross-version, 3 sub-tasks), and full Phase 4 closeout matrix are **deferred to follow-up tasks**.

The deviation was approved mid-session after discovery that (a) the plan's predicted FNames did not match v95 IDA, and (b) the per-packet audit + triage + IDA decompile transcription work is genuinely multi-session in scope. The follow-up tasks should reuse the tooling/routing infrastructure landed here.

## Scope deviation from plan.md (FNames)

Plan.md `Task 2`'s 31-row routing table contained **predicted** FNames (e.g. `CMobPool::OnSpawnMob`, `CMob::OnMobDamaged`, `CUserLocal::OnPet*`, `CWvsContext::SendPet*Packet`). None of those FNames exist in GMS v95. The actual v95 architecture has dispatchers (`CMobPool::OnPacket` → `CMobPool::OnMobPacket` → per-mob `CMob::OnXxx` leaf methods, `CUserPool::OnUserRemotePacket` → per-pet `CPet::On*` leaf methods, etc.). The CSV files (`docs/packets/MapleStory Ops - ClientBound.csv` and `ServerBound.csv`) carry the canonical verified mapping. The committed routing entries use the CSV-verified FNames.

## Tooling improvements

Two extensions to `tools/packet-audit/` landed in this PR:

1. **Sub-domain disambiguation** (`feat(packet-audit): sub-domain disambiguation via candidate.pkg`). Added `pkg string` field on `candidate{}`; `locateAtlasFile(root, name, pkg, dir)` now filters by `/{pkg}/{dir}/` substring; report filename becomes `titlecase(pkg)+name` (e.g. `MonsterSpawn.md`, `DropSpawn.md`). Required because combat sub-domains (`monster`, `drop`, `reactor`, `pet`) reuse short struct names (`Spawn`, `Destroy`, `Damage`, `Hit`, `Movement`, `Activated`) that collide across sub-domains — the existing character/login routing kept unique prefixed names and was unaffected.
2. **Combat-domain routing** (`feat(packet-audit): route 31 combat-domain FNames to atlas writers/handlers`). 31 entries in `candidatesFromFName`, each with `pkg` hint. Routes for pet/drop/reactor are committed but **only the 9 monster clientbound entries have IDA exports populated in this PR**; the others will produce no report until follow-up tasks populate `gms_v95.json` for them.

## Real wire bugs identified (none fixed in this PR)

| Packet | Issue | Why deferred |
|---|---|---|
| MonsterDestroy (`CMobPool::OnMobLeaveField@0x658b90`) | Atlas missing optional `WriteInt(swallowCharacterId)` when destroyType == 4 (swallowed by character-eater mob). Real wire bug — narrow scope (swallow eaters only, e.g. Yeti-and-Pepe boss). | Constructor signature change `NewMonsterDestroy` would affect callers in `services/atlas-channel`. Defer to a follow-up that updates the call sites + adds a hex baseline test. |
| MonsterControl (`CMobPool::OnMobChangeController@0x658d10`) | Atlas wire shape fundamentally differs from v95. Atlas writes `int8 controlType + int32 uniqueId + (if type>0: byte(5) + int32 monsterId + MonsterModel)`. v95 reads `byte controlMode + (if controlMode && opt: int32×3 seed) + int32 mobId + (if controlMode: byte aggro)`. | Looks like atlas implements an older-protocol shape; v95 controllers carry a movement-seed instead of MonsterModel. Defer to follow-up — needs v83/v87 cross-version IDA pass to understand when the shape changed. |

## Analyzer false positives surfaced

| Packet | Cause | Path to resolution |
|---|---|---|
| MonsterSpawn | (a) Atlas `if (region/version) { if controlled then WriteByte(1) else WriteByte(5) }` if/else expands into two consecutive `WriteByte` entries in the flat call list; analyzer doesn't model mutual exclusion. (b) `m.monster.Encode` MonsterModel sub-struct cannot be resolved because registry keys on unqualified struct names — there are 4 `Spawn` types across monster/drop/reactor/pet sub-domains; last-write-wins in `r.types` loses the `monster` field-type binding. | Registry should track qualified struct names (e.g. `monster/clientbound.Spawn`). Analyzer should detect mutually-exclusive if/else writes. |
| MonsterStatSet / MonsterStatReset | Atlas writes `uniqueId + MonsterTemporaryStat.Encode(mask + per-bit data) + int16(tDelay=0) + byte(nCalcDamageStatIndex=0) + optional byte(bStat)`. v95 `OnStatSet` top-level reads `mobId + DecodeBuffer(0x10) mask + delegate to ProcessStatSet`. The post-mask trailing fields (tDelay/calcIndex/bStat) live inside `CMob::ProcessStatSet`/`ProcessStatReset` which the audit pipeline cannot descend into. | Manual ProcessStatSet/ProcessStatReset decompile + entry expansion (sub-op enum drift candidate per plan.md). |
| MonsterMovement | Sub-struct expansion of `MultiTargetForBall`, `RandTimeForAreaAttack`, and `Movement` is incomplete (same struct-name collision as MonsterSpawn). Plus skill block layout: atlas writes `Decode2 skillId + Decode2 skillLevel` (separate fields), v95 IDA reads `Decode4 sEffect.m_Data` (packed). Same 4 wire bytes, different field decomposition. | Registry qualified-name fix unblocks this. The skill packing is likely a benign equivalence — atlas's two int16s pack into v95's one int32 in little-endian order. |

## Out-of-scope cleanly deferred

- **Phase 2b — Pet sub-domain (14 packets)**. Routing entries committed; IDA exports unpopulated.
- **Phase 2c — Drop sub-domain (3 packets)**. Routing entries committed; IDA exports unpopulated.
- **Phase 2d — Reactor sub-domain (4 packets)**. Routing entries committed; IDA exports unpopulated.
- **Phase 3 — Cross-version passes** (`gms_v83`, `gms_v87`, `jms_v185`). Need Phase 2 complete first.
- **Monster serverbound `CMob::GenerateMovePath` → MonsterMovementHandle**. The IDA function is 4 KB+ on the Encode side; deferred pending decision on how to model Encode→Decode equivalence in the audit pipeline for `Send*` sources.

## Audit-tool follow-ups recommended

1. **Registry qualified type names** — `r.types` should key on `<pkg>.<name>` so cross-sub-domain struct collisions (4 `Spawn` types, 4 `Destroy` types, etc.) preserve per-package field-type info needed by `resolveRecurse`.
2. **If/else mutual-exclusion modeling** — analyzer should detect `if X { WriteByte(a) } else { WriteByte(b) }` and emit one position with a noted alternation rather than two consecutive entries that misalign the diff.
3. **Dispatcher prefix annotation** — per-mob op IDA entries currently manually prepend `Decode4 (mobId)` (matching task-028's per-character op convention). Worth a helper that auto-prepends when an entry is marked as a sub-op of a dispatcher.
4. **Encode→Decode equivalence for Send\* sources** — `Send*` outbound functions in IDA do Encode×N. The atlas serverbound handler does Decode×N. The audit's diff engine should bind Encode-to-Decode equivalents by bit-width so the same JSON entry can describe both sides.

## Verification matrix run

```
go build ./...                                  # clean
go vet ./libs/atlas-packet/...                  # clean
go vet ./tools/packet-audit/...                 # clean
go test -race ./libs/atlas-packet/...           # clean
go test -race ./tools/packet-audit/...          # clean
```

`docker build` not required — no `go.mod` or `Dockerfile` files were touched.

`gitleaks` scrub of `docs/packets/audits/gms_v95/Monster*.md`: no `/home/` paths present.

## Commits on this branch above task-028 baseline

```
544e4f44e audit(monster): GMS v95 sub-domain audit (9 clientbound packets)
bf42c5dfd test(atlas-packet,monster/movement): add 5-variant round-trip baseline
57fb768f8 test(packet-audit): MonsterSpawn/StatSet flatten safety fixtures
f38916d81 feat(packet-audit): route 31 combat-domain FNames to atlas writers/handlers
eab8e64d8 feat(packet-audit): sub-domain disambiguation via candidate.pkg
2ae7cf590 test(packet-audit): assert combat sub-structs MonsterModel/TemporaryStat/MultiTargetForBall/RandTimeForAreaAttack registered
```

Plus three earlier docs commits (spec, design, plan).
