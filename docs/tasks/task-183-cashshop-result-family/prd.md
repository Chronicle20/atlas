# CashShopOperation Result Family — Full `OnCashItemResult` Enumeration & Codecs — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-27
---

## 1. Overview

`CCashShop::OnCashItemResult` is the client's single clientbound dispatcher for the cash-shop
result family. Its leading `Decode1` byte switches to one of ~60 per-mode handler functions
(`OnCashItemResLoadLockerDone`, `OnCashItemResBuyDone`, `OnCashItemResGiftDone`, …). Atlas
currently models only **9** of those arms as discrete writer structs in
`libs/atlas-packet/cash/clientbound/`, and the dispatcher family source-of-truth
(`docs/packets/dispatchers/cash_shop_operation.yaml`) enumerates the same 9 arms across only
5 versions (gms_v83/v84/v87/v95/jms_v185). The remaining ~50 result arms — gift, coupons,
trunk/character-slot counts, buy-character, equip-slot-extension, destroy/expire, rebate,
couple, packages, buy-normal, friendship, free-item, purchase-record, name-change,
world-transfer, gachapon (open/copy), and maple-point — have no clientbound codec and no
per-version mode enumeration, even though many already have a **serverbound request codec**
in `libs/atlas-packet/cash/serverbound/`.

This task closes that gap in three coordinated layers, driven by a full reverse-engineering
pass of the dispatcher in **all nine client IDBs** with **gms_v95.1 as the reference**:

1. **Reverse-engineering** — fully RE `CCashShop::OnCashItemResult` and every branch handler it
   dispatches to, in each of the 9 IDBs. Every branch function is verified and **named in the
   IDB** (per the project's "name symbols while reversing" discipline). The v95 IDB is the
   canonical read-order/field-shape reference; other versions are diffed against it.
2. **Codecs + enumeration** — a discrete per-mode writer struct (Encode + Decode) for every
   result arm that exists in each version, with the per-version mode byte enumerated in
   `cash_shop_operation.yaml` and propagated into every version template's
   `CashShopOperation` writer `operations` map.
3. **Verification** — each arm × version cell promoted to `✅` in the packet coverage matrix
   via the `packet-verifier` fixture procedure (client-read-order evidence pinned).

This is explicitly a **codec + enumeration + verification** task. It does **not** add new
server-side feature behavior (no new gifting/coupon/gachapon domain flows). Producers are
wired only where a domain flow already emits the corresponding result.

## 2. Goals

Primary goals:

- Produce a complete, IDA-verified enumeration of every `CCashShop::OnCashItemResult` branch
  arm, per version, for all 9 matrix versions (gms_v48, gms_v61, gms_v72, gms_v79, gms_v83,
  gms_v84, gms_v87, gms_v95, jms_v185).
- Fully reverse-engineer and **name in each IDB** the root `OnCashItemResult` dispatcher and
  **every** branch handler function it calls (target: zero `sub_*`/unnamed arms remaining in
  the dispatcher's call tree for the covered versions). v95 is the reference.
- Add a discrete clientbound writer struct (Encode + Decode + `Operation()` + `String()`) for
  every result arm present in a given version, matching the existing per-mode-struct pattern
  in `shop_operation_result.go` / `shop_operation_body.go`.
- Extend `docs/packets/dispatchers/cash_shop_operation.yaml` to the full arm set with
  per-version mode bytes for all 9 versions, keeping it the single source of truth for the
  template `operations` map.
- Propagate every new operation key into the `CashShopOperation` writer `operations` map of
  **every** version template under
  `services/atlas-configurations/seed-data/templates/`.
- Promote every applicable arm × version cell in the packet coverage matrix
  (`docs/packets/audits/STATUS.md` / `status.json`) to `✅` with a pinned evidence record.

Non-goals:

- No new cash-shop **feature logic** (actual gifting, coupon redemption, gachapon rolls,
  world transfer, maple-point conversion). Server behavior that produces these results is out
  of scope; writers are added as codecs, and producers are wired only for flows that already
  exist.
- No serverbound request-codec changes beyond what is needed to keep an existing request/result
  pair consistent (the request side is largely already modeled).
- No changes to already-verified arms/versions except where the RE pass discovers a
  correctness defect (which must be documented and fixed on-branch, not deferred).
- No UI (atlas-ui) changes.

## 3. User Stories

- As a **packet engineer**, I want every `OnCashItemResult` arm enumerated with IDA-verified
  per-version mode bytes so that future cash-shop features can emit any result without
  re-deriving the wire format.
- As a **reverse engineer**, I want every branch handler named in each IDB so that subsequent
  investigations of the cash-shop family start from named, verified functions rather than
  `sub_*` stubs.
- As a **tenant operator**, I want every version template's `CashShopOperation` `operations`
  map to carry the full arm set so that a server emitting any cash result resolves the correct
  per-version mode byte instead of silently dropping the packet.
- As a **maintainer**, I want each arm × version cell verified in the coverage matrix so that
  "implemented" is backed by client-read-order evidence, not a prose claim.

## 4. Functional Requirements

### 4.1 Reverse-engineering (all 9 IDBs, v95 reference)

- **FR-1.1** RE `CCashShop::OnCashItemResult` (the root dispatcher) in each of the 9 IDBs.
  Record the full switch: every leading-byte case value and the handler it calls. For legacy
  versions where the dispatcher lives under a different opcode/name (registry notes: v61
  `CASHSHOP_OPERATION`=196, v72 op 291, v79 op 303, legacy send-sites `COutPacket(160)`),
  locate and confirm the equivalent dispatcher.
- **FR-1.2** RE every branch handler function reachable from the dispatcher (each
  `OnCashItemRes*` arm) in every version. Derive the exact client read order and field types
  for each.
- **FR-1.3** **Name every function** touched — the root dispatcher and all branch handlers —
  in each IDB, using the canonical v95 names as the naming baseline (e.g.
  `OnCashItemResGiftDone`). Unnamed/`sub_*` functions that are confirmed arms must be renamed;
  a confirmed arm may not be left unnamed. Save the IDB after naming.
- **FR-1.4** Where a version's arm diverges from v95 in read order or field width, the
  divergence is documented (per-version RE notes under the task folder) and version-gated in
  the codec (see FR-2.4).
- **FR-1.5** Where an arm is **absent** in a version (feature did not exist in that client —
  e.g. gachapon/world-transfer/maple-point in early legacy clients), record it as `n-a` for
  that version in the yaml and matrix; do not fabricate a mode byte.

### 4.2 Codecs (libs/atlas-packet/cash/clientbound)

- **FR-2.1** Each result arm present in ≥1 version gets a discrete per-mode struct following
  the existing pattern: private fields, `New…` constructor, accessor methods, `Operation()`
  returning `CashShopOperationWriter`, `String()`, and both `Encode` and `Decode`.
- **FR-2.2** The struct **fixes its operation key** and resolves its mode byte from config via
  `atlas_packet.ResolveCode` / `WithResolvedCode` against the `"operations"` table — never
  accepts a caller-supplied mode (matches the DOM-25 config-resolved-wire-value rule and the
  existing body-func pattern).
- **FR-2.3** Failure arms resolve their reason byte from the writer's `"errors"` table (as
  `LoadInventoryFailure` / `InventoryCapacityFailed` already do). Whether a given `…Failed` arm
  is a distinct struct or reuses a shared `mode + reason` failure struct is decided per arm from
  the RE (some failed arms carry extra fields; those need their own struct).
- **FR-2.4** Version-divergent fields are gated with the `MajorAtLeast`/`MajorVersion` idiom
  (never a raw `> N`), consistent with the rest of `libs/atlas-packet`. No wire change is made
  to an already-verified arm/version.
- **FR-2.5** New operation-key constants are added to `shop_operation_body.go` alongside the
  existing 9, and a body-builder func is added per arm mirroring the existing
  `CashShop…Body` helpers.
- **FR-2.6** The existing `CashShopCashGiftsBody` `TODO` (hardcoded `0x4D`) is resolved: the
  gift arm's mode byte becomes config-resolved from the `operations` map like every other arm.

### 4.3 Enumeration & templates

- **FR-3.1** `docs/packets/dispatchers/cash_shop_operation.yaml` is extended to the full arm
  set. Each operation entry lists `key`, `handler` (the IDB-verified branch function name), and
  `modes` for all 9 versions (or `n-a`).
- **FR-3.2** `packet-audit operations` regenerates each version's `CashShopOperation` writer
  `operations` map from the yaml; every version template under
  `services/atlas-configurations/seed-data/templates/` is updated. Per the project rule, a new
  operation key must appear in **every** version template that supports it.
- **FR-3.3** `packet-audit operations --check` passes (yaml ↔ template sync), and the
  template-opcode-order guard passes for any touched template.

### 4.4 Verification

- **FR-4.1** Every applicable arm × version cell is promoted to `✅` in the coverage matrix via
  the `packet-verifier` procedure: client-read-order fixture with a `packet-audit:verify`
  marker, pinned evidence record, matrix regenerated.
- **FR-4.2** `n-a` cells pass the n-a consistency gate (arm absent in that version's IDB).
- **FR-4.3** Codec Encode/Decode round-trip unit tests exist for every new struct.

## 5. API Surface

No REST/JSON:API surface changes. The "API" here is the wire protocol:

- **New clientbound writer arms** under the existing `CashShopOperation` writer (clientbound
  opcode is the per-version `CASHSHOP_OPERATION`/`CASHSHOP_MESSAGE` opcode; v95 `0x180`). Each
  arm is `mode byte` + arm-specific body, dispatched by the client's leading-byte switch.
- **Config contract:** each arm's mode byte is resolved from the tenant socket-config
  `CashShopOperation` writer `options.operations["<KEY>"]`; failure reason bytes from
  `options.errors["<REASON>"]`.

A per-arm wire-shape table (mode, field list, version divergences, n-a versions) is produced as
a supporting artifact (`arm-catalog.md`) during design/planning, seeded from the RE pass.

## 6. Data Model

No database entities. The durable "data model" is:

- `docs/packets/dispatchers/cash_shop_operation.yaml` — the per-version mode table (extended).
- The `CashShopOperation` writer `operations` map in each version template JSON.
- The packet coverage matrix rows (`status.json`) for each arm × version cell.
- Per-IDB named functions (persisted in the IDBs via `idb_save`).

## 7. Service Impact

- **libs/atlas-packet** — new clientbound structs + body builders in `cash/clientbound/`; new
  operation-key constants; possible shared failure struct. Round-trip tests. (Primary code
  change.)
- **services/atlas-configurations** — `seed-data/templates/template_<ver>_1.json` updated
  `CashShopOperation` `operations` maps (regenerated, not hand-edited).
- **services/atlas-channel** — no new feature flows. Only touched if an already-existing cash
  flow currently emits via an inline/incorrect mode and should route through a new body builder
  (documented case-by-case; default is no change).
- **docs/packets** — `cash_shop_operation.yaml`, coverage matrix (`STATUS.md`/`status.json`),
  evidence records, per-cell fixtures.
- **IDBs (9)** — renamed/verified dispatcher + branch functions, saved.

No `go.mod` is added; `libs/atlas-packet` is already registered. Standard build/verify per
CLAUDE.md applies to `libs/atlas-packet` and `services/atlas-configurations` seed changes.

## 8. Non-Functional Requirements

- **Grounding:** every mode byte, field, and arm presence comes from the IDB (v95 reference) or
  the checked-in export — never from memory or general MapleStory knowledge. Unverified fnames
  escalate rather than being guessed (project fname-escalation rule).
- **Config-resolved wire values (DOM-25):** no hardcoded mode/opcode bytes in codecs; all
  version-specific values resolve from config. Resolves the existing `0x4D` gift TODO.
- **No silent drops:** every supported arm must exist in every supporting version template, or
  the server would drop the packet at emit time (known bug class: "new opcodes missing from
  live tenant config → dropped").
- **No stubs:** no writer arm is added with a `TODO`/hardcoded placeholder mode. An arm is
  either fully modeled + enumerated + verified, or recorded `n-a`.
- **Version gating idiom:** `MajorAtLeast`, not raw integer comparison.
- **Determinism:** template `operations`/`writers` arrays keep strictly ascending order
  (template-opcode-order guard).

## 9. Open Questions

- **Q1 — Legacy arm presence:** Which arms actually exist in gms_v48/v61/v72/v79? Registry
  notes suggest early clients collapse the family to a cash-purchase subset (`COutPacket(160)`
  + Enc1(sub-op) for Buy/BuyCouple/BuyFriendship/BuyPackage/Gift/SetWish). The RE pass (FR-1)
  resolves this; arms absent in a version are `n-a`. Expected: gachapon, world-transfer,
  maple-point, purchase-record, name-change, free-item are likely `n-a` in the oldest clients.
- **Q2 — Failure-arm modeling:** How many `…Failed` arms carry fields beyond `mode + reason`?
  The RE determines whether a shared failure struct suffices or each needs its own. (User has
  chosen "all ~50 arms"; failure-struct sharing is an implementation detail, not a scope cut.)
- **Q3 — Producer wiring:** Are there existing atlas-channel/atlas-cashshop flows that today
  emit any of these results via an inline encoding that should be re-routed through the new
  body builders? Enumerated during planning; default assumption is none.
- **Q4 — jms_v185 divergence:** jms arm set/mode bytes are enumerated from the jms IDB
  independently (jms audit-dir naming differs — see project notes).

## 10. Acceptance Criteria

- [ ] `CCashShop::OnCashItemResult` and **every** branch handler are named/verified in all 9
      IDBs (v95 as naming baseline); IDBs saved. Zero confirmed-but-unnamed arms remain.
- [ ] A per-arm × per-version catalog (`arm-catalog.md`) documents mode byte, field shape,
      version divergences, and `n-a` versions, each traceable to an IDB function.
- [ ] Every result arm present in ≥1 version has a discrete clientbound struct with Encode +
      Decode + `Operation()` + config-resolved mode (no hardcoded bytes; `0x4D` gift TODO gone).
- [ ] Round-trip Encode/Decode unit test passes for every new struct.
- [ ] `cash_shop_operation.yaml` enumerates the full arm set with per-version modes / `n-a`
      for all 9 versions.
- [ ] Every version template's `CashShopOperation` `operations` map is regenerated from the
      yaml; `packet-audit operations --check` passes; template-opcode-order guard passes.
- [ ] Every applicable arm × version cell is `✅` in the coverage matrix with pinned evidence;
      `n-a` cells pass the n-a consistency gate.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `libs/atlas-packet` and
      any touched module; `docker buildx bake` for any service whose `go.mod` was touched
      (expected: none beyond seed-data, which needs no bake — confirm).
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` clean.
- [ ] Code review (plan-adherence + backend-guidelines + packet-completeness-critic) run before
      PR.
