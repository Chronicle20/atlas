# Review — task-240 Task 1 (derive the four blocking client facts)

**Range:** `a510b7650..7fd10bf15` (1 commit, `7fd10bf15`)
**Artifact under review:** `docs/tasks/task-240-cash-shop-stub-operations/derivation.md` (488 lines, new file — the only change in the range)
**Reviewer surface:** the diff (whole new file, doc-only) plus the repo files it cites: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`, `template_gms_92_1.json`, `libs/atlas-packet/cash/clientbound/shop_operation_body.go`, `libs/atlas-packet/cash/serverbound/shop_operation_buy_package.go`, `shop_operation_gift.go`, `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`. No IDA tools available in this session, so no wire-level fact was re-derived — only checked for (a) presence of a concrete address/excerpt, (b) whether the excerpt actually supports the stated claim, (c) honest labelling of inference, and (d) the checkable repo-side claims (template rows, existing constructor signatures, handler line numbers).

## Scope confirmation

`git log --oneline a510b7650..7fd10bf15` shows exactly one commit; `git diff --stat` shows exactly one file added, `derivation.md`, 488 insertions, 0 deletions. This matches the brief's Step 6/7 exactly — no code changed, nothing else touched. Scope confirmed.

## Global constraint check — IDB identity (§0)

- The doc states plainly, at the top of §0, that "No v95.1 database exists" and the database actually used is v95.0, with `filename: GMS_v95.0_U_DEVM.exe.i64`, `input_path`, `session_id: 32c8836f`. It does **not** silently claim v95.1 anywhere in the 488 lines (checked: no bare "v95.1" claim appears without the disclaiming context of §0). PASS.
- The doc also notes the `input_path` it got from the tool differs from the brief's quoted path, and says explicitly it used the tool's returned value rather than the brief's — correct handling of a discrepancy, not silently resolved. PASS.
- Mode-byte provenance: §0 states serverbound/clientbound mode numbers come from `template_gms_95_1.json` lines 2307–2319 and 4812–4847, not from memory, and explicitly flags that v83 doc-comment mode numbers found elsewhere in the repo are not reused. Verified against the template directly (see below) — the line numbers and values are exact.

## Per-answer verification

### D1 — v95 opcode for `CStage::OnSetCashShop` → `0x8F` (143)

- Concrete evidence present: function address `0x71adf0`, dispatch site `0x71b0cd` inside `CStage::OnPacket` at `0x71b0b0`, quoted disassembly/decompile of the switch with cases 141/142/143. Not a bare assertion.
- Excerpt supports the claim: the switch literally shows `case 143: CStage::OnSetCashShop(...)`.
- Cross-check against template rows 141→`0x8D`/`SetField`, 142→`0x8E`/`SetItc` — **verified directly against the repo**: `template_gms_95_1.json:3272` is `opCode 0x8D, writer SetField, fname CStage::OnSetField`; `:3285` is `opCode 0x8E, writer SetItc, fname CStage::OnSetITC`. Both match the doc's table exactly.
- Slot-availability claim ("`0x8F` is unclaimed in the writers array; the `0x8F` at line 987 is in the serverbound `handlers` array, `MessengerOperationHandle`/`CFadeWnd::SendCloseMessage`") — **verified directly**: grepped all `opCode` values 0x8C–0x93 in the file. The `writers` array (the clientbound one starting ~line 3255) runs `0x8C→0x8D→0x8E→0x93` with `0x8F` genuinely absent. Line 987 is inside the `socket.handlers` (serverbound) array at the top of the file, and reads exactly `{"opCode": "0x8F", "validator": "LoggedInValidator", "handler": "MessengerOperationHandle", "fname": "CFadeWnd::SendCloseMessage", "services": ["channel"]}`. **This is the fact Task 2 depends on, and it is correct** — registering `0x8F` as a new clientbound writer entry will not collide with the existing serverbound handler entry at the same opcode value, because the two arrays are different namespaces (client-to-server vs server-to-client).
- v92 non-copy-forward claim — **verified directly**: `template_gms_92_1.json:2413-2419` reads `{"opCode": "0x8E", "writer": "CashShopOpen", "fname": "CStage::OnSetCashShop", ...}`. On v95, `0x8E` in the same (writers) array is `SetItc`/`CStage::OnSetITC` (confirmed above) — copying the v92 value forward would indeed collide. Correct warning.
- No inference flagged as fact here; the cross-check is presented as corroboration, not as the sole basis. PASS.

### D2a — APPLY_WISHLIST body → empty

- Concrete address (`0x482ea0`), full decompilation quoted showing `Encode1(0x23)` then `SendPacket` with no intervening `Encode*`. Reasoning that "no inlined `ZArray` append" is visible is inherently limited to what the decompiler surfaces, but the doc explicitly gives a contrast case (§4, `OnBuyCouple`) as the pattern it would expect to see if there were more fields — a legitimate methodology, not an unlabelled gap. PASS, no finding.

### D2b — reply arm → `UPDATE_WISHLIST` (98)

- Two independent evidentiary chains given: (1) the `m_bCashShopRequestSent` latch sweep with specific addresses for both writer and reader sites, cross-referenced against six other cash-shop sends that share the same gate; (2) the payload shape of `OnCashItemResSetWishDone` (`DecodeBuffer(..., 40)` = 10×uint32) matching `CashShopWishListUpdateBody(sns []uint32)`.
- **Repo-side check**: `CashShopWishListLoadBody` and `CashShopWishListUpdateBody` both exist in `libs/atlas-packet/cash/clientbound/shop_operation_body.go:181-195` exactly as named and described.
- Crucially, the doc labels this binding as inferential in an explicit "**Residual caveat (honest)**" callout (derivation.md:197-201), stating the client has no request→response correlation table and naming the observable symptom if the binding is wrong (wedge, not mis-render). This is exactly the labelling behaviour requirement 3 of the review brief calls for. PASS, and the implementer report also surfaces this caveat at the top level rather than burying it — good faith.

### D3a — BUY_OTHER_PACKAGE body → different shape (spw/SN/name/message, no pointType/option)

- Field table with per-field address citations (`0x490b93`, `0x490be2`, `0x490c01`, `0x490c1d`) plus a full quoted decompile excerpt. Contrast case (mode 32, `OnBuyPackage`) quoted immediately after, showing the shape that **is** `ShopOperationBuyPackage`.
- **Repo-side check**: `libs/atlas-packet/cash/serverbound/shop_operation_buy_package.go:25-29` — `type ShopOperationBuyPackage struct { pointType bool; option uint32; serialNumber uint32 }`. Matches exactly what the doc cites as "for contrast," confirming the doc did not misstate the existing struct it is comparing against. PASS.

### D3b — reply arm → `GIFT_PACKAGE_SUCCESS`(156) / `GIFT_PACKAGE_FAILED`(157)

- Dispatch case values (`0x9C`=156, `0x9D`=157) and a full decompile of `OnCashItemResGiftPackageDone` showing the exact field-read order (`DecodeStr, Decode4, Decode2, Decode2, Decode4`), plus the contrasting 154 arm's shape (count + item-blob, no recipient name) to rule out the alternative.
- **Repo-side check**: `CashShopGiftPackageDoneBody(recipientName string, packageId int32, unused1 uint16, unused2 uint16, nxCashSpent int32)` exists verbatim at `shop_operation_body.go:577`, matching the field-for-field claim. Template values `GIFT_PACKAGE_SUCCESS: 156` / `GIFT_PACKAGE_FAILED: 157` confirmed at `template_gms_95_1.json:4846-4847`. PASS.

### D4a — `option` → payment-method selector (1/2/4), not spare

- Three-step chain with quoted decompile/disassembly at each step (bitmask seeding in `OnBuyPackage`/`OnBuyCouple`, the by-pointer narrowing in `CConfirmPurchaseDlg::Confirm` with the client-side "choose only one" refusal string, and the `pointType = (dwOption==2)` derivation showing why `pointType` alone is lossy).
- **Repo-side check of the "server only logs it" claim**: `grep -n "sp.Option()" cash_shop_operation.go` returns exactly lines 167, 174, 184 — matching the doc's citations verbatim, all inside `l.Infof(...)` calls, no branching. Broader sweep (`grep -rn "\.Option()" services/ libs/`) turns up no other production consumer anywhere in the repo — only test-file round-trip assertions and an unrelated `attack.go` `Option()` on a different type. The doc's "no consumer that branches on it anywhere in `services/` or `libs/`" claim holds up under an independent, broader sweep than the doc itself performed. PASS.
- The verdict is correctly hedged: "currently unconsumed... not semantically spare... must not be documented as unused/reserved" — this is not overclaiming that the field can be dropped; it distinguishes "safe for a stub to ignore today" from "safe to design out." Good.

### D4b — `oneADay` → client-set request marker; server enforcement not determinable from client

- Offset (`+0x3B4C`) established from a named function's decompile+disasm, an 11-site `search_text` sweep classified into resets/writes/reads with addresses for each, and a full decompile of `CCSWnd_OneADay::OnButtonClicked` showing set-for-today / clear-for-previous-days behaviour.
- Part (c) is an explicit negative finding, correctly not overclaimed: "Whether the *server* rejects a second one-a-day purchase... cannot be read out of the binary... Do **not** document it as 'server-enforced per-day limit.'" This is exactly the FR-V95-3-style honesty the brief demands for an inconclusive sub-question, applied even though the top-level D4b was still marked RESOLVED (correctly, since the client-side half — what the byte means and where it sits on the wire — was resolved; only the server-enforcement question was out of reach of the binary, and that's stated).
- **Repo-side check of the wire-position claim**: `libs/atlas-packet/cash/serverbound/shop_operation_gift.go`'s GMS≥95 `encodeGMS`/`decodeGMS` arms write/read in exactly the order the doc's decompile excerpt shows: spw string, serialNumber, oneADay byte, name, message (`shop_operation_gift.go:63-73`). Matches. PASS.

## Cross-check: incidental sanity table in the report

The implementer report (not part of the committed artifact) claims every serverbound mode name in the `gms_95_1` operations table maps cleanly onto a v95 client sender. This is outside the reviewed diff (it's not in `derivation.md`) and not itself a claim the artifact makes, so it is not scored, but nothing in it contradicts the derivation.md content that IS scored.

## Findings

No blocking findings. Every one of the seven answers (D1, D2a, D2b, D3a, D3b, D4a, D4b) carries a concrete address and a decompilation/disassembly excerpt rather than a bare assertion; in every case the excerpt actually supports the claim made above it (no case of an excerpt supporting a weaker claim than asserted); the one genuinely inferential binding (D2b's request→response pairing) is explicitly flagged as inferential with the observable failure symptom named, and the one genuinely unknowable sub-question (D4b(c), server-side enforcement) is explicitly declined rather than guessed. The IDB identity is recorded accurately as v95.0, never silently upgraded to "v95.1." The D1 template-slot claim that gates Task 2 (opcode `0x8F` free in `writers`, occupied by an unrelated serverbound entry in `handlers`) was independently verified against the template file and is correct — Task 2 will not write a colliding registration on the basis of this fact. All checkable repo-side citations (template line numbers/values, existing constructor signatures, handler log-site line numbers) match exactly what the doc states.

Two minor observations, both non-blocking:

1. `derivation.md:82` table row for D1 shows the future Task 2 registration in the same table as the two existing cross-check rows — stylistically this mixes "evidence" with "recommendation" in one table, but it's clearly labelled `(absent — this is what Task 2 adds)` so there's no ambiguity about what's derived vs. what's prescribed.
2. The report says "no subagents dispatched" and "~35 of 120 tool calls used" — consistent with a documentation-only task; nothing to flag.

## Not evaluable

- All primary wire-level facts (D1's dispatch site, D2a's empty-body decompile, D2b's latch sweep and payload shape, D3a's field table, D3b's dispatch arms, D4a's dialog logic, D4b's offset sweep) were derived from IDA and cannot be re-derived or independently confirmed in this review session — no IDA tooling is available here. These are reported as "cannot verify from IDB directly," per the task instructions, not as findings. What COULD be checked (repo-side citations: template rows, existing Go struct/constructor shapes, handler log-site line numbers) was checked and matched in every instance enumerated above.

## Verdict

APPROVED. Every checkable claim matches the repo; every uncheckable claim carries a concrete address/excerpt rather than a bare assertion; inference and genuine unknowns are labelled where the brief requires it; the IDB identity is recorded honestly; the fact gating Task 2 is correct.
