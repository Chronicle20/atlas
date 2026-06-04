# Atlas Packet Library — Cross-Task Audit Ledger (GMS v95 baseline)

> **Status: BASELINE COMPLETE — zero open actionable deferrals (task-080).**
> **Last updated:** 2026-06-04
> **Maintenance:** To add a domain, append a row to §2 with task-id, file count, and
> verdict roll-up, then update the date. The four-version baseline is closed; the
> disposition of record for every residual `❌`/`🔍` is the curated accepted-exclusions
> registry at `docs/packets/ida-exports/_pending.md` (§3/§5). The next version pass
> follows `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`.
>
> **Note on counts:** sibling-task domains (027/028/065-068) are MERGED to `main`; their
> verdict counts here are read from `main:docs/packets/audits/gms_v95/SUMMARY.md`.
> task-069 (misc) counts are from this branch. task-069 forked from main @ `3bab0d885`
> (before the siblings merged), so the final PR integrates the misc reports into the
> merged tree — see post-phase-b.md §Integration. task-080 then re-ran all four versions
> through the enhanced analyzer (§4) and curated the residue into `_pending.md`.

## 1. Contributing tasks

| Task | Domain(s) | PR | Status |
|---|---|---|---|
| task-027 | login + audit pipeline | #438 | shipped |
| task-028 | character | #461 | shipped |
| task-065 | combat (monster, drop, reactor) | #476 | shipped |
| task-066 | social (buddy, messenger, note, chat, party, guild) | #609 | shipped |
| task-067 | commerce (inventory, pet, storage, cash, interaction) | #615 | shipped |
| task-068 | world (field, portal, npc) | #622 | shipped |
| task-069 | misc (account, fame, stat, ui, socket, channel, merchant/employee-shop, quest, tool) | #657 | shipped |
| task-080 | packet-audit closeout (analyzer A1–A5, B1/B2/B5/B6 fixes, four-version curation) | (this branch) | shipped |

## 2. Coverage matrix — `libs/atlas-packet/` (four-version baseline)

### 2a. Four-version verdict roll-up (from the regenerated SUMMARYs)

Each SUMMARY row carries exactly one verdict glyph; ✅ = total rows − ❌ − 🔍.
Counts below are read from the post-task-080 (enhanced-analyzer) SUMMARYs at
`docs/packets/audits/<version>/SUMMARY.md`. Every residual `❌`/`🔍` is classified
into an accepted-exclusion category in the curated registry
`docs/packets/ida-exports/_pending.md` — **none is an open actionable deferral**
(the one tracked real wire bug, BuddyInvite, is a separate follow-up task, not a
`_pending` entry — see §3/§5).

| Version | Rows | ✅ | ❌ | 🔍 |
|---|---|---|---|---|
| gms_v83 | 253 | 169 | 80 | 4 |
| gms_v87 | 249 | 172 | 75 | 2 |
| gms_v95 | 348 | 263 | 77 | 8 |
| jms_v185 | 209 | 130 | 77 | 2 |
| **Total** | **1059** | **734** | **309** | **16** |

The ❌/🔍 residue is the expected analyzer floor after the A1–A5 de-noising
enhancements (§4): export read-order truncation, genuinely-opaque IDA types,
version-absent features, and representation-equivalence. It is fully enumerated and
blessed in `_pending.md` (TRUNCATION / OPAQUE / REPRESENTATION / VERSION-ABSENT
buckets), not carried as work.

### 2b. Per-directory matrix (GMS v95)

Verdict counts are ✅ correct / ❌ flagged. Per the audit-tool limitations (see §4),
many ❌ are static-analyzer artifacts on mask/mode/variable-length packets where the
real wire shape is correct (verified by byte-level tests + manual IDA in each report).

| Directory | Owning task | ✅ | ❌ | Notes |
|---|---|---|---|---|
| account/ | task-069 | 3 | 0 | AcceptTos audited under task-027 |
| buddy/ | task-066 | 2 | 4 | |
| cash/ | task-067 | 19 | 7 | |
| channel/ | task-069 | 1 | 0 | clientbound ChannelChange ❌ is a locateAtlasFile collision artifact (audits buddy file); packet verified correct (wire-shape test) |
| character/ | task-028 | 30 | 22 | |
| chat/ | task-066 | 1 | 1 | |
| drop/ | task-065 | 1 | 2 | |
| fame/ | task-069 | 3 | 1 | GiveResponse ❌ is a WriteInt16+WriteShort(0)==int32 artifact (wire correct) |
| field/ | task-068 | 13 | 3 | |
| guild/ | task-066 | 25 | 10 | BBS packets |
| interaction/ | task-067 | 26 | 4 | hire-merchant subset |
| inventory/ | task-067 | 10 | 1 | |
| login/ | task-027 | 26 | 1 | CharacterList ❌ = early-return over-count artifact |
| merchant/ | task-069 | 7 | 0 | employee-shop scope; hire-merchant → task-067; serverbound handler bare (deferred) |
| messenger/ | task-066 | 11 | 2 | |
| model/ | — | — | — | shared types; not wire-bound |
| monster/ | task-065 | 4 | 5 | |
| note/ | task-066 | 6 | 1 | |
| npc/ | task-068 | 29 | 4 | |
| party/ | task-066 | 3 | 12 | |
| pet/ | task-067 | 4 | 10 | |
| portal/ | task-068 | 1 | 0 | |
| quest/ | task-069 | 4 | 0 | ActionStart/ActionComplete/ActionRestoreLostItem deferred (need atlas-channel handler changes) |
| reactor/ | task-065 | 4 | 0 | |
| socket/ | task-069 | 5 | 0 | critical path; Hello/ChannelConnect ❌ are width-label artifacts (wire correct); JMS ChannelConnect gm field widened |
| stat/ | task-069 | 0 | 1 | Changed ❌ = mask-driven static artifact; 2 real v95 wire bugs FIXED (HP/MP int32, 2nd trailing byte) |
| storage/ | task-067 | 7 | 1 | |
| test/ | — | — | — | test harness; not wire-bound |
| tool/ | — | — | — | utility (uint128); no packets |
| ui/ | task-069 | 3 | 0 | |

Top-level files (`packet.go`, `resolve.go`, …) are library plumbing, not domains.

## 3. Gaps & open deferrals — **baseline complete, zero open actionable deferrals**

**Directory coverage.** Every `libs/atlas-packet/` directory is owned by a contributing
task (§2) or is a non-wire-bound exclusion (model/, test/, tool/). The
`find libs/atlas-packet -maxdepth 1 -type d` sweep leaves no unmapped directory.

| Directory | Reason |
|---|---|
| (none) | Every `libs/atlas-packet/` directory is owned by a task or is a non-wire utility/test/model dir. |

**Open deferrals.** **None.** Task-080 closed every actionable deferral from the prior
ledger (real wire bugs B1.1/B1.2/B1.4/B1.5, channel-handler logic B2.1/B2.3, JMS
cash-shop bodies B5.1, login trailer B6) and resolved every IDA-verification spike to a
verdict. The disposition of record for the four-version residual `❌`/`🔍` is the curated
accepted-exclusions registry **`docs/packets/ida-exports/_pending.md`** — a registry, not
a deferral ledger, holding **zero actionable items**. Every residual glyph is classified
there into a blessed permanent-exclusion category (TRUNCATION / OPAQUE / REPRESENTATION /
VERSION-ABSENT, plus OP/MODE-PREFIX and LOOP/EXCLUSIVE-BRANCH dispatcher artifacts) with
IDA evidence.

**One tracked follow-up (NOT a `_pending` deferral).** A single genuine buddy-domain wire
bug was surfaced rather than blessed: **`BuddyInvite`** (`buddy/clientbound/invite.go`,
`CWvsContext::OnFriendResult#Invite`, GMS v95 `@0xa12630`; `🔍` all four versions) — the
client reads two extra `Decode4` (`v25`/`v26`) between `originatorName` and the `GW_Friend`
insert plus a trailing `inShop` byte that Atlas does not write. task-080 did not own the
buddy domain and did not fix it; it is registered as a **dedicated follow-up task** (see
`_pending.md` §9), not parked here as accepted.

## 4. Audit-tool limitations (why some ❌ are not bugs)

The static analyzer (`tools/packet-audit`) flattens an encoder's `Encode` switch in source
order and diffs positionally against the IDA op list. task-080 added five de-noising
enhancements (§4.7, the de-noising baseline) so a clean re-run reads a trustworthy signal:

- **A1 width-equivalence** — `WriteByteArray(N)`/`WriteLong`/`WriteInt16+WriteShort(0)` now
  match a same-width `DecodeBuf`/`EncodeBuffer`; the width-label false-positive class is
  suppressed.
- **A2 name-qualification** — `candidatesFromFName` qualifies colliding struct names, so the
  `locateAtlasFile` collision class (e.g. `ChannelChange` → buddy file) no longer mis-audits.
- **A3 sub-struct / opaque descent** — the walker descends self-describing sub-structs and
  flags only genuinely-opaque residue (register-boundary types).
- **A4 early-return modeling** — exclusive `if/else` and early-`return` guards are no longer
  double-counted (verified, no analyzer change needed beyond A1–A3 coverage).
- **A5 region-dispatch helper descent** — the analyzer descends into same-receiver
  `m.encodeJMS(w)` / `m.encodeGMS(t,w)` helpers, so region-dispatched packets (B1.5, B5.1)
  analyze correctly instead of reporting an empty body.

Even after A1–A5 the analyzer **cannot** resolve two residue classes by construction; these
are expected, not bugs:
- **export read-order truncation** — the IDA-export JSON ends before a real Atlas trailing
  field, producing phantom `extra`/`short` rows (TRUNCATION bucket).
- **genuinely-opaque IDA types** — a single `DecodeBuf` token or a struct with no
  decomposable layout (mob body, AvatarLook, `model.Asset`, `GW_ItemSlotBase`); the analyzer
  stops at the register boundary (OPAQUE bucket).

It also remains imprecise for **mask/mode-driven packets** (stat Changed, fame mode-dispatch)
where atlas emits only set fields in config-mask order. For all of these, the authoritative
verdict is the per-report `## Manual analysis` section plus byte-level wire-shape tests, and
the residue is enumerated and blessed in `_pending.md` (§3/§5).

## 5. Coverage-completeness statement — baseline complete

The four-version packet-audit baseline is **complete with zero open actionable deferrals**
as of task-080 (this branch). Concretely:

1. **Coverage.** Every wire-bound `libs/atlas-packet/` directory is audited across the
   four-version baseline GMS v83 / v87 / v95 + JMS v185
   (`docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/`); non-wire directories
   (model/, test/, tool/) are documented exclusions.
2. **Deferrals.** Zero open actionable deferrals remain. Every residual `❌`/`🔍` in the
   regenerated SUMMARYs (§2a roll-up: 309 ❌ / 16 🔍 across 1059 rows) is classified into a
   blessed permanent-exclusion category in **`docs/packets/ida-exports/_pending.md`**, the
   disposition of record. That registry holds zero actionable items.
3. **Follow-up.** Exactly one genuine wire bug — **BuddyInvite** — was surfaced as a
   dedicated follow-up task rather than blessed (§3, `_pending.md` §9). It is the only
   tracked piece of remaining wire work, and it is registered as a separate task, not a
   `_pending` deferral.

The reusable playbook for the next version pass (where IDBs go, the corrected
`packet-audit` invocation, the gate / region-dispatch conventions, and how to tell expected
residue from a new finding) is **`docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`**.
