# Task 262 — Provenance of the withdrawn oracle

Companion to `reference-fidelity.md`. That document adjudicates each of the 21
originally-flagged `Reactor.wz` items and concludes `INPUT-MISMATCH` for all of
them. This document is the durable record of *why*: the independent,
cross-archive evidence that the HaRepacker `.img.xml` dump referred to
throughout `prd.md` and `design.md` as `$WZ_REFERENCE` was never exported from
the archive supplied for this task (`$WZ_ARCHIVE`).

## The three independent 1620 counts (`Npc.wz`)

Three unrelated methods of counting `Npc.wz`'s images agree with each other and
disagree with the dump:

| Source | Method | Count |
|---|---|---|
| Our reader | `Directory.Images()` enumeration via `wzdiff` | **1620** |
| Raw byte walk | `declaredCount` read directly off the root directory header, no property decode at all | **1620** |
| HaRepacker | Run by the user directly against `$WZ_ARCHIVE`'s `Npc.wz` | **1620** |
| `$WZ_REFERENCE` dump | Count of `.img.xml` files under the tenant dump's `Npc.wz` directory | **6962** |

The first three are independent implementations reading the same bytes and
agreeing exactly. The fourth reads a different number from a directory that
claims to be the same archive. Whatever produced 6962 did not read
`$WZ_ARCHIVE`'s `Npc.wz`.

This is the same relationship `reference-fidelity.md` records for `Reactor.wz`
directly (`local image count: 6962 ours: 1620` — note the tool's `Npc.wz`
row uses the dump's own file count as "local" in that report's terminology).
For `Reactor.wz` itself the analogous three-way agreement is **419**
(our reader) vs. **421** (the dump); the two extra dump images, `9400300.img`
and `9400301.img`, are the subject of the Reactor-specific evidence in
`reference-fidelity.md`'s per-image sections.

## The dump: 6962 images in a single run

Every one of the 6962 `Npc.wz` `.img.xml` files under the tenant dump carries
the **same mtime**, `2026-02-12T09:03`. A single export run produced the whole
directory. Whatever `.wz` file that run read, it held roughly 4.3x as many NPC
images as the stock archive supplied for this task.

## The name-set diff

Comparing the dump's `.img.xml` names against the archive's own enumerated
image names, per archive:

| Archive | Dump-only names | Archive-only names |
|---|---|---|
| `Npc.wz` | **5342** | **0** |
| `Reactor.wz` | **2** (`9400300.img`, `9400301.img`) | **0** |

Zero archive-only names in both cases means the dump is a strict superset of
what the archive contains by name — every image our reader enumerates has a
same-named counterpart in the dump. That is consistent with the dump having
been produced from a *larger* or *later* WZ set that a stock 2010-02-17 GMS
v83 archive is a subset of, not with our reader silently dropping images (a
dropped image would show up as an archive-only name with no dump counterpart,
and there are none).

`reference-fidelity.md` additionally records, for `Npc.wz`, that the 5342
dump-only names sit in the `9901xxx` range — a range absent from the stock
archive entirely.

## The `9901000` check

Separately from the dump, the user ran HaRepacker directly against
`$WZ_ARCHIVE`'s `Npc.wz` and confirmed image `9901000` is present in the
archive itself. Our reader also enumerates it. So wherever the user's own
HaRepacker sighting of `9901000` came from, it was always consistent with our
reader reading `$WZ_ARCHIVE` — the discrepancy is entirely between the
*dump* and the archive, not between the reader and the archive.

## Reactor.wz cross-reference

`reference-fidelity.md` §"B. The reference dataset is a different, heavily
customised WZ set" carries the equivalent finding for `Reactor.wz`,
`Etc.wz`, and `Quest.wz`, plus the per-image byte adjudication (all 21 items
`INPUT-MISMATCH`) that established our reader is byte-faithful to
`$WZ_ARCHIVE` everywhere it was checked: 1136 type-9 sub-objects traced
across the **19 divergent images**, 0 instances of `actualEnd != endPos`
(Task R2's later whole-archive self-check corroborates this at 15428
sub-objects, 0 violations, 0 parse errors across **all 419 images in
`$WZ_ARCHIVE`**). That adjudication is the authority for
"our reader is exonerated"; this document is the authority for "the dump is
not this archive."

## What remains open — explicitly out of scope for task-262

Something exported 6962 `Npc.wz` images (and the corresponding `Reactor.wz`,
`Etc.wz` deltas) in a single run on 2026-02-12, and it was not
`$WZ_ARCHIVE`. **The source `.wz` file — or files — that dump was produced
from is unidentified.**

The user separately states the dump was produced by an earlier version of the
`atlas-data` service rather than by HaRepacker. That statement is recorded
here for completeness; it identifies a candidate *exporter*, not the
underlying WZ content the exporter read, and it has not been independently
confirmed with the same byte-level rigor as the count evidence above. It does
not change any conclusion in this document: whichever tool produced the
dump, it was not reading `$WZ_ARCHIVE`.

This is a data-provenance question, not a parser question, and it is **out of
scope for task-262**. Task-262 owns `libs/atlas-wz/wz`'s fidelity to whatever
archive it is given; it does not own identifying, locating, or re-exporting
against a different tenant's customised WZ set. If that identification is ever
wanted, it is a separate task with its own owner.

Do not overstate what is known here: we know the dump did not come from
`$WZ_ARCHIVE`. We do **not** know what archive it did come from, who produced
it, or whether it is even a single archive per domain rather than several.
Say exactly that, and no more.
