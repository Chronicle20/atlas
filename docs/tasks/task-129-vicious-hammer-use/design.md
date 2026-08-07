# Vicious Hammer Use — Design

Task: task-129-vicious-hammer-use
Status: Draft (design phase)
Created: 2026-07-02
Depends on: PRD `docs/tasks/task-129-vicious-hammer-use/prd.md`

---

## 1. Summary

The Vicious Hammer (item 5570000, classification 557) adds +1 upgrade slot to a target
equip, capped at 2 hammers per equip. This design implements the flow end-to-end across
all supported GMS tenant versions.

The central finding of the design phase is that **the client-side flow is a stateful,
two-phase "gauge" dialog, not a single request/response.** This was reverse-engineered
live from the v83 IDB (port 13342, `MapleStory_dump.exe`) and cross-verified against the
v95 IDB (port 13341, `GMS_v95.0_U_DEVM.exe`), which carries named symbols that confirm the
v83 inferences. This answers PRD Open Questions 1, 2, and 3 directly with IDA evidence, and
it is the primary driver of the architecture below.

Everything downstream of the packet layer is already plumbed: `slots` and `hammersApplied`
travel on the existing `MODIFY_EQUIPMENT` inventory command, the v84+ equip encoder already
writes `hammersApplied`, and the atlas-consumables scroll flow is a near-exact structural
precedent for consume+mutate atomicity. The new work is almost entirely: the two serverbound
codecs, the real (mode-prefixed) clientbound body, one new consumables command, and the
channel wiring between them.

---

## 2. Verified client protocol (IDA evidence)

### 2.1 The two-phase gauge flow

The Vicious Hammer UI is `CUIItemUpgrade`. Its lifecycle:

1. **Double-click hammer** → `CWvsContext::SendConsumeCashItemUseRequest` (v83 `0xa0a63f`),
   jumptable **case 66** (`0xa0b4f9`), constructs the `CUIItemUpgrade` dialog
   (v83 `sub_82A754` / v95 ctor `CUIItemUpgrade::CUIItemUpgrade(COutPacket, J, J)` `0x7bfd40`).
   The constructor is handed a **pre-built cash-item-use `COutPacket`** which it stashes in
   the dialog (`this+1512`, field `m_oPacket`). No packet is sent yet — the dialog just opens.
   Confirmed Vicious-Hammer identity via string `SP_5049_UI_UIWINDOWIMG_VICIOUSHAMMER_BACKGRND`
   used in the constructor.

2. **Drop target equip into dialog** → sets `m_pSelectedItem`, `m_nItemTI`
   (inventory type-index), `m_nSlotPosition` (target equip slot).

3. **Click "Upgrade" (button id 2000)** → `CUIItemUpgrade::OnButtonClicked`
   (v83 `sub_82AED3` / v95 `0x7c0ca0`). If a target is selected and no request is in flight,
   it appends to the pre-built cash-use packet and sends it:

   ```
   Encode4(m_oPacket, m_nItemTI)        // target inventory type-index
   Encode4(m_oPacket, m_nSlotPosition)  // target equip slot position
   Encode4(m_oPacket, get_update_time)  // trailing updateTime  <-- the TODO at
                                        // character_cash_item_use.go:106
   SendPacket(m_oPacket)                // = CASH_ITEM_USE opcode, extended body
   m_nState = 1; m_bRequestSent = 1     // gauge starts
   ```

   This is **Serverbound Packet A** — the existing `CASH_ITEM_USE` opcode (already routed to
   `CharacterCashItemUseHandle`) with three trailing int32s for the hammer arm.

4. **Server must respond with the "open/arm" result** (see §2.2, `else` arm). This sets
   `m_nResultState = 1` and stores the round-trip token. Without it the gauge never confirms.

5. **Gauge fills** (~1s, client-side animation; `m_nState` 1→2 via the gauge tick), then
   `CUIItemUpgrade::Update` (v83 `0x82ae28` / v95 `0x7bef50`) fires once `m_nState==2 &&
   m_nResultState==1 && now>m_tEnd`:

   ```
   COutPacket(ITEM_UPGRADE_UPDATE)      // v83 0x104 / v95 0x128 (=296)
   Encode4(m_nReturnResult)             // echo of the open-arm mode byte
   Encode4(m_nResult)                   // echo of the server's round-trip token
   m_nState = 3
   ```

   This is **Serverbound Packet B** — `ITEM_UPGRADE_UPDATE` / `CUIItemUpgrade::Update`,
   currently ❌ in every version of the matrix. Body = two int32s, both echoed from the
   server's open-arm response.

6. **Server applies the hammer and responds with the terminal result** (mode 61 success or
   62 failure). Both close the dialog.

**Consequence for the server design:** for Packet B (and thus promotion of the
`ITEM_UPGRADE_UPDATE` matrix cell, an explicit PRD acceptance criterion) to ever be sent by
the client, the server **must** answer Packet A with the non-terminal "open/arm" result, not
a direct success. The two-phase protocol is mandatory, not optional. See §4 for the decision.

### 2.2 Clientbound `VICIOUS_HAMMER` is a mode-prefixed dispatcher

`CField::OnItemUpgrade` (v83 `0x537f8c` / v95 `0x52a430`) is a thin vtable forwarder to
`CUIItemUpgrade::OnPacket` (v83 `sub_82B2C3`, reached via `sub_82B2AD` which gates on
`nType == 354 == 0x162`). The body reads a **leading mode byte** (`Decode1`) and branches:

| Mode byte | Arm | Body after mode byte | Meaning |
|-----------|-----|----------------------|---------|
| `61` (0x3D) | **Success** | `int32 flag` (0 = success; non-0 = "unknown error %d") | closes dialog, "Increased available upgrade by 1. N upgrades are left" where N = `2 - hammerCount` |
| `62` (0x3E) | **Failure** | `int32 errorCode` | closes dialog with a specific notice (see §6) |
| any other | **Open/arm** | `int32 token` + `int32 hammerCount` | arms confirm: `m_nResult=token`, `m_nResult(count)`, `m_nResultState=1` |

Mode `61` is hard-coded identically in both v83 (`sub_82B2C3`: `if(v5==61)`) and v95
(`CUIItemUpgrade::ShowResult` `0x7bec20`: `m_nReturnResult != 61`), and `62` likewise — so
the mode bytes appear **version-stable** (unlike the opcode tables). Per project convention
(`feedback_dispatcher_config_drive_all_modes`, owner-overruled task-103) they are still to be
**config-resolved from a per-version `operations` table**, never hard-coded literals.

The "open/arm" mode value is server-chosen: the client accepts *any* byte that is not 61 or
62 and simply round-trips it back in Packet B. It is a pure token; the server never needs to
interpret it. See §9 Open Question OQ-1 for picking its value.

### 2.3 Cross-version opcode map

| Version | CASH_ITEM_USE (Packet A) | ITEM_UPGRADE_UPDATE serverbound (Packet B) | VICIOUS_HAMMER clientbound | CashSlotItemType (557) |
|---------|--------------------------|--------------------------------------------|----------------------------|------------------------|
| v83 | existing (routed) | `0x104` | `0x162` | 66 |
| v84 | existing (routed) | `0x104` | `0x169` | 66 |
| v87 | existing (routed) | `0x112` | `0x177` | 66 |
| v92 | existing (routed) | *derive from lineage — no IDB* | *derive — no IDB* | 66 |
| v95 | existing (routed) | `0x128` | `0x1A9` | 67 |
| jms | existing (routed) | `0x114` | **absent from registry** | — |

Serverbound/clientbound opcodes for v83/v84/v87/v95/jms are from
`docs/packets/registry/gms_v*.yaml` / `jms_v185.yaml`. v83 (0x104/0x162) and v95
(0x128/0x1A9) are byte-verified live this phase. v84/v87/jms are verified against the
checked-in exports (`docs/packets/ida-exports/gms_v84.json`, `gms_v87.json`,
`gms_jms_185.json`) during implementation. **v92 has no IDB or export** — its opcodes/bodies
are interpolated from the template lineage and the cells are marked honestly (never claimed
✅ from interpolation), per PRD §4.1.

---

## 3. Architecture

```
                         ┌───────────────────────── atlas-channel ─────────────────────────┐
 (double-click hammer)   │                                                                  │
        client ──────────┼── CASH_ITEM_USE (type 66/67, +itemTI,+slotPos,+updateTime) ──▶   │
   [Packet A]            │        CharacterCashItemUseHandle  (new hammer arm)              │
                         │          • read target equip from local compartment cache        │
                         │          • cheap pre-check (exists / is equip / <2 / not excl.)   │
        client ◀─────────┼── VICIOUS_HAMMER open-arm (token=slotPos, count) ◀──             │
   (gauge fills ~1s)     │          OR VICIOUS_HAMMER failure (mode 62) if pre-check fails   │
                         │                                                                  │
        client ──────────┼── ITEM_UPGRADE_UPDATE (echo mode, echo token) ──────────────▶    │
   [Packet B]            │        ItemUpgradeUpdateHandle  (new handler)                     │
                         │          • emit REQUEST_VICIOUS_HAMMER command  ─────────────────┼──┐
                         │                                                                  │  │
                         │        HammerResult Kafka consumer  ◀────────────────────────────┼──┼─┐
        client ◀─────────┼── VICIOUS_HAMMER success (61) / failure (62)                     │  │ │
                         └──────────────────────────────────────────────────────────────────┘  │ │
                                                                                                │ │
              ┌───────────────────────── atlas-consumables ──────────────────────────┐         │ │
              │  RequestViciousHammer  ◀──────────────────────────────────────────────┼─────────┘ │
              │    • load char + inventory, authoritative validate (§6)               │           │
              │    • reserve the hammer (compartment.Reserves)                         │           │
              │    OneTime callback  ConsumeViciousHammer:                             │           │
              │      • equipable.ChangeStat(target, AddSlots(1), AddHammersApplied(1)) ┼──▶ MODIFY_EQUIPMENT ──▶ atlas-inventory
              │      • cpp.ConsumeItem(hammer)                                         │           │
              │    • emit VICIOUS_HAMMER result event (success / error) ───────────────┼───────────┘
              └────────────────────────────────────────────────────────────────────────┘
```

### 3.1 libs/atlas-packet

- **Serverbound `ItemUpgradeUpdate` codec** (new, `field/serverbound/` or
  `item/serverbound/` — mirror the package of the existing item-upgrade family): decode two
  int32s (`returnResult`, `result`). Byte-fixtured per version. Promotes the
  `ITEM_UPGRADE_UPDATE` serverbound matrix cells.
- **Extend the cash-item-use serverbound decode** so the hammer path can read the three
  trailing int32s (`itemTI`, `slotPosition`, `updateTime`). Decode them in the channel
  handler arm rather than unconditionally, since only type 66/67 carries them (resolves the
  `// TODO for v83 there is a trailing updateTime` at `character_cash_item_use.go:106`).
- **Clientbound `ViciousHammer` → mode-prefixed dispatcher family** (per
  `docs/packets/DISPATCHER_FAMILY.md`): replace the current empty body
  (`libs/atlas-packet/field/clientbound/vicious_hammer.go`) with discrete structs per arm —
  `Open{token, hammerCount}`, `Success{flag}`, `Failure{code}` — each encoding
  `WithResolvedCode("operations", KEY)` for its mode byte plus its int32 body. Add a
  `docs/packets/dispatchers/vicious_hammer.yaml` describing the arms and register it in the
  lint baseline transition. Byte-fixture every arm (no mode-byte-only enumeration —
  `feedback_dispatcher_mode_byte_is_false_pass`).

### 3.2 atlas-channel

- **`CharacterCashItemUseHandle` hammer arm** (`socket/handler/character_cash_item_use.go`):
  add the `CashSlotItemType 66/67` case (name the constants — currently unnamed at ~line
  114). Decode the trailing `itemTI/slotPosition/updateTime`. Read the target equip from the
  compartment the channel already tracks; run the cheap pre-check; write either the
  `VICIOUS_HAMMER` open-arm (token = `slotPosition`, count = `hammersApplied`) or the failure
  arm. Handler entry needs a `LoggedInValidator` in every seed template
  (`bug_socket_handler_missing_validator_silently_dropped`).
- **`ItemUpgradeUpdateHandle`** (new handler): decode Packet B; emit a
  `REQUEST_VICIOUS_HAMMER` command to atlas-consumables carrying `characterId`, the hammer's
  cash-compartment slot, the target `itemTI + slotPosition` (recovered from the echoed
  token), and a `transactionId`. Register with `LoggedInValidator` in all seed templates.
- **Hammer-result Kafka consumer** (new, mirrors the scroll consumer at
  `kafka/consumer/consumable/consumer.go:83`): on the consumables `VICIOUS_HAMMER` result
  event, invoke the `ViciousHammer` writer with the success or failure arm, routed to the
  character's session/field.

### 3.3 atlas-consumables

- **`AddHammersApplied(1)` change** (`equipable/processor.go`, next to `AddSlots` at line
  124): one-liner wrapping the existing `ModelBuilder.AddHammersApplied` (`asset/builder.go:275`).
  The producer already forwards `HammersApplied` (`equipable/producer.go:45`).
- **`RequestViciousHammer` + `ConsumeViciousHammer`** (`consumable/processor.go`, modeled on
  `RequestScroll`:515 / `ConsumeScroll`:606): load character + inventory, authoritative
  validate (§6), reserve the hammer via `compartment.Reserves` + a OneTime status consumer
  keyed by `transactionId`; the reservation callback applies
  `equipable.ChangeStat(target, AddSlots(1), AddHammersApplied(1))` then
  `ConsumeItem(hammer)`, and emits the `VICIOUS_HAMMER` result event. On any failure emit the
  error/failure event (which releases the reservation) — this is the atomic boundary.
- New Kafka message types: command `REQUEST_VICIOUS_HAMMER`, event `VICIOUS_HAMMER`
  (`kafka/message/consumable/kafka.go`), plus channel-side producer/consumer wrappers.

### 3.4 atlas-inventory

**No change.** `MODIFY_EQUIPMENT` already sets both `Slots` and `HammersApplied` in one DB
update and re-emits the equip (`kafka/consumer/compartment/consumer.go:358,363` →
`ModifyEquipmentAndEmit`). Verify-only: confirm both fields persist and re-encode after a
hammer.

### 3.5 Tenant configuration

- Add the two serverbound handler entries (both with `LoggedInValidator`) and the
  `ViciousHammer` writer stays registered but now with a real (dispatcher) body, to every
  supported version's `services/atlas-configurations/seed-data/templates/template_gms_*.json`.
- Populate the per-version `operations` mode table for the `VICIOUS_HAMMER` dispatcher in
  every template it applies to (missing table → `ResolveCode` returns 99 → client crash,
  `bug_operations_mode_tables_missing_v87_v95_jms`).
- Seed templates apply only at tenant creation → document and perform the live-tenant config
  patch + channel restart for existing tenants
  (`bug_new_opcodes_not_in_live_tenant_config`).

---

## 4. Key decision: two-phase (faithful) vs single-phase (shortcut)

Because the target equip and updateTime already ride on Packet A (cash-use), a server *could*
validate + consume + apply entirely on Packet A and reply with the terminal success (mode 61),
skipping the confirm packet. Two designs:

**Design A — single-phase (apply on cash-use).** Simplest: one handler, no
`ITEM_UPGRADE_UPDATE` codec, no round-trip token. But the client only sends
`ITEM_UPGRADE_UPDATE` after receiving a non-terminal open-arm; a direct mode-61 closes the
dialog and the confirm packet is **never sent**. This means the `ITEM_UPGRADE_UPDATE`
serverbound cells **cannot be promoted** — violating a PRD §4.6 / §10 acceptance criterion —
and the retail gauge animation is skipped.

**Design B — two-phase (retail-faithful).** ✅ **Recommended.** The client's real behavior.
Packet A → open-arm (validate-light, arm the gauge); Packet B → apply + consume + terminal
result. Promotes `ITEM_UPGRADE_UPDATE`, preserves the gauge, matches every acceptance
criterion. Cost: a second handler and a round-trip token (kept **stateless** by encoding the
target slot into the token the client echoes — no server-side pending-transaction state).

We take **Design B**. It is what the client actually does (verified), and it is the only
option that satisfies the PRD as written. The extra complexity is small because the token
makes the confirm self-describing.

### 4.1 Atomicity boundary

All consumption + mutation happens on **Packet B**, inside the consumables
reserve→consume-callback (the scroll model). Packet A performs **no** mutation — only a
read-only pre-check to arm or early-reject the dialog. Therefore:
- Abandoning the dialog mid-gauge (logout, cancel) consumes nothing.
- The authoritative validation re-runs on Packet B against fresh state (idempotency /
  anti-replay, PRD §8): a duplicate/replayed confirm re-checks `hammersApplied < 2` at
  execution time and is rejected if already applied.
- `ExecuteTransaction` is a known no-op (`bug_execute_transaction_noop`); rely on the
  reserve→consume ordering + compensating error event for atomicity exactly as `ConsumeScroll`
  does, **not** on a DB transaction wrapper.

---

## 5. Data flow (happy path)

1. Client: double-click hammer → dialog opens (no packet).
2. Client: drop equip, click Upgrade → **CASH_ITEM_USE** (+itemTI,+slotPos,+updateTime).
3. Channel: `CharacterCashItemUseHandle` hammer arm → read target equip → pre-check passes →
   send **VICIOUS_HAMMER open-arm** (token=slotPos, count=hammersApplied).
4. Client: gauge fills → **ITEM_UPGRADE_UPDATE** (echo mode, echo token).
5. Channel: `ItemUpgradeUpdateHandle` → emit **REQUEST_VICIOUS_HAMMER** to consumables.
6. Consumables: validate → reserve hammer → callback: `ChangeStat(AddSlots(1),
   AddHammersApplied(1))` → **MODIFY_EQUIPMENT** to inventory → `ConsumeItem(hammer)` → emit
   **VICIOUS_HAMMER success** event.
7. Inventory: `MODIFY_EQUIPMENT` persists slots+1/hammers+1, re-emits the equip → existing
   inventory-update writers refresh the client's equip window.
8. Channel: hammer-result consumer → send **VICIOUS_HAMMER mode 61** → dialog shows
   "Increased available upgrade by 1. N left" and closes.

Failure paths short-circuit to a **VICIOUS_HAMMER mode 62** (with the §6 code) at either the
Packet-A pre-check (channel) or the Packet-B authoritative validation (consumables →
failure event → channel writer).

---

## 6. Eligibility & error taxonomy (IDA-verified)

The client's mode-62 switch (`sub_82B2C3`) enumerates the server's rejection reasons, which
tells us exactly what the server must validate and what code to send:

| Error code (int32) | Client string | Server rule |
|--------------------|---------------|-------------|
| `1` | "The item is not upgradable" | target is not a hammer-eligible equip (non-equip, or an equip WZ-flagged not-upgradable / zero base slots / cash equip — **verify the exact predicate against WZ during implementation**) |
| `2` | "2 upgrade increases have been used already" | `hammersApplied >= 2` — **confirms the cap = 2** |
| `3` | "You can't use Vicious Hammer on Horntail Necklace" | specific item exclusion (Horntail Necklace; identify the exact item id(s) from WZ during implementation) |
| default | "Unknown error %d" (%d = echoed token) | any other server rejection |

The **cap of 2** is now IDA-verified two independent ways (the `2 - hammerCount`
"upgrades-left" display and error code 2), resolving PRD Open Question 3 — no need to trust
Cosmic. The success path also uses `2 - count`, so the open-arm's `hammerCount` field must be
the target's current `hammersApplied` for the "N left" text to render correctly.

**Eligibility predicate (exclusion rules)** — the WZ-derived "not upgradable" set (code 1)
and the Horntail-Necklace item id (code 3) must be verified against local WZ data during
implementation, per CLAUDE.md "Verification Over Memory". Do not copy Cosmic's list on faith.
This is the one validation-input still requiring WZ work; it is producible (WZ is local) and
therefore not a blocker.

---

## 7. Target-equip addressing

Packet A carries `itemTI` (inventory type-index) + `slotPosition`. `slotPosition` is a signed
slot: **negative = an equipped item, positive = an equip in the equip inventory**. So the
dialog can target both equipped and inventoried equips (resolves PRD Open Question 4 — verify
the exact sign convention and whether the equip tab is the only valid `itemTI` during
implementation via the `PutItem`/`OnDragDrop` decompile). The channel resolves this slot
against the character's equipable compartment to obtain the asset; the same slot is echoed
through the token so consumables re-resolves it authoritatively on Packet B.

---

## 8. Versioning & verification plan

- **v83, v95:** protocol byte-verified live this phase. Write byte-fixture tests with
  `packet-audit:verify` markers for both serverbound codecs and all three clientbound arms;
  pin evidence records; regenerate the matrix (`ITEM_UPGRADE_UPDATE` serverbound → ✅,
  `VICIOUS_HAMMER` clientbound stays ✅ with the real body).
- **v84, v87, jms:** verify opcodes + the same dialog family against the checked-in exports
  (`docs/packets/ida-exports/gms_v84.json`, `gms_v87.json`, `gms_jms_185.json`). For jms see
  OQ-2.
- **v92:** no IDB/export. Derive opcodes from the template lineage; mark the v92 cells with
  whatever state the evidence honestly supports (not ✅ from interpolation), per PRD §4.1.
- Run `packet-audit dispatcher-lint` (+ `matrix`/`fname-doc`/`operations --check`) to exit 0
  for the new dispatcher family.

---

## 9. Open questions (carry into planning / implementation)

- **OQ-1 — open-arm mode byte value.** The client accepts any non-{61,62} byte and
  round-trips it; the server picks it. Choose a fixed value (e.g. from a reference GMS server
  capture or a deliberate project constant) and record it in the `operations` table. Not a
  blocker — any valid non-terminal value works; pick and document.
- **OQ-2 — jms inclusion.** jms has `ITEM_UPGRADE_UPDATE` serverbound (0x114) but **no**
  clientbound `VICIOUS_HAMMER` registry row. Either jms routes the result through a different
  clientbound op or the jms hammer flow differs. Check `gms_jms_185.json` during
  implementation; if the result op is genuinely absent, jms hammer *result* cannot be sent →
  keep jms out of scope (PRD non-goal) and document why.
- **OQ-3 — exact WZ eligibility predicate** for error code 1 (not-upgradable set) and the
  Horntail-Necklace item id(s) for code 3. Producible from local WZ; verify, don't copy
  Cosmic.
- **OQ-4 — `itemTI`/`slotPosition` exact semantics** (which inventory types the dialog
  accepts; sign convention for equipped vs inventory). Confirm from the `PutItem` /
  `OnDragDrop` decompile when writing the Packet A decode.

None of these block starting the packet-layer and consumables work; they are per-version /
per-fixture verification items resolvable during implementation.

---

## 10. Testing

- **Packet layer:** byte-fixture tests for `ItemUpgradeUpdate` serverbound and each
  `ViciousHammer` clientbound arm, v83 + v95 minimum, per `VERIFYING_A_PACKET.md`.
- **Consumables:** unit tests (Builder pattern, no `*_testhelpers.go`) for validation (cap,
  eligibility, ownership), the `AddSlots+AddHammersApplied` change pair, and the
  reserve→consume→error compensation path.
- **Full-stack per version:** double-click → drop → upgrade consumes exactly one hammer,
  slots+1, hammersApplied+1, equip window updates without relog; a 3rd hammer is rejected
  (mode 62 code 2, no consumption); ineligible target rejected (mode 62 code 1/3); persistence
  survives relog and encodes in v84+ equip packets.
- **CLAUDE.md gates:** `go test -race`, `go vet`, `go build` clean per changed module;
  `docker buildx bake atlas-channel atlas-consumables` (and any lib-touch rebuild);
  `tools/redis-key-guard.sh` clean; `packet-audit dispatcher-lint` exit 0.

---

## 11. Alternatives considered

- **Single-phase apply-on-cash-use (Design A, §4):** rejected — cannot promote
  `ITEM_UPGRADE_UPDATE`, drops the gauge, violates PRD acceptance criteria.
- **Reuse `RequestItemConsume` (pet-consumable arm) instead of a scroll-style flow:**
  rejected — it only consumes, it does not mutate the target equip; the hammer needs the
  reserve→consume→`ChangeStat` machinery that only the scroll flow provides.
- **Field-effect saga model (the type-16 cash arm):** rejected as heavier than needed; the
  scroll reserve/consume callback already gives single-service atomicity for consume+mutate.
- **Hard-coding modes 61/62 as literals** (they are version-stable): rejected — project
  convention (`feedback_dispatcher_config_drive_all_modes`) requires config-resolved modes
  even for stable arms; keeps the dispatcher uniform and lint-clean.
- **Stateful pending-transaction between Packet A and B:** rejected in favor of encoding the
  target slot into the round-trip token, keeping the confirm handler stateless.
