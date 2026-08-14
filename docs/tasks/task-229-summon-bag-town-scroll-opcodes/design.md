# Summoning Sack & Town Scroll Opcode Registration — Design

Task: task-229
Phase: 2 (design)
Created: 2026-08-14
Status: Draft — decisions D1…D5 need sign-off before `/plan-task`

---

## 1. What changed between the PRD and this design

The PRD framed this as "a registration and verification task, not a feature build…
No Go code is expected to change." The template-binding half of that holds. The
**verification half does not**: the two target ops cannot reach `✅` under the current
matrix mechanics, no matter how the templates are edited, because neither op is linked
to any codec and neither sender exists in any committed IDA export.

Four findings drive the whole design. Each is cited; none is inferred.

### F1 — `USE_SUMMON_BAG` and `USE_RETURN_SCROLL` have no packet linkage at all

Their rows in `docs/packets/audits/status.json` carry **no `packet` key** and every
one of the ten columns reads `incomplete / "no audit report"` — including the
"reference" columns gms_v83 and gms_v84, which the PRD assumed were already good:

```
USE_SUMMON_BAG    (no packet)  v48 n-a | v61…jms all "incomplete: no audit report"
USE_RETURN_SCROLL (no packet)  v48 n-a | v61…jms all "incomplete: no audit report"
```

Compare `USE_ITEM`, which does carry `packet: inventory/serverbound/InventoryItemUse`
and grades `verified` on six columns. The matrix resolves a row's packet in exactly two
ways (`tools/packet-audit/internal/matrix/grade.go`):

- **Path A — via an audit report.** `grade.go:273-280` joins the registry entry's
  primary `fname` to a per-packet report under `docs/packets/audits/<version>/*.json`,
  indexed by the report's `IDAName` (`build.go:99-111`); the packet id comes from
  `AtlasFile` + `WriterName` (`load.go:62-68`).
- **Path B — via the registry's own `packet:` field.** `opregistry.Entry.Packet`
  (`internal/opregistry/opregistry.go:43-50`) is an optional yaml key; `grade.go:126-133`
  uses it directly as the packet id when no report exists.

Neither resolves today, so grading short-circuits at `grade.go:219` — the literal string
`"no audit report"`. **Editing a template cannot change this.** Template routing is only
consulted *after* a packet is resolved (an unrouted-but-routed-elsewhere packet becomes
`conflict`, `grade.go:220-238`).

### F2 — neither sender function exists in any committed IDA export

The registry fnames `CWvsContext::SendMobSummonItemUseRequest` and
`CWvsContext::SendPortalScrollUseRequest` are **csv-import provenance**, not
`ida-discovered` (e.g. `docs/packets/registry/gms_v87.yaml:2364-2369`, `:2420-2425`).
Grepping all ten exports for those symbols returns zero hits on every version:

| export | `SendStatChangeItemUseRequest` | `SendMobSummonItemUseRequest` | `SendPortalScrollUseRequest` |
|---|---|---|---|
| gms_v48 | 1 | 0 | 0 |
| gms_v83 / v87 / v92 / v95 / jms_185 | 2 each | 0 | 0 |

Evidence freshness is computed by hashing the named function's decompile out of the
export (`internal/matrix/evidence_input.go:12-46`), so `evidence pin` will fail
*"not in export"* for both ops on every version until the senders are harvested and
spliced in (`docs/packets/audits/VERIFYING_A_PACKET.md:231-232`, and the
non-idempotent-export rule at `VERIFYING_A_PACKET.md:150-160`). The senders are almost
certainly **unnamed subs** in each IDB — the v48 probe below found exactly that.

### F3 — gms_92 is missing a fifth item-use handler the PRD did not list

Dumping the item-use handler bindings from every template confirms the PRD's §4.1 table
and adds one row it missed — `PetFoodHandle` (registry `PET_FOOD` = `0x53`,
`docs/packets/registry/gms_v92.yaml`) is **also unbound on gms_92**:

| template | ItemUse | SummonBag | TownScroll | Scroll | **PetFood** | ShopScanner | Lottery |
|---|---|---|---|---|---|---|---|
| gms_48 | `0x41` | `0x3B` | — | `0x42` | `0x3C` | — | — |
| gms_83 | `0x48` | `0x4B` | `0x55` | `0x56` | `0x4C` | `0x53` | `0x70` |
| gms_87 | `0x4B` | **—** | **—** | `0x59` | `0x4F` | — | `0x73` |
| gms_92 | **—** | **—** | **—** | **—** | **—** | `0x5A` | — |
| gms_95 | `0x4E` | **—** | **—** | `0x5D` | `0x52` | `0x5A` | `0x7C` |
| jms_185 | `0x40` | **—** | **—** | `0x4E` | `0x44` | — | `0x6B` |

gms_92 is not "missing two opcodes"; its entire item-use block below `0x54` is unrouted.
Pet food is core, its handler and codec already exist and are verified on every other
column (`PET_FOOD` row: `pet/serverbound/PetFood`, tier-1, `gms_v92: partial`), and the
binding is one line in the same file and the same sorted region as FR-2.3. See **D3**.

### F4 — resolved registry opcodes (FR-2.4's open question, and its neighbours)

Read out of `docs/packets/registry/*.yaml`, not inferred from a neighbouring column:

| version | `USE_ITEM` | `USE_SUMMON_BAG` | `USE_RETURN_SCROLL` | `USE_UPGRADE_SCROLL` | `PET_FOOD` |
|---|---|---|---|---|---|
| gms_v83 | `0x48` | `0x4B` | `0x55` | `0x56` | `0x4C` |
| gms_v87 | `0x4B` | `0x4E` | `0x58` | `0x59` | `0x4F` |
| gms_v92 | `0x4F` | `0x52` | `0x5C` | **`0x5D`** | `0x53` |
| gms_v95 | `0x4E` | `0x51` | `0x5C` | `0x5D` | `0x52` |
| jms_v185 | `0x40` | `0x43` | `0x4D` | `0x4E` | `0x44` |

**FR-2.4 is answered: gms_v92 `USE_UPGRADE_SCROLL` = 93 = `0x5D`**, fname
`CWvsContext::SendUpgradeItemUseRequest` (`docs/packets/registry/gms_v92.yaml:2632-2636`).
Every other opcode the PRD asserted is confirmed against the registry.

---

## 2. Architecture: how these two ops become verifiable

### The choice

**D1 — how do `USE_SUMMON_BAG` / `USE_RETURN_SCROLL` acquire a codec linkage?**

*Option A — alias the shared codec.* Add
`packet: inventory/serverbound/InventoryItemUse` to both ops in every registry. Cheap:
Path B resolves, and because evidence is keyed by `{packet, version}`, the **existing**
`InventoryItemUse` evidence records (which already exist for v48/v83/v84/v87/v95/jms)
would immediately grade several cells `verified` with no new work at all.

**Reject.** That evidence pins the decompile of `SendStatChangeItemUseRequest` — the
*potion* sender. Promoting the summon-bag and return-scroll cells off it asserts, with
zero decompiled proof, that three different client send sites encode identically on
every version. It is a manufactured `✅`, the precise inverse of the failure mode PRD
FR-6.3 warns about, and it would be indistinguishable in the matrix from real work.

*Option B — one wrapper struct per op (**recommended**).* This is the repo's documented
doctrine for exactly this situation, `docs/packets/audits/VERIFYING_A_PACKET.md:138-145`:

> **Shared-model ops.** When several ops share one decoder … create a thin per-op
> wrapper struct in `<pkg>/serverbound/` that embeds the shared model and delegates
> Encode/Decode… One wrapper per op = one packet/evidence per op. … The wrapper may be
> an uncalled audit codec; the production handler keeps decoding the shared model
> directly.

That last sentence is why this keeps the PRD's "no behaviour change" promise:
`services/atlas-channel/.../socket/handler/character_item_use.go` keeps calling
`inventory2.NewItemUse(...)` for all three handlers; the wrappers exist to carry an
fname, a packet id, a fixture and an evidence record. `LotteryItemUse`
(`libs/atlas-packet/inventory/serverbound/lottery_item_use.go`, task-131) is the
in-repo precedent: discrete struct, its own `packet-audit:fname` marker, one
`packet-audit:verify` line per version in `lottery_item_use_test.go`, one evidence
record per version.

*Option C — full audit reports (Path A).* Generate per-version reports for the two
senders by running the analyzer to a temp `-output` and copying the reports in
(`VERIFYING_A_PACKET.md:115-126`). Strictly more work than Option B for the same
outcome, and reports still require the harvest of F2. **Not recommended standalone**;
it remains available if a report happens to fall out of the harvest.

### Recommended shape (Option B, concretely)

Two new files under `libs/atlas-packet/inventory/serverbound/`:

| file | struct | fname marker | resulting packet id |
|---|---|---|---|
| `summon_bag_item_use.go` | `SummonBagItemUse` | `CWvsContext::SendMobSummonItemUseRequest` | `inventory/serverbound/InventorySummonBagItemUse` |
| `return_scroll_item_use.go` | `ReturnScrollItemUse` | `CWvsContext::SendPortalScrollUseRequest` | `inventory/serverbound/InventoryReturnScrollItemUse` |

Packet id = `qualifiedWriterName(pkg, name)` = TitleCase(pkg) + struct name
(`VERIFYING_A_PACKET.md:128-136`) — hence the `Inventory` prefix, matching
`InventoryLotteryItemUse`.

Each wrapper embeds `ItemUse` and delegates `Encode`/`Decode` (the analyzer recurses
into the embedded field), so there is exactly one definition of the wire body and no
possibility of the three drifting apart. Body, confirmed from
`libs/atlas-packet/inventory/serverbound/item_use.go:45-59`:
`Encode4(updateTime) + Encode2(slot) + Encode4(itemId)`.

Three supporting edits:

1. `tools/packet-audit/cmd/run.go` — two new `candidatesFromFName` cases mapping each
   fname to `{name: "SummonBagItemUse"|"ReturnScrollItemUse", pkg: "inventory",
   dir: DirServerbound}`, alongside the existing
   `case "CWvsContext::SendStatChangeItemUseRequest"` at `run.go:2198`. Required for a
   new serverbound op (`VERIFYING_A_PACKET.md:128-136`); v48's unnamed-sub case at
   `run.go:2188-2193` is the pattern to copy when a version's primary fname is a
   `sub_XXXXXX`.
2. `docs/packets/registry/<version>.yaml` — add `packet:` to both ops per version, so
   Path B resolves without needing a report.
3. `docs/packets/evidence/<version>/inventory.serverbound.Inventory{SummonBag,ReturnScroll}ItemUse.yaml`
   — one pinned record per op × version, each naming **that op's own** sender function
   and its own `decompile_sha256`.

Verification then follows the ordinary single-cell procedure
(`/verify-packet` → `docs/packets/audits/VERIFYING_A_PACKET.md`), unchanged.

### D2 — the IDA harvest is the real cost centre, and it is unavoidable

Per F2, for every op × version cell we intend to promote, the pass is:

1. Locate the send site in that version's IDB by the opcode-construction invariant —
   `COutPacket::COutPacket(&pkt, <opcode>)` — never by symbol name
   (`VERIFYING_A_PACKET.md:167-171`, "Distrust IDB function names"). Byte signature
   `6A <op> 8D 8D ?? ?? ?? ?? E8` / `6A <op> 8D 4D …` locates it
   (`VERIFYING_A_PACKET.md:173-178`).
2. **Name it in the IDB** while there (CLAUDE.md "No Deferring Producible Work"; an
   unnamed sub is a prerequisite we can produce, not a blocker).
3. Harvest with `-prior-export "" -pending <roster.md> -descent-depth 12` to a temp
   file and **splice only the needed entries** into the committed export. Never
   re-run a whole-export `packet-audit export` — it is not idempotent and drifts ~150
   unrelated function keys, degrading unrelated cells (`VERIFYING_A_PACKET.md:150-156`).
   Watch for the `COutPacket`-delegate harvest artifact (`:161-166`).
4. `packet-audit evidence pin`, add the `packet-audit:verify` marker line to the
   fixture test, regenerate the matrix.

All ten IDBs are currently loaded and adoptable (`idb_list`, 2026-08-14), including
`GMS_v48_1_DEVM.exe.i64`. Resolve sessions by **binary name** and pass the session id as
the `database` parameter — `select_instance`/port selection is dead. Batch one IDB at a
time; the harvest fan-out is per-version, so it parallelises across `packet-verifier`
agents only if each agent owns a distinct IDB.

### D3 — scope of the template edits

Recommended: bind everything a template is missing from the item-use block it already
half-carries, i.e. the PRD's FR-1…FR-4 **plus `PetFoodHandle` on gms_92** (F3). The
cost is one more entry in the same file and the same sorted region; the alternative is
shipping a v92 tenant where pet food is still silently dead and filing a follow-up for
one line. `ShopScannerItemUseHandle` (gms_87, jms_185) and `CharacterItemUseLotteryHandle`
(gms_92) remain **out of scope** as the PRD directed — they are different ops with
different codecs and their own verification cost, not part of the shared-`ItemUse`
family. gms_12 remains out of scope (no item-use handlers at all; a version pass).

Every new entry: `validator: "LoggedInValidator"`, `services: ["channel"]`, the registry
`fname`, inserted at its strictly-ascending sorted position
(`tools/template-opcode-order-guard.sh`), no duplicate `(name, opcode)` pair
(`tools/template-duplicate-binding-guard.sh`).

### D4 — how wide does verification go?

The PRD scopes verification to the five columns whose bindings change. But F1 means the
two op-rows are `incomplete` on **all** ten columns, including v83/v84 which the PRD
treats as the verified reference. Once the wrappers and the `candidatesFromFName` cases
exist, extending to the already-routed legacy columns costs one harvest + one pin per
op × version — the same unit of work, no new machinery.

- **Mandatory (PRD scope):** `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`, `gms_v48`.
- **Recommended extension:** `gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84` —
  10 further cells (2 ops × 5 versions). Leaving them `incomplete` while the newer
  columns read `✅` inverts the usual reading of the matrix and invites the "❌ means
  unverified shared codec" confusion the next reader will have to re-derive. v72 and v79
  already carry `ida-discovered` return-scroll fnames
  (`gms_v72.yaml:2407-2413` → `CWvsContext::SendReturnScrollUseRequest` @9531937;
  `gms_v79.yaml:2791-2797` @9866322), so those two are the cheapest of the five.

**Recommendation: take the extension.** It is the difference between two complete rows
and two rows that will need a second task. Flagged for sign-off because it roughly
doubles the harvest count (10 mandatory cells → 20).

### D5 — gms_48 (FR-5), with a first probe already done

The v48 matrix cells for both ops are `n-a` with `opcode: -1`, while
`template_gms_48_1.json` **routes `CharacterItemUseSummonBagHandle` at `0x3B`**. That is
a live contradiction: the matrix asserts the op is absent from the version while the
tenant config dispatches it. The n-a consistency gate (`cmd/na_consistency.go:130-222`,
wired into `matrix --check` at `cmd/matrix.go:278-290`) requires positive absence
evidence in `docs/packets/feature-na-evidence.yaml` for any `n-a` member whose family
has a verified sibling on that version — so a `USE_ITEM`-verified v48 with an
evidence-free `n-a` sibling is a `--check` failure waiting to happen.

**Design-phase probe (GMS_v48_1_DEVM.exe.i64, session `93cc947e`):** searching for the
opcode-construction invariant at `0x3B` found exactly one site, inside `sub_70DDAA`:

```
sub_4A2518(200, 0);                                  // CanSendExclRequest guard
COutPacket::COutPacket((COutPacket *)v18, 59);       // 59 == 0x3B
COutPacket::Encode4((COutPacket *)v18, v10);         // updateTime
COutPacket::Encode2((COutPacket *)v18, a2);          // slot
COutPacket::Encode4((COutPacket *)v18, a3);          // itemId
CClientSocket::SendPacket(...);                      // @0x70dec6 … 0x70defc
```

Its gates: `sub_713039(itemId)` → `CWvsContext::IsAbleToConsume` (a character-level
check, decompiled at `0x713039`), and a field-limit bit test
`(*((_DWORD *)get_field() + 58) >> 2) & 1` guarding a `CUtilDlg::Notice` (string 270).

**Established:** v48 opcode `0x3B` is a real send site carrying the exact three-field
`ItemUse` body under the standard excl-request guard, so the template's existing binding
is at minimum item-use-shaped and not a typo. **Not yet established:** that it is the
*mob-summon* sender specifically. The level gate and the field-limit bit are consistent
with a summon-bag send, but consistency is not identification. Implementation must close
it by structural comparison against v61's named twin
(`gms_v61.yaml:2294-2300`, `CWvsContext::SendMobSummonItemUseRequest` @8592515) before
FR-5.4 backfills the registry entry. Name `sub_70DDAA` in the IDB either way.

For FR-5.2/5.3 (return scroll at v48): enumerate item-use-shaped send sites across the
v48 IDB by the invariant, not by symbol — `gms_v72.yaml`/`gms_v79.yaml` both record that
the era's obvious `SendMapTransferItemUseRequest` symbol was a *mislabel* for the
return-scroll sender, so expect the same trap. Serverbound `0x38` and `0x3B` are the only
gaps in the v48 item-use neighbourhood (`0x39` `CANCEL_ITEM_EFFECT` … `0x43`
`DISTRIBUTE_AP`), but v48's serverbound table is **not** a shifted copy of v61's
(v48 puts `USE_ITEM` at `0x41`, *after* `USE_SKILL_BOOK` `0x40`, whereas v61+ put it
first) — so positional inference is invalid here and only the IDB decides. If absent,
record positive absence per `VERIFYING_A_PACKET.md`'s "Is this cell `n-a`?" bar,
including the mandatory sibling cross-check, into `docs/packets/feature-na-evidence.yaml`.

---

## 3. Component boundaries

| Unit | Responsibility | Depends on | Changes |
|---|---|---|---|
| `libs/atlas-packet/inventory/serverbound/item_use.go` | the one definition of the shared 3-field body | — | **none** (FR-6.3 / PRD §8) |
| `…/summon_bag_item_use.go`, `…/return_scroll_item_use.go` | per-op audit codecs: fname marker, packet id, evidence anchor | embeds `ItemUse` | new |
| `…/*_item_use_test.go` | byte fixtures + one `packet-audit:verify` line per verified version | `libs/atlas-packet/test` (`Variants` includes GMS v48/61/72/79/83/84/86/87/92/95 and JMS v185) | new |
| `tools/packet-audit/cmd/run.go` | fname → codec candidate mapping | — | 2 cases (+1 per unnamed-sub version, e.g. v48) |
| `docs/packets/registry/<v>.yaml` | op → opcode/fname/packet | — | `packet:` on 2 ops × N versions; v48 gains `USE_SUMMON_BAG` (+ `USE_RETURN_SCROLL` iff D5 finds it) |
| `docs/packets/ida-exports/<v>.json` | decompile corpus evidence hashes against | IDBs | **surgical splice only** — never regenerate |
| `docs/packets/evidence/<v>/*.yaml` | pinned per-op-per-version proof | export | new records |
| `services/atlas-configurations/seed-data/templates/*.json` | tenant socket routing | registry opcodes | 2–5 handler entries on 4–5 templates |
| `services/atlas-channel`, `services/atlas-consumables` | dispatch + consumption | — | **none** |

---

## 4. Expected matrix delta

| op | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms |
|---|---|---|---|---|---|---|---|---|---|---|
| `USE_SUMMON_BAG` (now: `n-a`, rest `incomplete`) | ✅ or evidenced `n-a` | ext | ext | ext | ext | ext | ✅ | ✅ | ✅ | ✅ |
| `USE_RETURN_SCROLL` (now: `n-a`, rest `incomplete`) | ✅ or evidenced `n-a` | ext | ext | ext | ext | ext | ✅ | ✅ | ✅ | ✅ |
| `USE_ITEM` (v92 `partial`) | — | — | — | — | — | — | — | ✅ | — | — |
| `USE_UPGRADE_SCROLL` (v92 `partial`) | — | — | — | — | — | — | — | ✅ | — | — |
| `PET_FOOD` (v92 `partial`, if D3 accepted) | — | — | — | — | — | — | — | ✅ | — | — |

`ext` = promoted only if D4's extension is accepted; otherwise those cells stay
`incomplete` and the two rows remain mixed. The three v92 `partial` rows are
`"tier-1: needs byte-fixture test to verify"` (`grade.go:251`) — a fixture + pinned
evidence for the v92 column promotes them; they need no new codec.

A cell that does not mechanically promote in `status.json` / `STATUS.md` is reported as
a failure, per PRD FR-6.2. No prose substitutes.

---

## 5. Risks

| Risk | Mitigation |
|---|---|
| Aliasing shortcut re-enters via a later "simplification" | Wrapper files carry a header comment stating they exist to give each op its own evidence key, citing task-229 and `VERIFYING_A_PACKET.md`'s shared-model rule |
| Whole-export regeneration degrades ~150 unrelated cells | Splice only; diff the export per commit and reject any hunk outside the spliced functions |
| The three senders are *not* byte-identical on some version | That is precisely what per-op evidence is for; a divergence found mid-task means the wrapper stops embedding and gets its own gated body, and the affected column is reported, not smoothed over |
| v48 `0x3B` turns out not to be the mob-summon sender | D5 treats it as unconfirmed; if the comparison fails, correct the template binding and say so explicitly (PRD FR-5.4) |
| n-a consistency gate trips on v48 once `USE_ITEM` siblings verify | Resolve v48 in this task (D5), never leave the cells silently `n-a` |
| Seed-template-only fix does not reach provisioned tenants | Unchanged from PRD §10; operational, called out in the PR body |

---

## 6. Verification plan

Per CLAUDE.md, from the worktree root:

1. `go test -race ./...` and `go vet ./...` in `libs/atlas-packet` and
   `tools/packet-audit` (both change).
2. `go run ./tools/packet-audit matrix` then `matrix --check` (includes the n-a
   consistency gate) — the authoritative promotion check.
3. `go run ./tools/packet-audit fname-doc --check`, `operations --check`,
   `doc-freshness --check`, `gate-check --check` — the CI gates in
   `.github/workflows/packet-matrix.yml`.
4. `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`,
   `tools/template-movement-types-guard.sh` (templates changed).
5. `tools/lint.sh --check` from the repo root.
6. `git diff --stat` must show **zero** changes to `template_gms_61/72/79/83/84_1.json`
   (unless D4's extension adds nothing to them — it adds no template entries, only
   evidence) and zero wire changes to `item_use.go`.
7. `superpowers:requesting-code-review` before the PR.

No service `go.mod` is touched, so no `docker buildx bake` target is implicated; confirm
at execution time before skipping it.

---

## 7. Open items for sign-off

- **D1** — per-op wrapper structs (recommended) vs registry aliasing. Accepting D1
  means the PRD's "no Go code changes" becomes "no Go *behaviour* changes": two new
  audit-only codecs plus two `candidatesFromFName` cases.
- **D3** — include `PetFoodHandle` on gms_92 (recommended).
- **D4** — extend verification to gms_v61/72/79/83/84 (recommended; ~doubles harvest).
- **D5** — v48 identification work is in-task, not deferred; the return-scroll answer
  may still land as an evidenced `n-a`.
