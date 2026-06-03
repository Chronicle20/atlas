# Commerce-Domain Packet Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the audit pipeline shipped in task-027/028/066 to the **78 commerce-domain packet source files (~89 wire shapes)** in `libs/atlas-packet/{cash,interaction,inventory,storage}/{clientbound,serverbound}/`, ship wire-bug + template fixes against GMS v95 IDA, re-verify across v83/v87/JMS v185, and confirm no regression in the login (task-027), character (task-028), or social (task-066) audits.

**Architecture:** Phase 0 surveys `libs/atlas-packet/` for non-`Encode`/`Write` foreign-encoder methods (`EncodeBytes`, `EncodeEntry`), confirms `model.Asset.Encode` registration, and — if survey supports it — extends `tools/packet-audit/internal/atlaspacket/registry.go`'s method-name switch to recognise `EncodeBytes` and `EncodeEntry` (design §8 Option A) with fixtures. Phase 1 audits the 78 files in 4 sub-domain sub-tasks (storage → inventory → interaction → cash — small to big, durable to ephemeral, simple to cross-cutting). Phase 2 re-runs the audit against v83 / v87 / JMS v185 IDA, gating fixes by `Region/MajorVersion`. Phase 3 re-runs login + character + social audits as a regression gate. Phase 4 ships `post-phase-b.md`, full verification, code review, and PR.

**Tech Stack:** Go 1.24 (`go/parser` + `go/ast` for AST analysis already in `tools/packet-audit/internal/atlaspacket/`), `mcp__ida-pro__*` MCP tools for live IDA decompiles, `libs/atlas-socket` reader/writer + `libs/atlas-packet/test` `pt.Variants` for 4-variant round-trip tests, GORM JSON-blob columns in `services/atlas-configurations` for template overrides. No new runtime dependencies; this task ships audit reports + targeted code/template fixes.

---

## Conventions used by every task

- **Worktree.** All work happens in `.worktrees/task-067-commerce-domain-packet-audit/` on branch `task-067-commerce-domain-packet-audit`. Before *every* commit run `git rev-parse --show-toplevel` (must end with `/.worktrees/task-067-commerce-domain-packet-audit`) and `git branch --show-current` (must be `task-067-commerce-domain-packet-audit`); if either disagrees, STOP.
- **TDD cadence (analyzer/registry).** Test first → run-to-fail → minimal implementation → run-to-pass → commit. Steps below spell each phase out.
- **Verification cadence (registry changes).** `go test -race ./tools/packet-audit/...` clean before commit.
- **Verification cadence (atlas-packet edits).** `go test -race ./libs/atlas-packet/...` clean. Every encoder fix lands with a 4-variant test sweep covering GMS v28 / v83 / v95 + JMS v185 (use the existing `pt.Variants` pattern in `libs/atlas-packet/test/context.go`; v87 added when it surfaces during Phase 2).
- **No `*_testhelpers.go` files.** Use the project's Builder pattern. Per-test data is constructed via the existing `New<Packet>(...)` constructors in each clientbound/serverbound package.
- **No `reflect`, no new `interface{}` params, no benchmarks** in atlas-packet edits (design §1, inherited from task-028 §8 / task-066 §6).
- **Hard cap: 2 nested region/version guards per encoder** (design §9, carries from task-028 §7 / task-066). 3+ → STOP, log `_pending.md` row, do not refactor under audit cover.
- **No gitleaks bait.** Absolute paths like `/home/<user>/` must not appear in any file under `docs/packets/audits/gms_v95/{cash,interaction,inventory,storage}/`. Pre-PR check is mandatory (Task 12 Step 4).
- **Tracking sub-tasks vs PR-sized commits.** Phase 1 sub-tasks (Tasks 4–7) and Phase 2 sub-tasks (Tasks 8–10) are *tracking* units, not single commits. Each ❌ verdict inside a sub-task triggers an independent fix commit (one fix = one commit). A sub-task is "done" when every wire shape in its bucket has a verdict in `SUMMARY.md` and every ❌ has either a fix commit on this branch or a `_pending.md` row.
- **`_pending.md` row grouping.** Group deferrals by *cause*, not by *file*. One row per limitation with a sub-list of affected files (design §11, carries from task-066). One row per bare-handler family, not per handler.
- **One row per wire shape, not per file.** Design §3.1 itemises five multi-shape files (`cash/clientbound/shop_inventory.go` × 4, `cash/clientbound/shop_operation_result.go` × 4, `cash/clientbound/shop_item_moved.go` × 2, `storage/clientbound/error.go` × 3, `inventory/clientbound/change.go` × 4). SUMMARY.md uses the `<path>:<TypeName>` row template for these so reviewers can find each shape unambiguously. Expected total: ~89 wire-shape rows over 78 src files.

---

## Phase 0 — Foreign-encoder survey + (optional) registry extension (gate)

Three tasks. Exit when `go test -race ./tools/packet-audit/...` is clean and the predicted commerce sub-struct types resolve through the registry.

### Task 1: Foreign-encoder method-name survey (design §8 Phase 0a/0b)

The audit pipeline's `TypeRegistry.NewTypeRegistry` (`tools/packet-audit/internal/atlaspacket/registry.go:81-116`) only switches on the method names `Encode` and `Write`. Any sub-struct whose write helper is named differently is invisible to recursion. Design §8 predicts only `CashInventoryItem.EncodeBytes` exists; a real survey may find more (e.g. `EncodeEntry`, `EncodeForeign`). This task is the survey and locks the Phase 0c decision.

**Files:**
- Create: `docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md` (transient working memo; folded into `post-phase-b.md` in Phase 4).
- Modify (if survey surfaces new types): `docs/packets/audits/gms_v95/_pending.md`.

- [ ] **Step 1: Enumerate non-`Encode`/`Write` foreign-encoder methods.**

```bash
grep -rIn "func .*Encode\(\|func .*Write\b" libs/atlas-packet/ \
  | grep -v _test.go \
  | grep -oE "func \([^)]+\) [A-Z][A-Za-z]+\(" \
  | sort -u
```

Expected: A list of receiver methods. The interesting rows are the ones whose method name is NOT `Encode` or `Write` (i.e., not registry-visible). Filter to non-`Encode`/`Write`:

```bash
grep -rIn "func .*) [A-Z][A-Za-z]\+(" libs/atlas-packet/ \
  | grep -v _test.go \
  | grep -E "EncodeBytes|EncodeForeign|EncodeEntry"
```

As of design-time enumeration the expected hits are:
- `libs/atlas-packet/cash/clientbound/shop_inventory.go:25` — `CashInventoryItem.EncodeBytes` (flat `[]byte` return; not a closure).
- `libs/atlas-packet/inventory/change_entry.go:73` — `AddEntry.EncodeEntry` (closure return — `func(options map[string]interface{}) []byte`).
- `libs/atlas-packet/inventory/change_entry.go:103` — `QuantityUpdateEntry.EncodeEntry` (closure return).
- `libs/atlas-packet/inventory/change_entry.go:141` — `MoveEntry.EncodeEntry` (closure return).
- `libs/atlas-packet/inventory/change_entry.go:173` — `RemoveEntry.EncodeEntry` (closure return).
- `libs/atlas-packet/model/character_temporary_stat.go:575` — `CharacterTemporaryStat.EncodeForeign` (closure return; **already in scope of task-028; do not re-audit**).

If the survey returns ANY other hits, add them to the table in Step 2 — the plan must not silently skip a foreign-encoder method.

- [ ] **Step 2: Confirm auto-discovered registry coverage.**

```bash
grep -n "func .*Asset.*) Encode(\|func .*Encode\b" libs/atlas-packet/model/asset.go
```

Expected: `libs/atlas-packet/model/asset.go:164: func (m *Asset) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {` — present, auto-discovered.

```bash
grep -n "func .*) Encode(" libs/atlas-packet/cash/clientbound/*.go libs/atlas-packet/cash/serverbound/*.go libs/atlas-packet/interaction/clientbound/*.go libs/atlas-packet/interaction/serverbound/*.go libs/atlas-packet/inventory/clientbound/*.go libs/atlas-packet/inventory/serverbound/*.go libs/atlas-packet/storage/clientbound/*.go libs/atlas-packet/storage/serverbound/*.go | grep -v _test.go | wc -l
```

Expected: ≥78 (one `Encode` per top-level packet struct; multi-shape files contribute >1).

- [ ] **Step 3: Write `phase-0-survey.md`.**

Create the memo at `docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md` containing:

```markdown
# Phase 0 Survey — Foreign-Encoder Methods (task-067)

## Non-`Encode`/`Write` foreign-encoder methods in libs/atlas-packet/

| File:Line | Receiver | Method | Return shape | In task-067 scope? |
|---|---|---|---|---|
| cash/clientbound/shop_inventory.go:25 | CashInventoryItem | EncodeBytes | []byte (flat) | yes — Task 2 |
| inventory/change_entry.go:73 | AddEntry | EncodeEntry | func(opts) []byte (closure) | yes — Task 2 |
| inventory/change_entry.go:103 | QuantityUpdateEntry | EncodeEntry | closure | yes — Task 2 |
| inventory/change_entry.go:141 | MoveEntry | EncodeEntry | closure | yes — Task 2 |
| inventory/change_entry.go:173 | RemoveEntry | EncodeEntry | closure | yes — Task 2 |
| model/character_temporary_stat.go:575 | CharacterTemporaryStat | EncodeForeign | closure | NO — task-028 character scope; out of bounds |

## Auto-discovered (no action)

- `model.Asset.Encode` (asset.go:164) — inventory item slot sub-struct; used by `inventory/clientbound/change.go:Add`, `cash/clientbound/shop_item_moved.go:CashItemMovedToInventory`, and storage panels.
- All top-level packet structs (cash/interaction/inventory/storage × clientbound/serverbound) expose `Encode` on the receiver; registry pass-2 picks them up.

## Decision (design §8)

- If only `CashInventoryItem.EncodeBytes` was found: **Option A1** (recognise `EncodeBytes` only; ~10 LOC `registry.go` change).
- If `EncodeEntry` was also found: **Option A2** (recognise both `EncodeBytes` and `EncodeEntry`; ~14 LOC `registry.go` change). EncodeEntry has the same closure shape as `Encode`, so the existing `findReturnClosure` path handles it.
- If a method with semantics divergent from both is found: **Option B** (per-call ack in `_pending.md`).

Expected outcome: **Option A2** — extend the switch to recognise both names.
```

- [ ] **Step 4: Commit the survey.**

```bash
git add docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md
git commit -m "docs(task-067): phase-0 foreign-encoder method survey

Enumerates non-Encode/Write foreign-encoder methods in libs/atlas-packet/.
Confirms CashInventoryItem.EncodeBytes (flat []byte) and four
inventory.*Entry.EncodeEntry (closure) as in-scope; EncodeForeign is
task-028 character scope. Locks Phase 0c registry-extension decision."
```

Verify post-commit:

```bash
git rev-parse --show-toplevel
git branch --show-current
```

Expected: ends with `/.worktrees/task-067-commerce-domain-packet-audit` and branch is `task-067-commerce-domain-packet-audit`. If either disagrees, STOP.

### Task 2: Registry extension for `EncodeBytes` + `EncodeEntry` (Option A2)

Extend the `TypeRegistry`'s pass-2 method-name switch in `tools/packet-audit/internal/atlaspacket/registry.go:101-114` to recognise two new method names:

- `EncodeBytes` — flat body (no closure return). Analyze `fd.Body` directly (mirrors the existing `Write` case).
- `EncodeEntry` — closure return body (like `Encode`). Analyze via `findReturnClosure(fd.Body)` (mirrors the existing `Encode` case).

`Encode` wins over `EncodeBytes`/`EncodeEntry`/`Write` per the existing precedence semantics. Phase 0a survey (Task 1) must have completed before Task 2 starts; if the survey surfaced any divergent-semantics method, fall back to design §8 Option B by reverting this task to a `_pending.md` row.

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry.go:101-114`
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go` (append new fixtures)

- [ ] **Step 1: Write the failing registry fixture for commerce sub-structs.**

Append to `tools/packet-audit/internal/atlaspacket/registry_test.go`:

```go
func TestRegistryRegistersCommerceSubStructs(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"CashInventoryItem", // EncodeBytes (flat)
		"AddEntry",          // EncodeEntry (closure)
		"QuantityUpdateEntry",
		"MoveEntry",
		"RemoveEntry",
	} {
		if !reg.HasType(name) {
			t.Errorf("registry missing type %s", name)
			continue
		}
		calls, ok := reg.Calls(name)
		if !ok || len(calls) == 0 {
			t.Errorf("%s produced no calls (ok=%v len=%d)", name, ok, len(calls))
		}
	}
}
```

If Task 1 Step 1 surfaced an additional foreign-encoder method receiver type, append that type name to the slice literal above before running Step 2.

- [ ] **Step 2: Run the test to verify it fails.**

```bash
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersCommerceSubStructs -v
```

Expected: FAIL with `"CashInventoryItem produced no calls"` and `"AddEntry produced no calls"` (and three more for the other inventory entries). Receivers are registered by pass-1 (type discovery), but pass-2 doesn't analyze their bodies because their method names aren't `Encode` or `Write`.

- [ ] **Step 3: Extend the pass-2 method-name switch.**

Edit `tools/packet-audit/internal/atlaspacket/registry.go:101-114` from:

```go
			switch fd.Name.Name {
			case "Encode":
				body := findReturnClosure(fd.Body)
				if body == nil {
					body = fd.Body
				}
				entry.Calls = collectCallsWithCtx(body, fc.fset, reg, recvType)
			case "Write":
				// Write methods have a flat body (no closure return) and accept *response.Writer.
				// Only register if no Encode method was already found.
				if entry.Calls == nil {
					entry.Calls = collectCallsWithCtx(fd.Body, fc.fset, reg, recvType)
				}
			}
```

…to:

```go
			switch fd.Name.Name {
			case "Encode":
				body := findReturnClosure(fd.Body)
				if body == nil {
					body = fd.Body
				}
				entry.Calls = collectCallsWithCtx(body, fc.fset, reg, recvType)
			case "EncodeEntry":
				// EncodeEntry returns a closure (same shape as Encode) but the method
				// name differs because the type is a list-entry sub-struct (e.g.
				// inventory/change_entry.go's AddEntry/MoveEntry). Encode wins over
				// EncodeEntry per the precedence below.
				if entry.Calls == nil {
					body := findReturnClosure(fd.Body)
					if body == nil {
						body = fd.Body
					}
					entry.Calls = collectCallsWithCtx(body, fc.fset, reg, recvType)
				}
			case "EncodeBytes":
				// EncodeBytes returns a flat []byte (no closure). Used for sub-structs
				// embedded inside a top-level Encode via WriteByteArray (e.g.
				// cash/clientbound/shop_inventory.go's CashInventoryItem).
				if entry.Calls == nil {
					entry.Calls = collectCallsWithCtx(fd.Body, fc.fset, reg, recvType)
				}
			case "Write":
				// Write methods have a flat body (no closure return) and accept *response.Writer.
				// Only register if no Encode method was already found.
				if entry.Calls == nil {
					entry.Calls = collectCallsWithCtx(fd.Body, fc.fset, reg, recvType)
				}
			}
```

- [ ] **Step 4: Re-run the fixture to verify it passes.**

```bash
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersCommerceSubStructs -v
```

Expected: PASS.

- [ ] **Step 5: Run the full audit-tool test suite to confirm no regression.**

```bash
go test -race ./tools/packet-audit/...
```

Expected: clean. If a previously-green test now fails, the most likely cause is that some character-domain `EncodeForeign` was already covered by an indirect mechanism — re-read the failing test before retrying.

- [ ] **Step 6: Commit the registry extension.**

```bash
git add tools/packet-audit/internal/atlaspacket/registry.go \
        tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "feat(packet-audit): recognise EncodeBytes and EncodeEntry method names

Extends TypeRegistry pass-2 method-name switch to register foreign-
encoder methods. EncodeBytes (flat []byte) covers CashInventoryItem;
EncodeEntry (closure) covers inventory.AddEntry/QuantityUpdateEntry/
MoveEntry/RemoveEntry. Encode still wins over both via the existing
precedence check. Adds registry fixture asserting commerce sub-struct
coverage.

Phase 0 of task-067 (commerce-domain packet audit)."
```

Verify post-commit:

```bash
git rev-parse --show-toplevel
git branch --show-current
```

Expected: as above. If either disagrees, STOP.

### Task 3: Cash-shop constructor-to-struct enumeration (design §4.4 / §10 Phase 1d preamble)

Before opening the cash audit (Task 7), enumerate which top-level cash-shop struct each `shop_operation_body.go` constructor delegates to. This is the design §16 "exact wire-shape count for cash, currently estimated at ~36" — Phase 1d's denominator depends on it.

**Files:**
- Modify: `docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md` (append a "Cash constructor ↔ struct map" section).

- [ ] **Step 1: Read `cash/clientbound/shop_operation_body.go` end to end.**

```bash
wc -l libs/atlas-packet/cash/clientbound/shop_operation_body.go
sed -n '1,50p' libs/atlas-packet/cash/clientbound/shop_operation_body.go
```

Expected: a constants block followed by `New<Op>Body` factory functions, each calling `atlas_packet.ResolveCode("operations", "...")` and constructing a struct from another `cash/clientbound/*.go` file.

- [ ] **Step 2: Enumerate every constructor and its target struct.**

```bash
grep -n "^func New" libs/atlas-packet/cash/clientbound/shop_operation_body.go
```

For each `NewXxxBody(...)`, identify the target struct it delegates to (read the function body). Expected delegations per design §3.1:

- `NewCashShopWishListBody` → `WishList` (defined in `shop_operation_result.go`).
- `NewCashShopCashInventoryBody` → `CashShopInventory` (defined in `shop_inventory.go`).
- `NewCashShopCashInventoryPurchaseSuccessBody` → `CashShopPurchaseSuccess` (defined in `shop_inventory.go`).
- `NewCashShopCashItemMovedToInventoryBody` → `CashItemMovedToInventory` (defined in `shop_item_moved.go`).
- `NewCashShopCashItemMovedToCashInventoryBody` → `CashItemMovedToCashInventory` (defined in `shop_item_moved.go`).
- `NewCashShopInventoryCapacityIncreaseSuccessBody` → `InventoryCapacitySuccess` (defined in `shop_operation_result.go`).
- `NewCashShopInventoryCapacityIncreaseFailedBody` → `InventoryCapacityFailed` (defined in `shop_operation_result.go`).
- `NewCashShopCashGiftsBody` → `CashShopGifts` (defined in `shop_inventory.go`).

If the actual function list does not match the design's prediction (e.g., a constructor is missing, an extra one exists), capture the actual list in Step 3 — the design's count is a prediction, the code is authoritative.

- [ ] **Step 3: Append "Cash constructor ↔ struct map" to `phase-0-survey.md`.**

Append the following heading + table (filling actual rows from Step 2):

```markdown
## Cash constructor ↔ struct map (design §4.4)

| Constructor (shop_operation_body.go) | Target struct | Defined in |
|---|---|---|
| NewCashShopWishListBody | WishList | shop_operation_result.go |
| NewCashShopCashInventoryBody | CashShopInventory | shop_inventory.go |
| NewCashShopCashInventoryPurchaseSuccessBody | CashShopPurchaseSuccess | shop_inventory.go |
| NewCashShopCashItemMovedToInventoryBody | CashItemMovedToInventory | shop_item_moved.go |
| NewCashShopCashItemMovedToCashInventoryBody | CashItemMovedToCashInventory | shop_item_moved.go |
| NewCashShopInventoryCapacityIncreaseSuccessBody | InventoryCapacitySuccess | shop_operation_result.go |
| NewCashShopInventoryCapacityIncreaseFailedBody | InventoryCapacityFailed | shop_operation_result.go |
| NewCashShopCashGiftsBody | CashShopGifts | shop_inventory.go |

Implication for Phase 1d: shop_operation_body.go gets ONE row in SUMMARY.md
(verdict ⚠️ "router; per-shape rows recorded under target structs above").
The cash wire-shape denominator stays at ~36; Phase 1d does not duplicate
constructor rows.
```

- [ ] **Step 4: Commit the cash map.**

```bash
git add docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md
git commit -m "docs(task-067): cash-shop constructor-to-struct map

Phase 0c map of NewCashShopXxxBody factories in shop_operation_body.go
to their target structs in shop_inventory.go / shop_operation_result.go /
shop_item_moved.go. Locks cash wire-shape denominator at ~36 for Phase 1d."
```

Verify post-commit as above.

**Phase 0 exit:** `go test -race ./tools/packet-audit/...` green; `phase-0-survey.md` committed with foreign-encoder findings and cash constructor map; registry recognises `EncodeBytes` + `EncodeEntry` (or, fallback, `_pending.md` row written and EncodeBytes call sites tagged in Phase 1d).

---

## Phase 1 — v95 audit by sub-domain

Four tracking sub-tasks (Tasks 4–7), one per commerce sub-domain. Ordering per design §10 Phase 1: **storage → inventory → interaction → cash** — smallest/durable first, biggest/version-sensitive last.

The audit command is the same for every sub-task in this phase:

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

It produces per-packet reports under `docs/packets/audits/gms_v95/<domain>/<PacketName>.{md,json}` and updates `docs/packets/audits/gms_v95/SUMMARY.md`. Run once per sub-task; commit the report files alongside the fix commits.

Before starting Phase 1, the user must have v95 IDA loaded so MCP `mcp__ida-pro__*` calls resolve. Each sub-task's IDA additions land in `docs/packets/ida-exports/gms_v95.json` (append) in the same commit as the audit-report bucket commit.

### Verdict triage rules (apply within every sub-task)

For each report under `docs/packets/audits/gms_v95/<domain>/`:

- **✅** → no action; row already in `SUMMARY.md`.
- **⚠️** → annotate the report manually with a one-line "ack: <reason>" footer; commit alone (commit message: `audit(<domain>/<pkt>): ack <reason>`). Examples: tool-limited package-level write helper, loop-flattened lists (analyzer flattens fixed-count loops; cite IDA loop bound), `EncodeMask` / sub-struct method calls that emit multiple bytes per analyzer call.
- **❌** → triage by flavour:
  - **Atlas wire bug** (width / order / missing field / silent-success) → fix in `libs/atlas-packet/<domain>/<dir>/<pkt>.go`. Add or extend a 4-variant test sweep in `<pkt>_test.go` (template provided in the per-fix recipe below).
  - **Template opcode drift** → fix in `services/atlas-configurations/seed-data/templates/template_gms_*_1.json` and/or `template_jms_185_1.json`. Atlas-packet stays untouched.
  - **Sub-op enum drift** (cash operations/errors enum, interaction mode-byte map, storage operation mode bytes, inventory ChangeMode) → defer to `_pending.md` per design §11, grouped under one heading "Sub-op enum / sub-struct deferrals — commerce domain (task-067)".
  - **Bare handler / no atlas-packet decoder** → defer to `_pending.md` under a new "## Bare handlers — commerce domain (task-067)" heading (create on first hit). Do not descend into atlas-channel / atlas-cashshop / atlas-inventory / atlas-storage service code (PRD §3 non-goal).
  - **Operation-dispatcher op-byte parameter** (per design §5: `interaction/serverbound/operation.go`, `storage/serverbound/operation.go`) → ⚠️ verdict + footer noting "op-byte value supplied by caller; see `_pending.md` row OP-FAMILY-{interaction,storage}". Add the OP-FAMILY rows to `_pending.md` in the first sub-task that encounters each family.

### Per-fix recipe (used inside every Phase 1 sub-task)

For each ❌ atlas-wire-bug fix:

1. **Fetch IDA evidence.** For clientbound packets the IDA function is `CClientSocket::SendXxx` (or the named writer); for serverbound, `CWvsContext::OnXxxPacket` / `CCashShop::OnPacket` / `CUserLocal::OnInteract` / `CWvsContext::OnInventoryOperation` etc. Use the FName from the audit report:

   ```
   mcp__ida-pro__get_function_by_name("<FName>")
   mcp__ida-pro__decompile_function(<addr>)
   ```

   Append the function's signature + address + a `Decode*`/`Encode*` op summary (matching the existing `gms_v95.json` schema's `Decode1/2/4/Str/Buffer/Loop` shape) to `docs/packets/ida-exports/gms_v95.json`.

2. **Edit the encoder.** Apply the minimum `Write*` change needed to match IDA. For version-conditional fixes use the existing `tenant.Model.Region()` / `tenant.Model.MajorVersion()` axes; respect the 2-nested-guard cap.

3. **Add the 4-variant test sweep.** Mirror task-066's pattern; for clientbound:

   ```go
   func TestCashShopInventoryByteForByte(t *testing.T) {
       cases := []struct {
           name string
           tn   tenant.Model
           want string // hex
       }{
           {"gms_v83", pt.GMSv83(), "<hex from IDA>"},
           {"gms_v95", pt.GMSv95(), "<hex from IDA>"},
           {"jms_v185", pt.JMSv185(), "<hex from IDA>"},
           // pt.Variants iteration is the canonical form; this expansion is here to
           // make the IDA hex source explicit per variant. If a fourth variant
           // (GMS v28 or v87) is in pt.Variants at this commit, add it.
       }
       for _, tc := range cases {
           t.Run(tc.name, func(t *testing.T) {
               ctx := tenant.WithContext(context.Background(), tc.tn)
               got := NewCashShopInventory(/* fields per packet */).Encode(testLogger(), ctx)(nil)
               if hex.EncodeToString(got) != tc.want {
                   t.Fatalf("encode mismatch\n got %s\nwant %s", hex.EncodeToString(got), tc.want)
               }
           })
       }
   }
   ```

   For serverbound, use the round-trip pattern from `libs/atlas-packet/test/roundtrip.go:12-24` — decode known-good IDA hex bytes, assert `r.Available() == 0` and field values.

   Hex values are captured from IDA by hand: in the decompile, find the call-site or the case-statement body and translate the `WriteXxx` / `ReadXxx` sequence to bytes.

4. **Run the affected packet's tests:**

   ```bash
   go test -race ./libs/atlas-packet/<domain>/<dir>/... -run Test<Pkt> -v
   ```

   Expect: clean.

5. **Commit per fix.**

   For atlas-packet fixes:

   ```bash
   git add libs/atlas-packet/<domain>/<dir>/<pkt>.go \
           libs/atlas-packet/<domain>/<dir>/<pkt>_test.go
   git commit -m "fix(atlas-packet,<domain>/<pkt>): <one-line summary>

   Cites IDA <CClientSocket::SendXxx>@<addr>: <one-line evidence>."
   ```

   For template fixes:

   ```bash
   git add services/atlas-configurations/seed-data/templates/template_*.json
   git commit -m "fix(configurations,templates): <pkt> opcode <old>→<new> for <region/version>

   IDA case-statement value at <CWvsContext::OnXxxPacket>@<addr>."
   ```

   For `_pending.md` deferrals:

   ```bash
   git add docs/packets/audits/gms_v95/_pending.md
   git commit -m "audit(<domain>/<pkt>): defer — <one-line reason>"
   ```

### Bucket-commit recipe (end of every Phase 1 sub-task)

After all per-fix commits land, commit the audit reports + SUMMARY + IDA-export append in one bucket commit:

```bash
git add docs/packets/audits/gms_v95/<domain>/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(<domain>): v95 audit (<n> packets / <m> wire shapes)"
```

### Exit gate (every Phase 1 sub-task)

```bash
ls docs/packets/audits/gms_v95/<domain>/*.md | wc -l
```

Must equal the bucket wire-shape count (per design §3 table). Then:

```bash
grep -c "<domain>/" docs/packets/audits/gms_v95/SUMMARY.md
```

Must equal the same count (rows are per wire shape, not per file). Every ❌ in `SUMMARY.md` for this sub-domain has either a fix commit on this branch (`git log --oneline | grep "<domain>/<pkt>"`) or a row in `_pending.md`.

---

### Task 4: Phase 1a — storage (7 src files, ~9 wire shapes)

**Packets — clientbound (3 files / 5 shapes):**
- `error.go` → 3 shapes: `ErrorSimple`, `UpdateMeso`, `ErrorMessage`
- `show.go` → 1 shape
- `update_assets.go` → 1 shape (durable storage panel writer per design §7)

**Packets — serverbound (4 files / 4 shapes):**
- `operation.go` (dispatcher per design §5/§7 — op-byte only)
- `operation_meso.go`, `operation_retrieve_asset.go`, `operation_store_asset.go`

`storage/clientbound/update_assets.go` is the durable hot path (design §7): one wire-shape error desyncs the storage panel until the player logs out. 4-variant byte-output sweep is mandatory for any fix here. `storage/serverbound/operation.go` is the dispatcher → OP-FAMILY-storage row in `_pending.md`; ⚠️ verdict on the file. `storage/clientbound/error.go`'s three shapes each carry their own mode byte — audit independently.

**Files:**
- Modify (per fix): `libs/atlas-packet/storage/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/storage/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/storage/<PacketName>.{md,json}` (audit-generated)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append: `docs/packets/ida-exports/gms_v95.json`
- Append (if deferrals): `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is loaded.**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches GMS v95. If not, ask the user to swap before continuing.

- [ ] **Step 2: Run the audit (full pipeline; the tool produces storage/* reports as a subset of the run).**

Use the audit command from the Phase 1 preamble. Expected runtime: ≤ 90 s.

- [ ] **Step 3: Triage each storage/* report.**

Apply the verdict triage rules from the Phase 1 preamble. `storage/serverbound/operation.go` is the operation-dispatcher → record OP-FAMILY-storage row in `_pending.md` if any ❌ triggers it; verdict ⚠️ on the file. `storage/clientbound/error.go`'s three shapes get individual rows in SUMMARY.md (`error.go:ErrorSimple`, `error.go:UpdateMeso`, `error.go:ErrorMessage`); the dispatch byte each writes is the per-shape audit row's evidence anchor.

- [ ] **Step 4: Per-fix loop — for each ❌, follow the per-fix recipe (5 sub-steps) from the Phase 1 preamble.**

Hot-path discipline: 4-variant byte-output sweep is mandatory for any fix to `update_assets.go`. Cite the IDA dispatcher offset for the storage-panel writer (`CStorageManMan::Show` or equivalent) in the fix comment.

- [ ] **Step 5: Bucket commit (audit reports + SUMMARY + IDA-export append).**

```bash
git add docs/packets/audits/gms_v95/storage/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(storage): v95 audit (7 packets / 9 wire shapes)"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/storage/*.md | wc -l
```

Expected: 9 (with cb/sb name disambiguation if the tool collides `operation.md` filenames; hand-rename to `operation_cb.md` / `operation_sb.md` consistently and re-stage). `error.md` should appear as three separate per-shape report files if the tool emits one per shape, OR as a single `error.md` with three sections; either form is acceptable as long as SUMMARY.md has three rows.

```bash
grep -c "atlas-packet/storage/" docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: 9.

---

### Task 5: Phase 1b — inventory (9 src files, ~12 wire shapes)

**Packets — clientbound (4 files / 7 shapes):**
- `change.go` → 4 shapes: `QuantityUpdate` (mode 1), `ChangeMove` (mode 2), `Remove` (mode 3), `Add` (mode 0) — each writes its own change-mode discriminator byte
- `change_batch.go` → 1 shape (consumes `inventory.ChangeEntry` family registered in Phase 0)
- `compartment_merge.go` → 1 shape
- `compartment_sort.go` → 1 shape

**Packets — serverbound (5 files / 5 shapes):**
- `compartment_merge.go`, `compartment_sort.go`, `item_use.go`, `move.go`, `scroll_use.go`

`inventory/clientbound/change.go` is THE silent-loss hot path per design §6 — fired on every item-affecting transition. Each of the four shapes carries the inventory-type byte; equip-slot moves (`inventoryType == 1 && slot < 0`) write an additional `addMov` byte (verified at `change.go:92-99` and `change.go:146-148`). The IDA dispatcher for `CWvsContext::OnInventoryOperation` is the source of truth.

Symmetry: `inventory/serverbound/move.go` reads what the client sends; the server echoes via `change.go:ChangeMove`. Wire-shape mismatch between the serverbound `Decode` and the clientbound `Encode` for the equip-slot `addMov` byte is the single most likely silent-corruption bug in the commerce domain. This task audits both halves of the move flow and asserts symmetry.

**Files:**
- Modify (per fix): `libs/atlas-packet/inventory/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/inventory/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/inventory/<PacketName>.{md,json}` (audit-generated)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append: `docs/packets/ida-exports/gms_v95.json`
- Append (if deferrals): `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is still loaded** (`mcp__ida-pro__get_metadata`).

- [ ] **Step 2: Run the audit (Phase 1 preamble command).**

- [ ] **Step 3: Triage each inventory/* report.**

`change.go` gets four rows in SUMMARY.md (one per `QuantityUpdate`/`ChangeMove`/`Remove`/`Add`). Treat each independently — a fix on `ChangeMove` does not implicate `Add`. `change_batch.go` exercises the Phase 0–registered `ChangeEntry` sub-struct family — verdict should now resolve recursion bytes rather than reporting opaque `WriteByteArray` calls. If recursion fails to resolve, re-check the Phase 0 registry extension.

`inventory.ChangeMode*` enum (4 values: 0=Add, 1=QuantityUpdate, 2=Move, 3=Remove — verified at `change.go:43,88,143,192`) is sub-op enum surface; if the IDA dispatcher case-statement values disagree with these constants, record under the "Sub-op enum / sub-struct deferrals — commerce domain (task-067)" heading in `_pending.md` (single row, sub-list).

- [ ] **Step 4: Per-fix loop — Phase 1 per-fix recipe.**

Hot-path discipline (`change.go` four shapes): 4-variant byte-output sweep is mandatory; the `silent` boolean (used by `Add`/`Remove`/`ChangeMove`) is a separate axis — exercise both `silent=true` and `silent=false` per variant. For any equip-slot fix, also exercise `inventoryType == 1` (equip) with `slot < 0` and `slot >= 0` to cover the `addMov` branch.

Symmetry assertion: after any fix to `move.go` (serverbound) or `change.go:ChangeMove` (clientbound), add a round-trip test in `move_test.go` that decodes the serverbound bytes, applies the move, encodes the clientbound `ChangeMove` response, and asserts byte-level equality with a known-good IDA capture.

- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/inventory/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(inventory): v95 audit (9 packets / 12 wire shapes)"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/inventory/*.md | wc -l   # Expected: 12 (4 change.go shapes + 8 others; tool may emit one file per shape or one file per src — accept either)
grep -c "atlas-packet/inventory/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: 12
```

---

### Task 6: Phase 1c — interaction (32 src files, 32 wire shapes)

**Packets — clientbound (2 files / 2 shapes):**
- `interaction.go`, `interaction_body.go`

**Packets — serverbound (30 files / 30 shapes):**
- `operation.go` (shared dispatcher per design §5 — writes only the mode byte at `operation.go:32`)
- 29 named sub-op files: `operation_cash_trade_open.go`, `operation_chat.go`, `operation_create.go`, `operation_field_add_to_black_list.go`, `operation_field_remove_from_black_list.go`, `operation_invite.go`, `operation_invite_decline.go`, `operation_memory_game_flip_card.go`, `operation_memory_game_move_stone.go`, `operation_memory_game_retreat_answer.go`, `operation_memory_game_tie_answer.go`, `operation_merchant_add_to_black_list.go`, `operation_merchant_buy.go`, `operation_merchant_name_change.go`, `operation_merchant_put_item.go`, `operation_merchant_remove_from_black_list.go`, `operation_merchant_remove_item.go`, `operation_open.go`, `operation_personal_store_add_to_black_list.go`, `operation_personal_store_buy.go`, `operation_personal_store_put_item.go`, `operation_personal_store_remove_item.go`, `operation_personal_store_set_black_list.go`, `operation_personal_store_set_visitor.go`, `operation_trade_add_meso.go`, `operation_trade_confirm.go`, `operation_trade_put_item.go`, `operation_transaction.go`, `operation_visit.go`

This is the design §5 best-case scenario: each sub-op file is one independent wire shape, addressed individually by the analyzer. `operation.go` (the dispatcher) gets ⚠️ "tool-limitation: mode byte supplied by caller; sub-op routing recorded in _pending.md row INTERACTION-MODE-MAP" — the mode-byte → sub-op-file mapping lives in atlas-channel routing, outside `libs/atlas-packet/`. One `_pending.md` row, not 30.

`interaction/clientbound/interaction.go` and `interaction_body.go` are likely router/body structs per design §5.1 — verify Phase 1c reading; if either turns out to be a per-mode constructor block (parallel to cash's `shop_operation_body.go`), record the constructor-to-struct map mid-task (mirror Task 3's pattern) before triaging the 30 sub-op rows. If they are standalone shapes (single Encode each), audit normally.

**Files:**
- Modify (per fix): `libs/atlas-packet/interaction/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/interaction/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/interaction/<PacketName>.{md,json}` (audit-generated)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append: `docs/packets/ida-exports/gms_v95.json`
- Append (if deferrals): `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is still loaded** (`mcp__ida-pro__get_metadata`).

- [ ] **Step 2: Run the audit (Phase 1 preamble command).**

- [ ] **Step 3: Pre-triage — confirm `operation.go` is dispatcher-only.**

```bash
sed -n '25,45p' libs/atlas-packet/interaction/serverbound/operation.go
```

Expected: an `Encode`/`Decode` body that writes/reads only the `mode` byte. If the body emits more than one `WriteByte(...)` (i.e., carries payload beyond the op byte), design §12 risk #3 ("interaction sub-op family proves to have shared payload bytes") has materialised — record the actual dispatcher shape in the audit report instead of treating as op-byte-only, and audit accordingly. Do not rewrite the plan.

If a sub-op file (e.g. `operation_merchant_buy.go`) re-writes the mode byte before its payload, that's a wire-bug fix candidate — treat per the per-fix recipe.

- [ ] **Step 4: Triage clientbound (`interaction.go`, `interaction_body.go`).**

Read both files:

```bash
sed -n '1,80p' libs/atlas-packet/interaction/clientbound/interaction.go
sed -n '1,80p' libs/atlas-packet/interaction/clientbound/interaction_body.go
```

Decide:
- **Both are standalone wire shapes** (one `Encode` each, no constructor factory cluster) → audit each as one row.
- **`interaction_body.go` is a constructor block** (multiple `NewXxxBody` factories per shape, parallel to `cash/clientbound/shop_operation_body.go`) → write a constructor ↔ struct map in `phase-0-survey.md` (append a new section); each target struct becomes its own row. Update SUMMARY.md row template (`<file>:<TypeName>`) to match Task 3's convention.

Either outcome is fine; the row count for interaction is locked at 32 (30 SB sub-ops + 2 CB shapes) unless this step expands CB.

- [ ] **Step 5: Per-fix loop — Phase 1 per-fix recipe.**

Add OP-FAMILY-interaction (the dispatcher mode-byte map) row + INTERACTION-CB-MODE-MAP row (if interaction_body.go is a router) to `_pending.md` if not already present. Group under the "Sub-op enum / sub-struct deferrals — commerce domain (task-067)" heading.

Specific risks per design §5:
- Dispatcher-layer offset (CUserPool dispatchers prepend `characterId` before routing — atlas wire includes it at offset 0). Trade and personal-store packets encoding buyer/seller IDs are susceptible. Verify on each sub-op file by comparing the atlas serverbound `Decode` body's first field against the IDA dispatcher's case body.

- [ ] **Step 6: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/interaction/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/audits/gms_v95/_pending.md
git commit -m "audit(interaction): v95 audit (32 packets / 32 wire shapes)"
```

- [ ] **Step 7: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/interaction/*.md | wc -l   # Expected: 32 (with cb/sb collision rename if needed)
grep -c "atlas-packet/interaction/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: 32
```

If `interaction_body.go` turned out to be a router per Step 4, the row count grows by however many target shapes it routes to; document the new denominator in the bucket commit message.

---

### Task 7: Phase 1d — cash (30 src files, ~36 wire shapes; highest version-divergence risk)

**Packets — clientbound (6 files / ~14 shapes per design §3.1):**
- `query_result.go` → 1 shape
- `shop_inventory.go` → 4 shapes: `CashInventoryItem` (sub-struct), `CashShopInventory`, `CashShopPurchaseSuccess`, `CashShopGifts`
- `shop_item_moved.go` → 2 shapes: `CashItemMovedToInventory`, `CashItemMovedToCashInventory`
- `shop_open.go` → 1 shape
- `shop_operation_body.go` → 1 row (router; per-shape rows tracked under `shop_inventory.go` / `shop_operation_result.go` / `shop_item_moved.go` per Task 3 map)
- `shop_operation_result.go` → 4 shapes: `OperationError`, `InventoryCapacitySuccess`, `InventoryCapacityFailed`, `WishList`

**Packets — serverbound (24 files / 24 shapes):**
- `check_wallet.go`, `item_use.go`, `item_use_chalkboard.go`, `item_use_field_effect.go`, `item_use_pet_consumable.go`, `shop_entry.go`, `shop_operation.go`
- `shop_operation_buy.go` (umbrella) + 6 named purchase variants: `shop_operation_buy_couple.go`, `shop_operation_buy_friendship.go`, `shop_operation_buy_name_change.go`, `shop_operation_buy_normal.go`, `shop_operation_buy_package.go`, `shop_operation_buy_world_transfer.go`
- `shop_operation_enable_equip_slot.go`, `shop_operation_get_purchase_record.go`, `shop_operation_gift.go`
- 3 capacity-increase variants: `shop_operation_increase_character_slot.go`, `shop_operation_increase_inventory.go`, `shop_operation_increase_storage.go`
- 2 move variants: `shop_operation_move_from_cash_inventory.go`, `shop_operation_move_to_cash_inventory.go`
- `shop_operation_rebate_locker_item.go`, `shop_operation_set_wishlist.go`

Cash is the design §1/§4 heavy domain. Three sub-problems converge here:

1. **`shop_operation_buy_*` family (§4.1):** 7 named purchase variants (`_buy.go` umbrella + 6 named flavours). Each is its own row, its own commit, its own 4-variant test. No multi-variant fix commits — if a fix would touch 2+ variant files, treat as "shared encoder needs split" and pause for triage per design §12 risk #6.
2. **`shop_operation_increase_*` family (§4.2):** 3 capacity-increase variants. Same per-variant treatment.
3. **`shop_operation_body.go` constants block (§4.3):** 78 named operation/error codes resolved via `atlas_packet.ResolveCode("operations", ...)` / `ResolveCode("errors", ...)`. Code-name → wire-byte mapping lives in `template_*.json`. Cap: if Phase 1d surfaces >10 stale code values in v95, pause and triage — the template's regional split may need restructuring (sibling task), not a 10-line fix.

`CashInventoryItem` (the `EncodeBytes` sub-struct registered in Phase 0) recurses inside `CashShopInventory.Encode` (loop), `CashShopPurchaseSuccess.Encode`, and `CashItemMovedToCashInventory.Encode`. After the Phase 0 registry extension, the analyzer should resolve its bytes inline rather than reporting opaque `WriteByteArray` calls. If it doesn't, the Phase 0 extension didn't take effect — re-run `go test -race ./tools/packet-audit/...` against the fixture before proceeding.

**Files:**
- Modify (per fix): `libs/atlas-packet/cash/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/cash/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/cash/<PacketName>.{md,json}` (audit-generated)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append: `docs/packets/ida-exports/gms_v95.json`
- Append (if deferrals): `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is still loaded** (`mcp__ida-pro__get_metadata`).

- [ ] **Step 2: Re-verify the cash wire-shape denominator from Task 3.**

Open `docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md` and confirm the "Cash constructor ↔ struct map" section is present and accurate. If a new constructor/struct surfaced during Phase 1a–1c, append it now before running the audit; otherwise the denominator stays at ~36.

- [ ] **Step 3: Run the audit (Phase 1 preamble command).**

- [ ] **Step 4: Triage each cash/* report.**

For multi-shape files, SUMMARY.md rows use the `<path>:<TypeName>` template (e.g. `cash/clientbound/shop_inventory.go:CashShopInventory`, `cash/clientbound/shop_inventory.go:CashShopPurchaseSuccess`, …). `shop_operation_body.go` gets ONE row with verdict ⚠️ "router; per-shape rows tracked under target structs".

For the `shop_operation_buy_*` family: each variant is its own row. Verify against the IDA `CCashShop::OnPacket` dispatcher case body for the matching purchase-type byte. Same per-variant treatment for `shop_operation_increase_*` (inventory / storage / character-slot) and `shop_operation_move_*` (from / to cash inventory).

For `shop_operation_body.go`'s 78 named constants: do not audit each constant individually. Audit the resolution path (does atlas resolve the code name → the wire byte the IDA case-statement expects?) for each *dispatcher function*. Group the IDA case-statement evidence by dispatcher (`load-inventory`, `purchase`, `move`, `wishlist`, etc.) in the audit report. If a single dispatcher shows >10 stale code values, pause and triage per design §4.3 cap.

- [ ] **Step 5: Per-fix loop — Phase 1 per-fix recipe.**

For `CashInventoryItem.EncodeBytes` (a sub-struct, not a top-level packet), the audit report's verdict applies to all three call sites (`CashShopInventory`, `CashShopPurchaseSuccess`, `CashItemMovedToCashInventory`); a fix to `EncodeBytes` affects all three rows simultaneously. Update each row's verdict in the same fix commit; bucket-commit the SUMMARY changes after all per-shape commits land.

Cash-shop risk specifics:
- v95-only fields (Maple Points / NX Credit / NX Prepaid split per design §9). If the v95 IDA dispatcher emits more fields than the atlas encoder, that's a wire-add fix gated to `MajorVersion() >= 95` or `Region() == "GMS" && MajorVersion() >= 95`.
- Gift-from padding (`WritePaddedString(GiftFrom, 13)` at `shop_inventory.go:33`) — v83 may have a different padding width. Cross-version Phase 2 v83 will catch this; if v95 IDA itself disagrees with `13`, fix here.
- World-transfer SKU (`shop_operation_buy_world_transfer.go`): a v95-only feature (design §9). v83 likely has no IDA case for this mode — audit verdict on the file is `✅ N/A — feature gated correctly` if `Region() == "GMS" && MajorVersion() >= 95` (or similar) wraps the entire `Encode`. If atlas always encodes it: ❌ → add the gate.

- [ ] **Step 6: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/cash/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/audits/gms_v95/_pending.md
git commit -m "audit(cash): v95 audit (30 packets / ~36 wire shapes)"
```

- [ ] **Step 7: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/cash/*.md | wc -l   # Expected: ~36 (with multi-shape files emitting one report per shape or one report per src — accept either)
grep -c "atlas-packet/cash/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: ~36
```

The exact wire-shape denominator was locked in Task 3 and re-verified in Step 2 of this task. Confirm the row count matches the locked denominator before exiting.

**Phase 1 exit:** `SUMMARY.md` contains rows for all 78 commerce-domain src files (~89 wire shapes). Every ❌ has a fix commit on this branch OR a `_pending.md` row.

```bash
grep -c "atlas-packet/\(cash\|interaction\|inventory\|storage\)/" docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: ≥78 (rows are per wire shape, not per file — the count will be ~89 if SUMMARY.md uses the per-shape row template throughout, or 78 if it falls back to per-file rows for non-multi-shape files). If less than 78, identify the missing files via:

```bash
diff <(ls libs/atlas-packet/{cash,interaction,inventory,storage}/{clientbound,serverbound}/*.go 2>/dev/null | grep -v _test.go | xargs -n1 basename | sort -u) \
     <(grep -oE "atlas-packet/(cash|interaction|inventory|storage)/(clientbound|serverbound)/[a-z_]+\.go" docs/packets/audits/gms_v95/SUMMARY.md | xargs -n1 basename | sort -u)
```

Investigate every missing file before exiting Phase 1.

---

## Phase 2 — Cross-version pass (v83 → v87 → JMS v185)

Three tracking sub-tasks (Tasks 8–10), one per version. Each requires a user-driven IDA binary swap before starting (PRD §4.6, design §10 Phase 2). A sub-task is "done" when:

- `docs/packets/ida-exports/gms_{v83,v87}.json` or `gms_jms_185.json` contains commerce-domain entries for every FName resolved during Phase 1.
- The audit has been re-run against the version's template + IDA export, producing reports under `docs/packets/audits/<version>/<commerce-domain>/`.
- Every divergence vs v95 atlas-packet behaviour has either:
  - A `Region/MajorVersion` gate that already handles it (audit report captures evidence; no code change),
  - A gate fix on this branch with a 4-variant test sweep, OR
  - A template fix.
- Hard cap: if any single commerce-domain encoder now contains 3+ nested `if t.Region()` / `if t.MajorVersion()` levels, STOP per design §9 / task-028 §7. Append a row to `_pending.md` describing the encoder + which version chain triggered it. Do NOT refactor in this task.

### Task 8: GMS v83 cross-version pass

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (append commerce entries — file already exists with login + character + social entries from prior tasks)
- Modify (per fix): `libs/atlas-packet/<domain>/<dir>/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Create or modify: `docs/packets/audits/gms_v83/<domain>/<pkt>.{md,json}` (tool creates the directory)
- Modify: `docs/packets/audits/gms_v83/SUMMARY.md`
- Append (if deferrals): `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Confirm v83 IDA is loaded.**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches GMS v83. If not, ask the user to swap before continuing.

- [ ] **Step 2: For each commerce-domain FName resolved during Phase 1, populate `gms_v83.json`.**

Workflow (per FName):
- `mcp__ida-pro__get_function_by_name("<FName>")` (or `get_function_by_address` if names diverge).
- `mcp__ida-pro__decompile_function(<addr>)`.
- Translate to the existing `gms_v83.json` schema (`Decode1/2/4/Str/Buffer/Loop` op list with guard expressions). Append entries; do not reorder existing login/character/social entries.

If a FName has no v83 equivalent (different opcode space, no matching function, or the feature is v95-only — design §9 names cash-shop SKU expansion / world-transfer / friendship as candidates), record the v83-side FName + address as a separate entry annotated with `"region": "GMS"` and `"version": 83`. Do NOT reuse v95 FNames for unrelated v83 functions.

- [ ] **Step 3: Re-run the audit against v83.**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v83.json \
  --output           docs/packets/audits/gms_v83
```

If `docs/packets/audits/gms_v83/` doesn't exist yet, the tool creates it.

- [ ] **Step 4: Triage divergences.**

For each ❌ in the v83 commerce audit:
- Was the v95 fix gated on `MajorVersion() >= 95`? → no v83 regression. Audit-report-only.
- Was the v95 fix gated on `Region() == "GMS"` (no major-version filter)? → check whether v83 IDA confirms the same behaviour. If yes: tighten the gate so v83 keeps its old shape. If no: leave as-is and document.
- Is this a *new* v83-only mismatch the v95 audit didn't surface? → genuine cross-version bug. Fix with 4-variant test sweep + `Region/MajorVersion` gate. Per-fix recipe from Phase 1 preamble.
- Is the v83 dispatcher missing the case entirely (e.g., world-transfer, friendship SKU)? → atlas must not encode for v83. If atlas attempts to encode, the verdict is ❌ "missing v83 gate" and the fix is to add the gate. If atlas already gates it, the row is `✅ N/A — feature absent in v83; gate correct`.
- **Hard cap:** if the fix makes the encoder breach 3 nested guards, STOP — `_pending.md` row, no refactor.

Specific design §9 watch-list for v83:
- `CashInventoryItem` 49-byte width — `WritePaddedString(GiftFrom, 13)` + two trailing `WriteInt(0)` padding bytes are version-sensitive. Verify gift-from field width and presence against v83 IDA decompile of the cash-inventory shop dispatcher.
- Inventory item-slot width — parallel to task-028's `GW_CharacterStat` HP/MP widening. Inspect every `int16` slot field for v83 vs v95 width drift.
- Storage panel size — v83 may have a different storage-slot count field width (`update_assets.go`).

- [ ] **Step 5: Commit per-fix; bucket commit for the version.**

Per-fix commit format:

```
fix(atlas-packet,<domain>/<pkt>): widen/narrow v83 gate for <field>

Cites IDA v83 <CClientSocket::SendXxx>@<addr>: <one-line evidence>.
```

Final bucket commit:

```bash
git add docs/packets/ida-exports/gms_v83.json \
        docs/packets/audits/gms_v83/
git commit -m "audit(commerce): GMS v83 cross-version pass (commerce domain)"
```

- [ ] **Step 6: Hard-cap check.**

```bash
for f in libs/atlas-packet/{cash,interaction,inventory,storage}/{clientbound,serverbound}/*.go; do
    [[ "$f" == *_test.go ]] && continue
    nested=$(awk '
        /if t\.Region\(\)|if t\.MajorVersion\(\)|if .*\.Region\(\)|if .*\.MajorVersion\(\)/ { d++; if (d > max) max = d }
        /^}/ { if (d > 0) d-- }
        END { print max+0 }
    ' "$f")
    if (( nested >= 3 )); then
        echo "OVER CAP: $f ($nested nested guards)"
    fi
done
```

If any "OVER CAP" line appears, append a row to `docs/packets/audits/gms_v95/_pending.md` describing the encoder + which version chain triggered it; do NOT refactor in this task.

---

### Task 9: GMS v87 cross-version pass

Identical shape to Task 8. Replace `v83` with `v87` everywhere. Templates: `template_gms_87_1.json`. Export file: `docs/packets/ida-exports/gms_v87.json` (may not exist yet; the social audit task-066 was scheduled to create it — confirm via `ls docs/packets/ida-exports/`. If absent, this task creates the file by Step 2's first append).

**Files:**
- Create or modify: `docs/packets/ida-exports/gms_v87.json`
- Modify (per fix): `libs/atlas-packet/<domain>/<dir>/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Create or modify: `docs/packets/audits/gms_v87/<domain>/<pkt>.{md,json}`
- Modify: `docs/packets/audits/gms_v87/SUMMARY.md`
- Append (if deferrals): `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Confirm v87 IDA is loaded** (`mcp__ida-pro__get_metadata`; binary field == GMS v87).

- [ ] **Step 2: Populate `gms_v87.json` for the commerce FNames from Phase 1.** Workflow per Task 8 Step 2; if the file doesn't exist, this task creates it from scratch.

- [ ] **Step 3: Re-run the audit.**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v87.json \
  --output           docs/packets/audits/gms_v87
```

- [ ] **Step 4: Triage divergences** (per Task 8 Step 4). v87 is structurally close to v95 in cash-shop; expect minor opcode/template drift, not wire-shape rewrites.

- [ ] **Step 5: Per-fix commits + bucket commit.**

```bash
git add docs/packets/ida-exports/gms_v87.json \
        docs/packets/audits/gms_v87/
git commit -m "audit(commerce): GMS v87 cross-version pass (commerce domain)"
```

- [ ] **Step 6: Hard-cap check** (per Task 8 Step 6 with the same awk loop).

---

### Task 10: JMS v185 cross-version pass

JMS v185 had a separate opcode space for login/character (task-027/028 finding) and PRD §9 q3 flags JMS NX point system as uncharted territory. Expect heavier divergence than GMS v83/v87 in cash-shop specifically; storage/interaction/inventory are likely closer to GMS shape.

**Files:**
- Create or modify: `docs/packets/ida-exports/gms_jms_185.json`
- Modify (per fix): `libs/atlas-packet/<domain>/<dir>/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Create or modify: `docs/packets/audits/jms_v185/<domain>/<pkt>.{md,json}`
- Modify: `docs/packets/audits/jms_v185/SUMMARY.md`
- Append (if deferrals): `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Confirm JMS v185 IDA is loaded** (`mcp__ida-pro__get_metadata`).

- [ ] **Step 2: Populate `gms_jms_185.json` for the commerce FNames from Phase 1.**

If a FName has no JMS equivalent, record the JMS-side FName + address as a separate entry annotated with `"region": "JMS"` and `"version": 185`. Do NOT reuse GMS FNames for unrelated JMS functions.

- [ ] **Step 3: Re-run the audit.**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_jms_185.json \
  --output           docs/packets/audits/jms_v185
```

- [ ] **Step 4: Triage per design §10 / §13 in-scope rules:**
  - In scope: atlas-packet writes bytes the JMS client decodes wrong.
  - Out of scope: JMS-specific cash-shop feature the service doesn't wire through (JMS-only payment category, JMS-only SKU) → `_pending.md` row + sibling-task suggestion.
  - In scope: width mismatch on a field both versions decode.
  - Out of scope: JMS template opcode wrong when v95 is right → fix the template, atlas-packet untouched.
  - **Hard cap:** if the JMS-only NX system requires a 3+ gate per encoder (design §9), STOP — `_pending.md` row + sibling task. Don't ship a 3-gate encoder under cover of an audit task.

- [ ] **Step 5: Per-fix commits + bucket commit.**

```bash
git add docs/packets/ida-exports/gms_jms_185.json \
        docs/packets/audits/jms_v185/
git commit -m "audit(commerce): JMS v185 cross-version pass (commerce domain)"
```

- [ ] **Step 6: Hard-cap check** (per Task 8 Step 6).

---

## Phase 3 — Login + character + social regression confirm

Mechanical re-run of the existing login (task-027), character (task-028), and social (task-066) audits. No verdict regression vs the snapshot recorded in their `post-phase-b.md` files. Cap: 2 new ❌s across login + character + social is the budget before stop-and-split (design §10 Phase 3). The Phase 0 §8 registry extension (`EncodeBytes` / `EncodeEntry`) is the single most likely source of regression — if recognition surfaces a previously-hidden recurse mismatch in a login/character/social packet, that flip is in-scope to triage.

### Task 11: Re-run login + character + social audits + assert no verdict regression

**Files:**
- Modify (read-only assertion; should not change unless regression): `docs/packets/audits/gms_v95/SUMMARY.md`
- Create (if regression diagnosis needed): `docs/tasks/task-067-commerce-domain-packet-audit/regression-notes.md`

- [ ] **Step 1: Snapshot the current SUMMARY verdicts before re-run.**

```bash
cp docs/packets/audits/gms_v95/SUMMARY.md /tmp/summary-pre-phase3.md
```

- [ ] **Step 2: Re-run the v95 audit (full pipeline; covers login + character + social + commerce).**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

- [ ] **Step 3: Diff the SUMMARY against the pre-Phase-3 snapshot.**

```bash
diff /tmp/summary-pre-phase3.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: empty (no diff) for login (`atlas-packet/login/`), character (`atlas-packet/character/`), and social (`atlas-packet/{guild,party,buddy,messenger,note,chat}/`) rows.

If a login/character/social row's verdict changed:
- **Verdict regressed (✅ → ❌ or ✅ → ⚠️ for any prior-task row)** → in-scope to triage. For each regressed packet:
  - Read the new audit report (`docs/packets/audits/gms_v95/<area>/<PacketName>.md`).
  - Decompile via `mcp__ida-pro__decompile_function`.
  - Identify whether a Phase 1/2 commerce fix's gate or the Phase 0 registry extension accidentally affected the prior-task encoder (e.g. a shared sub-struct, a shared template entry, a recurse that now resolves bytes it previously skipped).
  - Fix as a one-commit follow-up here, with a 4-variant sweep in the affected `libs/atlas-packet/<area>/`.
  - Cap: 2 regressions across login + character + social is the budget. 3+ → STOP. Document each in a new `regression-notes.md` and ask the user to spin up a sibling task. Do not proceed to Phase 4 until cleared.
- **Verdict improved (❌ → ✅)** → no action; record in Phase 4 `post-phase-b.md` "Tooling improvements" section as a side-effect win. This is the expected outcome if the Phase 0 registry extension exposes correct sub-struct recursion that previously read as opaque `WriteByteArray`.

- [ ] **Step 4: Commit the (possibly unchanged) SUMMARY + any regression-fix commits individually.**

If no diff, no commit needed for the regression check itself. If fix commits surfaced in Step 3, each commits as `fix(atlas-packet,<area>/<pkt>): <reason>` per the per-fix recipe. After all regression fixes:

```bash
git add docs/packets/audits/gms_v95/SUMMARY.md
git commit -m "audit(commerce): no login/character/social verdict regression"
```

(Skip this commit if Step 3's diff was empty AND no fix commits ran.)

---

## Phase 4 — Closeout

### Task 12: `post-phase-b.md`, full verification, code review, PR

**Files:**
- Create: `docs/tasks/task-067-commerce-domain-packet-audit/post-phase-b.md`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (final tally row, if not already present from earlier sub-tasks)
- Modify: `docs/packets/audits/gms_v95/_pending.md` (final commerce-domain section state)

- [ ] **Step 1: Write `post-phase-b.md`.**

Mirror task-027/028/066 structure verbatim. Five sections:

```markdown
# Task-067 Post-Phase-B — Commerce-Domain Audit Closeout

## Final state
- Source files audited: 78 (15 clientbound + 63 serverbound across cash, interaction, inventory, storage).
- Wire shapes audited: ~89 (per design §3 — multi-shape files itemised in §3.1).
- Verdicts: ✅ <n_pass> / ⚠️ <n_warn> / ❌ <n_fail> / 🔍 <n_review> / pending <n_pending>.
- IDA-export coverage: v83 / v87 / v95 / JMS v185 — commerce FNames populated.

## Real wire bugs fixed
| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
(one row per fix commit from Phase 1/2/3)

## Template opcode / enum fixes
| Template file | Old → New | IDA case-statement | Reason |
|---|---|---|---|

## Tooling improvements
- Registry extension recognising `EncodeBytes` (flat) and `EncodeEntry` (closure) method names (Phase 0 Task 2).
- Registry fixture asserting commerce sub-struct coverage (`CashInventoryItem`, `AddEntry`, `QuantityUpdateEntry`, `MoveEntry`, `RemoveEntry`).
- Documented cash constructor ↔ struct map in `phase-0-survey.md` (Task 3).
- Documented OP-FAMILY rows for interaction / storage dispatchers + INTERACTION-MODE-MAP + (if applicable) INTERACTION-CB-MODE-MAP.
- Documented "Sub-op enum / sub-struct deferrals — commerce domain" consolidated deferral row.
- (Any side-effect login/character/social verdict improvements from Phase 3.)

## Remaining work
| Area | What | Why deferred |
|---|---|---|
(rows from `_pending.md` commerce-domain section + any §9 hard-cap stops + bare-handler families)
```

Fill in actual numbers and rows from the commit history:

```bash
BASE=$(git merge-base main HEAD)
git log --oneline ${BASE}..HEAD | grep -E "^[0-9a-f]+ (fix|audit|feat)" > /tmp/commerce-commits.txt
```

- [ ] **Step 2: Run the four PRD §10 verification commands.**

```bash
go build ./...
go vet ./libs/atlas-packet/...
go test -race ./libs/atlas-packet/...
go test -race ./tools/packet-audit/...
```

All four must be clean. If `go test -race ./libs/atlas-packet/...` fails, the most likely cause is a 4-variant test asserting a hex value that needs updating after a fix landed late in Phase 2 — re-read the failing test's hex constants against the current encoder before retrying.

- [ ] **Step 3: Decide whether `docker build` is required.**

Per CLAUDE.md Build & Verification §3: required when a service `Dockerfile` or `go.mod` was touched. This task is expected to touch only `template_*.json` files under `services/atlas-configurations/seed-data/`. Confirm:

```bash
BASE=$(git merge-base main HEAD)
git diff --name-only ${BASE}..HEAD -- services/atlas-configurations/ | grep -v 'seed-data/templates/'
git diff --name-only ${BASE}..HEAD -- services/atlas-cashshop/ services/atlas-inventory/ services/atlas-storage/
```

If both commands return empty: skip `docker build`. Otherwise:

```bash
docker build -f services/atlas-configurations/Dockerfile .
docker build -f services/atlas-cashshop/Dockerfile .
docker build -f services/atlas-inventory/Dockerfile .
docker build -f services/atlas-storage/Dockerfile .
```

Run each command for any service whose `go.mod` or `Dockerfile` was touched. Expected: clean. If it fails on workspace replace lines, the affected Dockerfile needs its four `Chronicle20/atlas/libs/atlas-*` blocks updated (the go.mod stage `COPY`s, the synthesized `go.work use(...)` block, the source `COPY`s, and the explicit `go mod edit -replace=...` flags per CLAUDE.md) — fix and re-run.

PRD §8 ripple watch: any encoder constructor that gained a field rippling into `atlas-cashshop` / `atlas-inventory` / `atlas-channel` handler call sites lands a rebuild requirement; check via `git diff ${BASE}..HEAD -- services/atlas-{cashshop,inventory,channel}/` for actual code changes, not just go.mod/Dockerfile touches.

- [ ] **Step 4: gitleaks scrub.**

```bash
grep -r '/home/' docs/packets/audits/gms_v95/{cash,interaction,inventory,storage}/ \
                 docs/packets/audits/gms_v83/{cash,interaction,inventory,storage}/ \
                 docs/packets/audits/gms_v87/{cash,interaction,inventory,storage}/ \
                 docs/packets/audits/jms_v185/{cash,interaction,inventory,storage}/ \
                 2>/dev/null
```

Expected: no output. If any user-home path appears in an audit report, scrub it and commit:

```bash
sed -i 's|/home/[^/]*/source/atlas-ms/atlas/||g' <file>
git commit -am "audit: scrub absolute user-home paths from commerce/* reports"
```

Also scrub `docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md`:

```bash
grep -n '/home/' docs/tasks/task-067-commerce-domain-packet-audit/phase-0-survey.md
```

Expected: no output. If output appears, sed-scrub and commit similarly.

- [ ] **Step 5: Commit `post-phase-b.md`.**

```bash
git add docs/tasks/task-067-commerce-domain-packet-audit/post-phase-b.md \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/audits/gms_v95/_pending.md
git commit -m "docs(task-067): post-phase-b closeout"
```

- [ ] **Step 6: Run code review.**

Invoke `superpowers:requesting-code-review`. Allow the orchestration skill to dispatch:
- `plan-adherence-reviewer` — verifies every checkbox in this plan has commit evidence.
- `backend-guidelines-reviewer` — DOM-* Go audit on `libs/atlas-packet/` and `tools/packet-audit/` changes.

Read the resulting `audit.md` and act on every BLOCKER / MAJOR finding before opening a PR. Re-run reviews after fix commits land.

- [ ] **Step 7: Open the PR.**

Title: `task-067: commerce-domain packet audit (v83/v87/v95/JMS185)`

Body: short summary + link to `post-phase-b.md` for the full bug ledger. Use `superpowers:finishing-a-development-branch` to drive the PR creation.

---

## Self-review notes

Run through the plan once more with fresh eyes before committing it.

- **Spec coverage** — every PRD §4 functional requirement is covered by an explicit task above:
  - §4.1 coverage matrix (78 src files / ~89 wire shapes per design §3) → Phase 1 (Tasks 4–7).
  - §4.2 IDA exports → Phase 1 v95 + Phase 2 v83/v87/JMS-185 (Tasks 4–10).
  - §4.3 wire-bug fixes → embedded per-fix recipe in every Phase 1 + Phase 2 sub-task.
  - §4.4 template fixes → embedded per-fix recipe.
  - §4.5 TypeRegistry extensions → Phase 0 (Tasks 1–2) registry extension for `EncodeBytes` + `EncodeEntry`.
  - §4.6 cross-version re-verification → Phase 2 (Tasks 8–10).
  - §4.7 deferral handling → embedded triage rules in Phase 1 preamble; `_pending.md` headings created in Task 4 (first sub-task that hits them).

- **Acceptance criteria coverage** — every PRD §10 acceptance bullet maps to a Task:
  - "all 78 listed packet src files have audit reports" — Phase 1 exit gates (Tasks 4–7).
  - "every ❌ verdict has either a fix commit OR a `_pending.md` row" — Phase 1 sub-task exit gates.
  - "all 4 verification commands pass cleanly" — Task 12 Step 2.
  - "docker build clean for any service whose go.mod or Dockerfile was touched" — Task 12 Step 3.
  - "gitleaks scrub clean" — Task 12 Step 4.
  - "post-phase-b.md ledger written" — Task 12 Step 1.
  - "plan-adherence-reviewer and backend-guidelines-reviewer dispatched" — Task 12 Step 6.
  - "login (task-027), character (task-028), social (task-066) verdicts unchanged" — Phase 3 (Task 11).

- **No placeholders** — every step contains either an exact command, an exact code block, or an exact file path. No "TBD" / "similar to" / "fill in".

- **Type consistency** — `CashInventoryItem`, `AddEntry`, `QuantityUpdateEntry`, `MoveEntry`, `RemoveEntry`, `model.Asset` are referenced consistently across Tasks 1, 2, 5, 7. Audit-output paths `docs/packets/audits/gms_v95/<domain>/` (four sub-domains) are consistent across Tasks 4–7. IDA-export filenames `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` consistent across Tasks 8–10. Template filenames `template_gms_83_1.json`, `template_gms_87_1.json`, `template_jms_185_1.json` consistent.

- **Loop-internal early-return / analyzer surgery is explicitly out of scope** per design §1. Phase 0 extends the registry method-name switch only (one additive method-name match per non-`Encode`/`Write` name); the analyzer's call-collection logic is unchanged.

- **Sub-op enum drift** — Phase 1 preamble defers cash operations/errors enum + interaction mode-byte map + storage operation mode bytes + inventory ChangeMode to `_pending.md`. Single row per *cause*, not per file (design §11). Encoder change for these is forbidden in this task.

- **Hot-path discipline** — Task 4 (storage `update_assets.go`) and Task 5 (inventory `change.go` four shapes + serverbound `move.go` symmetry) call out 4-variant byte-output sweep + IDA dispatcher offset citation per design §6/§7.

- **No `reflect`, no `interface{}`, no benchmarks** — none of the code in the plan uses `reflect.*` or adds an `interface{}` parameter to an encoder. Per-fix recipe Step 3 is byte-output assertion, not benchmark.

- **2-nested-guard hard cap** — Phase 2 hard-cap check (Task 8/9/10 Step 6) enforces it via an awk scan. 3+ → `_pending.md`, no refactor.

- **Bucket commit cadence** — Tasks 4–7 each produce 0–N fix commits before the bucket commit. Maintain ordering: fixes first, audit-report bucket commit last, so the bucket reflects post-fix state.

- **Worktree discipline** — every task ends with the `git rev-parse --show-toplevel` + `git branch --show-current` check baked into the conventions section.

- **Gitleaks scrub** — Task 12 Step 4 is mandatory, covers `docs/packets/audits/gms_{v95,v83,v87}/{cash,interaction,inventory,storage}/` and `docs/packets/audits/jms_v185/...` and `phase-0-survey.md`.

- **Docker build** — Task 12 Step 3 explicitly enumerates `atlas-configurations`, `atlas-cashshop`, `atlas-inventory`, `atlas-storage` Dockerfiles as candidates whose `go.mod` / `Dockerfile` may have been touched by constructor ripple. PRD §8 mandates per-service `docker build` when either is touched.
