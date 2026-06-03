# Combat-Domain Packet Audit — Design

Version: v1
Status: Proposed
Created: 2026-05-15
PRD: `prd.md`
Prior art:
- `../task-027-atlas-packet-v95-audit/{design,plan,post-phase-b}.md` (pipeline baseline)
- `../task-028-character-domain-audit/{design,plan,post-phase-b}.md` (sibling-domain template)

---

## 1. Design Goals

This task is the third application of a pipeline (`tools/packet-audit/`) that has now shipped twice — login (28 packets, task-027) and character (52 packets, task-028). The audit pipeline, analyzer fix-set, IDA exports for v83/v87/v95/JMS-185, TypeRegistry pattern, 4-variant `pt.Variants` test sweep, `EncodeForeign` registry, `FlattenWithRegistry` cycle guard, and post-phase-b memo format are all settled. Architecture for this task is "what task-028 did, on a different 59-packet inventory" — the deltas come from the *content* of the combat domain, not from re-tooling.

Constraints driving the decisions below:

- **Don't re-design the pipeline.** task-028's analyzer fixes (early-return suffix-taint, `EncodeForeign` registration, cycle guard) shipped. Reuse them. New tooling work is in-scope only when a combat-domain packet surfaces a *new* class of analyzer limitation — and even then, the default is to log it in `_pending.md` and defer rather than expand scope.
- **Combat packets are hot in a different way than character packets.** Character packets fire per-character-action; combat packets fire per-mob-action and there are 50–200 mobs in a populated field. `monster/clientbound/movement.go` ships every monster's movement state every few hundred ms. A wire-bug there is a per-mob per-tick multiplier; the audit's job is to surface it before a tenant ships it.
- **The combat domain is split across more sub-domains than character was.** Character was one package tree; combat is four (`monster/`, `drop/`, `reactor/`, `pet/`). Each has its own clientbound + serverbound. Phasing must respect that split so a regression in one sub-domain doesn't block fixes in another.
- **No retroactive scope creep.** task-028 closed clean; task-027 ballooned. This task explicitly bounds scope: 59 packets, 4 versions, no service-layer business logic changes, no analyzer rewrites. New analyzer capability requests get logged for a sibling task.
- **Bare-handler exclusion stays.** Same rationale as task-028 §1.
- **Sub-op enum drift stays a `_pending.md` row.** Same rationale as task-028 §9. Combat domain has its own crop of sub-op packets (`monster/clientbound/stat.go`, pet command dispatch); we document, we don't fix the analyzer.

---

## 2. Architecture Overview

No new architecture. Data flow is identical to task-027 §2 and task-028 §2:

```
CSV ─→ template ─→ IDA source ─→ atlas-packet analyzer ─→ diff engine ─→ report writer
                                                ↑
                                                │
                                          TypeRegistry
```

What changes is the inputs:

| Piece                | task-028 input                                   | task-065 input                                                                |
|----------------------|--------------------------------------------------|-------------------------------------------------------------------------------|
| Atlas source         | `libs/atlas-packet/character/{cb,sb}/`           | `libs/atlas-packet/{monster,drop,reactor,pet}/{cb,sb}/`                       |
| IDA exports (v95)    | character FNames appended                        | combat FNames appended (monster/drop/reactor/pet)                             |
| IDA exports (cross)  | character FNames in v83/v87/JMS                  | combat FNames in v83/v87/JMS                                                  |
| Templates            | 10 missing character writers added across v95/v87/JMS | Expect ≥ 15 combat writers missing or drifted (monster control/spawn/damage, drop spawn/destroy, reactor hit, pet command). Verify per audit. |
| TypeRegistry         | `+AttackInfo`, `+CharacterTemporaryStat::EncodeForeign`, `+Pet`, `+Movement`, `+DamageTakenInfo` | `+MonsterModel`, `+model.Movement` (re-confirm cycle-safe), `+MultiTargetForBall`, `+RandTimeForAreaAttack`, `+activated_body` (pet), `+Position` (verify), drop coordinate block (verify) |
| Analyzer             | early-return suffix-taint shipped                | reused as-is; no new analyzer work planned                                    |

The pipeline is **read-only** against `libs/atlas-packet/`. Writes:

- `tools/packet-audit/internal/atlaspacket/registry.go` — sub-struct additions (Phase 1).
- `tools/packet-audit/cmd/run.go` — combat FName → atlas writer candidates (Phase 1).
- `libs/atlas-packet/{monster,drop,reactor,pet}/{cb,sb}/*.go` — wire-bug fixes (Phase 2).
- `services/atlas-configurations/seed-data/templates/template_*.json` — opcode / sub-op fixes (Phase 2).
- `docs/packets/audits/gms_v95/{monster,drop,reactor,pet}/` — per-packet reports (Phase 2).
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` — combat FName entries (Phase 2/3).
- `docs/packets/audits/_pending.md` — bare-handler + sub-op enum deferrals (Phase 2).
- `docs/tasks/task-065-combat-domain-audit/post-phase-b.md` — closing memo (Phase 4).

---

## 3. The hard part #1: monster spawn is the new "character spawn"

`libs/atlas-packet/monster/clientbound/spawn.go:54` ends with `w.WriteByteArray(m.monster.Encode(l, ctx)(options))`. `model.MonsterModel` is 548 LOC of sub-struct encoder (see `libs/atlas-packet/model/monster.go`). Every monster spawn — i.e. every entity entering a field — calls this. It's the combat-domain analogue of `character/clientbound/spawn.go` from task-028: a hot packet whose verdict depends on the analyzer correctly descending into a large, multi-version-branched sub-struct.

What task-028 learned and we inherit here:

- The analyzer's flat call-list collapses across nested if-branches and reports wire bytes that the *union* of branches would emit. For deeply branched packets (`character/spawn.go`, and now `monster/spawn.go` plus `model.MonsterModel.Encode`), this produces ❌ false positives. task-028 documented this as the "sub-struct descent + branch flattening" limitation and left affected packets as ❌ with a `_pending.md` row.
- The pragmatic move is to **document the limitation per packet**, not to fix the analyzer. Manual IDA cross-check stays the source of truth for spawn-class packets. Real wire bugs get filed on the spawn packet only when manual IDA inspection finds one, not when the analyzer reports ❌.

So: monster spawn is expected to land ❌ in the analyzer output. The audit report's job is to capture the manual IDA verdict in prose; the SUMMARY.md row stays ❌ with a `_pending.md` reference. This mirrors task-028's `CharacterSpawn ❌ → Phase 3 sub-struct descent` row.

### 3.1 What gets manually verified on monster spawn

The version-gated fields in `monster/clientbound/spawn.go` and the cascading branches inside `model.MonsterModel.Encode` are the locus of bugs:

- Control byte: `if (Region=="GMS" && MajorVersion>12) || Region=="JMS"` writes 1 or 5. v12 and below skip the byte. Verify against `CMobPool::OnSpawnMob` / `CMob::Init` per version.
- MonsterModel internals: stat flags, foothold, summon type, position, MBR (move-by-roll) block, skill list. Each has its own region/version gate.

Acceptance: manual IDA notes captured in `docs/packets/audits/gms_v95/monster/SpawnMonster.md` with file:line citations to `model/monster.go` and the IDA function address.

---

## 4. The hard part #2: monster movement and sub-target sub-structs

`monster/clientbound/movement.go` references three model types: `model.Movement`, `model.MultiTargetForBall`, `model.RandTimeForAreaAttack`. Two of those are combat-specific (not exercised by character-domain audits). The Movement type itself was registered for character-domain (`Move`), but the combat-side encoder differs in the wrapping fields — `bNotForceLandingWhenDiscard`, `bNotChangeAction`, `bNextAttackPossible`, `bLeft`, `skillId`, `skillLevel` — and the version gates on the multi-target/area-attack sub-structs.

### 4.1 Registration discipline (carried forward from task-028 §4.1)

For each new sub-struct introduced by combat-domain analysis:

1. Add entry to `tools/packet-audit/internal/atlaspacket/registry.go` pointing at the actual Go type with the right method name (`Encode` typically; `EncodeForeign` if the wire path is foreign-perspective).
2. Add a `registry_test.go` fixture asserting the primitive-field list.
3. Register only what the audited writers actually call. No pre-emptive registration.
4. Pair the registration commit with the first packet audit that exercises it ("audit monster_movement + register MultiTargetForBall").

### 4.2 Predicted registry additions (verify during execution)

- `model.MonsterModel` — referenced by `monster/clientbound/spawn.go` and `monster/clientbound/control.go`.
- `model.MultiTargetForBall` — referenced by `monster/clientbound/movement.go` and possibly `monster/serverbound/movement.go`.
- `model.RandTimeForAreaAttack` — same call sites as above.
- `activated_body` (pet) — referenced by `pet/clientbound/activated.go` for the v95 active-spawn body. Note: `activated.go` shown in §6.1 inlines the body; if `activated_body.go` is unused, log via `_pending.md`.
- `model.Position` — referenced across drop/reactor spawn coordinates; verify.

Per task-028 lessons, `model.Movement` registration is already in place from task-028 — confirm it covers monster/pet movement call sites or extend.

### 4.3 Cycle-safety re-check

task-028 shipped a visited-set cycle guard in `FlattenWithRegistry` (post-phase-b commit `32b585e8f`). Combat-domain sub-struct chains are flatter than character's (no `CharacterTemporaryStat` 13-bit-mask analogue), so cycle risk is low — but the registry tests added in Phase 1 are the gate: any new entry whose flattening triggers a fixture failure indicates either a real cycle or a real sub-struct call we missed.

---

## 5. The hard part #3: pet has both client-prepend dispatcher offset AND sub-op dispatch

Character-domain `spawn.go` taught us about dispatcher-layer offsets: `CUserPool` prepends `characterId` before routing, so atlas's wire includes it at offset 0. Pet packets exhibit two flavours of this:

### 5.1 Pet self vs foreign

`pet/clientbound/activated.go:50` writes `m.ownerId` (the character ID) first. Whether `ownerId` is the *receiving* character or a *foreign* character determines whether dispatcher offsets apply. Confirm against `CUserPool::OnRemotePacket` vs `CUserLocal::OnLocalPacket` in IDA. If the encoder is reused for both perspectives, the audit must surface the perspective-axis the same way task-028 did for `BuffGive` / `BuffGiveForeign`.

### 5.2 Pet command sub-op dispatch (defer to `_pending.md`)

The PRD §10 names pet command packets as sub-op-dispatched (leading byte routes to per-command payload). The analyzer cannot model this — same limitation as character `BuffGive` / `EffectSimple`. Defer per task-028 §9. Document in `_pending.md` with the IDA function name and the command-byte-to-payload-shape table that future Phase 3 tooling would consume.

### 5.3 Pet serverbound count is anomalous

PRD §4.1 lists 16 serverbound pet packets — much higher per-domain density than monster (2), drop (2), or reactor (2). PRD Open Question §13.2 asks whether this is per-command-file decomposition. Working hypothesis: yes — the v95 client multiplexes pet input through a single dispatcher byte and the atlas-packet library has chosen to give each command its own file (`chat`, `command`, `drop_pick_up`, `exclude_item`, `food`, `item_use`, `movement`, `spawn`). Each serverbound file is its own decoder; the audit treats each as a discrete row. Confirm during Phase 2b serverbound enumeration.

---

## 6. The hard part #4: PRD's "stub-only file" flag is wrong (and the audit should fix the test gap)

PRD §2 notes:

> `monster/clientbound/movement.go` and `pet/clientbound/movement.go` have no `_test.go` siblings — they appear to be stub-only files. Flag in `_pending.md` if the audit finds no encoder body to analyze.

Inspection of both files (this design phase):

- `monster/clientbound/movement.go` (89 LOC) is a full encoder with version-gated sub-struct calls. Not a stub.
- `pet/clientbound/movement.go` (46 LOC) is a full encoder calling `model.Movement.Encode`. Not a stub.

Both have **missing test files**, not missing encoders. The audit should:

1. Produce an audit report for each, same as any other packet.
2. Add a `<name>_test.go` with 4-variant `pt.Variants` round-trip coverage as part of Phase 2 (treat the missing test as a bug-fix gap, not as a `_pending.md` deferral).

Rationale: shipping fixes without tests is a regression vector. The hot-path discipline from task-028 §8 mandates byte-output assertions before any encoder mutation; without baseline tests, even a no-op fix could silently change bytes.

---

## 7. Phasing — concrete artifacts

Mirrors task-028 §5 structure, scaled to 59 packets across 4 sub-domains.

### Phase 1 — TypeRegistry extension batch (no Phase 0 needed)

task-028 needed Phase 0 to ship the analyzer early-return fix. The fix is already in `tools/packet-audit/`. **No analyzer work is planned for this task.** Phase 1 is the entry point.

Artifacts:
- `tools/packet-audit/internal/atlaspacket/registry.go` — predicted sub-struct registrations (`model.MonsterModel`, `model.MultiTargetForBall`, `model.RandTimeForAreaAttack`, `model.Position` if needed). One commit per type, each with a fixture.
- `tools/packet-audit/internal/atlaspacket/registry_test.go` — fixtures.
- `tools/packet-audit/cmd/run.go` — `candidatesFromFName` entries for combat-domain FNames (monster, drop, reactor, pet writers). One commit covering all four sub-domains, since the routing table is shared.

Exit: `go test ./tools/packet-audit/...` clean with new fixtures asserting primitive-field decomposition.

### Phase 2 — v95 combat audit (clientbound + serverbound, per sub-domain)

The four sub-domains are independent; sub-phase per domain to keep PR-sized chunks. Order chosen by criticality (hot path first):

- **2a — monster/** (15 cb + 2 sb = 17): hottest. Hot-path packets: `spawn`, `damage`, `movement`, `control`. Run audit, triage, fix, test.
- **2b — pet/** (12 cb + 16 sb = 28): largest count, mostly cold-path; serverbound dominates volume. Run audit, triage. Many sub-op deferrals expected.
- **2c — drop/** (4 cb + 2 sb = 6): smallest, mostly cold but `drop/clientbound/spawn.go` fires per-drop in field. Run audit, triage.
- **2d — reactor/** (6 cb + 2 sb = 8): cold-path. Run audit, triage.

For each sub-domain:
- Generate v95 audit reports under `docs/packets/audits/gms_v95/<domain>/`.
- Append per-packet rows to `docs/packets/audits/gms_v95/SUMMARY.md`.
- ❌ verdicts: triage to real bug | template drift | analyzer FP. Real bug → fix + 4-variant test. Template drift → fix + cite case-statement. Analyzer FP → `_pending.md` row.
- Each sub-domain ends with `go test -race ./libs/atlas-packet/<domain>/... ./tools/packet-audit/...` clean.

The audit is "done" for a sub-domain when SUMMARY.md has a verdict (✅ / ❌ + fixed / ❌ + `_pending.md`) for every listed packet. No silent skips.

### Phase 3 — Cross-version pass (v83 → v87 → JMS v185)

Per PRD §4.6. One IDA load per version, user-driven. For each version:

1. Load the IDA database.
2. Walk the combat-domain FName list established in Phase 2.
3. Populate `gms_v{83,87}.json` / `gms_jms_185.json` combat FName entries.
4. Re-run the audit with that version's IDA source + template.
5. Per-version divergence triage (same three buckets as task-028 §5 Phase 3: gate is correct → export only; gate is wrong → fix + test sweep; template drift → fix + cite).

Commit naming: `phase-3-v83`, `phase-3-v87`, `phase-3-jms-185`. Same convention as task-028.

### Phase 4 — Post-phase-b memo + verification + PR

Mirror task-028 §5 Phase 4:

- `docs/tasks/task-065-combat-domain-audit/post-phase-b.md` — five sections (final state, real wire bugs fixed, template fixes, tooling improvements, remaining work).
- `go build ./...`, `go vet ./libs/atlas-packet/...`, `go test -race ./libs/atlas-packet/...`, `go test -race ./tools/packet-audit/...` clean.
- `docker build` not expected (no `go.mod`/`Dockerfile` changes anticipated; verify before assuming).
- `superpowers:requesting-code-review` (plan-adherence + backend-guidelines).
- PR.

---

## 8. v28 coverage (carried from task-028 §6)

Same recommendation: **no for this task**. Reasons unchanged. If v28 binary surfaces mid-task, sibling-task it.

---

## 9. JMS divergence (carried from task-028 §7)

Same policy:

- Identical wire, divergent opcode → template-only fix.
- Divergent wire → `Region()=="JMS"` branch in encoder, gate-fixed + 4-variant tests.
- Extensive divergence → split file `<name>_jms.go` (precedent from task-027 §5.3).
- Hard cap on nested guards: 2 levels. 3+ → `_pending.md` + sibling task.

### 9.1 Combat-specific JMS risks

- `MoveMonster` (movement) has version-gated sub-struct calls (multi-target, area-attack). JMS may diverge on either sub-struct's encoding. Confirm during Phase 3.
- `MobDamaged` (`monster/clientbound/damage.go`) has no version gates today (uniform across all versions). If JMS IDA shows a different layout, gates must be introduced. Watch.
- Pet packets are cosmetic in older clients; v95 added pet skills + cash food. JMS v185 likely has its own pet sub-system divergences.

---

## 10. Hot-path testing discipline (combat-specific)

Combat-domain hot paths:

- `monster/clientbound/movement.go` — fires per-monster per-tick. ~50–200 mobs in a populated field × tick rate = highest packet-volume encoder in the system.
- `monster/clientbound/spawn.go` + `model.MonsterModel.Encode` — fires per monster entering field. Larger bytes-per-call than movement.
- `monster/clientbound/damage.go` — per-hit, per-target.
- `monster/clientbound/control.go` — per-monster control hand-off; tied to channel ownership.
- `drop/clientbound/spawn.go` — per-drop landing.
- `reactor/clientbound/hit.go` — per-reactor interaction.
- `pet/clientbound/movement.go` — per-pet per-tick, ~1 per character with active pet.

Patterns to use (carried from task-028 §8):

- No `reflect.*` in encoder fixes.
- No new `interface{}` parameters. Variant axes come from `tenant.Model`.
- Tests assert byte output (hex strings from IDA decompile, verified by analyzer re-run).
- Missing test files (monster + pet movement) get filled in as part of the fix, even when no behaviour change is needed. The audit's responsibility is to leave the encoder defended.
- No benchmarks added to CI; benchmark-on-suspicion remains local discipline.

---

## 11. Template fixes — opcodes vs sub-ops (carried from task-028 §9)

Same dual surface. Predicted combat-domain template drift (verify per audit):

**Opcode drift candidates:**
- `MonsterSpawn` (`OnMobEnterField`).
- `MonsterDamage` / `MonsterHealth` (`OnMobDamaged` / `OnMobHpIndicator`).
- `MoveMonster` (`OnMobMove`).
- `DropEnterField` / `DropLeaveField`.
- `ReactorChangeState` (`OnReactorChangeState`).
- `ReactorEnterField` / `ReactorLeaveField`.
- Pet command writers — heterogeneous; expect multiple opcode drifts.

**Sub-op (enum) drift candidates — defer to `_pending.md`:**
- `monster/clientbound/stat.go` — monster temporary stat set/reset; sub-op byte for stat type.
- `monster/clientbound/damage.go` — `MonsterDamageType` is currently a flat 3-value enum atlas-side (Unk1/Unk2/Unk3); IDA likely has more types. Document the mapping gap.
- Pet command sub-op dispatch — see §5.2.

Sub-op drift documentation format mirrors task-028: per-packet `_pending.md` row + the IDA function name + the case-statement-to-meaning table.

---

## 12. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Monster spawn analyzer ❌ misread as real bug; reviewer demands an analyzer fix mid-task | Medium | Medium | Phase 2a opens with a manual IDA cross-check of `SpawnMonster.md`; verdict written as "❌ analyzer FP (sub-struct descent) — manual IDA confirms ✅". `_pending.md` row tracks the analyzer follow-up. Don't expand scope. |
| `model.MonsterModel` registration triggers a cycle | Low | Medium | Phase 1 registry fixture catches it. If hit, the cycle guard logic is already in place from task-028; debug + log to `_pending.md` if guard isn't sufficient. |
| Pet serverbound 16-file count is structural and the audit ends up auditing the same dispatcher 16 times | Medium | Low | Phase 2b first action: enumerate `pet/serverbound/` and confirm the per-command decomposition. If each file is a distinct decoder, audit each. If they're all wrappers over one decoder, collapse rows and document. |
| Missing test files for `monster/cb/movement` + `pet/cb/movement` mean fixes ship without byte-baseline | Low | High | Phase 2a (monster) and Phase 2b (pet) require new test files **before** any encoder mutation. Treat as "test gap" bug per §6. |
| Cross-version pass surfaces a v83 regression on `MoveMonster` because the version-gated multi-target sub-struct wasn't fully captured at v95 | Medium | High | 4-variant test sweep is mandatory. Cross-version Phase 3 catches gate width errors. Plan owner reviews every Phase 3 diff. |
| Template opcode fixes touch hot-path writers (spawn, damage, movement) and a wrong opcode silently dispatches to a different IDA function | High | High | Every template opcode fix cites the case-statement value in the commit message. Reviewer (plan-adherence + backend-guidelines) verifies citation before approve. Don't guess template values. |
| File-level conflict with task-033/034/035/036/057/060/061 monster work | Medium | Low | This task only touches `libs/atlas-packet/monster/` (wire shape), not `services/atlas-monsters/` (behaviour). Coordinate on branch ordering at PR time, not at design time. |
| Pet sub-op enum drift is so extensive that "defer to `_pending.md`" produces a 16-row deferral list | High | Low | Acceptable. Sub-op deferrals are a known pipeline limitation; volume in pet domain is expected. Each row is a future Phase 3 tooling input. |
| Retroactive scope expansion (task-027 pattern) | Medium | Medium | Explicit out-of-scope list (§13). Every "while we're here" is a sibling task. |
| Branch hygiene drift: 59 packets × triage cycle generates a long commit log | High | Low | One commit per fix or per sub-domain audit batch. `superpowers:finishing-a-development-branch` rebase-cleans before PR. |
| gitleaks catches absolute paths in audit reports | High | Low | Phase 4 pre-PR check: `grep -r '/home/' docs/packets/audits/gms_v95/{monster,drop,reactor,pet}/` must be empty. |

---

## 13. Out of scope (explicit)

To anchor scope:

- Business logic in `services/atlas-monsters/`, `services/atlas-drops/`, `services/atlas-reactors/`, `services/atlas-pets/` (PRD §3).
- Bare-handler descent into atlas-channel service code (PRD §3 mirror of task-028 §11).
- v28 binary integration (§8).
- Sub-op enum modeling in the audit pipeline (§11).
- Analyzer enhancements beyond the existing task-028 fix-set (no Phase 0 in this task).
- Sub-struct registry coverage for any type the combat domain doesn't reference (§4.1).
- Performance work on hot packets (PRD §8 mirror).
- Login, character, npc, field, inventory, party, guild, buddy, chat domain audits — all are sibling tasks.
- Service-layer adapter changes beyond the minimum needed to wire a fix through.
- Generic packet-DSL or schema-first encoder rewrite.
- `monster/clientbound/stat.go` sub-op enum expansion — defer.
- Pet command sub-op dispatcher tracing — defer.
- Loop-aware analyzer modeling — defer (task-028 already documented as `_pending.md`).
- Coordination with task-033/034/035/036/057/060/061 monster business-logic work beyond branch ordering at PR time.

---

## 14. Reference points in the existing tree

- `libs/atlas-packet/monster/clientbound/spawn.go:54` — load-bearing sub-struct call (`m.monster.Encode`).
- `libs/atlas-packet/monster/clientbound/movement.go` (no test file) — Phase 2a fills the test gap.
- `libs/atlas-packet/monster/clientbound/damage.go:14-20` — `MonsterDamageType` flat enum; sub-op drift candidate.
- `libs/atlas-packet/monster/clientbound/stat.go` — sub-op dispatched, expected `_pending.md` row.
- `libs/atlas-packet/drop/clientbound/spawn.go` — drop hot-path; coordinate block sub-struct candidate.
- `libs/atlas-packet/reactor/clientbound/hit.go` — reactor hot-path.
- `libs/atlas-packet/pet/clientbound/activated.go:47-69` — self vs foreign perspective candidate; sub-op dispatch on `active` bool.
- `libs/atlas-packet/pet/clientbound/activated_body.go` (27 LOC) — confirm whether this is consumed by `activated.go` or orphaned.
- `libs/atlas-packet/pet/clientbound/movement.go` (no test file) — Phase 2b fills the test gap.
- `libs/atlas-packet/pet/serverbound/*.go` (16 files) — per-command decoders, audit each.
- `libs/atlas-packet/model/monster.go` (548 LOC) — primary sub-struct, registry candidate.
- `libs/atlas-packet/model/multi_target_for_ball.go`, `rand_time_for_area_attack.go` — combat-specific sub-structs, registry candidates.
- `libs/atlas-packet/model/movement.go` (320 LOC) — already registered; confirm reuse.
- `tools/packet-audit/internal/atlaspacket/registry.go` — registration site.
- `tools/packet-audit/cmd/run.go` — `candidatesFromFName` for combat FNames.
- `docs/packets/audits/gms_v95/SUMMARY.md` — combat domain rows appended.
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` — combat FName entries.
- `docs/packets/audits/_pending.md` — bare-handler + sub-op deferrals.
- `../task-028-character-domain-audit/post-phase-b.md` — closing memo template.

---

## 15. What plan-task should do next

The plan should split this design into **explicit, sequenced, small** tasks. Suggested structure (paralleling task-028's split):

- **2–4 tasks for Phase 1** — predicted registry batch + FName routing. Each registration is its own commit + fixture.
- **One task per sub-domain × cb/sb for Phase 2:**
  - 2a — monster cb (15 packets) + monster sb (2 packets).
  - 2b — pet cb (12 packets) + pet sb (16 packets, possibly grouped by command if the per-file decomposition is shared-dispatcher).
  - 2c — drop cb (4) + sb (2).
  - 2d — reactor cb (6) + sb (2).
- **One task per version for Phase 3** — 3 sub-tasks (v83, v87, JMS-185).
- **One task for Phase 4** — post-phase-b + verification + code review.

Total target: ~10–14 plan tasks. Anything more re-derives the audit per-packet; anything less hides scope.

Specifically, plan-task should answer:

- Front-load which combat-cb packets are most likely to surface real wire bugs (suggest: `monster_spawn`, `monster_movement`, `monster_damage`, `monster_control`, `drop_spawn`, `pet_activated`). Exercise the registry on hot packets before cold.
- Resolve the `pet/serverbound/` per-file question (16 files → 16 audit rows or N rows of grouped dispatcher? Phase 2b first task is enumeration).
- Resolve the `pet/clientbound/activated_body.go` consumer question (consumed by `activated.go` or orphaned? Phase 2b first task answers).
- Exact registry test format reuses task-028's fixtures; match them.
- Whether to bundle all v95 fixes into one PR or split per sub-domain (suggestion: split per Phase 2 sub-task — natural reviewable chunks).
- Phase 3 commit naming convention (`phase-3-v83`, `phase-3-v87`, `phase-3-jms-185`) carries forward from task-028.
- Coordinate with task-033/034/035/036/057/060/061 on branch ordering, but no design-time changes required.
