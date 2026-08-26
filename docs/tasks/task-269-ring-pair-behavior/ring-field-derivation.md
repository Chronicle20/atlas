# Ring field derivation — task-269

Written by **Task 1** of the task-269 plan. Its purpose is narrow and
blocking: settle the `dwPairCharacterId` vs `itemId` conflict for the
trailing 4-byte field of the spawn/avatar ring block, and confirm or refute
`design.md` §3.1's name for the first field of the marriage arm. Tasks 2 and
3 take their struct field names from the verdicts below.

Evidence discipline per `docs/packets/audits/VERIFYING_A_PACKET.md` §5: every
claim below is a decompile line or a disassembly line from a named IDA
session, quoted verbatim with its address. Nothing here is inferred from
remembered MapleStory knowledge.

Task 3 appends to this file. Blocks are sectioned per ring arm so it can
extend them without touching the verdicts.

## IDA sessions used

Resolved per `docs/reverse-engineering.md` — `idb_list`, matched by binary
name, session id passed as `database` on every call. Port-based selection is
dead.

| Column | Binary (filename in `idb_list`) | Session id |
|---|---|---|
| GMS v95 | `GMS_v95.0_U_DEVM.exe.i64` | `ecc757f4` |
| GMS v83 | `MapleStory_dump.exe.i64` | `754107bf` |

Both were open and adopted. Everything in this document is
**decompile-derived**, not export-derived. The checked-in exports under
`docs/packets/ida-exports/` are used below only as the artifact *under test*.

## The wire block (GMS, `CUserRemote::OnAvatarModified`)

v95 `CUserRemote::OnAvatarModified` @0x954110 (symbolised; `decompile`,
session `ecc757f4`) is the whole block in one function:

```c
  v7 = CInPacket::Decode1(iPacket);                                       /*0x9541b7*/
  bCouple = v7;
  if ( v7 )
  {
    CInPacket::DecodeBuffer(iPacket, &this->m_liCoupleItemSN, 8u);        /*0x9541d2*/
    CInPacket::DecodeBuffer(iPacket, &this->m_liPairItemSN, 8u);          /*0x9541e2*/
    v9 = CInPacket::Decode4(iPacket);                                     /*0x9541ef*/
    CUserPool::OnCoupleRecordAdd(m_pInterface, &this->m_liCoupleItemSN, this, v9); /*0x9541f9*/
  }
  v10 = CInPacket::Decode1(iPacket);                                      /*0x954202*/
  if ( v10 )
  {
    CInPacket::DecodeBuffer(iPacket, &this->m_liFriendshipItemSN, 8u);        /*0x95421d*/
    CInPacket::DecodeBuffer(iPacket, &this->m_liFriendshipPairItemSN, 8u);    /*0x95422d*/
    v13 = CInPacket::Decode4(iPacket);                                       /*0x95423a*/
    CUserPool::OnFriendRecordAdd(v12, &this->m_liFriendshipItemSN, this, v13); /*0x954244*/
  }
  if ( CInPacket::Decode1(iPacket) )                                      /*0x954251*/
  {
    this->m_dwMarriageCharacterID     = CInPacket::Decode4(iPacket);      /*0x954263*/
    this->m_dwMarriagePairCharacterID = CInPacket::Decode4(iPacket);      /*0x954270*/
    v14 = CInPacket::Decode4(iPacket);                                    /*0x954276*/
    this->m_nWeddingRingID = v14;                                         /*0x95427c*/
    CUserPool::OnMarriageRecordAdd(..., this->m_dwMarriageCharacterID, this, v14); /*0x954290*/
  }
```

The same shape on v83 `CUserRemote::OnAvatarModified` @0x98367e (session
`754107bf`), with raw offsets because v83 carries no symbols:

```c
  v6 = CInPacket::Decode1(a2);                       /*0x98372b*/  // bCouple
  if ( v6 ) {
    CInPacket::DecodeBuffer(a2, this + 7952, 8);     /*0x983745*/  // +0x1F10 own SN
    CInPacket::DecodeBuffer(a2, this + 7960, 8);     /*0x983755*/  // +0x1F18 pair SN
    v7 = CInPacket::Decode4(a2);                     /*0x983764*/  // <-- contested field
    sub_972A5E(a2b, this + 1988, this, v7);          /*0x98376f*/  // = OnCoupleRecordAdd
  }
  v8 = CInPacket::Decode1(a2);                       /*0x983778*/  // bFriendship
  if ( v8 ) {
    CInPacket::DecodeBuffer(a2, this + 8072, 8);     /*0x983792*/  // +0x1F88 own SN
    CInPacket::DecodeBuffer(a2, this + 8080, 8);     /*0x9837a2*/  // +0x1F90 pair SN
    v9 = CInPacket::Decode4(a2);                     /*0x9837b1*/  // <-- contested field
    sub_972BD9(a2c, this + 2018, this, v9);          /*0x9837bc*/  // = OnFriendRecordAdd
  }
  if ( CInPacket::Decode1(a2) ) {                    /*0x9837c5*/  // bMarriage
    *(this + 1997) = CInPacket::Decode4(a2);         /*0x9837d7*/  // +0x1F34
    *(this + 1998) = CInPacket::Decode4(a2);         /*0x9837e4*/  // +0x1F38
    v10            = CInPacket::Decode4(a2);         /*0x9837ea*/  // +0x1F3C
    sub_972D54(..., *(this + 1997), this, v10);      /*0x983803*/  // = OnMarriageRecordAdd
  }
```

And the same block again in v83 `CUserRemote::Init` @0x97f55d, raw
disassembly over [0x97fb20, 0x97fc00) (`insn_query`, session `754107bf`) —
byte-identical field order, so `Init` and `OnAvatarModified` share one shape:

```
0x97fb28  call ?Decode1@CInPacket@@QAEEXZ                 ; bCouple
0x97fb33  lea  edi, [ebx+1F10h]
0x97fb3c  call ?DecodeBuffer@CInPacket@@QAEXPAXI@Z        ; push 8  -> +0x1F10
0x97fb43  lea  eax, [ebx+1F18h]
0x97fb4c  call ?DecodeBuffer@CInPacket@@QAEXPAXI@Z        ; push 8  -> +0x1F18
0x97fb5b  call ?Decode4@CInPacket@@QAEKXZ                 ; <-- contested field
0x97fb63  push eax                                        ;     pushed straight through
0x97fb66  call sub_972A5E                                 ; = OnCoupleRecordAdd
0x97fb6d  call ?Decode1@CInPacket@@QAEEXZ                 ; bFriendship
0x97fb78  lea  edi, [ebx+1F88h]
0x97fb81  call ?DecodeBuffer@CInPacket@@QAEXPAXI@Z        ;         -> +0x1F88
0x97fb88  lea  eax, [ebx+1F90h]
0x97fb91  call ?DecodeBuffer@CInPacket@@QAEXPAXI@Z        ;         -> +0x1F90
0x97fba0  call ?Decode4@CInPacket@@QAEKXZ                 ; <-- contested field
0x97fbab  call sub_972BD9                                 ; = OnFriendRecordAdd
0x97fbb2  call ?Decode1@CInPacket@@QAEEXZ                 ; bMarriage
0x97fbbd  call ?Decode4@CInPacket@@QAEKXZ
0x97fbc4  mov  [ebx+1F34h], eax
0x97fbca  call ?Decode4@CInPacket@@QAEKXZ
0x97fbd1  mov  [ebx+1F38h], eax
0x97fbd7  call ?Decode4@CInPacket@@QAEKXZ
0x97fbde  push dword ptr [ebx+1F34h]                      ; 2nd arg to the record-add
0x97fbe4  mov  [ebx+1F3Ch], eax
0x97fbf0  call sub_972D54                                 ; = OnMarriageRecordAdd
```

Note at 0x97fb5b/0x97fba0 and at 0x9541ef/0x95423a: the couple and friendship
`Decode4` result is **never stored into any `CUser` member**. It is pushed
directly as the fourth argument of the record registrar and then discarded.
There is therefore no member offset to run `xrefs_to_field` against — the
absence of a store is itself part of the answer, and it is the first
structural reason the value cannot be a "pair character id": the client keeps
every ring *character id* on `CUser` (marriage does, at +0x1F34/+0x1F38) and
keeps the couple/friendship pair identity as *item SNs* instead.

## Block 1 — couple ring

v95 `CUserPool::OnCoupleRecordAdd` @0x94d600, symbolised
`OnCoupleRecordAdd(CUserPool *, __POSITION *liSN, CUser *pUser, int nItemID)`:

```c
      v9 = ZList<CUserPool::COUPLEENTRY>::AddTail(&this->m_lCouple);  /*0x94d6a3*/
      v9->dwCharacterID     = v8->m_dwCharacterId;                    /*0x94d6ae*/
      v9->dwPairCharacterID = p->m_dwCharacterId;                     /*0x94d6b6*/
      v9->liSN              = v8->m_liCoupleItemSN.QuadPart;          /*0x94d6bf*/
      v9->liPairSN.LowPart  = p->m_liCoupleItemSN.LowPart;            /*0x94d6d1*/
      v10 = nItemID;                                                  /*0x94d6da*/
      v9->liPairSN.HighPart = p->m_liCoupleItemSN.HighPart;           /*0x94d6de*/
      v9->nItemID           = v10;                                    /*0x94d6e1*/
      v9->nStatus           = 0;                                      /*0x94d6e4*/
```

This is decisive on its own. `COUPLEENTRY` has *both* a `dwPairCharacterID`
member and an `nItemID` member. `dwPairCharacterID` is filled from
`p->m_dwCharacterId` — a `CUser` the client found by **searching the user
pool** (@0x94d637 against the local user's `m_liPairItemSN`, @0x94d675 across
`m_lUserRemote`) — i.e. the client *derives* the pair character id locally and
never receives it. The wire value lands in `nItemID`.

The v83 twin `sub_972A5E` @0x972a5e stores the same way, offsets instead of
names:

```c
        result = sub_973B06(this + 14);         /*0x972af9*/   // AddTail(m_lCouple)
        *result   = *(v14 + 4520);              /*0x972b07*/   // dwCharacterID  <- CUser+0x11A8
        result[1] = a2[1130];                   /*0x972b12*/   // dwPairCharacterID <- CUser+0x11A8
        result[2] = *(v14 + 7952);              /*0x972b20*/   // liSN.Low   <- CUser+0x1F10
        result[3] = *(v11 + 4);                 /*0x972b26*/   // liSN.High
        result[4] = a2[1988];                   /*0x972b34*/   // liPairSN.Low  <- CUser+0x1F10 of pair
        result[5] = v13;                        /*0x972b3e*/   // liPairSN.High
        result[6] = a4;                         /*0x972b44*/   // nItemID  <- the wire Decode4
        result[7] = 0;                          /*0x972b3a*/   // nStatus
```

The independent producer-side confirmation is `CUserLocal::SetPairCharacterID`
@0x908680 (v95) — the *local* player's path into the same registrar, which
does not go through the wire at all:

```c
  while ( 1 ) {                                                 /*0x908712*/
    v6 = *&m_pStr[8 * *v5 + 14932];                             // equipped item in a ring body part
    if ( v6 ) {
      Data = TSecType<long>::GetData(v6 + 1);                   /*0x908725*/
      if ( Data / 100 == 11120 && Data != 1112000 )             /*0x908745*/  // couple-ring template range
        break;
    }
    if ( ++v5 >= g_anPetAbilBodyPart ) { ... }
  }
  ...
  v8 = TSecType<long>::GetData(v6 + 1);                         /*0x9087d1*/
  ...
  CUserPool::OnCoupleRecordAdd(..., &this->m_liCoupleItemSN, this, v8);  /*0x9087fb*/
```

The fourth argument is the value the client just range-checked as
`Data / 100 == 11120` — the couple-ring **item template id**. The wire
`Decode4` feeds the same parameter slot. A character id could not survive
that test.

The consumer is a WZ effect load, not a user-pool lookup:
`CUserPool::Update` @0x94c370 walks `m_lCouple` and calls
`CUser::SetCoupleItemEffect(long, CUser *, long)` @0x8f05d0 (xref @0x94c4f4).
`callees` on 0x8f05d0 reports `StringPool::GetInstance` / `GetStringW`,
`ZXString<wchar_t>::Format` @0x417d40 and
`CAnimationDisplayer::LoadLayer` @0x451b10 — a formatted WZ node path and a
layer load. There is no name resolve and no character lookup in it.

| field name | wire width | IDA address | what reads it |
|---|---|---|---|
| `bCouple` (flag) | 1 (`Decode1`) | v95 @0x9541b7 / v83 @0x98372b, `Init` @0x97fb28 | gates the block; cleared back to 0 at v95 @0x9542af if unset |
| `liCoupleItemSN` (own ring cash SN) | 8 (`DecodeBuffer`) | v95 @0x9541d2 → `CUser::m_liCoupleItemSN`; v83 @0x983745 → `CUser+0x1F10` | `OnCoupleRecordAdd` arg 2; stored as `COUPLEENTRY::liSN` @0x94d6bf |
| `liPairItemSN` (partner's ring cash SN) | 8 (`DecodeBuffer`) | v95 @0x9541e2 → `CUser::m_liPairItemSN`; v83 @0x983755 → `CUser+0x1F18` | the pool search key at @0x94d637 / @0x94d675 — this is what identifies the partner |
| **`nItemID` (ring item template id)** | **4 (`Decode4`)** | v95 @0x9541ef, v83 @0x983764, v83 `Init` @0x97fb5b | passed as `nItemID` to `OnCoupleRecordAdd`; stored `COUPLEENTRY::nItemID` @0x94d6e1; consumed by `CUser::SetCoupleItemEffect` @0x8f05d0 via `Format` + `CAnimationDisplayer::LoadLayer` (WZ node fetch) |

## Block 2 — friendship ring

Structurally identical. v95 `CUserPool::OnFriendRecordAdd` @0x94d700,
`OnFriendRecordAdd(CUserPool *, __POSITION *liSN, CUser *pUser, int nItemID)`:

```c
      v9 = ZList<CUserPool::FRIENDENTRY>::AddTail(&this->m_lFriend);  /*0x94d7a3*/
      v9->dwCharacterID     = v8->m_dwCharacterId;                    /*0x94d7ae*/
      v9->dwPairCharacterID = p->m_dwCharacterId;                     /*0x94d7b6*/
      v9->liSN              = v8->m_liFriendshipItemSN.QuadPart;      /*0x94d7bf*/
      ...
      v9->nItemID           = v10;                                    /*0x94d7e1*/
```

Pool search key is `m_liFriendshipPairItemSN` (@0x94d737 local, @0x94d775
remote). Producer-side twin is `CUserLocal::SetFriendPairCharacterID`
@0x908810 (xref into 0x94d700 @0x90899b). v83 twin is `sub_972BD9`, called at
@0x9837bc and @0x97fbab.

| field name | wire width | IDA address | what reads it |
|---|---|---|---|
| `bFriendship` (flag) | 1 (`Decode1`) | v95 @0x954202 / v83 @0x983778, `Init` @0x97fb6d | gates the block; cleared at v95 @0x9542cb if unset |
| `liFriendshipItemSN` | 8 (`DecodeBuffer`) | v95 @0x95421d → `CUser::m_liFriendshipItemSN`; v83 @0x983792 → `CUser+0x1F88` | `OnFriendRecordAdd` arg 2; `FRIENDENTRY::liSN` @0x94d7bf |
| `liFriendshipPairItemSN` | 8 (`DecodeBuffer`) | v95 @0x95422d → `CUser::m_liFriendshipPairItemSN`; v83 @0x9837a2 → `CUser+0x1F90` | pool search key @0x94d737 / @0x94d775 |
| **`nItemID` (ring item template id)** | **4 (`Decode4`)** | v95 @0x95423a, v83 @0x9837b1, v83 `Init` @0x97fba0 | `FRIENDENTRY::nItemID` @0x94d7e1 |

## Block 3 — marriage

v95 `CUserPool::OnMarriageRecordAdd` @0x94d800, symbolised
`OnMarriageRecordAdd(CUserPool *, unsigned int dwCharacterID, __POSITION *pUser, int nRingID)`:

```c
    if ( dwCharacterID != p->m_dwMarriagePairCharacterID )   /*0x94d824*/  // p = local user
    { ... p = <remote>; if ( dwCharacterID == p->m_dwMarriagePairCharacterID ) break; /*0x94d84a*/ }
    ...
      v7 = ZList<CUserPool::MARRIAGEENTRY>::AddTail(&this->m_lMarriage);  /*0x94d879*/
      v7->dwCharacterID     = pUser1->m_dwCharacterId;                    /*0x94d888*/
      v7->dwPairCharacterID = p->m_dwCharacterId;                         /*0x94d894*/
      v7->nWeddingRingID    = nRingID;                                    /*0x94d897*/
      v7->nStatus           = 0;                                          /*0x94d89a*/
```

The contested first parameter is compared against **another** user's
`m_dwMarriagePairCharacterID`. So it is a character id, and specifically the
*subject's own* character id: my `m_dwMarriageCharacterID` matches my
spouse's `m_dwMarriagePairCharacterID`.

`CUserLocal::SetMarriagePairCharacterID` @0x9089b0 nails the semantics from
the local (non-wire) side, reading the `GW_MarriageRecord` the client already
holds:

```c
  if ( *(m_pStr + 2061) )                                       /*0x908a9f*/   // gender
  {
    this->m_dwMarriageCharacterID = Next->dwBrideID;            /*0x908ab8*/
    dwGroomID = Next->dwGroomID;                                /*0x908abe*/
  }
  else
  {
    this->m_dwMarriageCharacterID = Next->dwGroomID;            /*0x908aaa*/
    dwGroomID = Next->dwBrideID;                                /*0x908ab0*/
  }
  this->m_dwMarriagePairCharacterID = dwGroomID;                /*0x908ac7*/
  CUserPool::OnMarriageRecordAdd(..., m_dwMarriageCharacterID, this, Next->nGroomItemID);  /*0x908ad9*/
```

Two things fall out. First: `m_dwMarriageCharacterID` is **this character's
own character id** (bride id if the local player is the bride, groom id
otherwise), and `m_dwMarriagePairCharacterID` is the spouse's. It is not a
marriage/wedding record id. Second: the third wire field is fed on this path
from `GW_MarriageRecord::nGroomItemID` — an item id by the client's own field
name, matching the `Data == 1112803 || 1112806 || 1112807 || 1112809` wedding
ring template check at @0x908a5f in the same function.

Consumer is again a WZ effect load, not a lookup: `CUserPool::Update`
@0x94c370 → `CUser::SetWeddingRingEffect(long, CUser *, long)` @0x8f18e0
(xref @0x94c89c), whose `callees` are `StringPool::GetStringW`,
`ZXString<wchar_t>::Format` @0x417d40, `CAnimationDisplayer::LoadLayer`
@0x451b10.

| field name | wire width | IDA address | what reads it |
|---|---|---|---|
| **`dwMarriageCharacterID` (this character's OWN character id)** | 4 (`Decode4`) | v95 @0x954263 → `CUser::m_dwMarriageCharacterID`; v83 @0x9837d7 → `CUser+0x1F34`; v83 `Init` @0x97fbc4 | `OnMarriageRecordAdd` arg 2; matched against a *different* user's `m_dwMarriagePairCharacterID` @0x94d824 / @0x94d84a; written from `GW_MarriageRecord::dwGroomID`/`dwBrideID` at @0x908aaa / @0x908ab8 |
| `dwMarriagePairCharacterID` (spouse's character id) | 4 (`Decode4`) | v95 @0x954270; v83 @0x9837e4 → `CUser+0x1F38`; v83 `Init` @0x97fbd1 | the search target at @0x94d824 / @0x94d84a; written from the other of groom/bride at @0x908ac7 |
| `nWeddingRingID` (ring item template id) | 4 (`Decode4`) | v95 @0x954276; v83 @0x9837ea → `CUser+0x1F3C`; v83 `Init` @0x97fbd7 | `MARRIAGEENTRY::nWeddingRingID` @0x94d897; fed locally from `GW_MarriageRecord::nGroomItemID` @0x908ad9; consumed by `CUser::SetWeddingRingEffect` @0x8f18e0 |

When the flag is 0 the client zeroes all three (`v95 @0x954297/@0x95429d/@0x9542a3`).

## Verdict

**Trailing 4-byte field (couple and friendship arms): it is the ring's item
template id, not a pair character id.** It is passed straight into the
`int nItemID` parameter of `CUserPool::On{Couple,Friend}RecordAdd` (v95
@0x94d600 / @0x94d700), stored in `COUPLEENTRY::nItemID` @0x94d6e1 /
`FRIENDENTRY::nItemID` @0x94d7e1, consumed by a WZ effect-layer load
(`SetCoupleItemEffect` @0x8f05d0), and never stored on `CUser` at all; the
same parameter is fed on the local path from a value the client range-checks
as `Data / 100 == 11120` (`SetPairCharacterID` @0x908745). The pair character
id is *derived* by the client from the two SNs via a user-pool search
(@0x94d637 / @0x94d675) and never crosses the wire in this block.

**Marriage arm first field: it is `dwMarriageCharacterID` — the subject
character's OWN character id — not a marriage/record id.** `design.md` §3.1's
`MarriageId` name is **refuted**; the offset it cites (`CUser+0x1F34`) is
correct (v83 @0x9837d7 / @0x97fbc4), only the name is wrong. Tasks 2 and 3
must name it after a character id, not after a marriage record. Note it is
*not* redundant with the enclosing character's id in every case only because
the client compares it against the *pair* field of other users; on the wire
for character C it equals C's own character id (@0x908aaa / @0x908ab8).

Both couple and friendship arms therefore carry: own ring cash SN (8),
partner ring cash SN (8), ring item template id (4). Neither carries a
character id.

## Defect in a checked-in export — do NOT hand-edit

`gms_v83.json`'s comment is the wrong one. `gms_jms_185.json`'s
`CUserRemote::Init` `itemId` comment is **right**.

Wrong entries (`docs/packets/ida-exports/`), all in
`functions["CUserRemote::OnAvatarModified"]`:

| Export | Comment as checked in | Correct name |
|---|---|---|
| `gms_v83.json` (@0x98367e) | `liCoupleItemSN (8 bytes) + liPairItemSN (8 bytes) + dwPairCharacterId (4 bytes)` | trailing 4 bytes are `nItemID` |
| `gms_v83.json` (@0x98367e) | `liFriendshipItemSN (8 bytes) + liFriendshipPairItemSN (8 bytes) + dwFriendCharacterId (4 bytes)` | trailing 4 bytes are `nItemID` |
| `gms_v87.json` (@0xa090f4) | same two comments | same |
| `gms_v95.json` (@0x954110) | same two comments | same |
| `gms_jms_185.json` (@0xa57221) | `pair characterId (per entry)` / `friendship pair characterId (per entry)` | per-entry trailing 4 bytes are the ring `itemId` |

Correct entries, kept as the reference wording:

- `gms_jms_185.json` `CUserRemote::Init` (@0xa52876): `couple-ring itemId (per entry)` and `friendship-ring itemId (per entry)` — **this is the accurate one**. The same file's `OnAvatarModified` entry contradicts it; `Init` wins.
- All four GMS/JMS `OnAvatarModified` entries already name the marriage triple `dwMarriageCharacterID` / `dwMarriagePairCharacterID` / `nWeddingRingID` — those three are correct as written. Only `design.md`'s `MarriageId` rename was wrong.

A second, smaller inaccuracy in the same GMS entries: they record the couple
and friendship bodies as a *single* `DecodeBuf` of 20 bytes. The client
actually issues three separate reads — `DecodeBuffer(8)`, `DecodeBuffer(8)`,
`Decode4` (v95 @0x9541d2 / @0x9541e2 / @0x9541ef). Byte-equivalent on the
wire, but the op list is not what the client does.

**These exports must not be hand-edited.** They are harvested artifacts
regenerated by `packet-audit export`; a hand edit is silently overwritten on
the next refresh. The corrections belong in the IDB comments that the
exporter harvests, and land in the JSON via a follow-up `packet-audit export`
refresh. Recorded here so the next task's field names do not inherit the
defect.
