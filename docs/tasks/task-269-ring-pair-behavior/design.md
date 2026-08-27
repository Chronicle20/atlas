# Ring Pair Field Behavior — Design

Task: task-269-ring-pair-behavior
PRD: [`prd.md`](prd.md) (v1, approved)
Status: Draft
Created: 2026-08-26

IDA sessions used (via `ida-pro` MCP, resolved by binary name per
[`docs/reverse-engineering.md`](../../reverse-engineering.md)):
`MapleStory_dump.exe.i64` (GMS v83), `GMS_v95.0_U_DEVM.exe.i64` (GMS v95).
Checked-in exports used: `docs/packets/ida-exports/gms_v48.json`,
`gms_jms_185.json`, `gms_v83.json`, `gms_v87.json`, `gms_v95.json`.

---

## 1. Scope correction — the PRD counts four stub literals; there are eleven, across three encoders

The PRD's §1 asserts the gap's "entire surface area on the wire" is three
`WriteByte(0)` calls in `spawn.go:174-176` plus one `WriteBool(false)` in
`info.go`. Derivation shows that is one of **three** encoders the v83 client
feeds from the same ring state, and that the one the PRD omits entirely is the
one that explains the tester's report most directly.

| # | Site | Current stub | Client reader |
|---|---|---|---|
| A | `libs/atlas-packet/character/data.go:760-766` `encodeRings` | `WriteShort(0)` ×3 | `CharacterData::Decode` v83 @0x4e592d, ring section @0x4e6333-0x4e63ba |
| B | `libs/atlas-packet/character/clientbound/spawn.go:174-176` | `WriteByte(0)` ×3 | `CUserRemote::Init` v83 @0x97f55d, ring block @0x97fb28-0x97fbf0 |
| C | `libs/atlas-packet/character/clientbound/appearance_update.go:36-38` | `WriteByte(0)` ×3 | `CUserRemote::OnAvatarModified` v83 @0x98367e, ring block @0x98372b-0x983803 |
| D | `libs/atlas-packet/character/clientbound/info.go:131` | `WriteBool(false)` | `CWvsContext::OnCharacterInfo` v83 @0xa2370b (bare bool — see §3.4) |

Site **A** is the local player's own ring records — the data behind "who is my
ring paired with" in the player's *own* UI and on the *own* item tooltip.
Site **B** is remote players. Site **C** is the mid-map update path, which the
PRD (FR-12) assumed did not exist. All three feed the same three
`CUserPool::On{Couple,Friend,Marriage}RecordAdd` registrars — proven by
`xrefs_to` on the v83 registrars, which returns exactly
`CUserLocal::SetPairCharacterID` (@0x95b335, fed by A),
`CUserRemote::Init` (@0x97f55d, B), and
`CUserRemote::OnAvatarModified` (@0x98367e, C).

**Consequence.** Filling only B and D — the PRD's literal scope — would leave
the buyer unable to see their own ring at all (A) and would make ring state
stale on any equip change until a map transfer (C). This design takes all four
sites. This is a scope *correction*, not a scope *expansion*: A and C are the
same feature, read by the same client code, from the same `cash_rings` rows,
and no acceptance criterion in PRD §10 is satisfiable without A.

## 2. Derivation results — all six open questions closed

### OQ-1 — What does the client read behind each ring flag?

**Answer: `Decode1` flag; if set, two 8-byte buffers then one `Decode4`.**
Marriage is `Decode1` flag then three `Decode4`. No count prefix on any GMS
column. Derived from v83 instruction-level disassembly:

```
0x97fb28  call CInPacket::Decode1              ; couple flag
0x97fb2f  jz   loc_97FB6B                      ; skip block if 0
0x97fb31  push 8   ; lea edi, [ebx+1F10h]
0x97fb3c  call CInPacket::DecodeBuffer         ; 8 bytes -> own ring SN
0x97fb41  push 8   ; lea eax, [ebx+1F18h]
0x97fb4c  call CInPacket::DecodeBuffer         ; 8 bytes -> partner ring SN
0x97fb5b  call CInPacket::Decode4              ; itemId
0x97fb66  call sub_972A5E                      ; CUserPool::OnCoupleRecordAdd

0x97fb6d  call CInPacket::Decode1              ; friendship flag
0x97fb81  call CInPacket::DecodeBuffer         ; 8 bytes -> own ring SN   (+1F88)
0x97fb91  call CInPacket::DecodeBuffer         ; 8 bytes -> partner SN     (+1F90)
0x97fba0  call CInPacket::Decode4              ; itemId
0x97fbab  call sub_972BD9                      ; CUserPool::OnFriendRecordAdd

0x97fbb2  call CInPacket::Decode1              ; marriage flag
0x97fbbd  call CInPacket::Decode4              ; -> [ebx+1F34]
0x97fbca  call CInPacket::Decode4              ; -> [ebx+1F38]
0x97fbd7  call CInPacket::Decode4              ; -> [ebx+1F3C]
0x97fbf0  call sub_972D54                      ; CUserPool::OnMarriageRecordAdd
```

The v95 IDB carries the same block with **symbols**, which name the types and
confirm the layout is unchanged from v83 to v95:

```
0x955ba2 Decode1 / 0x955bb6 DecodeBuffer / 0x955bc6 DecodeBuffer / 0x955bd3 Decode4
0x955bdd call CUserPool::OnCoupleRecordAdd(const _LARGE_INTEGER &, CUser *, long)
0x955c1f call CUserPool::OnFriendRecordAdd(const _LARGE_INTEGER &, CUser *, long)
0x955c65 call CUserPool::OnMarriageRecordAdd(unsigned long, CUser *, long)
```

`_LARGE_INTEGER` fixes each 8-byte buffer as a little-endian int64 — the cash
serial number (`GW_ItemSlotBase::liSN`), not an opaque blob.

The two ends of the supported range corroborate:

- **v48** (`gms_v48.json`, `CUserPool::OnUserEnterField` @0x6bbc17 with
  `CUserRemote::Init` inlined, calls 49-60): flag → `DecodeBuf`, `DecodeBuf`,
  `Decode4` for couple; same for friendship; flag → 3×`Decode4` for marriage.
  Identical to v83.
- **jms_v185** (`gms_jms_185.json`, `CUserRemote::Init` @0xa52876, calls 34-45,
  annotated): couple flag → `Decode4` **count** → per entry `DecodeBuf(16)` +
  `Decode4` itemId. JMS is the only column with a count, and its 16-byte entry
  is the two SNs as one buffer.

**Conclusion: the GMS spawn/avatar ring block is version-stable from v48 to
v95.** No `MajorAtLeast` gate is needed inside the GMS arm; the only gate is
GMS-vs-JMS.

### OQ-2 — Does the v83 client evaluate couple-ring proximity itself?

**Answer: yes. There is no server-sent effect packet, and none is needed.**
`sub_972A5E` (v83 `CUserPool::OnCoupleRecordAdd`) decompiles to a pairing
search over the user pool:

```c
result = *a2;              // low  dword of the incoming user's own ring SN
v6     = a2[1];            // high dword
v7     = this[2];          // CUserPool's local CUser
if (result == *(v7 + 7960) && v6 == *(v7 + 7964))   // +0x1F18 = local's PARTNER SN
    goto LABEL_9;
v8 = this[12];             // else walk the remote-user list
while (v8) { ... if (result == *(v10 + 7960) && v6 == *(v10 + 7964)) goto LABEL_9; }
LABEL_9:                   // matched: append a record carrying both character
                           // ids (+0x11A8), both SNs, and the itemId (a4)
```

The client matches *my* ring SN against *your* partner-SN field, across the
local user and every remote user in the pool, and maintains its own record
list from which the effect is rendered. The server's whole obligation is to
put the correct SN pair in the spawn/avatar block for every visible character.
**FR-10 stands as written; FR-11's scope-change branch does not trigger.**

### OQ-3 — Does `CharacterInfo` carry a partner block behind the marriage flag?

**Answer: no. It is a bare bool.** `CWvsContext::OnCharacterInfo` reads
`Decode1 bMarriageRing` and then goes straight to `DecodeStr sCommunity` on
every column, unconditionally — v83 @0xa2370b, v87 @0xabb181, v95 @0xa05750,
jms @0xb0aa6e (all four annotated `bMarriageRing (bool)` / `bIsMarried (bool)`
in the checked-in exports, with `sCommunity` as the next call and no guard).
Site D is a one-line change: replace `WriteBool(false)` with the computed flag.
The v29..v60 arm has no such bool and is untouched, as FR-8 requires.

### OQ-4 — Is `cash_rings.AssetId` the identifier the channel sees on an equipped item?

**Answer: no, and the correct join key is `cashId`, which the wire needs anyway.**

`PurchaseRingAndEmit` (task-240 branch, `cashshop/processor_ring.go` step 7)
stores `buyerAsset.Id()` / `partnerAsset.Id()` — the primary key of
`atlas-cashshop`'s own locker asset (`cashshop/inventory/asset.Model.id
uint32`). When the item leaves the locker,
`compartment.Release` emits `ReleasedStatusEventProvider(accountId,
characterId, id, type_, transactionId, assetId, cashId, templateId)` and the
receiving service mints a *new* row with a *new* id. The identifier that
survives is `cashId int64`, and the channel already carries it:
`services/atlas-channel/atlas.com/channel/asset/model.go:110` `CashId() int64`.

This is fortunate rather than merely acceptable: OQ-1 shows the wire field
**is** the cash SN. The same value serves as the join key and as the encoded
payload, so no id-translation layer is required anywhere.

`cash_rings` does not store `cashId`. `atlas-cashshop` resolves it in-service
(it owns both tables) and exposes it on the read model — the read-side addition
PRD §7 pre-authorizes.

### OQ-5 — Does the ring block need the partner's name or only their SN?

**Answer: both, at different sites.** Sites B and C carry no name — only two
SNs and an itemId (OQ-1). Site A does: `GW_CoupleRecord::Decode` is a single
fixed-width buffer whose width is derived per version.

| Record | v83 | v95 |
|---|---|---|
| `GW_CoupleRecord` | `DecodeBuffer(0x21)` = **33** bytes (`sub_4E48B0` @0x4e48b0) | `DecodeBuffer(0x21)` = **33** bytes (@0x4f2b60) |
| `GW_FriendRecord` | `DecodeBuffer(0x25)` = **37** bytes (`sub_4E48CE` @0x4e48ce) | (@0x4f2b70) |
| `GW_MarriageRecord` | `DecodeBuffer(0x30)` = **48** bytes (@0x4e4856) | (@0x4f2b50) |

33 = 8 (own SN) + 8 (partner SN) + 4 (itemId) + 13 (`sPairCharacterName`,
the fixed 13-byte MapleStory name field). The v95 decompiler names the
surrounding local `sPairCharacterName` at 0x4fde40, confirming the trailing
field. `GW_FriendRecord` is 33 + 4; `GW_MarriageRecord` is 48. Their
intra-record field splits are pinned to those exact widths and are resolved
during implementation against the same two IDBs; the widths themselves are
already fixed above, so no section can shift.

**Therefore the channel must resolve `partnerCharacterId` → name.** Two
options, resolved in §5.

### OQ-6 — Which versions are in scope?

**All ten columns.** The spawn/avatar block (OQ-1) is byte-identical across
GMS v48…v95, so a single GMS arm covers eight columns at once; JMS needs its
own count-prefixed arm. `CharacterData`'s ring section is count-prefixed on
every column already (`Decode2` count ×3 — v83 @0x4e6333/0x4e6361/0x4e638f,
v95 @0x4fde4e/0x4fde7a/0x4fdea6), and the existing encoder already writes
three zero shorts there gated on `> 28 || JMS`, so the version shape of site A
is unchanged — only the count values and record bodies become real.

`CHAR_INFO` and `SPAWN_PLAYER` are `verified` on 9 of 10 columns today
(`gms_v92` is `incomplete`, pre-existing). Every column stays byte-identical
on the empty path (§6).

## 3. Wire design

### 3.1 A shared codec, not three copies

The couple and friendship blocks are the same three fields at three sites, and
the marriage block is the same three fields at two. Three independent
hand-written copies is how the four literals became eleven. Introduce one
model type per record in `libs/atlas-packet/model`:

```go
// model/ring.go
type PairRing struct {          // couple and friendship share this shape
    OwnSN     int64             // GW_ItemSlotBase::liSN of this character's half
    PartnerSN int64             // liSN of the partner's half
    ItemId    uint32
}
type MarriageRing struct {
    MarriageId       uint32     // -> CUser+0x1F34
    PartnerCharacterId uint32   // -> CUser+0x1F38
    ItemId           uint32     // -> CUser+0x1F3C
}
type RingSet struct {           // what one character carries onto the field
    Couple     *PairRing
    Friendship *PairRing
    Marriage   *MarriageRing
}
```

`RingSet` gets `EncodeField`/`DecodeField` (sites B and C — flag-gated,
GMS/JMS arms) and `EncodeRecords`/`DecodeRecords` (site A — count-prefixed
fixed-width records). Sites B and C then each call one method, and the
version gate lives in exactly one file.

Pointer-nil is the "no ring" signal; `RingSet{}` encodes as three zero bytes
at B/C and three zero shorts at A — byte-identical to today (§6).

### 3.2 Site B / C — field blocks

```go
func (r RingSet) EncodeField(w *response.Writer, t tenant.Model) {
    if t.Region() == "JMS" { r.encodeFieldJMS(w); return }
    encodePairRing(w, r.Couple)      // Decode1 flag + 8 + 8 + 4
    encodePairRing(w, r.Friendship)
    encodeMarriageRing(w, r.Marriage) // Decode1 flag + 4 + 4 + 4
}
```

The JMS arm writes `flag / Int count / per entry {16-byte SN pair, Int itemId}`
per `gms_jms_185.json` calls 34-41, with count ∈ {0,1} in practice.

`CharacterSpawn` and `CharacterAppearanceUpdate` each gain a `rings RingSet`
constructor parameter, replacing the three literals.

> **Pre-existing divergence noted, not fixed here.** v83
> `CUserRemote::OnAvatarModified` @0x98367e reads `Decode1` mode, `AvatarLook`,
> `Decode1` (@0x9836f7), `Decode1` → `SetCarryItemEffect` (@0x983719), then the
> three ring blocks, and **no trailing `Decode4`**. The current
> `appearance_update.go` writes mode, avatar, three ring bytes, and
> `WriteInt(0) // completed set item id` — two bytes short and one int long.
> The ring blocks cannot be filled correctly without also correcting the
> surrounding frame, so that correction is in scope for this task and is called
> out explicitly so it is not mistaken for drive-by churn. Its matrix cell is
> re-verified alongside the rings.

### 3.3 Site A — `CharacterData` records

`encodeRings` becomes:

```go
func (m *CharacterData) encodeRings(w *response.Writer, t tenant.Model) {
    writeRingRecords(w, m.Rings.CoupleRecords, coupleRecordWidth(t))
    if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
        writeRingRecords(w, m.Rings.FriendRecords, friendRecordWidth(t))
        writeRingRecords(w, m.Rings.MarriageRecords, marriageRecordWidth(t))
    }
}
```

Each record is written as its exact fixed width (33 / 37 / 48 on v83 and v95),
name field zero-padded to 13 bytes and truncated at 13. The existing
`> 28 || JMS` gate is preserved verbatim so the v29..v60 shape does not move.

### 3.4 Site D — `CharacterInfo`

`w.WriteBool(false) // marriage ring` → `w.WriteBool(m.hasMarriageRing)`.
Nothing else changes; the client reads nothing more (OQ-3). The v29..v60 arm
is untouched.

## 4. Channel-side architecture

```
atlas-cashshop  GET /rings?filter[characterId]=N   (+cashId, +partnerCashId, +partnerName)
        │
        ▼
atlas-channel/ring          read-only processor + per-(tenant,character) cache
        │
        ├── character load / map enter  → populate
        ├── RING_PURCHASED status event → invalidate
        └── equip/unequip of a ring     → recompute RingSet, emit AppearanceUpdate
        │
        ▼
socket/writer/{character_spawn, character_appearance_update, character_info, set_field}
```

### 4.1 `atlas-channel/ring` package

A standard channel REST consumer (`rest.go` / `processor.go` / `model.go`,
shaped after `services/atlas-channel/atlas.com/channel/door/rest.go`), plus:

- **Cache.** Keyed by `(tenantId, characterId)`, populated on character load,
  never on encode (NFR: spawn is hot). Entry holds the raw `ACTIVE` halves.
- **`GetRingSet(characterId, equipped []asset.Model) RingSet`.** Pure function
  over cached halves + the character's equipped compartment. Joins
  `half.CashId == equipped.CashId()` (OQ-4), drops halves with no equipped
  match (FR-14), drops non-`ACTIVE` halves client-side (FR-3).
- **Selection (FR-15).** The GMS wire admits one couple and one friendship
  ring. When more than one `ACTIVE` half of a type is equipped, take the one
  in the **lowest equipped slot** (`asset.Slot()`, most negative first —
  equipped slots are negative), ties broken by lowest `cashId`. Deterministic,
  matches "the ring the client draws first," and stated in the model's doc
  comment so it is not re-litigated.
- **Failure (FR-5).** `GetRingSet` returns the zero `RingSet` on any error and
  logs at warn with the character id. Callers do not branch — the zero value
  already encodes to today's bytes.

### 4.2 `atlas-cashshop` read-model addition

`ring.RestModel` gains three fields, all resolvable in-service:

| Field | Source |
|---|---|
| `cashId int64` | join `cash_rings.AssetId` → `cashshop.inventory.asset.cash_id` |
| `partnerCashId int64` | same join via the sibling row sharing `PairId` |
| `partnerName string` | `chaP.GetById(partnerCharacterId)` — already a dependency of `PurchaseRingAndEmit` |

`partnerName` is what site A's 13-byte `sPairCharacterName` needs (OQ-5).
Putting it on the cash-shop read model rather than resolving it channel-side
keeps the channel to one upstream call per character on a hot path; the
alternative is rejected in §5.

### 4.3 Cache invalidation and the FR-12 limitation

- `RING_PURCHASED` (`kafka/consumer/cashshop/consumer.go:461`) invalidates the
  buyer and, when a session for them exists on this channel, the partner.
- Map/channel transfer drops the entry.
- **Equip/unequip of a ring** recomputes the `RingSet` and emits a
  `CharacterAppearanceUpdate` — which site C now carries correctly. This is why
  C is in scope: with it, ring state is live within a map, and **FR-12's
  documented limitation shrinks to the one case it cannot cover** — a ring
  purchased by a partner who is already standing on your map, where the
  partner's own client learns of its new half via `RING_PURCHASED` and emits
  its own appearance update. Service docs record the residual case rather than
  the blanket "requires a map change."

## 5. Alternatives considered

**Resolve the partner name in `atlas-channel` instead of `atlas-cashshop`.**
Rejected. It adds an N+1 character lookup on the spawn-population path for a
value `atlas-cashshop` already has in hand (it resolves the partner character
during purchase). The channel would need its own name cache with its own
invalidation, duplicating what the ring cache already does.

**Filter `state == ACTIVE` server-side in `atlas-cashshop`.** Rejected, per
FR-3, and the derivation reinforces it: a `BROKEN` half still occupies an equip
slot, and a future "show the broken ring greyed out" behavior would need the
row. One filter, channel-side, in `GetRingSet`.

**Fill only sites B and D (the PRD's literal scope).** Rejected — see §1. It
cannot satisfy PRD §10's own criteria ("see who my ring is paired with when I
inspect my own character", "the partner name on the ring item in my equip
inventory"), both of which are site A.

**Add `cashId` to `cash_rings` as a stored column.** Rejected. It duplicates a
value the owning table already holds within the same service and same
transaction, and introduces a second place for it to go stale. The join is
cheap and `atlas-cashshop` owns both sides.

**Server-side proximity evaluation.** Rejected on evidence, not preference —
OQ-2 shows the client does it. Adding a server tick would be dead code.

**Three hand-written copies of the block, one per encoder.** Rejected — §3.1.

## 6. The empty-path invariant (FR-9)

The hard constraint. For a character with no `ACTIVE` equipped ring half:

- Site A writes `WriteShort(0)` ×3 — identical to today's `encodeRings`.
- Sites B and C write `WriteByte(0)` ×3 — identical to today's literals,
  because `Decode1(0)` skips the block (v83 `jz loc_97FB6B` @0x97fb2f).
- Site D writes `WriteBool(false)` — identical to today.

A regression test asserts byte-identical output versus current `main` for the
zero `RingSet` on **every** currently-verified column of `SPAWN_PLAYER`,
`CHAR_INFO`, `CHARACTER_DATA`, and the appearance-update op. Site C is the one
exception and is deliberate: its frame is being corrected (§3.2), so its
expected bytes change on the empty path too, and its cell is re-verified rather
than pinned to the old output.

## 7. Coverage

`coverage-manifest.yaml` (FR-18) declares:

| Codec | Columns |
|---|---|
| `character/clientbound/CharacterSpawn` | all 10 |
| `character/clientbound/CharacterInfo` | all 10 (v29..v60 arm unchanged) |
| `character/clientbound/CharacterAppearanceUpdate` | all 10 |
| `character/CharacterData` (via `field/clientbound/SetField`) | all 10 |

Each op × version cell gets a `packet-audit:verify` byte fixture covering
**both** a populated and an empty `RingSet`, its evidence record pinned against
the addresses cited in §2, and `docs/packets/audits/STATUS.md` regenerated.
`packet-completeness-critic` runs before the PR.

`gms_v92` is `incomplete` for `SPAWN_PLAYER` and `CHAR_INFO` today. This task
touches those codecs, so the manifest claims v92 and the cells are promoted
rather than left at their pre-existing `incomplete` — a codec this task edits
is a codec this task verifies.

## 8. Dependency (unchanged)

PRD §11 stands: `cash_rings`, the `ring` package, `GET /rings`, and
`RING_PURCHASED` exist only on `origin/task-240-cash-shop-stub-operations`
(verified: `git ls-tree origin/main -- services/atlas-cashshop/.../ring`
returns nothing). This design was derived against that branch. **Planning and
execution must not begin until #1426 is on `main`.**

## 9. Items for the user

1. **Scope.** §1 takes two encoder sites (`character/data.go`,
   `appearance_update.go`) beyond the PRD's stated four literals, and §3.2
   corrects a pre-existing frame defect in `appearance_update.go`. Both are
   load-bearing for the PRD's own acceptance criteria, but they widen the diff
   and add two ops to the coverage manifest. Confirm before planning.
2. **`atlas-cashshop` change.** §4.2 adds three read-only fields to
   `ring.RestModel` on a branch that has not merged yet. That is a change to
   task-240's surface after its review; it may be cleaner to land it as a
   follow-up on `main` than to amend #1426.
