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
