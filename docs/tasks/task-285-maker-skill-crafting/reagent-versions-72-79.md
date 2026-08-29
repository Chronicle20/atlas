# Reagent table: GMS 72.1 / 79.1 vs 83.1

Task 285. Resolves the open question left by Task 1: `MAKER_SKILL` is registered on
`gms_v72` and `gms_v79`, but `deploy/seed/gms/*/reagents/` only existed for `83_1`, so
gems would have contributed nothing on the two older versions.

**Verdict: the three versions' `Item.wz/Etc/0425.img` gem tables are identical.**
45 children on each, same ids, same stat, same value. Seed files were written for
`72_1` and `79_1` with content identical to `83_1`.

This document does not restate the loader derivation — see
[`reagent-derivation.md`](reagent-derivation.md) §1 for
`CItemMakerInfo::Load_GemEffect`, the node path, the 15-field order and the
first-non-zero rule. Only the **data** was in question here; the loader is
byte-identical between `gms_v72` and `gms_v83` (derivation §2).

---

## 1. Method

The canonical/seeded Postgres dumps were checked first and are a dead end for this
question: `atlas-canonical/baseline/regions/GMS/versions/72.1/documents.dump` carries
only `{price, unitPrice, slotMax, timeLimited, tradeBlock, only, tradeAvailable}` on
`etcs` documents — `incPAD` appears **0 times** in the entire 72.1 dump. The gem stat
fields are dropped by the etc reader. The raw `Item.wz` archive is the only source.

Extraction was done with the repo's own WZ parser, `libs/atlas-wz` (`wz.Open` →
`File.Root()` → `Directory("Etc")` → `Image("0425")` → `Image.Properties()`), driven by
a throwaway program kept outside the repo. It applies the client's algorithm literally:

1. enumerate every child of the `0425` image root;
2. `int(name)` for the reagent item id (zero-padded decimal, base 10);
3. descend into the child's `info` sub-property;
4. walk the 15 fields in derivation §1.3 order — `incPAD, incMAD, incACC, incEVA,
   incSpeed, incJump, incMaxHP, incMaxMP, incSTR, incINT, incLUK, incDEX, incReqLevel,
   randOption, randStat` — and take the **first non-zero**;
5. absent field ⇒ 0, matching the client's `get_int32(info[name], 0)` default.

The program also counted, per child, how many of the 15 fields were non-zero, so the
"first non-zero" rule could be checked for ambiguity rather than assumed.

Note on the parser: `libs/atlas-wz` exposes image names **without** the `.img` suffix
(`Etc/0425`, not `Etc/0425.img`). It opened all three archives directly with no version
key supplied — `File.detectVersion()` recovered the version itself
(`gameVersion=72 hash=1843` for the 72.1 archive).

### 1.1 Archive objects used

Pulled from MinIO (`mc` alias `atlaswz`), path
`atlaswz/atlas-wz/shared/regions/GMS/versions/<ver>/Item.wz`:

| Version | Size | ETag |
|---|---|---|
| 72.1 | 14 MiB | `7a62a8b72f4e74f3bba1c25daca483ce` |
| 79.1 | 16 MiB | `841189dc520cbf6d67eb6e427ed86d76` |
| 83.1 | 18 MiB | `9fb8c9b6a89a9d39018532b4f66cc3f2-2` |

The three archives are distinct objects (different ETags and sizes); the identity below
is a property of the `0425.img` node, not of the files.

### 1.2 Parser cross-check

The 83.1 archive was parsed by the same program and its 45 `(itemId, stat, value)` rows
were diffed against the 45 existing `deploy/seed/gms/83_1/reagents/reagent-*.json` files
— which were produced independently, from the extracted-XML dump, by the derivation
doc's pass. The diff is empty: **45/45 exact match on id, stat and value.** The raw-WZ
extraction path and the extracted-XML path agree, so the parse is not the weak link in
what follows.

---

## 2. Per-version sweep result

Identical for all three versions (not a spot check — every child of `0425.img` was
enumerated on each archive):

| Metric | 72.1 | 79.1 | 83.1 |
|---|---:|---:|---:|
| Children enumerated | 45 | 45 | 45 |
| Rows produced | 45 | 45 | 45 |
| Children missing an `info` node | 0 | 0 | 0 |
| Children with **exactly one** non-zero field | 45 | 45 | 45 |
| Children hitting the all-zero degenerate case | 0 | 0 | 0 |
| Children with >1 non-zero field (rule ambiguous) | 0 | 0 | 0 |

Every gem on every version declares exactly one of the 15 fields, so the first-non-zero
rule is never actually contended in any of the three archives. Each child's `info` also
carries `icon`, `iconRaw` and `price`, none of which the client's `RegisterGemEffect`
reads.

### 2.1 The table

All three versions produce the same 45 rows. They are the table already recorded in
[`reagent-derivation.md`](reagent-derivation.md) §3.1 — id range `4250000`–`4251402`,
all 15 stat names exercised at 3 tiers each (15 × 3 = 45), `incReqLevel` negative
(`-1 / -2 / -3`) for its three rows. It is not duplicated here; §3.1 is the single copy,
and it is now confirmed against three archives rather than one.

---

## 3. Three-way diff

Comparing `(node name, item id, stat, value)` for all 45 rows:

| Comparison | Result |
|---|---|
| 72.1 vs 83.1 | **identical** — `diff` empty, 45/45 |
| 79.1 vs 83.1 | **identical** — `diff` empty, 45/45 |
| 72.1 vs 79.1 | identical (transitively, and directly — both equal 83.1 row-for-row) |

Broken out against the report's requested categories:

- Rows present in 83.1 but **absent** in 72.1 / 79.1: **0**.
- Rows absent in 83.1 but present in 72.1 / 79.1: **0**.
- Rows whose **stat** differs: **0**.
- Rows whose **value** differs: **0**.

There is no version divergence in the maker gem table across GMS 72.1 → 83.1.

---

## 4. Seed files written

Because each client version needs its own seed rows regardless of content identity, the
45 rows were written for both older versions:

- `deploy/seed/gms/72_1/reagents/reagent-*.json` (45 files)
- `deploy/seed/gms/79_1/reagents/reagent-*.json` (45 files)

Schema, file naming and formatting mirror `deploy/seed/gms/83_1/reagents/` exactly
(JSON:API single-resource document, `type: "reagent"`, `id` the decimal gem item id,
attributes `stat` and `value`; two-space indent, LF line endings, trailing newline).
`diff -r` against the `83_1` directory is empty for both.

**The content is identical to `83_1`** — this is deliberate duplication of verified
data across per-version seed catalogs, not three independently-derived tables.
`CATALOG_REVISION` was not touched, matching the original `83_1` reagent seed commit,
which likewise did not bump it.

---

## 5. Unverified

- **Only 72.1, 79.1 and 83.1 were parsed.** The bucket also holds `Item.wz` for 48.1,
  61.1, 84.1, 87.1, 92.1 and 95.1. Nothing here says whether `0425.img` matches on
  those; they were out of scope because `MAKER_SKILL` is registered only on
  `gms_v72` / `gms_v79` / `gms_v83`. If maker is later enabled on another version,
  re-run this extraction against that archive rather than assuming identity.
- **The archives are the shared GMS reference set**, not a per-tenant dump. Whether any
  provisioned tenant ships a modified `0425.img` for 72.1 or 79.1 was not checked; the
  derivation doc §3 established that the three provisioned tenants agree for 83.1.
- **Value semantics are unchanged and inherit the derivation doc's limits.** Nothing in
  this pass re-examined how the client *applies* the pair — derivation §1.5 through
  §1.7 remain the authority, including the honest limit that no client-side consumer of
  the map was located, and that `randOption` / `randStat` are not simple `stat += value`
  deltas.
- **`gms_v79`'s loader was not decompiled.** Derivation §2 compared `gms_v72` against
  `gms_v83` and found the loader byte-identical in behaviour; 79.1 sits between them and
  its `Load_GemEffect` was assumed to follow suit rather than being read. Since the data
  is identical on all three, this only matters if the 79.1 client changed the field
  order or the selection rule — unverified either way.
