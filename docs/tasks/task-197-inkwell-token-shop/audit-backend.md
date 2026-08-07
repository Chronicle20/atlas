# Backend Audit — atlas-npc-shops (task-197-inkwell-token-shop)

- **Service Path:** services/atlas-npc-shops/atlas.com/npc
- **Scope:** `shops/token.go` (new), `shops/token_test.go` (new), `shops/processor.go` (1-line
  modification), `services/atlas-npc-shops/docs/domain.md` (doc update)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-06
- **Build:** PASS
- **Tests:** `go test ./shops/... -run 'TestPlanTokenSpend|TestBuyWithTokens' -v` — 9 + 2 + 7 = 18
  subtests, all PASS (re-run directly, see evidence below). Prior full-suite gate report
  (`.superpowers/sdd/plan/task-9-gates-report.md`, Step 1) additionally shows `atlas-npc/shops`
  passing under `go test -race ./...` and `go vet ./...`/`go build ./...` clean.
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-npc-shops/atlas.com/npc
go test ./shops/... -run 'TestPlanTokenSpend|TestBuyWithTokens' -v
...
--- PASS: TestPlanTokenSpend (0.00s)                              (9 subtests, all PASS)
--- PASS: TestBuyWithTokensSufficientBalanceSpansSlots (0.00s)
--- PASS: TestBuyWithTokensQuantityMultipliesCost (0.00s)
--- PASS: TestBuyWithTokensRefusals (0.00s)                        (7 subtests, all PASS)
PASS
ok  	atlas-npc/shops	0.014s
```

The prior gate report flagged two now-resolved items: `services/atlas-npc-shops/docs/domain.md`
was stale (still described the token path as "not implemented") at the time of that report. HEAD
(`52a878b13`) includes a subsequent commit that fixes this — verified below (see "Documentation").
The gate report's Step 11 file-count concern (3 vs. expected 4 npc-shops files) is resolved for the
same reason: `docs/domain.md` is now part of the diff.

## Domain Checklist Results — `shops` (domain package, `model.go` present)

Scope note: `shops` is a pre-existing domain package; only the checks touched by this diff
(`token.go`, `token_test.go`, the one-line `processor.go` change) are evaluated. Checks about
files untouched by this branch (`rest.go`, `resource.go`, `provider.go`, `entity.go`,
`administrator.go`, `builder.go`) are marked N/A — not because the package is exempt from them,
but because this audit's scope is the diff and those files carry no changes to grade.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS (pre-existing, unaffected) | `shops/processor.go:70` — `l logrus.FieldLogger`; `buyWithTokens` is a method on the same `*ProcessorImpl` receiver, `shops/token.go:81`. |
| DOM-09 | Transform errors handled | N/A | No `rest.go` change in this diff. |
| DOM-12 | No `os.Getenv()` in handlers | N/A | No `resource.go` change in this diff. |
| DOM-13/14/15 | No cross-domain logic / provider calls / direct entity creation in handlers | N/A | No `resource.go` change in this diff. |
| DOM-17 | Domain error → HTTP status mapping | N/A | No `resource.go` change in this diff. |
| DOM-20 | Table-driven tests | PASS | `shops/token_test.go:46-52` (`TestPlanTokenSpend`, `tests := []struct{...}` + `t.Run`); `shops/token_test.go:337-346` (`TestBuyWithTokensRefusals`, same pattern). |
| DOM-21 | No duplication of atlas-constants types | PASS | `shops/token.go:12-13` imports `github.com/Chronicle20/atlas/libs/atlas-constants/inventory` and `.../item` directly; `shops/token.go:94,99` call `inventory.TypeFromItemId(item.Id(...))` — the shared classifier, not a reimplementation. `tokenDraw` (`shops/token.go:19-22`, a private `{slot, quantity}` pairing local to the withdrawal plan) has no equivalent in `libs/atlas-constants` — it is not an id/classification type. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (N/A in substance) | `shops/token_test.go` never calls `message.Emit(...)` or a `*AndEmit` method. `buf := message.NewBuffer()` (`token_test.go:264`) and `p.buyWithTokens(buf)(...)` only reach `mb.Put(...)`, which appends to an in-memory map (`kafka/message/message.go:24-33`) with no network I/O. `message.Emit` (the function that actually calls the producer, `kafka/message/message.go:46`) is never invoked by any test in this diff, so no producer stub is required. |
| DOM-26 | Goroutines via `routine.Go` | PASS | No `go` statement of any kind in `shops/token.go` or `shops/token_test.go` (grep confirms zero matches). |
| Error-handling coherence (user-specified focus) | PASS | Every guard in `buyWithTokens` that refuses without touching inventory returns `mb.Put(...)` and `nil` as the function's logical outcome (`shops/token.go:87,91,97,102,111,118,123` — misconfiguration/insufficient-balance/no-free-slot), matching the pre-existing `Buy()` branches (`processor.go:399,412,418,425,436,440,445,462,466,472` use the identical `return mb.Put(...)` shape). Only the two `p.compP.Request*Item` calls propagate a raw `err` (`shops/token.go:127-129,131-133`) — that path is infra failure (Kafka buffer/produce-side), never a business refusal, exactly matching the existing rechargeable/meso branches' treatment of `RequestChangeMeso`/`RequestCreateItem` errors (`processor.go:447-449,450-452,474-476,477-479`). Coherent, not a new deviation. |

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in `processor.go` or a `processor_<group>.go` split | **FAIL (Important)** | `func (p *ProcessorImpl) buyWithTokens(...)` is defined at `shops/token.go:81`, not in `processor.go` and not in a `processor_<group>.go`-named file. The guideline's own FAIL criterion lists exactly this shape as prohibited: "FAIL (Important) if any `ProcessorImpl` method or the interface is in a NON-processor-named file: ... or a bare topic name like `custody.go`/`register.go`." `token.go` is that pattern — a bare topic name holding a `ProcessorImpl` method. The repo has an established, correctly-named alternative for exactly this situation (a large `Buy()`/purchase-path split): `services/atlas-mts/atlas.com/mts/listing/processor_custody.go`, `services/atlas-mts/atlas.com/mts/holding/processor_custody.go`, `services/atlas-mts/atlas.com/mts/wish/processor_register.go`, `services/atlas-inventory/atlas.com/inventory/compartment/processor_accommodation.go` — all four use `processor_<group>.go`. `token.go` should have been named `processor_token.go` (or `buyWithTokens` moved into `processor.go` directly) to conform. `tokenDraw` and `planTokenSpend` (`shops/token.go:19-22,32-63`) are pure, non-`ProcessorImpl` helpers with no assigned file in the responsibilities table; they are not independently a violation, but their co-location with `buyWithTokens` in a bare-topic file does not cure FILE-01. |
| FILE-02/03/04/05 | RestModel/rest.go, requests.go, entity.go, builder.go/administrator.go/provider.go/state.go placement | N/A | No files of these kinds were added or modified in this diff. |
| FILE-06 | No package-named catch-all file bundling ≥2 responsibilities | PASS (narrowly) | `token.go` bundles a `ProcessorImpl` method with pure helpers, which is the FILE-01 violation above, but it does not additionally bundle a second *table-listed* responsibility (no `RestModel`, no `requests.go`-style client call, no `Entity`, no `Builder`, no administrator write, no provider read, no state enum) — so it does not independently trip the "≥2 responsibilities" FILE-06 threshold on top of FILE-01. |

## Sub-Domain / Support-Package / External-HTTP-Client Checklists

Not applicable — `shops` has `model.go` (full domain package), and this diff makes no external
HTTP call (`buyWithTokens` takes an already-resolved `character.Model` parameter rather than
calling `character.Processor.GetById`, per design D7 — no `requests.GetRequest`/`PostRequest`
call sites were added).

## Documentation

`services/atlas-npc-shops/docs/domain.md:170,191` (diff `31c7a664f..52a878b13`) was rewritten to
describe the three-branch `Buy()` structure (rechargeable / meso / token) and the token guard
order, replacing the stale "Token-priced commodities ... are not implemented and emit a
GENERIC_ERROR_WITH_REASON status event" sentence. Verified no residual `"not implemented"` or
`TokenItem`-TODO string remains anywhere under `services/atlas-npc-shops` or `services/atlas-npc-shops/atlas.com/npc/shops` (`grep -n 'not implemented\|TokenItem' ... ` — zero hits). This
resolves the gap the prior gate report (`task-9-gates-report.md` Steps 7 and 11) flagged; that
report predates the doc-fix commit.

## Deliberate Design Decisions — audited on the merits

- **`buyWithTokens` takes `character.Model` as a parameter, no HTTP seam in `Buy()`.** Confirmed:
  `Buy()` still resolves the character via `p.charP.GetById(p.charP.InventoryDecorator)(characterId)`
  (`processor.go:415`) before any of the three branches, and passes the resolved `c` into
  `buyWithTokens(mb)(c, cm, itemTemplateId, quantity)` (`processor.go:484`). No test exercises
  `Buy()`/`BuyAndEmit()` directly for the token path — acceptable per the documented seam rationale
  (testing-guide.md's "Processors — Test pure and AndEmit forms separately" is satisfied at the
  `buyWithTokens` level, which is the newly-added surface; `Buy()`'s dispatch to it is a single
  line with nothing left to unit-test in isolation).
- **No `discountPrice` parameter.** Confirmed structurally: `buyWithTokens`'s signature
  (`shops/token.go:81`) has no `discountPrice` field, and no local variable of that name appears
  anywhere in `token.go`. `Buy()` still receives `discountPrice` in its own signature
  (`processor.go:386-387`) but never forwards it into the token branch call
  (`processor.go:484` — `p.buyWithTokens(mb)(c, cm, itemTemplateId, quantity)`, four args, no
  discount). Confirmed correct.
- **Branch ordering (rechargeable → meso → token) unchanged.** `processor.go:421` (`if
  item.IsRechargeable(...)`) and `processor.go:457` (`if cm.MesoPrice() > 0`) are untouched by the
  diff (only the final `return` at `processor.go:484` changed) — confirmed via `git diff`, a
  1-line replacement.
- **Free-slot probe precedes consumption.** `shops/token.go:121-124` (`NextFreeSlot()` check) runs
  before the destroy/create loop at `shops/token.go:126-133`. Matches the meso path's ordering
  (`processor.go:469-473` before `474-479`).
- **Multi-slot consumption in ascending slot order, one DESTROY per slot.** `planTokenSpend`
  (`shops/token.go:32-63`) sorts `matching` by `Slot()` ascending (`token.go:40-42`) before
  building draws; `buyWithTokens`'s loop (`token.go:126-130`) issues one
  `RequestDestroyItem` call per draw in that order. Verified by test
  `TestBuyWithTokensSufficientBalanceSpansSlots` (`token_test.go:262-307`), which asserts slot 3
  destroyed before slot 7 despite being constructed in reverse order in the fixture
  (`token_test.go:266-269`).

## Summary

### Blocking (must fix)

- **FILE-01** — `buyWithTokens` (`services/atlas-npc-shops/atlas.com/npc/shops/token.go:81`) is a
  `*ProcessorImpl` method living in a bare-topic-named file. Per the file-responsibilities
  checklist this must live in `processor.go` or be renamed to the established
  `processor_<group>.go` split convention (e.g. `processor_token.go`), matching the precedent at
  `services/atlas-mts/atlas.com/mts/listing/processor_custody.go`,
  `services/atlas-mts/atlas.com/mts/holding/processor_custody.go`,
  `services/atlas-mts/atlas.com/mts/wish/processor_register.go`, and
  `services/atlas-inventory/atlas.com/inventory/compartment/processor_accommodation.go`. The pure
  helpers `tokenDraw`/`planTokenSpend` may remain alongside it once renamed, or move with it —
  the violation is the file name/placement of the `ProcessorImpl` method, not the pure helpers'
  existence.

### Non-Blocking (should fix)

- None identified beyond the blocking item above. The `discountPrice`-absence, guard ordering,
  multi-slot consumption, error-handling coherence, DOM-21 reuse, and documentation-accuracy
  checks all pass with direct file:line evidence.
