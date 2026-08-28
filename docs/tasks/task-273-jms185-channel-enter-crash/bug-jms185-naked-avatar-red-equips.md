# bug: JMS 185.1 — in-field avatar renders naked, equipped items render "red", double-click self crashes

Status: **the double-click crash is root-caused and proven. The naked/red
symptom has one strong, evidence-backed hypothesis (nDurability = 0) that the
live re-test must confirm.** Two further JMS defects were found along the way,
both proven against the client and both latent for this character.

Follows `bug-jms185-channel-enter-crash.md` (round 7, `550ec76d0`), which fixed
the channel-enter crash. The client now reaches the field; these are the next
symptoms.

## Reproduced

Live, in the **PR environment** (not `atlas-main`), with per-tenant packet trace
logging enabled by the user:

- Tenant `728cc816-4e9e-4c9b-96ba-e6c9a50fdf4a` ("185"), region `JMS`,
  `ms.version 185.1`, environment **`pr-1555`**.
- Namespace `atlas-pr-1555` carries only `atlas-channel`
  (`atlas-channel-64ccc5867-nrln9`), `atlas-login`, `atlas-ingress`; every other
  service comes from the shared `atlas-main` deployment.
- Character **41** "Atlas", account 24, level 200, job **412**, gender 0,
  skin 0, hair 30030, face 20000.
- Crash session `af716794-5a5d-47c1-83d4-2351152aa00e`, 2026-08-28T16:51Z.

Character 41's inventory (atlas-inventory, tenant above) — **no cash equips, no
pets**:

| asset | slot | templateId | cashId |
|---|---|---|---|
| 423 | -1  | 1002357 | 0 |
| 428 | -2  | 1012098 | 0 |
| 427 | -4  | 1032031 | 0 |
| 422 | -5  | 1052163 | 0 |
| 426 | -8  | 1082223 | 0 |
| 424 | -9  | 1102041 | 0 |
| 421 | -11 | 1472032 | 0 |
| 425 | 1 (equip inv) | 1072344 | 0 |
| 417–420 | 1–4 (use inv) | 2000004, 2000005, 2030019, 2070016 | 0 |

`1012098` is the user's "maple leaf face accessory"; atlas-data returns
`reqLevel 0, reqJob 0`, all stat reqs 0, `cash false`, `timeLimited true`. A
zero-requirement, unisex item rendering red is what rules out a requirement
check as the explanation.

## Observed

1. In the field the character renders **naked**, despite 7 equipped items.
2. The same character renders **correctly equipped** on login character-select
   and in the cash shop (both server-built `AvatarLook`, no item structs).
3. In-game the inventory/equip window shows the **correct icons, tinted red**
   (confirmed by the user) — so the client resolved every item id against its
   own WZ.
4. **Double-clicking the character kills the client.**

## Root cause 1 — `CharacterInfo` (0x0035) is 8 bytes short on JMS (PROVEN)

The trace names it exactly. Double-click sends
`CharacterInfoRequestHandle op=0x005c`, the server answers
`writer=CharacterInfo op=0x0035 len=50`, and the client emits nothing further
except the 0xEA idle tick before the connection dies — the same over-read →
`CInPacket` throw → `CWvsApp::WindowProc` teardown chain proven in round 6 of
the previous bug file.

Wire bytes sent (48-byte body):

```
3500 29000000 c8 9c01 0000 00 0000 0000 00 00 00 00 00000000 ... 00
```

`CWvsContext::OnCharacterInfo` @0xb0aa6e (JMS 185) reads:

| # | read | Atlas JMS arm (`info.go`) |
|---|---|---|
| 1 | `Decode4` dwCharacterID | `WriteInt` ✅ |
| 2 | `Decode1` nLevel | `WriteByte` ✅ |
| 3 | `Decode2` nJob | `WriteShort` ✅ |
| 4 | `Decode2` nPOP | `WriteInt16` ✅ |
| 5 | `Decode1` bIsMarried | `WriteBool` ✅ |
| 6 | `DecodeStr` sCommunity | `WriteAsciiString(guild)` ✅ |
| 7 | `DecodeStr` sAlliance | `WriteAsciiString("")` ✅ |
| 8 | **`Decode4` @0xb0ab0f** | **MISSING** |
| 9 | **`Decode4` @0xb0ab19** | **MISSING** |
| 10 | `Decode1` @0xb0ab23 (medal info) | `WriteByte(0)` ✅ |
| 11 | `Decode1` @0xb0ab30 bPetActivated | `WriteBool(false)` ✅ |
| 12 | `SetMultiPetInfo` @0x9bb959 — reads nothing when #11 is 0 | (no pets) ✅ |
| 13 | `Decode1` taming-mob flag, then 3×`Decode4` if set | `WriteByte(0)` ✅ |
| 14 | `Decode1` wishlist count + `DecodeBuffer(4*n)` | ✅ |
| 15 | `SomethingMonsterBook` @0x70522a — 5×`Decode4` | 5×`WriteInt` ✅ |
| 16 | `MedalAchievementInfo::Decode` @0x9bcacf — `Decode4` + `Decode2` count + count×`Decode2` | `WriteInt(medalId)` + `WriteShort(0)` ✅ |
| 17 | `Decode4` count + `DecodeBuffer(4*count)` (chair items) | `WriteInt(0)` ✅ |

Rows 8 and 9 are the defect: **8 bytes**. The client needs 56 body bytes and
gets 48.

They are a genuine JMS divergence, not a misread. GMS v95's
`CWvsContext::OnCharacterInfo` @0xa05750 goes straight from the alliance string
to `Decode1 pMedalInfo` / `Decode1 bPetActivated` — no such ints. In JMS both
values are pushed to `CUIUserInfo::SetUserInfo` (@0xb0ab70 `push [ebp+p]`,
@0xb0ab79 `push [ebp+var_20]`, ahead of the nine GMS arguments), and
`SetUserInfo` @0x9bb7e6 stores them at `this+670` and `this+671` — display-only
fields adjacent to the community/alliance strings. **Their meaning is not
established** (the JMS dump does not name them); they are most plausibly ids
paired with the two strings. Character 41 has no guild and no alliance, so 0 is
correct in effect here regardless — but the encoder comment must say the fields
are unidentified rather than assert a name.

## Root cause 2 — every JMS equip is sent with `nDurability = 0` (STRONG HYPOTHESIS)

The SetField payload is otherwise proven correct. From the same session,
`writer=SetField op=0x007b len=1671`: all seven equips are present in the
equipped section, in **98-byte blocks** (the exact JMS block size), with correct
slots and ids:

```
off 180 slot 1  type 1 tid 1002357 cash 0 expire -1
off 278 slot 2  type 1 tid 1012098 cash 0 expire -1
off 376 slot 4  type 1 tid 1032031 cash 0 expire -1
off 474 slot 9  type 1 tid 1102041 cash 0 expire -1
off 572 slot 5  type 1 tid 1052163 cash 0 expire -1
off 670 slot 11 type 1 tid 1472032 cash 0 expire -1
off 768 slot 8  type 1 tid 1082223 cash 0 expire -1
terminator 00000000
```

So the equips reach the client, aligned, in range, and the client stores and
renders their icons. What is wrong is a *value*, and the field-order alignment
against GMS v95 identifies which one.

GMS v95's `GW_ItemSlotEquip::RawDecode` @0x4f8360 is fully typed and gives the
authoritative field order:

```
nRUC(1) nCUC(1) | 15×short niSTR..niJump | sTitle str | nAttribute(2)
nLevelUpType(1) nLevel(1) nEXP(4) nDurability(4) nIUC(4)
nGrade(1) nCHUC(1) nOption1..3(2,2,2) nSocket1(2) nSocket2(2)
[liSN(8) if !cash] ftEquipped(8) nPrevBonusExpRate(4)
```

JMS 185's @0x50feb9 is the same skeleton with one extra leading `Decode1` and a
relocated trailing `Decode4`:

```
Decode1 ×3 | 15×Decode2 | DecodeStr | Decode2 | Decode1 Decode1
Decode4 Decode4 | Decode1 | Decode2 ×5 | Decode4
[DecodeBuffer(8) if !cash] DecodeBuffer(8) Decode4
```

Aligning them, the `Decode4` immediately after `nEXP` is **`nDurability`**.
Atlas's JMS arm writes `m.hammersApplied` there
(`libs/atlas-packet/model/asset.go:277`), which is **0** for every one of these
items — while the GMS v84+ arm deliberately writes `-1` at the same position
with the comment "-1 = no durability". Atlas never sends `-1` on the JMS path.

A durability of 0 is the client's "broken" state: broken equips are drawn with
the red overlay and are not worn on the avatar. That accounts for both symptoms
at once, uniformly, on every item regardless of requirements — which is exactly
what was reported, including the requirement-free face accessory.

This is a hypothesis, not a proof: the red-overlay predicate itself was not
located (`CUIEquip`'s methods are not symbolized in this dump). It is confirmed
or refuted by one live re-test.

`nIUC` (hammersApplied) has no proven JMS position. The trailing `Decode4`
before the buffers is the natural candidate given v95's field set, but that
ordering is **not established**, so this fix must not silently move
hammersApplied there — leave that slot as it is and record the gap.

## Ruled out (with evidence)

- **The equips missing from the packet** — all seven present, above.
- **Slot rejection.** `CharacterData::Decode` @0x513b60 accepts equipped slots
  `1..0x24` (@0x513b95/@0x513b9e) and stores to `[charData + slot*8 + 0xFD]`
  (@0x513be0). Slots 1–11 are in range.
- **The avatar builder filtering on anything.** `CUserLocal::Init` @0xa09bbe →
  `sub_42A915` → `sub_514F91` @0x514f91 loops slots 1..36, preferring the cash
  item (slot 11 excluded) over the regular one, and reads the item id straight
  out of the ZRef. Disassembly confirms it reads at exactly the offsets the
  decoder writes (regular ZRef object `0xFD + 8s`, its `p` at `+4` = `0x101 +
  8s`, which is what @0x514fe6/@0x514ff9 dereference; cash likewise `0x225 +
  8s` / `0x229 + 8s`). **No level, job, stat or expiry check anywhere.** So
  `dateExpire = -1` cannot cause the naked avatar.
- **Body-part / gender mismatch.** `is_correct_bodypart` @0x46b993 maps
  `itemId/10000`: 100→1, 101→2, 103→4, 105→5, 108→8, 110→9, 13x/14x/16x/17x→11.
  Every one of character 41's items matches its slot, and all are unisex
  (`(id/1000)%10 == 2`).
- **Level/job/stat requirements** — `1012098` has none.
- **The stat block.** `GW_CharacterStat::Decode` @0x50ec17 is fixed-length apart
  from one job branch, `sub_5163A2`: `job/1000==3 || job/100==22 || job==2001`.
  Job 412 takes the same arm as job 0, and the JMS arm of `encodeStats` matches
  the client read-for-read (10 shorts after nJob, then 4,2,4,4,1,2,8,4,4,4).
- **The non-cash equip block desyncing** — byte-exact against @0x50feb9, and the
  98-byte block spacing in the capture confirms it on the wire.
- **Missing server-side item data** — atlas-data returns 200 for all four ids
  checked under this tenant.

## Latent defects found (real, proven, NOT the reported symptoms)

Both are certain desyncs waiting on a character that trips them; fix them in
this round with byte-length fixtures.

1. **`encodeCashEquipableInfo` is 15 bytes short on JMS.**
   `GW_ItemSlotEquip::RawDecode` @0x50feb9 reads the JMS block — `Decode1` +
   `Decode2`×5 + `Decode4` — unconditionally for cash and non-cash items alike;
   only the *first* trailing `DecodeBuffer(8)` is gated on `liCashItemSN == 0`.
   The non-cash arm emits that block (`asset.go:294-302`); the cash arm jumps
   from its 10-byte filler straight to `WriteInt64`/`WriteInt32`. Any JMS
   character holding a cash equip desyncs `CharacterData` by 15 bytes.

2. **The extended-SP job gate is GMS-only.** `encodeStats`/`decodeStats` gate on
   `t.IsRegion("GMS") && t.MajorAtLeast(84) && isEvanJob(jobId)`, so JMS always
   writes the `nSP` short. JMS 185 gates on `sub_5163A2`:
   `job/1000 == 3 || job/100 == 22 || job == 2001` — i.e. **Resistance (3xxx)
   too**, not just Evan. A JMS Evan or Resistance character desyncs at the stat
   block.

## Fix

- `libs/atlas-packet/character/clientbound/info.go` — `CharacterInfo.Encode`,
  JMS arm: emit two `WriteInt(0)` between the alliance `WriteAsciiString("")`
  and the `WriteByte(0)` medal-info byte, gated `t.Region() == "JMS"` only.
  Comment them as the two unidentified int32s read at
  `CWvsContext::OnCharacterInfo` @0xb0ab0f/@0xb0ab19 and stored at
  `CUIUserInfo+670/+671` by `SetUserInfo` @0x9bb7e6 — do **not** name them.
  Mirror the same two reads in `CharacterInfo.Decode`. Every GMS arm must stay
  byte-identical.
- `libs/atlas-packet/character/clientbound/info_test.go` — JMS byte-length
  fixture pinning the encoded body at **56** bytes for a character with no
  guild/alliance/pets/mount/wishlist, plus an Encode/Decode round-trip. Keep the
  existing GMS fixtures unchanged and passing byte-identically.
- `libs/atlas-packet/model/asset.go` — `encodeEquipableInfo`, JMS arm: the
  `Decode4` immediately after `experience` is `nDurability`, not
  `hammersApplied`. Emit `WriteInt32(-1)` there for JMS, matching what the
  GMS v84+ arm already does at the same position. Leave the JMS trailing block
  (`WriteByte(0)`, 5×`WriteShort(0)`, `WriteInt(0)`) and every GMS arm exactly
  as they are — `nIUC`'s JMS position is unproven, so hammersApplied stays
  unsent on JMS and that is recorded, not guessed at.
- `libs/atlas-packet/model/asset.go` — `encodeCashEquipableInfo`, JMS arm: emit
  the same JMS block the non-cash arm emits (`WriteByte(0)`, five
  `WriteShort(0)`, `WriteInt(0)`) between the filler and the trailing
  `WriteInt64`/`WriteInt32`. GMS arms unchanged.
- `libs/atlas-packet/model/asset_jms185_test.go` (new) — byte-length fixtures
  for a JMS non-cash equip (98 bytes including the 2-byte slot prefix, matching
  the captured block size) and a JMS cash equip, both pinned against
  `GW_ItemSlotEquip::RawDecode` @0x50feb9, plus an assertion that the int32
  after `experience` is `-1` on the JMS arm.
- `libs/atlas-packet/character/data.go` — `encodeStats`/`decodeStats`: replace
  the GMS-only Evan gate with a region-correct one. JMS must use
  `job/1000 == 3 || job/100 == 22 || job == 2001` (`sub_5163A2` @0x5163a2);
  the GMS gate (`>= 84 && isEvanJob`) is unchanged. Add a JMS fixture for a
  job-3xxx character asserting the extended-SP byte replaces the `nSP` short.

- `libs/atlas-packet/parcel/clientbound/v185_test.go` — **added to this
  inventory after the fact.** Its hand-built JMS golden `wantEquipItemBytesV185`
  pins the same shared `encodeEquipableInfo` output and labels the int32 after
  `experience` as `hammersApplied = 0`, citing an earlier IDA session. That is
  the same wire address (@0x5100e1) this fix reinterprets as `nDurability` via
  the typed GMS v95 cross-reference — a corrected field *name* at an unchanged
  offset, not a competing measurement. The fixture and its comment must be
  updated to say so rather than silently re-baselined. Missing this file was an
  omission in the original inventory, not a scope expansion by the implementer.

Every module-local `go build ./...` / `go test ./...` in `libs/atlas-packet`
must pass, and no GMS fixture may change by a single byte.

## Resolution

Fixed in **`5e08e8365`**, merged with `origin/main` in **`720a56c36`** and
pushed to PR 1555. Reviewed by `task-reviewer`:
**APPROVED_WITH_FINDINGS**, 0 blocking (see
`reviews/review-bug-jms185-naked-red.md`).

**Confirmed live by the user on the JMS 185 client**: the character renders
dressed and the equips no longer render red, and double-clicking the character
no longer kills the client. Both root causes are therefore closed:

- Root cause 1 (`CharacterInfo` 0x0035 eight bytes short) was proven from the
  binary before the fix and is confirmed by the live re-test.
- Root cause 2 (`nDurability = 0` on every JMS equip) was only a hypothesis when
  the fix landed — the red-overlay predicate itself was never located. **The
  live re-test is what confirms it.** Sending `-1` at the `Decode4` after `nEXP`
  dresses the avatar and clears the red overlay, which is the behaviour a
  "broken item" reading of durability 0 predicts. That is behavioural
  confirmation, not a located predicate; the client function that consumes the
  field remains unidentified.

The merge with `origin/main` also collapsed latent defect 2 into main's own
work: #1549 had already added `job.UsesExtendedSP(jobId, region, major)` to
atlas-constants, and its JMS arm is byte-identical to `sub_5163A2`
(`jobId/1000 == 3 || jobId/100 == 22 || jobId == EvanId`) — an independent
derivation reaching the same rule. The local `isJmsExtendedSpJob`/`isEvanJob`
helpers were deleted in favour of that shared predicate; the JMS test was
retargeted at it so the codec stays pinned if it is ever edited, and now also
asserts the GMS v84..91 arm stays Evan-only.

Latent defect 1 (the 15-byte cash-equip shortfall) is fixed and fixture-pinned
but remains **unexercised live** — character 41 holds no cash equips, so no
session has yet put a JMS cash equip on the wire.

## Not yet answered

- **What the two JMS `CharacterInfo` int32s mean.** Stored at `CUIUserInfo+670`
  and `+671`, display-only, unnamed in this dump. Sent as 0; correct in effect
  for a guildless character, unverified for one in a guild.
- **Whether `nDurability = 0` is really what turns the items red.** The
  red-overlay predicate was not located. The live re-test decides it: if the
  avatar dresses and the red clears, root cause 2 is confirmed.
- **`nIUC` (hammersApplied) has no proven position on the JMS wire.** After this
  fix it is not sent at all on JMS. Harmless today (every item has 0 hammers),
  but it is a known gap, not a solved field.
- The movement decoder logs `Code [N] not configured for use in movement` errors
  for this tenant (e.g. 16:26:35.799, 16:30:24.319) — a separate JMS movement
  fragment divergence, out of scope here.
