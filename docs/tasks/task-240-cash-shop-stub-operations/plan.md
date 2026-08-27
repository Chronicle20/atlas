# Cash Shop Stub Operations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace every log-and-return arm in `CashShopOperationHandleFunc` with a real implementation that mutates durable state and answers the client on its own result arm, and route `BUY_OTHER_PACKAGE`, which has no dispatch arm today.

**Architecture:** Two skeletons, per design §1.1. Mutating arms (GIFT, BUY_NORMAL, REBATE, COUPLE, FRIENDSHIP, BUY_PACKAGE, BUY_OTHER_PACKAGE, ENABLE_EQUIP_SLOT) decode at the channel, gate on the secondary credential, mint a `transactionId`, and issue a Kafka command to `atlas-cashshop`, which does the whole mutation in one DB transaction and emits a status event the channel consumer turns into a result packet. Read-only arms (APPLY_WISHLIST, GET_PURCHASE_RECORD) answer from a REST read with no Kafka round-trip. Two new domains land: a cash-package catalogue in `atlas-data` (ingested from `Etc.wz/CashPackage.img`) and a ring-pairing domain inside `atlas-cashshop` (co-located with the locker assets so a pair and its two assets commit atomically).

**Tech Stack:** Go 1.24 microservices, GORM + Postgres (sqlite in tests), Kafka via `libs/atlas-kafka`, JSON:API REST via `libs/atlas-rest`, packet codecs in `libs/atlas-packet`, config-resolved mode bytes from `atlas-configurations` seed templates.

**Spec:** [design.md](./design.md) (PRD at [prd.md](./prd.md))

## Global Constraints

- **No invented values.** No opcode, mode byte, field order, slot index, or item-template offset may be written from memory. Every wire-level fact comes from the GMS v95.1 IDB (per `docs/reverse-engineering.md`) or from an existing checked-in codec/doc comment. Where a derivation comes back inconclusive, the arm lands its verifiable half plus a typed failure and the gap is recorded in `derivation.md` — never a stub, never a guess.
- **No stubs.** No `// TODO`, no unimplemented handler, no hard-coded success response. CLAUDE.md "Never do this".
- **Mode bytes are config-resolved.** Every new serverbound dispatch goes through `isCashShopOperation(l)(readerOptions, op, <const>)`; every clientbound arm goes through an existing `cashcb.CashShop*Body` constructor which resolves its mode via `atlas_packet.ResolveCode`/`WithResolvedCode`. Never compare a literal byte.
- **Version-divergent packet fields use the `MajorAtLeast` idiom**, never a raw `> N` comparison.
- **Every new entity carries `TenantId`** and every query filters on it. Tenant comes from `tenant.MustFromContext(ctx)`.
- **Every new Kafka command is replay-safe.** The idempotency key is the `TransactionId` minted per click at the channel; enforcement is a unique-constraint insert as the first statement inside the transaction (Task 10).
- **Constants:** check `libs/atlas-constants/` before defining any new domain type, alias, or numeric constant. (Already checked for this task: `item.ClassificationRing = Classification(111)` exists at `libs/atlas-constants/item/constants.go:24`; there is no ring-pair or equip-slot-extension constant anywhere in the library.)
- **Test setup uses the project's Builder pattern.** No `*_testhelpers.go` files.
- **Module-local verification only per task:** `go build ./... && go test ./...` from the task's module root. Repo-wide `tools/verify.sh` runs once at the end (Task 24), not per task.

---

## Task ordering and dependencies

| Task | Deliverable | Depends on |
|---|---|---|
| 1 | `derivation.md` — v95 `CStage::OnSetCashShop` opcode, APPLY_WISHLIST payload + response arm, BUY_OTHER_PACKAGE body, `option`/`oneADay` disposition | — |
| 2 | `gms_95` CashShopOpen registration | 1 |
| 3 | `ErrorEventBody.Operation` + channel failure-routing switch | — |
| 4 | Secondary-credential gate | — |
| 5 | Purchase-record domain + write inside the purchase transaction | — |
| 6 | Purchase-record backfill from `cash_assets` | 5 |
| 7 | GET_PURCHASE_RECORD arm | 5 |
| 8 | APPLY_WISHLIST arm | 1 |
| 9 | BUY_NORMAL arm | 3 |
| 10 | Shared command ledger (idempotency) | — |
| 11 | REBATE_LOCKER_ITEM — `atlas-cashshop` side | 3, 10 |
| 12 | REBATE_LOCKER_ITEM — `atlas-channel` side | 4, 11 |
| 13 | GIFT — `atlas-cashshop` side (incl. `cash_assets` gift columns) | 3, 5, 10 |
| 14 | GIFT — `atlas-channel` side | 4, 13 |
| 15 | `atlas-data` cash-package catalogue | — |
| 16 | `atlas-cashshop` package-purchase side | 5, 10, 15 |
| 17 | BUY_PACKAGE / BUY_OTHER_PACKAGE — `atlas-channel` side | 1, 16 |
| 18 | Ring domain in `atlas-cashshop` | — |
| 19 | Ring purchase transaction + command | 5, 10, 18 |
| 20 | BUY_COUPLE / BUY_FRIENDSHIP — `atlas-channel` side | 4, 19 |
| 21 | `derivation-equip-slot.md` — OQ-E1/OQ-E2 | — |
| 22 | `atlas-character` equip-slot-extension domain | 21 |
| 23 | ENABLE_EQUIP_SLOT end to end | 3, 10, 22 |
| 24 | Coverage manifest, packet-audit promotion, repo-wide verification | all |

---

## Task 1: Derive the four blocking client facts

Produces the evidence every derivation-blocked task downstream cites. Nothing here changes code.

### Files

- `docs/tasks/task-240-cash-shop-stub-operations/derivation.md` — **new file**; the deliverable
- `docs/reverse-engineering.md` — read-only; `func_query` usage and `idb_list` session resolution
- `docs/packets/PROCESS.md` — read-only; entry point for packet/protocol work
- `docs/tasks/task-227-cash-name-change-world-transfer/derivation.md` — read-only; the format to copy
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — read-only; carries the v92 registration `{"opCode":"0x8E","writer":"CashShopOpen","fname":"CStage::OnSetCashShop","services":["channel"]}` that must **not** be copied forward
- `libs/atlas-packet/cash/clientbound/shop_operation_body.go` — read-only; the existing result-arm constructors and their mode-name constants

Module root: none (documentation only).

- [ ] **Step 1: Open the GMS v95.1 IDB**

Follow `docs/reverse-engineering.md`: resolve the session with `idb_list`, then `idb_open` the GMS v95.1 database. Record the IDB identity (path + hash as reported) at the top of `derivation.md` so a later reader can tell whether the answers are still pinned to the same binary.

- [ ] **Step 2: Derive D1 — the v95 opcode for `CStage::OnSetCashShop`**

Locate `CStage::OnSetCashShop` by name (`lookup_funcs` / `find`), then find the opcode that dispatches to it in the client's clientbound handler table. Record: the function address, the dispatch site, and the opcode value in hex.

If the function does not exist in the v95 binary, or exists but is unreachable from the dispatch table, that is the `n-a` answer (FR-V95-3) — record the evidence for the negative, not just the absence of a positive.

- [ ] **Step 3: Derive D2 — the APPLY_WISHLIST (mode 35 on v95) serverbound body and its response arm**

Find the `CCashShop` sender for serverbound mode 35 (the v83 mode is 33; resolve the v95 mode from the tenant `operations` table, already `"APPLY_WISHLIST": 35` in `template_gms_95_1.json`). Answer two questions:

- **D2a:** does the client write any bytes after the mode byte? If yes, record the exact field order and widths.
- **D2b:** which `CCashShop::OnCashItemResult` arm does the client expect in reply? The two candidates already exist in the repo: `LOAD_WISHLIST` (mode 92 on gms_95, built by `cashcb.CashShopWishListLoadBody(sns []uint32)`) and `UPDATE_WISHLIST` (mode 98, built by `cashcb.CashShopWishListUpdateBody(sns []uint32)`). Record which one, with the client-side evidence.

- [ ] **Step 4: Derive D3 — the BUY_OTHER_PACKAGE (mode 33 on v95) serverbound body**

Find the sender for serverbound mode 33. Answer:

- **D3a:** is the body byte-identical to `ShopOperationBuyPackage` (`pointType bool`, `option uint32`, `serialNumber uint32` — see `libs/atlas-packet/cash/serverbound/shop_operation_buy_package.go:61`), or does it carry additional fields (a recipient name, a message)?
- **D3b:** does the client expect `GIFT_PACKAGE_SUCCESS` (mode 156) / `GIFT_PACKAGE_FAILED` (157) in reply, rather than `BUY_PACKAGE_SUCCESS` (154) / `BUY_PACKAGE_FAILED` (155)? Design §0 records this as a strong hypothesis from `CashShopGiftPackageDoneBody(recipientName string, packageId int32, unused1, unused2 uint16, nxCashSpent int32)`; confirm or refute it against the client.

- [ ] **Step 5: Derive D4 — `option` and `oneADay`**

- **D4a:** `option uint32` on `ShopOperationBuyPackage`, `ShopOperationBuyCouple`, `ShopOperationBuyFriendship`. What does the client put there, and does the server consume it? Design §13 OQ-O1 requires it be *proven* ignorable, not assumed.
- **D4b:** `oneADay byte` on `ShopOperationGift` (`m_bRequestBuyOneADay`). Server-enforced per-day gift limit, or client-side UI flag?

- [ ] **Step 6: Write `derivation.md`**

One section per answer, each with: the question, the answer, the address/decompilation excerpt that proves it, and the task number that consumes it. Any answer that could not be established is written as **UNRESOLVED** with what was tried and what the consuming task must do instead (design §14: land the verifiable half plus a typed failure). Do not write a plausible value for an unresolved question.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-240-cash-shop-stub-operations/derivation.md
git commit -m "docs(task-240): derive v95 cash shop open opcode, wishlist/other-package bodies, option semantics"
```

---

## Task 2: Register `CashShopOpen` for `gms_95`

Without this, `CashShopEntryHandleFunc` cannot announce the cash shop open packet on a v95 tenant, and every arm in this plan is unreachable there.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — add the `CashShopOpen` writer registration
- `docs/tasks/task-240-cash-shop-stub-operations/derivation.md` — read-only (new file, written by Task 1); D1 supplies the opcode
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — read-only; the shape of the registration, **not** the opcode value
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go` — read-only; the announce site (`:67`) that consumes the writer

Module root: none for the template edit; `services/atlas-channel/atlas.com/channel` if a test is added there.

- [ ] **Step 1: Confirm the current gap**

Run:

```sh
grep -o CashShopOpen services/atlas-configurations/seed-data/templates/template_gms_92_1.json
python3 -c "print(open('services/atlas-configurations/seed-data/templates/template_gms_95_1.json').read().count('CashShopOpen'))"
```

Expected: one `CashShopOpen` line from gms_92, then `0` from gms_95 — the gap this task closes.

- [ ] **Step 2: Apply D1**

If D1 gave an opcode, add to the writer list in `template_gms_95_1.json`, using the derived opcode and the same `fname`/`services`:

```json
{"opCode":"<derived hex from derivation.md D1>","writer":"CashShopOpen","fname":"CStage::OnSetCashShop","services":["channel"]}
```

If D1 established that v95 does not carry the packet, skip the registration and instead record the `n-a` proof in the coverage matrix (Task 24 folds this in) — and say so explicitly in `context.md`.

- [ ] **Step 3: Verify the template still parses and the writer resolves**

Run:

```sh
python3 -c "import json;json.load(open('services/atlas-configurations/seed-data/templates/template_gms_95_1.json'));print('ok')"
python3 -c "print(open('services/atlas-configurations/seed-data/templates/template_gms_95_1.json').read().count('CashShopOpen'))"
```

Expected: `ok`, then `1`.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "fix(configurations): register CashShopOpen writer for gms_95"
```

**Live validation (FR-V95-4)** — a v95 tenant opening the cash shop — is not a template diff and cannot be asserted here; it is recorded as an open validation item for Task 24 / post-PR testing.

---

## Task 3: Failure routing — `ErrorEventBody.Operation`

Today `handleStatusEventError` unconditionally announces `CashShopInventoryCapacityIncreaseFailedBody`, which is the wrong mode byte for eight of the ten arms in this plan. This task adds the discriminator, backward compatibly.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` — add `Operation string` to `ErrorEventBody`
- `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go` — add `ErrorStatusEventForOperationProvider`
- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` — mirror the field
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — switch on `Operation` in `handleStatusEventError` (`:290`)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_failure_routing_test.go` — **new file**; the tests
- `libs/atlas-packet/cash/clientbound/shop_operation_body.go` — read-only; the `*FailedBody` constructors this switch selects between

Module roots: `services/atlas-cashshop/atlas.com/cashshop` and `services/atlas-channel/atlas.com/channel`.

**Interfaces produced (later tasks rely on these exact names):**

```go
// kafka/message/cashshop/kafka.go — the operation discriminator values
const (
	ErrorOperationGift            = "GIFT"
	ErrorOperationBuyNormal       = "BUY_NORMAL"
	ErrorOperationRebate          = "REBATE"
	ErrorOperationCouple          = "COUPLE"
	ErrorOperationFriendship      = "FRIENDSHIP"
	ErrorOperationBuyPackage      = "BUY_PACKAGE"
	ErrorOperationGiftPackage     = "GIFT_PACKAGE"
	ErrorOperationEnableEquipSlot = "ENABLE_EQUIP_SLOT"
)

// kafka/producer/cashshop/producer.go (atlas-cashshop)
func ErrorStatusEventForOperationProvider(characterId uint32, operation string, error string, transactionId uuid.UUID) model.Provider[[]kafka.Message]
```

- [ ] **Step 1: Write the failing tests**

`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_failure_routing_test.go`.

The routing decision is extracted into a pure function so it is testable without a live writer registry or session — same shape as `worldTransferRejectionReason` in `socket/handler/cash_shop_operation_reason_test.go:22`.

`TestFailureBodyForOperation` — table-driven. Each case calls `failureBodyForOperation(op, reason)` and encodes the returned body against a fixed options map, asserting the **mode byte** it produced. Options map carries the gms_95 values verbatim (from `template_gms_95_1.json`):

```go
options := map[string]interface{}{
	"operations": map[string]interface{}{
		cashcb.CashShopOperationInventoryCapacityIncreaseFailed: float64(110),
		cashcb.CashShopOperationGiftFailed:                      float64(108),
		cashcb.CashShopOperationBuyNormalFailed:                 float64(159),
		cashcb.CashShopOperationRebateFailed:                    float64(151),
		cashcb.CashShopOperationCoupleFailed:                    float64(153),
		cashcb.CashShopOperationFriendshipFailed:                float64(163),
		cashcb.CashShopOperationBuyPackageFailed:                float64(155),
		cashcb.CashShopOperationGiftPackageFailed:               float64(157),
		cashcb.CashShopOperationEnableEquipSlotExtFailed:        float64(118),
	},
	"errors": map[string]interface{}{
		"NOT_ENOUGH_CASH": float64(3),
		"INVENTORY_FULL":  float64(25),
		"unknown_error":   float64(69),
	},
}
```

Resolve each `cashcb.CashShopOperation*Failed` constant's real name from `libs/atlas-packet/cash/clientbound/shop_operation_body.go` before writing the test — do not assume the identifier spelling.

| subtest | `operation` | `reason` | expect mode byte | expect error byte |
|---|---|---|---|---|
| `empty operation keeps today's arm` | `""` | `"INVENTORY_FULL"` | 110 | 25 |
| `unknown operation keeps today's arm` | `"SOMETHING_ELSE"` | `"INVENTORY_FULL"` | 110 | 25 |
| `gift` | `"GIFT"` | `"NOT_ENOUGH_CASH"` | 108 | 3 |
| `buy normal` | `"BUY_NORMAL"` | `"NOT_ENOUGH_CASH"` | 159 | 3 |
| `rebate` | `"REBATE"` | `"unknown_error"` | 151 | 69 |
| `couple` | `"COUPLE"` | `"NOT_ENOUGH_CASH"` | 153 | 3 |
| `friendship` | `"FRIENDSHIP"` | `"NOT_ENOUGH_CASH"` | 163 | 3 |
| `buy package` | `"BUY_PACKAGE"` | `"INVENTORY_FULL"` | 155 | 25 |
| `gift package` | `"GIFT_PACKAGE"` | `"INVENTORY_FULL"` | 157 | 25 |
| `enable equip slot` | `"ENABLE_EQUIP_SLOT"` | `"NOT_ENOUGH_CASH"` | 118 | 3 |

Every `*FailedBody` in this family encodes as exactly two bytes (mode, errorCode) — assert `len(body) == 2` first, as `cash_shop_operation_reason_test.go:52` does.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/cashshop/... -run TestFailureBodyForOperation -v
```

Expected: compile failure — `undefined: failureBodyForOperation`.

- [ ] **Step 3: Add `Operation` to both `ErrorEventBody` copies**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` and its channel mirror, add:

```go
// Operation names the cash shop arm this failure belongs to, so the channel
// can answer on that arm's own *_FAILED mode byte. Empty means "the legacy
// capacity-increase arm" -- every producer that predates this field leaves it
// empty and keeps its existing behavior byte for byte.
Operation string `json:"operation,omitempty"`
```

Add the `ErrorOperation*` constants (listed under **Interfaces produced** above) to the atlas-cashshop copy and mirror them into the channel copy.

- [ ] **Step 4: Add the producer variant**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go`, add `ErrorStatusEventForOperationProvider` alongside the existing `ErrorStatusEventProvider` (`:13`), copying its message shape and setting `Operation`. Leave `ErrorStatusEventProvider` untouched — existing callers keep the empty-operation behavior.

- [ ] **Step 5: Implement the channel switch**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`, add `failureBodyForOperation(operation string, reason string)` returning the body-builder func type the `session.Announce` chain takes, with a `switch operation` whose `default` returns `cashpkt.CashShopInventoryCapacityIncreaseFailedBody(reason)` — today's exact behavior. Call it from `handleStatusEventError` in place of the current unconditional builder, **after** the existing pending-change branch (which must keep priority: a name-change/world-transfer failure still answers on its own arm).

- [ ] **Step 6: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./kafka/consumer/cashshop/... -v
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./kafka/... -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/kafka services/atlas-channel/atlas.com/channel/kafka
git commit -m "feat(cashshop): route cash shop failures to their own result arm via ErrorEventBody.Operation"
```

---

## Task 4: The secondary-credential gate

Six arms carry the client's `ask_SPW` slot (GIFT, BUY_COUPLE, BUY_FRIENDSHIP, REBATE_LOCKER_ITEM, and on legacy v48 BUY_NORMAL). None of them look at it today.

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential.go` — **new file**; the gate
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential_test.go` — **new file**; the tests
- `services/atlas-channel/atlas.com/channel/account/processor.go` — read-only; `GetById` (`:54`), `RecordPicAttempt` (`:85`)
- `services/atlas-channel/atlas.com/channel/account/model.go` — read-only; `PIC()`, `BirthDate()` (`:37`), and the builder used to construct test accounts
- `services/atlas-login/atlas.com/login/socket/handler/character_selected_pic.go` — read-only; the plaintext-comparison precedent at `:49`

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces produced:**

```go
// ErrCredentialMismatch is the sentinel every gated arm maps to its own
// *_FAILED body with the errors-table key "INVALID_BIRTHDAY".
var ErrCredentialMismatch = errors.New("secondary credential mismatch")

// credentialMatches is the pure decision the gate makes once the account is
// resolved. usesPIC is true when the tenant is GMS major >= 95.
func credentialMatches(usesPIC bool, storedPIC string, storedBirthDate uint32, spw string, birthday uint32) bool

// verifySecondaryCredential resolves the session's account, applies
// credentialMatches, records the attempt, and returns ErrCredentialMismatch
// on failure.
func verifySecondaryCredential(l logrus.FieldLogger, ctx context.Context) func(s session.Model, spw string, birthday uint32) error
```

- [ ] **Step 1: Write the failing test**

`cash_shop_credential_test.go`. `TestCredentialMatches` — table-driven over the pure function, no session, no processor:

| subtest | usesPIC | storedPIC | storedBirthDate | spw | birthday | expect |
|---|---|---|---|---|---|---|
| `pic matches` | true | `"5678"` | 19940203 | `"5678"` | 0 | true |
| `pic mismatches` | true | `"5678"` | 19940203 | `"1234"` | 0 | false |
| `pic empty passes` | true | `""` | 19940203 | `"1234"` | 0 | true |
| `pic empty and empty spw passes` | true | `""` | 0 | `""` | 0 | true |
| `birthday matches` | false | `"5678"` | 19940203 | `""` | 19940203 | true |
| `birthday mismatches` | false | `"5678"` | 19940203 | `""` | 19700101 | false |
| `birthday unset passes` | false | `"5678"` | 0 | `""` | 19700101 | true |
| `pre-95 ignores pic entirely` | false | `"5678"` | 19940203 | `"wrong"` | 19940203 | true |

The last case is the one that matters: on a pre-95 tenant the client sends `birthday` in the same wire slot, so `spw` is meaningless there and must not be consulted.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run TestCredentialMatches -v
```

Expected: compile failure — `undefined: credentialMatches`.

- [ ] **Step 3: Implement `credentialMatches`**

```go
// credentialMatches decides the gate. It PASSES when the account has no
// credential of the applicable kind set: a server that never collected the
// value cannot meaningfully check it, and failing closed would make every
// gift, ring, and rebate unusable on a fresh tenant (design section 2 step 4).
func credentialMatches(usesPIC bool, storedPIC string, storedBirthDate uint32, spw string, birthday uint32) bool {
	if usesPIC {
		if storedPIC == "" {
			return true
		}
		return storedPIC == spw
	}
	if storedBirthDate == 0 {
		return true
	}
	return storedBirthDate == birthday
}
```

- [ ] **Step 4: Implement `verifySecondaryCredential`**

Resolve the account with `account.NewProcessor(l, ctx).GetById(s.AccountId())`. Decide `usesPIC` from the tenant with the `MajorAtLeast` idiom against major 95 — grep `MajorAtLeast` in `libs/atlas-packet/cash/serverbound/shop_operation_gift.go` for the exact call shape used against a tenant in this repo, and use that shape; do not write a `>` comparison. On mismatch, call `account.NewProcessor(l, ctx).RecordPicAttempt(s.AccountId(), false, s.IP(), "")` (resolve the session's IP accessor name from `session.Model` before writing this) and return `ErrCredentialMismatch`; on a pass where the credential was unset, log at debug that the gate was inert. An account-lookup error returns that error (the arm maps it to `unknown_error`), never a silent pass.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential.go services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential_test.go
git commit -m "feat(channel): add the cash shop secondary-credential gate"
```

---

## Task 5: Purchase records

`atlas-cashshop` persists no purchase history today: `cash_assets` is soft-deleted on withdrawal to a character inventory and on rebate, so "did this account ever buy SN X" cannot be answered from live locker contents. GET_PURCHASE_RECORD (Task 7) needs a durable record.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/entity.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator.go` — **new file**; the upsert
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/processor.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator_test.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go` — record the purchase inside `Purchase`'s transaction (`:100`)
- `services/atlas-cashshop/atlas.com/cashshop/main.go` — add `purchaserecord.Migration` to the `database.SetMigrations` list (`:62`)

Patterns to copy: `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/entity.go` (tenant-scoped ledger entity + `Migration`), `services/atlas-cashshop/atlas.com/cashshop/wishlist/administrator.go` (administrator shape).

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**Interfaces produced:**

```go
// purchaserecord
func Migration(db *gorm.DB) error

// Record upserts one purchase. It is called INSIDE the purchase transaction,
// so it takes the tx handle rather than the processor's own db.
func Record(db *gorm.DB, tenantId uuid.UUID, accountId uint32, serialNumber uint32) error

// Get answers "has this account ever bought this serial number", and how many
// times. A miss is (0, nil) -- not an error.
func Get(db *gorm.DB, tenantId uuid.UUID, accountId uint32, serialNumber uint32) (uint32, error)
```

- [ ] **Step 1: Write the failing test**

`administrator_test.go`. Open an in-memory sqlite DB and `Migration` it — copy the DB-setup shape from `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/administrator_test.go`.

`TestRecordUpsertsAndCounts` — sequential, one `t.Run` per step against the same DB, tenant `A = uuid.New()`, account `42`, serial `10000`:

| subtest | action | expect |
|---|---|---|
| `first purchase creates` | `Record(db, A, 42, 10000)` then `Get(db, A, 42, 10000)` | count `1`, no error |
| `second purchase increments` | `Record(db, A, 42, 10000)` then `Get` | count `2` |
| `different serial is separate` | `Record(db, A, 42, 20000)` then `Get(db, A, 42, 20000)` | count `1`; `Get(db, A, 42, 10000)` still `2` |
| `different account is separate` | `Get(db, A, 99, 10000)` | count `0`, no error |
| `different tenant is separate` | `Get(db, uuid.New(), 42, 10000)` | count `0`, no error |
| `miss is not an error` | `Get(db, A, 42, 30000)` | count `0`, error `nil` |

The tenant-isolation cases are not optional — they are what proves the unique index is on `(TenantId, AccountId, SerialNumber)` and not on the last two alone.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./purchaserecord/... -v
```

Expected: build failure — package does not exist.

- [ ] **Step 3: Write the entity**

```go
// entity is the durable answer to "has this account ever bought serial X".
// It exists because cash_assets is soft-deleted on withdrawal and on rebate,
// so live locker contents cannot answer the question -- and FR-REC-2 is
// explicit that a consumed or discarded item still counts as purchased.
// A rebate does NOT remove a record: "purchased" is a historical fact.
type entity struct {
	Id           uuid.UUID `gorm:"primaryKey;not null"`
	TenantId     uuid.UUID `gorm:"not null;uniqueIndex:idx_purchase_record_unique"`
	AccountId    uint32    `gorm:"not null;uniqueIndex:idx_purchase_record_unique"`
	SerialNumber uint32    `gorm:"not null;uniqueIndex:idx_purchase_record_unique"`
	Count        uint32    `gorm:"not null"`
	FirstAt      time.Time `gorm:"not null"`
	LastAt       time.Time `gorm:"not null"`
}

func (e entity) TableName() string { return "cash_purchase_records" }
```

- [ ] **Step 4: Write the administrator**

`Record` is an upsert on the unique index — `Count = Count + 1`, `LastAt = time.Now()` on conflict. Use gorm's `clause.OnConflict` with `Columns` naming the three index columns and `DoUpdates: clause.Assignments(...)`, so two concurrent inserts cannot both land. `Get` selects `Count`, returning `(0, nil)` on `gorm.ErrRecordNotFound`.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./purchaserecord/... -v
```

Expected: PASS.

- [ ] **Step 6: Write the record inside the purchase transaction**

In `cashshop/processor.go`'s `Purchase` closure, immediately after the successful `p.astP.Create`/`CreateWithCashId` and before the `mb.Put(...PurchaseStatusEventProvider...)`, call `purchaserecord.Record(tx, p.t.Id(), c.AccountId(), serialNumber)`. A failure returns the error (rolling the purchase back) with a `rejectEmit` set to `ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", transactionId)` — matching every other in-transaction failure on that path. Add `purchaserecord.Migration` to the migration list in `main.go:62`.

- [ ] **Step 7: Build and run the service tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./... 
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): persist durable purchase records inside the purchase transaction"
```

---

## Task 6: Backfill purchase records from existing locker rows

Accounts that bought before Task 5 landed have no records, so `GET_PURCHASE_RECORD` would answer "never purchased" for everything they own. `cash_assets` still carries `CommodityId` and `PurchasedBy` (including on soft-deleted rows), so most of that history is recoverable.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill_test.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/main.go` — invoke the backfill once at startup, after migrations
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/entity.go` — read-only; `cash_assets` columns (`CommodityId`, `PurchasedBy`, `CreatedAt`, `DeletedAt`)
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/` — read-only; the compartment entity that joins an asset to its owning account

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**Interfaces produced:**

```go
// Backfill seeds cash_purchase_records from existing cash_assets rows, once.
// It is a no-op on a database that already has records, so it is safe to run
// on every boot. Returns the number of records written.
func Backfill(l logrus.FieldLogger, db *gorm.DB) (int, error)
```

- [ ] **Step 1: Write the failing test**

`backfill_test.go`, in-memory sqlite with `Migration` plus the asset and compartment migrations.

`TestBackfill` — cases:

| subtest | fixture | expect |
|---|---|---|
| `seeds from live assets` | one compartment for account 42, two assets with `CommodityId` 10000 and 20000 | `Backfill` returns 2; `Get(db, A, 42, 10000)` == 1, `Get(db, A, 42, 20000)` == 1 |
| `counts duplicates` | three assets with `CommodityId` 10000 for account 42 | `Get(db, A, 42, 10000)` == 3 |
| `includes soft-deleted assets` | one asset with `CommodityId` 30000 and `DeletedAt` set | `Get(db, A, 42, 30000)` == 1 |
| `skips zero commodity id` | one asset with `CommodityId` 0 | no record for serial 0 |
| `is idempotent` | run `Backfill` a second time on the same DB | returns 0; every count above is unchanged |

The idempotency case is the load-bearing one: this runs on every boot, and a backfill that increments on each restart is worse than no backfill.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./purchaserecord/... -run TestBackfill -v
```

Expected: compile failure — `undefined: Backfill`.

- [ ] **Step 3: Implement `Backfill`**

Short-circuit and return `(0, nil)` when `cash_purchase_records` already has at least one row — that is what makes it idempotent and cheap on every subsequent boot. Otherwise select, grouped, `(tenant_id, account_id, commodity_id, count(*), min(created_at), max(created_at))` from `cash_assets` joined to its compartment for the account id, with `Unscoped()` so soft-deleted rows are included and `commodity_id <> 0`, and insert one record per group directly (not via `Record`, so the counts land in one statement each rather than N increments). Log the total written, and log explicitly that assets already hard-deleted are unrecoverable.

- [ ] **Step 4: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./purchaserecord/... -v
```

Expected: PASS.

- [ ] **Step 5: Wire it into startup**

In `main.go`, after `database.Connect(...)`, call `purchaserecord.Backfill(l, db)` and log the count. A backfill error is logged as a warning and does not prevent the service from starting — a missing history is a degraded answer, not a reason to refuse to serve the cash shop.

- [ ] **Step 6: Build and test**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): backfill purchase records from existing locker assets"
```

---

## Task 7: GET_PURCHASE_RECORD (mode 40/44)

Skeleton B — a pure read, no Kafka round-trip.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/rest.go` — **new file**; the JSON:API model
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/resource.go` — **new file**; the GET route
- `services/atlas-cashshop/atlas.com/cashshop/main.go` — add the route initializer alongside `:126-133`
- `services/atlas-channel/atlas.com/channel/cashshop/purchaserecord/` — **new files** `rest.go`, `requests.go`, `processor.go`; the channel-side client
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` — replace the log-only arm at `:186-191`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_purchase_record_test.go` — **new file**

Patterns to copy: `services/atlas-cashshop/atlas.com/cashshop/wishlist/resource.go` (resource shape), `services/atlas-channel/atlas.com/channel/cashshop/wishlist/` (channel client shape, file-for-file).

Module roots: `services/atlas-cashshop/atlas.com/cashshop`, `services/atlas-channel/atlas.com/channel`.

**Interfaces produced:**

```go
// atlas-cashshop REST: GET /accounts/{accountId}/purchaseRecords/{serialNumber}
// A miss is 200 with Purchased=false, never 404 -- the client needs an answer.
type RestModel struct {
	SerialNumber uint32 `json:"-"`
	Purchased    bool   `json:"purchased"`
	Count        uint32 `json:"count"`
}

// atlas-channel client
func (p *ProcessorImpl) GetForAccount(accountId uint32, serialNumber uint32) (Model, error)
func (m Model) Purchased() bool
func (m Model) Count() uint32
```

- [ ] **Step 1: Write the failing test**

`cash_shop_purchase_record_test.go`. The wire assertion is on the body builder, following `cash_shop_operation_reason_test.go:38`'s shape.

`TestCashShopPurchaseRecordDoneBodyEncodesPurchasedFlag` — table-driven over `cashcb.CashShopPurchaseRecordDoneBody(goodsSN int32, purchased byte)` with gms_95 options `{"operations": {cashcb.CashShopOperationPurchaseRecord: float64(175)}}` (resolve the constant's real identifier from `shop_operation_body.go:640` before writing):

| subtest | goodsSN | purchased | expect first byte | expect remaining |
|---|---|---|---|---|
| `purchased` | 10000 | 1 | 175 | little-endian int32 `10000` then byte `1` |
| `not purchased` | 10000 | 0 | 175 | little-endian int32 `10000` then byte `0` |

Read `NewPurchaseRecordDone`'s `Encode` in `shop_operation_body.go` first and assert the exact byte layout it produces — do not assume the field order from the constructor signature.

`TestPurchaseRecordFlag` — the pure mapper the handler uses, `purchaseRecordFlag(count uint32) byte`: `0 -> 0`, `1 -> 1`, `7 -> 1`.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'TestCashShopPurchaseRecord|TestPurchaseRecordFlag' -v
```

Expected: compile failure — `undefined: purchaseRecordFlag`.

- [ ] **Step 3: Add the atlas-cashshop REST surface**

`rest.go` + `resource.go` reading through `purchaserecord.Get`, mounted in `main.go`. A serial number with no record returns `200` with `Purchased: false, Count: 0`.

- [ ] **Step 4: Add the atlas-channel client**

`cashshop/purchaserecord/{rest,requests,processor}.go`, mirroring `cashshop/wishlist/` file-for-file.

- [ ] **Step 5: Replace the log-only arm**

In `cash_shop_operation.go`, replace the `CashShopOperationGetPurchaseRecord` body: read through the client, announce `cashcb.CashShopPurchaseRecordDoneBody(int32(sp.SerialNumber()), purchaseRecordFlag(m.Count()))` on success; on a read error announce `cashcb.CashShopPurchaseRecordFailedBody("unknown_error")` — never a silent return (FR-X-2).

- [ ] **Step 6: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... -v
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop services/atlas-channel/atlas.com/channel
git commit -m "feat(channel): implement GET_PURCHASE_RECORD against durable purchase history"
```

---

## Task 8: APPLY_WISHLIST (mode 33/35)

The current arm decodes nothing and logs. Skeleton B.

### Files

- `docs/tasks/task-240-cash-shop-stub-operations/derivation.md` — read-only (new file, written by Task 1); **D2a** gives the serverbound body, **D2b** the response arm
- `libs/atlas-packet/cash/serverbound/shop_operation_apply_wishlist.go` — **new file**, *only if D2a found a non-empty body*
- `libs/atlas-packet/cash/serverbound/shop_operation_apply_wishlist_test.go` — **new file**, same condition
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` — replace the log-only arm at `:175-178`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_apply_wishlist_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/cashshop/wishlist/processor.go` — read-only; `GetByCharacterId` (`:15`) is the read

Patterns to copy: `libs/atlas-packet/cash/serverbound/shop_operation_set_wishlist.go` (+ its `_test.go`) if a codec is needed; the existing `SET_WISHLIST` arm at `cash_shop_operation.go:66-88` for the announce shape.

Module roots: `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Read D2 and record the branch taken**

Open `derivation.md`. Write one sentence at the top of the new handler's doc comment naming D2a's answer (payload or no payload) and D2b's answer (which result arm), citing the derivation section. If D2 came back **UNRESOLVED**, stop and implement only the no-payload read-and-answer path using `cashcb.CashShopWishListLoadBody` — the LOAD arm is the read-response sibling of the UPDATE arm `SET_WISHLIST` already uses — and record in `context.md` that the arm choice is unconfirmed.

- [ ] **Step 2: Write the failing test**

`cash_shop_apply_wishlist_test.go`. `TestCashShopWishListBodyEncodesStoredSerials` — table-driven over whichever constructor D2b named, with gms_95 options `{"operations": {<the arm's mode constant>: float64(<92 for LOAD_WISHLIST, 98 for UPDATE_WISHLIST>)}}`:

| subtest | input `sns` | expect |
|---|---|---|
| `full wishlist` | `[]uint32{10000, 20000, 30000}` | first byte = the arm's mode; then the exact payload `NewWishListLoad`/`NewWishListUpdate`'s `Encode` produces for those three values |
| `empty wishlist` | `[]uint32{}` | first byte = the arm's mode; a well-formed body, **not** zero bytes and **not** an error (FR-WISH-3) |

Read the encoder in `libs/atlas-packet/cash/clientbound/shop_operation_body.go` and write the expected byte slice out in full — the wishlist arms pad to a fixed slot count, so the empty case is not an empty payload.

If D2a found a serverbound body, also add `shop_operation_apply_wishlist_test.go` in `libs/atlas-packet/cash/serverbound/` with a byte-fixture round-trip carrying a `packet-audit:verify` marker, following `shop_operation_set_wishlist_test.go` exactly.

- [ ] **Step 3: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run TestCashShopWishList -v
```

Expected: FAIL on the byte assertion (the constructors already exist, so this is a red-then-green on the expectation, not a compile error) — unless a new codec is being added, in which case the `libs/atlas-packet` test fails to compile first.

- [ ] **Step 4: Add the serverbound codec if D2a requires one**

Follow `docs/packets/IMPLEMENTING_A_PACKET.md`: immutable struct, both `Encode` and `Decode`, version gates via `MajorAtLeast`, accessors mirroring `ShopOperationSetWishlist`.

- [ ] **Step 5: Replace the log-only arm**

Decode the body if there is one, read `wishlist.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())`, project to `[]uint32` of serial numbers exactly as the `SET_WISHLIST` arm does at `cash_shop_operation.go:79-82`, and announce D2b's body. A read error announces `cashcb.CashShopWishListLoadFailedBody`/the LOAD_WISH_FAILED arm (resolve its real constructor name in `shop_operation_body.go`); never a silent return.

- [ ] **Step 6: Run the tests**

```sh
cd libs/atlas-packet && go build ./... && go test ./cash/...
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet services/atlas-channel/atlas.com/channel
git commit -m "feat(channel): implement APPLY_WISHLIST"
```

---

## Task 9: BUY_NORMAL (mode 20/34)

The smallest mutating arm — it reuses the existing `RequestPurchase` pipeline unchanged, so it validates Skeleton A end to end with no new command.

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` — replace the log-only arm at `:152-157`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — no change needed; the existing `handleStatusEventPurchase` (`:142`) already answers, but confirm the arm it picks
- `services/atlas-channel/atlas.com/channel/cashshop/processor.go` — read-only; `RequestPurchase` (`:98`) and `resolvePurchaseCurrency` (`:118`)
- `libs/atlas-packet/cash/serverbound/shop_operation_buy_normal.go` — read-only; `SerialNumber()` (`:30`), and `:23-28` documenting that the v83+ body carries nothing else
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_buy_normal_test.go` — **new file**

Module root: `services/atlas-channel/atlas.com/channel`.

**Currency derivation (design §0, PRD FR-BUYN-2):** `ShopOperationBuyNormal` carries no `isPoints`/`currency` — on v83+ the entire body is a 4-byte `serialNumber`. This is the identical situation task-227 resolved for `BUY_NAME_CHANGE`, and the answer is the same: `isPoints=false, currency=0`. Cite `shop_operation_buy_normal.go:23-28` in the handler's doc comment.

- [ ] **Step 1: Write the failing test**

`cash_shop_buy_normal_test.go`. `TestBuyNormalPurchaseCurrency` asserts the derivation is applied rather than invented, over the existing `resolvePurchaseCurrency` in `cashshop/processor.go`:

| subtest | isPoints | currency | expect |
|---|---|---|---|
| `buy normal sends credit` | false | 0 | 0 |
| `points buy with no currency steers to maple points` | true | 0 | 2 |
| `explicit currency passes through` | false | 4 | 4 |

`TestCashShopBuyNormalDoneBodyMode` asserts the success arm the consumer must announce for this op resolves to gms_95's `BUY_NORMAL_SUCCESS`: build `cashcb.CashShopBuyNormalDoneBody(refs)` with a single-element `[]cashcb.PackedCashItemRef` and options `{"operations": {cashcb.CashShopOperationBuyNormalSuccess: float64(158)}}`, assert the first byte is `158`. Read `PackedCashItemRef`'s definition and `NewBuyNormalDone`'s `Encode` in `shop_operation_body.go` and write the full expected byte slice.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./cashshop/... ./socket/handler/... -run 'TestBuyNormal|TestCashShopBuyNormal' -v
```

Expected: FAIL on the byte assertion.

- [ ] **Step 3: Replace the log-only arm**

```go
if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyNormal) {
	sp := &cashsb.ShopOperationBuyNormal{}
	sp.Decode(l, ctx)(r, readerOptions)
	if err = cashshop.NewProcessor(l, ctx).RequestPurchase(s.CharacterId(), sp.SerialNumber(), false, 0, 0, uuid.New()); err != nil {
		l.WithError(err).Errorf("Unable to request BUY_NORMAL purchase for character [%d] serial number [%d].", s.CharacterId(), sp.SerialNumber())
	}
	return
}
```

The `transactionId` is minted here per click (design §8), so a Kafka redelivery replays one id while a genuine second click legitimately charges twice.

- [ ] **Step 4: Route the success and failure arms**

`handleStatusEventPurchase` currently announces `CashShopCashInventoryPurchaseSuccessBody` for any purchase without a pending-change correlation. BUY_NORMAL must answer on `BUY_NORMAL_SUCCESS` instead. Add a `BuyNormal bool` (or an operation discriminator matching Task 3's `ErrorOperation*` vocabulary) to `RequestPurchaseCommandBody` and `PurchaseEventBody` so the consumer can tell the two apart, defaulting to today's behavior when unset — the same additive-with-empty-default discipline Task 3 used for `ErrorEventBody`. On the failure side, the atlas-cashshop purchase path emits `ErrorStatusEventForOperationProvider(..., cashshop.ErrorOperationBuyNormal, ...)` for this discriminator so Task 3's switch picks `BUY_NORMAL_FAILED`.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(channel): implement BUY_NORMAL through the existing purchase pipeline"
```

---

## Task 10: The shared command ledger

Every new command in this plan is replay-safe via a `TransactionId` uniqueness claim inserted as the first statement inside its transaction. One shared table, not one per command — the key is globally unique and the row's only job is to be that claim.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/ledger/entity.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ledger/administrator.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ledger/duplicate.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ledger/administrator_test.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/main.go` — add `ledger.Migration` to `database.SetMigrations` (`:62`)

Patterns to copy: `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/` — `entity.go`, `administrator.go`, and `duplicate.go` are the exact shape to follow, including the dual-driver duplicate-key detection (Postgres SQLSTATE 23505 in production, sqlite extended codes 1555/2067 in tests). `duplicate.go` is a verbatim copy with the package name changed; that duplication is deliberate and already justified in `surprise/opening/duplicate.go`'s own doc comment.

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**Interfaces produced:**

```go
// ErrAlreadyProcessed means this (tenant, transaction) pair has already been
// committed -- a Kafka redelivery, not a new click. Callers treat it as
// success-without-effect, not as a failure to report to the client.
var ErrAlreadyProcessed = errors.New("command already processed for this transaction")

// Claim writes the ledger row. It MUST be the first statement in the command's
// transaction so a duplicate aborts before any state changes.
func Claim(db *gorm.DB, tenantId uuid.UUID, transactionId uuid.UUID, commandType string, characterId uint32) error
```

- [ ] **Step 1: Write the failing test**

`administrator_test.go`, in-memory sqlite with `Migration`. Tenant `A = uuid.New()`, transaction `X = uuid.New()`.

| subtest | action | expect |
|---|---|---|
| `first claim succeeds` | `Claim(db, A, X, "REQUEST_GIFT_PURCHASE", 42)` | `nil` |
| `replay is rejected` | `Claim(db, A, X, "REQUEST_GIFT_PURCHASE", 42)` again | `ErrAlreadyProcessed` |
| `replay under a different command type is still rejected` | `Claim(db, A, X, "REQUEST_LOCKER_REBATE", 42)` | `ErrAlreadyProcessed` — the key is (tenant, transaction), not (tenant, transaction, type) |
| `a different transaction succeeds` | `Claim(db, A, uuid.New(), "REQUEST_GIFT_PURCHASE", 42)` | `nil` |
| `the same transaction under a different tenant succeeds` | `Claim(db, uuid.New(), X, "REQUEST_GIFT_PURCHASE", 42)` | `nil` |
| `the zero transaction id is rejected outright` | `Claim(db, A, uuid.Nil, "REQUEST_GIFT_PURCHASE", 42)` | a non-nil error that is **not** `ErrAlreadyProcessed` |

The zero-UUID case matters: `RequestPurchaseCommandBody` documents `uuid.Nil` as "no correlation", so a nil id must never be allowed to become a shared uniqueness claim that blocks every subsequent uncorrelated command.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./ledger/... -v
```

Expected: build failure — package does not exist.

- [ ] **Step 3: Write the entity, duplicate detection, and administrator**

```go
// entity is the shared cash shop command ledger: one row per committed
// command, keyed by the transaction id atlas-channel mints per click. Its
// insert is the FIRST statement in each command's transaction, so a Kafka
// redelivery hits the primary-key violation and the whole transaction aborts
// without charging or delivering anything a second time.
//
// One shared table rather than one per command: the key is globally unique
// and the row's only job is to be a uniqueness claim. CommandType and
// CharacterId are recorded for the audit trail, not for the constraint.
type entity struct {
	TenantId      uuid.UUID `gorm:"primaryKey;not null"`
	TransactionId uuid.UUID `gorm:"primaryKey;not null"`
	CommandType   string    `gorm:"not null"`
	CharacterId   uint32    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null"`
}

func (e entity) TableName() string { return "cash_command_ledger" }
```

`Claim` returns an explicit error for `uuid.Nil`, inserts otherwise, and maps a duplicate-key violation to `ErrAlreadyProcessed` via `isDuplicateKeyError`.

- [ ] **Step 4: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./ledger/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): add the shared command ledger for transaction idempotency"
```

---

## Task 11: REBATE_LOCKER_ITEM — `atlas-cashshop` side

Refunds a locker item's purchase price and removes the asset, atomically and idempotently.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` — add `CommandTypeRequestLockerRebate`, `RequestLockerRebateCommandBody`, `StatusEventTypeLockerRebated`, `LockerRebatedBody`
- `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go` — add `LockerRebatedStatusEventProvider`
- `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go` — register and handle the command
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/rebate.go` — **new file**; the transaction
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/rebate_test.go` — **new file**

Patterns to copy: `cashshop/processor.go:100` `Purchase` (the `rejectEmit` + `ExecuteTransaction` + `message.Emit` shape, including why a rejection fires on the direct producer path rather than the outbox), `surprise/processor.go` (a ledger-claim-first transaction).

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**Interfaces produced:**

```go
const (
	CommandTypeRequestLockerRebate = "REQUEST_LOCKER_REBATE"
	StatusEventTypeLockerRebated   = "LOCKER_REBATED"
)

// RequestLockerRebateCommandBody refunds one locker asset. CashId is the
// client's GW_ItemSlotBase::liCashItemSN (cash_assets.CashId), NOT the row id
// -- see shop_operation_rebate_locker_item.go:18-21.
type RequestLockerRebateCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	CashId        int64     `json:"cashId"`
}

type LockerRebatedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CashId        int64     `json:"cashId"`
	Amount        int32     `json:"amount"`
	Currency      uint32    `json:"currency"`
}

func (p *ProcessorImpl) RebateAndEmit(characterId uint32, accountId uint32, cashId int64, transactionId uuid.UUID) error
```

- [ ] **Step 1: Write the failing test**

`cashshop/rebate_test.go`, in-memory sqlite migrated with the wallet, compartment, asset, purchaserecord, and ledger migrations. Build fixtures with the project's Builder pattern (see `cashshop/processor_test.go` for the existing setup in this exact package).

`TestRebate` — one `t.Run` per case:

| subtest | fixture | expect |
|---|---|---|
| `refunds the commodity price` | account 42 wallet credit 0; one asset with `CashId` 900001, `CommodityId` 10000 whose commodity `Price` is 1200 | asset gone from the compartment; wallet credit == 1200; a `LOCKER_REBATED` event with `Amount` 1200 and `CashId` 900001 |
| `replay refunds once` | run the same command a second time with the **same** `TransactionId` | wallet credit still 1200; asset still gone; no second event |
| `a new transaction id on a gone asset is rejected` | run with a **new** `TransactionId` for `CashId` 900001 | wallet unchanged; an `ERROR` event with `Operation == "REBATE"` and reason `"unknown_error"` |
| `an asset owned by another account is rejected` | `CashId` 900002 in account 99's compartment, requested for account 42 | wallet unchanged; `ERROR` with `Operation == "REBATE"` |
| `an expired asset is rejected` | asset with `Expiration` in the past | wallet unchanged; `ERROR` with `Operation == "REBATE"` |
| `an asset with no commodity id is rejected` | asset with `CommodityId` 0 (a gift, coupon reward, or surprise drop — never bought with currency) | wallet unchanged; `ERROR` with `Operation == "REBATE"` (FR-REB-4) |
| `the purchase record survives the rebate` | after the successful case, `purchaserecord.Get(db, tenant, 42, 10000)` | still `1` — "purchased" is a historical fact (design §6) |

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./cashshop/... -run TestRebate -v
```

Expected: compile failure — `undefined: RebateAndEmit`.

- [ ] **Step 3: Implement the transaction**

Order inside `database.ExecuteTransaction`, strictly:

1. `ledger.Claim(tx, p.t.Id(), transactionId, cashshop.CommandTypeRequestLockerRebate, characterId)` — first statement. `ErrAlreadyProcessed` returns without emitting anything and without an error (a redelivery is success-without-effect).
2. Resolve the asset by `CashId` **within the requesting account's compartments only** — an asset in another account's compartment is simply absent from the scan, so it reports the same rejection as a missing one.
3. Reject when `CommodityId == 0`, when `Expiration` is in the past, or when the commodity cannot be resolved.
4. Delete the asset (`p.astP.Delete(mb)(am.Id())`).
5. Credit the wallet by the commodity's `Price`, on the currency the design records as the purchase currency. **The wallet currency the purchase was drawn from is not recorded on `cash_assets` today** — resolve it explicitly: either read it from the purchase record, or extend the asset row with the currency it was bought on. Pick one, implement it, and state which in the doc comment. Do not credit a guessed currency.
6. `mb.Put(...LockerRebatedStatusEventProvider(...))`.

Every rejection sets `rejectEmit` to `ErrorStatusEventForOperationProvider(characterId, cashshop.ErrorOperationRebate, <reason>, transactionId)` and returns the sentinel, exactly as `Purchase` does.

- [ ] **Step 4: Register the command handler**

Add `handleCommandRequestLockerRebate` to `kafka/consumer/cashshop/consumer.go` and register it in `InitHandlers` alongside the existing nine.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): implement idempotent locker rebate"
```

---

## Task 12: REBATE_LOCKER_ITEM — `atlas-channel` side

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` — mirror the command and event types from Task 11
- `services/atlas-channel/atlas.com/channel/cashshop/producer.go` — add `RequestLockerRebateCommandProvider`
- `services/atlas-channel/atlas.com/channel/cashshop/processor.go` — add `RequestLockerRebate` to the `Processor` interface (`:19`) and `ProcessorImpl`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — add `handleStatusEventLockerRebated`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` — replace the log-only arm at `:158-163`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_rebate_test.go` — **new file**

Patterns to copy: `cashshop/producer.go:140` `OpenSurpriseCommandProvider` (a transaction-id-carrying command), `kafka/consumer/cashshop/consumer.go:339` `handleStatusEventSurpriseOpened` (a status-event handler that announces a specific arm).

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

`cash_shop_rebate_test.go`. `TestCashShopRebateDoneBodyEncodes` over `cashcb.CashShopRebateDoneBody(sn int64, amount int32)` with options `{"operations": {cashcb.CashShopOperationRebateSuccess: float64(150)}}` (resolve the constant's real identifier from `shop_operation_body.go:600`):

| subtest | sn | amount | expect |
|---|---|---|---|
| `refund` | 900001 | 1200 | first byte 150, then the exact bytes `NewRebateDone`'s `Encode` writes for `(900001, 1200)` |

Read the encoder and write the full expected slice — do not infer the widths from the Go parameter types.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run TestCashShopRebateDoneBody -v
```

Expected: FAIL on the byte assertion.

- [ ] **Step 3: Mirror the Kafka types and add the producer + processor method**

`RequestLockerRebate(accountId uint32, characterId uint32, cashId int64, transactionId uuid.UUID) error`, emitting on `cashshop.EnvCommandTopic`.

- [ ] **Step 4: Replace the log-only arm**

Decode, run `verifySecondaryCredential(l, ctx)(s, sp.SPW(), sp.Birthday())` from Task 4 — on `ErrCredentialMismatch` announce `cashcb.CashShopRebateFailedBody("INVALID_BIRTHDAY")` and charge nothing — then mint `uuid.New()` and call `RequestLockerRebate(s.AccountId(), s.CharacterId(), int64(sp.Unk()), transactionId)`. `sp.Unk()` is the locker cash serial (design §0 row 3, citing `shop_operation_rebate_locker_item.go:18-21`); name the local variable `cashId`, not `unk`.

- [ ] **Step 5: Add the consumer handler**

`handleStatusEventLockerRebated` announces `cashcb.CashShopRebateDoneBody(e.Body.CashId, e.Body.Amount)`. Register it in `InitHandlers` alongside the existing seven.

- [ ] **Step 6: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel
git commit -m "feat(channel): implement REBATE_LOCKER_ITEM end to end"
```

---

## Task 13: GIFT — `atlas-cashshop` side

Charges the sender's wallet and creates the item in the **recipient's** locker, atomically.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/entity.go` — add `GiftFrom` and `GiftMessage` columns
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/model.go` — add the accessors and builder setters
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/administrator.go` — carry the new columns through create
- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` — add `CommandTypeRequestGiftPurchase`, `RequestGiftPurchaseCommandBody`, `StatusEventTypeGiftPurchased`, `GiftPurchasedBody`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/gift.go` — **new file**; the transaction
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/gift_test.go` — **new file**

Also touched (registration only, one line each): `kafka/producer/cashshop/producer.go` (`GiftPurchasedStatusEventProvider`), `kafka/consumer/cashshop/consumer.go` (`handleCommandRequestGiftPurchase` + its `InitHandlers` line).

Patterns to copy: `cashshop/processor.go:100` `Purchase`.

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**Migration note:** the two new `cash_assets` columns are additive, defaulted-empty strings. Existing rows are unaffected and no backfill is needed. `GiftFrom` is bounded at 13 characters (the padded encode width in `CashInventoryItem`) and `GiftMessage` at 73 (the width in `GiftListEntry`) — verify both widths against `libs/atlas-packet/cash/clientbound/` before writing the column definitions rather than trusting these numbers.

**Interfaces produced:**

```go
const (
	CommandTypeRequestGiftPurchase = "REQUEST_GIFT_PURCHASE"
	StatusEventTypeGiftPurchased   = "GIFT_PURCHASED"
)

// RequestGiftPurchaseCommandBody. The channel resolves the recipient NAME to
// a character id before sending, because atlas-cashshop's character client has
// only GetById (character/processor.go:15) -- there is no name lookup here.
type RequestGiftPurchaseCommandBody struct {
	TransactionId        uuid.UUID `json:"transactionId"`
	SerialNumber         uint32    `json:"serialNumber"`
	RecipientCharacterId uint32    `json:"recipientCharacterId"`
	SenderName           string    `json:"senderName"`
	Message              string    `json:"message"`
}

type GiftPurchasedBody struct {
	TransactionId        uuid.UUID `json:"transactionId"`
	RecipientName        string    `json:"recipientName"`
	TemplateId           uint32    `json:"templateId"`
	Quantity             uint16    `json:"quantity"`
	Price                uint32    `json:"price"`
	RecipientCharacterId uint32    `json:"recipientCharacterId"`
}

func (p *ProcessorImpl) GiftAndEmit(characterId uint32, transactionId uuid.UUID, serialNumber uint32, recipientCharacterId uint32, senderName string, message string) error
```

- [ ] **Step 1: Write the failing test**

`cashshop/gift_test.go`, same DB setup as Task 11.

`TestGift` — sender character 42 on account 1, recipient character 77 on account 2, commodity 10000 with `Price` 1200 and `ItemId` 5010000:

| subtest | fixture | expect |
|---|---|---|
| `delivers to the recipient locker` | sender account 1 credit 5000 | sender wallet credit == 3800; a `cash_assets` row in **account 2's** compartment with `TemplateId` 5010000, `GiftFrom` == the sender name, `GiftMessage` == the message; **no** new row in account 1's compartment; a `GIFT_PURCHASED` event |
| `insufficient funds charges nothing` | sender credit 100 | no asset in either compartment; sender credit still 100; `ERROR` with `Operation == "GIFT"`, reason `"NOT_ENOUGH_CASH"` |
| `recipient locker full charges nothing` | recipient compartment at capacity | no asset created; sender credit unchanged; `ERROR` with `Operation == "GIFT"`, reason `"CANNOT_GIFT_RECIPIENT_INVENTORY_FULL"` (FR-GIFT-6; design §13 decides reject-not-queue) |
| `unknown commodity charges nothing` | serial 99999 with no commodity | no asset; credit unchanged; `ERROR` with `Operation == "GIFT"` |
| `replay delivers once` | rerun with the same `TransactionId` | exactly one asset in account 2's compartment; sender credit still 3800 |
| `records the purchase for the sender` | after the success case | `purchaserecord.Get(db, tenant, 1, 10000)` == 1 and `Get(db, tenant, 2, 10000)` == 0 — the **sender** bought it |

The capacity case is the one that proves the check runs against the *recipient's* compartment, not the sender's.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./cashshop/... -run TestGift -v
```

Expected: compile failure — `undefined: GiftAndEmit`.

- [ ] **Step 3: Add the `cash_assets` gift columns**

Entity, model accessors, builder setters, `Make`, and the administrator's create path. `asset.Migration` already runs `AutoMigrate`, so the additive columns land on the existing table with no separate migration file.

- [ ] **Step 4: Implement the transaction**

Order inside `database.ExecuteTransaction`:

1. `ledger.Claim` — first statement.
2. Resolve the commodity by `serialNumber`.
3. Resolve the sender's character → account, and the recipient's character → account.
4. Resolve the recipient's compartment for their job type (the same explorer/cygnus/legend selection `Purchase` makes at `cashshop/processor.go:139-146`) and check its capacity.
5. Debit the sender's wallet.
6. Create the asset in the **recipient's** compartment, carrying `GiftFrom`/`GiftMessage`, with `PurchasedBy` set to the sender's character id.
7. `purchaserecord.Record(tx, tenant, senderAccountId, serialNumber)`.
8. `mb.Put(...GiftPurchasedStatusEventProvider(...))`.

Every rejection sets `rejectEmit` with `ErrorOperationGift`.

- [ ] **Step 5: Register the command handler and producer**

- [ ] **Step 6: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): implement gift purchase into the recipient locker"
```

---

## Task 14: GIFT — `atlas-channel` side

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` — mirror Task 13's types
- `services/atlas-channel/atlas.com/channel/cashshop/producer.go` — add `RequestGiftPurchaseCommandProvider`
- `services/atlas-channel/atlas.com/channel/cashshop/processor.go` — add `RequestGiftPurchase` to the interface and impl
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — add `handleStatusEventGiftPurchased`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_gift.go` — **new file**; `handleGift`, mirroring `handleBuyNameChange`'s placement in `cash_shop_operation.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_gift_test.go` — **new file**

Also touched (one line): `socket/handler/cash_shop_operation.go` — the `CashShopOperationGift` arm at `:62-67` calls `handleGift`.

Patterns to copy: `socket/handler/cash_shop_operation.go:243` `handleBuyNameChange` (edge validation, then a Kafka request, with every rejection answered).

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

`cash_shop_gift_test.go`. Two pure-function tests plus a body test.

`TestGiftRejectionReason` — the mapper from an edge rejection to an errors-table key:

| subtest | condition | expect key |
|---|---|---|
| `unknown recipient` | `character.GetByName` returned not-found | `"INCORRECT_NAME"` |
| `recipient on the sender's own account` | recipient account == sender account | `"CANNOT_GIFT_TO_OWN_ACCOUNT"` |
| `credential mismatch` | `ErrCredentialMismatch` | `"INVALID_BIRTHDAY"` |
| `anything else` | an arbitrary error | `"unknown_error"` |

Every one of these keys is bound in `template_gms_95_1.json`'s `errors` table (values 7, 6, 34, 69 respectively) — verified at plan time.

`TestCashShopGiftDoneBodyEncodes` over `cashcb.CashShopGiftDoneBody(recipientName string, itemId int32, quantity uint16, nxCashSpent int32)` with `{"operations": {cashcb.CashShopOperationGiftSuccess: float64(107)}}`: assert the first byte is 107 and the remaining bytes match `NewGiftDone`'s `Encode` for `("Recipient", 5010000, 1, 1200)`. Read the encoder and write the slice out in full.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'TestGiftRejectionReason|TestCashShopGiftDoneBody' -v
```

Expected: compile failure — `undefined: giftRejectionReason`.

- [ ] **Step 3: Implement `handleGift`**

In order, each failure answering `cashcb.CashShopGiftFailedBody(giftRejectionReason(err))` and charging nothing:

1. `verifySecondaryCredential(l, ctx)(s, sp.SPW(), sp.Birthday())` (FR-GIFT-2).
2. `character.NewProcessor(l, ctx).GetByName(sp.Name())` — unknown name rejects (FR-GIFT-1). Confirm the resolved character is in the session's world before accepting it.
3. Reject when the recipient's `AccountId()` equals `s.AccountId()` (FR-GIFT-3).
4. Resolve the sender's own character name for `SenderName` (the recipient's locker shows who gifted it).
5. Mint `uuid.New()` and call `RequestGiftPurchase(...)`.

Apply `derivation.md` D4b for `oneADay`: if it is a server-enforced per-day limit, enforce it here and reject with the errors-table key D4b names; if it is a client-side UI flag, say so in the doc comment and ignore the field deliberately. Do not silently drop it without recording which.

Before announcing any failure, run the reason through the existing `atlaspacket.CodeConfigured(opts, "errors", reason)` predicate the way `transferFailureReasonConfigured` does at `cash_shop_operation.go`, logging a warning when a tenant does not bind the key and sending anyway — a wedged cash shop dialog is worse than a generic code.

- [ ] **Step 4: Add the consumer handler**

`handleStatusEventGiftPurchased` announces `cashcb.CashShopGiftDoneBody(e.Body.RecipientName, int32(e.Body.TemplateId), e.Body.Quantity, int32(e.Body.Price))` to the sender's session.

**Recipient-side live refresh is deliberately out of scope.** `REFRESH_LOCKER` (mode 162) is not bound in the `operations` table of *any* GMS seed template — verified at plan time — so announcing it would resolve to the `ResolveCode` sentinel. The gifted asset is durable in the recipient's locker either way (FR-GIFT-6), and they see it on their next locker load. Record this in `context.md`.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel
git commit -m "feat(channel): implement GIFT end to end"
```

---

## Task 15: Cash package catalogue in `atlas-data`

Ingests `Etc.wz/CashPackage.img`, where each package is an `imgdir` keyed by the package **item id** (e.g. `9100000`) containing an `SN` `imgdir` of member commodity serial numbers.

### Files

- `services/atlas-data/atlas.com/data/cashpackage/rest.go` — **new file**
- `services/atlas-data/atlas.com/data/cashpackage/registry.go` — **new file**
- `services/atlas-data/atlas.com/data/cashpackage/reader.go` — **new file**
- `services/atlas-data/atlas.com/data/cashpackage/reader_test.go` — **new file**
- `services/atlas-data/atlas.com/data/cashpackage/processor.go` — **new file**
- `services/atlas-data/atlas.com/data/cashpackage/resource.go` — **new file**

Also touched (one line each): `data/processor.go:175-176` (the `WorkerCommodity` branch gains a second `RegisterFileData` call), `data/workers/commodity.go:38` (the worker gains a second register call), `main.go:192` (the route initializer).

Patterns to copy: `services/atlas-data/atlas.com/data/commodity/` — `rest.go`, `registry.go`, `reader.go`, `processor.go`, `resource.go` are the file-for-file template.

Module root: `services/atlas-data/atlas.com/data`.

**Design decision made at plan time:** the ingest rides inside the **existing `Commodity` worker** rather than a new worker. Both files live under `Etc.wz`, which the Commodity worker already fetches and serializes, so a new worker would re-download the archive for one small file. Recorded in `context.md`.

**Missing-file tolerance is already confirmed, not assumed:** `data/processor.go:298-303` shows `RegisterFileData` discards the register function's return value and always returns `nil`, so a tenant whose WZ dump lacks `CashPackage.img.xml` still boots. The worker path (`workers/commodity.go`) does *not* discard errors, so the cash-package register call there must be logged-and-continued rather than returned.

**Interfaces produced:**

```go
type RestModel struct {
	Id            uint32   `json:"-"`   // the imgdir key: the package ITEM id
	SerialNumbers []uint32 `json:"serialNumbers"`
}

func (r RestModel) GetName() string { return "cashPackages" }

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] // document key "CASH_PACKAGE"
func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel]
func (p *ProcessorImpl) RegisterCashPackage(path string) error
func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer
```

- [ ] **Step 1: Write the failing test**

`reader_test.go`. Build an `xml.Node` tree in Go — do **not** read a fixture file from disk; `Etc.wz` dumps are external to this repository. Copy the node-construction shape from `services/atlas-data/atlas.com/data/commodity/reader_test.go`.

`TestReadCashPackages` — one root node with three children:

| child imgdir name | children | expect |
|---|---|---|
| `9100000` | an `SN` imgdir with integer children `0`=10000, `1`=10001, `2`=10002 | `RestModel{Id: 9100000, SerialNumbers: []uint32{10000, 10001, 10002}}` |
| `9100001` | an `SN` imgdir with one integer child `0`=20000 | `RestModel{Id: 9100001, SerialNumbers: []uint32{20000}}` |
| `9100002` | no `SN` child at all | `RestModel{Id: 9100002, SerialNumbers: []uint32{}}` — an empty slice, never `nil`, so the JSON:API body carries `[]` rather than `null` |

Assert the full returned slice, in order, all three elements.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-data/atlas.com/data && go test ./cashpackage/... -v
```

Expected: build failure — package does not exist.

- [ ] **Step 3: Write `rest.go`, `registry.go`, `reader.go`**

`Read` iterates the root's child nodes; `Id` is the child's own name parsed as a uint32 (a child whose name does not parse is skipped and logged); `SerialNumbers` are the integer values under its `SN` child. Read `commodity/reader.go` and the `xml.Node` accessors it uses before writing this — the `SN` values are a nested imgdir of integers, not a single `GetIntegerWithDefault` call, so the accessor differs from commodity's.

- [ ] **Step 4: Run the test**

```sh
cd services/atlas-data/atlas.com/data && go test ./cashpackage/... -v
```

Expected: PASS.

- [ ] **Step 5: Write `processor.go` and `resource.go`**

`processor.go` mirrors `commodity/processor.go` verbatim with document key `"CASH_PACKAGE"` and `RegisterCashPackage(path string)`. `resource.go` mirrors `commodity/resource.go`: `GET /data/cashPackages` (paginated) and `GET /data/cashPackages/{packageId}`, where a miss is a `404`.

- [ ] **Step 6: Wire the ingest and the route**

Add the second `RegisterFileData(path, filepath.Join("Etc.wz", "CashPackage.img.xml"), cashpackage.NewProcessor(...).RegisterCashPackage)()` call in the `WorkerCommodity` branch of `data/processor.go`, the corresponding logged-and-continued call in `workers/commodity.go`, and `AddRouteInitializer(cashpackage.InitResource(db)(GetServer()))` in `main.go` next to `commodity`'s at `:192`.

- [ ] **Step 7: Build and test**

```sh
cd services/atlas-data/atlas.com/data && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data
git commit -m "feat(data): ingest and expose the cash package catalogue"
```

---

## Task 16: Package purchase — `atlas-cashshop` side

One command covers both package modes, discriminated by `RecipientCharacterId` — modes 30 and 31 differ only in *who* receives the members; resolution, capacity, atomicity, and pricing are identical.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/data/cashpackage/` — **new files** `model.go`, `rest.go`, `requests.go`, `processor.go`; the atlas-data client
- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` — add `CommandTypeRequestPackagePurchase`, `RequestPackagePurchaseCommandBody`, `StatusEventTypePackagePurchased`, `PackagePurchasedBody`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/package.go` — **new file**; the transaction
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/package_test.go` — **new file**

Also touched (one line each): `kafka/producer/cashshop/producer.go`, `kafka/consumer/cashshop/consumer.go`.

Patterns to copy: `services/atlas-cashshop/atlas.com/cashshop/data/pet/` (the atlas-data client, file-for-file), `cashshop/processor.go:100` `Purchase`.

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**The resolution chain (design §5.1) — the package id is an item id, not a serial number:**

```
sp.SerialNumber() -> commodity(SN) -> commodity.ItemId == the CashPackage.img key
                  -> cashPackage(ItemId).SerialNumbers -> member commodity SNs
                  -> for each: commodity(memberSN) -> ItemId, Count, Period
                  -> one cash_assets row per member
```

The price charged once is the **package commodity's own `Price`** — already resolved in step 1, so the "sum of members" mistake is structurally impossible (FR-PKG-5).

**Interfaces produced:**

```go
const (
	CommandTypeRequestPackagePurchase = "REQUEST_PACKAGE_PURCHASE"
	StatusEventTypePackagePurchased   = "PACKAGE_PURCHASED"
)

// RecipientCharacterId is ZERO for a buy-for-self (mode 30) and non-zero for
// a gift (mode 31). One command, because both modes share every rule except
// which locker the members land in.
type RequestPackagePurchaseCommandBody struct {
	TransactionId        uuid.UUID `json:"transactionId"`
	Currency             uint32    `json:"currency"`
	SerialNumber         uint32    `json:"serialNumber"`
	RecipientCharacterId uint32    `json:"recipientCharacterId"`
	SenderName           string    `json:"senderName"`
}

type PackagePurchasedBody struct {
	TransactionId        uuid.UUID `json:"transactionId"`
	CompartmentId        uuid.UUID `json:"compartmentId"`
	AssetIds             []uint32  `json:"assetIds"`
	PackageTemplateId    uint32    `json:"packageTemplateId"`
	Price                uint32    `json:"price"`
	RecipientCharacterId uint32    `json:"recipientCharacterId"`
	RecipientName        string    `json:"recipientName"`
}

func (p *ProcessorImpl) PurchasePackageAndEmit(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, recipientCharacterId uint32, senderName string) error
```

- [ ] **Step 1: Write the failing test**

`cashshop/package_test.go`. Fixture: package commodity SN 50000 → `ItemId` 9100000, `Price` 3000. Cash package 9100000 → member SNs `{10000, 10001, 10002}`, whose commodities have `ItemId` 5010000/5010001/5010002, `Count` 1 each and `Price` 1200 each. Stub the atlas-data client so the test does not hit the network — follow how `cashshop/processor_test.go` stubs `dataPetP`.

| subtest | fixture | expect |
|---|---|---|
| `creates one asset per member` | buyer account 1, credit 5000, empty compartment capacity 10 | three `cash_assets` rows with template ids 5010000/5010001/5010002; wallet credit == 2000 (`5000 - 3000`, the **package** price, not `3 x 1200`); a `PACKAGE_PURCHASED` event whose `AssetIds` has length 3 |
| `an unresolvable member creates nothing` | member SN 10001 has no commodity | zero assets created; credit still 5000; `ERROR` with `Operation == "BUY_PACKAGE"` (FR-PKG-4) |
| `capacity is checked against the full member count before charging` | compartment capacity 2 | zero assets created; credit still 5000; `ERROR` with `Operation == "BUY_PACKAGE"`, reason `"INVENTORY_FULL"` (FR-PKG-6) |
| `insufficient funds charges nothing` | credit 100 | zero assets; `ERROR` reason `"NOT_ENOUGH_CASH"` |
| `an unknown package creates nothing` | package commodity resolves but 9100000 has no CashPackage entry | zero assets; credit unchanged; `ERROR` with `Operation == "BUY_PACKAGE"` |
| `gift mode delivers to the recipient` | `RecipientCharacterId` = 77 on account 2 | three assets in **account 2's** compartment, none in account 1's; account 1's credit == 2000; the event's `Operation`-side error key on failure is `"GIFT_PACKAGE"` |
| `replay delivers once` | rerun with the same `TransactionId` | still exactly three assets; credit still 2000 |
| `records the package and every member` | after the first success | `purchaserecord.Get` returns 1 for SN 50000 **and** for 10000, 10001, 10002 — the client can ask about either (design §6) |

The capacity case is the one that proves the check precedes the debit rather than failing partway through.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./cashshop/... -run TestPurchasePackage -v
```

Expected: compile failure — `undefined: PurchasePackageAndEmit`.

- [ ] **Step 3: Write the atlas-data cash-package client**

`data/cashpackage/{model,rest,requests,processor}.go`, mirroring `data/pet/` file-for-file, against `GET /data/cashPackages/{packageId}`.

- [ ] **Step 4: Implement the transaction**

Order inside `database.ExecuteTransaction`: ledger claim → resolve the package commodity → resolve the cash package → resolve **every** member commodity (any failure aborts before anything is written) → resolve the target compartment (recipient's when `RecipientCharacterId` is non-zero, otherwise the buyer's) → check `Capacity() - len(Assets()) >= len(members)` → debit the buyer's wallet by the package commodity's `Price` → create one asset per member → `purchaserecord.Record` for the package SN and each member SN → emit.

Rejections use `ErrorOperationBuyPackage` when `RecipientCharacterId == 0` and `ErrorOperationGiftPackage` otherwise, so Task 3's switch picks the arm the client is waiting on.

- [ ] **Step 5: Register the command handler and producer**

- [ ] **Step 6: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): implement atomic cash package purchase"
```

---

## Task 17: BUY_PACKAGE (30/32) and BUY_OTHER_PACKAGE (31/33) — `atlas-channel` side

`BUY_OTHER_PACKAGE` has no dispatch arm today — `CashShopOperationBuyOtherPackage` is declared at `cash_shop_operation.go:40` and referenced nowhere else. It *is* already bound in every GMS template's `operations` table (gms_95 = 33, gms_83 = 31), so this is a Go gap, not a config gap.

### Files

- `docs/tasks/task-240-cash-shop-stub-operations/derivation.md` — read-only (new file, written by Task 1); **D3a** gives mode 31's body, **D3b** its result arm
- `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go` — **new file**, *only if D3a found a body distinct from `ShopOperationBuyPackage`*
- `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package_test.go` — **new file**, same condition
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_package.go` — **new file**; `handleBuyPackage` and `handleBuyOtherPackage`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_package_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — add `handleStatusEventPackagePurchased`

Also touched: `socket/handler/cash_shop_operation.go` (the `BUY_PACKAGE` arm at `:169-175` calls `handleBuyPackage`; a **new** `BUY_OTHER_PACKAGE` arm calls `handleBuyOtherPackage`), `cashshop/producer.go` + `cashshop/processor.go` (the `RequestPackagePurchase` emit), `kafka/message/cashshop/kafka.go` (mirror Task 16's types).

Module roots: `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Read D3 and record the branch taken**

If D3a says mode 31 carries the same body as mode 30, reuse `cashsb.ShopOperationBuyPackage` for both arms and say so in `handleBuyOtherPackage`'s doc comment, citing the derivation section. If it carries a distinct body, add the codec. If D3 came back **UNRESOLVED**, route the arm to a typed `cashcb.CashShopGiftPackageFailedBody("unknown_error")` with a logged warning naming the unresolved derivation — an unrouted arm that falls through to `Unhandled Cash Shop Operation` is what this task exists to eliminate, and a typed failure is not a stub.

- [ ] **Step 2: Write the failing test**

`cash_shop_package_test.go`. `TestCashShopPackageResultBodies` — table-driven over both success arms, options carrying the gms_95 values:

```go
options := map[string]interface{}{
	"operations": map[string]interface{}{
		cashcb.CashShopOperationBuyPackageSuccess:  float64(154),
		cashcb.CashShopOperationGiftPackageSuccess: float64(156),
	},
}
```

| subtest | constructor + args | expect first byte | expect remainder |
|---|---|---|---|
| `buy package` | `cashcb.CashShopBuyPackageDoneBody(items, 0)` with a one-element `[]cashcb.CashInventoryItem` | 154 | the exact bytes `NewBuyPackageDone`'s `Encode` writes |
| `gift package` | `cashcb.CashShopGiftPackageDoneBody("Recipient", 9100000, 0, 0, 3000)` | 156 | the exact bytes `NewGiftPackageDone`'s `Encode` writes |

Read both encoders in `shop_operation_body.go` and write both expected slices in full.

`TestBuyOtherPackageIsDispatched` — assert `isCashShopOperation(logrus.New())(options, 33, CashShopOperationBuyOtherPackage)` returns `true` for `options = map[string]interface{}{"operations": map[string]interface{}{"BUY_OTHER_PACKAGE": float64(33)}}` and `false` for op `32`. This is the acceptance criterion "grep shows it referenced beyond its declaration", turned into a test.

- [ ] **Step 3: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'TestCashShopPackage|TestBuyOtherPackage' -v
```

Expected: FAIL on the byte assertions.

- [ ] **Step 4: Implement both arms**

`handleBuyPackage`: decode, `resolvePurchaseCurrency(sp.PointType(), 0)` (the body carries `pointType bool` and no currency int — `shop_operation_buy_package.go:61`), mint `uuid.New()`, call `RequestPackagePurchase(...)` with `RecipientCharacterId = 0`.

`handleBuyOtherPackage`: same, plus recipient-name resolution and the self-gift rejection, exactly as `handleGift` does — reuse `giftRejectionReason` from Task 14 rather than writing a second mapper — answering on `cashcb.CashShopGiftPackageFailedBody`.

Apply `derivation.md` D4a for `option`: if D4a proved it ignorable, say so in the doc comment citing the section; if it carries meaning, honor it. Do not ignore it silently.

- [ ] **Step 5: Add the consumer handler**

`handleStatusEventPackagePurchased` picks the body from `e.Body.RecipientCharacterId`: zero → `CashShopBuyPackageDoneBody` built from the assets named in `AssetIds` (project each to a `cashpkt.CashInventoryItem` the same way `handleStatusEventPurchase` does at `consumer.go:160-169`), non-zero → `CashShopGiftPackageDoneBody(e.Body.RecipientName, int32(e.Body.PackageTemplateId), 0, 0, int32(e.Body.Price))`. The `unused1`/`unused2` arguments are named "unused" in the existing constructor; keep them zero and note that the derivation did not contradict it.

- [ ] **Step 6: Run the tests**

```sh
cd libs/atlas-packet && go build ./... && go test ./cash/...
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet services/atlas-channel/atlas.com/channel
git commit -m "feat(channel): implement BUY_PACKAGE and route BUY_OTHER_PACKAGE"
```

---

## Task 18: The ring-pairing domain

Placed **inside `atlas-cashshop`** (design §4.1). The deciding argument is FR-RING-4: the pair record and both ring assets must commit in one transaction, and the assets live in this service's database. Cross-service atomicity would need a saga, and a saga for a two-row insert is not a trade this task should make. A future effects consumer reads the pair over REST — a normal cross-service read.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/ring/entity.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ring/model.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ring/administrator.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ring/provider.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ring/processor.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ring/administrator_test.go` — **new file**

Also touched (one line each): `main.go:62` (`ring.Migration`).

Patterns to copy: `services/atlas-cashshop/atlas.com/cashshop/wishlist/` — `model.go`, `entity.go`, `administrator.go`, `provider.go`, `processor.go` file-for-file.

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**Constants check performed at plan time:** `libs/atlas-constants/item/constants.go:24` defines `ClassificationRing = Classification(111)` — an *item* classification, not a pairing type. There is no ring-pair or ring-type constant anywhere in `libs/atlas-constants/`. `RingType` is therefore a service-local typed string in this package.

**Interfaces produced:**

```go
type Type string

const (
	TypeCouple     = Type("COUPLE")
	TypeFriendship = Type("FRIENDSHIP")
)

type State string

const (
	StateActive  = State("ACTIVE")
	StateBroken  = State("BROKEN")
	StateExpired = State("EXPIRED")
)

func Migration(db *gorm.DB) error

// CreatePair inserts BOTH halves of a pair in one call. It is called inside
// the ring purchase transaction; a partial pair must never be persistable
// (FR-RING-4).
func CreatePair(db *gorm.DB, tenantId uuid.UUID, ringType Type, a Half, b Half) (uuid.UUID, error)

// Half is one side of a pair before it has a pair id.
type Half struct {
	CharacterId    uint32
	AssetId        uint32
	ItemTemplateId uint32
}

func GetByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error)
func GetById(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) (Model, error)
```

- [ ] **Step 1: Write the failing test**

`administrator_test.go`, in-memory sqlite with `Migration`.

`TestCreatePair` — tenant `A`, characters 42 and 77:

| subtest | action | expect |
|---|---|---|
| `creates two halves sharing a pair id` | `CreatePair(db, A, TypeCouple, Half{42, 1001, 1112000}, Half{77, 1002, 1112000})` | returns a non-nil pair id; `GetByCharacterId(db, A, 42)` returns exactly one model whose `PairId()` equals it, `CharacterId()` == 42, `PartnerCharacterId()` == 77, `AssetId()` == 1001, `Type()` == `TypeCouple`, `State()` == `StateActive` |
| `the partner half mirrors it` | `GetByCharacterId(db, A, 77)` | one model, same `PairId()`, `CharacterId()` == 77, `PartnerCharacterId()` == 42, `AssetId()` == 1002 |
| `friendship pairs are distinguishable` | `CreatePair(db, A, TypeFriendship, Half{42, 1003, 1112800}, Half{88, 1004, 1112800})` | `GetByCharacterId(db, A, 42)` now returns two models; exactly one has `Type() == TypeFriendship` |
| `another tenant sees nothing` | `GetByCharacterId(db, uuid.New(), 42)` | empty slice, no error |
| `a character with no rings is not an error` | `GetByCharacterId(db, A, 999)` | empty slice, `nil` error |

`TestCreatePairIsAtomic` — call `CreatePair` inside a transaction that is then rolled back, and assert `GetByCharacterId` afterwards returns empty for both characters. A partial pair must not be persistable.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./ring/... -v
```

Expected: build failure — package does not exist.

- [ ] **Step 3: Write the entity, model, administrator, provider, processor**

```go
// Entity is one HALF of a ring pair. Two rows share a PairId; both are
// inserted in the same transaction as the two cash assets and the wallet
// debit, so a partial pair is not persistable (FR-RING-4).
//
// State rather than delete-only: a later task that breaks, expires, or
// un-equips a ring needs somewhere to record that without losing the history
// (FR-RING-9).
type Entity struct {
	Id                 uuid.UUID `gorm:"primaryKey;not null"`
	TenantId           uuid.UUID `gorm:"not null;index"`
	PairId             uuid.UUID `gorm:"not null;index"`
	CharacterId        uint32    `gorm:"not null;index"`
	PartnerCharacterId uint32    `gorm:"not null"`
	AssetId            uint32    `gorm:"not null"`
	ItemTemplateId     uint32    `gorm:"not null"`
	RingType           string    `gorm:"not null"`
	State              string    `gorm:"not null"`
	CreatedAt          time.Time `gorm:"not null"`
}

func (e Entity) TableName() string { return "cash_rings" }
```

`CreatePair` mints one `PairId`, builds both rows, and inserts them with a single `db.Create(&[]Entity{...})` so the two land or neither does.

- [ ] **Step 4: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./ring/... -v
```

Expected: PASS.

- [ ] **Step 5: Add the migration**

`ring.Migration` in `main.go:62`'s `database.SetMigrations` list.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): add the ring pairing domain"
```

---

## Task 19: Ring purchase transaction and REST surface

### Files

- `services/atlas-cashshop/atlas.com/cashshop/ring/rest.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/ring/resource.go` — **new file**
- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` — add `CommandTypeRequestRingPurchase`, `RequestRingPurchaseCommandBody`, `StatusEventTypeRingPurchased`, `RingPurchasedBody`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/ring.go` — **new file**; the transaction
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/ring_test.go` — **new file**

Also touched (one line each): `main.go` (route initializer next to `:127`), `kafka/producer/cashshop/producer.go`, `kafka/consumer/cashshop/consumer.go`.

Patterns to copy: `wishlist/resource.go` (the filtered-by-character REST shape), `cashshop/gift.go` from Task 13 (a two-account transaction).

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

**Interfaces produced:**

```go
const (
	CommandTypeRequestRingPurchase = "REQUEST_RING_PURCHASE"
	StatusEventTypeRingPurchased   = "RING_PURCHASED"
)

type RequestRingPurchaseCommandBody struct {
	TransactionId      uuid.UUID `json:"transactionId"`
	Currency           uint32    `json:"currency"`
	SerialNumber       uint32    `json:"serialNumber"`
	PartnerCharacterId uint32    `json:"partnerCharacterId"`
	SenderName         string    `json:"senderName"`
	Message            string    `json:"message"`
	RingType           string    `json:"ringType"` // ring.TypeCouple | ring.TypeFriendship
}

type RingPurchasedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	PartnerName   string    `json:"partnerName"`
	TemplateId    uint32    `json:"templateId"`
	Quantity      uint16    `json:"quantity"`
	RingType      string    `json:"ringType"`
	PairId        uuid.UUID `json:"pairId"`
}

func (p *ProcessorImpl) PurchaseRingAndEmit(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, partnerCharacterId uint32, senderName string, message string, ringType string) error
```

**Item-template selection (design §4.3, OQ-R1).** The arm carries one `serialNumber`, i.e. one commodity, but a pair needs two ring items. Implement the **same-template** path: both halves are created from the resolved commodity's `ItemId`. This is the confirmed-correct case for friendship rings. If the commodity's item id turns out to belong to a couple-ring range with distinct halves, reject with a typed `COUPLE_FAILED`/`FRIENDSHIP_FAILED` rather than deriving a partner template by an invented offset. Do **not** write a `+1` or gender-based rule. Record the rejection path and its reason in `context.md`.

- [ ] **Step 1: Write the failing test**

`cashshop/ring_test.go`. Buyer character 42 on account 1, partner character 77 on account 2. Commodity SN 60000 → `ItemId` 1112800 (a friendship ring), `Price` 2500, `Count` 1.

| subtest | fixture | expect |
|---|---|---|
| `creates two assets and one pair` | buyer credit 5000, both compartments empty capacity 10 | one asset in account 1's compartment and one in account 2's, both `TemplateId` 1112800; `ring.GetByCharacterId` returns one row for 42 and one for 77 sharing a `PairId`; buyer credit == 2500; a `RING_PURCHASED` event |
| `partner locker full creates neither` | account 2's compartment at capacity | zero assets in **both** compartments; zero ring rows; credit still 5000; `ERROR` with `Operation == "FRIENDSHIP"` |
| `insufficient funds creates neither` | credit 100 | zero assets; zero ring rows; `ERROR` reason `"NOT_ENOUGH_CASH"` |
| `unknown commodity creates neither` | SN 99999 | zero assets; zero ring rows; `ERROR` with `Operation == "FRIENDSHIP"` |
| `couple type is recorded distinctly` | `RingType` `"COUPLE"` | both ring rows have `Type() == ring.TypeCouple`; the failure operation key is `"COUPLE"` |
| `replay creates one pair` | rerun with the same `TransactionId` | still exactly two assets and two ring rows; credit still 2500 |
| `the pair is queryable by character id` | after the success case | `ring.GetByCharacterId(db, tenant, 42)[0].PartnerCharacterId() == 77` (FR-RING-7) |

The "creates neither" cases are the whole point — a half-created pair is the failure this domain's placement exists to prevent.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./cashshop/... -run TestPurchaseRing -v
```

Expected: compile failure — `undefined: PurchaseRingAndEmit`.

- [ ] **Step 3: Implement the transaction**

Ledger claim → resolve commodity → resolve buyer and partner characters and their compartments → check **both** compartments have room → debit the buyer's wallet by the commodity `Price` → create the buyer's asset → create the partner's asset (carrying `GiftFrom`/`GiftMessage` from Task 13's columns, since the partner receives it with the sender's message, FR-RING-3) → `ring.CreatePair` with both asset ids → `purchaserecord.Record` for the buyer → emit. Rejections use `ErrorOperationCouple` or `ErrorOperationFriendship` from `RingType`.

- [ ] **Step 4: Add the REST surface**

`GET /rings?filter[characterId]=N` (paginated, tenant-scoped) and `GET /rings/{ringId}`, mounted in `main.go`. This closes PRD §5.4's open question about the owning service.

- [ ] **Step 5: Register the command handler and producer**

- [ ] **Step 6: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(cashshop): implement atomic ring pair purchase"
```

---

## Task 20: BUY_COUPLE (29/31) and BUY_FRIENDSHIP (35/37) — `atlas-channel` side

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` — mirror Task 19's types
- `services/atlas-channel/atlas.com/channel/cashshop/producer.go` — add `RequestRingPurchaseCommandProvider`
- `services/atlas-channel/atlas.com/channel/cashshop/processor.go` — add `RequestRingPurchase`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — add `handleStatusEventRingPurchased`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring.go` — **new file**; `handleBuyCouple` and `handleBuyFriendship`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_ring_test.go` — **new file**

Also touched: `socket/handler/cash_shop_operation.go` — the `BUY_COUPLE` arm at `:164-169` and the `BUY_FRIENDSHIP` arm at `:180-186`.

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

`cash_shop_ring_test.go`. `TestCashShopRingResultBodies` with options:

```go
options := map[string]interface{}{
	"operations": map[string]interface{}{
		cashcb.CashShopOperationCoupleSuccess:     float64(152),
		cashcb.CashShopOperationFriendshipSuccess: float64(162),
	},
}
```

| subtest | constructor + args | expect first byte |
|---|---|---|
| `couple` | `cashcb.CashShopCoupleDoneBody(item, "Partner", 1112000, 1)` | 152 |
| `friendship` | `cashcb.CashShopFriendshipDoneBody(item, "Partner", 1112800, 1)` | 162 |

`item` is a `cashcb.CashInventoryItem` built with fixed field values; read `NewCoupleDone`/`NewFriendshipDone`'s `Encode` in `shop_operation_body.go` and assert the whole byte slice for each, not just the mode byte.

`TestRingTypeForOperation` — the pure mapper the two handlers share: `CashShopOperationBuyCouple -> "COUPLE"`, `CashShopOperationBuyFriendship -> "FRIENDSHIP"`.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'TestCashShopRing|TestRingTypeForOperation' -v
```

Expected: compile failure — `undefined: ringTypeForOperation`.

- [ ] **Step 3: Implement both arms**

One shared `handleRingPurchase(l, ctx, wp)(s, ringType string, isPoints bool, currency uint32, spw string, birthday uint32, serialNumber uint32, name string, message string)`, with the two exported handlers as thin adapters that unpack their own struct — the two serverbound bodies differ only by `ShopOperationBuyFriendship`'s v48-only `flag` byte, which `shop_operation_buy_friendship.go:26,83-90` already documents as a client-hard-coded constant absent on v83+ and which therefore needs no handling.

In order: `verifySecondaryCredential` (FR-RING-5) → resolve the partner by name, rejecting an unknown name (FR-RING-2) → reject a partner on the sender's own account → mint `uuid.New()` → `RequestRingPurchase(...)` with `resolvePurchaseCurrency(sp.IsPoints(), sp.Currency())`. Failures announce `cashcb.CashShopCoupleFailedBody` / `cashcb.CashShopFriendshipFailedBody` with a `giftRejectionReason`-derived key.

Apply `derivation.md` D4a for `option` on both structs, as in Task 17.

- [ ] **Step 4: Add the consumer handler**

`handleStatusEventRingPurchased` builds the `cashpkt.CashInventoryItem` from the asset named in `e.Body.AssetId` (same projection as `handleStatusEventPurchase`) and picks `CashShopCoupleDoneBody` or `CashShopFriendshipDoneBody` from `e.Body.RingType`.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel
git commit -m "feat(channel): implement BUY_COUPLE and BUY_FRIENDSHIP"
```

---

## Task 21: Derive the equip-slot extension facts

The purchase half of ENABLE_EQUIP_SLOT is ordinary Skeleton A. The **effect** half is not, and this task settles it before any code is written.

### Files

- `docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md` — **new file**; the deliverable
- `libs/atlas-constants/inventory/slot/constants.go` — read-only; there is **no** extra-pendant slot here today; the highest-numbered entries stop at `pet3ItemIgnore` (-48)
- `libs/atlas-packet/cash/clientbound/shop_operation_body.go` — read-only; `CashShopEnableEquipSlotExtSuccessBody(slotIndex uint16, days uint16)` at `:527`
- `libs/atlas-packet/cash/serverbound/shop_operation_enable_equip_slot.go` — read-only; the v83+ body is `pointType bool` + `serialNumber uint32`, no currency int (`:58-72`)
- `docs/reverse-engineering.md` — read-only

Module root: none (documentation only).

- [ ] **Step 1: Derive E1 — the slot index**

From `CCashShop::OnEnableEquipSlotExt`'s result handler in the GMS v95.1 IDB: what does the client do with the `slotIndex uint16` it reads, and which equipped-inventory slot position does the extended pendant occupy? Record the address, the decompilation, and the numeric slot value.

- [ ] **Step 2: Derive E2 — how the extension survives a relog**

How does the client learn the extension is active after a channel change or relog? A field on `GW_CharacterStat`? A re-sent `CashShopEnableEquipSlotExtSuccess`? An avatar-look consequence? FR-SLOT-3 cannot be satisfied without this answer.

- [ ] **Step 3: Decide the `libs/atlas-constants` consequence**

If E1 yields a concrete slot number, state whether `libs/atlas-constants/inventory/slot/constants.go` should gain an entry for it, and what it should be named. This is a shared-library change that ripples, so the decision belongs here rather than mid-implementation.

- [ ] **Step 4: Write `derivation-equip-slot.md`**

Same format as Task 1. Any answer that could not be established is written as **UNRESOLVED** with what was tried, and Task 23 lands the purchase + persistence + a typed failure for the un-derivable path — a partial arm with a stated reason, never an invented slot number (design §9).

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md
git commit -m "docs(task-240): derive the equip slot extension slot index and propagation"
```

---

## Task 22: Equip-slot extension domain in `atlas-character`

`atlas-character` has no pendant, slot-extension, or equipped-capacity concept anywhere today (verified by sweep at design time: zero matches for pendant/slotExt in the service).

### Files

- `services/atlas-character/atlas.com/character/equipslot/entity.go` — **new file**
- `services/atlas-character/atlas.com/character/equipslot/model.go` — **new file**
- `services/atlas-character/atlas.com/character/equipslot/administrator.go` — **new file**
- `services/atlas-character/atlas.com/character/equipslot/processor.go` — **new file**
- `services/atlas-character/atlas.com/character/equipslot/administrator_test.go` — **new file**
- `docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md` — read-only (new file, written by Task 21); E1 names the slot index

Also touched: `services/atlas-character/atlas.com/character/main.go` — the migration list. Locate it with `grep -n SetMigrations services/atlas-character/atlas.com/character/main.go` rather than assuming a line number.

Patterns to copy: the nearest small tenant-scoped domain already in `atlas-character`; identify it with `ls services/atlas-character/atlas.com/character/` and copy the file-for-file layout of one that has `entity.go`/`model.go`/`administrator.go`/`processor.go` and nothing else.

Module root: `services/atlas-character/atlas.com/character`.

**Interfaces produced:**

```go
func Migration(db *gorm.DB) error

// Extend upserts the character's slot extension. It EXTENDS rather than
// duplicates (FR-SLOT-4): the new expiry is max(now, existing) + period, so
// buying it again while it is still active adds to the remaining time instead
// of resetting or creating a second row.
func Extend(db *gorm.DB, tenantId uuid.UUID, characterId uint32, slotIndex int16, period time.Duration) (time.Time, error)

// GetActive returns the character's currently-active extensions, i.e. those
// whose ExpiresAt is in the future. An expired row is not returned and is not
// deleted -- the history is kept.
func GetActive(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error)
```

- [ ] **Step 1: Write the failing test**

`administrator_test.go`, in-memory sqlite with `Migration`. Tenant `A`, character 42, slot index from E1 (call it `S`).

| subtest | action | expect |
|---|---|---|
| `first purchase creates` | `Extend(db, A, 42, S, 30*24*time.Hour)` | returns an expiry ~30 days out; `GetActive(db, A, 42)` returns one model with `SlotIndex() == S` |
| `second purchase extends, not duplicates` | `Extend(db, A, 42, S, 30*24*time.Hour)` again | returns an expiry ~60 days out; `GetActive` still returns exactly **one** model (FR-SLOT-4) |
| `an expired extension restarts from now` | insert a row with `ExpiresAt` 10 days in the past, then `Extend(..., 30*24*time.Hour)` | the new expiry is ~30 days from now, not ~20 |
| `an expired extension is not active` | a row with `ExpiresAt` in the past and no subsequent extend | `GetActive` returns an empty slice |
| `another character is separate` | `GetActive(db, A, 99)` | empty slice, no error |
| `another tenant is separate` | `GetActive(db, uuid.New(), 42)` | empty slice, no error |

Compare expiries with a tolerance (a minute is plenty) rather than exact equality — the implementation calls `time.Now()`.

- [ ] **Step 2: Run the test and watch it fail**

```sh
cd services/atlas-character/atlas.com/character && go test ./equipslot/... -v
```

Expected: build failure — package does not exist.

- [ ] **Step 3: Write the entity, model, administrator, processor**

```go
// Entity records one purchased equipped-inventory slot extension.
//
// SlotIndex is the client-facing slot the extension unlocks; its value comes
// from the GMS v95.1 IDB (see derivation-equip-slot.md E1) and is NOT invented.
type Entity struct {
	Id          uuid.UUID `gorm:"primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_equipslot_unique"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_equipslot_unique"`
	SlotIndex   int16     `gorm:"not null;uniqueIndex:idx_equipslot_unique"`
	ExpiresAt   time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (e Entity) TableName() string { return "character_equip_slot_extensions" }
```

- [ ] **Step 4: Run the tests**

```sh
cd services/atlas-character/atlas.com/character && go build ./... && go test ./equipslot/... -v
```

Expected: PASS.

- [ ] **Step 5: Apply E2**

If E2 established how the client is told about an active extension across a relog, implement that propagation now — the field on the character stat payload, the re-sent packet, or whatever E2 named. If E2 came back **UNRESOLVED**, the persistence above still lands and `context.md` records that FR-SLOT-3 ("survives a channel change") is not satisfied and why. Do not fabricate a propagation mechanism.

- [ ] **Step 6: Add the migration and commit**

```bash
git add services/atlas-character/atlas.com/character
git commit -m "feat(character): persist purchased equip slot extensions"
```

---

## Task 23: ENABLE_EQUIP_SLOT (mode 9/10) end to end

### Files

- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` — add `CommandTypeRequestEquipSlotIncrease`, `RequestEquipSlotIncreaseCommandBody`, `StatusEventTypeEquipSlotIncreased`, `EquipSlotIncreasedBody`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/equipslot.go` — **new file**; the purchase transaction
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/equipslot_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` — replace the log-only arm at `:123-129`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — add `handleStatusEventEquipSlotIncreased`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_equip_slot_test.go` — **new file**

Also touched (one line each): both `kafka/message/cashshop/kafka.go` mirrors, `kafka/producer/cashshop/producer.go`, `kafka/consumer/cashshop/consumer.go` (atlas-cashshop), `cashshop/producer.go` + `cashshop/processor.go` (channel).

Module roots: `services/atlas-cashshop/atlas.com/cashshop`, `services/atlas-channel/atlas.com/channel`.

**Currency (design §9, verified):** the v83+ wire body is `pointType bool` + `serialNumber uint32` with no currency int, so the channel requests with `resolvePurchaseCurrency(sp.PointType(), 0)` — a points buy steers to wallet currency 2, a credit buy to 0, matching `cashshop/processor.go:118-127`.

- [ ] **Step 1: Write the failing test**

`cashshop/equipslot_test.go`. Buyer character 42, account 1, credit 5000. Commodity SN 70000 → `Price` 4000, `Period` 30 (days).

| subtest | fixture | expect |
|---|---|---|
| `charges and emits the slot and duration` | as above | credit == 1000; an `EQUIP_SLOT_INCREASED` event with `Days` 30 and `SlotIndex` from E1 |
| `insufficient funds charges nothing` | credit 100 | credit still 100; `ERROR` with `Operation == "ENABLE_EQUIP_SLOT"`, reason `"NOT_ENOUGH_CASH"` |
| `unknown commodity charges nothing` | SN 99999 | credit unchanged; `ERROR` with `Operation == "ENABLE_EQUIP_SLOT"` |
| `replay charges once` | rerun with the same `TransactionId` | credit still 1000 |

`cash_shop_equip_slot_test.go` — `TestCashShopEnableEquipSlotExtSuccessBodyEncodes` over `cashcb.CashShopEnableEquipSlotExtSuccessBody(slotIndex, days)` with options `{"operations": {cashcb.CashShopOperationEnableEquipSlotExtSuccess: float64(117)}}`: assert the first byte is 117 and the remaining bytes match `NewEnableEquipSlotExtSuccess`'s `Encode` for the E1 slot index and days 30.

- [ ] **Step 2: Run the tests and watch them fail**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go test ./cashshop/... -run TestPurchaseEquipSlot -v
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run TestCashShopEnableEquipSlot -v
```

Expected: compile failure, then a byte assertion failure.

- [ ] **Step 3: Implement the atlas-cashshop transaction**

Ledger claim → resolve commodity → debit wallet → call `atlas-character`'s equip-slot extension over the service's existing character client (add an `ExtendEquipSlot` request alongside the existing calls in `services/atlas-cashshop/atlas.com/cashshop/character/`) → `purchaserecord.Record` → emit `EquipSlotIncreasedBody`. Rejections use `ErrorOperationEnableEquipSlot`.

- [ ] **Step 4: Replace the channel arm and add the consumer handler**

The arm decodes, mints `uuid.New()`, and calls `RequestEquipSlotIncrease(s.CharacterId(), transactionId, resolvePurchaseCurrency(sp.PointType(), 0), sp.SerialNumber())`. The consumer announces `cashcb.CashShopEnableEquipSlotExtSuccessBody(e.Body.SlotIndex, e.Body.Days)`.

If E1 came back **UNRESOLVED**, the arm still lands the purchase and the persistence, and the consumer announces `cashcb.CashShopEnableEquipSlotExtFailedBody("unknown_error")` with a logged warning naming the unresolved derivation — the player is not charged silently and the dialog is not left wedged. Record it in `context.md`.

- [ ] **Step 5: Run the tests**

```sh
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
cd services/atlas-character/atlas.com/character && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop services/atlas-channel services/atlas-character
git commit -m "feat(channel): implement ENABLE_EQUIP_SLOT end to end"
```

---

## Task 24: Coverage, cross-service seam trace, and repo-wide verification

### Files

- `docs/tasks/task-240-cash-shop-stub-operations/coverage-manifest.yaml` — **new file**; the op x version surface this task claims
- `docs/packets/audits/status.json` — regenerated by `packet-audit`, not hand-edited
- `docs/packets/audits/STATUS.md` — regenerated
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` — read-only for the final sweep
- `docs/tasks/task-227-cash-name-change-world-transfer/coverage-manifest.yaml` — read-only; the manifest format to copy

Module root: repo root.

- [ ] **Step 1: Sweep every arm**

Read `cash_shop_operation.go` end to end and confirm no arm is a decode-log-return. Run:

```sh
grep -n 'l.Infof' services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go
grep -rn 'CashShopOperationBuyOtherPackage' services/atlas-channel/atlas.com/channel/
```

Expected: no `l.Infof`-and-return arm remains; `CashShopOperationBuyOtherPackage` appears at its declaration **and** at a dispatch site.

- [ ] **Step 2: Trace each new event into its consumer by hand**

For each of `GIFT_PURCHASED`, `PACKAGE_PURCHASED`, `RING_PURCHASED`, `LOCKER_REBATED`, `EQUIP_SLOT_INCREASED`, and the extended `ERROR`: confirm the producer's field names and the consumer's field reads agree, and that a test asserts the **new** contract (CLAUDE.md "Done means verified" — a green build cannot see a cross-service seam defect). Fix any disagreement here rather than filing it.

- [ ] **Step 3: Write the coverage manifest and promote the matrix**

Declare every op x version this task touched. Run `packet-audit` per `docs/packets/audits/VERIFYING_A_PACKET.md` for any codec added in Tasks 8 and 17, and record the `n-a` proof for gms_95 `CashShopOpen` if Task 1's D1 came back negative.

- [ ] **Step 4: Run the flagless verification gate**

```sh
tools/verify.sh
```

Expected: exit 0. Only the flagless invocation counts — `--quick`/`--no-docker` skip the bake and `-race`.

- [ ] **Step 5: Update `context.md` with the final state**

Every deliberately-deferred item and every UNRESOLVED derivation, each with its reason and the FR it leaves unsatisfied.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-240-cash-shop-stub-operations docs/packets/audits
git commit -m "docs(task-240): coverage manifest, matrix promotion, and final context"
```

- [ ] **Step 7: Code review**

Run the code-review step before opening the PR — `backend-guidelines-reviewer` over the changed Go packages and `plan-adherence-reviewer` over this plan. Do not open the PR without it (CLAUDE.md "Never open a PR without code review").
