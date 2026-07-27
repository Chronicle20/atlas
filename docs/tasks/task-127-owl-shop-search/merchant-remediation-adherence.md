# Plan Adherence — merchant lifecycle remediation (task-127 addendum)

**Plan:** `docs/tasks/task-127-owl-shop-search/merchant-lifecycle-audit.md` §4 (Phases A1–A3, B1–B3, C1–C3), owner decisions §5 (Q1/Q4/Q5)
**Audit Date:** 2026-07-14
**Branch:** `task-127-owl-shop-search`
**Commit range audited:** `08e01ec74..HEAD` (6 commits: 08e01ec74, 0a1c5cbb2, 4c14d19a7, 6432c9203, c4aca9b50, 9a5abb75f)

## Executive Summary

7 of 9 plan items are fully implemented with file:line evidence; B1 and C3 are
PARTIAL. B1's replies/precheck/Fredrick plumbing is complete and
config-resolved, but the Q3 sub-bullet ("per-version mode bytes IDA-verified")
has evidence only for v83 within this commit range. C3's backlog listing is
explicit (F9), but no closure note exists for its "map remaining verified
enter-error codes" half — §0's implementation-status list names A1–C2 and
silently omits C3. Builds and tests pass in all three touched modules
(`libs/atlas-packet`, atlas-channel, atlas-merchant). Live verification (§0
checklist) is still pending by design.

## Per-Item Verdicts

| Item | Verdict | Summary |
|---|---|---|
| A1 | DONE | Position byte + owner-block re-derivation + comments + fixtures + semantic assertions + real slots |
| A2 | DONE (deviation noted) | Owner-occupancy resolution in merchant service; extra channel EXIT-semantics changes beyond "no channel handler changes" |
| A3 | DONE | EXIT-in-Draft closes; logout reaps Draft both types (Q5); expiry includes Draft |
| B1 | PARTIAL | Replies + filtered precheck + Fredrick all wired; Q3 per-version IDA derivation unevidenced beyond v83 |
| B2 | DONE | Verification-resolution (OPEN 0x0B / MAINTENANCE_OFF 0x27) consistent with code; no new wiring needed |
| B3 | DONE | Permit family + cash-inventory validation at CREATE; never consumed (Q1) |
| C1 | DONE | State/type-filtered resolution for VISIT and maintenance re-entry |
| C2 | DONE | Owner ENTER_RESULT re-send removed from SHOP_OPENED |
| C3 | PARTIAL | Backlog explicitly listed (F9), but no new error codes mapped and no disposition note; omitted from §0 status list |

## Evidence Detail

### A1 (F1) — position byte, owner block, comments, fixtures — DONE
Commit `08e01ec74`.
- `ownerView bool` → `position byte` (0 = owner, 1..3 = true visitor slot): `libs/atlas-packet/interaction/room.go:49`, written at `room.go:166`; zero residual `ownerView` references repo-wide (grep clean).
- Merchant owner block moved behind `position == 0`: `room.go:206-218` (encode), `room.go:285-298` (decode).
- Owner-block fields re-derived per Cosmic/IDA: packed open-time shorts `room.go:207-208`, firstTime byte `room.go:209` (@0x518b0a), sale ledger `room.go:210-216` (sub_518EFD), accrued meso total `room.go:217` (@0x518fbc). Populated via `SetOwnerLedger` (`room.go:129-135`). The old mislabeled meso/"not sold out"/soldTotal encoding is gone.
- Inverted comments fixed: `room.go:78-84`, `room.go:97-107`, `room.go:160-165` all now state 0 = owner with the IDA citations (@0x65ec6b, @0x6fc528, @0x518a7e).
- Channel room builders pass the recipient's real position from the insertion-ordered visitor registry: `viewerPosition` at `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go:575-585`, used by `buildShopRoomFirstTime` (`consumer.go:559`) feeding both `buildMerchantShopRoom` (`consumer.go:587`, ledger set only for position 0 at `consumer.go:620-622`) and `buildPersonalShopRoom` (`consumer.go:642`). Q4 (true slot 1–3) honored at `consumer.go:581`.
- Semantic assertions added: `libs/atlas-packet/interaction/room_test.go:160-175` (`TestRoomPositionByteSemantics` — owner byte 0x00, visitor byte = true slot at the raw-byte level), `room_test.go:121-152` (`TestMerchantShopRoomOwnerLedgerRoundTrip` — owner variant carries the ledger block), `room_test.go:80-118` (visitor variant carries no ledger).
- Per-version fixtures updated: `interaction/clientbound/interaction_test.go` (79 lines changed) and `v48_test.go`/`v61_test.go`/`v72_test.go`/`v79_test.go`. All pass (see Build & Test).
- Supporting: SHOP_SETUP sends firstTime=1 (`consumer.go:182`, `consumer.go:545-547`); shop `createdAt` exposed over REST for the open-time display (`services/atlas-merchant/atlas.com/merchant/shop/rest.go`, channel `merchant/model.go`/`rest.go`); dead `ToPacketRoom` surface removed (`socket/model/mini_room.go`, −52 lines).

### A2 (F2) — owner-occupancy resolution + owner exit semantics — DONE (deviation noted)
Commit `0a1c5cbb2`.
- `GetShopForCharacter` (`services/atlas-merchant/atlas.com/merchant/shop/processor.go:945-972`) now falls back from the visitor registry to the character's own `activeShops` entry: personal shop in any non-Closed state, hired merchant only in Draft/Maintenance (Open merchant = owner-detached). Serves the existing `GET /characters/{id}/visiting`, so all owner-side channel ops (OPEN, PUT_ITEM, EXIT, CHAT, MERCHANT_*) resolve. Tests: `processor_test.go:762,777,795,821`.
- Deviations vs the plan text, both reasoned and evidence-backed:
  1. Plan said "active-own-shop-first, visitor-registry-second"; implementation is visitor-first, owner second. Semantically equivalent — an owner occupying their own shop is never in the visitor registry (`AddVisitor` is only called from `EnterShop`), so the fallback order cannot change any outcome.
  2. Plan said "No channel handler changes", but `character_interaction.go` gained owner EXIT semantics: EXIT from a merchant in Maintenance → `ExitMaintenance` instead of close (`character_interaction.go:190-199`), and MERCHANT_EXIT from the owner → full `CloseShop` (Cosmic CLOSE_MERCHANT, `character_interaction.go:414-421`). These are additive corrections required for the Cosmic-faithful lifecycle (§1 reference lifecycle step 5) and are documented in the commit message — not silent scope creep.

### A3 (F3) — Draft reaping — DONE
Commits `0a1c5cbb2` + `4c14d19a7`.
- EXIT-in-Draft closes via A2: channel EXIT owner branch reaches `CloseShop` (`character_interaction.go:195-199`); merchant `CloseShop` already permits Draft (pre-existing, `processor.go:469` ff.).
- Logout reaper closes Draft shops of BOTH types per Q5: policy isolated in `services/atlas-merchant/atlas.com/merchant/shop/logout_policy.go:24-39` (`LogoutAction` — Draft→Close both types, personal→Close any live state, Open merchant survives, Maintenance merchant→ExitMaintenance), applied in `kafka/consumer/character/consumer.go:51-62`. Table-driven test `logout_policy_test.go:17`.
- Expiry includes Draft: `shop/task.go:34` and `shop/provider.go:85` both match `state IN (Draft, Open, Maintenance)`; cutoff bound Go-side (portable to the sqlite tests). Test `processor_test.go:841` (`TestGetExpired_IncludesDraftHiredMerchant`).
- The Maintenance→ExitMaintenance logout arm is a refinement beyond the plan's literal text (plan: "close Draft shops of both types; Open merchants keep survive-logout"), preventing a Maintenance-stranded merchant; consistent with Cosmic `closeHiredMerchant(false)`.

### B1 (F4) — ENTRUSTED_SHOP_CHECK_RESULT replies + filtered precheck — PARTIAL
Commit `6432c9203`.
- Replies wired (`services/atlas-channel/atlas.com/channel/socket/handler/hired_merchant_operation.go`): pass → `OPEN_SHOP` (`:95`), already-running → `ERROR_UNKNOWN` with mapId+channel (`:69`), Fredrick pending → `ERROR_RETRIEVE_FROM_FREDRICK` (`:87`), lingering Draft / query failure → `ERROR_UNABLE_TO_OPEN_THE_STORE` (`:57`, `:74`, `:82`). No silent no-reply path remains.
- Precheck filtered: non-Closed + `HiredMerchantShopType` only (`hired_merchant_operation.go:60-76`) — Closed history rows and personal shops no longer poison the check.
- Fredrick-pending check: `HasFrederickPending` (`services/atlas-channel/atlas.com/channel/merchant/processor.go:72`, `requests.go:18`) → new `GET /characters/{id}/frederick` (`services/atlas-merchant/atlas.com/merchant/shop/resource.go:38`, handler `:271`, provider `frederick/provider.go:32` with tests `frederick/processor_test.go:214,223,237`).
- Mode bytes config-resolved, never hard-coded: `libs/atlas-packet/merchant/operation_body.go:41-45` (new `HiredMerchantOperationErrorUnknownBody` via `WithResolvedCode`); `EntrustedShopUnknownChannel` now takes the mode as a parameter (`libs/atlas-packet/merchant/clientbound/operation.go`, hard-coded `mode: 8` removed) and its int is re-documented as the map-derived FM-room value (v83 @0xa27e6c).
- All 8 seed templates carry the operations entries (`OPEN_SHOP:7, ERROR_UNKNOWN:8, ERROR_RETRIEVE_FROM_FREDRICK:9, ERROR_UNABLE_TO_OPEN_THE_STORE:11` in `services/atlas-configurations/seed-data/templates/template_{gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_95,jms_185}_1.json`).
- **Gap (the PARTIAL):** the sub-bullet "per-version mode bytes IDA-verified (Q3)" — Q3 says "IDA-derive per version from each IDB's `OnEntrustedShopCheckResult` switch during B1". The commit evidences v83 only (@0xa27d75/@0xa27de3/@0xa27e6c); the JMS185 citation (@0xb0ee59) pre-dates this work. The template values are uniform (7/8/9/11) and pre-existed this commit range unchanged; no per-version switch derivation for v61/72/79/84/87/95 is recorded in the commits, the audit doc, or the checked-in IDA exports (the exports' `OnEntrustedShopCheckResult` entries contain no extractable case tables). Impact is bounded — a wrong per-version byte is a template data fix, not a code change — and §0's own checklist flags the related channel-byte display as unverified. But per the plan's wording this sub-bullet is not evidenced.

### B2 (F5) — merchant open op — DONE (verification-resolution consistent with code)
Commit `c4aca9b50` (doc-only for this item, as the resolution requires).
- Documented resolution (audit doc F5, §5 Q2): both dialogs go live via serverbound `OPEN` 0x0B (`CPersonalShopDlg::OnCorrectSSN2` @0x6fcbac, `CEntrustedShopDlg::OnCorrectSSN2` @0x5187db, payload `[0x7B][0x0B][0x01]`); maintenance re-open is `MAINTENANCE_OFF` 0x27 (`OnGoOut` @0x51925e).
- Code consistency: the `OPEN` arm routes to `OpenShop` for whichever shop the sender occupies (`character_interaction.go:205-216`) — with A2's owner resolution this now reaches a Draft merchant, which is exactly why no new wiring was needed; `MERCHANT_MERCHANT_OFF` (0x27) routes to `ExitMaintenance` (`character_interaction.go:391-400`); the `CASH_TRADE_OPEN nProc==11` birthday arm remains a logged no-op (`character_interaction.go:249-252`), matching F5's finding that it is not the open path. "Keep OPEN for personal shops" holds — the same arm serves both types via occupancy resolution.

### B3 (F7) — permit validation at CREATE — DONE
Commit `c4aca9b50`.
- `character_interaction.go:93-117`: permit family must match room type via `item.GetClassification` (`ClassificationStorePermit` ↔ personal store `:99-102`, `ClassificationHiredMerchant` ↔ hired merchant `:103-106`), and the claimed item must exist in the CASH inventory (`c.Inventory().Cash().FindFirstByItemId`, `:114-117`). Rejection sends miniroom error 6 (`CharacterInteractionEnterErrorModeUnable`, `:88-91`) — Cosmic `getMiniRoomError(6)` parity.
- Q1 honored: permits are validated only, never consumed — no consumption/saga code anywhere in the range (the only inventory `ReleaseAsset` remains listing stock), comment at `:96-97` records the owner decision.

### C1 (F6) — state/type-filtered shop resolution — DONE
Commit `9a5abb75f`.
- VISIT: `pickShopByState(shops, StateOpen)` with Maintenance forwarded for the faithful rejection (`character_interaction.go:155-165`); no more `shops[0]`.
- Merchant maintenance re-entry: `pickMerchantByState(shops, StateOpen)` — running hired merchant specifically, not a Closed row or the personal shop (`character_interaction.go:239-246`).
- Helpers at `character_interaction.go:516-534`.

### C2 (F8) — no owner room re-send at open — DONE
Commit `9a5abb75f`.
- `handleShopOpenedEvent` no longer sends `ENTER_RESULT` to the owner — only the map box (personal) / employee NPC (merchant) broadcast remains; explanatory comment at `consumer.go:159-162`. The maintenance-exit refresh path is untouched (separate handler).
- Note: the plan said "after live verification"; the removal landed ahead of live verification with a compensating safety property (an Open merchant no longer resolves as owner-occupied, so the client's trailing EXIT after go-live is benign — commit message + `processor.go:945-972`). §0's live checklist still covers this flow, so the reordering is visible, not silent.

### C3 (F9) — error-code mapping + explicit backlog — PARTIAL
- Backlog half: satisfied. F9 explicitly lists ORGANIZE/WITHDRAW_MESO/VIEW_VISIT_LIST/blacklist ops (`merchant-lifecycle-audit.md:138`) and the 0/58 `legacy-merchant-audit-remediation` doc (`:139`) as not-silently-dropped backlog.
- Mapping half: no new enter-error codes were mapped in the commit range. The existing `shopCreateFailureMode` mapping (`consumer.go:390-400`: portal/miniroom/free-market/unable) pre-dates this range (commit `a54244a90`); only FULL and UNABLE are otherwise emitted; B3 reuses UNABLE. That may be a legitimate no-op if no additional codes were verified during Phase C — but nothing says so: §0's implementation-status list enumerates A1, A2, A3, B1, B2, B3, C1, C2 and omits C3 entirely, and no scope note records "nothing newly verified to map". Per the plan's own standard ("not silently dropped"), C3's disposition is undocumented. Impact: documentation-only; no player-facing behavior gap identified.

## Owner Decisions (§5) Compliance

- **Q1 (never consume permits):** honored — B3 validates only; no consumption path exists (`character_interaction.go:93-117`, comment `:96-97`).
- **Q4 (true slot bytes):** honored — `viewerPosition` returns the 1-indexed registry slot (`consumer.go:575-585`); pinned by `TestRoomPositionByteSemantics` (`room_test.go:170-174`). Live slots-2–3 rendering remains on the §0 checklist as specified.
- **Q5 (close Draft on logout):** honored — `LogoutAction` closes Draft for both types (`logout_policy.go:31-33`), Open merchants survive (`:36-37`).

## Build & Test Results (run 2026-07-14, this audit)

| Module | `go build ./...` | `go test ./... -count=1` |
|---|---|---|
| `libs/atlas-packet` | PASS | PASS (all packages ok; zero failures — includes interaction fixture suites v48/61/72/79/83+ and the new room semantic tests) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS (all packages ok, incl. `socket/handler` 0.185s, `kafka` consumers) |
| `services/atlas-merchant/atlas.com/merchant` | PASS | PASS (all packages ok, incl. `shop` 10.874s, `frederick`, `visitor`) |

No test failures in any module. (The branch's own gate claims in §0 additionally cite `-race`, vet, guards, and matrix `--check`; those were not re-run for this adherence audit.)

## Action Items

1. **B1/Q3:** Record per-version verification for the ENTRUSTED_SHOP_CHECK_RESULT mode bytes (v61/72/79/84/87/95/jms) — derive each IDB's `OnEntrustedShopCheckResult` switch and note it (audit doc or evidence records), or explicitly document that the uniform 7/8/9/11 values are asserted-from-v83 and pending per-version confirmation. The wiring itself needs no change (config-resolved).
2. **C3:** Add one line to §0 (or F9) recording C3's disposition — e.g. "C3: no additional enter-error codes verified during Phase C; mapping unchanged from `a54244a90`; ORGANIZE/WITHDRAW_MESO/blacklist + 0/58 doc remain listed backlog."
3. Proceed to the §0 live-verification checklist (v83 tenant) — all nine items remain unchecked, by design.
