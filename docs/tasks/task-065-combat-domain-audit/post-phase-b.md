# Task-065 Post-Phase-B — Combat-Domain Audit Closeout

## Final state

- **Packets audited**: 30 (24 clientbound + 6 serverbound) — monster (9 cb), pet (6 cb + 8 sb), drop (2 cb + 1 sb), reactor (3 cb + 1 sb).
- **Cross-version passes**: GMS v83 ✅, GMS v87 ✅, JMS v185 ✅.

### Verdict roll-up per version (combat domain)

| Version | ✅ | 🔍 | ❌ | Total |
|---|---|---|---|---|
| GMS v95 (source of truth) | 11 | 1 | 18 | 30 |
| GMS v83 | 11 | 0 | 19 | 30 |
| GMS v87 | 12 | 1 | 18 | 30 (PetSpawn sb routes; MonsterMovement 🔍 same as v95) |
| JMS v185 | 11 | 1 | 18 | 30 |

- **IDA-export coverage**: GMS v95, GMS v83, GMS v87, JMS v185 — combat FNames populated for each.
- **Total commits on branch above task-028 baseline**: 19 (3 phase-1-prep docs + 16 implementation/audit/closeout).
- **Single packet deferred**: monster serverbound `MonsterMovementHandle` ← `CMob::GenerateMovePath` is a 4 KB+ encode-side function that requires dedicated decompile + transcription. Documented in `_pending.md`.

## Real wire bugs identified (none fixed in this PR)

| Packet | Issue | Why deferred |
|---|---|---|
| MonsterDestroy (`CMobPool::OnMobLeaveField`) | Atlas missing optional `WriteInt(swallowCharacterId)` when `destroyType == 4` (swallowed by character-eater mob). Narrow scope — swallow eaters only (e.g. Yeti-and-Pepe boss). | Constructor signature change `NewMonsterDestroy` affects callers in `services/atlas-channel`. Defer to follow-up + 4-variant hex test. |
| MonsterControl (`CMobPool::OnMobChangeController`) | Atlas wire shape fundamentally differs from v95. Atlas writes `int8 controlType + int32 uniqueId + (if type>0: byte(5) + int32 monsterId + MonsterModel)`. v95 reads `byte controlMode + (if controlMode && opt: int32×3 seed) + int32 mobId + (if controlMode: byte aggro)`. | Atlas appears to implement an older protocol shape; v95 controllers carry a movement-seed instead of MonsterModel. **Cross-version note:** v83 has the same `CMobPool::OnMobChangeController` shape — likely the divergence is pre-v83. Defer to follow-up — needs full cross-version IDA pass to understand when the shape changed. |
| DropDestroy (`CDropPool::OnDropLeaveField`) | Atlas's destroy encoder for `destroyType == 4` (explode) writes `WriteInt(characterId)` + optional `WriteByte(petSlot)` but v95 reads `Decode2(tLeaveDelay)`. Wire desync on explode. Also `destroyType == 5` (pet pickup) may diverge — v95 reads an extra `Decode4` inside the case. | Defer to follow-up that adds the explode-delay field + tightens pet-pickup wire shape. Needs constructor update + 4-variant test. |

## Analyzer false positives surfaced (no atlas change needed)

The bulk of ❌ verdicts trace to three audit-tool limitations, not real wire bugs. Documented per-packet in `_pending.md`.

1. **Registry struct-name collision** — combat sub-domains (`monster`, `drop`, `reactor`, `pet`) reuse short struct names (`Spawn`, `Destroy`, `Damage`, `Hit`, `Movement`, `Activated`, etc.) that collide across sub-domains. `r.types` is keyed on unqualified names; last-write-wins loses per-package field-type info. This breaks `m.monster.Encode` style sub-struct expansion in MonsterSpawn / MonsterControl / PetActivated / PetMovement / etc.

2. **If/else branch double-counting** — atlas patterns like `if isMeso { WriteInt(meso) } else { WriteInt(itemId) }` (DropSpawn), `if controlled { WriteByte(1) } else { WriteByte(5) }` (MonsterSpawn), `if isSkill { WriteInt(1) } else { WriteInt(0) }` (ReactorHitRequest) flatten into two consecutive Encode entries, throwing off positions downstream. The wire is mutually exclusive — only one branch fires per call — but the analyzer can't model that.

3. **DecodeBuf / EncodeBuf sub-struct expansion gap** — when a function delegates to a sub-function the audit pipeline can't descend into (e.g. `CMob::ProcessStatSet`, `CMob::Init`, `CMovePath::OnMovePacket`, `CPet::Init`), the IDA JSON uses a `DecodeBuf` placeholder. The diff engine reports width mismatch even though wire bytes match.

These FPs are independent of region/version — they show up consistently across v83/v87/v95/JMS-v185.

## Audit-tool follow-ups recommended

1. **Registry qualified type names** — `r.types` should key on `<pkg>.<name>` so cross-sub-domain struct collisions preserve per-package field-type info needed by `resolveRecurse`.
2. **If/else mutual-exclusion modeling** — analyzer should detect `if X { WriteByte(a) } else { WriteByte(b) }` and emit one position with a noted alternation rather than two consecutive entries that misalign the diff.
3. **Dispatcher prefix annotation** — per-mob and per-pet op IDA entries currently manually prepend `Decode4(mobId)` or `Decode4(characterId) + Decode1(slot)`. Worth a helper that auto-prepends when an entry is marked as a sub-op of a dispatcher.
4. **Encode→Decode equivalence for Send\* sources** — `Send*` outbound functions in IDA do `Encode×N`. The atlas serverbound handler does `Decode×N`. The audit's diff engine should bind Encode-to-Decode equivalents by bit-width so the same JSON entry can describe both sides.
5. **Sub-function descent for delegate handlers** — `CMob::ProcessStatSet`, `CMob::Init`, `CPet::Init`, `CMovePath::OnMovePacket` are reachable from the audited top-level handlers but the pipeline doesn't descend into them. A configurable descent depth (or explicit "expand sub-call" annotations) would let the audit verify the full wire shape end-to-end.

## Per-version cross-cutting notes

### GMS v83

- `CWvsContext::SendActivatePetRequest` does not exist by name in v83. Atlas's PetSpawn handler routes through this FName; in v83 the wire request may be assembled inline in a different function (e.g. `CFuncKeyMappedMan::OnInit` for pet-keymap-driven activation, but the equivalent for the user-initiated spawn isn't bound by name). The audit pipeline correctly produces no PetSpawn report for v83.
- Atlas's `(GMS && >83) || JMS` gate on `monster/clientbound/movement.go` is verified correct against v83 IDA `CMob::OnMove` at `0x66be61` — v83 lacks the `bNotChangeAction` byte and the `multiTargetForBall` / `randTimeForAreaAttack` loops.
- v83 `CMob::OnMove`'s packed `Decode4(sEffect.m_Data)` corresponds to atlas's separate `WriteInt16(skillId) + WriteInt16(skillLevel)` — same 4 wire bytes, different field decomposition. The diff engine over-reports width mismatch on this single position; wire is correct.

### GMS v87

- All 30 FNames present. Verdict distribution matches v95.
- v87 has `CUser::OnPetPacket` but `CUserRemote::OnPetActivated` is not a named export (the dispatcher calls it via virtual offset 36). The audit can't reach the leaf by FName lookup; this is a v87-specific limitation not present in v83 or v95.

### JMS v185

- All 30 FNames present. Verdict distribution matches v95.
- Atlas has no `if Region == "JMS"` paths in monster/pet/drop/reactor encoders. Wire shape is identical to v95 across all 30 packets per the `|| JMS` gate semantics.

## Out-of-scope cleanly deferred

- **Monster serverbound `MonsterMovementHandle`** (← `CMob::GenerateMovePath`, 4 KB+ encode-side function) — defer pending decision on how to model `Encode→Decode` equivalence in the audit pipeline for `Send*` sources.
- The audit-tool follow-ups listed above are out of scope for task-065 — they would each be their own task with cross-domain (login + character + combat) benefits.

## Verification matrix run

```
go build ./libs/atlas-packet/... ./tools/packet-audit/...      # clean
go vet ./libs/atlas-packet/...                                  # clean
go vet ./tools/packet-audit/...                                 # clean
go test -race ./libs/atlas-packet/...                           # clean
go test -race ./tools/packet-audit/...                          # clean
```

`docker build` not required — no `go.mod` or `Dockerfile` files were touched.

`gitleaks` scrub of `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/{Monster,Pet,Drop,Reactor}*.md`: no `/home/` paths present.

## Commits on this branch above task-028 baseline

```
8d18e7ffe audit(combat): JMS v185 cross-version pass (phase-3-jms-185)
bb730d66d audit(combat): GMS v87 cross-version pass (phase-3-v87)
f345c30b5 audit(combat): GMS v83 cross-version pass (phase-3-v83)
db7da6540 audit(reactor): GMS v95 sub-domain audit (4 packets)
0d617d1b6 audit(drop): GMS v95 sub-domain audit (3 packets)
36b719ffe audit(pet): GMS v95 sub-domain audit (14 packets)
13c4e4f6a test(atlas-packet,pet/movement): add 5-variant round-trip baseline
09b198006 docs(task-065): code review audit reports (plan-adherence + backend-guidelines)
6483ac413 docs(task-065): post-phase-b closeout (monster-only audit scope)
544e4f44e audit(monster): GMS v95 sub-domain audit (9 clientbound packets)
bf42c5dfd test(atlas-packet,monster/movement): add 5-variant round-trip baseline
57fb768f8 test(packet-audit): MonsterSpawn/StatSet flatten safety fixtures
f38916d81 feat(packet-audit): route 31 combat-domain FNames to atlas writers/handlers
eab8e64d8 feat(packet-audit): sub-domain disambiguation via candidate.pkg
2ae7cf590 test(packet-audit): assert combat sub-structs registered
```

Plus three earlier docs commits (spec, design, plan). 19 total commits ahead of task-028 baseline (e4113fd3b).
