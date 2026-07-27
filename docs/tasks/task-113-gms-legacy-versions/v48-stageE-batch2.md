# v48 Stage E — Batch 2 (party / summon / inventory / pet)

Anchor = gms_v61 (verified 208). IDB port 13337 (`GMS_v48_1_DEVM.exe`). All
byte fixtures trace to a live decompile line; distrust-symbol-names honoured
(every send-site body-verified, and one registry note corrected — see PET_CHAT).

## Per-cell outcome

### summon (2 op cells) — VERIFIED
| cell | outcome | evidence |
|---|---|---|
| `SUMMON_ATTACK` op121 / `SummonAttackHandle` | ✅ v48-gated | send-site sub_5D9424@0x5d9bae. v48 **diverges from v61**: leads with Encode4(summonSkillId this[33]) then Encode1(action|left) + Encode1(count) — **NO updateTime int** (v61 @0x67b3aa has it). Gate `hasSummonAttackUpdateTime` (GMS<61 omits). No skillCRC (v48<79), no templateId (legacy). |
| `DAMAGE_SUMMON` op122 / `SummonDamageHandle` | ✅ =v61 | send-site sub_5DA381@0x5da5a7. Encode4(summonId)+mob/no-mob branch (0xFE) — byte-identical to v61. Fast path. |

Codec gate added: `libs/atlas-packet/summon/serverbound/attack.go` `hasSummonAttackUpdateTime`.

### pet (4 op cells) — VERIFIED
v48 predates multi-pet: **all four pet action send-sites drop the leading
`EncodeBuffer(petId,8)`** that v61+ carries. Gate `hasLeadingPetId` (GMS<61) in
new file `libs/atlas-packet/pet/serverbound/legacy.go`.

| cell | outcome | send-site |
|---|---|---|
| `MOVE_PET` op113 / `PetMovementRequest` | ✅ v48-gated | sub_6E5BD6@0x6e5bff — COutPacket(113)+CMovePath::Flush only (no petId) |
| `PET_CHAT` op114 / `PetChatRequest` | ✅ v48-gated | CPet::DoAction@0x58e90b — Encode1(type)+Encode1(action)+EncodeStr(msg). **Registry note was WRONG** (claimed leading petId buffer); send-site proves it absent. |
| `PET_COMMAND` op115 / `PetCommand` | ✅ v48-gated | sub_58DF8A@0x58e1b8 — Encode1(byName)+Encode1(command) (no petId) |
| `PET_LOOT` op116 / `PetDropPickUp` | ✅ v48-gated | sub_58ED98@0x58edb0 — fieldKey+time+x+y+dropId+3 pet-flags (no petId, no crc<83) |

Round-trip `petId` assertions guarded for GMS<61 (test variant `GMS v28`).

### inventory (2 op cells + ITEM_SORT resolution)
| cell | outcome | send-site |
|---|---|---|
| `ITEM_MOVE` op55 / `InventoryMove` | ✅ =v83..v95 | sub_70D8DE@0x70d905 — Encode4(updateTime)+Encode1(invType)+Encode2(src)+Encode2(dst)+Encode2(count). Matches struct exactly, no gate. |
| `USE_ITEM` op65 / `InventoryItemUse` | ✅ =v83..v95 | sub_719DD9@0x719f8e — Encode4(updateTime)+Encode2(slot)+Encode4(itemId). No gate. |

**ITEM_SORT / ITEM_SORT2 — RESOLVED n-a (version-absent, decompile evidence).**
`docs/packets/audits/gms_v48/_unimplemented.json`. The v61 gather/sort senders
(`SendGatherItemRequest` sub_8314D0@0x8314d0 / `SendSortItemRequest`
sub_831564@0x831564) are `if(compartment∈[1,5]){throttle; COutPacket(N)+Encode4+
Encode1(compartment)}` in the v61 **0x831xxx** secondary send region. v48 has NO
0x831xxx region. A **complete** `xrefs_to sub_4A2518` (the 500 ms exclusive-request
throttle; 72 refs, `more:false`) enumeration of the v48 inventory send cluster
(0x70dxxx–0x71xxxx) yields sub_70D8DE(op55), sub_70D987(drop), sub_70DA60(op66),
sub_70DC8D(op117), sub_70DF2F(op60), sub_70E00B(op61), sub_719DD9(op65) — **none**
carries the [1,5]-guarded Encode4+Encode1 gather/sort body. v48 op65 is USE_ITEM,
not the v61 ITEM_SORT2 slot. Feature version-absent; Stage B flag cleared.
GATHER_ITEM_RESULT/SORT_ITEM_RESULT (clientbound) follow (no request path).

## Tool/report plumbing
- `candidatesFromFName` (`tools/packet-audit/cmd/run.go`): added v48 unnamed-sub
  cases sub_5D9424, sub_5DA381, sub_6E5BD6, sub_58DF8A, sub_58ED98, sub_70D8DE,
  sub_719DD9 → their codecs (so op rows link on the v48 primary fname).
- Surgically stripped the phantom **unresolved** named export stubs (empty
  address, "function not found in IDB") from `docs/packets/ida-exports/gms_v48.json`
  for the 7 above so report-gen resolves the real sub_ addresses. Line-precise
  removals only (no unicode/format churn; validated JSON).
- Regenerated + copied the 8 v48 audit reports; pinned 8 TIER1-FIXTURE evidence
  records with `verifies:`.

## Gates (final)
- `go test -race` green: summon/pet/inventory serverbound.
- `go vet` clean (changed packages); `go build ./tools/packet-audit/...` clean.
- `packet-audit matrix --check` → **exit 0**; problem-grep (orphan|dangling|
  stale|drift|unresolv|malformed) = **0**; v48 conflicts = **0**.
- Regression: existing verified counts UNCHANGED — v61 208, v72 216, v79 228,
  v83 367, v84 345, v87 379, v95 399, jms 362. v48 17→**25** (net +8 op cells;
  op rows consume their sub-struct reports, matching the anchor's display shape).
- Branch verified `task-113-gms-legacy-versions` after every commit.

## Commits (5)
1. `1430b84` summon SummonAttackHandle + SummonDamageHandle (fixtures/evidence + attack.go gate)
2. `c087921` link summon SUMMON_ATTACK/DAMAGE_SUMMON op rows (candidatesFromFName + stub strip + reports)
3. `ef5f204` pet MOVE_PET/PET_CHAT/PET_COMMAND/PET_LOOT
4. `2e0f38a` inventory ITEM_MOVE + USE_ITEM
5. `2f9092c` ITEM_SORT/ITEM_SORT2 n-a disposition

## Remaining in-scope cells (NOT completed — flagged, not rushed)
Deliberately left for a follow-up pass rather than risk a dispatcher false-pass:
- **party family** (`PARTY_OPERATION` cl + se, `DENY_PARTY_REQUEST` se, and the 5
  serverbound sub-structs Operation/ChangeLeader/Expel/Invite/Join). This is the
  Stage-C-flagged **carried-unverified** case with **no v61 anchor** (v61 itself is
  incomplete/partial for these). It is a mode-prefix dispatcher: op94 is emitted
  from multiple UI-result send-sites with distinct mode bytes (verified sample:
  sub_4BE880 case 4 @0x4be91d = COutPacket(94)+Encode1(3)+Encode4(int)). Correct
  verification requires enumerating every op94/op95 send-site and mapping each mode
  → body from the v48 switch before fixturing the arms — not safely completable in
  this batch without a false-pass risk on the mode mapping.
- `INVENTORY_OPERATION` op25 clientbound (`CWvsContext::OnInventoryOperation`) — a
  large clientbound add/remove/change dispatcher; not started.
- `pet/serverbound/PetFood` and `pet/clientbound/PetActivated` — no v48 registry op;
  both are **unresolved** export stubs (`SendPetFoodItemUseRequest`,
  `OnPetActivated` — empty address). Require unnamed send/read-site archaeology
  (byte-signature + twin match) to locate first.
