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

---

# Amendment 1 context (Tasks 16–26)

Added at the second `/plan-task` pass, after `design.md` §11 was appended
(`fb70df66d`) and Tasks 1–14 had already landed and been reviewed. Everything
above this line describes the original plan and still stands except where §11
supersedes it.

## A. Where Tasks 1–15 actually stand

- **Tasks 1–14: implemented, reviewed, committed** — `ddef3d665` … `0777d508c`,
  with `review-task-1.md` … `review-task-14.md`. The plan's checkboxes were
  never ticked; the commits and review artifacts are the record, not the boxes.
- **Task 15: never run.** Superseded by Task 26, which runs the same steps plus
  this amendment's blast-radius checks.
- **§11 A1's "the open-phase machinery must be deleted" is already done.**
  `maplelife/registry.go` carries only `PhaseSubmitted` and `SubmittedTTL`;
  there is no `PhaseOpen`, no `OpenTTL`, and `Sweep` is scoped to the one phase.
  Verified by reading the file at plan time, so no task was written for it.

## B. The finding that reshaped A5

§11 A5 argues at length that `POST characters/seed` cannot express a level-30
first-job character, and concludes "a Maple Life creation path in
`atlas-character-factory`" is needed. That is right, but A5 was written without
knowing that a *second* creation path already exists.

`POST /factory/characters/from-preset` and `buildPresetCharacterCreationSaga`
predate this branch entirely (`origin/main`, `0e3c15927` / `bd1447a73`,
task-171 era). `preset.Attributes` already carries `JobId`, `Level`, `MapId`,
`Meso`, `Stats`, `Equipment`, `Inventory` and `Skills[]{SkillId, Level}`, and
the builder already emits `CreateCharacter` + `AwaitInventoryCreated` +
`AwardAsset` per item + `CreateAndEquipAsset` per equip + `CreateSkill` per
skill, with atlas-data revalidation and a step-count-scaled timeout. That is
A5's contract minus three things: the player's look, the player's gender, and
the player's SP level.

**Decision (user ruling, this pass):** the `mapleLife` block is *self-contained
and ordinal-addressed* exactly as A5/A6 specify — it does **not** reference
admin preset UUIDs, so an admin editing or deleting a preset cannot change game
behaviour — but Task 22 converts a resolved class entry into a
`preset.RestModel` and reuses `buildPresetCharacterCreationSaga` rather than
writing a second saga builder. Rejected: pointing `mapleLife` entries at preset
ids (couples game content to admin data), and a dedicated
`buildMapleLifeCreationSaga` (duplicates the item/equip/skill construction and
its revalidation).

**Also discovered:** design C2's "the factory's seed path performs no duplicate
check of its own" is true of `Create` but **false of `CreateFromPreset`**, which
calls `p.nameClient.Check` at `factory/processor.go:286-296`. The new Maple
Life path inherits that check, so the channel's gate 4 name re-check is now a
second gate rather than the only one. Kept anyway — it produces the specific
`NAME_TAKEN_AT_SUBMIT` arm one round-trip earlier, and §5.2's TOCTOU note
already covers the window between them.

## C. The AP/SP gap and why it crosses three modules

§11 A2 requires "all remaining AP and SP left unspent" and A4 requires `nSP` to
be deducted from the SP pool. Neither is expressible today:
`saga.CharacterCreatePayload` (`libs/atlas-saga/payloads.go:1021-1044`) carries
`Level`, `Meso`, `Gm` and the four spent stats but **no AP or SP**, and neither
does the orchestrator's `CreateCharacterCommandBody` or `atlas-character`'s
`create(...)`. The storage already exists (`character/entity.go:57-58`:
`AP uint16`, `SP string`). There is no `AwardAP`/`AwardSP` saga action — only
`RebalanceAP` and `TransferAP`, which move *already-spent* points.

**Decision (user ruling):** extend the contract additively — `ap`/`sp` with
`omitempty` at every layer (Tasks 17–18). Rejected: creating at level 1 and
emitting 29 `AwardLevel` steps so `computeOnLevelAddedAP`/`computeOnLevelAddedSP`
grant the points (correct totals, but makes the result a function of
`atlas-character`'s level-up code rather than configuration, and inflates every
creation's saga); and shipping with `AP=0`/`SP=0` (contradicts A2/A4 outright).

Two deliberate choices inside that change:

- **`sp` is a `string`, not `[]uint32`.** `atlas-character` persists a ten-slot
  comma list (`entity.go:58`, parsed by `Model.SPs()` at `model.go:124-135`).
  Passing the same representation avoids a second shape of one value on the
  wire.
- **`RequestCreateCharacterProvider` stays positional at 21 parameters.** It is
  already 19. A struct refactor touches a contract `atlas-login`'s path also
  drives and is out of this amendment's scope — noted and declined, not
  overlooked. Task 17 Step 2 says so in the plan so a reviewer does not
  re-raise it as an oversight.

**Pre-existing bug found while reading that path, fixed in Task 18.**
`character/administrator.go:40` writes `SP: "0, 0, 0, 0, 0, 0, 0, 0, 0, 0"` —
with spaces — while `Model.SPs()` splits on `","` and `Atoi`s each element,
returning early on the first failure. `strconv.Atoi(" 0")` errors, so `SPs()`
has been returning a **one-element** slice for every character created through
this path instead of ten zeroes. Task 18's second test case asserts
`len(SPs()) == 10`, which fails against today's code.

## D. Two places the plan overrides `design.md` §11

Both because derived evidence outranks the amendment's prose (CLAUDE.md,
"Evidence & grounding"):

1. **A7: "each maps to a distinct `MAPLELIFE_ERROR` code."** No such enumeration
   exists. `CUICharacterSaleDlg::OnCreateNewCharacterResult` is a closed
   three-semantic-arm switch — `SUCCESS`, `NAME_TAKEN_AT_SUBMIT`,
   `UNKNOWN_ERROR` — decompiled on gms_v83 `0x7d77b0`, v87 `0x82e252`,
   v92 `0x7564f0`, v95 `0x777fc0` (`error.go:22-46`). Every gate that is not a
   name collision writes `UNKNOWN_ERROR` and is distinguished in the log.
   The one gain: a factory `409` now maps onto `NAME_TAKEN_AT_SUBMIT` instead of
   the generic arm.
2. **A4: `nSP` on ordinals ≥ 2 "logged and clamped to `0`."** A7 is the later
   clause and says "rejected, not clamped-and-created". The plan rejects — a
   clamped submit silently creates a character the player did not ask for.

## E. Why Task 16 exists

None of the Maple Life *content* is derivable from anything in this repo:

- WZ data is not checked in — `atlas-data` ingests
  `Etc.wz/MakeCharInfo.img.xml` at runtime (`data/data/processor.go:190`).
- The two per-gender WZ paths `LoadNewCharInfo` (gms_v95 `0x777790`) uses come
  from `StringPool::GetBSTR(1525)` / `(1526)`, and the literal is **not in the
  binary**: searching the gms_v95 IDB for `MakeCharInfo` as UTF-16LE returns
  zero matches, so `StringPool` is not plain-literal backed.
- No item-name catalogue exists in-repo; `2000002` appears only in
  `atlas-reward-pools` **test fixtures**.
- The seeded `characters.presets` are all level 120–200 admin GM presets — no
  first-job grounding.

So Task 16 derives it all into `maple-life-content.md` first, and Task 20 reads
`<content §N>` markers from it. This mirrors exactly how Tasks 1–2 handled wire
facts, including the placeholder-scan note.

**Strong lead, not a finding:** `atlas-data`'s own reader treats every
root-level `MakeCharInfo.img` node that is not `Info` or `Name` as a character
type (`characters/templates/reader.go:31-38`), and its test fixture carries a
`PremiumCharMale` beside `Info/CharMale` and `Info/CharFemale`
(`reader_test.go:103`). Task 16 must confirm `PremiumChar{Male,Female}` against
real WZ data before recording it, and must not fall back to copying the
`CharMale`/`CharFemale` lists — those are a different node and may legitimately
differ.

**`GET /data/characters/templates` already exposes this data** and nothing in
the repo consumes it. The plan does not have the factory call it: the factory
validates against tenant configuration everywhere else (`validOption`,
`processor.go:474-485`), and the option lists are content an operator should be
able to correct without a redeploy. The endpoint is the source operators
populate the seed data *from*.

## F. Task sizing

`tools/plan-lint.sh` F4 warns on any task over ~6 files or crossing more than
one service. Two amendment tasks trip it deliberately:

- **Task 17** spans `libs/atlas-saga` and `atlas-saga-orchestrator` (6 files).
  Splitting it would land a payload field nothing reads, which is a stub by
  another name. It is a single mechanical addition repeated at six sites — the
  "same mechanical change repeated" case the sizing rule exempts.
- **Task 22** carries the widening of `preset.Attributes` with `AP`/`SP` across
  three mirrors plus the conversion and the saga hand-off. Its Step 2 carries an
  explicit `PARTIAL` instruction: if the widening reaches beyond those four
  files, hand back rather than grow, and it splits into "widen the preset
  attributes" / "convert and build".

Everything else is one module and ≤ 6 files.

`tools/plan-lint.sh` exits 0 on the combined document. Its four remaining
F4 warnings are all in the ORIGINAL plan's Tasks 5, 9, 10 and 14 — already
implemented and reviewed — not in this amendment; no amendment task trips
F4.

## G. Still open after this plan

- **Class ordinals 2/3/4.** §11 A6 — config-ordered, so a wrong order is a
  seed-data fix. Task 16 Step 2 attempts `CUICharacterSaleDlg::OnCreate`
  (gms_v95 `0x77adc0`) and its `m_strClassName[5]` / `m_apCanvasClass` sources;
  if that fails, the value ships marked UNCONFIRMED and **must be pinned before
  live testing** by reading the received ordinal from channel logs while
  picking each class in a real client.
- **Gender.** §11 A9 — the MapleSEA guide says gender is not player-selectable,
  yet gms_v95 `OnButtonClicked` toggles `m_nGender` on control index 4 and the
  wire carries it. Decided for now: the channel forwards the packet's value and
  the factory validates it is `0` or `1`. Whether to override it from the
  account's existing characters is not decided and is not blocking.
- **SP slot 0.** The plan spends the player's `nSP` out of slot 0 of the
  configured pool. That is right for Explorer-family first jobs; the ten-slot
  array exists for Evan. Recorded here because it is an assumption, not a
  derivation.
- **The `MAPLELIFE_ERROR` `nParam` diagnostic** is still always sent as `0`
  (`error.go`'s `MapleLifeErrorBody`). Threading a real diagnostic value through
  would let the client's "unknown error (%d)" string distinguish the gates the
  three-arm enumeration collapses. Out of scope; noted as the natural follow-up
  if operators find `UNKNOWN_ERROR` too coarse in practice.
