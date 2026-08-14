# Summoning Sack & Town Scroll Opcode Registration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make summoning sacks and town/return scrolls actually dispatch on every supported client version by binding their serverbound opcodes in the tenant seed templates, and promote both op-rows of the packet coverage matrix from `incomplete` to `✅` with real, per-op, per-version IDA evidence.

**Architecture:** Two audit-only wrapper codecs (`SummonBagItemUse`, `ReturnScrollItemUse`) embed the existing shared `inventory/serverbound.ItemUse` body so each op gets its own packet id, fixture and evidence key without touching the production decode path. The registry gains a `packet:` link per op per version (matrix grading Path B), the seed templates gain the missing handler bindings, and each op × version cell is promoted by harvesting its real send site out of that version's IDB, splicing it into the committed export, pinning evidence and adding a `packet-audit:verify` marker.

**Tech Stack:** Go 1.x (`libs/atlas-packet`, `tools/packet-audit`), JSON seed templates (`services/atlas-configurations/seed-data/templates/`), YAML op registry + evidence ledger (`docs/packets/`), IDA Pro via `ida-pro-mcp` session server.

**Spec:** `docs/tasks/task-229-summon-bag-town-scroll-opcodes/design.md` (PRD: `prd.md` in the same folder)

---

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-229-summon-bag-town-scroll-opcodes` on branch `task-229-summon-bag-town-scroll-opcodes`. Never edit the main checkout.
- **No production behaviour change.** `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go` and `services/atlas-consumables` are NOT modified. The wire body in `libs/atlas-packet/inventory/serverbound/item_use.go` is NOT modified (only the file's neighbours are added).
- **Zero diff on already-complete templates.** `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json` must be byte-identical at the end. (`template_gms_48_1.json` gains only an `fname` on an existing entry, plus possibly one new entry from Task 15.)
- **`template_gms_12_1.json` is out of scope.** It carries no item-use handlers at all and is not a matrix column.
- **Out of scope:** `ShopScannerItemUseHandle` (missing on gms_87, jms_185) and `CharacterItemUseLotteryHandle` (missing on gms_92).
- **The committed IDA exports are never regenerated.** `docs/packets/ida-exports/*.json` may only be edited by surgical hand-splice of the specific function entries you harvested. `packet-audit export -splice` is BANNED — it round-trips the whole file through a struct that drops unrecognized fields (`region`, `note`) and reindents, corrupting ~20 unrelated entries. Verify every export edit with `git diff docs/packets/ida-exports/<v>.json` showing only your entries.
- **IDA endpoint.** The live session server is `http://192.168.20.3:8745/mcp`. `packet-audit`'s `-ida-url` default (`http://192.168.20.3:13337/mcp`) is a dead port that answers with the WRONG binary instead of erroring — pass `-ida-url` explicitly on every invocation. Select the IDB with `-ida-database <session_id>`; `-ida-port` / `select_instance` are dead.
- **Session ids shift.** Always run `idb_list` first and match by `filename`. Sessions observed 2026-08-14 (re-resolve, do not trust these):
  `MapleStory_dump.exe.i64`=gms_v83, `GMS_v61.1_U_DEVM.exe.i64`=gms_v61, `GMS_v72.1_U_DEVM.exe.i64`=gms_v72, `GMS_v79_1_DEVM.exe.i64`=gms_v79, `GMS_v84.1_U_DEVM.i64`=gms_v84, `GMSv87_4GB.exe.i64`=gms_v87, `GMS_v92_1_DEVM.exe.i64`=gms_v92, `GMS_v95.0_U_DEVM.exe.i64`=gms_v95, `MapleStory_dump_SCY.exe.i64`=jms_v185, `GMS_v48_1_DEVM.exe.i64`=gms_v48.
- **Distrust IDB symbol names.** Ground truth for a send op is the integer in `COutPacket::COutPacket(&pkt, OPCODE)`. Byte signature to locate one: `6A <op> 8D 8D ?? ?? ?? ?? E8` or `6A <op> 8D 4D ?? E8`.
- **Name every unnamed sender in the IDB** while you are there (`mcp__ida-pro__rename`). An unnamed sub is a producible prerequisite, not a blocker.
- **Export path exception:** jms_v185's export file is `docs/packets/ida-exports/gms_jms_185.json` (not `jms_v185.json`); its audit/evidence dirs are `jms_v185`.
- **A cell that does not mechanically promote in `docs/packets/audits/status.json` is a failure**, reported as such. No prose claim substitutes.
- **Baseline is green.** `go run ./tools/packet-audit matrix --check` exits 0 on the branch tip as of 2026-08-14. Any non-zero exit you see later is yours.

### Resolved opcodes (read out of `docs/packets/registry/*.yaml`, 2026-08-14)

| version | USE_ITEM | USE_SUMMON_BAG | USE_RETURN_SCROLL | USE_UPGRADE_SCROLL | PET_FOOD |
|---|---|---|---|---|---|
| gms_v48 | 65 `0x41` | *(unregistered; template binds `0x3B`)* | *(unregistered — Task 15)* | 66 `0x42` | 60 `0x3C` |
| gms_v61 | 67 `0x43` | 70 `0x46` | 78 `0x4E` | 79 `0x4F` | 71 `0x47` |
| gms_v72 | 71 `0x47` | 74 `0x4A` | 84 `0x54` | 85 `0x55` | 75 `0x4B` |
| gms_v79 | 70 `0x46` | 73 `0x49` | 83 `0x53` | 84 `0x54` | 74 `0x4A` |
| gms_v83 | 72 `0x48` | 75 `0x4B` | 85 `0x55` | 86 `0x56` | 76 `0x4C` |
| gms_v84 | 72 `0x48` | 75 `0x4B` | 85 `0x55` | 86 `0x56` | 76 `0x4C` |
| gms_v87 | 75 `0x4B` | 78 `0x4E` | 88 `0x58` | 89 `0x59` | 79 `0x4F` |
| gms_v92 | 79 `0x4F` | 82 `0x52` | 92 `0x5C` | 93 `0x5D` | 83 `0x53` |
| gms_v95 | 78 `0x4E` | 81 `0x51` | 92 `0x5C` | 93 `0x5D` | 82 `0x52` |
| jms_v185 | 64 `0x40` | 67 `0x43` | 77 `0x4D` | 78 `0x4E` | 68 `0x44` |

### Registry primary fnames for the two target ops (these are what `candidatesFromFName` must key on)

| version | USE_SUMMON_BAG `fname` | USE_RETURN_SCROLL `fname` |
|---|---|---|
| gms_v61 | `CWvsContext::SendMobSummonItemUseRequest` | `sub_841AA5` |
| gms_v72 | `sub_955499` | `CWvsContext::SendReturnScrollUseRequest` |
| gms_v79 | `sub_955499` | `CWvsContext::SendReturnScrollUseRequest` |
| gms_v83 / v84 / v87 / v92 / v95 / jms_v185 | `CWvsContext::SendMobSummonItemUseRequest` | `CWvsContext::SendPortalScrollUseRequest` |
| gms_v48 | *(to be created — Task 15)* | *(to be created or proven absent — Task 15)* |

**None of these five fnames exists in ANY committed export** (verified 2026-08-14 by loading each `docs/packets/ida-exports/*.json` and testing key membership). Every one of the 20 cells therefore needs a harvest before `evidence pin` can succeed. This is the cost centre; it is unavoidable.

---

## File Structure

**Created**

| File | Responsibility |
|---|---|
| `libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go` | Audit-only wrapper codec `SummonBagItemUse`; carries the `packet-audit:fname` marker and the packet id `inventory/serverbound/InventorySummonBagItemUse`. |
| `libs/atlas-packet/inventory/serverbound/summon_bag_item_use_test.go` | Round-trip fixture + one `packet-audit:verify` marker per verified version. |
| `libs/atlas-packet/inventory/serverbound/return_scroll_item_use.go` | Audit-only wrapper codec `ReturnScrollItemUse`; packet id `inventory/serverbound/InventoryReturnScrollItemUse`. |
| `libs/atlas-packet/inventory/serverbound/return_scroll_item_use_test.go` | Round-trip fixture + verify markers. |
| `docs/packets/evidence/<version>/inventory.serverbound.InventorySummonBagItemUse.yaml` | Pinned evidence, one per version (tool-written by `evidence pin` — never hand-edited). |
| `docs/packets/evidence/<version>/inventory.serverbound.InventoryReturnScrollItemUse.yaml` | Ditto. |
| `docs/packets/audits/<version>/InventorySummonBagItemUse.{json,md}` | Generated audit report. |
| `docs/packets/audits/<version>/InventoryReturnScrollItemUse.{json,md}` | Generated audit report. |

**Modified**

| File | Change |
|---|---|
| `tools/packet-audit/cmd/run.go` | New `candidatesFromFName` cases mapping each registry fname to its wrapper struct. |
| `tools/packet-audit/cmd/disambiguation_test.go` | Table cases asserting those mappings. |
| `docs/packets/registry/gms_v{61,72,79,83,84,87,92,95}.yaml`, `jms_v185.yaml` | `packet:` field on the two ops. |
| `docs/packets/registry/gms_v48.yaml` | New `USE_SUMMON_BAG` entry; new `USE_RETURN_SCROLL` entry **or** nothing (Task 15). |
| `docs/packets/ida-exports/*.json` | Surgical splice of the harvested sender entries. |
| `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` | +2 handlers. |
| `…/template_gms_92_1.json` | +5 handlers. |
| `…/template_gms_95_1.json` | +2 handlers. |
| `…/template_jms_185_1.json` | +2 handlers. |
| `…/template_gms_48_1.json` | Task 15 only: `fname` backfilled on the existing SummonBag entry; +1 handler if a return-scroll sender is found. |
| `libs/atlas-packet/inventory/serverbound/item_use_test.go` | +1 verify marker (gms_v92). |
| `libs/atlas-packet/inventory/serverbound/scroll_use_test.go` | +1 verify marker (gms_v92). |
| `libs/atlas-packet/pet/serverbound/food_test.go` | +1 verify marker (gms_v92). |
| `docs/packets/feature-na-evidence.yaml` | Positive absence entry if Task 15 concludes `n-a`. |
| `docs/packets/audits/STATUS.md`, `status.json` | Regenerated. |

**Untouched (assert this):** `libs/atlas-packet/inventory/serverbound/item_use.go`, `services/atlas-channel/**`, `services/atlas-consumables/**`.

---

## Task 1: Audit-only wrapper codecs

**Files:**
- Create: `libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go`
- Create: `libs/atlas-packet/inventory/serverbound/summon_bag_item_use_test.go`
- Create: `libs/atlas-packet/inventory/serverbound/return_scroll_item_use.go`
- Create: `libs/atlas-packet/inventory/serverbound/return_scroll_item_use_test.go`
- Reference (do NOT modify): `libs/atlas-packet/inventory/serverbound/item_use.go`

**Interfaces:**
- Consumes: `ItemUse` (`item_use.go:20-59`) — private fields `updateTime uint32`, `source int16`, `itemId uint32`; getters `UpdateTime() uint32`, `Source() int16`, `ItemId() uint32`; `Encode(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`; `Decode(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{})` (pointer receiver). Handler-name constants `CharacterItemUseSummonBagHandle`, `CharacterItemUseTownScrollHandle` already live in `item_use.go:14-17` — do NOT redeclare them.
- Produces: `SummonBagItemUse` and `ReturnScrollItemUse`, each with an embedded `ItemUse`, an `Operation() string`, and promoted `Encode`/`Decode`. Packet ids `inventory/serverbound/InventorySummonBagItemUse` and `inventory/serverbound/InventoryReturnScrollItemUse` (used by every later task's marker/evidence/report path).

Packet id derivation: `qualifiedWriterName(pkg, name)` = TitleCase(pkg) + struct name, so pkg `inventory` + `SummonBagItemUse` → `InventorySummonBagItemUse`, and the marker path is `inventory/serverbound/InventorySummonBagItemUse`.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-packet/inventory/serverbound/summon_bag_item_use_test.go`:

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// SummonBagItemUse is an audit-only wrapper: it exists so USE_SUMMON_BAG gets
// its own packet id, audit report and evidence key instead of borrowing
// InventoryItemUse's (which pins the *potion* sender
// CWvsContext::SendStatChangeItemUseRequest). The production handler in
// atlas-channel keeps decoding the shared ItemUse directly. See task-229 and
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops".
//
// Byte fixtures with `packet-audit:verify` markers are appended per version by
// the per-version verification tasks; this round-trip covers every tenant
// variant.
func TestSummonBagItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SummonBagItemUse{ItemUse: ItemUse{
				operation:  CharacterItemUseSummonBagHandle,
				updateTime: 12345,
				source:     5,
				itemId:     2100000,
			}}
			output := SummonBagItemUse{ItemUse: ItemUse{operation: CharacterItemUseSummonBagHandle}}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Operation() != CharacterItemUseSummonBagHandle {
				t.Errorf("operation: got %v, want %v", output.Operation(), CharacterItemUseSummonBagHandle)
			}
		})
	}
}

// The wrapper must not drift from the shared body: byte-for-byte identical
// encodings are the whole point of embedding rather than redeclaring.
func TestSummonBagItemUseMatchesSharedBody(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	shared := ItemUse{operation: CharacterItemUseSummonBagHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 0x14151617}
	wrapped := SummonBagItemUse{ItemUse: shared}
	a := shared.Encode(nil, ctx)(nil)
	b := wrapped.Encode(nil, ctx)(nil)
	if len(a) != len(b) {
		t.Fatalf("len: shared %d, wrapper %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("bytes: shared % X, wrapper % X", a, b)
		}
	}
}
```

Create `libs/atlas-packet/inventory/serverbound/return_scroll_item_use_test.go` — the same two tests with `ReturnScrollItemUse`, `CharacterItemUseTownScrollHandle`, and `itemId: 2030000`:

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ReturnScrollItemUse is an audit-only wrapper: it exists so USE_RETURN_SCROLL
// gets its own packet id, audit report and evidence key instead of borrowing
// InventoryItemUse's (which pins the *potion* sender). The production handler in
// atlas-channel keeps decoding the shared ItemUse directly. See task-229 and
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops".
func TestReturnScrollItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ReturnScrollItemUse{ItemUse: ItemUse{
				operation:  CharacterItemUseTownScrollHandle,
				updateTime: 12345,
				source:     5,
				itemId:     2030000,
			}}
			output := ReturnScrollItemUse{ItemUse: ItemUse{operation: CharacterItemUseTownScrollHandle}}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Operation() != CharacterItemUseTownScrollHandle {
				t.Errorf("operation: got %v, want %v", output.Operation(), CharacterItemUseTownScrollHandle)
			}
		})
	}
}

func TestReturnScrollItemUseMatchesSharedBody(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	shared := ItemUse{operation: CharacterItemUseTownScrollHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 0x14151617}
	wrapped := ReturnScrollItemUse{ItemUse: shared}
	a := shared.Encode(nil, ctx)(nil)
	b := wrapped.Encode(nil, ctx)(nil)
	if len(a) != len(b) {
		t.Fatalf("len: shared %d, wrapper %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("bytes: shared % X, wrapper % X", a, b)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-packet && go test ./inventory/serverbound/ -run 'SummonBagItemUse|ReturnScrollItemUse' -v
```

Expected: FAIL — `undefined: SummonBagItemUse`, `undefined: ReturnScrollItemUse`.

- [ ] **Step 3: Write the wrapper codecs**

`libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go`:

```go
package serverbound

// SummonBagItemUse is an AUDIT-ONLY codec. USE_SUMMON_BAG shares its wire body
// with USE_ITEM and USE_RETURN_SCROLL (Encode4 updateTime + Encode2 slot +
// Encode4 itemId), but each op has a DIFFERENT client send site. Collapsing all
// three onto InventoryItemUse's packet id would pin every cell's evidence to the
// potion sender's decompile — a manufactured ✅. One wrapper per op = one packet
// id, one audit report and one evidence key per op, exactly as
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops" prescribes.
//
// Nothing calls this: atlas-channel's CharacterItemUseSummonBagHandleFunc keeps
// decoding the shared ItemUse. Do not "simplify" it away (task-229).
//
// packet-audit:fname CWvsContext::SendMobSummonItemUseRequest
type SummonBagItemUse struct {
	ItemUse
}

func NewSummonBagItemUse() SummonBagItemUse {
	return SummonBagItemUse{ItemUse: NewItemUse(CharacterItemUseSummonBagHandle)}
}
```

`libs/atlas-packet/inventory/serverbound/return_scroll_item_use.go`:

```go
package serverbound

// ReturnScrollItemUse is an AUDIT-ONLY codec. USE_RETURN_SCROLL shares its wire
// body with USE_ITEM and USE_SUMMON_BAG (Encode4 updateTime + Encode2 slot +
// Encode4 itemId) but has its own client send site — on v61/v72/v79 that sender
// reads the Return Scroll WZ props and is distinct from both the potion sender
// and the teleport-rock sender (see the corrections recorded in
// docs/packets/registry/gms_v72.yaml and gms_v79.yaml). One wrapper per op = one
// packet id, one audit report and one evidence key per op, per
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops".
//
// Nothing calls this: atlas-channel's CharacterItemUseTownScrollHandleFunc keeps
// decoding the shared ItemUse. Do not "simplify" it away (task-229).
//
// packet-audit:fname CWvsContext::SendPortalScrollUseRequest
type ReturnScrollItemUse struct {
	ItemUse
}

func NewReturnScrollItemUse() ReturnScrollItemUse {
	return ReturnScrollItemUse{ItemUse: NewItemUse(CharacterItemUseTownScrollHandle)}
}
```

Note: `Operation()`, `Encode()`, `Decode()`, `String()` and the getters are all promoted from the embedded `ItemUse` — do not redeclare them.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-packet && go test ./inventory/serverbound/ -run 'SummonBagItemUse|ReturnScrollItemUse' -v
```

Expected: PASS, one subtest per entry in `pt.Variants` (GMS v28/v83/v87/v95/v84/v86/v48/v61/v72/v79/v92 and JMS v185).

- [ ] **Step 5: Run the full module test + vet**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./...
```

Expected: PASS / no output.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go \
        libs/atlas-packet/inventory/serverbound/summon_bag_item_use_test.go \
        libs/atlas-packet/inventory/serverbound/return_scroll_item_use.go \
        libs/atlas-packet/inventory/serverbound/return_scroll_item_use_test.go
git commit -m "feat(task-229): audit-only SummonBagItemUse and ReturnScrollItemUse wrappers"
```

---

## Task 2: Link the registry fnames to the wrapper codecs

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (insert next to the existing item-use cases at ~`run.go:2196-2201`)
- Modify: `tools/packet-audit/cmd/disambiguation_test.go` (append a new test func)

**Interfaces:**
- Consumes: `candidate{name, pkg string, dir csvpkg.Direction}` and the `candidatesFromFName(fname string) []candidate` switch in `run.go`; struct names `SummonBagItemUse` / `ReturnScrollItemUse` from Task 1.
- Produces: report generation and packet-id resolution for both ops on gms_v61…jms_v185. (gms_v48's fnames are added in Task 15.)

Why this is needed: `locateAtlasFile` finds a codec by `type <name> struct` inside `<pkg>/serverbound/`, and the candidate comes from this switch keyed on the registry's **primary** `fname`. Without a case, report generation cannot find the struct and the report is never written (`VERIFYING_A_PACKET.md` §9 "Linkage").

- [ ] **Step 1: Write the failing test**

Append to `tools/packet-audit/cmd/disambiguation_test.go`:

```go
// task-229: USE_SUMMON_BAG and USE_RETURN_SCROLL share their wire body with
// USE_ITEM but have distinct client send sites, so each keys to its own audit
// wrapper struct in inventory/serverbound. The fnames below are the PRIMARY
// fnames those ops carry in docs/packets/registry/*.yaml — they differ per
// version (v72/v79 summon-bag is an unnamed sub; v61 return-scroll is an unnamed
// sub; v72/v79 return-scroll was renamed away from the SendMapTransferItemUse
// mislabel).
func TestCandidatesItemUseFamilyWrappers(t *testing.T) {
	cases := []struct {
		fname    string
		wantPkg  string
		wantName string
	}{
		{"CWvsContext::SendMobSummonItemUseRequest", "inventory", "SummonBagItemUse"},
		{"sub_955499", "inventory", "SummonBagItemUse"},
		{"CWvsContext::SendPortalScrollUseRequest", "inventory", "ReturnScrollItemUse"},
		{"CWvsContext::SendReturnScrollUseRequest", "inventory", "ReturnScrollItemUse"},
		{"sub_841AA5", "inventory", "ReturnScrollItemUse"},
		// Regression guard: the potion sender must still key to the shared body.
		{"CWvsContext::SendStatChangeItemUseRequest", "inventory", "ItemUse"},
	}
	for _, c := range cases {
		t.Run(c.fname, func(t *testing.T) {
			cands := candidatesFromFName(c.fname)
			if len(cands) == 0 {
				t.Fatalf("%s: no candidates", c.fname)
			}
			if cands[0].pkg != c.wantPkg || cands[0].name != c.wantName {
				t.Errorf("%s: got pkg=%q name=%q, want pkg=%q name=%q",
					c.fname, cands[0].pkg, cands[0].name, c.wantPkg, c.wantName)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd tools/packet-audit && go test ./cmd/ -run TestCandidatesItemUseFamilyWrappers -v
```

Expected: FAIL — five subtests report `no candidates` (only `SendStatChangeItemUseRequest` passes).

- [ ] **Step 3: Add the switch cases**

In `tools/packet-audit/cmd/run.go`, immediately after the existing
`case "CWvsContext::SendUpgradeItemUseRequest":` arm (~line 2200), insert:

```go
	// task-229 — USE_SUMMON_BAG / USE_RETURN_SCROLL. Same 3-field wire body as
	// USE_ITEM (Encode4 updateTime + Encode2 slot + Encode4 itemId) but distinct
	// client send sites, so each keys to its own audit-only wrapper struct in
	// inventory/serverbound. One wrapper per op = one packet id / report /
	// evidence key per op; see docs/packets/audits/VERIFYING_A_PACKET.md
	// "Shared-model ops".
	case "CWvsContext::SendMobSummonItemUseRequest":
		return []candidate{{name: "SummonBagItemUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "sub_955499":
		// USE_SUMMON_BAG on gms_v72 (@0x904154) and gms_v79 (@0x9555b0): the
		// summon-bag sender is UNNAMED in both IDBs and both registries carry
		// sub_955499 as the primary fname (the v79 IDB additionally mislabels the
		// function as SendEngagementRequest — opcode read from the body).
		return []candidate{{name: "SummonBagItemUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "CWvsContext::SendPortalScrollUseRequest":
		return []candidate{{name: "ReturnScrollItemUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "CWvsContext::SendReturnScrollUseRequest":
		// gms_v72 / gms_v79 primary fname. Renamed live in those IDBs on task-124
		// away from the inherited SendMapTransferItemUseRequest mislabel — that
		// symbol is the teleport-rock sender, a different op.
		return []candidate{{name: "ReturnScrollItemUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	case "sub_841AA5":
		// USE_RETURN_SCROLL on gms_v61 (send site @0x841cb8): unnamed in the IDB;
		// reads the Return Scroll WZ props (StringPool 2276/2277/2279) and is
		// distinct from USE_TELEPORT_ROCK (opcode 77 / sub_8327DB).
		return []candidate{{name: "ReturnScrollItemUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd tools/packet-audit && go test ./... && go vet ./...
```

Expected: all packages PASS (including the new subtests).

- [ ] **Step 5: Confirm the matrix is still clean**

```bash
go run ./tools/packet-audit matrix --check >/dev/null 2>&1; echo "EXIT=$?"
```

Expected: `EXIT=0`. (Nothing should move yet — the registry has no `packet:` link and no evidence exists.)

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/cmd/run.go tools/packet-audit/cmd/disambiguation_test.go
git commit -m "feat(task-229): key summon-bag and return-scroll fnames to their audit wrappers"
```

---

## Task 3: Declare the packet link in the op registry

**Files:**
- Modify: `docs/packets/registry/gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml`, `gms_v83.yaml`, `gms_v84.yaml`, `gms_v87.yaml`, `gms_v92.yaml`, `gms_v95.yaml`, `jms_v185.yaml`

**Interfaces:**
- Consumes: `opregistry.Entry.Packet` (`tools/packet-audit/internal/opregistry/opregistry.go:43-50`) — optional `packet:` yaml key.
- Produces: the Path-B linkage that lets a byte fixture + fresh evidence promote a cell with no audit report (`internal/matrix/grade.go:126-133`, `:206-215`), and suppresses the "dangling evidence" `--check` failure for these packets (`cmd/matrix.go:178-181`).

For each of the nine files, add exactly one line to the `USE_SUMMON_BAG` (serverbound) entry and one to the `USE_RETURN_SCROLL` (serverbound) entry. Keep the existing key order; append `packet:` after `provenance:`/`ida:`/`note:`.

- [ ] **Step 1: Add the `packet:` key to both ops in all nine registries**

For each file, the `USE_SUMMON_BAG` serverbound entry gains:

```yaml
  packet: inventory/serverbound/InventorySummonBagItemUse
```

and the `USE_RETURN_SCROLL` serverbound entry gains:

```yaml
  packet: inventory/serverbound/InventoryReturnScrollItemUse
```

Use per-file `Edit` calls (one edit per entry, anchored on the surrounding lines) — do NOT run a shell patch loop. Line anchors as of 2026-08-14:

| file | `USE_SUMMON_BAG` at | `USE_RETURN_SCROLL` at |
|---|---|---|
| `gms_v61.yaml` | 2294 | 2466 |
| `gms_v72.yaml` | 2338 | 2407 |
| `gms_v79.yaml` | 2911 | 2791 |
| `gms_v83.yaml` | 2230 | 2289 |
| `gms_v84.yaml` | 2891 | 2957 |
| `gms_v87.yaml` | 2364 | 2420 |
| `gms_v92.yaml` | 2571 | 2627 |
| `gms_v95.yaml` | 2582 | 2643 |
| `jms_v185.yaml` | 2307 | 2363 |

- [ ] **Step 2: Verify all eighteen edits landed**

```bash
grep -c "InventorySummonBagItemUse\|InventoryReturnScrollItemUse" docs/packets/registry/gms_v61.yaml docs/packets/registry/gms_v72.yaml docs/packets/registry/gms_v79.yaml docs/packets/registry/gms_v83.yaml docs/packets/registry/gms_v84.yaml docs/packets/registry/gms_v87.yaml docs/packets/registry/gms_v92.yaml docs/packets/registry/gms_v95.yaml docs/packets/registry/jms_v185.yaml
```

Expected: `2` for every file.

- [ ] **Step 3: Verify the registry still parses and the matrix is unchanged**

```bash
cd tools/packet-audit && go test ./... && cd ../..
go run ./tools/packet-audit matrix --check >/dev/null 2>&1; echo "EXIT=$?"
```

Expected: tests PASS, `EXIT=0`. The two rows stay `incomplete` — a `packet:` link with no evidence promotes nothing.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/registry
git commit -m "feat(task-229): link USE_SUMMON_BAG and USE_RETURN_SCROLL to their audit packets"
```

---

## Task 4: Bind the missing item-use handlers in the seed templates

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`

**Interfaces:**
- Consumes: the opcode table in Global Constraints; the handler-name constants already registered in `services/atlas-channel/atlas.com/channel/main.go` (no Go change needed).
- Produces: routed opcodes so `matrix` grades these cells `routed`, and so the client's requests actually dispatch.

Every new entry has exactly this shape (matching the gms_83 reference entries):

```json
{ "opCode": "0x4E", "validator": "LoggedInValidator", "handler": "CharacterItemUseSummonBagHandle", "fname": "CWvsContext::SendMobSummonItemUseRequest", "services": ["channel"] }
```

**Insert each entry at its strictly-ascending sorted position in the `handlers` array** — never appended next to a semantically-related entry (`tools/template-opcode-order-guard.sh`). `LoggedInValidator` is already declared in all five templates; a handler naming a missing validator is silently dropped at load time.

- [ ] **Step 1: gms_87 — two entries**

| opCode | handler | fname |
|---|---|---|
| `0x4E` | `CharacterItemUseSummonBagHandle` | `CWvsContext::SendMobSummonItemUseRequest` |
| `0x58` | `CharacterItemUseTownScrollHandle` | `CWvsContext::SendPortalScrollUseRequest` |

Both with `"validator": "LoggedInValidator"`, `"services": ["channel"]`.

- [ ] **Step 2: gms_92 — five entries**

| opCode | handler | fname |
|---|---|---|
| `0x4F` | `CharacterItemUseHandle` | `CWvsContext::SendStatChangeItemUseRequest` |
| `0x52` | `CharacterItemUseSummonBagHandle` | `CWvsContext::SendMobSummonItemUseRequest` |
| `0x53` | `PetFoodHandle` | `CWvsContext::SendPetFoodItemUseRequest` |
| `0x5C` | `CharacterItemUseTownScrollHandle` | `CWvsContext::SendPortalScrollUseRequest` |
| `0x5D` | `CharacterItemUseScrollHandle` | `CWvsContext::SendUpgradeItemUseRequest` |

All with `"validator": "LoggedInValidator"`, `"services": ["channel"]`. (gms_92's entire item-use block below `0x54` was unrouted — ordinary potion use, scroll use and pet food were all dead on that column, not just the two target ops. `PetFoodHandle` is design decision D3.)

- [ ] **Step 3: gms_95 — two entries**

| opCode | handler | fname |
|---|---|---|
| `0x51` | `CharacterItemUseSummonBagHandle` | `CWvsContext::SendMobSummonItemUseRequest` |
| `0x5C` | `CharacterItemUseTownScrollHandle` | `CWvsContext::SendPortalScrollUseRequest` |

- [ ] **Step 4: jms_185 — two entries**

| opCode | handler | fname |
|---|---|---|
| `0x43` | `CharacterItemUseSummonBagHandle` | `CWvsContext::SendMobSummonItemUseRequest` |
| `0x4D` | `CharacterItemUseTownScrollHandle` | `CWvsContext::SendPortalScrollUseRequest` |

- [ ] **Step 5: gms_48 — make NO change in this task**

`template_gms_48_1.json` already binds `CharacterItemUseSummonBagHandle` at `0x3B`, but that entry has **no `fname` key** while every sibling item-use entry does. Filling it in requires a symbol resolved from the v48 IDB, which is Task 15's job — do not guess one here. This step is deliberately a no-op; leave `template_gms_48_1.json` untouched and move on.

- [ ] **Step 6: Run the template guards**

```bash
tools/template-opcode-order-guard.sh; echo "ORDER=$?"
tools/template-duplicate-binding-guard.sh; echo "DUP=$?"
tools/template-movement-types-guard.sh; echo "MOVE=$?"
```

Expected: `ORDER=0 DUP=0 MOVE=0`.

- [ ] **Step 7: Confirm the untouched templates really are untouched**

```bash
git diff --stat services/atlas-configurations/seed-data/templates/
```

Expected: exactly four files changed (`template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`). If `template_gms_61_1.json`, `_72_`, `_79_`, `_83_` or `_84_` appears, revert it — that is a defect.

- [ ] **Step 8: Confirm the matrix stays clean**

```bash
go run ./tools/packet-audit matrix --check 2>&1 | tail -20; go run ./tools/packet-audit matrix --check >/dev/null 2>&1; echo "EXIT=$?"
```

Expected: `EXIT=0`. Routing alone promotes nothing — evidence is still missing — but a newly-routed opcode must not introduce a conflict anywhere.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-configurations/seed-data/templates
git commit -m "fix(task-229): bind the missing item-use opcodes on gms_87/92/95 and jms_185"
```

---

## Task 5: Promote the three v92 item-use rows that only need a fixture

**Files:**
- Modify: `libs/atlas-packet/inventory/serverbound/item_use_test.go`
- Modify: `libs/atlas-packet/inventory/serverbound/scroll_use_test.go`
- Modify: `libs/atlas-packet/pet/serverbound/food_test.go`
- Create (tool-written): `docs/packets/evidence/gms_v92/inventory.serverbound.InventoryItemUse.yaml`, `…InventoryScrollUse.yaml`, `docs/packets/evidence/gms_v92/pet.serverbound.PetFood.yaml`
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: the existing audit reports `docs/packets/audits/gms_v92/{InventoryItemUse,InventoryScrollUse,PetFood}.json` and the already-present export entries.
- Produces: three `verified` cells; no new codec.

`USE_ITEM`, `USE_UPGRADE_SCROLL` and `PET_FOOD` all read `partial — "tier-1: needs byte-fixture test to verify"` on gms_v92. The senders are already in `docs/packets/ida-exports/gms_v92.json`, the reports already exist, and Task 4 just routed the opcodes. All three need is a marker plus pinned evidence.

Export addresses (read from `gms_v92.json`, 2026-08-14):

| fname | address |
|---|---|
| `CWvsContext::SendStatChangeItemUseRequest` | `0x9b3600` |
| `CWvsContext::SendUpgradeItemUseRequest` | `0x9ab2f0` |
| `CWvsContext::SendPetFoodItemUseRequest` | `0x9afa50` |

- [ ] **Step 1: Confirm the read order in the v92 IDB before pinning**

Resolve the gms_v92 session (`idb_list` → `GMS_v92_1_DEVM.exe.i64`) and decompile each of the three addresses. Confirm each builds `COutPacket::COutPacket(&pkt, <opcode>)` with the opcode from the table (79 / 93 / 83 respectively) and that the field order matches the committed codec. **If an address or opcode disagrees, stop and report — do not pin.**

- [ ] **Step 2: Pin the three evidence records**

```bash
go run ./tools/packet-audit evidence pin \
  --packet inventory/serverbound/InventoryItemUse --version gms_v92 \
  --ida 'CWvsContext::SendStatChangeItemUseRequest' --category TIER1-FIXTURE \
  --verifies 'libs/atlas-packet/inventory/serverbound/item_use_test.go#TestItemUseRoundTrip'

go run ./tools/packet-audit evidence pin \
  --packet inventory/serverbound/InventoryScrollUse --version gms_v92 \
  --ida 'CWvsContext::SendUpgradeItemUseRequest' --category TIER1-FIXTURE \
  --verifies 'libs/atlas-packet/inventory/serverbound/scroll_use_test.go#TestScrollUseRoundTrip'

go run ./tools/packet-audit evidence pin \
  --packet pet/serverbound/PetFood --version gms_v92 \
  --ida 'CWvsContext::SendPetFoodItemUseRequest' --category TIER1-FIXTURE \
  --verifies 'libs/atlas-packet/pet/serverbound/food_test.go#TestPetFoodRoundTrip'
```

If a `--verifies` test name does not exist, read the file and use the real one — do not invent it. Each command prints `pinned <path>`.

- [ ] **Step 3: Add the three verify markers**

Append to the existing marker block above the round-trip test in each file:

```go
// packet-audit:verify packet=inventory/serverbound/InventoryItemUse version=gms_v92 ida=0x9b3600
```

```go
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v92 ida=0x9ab2f0
```

```go
// packet-audit:verify packet=pet/serverbound/PetFood version=gms_v92 ida=0x9afa50
```

- [ ] **Step 4: Regenerate the matrix and confirm all three promoted**

```bash
go run ./tools/packet-audit matrix
python3 - <<'EOF'
import json
d=json.load(open('docs/packets/audits/status.json'))
for r in d['rows']:
    if r.get('op') in ('USE_ITEM','USE_UPGRADE_SCROLL','PET_FOOD'):
        print(r['op'], r['cells']['gms_v92'])
EOF
go run ./tools/packet-audit matrix --check >/dev/null 2>&1; echo "EXIT=$?"
```

Expected: each prints `{'state': 'verified', 'opcode': …}`, and `EXIT=0`. Anything else is a failure to report, not to narrate around.

- [ ] **Step 5: Run the packet tests**

```bash
cd libs/atlas-packet && go test -race ./... && cd ../..
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet docs/packets/evidence/gms_v92 docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(task-229): promote USE_ITEM, USE_UPGRADE_SCROLL and PET_FOOD on gms_v92"
```

---

## Tasks 6–14: Per-version verification of the two ops

Tasks 6 through 14 are **the same nine steps run against a different IDB**. They are listed once here in full, then each task names only its version-specific parameters. Execute them one IDB at a time — never two in parallel against the same IDA server.

### The per-version procedure (referenced by Tasks 6–14)

Let `<V>` be the version key (e.g. `gms_v83`), `<SESSION>` the session id resolved from `idb_list` by filename, `<EXPORT>` the export path (`docs/packets/ida-exports/<V>.json`, except jms_v185 → `gms_jms_185.json`), `<SB_OP>` and `<RS_OP>` the decimal `USE_SUMMON_BAG` / `USE_RETURN_SCROLL` opcodes from the Global Constraints table, and `<SB_FNAME>` / `<RS_FNAME>` the registry primary fnames from the fname table.

- [ ] **Step A: Resolve the session**

```
mcp__ida-pro__idb_list
```

Match the target binary by `filename`. Record the `session_id`. Never reuse a session id from this document.

- [ ] **Step B: Locate both send sites by the opcode-construction invariant**

Search the IDB for `COutPacket::COutPacket(&pkt, <SB_OP>)` and `COutPacket::COutPacket(&pkt, <RS_OP>)` using `mcp__ida-pro__find_bytes` with the pattern `6A <op-hex> 8D 8D` and `6A <op-hex> 8D 4D`, then decompile each hit. The correct site is the one whose body is `Encode4(updateTime) + Encode2(slot) + Encode4(itemId)` under a `CanSendExclRequest`-style guard, and whose item-category / WZ-prop gate matches the feature:
- **summon bag** — the mob-summon gate; structural twin is gms_v61 `CWvsContext::SendMobSummonItemUseRequest` @`0x832003` (registry `ida.address: 8592515`).
- **return scroll** — reads the Return Scroll WZ props via `IWzProperty::Getitem` (StringPool ids in the 22xx range), with **no** `RunMapTransferItem` call and **no** by-name / target-name branch. That last pair is what separates it from the teleport-rock sender, which is a different op. Registry notes on `gms_v72.yaml:2414` and `gms_v79.yaml:2798` record this exact trap: the era's `SendMapTransferItemUseRequest` symbol is a *mislabel*.

Do not accept a function because its name looks right. Record the decompiled read order and the address for both.

- [ ] **Step C: Name the senders in the IDB if unnamed**

If either function is a `sub_XXXXXX`, rename it with `mcp__ida-pro__rename` to the registry's primary fname for that version, then `mcp__ida-pro__idb_save`. If the registry's primary fname IS the `sub_XXXXXX` form (v72/v79 summon bag, v61 return scroll), **keep the `sub_` name** — the registry, the `candidatesFromFName` case and the export all key on it, and renaming it would break all three.

- [ ] **Step D: Harvest the two functions to a scratch file**

Write a one-off roster containing just the two fnames (the harvester scrapes fname-shaped tokens out of any markdown):

```bash
SCRATCH=/tmp/claude-1000/-home-tumidanski-source-atlas-ms-atlas/ced8fd96-ef2e-4a3e-a34f-ab9b916d068c/scratchpad
mkdir -p "$SCRATCH"
printf '%s\n%s\n' '<SB_FNAME>' '<RS_FNAME>' > "$SCRATCH/roster-<V>.md"

go run ./tools/packet-audit export \
  -version <V> \
  -ida-url http://192.168.20.3:8745/mcp \
  -ida-database <SESSION> \
  -prior-export "" \
  -pending "$SCRATCH/roster-<V>.md" \
  -descent-depth 12 \
  -output "$SCRATCH/harvest-<V>.json"
```

- [ ] **Step E: Cross-check the harvest, then hand-splice it into the committed export**

```bash
python3 - <<EOF
import json
h=json.load(open("$SCRATCH/harvest-<V>.json"))['functions']
for n in ('<SB_FNAME>','<RS_FNAME>'):
    print(n, h.get(n,{}).get('address'), len(h.get(n,{}).get('calls') or []))
EOF
```

The printed addresses MUST equal the addresses you decompiled in Step B. If they differ, the harvest hit the wrong binary (usually a stale `-ida-url`) — stop and re-harvest.

Then add the two entries to the committed export **by hand**, with `Edit`, inserting the JSON objects into the `"functions"` map and preserving the file's existing indentation. **Do not use `packet-audit export -splice`** — it drops unrecognized fields (`region`, `note`) from ~20 unrelated entries and reindents the whole file.

If the harvested entry contains a `{"op": "Delegate", "ref": "COutPacket"}` call, strip that one call before splicing — it is the packet constructor, not a wire read, and report-gen fails on it with "delegate to COutPacket: not in export".

```bash
git diff --stat <EXPORT>
```

Expected: one file changed, and inspecting `git diff <EXPORT>` shows only the two added entries.

- [ ] **Step F: Generate the two audit reports**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/<TEMPLATE> \
  -ida-source <EXPORT> \
  -output "$SCRATCH/rpt-<V>"

cp "$SCRATCH/rpt-<V>/<V>/InventorySummonBagItemUse.json" \
   "$SCRATCH/rpt-<V>/<V>/InventorySummonBagItemUse.md" \
   "$SCRATCH/rpt-<V>/<V>/InventoryReturnScrollItemUse.json" \
   "$SCRATCH/rpt-<V>/<V>/InventoryReturnScrollItemUse.md" \
   docs/packets/audits/<V>/
```

If a report is not written, the linkage failed — re-check the Task 2 switch case matches the registry's primary fname for THIS version exactly, and that the export entry key matches too.

- [ ] **Step G: Pin evidence and add the verify markers**

```bash
go run ./tools/packet-audit evidence pin \
  --packet inventory/serverbound/InventorySummonBagItemUse --version <V> \
  --ida '<SB_FNAME>' --category TIER1-FIXTURE \
  --verifies 'libs/atlas-packet/inventory/serverbound/summon_bag_item_use_test.go#TestSummonBagItemUseRoundTrip'

go run ./tools/packet-audit evidence pin \
  --packet inventory/serverbound/InventoryReturnScrollItemUse --version <V> \
  --ida '<RS_FNAME>' --category TIER1-FIXTURE \
  --verifies 'libs/atlas-packet/inventory/serverbound/return_scroll_item_use_test.go#TestReturnScrollItemUseRoundTrip'
```

Then append one marker line to each test file's doc-comment block, using the address you confirmed in Step B (lowercase hex, `0x`-prefixed):

```go
// packet-audit:verify packet=inventory/serverbound/InventorySummonBagItemUse version=<V> ida=0x<addr>
```

```go
// packet-audit:verify packet=inventory/serverbound/InventoryReturnScrollItemUse version=<V> ida=0x<addr>
```

The marker's `ida=` address must match the evidence record's `ida.address`, or `matrix --check` reports an orphan marker.

- [ ] **Step H: Regenerate the matrix and confirm both cells promoted**

```bash
go run ./tools/packet-audit matrix
python3 - <<'EOF'
import json
d=json.load(open('docs/packets/audits/status.json'))
for r in d['rows']:
    if r.get('op') in ('USE_SUMMON_BAG','USE_RETURN_SCROLL'):
        print(r['op'], {k:v['state'] for k,v in sorted(r['cells'].items())})
EOF
go run ./tools/packet-audit matrix --check >/dev/null 2>&1; echo "EXIT=$?"
```

Expected: `<V>` reads `verified` for both ops; `EXIT=0`. A cell that did not move is a failure — report it with the `matrix --check` stderr, do not proceed to the next version.

- [ ] **Step I: Test and commit**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && cd ../..
cd tools/packet-audit && go test ./... && cd ../..
git add libs/atlas-packet/inventory/serverbound docs/packets/ida-exports docs/packets/audits docs/packets/evidence
git commit -m "verify(task-229): USE_SUMMON_BAG and USE_RETURN_SCROLL on <V>"
```

---

### Task 6: gms_v83

Run the per-version procedure with:

- `<V>` = `gms_v83`, IDB filename `MapleStory_dump.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v83.json`
- `<TEMPLATE>` = `template_gms_83_1.json`
- `<SB_OP>` = 75 (`0x4B`), `<SB_FNAME>` = `CWvsContext::SendMobSummonItemUseRequest`
- `<RS_OP>` = 85 (`0x55`), `<RS_FNAME>` = `CWvsContext::SendPortalScrollUseRequest`

Do this version first: it is the reference column, both handlers are already routed in `template_gms_83_1.json` (`0x4B`, `0x55`), and the v83 IDB has the richest handler naming. It proves the whole pipeline — struct → switch case → export splice → report → evidence → marker → promotion — before nine repeats. If the pipeline does not promote here, stop and fix the pipeline, not the version.

Both registry entries are `provenance: csv-import` with no `ida:` block. Once you have the real address, also update `docs/packets/registry/gms_v83.yaml` for both ops: set `provenance: ida-discovered`, add `ida: {address: <decimal>}`, and add a `note:` recording the opcode-construction site and the read order. Commit that with the rest.

### Task 7: gms_v61

- `<V>` = `gms_v61`, IDB filename `GMS_v61.1_U_DEVM.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v61.json`
- `<TEMPLATE>` = `template_gms_61_1.json`
- `<SB_OP>` = 70 (`0x46`), `<SB_FNAME>` = `CWvsContext::SendMobSummonItemUseRequest` (already **named** in this IDB; registry `ida.address: 8592515` = `0x832003`)
- `<RS_OP>` = 78 (`0x4E`), `<RS_FNAME>` = `sub_841AA5` (unnamed; registry `ida.address: 8658104`; the send site itself is at `0x841cb8`) — **keep the `sub_` name**, the registry and switch case key on it.

Do this second: v61 carries the only IDB-named `SendMobSummonItemUseRequest` in the whole set, so its decompile is the structural twin every other version's unnamed sender is matched against. Record that read order in your notes — Tasks 8–15 compare against it.

`template_gms_61_1.json` already routes both (`0x46`, `0x4E`); make no template change.

### Task 8: gms_v84

- `<V>` = `gms_v84`, IDB filename `GMS_v84.1_U_DEVM.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v84.json`
- `<TEMPLATE>` = `template_gms_84_1.json`
- `<SB_OP>` = 75 (`0x4B`), `<SB_FNAME>` = `CWvsContext::SendMobSummonItemUseRequest`
- `<RS_OP>` = 85 (`0x55`), `<RS_FNAME>` = `CWvsContext::SendPortalScrollUseRequest`

Already routed; no template change. Both registry entries carry the "seeded from the v83 CSV column" note — after confirming the addresses, upgrade `provenance` to `ida-discovered` and add the `ida:` block, keeping the existing note text and appending the task-229 finding.

### Task 9: gms_v87

- `<V>` = `gms_v87`, IDB filename `GMSv87_4GB.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v87.json`
- `<TEMPLATE>` = `template_gms_87_1.json`
- `<SB_OP>` = 78 (`0x4E`), `<SB_FNAME>` = `CWvsContext::SendMobSummonItemUseRequest`
- `<RS_OP>` = 88 (`0x58`), `<RS_FNAME>` = `CWvsContext::SendPortalScrollUseRequest`

Routed by Task 4.

### Task 10: gms_v92

- `<V>` = `gms_v92`, IDB filename `GMS_v92_1_DEVM.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v92.json`
- `<TEMPLATE>` = `template_gms_92_1.json`
- `<SB_OP>` = 82 (`0x52`), `<SB_FNAME>` = `CWvsContext::SendMobSummonItemUseRequest`
- `<RS_OP>` = 92 (`0x5C`), `<RS_FNAME>` = `CWvsContext::SendPortalScrollUseRequest`

Routed by Task 4.

### Task 11: gms_v95

- `<V>` = `gms_v95`, IDB filename `GMS_v95.0_U_DEVM.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v95.json`
- `<TEMPLATE>` = `template_gms_95_1.json`
- `<SB_OP>` = 81 (`0x51`), `<SB_FNAME>` = `CWvsContext::SendMobSummonItemUseRequest`
- `<RS_OP>` = 92 (`0x5C`), `<RS_FNAME>` = `CWvsContext::SendPortalScrollUseRequest`

Routed by Task 4.

### Task 12: jms_v185

- `<V>` = `jms_v185`, IDB filename `MapleStory_dump_SCY.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_jms_185.json` — **not** `jms_v185.json`. Audit dir and evidence dir remain `jms_v185`.
- `<TEMPLATE>` = `template_jms_185_1.json`
- `<SB_OP>` = 67 (`0x43`), `<SB_FNAME>` = `CWvsContext::SendMobSummonItemUseRequest`
- `<RS_OP>` = 77 (`0x4D`), `<RS_FNAME>` = `CWvsContext::SendPortalScrollUseRequest`

Routed by Task 4. `MapleStory_dump_SCY.exe.i64` is the clean `_U_DEVM`-equivalent dump; if you find yourself looking at SMC / control-flow-virtualized code where Hex-Rays fails, you are on the retail IDB — re-resolve the session.

### Task 13: gms_v72

- `<V>` = `gms_v72`, IDB filename `GMS_v72.1_U_DEVM.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v72.json`
- `<TEMPLATE>` = `template_gms_72_1.json`
- `<SB_OP>` = 74 (`0x4A`), `<SB_FNAME>` = `sub_955499` (registry `ida.address: 9453908`; send site `0x904154`) — **keep the `sub_` name**
- `<RS_OP>` = 84 (`0x54`), `<RS_FNAME>` = `CWvsContext::SendReturnScrollUseRequest` (renamed live on task-124; registry `ida.address: 9531937`; send site `0x917221`)

Already routed; no template change.

### Task 14: gms_v79

- `<V>` = `gms_v79`, IDB filename `GMS_v79_1_DEVM.exe.i64`
- `<EXPORT>` = `docs/packets/ida-exports/gms_v79.json`
- `<TEMPLATE>` = `template_gms_79_1.json`
- `<SB_OP>` = 73 (`0x49`), `<SB_FNAME>` = `sub_955499` (registry `ida.address: 9786521`; send site `0x9555b0`; **the v79 IDB mislabels this function as `SendEngagementRequest`** — read the opcode from the body) — **keep the `sub_` name**
- `<RS_OP>` = 83 (`0x53`), `<RS_FNAME>` = `CWvsContext::SendReturnScrollUseRequest` (registry `ida.address: 9866322`; send site `0x968c52`)

Already routed; no template change.

---

## Task 15: Resolve gms_v48

**Files:**
- Modify: `docs/packets/registry/gms_v48.yaml`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`
- Modify: `docs/packets/ida-exports/gms_v48.json`
- Create: `docs/packets/audits/gms_v48/InventorySummonBagItemUse.{json,md}` (+ ReturnScroll pair if present)
- Create: `docs/packets/evidence/gms_v48/inventory.serverbound.InventorySummonBagItemUse.yaml` (+ ReturnScroll if present)
- Modify: `tools/packet-audit/cmd/run.go`, `tools/packet-audit/cmd/disambiguation_test.go`
- Modify (conditional): `docs/packets/feature-na-evidence.yaml`
- Modify: `libs/atlas-packet/inventory/serverbound/summon_bag_item_use_test.go` (+ return_scroll test if present)

**Interfaces:**
- Consumes: the v61 summon-bag read order recorded in Task 7; the per-version procedure of Tasks 6–14.
- Produces: a `gms_v48` `USE_SUMMON_BAG` registry entry with a resolved fname, and a resolved `USE_RETURN_SCROLL` answer — either a verified cell or an evidenced `n-a`. Never a silent absence.

Current state: `docs/packets/registry/gms_v48.yaml` has **neither** op. `template_gms_48_1.json` nonetheless routes `CharacterItemUseSummonBagHandle` at `0x3B` with no `fname`. The matrix reads `n-a / opcode -1` for both ops on gms_v48. That contradiction is what this task closes.

Design-phase probe already done (GMS_v48_1_DEVM.exe.i64): the only `COutPacket::COutPacket(&pkt, 59)` site in the binary is inside `sub_70DDAA`, sending `Encode4(updateTime) + Encode2(slot) + Encode4(itemId)` after a `sub_4A2518(200, 0)` excl-request guard, gated on `sub_713039(itemId)` (`CWvsContext::IsAbleToConsume`) and a field-limit bit `(*((_DWORD *)get_field() + 58) >> 2) & 1` guarding a `CUtilDlg::Notice` (string 270). **Established:** `0x3B` is a real item-use-shaped send site, so the template binding is not a typo. **Not established:** that it is the mob-summon sender specifically.

- [ ] **Step 1: Confirm `0x3B` is the summon-bag sender**

Resolve the gms_v48 session (`GMS_v48_1_DEVM.exe.i64`), decompile `sub_70DDAA`, and compare it structurally against the v61 `CWvsContext::SendMobSummonItemUseRequest` decompile you recorded in Task 7 — same guard shape, same field order, same category/WZ-prop gate role. Also enumerate every other `COutPacket::COutPacket(&pkt, N)` site in the v48 item-use neighbourhood so you can say what `0x38` and `0x3B` each are.

Do NOT reason from opcode position: v48's serverbound table is **not** a shifted copy of v61's (v48 puts `USE_ITEM` at `0x41`, *after* `USE_SKILL_BOOK` `0x40`; v61+ put it first).

If the comparison fails — if `sub_70DDAA` is some other item-use op — say so explicitly, correct the template binding to whatever the IDB proves, and record the correction in the registry note and the PR body.

- [ ] **Step 2: Search for a v48 return-scroll sender**

Enumerate item-use-shaped send sites across the whole v48 binary by the opcode-construction invariant, not by symbol name. The lead worth chasing first is already recorded in the registry: `gms_v48.yaml:978` notes that the `USE_ITEM` sender `sub_719DD9` (opcode 65) itself performs a **"return-scroll target-map check (sub_71E860, notice 2678)"** — i.e. v48 may route return scrolls through the generic item-use op rather than a dedicated one. Confirm or refute that by decompiling `sub_71E860` and tracing which send sites reach it.

The mandatory checks before concluding absence (`VERIFYING_A_PACKET.md` "Is this cell n-a?"):
1. A failed name search is not evidence. Anchor on the opcode-construction invariant, the Return Scroll WZ props read via `IWzProperty::Getitem`, and the item-category gate.
2. Sibling cross-check: decompile the clientbound receive side of the same feature on v48. If the receive side handles the feature, the send side exists somewhere — keep looking.
3. Expect the era's `SendMapTransferItemUseRequest`-style symbol to be a mislabel for the teleport-rock sender, exactly as `gms_v72.yaml:2414` and `gms_v79.yaml:2798` record.

- [ ] **Step 3a (branch: the sender EXISTS) — register, bind, verify**

1. Name it in the IDB and `idb_save`.
2. Add a `USE_RETURN_SCROLL` entry to `docs/packets/registry/gms_v48.yaml` at its sorted position, with the resolved opcode, `fname`, `provenance: ida-discovered`, `ida: {address: <decimal>}`, `packet: inventory/serverbound/InventoryReturnScrollItemUse`, and a `note:` recording the send site and read order.
3. If the resolved fname is a `sub_XXXXXX`, add a `candidatesFromFName` case for it in `run.go` (copy the `sub_719DD9` comment style at `run.go:2188-2193`) plus a case in `TestCandidatesItemUseFamilyWrappers`.
4. Bind `CharacterItemUseTownScrollHandle` in `template_gms_48_1.json` at that opcode, `validator: "LoggedInValidator"`, the resolved `fname`, `services: ["channel"]`, at its sorted position.
5. Run Steps D–I of the per-version procedure for `gms_v48` for both ops.

- [ ] **Step 3b (branch: the sender is ABSENT) — record positive absence**

Add an entry to `docs/packets/feature-na-evidence.yaml` in the existing shape:

```yaml
  - op: USE_RETURN_SCROLL
    version: gms_v48
    evidence: >
      <what you searched, exhaustively, and what you found instead — the count of
      COutPacket::COutPacket(long) sites examined, the item-category gates that
      route return scrolls, the sibling clientbound receive-handler result, and
      where return-scroll use actually lands on v48 (e.g. the generic USE_ITEM
      op 65 sender sub_719DD9, whose target-map check is sub_71E860 / notice
      2678). (task-229)>
```

Leave the registry without a `USE_RETURN_SCROLL` entry so the cell stays `n-a`. Note that neither op is currently a member of any family in `docs/packets/feature-families.yaml`, so this entry is not yet *consumed* by the n-a consistency gate — it is recorded so the claim is auditable and so the gate has proof the moment an `item_use` family is declared. Do not add the family in this task.

- [ ] **Step 4: Register `USE_SUMMON_BAG` for gms_v48 regardless of branch**

Add to `docs/packets/registry/gms_v48.yaml`, at its sorted position:

```yaml
- op: USE_SUMMON_BAG
  direction: serverbound
  opcode: 59
  fname: <resolved fname — sub_70DDAA if it stays unnamed, or the name you gave it>
  provenance: ida-discovered
  ida:
    address: <decimal address>
  packet: inventory/serverbound/InventorySummonBagItemUse
  note: '<send site, opcode-construction address, read order Encode4(updateTime)+Encode2(slot)+Encode4(itemId), the guards (sub_4A2518(200,0) excl-request; sub_713039 IsAbleToConsume; field-limit bit (get_field+58)>>2 &1 → CUtilDlg::Notice 270), and the structural match against the v61 named twin. (task-229)>'
```

Add the matching `candidatesFromFName` case for that fname, plus a case in `TestCandidatesItemUseFamilyWrappers`. Then backfill the `fname` on the existing `CharacterItemUseSummonBagHandle` entry in `template_gms_48_1.json` (opCode `0x3B`) so it matches its siblings.

- [ ] **Step 5: Verify**

```bash
cd tools/packet-audit && go test ./... && cd ../..
cd libs/atlas-packet && go test -race ./... && cd ../..
tools/template-opcode-order-guard.sh; echo "ORDER=$?"
tools/template-duplicate-binding-guard.sh; echo "DUP=$?"
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | tail -20
go run ./tools/packet-audit matrix --check >/dev/null 2>&1; echo "EXIT=$?"
```

Expected: `ORDER=0 DUP=0 EXIT=0`, `USE_SUMMON_BAG × gms_v48` reads `verified` (no longer `n-a`), and `USE_RETURN_SCROLL × gms_v48` reads either `verified` or `n-a` with the evidence entry recorded.

- [ ] **Step 6: Commit**

```bash
git add docs/packets services/atlas-configurations/seed-data/templates/template_gms_48_1.json \
        tools/packet-audit libs/atlas-packet
git commit -m "verify(task-229): resolve USE_SUMMON_BAG and USE_RETURN_SCROLL on gms_v48"
```

---

## Task 16: Full verification and PR preparation

**Files:**
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`
- Create: `docs/tasks/task-229-summon-bag-town-scroll-opcodes/audit.md` (written by the review agents)

**Interfaces:**
- Consumes: everything from Tasks 1–15.
- Produces: a green branch and a PR-ready summary.

- [ ] **Step 1: Confirm the final matrix state**

```bash
go run ./tools/packet-audit matrix
python3 - <<'EOF'
import json
d=json.load(open('docs/packets/audits/status.json'))
for r in d['rows']:
    if r.get('op') in ('USE_SUMMON_BAG','USE_RETURN_SCROLL','USE_ITEM','USE_UPGRADE_SCROLL','PET_FOOD'):
        print(r['op'], r.get('packet'))
        for k,v in sorted(r['cells'].items()):
            print('   ', k, v['state'], v.get('note',''))
EOF
```

Expected: `USE_SUMMON_BAG` and `USE_RETURN_SCROLL` carry their packet ids and read `verified` on all ten columns (gms_v48 `USE_RETURN_SCROLL` may legitimately read `n-a`); the three v92 rows read `verified`. Any residual `incomplete`/`partial`/`conflict` is a failure to report by name.

- [ ] **Step 2: Run every CI gate**

```bash
cd tools/packet-audit && go test ./... && cd ../..
go run ./tools/packet-audit fname-doc --check;    echo "FNAME=$?"
go run ./tools/packet-audit operations --check;   echo "OPS=$?"
go run ./tools/packet-audit dispatcher-lint;      echo "DISP=$?"
go run ./tools/packet-audit doc-freshness --check; echo "FRESH=$?"
go run ./tools/packet-audit gate-check --check;   echo "GATE=$?"
go run ./tools/packet-audit matrix --check >/dev/null 2>&1; echo "MATRIX=$?"
```

Expected: all `0`.

- [ ] **Step 3: Run the module builds, tests and repo guards**

```bash
cd libs/atlas-packet && go build ./... && go test -race ./... && go vet ./... && cd ../..
cd tools/packet-audit && go build ./... && go vet ./... && cd ../..
tools/template-opcode-order-guard.sh;      echo "ORDER=$?"
tools/template-duplicate-binding-guard.sh; echo "DUP=$?"
tools/template-movement-types-guard.sh;    echo "MOVE=$?"
tools/redis-key-guard.sh;                  echo "REDIS=$?"
tools/goroutine-guard.sh;                  echo "GOROUTINE=$?"
tools/lint.sh --check;                     echo "LINT=$?"
```

Expected: all `0`. `tools/lint.sh --check` needs nvm on PATH for the atlas-ui half; if it false-fails without it, source nvm and re-run rather than declaring it broken.

No service `go.mod` was touched, so `docker buildx bake` has no target here — confirm with `git diff --name-only main... | grep 'go.mod'` returning nothing before skipping it.

- [ ] **Step 4: Assert the no-regression constraints**

```bash
git diff --stat main... -- services/atlas-configurations/seed-data/templates/
git diff --stat main... -- libs/atlas-packet/inventory/serverbound/item_use.go
git diff --stat main... -- services/atlas-channel services/atlas-consumables
```

Expected: templates show only `template_gms_48_1.json`, `_87_`, `_92_`, `_95_`, `template_jms_185_1.json`; the other two commands produce **no output at all**.

- [ ] **Step 5: Code review**

Invoke `superpowers:requesting-code-review` from the worktree. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go files changed; no atlas-ui changes, so no frontend reviewer). Pin the reviewer subagents to a cheaper model per the project's model preference. Ensure they operate inside this worktree and write `audit.md` here, not into the main repo. Address findings before opening the PR.

- [ ] **Step 6: Commit the regenerated matrix and any review fixes**

```bash
git add docs/packets/audits/STATUS.md docs/packets/audits/status.json docs/tasks/task-229-summon-bag-town-scroll-opcodes/audit.md
git commit -m "chore(task-229): regenerate coverage matrix and record review findings"
```

- [ ] **Step 7: PR body must call out the operational gap**

The PR body must state: these are **seed-template** changes. Tenants already provisioned from an earlier template revision do not pick up the new bindings until they are reseeded or their live socket configuration is PATCHed. Whether that happens by reseed or by PATCH is the deployer's call and is out of scope for this task (PRD §2, §10).

---

## Design deviations recorded during planning

Two things were verified against the tooling source while writing this plan and differ from the design's assumptions. Both are noted here so the executor does not re-derive them.

1. **An audit report is not strictly required for promotion.** `grade.go:206-215` promotes a cell to `verified` with **no report** when the registry declares `packet:` and a fresh evidence record backs a found marker; `cmd/matrix.go:178-181` exempts exactly that case from the "dangling evidence" `--check` failure. The plan still generates reports (design D1 Option C, and the `InventoryLotteryItemUse` precedent has them) because they carry the decompiled read order beside the codec — but if report generation fails on some version for a mechanical reason, the cell can still legitimately promote via the registry `packet:` link. Do not treat a missing report as a blocker without first checking whether the cell promoted anyway.

2. **The n-a consistency gate does not currently apply to these ops.** `naConsistencyCheck` (`cmd/na_consistency.go:166-200`) only examines ops that are members of a family declared in `docs/packets/feature-families.yaml`, and neither `USE_SUMMON_BAG` nor `USE_RETURN_SCROLL` is in any family. The design described the v48 `n-a` as "a `--check` failure waiting to happen"; today it is not one, and `matrix --check` exits 0 on the branch tip. Task 15 still records positive absence evidence if the v48 return-scroll sender proves absent — because PRD FR-5.3 requires it and because an unconsumed entry is harmless (it is reported as a note, never a failure) — but no family is declared in this task.
