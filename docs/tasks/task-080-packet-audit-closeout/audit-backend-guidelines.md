# Backend Audit — task-080-packet-audit-closeout (DOM/SUB/SEC)

- **Worktree:** `<repo-root>/.worktrees/task-080-packet-audit-closeout`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-04
- **Scope:** Go changes on branch vs `main` (excluding generated `docs/packets/audits/**` and `docs/tasks/**`)
- **Build:** PASS (all 4 modules)
- **Tests:** PASS (all changed packages)
- **go vet:** PASS (all 4 modules)
- **gofmt:** FAIL — 6 changed files not gofmt-clean (see Finding 1)
- **Overall:** NEEDS-WORK (one Minor gofmt finding; no blocking/architectural violations)

## Build & Test Results

All four touched Go modules build, vet, and test clean:

| Module | go build | go vet | go test |
|--------|----------|--------|---------|
| `libs/atlas-packet` | PASS | PASS | PASS |
| `services/atlas-maps/atlas.com/maps` | PASS | PASS | PASS (`atlas-maps/mist` ok) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS (cashshop, kafka/consumer/mist, socket/handler ok) |
| `tools/packet-audit` | PASS | PASS | PASS |

## Package Classification

The task-080 Go changes are NOT a conventional CRUD domain. They split into:

- **Packet codecs** (`libs/atlas-packet/**`) — immutable structs with `Encode`/`Decode`, no `model.go`/`entity.go`/`rest.go`. The DOM CRUD checklist (DOM-01..DOM-22) does not apply; the relevant rules are immutability, region-dispatch symmetry, and DOM-21.
- **`services/atlas-maps/.../mist`** — domain package (`model.go` + builder + `processor.go` + `producer.go` + Kafka). DOM immutability/builder/Kafka rules apply.
- **`services/atlas-channel/.../{cashshop,kafka/consumer/mist,socket/handler}`** — processor + consumer + socket handlers. Processor pattern and handler→processor layering apply.
- **`tools/packet-audit/**`** — internal CLI analyzer/diff tool, classified **Support**. DOM checklist skipped.

## Checklist Results

| ID / Rule | Status | Evidence |
|-----------|--------|----------|
| Immutable model (private fields + getters) — mist | PASS | `services/atlas-maps/atlas.com/maps/mist/model.go:16-38` private fields; getters `:41-158`; no public mutable fields |
| Builder pattern — mist | PASS | `mist/model.go:191-312` `Builder` + `NewBuilder` + fluent setters + `Build()`; new `SetType`/`Type()` (B1.1a) wired symmetrically `:244-248,86-88,298` |
| Immutable packet structs (private fields + getters) | PASS | e.g. `cash/serverbound/shop_operation_buy.go:16-30`, `field/clientbound/affected_area_created.go:30-79`, `login/serverbound/request.go:25-58` — all private fields, value-receiver getters |
| No test-only constructors / `*_testhelpers.go` | PASS | `git diff main...HEAD --name-only` shows no `*_testhelpers.go`; tests use builders/constructors |
| Processor interface+impl, `NewProcessor(l, ctx)` — cashshop | PASS | `cashshop/processor.go:17-42` `Processor` iface + `ProcessorImpl` + `NewProcessor(l, ctx)` |
| Processor interface+impl, `NewProcessor(l, ctx)` — mist (maps) | PASS | mist processor present (`mist/processor.go`), emits via `message.Emit` `:75,94` |
| Pure-vs-side-effecting split (Kafka) — mist producer | PASS | `mist/producer.go:14,43` build providers only; side-effecting emit via `message.Emit(p.p)` + `buf.Put` in `mist/processor.go:75-76,94-95` |
| Kafka `message.Buffer`/`Emit` pattern where touched | PASS | `mist/processor.go:75-96` uses `message.Emit(p.p)(func(buf *message.Buffer) error { return buf.Put(...) })`; providers use `producer.SingleMessageProvider` (`producer.go:39,55`) |
| Handler → processor layering (no direct provider/db in handler) | PASS | `socket/handler/hired_merchant_operation.go:36-37` calls `merchant.NewProcessor(...).GetByCharacterId`; `quest_action.go` and `npc_continue_conversation.go` call only `*.NewProcessor(...)` methods; no `db.Create/Save/Delete`, no `os.Getenv` |
| Multi-tenancy `tenant.MustFromContext(ctx)` in codecs | PASS | `shop_operation_buy.go:42,73`; `effect_weather.go:38,68`; `login/serverbound/request.go:70,87`; `chat/serverbound/multi.go:53,70`; `affected_area_created.go:88`; consumer `kafka/consumer/mist/consumer.go:82,108` |
| **Region/version guard symmetry (Encode ↔ Decode)** | PASS | Verified symmetric on every dispatching codec: `shop_operation_buy.go` GMS≥87 branch `:57-62`↔`:87-92`; `shop_operation_gift.go` v95/v87 branches `:59-69`↔`:92-102`; `shop_operation_rebate_locker_item.go:52-57`↔`:81-86`; `effect_weather.go:49-65`↔`:78-94`; `login/serverbound/request.go:78-81`↔`:95-98`; `chat/serverbound/multi.go:54-66`↔`:71-83`. `affected_area_created.go` is clientbound-only (Encode only, `:91,104`) — no Decode required |
| Region-dispatch idiom ≤2 nested guards (no 3rd nested if) | PASS | Top-level `if JMS {encodeJMS} else {encodeGMS}`; `encodeGMS` uses sibling (non-nested) version guards, e.g. `shop_operation_gift.go:58-70` has two independent `if` blocks, not nested. Max depth 2 |
| DOM-21 (reuse `libs/atlas-constants`, no reinvented domain types/consts) | PASS | mist model imports `world`/`channel`/`_map`/`field` from atlas-constants (`mist/model.go:6-9`); event body reuses `world.Id`/`channel.Id`/`_map.Id` (`kafka/message/mist/kafka.go:33-35,61-66`). No reinvented item-id/inventory/world-width constants. `resolvePurchaseCurrency` const `walletCurrencyMaplePoints=2` (`cashshop/processor.go:105`) is a wallet wire-protocol code, not an atlas-constants domain type — no equivalent exists |
| No new `// TODO` / stubs introduced by task-080 | PASS (with pre-existing note) | See Finding 2 — all `// TODO` markers in `npc_continue_conversation.go` and `cashshop/processor.go` pre-date the branch; B2.2 (`hired_merchant_operation.go`) replaced a stub with a real processor call (`:36-49`), leaving no TODO |
| Error handling (no swallowed errors on side-effecting paths) | PASS | Consumer broadcasters log on error (`kafka/consumer/mist/consumer.go:64-66,72-74`); handlers log `WithError` (`quest_action.go:45-47` etc.). The `_ = npc.NewProcessor(...)` discards in `npc_continue_conversation.go:57-76` are pre-existing (present on `main`) |
| Dead code / naming | PASS | No new dead code introduced; `EntrustedShopOperationMode11` (`merchant/clientbound/operation.go:19`) is an intentionally-registered named constant, documented `:14-18`. `FreeFormNotice` writes constant `true` and discards on decode (`:257,266`) — benign constant field, documented |
| SEC (auth) | N/A | `login/serverbound/request.go` is a packet codec only; `partnerCode` is read-and-discard (`:96-97`, documented `:21-24`). No JWT validation, token revocation, redirect, or secret handling in scope |
| **gofmt clean** | **FAIL** | See Finding 1 |

## Findings

### Minor 1 — Six task-080 files are not gofmt-clean
`go vet` passes but `gofmt -l` flags formatting drift (manual edits left getter/struct/comment columns misaligned). CI's format gate may reject these.

Files and representative lines:
- `libs/atlas-packet/cash/serverbound/shop_operation_buy.go:26-30` (getter column alignment)
- `libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:26-30`
- `libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:26-30`
- `libs/atlas-packet/cash/serverbound/shop_operation_gift.go:30-35`
- `services/atlas-maps/atlas.com/maps/mist/model.go:17-18` (struct field `id`/`f` alignment)
- `tools/packet-audit/internal/atlaspacket/analyzer.go:42-44` and `:304-306` (struct comment + doc-comment tab indentation)

Fix: `gofmt -w` on the six files. Purely cosmetic; no behavioral impact.

### Minor 2 — Pre-existing `// TODO` markers carried through touched files (not introduced by task-080)
The no-TODO-in-deliverables rule is satisfied for task-080's new work, but task-080 edited lines adjacent to pre-existing TODOs:
- `services/atlas-channel/.../socket/handler/npc_continue_conversation.go:46,54,56,60,67,71,75` — TODOs ("handle quest in progress", "set return text") and a commented-out `//returnText := ""` (`:46`). Confirmed present on `main` (`git show main:...` identical TODOs). Task-080 B2.1 refactored only the `lastMessageType` discriminator into `bodyKindFor` (`:14-37,49-77`); it carried the existing TODOs/dead plumbing along rather than introducing them.
- `services/atlas-channel/.../cashshop/processor.go:120,165` — "identify correct compartment type" TODOs, also pre-existing on `main` (the task-080 diff added only `resolvePurchaseCurrency`).

Not a task-080 regression. Worth a follow-up cleanup ticket since the file was touched, but not blocking and not attributable to this branch.

## Summary

### Blocking (must fix)
- None. Build, vet, and all tests pass; no layer violations, no broken Encode/Decode symmetry, no DOM-21 reinvention, no security exposure.

### Non-Blocking (should fix)
- **Minor 1 (gofmt):** run `gofmt -w` on the 6 listed files before merge to keep CI's format gate green.
- **Minor 2 (pre-existing TODOs):** optional follow-up to retire the carried-over `// TODO`/`//returnText` lines in `npc_continue_conversation.go` and the compartment-type TODOs in `cashshop/processor.go`. Pre-existing, not a task-080 defect.
