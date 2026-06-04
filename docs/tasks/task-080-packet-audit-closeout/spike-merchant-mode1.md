# Spike: Merchant entrusted-shop-check-result modes 1 / 8 / 11

Task B2.3 — disposition of three modes of the clientbound entrusted-shop-check-result
packet (`HiredMerchantOperation` writer, `merchant/clientbound/operation.go`).

The clientbound dispatcher is `CWvsContext::OnEntrustedShopCheckResult`, which reads a
leading `Decode1` mode byte and switches on it.

## Reference

- **JMS185** `CWvsContext::OnEntrustedShopCheckResult` @ `0xb0ee59` (decompiled).
- **GMS v95** `CWvsContext::OnEntrustedShopCheckResult` @ `0x9ffcb0` (mode 18 / FREE_FORM_NOTICE
  wire shape already proven in `operation_test.go::TestFreeFormNoticeWireShape`).

In JMS185 the switch's defined cases are **7, 8, 9, 10, 11, 13, 14, 15, 16, 17**. The
**lowest case is 7** — there is **no `case 1`**.

## Mode 1 — VERDICT: absent / NOT implemented

`OnEntrustedShopCheckResult` in **JMS185 has no `case 1`** (lowest case is 7), and the
v95 plan context likewise records mode 1 as absent from the baseline. Mode 1 is therefore
**client/KMS-only** — not present in either of our reference clients — and we do **not**
emit it. No emitter, no constant; mode 1 is a documented absence, not code.

## Mode 8 — IMPLEMENTED (emitter added)

Mode 8 is the "unknown channel" notice. Body (after the mode byte):

```
Decode4  shopId       (int, little-endian)
Decode1  channelId    (byte)
```

The client uses shopId + channelId to redirect the player toward the channel where the
shop actually lives. Implemented as the `EntrustedShopUnknownChannel{mode=8, shopId, channelId}`
clientbound type in `operation.go`, matching the sibling emitter idiom
(`RemoteShopWarp` has the identical mode/shopId/channelId body shape).

### Step-4 disposition — emitter available, NOT wired (no clean server trigger)

The only server-side entrusted-shop entry point in the current codebase is
`socket/handler/hired_merchant_operation.go` (landed in B2.2). That handler services
`ModeEntrustedShopCheck` (the permit check) and only decides whether the player may open a
**new** shop — it returns silently on success and on the "already operates a shop" branch.

There is **no current server path** that resolves a *remote, already-existing* shop by id
and discovers it lives on a different channel — which is exactly the condition mode 8
reports. Producing that signal today would require inventing a trigger that the codebase
does not have. Per the task's "do not invent a fake trigger" instruction, the mode-8
emitter is left **defined and available** to be hooked up when a remote-shop-lookup server
path exists. Decision: **emitter available, wiring deferred** — `atlas-channel` was not modified.

**Future-wiring caveats (recorded so the integration isn't surprising):**
- `EntrustedShopUnknownChannel` is currently the **only** unwired emitter in
  `merchant/clientbound/operation.go` — every sibling (OpenShop, ErrorSimple, ShopSearch,
  ShopRename, RemoteShopWarp, ConfirmManage, FreeFormNotice) is wired via a `*Body()` wrapper
  in `merchant/operation_body.go`. When mode 8 is wired it should get a matching `*Body()` wrapper.
- The emitter **hardcodes `mode = 8`**, whereas siblings resolve their mode from per-tenant
  config via `WithResolvedCode("operations", <KEY>, ...)`. The hardcode is correct while
  unwired (and the byte test pins `b[0]==8`), but a future `*Body()` wrapper must reconcile
  with the config-resolution path. Note `operation_body.go` already annotates the
  `ERROR_UNKNOWN` string code as `// 8`; confirm there is no code-8 contention before wiring.

## Mode 11 — constant only (no emitter)

Mode 11 is a defined case in `OnEntrustedShopCheckResult` that takes **no body**: it merely
displays a notice string from the client StringPool (JMS185 entry 3638; the v95 plan
context cites 3508). Because it carries no payload and no server path currently raises it,
it is registered as a **named constant only** (`EntrustedShopOperationMode11 = 11`) with a
comment citing the StringPool notice. No emitter is added — a body-less mode is fully
expressed by `NewMerchantErrorSimple(11)` (the existing mode-only `ErrorSimple` emitter)
if/when a server path needs to send it; defining a dedicated empty-body type would be
redundant. Disposition: **defined-but-unused constant**.
