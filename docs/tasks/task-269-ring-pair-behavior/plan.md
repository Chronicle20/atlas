# Ring Pair Field Behavior Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a purchased cash-shop ring pair observable in the field — the partner's identity reaches the client at every encoder site that feeds `CUserPool::On{Couple,Friend,Marriage}RecordAdd`, so the client's own proximity logic can render the ring effect.

**Architecture:** One shared codec (`libs/atlas-packet/model/ring.go`) supplies four encoder sites: `CharacterData` (own records, count-prefixed fixed-width), `CharacterSpawn` and `CharacterAppearanceUpdate` (remote players, flag-gated field blocks), and `CharacterInfo` (a bare marriage bool). `atlas-cashshop` gains three read-only fields on `ring.RestModel`; `atlas-channel` gains a read-only `ring` package with a per-(tenant, character) cache that joins ring halves to equipped cash assets by `cashId`.

**Tech Stack:** Go 1.27, `libs/atlas-packet`, `libs/atlas-tenant`, `libs/atlas-constants`, GORM, JSON:API (`atlas-rest`), Kafka (`atlas-kafka`), `tools/packet-audit`.

**Spec:** [`design.md`](design.md) (PRD at [`prd.md`](prd.md))

## Global Constraints

- **Branch base.** This branch is rebased onto `origin/task-240-cash-shop-stub-operations` (head `dd7e0bbb4`), NOT `main`. The `cash_rings` entity, the `ring` package, `GET /rings`, and `RING_PURCHASED` come from that branch. Do not rebase onto `main` until PR #1426 merges.
- **Empty-path invariant (PRD FR-9).** For a character with no `ACTIVE` equipped ring half, sites A, B and D MUST emit bytes identical to the pre-task encoder: `WriteShort(0)`×3 (A), `WriteByte(0)`×3 (B), `WriteBool(false)` (D). Site C is the sole exception — its frame is corrected (Task 6), so its empty-path bytes change by exactly the four removed trailing bytes.
- **Version gate idiom.** Match the surrounding lines of the file you edit. `libs/atlas-packet/character/data.go` uses `t.Region() == "GMS" && t.MajorVersion() > N` exclusively. `spawn.go` and `info.go` mix that with `t.IsRegion("GMS") && t.MajorAtLeast(N)`. Do not introduce a third idiom. `tenant.Model` provides all four: `Region()`, `MajorVersion()`, `IsRegion()`, `MajorAtLeast()` (`libs/atlas-tenant/tenant.go:21,25,88,93`).
- **Legacy arm untouched (PRD FR-8).** The GMS v29..v60 arm of `info.go` has no marriage bool. Do not add one.
- **Existing `> 28 || JMS` gate in `encodeRings` is preserved verbatim** so the v29..v60 shape of `CharacterData` does not move.
- **No invented values.** Every byte in a fixture traces to a decompile line or a checked-in export entry, per `docs/packets/audits/VERIFYING_A_PACKET.md` §5. `ida=` addresses in `packet-audit:verify` markers must be real.
- **Never commit to `main`.** All work lands on `task-269-ring-pair-behavior`.

## Derivation conflict resolved by Task 1

The 4-byte field trailing the two 8-byte SNs in the couple/friendship block is
**not yet pinned**. Two sources disagree:

| Source | Says the field is |
|---|---|
| `docs/packets/ida-exports/gms_v83.json`, `gms_v87.json`, `gms_v95.json` → `CUserRemote::OnAvatarModified` | `dwPairCharacterId` (4 bytes) |
| `gms_jms_185.json` → `CUserRemote::OnAvatarModified` | `friendship pair characterId (per entry)` |
| `gms_jms_185.json` → `CUserRemote::Init` @0xa52876 | `couple-ring itemId (per entry)` |
| `design.md` §2 raw disassembly (v83 @0x97fb5b) | `itemId`, passed as `a4` to `CUserPool::OnCoupleRecordAdd` |

Task 1 settles this from IDA and every later task consumes its answer. **No
codec is written before Task 1 reports.** The field is referred to below as
`PairRing.Trailing` only in Task 1; from Task 2 onward it carries the name
Task 1 pinned.

---

## File Structure

**New files**

| Path | Responsibility |
|---|---|
| `libs/atlas-packet/model/ring.go` | `PairRing`, `MarriageRing`, `RingSet` + `EncodeField`/`DecodeField` (sites B, C) and `EncodeRecords`/`DecodeRecords` (site A) |
| `libs/atlas-packet/model/ring_test.go` | codec unit + version-gate tests |
| `services/atlas-channel/atlas.com/channel/ring/model.go` | channel-side `Model` (one half) + `RingSet` assembly |
| `services/atlas-channel/atlas.com/channel/ring/rest.go` | `RestModel` + `Extract` |
| `services/atlas-channel/atlas.com/channel/ring/requests.go` | `GET /rings?filter[characterId]=` URL builder |
| `services/atlas-channel/atlas.com/channel/ring/processor.go` | `Processor` + `GetRingSet` |
| `services/atlas-channel/atlas.com/channel/ring/cache.go` | per-(tenant, character) cache + `EvictTenant` |
| `docs/tasks/task-269-ring-pair-behavior/coverage-manifest.yaml` | PRD FR-18 declaration |

**Modified files**

| Path | Change |
|---|---|
| `libs/atlas-packet/character/data.go:106-121,765-776` | `Rings` field; `encodeRings`/`decodeRings` become record-driven |
| `libs/atlas-packet/character/clientbound/spawn.go:48-57,174-176` | `rings` param; three literals become `RingSet.EncodeField` |
| `libs/atlas-packet/character/clientbound/appearance_update.go:16-55` | `rings` field; frame correction; tenant plumbing |
| `libs/atlas-packet/character/clientbound/info.go:63-65,131` | `hasMarriageRing` param; `WriteBool` becomes data-driven |
| `services/atlas-cashshop/.../ring/{model.go,rest.go,processor.go,provider.go}` | `cashId`, `partnerCashId`, `partnerName` |
| `services/atlas-channel/.../socket/writer/{character_spawn.go,character_info.go,character_data.go}` | pass ring data |
| `services/atlas-channel/.../kafka/consumer/asset/consumer.go:419` | pass ring data to appearance update |
| `services/atlas-channel/.../kafka/consumer/cashshop/consumer.go:496-538` | cache invalidation |

---

## Task 1: Pin the trailing 4-byte ring field from IDA

Settles the `dwPairCharacterId` vs `itemId` conflict above. Produces no code — a
written derivation the rest of the plan consumes. This task is deliberately
first and deliberately blocking.

### Files

- `docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md` — new file; the derivation record
- `docs/packets/ida-exports/gms_v83.json` — read-only; the export whose comment is under test
- `docs/packets/ida-exports/gms_jms_185.json` — read-only; the export that contradicts itself
- `docs/reverse-engineering.md` — read-only; how to resolve the IDA session by binary name

Module root: none (docs only).

Patterns to copy: `docs/tasks/task-252-jukebox-cash-item/coverage-manifest.yaml` (the "positive proof recorded" comment style for a contested derivation).

- [ ] **Step 1: Resolve the IDA sessions**

Per `docs/reverse-engineering.md`, call `mcp__ida-pro__idb_list` and match by
binary name: `MapleStory_dump.exe.i64` (GMS v83) and `GMS_v95.0_U_DEVM.exe.i64`
(GMS v95). Pass the matched session id as the `database` argument on every
subsequent call. Port-based selection is dead.

- [ ] **Step 2: Decompile the three record registrars on v95 (symbols present)**

```
mcp__ida-pro__decompile   CUserPool::OnCoupleRecordAdd
mcp__ida-pro__decompile   CUserPool::OnFriendRecordAdd
mcp__ida-pro__decompile   CUserPool::OnMarriageRecordAdd
```

v95 carries the symbolised signature
`OnCoupleRecordAdd(const _LARGE_INTEGER &, CUser *, long)`. The question is what
the **third** parameter (`long`) is. Record what the function stores it into and
what reads that member afterwards — specifically whether it reaches an item-id
consumer (a `CItemInfo`/`GetItemInfo` lookup, a WZ node fetch) or a
character-id consumer (a user-pool lookup, a name resolve).

- [ ] **Step 3: Cross-check the caller on v83**

Decompile `CUserRemote::Init` @0x97f55d and `CUserRemote::OnAvatarModified`
@0x98367e. The couple `Decode4` is at @0x97fb5b (design.md §2). Record which
`CUser` member offset it lands in, then run `mcp__ida-pro__xrefs_to_field` on
that offset to find every reader.

- [ ] **Step 4: Check the marriage arm too**

The exports name the marriage triple `dwMarriageCharacterID`,
`dwMarriagePairCharacterID`, `nWeddingRingID` (v83/v87/v95 `OnAvatarModified`).
`design.md` §3.1 names the first field `MarriageId`. Confirm or refute against
`CUserPool::OnMarriageRecordAdd(unsigned long, CUser *, long)` — the `unsigned
long` first parameter is the contested one.

- [ ] **Step 5: Write the derivation record**

Create `docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md` with,
for each of the three blocks, a table of `field name | wire width | IDA address
| what reads it`. State the verdict for the trailing 4-byte field in one
sentence, and state explicitly whether `gms_v83.json`'s
`dwPairCharacterId` comment or `gms_jms_185.json` `Init`'s `itemId` comment is
wrong — one of them is, and the wrong one is a defect in a checked-in export
that the next task's field names must not inherit.

- [ ] **Step 6: If the export comment is wrong, say so but do NOT edit the export**

An export is a harvested artifact regenerated by `packet-audit export`. A
hand-edit would be silently overwritten. Record the discrepancy in the
derivation doc and note it for a follow-up `export` refresh.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md
git commit -m "derive(task-269): pin the trailing 4-byte ring field from IDA"
```

---

## Task 2: The shared ring codec — field blocks (sites B and C)

### Files

- `libs/atlas-packet/model/ring.go` — new file; `PairRing`, `MarriageRing`, `RingSet`, `EncodeField`, `DecodeField`
- `libs/atlas-packet/model/ring_test.go` — new file
- `libs/atlas-packet/model/avatar.go` — read-only; the version-gated small-model precedent
- `docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md` — new file, created by Task 1; read-only here. Task 1's verdict supplies the third field's name and meaning

Module root: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/model/avatar.go:47-110` (struct + `Encode`/`Decode` both calling `tenant.MustFromContext(ctx)` and branching on region/version). Test setup: `libs/atlas-packet/model/asset_test.go:14-38` (`TestEncodeSlotVersionGate` — build, encode under two explicit `pt.CreateContext(region, major, minor)` calls, compare exact lengths and byte prefixes).

**Interfaces**

- Produces, for Tasks 4, 5, 6:
  - `type PairRing struct { OwnSN int64; PartnerSN int64; <third field per Task 1> uint32 }`
  - `type MarriageRing struct { <per Task 1> uint32; PartnerCharacterId uint32; ItemId uint32 }`
  - `type RingSet struct { Couple *PairRing; Friendship *PairRing; Marriage *MarriageRing }`
  - `func (r RingSet) EncodeField(w *response.Writer, t tenant.Model)`
  - `func (r *RingSet) DecodeField(rd *request.Reader, t tenant.Model)`

- [ ] **Step 1: Write the failing test**

`TestRingSetEncodeField` in `libs/atlas-packet/model/ring_test.go`. Table-driven
over `pt.Variants` indices, setup copied from
`libs/atlas-packet/model/asset_test.go:14-38`.

Fixture values used by every case:

```
couple     = PairRing{OwnSN: 0x1122334455667788, PartnerSN: 0x99AABBCCDDEEFF00, Third: 0x00001234}
friendship = PairRing{OwnSN: 0x0102030405060708, PartnerSN: 0x1112131415161718, Third: 0x00005678}
marriage   = MarriageRing{First: 0x000000AA, PartnerCharacterId: 0x000000BB, ItemId: 0x0000CCDD}
```

| case | variant | RingSet | expected bytes (hex, little-endian per `response.Writer`) | total len |
|---|---|---|---|---|
| GMS empty | `GMS v83` | `RingSet{}` | `00 00 00` | 3 |
| GMS couple only | `GMS v83` | `Couple` set | `01` `8877665544332211` `00FFEEDDCCBBAA99` `34120000` `00` `00` | 24 |
| GMS friendship only | `GMS v83` | `Friendship` set | `00` `01` `0807060504030201` `1817161514131211` `78560000` `00` | 24 |
| GMS marriage only | `GMS v83` | `Marriage` set | `00` `00` `01` `AA000000` `BB000000` `DDCC0000` | 15 |
| GMS all three | `GMS v83` | all set | concatenation of the three populated blocks, no separators | 45 |
| GMS v48 empty | `GMS v48` | `RingSet{}` | `00 00 00` | 3 |
| GMS v95 all three | `GMS v95` | all set | byte-identical to the `GMS v83` all-three case | 45 |
| JMS empty | `JMS v185` | `RingSet{}` | `00 00 00` | 3 |
| JMS couple only | `JMS v185` | `Couple` set | `01` `01000000` `8877665544332211` `00FFEEDDCCBBAA99` `34120000` `00` `00` | 28 |

The `GMS v48` and `GMS v95` cases exist to assert design.md §2's conclusion that
the GMS block is version-stable from v48 to v95 — they must produce the same
bytes as `GMS v83` for the same input. If they do not, the codec has an
unintended gate.

The JMS shape is `flag / Int count / per entry {16-byte SN pair, Int third}` per
`gms_jms_185.json` `CUserRemote::Init` @0xa52876 calls 34-41; count is 1 for a
populated ring and the block is absent (flag only) when nil. The JMS marriage
arm is identical to GMS (`flag` + 3×`Int`).

Add `TestRingSetFieldRoundTrip`: for each variant and each of the five
populated combinations, `EncodeField` then `DecodeField` and assert field-by-field
equality, including that a nil pointer decodes back to nil.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./model/ -run 'TestRingSet' -v`
Expected: FAIL — `undefined: RingSet`.

- [ ] **Step 3: Write the codec**

`libs/atlas-packet/model/ring.go`. Exported struct fields (not the unexported +
getter shape) because `RingSet` is constructed by callers in three packages and
carries no invariant to protect; `model.Avatar` and `model.Pet` are the same.
Nil pointer is the "no ring" signal.

`EncodeField` branches once on `t.Region() == "JMS"`; there is no version gate
inside the GMS arm (design.md §2 OQ-1). Document that absence in a comment
citing v48 `CUserPool::OnUserEnterField` @0x6b277b calls 49-60 and v95
`CUserRemote::OnAvatarModified` @0x954110, so a later reader does not "fix" it
by adding one.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./model/ -run 'TestRingSet' -v`
Expected: PASS, all cases.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/ring.go libs/atlas-packet/model/ring_test.go
git commit -m "feat(atlas-packet): shared ring field codec for spawn and avatar-update"
```

---

## Task 3: The shared ring codec — record blocks (site A)

Separate task from Task 2 because it is a different wire shape (count-prefixed
fixed-width records, not flag-gated field blocks) with its own derivation and
its own reviewer gate.

### Files

- `libs/atlas-packet/model/ring.go` — modify; add `CoupleRecord`, `FriendRecord`, `MarriageRecord`, `RingRecords`, `EncodeRecords`, `DecodeRecords`
- `libs/atlas-packet/model/ring_test.go` — new file in Task 2; extended here with record tests
- `libs/atlas-packet/model/padded_string.go` — read-only; check whether it already provides the fixed-width name write before adding one
- `docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md` — new file, created by Task 1; appended here in Step 1

Module root: `libs/atlas-packet`.

**Interfaces**

- Consumes from Task 2: `PairRing`, `MarriageRing`.
- Produces, for Task 4:
  - `type RingRecords struct { Couple []CoupleRecord; Friend []FriendRecord; Marriage []MarriageRecord }`
  - `func (r RingRecords) EncodeRecords(w *response.Writer, t tenant.Model)`
  - `func (r *RingRecords) DecodeRecords(rd *request.Reader, t tenant.Model)`

- [ ] **Step 1: Derive the intra-record field splits**

The record **widths** are already fixed by design.md §2 OQ-5 and must not move:

| Record | Width | v83 decoder | v95 decoder |
|---|---|---|---|
| `GW_CoupleRecord` | `DecodeBuffer(0x21)` = 33 | `sub_4E48B0` @0x4e48b0 | @0x4f2b60 |
| `GW_FriendRecord` | `DecodeBuffer(0x25)` = 37 | `sub_4E48CE` @0x4e48ce | @0x4f2b70 |
| `GW_MarriageRecord` | `DecodeBuffer(0x30)` = 48 | @0x4e4856 | @0x4f2b50 |

design.md §2 proposes 33 = 8 (own SN) + 8 (partner SN) + 4 (itemId) + 13
(`sPairCharacterName`), citing the v95 decompiler naming a local
`sPairCharacterName` at 0x4fde40. **Confirm that split from IDA** before writing
the encoder, and confirm the two remaining splits (37 = 33 + 4 of what; 48 = ?).
Append the result to `ring-field-derivation.md` as a fourth table. If a split
cannot be pinned, stop and report — do not pad with a guessed field.

- [ ] **Step 2: Write the failing test**

`TestRingRecordsEncode` in `libs/atlas-packet/model/ring_test.go`, same setup
shape as Task 2.

| case | variant | records | expected |
|---|---|---|---|
| empty, modern | `GMS v83` | none | `0000` `0000` `0000` (three `WriteShort(0)`) — 6 bytes |
| empty, legacy | `GMS v28` | none | `0000` only (the `> 28` gate excludes friend/marriage) — 2 bytes |
| empty, JMS | `JMS v185` | none | `0000` `0000` `0000` — 6 bytes |
| one couple record | `GMS v83` | 1 couple | `0100` + 33 bytes + `0000` + `0000` — 39 bytes |
| one of each | `GMS v83` | 1/1/1 | `0100` + 33 + `0100` + 37 + `0100` + 48 — 126 bytes |
| name truncation | `GMS v83` | 1 couple, partner name 20 chars | record byte 20..32 is the first 13 bytes of the name, no NUL terminator past the field |
| name padding | `GMS v83` | 1 couple, partner name `"Ab"` | record byte 20..32 is `41 62` followed by eleven `00` |

The two name cases pin the 13-byte `sPairCharacterName` field's exact behavior;
their byte offsets assume the Step-1 split — update them to the derived offsets
if Step 1 lands elsewhere.

Add `TestRingRecordsRoundTrip` covering the same cases.

- [ ] **Step 3: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./model/ -run 'TestRingRecords' -v`
Expected: FAIL — `undefined: RingRecords`.

- [ ] **Step 4: Implement**

Write each record at its exact fixed width, name zero-padded to 13 and truncated
at 13. The gate is `(t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS"`
— copied verbatim from the existing `encodeRings` at
`libs/atlas-packet/character/data.go:767`, matching that file's idiom, so the
v29..v60 shape does not move.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./model/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/model/ring.go libs/atlas-packet/model/ring_test.go docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md
git commit -m "feat(atlas-packet): ring record codec for CharacterData"
```

---

## Task 4: Wire site A — `CharacterData`

### Files

- `libs/atlas-packet/character/data.go` — modify: `Rings` field on the struct (lines 106-121), `encodeRings`/`decodeRings` (lines 765-776)
- `libs/atlas-packet/character/data_test.go` — modify: add ring round-trip test
- `libs/atlas-packet/field/clientbound/set_field_test.go` — read-only; the outer packet that consumes `CharacterData` as an opaque span

Module root: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/character/data_test.go` uses the round-trip idiom (`Encode` then `Decode` then field-by-field equality), not golden bytes — there are no `packet-audit:verify` markers in that file. Follow it.

**Interfaces**

- Consumes from Task 3: `model.RingRecords`, `EncodeRecords`, `DecodeRecords`.
- Produces, for Task 9: `CharacterData.Rings model.RingRecords` (an exported struct field; `CharacterData` uses exported fields throughout, e.g. `Stats`, `Inventory`).

- [ ] **Step 1: Write the failing test**

`TestCharacterDataRingsRoundTrip` in `libs/atlas-packet/character/data_test.go`,
setup copied from the existing `TestCharacterData*RoundTrip` functions in the
same file.

| case | variant | `Rings` | assert |
|---|---|---|---|
| empty is unchanged | `GMS v83` | zero value | encoded output byte-identical to the same `CharacterData` encoded before this task — capture the pre-task bytes as a hex literal in the test and compare |
| empty legacy | `GMS v28` | zero value | same, 2 ring bytes not 6 |
| populated | `GMS v83` | 1 couple, 1 friend | round-trips field-for-field |
| populated JMS | `JMS v185` | 1 couple | round-trips field-for-field |

The "byte-identical to before this task" hex literal is the FR-9 regression
guard for site A. Generate it by running the current encoder **before** editing
`data.go`, and paste the result; do not compute it by hand.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/ -run 'TestCharacterDataRings' -v`
Expected: FAIL — `unknown field Rings`.

- [ ] **Step 3: Implement**

Add `Rings model.RingRecords` to the `CharacterData` struct. Replace
`encodeRings`/`decodeRings` bodies with `m.Rings.EncodeRecords(w, t)` /
`m.Rings.DecodeRecords(r, t)`. The call sites at `data.go:167` (Encode) and
`data.go:239` (Decode) do not move.

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-packet && go test ./character/... ./field/... -v`
Expected: PASS, including the pre-existing `set_field_test.go` opaque-span
assertions (the empty path must not have moved them).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/data.go libs/atlas-packet/character/data_test.go
git commit -m "feat(atlas-packet): drive CharacterData ring records from RingRecords"
```

---

## Task 5: Wire sites B and D — `CharacterSpawn` and `CharacterInfo`

Two encoders in one task because both are single-line replacements in the same
package with the same test file shape, and both are pure constructor-parameter
additions. Four files total, one module.

### Files

- `libs/atlas-packet/character/clientbound/spawn.go` — modify: `NewCharacterSpawn` signature (lines 48-57), ring literals (lines 174-176), add a `Rings()` getter beside the others at lines 208-219
- `libs/atlas-packet/character/clientbound/spawn_test.go` — modify
- `libs/atlas-packet/character/clientbound/info.go` — modify: `NewCharacterInfo` signature (lines 63-65), `WriteBool(false)` (line 131), add a `HasMarriageRing()` getter beside the others at lines 200-210
- `libs/atlas-packet/character/clientbound/info_test.go` — modify

Module root: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/character/clientbound/spawn_test.go:38-83` (`TestCharacterSpawnJMSGolden` — build via constructor, `.Encode(nil, ctx)(nil)`, assert exact length, `hex.DecodeString` + `bytes.Equal` against prefix/tail slices). Marker convention: `spawn_test.go:15-19`, `info_test.go:13-18`.

**Interfaces**

- Consumes from Task 2: `model.RingSet`, `EncodeField`.
- Produces, for Task 7:
  - `NewCharacterSpawn(characterId uint32, level byte, name string, guild GuildEmblem, cts *model.CharacterTemporaryStat, jobId uint16, avatar model.Avatar, pets []SpawnPet, enteringField bool, x int16, y int16, stance byte, fh int16, rings model.RingSet) CharacterSpawn` — `rings` appended last so the existing positional call reads unchanged up to that point.
  - `NewCharacterInfo(characterId uint32, level byte, jobId uint16, fame int16, guildName string, pets []InfoPet, wishList []uint32, medalId uint32, monsterBook MonsterBookInfo, mount MountInfo, hasMarriageRing bool) CharacterInfo` — likewise appended last.

- [ ] **Step 1: Write the failing tests**

In `spawn_test.go`, `TestCharacterSpawnRingBlocks`:

| case | variant | rings | assert |
|---|---|---|---|
| empty is unchanged | `GMS v83` | `model.RingSet{}` | full encoded output byte-identical to pre-task hex literal (capture before editing) |
| empty is unchanged | `GMS v95` | `model.RingSet{}` | same |
| empty is unchanged | `JMS v185` | `model.RingSet{}` | same |
| couple populated | `GMS v83` | `Couple` set to the Task 2 fixture | the 3-byte ring span at the known tail offset is replaced by the 21-byte populated couple block; total length grows by exactly 18 |

The empty cases are the FR-9 guard for site B. Extend the existing
`TestCharacterSpawnJMSGolden` (`spawn_test.go:38-83`) rather than duplicating
its scaffolding — its `enteringField`/tail assertions already pin the
surrounding frame.

In `info_test.go`, `TestCharacterInfoMarriageFlag`:

| case | variant | `hasMarriageRing` | expected byte at the marriage-flag offset | assert |
|---|---|---|---|---|
| false, modern | `GMS v83` | `false` | `0x00` | full output byte-identical to pre-task hex literal |
| true, modern | `GMS v83` | `true` | `0x01` | output differs from the false case in exactly one byte, and the guild-name string follows immediately (no partner block) |
| legacy arm untouched | `GMS v28` | `true` | — | output byte-identical to the pre-task v28 output; no marriage byte appears anywhere |

The `GMS v28` case is the FR-8 guard. The "differs in exactly one byte" assertion
is the OQ-3 guard — design.md §2 OQ-3 established from v83 @0xa2370b, v87
@0xabb181, v95 @0xa05750 and jms @0xb0aa6e that the client reads `sCommunity`
unconditionally next, so a partner block here would desynchronise the stream.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run 'TestCharacterSpawnRingBlocks|TestCharacterInfoMarriageFlag' -v`
Expected: FAIL — too few arguments to `NewCharacterSpawn` / `NewCharacterInfo`.

- [ ] **Step 3: Implement**

`spawn.go`: add `rings model.RingSet` to the struct and constructor; replace the
three `WriteByte(0)` at lines 174-176 with `m.rings.EncodeField(w, t)`. `t` is
already in scope in `Encode`. Mirror in `Decode` with `DecodeField`.

`info.go`: add `hasMarriageRing bool`; replace line 131 with
`w.WriteBool(m.hasMarriageRing)`. Nothing else in that function changes. Mirror
in `Decode`.

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-packet && go test ./character/... -v`
Expected: PASS. Compilation of `services/atlas-channel` will still fail — Task 7
fixes the call sites.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/clientbound/spawn.go libs/atlas-packet/character/clientbound/spawn_test.go libs/atlas-packet/character/clientbound/info.go libs/atlas-packet/character/clientbound/info_test.go
git commit -m "feat(atlas-packet): drive CharacterSpawn ring blocks and CharacterInfo marriage flag"
```

---

## Task 6: Wire site C — `CharacterAppearanceUpdate`, including the frame correction

Its own task because it changes bytes on the empty path — the only site that
does — and therefore invalidates eight pinned evidence records and six
`packet-audit:verify` markers. A reviewer must be able to reject this
independently of Task 5.

### Files

- `libs/atlas-packet/character/clientbound/appearance_update.go` — modify: whole file (55 lines)
- `libs/atlas-packet/character/clientbound/appearance_update_test.go` — modify: six marked tests at lines 33-36, 300, 368 and their bodies
- `docs/packets/ida-exports/gms_v83.json` — read-only; `CUserRemote::OnAvatarModified` @0x98367e is the authority for this frame

Module root: `libs/atlas-packet`.

**What the client actually reads.** From `gms_v83.json`
`CUserRemote::OnAvatarModified` @0x98367e, in order:

| # | op | comment | guard |
|---|---|---|---|
| 0 | `Decode4` | characterId — read by `CUserPool::OnUserRemotePacket` before dispatch | — |
| 1 | `Decode1` | flags byte: `bit0=avatarLook, bit1=speed, bit2=carryItem` | — |
| 2 | `DecodeBuf` | `AvatarLook::Decode` | `v4 & 1` |
| 3 | `Decode1` | `nSpeed` | `v4 & 2` |
| 4 | `Decode1` | `nCarryItemEffect` | `v4 & 4` |
| 5-12 | | the three ring blocks | per flag |

**There is no trailing `Decode4`.** design.md §3.2 characterises the current
encoder as "two bytes short and one int long." The "two bytes short" half is
wrong: the byte the encoder writes as `// mode` at line 34 is the flags byte
with only bit0 set, so the client correctly skips the `nSpeed` and
`nCarryItemEffect` reads. The genuine defect is the single trailing
`w.WriteInt(0) // completed set item id` at line 39, which no export shows the
client reading on any version. Remove that line and nothing else in the frame.

- [ ] **Step 1: Write the failing test**

Replace the bodies of the six marked tests. For each of `gms_v79`, `gms_v83`,
`gms_v84`, `gms_v87`, `gms_v95`, `jms_v185`, two cases:

| case | rings | expected |
|---|---|---|
| empty | `model.RingSet{}` | `characterId(4) + flags(1) + avatar + 00 00 00` — exactly 4 bytes shorter than the pre-task output for the same input, and the removed bytes are the trailing `00 00 00 00` |
| populated | Task 2's couple fixture | `characterId(4) + flags(1) + avatar + 01 <8B ownSN> <8B partnerSN> <4B third> 00 00` |

Assert the empty case's length against the pre-task length minus 4 explicitly,
so the frame correction is visible in the test rather than implied.

Keep the marker comment block above each test and update each `ida=` address to
the one the export gives for that version's `CUserRemote::OnAvatarModified`:
v83 `0x98367e`, v87 `0xa090f4`, v95 `0x954110`, jms `0xa57221`, v84 `0x9c3a1c`,
v79 `0x8d9824` (the existing values at lines 33-36, 300, 368 are already these —
verify each against the export before keeping it).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run 'TestCharacterAppearanceUpdate' -v`
Expected: FAIL — length mismatch on every version (4 bytes long).

- [ ] **Step 3: Implement**

`appearance_update.go`:
- add `rings model.RingSet` to the struct and `NewCharacterAppearanceUpdate`
- add `t := tenant.MustFromContext(ctx)` to both `Encode` and `Decode`, and the
  `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` import — this file
  has no tenant plumbing today and `EncodeField` requires it
- replace the three `WriteByte(0)` at lines 36-38 with `m.rings.EncodeField(w, t)`
- delete `w.WriteInt(0) // completed set item id` (line 39) and its `Decode`
  mirror `_ = r.ReadUint32()` (line 54)
- add a comment above the flags byte recording that `WriteByte(1)` is
  `bit0=avatarLook` only, citing `gms_v83.json` @0x98367e, so it is not
  mistaken for a mode enum again

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-packet && go test ./character/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/clientbound/appearance_update.go libs/atlas-packet/character/clientbound/appearance_update_test.go
git commit -m "fix(atlas-packet): correct CharacterAppearanceUpdate frame and drive its ring blocks"
```

---

## Task 7: Restore `atlas-channel` compilation at the four call sites

Purely mechanical: four constructors gained a parameter. Passing the zero value
everywhere keeps the branch green and byte-identical while Tasks 8-11 build the
data source. Split from Task 5/6 so the packet-layer review is not entangled
with channel plumbing.

### Files

- `services/atlas-channel/atlas.com/channel/socket/writer/character_spawn.go:60` — pass `model.RingSet{}` to `NewCharacterSpawn`
- `services/atlas-channel/atlas.com/channel/socket/writer/character_info.go:54` — pass `false` to `NewCharacterInfo`
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:23` — leave `Rings` at its zero value in the `CharacterData` literal (no edit needed unless the build complains)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:419` — pass `packetmodel.RingSet{}` to `NewCharacterAppearanceUpdate`

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces**

- Consumes from Tasks 5, 6: the four new trailing parameters.
- Produces, for Task 11: the four call sites, now positioned to receive a real `RingSet`.

- [ ] **Step 1: Build to see the failures**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: FAIL — "not enough arguments" at the four sites above.

- [ ] **Step 2: Pass zero values**

`packetmodel` is already the import alias for `libs/atlas-packet/model` in
`character_spawn.go:13`; check the alias in `consumer.go` before using it.

- [ ] **Step 3: Build and test**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/... ./kafka/consumer/asset/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/character_spawn.go services/atlas-channel/atlas.com/channel/socket/writer/character_info.go services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go
git commit -m "chore(atlas-channel): pass empty RingSet at the four ring encoder call sites"
```

---

## Task 8: `atlas-cashshop` read model — `cashId`, `partnerCashId`, `partnerName`

### Files

- `services/atlas-cashshop/atlas.com/cashshop/ring/model.go` — modify: three fields + getters (struct at lines 29-40, getters from line 42)
- `services/atlas-cashshop/atlas.com/cashshop/ring/builder.go` — modify: three setters
- `services/atlas-cashshop/atlas.com/cashshop/ring/rest.go` — modify: `RestModel` (lines 15-26) + `Transform` (line 49)
- `services/atlas-cashshop/atlas.com/cashshop/ring/rest_test.go` — modify
- `services/atlas-cashshop/atlas.com/cashshop/ring/processor.go` — modify: enrich `GetByCharacterId` (line 47)
- `services/atlas-cashshop/atlas.com/cashshop/ring/provider.go` — read-only unless a `byPairId` provider is needed; `byCharacterIdProvider` is at line 13
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/model.go:34` — read-only; `func (m Model) CashId() int64`
- `services/atlas-cashshop/atlas.com/cashshop/ring/entity.go` — read-only; **do not add a stored column** (design.md §5 rejects it)

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

Patterns to copy: `services/atlas-cashshop/atlas.com/cashshop/ring/rest_test.go:10-101` (`TestTransform` — same-package struct literal with unexported fields → `Transform(m)` → field-by-field assertion).

**Why these three fields.** `cash_rings.AssetId` is the cashshop locker asset's
own primary key (`ring.Entity.AssetId` is populated from `astP.Create(...).Id()`
at `cashshop/processor_ring.go:176,192`). That id does not survive
`compartment.Release` — the receiving service mints a new row with a new id. The
identifier that survives is `cashId int64`, which is also the value the wire
needs (design.md §2 OQ-1: the 8-byte buffers are `GW_ItemSlotBase::liSN`). So
`cashId` is both the join key and the payload, and no id translation is needed.

**Interfaces**

- Produces, for Task 9:
  - `ring.Model` getters `CashId() int64`, `PartnerCashId() int64`, `PartnerName() string`
  - `ring.RestModel` JSON fields `cashId`, `partnerCashId`, `partnerName`

- [ ] **Step 1: Write the failing test**

Extend `TestTransform` in `ring/rest_test.go` with the three new fields on the
input `Model{...}` literal and the three new assertions:

| field | input | expected `RestModel` field |
|---|---|---|
| `cashId` | `int64(9007199254740993)` | `CashId == 9007199254740993` |
| `partnerCashId` | `int64(-1)` | `PartnerCashId == -1` |
| `partnerName` | `"PartnerChar"` | `PartnerName == "PartnerChar"` |

The `9007199254740993` value is deliberately above 2^53 — it asserts the field
survives as `int64` and is not silently narrowed by a JSON round-trip.

Add `TestGetByCharacterIdEnrichesCashIdAndPartnerName` in a processor test:

| case | setup | expect |
|---|---|---|
| both halves present | two `cash_rings` rows sharing a `PairId`, each with a locker asset carrying a distinct `cash_id` | each returned `Model` has its own `CashId` and the sibling's `CashId` as `PartnerCashId` |
| sibling row missing | one row only | `PartnerCashId == 0`, no error, the half is still returned |
| character service unavailable | `chaP.GetById` errors | `PartnerName == ""`, no error, the half is still returned |

The last two cases pin fail-soft behavior — the channel's fallback (PRD FR-5) is
downstream of this, and a hard error here would defeat it.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./ring/... -v`
Expected: FAIL — `unknown field CashId`.

- [ ] **Step 3: Implement**

Add the three unexported fields + getters to `ring.Model`, the three builder
setters, and the three `RestModel` JSON fields wired through `Transform`.

**Wiring change, not just fields.** `ring.ProcessorImpl` holds only `l, ctx, db, t`
(`processor.go:29`) — it has no character-service client, so `partnerName`
resolution requires threading that dependency into `NewProcessor`. Likewise
`ring.Model` is constructed only via `Make(e Entity)` in `entity.go`; the three
new fields are computed, not stored, so they are set through `builder.go` after
`Make`, never by widening the entity.

In `processor.go`, enrich `GetByCharacterId`: for each returned half, resolve its
own `cashId` by looking up `inventory/asset` by `AssetId`, resolve the sibling
row by `PairId` for `partnerCashId`, and resolve `partnerName` via
`chaP.GetById(partnerCharacterId)` — the character processor is already a
dependency of `PurchaseRingAndEmit` in this service. Every one of the three
resolutions fails soft to the zero value.

If a `byPairIdProvider` is needed, add it to `provider.go` beside
`byCharacterIdProvider` (line 13), matching its shape.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./... `
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/ring/
git commit -m "feat(atlas-cashshop): expose cashId, partnerCashId and partnerName on the ring read model"
```

---

## Task 9: `atlas-channel/ring` — REST consumer and model

### Files

- `services/atlas-channel/atlas.com/channel/ring/rest.go` — new file; `RestModel` + `Extract`
- `services/atlas-channel/atlas.com/channel/ring/model.go` — new file; `Model` + getters + `Type`/`State`
- `services/atlas-channel/atlas.com/channel/ring/requests.go` — new file; URL builder
- `services/atlas-channel/atlas.com/channel/ring/rest_test.go` — new file
- `services/atlas-channel/atlas.com/channel/door/rest.go` — read-only; `Extract` at line 52 is the shape to copy
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/requests.go` — read-only; `requests.RootUrlFor(ctx, "CASHSHOP")` at line 17 is the root this package needs
- `services/atlas-cashshop/atlas.com/cashshop/ring/rest.go` — read-only; the producing `RestModel` this must mirror

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `services/atlas-channel/atlas.com/channel/door/rest.go:14-52` (`RestModel` + JSON:API interface methods + `Extract`). Note `atlas-cashshop`'s `ring/rest.go` has **only** `Transform`, no `Extract` — the channel side must write its own; there is no symmetric function to reuse.

**Route.** `GET /rings?filter[characterId]=<id>`, registered at
`services/atlas-cashshop/atlas.com/cashshop/ring/resource.go:29`.
`filter[characterId]` is **required** — `handleGetRings` returns 400 without it
(`resource.go:39-...`). The response is paginated.

**Interfaces**

- Consumes from Task 8: the `cashId`, `partnerCashId`, `partnerName` JSON fields.
- Produces, for Task 10:
  - `type Model struct` with getters `Id() uuid.UUID`, `PairId() uuid.UUID`, `CharacterId() uint32`, `PartnerCharacterId() uint32`, `CashId() int64`, `PartnerCashId() int64`, `PartnerName() string`, `ItemTemplateId() uint32`, `Type() Type`, `State() State`
  - `const TypeCouple = Type("COUPLE")`, `TypeFriendship = Type("FRIENDSHIP")`, `StateActive = State("ACTIVE")`, `StateBroken = State("BROKEN")`, `StateExpired = State("EXPIRED")` — re-declared channel-side, matching `services/atlas-cashshop/atlas.com/cashshop/ring/model.go:14-27`. Do **not** promote these to `libs/atlas-constants`: `libs/atlas-constants/item/constants.go:24` already has `ClassificationRing = Classification(111)`, which is an item classification, not a pairing type; the cashshop package documents that distinction at `ring/model.go:9-11` and this package inherits it.
  - `func requestByCharacterId(ctx context.Context, characterId uint32) requests.Request[[]RestModel]`

- [ ] **Step 1: Write the failing test**

`TestExtract` in `services/atlas-channel/atlas.com/channel/ring/rest_test.go`,
shape copied from `door`'s extraction pattern.

| case | `RestModel` input | expect |
|---|---|---|
| full couple half | id/pairId valid UUID strings, `ringType: "COUPLE"`, `state: "ACTIVE"`, `cashId: 9007199254740993`, `partnerCashId: 42`, `partnerName: "PartnerChar"` | every getter returns the input value; `Type() == TypeCouple`; `State() == StateActive` |
| invalid uuid | `id: "not-a-uuid"` | returns an error, not a zero Model with a swallowed parse failure |
| unknown ringType | `ringType: "MYSTERY"` | returns an error — an unrecognised pairing type must not silently become a couple ring |
| large cashId survives | `cashId: 9007199254740993` | `CashId() == 9007199254740993` |

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./ring/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement**

Three files. `requests.go` mirrors
`cashshop/inventory/asset/requests.go`: `requests.RootUrlFor(ctx, "CASHSHOP")`
plus `const Resource = "rings"` and a `filter[characterId]=%d` query param.
Because the endpoint is paginated, consume it with `requests.DrainProvider`
and a bare URL, exactly as `door/requests.go:20-32` documents for its own
paginated list.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./ring/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/ring/
git commit -m "feat(atlas-channel): read-only ring REST consumer"
```

---

## Task 10: `atlas-channel/ring` — cache and `GetRingSet`

### Files

- `services/atlas-channel/atlas.com/channel/ring/cache.go` — new file
- `services/atlas-channel/atlas.com/channel/ring/processor.go` — new file
- `services/atlas-channel/atlas.com/channel/ring/cache_test.go` — new file
- `services/atlas-channel/atlas.com/channel/ring/processor_test.go` — new file
- `services/atlas-channel/atlas.com/channel/monster/information/cache.go` — read-only; the cache precedent
- `services/atlas-channel/atlas.com/channel/listener/evict.go` — read-only; `RegisterEvictor` at line 24
- `services/atlas-channel/atlas.com/channel/equipment/model.go` — read-only; `Slots() map[slot2.Type]slot.Model`
- `services/atlas-channel/atlas.com/channel/equipment/slot/model.go` — read-only; `slot.Model{Position, Equipable *asset.Model, CashEquipable *asset.Model}`
- `services/atlas-channel/atlas.com/channel/asset/model.go:110` — read-only; `func (m Model) CashId() int64`
- `services/atlas-channel/atlas.com/channel/character/model.go:244` — read-only; `func (m Model) Equipment() equipment.Model`

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `services/atlas-channel/atlas.com/channel/monster/information/cache.go:93-149` — `perTenant map[uuid.UUID]map[uint32]cacheEntry` guarded by `sync.RWMutex`, with `lookup(tid, key, now)`, `put(tid, key, e)`, and a package-level `EvictTenant(tid)` registered via `listener.RegisterEvictor`. Swap the `uint32` key from monsterId to characterId. Note `door/` has **no** cache — it is the REST-plumbing precedent only (Task 9), not the cache precedent.

**Equipped-asset shape — corrects design.md §4.1.** The design's
`GetRingSet(characterId, equipped []asset.Model)` signature does not match the
code. `character.Model.Equipment()` returns `equipment.Model`, whose `Slots()`
gives `map[slot.Type]slot.Model`, and each `slot.Model` carries
`Equipable *asset.Model` **and** `CashEquipable *asset.Model`. A cash-shop ring
is a cash equip, so the join reads `CashEquipable`. Take `equipment.Model`.

**Ring slot positions** (`libs/atlas-constants/inventory/slot/constants.go`):
`ring1` = -12, `ring2` = -13, `ring3` = -15, `ring4` = -16. The `petRing*` /
`pet2Ring*` / `pet3Ring*` entries at -21, -29, -31, -37, -39, -45 are pet
equipment and are **not** couple/friendship rings — exclude them.

**Interfaces**

- Consumes from Task 9: `ring.Model` and its getters.
- Consumes from Task 2: `packetmodel.RingSet`, `packetmodel.PairRing`, `packetmodel.MarriageRing`.
- Produces, for Tasks 11, 12:
  - `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`
  - `Processor` interface with `GetRingSet(characterId uint32, eq equipment.Model) packetmodel.RingSet` and `Invalidate(characterId uint32)`
  - `func EvictTenant(tid uuid.UUID)`

- [ ] **Step 1: Write the failing cache test**

`TestRingCacheTenantIsolation` in `cache_test.go`, setup copied from whatever
`monster/information` uses for its own cache test.

| case | setup | expect |
|---|---|---|
| tenant isolation | `put(tenantA, 100, e1)`, `put(tenantB, 100, e2)` | `lookup(tenantA, 100)` returns `e1`, `lookup(tenantB, 100)` returns `e2` — this is the PRD §8 multi-tenancy assertion |
| miss on unknown character | `put(tenantA, 100, e1)` | `lookup(tenantA, 101)` returns `false` |
| evict drops one tenant only | both tenants populated, `EvictTenant(tenantA)` | `lookup(tenantA, 100)` false, `lookup(tenantB, 100)` true |
| invalidate drops one character only | `put(tenantA, 100, e1)`, `put(tenantA, 200, e2)`, invalidate 100 | 100 misses, 200 hits |

- [ ] **Step 2: Write the failing `GetRingSet` test**

`TestGetRingSet` in `processor_test.go`. Override the upstream fetch with a test
double, following the `upstreamFn` test-override precedent at
`services/atlas-channel/atlas.com/channel/monster/information/cache.go:151`.

Fixture halves (all for character 100, partner 200):

```
coupleActive     = Model{cashId: 1111, partnerCashId: 2222, ringType: COUPLE,     state: ACTIVE, itemTemplateId: 1112001, partnerName: "Partner"}
friendshipActive = Model{cashId: 3333, partnerCashId: 4444, ringType: FRIENDSHIP, state: ACTIVE, itemTemplateId: 1112800, partnerName: "Partner"}
coupleBroken     = Model{cashId: 5555, partnerCashId: 6666, ringType: COUPLE,     state: BROKEN, itemTemplateId: 1112001, partnerName: "Partner"}
```

| case | halves returned | equipment | expect |
|---|---|---|---|
| no halves | none | empty | `RingSet{}` — `Couple`, `Friendship`, `Marriage` all nil |
| couple equipped | `coupleActive` | `ring1` `CashEquipable` with `CashId() == 1111` | `Couple != nil`, `OwnSN == 1111`, `PartnerSN == 2222` |
| friendship equipped | `friendshipActive` | `ring1` `CashEquipable` with `CashId() == 3333` | `Friendship != nil`, `Couple == nil` |
| owned but not equipped (FR-14) | `coupleActive` | empty | `RingSet{}` |
| BROKEN discarded (FR-3) | `coupleBroken` | `ring1` `CashEquipable` with `CashId() == 5555` | `RingSet{}` |
| equipped in a non-cash slot | `coupleActive` | `ring1` `Equipable` (not `CashEquipable`) with `CashId() == 1111` | `RingSet{}` — a cash ring never occupies the non-cash sub-slot |
| pet ring slot ignored | `coupleActive` | `petRing1` (-21) `CashEquipable` with `CashId() == 1111` | `RingSet{}` |
| two couple halves, lowest slot wins (FR-15) | two ACTIVE couple halves, cashIds 1111 and 7777 | 7777 in `ring1` (-12), 1111 in `ring2` (-13) | `Couple.OwnSN == 7777` — most negative position first |
| slot tie broken by cashId | two ACTIVE couple halves in the same slot type (defensive) | — | lower `cashId` wins |
| upstream error (FR-5) | fetch returns an error | `ring1` populated | `RingSet{}` returned, no panic, one warn-level log containing the character id |

The selection rule is: lowest equipped slot position (most negative first), ties
broken by lowest `cashId`. State it in `GetRingSet`'s doc comment so it is not
re-litigated.

`Marriage` is always nil in practice — PRD §2 lists marriage-ring acquisition as
a non-goal, and `cash_rings.RingType` only admits `COUPLE` and `FRIENDSHIP`
(`services/atlas-cashshop/atlas.com/cashshop/ring/model.go:15-18`). The field
exists because the wire block does; a test asserts it stays nil.

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./ring/... -v`
Expected: FAIL — `undefined: NewProcessor`.

- [ ] **Step 4: Implement**

`cache.go` follows `monster/information/cache.go:93-149` exactly. Register
`EvictTenant` with `listener.RegisterEvictor` from this package's `init()`, per
`listener/evict.go:22-23` ("Safe to call from `init()` of any package that holds
tenant-scoped state").

`GetRingSet` is a pure function over cached halves plus `equipment.Model`. It
never issues a REST call — PRD §8 requires population on character load, not on
encode. On a cache miss it returns `RingSet{}` and logs at debug; it does not
block the encode to fetch.

- [ ] **Step 5: Run tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./ring/... -v`
Expected: PASS, all cases.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/ring/
git commit -m "feat(atlas-channel): per-tenant ring cache and GetRingSet"
```

---

## Task 11: Feed the four encoder sites from the ring processor

### Files

- `services/atlas-channel/atlas.com/channel/socket/writer/character_spawn.go` — modify: line 60, replace the Task-7 zero value
- `services/atlas-channel/atlas.com/channel/socket/writer/character_info.go` — modify: line 54
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go` — modify: line 23, populate `Rings`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go` — modify: line 419, inside `updateAppearance`

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces**

- Consumes from Task 10: `ring.NewProcessor(...).GetRingSet(characterId, eq)`.

**Broadcast-loop caution.** `NewCharacterAppearanceUpdate` at `consumer.go:419`
sits inside `updateAppearance`, a `model.Operator[session.Model]` closure invoked
per recipient session via `_map.NewProcessor(...).ForSessionsInMap(...)`
(`consumer.go:381`). Resolve the `RingSet` **once** for the broadcasting
character, outside the per-session closure. Resolving it per recipient would
multiply the work by the number of observers on the map — exactly the hot-path
cost PRD §8 forbids.

- [ ] **Step 1: Write the failing test**

`TestCharacterSpawnBodyCarriesRings` in a writer test. Setup: a `character.Model`
whose `Equipment()` has a `ring1` `CashEquipable` matching a cached ACTIVE couple
half.

| case | cache state | expect |
|---|---|---|
| ring present | ACTIVE couple half cached, matching equip | encoded body contains the populated 21-byte couple block; length exceeds the empty-ring body by 18 |
| no ring | cache empty | encoded body byte-identical to the Task-7 output |

Add `TestCharacterInfoBodySetsMarriageFlag`: with no marriage half cached (the
only reachable state, see Task 10), the flag byte is `0x00` and the body is
byte-identical to Task 7's output.

Add `TestUpdateAppearanceResolvesRingsOnce`: a counting test double on the ring
processor; broadcast to three sessions; assert `GetRingSet` was called exactly
once. This is the PRD §8 hot-path guard and it fails loudly if a later refactor
moves the call inside the closure.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/... ./kafka/consumer/asset/... -v`
Expected: FAIL — empty ring block where a populated one is expected; call count 3, want 1.

- [ ] **Step 3: Implement**

Thread `ring.Processor` into each of the four sites. For `character_data.go`,
populate `Rings model.RingRecords` from the same cached halves — this is the
site that carries `partnerName`, so it consumes `ring.Model.PartnerName()`
(Task 8).

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... `
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/ services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go
git commit -m "feat(atlas-channel): feed ring data into the four encoder sites"
```

---

## Task 12: Cache population and `RING_PURCHASED` invalidation

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — modify: `handleStatusEventRingPurchased` at line 496 (design.md §4.3 cites line 461; the handler has since moved — the registration is at line 120)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go` — modify
- `services/atlas-channel/atlas.com/channel/ring/processor.go` — new file in Task 10; extended here with the population entry point

Module root: `services/atlas-channel/atlas.com/channel`.

**The event alone is insufficient.** `cashshop.RingPurchasedBody`
(`services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go:394-402`)
carries `TransactionId, CompartmentId, AssetId, PartnerName, TemplateId,
Quantity, RingType, PairId`. It carries **no `cashId`** for either half and no
partner character id. Invalidation must therefore drop the entry and let the
next population re-fetch; it cannot patch the cache in place from the payload.

The existing handler only announces to the buyer's own session via
`session.NewProcessor(...).IfPresentByCharacterId(sc.Channel())(e.CharacterId, ...)`.
Partner-side handling is new work.

- [ ] **Step 1: Write the failing test**

`TestRingPurchasedInvalidatesCache` in `consumer_test.go`, setup copied from the
existing handler tests in that file.

| case | setup | expect |
|---|---|---|
| buyer invalidated | buyer 100 cached, `RING_PURCHASED` for 100 | buyer's entry gone |
| partner invalidated when present | buyer 100 and partner 200 both cached, partner has a live session on this channel | both entries gone |
| partner absent is not an error | buyer 100 cached, partner has no session here | buyer's entry gone, handler returns no error |
| wrong tenant untouched | tenant A buyer 100 cached, event on tenant B | tenant A's entry survives |

Add `TestRingCachePopulatedOnCharacterLoad`: on the character-load path, the
processor issues exactly one `GET /rings` per character and the result is
cached; a second load in the same presence does not re-fetch.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/cashshop/... ./ring/... -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

Invalidation in `handleStatusEventRingPurchased`; population on the character-load
path (`BuildCharacterData` in `socket/writer/character_data.go` runs on
login/channel-enter; `CharacterSpawnBody` on map-enter). Map/channel transfer
drops the entry.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./... `
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/ services/atlas-channel/atlas.com/channel/ring/
git commit -m "feat(atlas-channel): populate and invalidate the ring cache"
```

---

## Task 13: Packet coverage — `CharacterSpawn` and `CharacterInfo`

Follows `docs/packets/audits/VERIFYING_A_PACKET.md` end to end.

### Files

- `libs/atlas-packet/character/clientbound/spawn_test.go` — modify: marker block at lines 15-19
- `libs/atlas-packet/character/clientbound/info_test.go` — modify: marker block at lines 13-18
- `docs/packets/evidence/gms_v48/character.clientbound.CharacterSpawn.yaml` — modify (and the v61/v72/v79/v83/v84/v87/v95/jms_v185 siblings)
- `docs/packets/evidence/gms_v48/character.clientbound.CharacterInfo.yaml` — modify (and its eight siblings)
- `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` — regenerated, not hand-edited

Module root: repo root for `go run ./tools/packet-audit`; `libs/atlas-packet` for tests.

**Current matrix state** (`STATUS.md:197` and `:86`): `SPAWN_PLAYER` and
`CHAR_INFO` are ✅ on nine columns and ❌ on `gms_v92`. Both codecs are edited by
this task, so v92 is claimed and promoted — a codec this task edits is a codec
this task verifies.

- [ ] **Step 1: Add a fixture per column covering both ring states**

Each of the ten columns gets a byte fixture asserting the empty `RingSet`
(byte-identical to pre-task) and a populated one. Marker format, one line per
column, directly above the test function:

```
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=<key> ida=<0xaddr>
```

The existing markers at `spawn_test.go:15-19` already cover v83/v87/v95/v84/jms;
add v48, v61, v72, v79, v92 with addresses read from that version's export, not
from memory.

- [ ] **Step 2: Pin evidence**

```bash
go run ./tools/packet-audit evidence pin \
  --packet character/clientbound/CharacterSpawn --version gms_v92 \
  --ida "CUserPool::OnUserEnterField" --category TIER1-FIXTURE
```

`--ida` takes the function name exactly as it keys the export's `functions` map,
not a hex address. After each command succeeds, open the written YAML and add
the `verifies:` field by hand:

```yaml
verifies:
  - libs/atlas-packet/character/clientbound/spawn_test.go#TestCharacterSpawnRingBlocks
```

- [ ] **Step 3: Regenerate and check the matrix**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: exit 0. `matrix --check` is a hard blocking CI gate
(`.github/workflows/packet-matrix.yml`) — any 🟥 conflict, orphan, dangling or
stale finding fails it. Every cell for both ops must read ✅.

- [ ] **Step 4: Commit test, evidence and matrix together**

```bash
git add libs/atlas-packet/character/clientbound/spawn_test.go libs/atlas-packet/character/clientbound/info_test.go docs/packets/evidence/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(atlas-packet): verify CharacterSpawn and CharacterInfo ring coverage"
```

---

## Task 14: Packet coverage — `CharacterAppearanceUpdate` and `CharacterData`

Separate from Task 13 because `UPDATE_CHAR_LOOK`'s bytes **changed** (Task 6), so
its eight existing evidence records are re-pinned rather than extended — a
different and riskier operation than adding a column.

### Files

- `libs/atlas-packet/character/clientbound/appearance_update_test.go` — modify: markers at lines 33-36, 300, 368
- `docs/packets/evidence/gms_v61/character.clientbound.CharacterAppearanceUpdate.yaml` — modify (and the v72/v79/v83/v84/v87/v95/jms_v185 siblings — eight records)
- `libs/atlas-packet/field/clientbound/set_field_test.go` — modify: the `CharacterData` opaque span
- `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` — regenerated

Module root: repo root for `packet-audit`; `libs/atlas-packet` for tests.

**Column scope — corrects design.md §7.** The design claims "all 10 columns" for
`CharacterAppearanceUpdate` and for `CharacterData` via `SetField`. The matrix
disagrees: `STATUS.md:264` shows `UPDATE_CHAR_LOOK` as `⬜` (n-a) on `gms_v48`
with no opcode, and `STATUS.md:180` shows `SET_FIELD` likewise `⬜` on `gms_v48`.
Both are **nine** columns, v61 through jms_v185. Do not attempt to verify a v48
cell for either — confirm the `n-a` disposition instead, per
`VERIFYING_A_PACKET.md` §1.

**`CharacterData` is an opaque span.** `set_field_test.go:33-34` already records
the OPAQUE_LEDGER caveat: the `CharacterData` middle cannot cite per-field
decompile lines and is derived from the Atlas encoder and asserted as an opaque
span. The ring records live inside that span. Follow the existing
VERIFIED-EXCEPTION discipline in that file and say so in the test comment;
do not invent per-field citations.

- [ ] **Step 1: Re-derive each `UPDATE_CHAR_LOOK` fixture against the corrected frame**

Every one of the eight columns' expected bytes shrinks by four (the removed
trailing int). Recompute each from the version's export entry for
`CUserRemote::OnAvatarModified`; do not adjust the old literals by subtracting
bytes, because that would propagate any error already in them.

- [ ] **Step 2: Re-pin the eight evidence records**

```bash
go run ./tools/packet-audit evidence pin \
  --packet character/clientbound/CharacterAppearanceUpdate --version <key> \
  --ida "CUserRemote::OnAvatarModified" --category TIER1-FIXTURE
```

for each of `gms_v61 gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v95 jms_v185`,
re-adding `verifies:` by hand each time.

- [ ] **Step 3: Extend the `SetField` fixture**

Add a populated-`Rings` case to `set_field_test.go` beside the existing empty
one, so the opaque span is exercised in both states.

- [ ] **Step 4: Regenerate and check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: exit 0, and no cell that was ✅ before this task is anything but ✅ now.
A degraded cell means the re-pin lost an artifact — stop and fix it rather than
committing.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/clientbound/appearance_update_test.go libs/atlas-packet/field/clientbound/set_field_test.go docs/packets/evidence/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(atlas-packet): re-verify CharacterAppearanceUpdate frame and CharacterData ring coverage"
```

---

## Task 15: Coverage manifest, service docs, and the full gate

### Files

- `docs/tasks/task-269-ring-pair-behavior/coverage-manifest.yaml` — new file
- `services/atlas-channel/docs/` — modify: record the FR-12 residual limitation (locate the right file with `Glob`; do not assume a path)
- `tools/verify.sh` — read-only

Module root: repo root.

Patterns to copy: `docs/tasks/task-252-jukebox-cash-item/coverage-manifest.yaml` — the `ops:` / `versions:` / `fields:` / `out_of_scope:` schema with a prose comment per entry. Note no `coverage-manifest.yaml` exists under `docs/packets/`; it is a per-task artifact that lives in the task folder.

- [ ] **Step 1: Write `coverage-manifest.yaml`**

```yaml
ops:
  - character/clientbound/CharacterSpawn
  - character/clientbound/CharacterInfo
  - character/clientbound/CharacterAppearanceUpdate
  - character/CharacterData
```

`versions:` lists all ten for `CharacterSpawn` and `CharacterInfo`, and nine
(v61..jms_v185, v48 as `n-a`) for `CharacterAppearanceUpdate` and
`CharacterData` — with the `⬜` disposition and its `STATUS.md` line cited in a
comment, per Task 14.

`fields:` states, in one entry per block, the exact wire layout Task 1 pinned.

`out_of_scope:` lists what a reader would otherwise expect to see changed:
`field/clientbound/FieldSetField` itself (only its opaque `CharacterData` span
moves; its own frame is untouched), and the wedding/spouse-chat ops PRD §2
declares non-goals.

- [ ] **Step 2: Record the FR-12 residual limitation in the service docs**

Site C is now live, so ring state updates within a map on equip/unequip. The
residual case — a ring purchased by a partner already standing on your map —
resolves when that partner's own client emits its appearance update on
`RING_PURCHASED`. Document that residual case, not the blanket "requires a map
change" the PRD originally assumed.

- [ ] **Step 3: Run the full gate**

Run: `tools/verify.sh`
Expected: exit 0. Flagless — `--quick` and `--no-docker` skip the bake and
`-race` and do not count (CLAUDE.md, "Done means verified").

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-269-ring-pair-behavior/coverage-manifest.yaml services/atlas-channel/docs/
git commit -m "docs(task-269): coverage manifest and the ring-update limitation"
```

- [ ] **Step 5: Pre-PR review**

Per CLAUDE.md, code review is a separate gate from `verify.sh`. Dispatch, each
with an explicit `model`:

- `backend-guidelines-reviewer` over `services/atlas-channel` and `services/atlas-cashshop`
- `packet-completeness-critic` for this task folder — it diffs
  `coverage-manifest.yaml` against the branch's git and matrix delta and reports
  CHANGED-BUT-UNCLAIMED and CLAIMED-BUT-UNVERIFIED
- `plan-adherence-reviewer` against this plan

This change crosses a service boundary (`atlas-cashshop` → `atlas-channel` via
`GET /rings`), so trace the read model into its consumer by hand and confirm a
test asserts the new contract — a green `verify.sh` cannot see that seam.
