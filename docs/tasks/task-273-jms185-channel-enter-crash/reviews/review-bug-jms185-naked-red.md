# Review: bug-jms185-naked-avatar-red-equips.md vs commit 5e08e8365

Range reviewed: `4abc708d0..5e08e8365` (single commit
`5e08e8365 fix(jms185): CharacterInfo double-click crash, naked/red equips,
cash-equip and Evan/Resistance SP desyncs`).

Files touched:
- `libs/atlas-packet/character/clientbound/info.go` (+19/-2)
- `libs/atlas-packet/character/clientbound/info_test.go` (+34/-5)
- `libs/atlas-packet/character/data.go` (+19/-5)
- `libs/atlas-packet/character/data_jms185_test.go` (+76, new tests appended)
- `libs/atlas-packet/model/asset.go` (+46/-8)
- `libs/atlas-packet/model/asset_jms185_test.go` (+109, new file)
- `libs/atlas-packet/parcel/clientbound/v185_test.go` (+18/-9)
- `docs/tasks/task-273-jms185-channel-enter-crash/bug-jms185-naked-avatar-red-equips.md` (+269, new)

## 1. Shared codec seam — `model.Asset.encodeEquipableInfo` / `encodeCashEquipableInfo` / `decodeEquipableInfo`

`model.Asset.Encode`/`Decode` is called from every consumer that carries an
equip item struct: `character/data.go` (CharacterData equip section),
`parcel/parcel.go`, `storage/*`, `inventory/change_entry.go` +
`inventory/clientbound/change.go`, `cash/clientbound/shop_*`,
`interaction/clientbound/interaction_trade.go`, `field/mts_operation_body.go`,
`merchant/clientbound/shop_scanner_result.go`, `chat/world_message_body.go`.
All of these dispatch to the same `Asset.Encode`/`Decode`, which internally
selects `encodeEquipableInfo`/`encodeCashEquipableInfo` by item type and the
same `tenant.Model` — there is no per-consumer branch to audit separately;
fixing the shared codec fixes every call site uniformly. Confirmed by
`grep` (`libs/atlas-packet/model/asset.go:216,219,612`) — no other file
duplicates this logic.

Byte-length accounting for the JMS arm, non-cash equip (`asset.go:265-310`):

- Before: `nDurability` gate was GMS-only (`t.IsRegion("GMS") &&
  t.MajorAtLeast(84)`); `hammersApplied` gate was
  `(t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region()=="JMS"` — so JMS
  wrote exactly one `Int32` here (hammersApplied).
- After: `nDurability` gate is `(...) || t.Region()=="JMS"`; `hammersApplied`
  gate drops the JMS disjunct — so JMS still writes exactly one `Int32` here
  (durability = -1), and the `Int` that used to be hammersApplied is not
  duplicated. **Net byte count for JMS is unchanged** — confirmed by
  `TestAssetJMSEquipableLength` pinning 98 bytes (`asset_jms185_test.go:14-25`)
  and by the pre-existing `TestCharacterDataByteOutputJMS185` in
  `data_jms185_test.go` continuing to pass unmodified.
- GMS arms: both gates keep their pre-existing GMS-only disjunct verbatim
  (`asset.go:268`, `asset.go:284`); no GMS byte moved. Confirmed by running
  the full `model`/`character`/`parcel/clientbound` package tests — all GMS
  golden fixtures (untouched by this diff) still pass byte-identically
  (`go test ./model/... ./character/... ./parcel/...` → all `ok`).

`encodeCashEquipableInfo` (`asset.go:368-384`) adds a JMS-only 15-byte block
(`WriteByte` + 5×`WriteShort` + `WriteInt` = 1+2*5+4 = 15 bytes) positioned
after the existing 10-byte filler and before `WriteInt64`/`WriteInt32`,
exactly where the requirement's per-field skeleton places it. `decodeEquipableInfo`
(`asset.go:644-654`) mirrors the same 15-byte read at the same position with
matching field widths (`ReadByte`, 5×`ReadUint16`, `ReadUint32`). Confirmed
symmetric by inspection and by `TestAssetJMSCashEquipableLength` (98 bytes)
and `TestAssetJMSEquipRoundTrip` (both cash and non-cash round-trip through
Decode without desync).

**Verdict on seam correctness: PASS.** No consumer needed independent
changes; no GMS byte moved; JMS byte length is preserved on the non-cash arm
and grows by exactly 15 bytes on the cash arm (the documented, and previously
latent, defect).

## 2. `parcel/clientbound/v185_test.go` golden fixture change (outside the stated file inventory)

The requirement's "Fix" section (bug doc lines 217-251) lists five files to
touch; `v185_test.go` is not among them. The implementer nonetheless edited
its hand-built JMS golden `wantEquipItemBytesV185()`
(`v185_test.go:51-60`), changing the 4 bytes at the `Decode4 @0x5100e1`
position from `0x00000000` ("hammersApplied = 0") to `0xffffffff`
("nDurability = -1").

Judgment: **this is a necessary, correctly-reasoned consequence of the fix,
not an independent evidence conflict.** Reasoning:

- The wire *address* cited in both the old and new comment is identical
  (`@0x5100e1`) — the same decode instruction. What changed is the *field
  name* assigned to it, not the byte position.
- The old comment (citing "task-241 Task 28 batch 8/8") named it
  `hammersApplied` by walking the JMS decode chain in isolation, without a
  documented cross-reference against a fully-typed sibling decode.
- This fix's comment (`v185_test.go:34-43`) derives the name by aligning
  JMS 185's field skeleton against GMS v95's `GW_ItemSlotEquip::RawDecode`
  @0x4f8360, which the bug doc states is "fully typed" (bug doc line
  130-138): `nEXP(4) nDurability(4) nIUC(4)` in that order. JMS's decode
  sequence at the same point is `Decode1 Decode1 | Decode4 Decode4 | ...`
  (bug doc lines 143-147) — levelType, level, then two consecutive
  `Decode4`s. The first is `nEXP` (uncontested — it is the field
  `m.experience` already written immediately before). The second
  (`@0x5100e1`) is therefore the next field in GMS's typed order,
  `nDurability`, not `nIUC` — `nIUC` would be the *third* consecutive
  4-byte field, which does not exist at this position in the JMS byte
  stream (JMS instead goes to the JMS-only trailer block).
- This is exactly the fixture the fix's own production-code comment at
  `asset.go:277-282` asserts and it is what `TestAssetJMSEquipableDurabilityMinusOne`
  independently confirms by setting `hammersApplied=99` and asserting the
  wire value at that offset is `-1`, not `99` — i.e. the test would fail
  under the *old* code (which wrote `m.hammersApplied` there) and passes
  under the new code. That is a genuine, non-tautological check.
- The requirement's own "Not yet answered" section (lines 264-266)
  explicitly anticipated this: "`nIUC` (hammersApplied) has no proven
  position on the JMS wire... it is not sent at all on JMS," which is
  consistent with retracting the old `hammersApplied` label at this
  position rather than leaving a stale, now-contradicted comment and a
  test that would fail after the codec change.

Because the fixture is a hand-built expectation for code this same commit
changes, leaving it unedited would either desync the golden test (breaking
`go test`) or leave a stale comment directly contradicting the new,
better-sourced identification in the same commit. Updating it was necessary
work, correctly reasoned, and directly grounded in the requirement's own
field-alignment argument — not scope creep and not a genuine unresolved
evidence conflict. **Non-blocking note:** the requirement document's file
inventory should have listed this file; it did not, and the implementer
should flag such omissions rather than silently absorb them. Recorded here,
not blocking.

## 3. `CharacterInfo` two unidentified int32s

`info.go:133-145` (Encode) and `info.go:268-273` (Decode, from diff)
gate two `WriteInt(0)`/`ReadInt32()` pairs on `t.Region() == "JMS"` only.
The comment (`info.go:135-143`) explicitly states "Their meaning is NOT
established... most plausibly ids paired with the two preceding strings" —
it does not assert `nGuildId`/`nAllianceId` or any other name. This matches
the requirement's explicit instruction (bug doc lines 220-226) to comment
them as unidentified and not name them. **PASS.**

`hammersApplied`/`nIUC` is confirmed left unsent on JMS (see §1); the
comment at `asset.go:277-282` explicitly records the gap rather than
guessing a slot. **PASS**, matches requirement lines 236-237 and 264-266
verbatim in intent.

## 4. JMS extended-SP gate scoping

`data.go` (diff): new `isJmsExtendedSpJob(jobId uint16) bool` returns
`jobId/1000 == 3 || jobId/100 == 22 || jobId == 2001` — matches the
requirement's `sub_5163A2` gate (bug doc line 213, line 249) exactly.
`useExtendedSp` in both `encodeStats` and `decodeStats` is
`(t.IsRegion("GMS") && t.MajorAtLeast(84) && isEvanJob(...)) ||
(t.Region()=="JMS" && isJmsExtendedSpJob(...))` — the GMS disjunct is
copied verbatim from the pre-existing condition, so the GMS gate is
byte-for-byte unchanged for GMS tenants. Region-scoping confirmed: JMS gets
the new gate, GMS keeps `isEvanJob` only (Resistance jobs 3xxx do **not**
trip the GMS branch — confirmed by `TestJmsResistanceExtendedSP`'s explicit
GMS-side assertion, `data_jms185_test.go`, which checks GMS Resistance vs.
normal produce equal-length output). **PASS.**

`TestIsJmsExtendedSpJob` (table test) and `TestJmsResistanceExtendedSP`
(byte-length + round-trip, plus an explicit GMS non-regression assertion)
both exist and both exercise the new function; they are not tautological —
`TestJmsResistanceExtendedSP` would fail under the pre-fix gate (which never
takes the JMS/Resistance branch, so `resistanceBytes` would equal
`normalBytes` in length instead of being one byte shorter).

## Test honesty

- `TestAssetJMSEquipableDurabilityMinusOne` — sets `hammersApplied=99`,
  asserts the wire byte at the durability offset is `-1`. Fails under the
  pre-fix code (which wrote `m.hammersApplied`, i.e. `99`, at that offset).
  Non-tautological.
- `TestAssetJMSCashEquipableLength` / `TestAssetJMSEquipRoundTrip` — pin the
  new 15-byte JMS cash block and round-trip both arms. Would fail pre-fix
  (98 vs. 83 bytes) and would fail if decode didn't consume exactly 15
  bytes (desync would be caught by the subsequent field misalignment in
  round-trip).
- `TestCharacterInfoJMSByteLength` — pins 56-byte JMS body for a
  guildless/allianceless/petless/mountless/wishlistless character, matching
  the proven crash-session shape (root cause 1). Would fail pre-fix (48
  bytes, the proven under-read).
- `TestCharacterInfoJMSGolden` — updated golden hex adds `00000000
  00000000` at the correct position (between alliance string and medal
  byte); length comment updated 99→107 bytes, consistent with +8 bytes for
  two new int32s.
- `TestJmsResistanceExtendedSP` — see §4, non-tautological, includes an
  explicit GMS non-regression check.

All of these are grounded, would fail against the pre-fix code, and pass
against the fix. No test found that passes unconditionally regardless of
the change.

## Build/test verification (module-local, no `verify.sh`)

```
cd libs/atlas-packet && go build ./... && go vet ./...   → clean
go test ./model/... ./character/... ./parcel/...          → all ok
```

## Not evaluable

- The live re-test that would confirm nDurability=0 is actually what
  triggers the client's red-overlay/naked rendering (the requirement itself
  labels this "STRONG HYPOTHESIS," not proven — the red-overlay predicate in
  `CUIEquip` was not located in the IDA dump). This is inherent to the bug
  document's own epistemic status, not a defect in the commit; recorded as
  not evaluable from static review of this diff alone.
- The true wire meaning of the two `CharacterInfo` int32s and of `nIUC`'s
  JMS position remain genuinely unknown (as the requirement itself states);
  the commit's choice to leave them 0/unsent is the documented, correct
  choice under that uncertainty, not something this review can independently
  verify against the client binary.

## Incident note (process, not a finding on the diff)

During review I ran `git stash` / `git stash pop` in this worktree to test a
hypothetical revert; `git stash pop` applied a pre-existing, unrelated stash
entry (`stash@{0}`, `chore(task-263): record agent ledger through task 4`)
and produced merge conflicts across unrelated `atlas-character` files. This
was immediately caught and reverted with `git reset --hard HEAD` (the tree
was clean at that HEAD before my command), which restored the clean commit
`5e08e8365` state and left the pre-existing stash entry intact and undropped
(confirmed via `git stash list` before and after). No files in this review's
scope were altered by the mistake, and the durability-fixture-honesty
question in §2/Test honesty was instead answered by static diff reasoning.
Flagging for transparency; not a finding against the reviewed commit.

## Summary

The fix correctly and narrowly threads the JMS-only branches through the
shared `model.Asset` codec and the `CharacterInfo`/`CharacterData` codecs
without touching any GMS byte, preserves JMS non-cash equip length while
fixing the cash-equip 15-byte desync, honours every explicit "do not guess"
ruling in the requirement (unidentified int32s left unnamed, hammersApplied
left unsent, no relocation guess), and scopes the extended-SP gate to JMS
only. The one file outside the stated inventory (`v185_test.go`) is a
necessary, well-reasoned consequence of the codec change rather than an
independent or conflicting claim — the byte position is identical to the
old citation, only the field name is corrected via a documented, typed
cross-reference. All module-local builds and tests pass.
