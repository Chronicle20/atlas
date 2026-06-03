# Pending items — GMS v95 packet audit

Rows in this file represent deferred audit items that require follow-up work:
either the IDA evidence is ambiguous, the atlas implementation is missing, or a
mode is out of scope for the current sub-phase.

---

## Bare handlers — misc domain (task-069)

| Handler constant | Location | Reason deferred |
|---|---|---|
| `HiredMerchantOperationHandle` | `libs/atlas-packet/merchant/serverbound/operation.go` | Bare constant only — no atlas-packet decoder struct. The actual serverbound parse is handled in `services/atlas-channel` socket handler (`hired_merchant_operation.go`). Out of scope for libs/atlas-packet audit. |

---

## Missing / unverified modes — merchant (task-069, sub-phase 2f)

| Mode | Constant | Reason deferred |
|---|---|---|
| 8 (0x08) | `HiredMerchantOperationModeErrorUnknown` | In IDA switch: `Decode4(shopId) + Decode1(channelId)` — client shows a channel-name notice. No atlas body function (`HiredMerchantOperationErrorUnknownBody`) exists; the mode is never emitted by any service. Missing implementation, not a struct wire bug. Requires a new struct or updated `ErrorSimple` body to include shopId + channelId payload. |
| 1 (0x01) | `HiredMerchantOperationModeErrorUnableToOpenTheStore` | Absent from `OnEntrustedShopCheckResult` IDA switch in GMS v95. May be a hire-merchant mode (task-067 / commerce scope), a KMS-only mode, or a client-side only constant. Needs cross-reference with task-067 and KMS client decompilation before implementing. |
| 11 (0x0B) | *(no atlas constant)* | Present in IDA switch (string-pool 3508 notice, no additional decode) but has no corresponding atlas constant or struct. Likely an additional error mode not yet modelled. Low priority — add constant + body function when the mode is exercised. |

---

## Still pending — quest bucket (task-069, sub-phase 2g)

### ActionStart and ActionComplete — missing delivery-item slot field

**Structs:** `libs/atlas-packet/quest/serverbound/action_start.go`,
             `libs/atlas-packet/quest/serverbound/action_complete.go`

**IDA evidence:** `CQuest::StartQuest` @ 0x6b40a0, actions 1 and 2.

In GMS v95, both action=1 (start quest) and action=2 (complete quest) write a `nItemPos`
field (uint32, delivery-item cash-shop slot position, 0 if no delivery item) **between**
`npcId` and the conditional `x,y` coordinates:

```
Encode4(npcId)            // present in atlas
Encode4(nItemPos)         // ← MISSING in atlas (always 0 for non-delivery quests)
Encode2(x) / Encode2(y)  // conditional on !IsAutoAlertQuest (atlas: conditional on autoStart)
```

For `ActionComplete` there is also a trailing `Encode4(nIdx/selection)` which IS present
in atlas.

**Why deferred:**
1. The `nItemPos` field is always present on the wire (value 0 for normal quests). Adding it
   is a structural change that requires updating both the atlas struct and the atlas-channel
   handler (`quest_action.go`), which currently doesn't read this field.
2. The `autoStart` boolean in atlas controls the conditional `x,y`. In IDA the gate is
   `!CQuestMan::IsAutoAlertQuest(questId)`. These are semantically equivalent (auto-alert
   quests don't send coordinates) but the naming mismatch warrants review before a fix.
3. Changes to these structs affect the live atlas-channel service and require corresponding
   handler updates — broader scope than the libs-only audit.

**Recommendation:** Fix in a dedicated follow-up task that updates both
`libs/atlas-packet/quest/serverbound/action_start.go`, `action_complete.go` AND
`services/atlas-channel/.../socket/handler/quest_action.go` together.

---

### ActionRestoreLostItem — data-dependent lost-item list

**Struct:** `libs/atlas-packet/quest/serverbound/action_restore_lost_item.go`

**IDA evidence:** `CQuest::OnCompleteQuestFailed` @ 0x6b1fc0, action=0.

When a quest-complete attempt fails due to missing items, the client shows a dialog and
on confirmation sends action=0 with a list of the lost items. The IDA packet construction
at ~0x6b2bde shows:

```
COutPacket::COutPacket(&oPacket, 119)
Encode1(0)                          // action = 0
Encode2(questId)                    // questId
Encode4(aLostItem[-1].lpVtbl)       // item count (from ZArray metadata)
EncodeBuffer(aLostItem, 4 * count)  // array of uint32 itemIds
```

The atlas `ActionRestoreLostItem` struct models only `unk1 (uint32) + itemId (uint32)` —
treating this as a single-item restore. The IDA evidence suggests it's a variable-length
list of item IDs with a count prefix.

**Why deferred:**
1. The IDA decompilation for `OnCompleteQuestFailed` is very large (1244 lines) and uses
   complex COM-style IWzProperty iteration, making exact field sizes hard to verify without
   more investigation.
2. The current atlas handler (`quest_action.go`) only reads `unk1 + itemId`, so updating
   the struct in isolation would break the handler.
3. This code path (delivery-quest failure restore) is rarely exercised and the current
   single-item model may work in practice for most quests.

**Recommendation:** Revisit when delivery-quest restore feature is actively tested. Full
fix requires struct redesign (slice of item IDs + count prefix) plus handler update.
