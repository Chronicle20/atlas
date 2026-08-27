# `SelectedAL` / `SendCreateNewCharacter` field-meaning derivation

Read-only IDA derivation pass answering `open-selected-al-mapping.md`'s four
questions. No Go code was changed. All addresses/session ids below were
queried live against the adopted IDBs; nothing is carried over from memory of
general MapleStory conventions unless explicitly marked as such.

## Versions actually decompiled

| version | session_id | binary | used for |
|---|---|---|---|
| gms_v83 | `754107bf` | `MapleStory_dump.exe.i64` (`v83_Me`) | Q1/Q2/Q3/Q4 — full re-derivation this pass |
| gms_v95 | `ecc757f4` | `GMS_v95.0_U_DEVM.exe.i64` | Q1/Q2/Q3/Q4 — full re-derivation this pass, plus `AvatarLook` type layout |

`gms_v87`/`gms_v92` were **not** independently re-decompiled this pass beyond
what `derivation.md` §2.2/§2.3 already recorded (their `SendCreateNewCharacter`
encode order was already confirmed identical in shape to v95 by that earlier
pass — see `derivation.md` §2.5's cross-version table). `gms_v84` is
VERSION-ABSENT (`derivation.md` §2.0 — no `CUICharacterSaleDlg` code path
exists in that binary at all), so it is out of scope for all four questions.

`gms_v83` was already adopted (`idb_list` returned `session_id: 754107bf`,
`adopted: true`, `is_analyzing: false`) — no `idb_open` call was needed.

---

## Q1 — what are the four `GetSelectedAL(this, i)` values?

**Answer: they are WZ item template ids (or a component of one), not indices
into a dialog-local option table, and not the same thing as each other in
kind** — one of the four (`i=1`) is a "hair style" id-shaped value that a
*different* code path (not the wire path) later combines with `i=2`'s value
to form a real hair-equip template id. Confirmed identical on gms_v83 and
gms_v95; the two agree exactly, offset-for-offset.

### gms_v95 — `CUICharacterSaleDlg::GetSelectedAL(this, int nIdx)` (`0x778cf0`)

```c
int __thiscall CUICharacterSaleDlg::GetSelectedAL(CUICharacterSaleDlg *this, int nIdx)
{
  ...
  if ( m_nGender ) {
    if ( m_nGender == 1 )
      ZArray<CUICharacterSaleDlg::ASITEM>::operator=(&aItem, &this->m_aFemaleItem[nIdx]);
  } else {
    ZArray<CUICharacterSaleDlg::ASITEM>::operator=(&aItem, &this->m_aMaleItem[nIdx]);
  }
  nItemId = aItem.a->nItemId;
  ...
  return nItemId;
}
```

This is decisive: the return value is `ASITEM.nItemId` — a named struct field
holding an item id — taken from element **0** (the front) of a
per-gender, per-slot `ZArray<ASITEM>` (`m_aMaleItem[nIdx]` /
`m_aFemaleItem[nIdx]`). `nIdx` selects *which appearance category's array* to
read, not a raw value to decode; what's returned is a real item id already
pulled from WZ data by the dialog, sitting at the front of that category's
array (i.e., the currently-selected option in that category).

### gms_v83 — `sub_7D5C57(this, int a2)` (`0x7d5c57`, unnamed but wired
identically — confirmed by direct comparison against v95, not assumed)

```c
int __thiscall sub_7D5C57(_BYTE *this, int a2)
{
  ...
  v2 = this[504];                 // m_nGender
  if ( !v2 ) { v3 = &this[4*a2 + 508]; goto LABEL_5; }   // male base 508
  if ( v2 == 1 ) { v3 = &this[4*a2 + 544]; goto LABEL_5; } // female base 544
LABEL_5:
  sub_7D7ADA(&v6, v3);   // == ZArray<ASITEM>::operator=
  ...
  v4 = *v6;              // first field of ASITEM == nItemId
  sub_7D7B4A(&v6);       // == ZArray::RemoveAll (scratch cleanup)
  return v4;
}
```

`sub_7D7ADA` was decompiled and confirmed to be the same `ZArray` deep-copy
routine used in v95 (allocates `8*count+4` bytes, copies elements two dwords
at a time) — i.e. `sub_7D7ADA(&v6, v3)` is `ZArray<ASITEM>::operator=`, and
`v4 = *v6` dereferences the copied array's first element's first field, exactly
mirroring v95's `aItem.a->nItemId`. **v83 and v95 agree exactly**: gendered
base array, indexed by the same `i`, first element's item-id field returned.

### Which appearance slot does each `i` correspond to?

Both versions expose a second overload, `GetSelectedAL(this, AvatarLook*)`,
that calls the four-index form once per `i` and writes the results into named
`AvatarLook` struct fields. This is the direct evidence for slot identity.

gms_v95 (`0x778d80`):

```c
void __thiscall CUICharacterSaleDlg::GetSelectedAL(CUICharacterSaleDlg *this, AvatarLook *al)
{
  al->nFace = CUICharacterSaleDlg::GetSelectedAL(this, 0);
  SelectedAL = CUICharacterSaleDlg::GetSelectedAL(this, 1);
  al->anHairEquip[0] = CUICharacterSaleDlg::GetSelectedAL(this, 2) + 10 * (SelectedAL / 10);
  al->nSkin = CUICharacterSaleDlg::GetSelectedAL(this, 3);
}
```

gms_v83 (`sub_7D5C0B`, `0x7d5c0b`, the same pattern, raw offsets instead of
named fields — offsets independently confirmed against the `AvatarLook` type
below):

```c
int __thiscall sub_7D5C0B(_BYTE *this, int a2)
{
  *(a2 + 17) = sub_7D5C57(this, 0);              // nFace  (0x11)
  v3 = sub_7D5C57(this, 1);
  *(a2 + 25) = sub_7D5C57(this, 2) + 10*(v3/10);  // anHairEquip[0] (0x19)
  result = sub_7D5C57(this, 3);
  *(a2 + 13) = result;                            // nSkin  (0xd)
  return result;
}
```

`AvatarLook`'s type layout (queried from the gms_v95 IDB's local type
catalog, `type_inspect`):

```
AvatarLook, size 517
  0x0   ZRefCounted (base)
  0xc   nGender            unsigned __int8
  0xd   nSkin              int
  0x11  nFace              int
  0x15  nWeaponStickerID   int
  0x19  anHairEquip        int[60]
  0x109 anUnseenEquip      int[60]
  0x1f9 anPetID            int[3]
```

`0xd == 13`, `0x11 == 17`, `0x19 == 25` — the v83 raw offsets (`+13`, `+17`,
`+25`) land on exactly `nSkin`, `nFace`, `anHairEquip[0]` respectively. This
independently confirms the v83/v95 index mapping is identical, not merely
structurally similar.

**Conclusion — slot mapping (both versions, in agreement):**

| index `i` | value returned | appearance slot |
|---|---|---|
| 0 | `ASITEM.nItemId`, front of the gendered array for slot 0 | `nFace` — a face template id, used directly |
| 1 | `ASITEM.nItemId`, front of the gendered array for slot 1 | hair **style** component — only used via `id / 10 * 10` (its last digit is discarded) when building the real hair-equip id |
| 2 | `ASITEM.nItemId`, front of the gendered array for slot 2 | hair **color** component — only its last digit is used (`+ (i2 % 10)` in effect, since `i1/10*10 + i2` — see caveat below) to complete the hair-equip id |
| 3 | `ASITEM.nItemId`, front of the gendered array for slot 3 | `nSkin` — a skin id, used directly |

**Important caveat for Q1**: `GetSelectedAL(this, AvatarLook*)` (the
combining overload above) is used only for **local avatar preview**
(`SetAvatar`, see Q3) — it is not what goes on the wire. `SendCreateNewCharacter`
calls the **raw, single-int** `GetSelectedAL(this, i)` / `sub_7D5C57(this, i)`
directly in its `al[0..3]` encode loop (§2.1/§2.4 of `derivation.md`, and
independently re-confirmed by decompile this pass — see the `SendCreateNewCharacter`
bodies quoted under Q3). So **the wire carries the four raw, uncombined
values** — `al1` is the hair-style component and `al2` is the hair-color
component, *not yet combined* into a final hair-equip id. The server, not the
client, must do `hair = al2_last_digit + (al1 / 10) * 10` (or decide not to
trust `al2`/`al1` independently) if it wants a real hair-equip template id.
This is a **derived fact about the wire**, not a design recommendation.

---

## Q2 — if indices, where does the dialog build the table, and what would the server need?

**Answer: they are not raw indices into a table the server would need to
resolve — each `GetSelectedAL(this, i)` call already returns a real WZ item
id taken from the front of a per-gender, per-slot candidate array
(`m_aMaleItem[i]` / `m_aFemaleItem[i]`, each a `ZArray<ASITEM>`).** The
"index into a table" is `i` itself (0..3, the appearance-slot selector,
confirmed above), not the four wire values — the four **values** are already
resolved to item ids client-side before they are ever put on the wire.

What was found about where that per-slot candidate array is populated:

- `CUICharacterSaleDlg::LoadNewCharInfo` (gms_v95 `0x777790`, decompiled in
  full) builds a related but **different** structure: `m_lNewEquip`, a
  `ZList<NEWEQUIP>` populated by iterating a WZ property (string-pool ids
  `0x5F5`/`0x5F6` region — the two `GetObjectA` calls under `nGender = 0` and
  `nGender = 1` branches) over exactly **4** `nType` values (`for ( nType = 0;
  nType < 4; ++nType )`), each holding a variable number of `nItemId` /
  `sItemName` pairs read from sequentially-numbered WZ sub-nodes. The 4
  `nType` values line up in count with the 4 `al[]` slots, but this function
  builds `m_lNewEquip`, not `m_aMaleItem`/`m_aFemaleItem` directly — I did
  **not** find, within this pass's budget, the exact call site that
  transfers `m_lNewEquip` into the four-slot `m_aMaleItem[4]`/`m_aFemaleItem[4]`
  arrays `GetSelectedAL` and `ShiftNewCharEquip` actually read from. **This
  is an evidence gap, marked unresolved** — plausibly a constructor-time or
  `OnCreate`-time step not covered by the functions read this pass
  (`CUICharacterSaleDlg::CUICharacterSaleDlg` at `0x778270` was listed by
  `func_query` but not decompiled).
- `CUICharacterSaleDlg::ShiftNewCharEquip` (gms_v95 `0x77a9f0`, decompiled in
  full — the brief's named neighbour) confirms the array's *shape and
  mutation*, not its origin: it rotates `m_aMaleItem[nType]` /
  `m_aFemaleItem[nType]` (moving the front element to the back or vice versa)
  in response to UI "next/previous option" clicks, and each element carries
  `nItemId` + `sItemName` (`ASITEM`'s two members, confirmed by the
  `v14->nItemId = itema; ZXString<char>::operator=(&v14->sItemName, &s);`
  writes). This is the "selection" mechanism: `GetSelectedAL` always reads
  index 0 of whichever array, and `ShiftNewCharEquip` is what changes which
  `ASITEM` sits at index 0.

**What the server would need, given the above:** nothing, to resolve an `i`
to a template id — the values arriving over the wire already ARE the ids (or,
for the hair pair, the two components of one). What the server *would* need,
if it wanted to validate that a submitted `al[]` set is one the dialog could
actually have offered (rather than trusting the client), is the same WZ
per-gender/per-class candidate pools `LoadNewCharInfo` reads — which this
pass did not locate the WZ node path for. That validation is out of scope for
the four questions but is the natural next step if "validate against a known
option set" is ever wanted (right now `factory.Processor`'s note that
"look fields go to the factory unvalidated by the channel, by design" — cited
in the currently-landed doc comment — means this gap is not currently
load-bearing).

---

## Q3 — where do `top`/`bottom`/`shoes`/`weapon` come from?

**Answer: nowhere in this packet, and nowhere in any function this pass
decompiled that participates in building the wire payload.** They are not
sent by the dialog, and there is no evidence the dialog derives them from
`nCurrentClass` either — the dialog appears not to construct starting
equipment identifiers at all.

Evidence:

1. `derivation.md` §2.1–§2.5 (the earlier pass's cross-version encode-order
   sweep, already exhaustive across gms_v83/v87/v92/v95) records the complete
   field list after `nPOS`/`nItemID` as exactly: `sName, al[0..3], nGender,
   nCurrentClass, nSP, update_time` — no top/bottom/shoes/weapon field
   anywhere, on any in-scope version. This pass's own re-decompile of
   `SendCreateNewCharacter` on gms_v83 (`0x7d7960`) and confirmation of
   gms_v95's shape corroborates that field list verbatim; no additional
   `Encode4`/`Encode2` calls were found beyond what `derivation.md` already
   listed.
2. `AvatarLook` (the struct the dialog's own local-preview path populates,
   `type_inspect` above) has **no top/bottom/shoes/weapon-shaped fields at
   all** — appearance is `nGender`, `nSkin`, `nFace`, `nWeaponStickerID`
   (a *sticker* id, cosmetic overlay — not a weapon template id), and two
   60-slot equip arrays (`anHairEquip`, `anUnseenEquip`) of which only index
   `[0]` is ever written by `GetSelectedAL(this, AvatarLook*)` (Q1). No code
   path decompiled this pass writes to any other index of either array.
3. `CUICharacterSaleDlg::SetAvatar` (gms_v95 `0x77a3c0`, decompiled in full)
   is the only caller of the combining `GetSelectedAL(this, AvatarLook*)`
   overload; it `memset`s the entire `AvatarLook` to zero
   (`memset(&v21.nSkin, 0, 252)`) before populating `nFace`/hair/`nSkin`, then
   feeds it to `CAvatar::Init` purely for the in-dialog render — this is
   local UI preview, not wire construction, and it never touches equipment
   slots either.

**Conclusion:** the client sends no wire value for `top`/`bottom`/`shoes`/`weapon`
under any circumstance in this packet. Whatever the character's starting
equipment is, it is **not derivable from this packet's contents on the client
side** — it must be a server-chosen default (either from `nCurrentClass`, if
the server maintains its own class→starting-equip table, or from the
tenant's creation-template defaults `factory.Processor`'s own doc comment
already names). Nothing in the binary says which of those two the server
*should* do — that is a server-design decision the client-side evidence
cannot settle, and this pass does not attempt to settle it.

---

## Q4 — what do `nSP` and `nCurrentClass` mean, and are `subJobIndex`/stats server-chosen defaults?

### `nCurrentClass`

**Derived, with moderate confidence on exact semantics:** `nCurrentClass` is
the dialog's currently-selected **class family** (a small ordinal, not a full
job code), used to index a fixed-size per-class canvas array for rendering
the class-selection UI. Evidence, gms_v95 `CUICharacterSaleDlg::ShowClass`
(`0x7761b0`, decompiled in full):

```c
v11 = this->m_apCanvasClass.a[this->m_nCurrentClass].m_pInterface;      // male-side class icon
...
v24 = this->m_apCanvasClass.a[this->m_nCurrentClass + 5].m_pInterface;  // female-side class icon (+5 offset)
```

The `+5` offset between the two gendered halves of `m_apCanvasClass` implies
the array holds (at least) 5 entries per gender — i.e. **`nCurrentClass`
ranges over 5 class-family values** (consistent with, but not textually
confirmed as, Warrior/Magician/Bowman/Thief/Pirate — no string evidence for
those specific labels was read this pass, so the ORDERING/NAMES are
**unresolved**, only the **count** (5) and the **role** (a bounded ordinal
selecting a class-family icon) are derived).

`nCurrentClass` is also the field the wire directly carries verbatim
(`*(this+440)`/`this->m_nCurrentClass`, per `derivation.md` §2.1/§2.4) — so
the server receives this same small ordinal unmodified.

**Whether raw `nCurrentClass` equals atlas-character-factory's `jobIndex`
encoding is unresolved** — this pass confirms `nCurrentClass` is a
class-family selector with the right *shape* for `jobIndex`, but did not
verify that the client's 0..4 ordering matches whatever job-id scheme
`factory.Processor.SeedCharacter`'s `jobIndex` expects (that's a
server/constants-side fact, not something the client binary can answer).

### `nSP`

**Not resolved to a specific meaning from this pass's decompiles; treated as
unresolved.** What was found: `CUICharacterSaleDlg::LoadSPInfo` (gms_v95
`0x776d00`, decompiled in full) loads eleven localized description strings
per class family into `this->strSPWarrior[0..10]` (and begins a
`strSPMagician` array before the decompiler lost the tail to a `JUMPOUT`) —
this is UI **display text** for an SP-related tooltip/description, keyed by
class family, and does not itself compute or write `m_nSP`. No function
decompiled this pass contains a direct write to `m_nSP`/`this->m_nSP`; a
`search_text` for the field name timed out and was not retried within this
pass's evidence-gathering, so **the actual write site for `nSP` is
unresolved**, not merely uninvestigated-but-assumed. The presence of the
`strSPWarrior[11]`-per-class string table is suggestive (an 11-tier SP
description ladder that would only make sense keyed by a job's SP-per-level
progression) but is not, by itself, evidence of what numeric value
`m_nSP` holds at send time — that is a hypothesis to check, not a finding.

**Consequence for the mapping table:** `factory.Processor.SeedCharacter` has
**no `sp`/`nSP` parameter at all** — `nSP` is carried on the wire but has no
destination slot in the 18-argument signature under any interpretation. It is
either purely informational/legacy or something the factory would need a new
parameter to receive; this pass takes no position on which.

### Are `subJobIndex` and the four stats server-chosen defaults?

**Yes, by elimination — not by any direct evidence of server-side default
logic (which lives outside the client binary and outside this pass's scope),
but because the wire genuinely carries nothing for any of them.**
`derivation.md`'s exhaustive field-order sweep (§2.1–§2.5) and this pass's
independent re-confirmation on gms_v83/gms_v95 both show the complete
post-header field list is `sName, al[0..3], nGender, nCurrentClass, nSP,
update_time` — there is no `subJobIndex`, `strength`, `dexterity`,
`intelligence`, or `luck` field on the wire, on any in-scope version. Unlike
`top`/`bottom`/`shoes`/`weapon` (Q3), there is not even a plausible
client-side derivation path decompiled this pass that could produce them
(no stat-roll or subjob-selection logic was found in any function read). So
these five values **must** be server-chosen — the same shape of default
atlas-login's seed path already uses (1/50/5/0-style constants), per the
brief's framing — but this pass did not decompile atlas-login's or any
server-side default table, so it cannot say *what* the defaults should be,
only that the client supplies none.

---

## Recommended mapping table — `SeedCharacter`'s eighteen arguments

Confidence key: **derived** = read directly out of decompiled client code or
the repo, with no unverified interpretive step. **inferred** = the wire
source is identified but an interpretive/design step (a combination formula,
an enum-ordering assumption, or a "this must be a default" inference) stands
between the evidence and the value. **unresolved** = no wire source exists
and no binary evidence determines the value; a server-side decision is
required that this pass cannot make.

| # | `SeedCharacter` arg | source | confidence |
|---|---|---|---|
| 1 | `accountId` | session (`s.AccountId()`), never the packet — packet carries no account field at all (§ Gate 5 comment in `maple_life_create.go`, consistent with Q1–Q4's field list) | derived |
| 2 | `worldId` | session (`s.WorldId()`) | derived |
| 3 | `name` | wire `sName` | derived |
| 4 | `jobIndex uint32` | wire `nCurrentClass` — a 5-valued class-family ordinal (Q4); whether its raw value equals the factory's `jobIndex` encoding is not confirmed | inferred |
| 5 | `subJobIndex uint16` | no wire field exists (Q4) | unresolved — server default |
| 6 | `face uint32` | wire `al0` — `GetSelectedAL(this,0)`, used directly as `nFace` client-side (Q1) | derived |
| 7 | `hair uint32` | wire `al1` — the hair-style component; **not** the final hair-equip id as sent (Q1's caveat); needs `al2`'s last digit merged in (`al2%10 + (al1/10)*10`) to match what the client itself would render, or the factory must accept it as-is and NOT expect a final id | inferred |
| 8 | `color uint32` | wire `al2` — the hair-color component (only its last digit is meaningful per the client's own combination formula, Q1) | inferred |
| 9 | `skinColor uint32` | wire `al3` — `GetSelectedAL(this,3)`, used directly as `nSkin` client-side (Q1) | derived |
| 10 | `gender byte` | wire `nGender` | derived |
| 11 | `top uint32` | no wire field, no client-side derivation found (Q3) | unresolved — server default |
| 12 | `bottom uint32` | no wire field, no client-side derivation found (Q3) | unresolved — server default |
| 13 | `shoes uint32` | no wire field, no client-side derivation found (Q3) | unresolved — server default |
| 14 | `weapon uint32` | no wire field, no client-side derivation found (Q3) | unresolved — server default |
| 15 | `strength byte` | no wire field (Q4) | unresolved — server default |
| 16 | `dexterity byte` | no wire field (Q4) | unresolved — server default |
| 17 | `intelligence byte` | no wire field (Q4) | unresolved — server default |
| 18 | `luck byte` | no wire field (Q4) | unresolved — server default |

Orphan wire field with no destination: `nSP` (wire) has no corresponding
`SeedCharacter` parameter under any interpretation (Q4) — flagged, not
mapped.

---

## Is the currently-landed placeholder correct?

Per-field verdict against `maple_life_create.go`'s `seedCharacterFunc`
(`al0..al3 → face/hair/hairColor/skinColor` positionally, `currentClass →
jobIndex`, everything else `0`):

| placeholder mapping | verdict | why |
|---|---|---|
| `al0 → face` | **correct in kind, correct in slot** | `al0` genuinely is `nFace`, used directly — matches (Q1) |
| `al1 → hair` | **wrong in kind** | `al1` is the hair-style component only; the real hair-equip id needs `al2`'s digit merged in. Passing `al1` straight through as "hair" would send an incomplete/wrong-by-up-to-9 template id whenever the selected hair color isn't the array's implicit "0" default (Q1) |
| `al2 → hairColor` | **wrong in kind** | `al2` is not an independent hair-color id in the sense a typical `hairColor` parameter would expect (e.g. atlas-login's separate small color code); it is a raw `ASITEM.nItemId` value from the color-option array, and only its last digit is meaningful once merged with `al1`. Using it standalone as `hairColor` conflates two different value spaces (Q1) |
| `al3 → skinColor` | **plausibly correct in kind, correct in slot** | `al3` genuinely is `nSkin`, used directly — matches, modulo whether the factory's `skinColor` parameter expects exactly this value space (not independently verified) (Q1) |
| `nCurrentClass → jobIndex` | **right slot, ordering unverified** | `nCurrentClass` genuinely is a class-family selector with the right *role* for `jobIndex` (Q4), but this pass did not confirm the raw 0..4 values match the factory's job encoding — not disproven, but not confirmed either |
| `subJobIndex, top, bottom, shoes, weapon, four stats → 0` | **correct that no wire value exists for any of them** (Q3, Q4) | the placeholder's honesty about "the packet carries no wire value for any of those" is fully corroborated by this pass — no additional derivation was missed. Whether literal `0` (vs. a tenant-template default) is the *right* default is a policy question outside what the binary can answer |

**Overall: the placeholder is wrong-in-kind for two of its four `al[]`
mappings (`hair`, `hairColor`), right-in-kind for the other two (`face`,
`skinColor`), plausible-but-unconfirmed for `jobIndex`, and honestly
unresolved (not wrong) for everything it already zeroed.** It must not ship
as-is: sending `al1`/`al2` straight through as `hair`/`hairColor` will, for
any character whose selected hair color isn't the array's front-of-list
default, produce a hair id the factory's tenant-template validation is
likely to reject (a 400), or — worse — silently accept a *different*,
valid-but-wrong hairstyle's id.

---

## Follow-up — the numeric domain of each `al[i]` value on the wire

Raised by the coordinator after review, with repo-side evidence that
`atlas-character-factory`'s `Create` validates `face`/`hair`/`hairColor`/
`skinColor` against `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
`/characters/templates[0]`: `faces` are full template ids (`20000`-range),
`hairs` are style ids ending in `0`, `hairColors`/`skinColors` are single
digits. This section asks, per `al[i]`, what numeric domain the client
actually puts on the wire — **not** what domain the combining formula or the
server's validator is consistent with (both were explicitly ruled out as
evidence per the coordinator's instruction, since either is circular).

### New evidence this section is based on

`CUICharacterSaleDlg::InitNewCharEquip` (gms_v95, `0x77abe0`, decompiled in
full this round) is the function that populates `m_aMaleItem`/`m_aFemaleItem`
from `m_lNewEquip` — this was Q2's previously-unresolved population site
(now moot for the mapping question, but load-bearing evidence for this one).
Three findings from it:

1. **`m_aMaleItem`/`m_aFemaleItem` are 9-element arrays (indices 0..8), not
   4-element.** The loop that seeds random initial selections runs
   `while (v15 < 9)`, and index 8 is hardcoded to a `nItemId = 0` /
   `sItemName = "Male"` (male array) or `"Female"` (female array) sentinel —
   confirmed by the literal string refs `aMale`/`aFema` in the decompile
   output. Only indices populated from `m_lNewEquip` (i.e. `Next->nType`,
   which `LoadNewCharInfo` restricts to `0..3` — `derivation.md`-adjacent
   evidence already quoted under Q2) receive real content; indices 4..7 are
   never written by this function. `SendCreateNewCharacter`'s wire loop
   (`v2 <= 3`, Q1/Q3) only ever reads indices 0..3 — so even though the
   dialog's internal array has room for more equip categories, nothing past
   index 3 is ever read for the wire OR shown to be populated by this
   function. This is corroborating (not new) evidence for Q3's "no
   top/bottom/shoes/weapon on the wire" conclusion — it does not change that
   conclusion.
2. Each populated entry is built by:
   `v8->nItemId = Next->nItemId;` followed immediately by
   `CUICharacterSaleDlg::GetNewCharItemName(this, &v24, Next->nGender,
   Next->nType, Next->nItemId)` — i.e., for every option in every slot, the
   dialog also resolves a **display name** for that same `nItemId`, using
   `Next->nType` (0..3, the same index Q1 established as the appearance
   slot) as a selector for *how* to resolve the name.
3. `GetNewCharItemName` (gms_v95, `0x778980`, decompiled in full earlier this
   pass) branches on that same `nType`:

   ```c
   if ( nType._m_pStr <= 0 || nType._m_pStr > 3 )
   {
     // nType == 0 falls in HERE (0 <= 0), and nType > 3 (never true for 0..3)
     ItemName = CItemInfo::GetItemName(TSingleton<CItemInfo>::ms_pInstance._m_pStr, &v22, nItemID);
     ZXString<char>::operator=(result, ItemName);
   }
   else
   {
     // nType == 1, 2, or 3 falls in HERE
     ... StringPool::GetBSTR(..., 0x5F7 or 0x5F8) ...
     ... ZXString<unsigned short>::Format(&nType, <resolved bstr>, v6._m_pStr, nItemID) ...
     ... IWzResMan::GetObjectA(g_rm.m_pInterface, &Destination, v17.m_Data, ...) ...
     ZXString<char>::Assign<unsigned short>(result, bstrVal, -1);
   }
   ```

   **Slot 0 (`al0`/face) is the only slot whose `nItemId` is resolved through
   `CItemInfo::GetItemName`** — the same singleton item-info lookup every
   ordinary WZ item in the game uses, keyed by ordinary item template ids.
   Slots 1, 2, and 3 (`al1`/`al2`/`al3`) are resolved through a **different,
   dialog-specific path**: a WZ node built by string-formatting a
   resource-string template (ids `0x5F7`/`0x5F8`, not literal strings — this
   binary has no resource string table reachable from `search_text`/
   `find_regex` this pass; both attempts returned zero hits, since resource
   strings live in the `.rc`/string-table resource, not as inline
   ASCII/Unicode literals in `.text`/`.rdata`) with `nType` and `nItemID`
   substituted in, then fetched via `IWzResMan::GetObjectA` — not `CItemInfo`.

### Per-value domain verdict

1. **`al3` (`nSkin`) — UNRESOLVED.** The array holding slot-3's candidate
   `ASITEM.nItemId` values is populated from `m_lNewEquip`, which in turn is
   populated by `LoadNewCharInfo` from a WZ property whose literal path this
   pass could not resolve (the `StringPool` resource-string ids involved —
   `0x5F5`/`0x5F6` for the gender-group fetch in `LoadNewCharInfo`, `0x5F7`/
   `0x5F8` for the per-item name fetch in `GetNewCharItemName` — are numeric
   resource-table ids, not strings reachable via `search_text`/`find_regex`
   over the disassembly). The actual numeric values living at that WZ node
   are external asset data, not present in this EXE at all. Positive
   evidence found: slot 3 is resolved via the **same non-`CItemInfo` path**
   as slots 1 and 2 (not the ordinary item-catalog path slot 0 uses) — this
   distinguishes it from `face`'s domain but does not pin it to "single
   digit" versus some other id shape. **What would answer it:** a WZ dump of
   whatever node `IWzResMan::GetObjectA` resolves via the `0x5F7`/`0x5F8`
   resource-string templates (or, more directly, a live packet capture of an
   actual `SendCreateNewCharacter` submission and cross-checking the observed
   `al3` value against the tenant template's `skinColors` list).
2. **`al2` (hair-colour component) — UNRESOLVED**, same reasoning and same
   evidence as `al3`: resolved via the non-`CItemInfo` WZ path, not the
   ordinary item catalog, but the literal numeric domain of that WZ node was
   not reachable this pass. Per the coordinator's framing, the combining
   expression `al2 + 10*(al1/10)` is **not** used as evidence for or against
   "single digit" here — it is consistent with either interpretation and was
   deliberately excluded.
3. **`al1` (hair-style component) — UNRESOLVED**, same reasoning. No
   evidence was found (format string, range check, or WZ dump) confirming
   whether the array's `nItemId` values already end in `0` or whether the
   client's own `/10*10` normalization is load-bearing against non-zero-ending
   values actually present in that array. The normalization's *existence* in
   the code is not treated as evidence of the array's contents, per the
   coordinator's instruction against inferring domain from the combining
   formula.
4. **`al0` (face) — CONFIRMED as a full item template id, not an ordinal.**
   This is the one slot this pass found positive, non-circular evidence for:
   slot 0's `nItemId` is looked up through `CItemInfo::GetItemName`, the
   same singleton lookup any ordinary WZ item (keyed by a real item template
   id) goes through elsewhere in the client. Nothing about this routing
   decision depends on the combining formula or on what the server's
   validator wants — it is the client using its general-purpose item-name
   resolver on the value, which only makes sense if the value is a
   genuine item-catalog id. This is consistent with (and independently
   corroborates, from a different code path than Q1's) the repo's
   `faces: [20000, 20001, 20002]`-shaped domain.

### Version coverage for this section

All of the evidence above (`InitNewCharEquip`, `GetNewCharItemName`) was
read on **gms_v95 only** this round. `gms_v83`'s equivalent functions were
not located by name (`func_query` for `InitNewCharEquip`/`GetNewCharItemName`
returned no matches — expected, since v83's `CUICharacterSaleDlg` methods are
largely unnamed/raw-offset in that IDB, per Q1's `sub_7D5C57` precedent) and
were not tracked down by address within this pass's budget. **This
follow-up's conclusions are not cross-version-confirmed** — they rest on
gms_v95 alone. Given `derivation.md` §2.5 and this pass's own Q1 findings
already show v83 and v95 agree exactly on the `SendCreateNewCharacter`/
`GetSelectedAL` structure, cross-version agreement on `InitNewCharEquip`'s
routing logic is a reasonable expectation, not a confirmed fact.
