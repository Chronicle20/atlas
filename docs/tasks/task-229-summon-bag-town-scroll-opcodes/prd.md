# Summoning Sack & Town Scroll Opcode Registration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-14
---

## 1. Overview

Summoning sacks (`Consume/0210.img.xml`, 283 items) and town/return scrolls are fully
implemented server-side. `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go`
carries `CharacterItemUseSummonBagHandleFunc` (line 45) and `CharacterItemUseTownScrollHandleFunc`
(line 27); both decode the shared 3-field `inventory/serverbound.ItemUse` struct
(`libs/atlas-packet/inventory/serverbound/item_use.go`) and call
`consumable.Processor.RequestItemConsume`. `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`
then branches **by item id** — not by handler name — to `ConsumeTownScroll` (line 490)
and `ConsumeSummoningSack` (line 658). Both handler names are registered in
`services/atlas-channel/atlas.com/channel/main.go` (lines 940, 952).

The feature is nevertheless dead on five of the eleven tenant socket-config templates,
because the *client-facing opcode binding* was never added to those templates. A handler
that is not bound to an opcode in a tenant's socket configuration is simply never
dispatched: the client sends the request, atlas-channel logs nothing, and the item
appears inert to the player. This is the same class of defect as
"new opcodes missing from live tenant config" — the code is right, the routing table is short.

This task is therefore a **registration and verification** task, not a feature build.
No Go code is expected to change. The work is: bind the two handlers (plus two
additional item-use handlers missing on gms_92) to their per-version opcodes in the
seed templates, back each binding with a packet-audit evidence record, and promote the
corresponding coverage-matrix cells.

## 2. Goals

Primary goals:

- Bind `CharacterItemUseSummonBagHandle` and `CharacterItemUseTownScrollHandle` to the
  correct serverbound opcode on every template where the client is known to send them.
- Close the gms_92 item-use hole: `CharacterItemUseHandle` and `CharacterItemUseScrollHandle`
  are *also* unbound there, which makes ordinary potion use and scroll use non-functional
  on that column and makes summon-bag testing impossible.
- Resolve, from the v48 IDB, whether a return/portal-scroll send site exists at that
  version — and either bind it or record a justified `n-a`.
- Promote the affected `USE_SUMMON_BAG` / `USE_RETURN_SCROLL` (and gms_92 `USE_ITEM` /
  `USE_UPGRADE_SCROLL`) coverage-matrix cells with pinned evidence.

Non-goals:

- Any change to `atlas-consumables` consumption logic, the `ItemUse` codec, or the
  atlas-channel handler functions. All three are already correct and shared across versions.
- `template_gms_12_1.json`. That template carries **no** item-use handlers at all
  (no `CharacterItemUseHandle`, no cash-item, no scroll, no pet) — it is a minimal
  bring-up stub, and completing it is a separate v12 version-pass, not a two-opcode fix.
- Adding `ShopScannerItemUseHandle` (missing on gms_87 and jms_185) or
  `CharacterItemUseLotteryHandle` (missing on gms_92). Out of scope; recorded in §9.
- PATCHing live tenant socket configurations. Seed templates and matrix only
  (see §10 for the operational note this leaves behind).

## 3. User Stories

- As a player on a v87/v92/v95/jms tenant, I want to double-click a summoning sack
  (e.g. a Wooden Box) so that the mobs it contains spawn on my field, exactly as they
  do on a v83 tenant.
- As a player on a v48/v87/v92/v95/jms tenant, I want to use a Return Scroll so that
  I am warped to the scroll's destination town and the scroll is consumed.
- As a player on a v92 tenant, I want to use a potion and use an upgrade scroll at all —
  today neither opcode is bound.
- As an operator, I want the seed templates to be the single accurate source for a
  tenant's socket routing, so that a freshly seeded v92 tenant is not silently missing
  core item-use dispatch.
- As a maintainer, I want each new binding backed by a pinned evidence record so the
  coverage matrix reflects verified reality rather than an assumed opcode.

## 4. Functional Requirements

### 4.1 Current state (verified 2026-08-14 against `services/atlas-configurations/seed-data/templates/`)

Item-use handler bindings present per template:

| template | `CharacterItemUseHandle` | `…SummonBagHandle` | `…TownScrollHandle` | `…ScrollHandle` |
|---|---|---|---|---|
| gms_12 | — | — | — | — |
| gms_48 | `0x41` | `0x3B` | **—** | `0x42` |
| gms_61 | `0x43` | `0x46` | `0x4E` | `0x4F` |
| gms_72 | `0x47` | `0x4A` | `0x54` | `0x55` |
| gms_79 | `0x46` | `0x49` | `0x53` | `0x54` |
| gms_83 | `0x48` | `0x4B` | `0x55` | `0x56` |
| gms_84 | `0x48` | `0x4B` | `0x55` | `0x56` |
| gms_87 | `0x4B` | **—** | **—** | `0x59` |
| gms_92 | **—** | **—** | **—** | **—** |
| gms_95 | `0x4E` | **—** | **—** | `0x5D` |
| jms_185 | `0x40` | **—** | **—** | `0x4E` |

This corrects the originating backlog note, which recorded gms_48 as `0/0` (SummonBag
is in fact bound there) and omitted jms_185 and the gms_92 base-handler gap entirely.

### 4.2 Opcodes already derived in the packet registry

`docs/packets/registry/*.yaml` already carries serverbound entries for the target
versions. These are the values the bindings must use; each still requires an evidence
record before its matrix cell may be promoted.

| version | `USE_ITEM` | `USE_SUMMON_BAG` | `USE_RETURN_SCROLL` |
|---|---|---|---|
| gms_v83 (reference) | `0x48` | `0x4B` | `0x55` |
| gms_v87 | `0x4B` | `0x4E` | `0x58` |
| gms_v92 | `0x4F` | `0x52` | `0x5C` |
| gms_v95 | `0x4E` | `0x51` | `0x5C` |
| jms_v185 | `0x40` | `0x43` | `0x4D` |

fnames: `CWvsContext::SendStatChangeItemUseRequest`, `CWvsContext::SendMobSummonItemUseRequest`,
`CWvsContext::SendPortalScrollUseRequest` respectively, on all five columns.

`docs/packets/registry/gms_v48.yaml` has **no** `USE_SUMMON_BAG` and **no**
`USE_RETURN_SCROLL` entry, despite the template already binding SummonBag at `0x3B`.
Nearby v48 opcodes are `PET_FOOD 0x3C`, `USE_MOUNT_FOOD 0x3D`, `USE_CASH_ITEM 0x3E`,
`USE_CATCH_ITEM 0x3F`, `USE_SKILL_BOOK 0x40`, `USE_ITEM 0x41`, `USE_UPGRADE_SCROLL 0x42` —
`0x3B` is consistent with the ordering but is currently an unbacked assertion.

### FR-1 — gms_87

FR-1.1 Bind `CharacterItemUseSummonBagHandle` at `0x4E`, validator `LoggedInValidator`,
`fname: CWvsContext::SendMobSummonItemUseRequest`, `services: ["channel"]`, inserted at
its sorted position in the `handlers` array.

FR-1.2 Bind `CharacterItemUseTownScrollHandle` at `0x58`, validator `LoggedInValidator`,
`fname: CWvsContext::SendPortalScrollUseRequest`, `services: ["channel"]`, at its sorted position.

### FR-2 — gms_92

FR-2.1 Bind `CharacterItemUseSummonBagHandle` at `0x52`.

FR-2.2 Bind `CharacterItemUseTownScrollHandle` at `0x5C`.

FR-2.3 Bind `CharacterItemUseHandle` at `0x4F`,
`fname: CWvsContext::SendStatChangeItemUseRequest`.

FR-2.4 Bind `CharacterItemUseScrollHandle` at the registry's v92 `USE_UPGRADE_SCROLL`
opcode, `fname: CWvsContext::SendUpgradeItemUseRequest`. The design phase must read the
exact value from `docs/packets/registry/gms_v92.yaml` rather than inferring it from a
neighbouring column.

FR-2.5 All four entries carry `validator: "LoggedInValidator"` and `services: ["channel"]`,
matching the gms_83 reference bindings.

### FR-3 — gms_95

FR-3.1 Bind `CharacterItemUseSummonBagHandle` at `0x51`.

FR-3.2 Bind `CharacterItemUseTownScrollHandle` at `0x5C`.

### FR-4 — jms_185

FR-4.1 Bind `CharacterItemUseSummonBagHandle` at `0x43`.

FR-4.2 Bind `CharacterItemUseTownScrollHandle` at `0x4D`.

### FR-5 — gms_48 return scroll

FR-5.1 Determine from the v48 IDB whether a portal/return-scroll send site exists.
Search for a `COutPacket` construction followed by `Encode4(updateTime) + Encode2(slot) + Encode4(itemId)`
with no `RunMapTransferItem` call, mirroring the v72/v79 pattern documented in
`docs/packets/registry/gms_v72.yaml` and `gms_v79.yaml` (both of which record that the
pre-existing `SendMapTransferItemUseRequest` symbol was a *mislabel* for the return-scroll
sender — expect the same trap at v48).

FR-5.2 If the send site exists: add a `USE_RETURN_SCROLL` entry to
`docs/packets/registry/gms_v48.yaml` with the resolved opcode and fname, and bind
`CharacterItemUseTownScrollHandle` in `template_gms_48_1.json` at that opcode.

FR-5.3 If it does not exist: record a justified `n-a` in the registry and the matrix,
with the search performed and the negative evidence cited, per the n-a consistency gate.
Do **not** leave the cell silently absent.

FR-5.4 Backfill a `USE_SUMMON_BAG` registry entry for gms_v48 documenting the `0x3B`
binding already present in the template, with a resolved fname (expected
`CWvsContext::SendMobSummonItemUseRequest`, or the `sub_XXXXXX` address if the symbol is
unnamed — name it in the IDB while reversing). If the IDB shows `0x3B` is wrong, correct
the template binding and say so explicitly.

### FR-6 — Evidence and matrix

FR-6.1 Each op × version cell touched by FR-1 … FR-5 is verified through the
single-cell procedure in `docs/packets/audits/VERIFYING_A_PACKET.md` (`/verify-packet`
command / `packet-verifier` agent): decompile the client read order, write the byte-fixture
test with a `packet-audit:verify` marker, pin the evidence record, regenerate the matrix.

FR-6.2 A cell that does not mechanically promote in `docs/packets/audits/status.json` /
`STATUS.md` is a failure, reported as such. No prose claim substitutes for a promoted cell.

FR-6.3 The shared `ItemUse` codec is already verified on the reference columns. Where a
target version's read order is byte-identical to gms_83, the evidence record must still
be pinned per version — a shared codec verified elsewhere is exactly the
"❌ can mean unverified shared codec" case, and re-using another column's fixture is not
verification.

### FR-7 — Guards

FR-7.1 `tools/template-opcode-order-guard.sh` clean: every new entry sits at its
strictly-ascending sorted position in the `handlers` array, never appended next to a
semantically related entry.

FR-7.2 `tools/template-duplicate-binding-guard.sh` clean: no `(handler name, numeric opCode)`
pair bound twice, including leading-zero-padded forms.

FR-7.3 Every new handler entry names a validator that exists in the template — a handler
whose validator is missing is silently dropped at load time.

## 5. API Surface

No REST endpoints are added or modified. The only externally visible surface change is
the tenant socket configuration document served by atlas-configurations, which gains
handler entries in the `handlers` array of the affected templates. Shape is unchanged:

```json
{ "opCode": "0x4E", "validator": "LoggedInValidator", "handler": "CharacterItemUseSummonBagHandle", "fname": "CWvsContext::SendMobSummonItemUseRequest", "services": ["channel"] }
```

## 6. Data Model

No schema changes. No migrations. The seed templates under
`services/atlas-configurations/seed-data/templates/` are JSON documents, not database
rows; they are consumed by the tenant configuration seeding path. Existing tenants
already provisioned from an earlier template revision will not pick up the new bindings
until they are reseeded or reconciled (§10, operational note).

## 7. Service Impact

| Service / area | Change |
|---|---|
| `services/atlas-configurations` | Handler entries added to `template_gms_48_1.json` (conditional on FR-5), `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`. |
| `docs/packets/registry/` | `gms_v48.yaml` gains `USE_SUMMON_BAG` and, if FR-5.2 holds, `USE_RETURN_SCROLL`. Other registries already carry the ops; notes may be extended with verification provenance. |
| `docs/packets/audits/` | Evidence records + regenerated `status.json` / `STATUS.md` for the promoted cells. |
| `libs/atlas-packet` | Fixture tests only. **No wire change.** The `ItemUse` struct is shared and already verified on gms_61/72/79/83/84 — modifying it would silently alter verified columns. |
| `services/atlas-channel` | None expected. Handler funcs and `main.go` registrations already exist. |
| `services/atlas-consumables` | None. Consumption already dispatches by item id. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** bindings are per-template, therefore per-tenant-version. A change to
  one template must not alter the wire behaviour of any other version's column.
- **No regression on verified columns:** gms_61/72/79/83/84 are already complete for both
  handlers and must be byte-identical after this task. Any diff touching them is a defect.
- **Observability:** after binding, a client-sent summon-bag or return-scroll request must
  produce the existing `l.Debugf("[%s] read [%s]", …)` line in atlas-channel with the
  matching operation name. Absence of that log line on a bound opcode is the primary
  symptom to check.
- **Evidence durability:** each pinned evidence record must survive `packet-audit` matrix
  regeneration without hash drift.

## 9. Open Questions

1. **gms_48 return scroll existence (FR-5).** Unresolved until the v48 IDB is read. The
   v72/v79 registry notes show the obvious symbol at that era is routinely mislabelled,
   so the answer must come from decompiled read order, not from a symbol name.
2. **gms_48 `0x3B` provenance (FR-5.4).** The template asserts it; the registry does not
   record it. Treated as unverified until the IDB confirms.
3. **gms_92 `USE_UPGRADE_SCROLL` opcode (FR-2.4).** Present in the registry but not read
   out during this interview; the design phase must quote it from the file.
4. **`ShopScannerItemUseHandle` missing on gms_87 and jms_185, `CharacterItemUseLotteryHandle`
   missing on gms_92.** Same class of gap, deliberately out of scope. Should become a
   follow-up task rather than being absorbed here.
5. **gms_12.** No item-use handlers at all; needs a version pass, not a patch.
6. **Live-tenant reconciliation.** Out of scope per §2, but a seeded-template-only fix
   does not reach already-provisioned tenants. Whether that is handled by a reseed or by
   a config PATCH is an operational decision left to the deployer.

## 10. Acceptance Criteria

- [ ] `template_gms_87_1.json` binds `CharacterItemUseSummonBagHandle` (`0x4E`) and
      `CharacterItemUseTownScrollHandle` (`0x58`), both with `LoggedInValidator`,
      `fname`, and `services: ["channel"]`.
- [ ] `template_gms_92_1.json` binds `CharacterItemUseHandle` (`0x4F`),
      `CharacterItemUseSummonBagHandle` (`0x52`), `CharacterItemUseTownScrollHandle` (`0x5C`),
      and `CharacterItemUseScrollHandle` (registry value), all four fully populated.
- [ ] `template_gms_95_1.json` binds `CharacterItemUseSummonBagHandle` (`0x51`) and
      `CharacterItemUseTownScrollHandle` (`0x5C`).
- [ ] `template_jms_185_1.json` binds `CharacterItemUseSummonBagHandle` (`0x43`) and
      `CharacterItemUseTownScrollHandle` (`0x4D`).
- [ ] gms_48 is resolved one way or the other: either a return-scroll binding + registry
      entry, or a documented `n-a` with the negative evidence recorded. `USE_SUMMON_BAG`
      is backfilled into `gms_v48.yaml` with a resolved fname.
- [ ] Every touched op × version cell is `✅` in `docs/packets/audits/STATUS.md`, backed by
      a pinned evidence record and a `packet-audit:verify`-marked byte fixture — verified
      by regenerating the matrix, not asserted.
- [ ] `tools/template-opcode-order-guard.sh` exits 0.
- [ ] `tools/template-duplicate-binding-guard.sh` exits 0.
- [ ] `git diff` shows **zero** changes to `template_gms_61_1.json`, `template_gms_72_1.json`,
      `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`,
      and zero changes under `libs/atlas-packet/inventory/serverbound/` other than tests.
- [ ] `tools/lint.sh --check` exits 0 from the repo root.
- [ ] Code review run (`superpowers:requesting-code-review`) before the PR is opened.
