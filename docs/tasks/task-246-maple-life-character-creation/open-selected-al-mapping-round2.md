# ROUND 2: what the derivation resolved, and the two decisions it exposed

**Status:** the wire side is now DERIVED. Two **server-side design decisions**
remain, and neither is derivable from any binary — they are choices about how
Atlas fills fields the client never sends.
**Raised:** controller session 4, after the IDA derivation pass.
**Inputs:** `selected-al-derivation.md` (the IDA pass), plus repo-side facts
established in the controller session and quoted below.

## What is now settled (wire side)

From `selected-al-derivation.md`, derived on **gms_v83 and gms_v95 and shown to
agree offset-for-offset** (v83's raw `+13/+17/+25` land exactly on gms_v95's
typed `AvatarLook.nSkin` / `.nFace` / `.anHairEquip[0]`):

- The four `al[]` values are **not indices**. `GetSelectedAL(this,i)` returns
  `ASITEM.nItemId` — a real WZ item id — from element 0 of a per-gender
  `ZArray<ASITEM>[i]`. Q2 (the index-resolution question) is therefore **moot**,
  not unanswered: there is no index for the server to resolve.
- Slot identity, from the client's own combining overload
  `GetSelectedAL(this, AvatarLook*)`:
  `al0 → nFace`, `al1 → hair **style** component`, `al2 → hair **colour**
  component`, `al3 → nSkin`.
- The client combines `al1`/`al2` **only for local preview**:
  `anHairEquip[0] = al2 + 10 * (al1 / 10)`. `SendCreateNewCharacter` calls the
  raw single-int form and puts the **uncombined** values on the wire.
- The wire carries **nothing** for `subJobIndex`, `top`, `bottom`, `shoes`,
  `weapon`, or the four stats — on any in-scope version. Those must be
  server-chosen.
- `nSP` has **no destination** in `SeedCharacter`'s 18-argument signature under
  any interpretation. Its client-side write site is unresolved (a `search_text`
  timed out). This is **not chased further**: it cannot change the mapping,
  because there is no parameter for it to land in.

## Repo-side facts established this session (quoted, not remembered)

`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`,
`/characters/templates[0]` (jobIndex 0, gender 0):

```
'faces': [20000, 20001, 20002]
'hairs': [30030, 30020, 30000]          # style ids — every entry ends in 0
'hairColors': [0, 2, 3, 7]              # single digits
'skinColors': [0, 1, 2, 3]              # single digits
'tops': [1040002, 1040006, 1040010]
'bottoms': [1060002, 1060006]
'shoes': [1072001, 1072005, 1072037, 1072038]
'weapons': [1302000, 1322005, 1312004]
```

`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:99-152`
validates **every** one of these against the tenant's creation template and
returns an error on any miss — `validFace`, `validHair`, `validHairColor`,
`validSkinColor`, `validTop`, `validBottom`, `validShoes`, `validWeapon`.

`services/atlas-character-factory/atlas.com/character-factory/job/model.go`:

```go
func JobFromIndex(jobIndex uint32, subJobIndex uint32) job.Id {
	jobId := job.BeginnerId
	if jobIndex == 0 {            jobId = job.NoblesseId
	} else if jobIndex == 1 {     if subJobIndex == 0 { jobId = job.BeginnerId }
	                              else if subJobIndex == 1 { /* BladeRecruit TODO */ }
	} else if jobIndex == 2 {     jobId = job.LegendId
	} else if jobIndex == 3 {     jobId = job.EvanId }
	return jobId
}
```

## Consequence: the landed placeholder is confirmed broken, per field

| SeedCharacter arg | placeholder | verdict |
|---|---|---|
| `face` | `al0` | **correct** — `al0` is a face template id and `faces` holds template ids |
| `hair` | `al1` | **nearly correct** — `al1` is the style component; `hairs` entries all end in `0`, and the client itself normalises with `al1 / 10 * 10`, so the server should pass `(al1/10)*10` |
| `color` | `al2` raw | **wrong in kind** — `hairColors` are single digits `[0,2,3,7]`; `al2` is a full item id. `validHairColor` rejects it → 400 on every creation. Correct value is `al2 % 10` |
| `skinColor` | `al3` | **unconfirmed in domain** — `skinColors` are single digits `[0,1,2,3]`, but the derivation says every `al[i]` is an `ASITEM.nItemId`. Whether `al3` arrives as `0..3` or as a `200x`-style id was **not** established |
| `jobIndex` | `nCurrentClass` | **wrong in kind — see Decision 1** |
| `top`/`bottom`/`shoes`/`weapon` | `0` | **wrong** — `0` is in none of the template lists, so `validTop` rejects → 400. See Decision 2 |
| `subJobIndex`, 4 stats | `0` | plausible but unchosen — see Decision 2 |

The placeholder would 400 on **every** creation on at least two independent
counts (`color`, and all four equipment slots). No test catches this because
`seedCharacterFunc` is swapped in every test.

## Decision 1 — `nCurrentClass` and `jobIndex` are different domains

The client's `nCurrentClass` is a **class-family ordinal with 5 values**
(derived: `m_apCanvasClass.a[m_nCurrentClass]` and `[m_nCurrentClass + 5]` for
the two gendered halves — so ≥5 entries per gender; the *names* and *ordering*
were explicitly **not** established).

The factory's `jobIndex` is a **creation-track index with 4 values**:
0 = Noblesse, 1 = Beginner, 2 = Legend, 3 = Evan.

These are not the same quantity and no arithmetic connects them. Passing
`nCurrentClass` through as `jobIndex` is exactly the "wrong in kind" failure the
round-1 document warned about — it will select the wrong creation track, or
none, whenever `nCurrentClass > 3`.

**This is a design decision, not a derivation.** No binary can answer "what
creation track should a Maple Life character be born into", because that is a
statement about Atlas's job model, not about the client's wire format.

## Decision 2 — where the six unsent values come from

`subJobIndex`, `top`, `bottom`, `shoes`, `weapon`, and the four stats are not on
the wire on any version. The factory validates the four equipment ids against
the tenant template and rejects anything absent from the list, so they cannot be
left at `0`. Someone must choose them, and the choice is Atlas's, not the
client's.

## Explicitly NOT chased further

- **`nSP`'s write site** — unresolved in the IDA pass, and deliberately left
  there: `SeedCharacter` has no parameter it could fill, so resolving it cannot
  change any mapping. Reopen only if `nSP` is ever given a destination.
- **Q2's `m_aMaleItem`/`m_aFemaleItem` population site** (likely the
  undecompiled constructor `0x778270`) — moot, since the values turned out to be
  item ids rather than indices.
- **v87/v92** were not independently re-decompiled this pass; they rest on
  `derivation.md`'s prior field-order sweep. v84 remains struck (VERSION-ABSENT).

## CORRECTION (controller, session 4): `al2`'s domain is NOT open

An earlier revision of this document, and the IDA follow-up pass, both listed
`al2`'s numeric domain (bare digit vs full item id) as UNRESOLVED, on the
grounds that reasoning from the combining formula would be circular. **That was
too strict, and the item is now closed.**

The circular reasoning to avoid is: "the server's `validHairColor` wants a
single digit, therefore `al2` is a single digit." That argument is indeed
worthless. But this one is not:

```
anHairEquip[0] = al2 + 10 * (al1 / 10)
```

`anHairEquip[0]` is an entry in `AvatarLook.anHairEquip[60]` and must hold a
real hair **equip id** — 5-digit `3xxxx`, matching the tenant template's
`hairs: [30030, 30020, 30000]`. The right-hand term `10 * (al1 / 10)` already
supplies the full style id (`30030` for `al1 = 30030`). Therefore `al2` must lie
in `0..9`; any larger value overflows the sum past a valid hair id.

That constraint is derived from the **client's own arithmetic and from what the
destination field must hold** — not from what the server's validator happens to
accept. Code read out of the binary is evidence, the same as data read out of
the binary.

**Conclusion:** `al2` is the bare colour digit. `hairColor = al2 % 10` and
`hair = (al1 / 10) * 10` are correct under every reading, so neither needs a
WZ dump, a live capture, nor a user ruling.

`al3` (`nSkin`) is **not** closed by this argument — it is not an operand of any
combining expression, so nothing constrains its magnitude. It remains open, and
the tenant template's `skinColors: [0, 1, 2, 3]` is the only thing suggesting a
small ordinal. Do not close it by analogy with `al2`.
