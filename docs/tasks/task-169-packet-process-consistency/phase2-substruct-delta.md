# Task 169 — T2.1 sub-struct n-a reclassification (AC-3 documented delta)

FR-4.1 threads a per-version `Unimplemented[version]set(packetID)` (derived from
each version's `docs/packets/audits/<ver>/_unimplemented.json`) into
`matrix.Build`. A sub-struct cell whose `(packetID, version)` is dispositioned as
version-absent now grades `n-a` (⬜, `StateNA`) instead of `incomplete` (❌).
This is the ONLY count-moving change in Phase 2.

## Matching rule (how `_unimplemented.json` names sub-structs)

`_unimplemented.json` entries are heterogeneous. A sub-struct disposition is
recognized ONLY when the entry either:

1. carries an explicit `packet` path (names the sub-struct directly — e.g. v48
   `{"fname":"CScriptMan::OnSayImage","packet":"npc/clientbound/NpcSayImageConversationDetail"}`), or
2. has a **suffix-qualified** `fname` containing `#` (e.g. v79
   `"CScriptMan::OnAskPet#AskPet"`), resolved to a packet ID through the global
   IDAName→packetID index built from all reports.

A **bare base fname** entry (e.g. `"CLogin::OnCheckPasswordResult"` + numeric
`case`) is deliberately NOT resolved: it disposes an unbuilt dispatcher *arm* by
`(fname, case)`, and its base name collides with the implemented sibling
struct's IDAName (`login/clientbound/AuthSuccess`). Matching it would wrongly
downgrade a built cell to n-a. Those entries remain the concern of the validate
bijection check, not the matrix.

## Per-version count delta (before → after)

Format: | ✅ | 🧩 | 🟡 | ❌ | ⬜ | 🟥 | verified% |

| version | ❌ before | ❌ after | ⬜ before | ⬜ after | verified% before | verified% after |
|---------|-----------|----------|-----------|---------|------------------|-----------------|
| v48     | 163       | 156 (−7) | 627       | 634 (+7)| 50.2%            | 51.2%           |
| v79     | 217       | 215 (−2) | 417       | 419 (+2)| 46.6%            | 46.8%           |
| v61, v72, v83, v84, v87, v95, JMS185 | — unchanged — | | | | | |

Total: **9 sub-struct cells reclassified `incomplete` → `n-a`** (v48: 7, v79: 2).
`✅ / 🧩 / 🟡 / 🟥` counts are byte-identical to `baseline-counts.md` in every
version. `verified%` rises only because its denominator (non-n-a cells) shrinks.

## The 9 reclassified cells

v48 (7):
- interaction/serverbound/InteractionOperationMerchantAddToBlackList
- interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList
- npc/clientbound/NpcSayImageConversationDetail
- npc/clientbound/NpcAskQuizConversationDetail
- npc/clientbound/NpcAskSpeedQuizConversationDetail
- npc/clientbound/NpcAskBoxTextConversationDetail
- npc/clientbound/NpcAskSlideMenuConversationDetail

v79 (2):
- npc/clientbound/NpcAskPetConversationDetail
- npc/clientbound/NpcAskSlideMenuConversationDetail

All nine were previously ❌ (`incomplete`, "no audit report"); each names a
feature the `_unimplemented.json` reason documents as version-absent.
