# Atlas Packet Library — Cross-Task Audit Ledger (GMS v95 baseline)

> **Last updated:** 2026-06-03
> **Maintenance:** To add a domain, append a row to §2 with task-id, file count, and
> verdict roll-up, then update the date. Recompute the §5 completeness statement if §3
> gains or loses a gap.
>
> **Note on counts:** sibling-task domains (027/028/065-068) are MERGED to `main`; their
> verdict counts here are read from `main:docs/packets/audits/gms_v95/SUMMARY.md`.
> task-069 (misc) counts are from this branch. task-069 forked from main @ `3bab0d885`
> (before the siblings merged), so the final PR integrates the misc reports into the
> merged tree — see post-phase-b.md §Integration.

## 1. Contributing tasks

| Task | Domain(s) | PR | Status |
|---|---|---|---|
| task-027 | login + audit pipeline | #438 | shipped |
| task-028 | character | #461 | shipped |
| task-065 | combat (monster, drop, reactor) | #476 | shipped |
| task-066 | social (buddy, messenger, note, chat, party, guild) | #609 | shipped |
| task-067 | commerce (inventory, pet, storage, cash, interaction) | #615 | shipped |
| task-068 | world (field, portal, npc) | #622 | shipped |
| task-069 | misc (account, fame, stat, ui, socket, channel, merchant/employee-shop, quest, tool) | (this branch) | in review |

## 2. Coverage matrix — `libs/atlas-packet/` (GMS v95)

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

## 3. Gaps

| Directory | Reason |
|---|---|
| (none) | Every `libs/atlas-packet/` directory is owned by a task or is a non-wire utility/test/model dir. |

Coverage is complete: the `find libs/atlas-packet -maxdepth 1 -type d` sweep maps every
directory to a contributing task (§2) or a non-wire-bound exclusion (model/, test/, tool/).

## 4. Audit-tool limitations (why some ❌ are not bugs)

The static analyzer (`tools/packet-audit`) flattens an encoder's `Encode` switch in source
order and diffs positionally against the IDA op list. It is unreliable for:
- **mask/mode-driven packets** (stat Changed, fame mode-dispatch) — atlas emits only set
  fields in config-mask order; the flattened all-branches list can't align positionally.
- **width-label false positives** — `WriteByteArray(4)` vs `Decode4`, `WriteLong` vs
  `EncodeBuffer(8)`, `WriteInt16+WriteShort(0)` vs `Decode4` are all equal on the wire.
- **early-return / exclusive branches** (login CharacterList) — over-counts conditional bytes.
- **struct-name collisions** in `locateAtlasFile` (ChannelChange → buddy file).

For these, the authoritative verdict is the per-report `## Manual analysis` section plus
byte-level wire-shape tests, not the auto-generated table.

## 5. Coverage-completeness statement

Coverage of `libs/atlas-packet/` for GMS v95 is **complete** as of task-069
(this branch). All wire-bound directories are audited across GMS v28/v83/v95 + JMS v185
(`docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/`); non-wire directories
(model/, test/, tool/) are documented exclusions.
