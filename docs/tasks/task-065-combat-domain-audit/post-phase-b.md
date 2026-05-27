# Task-065 Post-Phase-B — Combat-Domain Audit Closeout

## Final state

- **Packets audited**: 30 (24 clientbound + 6 serverbound) — monster (9 cb), pet (6 cb + 8 sb), drop (2 cb + 1 sb), reactor (3 cb + 1 sb).
- **Cross-version passes**: GMS v83 ✅, GMS v87 ✅, JMS v185 ✅.

### Verdict roll-up per version (combat domain) — after wire fixes + IDA corrections

| Version | ✅ | 🔍 | ❌ | Total |
|---|---|---|---|---|
| GMS v95 (source of truth) | 13 | 2 | 18 | 33 (30 packets + 1 added MovementHandle + 2 from MonsterControl now passing prefix) |
| GMS v83 | 12 | 0 | 19 | 31 |
| GMS v87 | 13 | 1 | 17 | 31 |
| JMS v185 | 12 | 2 | 17 | 31 |

Wire-bug fixes (`MonsterDestroy` swallow-id, `DropDestroy` explode/pet-pickup tail) shifted `MonsterDestroy` to ✅ across all 4 versions. `MonsterControl` now ❌ only on the analyzer's MonsterModel sub-struct expansion gap (positions 0–3 match cleanly). `MonsterMovementHandle` newly audited at 🔍.

- **IDA-export coverage**: GMS v95, GMS v83, GMS v87, JMS v185 — combat FNames populated for each.
- **Total commits on branch above task-028 baseline**: 19 (3 phase-1-prep docs + 16 implementation/audit/closeout).
- **MonsterMovementHandle (sb)** — audited after re-analysis. Decompiled JMS v185 `CMob::GenerateMovePath@0x6e8892` and verified atlas's `MovementRequest` encoder matches byte-for-byte across all v95+JMS-gated blocks. IDA entries added to gms_v95.json (0x651100) + gms_jms_185.json. v83/v87 IDA address lookups deferred to next IDA swap. Verdict: 🔍 (sub-struct expansion FP per the standard analyzer limitation — wire is correct).

## Real wire bugs — fixed in-branch

| Packet | Issue | Resolution |
|---|---|---|
| MonsterDestroy (`CMobPool::OnMobLeaveField`) | Atlas missing optional `WriteInt(swallowCharacterId)` when `destroyType == 4` (swallowed by character-eater mob like Yeti-and-Pepe boss). | **Fixed in `ac174269b`.** Added `DestroyTypeSwallow` enum + `swallowCharacterId` field + `NewMonsterDestroyBySwallow` constructor. v95 audit now ✅. 5-variant round-trip + 9-byte wire-length check pass. |
| DropDestroy (`CDropPool::OnDropLeaveField`) | Atlas's destroy encoder for `destroyType == 4` (explode) wrote `WriteInt(characterId)` + optional `WriteByte(petSlot)` but v95 reads `Decode2(tLeaveDelay)`. Wire desync on explode. `destroyType == 5` (pet pickup) was also wrong — v95 reads an extra `Decode4` inside the case. | **Fixed in `ac174269b`.** Replaced `petSlot int8` field with `explodeDelay int16` + `petPickupExtra uint32`. Encoder switches on `destroyType` correctly. Legacy `NewDropDestroy` constructor preserved; new `NewDropDestroyExplode` for the explicit-delay path. 5-variant round-trip + explicit 7/13-byte wire-length checks pass. |
| MonsterControl (`CMobPool::OnMobChangeController`) | Originally flagged as a fundamental shape mismatch (atlas writes `int8 controlType + int32 uniqueId + ...`; v95 reads `byte controlMode + 3×int32 seed + int32 mobId + ...`). | **Not a real bug** (fixed via IDA-entry correction in `e32a3d809`). Re-analysis after loading JMS v185 IDA showed the v95 `moveRandSeed` block is dev-mode-only (`CClientOptMan::GetOpt(2)`). Atlas server never enables opt 2, so seeds never appear on production wire. Atlas's wire shape matches production v83/v87/v95/JMS-v185 through positions 0–3 (controlMode + mobId + aggro + templateId). The hardcoded `byte(5)` at the aggro position is a *semantic* concern (atlas always sends 5 regardless of real aggro state) but not a wire-shape bug — width and position match. |

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

## Post-PR follow-up work landed in-branch

After the initial post-phase-b closeout, the deferred items surfaced during code review were re-opened and worked through on the same branch.

### PRD scope reconciliation (item 10)

`prd.md` §2, §4.1, §4.2, and §11 originally claimed 59 combat packets (37 cb + 22 sb). The actual `libs/atlas-packet/{monster,drop,reactor,pet}` source inventory is 31 packets (20 cb + 11 sb), which matches the `plan.md` per-phase breakdown (monster 9+1, pet 6+8, drop 2+1, reactor 3+1 = 31). The PRD over-counted; plan.md and the delivered audit were correct. PRD updated in-place to reflect the actual inventory, with a clarifying note that `stat.go` packs both StatSet and StatReset and `activated_body.go` is a wrapper rather than an independent packet.

### Combat template opcode audit (item 2)

PRD §4.4 required cross-checking combat template opcodes against IDA dispatcher case-statement values and landing fixes for any drift. Full findings in [`template-audit.md`](template-audit.md). Summary:

- ✅ **No writer/handler name string drift** — every combat opcode entry in every template uses the canonical name declared as a `const` in `libs/atlas-packet/`.
- ✅ **No combat-domain opcode collisions** within any template.
- ⚠️ **Template coverage gap surfaced as separate concern** — only `template_gms_83_1.json` is fully populated; v95 has zero combat entries; v12/v87/v92/jms_185 each have only 6 monster entries. Channel-servers booted against the under-populated templates emit `Service declares writer [...] but tenant config has no opcode mapping for it.` warnings from `libs/atlas-opcodes/producer.go:31` and silently drop combat traffic. This belongs in a follow-up task gated on IDA access (same gating as the v83/v87 `CMob::GenerateMovePath` lookup) and is **not** task-065's PRD §4.4 acceptance scope, which targets drift in existing entries.
- ⏸ **IDA dispatcher case-statement verification deferred** — the audit pipeline records function addresses + call sequences, not dispatcher case-statement values. Verifying opcode values against the client dispatcher requires per-version IDA decompile.

No template files were modified. No "Template opcode fixes" table is added to this ledger because no drift was found in existing entries.
