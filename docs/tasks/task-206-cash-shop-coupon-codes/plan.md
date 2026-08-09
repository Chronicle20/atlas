# Cash Shop Coupon-Code Redemption Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A player types a promo code into the Cash Shop Coupon tab and receives NX / Maple Points / cash-locker items atomically, with a correct client-side error message on every failure path, plus an admin surface to mint and audit codes.

**Architecture:** One new serverbound codec (`ShopOperationUseCoupon`) decoded by a new `USE_COUPON` arm in `atlas-channel`, which forwards a `REQUEST_COUPON_REDEMPTION` command to `atlas-cashshop`. `atlas-cashshop` owns a new `coupon` domain and performs the whole redemption inside **one local `database.ExecuteTransaction`** (no saga — see design §2), emitting `COUPON_REDEEMED` / `COUPON_FAILED` status events that the channel turns into `USE_COUPON_SUCCESS` / `USE_COUPON_FAILED` packets. Template `operations` and `errors` tables are **generated** from dispatcher YAML by an extended `packet-audit operations`, never hand-edited.

**Tech Stack:** Go 1.x (multi-module `go.work`), GORM + Postgres (jsonb), Kafka via `libs/atlas-kafka` + transactional outbox (`libs/atlas-outbox`), Redis via `libs/atlas-redis`, JSON:API via api2go, `libs/atlas-packet` codecs, `tools/packet-audit`, React 19 / TanStack Query / react-hook-form + Zod / shadcn-ui for `atlas-ui`.

## Global Constraints

- **No saga.** `libs/atlas-saga` and `services/atlas-saga-orchestrator` are NOT touched by this task (design §2, §13.1). Any step that reaches for a saga action is wrong.
- **Ten target versions**, exactly: `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`. `gms_12` is out of scope (no `CashShopOperationHandle`).
- **All-or-nothing dispatcher YAML rule** (design §5.1): `expectedTable` short-circuits on `len(expected) == 0`, so any version listed in a dispatcher YAML must be enumerated **completely**, or `--check` fails on every key already in that template. Never declare a partial per-version arm set.
- **No invented wire values.** Every mode byte, error byte, and codec field comes from a decompilation recorded in `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md`. If a value is not in that file, it is not written into code, YAML, or a template.
- **Version gating uses `t.MajorAtLeast(N)`** (`libs/atlas-tenant/tenant.go:93`) combined with `t.IsRegion("GMS")`. Raw `MajorVersion() > N` / `>= N` comparisons are banned (`bug_majorversion_gt83_is_off_by_one_v87`).
- **Client wire values are config-resolved** (DOM-25): mode bytes via `isCashShopOperation(...)` / `atlas_packet.ResolveCode`, never hard-coded in a handler or processor.
- **No raw `go-redis` keyed commands** outside `libs/atlas-redis` (`tools/redis-key-guard.sh`). No bare `go` statements (`tools/goroutine-guard.sh`).
- **Reuse existing types** before defining new ones (DOM-21). Currency ids follow the existing `wallet.Model.Balance` convention (`services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:33`): `1` = credit (NX), `2` = Maple Points, anything else = prepaid. **No new currency enum.**
- **No `// TODO`, stubs, or 501s** in landed commits.
- **Test setup uses the project's Builder pattern.** No `*_testhelpers.go` test-only constructors.
- **Repo-relative paths only** in committed files — never `/home/<user>/...`.
- Verification gates (design §11) run before the branch is called done; Task 24 is the gate.

---

## File Structure

**Created**

| File | Responsibility |
|---|---|
| `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md` | Every IDA-derived value: per-version serverbound arm tables, per-version `errors` enums, the `USE_COUPON` request body layout, and the `gms_v92` clientbound 57-arm enumeration. Single source for Tasks 5–9. |
| `docs/packets/dispatchers/cash_shop_operation_handle.yaml` | Serverbound `CashShopOperationHandle` per-version `operations` table (source of truth; generates all ten templates). |
| `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon.go` | `ShopOperationUseCoupon` codec (Encode + Decode). |
| `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon_test.go` | Per-version round-trip byte fixtures with `packet-audit:verify` markers. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/{model,entity,administrator,provider,processor,reward,granter,normalize,ratelimit,resource,rest}.go` | The coupon domain. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/{model,entity,administrator,provider,processor,resource,rest}.go` | Coupon batches (bulk generation). |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/{model,entity,administrator,provider,processor,resource,rest}.go` | Redemption records + history. |
| `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go` | Channel-side `Normalize` / `PlausibleCode` (mirrors the service rules). |
| `services/atlas-ui/src/services/api/coupons.service.ts` | JSON:API client for coupons / batches / redemptions. |
| `services/atlas-ui/src/types/models/coupon.ts` | Coupon TS types + Zod schemas. |
| `services/atlas-ui/src/pages/{CouponsPage,CouponDetailPage,coupons-columns}.tsx` | Admin UI trio. |

**Modified**

| File | Change |
|---|---|
| `tools/packet-audit/cmd/operations.go` | Generalize `operations` plumbing to a second, structurally identical `errors` table. |
| `tools/packet-audit/cmd/operations_test.go` | Tests for the new table. |
| `docs/packets/dispatchers/cash_shop_operation.yaml` | Add the complete `gms_v92` column + the per-version `errors:` section. |
| `services/atlas-configurations/seed-data/templates/template_{gms_48,gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json` | Regenerated `operations` (handler) + `errors` (writer) tables. **Generated, not hand-edited.** |
| `libs/atlas-packet/cash/clientbound/shop_operation_body.go:91` | `INVALID_COUPON_COUPON` → `INVALID_COUPON_CODE`. |
| `docs/packets/audits/status.json` / `STATUS.md` | New `cash/serverbound/CashShopOperationUseCoupon` sub-struct row. |
| `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go` | Add `Award(currency, amount) Model` (saturating add). |
| `services/atlas-cashshop/atlas.com/cashshop/main.go` | Register three migrations, the Redis client, and three route initializers. |
| `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` | New command + status event types/bodies. |
| `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go` | New status event providers. |
| `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go` | New command handler arm. |
| `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` | Mirror of the new contracts. |
| `services/atlas-channel/atlas.com/channel/cashshop/{processor,producer}.go` | `RequestCouponRedemption`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` | `USE_COUPON` constant + arm. |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` | `COUPON_REDEEMED` / `COUPON_FAILED` handlers. |
| `services/atlas-ui/src/App.tsx`, `src/components/app-sidebar-items.ts`, `src/lib/breadcrumbs/routes.ts` | Route + nav + breadcrumb registration. |

---

## Task 1: Extend `packet-audit operations` with an `errors` table

**Files:**
- Modify: `tools/packet-audit/cmd/operations.go`
- Test: `tools/packet-audit/cmd/operations_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: a dispatcher YAML may carry `errors:` (same `key`/`modes` shape as `operations:`), generated into and validated against the entry's `options.errors`. Reported with the same `DRIFT` / `MISSING` / `EXTRA` prefixes.

- [ ] **Step 1: Write the failing test**

Add to `tools/packet-audit/cmd/operations_test.go` (match the file's existing fixture helpers for building a temp dispatchers dir + templates dir; if it builds them inline, do the same here):

```go
func TestOperations_GeneratesErrorsTable(t *testing.T) {
	dir := t.TempDir()
	dispatchers := filepath.Join(dir, "dispatchers")
	templates := filepath.Join(dir, "templates")
	mustMkdirAll(t, dispatchers)
	mustMkdirAll(t, templates)

	mustWrite(t, filepath.Join(dispatchers, "d.yaml"), `
writer: TestWriter
op: TEST_OP
operations:
  - key: A_SUCCESS
    modes:
      gms_v83: 1
errors:
  - key: INVALID_COUPON_CODE
    modes:
      gms_v83: 7
  - key: COUPON_EXPIRED
    modes:
      gms_v83: 8
`)
	writeAllTemplates(t, templates, `{"socket":{"writers":[{"opCode":"0x1","writer":"TestWriter","options":{"operations":{"A_SUCCESS":1}}}],"handlers":[]}}`)

	var out, errOut bytes.Buffer
	if rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates}, &out, &errOut); rc != 0 {
		t.Fatalf("generate rc=%d stderr=%s", rc, errOut.String())
	}

	raw := mustRead(t, filepath.Join(templates, "template_gms_83_1.json"))
	if !strings.Contains(raw, `"errors"`) {
		t.Fatalf("errors table not written: %s", raw)
	}
	if !strings.Contains(raw, `"INVALID_COUPON_CODE": 7`) || !strings.Contains(raw, `"COUPON_EXPIRED": 8`) {
		t.Fatalf("errors values wrong: %s", raw)
	}

	out.Reset()
	errOut.Reset()
	if rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates, Check: true}, &out, &errOut); rc != 0 {
		t.Fatalf("check after generate rc=%d stderr=%s", rc, errOut.String())
	}
}

func TestOperations_CheckDetectsErrorsDriftMissingExtra(t *testing.T) {
	cases := []struct {
		name     string
		errsNode string
		want     string
	}{
		{"drift", `"errors":{"INVALID_COUPON_CODE":9}`, "operations DRIFT"},
		{"missing", `"errors":{}`, "operations MISSING"},
		{"extra", `"errors":{"INVALID_COUPON_CODE":7,"BOGUS":3}`, "operations EXTRA"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			dispatchers := filepath.Join(dir, "dispatchers")
			templates := filepath.Join(dir, "templates")
			mustMkdirAll(t, dispatchers)
			mustMkdirAll(t, templates)
			mustWrite(t, filepath.Join(dispatchers, "d.yaml"), `
writer: TestWriter
op: TEST_OP
errors:
  - key: INVALID_COUPON_CODE
    modes:
      gms_v83: 7
`)
			writeAllTemplates(t, templates,
				`{"socket":{"writers":[{"opCode":"0x1","writer":"TestWriter","options":{`+c.errsNode+`}}],"handlers":[]}}`)

			var out, errOut bytes.Buffer
			rc := operationsRun(operationsOpts{DispatchersDir: dispatchers, TemplatesDir: templates, Check: true}, &out, &errOut)
			if rc != 1 {
				t.Fatalf("rc=%d want 1; stderr=%s", rc, errOut.String())
			}
			if !strings.Contains(errOut.String(), c.want) {
				t.Fatalf("stderr %q does not contain %q", errOut.String(), c.want)
			}
		})
	}
}
```

If `mustMkdirAll` / `mustWrite` / `mustRead` / `writeAllTemplates` do not already exist in `operations_test.go`, add them as small local helpers that write every filename in `matrix.VersionKeys` (via `matrix.TemplatePath`) with the same body, so the loop in `operationsRun` finds a file for each version.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd tools/packet-audit && go test ./cmd/ -run 'TestOperations_.*Errors' -v`
Expected: FAIL — `errors table not written` (generate) and `rc=0 want 1` (check), because `dispatcherDoc` has no `Errors` field.

- [ ] **Step 3: Generalize the plumbing to a named table**

In `tools/packet-audit/cmd/operations.go`:

```go
type opEntry struct {
	Key   string         `yaml:"key"`
	Modes map[string]int `yaml:"modes"`
}

type dispatcherDoc struct {
	Writer     string            `yaml:"writer"`
	Handler    string            `yaml:"handler"`
	FName      string            `yaml:"fname"`
	Op         string            `yaml:"op"`
	Opcodes    map[string]string `yaml:"opcodes"`
	Operations []opEntry         `yaml:"operations"`
	// Errors is a second, structurally identical table generated into
	// options.errors. It backs ResolveCode(l, options, "errors", key) — the
	// reason byte of every *_FAILED arm. Hand-maintaining it recreates
	// exactly the drift that left three templates' operations tables empty.
	Errors []opEntry `yaml:"errors"`
}

// tables enumerates the option tables this doc drives, in emit order.
func (d dispatcherDoc) tables() []optionTable {
	return []optionTable{
		{name: "operations", entries: d.Operations},
		{name: "errors", entries: d.Errors},
	}
}

type optionTable struct {
	name    string
	entries []opEntry
}
```

Replace `expectedTable(d, vk)` with a table-scoped form, and thread the table name through `operationsOf` / `setOperations` / `buildOperationsNode` / `addEntry`:

```go
func expectedFor(entries []opEntry, version string) map[string]int {
	m := map[string]int{}
	for _, op := range entries {
		if v, ok := op.Modes[version]; ok {
			m[op.Key] = v
		}
	}
	return m
}

func tableOf(w *node, table string) map[string]int {
	out := map[string]int{}
	opts := w.obj["options"]
	if opts == nil || opts.kind != 'o' {
		return out
	}
	ops := opts.obj[table]
	if ops == nil || ops.kind != 'o' {
		return out
	}
	for _, k := range ops.keys {
		c := ops.obj[k]
		if c.kind != 's' {
			continue
		}
		var num json.Number
		if json.Unmarshal(c.raw, &num) == nil {
			if iv, err := num.Int64(); err == nil {
				out[k] = int(iv)
				continue
			}
		}
		var s string
		if json.Unmarshal(c.raw, &s) == nil {
			if iv, err := strconv.ParseUint(s, 0, 16); err == nil {
				out[k] = int(iv)
			}
		}
	}
	return out
}

func buildTableNode(entries []opEntry, expected map[string]int) *node {
	ops := &node{kind: 'o', obj: map[string]*node{}, dirty: true}
	for _, op := range entries {
		v, ok := expected[op.Key]
		if !ok {
			continue
		}
		ops.keys = append(ops.keys, op.Key)
		ops.obj[op.Key] = &node{kind: 's', raw: json.RawMessage(strconv.Itoa(v))}
	}
	return ops
}

func setTable(w *node, table string, entries []opEntry, expected map[string]int) bool {
	before := nodeBytes(w)
	opts := w.obj["options"]
	if opts == nil || opts.kind != 'o' {
		opts = &node{kind: 'o', obj: map[string]*node{}}
		w.keys = append(w.keys, "options")
		w.obj["options"] = opts
	}
	if _, ok := opts.obj[table]; !ok {
		opts.keys = append(opts.keys, table)
	}
	opts.obj[table] = buildTableNode(entries, expected)
	return !bytes.Equal(before, nodeBytes(w))
}
```

Rewrite the per-doc body of the `for _, vk := range matrix.VersionKeys` loop to iterate `doc.tables()`. The entry-absent branch (`w == nil`) keeps its existing behaviour but must be evaluated once per doc, driven by the union of all tables' expected sizes, and `addEntry` seeds every non-empty table:

```go
for _, doc := range docs {
	name := doc.targetName()
	tables := doc.tables()
	total := 0
	for _, tb := range tables {
		total += len(expectedFor(tb.entries, vk))
	}
	entries := entriesOf(root, doc.arrayKey())
	w := findEntryNode(entries, doc.entryKey(), name)
	if w == nil {
		if total == 0 {
			continue
		}
		oc, hasOC := doc.Opcodes[vk]
		if !hasOC {
			absent = append(absent, fmt.Sprintf("%s: %s %q not in template (cannot populate %d ops; add an opcodes entry to the YAML to wire it)", vk, doc.entryKey(), name, total))
			continue
		}
		if o.Check {
			missing = append(missing, fmt.Sprintf("%s %s: entry absent (should be wired at %s)", vk, name, oc))
			continue
		}
		if addEntry(root, doc, vk, oc) {
			changed = true
		}
		continue
	}
	for _, tb := range tables {
		expected := expectedFor(tb.entries, vk)
		if len(expected) == 0 {
			continue
		}
		got := tableOf(w, tb.name)
		for k, want := range expected {
			if gv, ok := got[k]; !ok {
				missing = append(missing, fmt.Sprintf("%s %s: %s key %q missing (want %d)", vk, name, tb.name, k, want))
			} else if gv != want {
				drift = append(drift, fmt.Sprintf("%s %s: %s key %q is %d, want %d", vk, name, tb.name, k, gv, want))
			}
		}
		for k := range got {
			if _, ok := expected[k]; !ok {
				extra = append(extra, fmt.Sprintf("%s %s: %s key %q in template but not in enumeration", vk, name, tb.name, k))
			}
		}
		if !o.Check && setTable(w, tb.name, tb.entries, expected) {
			changed = true
		}
	}
}
```

And `addEntry` becomes table-aware:

```go
func addEntry(root *node, doc dispatcherDoc, version string, opcode string) bool {
	arr := arrayNode(root, doc.arrayKey())
	if arr == nil {
		return false
	}
	entryKey := doc.entryKey()
	ocBytes, _ := json.Marshal(opcode)
	wnBytes, _ := json.Marshal(doc.targetName())
	opts := &node{kind: 'o', obj: map[string]*node{}, dirty: true}
	for _, tb := range doc.tables() {
		expected := expectedFor(tb.entries, version)
		if len(expected) == 0 {
			continue
		}
		opts.keys = append(opts.keys, tb.name)
		opts.obj[tb.name] = buildTableNode(tb.entries, expected)
	}
	w := &node{kind: 'o', dirty: true, obj: map[string]*node{
		"opCode":  {kind: 's', raw: ocBytes},
		entryKey:  {kind: 's', raw: wnBytes},
		"options": opts,
	}, keys: []string{"opCode", entryKey, "options"}}
	arr.arr = append(arr.arr, w)
	arr.dirty = true
	return true
}
```

Delete the now-unused `expectedTable`, `operationsOf`, `setOperations`, and `buildOperationsNode`. Update the file's leading doc comment to say the command drives `options.operations` **and** `options.errors`.

- [ ] **Step 4: Run the new tests and the whole package**

Run: `cd tools/packet-audit && go test ./... -race`
Expected: PASS, including the pre-existing operations tests (the `operations` table behaviour is unchanged).

- [ ] **Step 5: Verify no live drift was introduced**

Run: `go run ./tools/packet-audit operations --check` from the worktree root.
Expected: `operations check OK (...)` — exit 0. No YAML declares `errors:` yet, so every version's errors `expected` is empty and short-circuits.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/cmd/operations.go tools/packet-audit/cmd/operations_test.go
git commit -m "feat(packet-audit): generate and check template options.errors alongside operations"
```

---

## Task 2: Derive the modern-version wire values (gms_v83, v84, v87, v92, v95)

**Files:**
- Create: `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md`

**Interfaces:**
- Consumes: nothing.
- Produces: `derivation.md` §gms_v83 … §gms_v95, each containing (a) the **complete** `CCashShop::OnCashItemRequest` serverbound arm table (key → mode byte), (b) the **complete** `CCashShop::OnCashItemResult` failure-reason enum (key → byte), (c) the `USE_COUPON` request body field order, and for `gms_v92` additionally (d) the complete clientbound `OnCashItemResult` arm table (all 57+ arms). Every row cites an IDB address.

**Method (read `docs/packets/audits/VERIFYING_A_PACKET.md` first).** Resolve each session from `mcp__ida-pro__idb_list` **by binary name** and pass it as the `database` parameter — `select_instance(port)` is dead. Use `mcp__ida-pro__func_query` with `name_regex` for lookups. Name any unnamed function you rely on (`feedback_name_idb_symbols_while_reversing`); an unnamed function is not an absent one.

- [ ] **Step 1: Create the derivation doc skeleton**

Create `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md` with this exact frame (one `##` section per version, filled in as each is read):

```markdown
# task-206 — IDA derivation record

Every wire value used by this task originates here. A value absent from this
file may not appear in a codec, a dispatcher YAML, or a tenant template.

Method: docs/packets/audits/VERIFYING_A_PACKET.md. Sessions resolved from
`idb_list` by binary NAME and passed as the `database` parameter.

## gms_v83

- Serverbound dispatcher: `<fname>` @ `<addr>`
- Clientbound dispatcher: `<fname>` @ `<addr>`

### Serverbound `operations` (CashShopOperationHandle)

| Key | Mode | Evidence |
|---|---|---|

### `errors` enum (CashShopOperation writer)

| Key | Byte | Evidence |
|---|---|---|

### `USE_COUPON` request body

| # | Field | Read | Evidence |
|---|---|---|---|
```

- [ ] **Step 2: Derive `gms_v83` — the reference column**

Decompile the serverbound cash-shop request dispatcher and enumerate **every** arm of its leading mode switch. Cross-check the key names against the nineteen already in `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` (`CashShopOperationHandle` → `options.operations`) — those nineteen are the established key vocabulary; reuse those exact strings and add new keys only for arms Atlas has no name for yet. Record any disagreement between the template's existing value and the decompile as a template bug, with both values.

Then decompile the clientbound `CCashShop::OnCashItemResult` failure arms and enumerate the full reason enum, using the key strings already declared in `libs/atlas-packet/cash/clientbound/shop_operation_body.go:80-125` (`CashShopOperationError*`). **Use `INVALID_COUPON_CODE`, not the current `INVALID_COUPON_COUPON` typo** — Task 6 corrects the constant.

Finally, decompile the `USE_COUPON` arm of the request builder and record the exact read order for the body that follows the mode byte.

Fill in `## gms_v83` completely.

- [ ] **Step 3: Derive `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`**

Repeat Step 2 per version, one IDB at a time, completing each version's section before opening the next (design §12: do not interleave). For `gms_v92` additionally enumerate the **clientbound** arm table completely (all arms of `OnCashItemResult`), because `docs/packets/dispatchers/cash_shop_operation.yaml` has no v92 column and its template's 57 keys are currently validated by nothing. Record the derived-vs-template diff for v92 in a `### gms_v92 clientbound template diff` subsection — a disagreement is a **template bug**, not a derivation error (design §12).

- [ ] **Step 4: Self-check completeness**

For each of the five sections confirm: the serverbound arm table's row count equals the number of `case` labels in the decompiled switch; the errors table's row count equals the number of distinct reason values the failure arms accept; the `USE_COUPON` body table has one row per read call. Note explicitly (not silently) if a version's request switch has **no** `USE_COUPON` arm — that version becomes `n-a` in Task 5 with this enumeration as the evidence.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/derivation.md
git commit -m "docs(task-206): derive modern-version cash-shop serverbound arms, error enum, and USE_COUPON layout"
```

---

## Task 3: Derive the legacy-version wire values (gms_v48, v61, v72, v79)

**Files:**
- Modify: `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md`

**Interfaces:**
- Consumes: the doc frame from Task 2.
- Produces: `## gms_v48`, `## gms_v61`, `## gms_v72`, `## gms_v79` sections in the same shape.

- [ ] **Step 1: Derive each legacy version, one IDB at a time**

Same procedure as Task 2 Step 2, per version. Expect the legacy enums to be shifted non-uniformly against `gms_v83` — read each value, never offset-derive it (this is the trap `cash_shop_operation.yaml`'s header calls out). The templates already carry 13 / 15 / 16 / 17 serverbound keys respectively; treat those as the existing key vocabulary and record any value disagreement.

Note that `ShopOperationBuy` (`libs/atlas-packet/cash/serverbound/shop_operation_buy.go:67-90`) documents real legacy body divergence for the sibling `BUY` arm — expect the `USE_COUPON` body to diverge too, and record it rather than assuming the v83 shape.

- [ ] **Step 2: Self-check completeness**

Same checks as Task 2 Step 4, for all four sections.

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/derivation.md
git commit -m "docs(task-206): derive legacy-version cash-shop serverbound arms, error enum, and USE_COUPON layout"
```

---

## Task 4: Derive the `jms_v185` wire values

**Files:**
- Modify: `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md`

**Interfaces:**
- Consumes: the doc frame from Task 2.
- Produces: `## jms_v185` section in the same shape.

- [ ] **Step 1: Derive `jms_v185`**

Same procedure. `template_jms_185_1.json` has only five serverbound keys against `gms_83`'s nineteen, so the enumeration here is mostly new ground. JMS enums shift non-uniformly (see `character_interaction_handle.yaml`'s header for the precedent), so every value is read, never derived. Note the audit-dir naming quirk (`reference_packet_audit_jms_dirname_mismatch`) when citing evidence paths.

- [ ] **Step 2: Self-check completeness**

Same checks as Task 2 Step 4.

- [ ] **Step 3: Final cross-version sanity pass**

Confirm all ten `##` sections exist and none contains an unfilled `<...>` placeholder. Confirm every key string used in an `errors` table exists as a `CashShopOperationError*` constant in `libs/atlas-packet/cash/clientbound/shop_operation_body.go` (modulo the `INVALID_COUPON_CODE` correction Task 6 makes) — a key with no constant can never be resolved by `ResolveCode` and is dead configuration.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/derivation.md
git commit -m "docs(task-206): derive jms_v185 cash-shop serverbound arms, error enum, and USE_COUPON layout"
```

---

## Task 5: `ShopOperationUseCoupon` codec

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon.go`
- Create: `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon_test.go`

**Interfaces:**
- Consumes: `derivation.md` §`USE_COUPON` request body (all ten versions).
- Produces: `serverbound.ShopOperationUseCoupon` with `Code() string`, `Operation() string`, `String() string`, `Encode(l, ctx)(options) []byte`, `Decode(l, ctx)(r, options)`, and the exported handle constant `CashShopOperationUseCouponHandle`.

- [ ] **Step 1: Write the failing round-trip test**

Create `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon_test.go`. Follow the fixture style of `shop_operation_buy_test.go` (same tenant construction helper, same byte-slice comparison). One sub-test per version whose derivation section has a `USE_COUPON` arm; the `want` bytes are built from that section's field table:

```go
func TestShopOperationUseCoupon_RoundTrip(t *testing.T) {
	// packet-audit:verify packet=cash/serverbound/CashShopOperationUseCoupon version=gms_v83 ida=<addr from derivation.md §gms_v83>
	cases := []struct {
		name  string
		ten   tenant.Model
		model ShopOperationUseCoupon
		want  []byte
	}{
		// One entry per applicable version. `want` is the exact byte sequence
		// derivation.md §<version> "USE_COUPON request body" specifies for
		// code "TESTCODE123".
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := tenant.WithContext(context.Background(), c.ten)
			got := c.model.Encode(testLogger(), ctx)(map[string]interface{}{})
			if !bytes.Equal(got, c.want) {
				t.Fatalf("encode = % x, want % x", got, c.want)
			}
			var decoded ShopOperationUseCoupon
			decoded.Decode(testLogger(), ctx)(request.NewReader(&got), map[string]interface{}{})
			if decoded.Code() != c.model.Code() {
				t.Fatalf("round-trip code = %q, want %q", decoded.Code(), c.model.Code())
			}
		})
	}
}
```

Use whatever tenant-builder and reader-construction helpers `shop_operation_buy_test.go` already uses — do not invent new ones.

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestShopOperationUseCoupon -v`
Expected: FAIL — `undefined: ShopOperationUseCoupon`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopOperationUseCouponHandle = "CashShopOperationUseCouponHandle"

// ShopOperationUseCoupon — the USE_COUPON arm of the cash-shop request
// dispatcher. Field order per version is recorded in
// docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
// packet-audit:fname <the per-version fnames from derivation.md>
type ShopOperationUseCoupon struct {
	code string
}

func NewShopOperationUseCoupon(code string) ShopOperationUseCoupon {
	return ShopOperationUseCoupon{code: code}
}

func (m ShopOperationUseCoupon) Code() string { return m.code }

func (m ShopOperationUseCoupon) Operation() string {
	return CashShopOperationUseCouponHandle
}

func (m ShopOperationUseCoupon) String() string {
	return fmt.Sprintf("code [%s]", m.code)
}

func (m ShopOperationUseCoupon) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.code)
		// Any version-divergent trailing field goes here, gated with
		// t.IsRegion(...) && t.MajorAtLeast(N) per derivation.md. Raw
		// MajorVersion() comparisons are banned.
		_ = t
		return w.Bytes()
	}
}

func (m *ShopOperationUseCoupon) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadAsciiString()
		_ = t
	}
}
```

**Replace the body of `Encode`/`Decode` with exactly the reads `derivation.md` records** — including any additional field and its version gate — and delete the `_ = t` lines once `t` is genuinely used. If the ten versions agree on a bare length-prefixed string, keep it this simple and say so in the doc comment. If they diverge, add named predicate helpers in the style of `buyOmitsCurrency` / `buyOmitsTrailingZero` (`shop_operation_buy.go:79-90`), each with a comment citing the IDB address that proves the divergence.

- [ ] **Step 4: Run the tests**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/shop_operation_use_coupon.go libs/atlas-packet/cash/serverbound/shop_operation_use_coupon_test.go
git commit -m "feat(atlas-packet): add serverbound ShopOperationUseCoupon codec"
```

---

## Task 6: Correct the `INVALID_COUPON_COUPON` key string

**Files:**
- Modify: `libs/atlas-packet/cash/clientbound/shop_operation_body.go:91`

**Interfaces:**
- Consumes: nothing.
- Produces: `CashShopOperationErrorInvalidCouponCode = "INVALID_COUPON_CODE"` — the string every `errors` table and every service-side error mapping must use.

- [ ] **Step 1: Confirm nothing consumes the old string**

Run: `grep -rn 'INVALID_COUPON_COUPON' --include='*.go' --include='*.json' --include='*.yaml' --include='*.ts' .`
Expected: exactly one hit, the constant declaration itself. If any template, YAML, or service references the typo'd string, that reference is updated in the same commit.

- [ ] **Step 2: Make the edit**

In `libs/atlas-packet/cash/clientbound/shop_operation_body.go`:

```go
	// Corrected from the "INVALID_COUPON_COUPON" typo (task-206). Nothing
	// consumed the constant, so the wire contract is unaffected; the errors
	// tables generated in task-206 use this string.
	CashShopOperationErrorInvalidCouponCode = "INVALID_COUPON_CODE"
```

- [ ] **Step 3: Verify the package still builds and tests pass**

Run: `cd libs/atlas-packet && go build ./... && go test ./cash/... -race`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/cash/clientbound/shop_operation_body.go
git commit -m "fix(atlas-packet): correct INVALID_COUPON_COUPON key string to INVALID_COUPON_CODE"
```

---

## Task 7: Serverbound dispatcher YAML — `CashShopOperationHandle`

**Files:**
- Create: `docs/packets/dispatchers/cash_shop_operation_handle.yaml`
- Modify: all ten `services/atlas-configurations/seed-data/templates/template_*_1.json` (generated)

**Interfaces:**
- Consumes: `derivation.md` serverbound arm tables (all ten versions); the `errors` table extension from Task 1.
- Produces: every version's `CashShopOperationHandle` `options.operations` table, generated and CI-checked. Includes the `USE_COUPON` key.

- [ ] **Step 1: Author the YAML**

Create `docs/packets/dispatchers/cash_shop_operation_handle.yaml`, modelled on `docs/packets/dispatchers/character_interaction_handle.yaml` (the existing serverbound precedent):

```yaml
# CashShopOperationHandle — serverbound CCashShop::OnCashItemRequest mode table.
#
# SOURCE OF TRUTH for the tenant template `operations` map of the
# CashShopOperationHandle handler. `packet-audit operations` generates and
# validates template_<ver>_1.json handlers[].options.operations against this
# file. Every byte is IDA-derived per version; derivation record:
# docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
#
# ALL-OR-NOTHING: expectedTable short-circuits on an empty per-version map,
# so a version listed here must be enumerated COMPLETELY — a partial column
# makes --check report every other key in that template as EXTRA.
handler: CashShopOperationHandle
op: CASHSHOP_OPERATION
direction: serverbound
fname: CCashShop::OnCashItemRequest
opcodes:
  gms_v48: "0xA0"
  gms_v61: "0xC4"
  gms_v72: "0xDB"
  gms_v79: "0xDD"
  gms_v83: "0xE5"
  gms_v84: "0xEB"
  gms_v87: "0xF2"
  gms_v92: "0x10C"
  gms_v95: "0x113"
  jms_v185: "0xF5"
operations:
  # One entry per arm, keys in the vocabulary the templates already use
  # (BUY, GIFT, SET_WISHLIST, …) plus USE_COUPON. Every version that has the
  # arm carries a mode value read from derivation.md.
  - key: USE_COUPON
    modes:
      # gms_v48: <byte>   ← only versions whose derivation section records the arm
```

Fill `operations:` from `derivation.md` — **every** arm of **every** version, not just `USE_COUPON`. Verify the `opcodes:` values against each template's existing `CashShopOperationHandle` entry (`gms_v92`'s is `0x10C` per design §5.1; read the file rather than trusting the table). Omit a version from a key's `modes` only when that version's derivation section records the arm as absent.

- [ ] **Step 2: Run `--check` and observe the pre-generation state**

Run: `go run ./tools/packet-audit operations --check`
Expected: exit 1 with `MISSING` / `DRIFT` lines for the templates whose serverbound tables are empty or short (`gms_87`, `gms_92`, `gms_95`, `jms_185`) and `USE_COUPON` missing everywhere. Read every `DRIFT` line: a drift against a value the template already had means either the YAML transcribed the derivation wrong or the template was wrong. Resolve each one against `derivation.md` before generating, and record any template-was-wrong finding in `derivation.md` under the version's section.

- [ ] **Step 3: Generate**

Run: `go run ./tools/packet-audit operations`
Then: `go run ./tools/packet-audit operations --check`
Expected: `operations check OK`.

- [ ] **Step 4: Run the template guards**

Run:
```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```
Expected: all exit 0. No opcode was added or rebound, so a failure here means the generator reordered or duplicated something — fix the generator, not the template.

- [ ] **Step 5: Confirm the diff is options-only**

Run: `git diff --stat services/atlas-configurations/seed-data/templates/`
Then spot-check one modified template with `git diff` and confirm only `options.operations` content changed inside `CashShopOperationHandle` entries (plus a new entry where one was genuinely absent).

- [ ] **Step 6: Commit**

```bash
git add docs/packets/dispatchers/cash_shop_operation_handle.yaml services/atlas-configurations/seed-data/templates/
git commit -m "feat(packets): generate CashShopOperationHandle operations tables for all ten versions"
```

---

## Task 8: `errors` tables + `gms_v92` clientbound column

**Files:**
- Modify: `docs/packets/dispatchers/cash_shop_operation.yaml`
- Modify: all ten `services/atlas-configurations/seed-data/templates/template_*_1.json` (generated)

**Interfaces:**
- Consumes: `derivation.md` error enums (all ten) + the `gms_v92` clientbound arm table; Task 1's `errors:` support; Task 6's corrected key string.
- Produces: `options.errors` on every version's `CashShopOperation` writer, so `ResolveCode(l, options, "errors", …)` resolves; and a complete `gms_v92` column in the clientbound `operations` table.

- [ ] **Step 1: Add the `gms_v92` clientbound column**

In `docs/packets/dispatchers/cash_shop_operation.yaml`, add `gms_v92: <byte>` to the `modes:` map of **every** key the v92 derivation records, and add `gms_v92: "<opCode>"` to `opcodes:`. Update the file's header comment to note v92 is now covered and cite `derivation.md`. Partial coverage is forbidden (Global Constraints).

- [ ] **Step 2: Add the `errors:` section**

Append to the same file:

```yaml
# errors — the failure-reason byte table backing ResolveCode(l, options,
# "errors", key) for every *_FAILED arm of CCashShop::OnCashItemResult.
# No template carried this table before task-206, so every coupon (and every
# other) failure resolved to an unconfigured code. Per-version bytes from
# docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
errors:
  - key: UNKNOWN_ERROR
    modes:
      # gms_v48: <byte>
      # … one entry per version
  - key: INVALID_COUPON_CODE
    modes:
  # … the full per-version enum, one key per CashShopOperationError* constant
```

Fill from `derivation.md`. The seven coupon keys required by PRD FR-3.4 — `INVALID_COUPON_CODE`, `COUPON_EXPIRED`, `COUPON_ALREADY_USED`, `COUPON_USAGE_LIMIT`, `COUPON_NOT_REGISTERED`, `INVENTORY_FULL`, `UNKNOWN_ERROR` — must be present for all ten versions; the rest of the enum is derived while the record is in hand (design Q3).

- [ ] **Step 3: Generate and check**

Run:
```bash
go run ./tools/packet-audit operations
go run ./tools/packet-audit operations --check
```
Expected: `operations check OK`. Any `EXTRA` on a v92 clientbound key means the template's pre-existing (never-validated) value disagrees with the derivation — treat it as a template bug per design §12, confirm against `derivation.md`, and let the generator win. Record the diff in `derivation.md` §gms_v92.

- [ ] **Step 4: Run the template guards**

Run:
```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```
Expected: all exit 0.

- [ ] **Step 5: Verify every errors key resolves to a constant**

Run:
```bash
python3 - <<'EOF'
import json, re, pathlib
consts = set(re.findall(r'=\s*"([A-Z0-9_]+)"', pathlib.Path('libs/atlas-packet/cash/clientbound/shop_operation_body.go').read_text()))
bad = []
for p in pathlib.Path('services/atlas-configurations/seed-data/templates').glob('template_*_1.json'):
    d = json.loads(p.read_text())
    for w in d.get('socket', {}).get('writers', []):
        if w.get('writer') == 'CashShopOperation':
            for k in w.get('options', {}).get('errors', {}):
                if k not in consts:
                    bad.append((p.name, k))
print('unmatched:', bad)
EOF
```
Expected: `unmatched: []`. A key with no Go constant is dead configuration.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/dispatchers/cash_shop_operation.yaml services/atlas-configurations/seed-data/templates/
git commit -m "feat(packets): add per-version cash-shop errors tables and the gms_v92 clientbound column"
```

---

## Task 9: Promote the coverage-matrix cells

**Files:**
- Modify: `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md`
- Create: the evidence records the verify procedure pins

**Interfaces:**
- Consumes: Task 5's codec + fixtures, Task 2–4's derivation.
- Produces: a `cash/serverbound/CashShopOperationUseCoupon` sub-struct row with one cell per version.

- [ ] **Step 1: Confirm no new registry op is needed**

Run: `grep -rn 'CASHSHOP_OPERATION' docs/packets/registry/ | head`
Expected: the serverbound op already exists (design §4, §13.2). This task adds a **sub-struct matrix row**, not a registry op.

- [ ] **Step 2: Add the sub-struct row**

Follow the shape of the existing `cash/serverbound/CashShopOperationBuy` row in `docs/packets/audits/status.json` (`kind: "sub-struct"`, `tier1: true`, `cells` keyed by the ten version keys, `opcode: -1`). Any version whose derivation section records **no** `USE_COUPON` arm is `n-a` with the enumeration cited as evidence — see `reference_packet_na_consistency_gate`.

- [ ] **Step 3: Verify each cell through the single-cell procedure**

For each applicable version, follow `docs/packets/audits/VERIFYING_A_PACKET.md`: byte-fixture test with a `packet-audit:verify` marker (already written in Task 5), evidence record pinned, matrix regenerated. Do not hand-edit a cell to `verified` — a cell that does not promote is a failure report, not a prose claim (`bug_matrix_roundtrip_fixture_false_verify`).

- [ ] **Step 4: Run the machine checks**

Run:
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit operations --check
```
plus the fname-doc and n-a consistency checks the playbook names.
Expected: all exit 0.

- [ ] **Step 5: Confirm no previously-verified cell regressed**

Run: `git diff docs/packets/audits/status.json`
Expected: additions only, plus any cell this task legitimately promotes. A cell moving **away** from `verified` means Task 5 changed wire behaviour for an already-verified version — fix the codec's gating, not the matrix.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/audits/
git commit -m "test(packets): verify CashShopOperationUseCoupon matrix cells"
```

---

## Task 10: `wallet.Model.Award`

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/wallet/model_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func (m Model) Award(currency uint32, amount uint32) Model` — pure, saturating at `math.MaxUint32`, mirroring the existing `Purchase(currency, amount) Model`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-cashshop/atlas.com/cashshop/wallet/model_test.go`:

```go
func TestModel_Award(t *testing.T) {
	base := Model{credit: 100, points: 200, prepaid: 300}

	if got := base.Award(1, 50); got.Credit() != 150 || got.Points() != 200 || got.Prepaid() != 300 {
		t.Fatalf("Award(1,50) = %+v", got)
	}
	if got := base.Award(2, 50); got.Points() != 250 || got.Credit() != 100 {
		t.Fatalf("Award(2,50) = %+v", got)
	}
	if got := base.Award(3, 50); got.Prepaid() != 350 || got.Credit() != 100 {
		t.Fatalf("Award(3,50) = %+v", got)
	}
}

func TestModel_AwardSaturates(t *testing.T) {
	m := Model{credit: math.MaxUint32 - 5}
	if got := m.Award(1, 100); got.Credit() != math.MaxUint32 {
		t.Fatalf("Award saturation = %d, want %d", got.Credit(), uint32(math.MaxUint32))
	}
}

func TestModel_AwardIsPure(t *testing.T) {
	m := Model{credit: 10}
	_ = m.Award(1, 5)
	if m.Credit() != 10 {
		t.Fatalf("Award mutated the receiver: %d", m.Credit())
	}
}
```

Add `"math"` to the test file's imports.

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./wallet/ -run TestModel_Award -v`
Expected: FAIL — `m.Award undefined`.

- [ ] **Step 3: Implement**

In `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go`:

```go
// Award credits the wallet, mirroring Purchase. The add saturates at
// math.MaxUint32 so a malformed reward amount cannot wrap a balance to
// near-zero (task-206).
func (m Model) Award(currency uint32, amount uint32) Model {
	add := func(v uint32) uint32 {
		if math.MaxUint32-v < amount {
			return math.MaxUint32
		}
		return v + amount
	}

	newCredit := m.credit
	newPoints := m.points
	newPrepaid := m.prepaid
	if currency == 1 {
		newCredit = add(newCredit)
	} else if currency == 2 {
		newPoints = add(newPoints)
	} else {
		newPrepaid = add(newPrepaid)
	}

	return Model{
		id:        m.id,
		accountId: m.accountId,
		credit:    newCredit,
		points:    newPoints,
		prepaid:   newPrepaid,
	}
}
```

Add `"math"` to the file's imports.

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./wallet/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/wallet/
git commit -m "feat(atlas-cashshop): add saturating wallet.Model.Award"
```

---

## Task 11: Code normalization helpers (shared rules, two packages)

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize_test.go`
- Create: `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go`
- Create: `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces (identical in both packages): `const MaxCodeLength = 32`, `func Normalize(code string) string`, `func PlausibleCode(code string) bool`.

The two services are separate Go modules and this is 20 lines of pure logic; duplicating it with an identical test suite is cheaper and safer than a new shared lib (`feedback_audit_existing_libs_before_new_module`). The tests are byte-identical so a divergence fails immediately.

- [ ] **Step 1: Write the failing test (cashshop side)**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize_test.go`:

```go
package coupon

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc123", "ABC123"},
		{"  abc123  ", "ABC123"},
		{"\tAbC123\n", "ABC123"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		if got := Normalize(c.in); got != c.want {
			t.Fatalf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPlausibleCode(t *testing.T) {
	if PlausibleCode("") {
		t.Fatal("empty code should not be plausible")
	}
	if PlausibleCode(strings.Repeat("A", MaxCodeLength+1)) {
		t.Fatal("over-length code should not be plausible")
	}
	if !PlausibleCode("ABC123") {
		t.Fatal("ordinary code should be plausible")
	}
	if !PlausibleCode(strings.Repeat("A", MaxCodeLength)) {
		t.Fatal("max-length code should be plausible")
	}
}
```

Add `"strings"` to the test imports.

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -v`
Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize.go`:

```go
package coupon

import "strings"

// MaxCodeLength matches the coupons.code varchar(32) column. A longer
// submission cannot match any stored code, so it is rejected without a
// database round trip.
const MaxCodeLength = 32

// Normalize is the ONE canonical form of a coupon code: surrounding
// whitespace trimmed, then uppercased. Codes are stored normalized, so the
// unique index on (tenant_id, code) IS the case-insensitive uniqueness
// guarantee — there is no functional index over a raw value.
func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// PlausibleCode reports whether an ALREADY-NORMALIZED code could possibly
// match a stored one. It is the cheap first line of brute-force defence.
func PlausibleCode(code string) bool {
	return code != "" && len(code) <= MaxCodeLength
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -race`
Expected: PASS.

- [ ] **Step 5: Mirror into atlas-channel**

Create `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go` and `normalize_test.go` with byte-identical bodies (the package clause is the same, `package coupon`). Add to the `.go` file's doc comment: `// Mirrors services/atlas-cashshop/atlas.com/cashshop/coupon/normalize.go — the two are separate modules; the identical test suites are what keep them from diverging.`

Run: `cd services/atlas-channel/atlas.com/channel && go test ./cashshop/coupon/ -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/ services/atlas-channel/atlas.com/channel/cashshop/coupon/
git commit -m "feat(coupon): add shared code normalization and plausibility rules"
```

---

## Task 12: Reward model

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/reward.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/reward_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `RewardType` (`RewardTypeCurrency` = `"CURRENCY"`, `RewardTypeCashItem` = `"CASH_ITEM"`), `Reward` with `Type()`, `Currency()`, `Amount()`, `SerialNumber()`, `Quantity()`; `NewCurrencyReward(currency, amount uint32) Reward`; `NewCashItemReward(serialNumber, quantity uint32) Reward`; `Rewards []Reward` with `Value()`/`Scan()` for GORM jsonb; `func (r Reward) Validate() error`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/reward_test.go`:

```go
package coupon

import (
	"encoding/json"
	"testing"
)

func TestRewardsJSONRoundTrip(t *testing.T) {
	in := Rewards{
		NewCurrencyReward(2, 10000),
		NewCashItemReward(50200000, 1),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Rewards
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0].Type() != RewardTypeCurrency || out[0].Currency() != 2 || out[0].Amount() != 10000 {
		t.Fatalf("currency reward = %+v", out[0])
	}
	if out[1].Type() != RewardTypeCashItem || out[1].SerialNumber() != 50200000 || out[1].Quantity() != 1 {
		t.Fatalf("cash item reward = %+v", out[1])
	}
}

func TestRewardsValueScanRoundTrip(t *testing.T) {
	in := Rewards{NewCurrencyReward(1, 5)}
	v, err := in.Value()
	if err != nil {
		t.Fatal(err)
	}
	var out Rewards
	if err := out.Scan(v); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Amount() != 5 {
		t.Fatalf("scan = %+v", out)
	}
}

func TestRewardValidate(t *testing.T) {
	if err := NewCurrencyReward(1, 100).Validate(); err != nil {
		t.Fatalf("valid currency reward rejected: %v", err)
	}
	if err := NewCurrencyReward(1, 0).Validate(); err == nil {
		t.Fatal("zero-amount currency reward should be rejected")
	}
	if err := NewCashItemReward(50200000, 1).Validate(); err != nil {
		t.Fatalf("valid cash item reward rejected: %v", err)
	}
	if err := NewCashItemReward(0, 1).Validate(); err == nil {
		t.Fatal("zero serial number should be rejected")
	}
	if err := NewCashItemReward(50200000, 0).Validate(); err == nil {
		t.Fatal("zero quantity should be rejected")
	}
	if err := (Reward{rewardType: "MESO"}).Validate(); err == nil {
		t.Fatal("unknown reward type should be rejected")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -run 'TestReward' -v`
Expected: FAIL — undefined identifiers.

- [ ] **Step 3: Implement**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/reward.go`:

```go
package coupon

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type RewardType string

const (
	RewardTypeCurrency RewardType = "CURRENCY"
	RewardTypeCashItem RewardType = "CASH_ITEM"
)

// Reward is a discriminated value. Currency ids follow the existing
// wallet.Model.Balance convention (wallet/model.go:33): 1 = credit (NX),
// 2 = Maple Points, anything else = prepaid. No new currency enum is
// introduced (DOM-21).
type Reward struct {
	rewardType   RewardType
	currency     uint32
	amount       uint32
	serialNumber uint32
	quantity     uint32
}

func NewCurrencyReward(currency uint32, amount uint32) Reward {
	return Reward{rewardType: RewardTypeCurrency, currency: currency, amount: amount}
}

func NewCashItemReward(serialNumber uint32, quantity uint32) Reward {
	return Reward{rewardType: RewardTypeCashItem, serialNumber: serialNumber, quantity: quantity}
}

func (r Reward) Type() RewardType    { return r.rewardType }
func (r Reward) Currency() uint32    { return r.currency }
func (r Reward) Amount() uint32      { return r.amount }
func (r Reward) SerialNumber() uint32 { return r.serialNumber }
func (r Reward) Quantity() uint32    { return r.quantity }

var ErrInvalidReward = errors.New("invalid reward")

func (r Reward) Validate() error {
	switch r.rewardType {
	case RewardTypeCurrency:
		if r.amount == 0 {
			return fmt.Errorf("%w: currency amount must be positive", ErrInvalidReward)
		}
		return nil
	case RewardTypeCashItem:
		if r.serialNumber == 0 {
			return fmt.Errorf("%w: cash item serialNumber must be set", ErrInvalidReward)
		}
		if r.quantity == 0 {
			return fmt.Errorf("%w: cash item quantity must be positive", ErrInvalidReward)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown type %q", ErrInvalidReward, r.rewardType)
	}
}

// rewardJSON is the persisted / transported shape. The exported struct keeps
// private fields per the project's immutable-model rule, so marshalling goes
// through this mirror.
type rewardJSON struct {
	Type         RewardType `json:"type"`
	Currency     uint32     `json:"currency,omitempty"`
	Amount       uint32     `json:"amount,omitempty"`
	SerialNumber uint32     `json:"serialNumber,omitempty"`
	Quantity     uint32     `json:"quantity,omitempty"`
}

func (r Reward) MarshalJSON() ([]byte, error) {
	return json.Marshal(rewardJSON{
		Type:         r.rewardType,
		Currency:     r.currency,
		Amount:       r.amount,
		SerialNumber: r.serialNumber,
		Quantity:     r.quantity,
	})
}

func (r *Reward) UnmarshalJSON(b []byte) error {
	var j rewardJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	r.rewardType = j.Type
	r.currency = j.Currency
	r.amount = j.Amount
	r.serialNumber = j.SerialNumber
	r.quantity = j.Quantity
	return nil
}

// Rewards is the jsonb column type. The bundle is always read and written
// whole and never queried by reward attribute, which is why it is jsonb and
// not a child table.
type Rewards []Reward

func (rs Rewards) Value() (driver.Value, error) {
	return json.Marshal(rs)
}

func (rs *Rewards) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, rs)
	case string:
		return json.Unmarshal([]byte(v), rs)
	case nil:
		*rs = nil
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Rewards", src)
	}
}

func (rs Rewards) Validate() error {
	if len(rs) == 0 {
		return fmt.Errorf("%w: at least one reward is required", ErrInvalidReward)
	}
	for _, r := range rs {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CashItemCount is the number of locker slots this bundle will consume.
func (rs Rewards) CashItemCount() int {
	n := 0
	for _, r := range rs {
		if r.Type() == RewardTypeCashItem {
			n++
		}
	}
	return n
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/reward.go services/atlas-cashshop/atlas.com/cashshop/coupon/reward_test.go
git commit -m "feat(coupon): add reward value type with jsonb round-trip"
```

---

## Task 13: Coupon entity, model, and migration

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/{model,entity,administrator,provider}.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/entity_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go`

**Interfaces:**
- Consumes: `Rewards` (Task 12), `Normalize` (Task 11).
- Produces: `coupon.Model` (getters `Id() uuid.UUID`, `BatchId() *uuid.UUID`, `Code() string`, `Description() string`, `Active() bool`, `StartsAt() *time.Time`, `ExpiresAt() *time.Time`, `MaxUses() *uint32`, `RedemptionCount() uint32`, `Rewards() Rewards`), `coupon.Entity`, `coupon.Migration`, `coupon.Make(Entity) (Model, error)`, and package-private `createEntity` / `updateEntity` / `deleteEntity` / `incrementRedemptionCount` / `byCodeEntityProvider` / `byIdEntityProvider` / `pagedEntityProvider`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/entity_test.go`:

```go
package coupon

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMake(t *testing.T) {
	id := uuid.New()
	batch := uuid.New()
	starts := time.Now().UTC()
	max := uint32(5)
	e := Entity{
		Id:              id,
		BatchId:         &batch,
		Code:            "ABC123",
		Description:     "test",
		Active:          true,
		StartsAt:        &starts,
		MaxUses:         &max,
		RedemptionCount: 2,
		Rewards:         Rewards{NewCurrencyReward(1, 100)},
	}
	m, err := Make(e)
	if err != nil {
		t.Fatal(err)
	}
	if m.Id() != id || m.Code() != "ABC123" || !m.Active() || m.RedemptionCount() != 2 {
		t.Fatalf("Make = %+v", m)
	}
	if m.BatchId() == nil || *m.BatchId() != batch {
		t.Fatalf("BatchId = %v", m.BatchId())
	}
	if m.MaxUses() == nil || *m.MaxUses() != 5 {
		t.Fatalf("MaxUses = %v", m.MaxUses())
	}
	if m.ExpiresAt() != nil {
		t.Fatalf("ExpiresAt should be nil, got %v", m.ExpiresAt())
	}
	if len(m.Rewards()) != 1 {
		t.Fatalf("Rewards = %+v", m.Rewards())
	}
}

func TestEntityTableName(t *testing.T) {
	if (Entity{}).TableName() != "coupons" {
		t.Fatalf("TableName = %s", (Entity{}).TableName())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -run 'TestMake|TestEntityTableName' -v`
Expected: FAIL — `undefined: Entity`.

- [ ] **Step 3: Write the entity + model**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/entity.go`:

```go
package coupon

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id       uuid.UUID  `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId uuid.UUID  `gorm:"not null;index;uniqueIndex:idx_coupons_tenant_code"`
	BatchId  *uuid.UUID `gorm:"type:uuid;index:idx_coupons_tenant_batch"`
	// Code is stored NORMALIZED (trimmed, uppercased). The unique index over
	// the normalized value IS the case-insensitive uniqueness guarantee.
	Code            string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_coupons_tenant_code"`
	Description     string    `gorm:"type:text"`
	Active          bool      `gorm:"not null;default:true"`
	StartsAt        *time.Time
	ExpiresAt       *time.Time
	MaxUses         *uint32
	RedemptionCount uint32  `gorm:"not null;default:0"`
	Rewards         Rewards `gorm:"type:jsonb;not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (e Entity) TableName() string {
	return "coupons"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:              e.Id,
		batchId:         e.BatchId,
		code:            e.Code,
		description:     e.Description,
		active:          e.Active,
		startsAt:        e.StartsAt,
		expiresAt:       e.ExpiresAt,
		maxUses:         e.MaxUses,
		redemptionCount: e.RedemptionCount,
		rewards:         e.Rewards,
	}, nil
}
```

Create `model.go` with the immutable `Model` (private fields + the getters listed in **Interfaces**) and a `Builder` with `NewBuilder(code string) *Builder`, `SetBatchId`, `SetDescription`, `SetActive`, `SetStartsAt`, `SetExpiresAt`, `SetMaxUses`, `SetRedemptionCount`, `SetRewards`, and `Build() Model` — the project's Builder pattern is how tests construct models (no `*_testhelpers.go`).

- [ ] **Step 4: Write the administrator and provider**

Create `administrator.go`:

```go
package coupon

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func createEntity(db *gorm.DB, t tenant.Model, e Entity) (Model, error) {
	e.TenantId = t.Id()
	e.Code = Normalize(e.Code)
	if err := db.Create(&e).Error; err != nil {
		return Model{}, err
	}
	return Make(e)
}

func updateEntity(db *gorm.DB, t tenant.Model, id uuid.UUID, fields map[string]interface{}) error {
	return db.Model(&Entity{}).
		Where("tenant_id = ? AND id = ?", t.Id(), id).
		Updates(fields).Error
}

func deleteEntity(db *gorm.DB, t tenant.Model, id uuid.UUID) error {
	return db.Where("tenant_id = ? AND id = ?", t.Id(), id).Delete(&Entity{}).Error
}

// incrementRedemptionCount is the FR-5.5 conditional bump. The guard lives in
// the WHERE clause, so two concurrent redemptions of a max_uses=1 coupon see
// exactly one RowsAffected=1. A read-then-write here would be a race and is
// explicitly banned.
func incrementRedemptionCount(db *gorm.DB, t tenant.Model, id uuid.UUID) (bool, error) {
	res := db.Model(&Entity{}).
		Where("tenant_id = ? AND id = ? AND (max_uses IS NULL OR redemption_count < max_uses)", t.Id(), id).
		UpdateColumn("redemption_count", gorm.Expr("redemption_count + 1"))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// decrementRedemptionCount reverses a bump. It exists only for the REST
// delete-guard path; the redemption transaction rolls back instead.
func decrementRedemptionCount(db *gorm.DB, t tenant.Model, id uuid.UUID) error {
	return db.Model(&Entity{}).
		Where("tenant_id = ? AND id = ? AND redemption_count > 0", t.Id(), id).
		UpdateColumn("redemption_count", gorm.Expr("redemption_count - 1")).Error
}
```

Create `provider.go` with `byIdEntityProvider(t, id)`, `byCodeEntityProvider(t, normalizedCode)`, and `pagedEntityProvider(t, filters, page)` following the `database.EntityProvider` / `database.PagedQuery` shape of `wishlist/provider.go`. `byCodeEntityProvider` queries `tenant_id = ? AND code = ?` — the single indexed point read FR-8 requires.

- [ ] **Step 5: Register the migration**

In `services/atlas-cashshop/atlas.com/cashshop/main.go:57`:

```go
	db := database.Connect(l, database.SetMigrations(wallet.Migration, wishlist.Migration, compartment.Migration, asset.Migration, coupon.Migration, outboxlib.Migration))
```

Add `"atlas-cashshop/coupon"` to the imports.

- [ ] **Step 6: Run tests and build**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./coupon/ -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/ services/atlas-cashshop/atlas.com/cashshop/main.go
git commit -m "feat(coupon): add coupon entity, model, and migration"
```

---

## Task 14: Batch and redemption entities

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/{model,entity,administrator,provider}.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/{model,entity,administrator,provider}.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/entity_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go`

**Interfaces:**
- Consumes: `coupon.Rewards` (Task 12).
- Produces: `batch.Model` / `batch.Entity` (`coupon_batches`) / `batch.Migration` / `batch.Make`; `redemption.Model` / `redemption.Entity` (`coupon_redemptions`) / `redemption.Migration` / `redemption.Make`; `redemption.ErrAlreadyRedeemed`; `redemption.Create(db, t, e) (Model, error)` returning `ErrAlreadyRedeemed` on unique violation; `redemption.CountByCouponId(db, t, couponId) (int64, error)`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/entity_test.go`:

```go
package redemption

import "testing"

func TestEntityTableName(t *testing.T) {
	if (Entity{}).TableName() != "coupon_redemptions" {
		t.Fatalf("TableName = %s", (Entity{}).TableName())
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("23505 should be a unique violation")
	}
	if isUniqueViolation(errors.New("boom")) {
		t.Fatal("arbitrary error should not be a unique violation")
	}
}
```

Add the `errors` and `github.com/jackc/pgx/v5/pgconn` imports (confirm the pgx major version already in `services/atlas-cashshop/atlas.com/cashshop/go.mod` and use that one).

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/redemption/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement `redemption`**

`entity.go`:

```go
package redemption

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"atlas-cashshop/coupon"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id       uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId uuid.UUID `gorm:"not null;index:idx_redemptions_tenant_account;uniqueIndex:idx_redemptions_tenant_coupon_account"`
	CouponId uuid.UUID `gorm:"not null;index;uniqueIndex:idx_redemptions_tenant_coupon_account"`
	// AccountId completes the one-time-per-account unique index. The FR-5.4
	// ladder check is a friendly-error fast path; THIS index is the
	// enforcement, and it is what resolves a concurrent double-submit.
	AccountId      uint32 `gorm:"not null;index:idx_redemptions_tenant_account;uniqueIndex:idx_redemptions_tenant_coupon_account"`
	CharacterId    uint32 `gorm:"not null"`
	TransactionId  uuid.UUID `gorm:"type:uuid;not null"`
	// RewardsGranted is a SNAPSHOT, so later edits to the coupon's bundle do
	// not rewrite history.
	RewardsGranted coupon.Rewards `gorm:"type:jsonb;not null"`
	RedeemedAt     time.Time      `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "coupon_redemptions"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:             e.Id,
		couponId:       e.CouponId,
		accountId:      e.AccountId,
		characterId:    e.CharacterId,
		transactionId:  e.TransactionId,
		rewardsGranted: e.RewardsGranted,
		redeemedAt:     e.RedeemedAt,
	}, nil
}
```

`model.go` holds the immutable `Model` with private fields `id`, `couponId`,
`accountId`, `characterId`, `transactionId`, `rewardsGranted`, `redeemedAt`,
one getter each (`Id() uuid.UUID`, `CouponId() uuid.UUID`, `AccountId() uint32`,
`CharacterId() uint32`, `TransactionId() uuid.UUID`,
`RewardsGranted() coupon.Rewards`, `RedeemedAt() time.Time`), and a `Builder`
with a setter per field plus `Build() Model`.

`administrator.go` holds `Create` and `isUniqueViolation`:

```go
var ErrAlreadyRedeemed = errors.New("coupon already redeemed by this account")

func Create(db *gorm.DB, t tenant.Model, e Entity) (Model, error) {
	e.TenantId = t.Id()
	if e.RedeemedAt.IsZero() {
		e.RedeemedAt = time.Now().UTC()
	}
	if err := db.Create(&e).Error; err != nil {
		if isUniqueViolation(err) {
			return Model{}, ErrAlreadyRedeemed
		}
		return Model{}, err
	}
	return Make(e)
}

// isUniqueViolation recognises Postgres 23505 through whatever wrapping GORM
// applied, so the redemption path can map the index collision to
// COUPON_ALREADY_USED instead of UNKNOWN_ERROR.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
```

`provider.go` gets `byCouponIdPagedEntityProvider(t, couponId, page)`, `byAccountIdPagedEntityProvider(t, accountId, page)`, and `CountByCouponId(db, t, couponId) (int64, error)` (used by the REST delete guard).

- [ ] **Step 4: Implement `batch`**

`entity.go` with `coupon_batches` (`Id`, `TenantId` indexed, `Description`, `RequestedCount`, `GeneratedCount`, `CreatedAt`), `Migration`, `Make`, an immutable `Model` + `Builder`, `createEntity`, `byIdEntityProvider`, `pagedEntityProvider`.

- [ ] **Step 5: Register both migrations**

In `main.go`, extend `database.SetMigrations(...)` to `..., coupon.Migration, batch.Migration, redemption.Migration, outboxlib.Migration`. Import `"atlas-cashshop/coupon/batch"` and `"atlas-cashshop/coupon/redemption"`.

- [ ] **Step 6: Run tests and build**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./coupon/... -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/ services/atlas-cashshop/atlas.com/cashshop/main.go
git commit -m "feat(coupon): add batch and redemption entities with the one-per-account unique index"
```

---

## Task 15: Kafka contracts for coupon redemption

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`

**Interfaces:**
- Consumes: nothing.
- Produces (mirrored in both services): `CommandTypeRequestCouponRedemption = "REQUEST_COUPON_REDEMPTION"`, `RequestCouponRedemptionCommandBody{Code string}`, `StatusEventTypeCouponRedeemed = "COUPON_REDEEMED"`, `StatusEventTypeCouponFailed = "COUPON_FAILED"`, `CouponRedeemedBody{CompartmentId uuid.UUID; AssetIds []uint32; MaplePoints uint32}`, `CouponFailedBody{Error string}`; plus `CouponRedeemedStatusEventProvider(characterId, compartmentId, assetIds, maplePoints)` and `CouponFailedStatusEventProvider(characterId, errKey)` on the cashshop producer.

- [ ] **Step 1: Add the cashshop-side contracts**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`, extend the command const block and add the body:

```go
	CommandTypeRequestCouponRedemption = "REQUEST_COUPON_REDEMPTION"
```

```go
// RequestCouponRedemptionCommandBody carries an ALREADY-NORMALIZED code (the
// channel normalizes once, at the packet boundary). The service normalizes
// again defensively — it must not trust an input a future caller could send
// unnormalized. Only characterId travels; the account id is resolved
// service-side because wallets are account-scoped while the packet arrives on
// a character session.
type RequestCouponRedemptionCommandBody struct {
	Code string `json:"code"`
}
```

Extend the status const block and add the bodies:

```go
	StatusEventTypeCouponRedeemed = "COUPON_REDEEMED"
	StatusEventTypeCouponFailed   = "COUPON_FAILED"
```

```go
// CouponRedeemedBody carries asset IDS, not built CashInventoryItem records:
// atlas-channel already owns the asset-id → CashInventoryItem projection
// (kafka/consumer/cashshop/consumer.go:105-124), and duplicating it here
// would put packet concerns in atlas-cashshop.
type CouponRedeemedBody struct {
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetIds      []uint32  `json:"assetIds"`
	MaplePoints   uint32    `json:"maplePoints"`
}

// CouponFailedBody is a DISTINCT event from StatusEventTypeError because the
// generic ERROR event is announced as CashShopInventoryCapacityIncreaseFailedBody
// (a different mode byte). A coupon failure must go out on the
// USE_COUPON_FAILED arm, so the arm selection stays explicit.
type CouponFailedBody struct {
	Error string `json:"error"`
}
```

- [ ] **Step 2: Add the producers**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go`, following `PurchaseStatusEventProvider`:

```go
func CouponRedeemedStatusEventProvider(characterId uint32, compartmentId uuid.UUID, assetIds []uint32, maplePoints uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.CouponRedeemedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeCouponRedeemed,
		Body: cashshop.CouponRedeemedBody{
			CompartmentId: compartmentId,
			AssetIds:      assetIds,
			MaplePoints:   maplePoints,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CouponFailedStatusEventProvider(characterId uint32, errKey string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.CouponFailedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeCouponFailed,
		Body:        cashshop.CouponFailedBody{Error: errKey},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 3: Mirror into atlas-channel**

Add the same consts and bodies to `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`. The JSON tags must match byte-for-byte — this is the wire contract between the two services.

- [ ] **Step 4: Build both modules**

Run:
```bash
(cd services/atlas-cashshop/atlas.com/cashshop && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go build ./...)
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/kafka/ services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go
git commit -m "feat(cashshop): add coupon redemption command and status event contracts"
```

---

## Task 16: Rate limiter

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/ratelimit.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/ratelimit_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/*.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/go.mod`

**Interfaces:**
- Consumes: `libs/atlas-redis` `TenantCounter`.
- Produces: `type Limiter interface { Allow(ctx context.Context, t tenant.Model, accountId uint32) (bool, error); RecordFailure(ctx context.Context, t tenant.Model, accountId uint32) error }`, `func NewLimiter(client *goredis.Client, threshold uint32, window time.Duration) Limiter`, and `func NoopLimiter() Limiter`.

Implemented on `redis.TenantCounter.InitIfMissingAndDecrBy` — the counter is seeded to `threshold` on first failure and decremented per failure; a value `< 0` means the budget is spent. Redis serializes the script, so concurrent failures cannot lose a decrement (`libs/atlas-redis/counter.go:83-95`). The TTL refresh on each decrement makes this a **sliding** window; that is the intended behaviour for brute-force defence.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/ratelimit_test.go` using `miniredis` if it is already a test dependency of any module in this repo (check `go.sum` first); otherwise test against a `TenantCounter` backed by a `goredis.Client` pointed at a `miniredis` instance added as a test-only dependency:

```go
func TestLimiter_BlocksAfterThreshold(t *testing.T) {
	client, cleanup := newTestRedis(t)
	defer cleanup()
	lim := NewLimiter(client, 3, time.Minute)
	ten := testTenant(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		ok, err := lim.Allow(ctx, ten, 1)
		if err != nil || !ok {
			t.Fatalf("attempt %d blocked early (ok=%v err=%v)", i, ok, err)
		}
		if err := lim.RecordFailure(ctx, ten, 1); err != nil {
			t.Fatal(err)
		}
	}

	ok, err := lim.Allow(ctx, ten, 1)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("4th attempt should be rate limited")
	}
}

func TestLimiter_IsPerAccount(t *testing.T) {
	client, cleanup := newTestRedis(t)
	defer cleanup()
	lim := NewLimiter(client, 1, time.Minute)
	ten := testTenant(t)
	ctx := context.Background()

	_ = lim.RecordFailure(ctx, ten, 1)
	if ok, _ := lim.Allow(ctx, ten, 2); !ok {
		t.Fatal("account 2 must not be limited by account 1's failures")
	}
}

func TestNoopLimiter_AlwaysAllows(t *testing.T) {
	ok, err := NoopLimiter().Allow(context.Background(), testTenant(t), 1)
	if err != nil || !ok {
		t.Fatalf("noop limiter blocked (ok=%v err=%v)", ok, err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -run TestLimiter -v`
Expected: FAIL — `undefined: NewLimiter`.

- [ ] **Step 3: Implement**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/ratelimit.go`:

```go
package coupon

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Limiter bounds coupon brute-forcing per account. Short codes are the
// obvious attack surface, so a spent budget short-circuits the redemption
// ladder BEFORE any database lookup.
type Limiter interface {
	// Allow reports whether this account still has failed-attempt budget.
	Allow(ctx context.Context, t tenant.Model, accountId uint32) (bool, error)
	// RecordFailure consumes one unit of budget.
	RecordFailure(ctx context.Context, t tenant.Model, accountId uint32) error
}

type limiter struct {
	counter   *redis.TenantCounter
	threshold uint32
	window    time.Duration
}

// NewLimiter builds a sliding-window limiter. The counter is seeded to
// threshold on the first failure and decremented per failure; the TTL is
// refreshed on each decrement (sliding, by design). All Redis access goes
// through libs/atlas-redis — tools/redis-key-guard.sh bans keyed commands on
// the raw client outside it.
func NewLimiter(client *goredis.Client, threshold uint32, window time.Duration) Limiter {
	return &limiter{
		counter:   redis.NewTenantCounter(client, "coupon-attempts"),
		threshold: threshold,
		window:    window,
	}
}

func key(accountId uint32) string {
	return fmt.Sprintf("account:%d", accountId)
}

func (l *limiter) Allow(ctx context.Context, t tenant.Model, accountId uint32) (bool, error) {
	// A zero-delta decrement reads the current budget without consuming any,
	// and does NOT create the key when it is absent (a first-time caller is
	// always allowed).
	v, existed, err := l.counter.DecrByIfExists(ctx, t, key(accountId), 0, l.window)
	if err != nil {
		return false, err
	}
	if !existed {
		return true, nil
	}
	return v > 0, nil
}

func (l *limiter) RecordFailure(ctx context.Context, t tenant.Model, accountId uint32) error {
	_, err := l.counter.InitIfMissingAndDecrBy(ctx, t, key(accountId), int64(l.threshold), 1, l.window)
	return err
}

type noopLimiter struct{}

// NoopLimiter is used when the tenant has not configured a threshold.
func NoopLimiter() Limiter { return noopLimiter{} }

func (noopLimiter) Allow(_ context.Context, _ tenant.Model, _ uint32) (bool, error) { return true, nil }
func (noopLimiter) RecordFailure(_ context.Context, _ tenant.Model, _ uint32) error { return nil }
```

- [ ] **Step 4: Add the tenant configuration**

In `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/`, add to the `RestModel`:

```go
type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
	Coupons     coupons.RestModel     `json:"coupons"`
}
```

with a new `coupons` sub-package:

```go
package coupons

// RestModel is the tenant-configured coupon brute-force budget. Threshold and
// window are configuration, not constants (DOM-25). A tenant that has not set
// them gets the documented defaults resolved HERE, in the configuration layer
// — never a magic number at the call site.
type RestModel struct {
	AttemptThreshold uint32 `json:"attemptThreshold"`
	AttemptWindowSeconds uint32 `json:"attemptWindowSeconds"`
}

const (
	DefaultAttemptThreshold     = 10
	DefaultAttemptWindowSeconds = 300
)

// Resolved returns the effective values, substituting the documented default
// for any unset field.
func (r RestModel) Resolved() (threshold uint32, window time.Duration) {
	threshold = r.AttemptThreshold
	if threshold == 0 {
		threshold = DefaultAttemptThreshold
	}
	seconds := r.AttemptWindowSeconds
	if seconds == 0 {
		seconds = DefaultAttemptWindowSeconds
	}
	return threshold, time.Duration(seconds) * time.Second
}
```

Add a `configuration.GetCouponLimits(l, ctx, tenantId) (uint32, time.Duration)` accessor alongside the existing `GetHourlyExpirations` in `configuration/registry.go`, returning `Resolved()` and falling back to the defaults when the tenant config cannot be fetched.

- [ ] **Step 5: Wire the Redis client in main.go**

`atlas-cashshop` does not connect to Redis today. Add to `main.go`, after `database.Connect`:

```go
	redisClient := redis.Connect(l)
	rt.TeardownFunc(func() { _ = redisClient.Close() })
```

Import `redis "github.com/Chronicle20/atlas/libs/atlas-redis"` and add the module to `go.mod`'s `require` block (the `replace` directive at `go.mod:102` already exists). `REDIS_URL` is already supplied to every pod through the shared `atlas-env` configmap (`deploy/k8s/base/env-configmap.yaml:167`, consumed via the `envFrom` at `deploy/k8s/base/atlas-cashshop.yaml:21-23`), so **no k8s change is required** — confirm this by reading those two files rather than assuming.

- [ ] **Step 6: Run tests and build**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go mod tidy && go build ./... && go test ./coupon/... -race`
Expected: PASS.

- [ ] **Step 7: Run the redis guard**

Run: `tools/redis-key-guard.sh` from the worktree root.
Expected: exit 0 — every keyed command goes through `TenantCounter`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/
git commit -m "feat(coupon): add tenant-configured per-account attempt limiter"
```

---

## Task 17: Reward granters

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/granter.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/granter_test.go`

**Interfaces:**
- Consumes: `Reward` (Task 12), `wallet.Model.Award` (Task 10), `compartment.Processor`, `asset.Processor`.
- Produces: `type redemptionContext struct { accountId, characterId uint32; compartmentId uuid.UUID; compartmentCapacity uint32; compartmentUsed uint32 }`, `type grantedReward struct { assetId uint32; maplePoints uint32 }`, `type rewardGranter interface { Grant(mb *message.Buffer) func(rc *redemptionContext, r Reward) (grantedReward, error) }`, and `func granterFor(rt RewardType, ...) (rewardGranter, error)`.

Design §2.1: this is the single insertion point where a future out-of-`atlas-cashshop` reward type (regular inventory, mesos, experience) would introduce a saga. Everything today writes only to this service's own tables.

- [ ] **Step 1: Write the failing test**

Create `granter_test.go` with table-driven tests over `granterFor`:

```go
func TestGranterFor_KnownTypes(t *testing.T) {
	for _, rt := range []RewardType{RewardTypeCurrency, RewardTypeCashItem} {
		if _, err := granterFor(rt, nil, nil, nil); err != nil {
			t.Fatalf("granterFor(%s) = %v", rt, err)
		}
	}
}

func TestGranterFor_UnknownType(t *testing.T) {
	if _, err := granterFor(RewardType("MESO"), nil, nil, nil); err == nil {
		t.Fatal("unknown reward type should have no granter")
	}
}

func TestCashItemGranter_RejectsWhenLockerFull(t *testing.T) {
	g := &cashItemGranter{}
	rc := &redemptionContext{compartmentCapacity: 2, compartmentUsed: 2}
	if _, err := g.Grant(nil)(rc, NewCashItemReward(50200000, 1)); !errors.Is(err, ErrInventoryFull) {
		t.Fatalf("err = %v, want ErrInventoryFull", err)
	}
}

func TestCashItemGranter_ConsumesASlotPerGrant(t *testing.T) {
	rc := &redemptionContext{compartmentCapacity: 2, compartmentUsed: 0}
	// After a successful grant the context must reflect the consumed slot, so
	// a two-item bundle cannot overfill a one-slot locker.
	// (Full behavioural coverage lives in the integration test in Task 19.)
	if rc.freeSlots() != 2 {
		t.Fatalf("freeSlots = %d", rc.freeSlots())
	}
	rc.consumeSlot()
	if rc.freeSlots() != 1 {
		t.Fatalf("freeSlots after consume = %d", rc.freeSlots())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -run 'TestGranter|TestCashItemGranter' -v`
Expected: FAIL — undefined identifiers.

- [ ] **Step 3: Implement**

Create `granter.go`:

```go
package coupon

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/wallet"
)

var ErrInventoryFull = errors.New("cash locker full")

// redemptionContext is the mutable per-redemption state the granters share.
// compartmentUsed is advanced by each cash-item grant so a multi-item bundle
// cannot overfill a locker that had room for only some of it.
type redemptionContext struct {
	accountId           uint32
	characterId         uint32
	compartmentId       uuid.UUID
	compartmentCapacity uint32
	compartmentUsed     uint32
	walletModel         wallet.Model
}

func (rc *redemptionContext) freeSlots() uint32 {
	if rc.compartmentUsed >= rc.compartmentCapacity {
		return 0
	}
	return rc.compartmentCapacity - rc.compartmentUsed
}

func (rc *redemptionContext) consumeSlot() { rc.compartmentUsed++ }

type grantedReward struct {
	assetId     uint32
	maplePoints uint32
}

// rewardGranter applies ONE reward inside the redemption transaction. Every
// implementation today writes only to atlas-cashshop's own tables; when a
// reward type owned by another service is added, that granter is the single
// place a saga would be introduced (design §2.1).
type rewardGranter interface {
	Grant(mb *message.Buffer) func(rc *redemptionContext, r Reward) (grantedReward, error)
}

type currencyGranter struct {
	walP wallet.Processor
}

func (g *currencyGranter) Grant(mb *message.Buffer) func(rc *redemptionContext, r Reward) (grantedReward, error) {
	return func(rc *redemptionContext, r Reward) (grantedReward, error) {
		w := rc.walletModel.Award(r.Currency(), r.Amount())
		updated, err := g.walP.Update(mb)(rc.accountId)(w.Credit())(w.Points())(w.Prepaid())
		if err != nil {
			return grantedReward{}, err
		}
		rc.walletModel = updated
		return grantedReward{maplePoints: updated.Points()}, nil
	}
}

type cashItemGranter struct {
	astP asset.Processor
}

func (g *cashItemGranter) Grant(mb *message.Buffer) func(rc *redemptionContext, r Reward) (grantedReward, error) {
	return func(rc *redemptionContext, r Reward) (grantedReward, error) {
		// Re-check capacity HERE, inside the transaction, not only in the
		// pre-flight ladder: the locker can fill between the ladder check and
		// this grant (design Q6 — pre-flight for a deterministic error
		// ordering, in-transaction to close the TOCTOU window).
		if rc.freeSlots() == 0 {
			return grantedReward{}, ErrInventoryFull
		}
		am, err := g.astP.Create(mb)(rc.compartmentId, templateIdFor(r), r.SerialNumber(), r.Quantity(), 0, rc.characterId)
		if err != nil {
			return grantedReward{}, err
		}
		rc.consumeSlot()
		return grantedReward{assetId: am.Id()}, nil
	}
}

func granterFor(rt RewardType, walP wallet.Processor, astP asset.Processor) (rewardGranter, error) {
	switch rt {
	case RewardTypeCurrency:
		return &currencyGranter{walP: walP}, nil
	case RewardTypeCashItem:
		return &cashItemGranter{astP: astP}, nil
	default:
		return nil, fmt.Errorf("%w: no granter for reward type %q", ErrInvalidReward, rt)
	}
}
```

`templateIdFor(r)` resolves the commodity's `ItemId` — the `asset.Processor.Create` signature is `(compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, petId uint32, purchasedBy uint32)`, and `Purchase` (`cashshop/processor.go:197`) passes `ci.ItemId()` as `templateId` and `serialNumber` as `commodityId`. So the cash-item granter needs a `commodity.Processor` too: add `comP commodity.Processor` to `cashItemGranter`, call `g.comP.GetById(r.SerialNumber())` to obtain `ci`, pass `ci.ItemId()` as `templateId` and `r.SerialNumber()` as `commodityId`, and drop the `templateIdFor` helper. Adjust `granterFor`'s signature to `granterFor(rt RewardType, walP wallet.Processor, astP asset.Processor, comP commodity.Processor)` and update the test accordingly (`granterFor(rt, nil, nil, nil)`).

**Pets are out of scope.** Unlike `Purchase`, this granter does not reserve a cash id or create a pet row: a pet reward would need the `NextCashId` + `petP.Create` + `CreateWithCashId` triple (`cashshop/processor.go:157-198`). Reject a cash-item reward whose commodity resolves to `item.ClassificationPet` with `ErrInvalidReward` at **create/PATCH time** (Task 21's REST validation), so a coupon carrying one can never be minted.

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/granter.go services/atlas-cashshop/atlas.com/cashshop/coupon/granter_test.go
git commit -m "feat(coupon): add currency and cash-item reward granters"
```

---

## Task 18: The redemption transaction

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/processor.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/error.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/processor_test.go`

**Interfaces:**
- Consumes: everything from Tasks 10–17.
- Produces: `coupon.Processor` interface with `RedeemAndEmit(characterId uint32, code string) error`, `Redeem(mb *message.Buffer) func(characterId uint32, code string) error`, plus the CRUD methods Task 21 needs (`GetById`, `ByCodeProvider`, `PagedProvider`, `Create`, `Update`, `Delete`); `coupon.NewProcessor(l, ctx, db, lim Limiter) Processor`; and `redemptionError` with `Key() string`.

- [ ] **Step 1: Write the failing error-mapping test**

Create `processor_test.go`:

```go
func TestRedemptionErrorKeys(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrCodeNotFound, "INVALID_COUPON_CODE"},
		{ErrCouponInactive, "COUPON_NOT_REGISTERED"},
		{ErrCouponNotStarted, "COUPON_NOT_REGISTERED"},
		{ErrCouponExpired, "COUPON_EXPIRED"},
		{ErrAlreadyRedeemed, "COUPON_ALREADY_USED"},
		{ErrUsageLimit, "COUPON_USAGE_LIMIT"},
		{ErrInventoryFull, "INVENTORY_FULL"},
		{errors.New("something else"), "UNKNOWN_ERROR"},
	}
	for _, c := range cases {
		if got := ErrorKey(c.err); got != c.want {
			t.Fatalf("ErrorKey(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}

func TestValidateLadder_FirstFailureWins(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	max := uint32(1)

	// Inactive AND expired AND exhausted — the ladder must report the
	// EARLIEST failure so the returned error is deterministic.
	m := NewBuilder("ABC").
		SetActive(false).
		SetExpiresAt(&past).
		SetMaxUses(&max).
		SetRedemptionCount(1).
		Build()
	if err := validate(m, now); !errors.Is(err, ErrCouponInactive) {
		t.Fatalf("err = %v, want ErrCouponInactive", err)
	}

	m2 := NewBuilder("ABC").SetActive(true).SetStartsAt(&future).Build()
	if err := validate(m2, now); !errors.Is(err, ErrCouponNotStarted) {
		t.Fatalf("err = %v, want ErrCouponNotStarted", err)
	}

	m3 := NewBuilder("ABC").SetActive(true).SetExpiresAt(&past).Build()
	if err := validate(m3, now); !errors.Is(err, ErrCouponExpired) {
		t.Fatalf("err = %v, want ErrCouponExpired", err)
	}

	m4 := NewBuilder("ABC").SetActive(true).SetMaxUses(&max).SetRedemptionCount(1).Build()
	if err := validate(m4, now); !errors.Is(err, ErrUsageLimit) {
		t.Fatalf("err = %v, want ErrUsageLimit", err)
	}

	if err := validate(NewBuilder("ABC").SetActive(true).Build(), now); err != nil {
		t.Fatalf("valid coupon rejected: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -run 'TestRedemptionErrorKeys|TestValidateLadder' -v`
Expected: FAIL — `undefined: ErrorKey`.

- [ ] **Step 3: Implement the error mapping**

Create `error.go`:

```go
package coupon

import (
	"errors"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"

	"atlas-cashshop/coupon/redemption"
)

var (
	ErrCodeNotFound     = errors.New("coupon code not found")
	ErrCouponInactive   = errors.New("coupon inactive")
	ErrCouponNotStarted = errors.New("coupon not yet active")
	ErrCouponExpired    = errors.New("coupon expired")
	ErrUsageLimit       = errors.New("coupon usage limit reached")
)

// ErrAlreadyRedeemed is re-exported so callers map one sentinel regardless of
// whether the ladder or the unique index produced it.
var ErrAlreadyRedeemed = redemption.ErrAlreadyRedeemed

// ErrorKey maps a redemption failure to the client error key the
// USE_COUPON_FAILED arm resolves through the writer's "errors" table. The
// mapping lives HERE, once, so the transport layer never re-derives it
// (design §7.4).
func ErrorKey(err error) string {
	switch {
	case errors.Is(err, ErrCodeNotFound):
		return cashcb.CashShopOperationErrorInvalidCouponCode
	case errors.Is(err, ErrCouponInactive), errors.Is(err, ErrCouponNotStarted):
		return cashcb.CashShopOperationErrorCouponNotRegistered
	case errors.Is(err, ErrCouponExpired):
		return cashcb.CashShopOperationErrorCouponExpired
	case errors.Is(err, ErrAlreadyRedeemed):
		return cashcb.CashShopOperationErrorCouponAlreadyUsed
	case errors.Is(err, ErrUsageLimit):
		return cashcb.CashShopOperationErrorCouponUsageLimit
	case errors.Is(err, ErrInventoryFull):
		return cashcb.CashShopOperationErrorInventoryFull
	default:
		return cashcb.CashShopOperationErrorUnknown
	}
}
```

Add `github.com/Chronicle20/atlas/libs/atlas-packet` to `services/atlas-cashshop/atlas.com/cashshop/go.mod` if it is not already required, plus the matching `replace` line if the module's other `replace` entries follow that convention.

- [ ] **Step 4: Implement the validation ladder and the transaction**

Create `processor.go`. The ladder:

```go
// validate is FR-5.4 steps 1-6, first failure wins so the client-visible
// error is deterministic. Step 7 (locker capacity) needs the compartment and
// runs inside Redeem.
func validate(m Model, now time.Time) error {
	if !m.Active() {
		return ErrCouponInactive
	}
	if s := m.StartsAt(); s != nil && now.Before(*s) {
		return ErrCouponNotStarted
	}
	if e := m.ExpiresAt(); e != nil && now.After(*e) {
		return ErrCouponExpired
	}
	if mu := m.MaxUses(); mu != nil && m.RedemptionCount() >= *mu {
		return ErrUsageLimit
	}
	return nil
}
```

The transaction, mirroring `cashshop.ProcessorImpl.Purchase` (`cashshop/processor.go:98-220`) including its `rejectEmit` split:

```go
func (p *ProcessorImpl) RedeemAndEmit(characterId uint32, code string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx, p.lim).Redeem(buf)(characterId, code)
		})
	})
}
```

`Redeem(mb)` body, in order:

1. `code = Normalize(code)`; `if !PlausibleCode(code) { return p.fail(characterId, ErrCodeNotFound) }`.
2. Resolve the character: `c, err := p.chaP.GetById(p.chaP.InventoryDecorator)(characterId)` → `accountId := c.AccountId()`, and the compartment type by the same `job.GetType(c.JobId())` branch as `Purchase` (`cashshop/processor.go:130-136`).
3. Rate-limit gate: `ok, err := p.lim.Allow(p.ctx, p.t, accountId)`; `if !ok { return p.fail(characterId, ErrCodeNotFound) }` — returning the *invalid code* key rather than a distinct "rate limited" key is deliberate: it leaks nothing about whether the attempted code exists (design §7.5).
4. Look up the coupon by `(tenant_id, code)`. Not found → record a failure on the limiter, then `p.fail(characterId, ErrCodeNotFound)`.
5. `validate(m, time.Now().UTC())` → on error, record a limiter failure and `p.fail(characterId, err)`.
6. Prior-redemption check for this account (friendly fast path) → `ErrAlreadyRedeemed`.
7. Load the compartment (`p.cicP.GetByAccountIdAndType(accountId, compartmentType)`); `if uint32(len(ccm.Assets())) + uint32(m.Rewards().CashItemCount()) > ccm.Capacity() { … ErrInventoryFull }`.
8. `ok, err := incrementRedemptionCount(tx, p.t, m.Id())`; `if !ok { … ErrUsageLimit }` — the FR-5.5 conditional bump.
9. `redemption.Create(tx, p.t, redemption.Entity{...})` with a freshly generated `TransactionId` (a `uuid.New()` for log correlation and the audit trail — there is no saga transaction key, design §2.2). `ErrAlreadyRedeemed` from the unique index resolves the concurrent double-submit.
10. Build the `redemptionContext` and grant each reward through `granterFor(...)`, collecting `assetIds` and the final `maplePoints`.
11. `mb.Put(cashshop.EnvEventTopicStatus, cashshop2.CouponRedeemedStatusEventProvider(characterId, ccm.Id(), assetIds, maplePoints))` — enqueued **inside** the transaction so it rides the outbox with the commit.

`p.fail` mirrors `Purchase`'s `rejectEmit`: it captures a closure that fires `producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.CouponFailedStatusEventProvider(characterId, ErrorKey(err)))` on the **direct** producer path outside the transaction, and returns a sentinel that aborts the closure. An event asserting "nothing happened" must not ride an outbox that implies a commit (`cashshop/processor.go:100-103`). A failure discovered after writes have begun (step 10) rolls back and then emits on the direct path too.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./coupon/... -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/
git commit -m "feat(coupon): implement the local-transaction redemption path"
```

---

## Task 19: Concurrency and rollback tests

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/redeem_integration_test.go`

**Interfaces:**
- Consumes: Task 18's processor.
- Produces: the tests that prove the design §2 decision. These are the tests that actually matter here.

Use whatever real-database test harness this module already has (check `services/atlas-cashshop/atlas.com/cashshop/**/*_test.go` for an existing Postgres/testcontainers helper and reuse it). A mocked counter proves nothing — the whole point is the conditional `UPDATE`'s `RowsAffected` and the unique index.

- [ ] **Step 1: Write the failing tests**

```go
func TestRedeem_MaxUsesOneUnderConcurrency(t *testing.T) {
	db, ctx, cleanup := newTestDB(t)
	defer cleanup()
	max := uint32(1)
	c := seedCoupon(t, db, ctx, NewBuilder("RACE1").SetActive(true).SetMaxUses(&max).
		SetRewards(Rewards{NewCurrencyReward(2, 100)}).Build())

	// Two DIFFERENT accounts, so the per-account unique index is not what
	// resolves this — the conditional counter bump is.
	results := runConcurrently(t, 2, func(i int) error {
		return NewProcessor(testLogger(), ctx, db, NoopLimiter()).
			RedeemAndEmit(characterFor(t, i), "RACE1")
	})

	if got := countRedemptions(t, db, ctx, c.Id()); got != 1 {
		t.Fatalf("redemptions = %d, want 1", got)
	}
	if got := reloadCoupon(t, db, ctx, c.Id()).RedemptionCount(); got != 1 {
		t.Fatalf("redemption_count = %d, want 1", got)
	}
	assertExactlyOneFailureWithKey(t, results, "COUPON_USAGE_LIMIT")
}

func TestRedeem_SameAccountTwiceUnderConcurrency(t *testing.T) {
	db, ctx, cleanup := newTestDB(t)
	defer cleanup()
	c := seedCoupon(t, db, ctx, NewBuilder("RACE2").SetActive(true).
		SetRewards(Rewards{NewCurrencyReward(2, 100)}).Build())

	// Same account, no max_uses — the unique index on
	// (tenant_id, coupon_id, account_id) is what resolves this.
	results := runConcurrently(t, 2, func(int) error {
		return NewProcessor(testLogger(), ctx, db, NoopLimiter()).
			RedeemAndEmit(sameAccountCharacter(t), "RACE2")
	})

	if got := countRedemptions(t, db, ctx, c.Id()); got != 1 {
		t.Fatalf("redemptions = %d, want 1", got)
	}
	assertExactlyOneFailureWithKey(t, results, "COUPON_ALREADY_USED")
}

func TestRedeem_ItemGrantFailureRollsEverythingBack(t *testing.T) {
	db, ctx, cleanup := newTestDB(t)
	defer cleanup()
	// A bundle whose currency leg succeeds and whose item leg cannot: the
	// locker is seeded full. This is the single test that proves the
	// local-transaction decision (design §2) — with a saga it would be a
	// compensation assertion; here it is a rollback assertion.
	c := seedCoupon(t, db, ctx, NewBuilder("ROLLBACK").SetActive(true).
		SetRewards(Rewards{NewCurrencyReward(2, 500), NewCashItemReward(50200000, 1)}).Build())
	acct := seedAccountWithFullLocker(t, db, ctx)
	before := walletSnapshot(t, db, ctx, acct)

	err := NewProcessor(testLogger(), ctx, db, NoopLimiter()).
		RedeemAndEmit(characterOf(t, acct), "ROLLBACK")
	if ErrorKey(err) != "INVENTORY_FULL" && lastFailureKey(t) != "INVENTORY_FULL" {
		t.Fatalf("expected INVENTORY_FULL, got err=%v", err)
	}

	if after := walletSnapshot(t, db, ctx, acct); after != before {
		t.Fatalf("wallet changed: %+v -> %+v", before, after)
	}
	if got := lockerAssetCount(t, db, ctx, acct); got != fullLockerSize {
		t.Fatalf("locker changed: %d", got)
	}
	if got := reloadCoupon(t, db, ctx, c.Id()).RedemptionCount(); got != 0 {
		t.Fatalf("redemption_count = %d, want 0 (code must remain redeemable)", got)
	}
	if got := countRedemptions(t, db, ctx, c.Id()); got != 0 {
		t.Fatalf("redemption row survived rollback: %d", got)
	}
}

func TestRedeem_CaseAndWhitespaceInsensitive(t *testing.T) {
	db, ctx, cleanup := newTestDB(t)
	defer cleanup()
	seedCoupon(t, db, ctx, NewBuilder("MIXEDCASE").SetActive(true).
		SetRewards(Rewards{NewCurrencyReward(2, 1)}).Build())

	for _, submitted := range []string{"mixedcase", "  MixedCase  ", "\tMIXEDCASE\n"} {
		if err := NewProcessor(testLogger(), ctx, db, NoopLimiter()).
			RedeemAndEmit(freshCharacter(t), submitted); err != nil {
			t.Fatalf("submitting %q failed: %v", submitted, err)
		}
	}
}

func TestRedeem_EachLadderOutcomeMapsToItsKey(t *testing.T) {
	// One sub-test per FR-5.4 outcome -> FR-3.4 key:
	// INVALID_COUPON_CODE, COUPON_NOT_REGISTERED (inactive),
	// COUPON_NOT_REGISTERED (before starts_at), COUPON_EXPIRED,
	// COUPON_ALREADY_USED, COUPON_USAGE_LIMIT, INVENTORY_FULL.
	// Each seeds exactly the one condition and asserts the emitted
	// COUPON_FAILED body's Error field.
}
```

Implement `runConcurrently`, `seedCoupon`, `countRedemptions`, `reloadCoupon`, `walletSnapshot`, `lockerAssetCount`, and `assertExactlyOneFailureWithKey` as local test helpers in this file, building models through `NewBuilder` (Builder pattern — no `*_testhelpers.go`). `runConcurrently` launches its goroutines with `routine.Go` or inside a `//goroutine-guard:allow test fan-out` comment — `tools/goroutine-guard.sh` bans bare `go` statements.

- [ ] **Step 2: Run to verify they fail**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/ -run TestRedeem -race -v`
Expected: FAIL (helpers undefined, then real behavioural failures).

- [ ] **Step 3: Make them pass**

Fix Task 18's processor until every assertion holds. Expected real defects to find here: the `Redeem` closure returning `nil` on a rejection path so the transaction commits anyway; the `rejectEmit` closure not firing; `compartmentUsed` not advancing between two item grants.

- [ ] **Step 4: Run the full module suite**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./... -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/redeem_integration_test.go
git commit -m "test(coupon): prove concurrency guarantees and transaction rollback"
```

---

## Task 20: Consume the redemption command in `atlas-cashshop`

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go`

**Interfaces:**
- Consumes: Task 15's contracts, Task 16's limiter, Task 18's processor.
- Produces: `handleCommandRequestCouponRedemption(db, lim)` registered on `EnvCommandTopic`.

- [ ] **Step 1: Add the handler**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go`, following `handleCommandRequestPurchase`:

```go
func handleCommandRequestCouponRedemption(db *gorm.DB, lim coupon.Limiter) message.Handler[cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestCouponRedemption {
			return
		}
		_ = coupon.NewProcessor(l, ctx, db, lim).RedeemAndEmit(c.CharacterId, c.Body.Code)
	}
}
```

- [ ] **Step 2: Register it**

`InitHandlers` currently takes `(l)(db)(rf)`. Widen it to `(l)(db, lim)(rf)` and add:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCouponRedemption(db, lim)))); err != nil {
				return err
			}
```

- [ ] **Step 3: Build the limiter in main.go and pass it through**

In `main.go`, after the Redis client (Task 16 Step 5), construct the limiter from the resolved tenant configuration accessor and pass it to `cashshop.InitHandlers(l)(db, lim)`. Because the threshold and window are per tenant, construct the limiter with the **defaults** at boot and have the processor resolve the per-tenant values from `configuration.GetCouponLimits(l, ctx, t.Id())` at redemption time — the limiter holds only the Redis client, and `Allow`/`RecordFailure` take the threshold/window as parameters if the per-tenant lookup is done per call. Pick one shape and make `NewLimiter`'s signature match; if you move the threshold/window to per-call parameters, update Task 16's tests in the same commit.

- [ ] **Step 4: Build and test**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./... -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/
git commit -m "feat(atlas-cashshop): consume REQUEST_COUPON_REDEMPTION"
```

---

## Task 21: Admin REST surface

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/{resource,rest}.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/{processor,resource,rest}.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/{processor,resource,rest}.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/resource_test.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/generator_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go`

**Interfaces:**
- Consumes: Tasks 12–14, 18.
- Produces: the endpoints in PRD §5, exactly. `coupon.RestModel` (`GetName() == "coupons"`), `batch.RestModel` (`"coupon-batches"`), `redemption.RestModel` (`"coupon-redemptions"`); `batch.Generate(count, prefix, length uint32, ...) ([]coupon.Model, batch.Model, error)`.

- [ ] **Step 1: Write the failing generator test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/generator_test.go`:

```go
func TestGenerateCode_UsesUnambiguousAlphabet(t *testing.T) {
	for i := 0; i < 200; i++ {
		c, err := generateCode("", 12)
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != 12 {
			t.Fatalf("len = %d", len(c))
		}
		if strings.ContainsAny(c, "O0I1L") {
			t.Fatalf("ambiguous character in %q", c)
		}
		for _, r := range c {
			if !strings.ContainsRune(alphabet, r) {
				t.Fatalf("character %q not in alphabet", r)
			}
		}
	}
}

func TestGenerateCode_HonoursPrefix(t *testing.T) {
	c, err := generateCode("EVENT-", 6)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(c, "EVENT-") {
		t.Fatalf("prefix missing: %q", c)
	}
	if len(c) != len("EVENT-")+6 {
		t.Fatalf("len = %d", len(c))
	}
}

func TestGenerateCode_IsNotPredictable(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		c, err := generateCode("", 10)
		if err != nil {
			t.Fatal(err)
		}
		if seen[c] {
			t.Fatalf("duplicate in 500 draws: %q", c)
		}
		seen[c] = true
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go test ./coupon/batch/ -v`
Expected: FAIL — `undefined: generateCode`.

- [ ] **Step 3: Implement the generator**

```go
// alphabet excludes O/0 and I/1/L: these codes are read off a screen and
// typed by hand.
const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// generateCode draws from crypto/rand — a code from math/rand is guessable,
// and these are secrets.
func generateCode(prefix string, length uint32) (string, error) {
	b := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return coupon.Normalize(prefix + string(b)), nil
}
```

`Generate` inserts each code with the unique index as the collision detector and **retries a collision rather than skipping it**, so the response's created count always equals the requested `count`. Cap the retry loop (e.g. 10 attempts per code) and fail the whole batch with a clear error rather than returning a short count. Set `max_uses = 1` on every generated code and stamp `batch_id`.

- [ ] **Step 4: Write the resources**

Follow `wishlist/resource.go` exactly: `InitResource(si)(db)` returning a `server.RouteInitializer`, `rest.RegisterHandler(l)(si)` for reads, `rest.RegisterInputHandler[RestModel](l)(si)` for writes, `paginate.ParseParams` + `server.MarshalPaginatedResponse` for lists. Routes:

| Method | Path |
|---|---|
| GET | `/coupons` (filters `filter[code]`, `filter[active]`, `filter[batchId]`, `filter[expiresBefore]`, `filter[expiresAfter]`) |
| GET | `/coupons/{id}` |
| POST | `/coupons` |
| PATCH | `/coupons/{id}` |
| DELETE | `/coupons/{id}` |
| GET | `/coupons/{id}/redemptions` |
| POST | `/coupon-batches` |
| GET | `/coupon-batches`, `/coupon-batches/{id}` |
| GET | `/coupon-redemptions?filter[accountId]=` |

Error mapping (PRD §5):

| Condition | Status |
|---|---|
| Duplicate normalized code on create | `409` |
| Reward references an unknown commodity serial, or one whose item is `item.ClassificationPet` | `422` |
| `expiresAt` ≤ `startsAt` | `422` |
| `maxUses` < current `redemptionCount` on PATCH | `422` |
| Delete with redemptions present | `409` |

`filter[code]` is normalized before querying. Every query is scoped from `tenant.MustFromContext(ctx)`; **no endpoint accepts a tenant id in a body** (FR-7.4). **There is no redemption endpoint** — the packet path is the only trigger; a REST redeem endpoint would be an unauthenticated reward faucet.

- [ ] **Step 5: Write the resource tests**

Create `coupon/resource_test.go` covering: create returns 201 with the normalized code; a duplicate differing only in case returns 409; `expiresAt <= startsAt` returns 422; a reward with an unknown serial returns 422; PATCH lowering `maxUses` below `redemptionCount` returns 422; DELETE with a redemption returns 409; the list endpoint honours `filter[active]`. Model the HTTP plumbing on `wishlist/resource_paginate_test.go`.

- [ ] **Step 6: Register the routes**

In `main.go`, add three `AddRouteInitializer` lines after the existing five:

```go
			AddRouteInitializer(coupon.InitResource(GetServer())(db)).
			AddRouteInitializer(batch.InitResource(GetServer())(db)).
			AddRouteInitializer(redemption.InitResource(GetServer())(db)).
```

- [ ] **Step 7: Build and test**

Run: `cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./... -race`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/
git commit -m "feat(coupon): add admin REST surface for coupons, batches, and redemptions"
```

---

## Task 22: `atlas-channel` handler arm and response wiring

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/cashshop/{processor,producer}.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/cashshop/processor_test.go`

**Interfaces:**
- Consumes: Task 5's codec, Task 11's channel-side `coupon` helpers, Task 15's channel-side contracts.
- Produces: `cashshop.Processor.RequestCouponRedemption(characterId uint32, code string) error`, `RequestCouponRedemptionCommandProvider`, the `CashShopOperationUseCoupon` handler arm, and `handleStatusEventCouponRedeemed` / `handleStatusEventCouponFailed`.

- [ ] **Step 1: Add the producer and processor method**

In `services/atlas-channel/atlas.com/channel/cashshop/producer.go`, mirroring `RequestPurchaseCommandProvider`:

```go
func RequestCouponRedemptionCommandProvider(characterId uint32, code string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestCouponRedemption,
		Body:        cashshop.RequestCouponRedemptionCommandBody{Code: code},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `processor.go`, add to the `Processor` interface and implement:

```go
func (p *ProcessorImpl) RequestCouponRedemption(characterId uint32, code string) error {
	p.l.Infof("Character [%d] attempting coupon redemption.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestCouponRedemptionCommandProvider(characterId, code))
}
```

The log line deliberately omits the code: a successful code is a secret and a failed one is attacker-supplied. The **service** logs the normalized code with the outcome (PRD §8 observability) where it is already inside the tenant's own audit trail.

- [ ] **Step 2: Write the failing handler test**

Add to `services/atlas-channel/atlas.com/channel/cashshop/processor_test.go` (or a new `socket/handler/cash_shop_operation_test.go` if that is where handler tests live in this module — check first) a test that a submitted code of `"  abc123  "` reaches the producer as `"ABC123"`, and that an empty/over-length code produces **no** command:

```go
func TestUseCouponArm_NormalizesAndShortCircuits(t *testing.T) {
	cases := []struct {
		name      string
		submitted string
		wantSent  bool
		wantCode  string
	}{
		{"normalizes", "  abc123  ", true, "ABC123"},
		{"empty short-circuits", "   ", false, ""},
		{"over-length short-circuits", strings.Repeat("A", coupon.MaxCodeLength+1), false, ""},
	}
	// Drive CashShopOperationHandleFunc with a reader carrying the USE_COUPON
	// mode byte from the test tenant's readerOptions and assert on the
	// captured producer message / announced packet.
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./... -run TestUseCouponArm -v`
Expected: FAIL.

- [ ] **Step 4: Add the handler arm**

In `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`, add the constant to the block (in the block's existing style, with the mode comment omitted because it is per version and config-resolved):

```go
	CashShopOperationUseCoupon             = "USE_COUPON"
```

and the arm, placed to match the operation's position among its siblings:

```go
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationUseCoupon) {
			sp := &cashsb.ShopOperationUseCoupon{}
			sp.Decode(l, ctx)(r, readerOptions)
			code := coupon.Normalize(sp.Code())
			if !coupon.PlausibleCode(code) {
				// Empty or over-length: cannot match any stored code, so
				// answer directly instead of paying a round trip. Also the
				// first line of brute-force defence.
				_ = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(
					cashcb.CashShopUseCouponFailedBody(cashcb.CashShopOperationErrorInvalidCouponCode))(s)
				return
			}
			_ = cashshop.NewProcessor(l, ctx).RequestCouponRedemption(s.CharacterId(), code)
			return
		}
```

Import `"atlas-channel/cashshop/coupon"`.

- [ ] **Step 5: Add the status-event handlers**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`, register two new handlers and implement them. `handleStatusEventCouponRedeemed` reuses the asset-id → `CashInventoryItem` projection that `handleStatusEventPurchase` already performs (`consumer.go:105-124`), once per `AssetIds` entry, then announces:

```go
			err = session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(
				cashpkt.CashShopUseCouponDoneBody(items, int32(e.Body.MaplePoints), nil, 0))(s)
```

`refs` is `nil` and `meso` is `0` — meso rewards are explicitly out of scope (PRD §2). **`maplePoint` is populated as the absolute post-award balance only if `derivation.md` confirms the client treats it that way** (PRD Q5 / design §12); if the decompile shows a delta, change the event body's semantics and this call together, and record the answer in the codec's doc comment.

`handleStatusEventCouponFailed` announces `cashpkt.CashShopUseCouponFailedBody(e.Body.Error)`.

Both guard on `e.Type` and on `t.Is(sc.Tenant())` exactly as the existing handlers do.

- [ ] **Step 6: Run the tests and build**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(atlas-channel): wire the USE_COUPON request and response arms"
```

---

## Task 23: Admin UI

**Files:**
- Create: `services/atlas-ui/src/types/models/coupon.ts`
- Create: `services/atlas-ui/src/services/api/coupons.service.ts`
- Create: `services/atlas-ui/src/pages/{CouponsPage,CouponDetailPage,coupons-columns}.tsx`
- Create: `services/atlas-ui/src/pages/__tests__/CouponsPage.test.tsx`
- Modify: `services/atlas-ui/src/services/api/index.ts`, `src/App.tsx`, `src/components/app-sidebar-items.ts`, `src/lib/breadcrumbs/routes.ts`

**Interfaces:**
- Consumes: Task 21's endpoints.
- Produces: the `/coupons` and `/coupons/:id` routes.

- [ ] **Step 1: Types and Zod schemas**

Create `services/atlas-ui/src/types/models/coupon.ts` with `Coupon`, `CouponBatch`, `CouponRedemption`, and a **discriminated-union** Zod schema for rewards so an invalid combination cannot reach the API:

```ts
export const rewardSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("CURRENCY"),
    currency: z.union([z.literal(1), z.literal(2), z.literal(3)]),
    amount: z.coerce.number().int().positive(),
  }),
  z.object({
    type: z.literal("CASH_ITEM"),
    serialNumber: z.coerce.number().int().positive(),
    quantity: z.coerce.number().int().positive(),
  }),
]);

export const couponFormSchema = z
  .object({
    code: z.string().trim().max(32).optional(),
    description: z.string().optional(),
    active: z.boolean().default(true),
    startsAt: z.string().datetime().optional(),
    expiresAt: z.string().datetime().optional(),
    maxUses: z.coerce.number().int().positive().optional(),
    rewards: z.array(rewardSchema).min(1),
  })
  .refine((v) => !v.startsAt || !v.expiresAt || v.expiresAt > v.startsAt, {
    message: "expiresAt must be after startsAt",
    path: ["expiresAt"],
  });
```

- [ ] **Step 2: API client**

Create `coupons.service.ts` following `commodities.service.ts` (same `fetchAll` / pagination helpers). **Every POST/PATCH body uses the JSON:API envelope** (`{ data: { type, attributes } }`) — a bare body is silently dropped by the input handlers (`bug_ui_jsonapi_envelope_required_for_input_handlers`). Export `couponsService`, `couponBatchesService`, `couponRedemptionsService` and re-export from `src/services/api/index.ts`.

- [ ] **Step 3: Write the failing page test**

Create `services/atlas-ui/src/pages/__tests__/CouponsPage.test.tsx` modelled on `AccountsPage.test.tsx`: renders the list from a mocked service, filters by active, opens the create dialog, and asserts the submitted body is JSON:API-enveloped.

Run: `cd services/atlas-ui && npm test -- CouponsPage`
Expected: FAIL — module not found.

- [ ] **Step 4: Build the pages**

`CouponsPage.tsx` + `coupons-columns.tsx` + `CouponDetailPage.tsx`, matching the list/columns/detail trio of `AccountsPage.tsx`. TanStack Query for fetching, react-hook-form + Zod for the create form and bulk-generate dialog, shadcn/ui components, Tailwind.

- Columns: code, status (active / scheduled / expired / exhausted), reward summary, `redemptionCount` / `maxUses`, window.
- Create form: `code` optional (blank means "generate"); rewards as a field array switching shape on `type`.
- Bulk-generate dialog: reward definition + count; the response carries every generated code and the dialog offers a **client-side CSV download** built from it (no extra endpoint).
- Detail page: the code's redemption history via `/coupons/{id}/redemptions`.
- Disable/enable toggles `active` in place; delete is confirmed and **disabled once `redemptionCount > 0`**, matching the server's 409.
- **No global redemption list** (design Q7).

- [ ] **Step 5: Register route, nav, and breadcrumb**

`App.tsx` (lazy import + two `<Route>` entries mirroring `/accounts` and `/accounts/:id`), `src/components/app-sidebar-items.ts`, `src/lib/breadcrumbs/routes.ts`. Check whether `src/lib/__tests__/deployment-routes.test.ts` enumerates routes and needs the new entries.

- [ ] **Step 6: Run tests and build**

Run:
```bash
cd services/atlas-ui
npm test
npm run build
```
`npm run build` type-checks the tests too, so it is not optional (`feedback_atlas_ui_pertask_verify_needs_build_not_just_vitest`). If npm fails, load nvm 22 first (`reference_atlas_ui_npm_nvm_and_lint_baseline`).
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/
git commit -m "feat(atlas-ui): add coupon administration pages"
```

---

## Task 24: Full verification gate

**Files:** none created; this task produces evidence, not code.

**Interfaces:**
- Consumes: everything.
- Produces: a green run of every gate in `CLAUDE.md` + design §11, recorded in the commit message.

- [ ] **Step 1: Per-module test and vet**

Run, from the worktree root:
```bash
for m in libs/atlas-packet services/atlas-cashshop/atlas.com/cashshop services/atlas-channel/atlas.com/channel tools/packet-audit; do
  (cd "$m" && echo "== $m ==" && go vet ./... && go test -race ./...) || exit 1
done
```
Expected: clean in all four.

- [ ] **Step 2: Repo-root guards**

Run:
```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
```
Expected: every one exits 0.

- [ ] **Step 3: Lint**

Run: `tools/lint.sh` (fix mode) then `tools/lint.sh --check`.
Expected: `--check` exits 0. If it false-fails, confirm nvm 22 is active (`bug_lint_check_false_fails_without_nvm`) and that no other worktree holds the golangci-lint lock.

- [ ] **Step 4: packet-audit checks**

Run:
```bash
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit matrix
```
plus the fname-doc and n-a consistency checks named in `docs/packets/PROCESS.md`.
Expected: all exit 0.

- [ ] **Step 5: Docker bake**

Run: `docker buildx bake atlas-cashshop atlas-channel atlas-configurations`
Expected: all three succeed. `atlas-saga-orchestrator` is **not** in this list — design §2 removed it from the blast radius. `go.mod` was touched in `atlas-cashshop` (Task 16 added `atlas-redis`), so this step is mandatory: only the bake catches a missing `COPY libs/...` line in the shared root `Dockerfile`. If `libs/atlas-redis` or `libs/atlas-packet` is absent from that Dockerfile's mod-only or source COPY blocks, add the two lines and re-bake.

- [ ] **Step 6: atlas-ui**

Run: `cd services/atlas-ui && npm test && npm run build`
Expected: PASS.

- [ ] **Step 7: Code review before PR**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer` (Go changed), and `frontend-guidelines-reviewer` (TS changed). Also dispatch `packet-completeness-critic` — this is a packet task, so it needs `docs/tasks/task-206-cash-shop-coupon-codes/coverage-manifest.yaml` (schema in `docs/packets/PROCESS.md`) declaring the op × version cells this task claims. Write that manifest before dispatching. Pin review subagents to a cheaper model (`feedback_review_workflows_use_cheaper_model`).

- [ ] **Step 8: End-to-end on a live v83 tenant**

With the branch deployed, in the Cash Shop Coupon tab:
- a valid code credits the wallet and/or locker and the open Cash Shop window updates **without a relog**;
- the same code again shows the "already used" message;
- a garbage code shows the "invalid code" message;
- an expired code shows the "expired" message.

Record the outcome (including any failure) in `docs/tasks/task-206-cash-shop-coupon-codes/`. A step that could not be run is reported as not run — never as passed.

- [ ] **Step 9: Commit the evidence**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/
git commit -m "docs(task-206): record verification gate results and coverage manifest"
```

---

## Self-Review

**Spec coverage.** Design §1 Q1–Q7 → Tasks 18/19 (local transaction), 8 (full errors enum), 17/18 (Q6 dual capacity check), 2–4 (Q2, Q4, Q5), 23 (Q7 — no global list). §2 → 18, 19. §2.1 → 17. §2.2 → 14, 18. §3.1 → 15. §4 → 5, 9. §5.1 → 7. §5.2 → 8. §5.3 → 1, 8. §5.4 → 6. §5.5 → 7, 8, 24. §6 → 22. §7.1 → 13, 14. §7.2 → 10, 12. §7.3 → 18, 19. §7.4 → 18. §7.5 → 16. §8 → 21. §9 → 23. §10 → 5, 12, 16, 19, 21, 23. §11 → 24. §12 risks → 2–4 (version-by-version, no interleave), 8 (v92 diff as a template bug), 22 (Q5 blocking on the decompile), 16 (config-resolved defaults). PRD FR-6 is intentionally unimplemented — design §13.1 supersedes it, and the Global Constraints say so explicitly.

**Known gaps carried deliberately.** Task 20 Step 3 leaves one shape decision (per-tenant limiter values as constructor args vs per-call params) to the implementer with instructions to update Task 16's tests in the same commit — both shapes are correct and the choice depends on how `configuration.GetCouponLimits` reads at the call site. Task 17 Step 3 revises its own sketch mid-step (the `templateIdFor` placeholder is replaced by a `commodity.Processor` lookup) because the correct signature only becomes visible against `asset.Processor.Create`; the step states the final form explicitly.

**Type consistency.** `Normalize` / `PlausibleCode` / `MaxCodeLength` are identical in both `coupon` packages (11, 18, 22). `Rewards` is the jsonb type in `coupon` and is referenced as `coupon.Rewards` by `redemption` (12, 14). `ErrorKey` is the single error→key mapping (18) and is used by both the failure emitter (18) and the tests (19). `wallet.Model.Award` (10) is called only by `currencyGranter` (17). `CouponRedeemedBody` / `CouponFailedBody` field names and JSON tags are declared once (15) and mirrored verbatim into `atlas-channel` (15) before being consumed (22). `CashShopOperationErrorInvalidCouponCode` resolves to `"INVALID_COUPON_CODE"` after Task 6, and Tasks 8 and 18 both depend on that.
