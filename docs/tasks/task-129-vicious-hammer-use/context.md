# task-129 Vicious Hammer Use — Implementation Context

Companion to `plan.md`. Key files, locked decisions, and dependencies an
implementer (or reviewer) needs without re-deriving the design phase.

## The protocol in one paragraph

The hammer UI (`CUIItemUpgrade`) is a stateful two-phase gauge dialog. Packet A
is the existing `CASH_ITEM_USE` opcode with a 3-int32 tail (`itemTI`,
`slotPosition`, `updateTime`) appended by the dialog's Upgrade button. The
server must answer with the non-terminal `VICIOUS_HAMMER` open-arm (any mode
byte ∉ {61,62}, then `token` + `hammerCount`) or the client never confirms.
After the ~1s gauge, the client sends Packet B (`ITEM_UPGRADE_UPDATE`): two
int32s echoing the open mode byte and the token. The server then applies the
hammer atomically and answers mode 61 (success, flag=0) or 62 (failure, code).
All consumption/mutation happens on Packet B; Packet A is read-only pre-check.

## Locked decisions

| Decision | Value | Why |
|---|---|---|
| Flow shape | Two-phase (Design B) | Client never sends Packet B after a direct mode-61; only path that promotes the `ITEM_UPGRADE_UPDATE` matrix cells (design §4) |
| Open-arm mode byte (OQ-1) | `OPEN = 0` | Client accepts any byte ∉ {61,62}; recorded in `docs/packets/dispatchers/vicious_hammer.yaml` + all four templates |
| Round-trip token | `uint32(uint16(hammerSlot))<<16 \| uint32(uint16(equipSlot))` | Keeps Packet B stateless — both slots recovered from the client echo; no server-side pending state |
| TransactionId origin | Generated in atlas-consumables (`RequestViciousHammer`) | Mirrors `RequestScroll` exactly; the channel command does not carry one (deviation from design §3.2 wording, same guarantees) |
| Hammer identity | `item.GetClassification(id) == item.ClassificationViciousHammer` (557, new constant) | Covers all `0557.img` ids; v83 WZ contains only 5570000 |
| Cap | 2 (`maxHammersApplied`) | IDA-verified twice (error code 2 + `2 - count` display) |
| Eligibility (error 1) | equip-data `slots == 0` OR `cash == true`; data-fetch failure ⇒ reject | atlas-data equipment reader fields `tuc`/`cash`; fail-closed |
| Horntail exclusion (error 3) | `templateId == 1122000` | WZ-verified (`String.wz/Eqp.img.xml`, GMS 83.1 dump under repo `tmp/<uuid>/GMS/83.1/`); it has `tuc=3` so slots-based logic cannot catch it |
| Internal errors | code 0 → client "Unknown error 0", dialog closes | Never leaves the dialog stuck |
| Failure event | Dedicated `VICIOUS_HAMMER` event, NOT the generic `ERROR` event | Generic error → enable-actions stat packet; the hammer dialog needs mode-62 |
| jms (OQ-2) | Out of scope | No `VICIOUS_HAMMER` row in `jms_v185.yaml`; zero `CUIItemUpgrade` fns in `gms_jms_185.json` — result packet cannot exist, so the serverbound op is not routed |
| v92 | Nothing added | No registry/IDB/export; `template_gms_92_1.json` is a login-only stub (37 handlers, no `CharacterCashItemUseHandle`, no `ViciousHammer` writer) — no attachment point |

## Verified opcode / address table

| Version | Packet B serverbound | VICIOUS_HAMMER clientbound | Forwarder addr | Sender addr | IDB status |
|---|---|---|---|---|---|
| v83 | `0x104` (260) | `0x162` | `0x537f8c` | `0x82ae28` | live (design-verified) |
| v84 | `0x104` (260) | `0x169` | `0x544395`* | unknown — locate via byte signature | NOT loaded 2026-06-30 — gate |
| v87 | `0x112` (274) | `0x177` | `0x55fa12`* | unknown — locate via byte signature | NOT loaded 2026-06-30 — gate |
| v95 | `0x128` (296) | `0x1A9` | `0x52a430` | `0x7bef50` | live (design-verified) |
| jms | `0x114` (276) — unrouted | absent | — | — | out of scope |

\* from the retired `FieldViciousHammer` markers/evidence; re-confirm live before pinning.

**`CUIItemUpgrade::Update` exists in NO checked-in export** — every serverbound
evidence pin requires an export splice from a live IDB first (surgical, never a
regeneration; `VERIFYING_A_PACKET.md` §10).

## Key files

Packet lib (`libs/atlas-packet`):
- `cash/serverbound/item_use.go` — shared `ItemUse` prefix (v95 reads updateTime first); `item_use_field_effect.go` is the tail-codec pattern to mirror
- `field/clientbound/vicious_hammer.go` — currently the empty stub; rewritten to 3 discrete structs (`ViciousHammerWriter` const must survive — referenced by channel main.go writers list line 765)
- `field/field_effect_body.go` — the body-func pattern (`WithResolvedCode("operations", KEY, …)`, `resolve.go:13`)
- `field/clientbound/mts_operation.go` + `party/clientbound/invite_test.go` — discrete-per-mode struct + byte-fixture test references

Channel (`services/atlas-channel/atlas.com/channel`):
- `socket/handler/character_cash_item_use.go` — type-66/67 arm goes here; handler currently discards `wp`; the 557 branch (line ~469) gets the named constants; the line-108 TODO is resolved by the tail decode
- `character/processor.go:197` — `GetEquipableInSlot` (equip compartment includes equipped items at negative slots)
- `main.go:867` — handlerMap registration point (`fieldsb` alias already imported)
- `kafka/consumer/consumable/consumer.go` — scroll consumer is the result-consumer pattern (`handleScrollConsumableEvent:83`)
- `consumable/{processor,producer}.go` + `kafka/message/consumable/kafka.go` — command plumbing mirrors `RequestScrollUse`

Consumables (`services/atlas-consumables/atlas.com/consumables`):
- `consumable/processor.go` — `RequestScroll:515` / `ConsumeScroll:606` are the structural precedent; `ConsumeError:280` shows reservation-cancel + event emission (hammer gets its own `ViciousHammerError` on the new event type)
- `compartment/processor.go` — `Reserves`, `RequestReserve:33`, `ConsumeItem:43`, `CancelItemReservation:51`, `Consume:37`
- `kafka/once/compartment/once.go` — `ReservationValidator`
- `equipable/processor.go:124` (`AddSlots`) + `asset/builder.go:275` (`AddHammersApplied`) + `equipable/producer.go:45` (already forwards `HammersApplied` on `MODIFY_EQUIPMENT`)
- `data/equipable/{model,rest}.go` — `Cash` is on the RestModel (line 26) but NOT extracted into the domain model; plumbed in Task 6
- `kafka/consumer/consumable/consumer.go` — command-handler registration pattern

Inventory: **no change** — `kafka/consumer/compartment/consumer.go:358,363` already persists `Slots` + `HammersApplied` in one `ModifyEquipmentAndEmit`.

Tooling / docs:
- `docs/packets/DISPATCHER_FAMILY.md` — the enforced invariants (INV-1..5); baseline is EMPTY and only shrinks — new families must be born discrete-per-mode
- `tools/packet-audit/cmd/run.go:2421` — the `CField::OnItemUpgrade` candidate case to replace
- `docs/packets/dispatchers/field_effect.yaml` / `mts_operation.yaml` — yaml shape (jms omitted with a comment when registry-absent)
- `docs/packets/audits/VERIFYING_A_PACKET.md` — §9 serverbound three-artifact rule (marker + evidence + report, op routed in template); §10 export-splice hygiene
- Templates: `services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,95}_1.json` — handler entries (validator mandatory) + writer `options.operations`

## Stale artifacts retired in Task 5

The empty-body `ViciousHammer` codec was verified T1 (✅ v83/v84/v87/v95,
STATUS.md:461). Its 4 markers (in `vicious_hammer_test.go`), 4 evidence records
(`docs/packets/evidence/gms_v*/field.clientbound.FieldViciousHammer.yaml`) and
any `FieldViciousHammer.{json,md}` audit reports must be removed with the
struct or `matrix --check` reports dangling/orphan artifacts. Cells honestly
degrade between Task 5 and Tasks 14/15 — expected.

## External gates (may block)

1. **IDA instances rotate.** `list_instances` + match binary NAME every time.
   v83 dump + v95 `GMS_v95.0_U_DEVM.exe` were loaded as of 2026-06-30; v84/v87
   were NOT. Task 15 must STOP and ask the user to load them if absent — never
   interpolate addresses or fake evidence.
2. **`matrix --check` pre-existing conflict backlog** exits non-zero on main —
   the bar is "no NEW problems mentioning your packets", not exit 0.
3. **Live-tenant rollout** (config PATCH + channel restart) is documented in
   `rollout.md` (Task 16) and performed at deploy time, not during this branch.

## Kafka topology (new pieces)

```
channel ItemUpgradeUpdateHandle
  └─ COMMAND_TOPIC_CONSUMABLE : Command[RequestViciousHammerBody]{hammerSlot, equipSlot}  (REQUEST_VICIOUS_HAMMER)
consumables handleRequestViciousHammer → RequestViciousHammer
  ├─ COMMAND_TOPIC_COMPARTMENT : reserve hammer (cash compartment)
  ├─ (OneTime on EVENT_TOPIC_COMPARTMENT_STATUS keyed by transactionId+itemId) → ConsumeViciousHammer
  │    ├─ COMMAND_TOPIC_COMPARTMENT : MODIFY_EQUIPMENT (slots+1, hammersApplied+1 — one message)
  │    ├─ COMMAND_TOPIC_COMPARTMENT : consume hammer
  │    └─ EVENT_TOPIC_CONSUMABLE_STATUS : Event[ViciousHammerBody]{success:true}
  └─ on any failure: cancel reservation + Event[ViciousHammerBody]{success:false, errorCode}
channel handleViciousHammerConsumableEvent → VICIOUS_HAMMER writer (mode 61 / 62)
```

Env topic names reuse the existing consumable env vars — no new topics, no
deployment yaml changes.
