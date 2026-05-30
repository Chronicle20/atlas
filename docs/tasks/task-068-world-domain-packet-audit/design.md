# World-Domain Packet Audit — Design

Version: v1
Status: Proposed
Created: 2026-05-15
PRD: `prd.md`
Prior art:
- `../../../../task-027-atlas-packet-v95-audit/{design,plan,post-phase-b}.md` (login domain — pipeline shipped)
- `../../../task-028-character-domain-audit/{design,plan,post-phase-b}.md` (character domain — pipeline scaled, `EncodeForeign` registry, cycle guard, suffix-taint walker, ack pattern)

---

## 1. Design Goals

The audit pipeline is mature. Two prior tasks have shipped to `main`; the analyzer, type registry, cycle guard, and 4-variant `pt.Variants` test pattern are all in place. This task is a **third application of the same pipeline** to a different domain, not a re-design. Architecture decisions here flow from the deltas the world domain forces, not from a green-field re-think.

Constraints that drive every decision in this doc:

- **Don't re-design the pipeline.** Extend `TypeRegistry`, run the analyzer, ship audit reports + fixes. Anything else is scope creep.
- **The 57-file count straddles two qualitatively different shapes.** `portal/` is 2 files (trivial). `npc/` has 14 + 18 = 32 files with heavy `characterId`-prepend dispatcher exposure. `field/` has 21 + 2 = 23 files with multiple sub-op-dispatched effect types and the version-heaviest packet in the codebase (`set_field`).
- **`set_field.go` is the version-density outlier.** Current code is 4 sibling `Region/MajorVersion` checks at depth 1, not nested. The PRD's 3-deep exception bound is a ceiling on `_pending.md` deferral, not a license to nest. Reuse sibling guards over nesting whenever IDA evidence allows.
- **`npc/clientbound/conversation.go` is monolithic by design for this task.** Per PRD §4.5 the file holds 8 dialog-type sub-encoders (`say` / `askText` / `askYesNo` / `askMenu` / `askNumber` / `askAvatar` / `askPet` / `askBoxText`) behind a leading text-type byte discriminator. The audit produces one report file with per-type sections, each with its own verdict. The pipeline doesn't natively model per-type sections; the format extension lives in the report writer's manual annotation, not in the analyzer.
- **NPC dispatcher offset is the predictable footgun.** `CUserPool::OnPacket` prepends `characterId` for many serverbound NPC packets before the per-handler decoder runs. Atlas-side decoders that include `characterId` at offset 0 are correct; ones that don't either skip the field or expect a shifted layout. Same shape as task-028's `CUserRemote::OnPacket` prepends — we already know how to triage these.
- **task-027 ❌ and task-028 ❌ counts MUST stay byte-identical.** Phase 0 of every domain task re-runs the prior domains' audits as a regression check. If the registry additions or analyzer touches for this task perturb a login or character verdict, that's a Phase 0 STOP.
- **Bare-handler exclusion stays.** No descent into `services/atlas-channel/` or `services/atlas-npcs/` decoder code. Bare handlers with no `libs/atlas-packet` decoder defer to `_pending.md` with an explicit row each.

---

## 2. Architecture Overview

No new architecture. Data flow is identical to task-027 §2 / task-028 §2:

```
CSV ─→ template ─→ IDA source ─→ atlas-packet analyzer ─→ diff engine ─→ report writer
                                                ↑
                                                │
                                          TypeRegistry
```

What changes for this task is **what the pieces ingest** (mirror of task-028 §2):

| Piece                | task-028 input                                            | task-068 input                                              |
|----------------------|-----------------------------------------------------------|-------------------------------------------------------------|
| Atlas source         | `libs/atlas-packet/character/{clientbound,serverbound}/`  | `libs/atlas-packet/field/`, `portal/`, `npc/` (all sides)   |
| IDA exports          | `gms_v95.json` (character, appended)                       | `gms_v95.json` (world, **append** to existing)              |
| IDA exports (cross)  | `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` (new)  | Same three files, append world FNames after v95 pass        |
| Templates            | Character-domain opcodes/sub-ops                          | World-domain opcodes/sub-ops (NPC, field, portal)           |
| TypeRegistry         | `AttackInfo`, `Pet`, `MovementInfo`, `DamageTakenInfo`    | `SetField` map-header, `WarpToMap` coord block, NPC shop item, NPC conversation text-type sub-encoders, clock time-of-day (verify during execution; not exhaustive) |
| Analyzer             | Suffix-taint walker + cycle guard (both shipped)          | Re-use; no analyzer change anticipated. If one is forced, treat per §3                                                  |
| Audit-report folder  | flat `docs/packets/audits/gms_v95/`                       | flat `docs/packets/audits/gms_v95/` (see §10)              |

The audit pipeline is **read-only** against `libs/atlas-packet/`; the only writes are to `tools/packet-audit/internal/atlaspacket/registry.go` (+test), `libs/atlas-packet/{field,portal,npc}/` (wire-bug fixes), `services/atlas-configurations/seed-data/templates/template_*.json` (opcode/enum fixes), and `docs/packets/` (audit reports + IDA exports).

---

## 3. The hard part #1: analyzer reuse — when to extend vs defer

task-028 closed with an explicit list of pipeline limitations (its `post-phase-b.md` "Remaining work" table). The character-domain ❌ list is long but every entry triages to **one of three categorical pipeline limitations** that this task inherits:

1. **Sub-op dispatch on a leading-byte type discriminator** (`BuffGive/Foreign`, `Effect*`, `StatusMessage*`, `Attack`, `CharacterDamage`, `CharacterMovement`). Flat analyzer sees only the outermost `Decode` call sequence.
2. **Loop-count modeling** (`KeyMapChange`, `CharacterKeyMap`). Analyzer can't follow data-driven loop bounds.
3. **Deep sub-struct descent** (`CharacterSpawn`, `CharacterAppearanceUpdate`, `GW_CharacterStat` width-gated fields). Analyzer's flattener bottoms out before unwinding chains of registered sub-structs.

The world domain will hit all three categories. Specifically:

- **Sub-op dispatch**: `field/clientbound/effect.go` already breaks the type into 5+ distinct Go encoder structs (`EffectSummon`, `EffectTremble`, `EffectString`, `EffectBossHp`, `EffectRewardRullet`). Each gets its own audit row — that's the *good* form of sub-op modeling. `effect_weather.go`, `clock.go`, and `npc/clientbound/conversation.go` will look like the *bad* form (one Go struct, multiple wire shapes behind a mode byte).
- **Loop-count**: `npc/clientbound/shop_list.go` flattens a commodity-array loop. Same shape as character's `KeyMapChange`. Verdict will be ❌ with a "loop-count tool limitation" annotation; verify item-entry sub-struct bounds manually against IDA.
- **Deep sub-struct descent**: `field/clientbound/set_field.go` embeds `CharacterData.Encode()`, which is the same descent chain that produces the character `CharacterSpawn` FP. Either inherit the same `🔍 manual review` verdict pattern or skip the descent and audit `set_field` envelope-only with a `🔍` for the embedded `CharacterData` block. Recommendation: **envelope-only**, because `CharacterData`'s wire shape is already exercised by `CharacterSpawn` in task-028 and the v95 audit row for that already triages to "Phase 3: sub-struct descent".

**The analyzer fix policy for this task**: do not extend the analyzer. If a verdict's only escape route is an analyzer change, mark `❌ tool-limitation` and document in the audit report's "tool limitation" footer (mirror of task-028 form). The PRD §4.7 carve-out for TypeRegistry extensions stays; analyzer code stays untouched unless a defect surfaces (cycle, panic, false ✅). The task-028 closeout already lists "Phase 3: per-mode IDA sub-function trace" as the deferred follow-up; we don't pre-empt that work here.

If the world-domain audit re-run on prior domains (Phase 0) flips a prior verdict because of a registry addition cascading through `FlattenWithRegistry`, that's a STOP — investigate and either roll back the registry entry or split a sibling task. Same rule as task-028 §3.5.

---

## 4. The hard part #2: world-specific registry extensions

The PRD §4.7 names five likely additions. Triage now, refine in execution:

| Sub-struct                              | Source type / method                          | Consumed by                                                | Confidence |
|-----------------------------------------|-----------------------------------------------|------------------------------------------------------------|------------|
| `SetField` map-header                   | `libs/atlas-packet/model/...` map-header type | `field/clientbound/set_field.go` envelope                  | Medium — actual symbol TBD during execution; envelope-only audit may not need this entry at all. |
| `WarpToMap` coordinate block            | TBD — likely inline in `warp_to_map.go`       | `field/clientbound/warp_to_map.go`                         | Low — single-call site, an inline shape is fine; register only if the analyzer surfaces an unresolved type. |
| NPC shop item entry                     | `libs/atlas-packet/npc/...` commodity type    | `npc/clientbound/shop_list.go` commodity loop              | High — needed because shop_list is loop-count limited; the item entry sub-struct must analyze cleanly so the per-iteration bytes are verifiable. |
| NPC conversation text-type sub-encoders | `npc/clientbound/conversation.go` per-type    | `npc/clientbound/conversation.go` self                     | High — each text-type branch is its own encoder block; per PRD §4.5 they each get a section in the audit report. |
| Clock time-of-day                       | `field/clientbound/clock.go` per-mode         | `field/clientbound/clock.go` self                          | Medium — clock has at least two modes (hour-minute-second wall clock vs countdown timer); register the per-mode sub-encoder pair if needed. |

**Registration discipline (inherited from task-028 §4.1)**:

- Register only types the world domain *actually calls*. Don't pre-emptively register every type in `libs/atlas-packet/model/`.
- One registry entry + one `registry_test.go` fixture per addition. Mirror existing fixture format.
- Land the registry entry in the same commit as the first audited packet that references it. Keeps PRs reviewable.

**Cross-domain ripple guardrail**: every registry addition triggers a regression-run of the login and character SUMMARY rows. If a row flips ❌→✅ or ✅→❌ on a non-world packet, STOP and investigate before merging the registry entry. Diff engine treats registry additions as additive but real-world experience (task-028 commits `b1af67f6d`, `32b585e8f`) shows that registry changes can cascade through `FlattenWithRegistry` to verdict-affecting depth in non-obvious ways.

---

## 5. The hard part #3: `set_field.go` nesting policy

The PRD §4.6 grants `set_field.go` a **3-deep** nested guard exception. Reading the current code clarifies what this means:

- Current `set_field.go` Encode/Decode each contain **4 sibling guards at depth 1**, not nested:
  - `(GMS && v>83) || JMS` — pre-channelId short
  - `JMS` only — extra `Decode1 + Decode4`
  - `(GMS && v>28) || JMS` — damage-seed count branch (3 vs 4)
  - `(GMS && v>83) || JMS` — post-CharacterData logout-gift block
- None of these are nested. They are co-equal version branches sequenced through the same encoder.
- The 3-deep ceiling kicks in only if IDA reveals that one of these blocks **internally** also branches on a different gate dimension (`region × major × sub-version` or sub-version alone). Empirically: GMS v83 vs GMS v87 inside an already-`(GMS && v>83)`-guarded block is one such case.

**Operationalized policy**:

- 1-deep sibling guards (the current shape): **always preferred**. No nesting.
- 2-deep nested guards: allowed where IDA confirms inner branch is required AND splitting into siblings would require duplicating non-version-dependent bytes.
- 3-deep nested guards: **only in `set_field.go`**, only if IDA confirms three orthogonal axes (region, major-version, and a tertiary like sub-version or stock-vs-modified Nexon). Cite all three axes in the audit report header.
- 4+ deep: STOP, defer to `_pending.md`. Same hard cap as task-028 §7.

**Every other encoder stays under the 2-deep cap.** This includes `warp_to_map.go`, `effect.go`, `effect_weather.go`, `clock.go`, every NPC packet, and `transport.go`. PRD §4.6 is explicit.

The audit report for `SetField.md` MUST have an explicit "Nesting policy: 3-deep exception per PRD §4.6" header line so reviewers know to expect deeper guards there than elsewhere.

---

## 6. The hard part #4: `conversation.go` per-dialog-type audit shape

PRD §4.5 mandates one audit report covering 8 dialog-type sub-encoders inside a 360-line monolithic file. The pipeline doesn't natively produce per-section reports — it produces one report per `.go` writer file.

**Operationalized shape**:

- Audit-time, the analyzer produces one verdict for the file as a whole (almost certainly ❌ because of mode-byte sub-op dispatch — same shape as character `EffectSimple`).
- The report writer's automatic output becomes the **skeleton**. A human-authored "Per-dialog-type breakdown" section is appended manually. Each of the 8 dialog types gets a subsection:
  - **Verdict** (✅ / ⚠️ / ❌) for the sub-encoder.
  - **IDA dispatcher branch citation** — `CUser::OnQuestionAsk@<addr>` (or equivalent) case-statement value matching the mode byte.
  - **Wire-shape comparison** — atlas writer's byte list vs IDA's `Decode*` list for that branch.
  - **Fix block** if ❌, citing the encoder block in `conversation.go` and the test that exercises it.
- SUMMARY.md row for the file uses the *worst* verdict across the 8 sub-sections. If any one sub-section is ❌, the file row is ❌. If all 8 are ✅, the file row is ✅. ⚠️ if mixed but no ❌s.
- 4-variant `pt.Variants` tests cover each fixed dialog-type branch **independently** — a fix on `askMenu` doesn't blanket-test `say`.

**What if the text-type byte is not statically resolvable for a branch**: per PRD §4.5, defer that branch to `_pending.md` with a one-line rationale and mark the verdict ⚠️ for the file. Likelihood: medium — `conversation.go` may receive the mode byte from a runtime parameter rather than encoding it as a literal per-method. If this is the case for more than 2 of 8 branches, document it as a tooling limitation and the file gets ⚠️ overall with the unresolvable rows enumerated.

**Do not refactor `conversation.go`.** The PRD §3 non-goal is explicit. Splitting into one file per dialog type would also re-introduce all 8 paths into a `Region/MajorVersion` matrix and complicate cross-version verification. Audit-as-is, fix-in-place.

---

## 7. The hard part #5: NPC dispatcher offset (`characterId` prepend)

Character serverbound packets in task-028 hit this once (the `CUserRemote::OnPacket` dispatcher prepends `characterId` before the per-handler payload). NPC has a similar shape but routed through `CUserPool::OnPacket` for player-facing NPC actions: the dispatcher reads `characterId` (4 bytes) off the wire, looks up the user, then dispatches to the per-action decoder. Atlas-side decoders for `npc/serverbound/action.go`, `start_conversation.go`, etc. must either:

- Include the `characterId` field at offset 0 of their `Decode` method (atlas-consistent — looks identical to character serverbound dispatch).
- Or treat the field as already-consumed by the dispatcher layer (atlas reads starting at offset 4 conceptually).

This shape isn't a wire bug per se — it's an architectural choice about *where* the dispatcher boundary lives. The audit's job is to **document the boundary** (in the report's header) and verify the post-`characterId` payload against the IDA per-handler decoder. If atlas reads `characterId` at offset 0 AND the IDA per-handler decoder starts after the prepend, that's correct and matches the character-domain pattern (verdict ✅, with a one-line ack of the dispatcher offset).

**Predicted findings**:

- All NPC serverbound packets either consistently include `characterId` (correct) or consistently omit it (audit-as-is per atlas-channel handler contract).
- Inconsistency between two NPC serverbound packets — one includes, one omits — is a real bug; the inconsistent one fails to decode in the dispatcher. Fix lands in `libs/atlas-packet/npc/serverbound/<name>.go`.
- The dispatcher offset is also a likely source of ⚠️ verdicts where the per-action IDA function is reachable but atlas's matching field name is `unknown` or `field` instead of `characterId` — annotate but do not gate the verdict on field naming.

---

## 8. Field-effect sub-op enum drift (parallel to character `effect.go`)

`field/clientbound/effect.go` is the world-domain analog to character's `effect.go`. It defines 5+ separate Go encoder structs (`EffectSummon`, `EffectTremble`, `EffectString`, `EffectBossHp`, `EffectRewardRullet`), each writing its own leading byte (the field-effect type discriminator) plus its payload.

This is the *good* form of sub-op dispatch — the analyzer can audit each struct independently. Each gets its own audit row in SUMMARY.md.

What needs verification per IDA:

- The leading-byte value each struct writes (the field-effect type). These come from `CField::OnFieldEffect` (or similarly named IDA dispatcher) case-statement values.
- Each struct's payload field widths vs IDA's per-case decoder.
- Whether any field-effect types exist in IDA that have no atlas writer at all — those are gap entries, added to `_pending.md` (atlas-side bare implementation), not new packets to add.

`effect_weather.go` is the *bad* form — one struct, mode byte set in constructor methods (`NewEffectWeatherActive` vs `NewEffectWeatherInactive`), single `Encode` method. Analyzer sees the union. Expected verdict: ❌ tool-limitation. Fix is to manually verify against IDA's per-mode decoder branches and annotate the audit report. No refactor.

`clock.go` is also the *bad* form — one struct, two modes (wall-clock vs countdown timer). Same treatment as `effect_weather.go`.

The pipeline doesn't model sub-op enums (task-028 design §9 already documented this as a deferred tooling item). For this task, manually annotate the audit reports with sub-op value tables — same format as task-028's `StatusMessage*` reports.

---

## 9. Cross-version pass — phasing inherited from task-028

The PRD §4.8 cadence (v95 complete → v83 batch → v87 batch → JMS v185 batch) is the inherited task-028 pattern. The only adjustment for the world domain:

- **JMS v185 set_field divergence is the highest-risk single-file finding.** `set_field.go` already has explicit JMS branches at depth 1; v185 IDA may reveal that JMS itself sub-divides (early-JMS vs v185-JMS field-limits encoding). If so, the v185 pass produces the first known case of `Region() == "JMS"` requiring a major-version split, and the fix is a structural rewrite of the JMS branches. Hard cap from §5: 3-deep total. If the rewrite needs more, defer to `_pending.md`.
- **NPC dispatcher offset (§7) likely doesn't drift across versions.** The dispatcher prepends `characterId` for all 4 versions; that's a structural property of `CUserPool::OnPacket`. Audit the offset assumption once in v95 and re-verify only as a sanity check during cross-version passes.
- **Template opcode shifts for NPC and field are version-correlated.** `NpcConversationWriter`, `FieldEffectWriter`, `SetFieldWriter`, `WarpToMapWriter` all live in the per-version dispatcher tables. Each opcode shift identified during a non-v95 pass gets the same one-integer fix in `template_<version>.json` as task-028 §9 documented.

---

## 10. Audit-report folder structure — flat, not per-domain

The PRD §5 implies a per-domain subfolder layout (`docs/packets/audits/gms_v95/field/`, `portal/`, `npc/`). Reading the current state of `docs/packets/audits/gms_v95/` shows task-027 and task-028 left everything flat — login and character packet reports all live at the top level of `gms_v95/`.

**Decision: stay flat.** Rationale:

- The PRD's acceptance criteria (§10) reference only `docs/packets/audits/gms_v95/`, not subfolders. The "byte-identical SUMMARY rows" requirement for prior domains is easier to satisfy with one combined SUMMARY.md than with a multi-rooted layout.
- A move to per-domain subfolders would also require migrating the existing 80+ task-027/task-028 reports for symmetry. That's out of scope.
- Per-file SUMMARY.md rows already include the domain in the FName (`Npc*`, `Field*`, `SetField`, etc.), so the flat layout is searchable.

If a future task wants to reorganize by domain, that's a sibling refactor task. This task's audit reports land flat.

The SUMMARY.md remains the single shared index. Adding 57 world-domain rows alongside the existing 81 (28 login + 52 character + 1 misc) brings the index to ~138 rows. That's still scannable.

---

## 11. Phasing — concrete artifacts

Mirror of task-028 §5 phasing, world-domain inputs:

### Phase 0 — Regression baseline + analyzer sanity (gate)

Re-run the existing v95 audit unchanged against the current pipeline. Verify:

- Login SUMMARY rows byte-identical to pre-task state (28 rows).
- Character SUMMARY rows byte-identical to pre-task state (52 rows).
- No analyzer panics, no new cycle-guard fires.

Artifacts:
- Pipeline re-run output (no commit needed if no diffs).
- One commit: `audit(world): phase-0 regression baseline confirms prior-domain verdicts unchanged`.

Exit: prior SUMMARY rows confirmed unchanged.

### Phase 1 — TypeRegistry extension batch (predicted)

Register the high-confidence types up front per §4 triage:

- NPC shop item entry type (`shop_list` commodity loop).
- NPC conversation text-type sub-encoder cluster (8 types).
- `SetField` map-header type if envelope-only descent surfaces an unresolved sub-call (verify during execution).
- Clock time-of-day per-mode sub-encoder pair if needed.

Each registration lands with one `registry_test.go` fixture asserting the registered type analyzes to the expected primitive-field list. Predicted, not exhaustive — Phase 2 surfacings get added then. Don't register `WarpToMap` coord block until §4 confidence rises (likely never).

Exit: registry test suite green; SUMMARY rows for login + character byte-identical.

### Phase 2 — v95 world audit

Run the audit, triage findings, ship fixes. Sub-phases by sub-domain (PRD §4.1 ordering):

- **2a — `portal/serverbound/` (2 files)**: smallest domain, audit-and-go. Expected outcome: 2 ✅ rows; if anything else, the audit pipeline has a bug.
- **2b — `field/serverbound/change.go` (1 file)**: serverbound field-change packet, single-file pass.
- **2c — `field/clientbound/` non-effect (8 files: `affected_area_*`, `kite_*`, `set_field`, `transport`, `warp_to_map`)**: the version-density hotspots. `set_field` and `warp_to_map` are the highest-risk fixes. `transport.go` is in-place since `instance-based-transports` merged.
- **2d — `field/clientbound/` effect cluster (`effect.go`, `effect_weather.go`, `clock.go`)**: sub-op-dispatched, per §8. Manual annotation expected.
- **2e — `npc/clientbound/` non-conversation (7 files)**: `action`, `guide_talk`, `shop_list`, `shop_operation*`, `spawn`, `spawn_request_controller`. `shop_list` is the loop-count limitation case.
- **2f — `npc/clientbound/conversation.go`**: the per-dialog-type audit (§6). Largest single-file effort in the task. One report file, 8 per-type sub-sections.
- **2g — `npc/serverbound/` (9 functional files)**: `action`, `continue_conversation*` (3 files), `shop*` (4 files), `start_conversation`. Dispatcher-offset verification per §7.
- **2h — `_pending.md` updates**: bare-handler exclusions, unresolvable sub-op branches, loop-count limitations, deep sub-struct deferrals. Each gets an explicit row.

Each sub-phase ends with a verdict-triaged commit set: fix commits individually, audit-report commits batched per sub-phase. Mirror of task-028 §5 Phase 2.

The audit is "done" when SUMMARY.md shows a verdict for every one of the 57 packets OR an explicit `_pending.md` entry for any deferred. No silent skips.

### Phase 3 — Cross-version pass (v83 → v87 → JMS v185)

Per PRD §4.8 cadence. One IDA database at a time, user-driven swap. For each version:

1. User loads the IDA database.
2. Walk the world-domain FName list (already established by Phase 2's v95 export).
3. Populate matching `gms_v{83,87}.json` / `gms_jms_185.json` for world FNames.
4. Re-run the audit with that version's IDA source and template.
5. For each divergence vs v95 atlas-packet behaviour:
   - Existing `Region/MajorVersion` gate already correct → no atlas change; export row captures evidence.
   - Atlas gate wrong → fix the gate, sweep tests across 4 variants, document.
   - Template opcode drift → fix the template, cite case-statement value.

Each version's pass ships as its own commit batch: `audit(world): GMS v83 cross-version pass`, etc. The JMS v185 pass is the highest-attention version per §9.

### Phase 4 — post-phase-b.md + finishing-a-development-branch

Mirror of task-027/task-028 closing pattern. Write `docs/tasks/task-068-world-domain-packet-audit/post-phase-b.md` with sections:

- Final state (packets audited, verdict counts, IDA-export coverage).
- Real wire bugs fixed (table: packet, file, IDA citation, fix one-liner, affected versions).
- Template opcode/enum fixes (table: template file, old → new, IDA case-statement, reason).
- Tooling improvements (registry additions; analyzer should be untouched — if it isn't, explain why).
- Remaining work (deferred sub-op modeling, loop-count modeling, deep sub-struct descent — likely identical entries to task-028's remaining-work table).

Then run verification:
- `go build ./...` clean.
- `go vet ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
- `go test -race ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
- gitleaks scrub — `grep -r '/home/' docs/packets/audits/gms_v95/` empty.
- `docker build -f services/atlas-configurations/Dockerfile .` if templates changed in structure-affecting ways.

Code review via `superpowers:requesting-code-review` (plan-adherence + backend-guidelines) before PR.

---

## 12. PRD Open Questions — Resolutions

PRD §9 left three questions for this phase. Resolve as follows.

### 12.1 `set_field.go` current gate boundaries vs 3-deep cap

The PRD's "8 existing Region/MajorVersion references" double-counts encode+decode mirrors. The actual structural count is **4 sibling guards at depth 1, repeated in encode and decode**. None are currently nested. The 3-deep cap is a future ceiling, not a current overrun.

**Resolution**: `set_field.go` is presently within budget. The 3-deep exception remains pre-allocated for cases where IDA cross-version evidence forces nesting (e.g., a JMS-only branch that itself requires major-version sub-division). If Phase 2 or Phase 3 surfaces such a case, use the exception; otherwise leave the file at depth 1.

### 12.2 JMS v185 field-format divergence — third gate dimension needed?

Speculative until IDA confirms. Operating assumption: `Region() == "JMS"` is sufficient as a single axis. v185 may sub-divide within JMS (early-JMS pre-185 vs 185+) but the atlas codebase currently has no JMS major-version axis, only the `Region()` discriminator.

**Resolution**: defer concrete answer to Phase 3 step 4. If a v185-specific finding requires sub-major-version gating within JMS, that's a structural change introducing a new gate axis. Use the existing `MajorVersion()` API on the tenant model — it's already region-agnostic. Document the introduction explicitly in `post-phase-b.md`. Cap as in §5: 3-deep total in `set_field.go`, 2-deep elsewhere.

### 12.3 `conversation.go` text-type byte static resolvability

The analyzer can almost certainly resolve cases where the text-type byte is written as a literal inside an `if branch`. It cannot resolve cases where the byte is taken from a struct field set by the constructor.

**Resolution**: defer per-branch confirmation to Phase 2f. Plan-task assumption: **at least 6 of 8 dialog-type branches are statically resolvable** (those that write the mode byte as a literal). Up to 2 may require deferral to `_pending.md` with a one-line rationale. If more than 2 require deferral, escalate per PRD §4.5 — the file's overall SUMMARY row goes ⚠️ and the unresolved branches are enumerated.

---

## 13. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Registry addition cascades through `FlattenWithRegistry` and flips a prior-domain ✅ to ❌ (or vice versa) | Medium | High | Phase 0 + Phase 1 each re-run prior SUMMARY rows. Any drift triggers a STOP-and-investigate gate before merging the registry change. |
| `conversation.go` per-type audit shape doesn't fit the existing report-writer template | High | Medium | Generate analyzer skeleton; append manually-authored "Per-dialog-type breakdown" section. Don't rebuild the report writer. If template fragmentation grows beyond conversation, then revisit. |
| `set_field.go` JMS v185 pass forces 4+ deep nesting | Low-medium | High | §5 hard cap — defer to `_pending.md` with explicit rationale. Don't paper over with `Region()` string-suffix tricks. |
| NPC dispatcher offset inconsistency hides a real bug | Medium | Medium | §7 audit-report header includes dispatcher-offset boundary explicitly per packet. Cross-packet inconsistency surfaces during Phase 2g review. |
| `effect.go` 5-struct dispatch is incomplete — IDA reveals field-effect types with no atlas struct | Medium | Low-medium | Each unimplemented type → `_pending.md` row (atlas-side feature gap, not wire bug). Don't add new structs to `effect.go` outside of follow-up tasks. |
| Sub-op dispatched bad-form files (`effect_weather.go`, `clock.go`) eat manual-annotation budget | Medium | Medium | Manually annotate per task-028 `StatusMessage*` pattern. Cap effort at 4 hours per file; if over, defer to `_pending.md`. |
| Loop-count limitation produces a real wire bug in `shop_list.go` that the audit can't catch | Medium | High | Manual bounds check during Phase 2e: read `shop_list.go` against IDA `CUserShopDlg::SendShopList` (or equivalent). Document expected commodity count limit (likely 16 or 32 per shop) and verify atlas's loop bound matches. |
| Template opcode drift across 4 versions explodes the cross-version pass | Medium | Medium | One commit per opcode fix per version, citing IDA case-statement. Same cadence as task-028. |
| `transport.go` post-merge state has bugs the merged-PR test sweep missed | Low | Medium | Audit `transport.go` in-place per PRD §11. Existing tests should catch most; new audit catches the rest. |
| `_pending.md` row growth from this task crowds prior tasks' entries | Low | Low | Group new entries under a `## task-068` heading in `_pending.md`. Mirror existing per-task grouping if present. |
| gitleaks catches absolute paths in audit reports | High | Low | Phase 4 pre-PR check: `grep -r '/home/' docs/packets/audits/gms_v95/` empty. Same as task-028. |
| Pipeline panic or cycle when registering NPC text-type sub-encoder cluster (8 types) | Low-medium | Medium | Cycle guard from task-028 (`32b585e8f`) covers this. If a panic surfaces, log fixture and inspect — could be a registry-entry typo. |
| Mid-task scope creep into NPC scripting engine | Medium | Medium | PRD §3 non-goal is explicit. Refuse any "while we're here, fix the NPC script handling for X" excursion. Log to `_pending.md` instead. |
| Audit-report ack footer accidentally added before final run | Medium | Low | Convention: ack footer is the LAST line written. If a re-run is needed, `git checkout HEAD -- <report.md>` first, per task-028 closing memo. |

---

## 14. Out of scope (explicit)

- Bare-handler descent into atlas-channel or atlas-npcs service code (PRD §3 non-goal).
- Refactoring `conversation.go` into per-type files (PRD §3).
- NPC scripting engine business logic (PRD §3).
- Sub-op enum modeling in the audit pipeline (§3 — limitation acknowledged, not fixed).
- Loop-count modeling in the audit pipeline (§3 — same).
- Deep sub-struct descent in the audit pipeline (§3 — same).
- Performance work on hot packets (`set_field` is hot but PRD §8 is silent on perf; assume no regression as task-028 §8 does).
- Login (task-027) and character (task-028) verdict re-runs beyond regression-only.
- Monster, drop, mob-spawn, party, guild, buddy, chat domain audits — sibling tasks.
- v28 binary integration (inherited from task-028 §6; deferred).
- Service-layer changes beyond the minimum needed to wire a fix through (PRD §3).
- Migration of audit reports to per-domain subfolders (§10).
- Generic packet-DSL or schema-first encoder rewrite (carried forward from task-027 §12).

---

## 15. Reference points in the existing tree

- `libs/atlas-packet/field/clientbound/set_field.go` — version-density hotspot. 4 sibling guards at depth 1; the 3-deep exception target.
- `libs/atlas-packet/field/clientbound/warp_to_map.go` — coordinate-block envelope; cross-version-likely.
- `libs/atlas-packet/field/clientbound/transport.go` — recently merged from `instance-based-transports`; in-place audit.
- `libs/atlas-packet/field/clientbound/effect.go` — *good* form of sub-op dispatch (5+ Go structs, one per type).
- `libs/atlas-packet/field/clientbound/effect_weather.go` — *bad* form (one struct, mode byte in constructor).
- `libs/atlas-packet/field/clientbound/clock.go` — *bad* form, two modes.
- `libs/atlas-packet/field/clientbound/{affected_area_*,kite_*}.go` — affected-area / kite spawn/destroy/error; small files, expected mostly-clean verdicts.
- `libs/atlas-packet/field/serverbound/change.go` — field-change request; single file.
- `libs/atlas-packet/portal/serverbound/script.go` — 2-file portal domain; quick pass.
- `libs/atlas-packet/npc/clientbound/conversation.go` — 360-line monolithic dialog-type dispatch. Per-type audit per §6.
- `libs/atlas-packet/npc/clientbound/shop_list.go` — loop-count limitation case (commodity array).
- `libs/atlas-packet/npc/clientbound/{action,guide_talk,shop_operation,shop_operation_body,spawn,spawn_request_controller}.go` — straightforward clientbound writers.
- `libs/atlas-packet/npc/serverbound/{action,start_conversation,continue_conversation*,shop*}.go` — dispatcher-offset boundary verification per §7.
- `tools/packet-audit/internal/atlaspacket/registry.go` — additions per §4.
- `tools/packet-audit/internal/atlaspacket/registry_test.go` — fixture format to mirror.
- `tools/packet-audit/internal/atlaspacket/analyzer.go` — DO NOT TOUCH unless a panic or new cycle surfaces.
- `services/atlas-configurations/seed-data/templates/template_gms_{83,87,95}_1.json` — opcode/enum sites for world-domain writers (`SetFieldWriter`, `WarpToMapWriter`, `NpcConversationWriter`, `FieldEffectWriter`, `FieldEffectWeatherWriter`, `NpcShopWriter`, etc.).
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — JMS opcode/enum site.
- `docs/packets/audits/gms_v95/SUMMARY.md` — append world-domain rows to existing flat index.
- `docs/packets/audits/gms_v95/` — flat per-packet reports.
- `docs/packets/ida-exports/gms_v95.json` — append world FNames during Phase 2.
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_jms_185}.json` — append during Phase 3 per version.
- `docs/packets/ida-exports/_pending.md` — append world bare-handler exclusions and unresolvable branches.
- `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` — closing-memo template.
- `docs/tasks/task-028-character-domain-audit/post-phase-b.md` — closing-memo template (preferred — more recent, world-relevant pattern lists).

---

## 16. What plan-task should do next

The plan should split this design into **explicit, sequenced, small** tasks. Suggested structure:

- **2–3 tasks for Phase 0 + Phase 1** (regression baseline run + registry batch with predicted entries).
- **One task per Phase 2 sub-phase (a–h)**: 8 sub-tasks. Each ends with verdict-triaged commit set.
- **One task per cross-version pass (Phase 3 — v83, v87, JMS v185)**: 3 sub-tasks.
- **One task for Phase 4** (post-phase-b.md + verification + code review).

Total target: ~14–16 plan tasks. Anything more is the plan re-deriving the audit per-packet; anything less is hiding scope.

Specifically the plan should answer:

- The order of Phase 2 sub-phases. Recommended: 2a (portal) → 2b (field/serverbound) → 2c (field/clientbound non-effect) → 2d (field/clientbound effect cluster) → 2e (npc/clientbound non-conversation) → 2f (conversation.go) → 2g (npc/serverbound) → 2h (_pending.md sweep). This front-loads the easy wins, leaves the conversation.go effort and dispatcher-offset verification for later in the sub-phase sequence when accumulated context is highest.
- The exact list of high-confidence registry additions for Phase 1 (only the High-confidence rows from §4; Medium-confidence rows wait for Phase 2 evidence).
- The fixture format for new registry tests (mirror existing `CharacterStat::Encode` / `AttackInfo` patterns).
- The commit-naming convention. Recommended: `audit(world): <sub-phase> <file>` for audit-report commits; `fix(packet/<domain>): <packet> — <one-line>` for wire-bug fix commits; `audit(world): GMS v<N> cross-version pass` for Phase 3 batches.
- The pre-PR rebase strategy. Recommended: rebase-clean (squash repetitive `audit(world):` commits per sub-phase) before opening PR. The audit-report commits are mechanically generated; their individual history isn't load-bearing. Fix commits stay individual.
- Whether to bundle the v83/v87/JMS-185 passes into one PR or split per version (suggestion: one PR for the whole task per task-028 precedent; reviewer fatigue is a real risk but per-version PRs introduce three review cycles).
- Per-type fixture inputs for the 8 `conversation.go` dialog-type branches (mode byte value, payload shape, IDA dispatcher case address) — to the extent these are knowable pre-execution.

---

## 17. What is NOT being decided here (deferred to plan / execution)

- The exact symbol names for the world-domain sub-struct types (`SetField` map-header, `WarpToMap` coord block, NPC shop item entry). Inspect during execution.
- Whether `transport.go` needs registry additions for instance-route sub-structs. Read the merged code in Phase 2c.
- The IDA dispatcher case-statement values for `NpcConversationWriter`, `FieldEffectWriter`, etc. — these come from Phase 2 IDA work, not pre-execution speculation.
- The final count of `_pending.md` rows. Likely 10–25 per task-028 precedent, but the world domain's sub-op dispatch density may push higher.
- Whether to extend `pt.Variants` with a 5th variant for JMS pre-185 (early JMS). Defer to Phase 3 evidence.
- Whether Phase 0's regression baseline run shows any character-domain ❌ that was masked by analyzer drift since task-028 shipped. If yes, that's an unexpected regression and a STOP — investigate before proceeding.
