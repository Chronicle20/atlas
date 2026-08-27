# Maple Life — In-Game Character Creation (`Cash/0543`) — Design

Version: v2 (amended — see §11)
Status: Draft
Created: 2026-08-21
Input: `docs/tasks/task-246-maple-life-character-creation/prd.md` (approved)

---

## 0. Corrections to the PRD's factual premises

Four PRD statements did not survive contact with the repository. They change the
shape of the design, so they are settled here before anything is designed on top
of them. Each is sourced.

### C1 — `CUICharacterSaleDlg::SendCheckDuplicateIDPacket` is **not** `CHECK_CHAR_NAME`

PRD FR-3.1 asserts the Maple Life duplicate-name probe arrives as
`CHECK_CHAR_NAME`, "opcode 0x100 on gms_v83". The registry says otherwise.

From `docs/packets/audits/status.json`:

| Op (registry row) | fnames | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms_v185 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `CHECK_CHAR_NAME` (sb) | `CCashShop::SendCheckDuplicateIDPacket`, `CLogin::SendCheckDuplicateIDPacket`, `sub_500693`, `sub_565537` | 17 | 21 | 21 | 21 | 21 | 21 | 21 | 21 | 21 | 8 |
| `JMS_SLASH_COMMAND` (sb) | **`CUICharacterSaleDlg::SendCheckDuplicateIDPacket`** | -1 | -1 | -1 | -1 | **-1** | **-1** | **270** | **-1** | **311** | 271 |

Three consequences:

1. `CHECK_CHAR_NAME` is 21 (0x15) on every GMS build from v61 up, not 0x100.
   The v83 template already binds 0x15 for the `channel` service to
   `CashShopCheckNameChangeHandle` and for `login` to `CharacterCheckNameHandle`
   (`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`).
   Its fname list carries the `CCashShop::` and `CLogin::` senders — **not** the
   `CUICharacterSaleDlg::` one.
2. The registry row that *does* carry the Maple Life sender is named
   `JMS_SLASH_COMMAND`. That name is wrong for this fname, or the row is an
   accidental merge of two unrelated ops that happen to share an opcode slot on
   jms_v185. Either way the registry is currently unusable as the routing source
   for this family. **Fixing it is a prerequisite of this task, not a follow-up**
   (see §7).
3. On **gms_v83, v84, and v92** that row is `-1` — i.e. Atlas currently knows of
   no serverbound duplicate-check opcode on three of the five in-scope versions,
   even though `MAPLELIFE_RESULT` (`OnCheckDuplicatedIDResult`) *is* defined
   there (349 / 349 / 404). That gap is the single largest unknown in this task
   and is what Phase 0 of the plan must close first.

FR-3.1's requirement (answer the probe with `MAPLELIFE_RESULT`, scope `WORLD`,
reuse `checkNameValidity`) stands. Only its opcode claim is retracted.

### C2 — `POST characters/seed` is asynchronous and does **not** check names or slots

`services/atlas-character-factory/.../factory/resource.go:104-121` returns
**202 Accepted** with a `transactionId`; `processor.go:88-177` validates
synchronously and then emits a `CharacterCreation` saga.

What it validates synchronously (400 on failure, via `categorizeError`):
name *format*, gender, job index, creation-template existence for
(jobIndex, subJobIndex, gender), and face / hair / hairColor / skinColor / top /
bottom / shoes / weapon against that template.

What it does **not** do:

- **No duplicate-name check.** `Create` calls `validName` only. The
  `nameClient.Check` call exists solely on the `CreateFromPreset` path
  (`processor.go:283-292`). The seed path Atlas has used for login character
  creation since forever has never checked duplication.
- **No character-slot-limit check.** Nothing in `atlas-character-factory`,
  `atlas-character`, or the saga reads `Account.CharacterSlots`.

So FR-4.4 and FR-4.5 have no existing server-side owner. The channel must own
both pre-checks. This is not gold-plating — it is the only place the check can
live without widening the task into a factory redesign.

### C3 — the async completion signal already exists, keyed by account id only

`EVENT_TOPIC_SEED_STATUS` carries `CREATED` (with `characterId`) and `FAILED`
(with `reason`) events, both enveloped as
`StatusEvent{AccountId, Type, Body}` — see
`services/atlas-character-factory/.../kafka/message/seed/kafka.go` and its
producer. `atlas-character-factory`'s saga bridge
(`kafka/consumer/saga/consumer.go`) filters orchestrator status events by
`SagaType == CharacterCreation` and re-emits them onto that topic;
`atlas-login`'s `kafka/consumer/seed/consumer.go` consumes them and resolves the
waiting session by `AccountId`.

atlas-channel can consume the identical topic. The one weakness: **the envelope
carries no transaction id**, even though the bridge has `e.TransactionId` in
hand and drops it. §4.3 addresses that.

### C4 — no configuration work is needed; Open Question 5 is answered "already resolvable"

`requests.RootUrlFor(ctx, domain)` (`libs/atlas-rest/requests/url.go:34-64`)
resolves from **environment variables**, not tenant service configuration: a
`CHARACTER_FACTORY_SERVICE_URL` override if present, else `BASE_SERVICE_URL`,
namespace-rewritten for sparse environments. `BASE_SERVICE_URL` is set for every
service from the shared `deploy/k8s/base/env-configmap.yaml:19` (and the
per-overlay equivalents), and the ingress already routes
`^/api/characters/seed(/.*)?$` to `atlas-character-factory`
(`deploy/k8s/base/routes.conf.template.generated:386-389`).

atlas-channel therefore reaches `POST characters/seed` today with **no**
service-config seed row, no ingress change, and no `ns-vars` change. PRD §5
"Configuration" and §7's `deploy/k8s` row are both no-ops.

---

## 1. Architecture at a glance

```
                    channel socket
  USE_CASH_ITEM (543 sub-body)  ──┐
  USE_MAPLELIFE (v95, 303)      ──┼──> maplelife.Open      ── writes nothing;
                                   │                          records pending record
  MAPLELIFE_CHECK_NAME (per-ver) ──┼──> maplelife.CheckName ── checkNameValidity(WORLD)
                                   │                          ──> MAPLELIFE_RESULT
  <submit sub-body>              ──┴──> maplelife.Create   ── pre-checks
                                                              ──> POST characters/seed (202)
                                                              ──> pending.transactionId

  EVENT_TOPIC_SEED_STATUS  ──> channel kafka/consumer/seed
        CREATED ──> destroy cash item (saga) ──> MAPLELIFE_ERROR{success code}
        FAILED  ──> item untouched          ──> MAPLELIFE_ERROR{mapped code}
```

Five units, each independently testable:

| Unit | Package | Responsibility | Depends on |
|---|---|---|---|
| Dispatch arm | `socket/handler/character_cash_item_use.go` | classify 543, delegate | `item.Classification*`, tenant |
| Dialog handlers | `socket/handler/maple_life_*.go` | decode, pre-check, respond | pending store, REST clients |
| Pending store | `channel/maplelife` | per-account in-flight dialog state | nothing (in-memory) |
| Factory client | `channel/character/factory` | `POST characters/seed` | `requests.RootUrlFor` |
| Result consumer | `channel/kafka/consumer/seed` | CREATED/FAILED → consume + announce | pending store, saga, writers |

`libs/atlas-packet` gains the codecs; `atlas-configurations` gains the routing.
`atlas-login`, `atlas-character-factory`'s creation path, and `deploy/k8s` are
untouched (one narrow additive exception in §4.3, below).

---

## 2. Dispatch: route on classification only, never on the cash-slot type

**Decision.** The 543 arm branches on `item.GetClassification(itemId) ==
item.ClassificationCharacterCreation` (543,
`libs/atlas-constants/item/constants.go:116`) and on a version-applicability
helper. It never compares `it` (the `CashSlotItemType`) at all.

Placement: with the other classification-first branches, i.e. after the
`nameChangeCashSlotItemType` / `worldTransferCashSlotItemType` coupon-cancel
arms and immediately alongside `ClassificationRemoteMerchant` /
`ClassificationMegaphones` (`character_cash_item_use.go:786-793`) — **before**
any `it ==` comparison. `GetCashSlotItemType` itself is not modified (PRD
non-goal; task-227's boundary).

Why not gate on `it`, even with a helper: `GetCashSlotItemType`'s 543 branch
(`character_cash_item_use.go:1457-1469`) can return 57, 58, 65, or 66, and every
one of those values is claimed by something else —

| Value | Other claimant | Site |
|---|---|---|
| 57 / 58 | `ClassificationPetMultiConsumable` | `:1485-1490` |
| 65 | `CashSlotItemTypeSealTimedV95` | `:970` |
| 66 | `CashSlotItemTypeViciousHammer` (GMS < 95) | `:991` |

Routing on classification makes the collision structurally impossible rather
than merely avoided, and it makes PRD **Open Question 2 irrelevant to routing**:
whether the client's `itemId/1000-5431 > 1` comparison is signed or unsigned
decides *which sub-body the client encoded*, not *which Atlas arm runs*. The
question still has to be answered by FR-1.2's derivation — it decides whether
`ItemUseMapleLife` needs one body shape or two — but a wrong answer can no
longer misroute a packet into the pet or hammer arm.

**Version applicability.** A named helper, per FR-2.3:

```go
// mapleLifeSupported reports whether the tenant's client has the Maple Life
// dialog at all. GMS v83+ only: USE_MAPLELIFE, MAPLELIFE_RESULT and
// MAPLELIFE_ERROR are all n-a on gms_v48/61/72/79 and on jms_v185
// (docs/packets/audits/status.json).
func mapleLifeSupported(t tenant.Model) bool {
    return t.IsRegion("GMS") && t.MajorAtLeast(83)
}
```

On `!mapleLifeSupported(t)` the arm falls through to the existing warn path
without decoding a sub-body and without writing anything (FR-2.4). That is the
only place in this task where the pre-83 GMS builds and jms_v185 are mentioned,
and it is a read-only guard — their wire behaviour is unchanged.

**Rejected alternative:** a `mapleLifeCashSlotItemType(t)` helper mirroring
`nameChangeCashSlotItemType`. It reads consistently with task-227 but buys
nothing here: task-227's coupons have a *unique* type value per version, so `it`
is a sound discriminator there. 543's is not. Consistency with a precedent is
not worth reintroducing a collision the PRD explicitly forbids (FR-2.2).

---

## 3. Two entry points, one internal flow

> **SUPERSEDED by §11 A1.** There is no dialog-open packet: `Cash/0543` is the
> submit (`SendCreateNewCharacter`), and `USE_MAPLELIFE` (303) was struck as an
> orphan. This section's `beginMapleLife` normalisation is withdrawn.

PRD Open Question 1 asks whether v95 uses `USE_MAPLELIFE` (303) instead of, or
in addition to, the `USE_CASH_ITEM` sub-body. The design does not need the
answer to be structured correctly — only to be *wired* correctly.

Both entry points normalise onto one internal call:

```go
func beginMapleLife(l, ctx, wp) func(s session.Model, itemId item.Id, source slot.Position, updateTime uint32)
```

- `CharacterCashItemUseHandleFunc`'s 543 arm calls it after decoding the
  `ItemUseMapleLife` sub-body.
- `MapleLifeUseHandleFunc` (v95 only, opcode 303) calls it after decoding
  `maplelifesb.Use`.

If FR-1.3's derivation shows v95 sends **only** 303, the 543 arm still exists on
v95 as a defensive no-op-with-log rather than being version-gated out — a client
that sends the sub-body anyway must not hit the silent fallthrough FR-5.4
forbids. If it shows v95 sends **both in sequence**, the pending-record store
(§4) idempotently absorbs the second: an `Open` for an account that already has
a live record refreshes it rather than creating a second.

Cost of this shape: one extra indirection. Benefit: the plan's Phase 0 can
answer OQ-1 either way without restructuring anything downstream.

---

## 4. State and correlation

### 4.1 Why any state at all

> **PARTIALLY SUPERSEDED by §11 A1.** Item 1 below is void: the probe has its own
> opcode (routing outcome (A)) and is answered statelessly. Items 2 and 3 stand,
> but the `Open` phase in §4.2 does not — a record is created at submit time, in
> `Submitted`, and the open-phase machinery must be deleted.

Three things force a small amount of per-account transient state:

1. **The name probe may share an opcode.** If FR-1.4's derivation shows that on
   v83/84/92 the sale dialog reuses `CHECK_CHAR_NAME` (21) — which the channel
   already binds to `CashShopCheckNameChangeHandle` — then one opcode must
   produce two different clientbound replies (`CASHSHOP_CHECK_NAME_CHANGE` 328
   vs `MAPLELIFE_RESULT` 349 on v83). The packet body is `EncodeStr(name)` in
   both cases and carries nothing to distinguish them. **Only server-held state
   can disambiguate: which cash dialog did this character open?**
2. **The creation result is asynchronous** (C2). Something must remember which
   character/session/item the eventual `CREATED`/`FAILED` belongs to.
3. **Ownership at submit time** (FR-5.3). The item id and slot are re-verified
   against what the player actually held when the dialog opened.

### 4.2 The pending store

A new `atlas-channel/maplelife` package: an in-memory, tenant-partitioned map
keyed by **account id** (matching the seed event's key), holding

```
Pending{ tenantId, accountId, characterId, worldId, itemId, slot, updateTime,
         phase (Open|Submitted), transactionId, candidateName, openedAt }
```

- Never persisted (PRD §6). Dropped on disconnect and on a TTL sweep.
- `Open` is idempotent per account (see §3).
- A `CheckName` or `Create` with no live record is a rejection, logged, with the
  dialog's generic-failure code written — never a silent drop (FR-5.4).

TTL: the same order as the orchestrator's 10s character-creation backstop
(`buildCharacterCreationSaga`, `SetTimeout(10 * time.Second)`) for the
`Submitted` phase, and a longer human-scale TTL (minutes) for `Open`, since a
player types a name at human speed. Exact values are a plan-phase detail; the
constraint is that `Submitted` must outlive the saga's own timeout so a
timed-out saga's `FAILED` still finds its record.

**Rejected alternative:** hang the state off `session.Model`. The session is
keyed by character; the seed event is keyed by account; and the session may be
gone when the event lands (§5.4). An account-keyed store outside the session is
the honest shape.

### 4.3 Correlating the async result

**Chosen: transaction id, with account id as fallback.**

`POST characters/seed` returns `{transactionId}` (`CreateCharacterResponse`).
The channel stores it on the pending record. The factory's saga bridge already
has `e.TransactionId` in hand (`kafka/consumer/saga/consumer.go:57,88`) and
discards it; adding it to the seed status envelope is a **two-line additive
change**:

```go
type StatusEvent[E any] struct {
    AccountId     uint32 `json:"accountId"`
    TransactionId string `json:"transactionId,omitempty"`   // new
    Type          string `json:"type"`
    Body          E      `json:"body"`
}
```

`atlas-login` deserialises into its own copy of the struct and ignores unknown
fields, so **atlas-login stays byte-for-byte unmodified** (PRD §7's hard
constraint holds). This is exactly the "narrow addition" §7 permits, and it is
producible here rather than deferred.

The channel consumer matches on transaction id when present and falls back to
account id when it is absent (an older factory build mid-rollout).

**Rejected alternatives:**

- *Account id alone.* Simpler, zero factory change. Rejected because it is
  silently wrong whenever an account has any other in-flight seed — and the
  failure mode (consuming a Maple Life item because an unrelated creation
  succeeded) is exactly the "support ticket about a lost cash item" the PRD's
  operator story exists to prevent. The window is narrow; the consequence is
  not.
- *Channel polls `characters?accountId=&worldId=` until the character appears.*
  Rejected: the polling anti-pattern (CLAUDE.md / `docs/tooling-conventions.md`),
  and it has no failure signal at all — a failed saga looks identical to a slow
  one forever.

---

## 5. The creation flow, decision by decision

### 5.1 Duplicate-name probe (FR-3.1 – FR-3.3)

Reuse `character.NewProcessor(l, ctx).CheckNameValidity(name, worldId,
character.NameScopeWorld)` — the client already exists
(`atlas-channel/character/name_validity_requests.go`). **`WORLD`, not
`TENANT`**: this is a creation, and `TENANT` is deliberately the *stricter*
rename-only scope (that file's own doc comment; task-227 FR-3.2).

Reason mapping follows `cash_shop_check_name_change.go:82-104` exactly — a named
`map[string]MapleLifeResultCode`, not a switch, so a test can assert all four
`NameReason*` values are covered. An unmapped reason maps to the dialog's
generic-failure code and logs at **error** level (FR-3.3).

Whether this handler is a *new* handler bound to a Maple-Life-specific opcode
(v87/v95 path, per C1) or a *branch inside* `CashShopCheckNameChangeHandleFunc`
selected by the pending record (the shared-opcode contingency, §4.1) is settled
by Phase 0. The design accommodates both: the mapping table and the
`checkNameValidity` call live in `maple_life_check_name.go` either way; only the
caller differs.

### 5.2 Submit-time pre-checks, in order

> **SUPERSEDED by §11 A7.** Gate 1 has no subject and is dropped; gate 2 sources
> item id and slot from the submit packet's own header; a class/SP gate is added.

All run before `POST characters/seed`. Each has a distinct client-rendered
`MAPLELIFE_ERROR` code; none consumes the item.

1. **Live pending record exists** for this account, phase `Open`. Else reject.
2. **Ownership** (FR-5.3): `cashItemInSlotFunc(l, ctx, s.CharacterId(),
   int16(pending.slot))` still returns `pending.itemId`, and that item's
   classification is 543. Reuses the existing seam.
3. **Slot limit** (FR-4.4): `account` processor → `CharacterSlots()`
   (`atlas-channel/account/model.go:40`, already carried in the channel's
   account model) vs. the count from
   `characters?accountId=&worldId=` — the URL builder `accountInWorldUrl`
   already exists in `atlas-channel/character/requests.go` and is drained with
   `requests.DrainProvider` (paged). No new endpoint anywhere.
4. **Name re-check** (FR-4.5): `checkNameValidity(..., WORLD)` again. Passing
   the probe earlier is not sufficient; the factory's seed path performs no
   duplicate check of its own (C2), so this is the *only* duplicate gate.
5. **Account / world from session** (FR-4.2): `s.AccountId()`, `s.WorldId()`.
   The packet's own values, if the derivation shows it carries any, are decoded,
   logged on mismatch, and discarded.

TOCTOU on 3 and 4 is accepted and stated: an account has one channel session,
and the residual race (another player taking the name between check and saga)
surfaces as a saga `FAILED` → mapped error → item retained. The failure mode is
correct, just later.

### 5.3 Look validation — the factory is the single validator

> **SUPERSEDED by §11 A5.** The factory is still the single validator, but Maple
> Life does not go through `POST characters/seed` or its creation-template rules
> at all. The eleven-rule table below does not apply to this path; the HTTP
> status-mapping table further down does.

**Decision.** The channel does **not** re-implement look validation. It submits
and maps the factory's synchronous rejection.

FR-4.3 reads "MUST be validated … before the call" and then "Where
`atlas-character-factory` already validates, a factory rejection MUST be
surfaced as a client-rendered `MAPLELIFE_ERROR`". The second clause is the
operative one, because the factory validates **every** field FR-4.3 enumerates —
face, hair, hairColor, skinColor, top, bottom, shoes, weapon, gender, jobIndex,
subJobIndex — against the tenant's own creation template
(`factory/processor.go:106-153`), and returns `400` for each
(`resource.go:78-99`). Duplicating that in atlas-channel would mean fetching the
same tenant template through a second client and re-encoding eleven rules that
can drift silently from the factory's.

The channel therefore maps the HTTP status:

| Factory response | `MAPLELIFE_ERROR` code | Item |
|---|---|---|
| `202 Accepted` | (no immediate write; await the seed event) | held |
| `400 Bad Request` | invalid-look / invalid-name code | retained |
| `4xx` other, `5xx`, transport error | generic-failure code | retained |

The security requirement in PRD §8 ("a crafted packet must not create a
character with an arbitrary look") is satisfied — the look is validated
server-side, in the service that owns character creation. It is simply not
validated *twice*.

*This is the one place the design reads FR-4.3 more narrowly than its first
clause. Called out explicitly so it can be overridden at plan time if the
intent was genuinely a second copy.*

### 5.4 Item consumption — create first, consume on `CREATED`

FR-5.1/FR-5.2 prefer create-then-consume, and the async seed makes that not just
preferable but structurally necessary: there is no synchronous success to
consume against.

| Event | Item | Client |
|---|---|---|
| pre-check rejection | untouched | `MAPLELIFE_ERROR{specific code}` |
| factory 400 / 5xx / unreachable | untouched | `MAPLELIFE_ERROR{mapped code}` |
| seed `FAILED` | untouched | `MAPLELIFE_ERROR{generic failure}` |
| seed `CREATED` | destroyed | `MAPLELIFE_ERROR{success code}` |

Consumption uses a channel-side saga with a single
`saga.DestroyAssetFromSlot` step against the cash compartment
(`InventoryType: byte(inventory.TypeValueCash)`, `Slot: pending.slot`,
`TemplateId: pending.itemId`, `Quantity: 1`) — the same construction the
incubator arm uses at `character_cash_item_use.go:626-637`. Not
`RequestItemConsume`: that path routes through atlas-consumables' *use* semantics
(effects, cooldowns), which is not what happens here. The item is destroyed
because a purchase was fulfilled.

Because FR-5.2's compensating restore is never reached under this ordering,
**no restore path is written**. The acceptance criterion "a test covers the
restore path if consume-first ordering is forced" is satisfied vacuously, and
the plan must say so rather than leave the box ambiguous.

Two residual edges, both decided rather than left open:

- **Character created, destroy step fails.** The character is real and stays.
  The item survives. Logged at **error** with account/character/item/transaction
  ids. Rolling back a created character to reclaim a cash item is destructive
  and disproportionate; the failure is operator-visible instead.
- **Session gone before `CREATED` lands.** Still destroy the item — the
  entitlement was spent and the character exists; leaving it would let one item
  produce two characters. No client write is attempted; logged at info, mirroring
  atlas-login's own disconnected-session handling
  (`login/kafka/consumer/seed/consumer.go:82-88`).

---

## 6. Packet library shape

New package `libs/atlas-packet/maplelife/`, not an extension of `cash/`:

```
maplelife/clientbound/result.go        MAPLELIFE_RESULT  (OnCheckDuplicatedIDResult)
maplelife/clientbound/error.go         MAPLELIFE_ERROR   (OnCreateNewCharacterResult)
maplelife/serverbound/use.go           USE_MAPLELIFE     (gms_v95, 303)
maplelife/serverbound/check_name.go    the per-version duplicate probe (C1)
cash/serverbound/item_use_maple_life.go  the 543 ItemUse sub-body
```

The 543 sub-body stays in `cash/serverbound` because it *is* an `ItemUse`
sub-body and sits with its `item_use_*.go` siblings; the standalone ops get
their own family package rather than growing `cash/`, which already holds 30+
files.

Codecs are immutable with **both** `Encode` and `Decode` (FR-6.1). Version
divergence uses the `MajorAtLeast` idiom, never a raw `> N`.

**Result and error codes come from the template, not from Go.** Following
`CashShopCheckNameChange`'s precedent in
`template_gms_83_1.json` (`"options": {"operations": {"AVAILABLE": 0, "TAKEN": 1,
"UNKNOWN_ERROR": 255}}`), each writer declares its code table in
`options.codes` / `options.operations` per version. The full enumeration FR-1.4
derives per version lands in the templates; the Go side references named keys.
This is what makes "no hard-coded literals" (FR-2.3, PRD §8) true for a family
whose codes genuinely differ across five versions.

Writers `MapleLifeResultWriter` / `MapleLifeErrorWriter` registered in
`atlas-channel/main.go` alongside `cashcb.CashShopCheckNameChangeWriter`
(`main.go:685`); handlers in the `handlerMap` alongside
`cashsb.CashShopCheckNameChangeHandle` (`main.go:1005`).

Routing lands in `template_gms_{83,84,87,92,95}_1.json` with
`"services": ["channel"]`. `template_gms_{48,61,72,79}_1.json` and the jms
template are not opened (FR-6.4).

---

## 7. Registry hygiene: the `JMS_SLASH_COMMAND` row

Per C1, the registry row carrying
`CUICharacterSaleDlg::SendCheckDuplicateIDPacket` is named `JMS_SLASH_COMMAND`
and spans gms_v87 (270), gms_v95 (311), jms_v185 (271).

This must be resolved before any routing is written, because the coverage matrix
and the operations registry are the source of truth the templates and
`packet-audit` check against. The blocker is a prerequisite this task can
produce itself (CLAUDE.md "finish producible work"), so it is in scope:

1. Derive what jms_v185 opcode 271 actually is. If it genuinely is a JMS slash
   command, **split** the row: `JMS_SLASH_COMMAND` keeps jms_v185 271 and goes
   `n-a` on GMS; a new `MAPLELIFE_CHECK_NAME` serverbound row takes the GMS
   columns.
2. If instead 271 is jms's own sale-dialog probe, the row is simply misnamed —
   **rename** it, and re-examine whether the family is truly `n-a` on jms_v185
   (PRD scope says jms stays untouched; if the derivation contradicts that, it
   is an escalation to the user, not a silent scope change).
3. Fill the gms_v83/84/92 columns from the derivation, or record them as
   genuinely absent with the address evidence that shows the dialog uses another
   op there.

Only then are the templates edited.

---

## 8. Testing strategy

**Dispatch regression (acceptance criterion, non-negotiable).** A table-driven
test over `CharacterCashItemUseHandleFunc` asserting that on a pre-95 tenant and
a v95 tenant, `PetMultiConsumable` (57/58), `SealTimed`/`SealTimedV95` (64/65),
and `ViciousHammer`/`ViciousHammerV95` (66/67) still reach their existing arms
after the 543 branch is inserted. Because the 543 arm routes on classification
(§2), this test is asserting the collision *cannot* occur rather than that it
happens not to.

**Handler tests** use the package-var seam pattern already established in this
file (`cashItemInSlotFunc`, `requestItemConsumeFunc`, `karmaCharacterProcessorFunc`
— `character_cash_item_use.go:1012-1040`). New seams:
`seedCharacterFunc`, `mapleLifeNameValidityFunc`, `accountSlotsFunc`,
`charactersInWorldFunc`, `destroyCashItemFunc`. No live Kafka, no live REST.
Scope is asserted *through* the seam (a test proves the handler asked for
`WORLD`), following `checkNameChangeValidityFunc`'s doc comment.

**Consumer tests** drive `CREATED` and `FAILED` envelopes through the new
channel seed consumer against a seeded pending store, asserting: correct
transaction-id match, correct fallback on absent transaction id, item destroyed
exactly once on `CREATED`, never on `FAILED`, and the disconnected-session path
still destroys and does not panic.

**Packet tests** are byte fixtures with `packet-audit:verify` markers, one per
op × version, pinned evidence records, matrix regenerated
(`docs/packets/audits/VERIFYING_A_PACKET.md`). A cell that does not promote in
`STATUS.md` is a failure, not a claim (FR-6.3).

Builders per repo convention; no `*_testhelpers.go`.

---

## 9. Sequencing constraint for the plan

Everything downstream of the wire is blocked on derivation, and the derivation
has a branch in it. The plan must open with a **Phase 0** that produces
`derivation.md` and answers, with per-version IDA addresses:

- OQ-1 — does v95 use 303, the 543 sub-body, or both? (decides §3's wiring)
- C1 — what is the duplicate-probe opcode on v83/84/92? Does it collide with
  `CHECK_CHAR_NAME` (21)? (decides §5.1's caller and whether §4.1's
  disambiguation-by-state is needed at all)
- FR-1.2 — signed or unsigned `itemId/1000-5431` in the client's own
  `get_cashslot_item_type`? (decides one sub-body shape or two)
- FR-1.4 — the full `MAPLELIFE_RESULT` / `MAPLELIFE_ERROR` code enumerations per
  version (populates the template `options` tables of §6)
- §7 — what jms_v185 opcode 271 is (decides split vs rename)

No implementation phase may begin before Phase 0 lands. No arm, code, or opcode
may be written from this document — every table above that names a numeric value
cites `status.json` and must be re-read from the registry at implementation time.

---

## 10. Open questions carried forward

Resolved by this design (with sources), no longer open:

- **OQ-3** (what distinguishes the three 543 items) — irrelevant to routing
  under §2; still worth recording in `derivation.md` from `Item.wz` `spec`, but
  it gates nothing.
- **OQ-4** (is the probe already routed on the channel) — partially: on v83 the
  channel binds opcode 21 to `CashShopCheckNameChangeHandle`, and per C1 that is
  the *cash-shop* sender, not the sale dialog's. Whether the sale dialog reuses
  it is Phase 0's question.
- **OQ-5** (does the channel resolve `CHARACTER_FACTORY`) — yes, via
  `BASE_SERVICE_URL` + the existing ingress route (C4). No work.
- **OQ-6** (in-session refresh of the character list) — the channel has no
  character-select list to refresh; the new character appears at next login via
  atlas-login's existing path. Nothing to do unless Phase 0's dialog trace shows
  the client expects a further write.
- **OQ-7** (distinct slot-limit rejection from the factory) — moot: the factory
  never checks slots (C2), so the channel owns the check and owns the distinct
  code (§5.2 step 3). No factory change needed for this.

Still open, for the user rather than for derivation:

- **§5.3** — the design treats atlas-character-factory as the single look
  validator and surfaces its 400, rather than writing a second copy of the
  eleven template rules in atlas-channel. If FR-4.3's first clause was meant
  literally, say so at plan time.
- **§4.3** — adding `TransactionId` to the seed status envelope touches
  `atlas-character-factory`'s message + bridge (additive, login unaffected).
  PRD §7 permits a narrow factory addition; confirm this counts.

---

## 11. Amendment 1 — the submit rewire and the level-30 first-job contract

Version: v2
Amended: 2026-08-21
Supersedes: §3 in full, §4.1 item 1, §4.2's `Open` phase, §5.2 gate 1, §5.3 in
full, and §10's first "still open" bullet.
Sources: `bug-543-is-the-submit-not-the-open.md`, `derivation.md` §2,
`selected-al-derivation.md`, this pass's gms_v95 decompiles (session
`ecc757f4`), a user ruling recorded below, and the MapleSEA "Maple Life Item
Guide" the user supplied.

### A1 — there is no dialog-open packet; `Cash/0543` is the submit

§3 was written before `derivation.md` existed and assumed the `Cash/0543`
`ItemUse` sub-body signalled *dialog open*. It does not. The sub-body is
`CUICharacterSaleDlg::SendCreateNewCharacter`
(`libs/atlas-packet/cash/serverbound/item_use_maple_life.go:13-14`), and
gms_v95 `0x77a240` confirms its body verbatim:

```c
COutPacket::Encode4(&oPacket, get_update_time());
COutPacket::Encode2(&oPacket, this->m_nPOS);
COutPacket::Encode4(&oPacket, this->m_nItemID);
COutPacket::EncodeStr(&oPacket, CCtrlEdit::GetText(this->m_pEdit.p));
for (v2 = 0; v2 <= 3; ++v2)
    COutPacket::Encode4(&oPacket, CUICharacterSaleDlg::GetSelectedAL(this, v2));
COutPacket::Encode4(&oPacket, this->m_nGender);
COutPacket::Encode4(&oPacket, this->m_nCurrentClass);
COutPacket::Encode4(&oPacket, this->m_nSP);
COutPacket::Encode4(&oPacket, get_update_time());
```

The dialog is opened locally by the client. `USE_MAPLELIFE` (303) was struck in
Task 1 as an orphan CSV placeholder no client path constructs, so the feature's
**entire serverbound surface is two messages**: the duplicate-name probe
(`SendCheckDuplicateIDPacket`, its own opcode, routing outcome (A)) and this
submit.

Consequences, all of which the branch has already absorbed (`50c79fadf`,
`0777d508c`):

- §3's `beginMapleLife` normalisation and its "two entry points" framing are
  **withdrawn**. One arm, one flow: decode → gates → create.
- §4.1 item 1 ("the probe may share `CHECK_CHAR_NAME`'s opcode and need
  disambiguating") is **void** under routing outcome (A). The probe is answered
  statelessly.
- The pending registry keeps `Take`, `TakeByTransactionId`, `Submit`,
  `SubmittedTTL`; `PhaseOpen`, `OpenTTL` and the open-phase sweep are
  **vestigial and must be removed**, not left dormant.
- §5.2 gate 1 ("a live pending record, phase `Open`") has no subject and is
  **dropped**; the item id and slot it would have compared against come from the
  submit packet's own `ItemUse` header.

### A2 — what Maple Life actually creates

**User ruling (2026-08-21):** Maple Life produces a **level 30, first-job**
character. It does not fit the existing `jobIndex`/`subJobIndex` creation
modelling.

The MapleSEA guide the user supplied states the same behaviour and adds the
detail the wire cannot carry:

- Five classes at 1st-job status: Warrior, Magician, Bowman, Thief, Pirate.
- "Basic items and equipment and minimum AP points needed for your chosen 1st
  job"; all remaining AP and SP left unspent.
- Warrior and Magician only: an option to spend SP on Improved Max HP / Max MP
  Increase before creation.
- Every Maple Life character also receives a fixed package — basic equipment and
  weapon, 100 White Potion, 100 Mana Elixir, 100,000 mesos, The Relaxer.
- Gender is not player-selectable in the dialog.

The client corroborates the class count and the SP gate directly. gms_v95
`CUICharacterSaleDlg::OnButtonClicked` (`0x77edc0`):

```c
case 0x3ECu: this->m_nCurrentClass = (this->m_nCurrentClass + 1) % 5;   // 0..4
case 0x3EFu: this->m_nSP          = (this->m_nSP + 1) % 11;             // 0..10
case 0x3E8u: if (this->m_nCurrentStep == 4 && this->m_nCurrentClass >= 2u)
                 this->m_nCurrentStep = 5;      // SP step skipped for class >= 2
```

The step-skip for `m_nCurrentClass >= 2` is the client's own encoding of "only
Warrior and Magician get the SP choice", which pins **ordinal 0 = Warrior,
1 = Magician**. The struct layout agrees: `CUICharacterSaleDlg` carries
`strSPWarrior[11]`, `strSPMagician[11]`, `m_strStep4Text_Warrior` and
`m_strStep4Text_Magician`, and `LoadSPInfo` (`0x776d00`) fills
`strSPWarrior[0..10]` from consecutive StringPool ids `0x13AE..0x13B8` before
starting `strSPMagician`.

### A3 — the four `SelectedAL` values, derived

`CUICharacterSaleDlg::GetSelectedAL(AvatarLook&)` (gms_v95 `0x778d80`), in full:

```c
al->nFace          = GetSelectedAL(this, 0);
al->anHairEquip[0] = GetSelectedAL(this, 2) + 10 * (GetSelectedAL(this, 1) / 10);
al->nSkin          = GetSelectedAL(this, 3);
```

`GetSelectedAL(this, i)` returns `ASITEM.nItemId` from element 0 of a
per-gender `ZArray<ASITEM>[i]`, so the wire carries **values, not indices** —
Q2 of `open-selected-al-mapping.md` is moot, not unanswered. The arrays are
built by `LoadNewCharInfo` (`0x777790`) from a per-gender WZ property, and its
loop is `for (nType = 0; nType < 4; ++nType)` — **the dialog offers exactly four
choices and no equipment choice at all.**

| Wire field | Slot | Domain | How it was closed |
|---|---|---|---|
| `al0` | face | full template id (`2000x`) | `GetNewCharItemName` (`0x778980`) resolves type 0 through `CItemInfo::GetItemName(nItemId)` — a real item id — while types 1..3 take the WZ-path-format branch |
| `al1` | hair **style** | full hair id, style component (`3xxx0`) | the client itself normalises with `10 * (al1 / 10)`; server passes `(al1 / 10) * 10` |
| `al2` | hair **colour** | bare digit `0..9` | `anHairEquip[0]` must hold a real `3xxxx` hair equip id, and `10 * (al1/10)` already supplies the full style id, so any `al2 > 9` overflows past a valid id; server passes `al2 % 10` |
| `al3` | skin | bare ordinal | **closed this pass** — `al3` lands in `AvatarLook.nSkin`, and `AvatarLook::Decode` (`0x4f2c00`) reads `this->nSkin = CInPacket::Decode1(iPacket)`: a byte-domain ordinal, the same quantity Atlas already sends as `skinColor`. Not an item id |

`al3`'s closure is the same destination-field argument that closed `al2`: what
the field must hold, read out of the client's own code, is evidence. The earlier
"do not close `al3` by analogy with `al2`" note in
`open-selected-al-mapping-round2.md` is satisfied — this is not the analogy, it
is `nSkin`'s own decoder.

### A4 — `nSP` is a skill level, not a stat

`m_nSP` cycles `% 11`, giving `0..10`, and `LoadSPInfo` loads exactly eleven
description strings per class family. Combined with the guide, `nSP` is **the
level the player chose for Improved Max HP Increase (Warrior) or Improved Max MP
Increase (Magician)**, offered only for class ordinals 0 and 1.

**Ruling: honour it.** On creation, for class 0/1 only, the new character
receives that skill at level `nSP`, and `nSP` is deducted from its SP pool; all
remaining AP and SP stay unspent. For class ordinals ≥ 2 the field is decoded,
required to be `0`, and otherwise logged and clamped to `0` — the client never
offers the step there.

`selected-al-derivation.md`'s "`nSP` has no destination in `SeedCharacter`'s
18-argument signature" remains true and is now the *reason* the seed contract
changes rather than a reason to discard the field.

### A5 — the seed contract: a Maple Life creation path, not `POST characters/seed`

`POST characters/seed` cannot express this feature, on three independent counts:

1. **Job.** `buildCharacterCreationSaga` sets
   `JobId: job2.JobFromIndex(input.JobIndex, input.SubJobIndex)`
   (`services/atlas-character-factory/.../factory/processor.go`), and
   `JobFromIndex` yields only Noblesse / Beginner / Legend / Evan
   (`.../job/model.go`). There is no input that names Warrior-at-first-job.
2. **Level.** `RestModel` carries `Level`, but the channel's client
   (`services/atlas-channel/.../character/factory/processor.go:36-40`) does not
   pass it, and no creation template describes a level-30 starting state.
3. **Equipment and items.** `Create` validates `Top`/`Bottom`/`Shoes`/`Weapon`
   against a creation template found by `(jobIndex, subJobIndex, gender)`. The
   Maple Life dialog sends none of those, and no such template exists for a
   level-30 first-job character. The placeholder currently landed in
   `seedCharacterFunc` sends `0` for all four and would be rejected on every
   creation.

**Chosen design (user ruling): a new tenant configuration block plus a Maple
Life creation path in `atlas-character-factory`.** The channel forwards only what
the player chose; the factory owns what a Maple Life character *is*.

Tenant configuration gains a `mapleLife` block: an **ordered five-entry class
table** (index = the client's `nCurrentClass` ordinal), each entry naming

- the `jobId` to create at (first job of that class family),
- `level`,
- the stat allotment — the minimum AP the job requires, with the remainder left
  unspent,
- basic equipment: top, bottom, shoes, weapon template ids,
- the item/meso package (White Potion ×100, Mana Elixir ×100, 100,000 mesos, The
  Relaxer — values are tenant data, not constants in code),
- the starting map,
- optionally, the SP skill id offered at creation (Improved Max HP / MP
  Increase); absent means the class offers no SP step.

The factory's Maple Life path validates the four look values against the same
per-gender WZ-derived option lists the client draws from (face, hair style, hair
colour, skin), resolves the class ordinal through the table, rejects an ordinal
with no entry, and builds a saga that creates the character at the configured
job and level, awards the package, equips the basic equipment, and — for class
0/1 with `nSP > 0` — creates the configured skill at level `nSP`.

**This supersedes §5.3.** The factory remains the single validator, but it is no
longer the *creation-template* validator: §5.3's table of eleven template rules
does not apply to this path. The channel still does not re-implement validation;
it maps the factory's synchronous rejection exactly as §5.3's status table
describes, and that table stands.

**Rejected alternatives.**

- *Extend `characters/seed` with optional `jobId` + `level` + explicit
  equipment.* One endpoint, smaller diff — but it loosens a contract
  `atlas-login` also calls, and it pushes the per-class data (equipment, AP,
  package) into `atlas-channel`, which is the wrong owner.
- *Model the five class families as five more creation templates under a new
  `jobIndex` range.* No new factory code, but it overloads `jobIndex` with a
  second meaning and still cannot express level 30 or the SP skill.

### A6 — the class ordinal table

Derived: **0 = Warrior, 1 = Magician** (A2). The order of 2/3/4 within
{Bowman, Thief, Pirate} is **not** derived, and the user-supplied guide
contradicts itself on it — its section (a) lists "Warrior, Magician, Thief,
Bowman and Pirate", its point 5 lists "Warrior, Magician, Bowman, Thief and
Pirate".

**Ruling: the table is ordered tenant configuration**, so the ordinal→job
mapping is data, not code. Ship with the order flagged in seed data as
unconfirmed, and pin it before live testing by either continuing the IDA pass
into `CUICharacterSaleDlg::OnCreate` (`0x77adc0`) to resolve the
`m_strClassName[5]` / `m_apCanvasClass` sources, or by reading the received
ordinal from channel logs while picking each class in a real client. A wrong
order is then a seed-data fix, not a code change.

### A7 — revised submit-time gates (replaces §5.2)

All run in the `Cash/0543` arm before the factory call; none consumes the item;
each maps to a distinct `MAPLELIFE_ERROR` code.

1. **Ownership** (FR-5.3) — `cashItemInSlotFunc(l, ctx, s.CharacterId(),
   int16(source))` still returns `itemId`, and that item classifies as 543.
   Sourced from the submit packet's own header, not from a pending record. The
   common `ItemUse` prefix already runs this upstream at
   `character_cash_item_use.go:61-66`.
2. **Slot limit** (FR-4.4) — unchanged from §5.2 step 3.
3. **Name re-check** (FR-4.5) — unchanged from §5.2 step 4; still the only
   duplicate gate.
4. **Account / world from session** (FR-4.2) — unchanged from §5.2 step 5.
5. **Class ordinal and SP** — `nCurrentClass` must resolve to a configured class
   entry; `nSP` must be `0..10`, and `0` for ordinals with no configured SP
   skill. A packet failing either is rejected, not clamped-and-created.

On success the arm records a `PhaseSubmitted` registry entry carrying the
returned `transactionId`. §4.3's correlation and §5.4's create-then-consume are
unaffected.

### A8 — impact on what has already landed

- Tasks 1–10 stand: codecs, evidence records, template routing, the registry's
  `Submit`/`Take`/`TakeByTransactionId` surface, the factory REST client, and
  the `transactionId` on the seed status envelope.
- Task 11's `beginMapleLife` open flow is withdrawn (already replaced by
  `50c79fadf`); the registry's open-phase machinery must now be deleted.
- Task 13's `seedCharacterFunc` placeholder — `al0..al3` positional,
  `nCurrentClass` as `jobIndex`, zeros elsewhere — is **wrong in kind on three
  counts** and must be replaced wholesale by the A5 contract. Its own doc
  comment already names it unverified; no test asserts it, because
  `seedCharacterFunc` is swapped in every test. That absence of coverage is
  itself a finding: the new path needs a test that asserts the request body the
  channel sends, not just that a seam was called.
- Task 14's `CREATED`/`FAILED` consumer contract is unaffected.

### A9 — open items after this amendment

- **Class ordinals 2/3/4** — see A6. Config-ordered; pin before live testing.
- **The WZ property backing `LoadNewCharInfo`** — the two per-gender resource
  paths are fetched via `StringPool` ids 1525 / 1526 and were not resolved to
  literal WZ paths this pass. It matters only for building the factory's look
  option lists; the paths must be read from local WZ data at implementation
  time rather than guessed.
- **Gender** — the guide says gender is not player-selectable, yet gms_v95
  `OnButtonClicked` toggles `m_nGender` on control index 4 and the wire carries
  it. The channel takes the packet's value; whether to override it from the
  account's existing characters is not decided and is not blocking.
