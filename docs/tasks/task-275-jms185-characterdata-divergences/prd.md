# JMS 185 CharacterData Divergences — Master Level & Extended SP — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
Source: https://github.com/Chronicle20/atlas/issues/1544
---

## 1. Overview

`GW_CharacterData` carries two variable-shape fields whose presence is decided by a
client-side predicate rather than by a flag on the wire: the per-skill master-level
`Decode4` (`is_skill_need_master_level`) and the character's SP field, which is either a
single `Decode2` or a variable-length extended-SP array (`ExtendSP::Decode`). Because
neither is length-prefixed at the point of divergence, a server that answers either
predicate differently than the client does not produce a "wrong value" — it produces a
byte-offset shift that corrupts every subsequent field of the packet and, in practice,
crashes the client on field entry.

Atlas models both predicates in region/version-shared helpers:
`skill.NeedsMasterLevel` (`libs/atlas-constants/skill/model.go:115`) and `isEvanJob`
(`libs/atlas-packet/character/data.go:271`), gated at `data.go:316` (encode) and
`data.go:397` (decode). Both were derived from GMS clients at or below v87 and are wrong
for the newer clients Atlas now serves. Task-273's field-by-field JMS 185 sweep found
them; they were left unfixed because the crash under investigation (level 1, job 0)
could not reach either, and because a naive edit to a shared helper moves GMS behavior.

This task corrects both predicates against the IDA-derived per-version truth, for every
client version Atlas supports, without changing the bytes any already-verified cell
emits for a character an Atlas tenant can actually create today.

## 2. Goals

Primary goals:

- Make `skill.NeedsMasterLevel` answer per version, including the 430-434 (Dual Blade)
  arm that GMS v92, GMS v95 and JMS 185 all have and Atlas has none of.
- Make the extended-SP branch of `CharacterData` selection region-and-version correct, so
  JMS takes the extended-SP path for the jobs its client expects, and so the predicate
  body matches the client's (`job/1000 == 3 || job/100 == 22 || job == 2001` on the
  versions that use it).
- Prove both with byte-exact fixtures, and prove no already-verified GMS cell moved.

Non-goals:

- Introducing Resistance / 3000-3999 job identities, Dual Blade gameplay, skill trees,
  SP allocation, or any behavior beyond packet shape.
- Modelling `is_ignore_master_level_for_common` (GMS v95 @0x47cc20) beyond what is needed
  to keep v95 correct for jobs Atlas can create; see §9.
- Changing `DecodeChangeStat` / `OnStatChanged` (the same predicate appears there; see §9).
- Any change to atlas-channel behavior other than passing the tenant through if required.

## 3. User Stories

- As a JMS 185 player with a Dual Blade character, I want field entry to render my
  character instead of desyncing, so that the client does not corrupt or crash on
  `CharacterData`.
- As a JMS 185 player with an Evan (job 2001 or 22xx), I want my SP field encoded as the
  extended-SP array my client reads, so that every field after SP lands at the right
  offset.
- As an Atlas maintainer, I want one authority per client predicate, keyed by tenant
  version, so a future version bring-up cannot silently inherit a GMS-shaped rule.
- As an Atlas maintainer, I want a regression fixture per already-verified version, so a
  fix to one region cannot quietly move another region's wire bytes.

## 4. Functional Requirements

### 4.1 Client truth (IDA-derived, re-confirmed 2026-08-28)

`is_skill_need_master_level` — Dual Blade arm (`jobId/10 == 43`):

| Version | Address | Behavior for 430-434 |
|---|---|---|
| GMS v83 | @0x4e8f04 | no arm — falls through to `job%100 != 0 && job%10 == 2` |
| GMS v84 | @0x4f0ad2 | no arm — same fallthrough |
| GMS v87 | @0x508f33 | arm present, returns **false** for all of 430-434 (`v1/10 == 43 \|\| !(v1%100)` → 0) @0x508fa4 |
| GMS v92 | @0x4792f0 | `get_job_level(job) == 4` **or** skill ∈ {4311003, 4321000, 4331002, 4331005} @0x479371 |
| GMS v95 | @0x47ccb0 | same as v92, plus a leading `is_ignore_master_level_for_common(skill)` @0x47cc20 early-out |
| JMS 185 | @0x47d2a8 | same `get_job_level == 4` + same four ids @0x47d2f9; **no** Evan exception list |

Evan arm, for contrast (already modelled): every listed version returns true when
`get_job_level(job)` is 9 or 10 (i.e. job 2217/2218); GMS v84+ additionally returns true
for skills 22111001, 22141002, 22140000; GMS v83 and JMS 185 do not.

Extended-SP predicate (selects `ExtendSP::Decode` instead of the SP `Decode2`):

| Version | Address | Predicate |
|---|---|---|
| GMS v84 | inline in `GW_CharacterStat::Decode` @0x4e9da4 | `job/100 == 22 \|\| job == 2001` |
| GMS v87 | inline @0x501e9c (`idiv 100`, `cmp 16h`; `cmp 7D1h`) | `job/100 == 22 \|\| job == 2001` |
| GMS v92 | inline @0x4f50f4 (`/1000 == 3`), @0x4f5100 (`/100 == 22`), @0x4f510f (`== 7D1h`) | `job/1000 == 3 \|\| job/100 == 22 \|\| job == 2001` |
| GMS v95 | `is_extendsp_job` @0x4f1e30 | `job/1000 == 3 \|\| job/100 == 22 \|\| job == 2001` |
| JMS 185 | `sub_5163A2` @0x5163a2, called @0x50eda2 | `job/1000 == 3 \|\| job/100 == 22 \|\| job == 2001` |

Extended-SP body (all versions): `Decode1` count, then per entry `Decode1` master-level
index + `Decode1` sp (JMS: `sub_50E8B0` @0x50e8b0, count @0x50e8c6, pair @0x50e8de /
@0x50e8e0). Atlas already writes this shape; only the gate is wrong.

### 4.2 `skill.NeedsMasterLevel` must become version-aware

- FR-1: The function takes `tenant.Model` (per the confirmed decision) in place of the
  `evanExceptions bool`, and derives every arm — Evan exception list, Dual Blade arm,
  common fallthrough — from region + major version internally. Callers pass the tenant
  they already hold (`data.go:670`, `data.go:696`).
- FR-2: The Dual Blade arm is modelled exactly per §4.1: absent for GMS < 87; returns
  false for GMS 87; returns `jobLevel(job) == 4 || skill ∈ {4311003, 4321000, 4331002,
  4331005}` for GMS ≥ 92 and for JMS.
- FR-3: `get_job_level` semantics used by the arm must come from the existing
  atlas-constants job model, not a re-derivation; if no equivalent exists, the task adds
  one alongside the identities added in task-204 rather than inlining a magic table.
- FR-4: The Evan arm's current behavior (true for 2217/2218; GMS ≥ 84 exception list;
  no exceptions on JMS or GMS v83) is preserved bit-for-bit.
- FR-5: The stale comment at `model.go:110-114` ("atlas-constants defines no 430-434
  job") is corrected — `BladeRecruit`…`BladeMaster` (430-434) exist since task-204 and are
  present in the jms_185_1 availability table.

### 4.3 Extended-SP gate must become region- and version-correct

- FR-6: The gate at `data.go:316` (encode) and `data.go:397` (decode) selects extended SP
  when the tenant's client uses it: GMS ≥ 84, or JMS. It must not remain GMS-only.
- FR-7: The predicate body models the client's full rule on the versions that carry it:
  `job/1000 == 3 || job/100 == 22 || job == 2001` for GMS ≥ 92 and JMS 185; the narrower
  `job/100 == 22 || job == 2001` for GMS 84 and 87. The `job/1000 == 3` arm is modelled
  even though no Atlas job identity lands in 3000-3999 today (confirmed: the only 3xx
  identities are Bowman 300 … Marksman 322), with a test pinning it.
- FR-8: Encode and decode stay mirror images and read the gate from one exported helper,
  so they cannot diverge.
- FR-9: For any job an Atlas tenant can create today, no already-verified GMS cell may
  change a single byte. GMS v84/v87 keep the narrow predicate; GMS v92/v95 gain the
  `/1000 == 3` arm, which is unreachable for existing identities.

### 4.4 Interaction between the two fixes

- FR-10: A JMS Dual Blade is not an extended-SP job (`430/1000 == 0`), so it must take the
  plain SP `Decode2` path while still taking the master-level `Decode4` per skill. A
  fixture must cover this combination so the two fixes are not conflated.

## 5. API Surface

No HTTP/REST surface changes. The changed surfaces are Go library signatures:

- `skill.NeedsMasterLevel(skillId Id, evanExceptions bool) bool` →
  `skill.NeedsMasterLevel(skillId Id, t tenant.Model) bool` (exact shape is a design-phase
  call; the confirmed decision is that it takes the tenant).
- A new exported predicate in `libs/atlas-packet/character` (or atlas-constants/job if
  that is the better home) replacing the unexported `isEvanJob`, taking job id + tenant.

Both are internal to the monorepo; no external consumer exists outside
`libs/atlas-packet/character` and the two atlas-channel comment references at
`services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:76-80`.

## 6. Data Model

No persistence change. No new entity, column, or migration. `CharacterData.Skills[].NeedsMasterLevel`
(`data.go:74`) remains a derived field recomputed from the authority on both encode and
decode, per the existing comment at `data.go:668`.

## 7. Service Impact

| Component | Change |
|---|---|
| `libs/atlas-constants/skill` | `NeedsMasterLevel` signature + Dual Blade arm + version table; comment correction; test expansion |
| `libs/atlas-constants/job` | Possibly a `JobLevel` accessor if none exists (FR-3) |
| `libs/atlas-packet/character` | Extended-SP gate at `data.go:316`/`:397`; `isEvanJob` replaced; call sites at `:670`/`:696` pass tenant; new byte fixtures |
| `services/atlas-channel` | Comment references only (`character_data.go:76-80`); no behavior change expected |
| `docs/packets/evidence/**`, `docs/packets/audits/**` | Re-pin any cell whose bytes or read-order documentation changes |

No Kafka topic, REST route, or UI change.

## 8. Non-Functional Requirements

- **Correctness over convenience:** every arm cites an IDA address in a code comment, in
  the style already used at `model.go:100-114` and `data.go:295-330`.
- **Multi-tenancy:** the predicates are resolved from the tenant already in scope; no
  global or process-level version state.
- **No performance concern:** both predicates are integer arithmetic on a hot-ish encode
  path; the version resolution must not allocate per skill.
- **Observability:** none required; a desync is diagnosed from fixtures, not logs.

## 9. Open Questions

1. `GW_CharacterStat::DecodeChangeStat` (v84 @0x4ea2da, v87 @0x502252, v92 @0x4f5210)
   uses the same extended-SP predicate for the `OnStatChanged` path. Atlas's stat-change
   encoder is a separate code path; whether it shares the fixed helper is a design-phase
   determination. It is out of scope to *change* stat-change wire shape here, but if it
   reads the same helper the change reaches it implicitly and must be fixture-covered.
2. GMS v95's `is_ignore_master_level_for_common` (@0x47cc20) has no Atlas counterpart. Its
   contents must be read in design; if it excludes only skills no Atlas job can hold, it
   is documented and skipped, otherwise it becomes a v95 arm.
3. Whether `get_job_level` (JMS @0x47d347, v95 @0x47cb90) is faithfully expressible from
   the existing job advancement model, or needs its own ported table.
4. Whether the extended-SP predicate belongs in `libs/atlas-constants/job` (alongside the
   job identities) rather than `libs/atlas-packet/character`; the master-level rule
   already lives in atlas-constants, so symmetry argues for the move.

## 10. Acceptance Criteria

- [ ] `skill.NeedsMasterLevel` takes the tenant and resolves every arm from it; no caller
      passes a hand-derived `region == "GMS"` boolean.
- [ ] Table test pins the Dual Blade arm per version: no arm on v83/v84, false on v87,
      `jobLevel == 4 || skill ∈ {4311003, 4321000, 4331002, 4331005}` on v92, v95, jms185.
- [ ] Table test pins the unchanged Evan arm (2217/2218 on all; the three-skill exception
      list on GMS ≥ 84 only) — existing assertions in
      `libs/atlas-constants/skill/model_test.go` still pass, adapted to the new signature.
- [ ] The extended-SP gate fires for JMS; table test pins `job/1000 == 3 || job/100 == 22
      || job == 2001` on GMS ≥ 92 and jms185, and the narrow form on GMS 84/87.
- [ ] Byte-exact `CharacterData` fixture: JMS 185 Dual Blade with at least one
      master-level skill and one non-master-level skill, plain SP short present.
- [ ] Byte-exact `CharacterData` fixture: JMS 185 Evan (22xx or 2001) taking the
      extended-SP array.
- [ ] Regression fixtures prove no byte movement for an already-verified GMS cell with a
      job an Atlas tenant can create (at minimum v84 and v95).
- [ ] Encode and decode remain mirror images; a round-trip test covers both new shapes.
- [ ] Every new arm carries an IDA address citation in a comment.
- [ ] `model.go:110-114`'s stale "no 430-434 job" claim is corrected.
- [ ] Any packet-coverage evidence record whose bytes or documented read order changed is
      re-pinned, and `packet-audit` checks exit 0.
- [ ] Flagless `tools/verify.sh` exits 0.
