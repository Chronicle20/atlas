# Vicious Hammer Use — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

The Vicious Hammer (item 5570000, `Cash/0557.img.xml`, classification 557) is a cash item that adds +1 upgrade slot to a target equip, with a lifetime maximum of 2 hammers per equip. It is a staple progression item: players buy it from the cash shop and apply it to endgame equips before scrolling.

Atlas currently has only fragments of this feature. The clientbound `ViciousHammer` writer exists and is registered (`services/atlas-channel/atlas.com/channel/socket/writer/vicious_hammer.go`, STATUS.md row `VICIOUS_HAMMER` ✅ for v83/v84/v87/v95), but nothing ever invokes it — no Kafka consumer, no handler. The serverbound side is entirely missing: `ITEM_UPGRADE_UPDATE | CUIItemUpgrade::Update` is ❌ in every version of the packet coverage matrix (STATUS.md:778). The `hammersApplied` field is plumbed through every data-model layer (atlas-inventory entity/builder/administrator/Kafka, atlas-channel asset model, atlas-cashshop reference data, and the atlas-packet equip encoder where it is a v84+ wire field) but nothing increments it. Using a hammer in-game today is a silent no-op that logs a warning.

This task implements the flow end-to-end: decode the serverbound packet(s), validate the target equip, consume the hammer, apply `slots += 1` / `hammersApplied += 1` atomically, and send the correct clientbound response — across all supported tenant versions.

## 2. Goals

Primary goals:
- A player can use a Vicious Hammer from the cash inventory on an eligible equip and gain +1 upgrade slot.
- The 2-hammer lifetime cap per equip is enforced server-side (exact cap to be re-verified in IDA/WZ during design; 2 is the expected value).
- Item consumption and equip mutation are atomic — no consumed-hammer-without-slot or slot-without-consumption outcomes.
- The flow works on all supported GMS tenant versions (v83, v84, v87, v92, v95).
- New packet codecs are byte-fixture verified per `docs/packets/audits/VERIFYING_A_PACKET.md`, promoting the relevant matrix cells.

Non-goals:
- Cash shop purchase of the hammer (existing cash shop flow; unchanged).
- Any UI (atlas-ui) work.
- Other 55x cash item types (the fall-through warn in `character_cash_item_use.go` stays for them).
- JMS support, unless design-phase IDA work shows the jms client uses the same dialog flow (the clientbound `VICIOUS_HAMMER` op is absent from the jms registry; serverbound `ITEM_UPGRADE_UPDATE` exists at 0x114). Decision deferred to design.

## 3. User Stories

- As a player, I want to use a Vicious Hammer on an equip so that it gains an extra upgrade slot for scrolling.
- As a player, I want the hammer refused (and not consumed) when the target equip already has the maximum hammers applied, so I don't waste NX.
- As a player, I want my equip window to immediately reflect the new slot count and hammer count after use.
- As an operator, I want hammer application recorded (`hammersApplied`) so equip provenance is auditable and correctly encoded in v84+ equip packets.

## 4. Functional Requirements

### 4.1 Client protocol (design-phase IDA verification required)

The exact wire flow must be IDA-verified during design for each supported version — this is an explicit design task, not an implementation detail. Known facts and expectations:

- **Serverbound:** `ITEM_UPGRADE_UPDATE` (fname `CUIItemUpgrade::Update`) exists in all five IDBs (v83 0x104, v84 0x104, v87 0x112, v95 0x128, jms 0x114) and is unimplemented. Cosmic reference behavior (unverified against our clients) splits handling between `UseCashItemHandler` itemType 557 and a dedicated `UseHammerHandler` dialog flow. Design must determine, per version: which packet(s) the client actually sends when a hammer is used, and their full body layouts.
- **Clientbound:** `VICIOUS_HAMMER` (fname `CField::OnItemUpgrade`, v83 0x162 / v84 0x169 / v87 0x177 / v95 0x1A9) is registered with an **empty body**. The fname is a vtable forwarder into the item-upgrade dialog; design must decompile the dialog's receive path to determine the real body — including whether it is mode-prefixed (open / result / error sub-ops). If it is a mode-prefix dispatcher, follow `docs/packets/DISPATCHER_FAMILY.md` (discrete struct per mode, config-resolved mode bytes, per-mode fixtures — no mode-byte-only enumeration).
- The existing cash-item-use handler already classifies category 557 as CashSlotItemType 66 (67 for GMS ≥95) at `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:469` but has no handling arm. If design confirms the use flows through this packet, add the arm there; note the open `// TODO for v83 there is a trailing updateTime` at line 108 of that file.
- v92 has no IDB. Its opcodes/bodies must be derived from the template lineage (v87→v95 interpolation is NOT acceptable as "verified" — mark the v92 cells accordingly and flag any guesswork explicitly).

### 4.2 Validation (atlas-consumables)

On a hammer-use request, the server must validate before consuming anything:

1. The character owns a Vicious Hammer (5570000) in the cash compartment at the claimed slot.
2. The target equip exists in the character's possession (design determines whether the client targets equipped items, inventory items, or both — verify in IDA which the dialog offers).
3. `hammersApplied < 2` on the target equip (cap value re-verified during design).
4. The target equip is hammer-eligible. Exclusion rules (e.g., equips with zero base upgrade slots, cash equips) must be verified against IDA/WZ during design — do not copy Cosmic's list on faith.

A request failing validation must not consume the hammer and must not mutate the equip.

### 4.3 Application (atomic)

On successful validation:

1. Consume exactly one 5570000 from the cash compartment.
2. Apply to the target equip: `slots += 1` and `hammersApplied += 1`, via the existing `equipable.Change` machinery in atlas-consumables (`equipable.AddSlots(1)` exists at `equipable/processor.go:124`; a `hammersApplied` change function must be added — the producer already forwards the field at `equipable/producer.go:45`).
3. Steps 1–2 must be atomic in the same sense as the scroll flow (same saga/compensation or buffered-emit pattern the scroll path uses — follow `ConsumeScroll` at `consumable/processor.go:606` as the structural precedent).

### 4.4 Client feedback

1. On success: send the IDA-verified success response so the dialog closes/updates, and ensure the equip's new `slots`/`hammersApplied` reach the client through the existing inventory/asset update packets (the equip encoder already writes `hammersApplied` for v84+).
2. On failure: send the IDA-verified failure/rejection response (expected to exist per Q5 of the interview; confirm shape in IDA). If the client turns out to be fire-and-forget for some failure class, server-side validation still applies and the request is dropped with a warn log — but any client-side stuck-dialog state must be resolved (e.g., stat/enable-actions equivalent), verified in design.

### 4.5 Configuration & rollout

1. Add the serverbound handler entries (with `LoggedInValidator` — validator-less handler entries are silently dropped) and any new writer entries to the tenant seed templates for every supported version.
2. Document and perform the live-tenant config patch + channel restart for existing tenants (seed templates only apply at tenant creation).
3. If the clientbound packet is mode-prefixed, populate the per-version `operations` mode tables in every template it applies to (missing tables resolve to 99 and crash the client).

### 4.6 Packet verification

1. Every new codec gets byte-fixture tests with `packet-audit:verify` markers and evidence records per `VERIFYING_A_PACKET.md`.
2. Regenerate the matrix; the `ITEM_UPGRADE_UPDATE` serverbound cells and (if body changes) `VICIOUS_HAMMER` clientbound cells for versions with IDBs must promote to ✅. v92 cells: whatever state honestly reflects the evidence.

## 5. API Surface

No new REST endpoints expected. The flow is socket + Kafka:

- **Serverbound socket:** new handler(s) in atlas-channel for the hammer-use packet(s) (exact set per design).
- **Kafka:** a hammer-use command from atlas-channel to atlas-consumables (either a new command on the existing consumable command topic or reuse of `RequestItemConsume`-style flow — design decision), followed by the existing compartment/equipable change events to atlas-inventory.
- **Clientbound socket:** invocation of the `ViciousHammer` writer (body per IDA) plus existing inventory-update writers.

Error cases are conveyed via the clientbound failure response (4.4.2); no new error envelope.

## 6. Data Model

No schema changes. `hammersApplied` (uint32) and `slots` (uint16) already exist on the atlas-inventory asset entity (`asset/entity.go:55`), are persisted by the administrator (`asset/administrator.go:71`), and travel on the existing Kafka bodies. This task adds the first writer of `hammersApplied`.

## 7. Service Impact

- **libs/atlas-packet** — new serverbound codec(s) for the hammer flow (`ITEM_UPGRADE_UPDATE` at minimum, per design); real body for `field/clientbound/ViciousHammer` (currently empty; may become mode-structured). Byte fixtures for both.
- **services/atlas-channel** — serverbound handler wiring (possibly an arm in `character_cash_item_use.go` plus a dedicated dialog handler); Kafka consumer that invokes the `ViciousHammer` writer on result events; session/field routing.
- **services/atlas-consumables** — hammer validation + consumption + equip-change logic, following the scroll pattern; new `equipable.Change` for `hammersApplied`.
- **services/atlas-inventory** — expected no code change (mutation arrives via existing change/accept commands); verify the chosen command path actually persists both fields.
- **Tenant configuration** — seed template updates for all supported versions + live-tenant config patch at rollout.

Docker bake required for every touched service per CLAUDE.md verification rules.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all handlers/consumers resolve tenant from context; opcodes and any mode bytes are config-resolved per tenant version — never hard-coded (dispatcher-lint rules apply if mode-structured).
- **Atomicity:** consume+mutate must not partially apply on failure (compensation or single-transaction semantics, matching the scroll flow's guarantees).
- **Idempotency/abuse:** duplicate or replayed use packets for the same hammer slot must not double-apply (validation re-checks state at execution time).
- **Observability:** warn-level logs for rejected attempts (mirroring existing handler conventions); no new metrics required.
- **Testing:** unit tests for validation/mutation logic via the project Builder pattern (no `*_testhelpers.go`); byte-fixture packet tests; `go test -race`, `go vet`, `go build`, and `docker buildx bake` clean for all touched modules.

## 9. Open Questions

Deferred to the design phase (all require IDA/WZ verification, not judgment calls):

1. Per version: which serverbound packet(s) carry hammer use — cash-item-use (type 66/67), `ITEM_UPGRADE_UPDATE`, or both in sequence — and their exact bodies.
2. The real clientbound body of `CField::OnItemUpgrade`'s dialog path, including whether it is mode-prefixed and what the failure arm looks like.
3. The exact hammer cap (expected 2) and target-eligibility rules as the client enforces them.
4. Whether the dialog targets equipped items, inventory equips, or both.
5. JMS inclusion (clientbound op absent from jms registry — same-mechanism check).
6. v92 opcode/body derivation strategy given no IDB exists.

## 10. Acceptance Criteria

- [ ] On each supported GMS tenant version, using a Vicious Hammer on an eligible equip in-game: consumes exactly one hammer, the equip gains one upgrade slot, `hammersApplied` increments, and the equip window reflects both without relog.
- [ ] A third hammer on the same equip is rejected: no consumption, no mutation, client receives the verified failure response (or documented fire-and-forget behavior) and is not soft-locked.
- [ ] Hammer-ineligible targets (per verified rules) are rejected the same way.
- [ ] `hammersApplied` persists across relog and encodes correctly in v84+ equip packets.
- [ ] `ITEM_UPGRADE_UPDATE` serverbound matrix cells for IDB-backed versions are ✅ with pinned evidence; `VICIOUS_HAMMER` clientbound cells remain/promote to ✅ with the real body; matrix regenerated with no new conflicts.
- [ ] Seed templates updated for all supported versions (handlers include validators); live-tenant config patch documented and applied; if mode-prefixed, `operations` tables populated per version and `packet-audit dispatcher-lint` exits 0.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` clean for every touched service; `tools/redis-key-guard.sh` clean.
- [ ] Code review (plan-adherence + backend-guidelines) run before PR.
