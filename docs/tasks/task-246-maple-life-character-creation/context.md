# Maple Life (`Cash/0543`) — Implementation Context

Companion to `plan.md`. Everything an implementer or reviewer needs that is not a plan step.

---

## 1. Plan-time decisions the user made

Design §10 carried two questions forward. Both were answered before the plan was written; the plan encodes the answers, and neither is open.

| Question | Answer | Where it lands |
|---|---|---|
| **§5.3** — does atlas-channel re-validate the eleven look/equipment fields before calling the factory? | **No. `atlas-character-factory` is the sole validator.** The channel submits and maps the factory's `400` onto a client-rendered `MAPLELIFE_ERROR`. | Task 13, Step 3 |
| **§4.3** — add `TransactionId` to the seed status envelope? | **Yes.** Additive on `atlas-character-factory`'s message struct, both producers and the saga bridge. `atlas-login` is unmodified — it deserialises into its own copy of the struct and ignores unknown fields. | Task 10 |

The look-validation answer reads FR-4.3's second clause as operative over its first. The factory validates face, hair, hairColor, skinColor, top, bottom, shoes, weapon, gender, jobIndex and subJobIndex against the tenant's own creation template and returns `400` for each (`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:100-155`, `resource.go:73-101`). PRD §8's security requirement is met — the look **is** validated server-side, in the service that owns character creation. It is simply not validated twice.

---

## 2. The hard sequencing constraint

**Tasks 1 and 2 gate everything.** No task from Task 3 onward may begin until `derivation.md` exists and its §1–§6 are filled.

This is not a preference. Every opcode, field order, width, arm and error code in Tasks 3–7 is a `<derivation.md §N …>` marker in the plan, because the value does not exist yet and CLAUDE.md forbids inventing one. An implementer who reaches Task 4 with an empty §4 must stop and report BLOCKED, not guess from the sibling `CashShopCheckNameChange` codec.

Three branch points inside the derivation change the shape of downstream work:

| Derivation finding | Downstream effect |
|---|---|
| §1 — is the client's `itemId/1000-5431` comparison signed or unsigned? | Signed ⇒ 57/58 unreachable with shipped data, **one** sub-body shape in Task 6. Unsigned ⇒ 5430xxx reaches 57/58 and Task 6 may need **two**. |
| §3 — does gms_v95 send `USE_MAPLELIFE` (303), the `USE_CASH_ITEM` 543 sub-body, or both? | Decides whether Task 6 writes `maplelife/serverbound/use.go`, whether Task 7 adds the v95 handler entry, and whether Task 11 writes `MapleLifeUseHandleFunc`. The 543 arm exists on v95 regardless — a client that sends the sub-body anyway must not hit the silent fallthrough FR-5.4 forbids. |
| §6 — does any in-scope version reuse `CHECK_CHAR_NAME` (21) for the probe? | Outcome (A): standalone `MapleLifeCheckNameHandleFunc` per version. Outcome (B): a branch **inside** `CashShopCheckNameChangeHandleFunc` selected by the live pending record. Task 12 covers both; only the caller differs. |

The design was written to absorb any of these answers without restructuring. That is why `beginMapleLife` exists as a separate indirection both entry points converge on (design §3), and why the pending store is account-keyed rather than hung off the session.

---

## 3. Facts that corrected the PRD

The design's §0 settled four PRD premises against the repository. They are load-bearing for the plan and are restated here so a reviewer does not re-derive them.

**C1 — the probe is not `CHECK_CHAR_NAME`.** PRD FR-3.1 asserts opcode 0x100 on gms_v83. The registry says `CHECK_CHAR_NAME` is 21 (0x15) on every GMS build from v61 up, and its fname list carries `CCashShop::` and `CLogin::` senders — not `CUICharacterSaleDlg::`. The row that *does* carry the Maple Life sender is named `JMS_SLASH_COMMAND` (`docs/packets/registry/gms_v87.yaml:3651-3655`, `gms_v95.yaml:4100-4104`, `jms_v185.yaml:3640-3644`) and is `-1` on gms_v83/84/92. Task 3 fixes the registry; Task 2 §6 supplies the evidence.

**C2 — `POST characters/seed` is asynchronous and checks neither names nor slots.** It returns `202 Accepted` with a `transactionId` and emits a `CharacterCreation` saga. Its `Create` calls `validName` (format only) — the duplicate check lives solely on the `CreateFromPreset` path. Nothing anywhere reads `Account.CharacterSlots`. So FR-4.4 and FR-4.5 have no existing server-side owner and the channel must own both (Task 13, gates 3 and 4). This is not gold-plating; it is the only place the checks can live without a factory redesign.

**C3 — the completion signal exists but is keyed by account id only.** `EVENT_TOPIC_SEED_STATUS` carries `CREATED` (with `characterId`) and `FAILED` (with `reason`) as `StatusEvent{AccountId, Type, Body}`. Task 10 adds the transaction id the bridge already has in hand and drops.

**C4 — no configuration or deploy work.** `requests.RootUrlFor(ctx, "CHARACTER_FACTORY")` resolves from environment variables, not tenant service configuration (`libs/atlas-rest/requests/url.go:34-64`): a `CHARACTER_FACTORY_SERVICE_URL` override if present, else `BASE_SERVICE_URL`, namespace-rewritten for sparse environments. `BASE_SERVICE_URL` is set for every service from `deploy/k8s/base/env-configmap.yaml:19`, and the ingress already routes `^/api/characters/seed(/.*)?$` (`deploy/k8s/base/routes.conf.template.generated:386-389`). PRD §5's "Configuration" paragraph and PRD §7's `deploy/k8s` row are both no-ops. **Do not add a service-config seed row or an ingress entry** — Task 9 Step 3 states this as a deliberate non-action so it is not "discovered" as missing work later.

---

## 4. Why the dispatch arm routes on classification

`GetCashSlotItemType`'s 543 branch (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1457-1469`) returns 57, 58, 65 or 66. Every one of those values is claimed by something else:

| Value | Other claimant | Site |
|---|---|---|
| 57 / 58 | `ClassificationPetMultiConsumable` | `character_cash_item_use.go:1485-1490` |
| 65 | `CashSlotItemTypeSealTimedV95` | `character_cash_item_use.go:970` |
| 66 | `CashSlotItemTypeViciousHammer` (GMS < 95) | `character_cash_item_use.go:991` |

On v83 Maple Life lands on 65 — which `SealTimedV95` also claims. On v95 it lands on 66 — which `ViciousHammer` also claims. A type-based arm would necessarily break one of them. Routing on `item.GetClassification(itemId) == item.ClassificationCharacterCreation` makes the collision structurally impossible rather than merely avoided, and it makes design OQ-2 irrelevant to *routing*: signedness decides which sub-body the client encoded, not which Atlas arm runs.

A `mapleLifeCashSlotItemType(t)` helper mirroring task-227's `nameChangeCashSlotItemType` was considered and rejected. Task-227's coupons have a unique type value per version, so `it` is a sound discriminator there; 543's is not.

---

## 5. FR-5.2's restore path is vacuous — deliberately

FR-5.2 requires a compensating restore *if the client's protocol forces consume-first ordering*. It does not: the seed is asynchronous (C2), so there is no synchronous success to consume against, and the plan orders the flow create-then-consume throughout.

| Event | Item | Client |
|---|---|---|
| pre-check rejection | untouched | `MAPLELIFE_ERROR{specific arm}` |
| factory 400 / 5xx / unreachable | untouched | `MAPLELIFE_ERROR{mapped arm}` |
| seed `FAILED` | untouched | `MAPLELIFE_ERROR{generic failure}` |
| seed `CREATED` | destroyed | `MAPLELIFE_ERROR{success arm}` |

So **no restore path is written**, and the acceptance criterion "a test covers the restore path if consume-first ordering is forced" is satisfied vacuously. Task 13's `TestMapleLifeCreateNeverConsumesTheItem` is the executable form of that claim. Stated here so the box is not left ambiguous at review.

Two residual edges are decided rather than left open:

- **Character created, destroy step fails.** The character is real and stays; the item survives. Logged at error with account, character, item and transaction ids. Rolling back a created character to reclaim a cash item is destructive and disproportionate — the failure is operator-visible instead.
- **Session gone before `CREATED` lands.** Still destroy the item: the entitlement was spent and the character exists, so leaving it would let one item produce two characters. No client write is attempted; logged at info, mirroring `services/atlas-login/atlas.com/login/kafka/consumer/seed/consumer.go:82-88`.

---

## 6. Task sizing

Fifteen tasks. `tools/plan-lint.sh` exits 0 on errors; five F4/F5 advisories stand, each deliberate and recorded here.

**F4 oversized — Task 5 lists 8 files.** Two are `.go`, five are one-line-per-version evidence YAMLs, and one is a read-only pointer to `derivation.md`. The five evidence records are the same file written five times with a different version and address; splitting them from the codec they attest to would produce a task that cannot be verified on its own, because `packet-audit matrix` only promotes a cell when the marker, the fixture and the record all agree. Task 4 has the same shape and slips under the threshold only by listing one fewer file.

**F4 multi-service — Tasks 9, 10 and 14.**

- *Task 9* spans two services only in the linter's reading: every file it **writes** is under `services/atlas-channel`. The atlas-login entry is a read-only pattern reference, listed precisely so the implementer copies the `RestModel` field set from the real file instead of from the PRD.
- *Task 10* is entirely inside `atlas-character-factory`; the atlas-login entry is a read-only assertion target (Step 5 proves it is untouched and still compiles).
- *Task 14* genuinely spans `atlas-channel`, `libs/atlas-saga` and `atlas-saga-orchestrator`. See below.

**F5 unknown symbol — `NewItemUseMapleLife`.** Task 6 creates it; the plan declares it in Task 6's Interfaces block. The linter cannot see the declaration because the constructor is named in prose rather than in a fenced `go` block. Not a defect.

Every task touches ≤6 files and one service, except two, both deliberate:

- **Task 14** spans `atlas-channel`, `libs/atlas-saga` and `atlas-saga-orchestrator`. The orchestrator and shared-lib edits are four lines total — one type constant, two aliases, two list entries — and they cannot be split from the consumer that uses the type without landing an unreferenced constant. Splitting would produce a task whose only deliverable is a name.
- **Task 6** covers three serverbound codecs. Two of the three are conditional on derivation findings and may not exist at all; sizing them as separate tasks would risk two empty dispatches. They share one test file and one evidence pass.

Tasks 1 and 2 are IDA-heavy and read-only against the repo. Their tool-call cost is in `func_query`/`decompile` calls, not edits — if either hits the 120-call budget, the natural split is per-version rather than per-op, since each version's session is a separate IDB.

Task 4 and Task 5 are near-identical in shape. The plan writes both out in full rather than "same as Task 4": the implementer may read them out of order and each is dispatched to a fresh context.

---

## 7. Key files, grouped

**The dispatch seam**
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — `:779-793` the classification-first arms, `:1103-1120` the version-helper precedent, `:1012-1040` the package-var test seams, `:1457-1469` the 543 branch of `GetCashSlotItemType` (read-only — not modified by this task), `:61-66` the common-prefix ownership check that already covers FR-5.3 for the open path
- `libs/atlas-constants/item/constants.go:116` — `ClassificationCharacterCreation = Classification(543)`
- `libs/atlas-tenant/tenant.go:88,93` — `IsRegion`, `MajorAtLeast`

**The name-check precedent** (copy its shape, not its scope)
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change.go` — the seam with an explicit `scope` parameter, the map-not-switch reason table, the announce helper
- `services/atlas-channel/atlas.com/channel/character/name_validity_requests.go` — `NameScopeWorld` vs `NameScopeTenant` and why they differ; the four `NameReason*` constants

**The codec precedent**
- `libs/atlas-packet/cash/clientbound/check_name_change.go` — immutable struct, `WithResolvedCode` builders, per-version address block, arm enumeration as compiled
- `libs/atlas-packet/cash/serverbound/item_use.go:14-24` — `UpdateTimeFirst`: GMS ≥ v87 and JMS lead, gms_v83/84 trail
- `libs/atlas-packet/cash/serverbound/item_use_incubator.go` — the trailing-`update_time` sub-body shape
- `libs/atlas-packet/test/{context.go,roundtrip.go}` — `pt.Variants`, `pt.Encode`, `pt.RoundTrip`

**The async seam**
- `services/atlas-login/atlas.com/login/kafka/consumer/seed/consumer.go` — the consumer shape to mirror (read-only; atlas-login is not modified)
- `services/atlas-character-factory/atlas.com/character-factory/kafka/{message,producer,consumer}/…/seed`, `…/consumer/saga/consumer.go` — Task 10's three touch points
- `services/atlas-channel/atlas.com/channel/remotemerchant/registry.go` — the in-memory registry to copy, including the tenant-scoped-`Sweep` reasoning at `:100-114`

**Coverage tooling**
- `tools/packet-audit` — `matrix`, `operations --check`, `fname-doc --check`, `gate-check`, `template --check`
- `docs/packets/audits/VERIFYING_A_PACKET.md`, `docs/packets/IMPLEMENTING_A_PACKET.md`
- `docs/packets/evidence/gms_v83/cash.clientbound.CashCheckNameChange.yaml` — evidence-record shape
- `docs/packets/gates.yaml:26-34` — the gate-row schema, if a new `MajorAtLeast` boundary is introduced

---

## 8. IDA sessions

From `mcp__ida-pro__idb_list` at plan time:

| version | session | state |
|---|---|---|
| gms_v83 | — (`MapleStory_dump.exe.i64`, pid 10044, port 13339) | **discovered, not adopted** — `idb_open` it first |
| gms_v84 | `46c2a2eb` | adopted |
| gms_v87 | `c0829805` | adopted |
| gms_v92 | `019cd393` | adopted |
| gms_v95 | `ecc757f4` | adopted |
| jms_v185 | `a977912e` | adopted (needed only for §6.3's opcode-271 identification) |

Checked-in exports are available as a fallback at `docs/packets/ida-exports/gms_v{83,84,87,92,95}.json` and `gms_jms_185.json`.

---

## 9. What must not change

- `services/atlas-login/` — not one file. Task 10 Step 5 verifies this with `git status --porcelain`.
- `template_gms_{48,61,72,79}_1.json` and `template_jms_185_1.json` — not opened. Task 7 Step 4 and Task 15 Step 4 verify.
- `GetCashSlotItemType`'s ~40 sibling `MajorVersion() >= 95` branches — PRD non-goal, task-227's boundary.
- Any previously-verified coverage cell of any other op — Task 15 Step 3 diffs `status.json` against `origin/main` for state flips.
- The cash-shop rename's `NameScopeTenant` — Task 12's changes must leave `cash_shop_check_name_change_test.go` green.

---

## 10. Escalation triggers

Stop and ask the user rather than deciding, if:

- **§6.3 finds jms_v185 opcode 271 is the sale dialog's own probe.** PRD scope says jms stays untouched; that finding contradicts it. A rename plus a scope change is the user's call, not a silent widening.
- **§4/§5 find a clientbound arm that requires server state the design did not anticipate** — e.g. the dialog expecting an in-session character-list refresh (design OQ-6 assumed none, since the channel has no character-select list).
- **§2 finds the submit sub-body carries an operation beyond creation** — the fname is `CharacterSale`, and PRD §2 puts anything beyond creation out of scope unless the derivation proves the same sub-body carries it.
