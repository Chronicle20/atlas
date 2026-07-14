# Merchant Shop Lifecycle Audit — task-127 addendum

Status: Remediation IMPLEMENTED (Phases A–C, 2026-07-14) — awaiting live verification
Created: 2026-07-14

## 0. Disposition — what is FIXED vs what REMAINS

### Fixed on this branch (all gates green; pending live verification below)

| Finding | Symptom it caused | Fix | Commit |
|---|---|---|---|
| F1 inverted position byte | Owner saw the visitor "open shop listing" view of their own shop; visitors would have seen the owner management UI | Position-byte semantics corrected (0 = owner, 1–3 = true visitor slot); merchant owner-block moved to the owner view with re-derived fields; fixtures + semantic assertions | `08e01ec74` |
| F2 owner never resolvable | Owner could not stock, open, close, or chat in a freshly created shop — dead Draft record | Owner-occupancy resolution (own active shop when owner-attached) behind `/characters/{id}/visiting`; owner EXIT semantics per shop type/state | `0a1c5cbb2`, hardened `f60f5b47e` |
| F3 stranded Drafts block re-create | "Cannot open another shop until relog" (personal) / forever (merchant) | EXIT-in-Draft closes (via F2); logout closes Draft shops of both types; expiry reaper includes Draft | `0a1c5cbb2`, `4c14d19a7` |
| F4 503 permit check dead-ends + poisoned precheck | Hired-merchant permit was a silent no-op; any historical shop row refused the check forever | Check replies wired (modes 7/8/9/11, IDA-verified on all 8 feature IDBs); precheck filtered to non-Closed merchants; Fredrick standing exposed via `GET /characters/{id}/frederick` | `6432c9203`, `f60f5b47e` |
| F5 "no merchant open path" | (Suspected blocker) | **Non-issue by verification**: both dialogs go live via `OPEN` 0x0B, already wired — it only failed because of F2 | doc-only, `c4aca9b50` |
| F6 unfiltered `shops[0]` resolution | VISIT/maintenance could act on a Closed or wrong-type shop row | State(+type)-aware selection for VISIT and maintenance entry | `9a5abb75f` |
| F7 permit never validated | Any client could place shops with no permit / arbitrary permit id | Family + cash-inventory validation at CREATE (reject = mini-room error 6). Consumption intentionally none (owner decision Q1) | `c4aca9b50` |
| F8 owner room re-sent at open | Owner dialog re-created at go-live | SHOP_OPENED no longer re-sends ENTER_RESULT to the owner | `9a5abb75f` |

Gates on the final tree: `go test -race` / `go vet` / builds clean in
atlas-packet, atlas-channel, atlas-merchant; redis-key-guard,
goroutine-guard, `packet-audit matrix --check`, and `docker buildx bake`
for both services all clean. Both reviewer reports and their fixes are in
this folder (`merchant-remediation-review.md`, `merchant-remediation-adherence.md`).

### Remaining — known, documented, NOT fixed on this branch

None of these block the shop lifecycle or task-127's owl testing; they are
listed so nothing reads as silently done:

1. **F10 — owner messages not surfaced** (below): merchant service persists
   visitor messages; the shop REST payload and room builder never deliver
   them to the owner's management view. Needs REST enrichment + builder wiring.
2. **F9 — unwired merchant ops**: PARTIALLY RESOLVED (2026-07-14).
   - DONE: **MERCHANT_WITHDRAW_MESO** (credits the hired merchant's accrued
     balance to the owner + zeroes it + refresh) and **MERCHANT_ORGANIZE**
     (drops sold-out rows, compacts display order, closes when empty) are now
     full-stack — command → owner-guarded processor → SHOP_UPDATED refresh.
     The **maintenance/closed enter-error** is now surfaced (ENTER_FAILED →
     UNDERGOING_MAINTENANCE / ROOM_CLOSED) instead of a silent drop.
   - REMAINING (genuine new features, escalated — not wiring): blacklist
     (add/remove/**view**) and the **visit list** need a new per-shop
     persistence table AND, for the two *view* ops, an IDA-derived
     list-display packet layout that has not been resolved
     (`CEntrustedShopDlg::OnBlackList` @0x519382 / `OnVisitList` @0x5194af
     senders are known, but the clientbound list bodies are not fixtured).
     Blacklist *enforcement* also needs a name→characterId lookup
     (atlas-merchant → atlas-character). These are a scoped follow-up, not a
     stub: the handlers remain decode-and-log until the packets are derived.
   - INVITE/INVITE_RESULT interaction modes are trade-invite surface (trades
     are unimplemented in atlas-channel), out of merchant scope.
3. **jms185 free-form notice**: the client has no case 18 — the Fredrick
   reminder notice is a benign silent drop on jms until given another vehicle.
4. **v48 template inconsistency** (review W3) — RESOLVED: v48 predates the
   player/hired-shop feature (verified: the v48 `PLAYER_INTERACTION`
   dispatcher `sub_5459C4` @0x5459c4 has only base mini-room + trade +
   messaging arms, no shop room type; zero shop strings in the IDB; v48 is
   the sole legacy version lacking a `CharacterInteraction` clientbound
   writer). The unreachable shop sub-modes (`PERSONAL_STORE_*`, `MERCHANT_*`,
   `FIELD_*_BLACK_LIST`) were removed from the v48 handler operations table —
   they could only misfire (a v48 client byte misread as a shop op). v48
   keeps CREATE/INVITE/VISIT/CHAT/TRADE_* and gets no shop writer (no feature
   to serve). v61/v72/v79 have the writer + shop support (they carry owl).
5. **Channel-side state-byte mirror** (review W4): `StateDraft/StateClosed`
   hand-mirror atlas-merchant's states with no compile-time link; candidate
   for promotion to atlas-constants.
6. **Pre-existing frederick-package debt** (review): no `builder.go`, entity
   mapping placement — predates this branch, out of its blast radius.
7. **`legacy-merchant-audit-remediation` doc**: 0/58 items executed
   (provider/write-op layering, Kafka event-correctness items). One of its
   findings — logout never emitting SHOP_CLOSED — is now moot (the logout
   reaper emits via CloseShopAndEmit); the rest are unassessed debt.
8. **CloseShop has no server-side ownership check**: the Kafka command
   surface trusts the caller (commands originate only from atlas-channel,
   which does check); noted for the eventual hardening pass.
9. **Reference-citation sweep**: ~80 legacy "Cosmic" comments in unrelated
   features (summons, mounts, monsters, point-reset, packet lib) await a
   dedicated scrub with re-verified provenance; this branch's files are clean.

Live verification checklist (v83 tenant):
- [ ] 514 create → owner lands in the MANAGEMENT view (add-item UI), not the buy view; no map box yet.
- [ ] Stock an item, press Open (second-password prompt) → box appears map-wide; second character can enter (visitor sees the BUY view) and purchase.
- [ ] Abandon during setup (close window) → shop closes; immediate re-create succeeds.
- [ ] 503 permit use → check-result opens the create dialog; setup → open → employee NPC spawns; owner's trailing dialog-close EXIT does not kill the shop; owner logout leaves it selling.
- [ ] Merchant maintenance re-entry (double-click own NPC path / mode-17 confirm), MAINTENANCE_OFF resumes selling; EXIT from maintenance returns it to Open (or closes when empty).
- [ ] Visitor slots 2–3 render correctly (true-slot position bytes, Q4).
- [ ] Owl search → warp → auto-enter now shows the visitor buy view (task-127 unblocked).
- [ ] ERROR_UNKNOWN notice channel display (mapId%100 FM room + channel name) reads correctly — the channel byte's 0- vs 1-based display is unverified (`hired_merchant_operation.go`).
Scope: player shops (514-family permit) and hired merchants (503-family permit) — the substrate task-127's owl search/warp needs operational to be testable.

---

## 1. Why this audit exists

Task-127 (owl shop search) is functionally complete per `audit.md`, but end-to-end testing requires a working shop to search for — and live testing surfaced that the shop lifecycle itself is broken. Observed symptom on this branch's build:

1. Player uses a 514 permit and creates a shop. Instead of the **setup/maintenance view** (arrange items, then press Open), the client shows the **visitor "open shop" listing view** of the (empty) shop.
2. Closing that window does **not** destroy the shop.
3. The player cannot create another shop until logout/channel change.

All three symptoms are root-caused below with verified evidence. This document is the spec/design addendum for fixing them on this worktree.

### Reference sources

- **Cosmic** (v83 Java server, proven against the same client): `net/server/channel/handlers/PlayerInteractionHandler.java`, `server/maps/PlayerShop.java`, `server/maps/HiredMerchant.java`, `tools/PacketCreator.java`, `net/server/channel/handlers/HiredMerchantRequest.java` (paths under `src/main/java`).
- **IDA v83** (`MapleStory_dump.exe`, v83_Me IDB): `CMiniRoomBaseDlg::OnEnterResultBase` @0x65ec4a, `CPersonalShopDlg::OnEnterResult` @0x6fc45e (branch @0x6fc528), `CEntrustedShopDlg::OnEnterResult` @0x518873 (branch @0x518a7e). Decompiled during this audit — citations below are from live decompile, not memory.
- **Legacy task docs**: `docs/tasks/legacy-character-shop-merchant/`, `legacy-merchant-channel-integration/`, `legacy-merchant-notification-gaps/`, `legacy-merchant-shop-interactions/`, `legacy-merchant-audit-remediation/` (the last is 0/58 tasks executed).

### Reference lifecycle (Cosmic, faithful v83 behavior)

1. **CREATE** (`PLAYER_INTERACTION` action `CREATE`=0, createType 4/5): validates alive, Free-Market map, no chalkboard/event, `canPlaceStore` (no other shop within range, ≥120px from portal), permit present in cash inventory (`countById(itemId) >= 1`). Then:
   - Player shop: `new PlayerShop` with `open=false`, added to map objects, owner gets the room packet (owner view). **No map box broadcast yet** — other players cannot see or enter (`visitShop` rejects "not yet open").
   - Hired merchant: `new HiredMerchant` with `open=false`, registered with world/channel but **not added to the map**; owner gets the merchant room packet (owner view, firstTime=true).
2. **STOCK**: owner adds/removes items in the setup view (`ADD_ITEM`/`PUT_ITEM`). Cash items only allowed while never-yet-opened (merchant).
3. **OPEN** (`OPEN_STORE`=0x0B for player shop; `OPEN_CASH`=0x0E with birthday for merchant): re-validates placement; player shop → broadcast `updatePlayerShopBox` + `setOpen(true)`, optionally consume permit (config `USE_ERASE_PERMIT_ON_OPENSHOP`); merchant → `setOpen(true)`, **now** added to map, `spawnHiredMerchantBox` broadcast, owner detaches (`hiredMerchantOwnerMaintenanceLeave`).
4. **VISIT** (`VISIT`=4 + object id): gated on `open`, capacity 3, blacklist; visitor gets room packet (visitor view + their slot).
5. **EXIT** (`EXIT`=0x0A): visitor → leave slot. Owner of a player shop → full teardown: stock returned to inventory, visitors kicked, `removePlayerShopBox`. **A never-opened player shop is simply destroyed; the permit was never consumed at create, so nothing is lost.** Owner of a merchant with items → merchant keeps running; `CLOSE_MERCHANT`=0x29 is the full close (items to inventory or Fredrick).

## 2. Findings

Severity: 🔴 breaks the core flow · 🟡 wrong/fragile but not the present blocker · ⚪ debt/backlog.

### F1 🔴 ENTER_RESULT "second header byte" is inverted — owner gets the visitor view (the observed symptom #1)

The byte written after roomType+capacity in the enter-result room is the **recipient's position in the room**: `0` = owner, `1..3` = visitor slot. Atlas encodes it as a boolean with the opposite polarity — `1` for the owner, `0` for visitors (`libs/atlas-packet/interaction/room.go:119-126`, callers `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go:588,614`).

Evidence that `0` = owner:

- Cosmic writes `owner ? 0 : 1` in `getPlayerShop` (PacketCreator, `[ROOM(5)][4][4][owner?0:1]`) and sends the owner-only extra block (open-time, firstTime flag, sale ledger, merchant meso) **when the recipient is the owner** in `getHiredMerchant`.
- IDA v83 `CEntrustedShopDlg::OnEnterResult`: the `this[50] == 0` branch is the one that decodes that extra block (`Decode4` @0x518b04, `Decode1` firstTime @0x518b0a, ledger `sub_518EFD` @0x518b53) — i.e. the zero branch is the branch Cosmic feeds to owners. The owner-only management UI open (`CWvsContext::UI_Open`) is gated on `!this[50]` @0x518d3d, and two controls are enabled with `this[50] == 0` @0x518c69/@0x518c85.
- `this+50` is populated from this exact wire byte in `CMiniRoomBaseDlg::OnEnterResultBase` @0x65ec6b; `CPersonalShopDlg::OnEnterResult` branches on it @0x6fc528.

Consequences today:

- Owner (byte=1) → client takes the nonzero branch → **visitor buy view of their own shop** ("open shop listing" window). Exactly symptom #1.
- Visitor (byte=0) → client takes the zero branch → owner management view, plus (merchant) it decodes the owner-only ledger block and pops the owner UI.

Note the wire is **internally consistent** with the inversion: `Room.Encode` writes the ledger block under `if !rm.ownerView` (`room.go:185-196`), so owner and visitor packets both parse without error-38 — the views are just swapped. This is why the byte fixtures pass: encode/decode round-trips can't catch a semantic polarity error. The fix must flip **both** the byte and the block condition, and add a semantic assertion (owner room encodes position byte `0x00`; the owner variant carries the ledger block).

Also fold in while fixing (same code):
- Visitors should get their **actual slot** (1–3), not a constant. Cosmic's `owner?0:1` is a known simplification; Atlas should write the real slot from the visitor registry.
- Re-derive the owner-block field semantics against Cosmic/IDA: Cosmic writes `short 0, short timeOpen, byte firstTime, sold-list, int merchantMeso`; Atlas currently writes `int meso, byte 0 ("not sold out"), ledger, int soldTotal` (`room.go:186-195`). Field count coincides (which is why nothing crashes) but the meanings don't — the `Decode4` is the packed open-time shorts, the `Decode1` is the firstTime flag Cosmic sets `true` at create.
- Comments in `room.go:76-98` and `consumer.go:586-588,611-614` restate the inverted reading and must be corrected — they will otherwise re-poison the next reader.

### F2 🔴 Owner is never registered as an occupant of their own shop — every owner-side op dead-ends (symptoms #2 and #3's trigger)

Every owner in-shop operation resolves the shop via `GetVisitingShop` → `GET /characters/{id}/visiting` → visitor-registry reverse index: OPEN (`services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go:163`), CHAT (`:135`), EXIT (`:146`), PERSONAL_STORE_PUT_ITEM (`:234`), PERSONAL_STORE_REMOVE_ITEM (`:260`), MERCHANT_PUT_ITEM (`:303`), MERCHANT_REMOVE_ITEM (`:329`), MERCHANT_MERCHANT_OFF (`:340`), MERCHANT_EXIT (`:355`). But the only writer of that index is `AddVisitor`, called solely from `EnterShop` (`services/atlas-merchant/atlas.com/merchant/shop/processor.go:884`); `CreateShop` never registers the owner.

So immediately after CREATE (shop in `Draft`):

- `PUT_ITEM` fails → owner cannot stock the shop.
- `OPEN` fails → and even if it resolved, `OpenShop` requires ≥1 listing (`processor.go:358`), which can never be satisfied.
- `EXIT` hits the 404 and returns at `character_interaction.go:148-150` **before** `CloseShop` — the shop is never destroyed (symptom #2).

The natural fix is already half-built: `CreateShop` writes an `ActiveShopEntry` keyed by characterId into the Redis `activeShops` registry (`processor.go:270-281`). Owner-side resolution should consult that registry (owner → own active shop) before/instead of the visitor index. Options:

- **(a) preferred:** merchant service resolves `/characters/{id}/visiting` (or a new sibling endpoint) as activeShops-first, visitor-registry-second. Zero channel-side changes; the EXIT handler's `visiting.CharacterId() == s.CharacterId()` owner/visitor split (`character_interaction.go:151`) keeps working.
- (b) register the owner in the visitor registry at slot 0 on create. Rejected-by-default: slot 0 is semantically "owner" everywhere else (packet slot bytes, eject logic, `MaxVisitors` accounting) and this would corrupt visitor-count logic (`processor.go:879` counts registry entries against `MaxVisitors`).

### F3 🔴 Stranded `Draft` shops block re-creation — personal until relog, hired merchant forever (symptom #3)

`CreateShop`'s one-shop-per-type guard matches any state ≠ `Closed` (`getActiveByCharacterIdAndType`, `services/atlas-merchant/atlas.com/merchant/shop/provider.go:40-52`), so the stranded Draft from F2 trips `ErrShopLimitReached` on every retry. Cleanup coverage:

- Logout reaper closes only `ShopType == CharacterShop && state != Closed` (`services/atlas-merchant/atlas.com/merchant/kafka/consumer/character/consumer.go:52`) → personal-shop Draft clears on relog/CC (matches observed behavior); **hired-merchant Draft is skipped**.
- Expiry reaper matches `state IN (Open, Maintenance)` only (`shop/task.go:32`, `provider.go:79`) → **hired-merchant Draft never expires. It is permanently stuck** (only manual DB/REST intervention clears it).

Fixes (all three, they're complementary):
1. F2 makes owner EXIT reach `CloseShop`, which already permits closing from `Draft` (`processor.go:500` allows `Open|Maintenance|Draft`) — the primary path.
2. Logout reaper should also close `Draft` shops of **both** types (a Draft is an owner-attached setup session; the owner is gone). Open hired merchants keep their current survive-logout semantics.
3. Expiry query should include `Draft` (safety net; `ExpiresAt` is already set for merchants at create, `processor.go:255-258` — personal-shop Drafts are covered by 1+2).

### F4 🔴 The 503 permit-use flow stalls before the create dialog: no ENTRUSTED_SHOP_CHECK_RESULT reply, and the precheck is permanently poisoned

`HiredMerchantOperationHandleFunc` handles `ModeEntrustedShopCheck`, validates, and then only **logs** "permitted" (`services/atlas-channel/atlas.com/channel/socket/handler/hired_merchant_operation.go:36-50`). Cosmic replies with `ENTRUSTED_SHOP_CHECK_RESULT` byte `0x07` (`hiredMerchantBox()` — proceed) or `0x09` (Fredrick retrieval required) — without a reply the client never opens the merchant create dialog. The clientbound writer set exists (`libs/atlas-packet/merchant/clientbound/operation.go` — `OpenShop`, `ErrorSimple`, `ConfirmManage`, …) but only `FreeFormNotice` is ever sent (`consumer.go:436`).

Additionally the precheck's "already has a shop" test uses `GetByCharacterId`, which has **no state filter** (`shop/provider.go:29-38`, REST `GET /characters/{id}/merchants`) — it counts `Closed` rows and both shop types, so once a character has ever had any shop, `len(shops) > 0` refuses forever (`hired_merchant_operation.go:43-46`).

Fix: filter to non-Closed + `HiredMerchant` type (plus the Fredrick-pending check the merchant service already implements at create), and wire the pass/fail replies.

### F5 🔴→✅ RESOLVED BY VERIFICATION: the hired-merchant open op is `OPEN` (0x0B), already wired

IDA (v83): both dialogs send the identical go-live packet after the second-password prompt — `CPersonalShopDlg::OnCorrectSSN2(11)` and `CEntrustedShopDlg::OnCorrectSSN2(11)` each emit `[0x7B][0x0B][0x01]` (@0x6fcbac / @0x5187db). The merchant's later re-open from maintenance is `MAINTENANCE_OFF` 0x27 (`OnGoOut` @0x51925e, SPW-gated variant @0x5187a2), also already wired to `ExitMaintenance`. So the open path never needed new wiring — it only dead-ended because of F2 (owner resolution), fixed in Phase A. The `CASH_TRADE_OPEN` `nProc == 11` birthday arm is NOT the shop-open path and legitimately remains a logged no-op (its sender is elsewhere; unidentified, non-blocking).

### F6 🟡 `GetByCharacterId(...)[0]` used without state/type filtering in live paths

VISIT resolves the target shop as `shops[0]` of the unfiltered-by-state list (`character_interaction.go:122-127`), and merchant maintenance re-entry does the same for the owner (`:187-192`). With a history of closed shops (or one of each type) this can select a `Closed` or wrong-type row. Note VISIT's `EnterShop` does gate on `Open` server-side (`processor.go:866`), so the failure mode is a wrong-shop 404/no-op rather than entering a closed shop — still, resolution should be by state (and, for VISIT, by the field the visitor is standing in; the packet's `SerialNumber` is the owner id, `character_interaction.go:120`).

### F7 🟡 CREATE performs no permit-ownership validation (and consumption is entirely unimplemented)

- Cosmic validates the permit is in the CASH inventory at create (`countById(itemId) >= 1`, error 6 otherwise). Atlas `PlaceShop` trusts the client's `sp.ItemId()` completely (`character_interaction.go:96`) — any client can create shops with no permit, and with an arbitrary `permitItemId` (which also drives the merchant NPC sprite, `consumer.go:562`).
- The permit is **never consumed** anywhere: `permitItemId` is stored/echoed only; the sole inventory `ReleaseAsset` emission is for listing stock (`processor.go:641`). Legacy docs confirm this was deferred and never picked up ("Permit verification (Store Permit 514, Hired Merchant 503)" listed only as an atlas-cashshop integration point, `docs/tasks/legacy-character-shop-merchant/context.md`). Faithful consumption timing is at OPEN, config-gated in Cosmic (`USE_ERASE_PERMIT_ON_OPENSHOP`; player shop path) — see open question Q1. Critically, **create-then-exit must not consume** — which also means F2/F3's destroy-on-exit costs the player nothing, matching the user-expected "destruct on leave" semantics for 514.

### F8 🟡 SHOP_OPENED re-sends the full room to the owner

`handleShopOpenedEvent` both spawns the map object **and** re-sends `ENTER_RESULT` to the owner (`consumer.go:158-164`). In Cosmic, OPEN does not re-send the room — the owner's dialog simply persists; the only new packet is the map-box broadcast. Once F1/F2 make the setup dialog real, this second ENTER_RESULT likely re-creates/resets the owner's dialog at open time. Verify live after F1/F2 land; expected fix is to drop the owner re-send from the opened handler (keep it for maintenance-exit refresh of personal shops, where the legacy notification-gaps task deliberately added it).

### F10 🟡 Hired-merchant owner messages persisted but never surfaced (added 2026-07-14 — should have been a numbered finding in the original audit)

The merchant service persists visitor messages (`messages` table, `SEND_MESSAGE` command / `MESSAGE_SENT` event, `SendMessage` in the shop processor), but the shop REST payload does not include them and the channel room builder always passes an empty message list (`buildMerchantShopRoom`, `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`) — so the owner's management view never shows the messages visitors left while the merchant ran unattended. The wire slot exists (owner-view message list, Decode2 count @0x518888) and encodes correctly; only the data path is missing (REST enrichment of the owner-view shop fetch + builder wiring). Surfaced during A1 implementation as a code comment; promoted here to the findings list where it belonged. Backlog alongside F9 — does not block the shop lifecycle or task-127.

### F9 ⚪ Enumerated-but-unwired surface (backlog, not blocking)

- Enter-error sub-codes defined but never emitted except `FULL` and `UNABLE`; create-failure feedback now exists on this branch (`SHOP_CREATE_FAILED` → miniroom error mapping, `consumer.go:373-400`) — extend as codes get verified.
- Hired-merchant ops with no behavior: ORGANIZE (`character_interaction.go:348-351`), WITHDRAW_MESO (`:363-366`), VIEW_VISIT_LIST / VIEW_BLACK_LIST / blacklist add/remove (`:367-386`).
- `docs/tasks/legacy-merchant-audit-remediation/` is 0/58 executed (provider-write-op layering, missing logout `StatusEventShopClosed` emission, `ExitMaintenance` auto-close signaling). Not re-scoped here; listed for honesty.

## 3. Target flow after remediation (acceptance narrative)

**Player shop (514):**
1. Use permit in an FM room → CREATE → shop `Draft`; owner sees the **management view** (position byte 0), add-item UI, no map box; other players see nothing.
2. Owner stocks items (listings move stock out of inventory), presses Open → `Draft → Open`, map box spawns for everyone, owl search finds the listings.
3. Owner closing the window before opening → shop closed, stock (if any) returned, registry rows cleared, permit untouched; an immediate re-create succeeds.
4. Visitor VISITs → visitor view (position byte = slot), can buy; owner sees visitor enter/leave.
5. Owner EXIT after open → full close: visitors ejected, box despawns map-wide, stock returned.

**Hired merchant (503):**
1. Use permit → entrusted-shop check → server replies proceed/Fredrick-required → create dialog opens → CREATE → `Draft`, owner in management view, nothing on the map.
2. Owner stocks, opens (via the verified serverbound op) → `Draft → Open`, employee NPC spawns map-wide, owner detaches; merchant runs while owner logs off; 24 h expiry → Fredrick.
3. Owner closing the setup window before opening → shop closed (stock returned), re-create immediately possible; logout during `Draft` also cleans up.
4. Maintenance re-entry/exit, visit, buy as already implemented.

**Task-127 unblocked:** with ≥1 `Open` shop holding a listing, the owl search returns it, warp lands, and auto-enter delivers the **visitor** view — which F1 currently corrupts (visitors get the owner view), so owl testing is double-blocked until F1 lands.

## 4. Remediation plan (phased; one branch, this worktree)

### Phase A — unblock the create→stock→open→visit loop
- **A1 (F1):** `room.go` — replace `ownerView bool` with a `position byte` (0 = owner, else actual visitor slot); move the merchant owner-block behind `position == 0`; re-derive its fields (packed open-time shorts, firstTime byte, sold ledger, merchant meso) from Cosmic + IDA; fix the inverted comments; update both room builders in the channel consumer to pass the recipient's real position; update all interaction clientbound fixtures **and add semantic assertions** (owner encode ⇒ byte 0 + ledger block present for merchant rooms). Applies to all versions the interaction codec serves (v48/61/72/79/83/84/87/95/jms — re-run per-version fixtures).
- **A2 (F2):** owner-op resolution via activeShops registry (option (a)): merchant service answers "what shop is character X occupying" as active-own-shop-first, visitor-registry-second. No channel handler changes.
- **A3 (F3):** EXIT-in-Draft now closes via A2; extend logout reaper to close `Draft` shops of both types; extend expiry query to include `Draft`.

Exit criteria: the Player-shop narrative steps 1–5 pass live on a v83 tenant; re-create after abandon works without relog.

### Phase B — hired-merchant bring-up + permit semantics
- **B1 (F4):** wire ENTRUSTED_SHOP_CHECK_RESULT replies (proceed / Fredrick-required / error) with per-version mode bytes IDA-verified (Q3); fix the precheck query filter (non-Closed + type + Fredrick-pending).
- **B2 (F5):** IDA-verify which serverbound op the v83 merchant dialog sends to open (Q2); wire it to `OpenShop`; keep `OPEN` (0x0B) for personal shops.
- **B3 (F7):** validate permit presence at CREATE (inventory check by item id family 503/514 matching roomType — reject mismatches). Per Q1: permits are **never consumed** — validation only, no consumption flow.

Exit criteria: hired-merchant narrative steps 1–3 pass live; employee NPC spawn/despawn already covered by the field-spawn design in this worktree.

### Phase C — correctness polish
- **C1 (F6):** state/type-filtered shop resolution for VISIT and maintenance re-entry.
- **C2 (F8):** drop the owner ENTER_RESULT re-send from SHOP_OPENED after live verification.
- **C3 (F9):** map remaining verified enter-error codes; leave ORGANIZE/WITHDRAW_MESO/blacklist ops and the 0/58 remediation doc as explicitly-listed backlog (not silently dropped).
  - Disposition (2026-07-14): no additional error codes were verified beyond those already mapped (create-failure reasons → portal/proximity/free-market/unable; enter → FULL/UNABLE), so no new mappings landed; F9's backlog list stands as the explicit record of the unwired surface.

Standard gates per phase: `go test -race`, `go vet`, `go build`, `docker buildx bake` for atlas-merchant/atlas-channel/atlas-packet-touching services, redis-key-guard, goroutine-guard, packet fixtures + matrix `--check` where codecs changed, code review before PR.

## 5. Open questions — RESOLVED (owner decisions, 2026-07-14)

- **Q1 — permit consumption rule: NEVER CONSUME.** Permits are durable items; opening a shop consumes nothing (matches Cosmic's default with `USE_ERASE_PERMIT_ON_OPENSHOP` off). B3 shrinks to permit-ownership *validation* only — no consumption saga, no refund concerns. Create-then-abandon trivially costs nothing.
- **Q2 — merchant open op.** IDA-verify during B2 which serverbound op v83 `CEntrustedShopDlg` sends to go live (`OPEN` 0x0B vs `CASH_TRADE_OPEN` 0x0E nProc 11); wire the verified path.
- **Q3 — ENTRUSTED_SHOP_CHECK_RESULT mode bytes per version: VERIFIED on all eight feature IDBs.** `OnEntrustedShopCheckResult` switch cases are 7,8,9,10,11,13,14,15,16,17,18 on v61 @0x848c1c, v72 @0x91ff18, v79 @0x971dd8, v83 @0xa27d75, v84 @0xa73538, v87 @0xabf9ea, v95 @0x9ffcb0 — matching the seed-template operations tables unchanged. **jms185 @0xb0ee59 has NO case 18**: a FREE_FORM_NOTICE send on jms is silently ignored by the client (default arm returns) — benign drop; the Fredrick reminder needs a different vehicle on jms if ever required there. All modes B1 emits (7/8/9/11) exist on every version.
- **Q4 — visitor position bytes: TRUE SLOT (1–3).** Encode the visitor's actual slot from the visitor registry (client stores avatars slot-indexed, `OnEnterResultBase` @0x65ecac). Verify slots 2–3 render correctly during Phase A live testing.
- **Q5 — Draft-merchant logout policy: CLOSE ON LOGOUT.** A `Draft` is an owner-attached setup session; logout closes Draft shops of both types (staged stock follows the existing CloseShop return paths). `Open` hired merchants keep their survive-logout semantics unchanged.
