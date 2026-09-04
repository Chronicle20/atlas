# Reagent (gem) → `(stat, value)` derivation

Task 285, Task 17 Step 1. Resolves **OQ-5**: the gem/reagent stat mapping has no
`ItemMake.img` source, so it must come from the client's own loader plus the WZ archive.

**Status: complete.** The node the client reads *is* present in the local reference dump.
45 rows derived. No divergence between `gms_v72` and `gms_v83`.

---

## 1. The client loader

`CItemMakerInfo::Load_GemEffect`

| Version | IDB | Session | Address |
|---|---|---|---|
| `gms_v72` | `GMS_v72.1_U_DEVM.exe.i64` | `99e435d8` | `0x5a2cf5` |
| `gms_v83` | `MapleStory_dump.exe.i64` (v83_Me) | `754107bf` | `0x5e6f4c` |

Per-gem field reader (called once per child node):

| Version | Symbol | Address |
|---|---|---|
| `gms_v72` | `CItemMakerInfo::RegisterGemEffect` | `0x5a4145` |
| `gms_v83` | `sub_5E8573` (same function, unnamed) | `0x5e8573` |

### 1.1 The node path

Both versions load exactly one archive object, from the same string literal:

| Version | String symbol | Address | Value |
|---|---|---|---|
| `gms_v72` | `aItemEtc0425Img` | `0xa5bf6c` | `Item/Etc/0425.img` |
| `gms_v83` | `aItemEtc0425Img` | `0xaf67d8` | `Item/Etc/0425.img` |

**Node path: `Item.wz/Etc/0425.img`.**

In the repo's extracted-XML layout that is:

```
<ZIP_DIR>/<tenant-uuid>/GMS/83.1/Item.wz/Etc/0425.img.xml
```

### 1.2 The enumeration

`Load_GemEffect` is structurally identical in both binaries:

1. `IWzResMan::GetObjectA("Item/Etc/0425.img")` → `QueryInterface(IWzProperty)`.
   If the object is null the function returns `0` (no gem effects registered).
2. `IWzProperty::Get_NewEnum` → `IEnumVARIANT`, iterated one child at a time
   (`Next(1, &pvarg, &fetched)`).
3. Each child's **name** is converted to a number: `atoi(String)`
   (v72 `0x954720`, v83 `0xa6134a`). The names are zero-padded decimal item ids
   (`"04250000"`), and `atoi` parses base 10 — leading zeros are not octal — so
   `atoi("04250000") == 4250000`. **The key is the reagent item id.**
4. The child is re-fetched by name (`IWzProperty::Getitem`/`GetItem`) and then its
   `info` subnode is fetched (string `aInfo`: v72 `0xa5bdc0`, v83 `0xaf63a0`, value `"info"`).
5. `RegisterGemEffect(itemId, infoProperty)` is called with the `info` property.

> **Decompiler artifact, not a real narrowing.** Hex-Rays types the `atoi` result as
> `char` in both binaries (`char v15`/`char v12`). That is wrong. The mangled symbol
> `?RegisterGemEffect@CItemMakerInfo@@AAEXJV?$_com_ptr_t@...@@@Z` types the first
> parameter as `J` — MSVC's mangling for `long`. The key is a full signed 32-bit
> item id, not a byte. Do not truncate.

**Effective read path per gem: `Item.wz/Etc/0425.img/<itemIdName>/info/<field>`.**

### 1.3 The child field names, in the order the client reads them

`RegisterGemEffect` reads these 15 fields off the `info` property, in this exact order.
Each read is `get_int32(info["<name>"], 0)` — an explicit default of `0` when the field
is absent.

| Index | Field | v72 string addr | v83 string addr |
|---:|---|---|---|
| 0 | `incPAD` | `0xa5c0cc` | `0xaf6944` |
| 1 | `incMAD` | `0xa5c0c4` | `0xaf693c` |
| 2 | `incACC` | `0xa5c0bc` | `0xaf6934` |
| 3 | `incEVA` | `0xa5c0b4` | `0xaf692c` |
| 4 | `incSpeed` | `0xa5c0a8` | `0xaf6920` |
| 5 | `incJump` | `0xa5c0a0` | `0xaf6918` |
| 6 | `incMaxHP` | `0xa5c094` | `0xaf690c` |
| 7 | `incMaxMP` | `0xa5c088` | `0xaf6900` |
| 8 | `incSTR` | `0xa5c080` | `0xaf68f8` |
| 9 | `incINT` | `0xa5c078` | `0xaf68f0` |
| 10 | `incLUK` | `0xa5c070` | `0xaf68e8` |
| 11 | `incDEX` | `0xa5c068` | `0xaf68e0` |
| 12 | `incReqLevel` | `0xa5c05c` | `0xaf68d4` |
| 13 | `randOption` | `0xa5c050` | `0xaf68c8` |
| 14 | `randStat` | `0xa5c044` | `0xaf68bc` |

### 1.4 The selection rule — first non-zero wins

The 15 reads are a fall-through chain. After each `get_int32`, if the returned value is
non-zero the function short-circuits, recording that field's **index** as the stat and the
returned value as the value; otherwise it falls through to the next field. It never
accumulates and never records more than one field.

The stored pair is a two-`DWORD` struct:

```
v68[0] = <stat index 0..14>     // v72: v54[0] = v3
v68[1] = <the non-zero value>   // v72: v54[1] = v5
```

inserted into the per-`CItemMakerInfo` map keyed by item id
(v72 `sub_5A4CE7` @ `0x5a4ce7`; v83 `sub_5E9153` @ `0x5e9153`).

**Degenerate case:** if all 15 fields are absent or zero, the chain runs to the end and
still inserts `(0, 0)` — i.e. `incPAD` with value `0`. This does not occur in the local
dump (see §3): every one of the 45 children has exactly one non-zero field.

`price` is present on every `info` node in the archive and is **not** read by this
function. Nothing else on `info` (`icon`, `iconRaw`) is read either.

### 1.5 Value semantics

- **Integer width — signed 32-bit.** The reader is
  `?get_int32@@YAJAAVZtl_variant_t@@J@Z` (v83, named, `0x414d40`; v72 the same
  function unnamed at `0x41303a`). `J` in the MSVC mangling is `long` for both the
  return type and the default-value parameter — signed 32-bit in and out.
- **Signed.** Confirmed by data, not just by type: `incReqLevel` carries `-1`, `-2`,
  `-3` in the archive (§3, rows `04251200`–`04251202`).
- **Delta, not absolute.** Three independent grounds, all from the client/archive
  rather than memory:
  1. The 13 stat fields are prefixed `inc` — the same WZ-wide increment convention
     used by scroll `info` nodes.
  2. `incReqLevel` is negative in the archive. A negative *absolute* equip level
     requirement is not representable; a negative *delta* (reduce the requirement) is
     exactly what these gems are for.
  3. The magnitudes are small and tiered (`1/2/3`, `2/3/5`, `10/20/30`) — increments
     stacked per gem, not stat totals.
- **Note the `int16` narrowing.** The brief's `reagent.Model.Value() int16` is narrower
  than the client's `long`. Every derived value lies in `[-3, 30]`, so the narrowing is
  lossless for this dataset — but it is a *repo* choice, not the client's width. Record
  it as such.

### 1.6 `randOption` / `randStat` are not additive equip stats

Indices 13 and 14 are in the same chain and are stored identically, but they are the
equip random-option / random-stat variance keys, not stats to add to a single field.
The downstream stat-name enum must include them (the client can and does select them),
but Task 22/23's application logic must not treat them as a plain `stat += value`.
Flagged here because the enum and the applier share the name set.

### 1.7 Where the client *uses* the map — nothing found

Searched both binaries. The map's insert helper has exactly one caller
(`RegisterGemEffect`), which has exactly one caller (`Load_GemEffect`).
`CUIItemMaker::DrawGem` (v72 `0x75f426`) draws gem icons only and never touches the
map. **No read site for the gem-effect map was located in either binary.** The map is
loaded and, as far as these two versions go, write-only on the client — consistent with
the server owning the actual stat application. This is why §1.5's delta conclusion rests
on the archive's own shape rather than on an observed application site; stated plainly so
a downstream reader does not mistake it for a verified client behaviour.

---

## 2. Version cross-check: `gms_v72` vs `gms_v83`

**No divergence.** Compared item by item:

| Aspect | `gms_v72` @ `0x5a2cf5` | `gms_v83` @ `0x5e6f4c` | Same? |
|---|---|---|---|
| Node path string | `Item/Etc/0425.img` | `Item/Etc/0425.img` | yes |
| Subnode | `info` | `info` | yes |
| Key derivation | `atoi(childName)` | `atoi(childName)` | yes |
| Field count | 15 | 15 | yes |
| Field names | see §1.3 | identical, same addresses' strings | yes |
| Field order | `incPAD` … `randStat` | identical | yes |
| Stat indices | `0`…`14` in read order | `0`…`14` in read order | yes |
| Selection rule | first non-zero, short-circuit | first non-zero, short-circuit | yes |
| Value reader | `sub_41303A` (unnamed `get_int32`) | `get_int32` @ `0x414d40` | yes |
| Default on absent field | `0` | `0` | yes |
| Stored shape | `{statIndex, value}` DWORD pair | `{statIndex, value}` DWORD pair | yes |
| Null-object behaviour | return `0` | return `0` | yes |

The only differences are cosmetic decompiler output: v83 uses `_com_issue_error` where
v72's IDB shows the same COM error path as `IWzShape2D::Gety`, and v83 has more symbols
named. No behavioural delta.

---

## 3. The derived table

**Source archive:** `Item.wz/Etc/0425.img` from the local reference dump,
`<ZIP_DIR>/<tenant-uuid>/GMS/83.1/Item.wz/Etc/0425.img.xml`.

The file is present under all three provisioned tenants
(`ed65dc23-167a-4d27-85be-5a1778838a69`, `ec876921-c363-4cc6-9c51-5bb8d57f9553`,
`083839c6-c47c-42a6-9585-76492795d123`). Diffed all three: byte-identical for
`ed65dc23…` and `ec876921…`; `083839c6…` differs only in XML serialization
(`standalone="yes"`, self-closing tags, trailing newline) with **identical node,
attribute and value content**. One table, not three.

**Extraction method:** the client's algorithm from §1.4, applied literally — enumerate
`imgdir` children of the root, `int(name)` for the id, descend into `info`, walk the 15
fields in §1.3 order, take the first with a non-zero value.

**Sweep result — not a spot check:**

- 45 children enumerated, 45 rows produced, 0 skipped.
- 0 children hit the all-zero degenerate case of §1.4.
- 0 children carried more than one of the 15 fields — the "first non-zero" rule is never
  actually ambiguous in this dump; each gem declares exactly one field.
- All 15 stat names are exercised, 3 tiers each (15 × 3 = 45).

### 3.1 Rows

Every row's provenance is the same pair of facts, so they are stated once rather than
repeated 45 times:

- **Address:** `CItemMakerInfo::Load_GemEffect` — `gms_v72` `0x5a2cf5` / `gms_v83`
  `0x5e6f4c`, via `RegisterGemEffect` `0x5a4145` / `0x5e8573`.
- **Node path:** `Item.wz/Etc/0425.img/<node>/info/<stat>` — the `node` and `stat`
  columns below complete the path for each row.

| # | Node (`0425.img/<node>`) | `reagent_item_id` | `stat` | `value` | client stat index |
|---:|---|---:|---|---:|---:|
| 1 | `04250000` | 4250000 | `incPAD` | 1 | 0 |
| 2 | `04250001` | 4250001 | `incPAD` | 2 | 0 |
| 3 | `04250002` | 4250002 | `incPAD` | 3 | 0 |
| 4 | `04250100` | 4250100 | `incMAD` | 1 | 1 |
| 5 | `04250101` | 4250101 | `incMAD` | 2 | 1 |
| 6 | `04250102` | 4250102 | `incMAD` | 3 | 1 |
| 7 | `04250200` | 4250200 | `incACC` | 2 | 2 |
| 8 | `04250201` | 4250201 | `incACC` | 3 | 2 |
| 9 | `04250202` | 4250202 | `incACC` | 5 | 2 |
| 10 | `04250300` | 4250300 | `incEVA` | 2 | 3 |
| 11 | `04250301` | 4250301 | `incEVA` | 3 | 3 |
| 12 | `04250302` | 4250302 | `incEVA` | 5 | 3 |
| 13 | `04250400` | 4250400 | `incSpeed` | 2 | 4 |
| 14 | `04250401` | 4250401 | `incSpeed` | 3 | 4 |
| 15 | `04250402` | 4250402 | `incSpeed` | 5 | 4 |
| 16 | `04250500` | 4250500 | `incJump` | 1 | 5 |
| 17 | `04250501` | 4250501 | `incJump` | 2 | 5 |
| 18 | `04250502` | 4250502 | `incJump` | 3 | 5 |
| 19 | `04250600` | 4250600 | `incMaxHP` | 10 | 6 |
| 20 | `04250601` | 4250601 | `incMaxHP` | 20 | 6 |
| 21 | `04250602` | 4250602 | `incMaxHP` | 30 | 6 |
| 22 | `04250700` | 4250700 | `incMaxMP` | 10 | 7 |
| 23 | `04250701` | 4250701 | `incMaxMP` | 20 | 7 |
| 24 | `04250702` | 4250702 | `incMaxMP` | 30 | 7 |
| 25 | `04250800` | 4250800 | `incSTR` | 2 | 8 |
| 26 | `04250801` | 4250801 | `incSTR` | 3 | 8 |
| 27 | `04250802` | 4250802 | `incSTR` | 5 | 8 |
| 28 | `04250900` | 4250900 | `incINT` | 2 | 9 |
| 29 | `04250901` | 4250901 | `incINT` | 3 | 9 |
| 30 | `04250902` | 4250902 | `incINT` | 5 | 9 |
| 31 | `04251000` | 4251000 | `incLUK` | 2 | 10 |
| 32 | `04251001` | 4251001 | `incLUK` | 3 | 10 |
| 33 | `04251002` | 4251002 | `incLUK` | 5 | 10 |
| 34 | `04251100` | 4251100 | `incDEX` | 2 | 11 |
| 35 | `04251101` | 4251101 | `incDEX` | 3 | 11 |
| 36 | `04251102` | 4251102 | `incDEX` | 5 | 11 |
| 37 | `04251200` | 4251200 | `incReqLevel` | -1 | 12 |
| 38 | `04251201` | 4251201 | `incReqLevel` | -2 | 12 |
| 39 | `04251202` | 4251202 | `incReqLevel` | -3 | 12 |
| 40 | `04251300` | 4251300 | `randOption` | 1 | 13 |
| 41 | `04251301` | 4251301 | `randOption` | 2 | 13 |
| 42 | `04251302` | 4251302 | `randOption` | 3 | 13 |
| 43 | `04251400` | 4251400 | `randStat` | 2 | 14 |
| 44 | `04251401` | 4251401 | `randStat` | 3 | 14 |
| 45 | `04251402` | 4251402 | `randStat` | 5 | 14 |

### 3.2 The valid stat-name set

Exactly the 15 names of §1.3, in that order. This is the set Step 2's
`TestBuilderRejectsUnknownStat` enumerates and the builder validates against:

```
incPAD, incMAD, incACC, incEVA, incSpeed, incJump, incMaxHP, incMaxMP,
incSTR, incINT, incLUK, incDEX, incReqLevel, randOption, randStat
```

Spelled exactly as the client's string literals — case-sensitive, matching the WZ node
names verbatim. Do not normalize the casing; the archive and the client agree on this
spelling and the seed should too.

---

## 4. Notes for the implementer

- Seed all 45 rows of §3.1 verbatim. Nothing here is estimated; every id, stat and
  value is read out of `0425.img`.
- `value` is a **delta** applied to the crafted equip's corresponding stat
  (see §1.5 for the grounds, and §1.7 for the honest limit of that claim).
- `incReqLevel` is negative for all three of its rows. The column, the model accessor
  and any REST serialization must be signed.
- `randOption` and `randStat` are not `stat += value` — §1.6.
- The `(tenant_id, reagent_item_id)` unique index is safe: all 45 ids are distinct.
- Item-id range is `4250000`–`4251402`, all within `0425.img` (the Etc "gem" block).

---

## 5. Monster crystal level bands

Task 285, Task 18 Step 1. Derived from `CItemMakerInfo::Load_MonsterCrystalLevel`.

**Status: complete for the table; one semantic question left explicitly unverified
(§5.7).** The node the client reads *is* present in the local reference dump. 9 bands
derived. No behavioural divergence between `gms_v72` and `gms_v83`.

### 5.1 The client loader

| Version | IDB | Session | Address |
|---|---|---|---|
| `gms_v72` | `GMS_v72.1_U_DEVM.exe.i64` | `99e435d8` | `0x5a3033` |
| `gms_v83` | `MapleStory_dump.exe.i64` (v83_Me) | `754107bf` | `0x5e728a` |

Symbol in both: `?Load_MonsterCrystalLevel@CItemMakerInfo@@QAEHXZ`, size `0x3fd` in both
binaries. Called from `CItemMakerInfo::Load` (v72 `0x5a27cd`) as the third of four
loaders.

### 5.2 The node path

Both versions load exactly one archive object, from the same string literal:

| Version | String symbol | Address | Value |
|---|---|---|---|
| `gms_v72` | `aItemEtc0426Img` | `0xa5bf90` | `Item/Etc/0426.img` |
| `gms_v83` | `aItemEtc0426Img` | `0xaf67fc` | `Item/Etc/0426.img` |

**Node path: `Item.wz/Etc/0426.img`.** Loaded via
`IWzResMan::GetObjectA("Item/Etc/0426.img")` (v72 `0x5a307e`/`0x5a3095`,
v83 `0x5e72d5`/`0x5e72f3`).

Per-child read path, from the same two decompilations:

```
Item.wz/Etc/0426.img/<crystalItemIdName>/info/lvMin
Item.wz/Etc/0426.img/<crystalItemIdName>/info/lvMax
```

| Sub-node | v72 string addr | v83 string addr |
|---|---|---|
| `info` | `0xa5bdc0` (`aInfo`) | `0xaf63a0` (`aInfo`) |
| `lvMin` | `0xa5bf88` (`aLvmin`) | `0xaf67f4` (`aLvmin`) |
| `lvMax` | `0xa5bf80` (`aLvmax`) | `0xaf67ec` (`aLvmax`) |

**The table is WZ-driven, not hard-coded.** There is no immediate-operand band table
anywhere in either function; every boundary comes out of the archive.

### 5.3 The enumeration and the stored record

Structurally identical in both binaries:

1. `GetObjectA("Item/Etc/0426.img")` → `QueryInterface(IWzProperty)`. If the object is
   null the function returns `0` (v72 `0x5a3128`–`0x5a312d`; v83 `0x5e737f`–`0x5e7384`).
2. `IWzProperty::Get_NewEnum` → `IEnumVARIANT`, iterated one child at a time
   (`Next(1, &pvarg, &fetched)` — v72 `0x5a3182`, v83 `0x5e73d9`).
3. The child's **name** is converted with `atoi` (v72 `0x5a31c1`, v83 `0x5e7418`).
   Names are zero-padded decimal item ids (`"04260000"`); `atoi` is base 10, so
   `atoi("04260000") == 4260000`. **The crystal item id is the node name.**
4. The child is re-fetched by name, then its `info` subnode is fetched
   (v72 `0x5a31ee`/`0x5a3267`; v83 `0x5e744c`/`0x5e74c5`).
5. `lvMin` and `lvMax` are read with `get_int32` — v72 `sub_41303A` at call sites
   `0x5a3307` and `0x5a335b`; v83 `?get_int32@@YAJAAVZtl_variant_t@@J@Z` (`0x414d40`)
   at call sites `0x5e755e` and `0x5e75b2`. Signed 32-bit (`J` = MSVC `long`).
6. A **three-`DWORD`** record is appended to a vector member of `CItemMakerInfo`:

```
rec[0] = lvMin      // v72 0x5a3396 (*v34 = v51)      | v83 0x5e75ed (*v34 = v50)
rec[1] = lvMax      // v72 0x5a3397 (v34[1] = v52)    | v83 0x5e75ee (v34[1] = v51)
rec[2] = itemId     // v72 0x5a3398 (v34[2] = v53)    | v83 0x5e75ef (v34[2] = v52)
```

The container is at byte offset `0x60` of `CItemMakerInfo` in **both** versions:
v72 `sub_5A4E1A((char *)this + 96)` at `0x5a338c`; v83 `lea ecx, [eax+60h]` at
`0x5e75da` immediately before `call sub_5E9286` at `0x5e75e3`. (Hex-Rays prints the v83
call as `sub_5E9286(v55 + 24)` because it types `this` as `char *`; the disassembly at
`0x5e75da` is the ground truth — `0x60`, same as v72.) Both destructors confirm the slot:
v72 `sub_8F68B5` `this[24] → sub_5A4E58`, v83 `sub_9FA654` `this[24] → sub_5E92C4`.

**There is no count field.** The record is exactly `(lvMin, lvMax, itemId)`. Nothing in
`Load_MonsterCrystalLevel` reads a quantity, and `info` carries no such node beyond
`price`/`slotMax`/`icon`/`iconRaw`, none of which this function reads.

### 5.4 The derived table

**Source archive:** `Item.wz/Etc/0426.img` from the local reference dump,
`<ZIP_DIR>/<tenant-uuid>/GMS/83.1/Item.wz/Etc/0426.img.xml`. **Present locally.**

Found under all three provisioned tenants (`ed65dc23-167a-4d27-85be-5a1778838a69`,
`ec876921-c363-4cc6-9c51-5bb8d57f9553`, `083839c6-c47c-42a6-9585-76492795d123`).
Extracted all three with the client's algorithm from §5.3 (enumerate `imgdir` children,
`int(name)` for the id, descend into `info`, read `lvMin`/`lvMax`): **all three yield the
identical 9 rows below.** One table, not three.

Sweep, not a spot check: the root has exactly 9 `imgdir` children and all 9 appear here.

| # | Node (`0426.img/<node>`) | `crystal_item_id` | `lvMin` | `lvMax` | Loader evidence (v72 / v83) |
|---:|---|---:|---:|---:|---|
| 1 | `04260000` | 4260000 | 31 | 50 | `0x5a3307`/`0x5a335b` → `0x5a3396`-`98` / `0x5e755e`/`0x5e75b2` → `0x5e75ed`-`ef` |
| 2 | `04260001` | 4260001 | 51 | 60 | same loop body, same addresses |
| 3 | `04260002` | 4260002 | 61 | 70 | same loop body, same addresses |
| 4 | `04260003` | 4260003 | 71 | 80 | same loop body, same addresses |
| 5 | `04260004` | 4260004 | 81 | 90 | same loop body, same addresses |
| 6 | `04260005` | 4260005 | 91 | 100 | same loop body, same addresses |
| 7 | `04260006` | 4260006 | 101 | 110 | same loop body, same addresses |
| 8 | `04260007` | 4260007 | 111 | 120 | same loop body, same addresses |
| 9 | `04260008` | 4260008 | 121 | 200 | same loop body, same addresses |

> The loader is a single `while` loop over the enumerator, so every row comes from the
> *same* instruction addresses; the per-row differentiator is the archive node, cited in
> the "Node" column. This is the honest citation shape for a WZ-driven table — unlike
> §3.1's gem table it is not a fall-through chain with per-field addresses.

Bands are **inclusive on both ends** and **contiguous** from 31 to 200
(`lvMax(n) + 1 == lvMin(n+1)` for all eight adjacent pairs). No gaps, no overlaps.

### 5.5 Behaviour below the lowest band

**Lowest band starts at level 31** (`04260000`, `lvMin=31`). Levels `1`–`30` fall into
no band.

**What the client does: nothing — it never looks the table up.**

Grounds, swept in both binaries rather than inferred:

- The `CItemMakerInfo` singleton is `dword_AA8058` (v72, from
  `CWvsApp::InitializeGameData` `0x8f4e97`) and `dword_BF0EDC` (v83, from
  `CUIItemMaker::IsExistMakableItem` `0x822d0e`).
- `xrefs_to` the singleton returns **21 sites in v72** and **20 in v83** (v83's
  `InitializeGameData` is not identified as a function in that IDB, which accounts for
  the one-site difference).
- Every one of those sites is either the constructor, the destructor, or a
  `mov ecx, <singleton>` immediately followed by one of exactly **four** member-accessor
  thunks. Each thunk's `add ecx, N` fixes the member it reaches:

  | Member offset | v72 thunk | v83 thunk | Container |
  |---|---|---|---|
  | `+0x00` | `sub_5A46AC` | `sub_5E8ADA` | `ITEM_MAKE_INFO` map (recipes) |
  | `+0x18` | `sub_5A470E` (`add ecx, 18h` @ `0x5a4717`) | `sub_5E8B3C` (`add ecx, 18h` @ `0x5e8b45`) | `ZMap<long,ZList<long>,long>` |
  | `+0x30` | `sub_5A4728` (`add ecx, 30h` @ `0x5a4731`) | `sub_5E8B56` (`add ecx, 30h` @ `0x5e8b5f`) | `ZMap<long,ZList<long>,long>` |
  | `+0x74` | `sub_5A47D1` (`add ecx, 74h` @ `0x5a47da`) | `sub_5E8BFF` (`add ecx, 74h` @ `0x5e8c08`) | `ZMap<long,long,long>` (monster trophy) |

- `xrefs_to` those four thunks returns **16 call sites in v72 and 16 in v83**, in a 1:1
  correspondence (7 / 5 / 3 / 1 in both). **None of them is `+0x60`** — the
  monster-crystal band vector.
- `xrefs_to` the append helper is one site in each binary — the loader itself
  (v72 `sub_5A4E1A` ← `0x5a338c`; v83 `sub_5E9286` ← `0x5e75e3`).

**Conclusion: the band vector is write-only on the client in both `gms_v72` and
`gms_v83`.** It is filled at boot and never read. There is therefore **no client-side
fallback, clamp, or default to derive** for a level below 31 — the client defines no
behaviour at all, because it never asks the question. Exactly the same shape as the gem
map in §1.7.

**What this means for `TestCrystalForLevelBelowLowestBand`:** the correct assertion is
*no match* — a level below 31 resolves to no band and therefore to no crystal item id.
That is the only statement the client evidence supports. "Clamp to `4260000`" is **not**
grounded: nothing in either binary clamps, and the archive's `lvMin=31` is a hard lower
bound with no `0`-floored row. If the server is to clamp instead of returning
"no crystal", that is a **product decision for this task**, not a derived client
behaviour, and must be recorded as such — do not cite this document for it.

The symmetric case above the highest band (`lvMax=200`) is identical in kind: 201+ falls
in no band, with the same "no client behaviour" grounding.

### 5.6 Version cross-check: `gms_v72` vs `gms_v83`

**No divergence.** Compared item by item:

| Aspect | `gms_v72` @ `0x5a3033` | `gms_v83` @ `0x5e728a` | Same? |
|---|---|---|---|
| Function size | `0x3fd` | `0x3fd` | yes |
| Node path string | `Item/Etc/0426.img` | `Item/Etc/0426.img` | yes |
| Subnode | `info` | `info` | yes |
| Fields read | `lvMin`, `lvMax` (in that order) | `lvMin`, `lvMax` (in that order) | yes |
| Key derivation | `atoi(childName)` | `atoi(childName)` | yes |
| Value reader | `sub_41303A` (unnamed `get_int32`) | `get_int32` @ `0x414d40` | yes |
| Default on absent field | `0` (second arg to `get_int32`) | `0` | yes |
| Stored shape | `{lvMin, lvMax, itemId}` DWORD triple | `{lvMin, lvMax, itemId}` DWORD triple | yes |
| Container member offset | `+0x60` | `+0x60` | yes |
| Null-object behaviour | return `0` | return `0` | yes |
| Missing `info` node | skip child, continue loop (v72 `0x5a32c8`) | skip child, continue loop (v83 `0x5e751f`) | yes |
| Read sites for the table | none | none | yes |

The only differences are cosmetic decompiler output: v83 uses `_com_issue_error` where
v72's IDB renders the same COM error path as `IWzShape2D::Gety`, v83 has more symbols
named, and Hex-Rays mistypes `this` as `char *` in the v83 loader (§5.3). No behavioural
delta.

Because both versions read the same node from the same archive, and the archive file is
identical across all three local tenants, **the 9 rows of §5.4 hold for both versions.**

### 5.7 Unverified / unknown

Stated plainly rather than guessed:

1. **What "level" the band is compared against — UNVERIFIED.** The loader stores
   `lvMin`/`lvMax` and nothing in either binary ever reads them back (§5.5), so neither
   `gms_v72` nor `gms_v83` shows the comparison. Whether the operand is a crafted/
   disassembled equip's `reqLevel`, a monster's level, or a character's level **cannot be
   settled from these two IDBs.** The task brief frames it as an equip level requirement;
   that framing is *not* confirmed here. If it matters to the server's behaviour, it needs
   a source outside these binaries.
2. **Crystal count per craft — UNKNOWN.** `Load_MonsterCrystalLevel` yields no count
   (§5.3). No count node exists on `0426.img/<id>/info` in the local dump. Any per-craft
   quantity must come from `ItemMake.img` or the server, not from here.
3. **Whether some later client version reads this table** — not investigated. Only
   `gms_v72` and `gms_v83` were examined, per the task scope.
4. **Below-band clamp vs. no-match** — see §5.5. The *client* evidence supports
   "no match" only. A clamp is an unmade product decision, not a finding.
5. **`slotMax = 100` and `price = 1`** are present on all 9 `info` nodes in the archive
   but are **not read** by `Load_MonsterCrystalLevel`. Recorded for completeness; do not
   attribute maker semantics to them from this document.

### 5.8 Notes for the implementer

- Seed all 9 rows of §5.4 verbatim; every id and boundary is read out of `0426.img`.
- Bands are inclusive `[lvMin, lvMax]` and contiguous over `31..200`. A range check can
  rely on contiguity but should not rely on it silently — assert it in the seed test.
- `lvMin`/`lvMax` are signed 32-bit in the client. All derived values lie in `[31, 200]`,
  so any narrower repo type is lossless *for this dataset* — a repo choice, not the
  client's width. Same caveat as §1.5.
- Item ids `4260000`–`4260008` are all distinct; a `(tenant_id, crystal_item_id)` unique
  index is safe.
- There is no count column to model (§5.7 item 2).
