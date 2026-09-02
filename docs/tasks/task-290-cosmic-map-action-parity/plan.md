# Cosmic Map-Action Parity — Plan A: Foundation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct every pre-existing defect in the map-action engine and widen its
document contract, so that the 72 script conversions in Plan B and Plan C are
authored against an engine that cannot silently drop a field, silently ignore an
operation, or silently spawn into the wrong field instance.

**Architecture:** Five layers change, bottom-up. `libs/atlas-script-core/condition`
gains the one field (`values`) its model still lacks relative to the canonical
`saga.ValidationConditionInput`. `atlas-map-actions` stops truncating conditions on
the way to the query aggregator and stops swallowing unknown operations. The
`spawn_monster` path gains a real field instance, a `spawnIfAbsent` idempotency
guard decided inside `atlas-monsters` against its own registry (not a cross-service
read-then-write), and a `monsterIds` random-choice param. The JSON schema stops
being three hand-maintained lists and becomes a generated artifact with a `--check`
gate. `tools/catalog-lint` — which already models the map-action subdomains but runs
only in CI — gains the structural seed rules and gets wired into `tools/verify.sh`.

**Tech Stack:** Go 1.27.0, GORM, JSON:API (`manyminds/api2go`), `libs/atlas-saga`
step contracts, `libs/atlas-seeder` filesystem catalogs, bash generator scripts with
a `--check` mode, `tools/catalog-lint` (its own Go module, outside `go.work`).

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md))

## Global Constraints

- **Seed layout is the PRD's, not design D1's.** The reviewer rejected design D1's
  shared-root migration. Map-action seeds stay replicated byte-identically across
  all **11** version roots: `deploy/seed/gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}`
  and `deploy/seed/jms/185_1`. `script/groups.go:17` keeps
  `seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")` — do NOT
  switch it to `NewFilesystemCatalogSourceWithShared`. Design FR-1.6 therefore stands
  as written: a byte-identity replication check across the 11 roots (Task A10).
- **Every operation param is a string.** `map_script_schema.json:111-117` declares
  `params` as `"additionalProperties": {"type": "string"}`. Booleans and numbers are
  string-encoded (`"spawnIfAbsent": "true"`, `"x": "188"`). No task may introduce a
  nested or non-string param.
- **Seed document envelope** is JSON:API:
  `{"data": {"type": "map-action", "id": "<scriptName>", "attributes": {"scriptName": ..., "description": ..., "rules": [...]}}}`.
  No `scriptType` attribute — the hook is derived from the directory
  (`script/subdomain_on_user_enter.go`, `script/subdomain_on_first_user_enter.go`).
- **Seed JSON formatting**: 2-space indent, object keys sorted alphabetically at
  every level, trailing newline. Verify against
  `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json`.
- **Never land a stub.** Every switch arm, handler and status response this plan adds is
  fully implemented in the task that adds it — no deferral comments, no unimplemented
  arms, no not-implemented responses.
- **Preserve existing line endings.** Do not normalize CRLF→LF as a side effect.
- **Gate**: the flagless `tools/verify.sh` must exit 0 before this plan is called
  done. `--quick` and `--no-docker` do not count.

## Module roots (the `go build` / `go test` cwd for each area)

| Area | Module root |
|---|---|
| `libs/atlas-script-core/condition` | `libs/atlas-script-core` |
| `libs/atlas-saga` | `libs/atlas-saga` |
| `libs/atlas-seeder` | `libs/atlas-seeder` |
| `services/atlas-map-actions/atlas.com/map-actions/**` | `services/atlas-map-actions/atlas.com/map-actions` |
| `services/atlas-monsters/atlas.com/monsters/**` | `services/atlas-monsters/atlas.com/monsters` |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/**` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` |
| `tools/catalog-lint` | `tools/catalog-lint` (own module, run with `GOWORK=off`) |

---

## Task A1: `condition.Model` gains `values`

The canonical cross-service wire struct `saga.ValidationConditionInput`
(`libs/atlas-saga/validation.go:65-75`) carries `Values []int`. The script-core
condition model (`libs/atlas-script-core/condition/model.go:8-17`) carries every
other field — `step`, `worldId`, `channelId`, `includeEquipped` — but not `values`.
Without it the `in` operator (`libs/atlas-saga/validation.go:60`) is unreachable from
any script document, which blocks Plan C's `isCygnus()` (`jobId in [...]`) and
`transportState`.

`Builder.Build()` (`builder.go:73-94`) currently rejects an empty `value`. An `in`
condition supplies `values` and no scalar `value`, so that guard must accept either.

### Files

- `libs/atlas-script-core/condition/model.go` — add the `values []string` field and a `Values() []string` getter
- `libs/atlas-script-core/condition/builder.go` — add `values []string`, `SetValues`, `AddValue`; relax the `Build()` value guard
- `libs/atlas-script-core/condition/model_test.go` — **new file**
- `libs/atlas-saga/validation.go` — read-only; the canonical shape being mirrored

Module root: `libs/atlas-script-core`.

Patterns to copy: the existing setter/getter pair `SetIncludeEquipped`
(`libs/atlas-script-core/condition/builder.go:67-70`) and `IncludeEquipped()`
(`libs/atlas-script-core/condition/model.go:69-71`) — same shape, unexported field,
value receiver on the getter.

**Interfaces:**
- Produces: `condition.Model.Values() []string`; `condition.Builder.SetValues(vs []string) *Builder`; `condition.Builder.AddValue(v string) *Builder`. Values are kept as **strings** at this layer, matching how `value` and `referenceId` are already kept as strings (`model.go:11-12`) so a context reference like `{context.itemId}` survives; the map-actions evaluator converts to `[]int` at the aggregator boundary (Task A3).

- [ ] **Step 1: Write the failing test**

`libs/atlas-script-core/condition/model_test.go` — new file, package `condition`,
table-free (each case is its own `func Test...`). No fixtures or helpers exist in
this package today; construct through `NewBuilder()` directly.

| test function | build call | assertion |
|---|---|---|
| `TestBuilderSetValues` | `NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000","1100","1110"}).Build()` | no error; `m.Values()` deep-equals `[]string{"1000","1100","1110"}`; `m.Value()` == `""` |
| `TestBuilderAddValue` | `NewBuilder().SetType("jobId").SetOperator("in").AddValue("1000").AddValue("1100").Build()` | no error; `m.Values()` deep-equals `[]string{"1000","1100"}` |
| `TestBuilderValuesOmittedDefaultsNil` | `NewBuilder().SetType("level").SetOperator(">=").SetValue("10").Build()` | no error; `m.Values()` is `nil`; `m.Value()` == `"10"` |
| `TestBuildRequiresValueOrValues` | `NewBuilder().SetType("level").SetOperator(">=").Build()` | error, message exactly `"value or values is required"` |
| `TestBuildAcceptsValuesWithoutValue` | `NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000"}).Build()` | no error |
| `TestBuildStillRequiresType` | `NewBuilder().SetOperator("=").SetValue("1").Build()` | error, message exactly `"condition type is required"` |
| `TestBuildStillRequiresOperator` | `NewBuilder().SetType("level").SetValue("1").Build()` | error, message exactly `"operator is required"` |

Use `reflect.DeepEqual` for the slice comparisons.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd libs/atlas-script-core && go test ./condition/... -run 'TestBuilder|TestBuild' -v`
Expected: compile failure — `m.Values undefined`, `SetValues undefined`, `AddValue undefined`.

- [ ] **Step 3: Add the field, getter and setters**

In `model.go`, add to `Model` after `includeEquipped`:

```go
	values          []string // For the `in` operator: the candidate set. Kept as
	                         // strings for the same reason value/referenceId are —
	                         // a context reference must survive to the boundary.
```

and after `IncludeEquipped()`:

```go
// Values returns the candidate set for the `in` operator. Nil for scalar conditions.
func (c Model) Values() []string {
	return c.values
}
```

In `builder.go`, add `values []string` to `Builder`, and:

```go
// SetValues sets the candidate set for the `in` operator, replacing any prior set.
func (b *Builder) SetValues(values []string) *Builder {
	b.values = values
	return b
}

// AddValue appends one candidate to the `in` operator's set.
func (b *Builder) AddValue(value string) *Builder {
	b.values = append(b.values, value)
	return b
}
```

- [ ] **Step 4: Relax the `Build()` value guard**

Replace `builder.go:80-82`:

```go
	if b.value == "" {
		return Model{}, errors.New("value is required")
	}
```

with:

```go
	if b.value == "" && len(b.values) == 0 {
		return Model{}, errors.New("value or values is required")
	}
```

and add `values: b.values,` to the returned `Model` literal (`builder.go:84-93`).

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd libs/atlas-script-core && go test ./... -v`
Expected: PASS.

- [ ] **Step 6: Confirm no consumer broke**

Every consumer of this package builds conditions through `NewBuilder()`, never a
struct literal, so the field addition is additive. The changed `Build()` guard is
strictly more permissive. Confirm by building each consumer module:

Run:
```bash
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && cd -
cd services/atlas-portal-actions/atlas.com/portal-actions && go build ./... && cd -
cd services/atlas-reactor-actions/atlas.com/reactor-actions && go build ./... && cd -
cd services/atlas-party-quests/atlas.com/party-quests && go build ./... && cd -
```
Expected: all four exit 0 with no output.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-script-core/condition/
git commit -m "feat(script-core): add values to the condition model for the in operator"
```

---

## Task A2: map-actions stops truncating the condition contract

Per design F1, a map-action condition loses fields at two boundaries. The seed-JSON
decoder (`script/entity.go:47-53`, `entity.go:107-116`) and the REST model
(`script/rest.go:34-39`, `rest.go:113-131`, `rest.go:189-198`) both carry only
`type`/`operator`/`value`/`referenceId` — so a seed cannot express `step` at all,
and the aggregator **rejects** a `questProgress` condition that has no step. The
local `validation.ConditionInput` (`validation/model.go:9-18`) already carries
`Step`/`WorldId`/`ChannelId`/`IncludeEquipped` but lacks `Values`.

This task widens the two document representations and the aggregator input struct.
Populating them is Task A3.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/rest.go` — `RestConditionModel` gains 5 fields; `transformRule` and `extractCondition` carry them
- `services/atlas-map-actions/atlas.com/map-actions/script/entity.go` — `jsonCondition` gains the same 5 fields; `convertJsonCondition` carries them
- `services/atlas-map-actions/atlas.com/map-actions/validation/model.go` — `ConditionInput` gains `Values []int`
- `services/atlas-map-actions/atlas.com/map-actions/script/rest_test.go` — **new file**
- `libs/atlas-script-core/condition/model.go` — read-only; the getters being read
- `libs/atlas-saga/validation.go` — read-only; `ValidationConditionInput` is the shape `ConditionInput` must become a strict subset of

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `script/rest.go:189-198` (`extractCondition`) is the exact function
being extended; `script/entity.go:107-116` (`convertJsonCondition`) is its seed-JSON
twin and must be kept in lockstep — the two are parallel decoders of the same
document and a field added to one without the other is the F1 defect recurring.

**Interfaces:**
- Consumes: `condition.Model.Values()`, `condition.Builder.SetValues` (Task A1).
- Produces: `script.RestConditionModel` with JSON fields `type`, `operator`, `value`, `values`, `referenceId`, `step`, `worldId`, `channelId`, `includeEquipped`; `validation.ConditionInput` with `Values []int \`json:"values,omitempty"\``.

- [ ] **Step 1: Write the failing test**

`services/atlas-map-actions/atlas.com/map-actions/script/rest_test.go` — new file,
package `script`. No test file exists in this package other than `groups_test.go`;
copy its package declaration and import style from
`services/atlas-map-actions/atlas.com/map-actions/script/groups_test.go:1-12`.

Three test functions:

**`TestExtractConditionCarriesEveryField`** — build a `RestConditionModel` with every
field populated and assert each survives into the `condition.Model`:

| input field | input value | expected getter | expected value |
|---|---|---|---|
| `Type` | `"questProgress"` | `Type()` | `"questProgress"` |
| `Operator` | `"="` | `Operator()` | `"="` |
| `Value` | `"0"` | `Value()` | `"0"` |
| `Values` | `[]string{"1000","1100"}` | `Values()` | `[]string{"1000","1100"}` |
| `ReferenceId` | `"21747"` | `ReferenceIdRaw()` | `"21747"` |
| `Step` | `"9300351"` | `Step()` | `"9300351"` |
| `WorldId` | `"0"` | `WorldId()` | `"0"` |
| `ChannelId` | `"1"` | `ChannelId()` | `"1"` |
| `IncludeEquipped` | `true` | `IncludeEquipped()` | `true` |

**`TestTransformRuleRoundTripsEveryConditionField`** — build a `condition.Model`
through `condition.NewBuilder()` with the same nine values, wrap it in a `Rule` via
`NewRuleBuilder().SetId("r1").AddCondition(c)`, call `transformRule`, and assert the
resulting `RestConditionModel` equals the input `RestConditionModel` from the first
test field-for-field.

**`TestConvertJsonConditionCarriesEveryField`** — unmarshal this exact JSON into
`jsonCondition` and assert the same nine getters as the first test:

```json
{"type":"questProgress","operator":"=","value":"0","values":["1000","1100"],"referenceId":"21747","step":"9300351","worldId":"0","channelId":"1","includeEquipped":true}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run 'TestExtractCondition|TestTransformRule|TestConvertJsonCondition' -v`
Expected: compile failure — `RestConditionModel` has no field `Values`/`Step`/`WorldId`/`ChannelId`/`IncludeEquipped`.

- [ ] **Step 3: Widen `RestConditionModel`**

Replace `script/rest.go:33-39`:

```go
// RestConditionModel represents a condition in REST format
type RestConditionModel struct {
	Type            string   `json:"type"`
	Operator        string   `json:"operator"`
	Value           string   `json:"value,omitempty"`
	Values          []string `json:"values,omitempty"`
	ReferenceId     string   `json:"referenceId,omitempty"`
	Step            string   `json:"step,omitempty"`
	WorldId         string   `json:"worldId,omitempty"`
	ChannelId       string   `json:"channelId,omitempty"`
	IncludeEquipped bool     `json:"includeEquipped,omitempty"`
}
```

Note `Value` becomes `omitempty`: an `in` condition carries `values` and no `value`.

- [ ] **Step 4: Carry the fields through `transformRule` and `extractCondition`**

In `transformRule` (`script/rest.go:113-131`), replace the `RestConditionModel`
literal with:

```go
		conditions = append(conditions, RestConditionModel{
			Type:            cond.Type(),
			Operator:        cond.Operator(),
			Value:           cond.Value(),
			Values:          cond.Values(),
			ReferenceId:     cond.ReferenceIdRaw(),
			Step:            cond.Step(),
			WorldId:         cond.WorldId(),
			ChannelId:       cond.ChannelId(),
			IncludeEquipped: cond.IncludeEquipped(),
		})
```

In `extractCondition` (`script/rest.go:189-198`), replace the body with:

```go
func extractCondition(r RestConditionModel) (condition.Model, error) {
	builder := condition.NewBuilder().
		SetType(r.Type).
		SetOperator(r.Operator).
		SetValue(r.Value).
		SetIncludeEquipped(r.IncludeEquipped)

	if len(r.Values) > 0 {
		builder.SetValues(r.Values)
	}
	if r.ReferenceId != "" {
		builder.SetReferenceId(r.ReferenceId)
	}
	if r.Step != "" {
		builder.SetStep(r.Step)
	}
	if r.WorldId != "" {
		builder.SetWorldId(r.WorldId)
	}
	if r.ChannelId != "" {
		builder.SetChannelId(r.ChannelId)
	}

	return builder.Build()
}
```

- [ ] **Step 5: Mirror both changes in `entity.go`**

Replace `script/entity.go:46-53` with the identical field set (same JSON tags, same
`omitempty` on `value`):

```go
// jsonCondition represents a condition in JSON format
type jsonCondition struct {
	Type            string   `json:"type"`
	Operator        string   `json:"operator"`
	Value           string   `json:"value,omitempty"`
	Values          []string `json:"values,omitempty"`
	ReferenceId     string   `json:"referenceId,omitempty"`
	Step            string   `json:"step,omitempty"`
	WorldId         string   `json:"worldId,omitempty"`
	ChannelId       string   `json:"channelId,omitempty"`
	IncludeEquipped bool     `json:"includeEquipped,omitempty"`
}
```

and replace `convertJsonCondition` (`script/entity.go:106-116`) with the same builder
body as `extractCondition` above, reading from `jc` instead of `r`.

- [ ] **Step 6: Add `Values` to the aggregator input struct**

In `services/atlas-map-actions/atlas.com/map-actions/validation/model.go`, add to
`ConditionInput` immediately after `Value` (matching `saga.ValidationConditionInput`'s
field order at `libs/atlas-saga/validation.go:68-69`):

```go
	Values          []int      `json:"values,omitempty"`
```

- [ ] **Step 7: Run the tests and confirm they pass**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-map-actions/atlas.com/map-actions/script/rest.go \
        services/atlas-map-actions/atlas.com/map-actions/script/entity.go \
        services/atlas-map-actions/atlas.com/map-actions/script/rest_test.go \
        services/atlas-map-actions/atlas.com/map-actions/validation/model.go
git commit -m "fix(map-actions): stop truncating step/values/world/channel from conditions"
```

---

## Task A3: the evaluator populates every field, and `map_id` gains range operators

Two defects in one file. `evaluateViaQueryAggregator`
(`script/evaluator.go:58-90`) builds `validation.ConditionInput` from only four of
the nine fields (`evaluator.go:75-80`), so the fields Task A2 just plumbed still die
here — this is design F1's second half. And `evaluateMapId`
(`script/evaluator.go:40-56`) accepts only `=`, `==`, `!=`, which blocks G14a's
eight map-ID range branches in `explorationPoint`.

`evaluator.go:60-63` also unconditionally `strconv.Atoi`s the scalar value, which
makes an `in` condition — whose `value` is empty and whose payload lives in `values`
— a hard error before it ever reaches the aggregator.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/evaluator.go` — both functions
- `services/atlas-map-actions/atlas.com/map-actions/script/evaluator_test.go` — **new file**
- `services/atlas-map-actions/atlas.com/map-actions/validation/mock/processor.go` — read-only; the `ProcessorMock{ValidateCharacterStateFunc}` used to capture the built input
- `libs/atlas-constants/field/model.go` — read-only; `WorldId()`, `ChannelId()`, `MapId()` are the sources for the world/channel fields

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `services/atlas-map-actions/atlas.com/map-actions/script/groups_test.go:1-12` for package/import shape. `ConditionEvaluator`'s `validationP` field (`evaluator.go:18`) is unexported and set in the constructor (`evaluator.go:25`), so tests in package `script` assign it directly after `NewConditionEvaluator` — no new constructor.

**Interfaces:**
- Consumes: `validation.ConditionInput.Values` (Task A2); `condition.Model.Values()` (Task A1).
- Produces: no new exported symbols. `evaluateViaQueryAggregator` gains the `f field.Model` parameter: `func (e *ConditionEvaluator) evaluateViaQueryAggregator(f field.Model, characterId uint32, cond condition.Model) (bool, error)`. Its only caller is `EvaluateCondition` (`evaluator.go:36`), which already has `f`.

- [ ] **Step 1: Write the failing test**

`services/atlas-map-actions/atlas.com/map-actions/script/evaluator_test.go` — new
file, package `script`.

**`TestEvaluateMapIdOperators`** — table-driven. Build the field with
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()` and the
condition with `condition.NewBuilder().SetType("map_id").SetOperator(op).SetValue(v).Build()`.
No aggregator is involved (`map_id` is evaluated locally), so `validationP` may stay
as the constructor left it.

| subtest name | operator | value | expect result | expect error |
|---|---|---|---|---|
| `eq match` | `=` | `100000000` | `true` | nil |
| `eq no match` | `=` | `100000001` | `false` | nil |
| `eq eq match` | `==` | `100000000` | `true` | nil |
| `ne match` | `!=` | `100000001` | `true` | nil |
| `ne no match` | `!=` | `100000000` | `false` | nil |
| `gt true` | `>` | `99999999` | `true` | nil |
| `gt false equal` | `>` | `100000000` | `false` | nil |
| `gt false above` | `>` | `100000001` | `false` | nil |
| `lt true` | `<` | `100000001` | `true` | nil |
| `lt false equal` | `<` | `100000000` | `false` | nil |
| `gte true equal` | `>=` | `100000000` | `true` | nil |
| `gte true below` | `>=` | `99999999` | `true` | nil |
| `gte false` | `>=` | `100000001` | `false` | nil |
| `lte true equal` | `<=` | `100000000` | `true` | nil |
| `lte true above` | `<=` | `100000001` | `true` | nil |
| `lte false` | `<=` | `99999999` | `false` | nil |
| `unsupported operator` | `~=` | `100000000` | `false` | error containing `unsupported operator [~=] for map_id condition` |

**`TestEvaluateViaQueryAggregatorCarriesEveryField`** — the F1 regression test.
Install a `validation/mock.ProcessorMock` whose `ValidateCharacterStateFunc` captures
the `[]validation.ConditionInput` argument and returns
`validation.NewValidationResult(characterId, true), nil`. Field:
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()`.
Condition: `condition.NewBuilder().SetType("questProgress").SetOperator("=").SetValue("0").SetReferenceId("21747").SetStep("9300351").SetIncludeEquipped(true).Build()`.

Assert the captured slice has exactly one element, equal field-for-field to:

```go
validation.ConditionInput{
	Type:            "questProgress",
	Operator:        "=",
	Value:           0,
	Values:          nil,
	ReferenceId:     21747,
	Step:            "9300351",
	WorldId:         world.Id(0),
	ChannelId:       channel.Id(1),
	IncludeEquipped: true,
}
```

**`TestEvaluateViaQueryAggregatorInOperatorUsesValues`** — condition
`condition.NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000","1100","1110","1200","1210","1300","1310","1400","1410","1500","1510"}).Build()`.
Assert the captured input has `Value: 0`, `Values` deep-equal to the same eleven ints
in the same order, and that the call returns no error — today the empty scalar
`value` makes this a hard `strconv.Atoi` failure.

**`TestEvaluateViaQueryAggregatorRejectsNonIntegerScalarValue`** — condition
`condition.NewBuilder().SetType("level").SetOperator(">=").SetValue("ten").Build()`.
Assert the error message contains `invalid condition value [ten]` and that the
aggregator was never called.

**`TestEvaluateViaQueryAggregatorRejectsNonIntegerValuesEntry`** — condition
`SetType("jobId").SetOperator("in").SetValues([]string{"1000","abc"})`. Assert the
error message contains `invalid condition values entry [abc]`.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run TestEvaluate -v`
Expected: FAIL. `TestEvaluateMapIdOperators` fails the six range subtests with
`unsupported operator`; `TestEvaluateViaQueryAggregatorCarriesEveryField` fails on
`Step`/`WorldId`/`ChannelId`/`IncludeEquipped` being zero.

- [ ] **Step 3: Add the range operators to `evaluateMapId`**

Replace the switch at `script/evaluator.go:48-55`:

```go
	switch cond.Operator() {
	case "=", "==":
		return actualMapId == expectedMapId, nil
	case "!=":
		return actualMapId != expectedMapId, nil
	case ">":
		return actualMapId > expectedMapId, nil
	case "<":
		return actualMapId < expectedMapId, nil
	case ">=":
		return actualMapId >= expectedMapId, nil
	case "<=":
		return actualMapId <= expectedMapId, nil
	default:
		return false, fmt.Errorf("unsupported operator [%s] for map_id condition", cond.Operator())
	}
```

- [ ] **Step 4: Populate every field in `evaluateViaQueryAggregator`**

Change the signature at `script/evaluator.go:58` to take `f field.Model` as the first
parameter, and update its single call site at `script/evaluator.go:36` to
`return e.evaluateViaQueryAggregator(f, characterId, cond)`.

Replace the value parsing at `script/evaluator.go:59-63` and the struct literal at
`script/evaluator.go:75-80` with:

```go
	// Parse the scalar value only when one is present. An `in` condition
	// carries its payload in values and leaves value empty; eagerly Atoi-ing
	// it here made `in` unreachable from a map-action document.
	var intValue int
	if valueStr := cond.Value(); valueStr != "" {
		parsed, err := strconv.Atoi(valueStr)
		if err != nil {
			return false, fmt.Errorf("invalid condition value [%s]: %w", valueStr, err)
		}
		intValue = parsed
	}

	var intValues []int
	for _, v := range cond.Values() {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return false, fmt.Errorf("invalid condition values entry [%s]: %w", v, err)
		}
		intValues = append(intValues, parsed)
	}
```

and

```go
	validationCondition := validation.ConditionInput{
		Type:            cond.Type(),
		Operator:        cond.Operator(),
		Value:           intValue,
		Values:          intValues,
		ReferenceId:     resolvedReferenceId,
		Step:            cond.Step(),
		WorldId:         f.WorldId(),
		ChannelId:       f.ChannelId(),
		IncludeEquipped: cond.IncludeEquipped(),
	}
```

`WorldId`/`ChannelId` come from the field model, not from the condition document:
`mapCapacity` needs the world and channel the character is actually in, and a seed
has no way to know them. The `worldId`/`channelId` document fields plumbed in Task A2
remain available for a future context-reference use; they are deliberately not read
here, and that is stated in a comment above the literal:

```go
	// worldId/channelId come from the field the entry event carried, never from
	// the document — a seed is version-scoped, not world-scoped.
```

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-map-actions/atlas.com/map-actions/script/evaluator.go \
        services/atlas-map-actions/atlas.com/map-actions/script/evaluator_test.go
git commit -m "fix(map-actions): forward every condition field and add map_id range operators"
```

---

## Task A4: an unknown operation fails loudly

`ExecuteOperation`'s `default:` arm (`script/executor.go:49-51`) logs a warning and
returns `nil`, so a typo'd or not-yet-implemented operation is a permanent silent
no-op — design D3, PRD FR-3.0. It becomes an error.

`ExecuteOperations` (`script/executor.go:55-62`) aborts the rule's remaining
operations on the first error, so an unknown operation mid-rule now suppresses the
operations after it. That is intended — a partially-applied cutscene is worse than a
loud failure — and must be stated in the doc comment rather than left for a reader to
discover.

This change is only safe because Task A8 generates the schema's operation enum from
this same switch, so a seed cannot be authored with an operation the executor lacks.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — the `default:` arm and the `ExecuteOperations` doc comment
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — **new file**

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `services/atlas-map-actions/atlas.com/map-actions/script/groups_test.go:1-12` for package/import shape. `OperationExecutor.sagaP` (`executor.go:22`) is unexported and set in `NewOperationExecutor` (`executor.go:29`); tests in package `script` assign it directly. Use the saga processor mock at `services/atlas-map-actions/atlas.com/map-actions/saga/mock/processor.go` if one exists; if it does not, define a local `type recordingSagaProcessor struct{ created []saga.Saga }` in the test file implementing `mapactionsaga.Processor` — the test asserts on captured sagas, so a hand-rolled recorder in `_test.go` is correct here and is not a `*_testhelpers.go` production shim.

- [ ] **Step 1: Write the failing test**

`services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — new
file, package `script`.

**`TestExecuteOperationUnknownTypeErrors`** — operation built with
`operation.NewBuilder().SetType("play_sound").Build()`, field
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()`,
characterId `1`. Assert:
- the returned error is non-nil
- its message is exactly `unknown operation type [play_sound]`
- the saga recorder captured zero sagas

**`TestExecuteOperationsAbortsAfterUnknownOperation`** — three operations in order:
`field_effect` with `params{"path":"maplemap/enter/1000000"}`, then `play_sound`
(unknown), then `unlock_ui`. Assert:
- `ExecuteOperations` returns the same `unknown operation type [play_sound]` error
- the recorder captured exactly **one** saga (the `field_effect`), proving the
  third operation did not run

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run TestExecuteOperation -v`
Expected: FAIL — `ExecuteOperation` returns nil for the unknown type, and the second
test sees two captured sagas.

- [ ] **Step 3: Make the default arm error**

Replace `script/executor.go:49-51`:

```go
	default:
		// FR-3.0 / design D3: an unknown operation is a seed defect, not a
		// no-op. The schema's operation enum is generated from this switch
		// (tools/gen-map-action-schema.sh), so a document cannot name an
		// operation this switch lacks without failing catalog-lint first.
		return fmt.Errorf("unknown operation type [%s]", op.Type())
	}
```

Remove the now-unused `characterId` reference in the deleted `l.Warnf` only if it
leaves the parameter unused — it does not; `ExecuteOperation` logs it at
`executor.go:34`.

- [ ] **Step 4: Document the abort semantics**

Add above `ExecuteOperations` (`script/executor.go:55`):

```go
// ExecuteOperations runs a rule's operations in document order and stops at the
// first error. An unknown operation therefore suppresses every operation after
// it in the same rule (design D3). That is deliberate: a half-applied cutscene
// or a spawn without its announcement is worse than a loud failure at map entry.
```

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 6: Confirm no seeded document names an unknown operation**

Every operation in the seeded 99 documents must already be in the switch, or this
change breaks production seeds on the next map entry.

Run:
```bash
grep -rho '"type": *"[a-z_]*"' deploy/seed/*/*/map-actions/ | sort -u
```
Expected: only `"type": "map-action"` (the envelope), plus operation types drawn from
`field_effect`, `show_intro`, `spawn_monster`, `drop_message`, `lock_ui`, `unlock_ui`
and condition types `map_id`, `gender`. If any other operation type appears, STOP and
report it — it is a live silent no-op and needs its own decision.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-map-actions/atlas.com/map-actions/script/executor.go \
        services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go
git commit -m "fix(map-actions): fail loudly on an unknown operation type"
```

---

## Task A5: `spawn_monster` gains the field instance, `spawnIfAbsent` and `monsterIds`

Three changes to `executeSpawnMonster` (`script/executor.go:120-190`):

1. **Design F3 — the instance defect.** `executor.go:183` hard-codes
   `Instance: uuid.Nil`, discarding `f.Instance()`, which the field model exposes
   (`libs/atlas-constants/field/model.go:41`) and which the orchestrator threads
   straight into the atlas-monsters URL
   (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/requests.go:12,24`).
   On any instanced map the spawn lands in the non-instanced field. This is invisible
   today because all nine seeded scripts run on non-instanced maps; it goes live the
   moment Plan C lands the G5 party-quest set (`922000000`, `926000000`, `926000010`,
   `926120300`), which are instanced.
2. **PRD FR-2.1 — `spawnIfAbsent`.** A string-encoded boolean param, forwarded on the
   payload. The decision itself is made in atlas-monsters (Task A7).
3. **Design D9 — `monsterIds`.** A comma-separated list, mutually exclusive with
   `monsterId`; the executor picks one uniformly. This is Plan C's G7
   (`pepeking_effect` picks one of 3300005/3300006/3300007), landed here because it is
   a change to the same function.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — `executeSpawnMonster` only
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — extend the file created in Task A4
- `libs/atlas-saga/payloads.go` — read-only in this task; `SpawnMonsterPayload.SpawnIfAbsent` is added in Task A6
- `libs/atlas-constants/field/model.go` — read-only; `Instance()` at line 41

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: the existing optional-param blocks in the same function —
`executor.go:135-143` (`x`), `executor.go:145-153` (`y`), `executor.go:155-163`
(`count`) — for the `params[...]`-present / parse / error shape. Use
`math/rand`'s `rand.Intn` for the uniform pick — that is the repo's convention
(`services/atlas-channel/atlas.com/channel/session/model.go:4`,
`services/atlas-login/atlas.com/login/channel/processor.go:6` and six other call
sites all import `math/rand`, and nothing in the tree imports `math/rand/v2`).

**Interfaces:**
- Consumes: `saga.SpawnMonsterPayload.SpawnIfAbsent bool` (Task A6). Task A6 must land before this task compiles; if executing out of order, do A6 first.
- Produces: no new exported symbols. Two new document params on `spawn_monster`: `spawnIfAbsent` (`"true"`/`"false"`) and `monsterIds` (comma-separated decimal ids).

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go`.
All four tests use a field built with an explicit instance:

```go
inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()
```

and assert against the single captured saga's step payload, type-asserted to
`saga.SpawnMonsterPayload`.

**`TestExecuteSpawnMonsterCarriesFieldInstance`** — params
`{"monsterId":"9100013","x":"82","y":"200"}`. Assert `payload.Instance == inst`
(today it is `uuid.Nil` — this is the F3 regression test), and
`payload.MapId == _map.Id(926000000)`, `payload.MonsterId == 9100013`,
`payload.X == 82`, `payload.Y == 200`, `payload.Count == 1`.

**`TestExecuteSpawnMonsterSpawnIfAbsent`** — table-driven:

| subtest | `spawnIfAbsent` param | expect `payload.SpawnIfAbsent` | expect error |
|---|---|---|---|
| `absent` | (param not present) | `false` | nil |
| `true` | `"true"` | `true` | nil |
| `false` | `"false"` | `false` | nil |
| `invalid` | `"yes"` | — | error containing `invalid spawnIfAbsent [yes]` |

**`TestExecuteSpawnMonsterMonsterIdsPicksFromSet`** — params
`{"monsterIds":"3300005,3300006,3300007","x":"-28","y":"-67"}`. Run
`executeSpawnMonster` 200 times against a fresh recorder and assert:
- every captured `payload.MonsterId` is one of `3300005`, `3300006`, `3300007`
- all three appear at least once across the 200 runs
- `payload.X == -28`, `payload.Y == -67`

**`TestExecuteSpawnMonsterIdParamValidation`** — table-driven:

| subtest | params | expect error message contains |
|---|---|---|
| `neither` | `{"x":"0"}` | `spawn_monster operation requires exactly one of monsterId or monsterIds` |
| `both` | `{"monsterId":"1","monsterIds":"2,3"}` | `spawn_monster operation requires exactly one of monsterId or monsterIds` |
| `empty monsterIds` | `{"monsterIds":""}` | `spawn_monster operation requires exactly one of monsterId or monsterIds` |
| `non-numeric entry` | `{"monsterIds":"3300005,abc"}` | `invalid monsterIds entry [abc]` |
| `non-numeric monsterId` | `{"monsterId":"abc"}` | `invalid monsterId [abc]` |

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run TestExecuteSpawnMonster -v`
Expected: FAIL — instance is `uuid.Nil`, `SpawnIfAbsent` does not compile until Task A6,
`monsterIds` is ignored and the missing-`monsterId` case reports the old
`spawn_monster operation missing monsterId parameter`.

- [ ] **Step 3: Replace the monster-id resolution**

Replace `script/executor.go:122-128` (the `monsterId`-only block) with:

```go
	monsterIdStr, hasSingle := params["monsterId"]
	monsterIdsStr, hasList := params["monsterIds"]
	if hasList && strings.TrimSpace(monsterIdsStr) == "" {
		hasList = false
	}
	if hasSingle == hasList {
		return fmt.Errorf("spawn_monster operation requires exactly one of monsterId or monsterIds")
	}

	var monsterId uint64
	if hasSingle {
		parsed, err := strconv.ParseUint(monsterIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid monsterId [%s]: %w", monsterIdStr, err)
		}
		monsterId = parsed
	} else {
		// Design D9 (G7): pepeking_effect picks one of three monsters
		// uniformly. Randomizing here keeps the rule engine stateless — a
		// `random` rule selector would need a non-deterministic condition
		// type the aggregator has no concept of.
		candidates := strings.Split(monsterIdsStr, ",")
		ids := make([]uint64, 0, len(candidates))
		for _, c := range candidates {
			c = strings.TrimSpace(c)
			parsed, err := strconv.ParseUint(c, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid monsterIds entry [%s]: %w", c, err)
			}
			ids = append(ids, parsed)
		}
		monsterId = ids[rand.Intn(len(ids))]
	}
```

Add `"strings"` and `"math/rand"` to the import block (`executor.go:3-17`).

- [ ] **Step 4: Parse `spawnIfAbsent`**

Insert after the `count` block (`script/executor.go:155-163`):

```go
	// FR-2.1: Cosmic guards every map spawn with getMonsterById(id) != null.
	// The guard itself is decided in atlas-monsters against its own registry
	// (design D5/F6) — a read-then-write here would be a cross-service TOCTOU
	// two simultaneous map entries would both pass.
	var spawnIfAbsent bool
	if s, has := params["spawnIfAbsent"]; has {
		parsed, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("invalid spawnIfAbsent [%s]: %w", s, err)
		}
		spawnIfAbsent = parsed
	}
```

- [ ] **Step 5: Fix the payload**

In the `saga.SpawnMonsterPayload` literal (`script/executor.go:176-187`), replace
`Instance: uuid.Nil,` with `Instance: f.Instance(),` and add
`SpawnIfAbsent: spawnIfAbsent,` after `Count`. Remove the `"github.com/google/uuid"`
import (`executor.go:10`) if nothing else in the file uses it — check with
`grep -n 'uuid\.' services/atlas-map-actions/atlas.com/map-actions/script/executor.go`
before deleting.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-map-actions/atlas.com/map-actions/script/executor.go \
        services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go
git commit -m "fix(map-actions): spawn into the caller's field instance; add spawnIfAbsent and monsterIds"
```

---

## Task A6: `SpawnIfAbsent` crosses the saga boundary

The flag must ride `saga.SpawnMonsterPayload` → the orchestrator's
`monster.Processor.SpawnMonster` → the `POST .../monsters` body that atlas-monsters
already serves. Four files change, none of them behaviourally — this task only
carries the flag; Task A7 acts on it.

`SpawnMonsterPayload` (`libs/atlas-saga/payloads.go:512-523`) has no such field.
`monster.Processor.SpawnMonster`
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/processor.go:14`)
takes `(f field.Model, monsterId uint32, x, y, fh int16, team int8)` — six positional
params already, so a seventh positional `bool` is the wrong shape. Replace the
parameter list with the existing `SpawnRequest` struct
(`monster/rest.go:63-72`), which the implementation already builds internally at
`monster/processor.go:32-41`.

### Files

- `libs/atlas-saga/payloads.go` — `SpawnMonsterPayload` gains `SpawnIfAbsent`
- `libs/atlas-saga/payloads_test.go` — extend with a round-trip case
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/processor.go` — `SpawnMonster` takes a `SpawnRequest`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/rest.go` — `SpawnRequest` and `SpawnInputRestModel` gain `SpawnIfAbsent`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — `handleSpawnMonster` (around line 2067) builds the `SpawnRequest`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/mock/processor.go` — the mock's signature, if this file exists (`ls services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/` to check)

Module roots: `libs/atlas-saga` and
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` — this task builds and
tests **both**.

Patterns to copy: `libs/atlas-saga/payloads_test.go` for the payload round-trip test
shape (open it and copy the nearest `SpawnMonsterPayload` or sibling-payload case;
if none exists, copy the shape of any `TestUnmarshal*Payload` in that file).

**Interfaces:**
- Produces:
  - `saga.SpawnMonsterPayload.SpawnIfAbsent bool \`json:"spawnIfAbsent,omitempty"\``
  - `monster.SpawnRequest` gains `Fh int16` (already present) and `SpawnIfAbsent bool`
  - `monster.SpawnInputRestModel` gains `SpawnIfAbsent bool \`json:"spawnIfAbsent,omitempty"\``
  - `monster.Processor.SpawnMonster(f field.Model, req SpawnRequest) error` — signature change from `(f field.Model, monsterId uint32, x, y, fh int16, team int8) error`
- Consumed by: Task A5 (the payload field) and Task A7 (the REST field).

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-saga/payloads_test.go`:

**`TestSpawnMonsterPayloadRoundTripsSpawnIfAbsent`** — marshal a
`SpawnMonsterPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, MapId: 926000000, Instance: uuid.MustParse("11111111-2222-3333-4444-555555555555"), MonsterId: 9100013, X: 82, Y: 200, Count: 1, SpawnIfAbsent: true}`
to JSON, assert the JSON contains `"spawnIfAbsent":true`, unmarshal it back and
assert `SpawnIfAbsent == true`.

**`TestSpawnMonsterPayloadOmitsSpawnIfAbsentWhenFalse`** — the same payload with
`SpawnIfAbsent: false`; assert the marshalled JSON does **not** contain
`spawnIfAbsent`.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd libs/atlas-saga && go test ./... -run TestSpawnMonsterPayload -v`
Expected: compile failure — `unknown field SpawnIfAbsent`.

- [ ] **Step 3: Add the payload field**

In `libs/atlas-saga/payloads.go`, add to `SpawnMonsterPayload` after `Count`
(line 522):

```go
	SpawnIfAbsent bool `json:"spawnIfAbsent,omitempty"` // FR-2.1: suppress the spawn when a monster of this template is already on the field. The decision is made by atlas-monsters against its own registry, not here.
```

- [ ] **Step 4: Run the payload tests**

Run: `cd libs/atlas-saga && go test ./... -v`
Expected: PASS.

- [ ] **Step 5: Widen the orchestrator's REST models**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/rest.go`,
add `SpawnIfAbsent bool \`json:"spawnIfAbsent,omitempty"\`` to `SpawnInputRestModel`
(after `Team`, line 18) and `SpawnIfAbsent bool` to `SpawnRequest` (after `Team`,
line 71), and carry it in `ToRestModel()` (line 74-83):

```go
		SpawnIfAbsent: r.SpawnIfAbsent,
```

- [ ] **Step 6: Change the processor signature**

In `monster/processor.go`, replace the interface method (line 13-14) and the
implementation (line 31-41) so both take the request struct:

```go
	// SpawnMonster spawns a monster at the location described by req.
	SpawnMonster(f field.Model, req SpawnRequest) error
```

```go
func (p *ProcessorImpl) SpawnMonster(f field.Model, req SpawnRequest) error {
	_, err := requestSpawnMonster(p.ctx, f, req.ToRestModel())(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to spawn monster %d at (%d, %d) in map %d", req.MonsterId, req.X, req.Y, f.MapId())
		return err
	}

	p.l.Debugf("Successfully spawned monster %d at (%d, %d, fh=%d) in world %d, channel %d, map %d",
		req.MonsterId, req.X, req.Y, req.Fh, f.WorldId(), f.ChannelId(), f.MapId())
	return nil
}
```

Note the `WorldId`/`ChannelId`/`MapId` fields on `SpawnRequest` are now redundant with
`f` — leave them on the struct (other callers may set them) but do not read them here;
`requestSpawnMonster` already derives the URL from `f` (`monster/requests.go:24`).

- [ ] **Step 7: Update the handler call site**

In `saga/handler.go`'s `handleSpawnMonster`, replace the spawn loop body:

```go
	req := monster.SpawnRequest{
		WorldId:       payload.WorldId,
		ChannelId:     payload.ChannelId,
		MapId:         payload.MapId,
		MonsterId:     payload.MonsterId,
		X:             payload.X,
		Y:             payload.Y,
		Fh:            int16(fh),
		Team:          payload.Team,
		SpawnIfAbsent: payload.SpawnIfAbsent,
	}
	for i := 0; i < count; i++ {
		if err := h.monsterP.SpawnMonster(f, req); err != nil {
			h.logActionError(s, st, err, fmt.Sprintf("Failed to spawn monster %d/%d", i+1, count))
			return err
		}
	}
```

Add the `monster` package import to `handler.go` if it is not already present
(`grep -n 'saga-orchestrator/monster"' services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`).

- [ ] **Step 8: Update the mock, if one exists**

Run: `ls services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/`
If a `mock/` directory or `*_mock.go` exists, update the mocked `SpawnMonster`
signature to `func(f field.Model, req SpawnRequest) error` and fix every test that
sets it.

- [ ] **Step 9: Build and test both modules**

Run:
```bash
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
```
Expected: both exit 0.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-saga/payloads.go libs/atlas-saga/payloads_test.go \
        services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/ \
        services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go
git commit -m "feat(saga): carry spawnIfAbsent from the spawn_monster step to atlas-monsters"
```

---

## Task A7: atlas-monsters honors `spawnIfAbsent`

This is the cross-service seam. Design D5/F6: the guard belongs where the registry
is, because an orchestrator-side `GET`-then-`POST` is a TOCTOU two simultaneous map
entries would both pass. `monster.ProcessorImpl.Create`
(`services/atlas-monsters/atlas.com/monsters/monster/processor.go:259`)
unconditionally calls `GetMonsterRegistry().CreateMonster(...)` at line 267; it gains
a pre-check against `GetInField` (`monster/processor.go:208-210`, backed by
`Registry.GetMonstersInMap`).

Per the project's cross-service seam rule, the acceptance test for the NEW contract
lives here, in the consumer — a green build in atlas-map-actions or the orchestrator
alone does not cover it.

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/rest.go` — `RestModel` gains `SpawnIfAbsent`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — `Create` short-circuits
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` — extend
- `services/atlas-monsters/atlas.com/monsters/world/resource.go` — read-only; `handleCreateMonsterInMap` (around line 207) already decodes `RestModel` and passes it to `Create` unchanged, so no change is needed there — confirm this by reading it before assuming

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`
is large (96 KB); find its existing `TestCreate`-family cases and copy the nearest
one's setup verbatim — it establishes the tenant context, the Redis-backed registry,
and the `monsterInformation` seam that `Create` calls at `processor.go:261`. Do NOT
invent a new harness. Slice the file to locate the case rather than reading it whole:
`grep -n '^func Test' services/atlas-monsters/atlas.com/monsters/monster/processor_test.go | grep -i create`.

**Interfaces:**
- Consumes: `SpawnInputRestModel.SpawnIfAbsent` from Task A6 — the JSON field name `spawnIfAbsent` must match exactly on both sides.
- Produces: `monster.RestModel.SpawnIfAbsent bool \`json:"spawnIfAbsent,omitempty"\``; `Create` returns `(Model{}, nil)` — a zero Model and a **nil** error — when the guard suppresses the spawn.

- [ ] **Step 1: Decide and record the suppressed-spawn return**

`Create` returns `(Model, error)`. A suppressed spawn is not an error, so it returns
`(Model{}, nil)`. The caller `handleCreateMonsterInMap`
(`world/resource.go:207-231`) transforms the returned `Model` into a REST response.
Read that handler and confirm what it does with a zero `Model` — if it would emit a
misleading `200` with a zero-valued monster body, the handler must instead return
`204 No Content`. Record the decision in the commit message. Do not guess: read
`world/resource.go:207-231` first.

- [ ] **Step 2: Write the failing test**

Append to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`,
reusing the setup from the existing create-family test located in Step 1's grep.

**`TestCreateSpawnIfAbsentSuppressesWhenPresent`**
1. `Create` a monster with `RestModel{MonsterId: 9100013, X: 82, Y: 200, Fh: 0, Team: 0}` on field `f` — assert it succeeds and `GetInField(f)` returns exactly 1 monster.
2. `Create` again with the same template and `SpawnIfAbsent: true` — assert the returned error is nil, the returned `Model` has `UniqueId() == 0`, and `GetInField(f)` still returns exactly 1 monster.

**`TestCreateSpawnIfAbsentCreatesWhenAbsent`** — on an empty field, `Create` with
`SpawnIfAbsent: true` and `MonsterId: 9100013`. Assert no error, the returned `Model`
has a non-zero `UniqueId()`, and `GetInField(f)` returns exactly 1 monster.

**`TestCreateSpawnIfAbsentMatchesOnTemplateNotUniqueId`** — spawn `9100013`, then
`Create` `9300156` with `SpawnIfAbsent: true`. Assert the second spawn **does**
happen (different template) and `GetInField(f)` returns 2.

**`TestCreateWithoutSpawnIfAbsentStacks`** — the existing behavior must not change.
`Create` `9100013` twice with `SpawnIfAbsent` absent (zero value). Assert
`GetInField(f)` returns 2.

**`TestCreateSpawnIfAbsentIsFieldScoped`** — spawn `9100013` on field
`f1 = field.NewBuilder(0, 1, 926000000).SetInstance(instA).Build()`, then `Create`
`9100013` with `SpawnIfAbsent: true` on
`f2 = field.NewBuilder(0, 1, 926000000).SetInstance(instB).Build()`. Assert the
second spawn **does** happen — the guard is per field instance, which is precisely
why Task A5's F3 fix matters.

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run TestCreateSpawnIfAbsent -v`
Expected: compile failure — `RestModel` has no field `SpawnIfAbsent`.

- [ ] **Step 4: Add the REST field**

In `services/atlas-monsters/atlas.com/monsters/monster/rest.go`, add to `RestModel`
after `SpawnSourceId` (line 36):

```go
	SpawnIfAbsent          bool                `json:"spawnIfAbsent,omitempty"`
```

- [ ] **Step 5: Guard the create path**

In `monster/processor.go`, insert at the top of `Create`, immediately after the
opening `p.l.Debugf` (line 260) and **before** the `monsterInformation` lookup:

```go
	// FR-2.1 / design D5: the idempotency decision lives here, against this
	// service's own registry, because an orchestrator-side GET-then-POST is a
	// cross-service TOCTOU two simultaneous map entries would both pass.
	// Scoped to the field — including its instance — so two instances of the
	// same map each get their own monster.
	if input.SpawnIfAbsent {
		existing, err := p.GetInField(f)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to check field [%s] for an existing monster [%d].", f.Id(), input.MonsterId)
			return Model{}, err
		}
		for _, m := range existing {
			if m.MonsterId() == input.MonsterId {
				p.l.Debugf("Suppressing spawn of monster [%d] in field [%s]: already present as [%d].", input.MonsterId, f.Id(), m.UniqueId())
				return Model{}, nil
			}
		}
	}
```

- [ ] **Step 6: Apply the Step 1 handler decision**

If Step 1 concluded the handler must return `204` on a zero `Model`, make that change
in `world/resource.go`'s `handleCreateMonsterInMap` now, and add a handler-level test
asserting the status code. If Step 1 concluded the existing handler behavior is
already correct, write one sentence in the commit body saying why.

- [ ] **Step 7: Run the tests and confirm they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/ \
        services/atlas-monsters/atlas.com/monsters/world/resource.go
git commit -m "feat(monsters): honor spawnIfAbsent against the field registry"
```

---

## Task A8: generate the map-action schema from Go source

Design D4. `map_script_schema.json` hand-maintains three lists that drift from Go:

- the condition `type` enum (`map_script_schema.json:69-75`) has 5 entries; the
  aggregator supports the ~37 constants at `libs/atlas-saga/validation.go:9-51` plus
  the locally-evaluated `map_id`. Two of the five — `job` and `quest_status` — are
  names the aggregator does **not** accept (it wants `jobId` and `questStatus`), so a
  schema-valid document fails at runtime. That is PRD FR-1.1.
- the operator enum (`map_script_schema.json:80`) omits `in`
  (`libs/atlas-saga/validation.go:60`). That is PRD FR-1.2.
- the operation `type` enum (`map_script_schema.json:102-109`) is hand-kept in
  lockstep with `executor.go`'s switch. It happens to match today; nothing enforces
  it. That is PRD FR-3.0 and the safety net Task A4 depends on.

Follow the `gen-topics.sh` pattern (`tools/gen-topics.sh`), not the pure-bash
`gen-routes.sh`/`gen-tenant-tables.sh` ones: the inputs are Go constant declarations
and a Go `switch`, so the generator parses them with `go/ast` in its own module
outside `go.work`, and the shell script is a thin `GOWORK=off go run .` wrapper.

The per-operation `allOf` param blocks (`map_script_schema.json:119-216`) stay
hand-written — they document intent and are not derivable from Go — but the generator
asserts every operation in the generated enum has an `allOf` block, so a new executor
arm without one is a generator failure rather than a silent hole.

### Files

- `tools/gen-map-action-schema/main.go` — **new file**
- `tools/gen-map-action-schema/go.mod` — **new file**
- `tools/gen-map-action-schema/main_test.go` — **new file**
- `tools/gen-map-action-schema.sh` — **new file**
- `services/atlas-map-actions/docs/map_script_schema.json` — regenerated
- `libs/atlas-saga/validation.go` — read-only; the condition-type and operator constant blocks are the generator's input
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — read-only; `ExecuteOperation`'s switch cases are the generator's input
- `services/atlas-map-actions/atlas.com/map-actions/script/evaluator.go` — read-only; `EvaluateCondition`'s `case "map_id"` is the one locally-evaluated type

Module root: `tools/gen-map-action-schema` (own module, invoked with `GOWORK=off`).

Patterns to copy: `tools/gen-topics.sh` (all 17 lines) is the wrapper to copy verbatim
with the paths changed. `libs/atlas-kafka/gen/` is the Go-generator-in-its-own-module
precedent — read its `go.mod` and its `--check` handling and mirror both.
`tools/catalog-lint/go.mod` is the precedent for a `tools/`-rooted module with
`replace` directives.

**Interfaces:**
- Produces:
  - `tools/gen-map-action-schema.sh` — no args rewrites `services/atlas-map-actions/docs/map_script_schema.json`; `--check` exits 1 with a unified diff on drift and writes nothing.
  - The generated schema's condition `type` enum = `map_id` plus every string literal assigned in `libs/atlas-saga/validation.go`'s first `const` block, sorted with `map_id` first then the rest alphabetically.
  - The operator enum = every string literal in `libs/atlas-saga/validation.go`'s operator `const` block, in declaration order: `=`, `>`, `<`, `>=`, `<=`, `in`, plus `==` and `!=` (both accepted by `evaluateMapId` but not saga constants — the generator appends them from a named literal list with a comment saying why).
  - The operation `type` enum = every `case "..."` string in `ExecuteOperation`'s switch, in declaration order.

- [ ] **Step 1: Write the failing generator test**

`tools/gen-map-action-schema/main_test.go` — new file, package `main`.

**`TestParseConditionTypes`** — feed the parser this literal Go source (as a string,
parsed with `go/parser.ParseFile` from memory, not from disk):

```go
package saga

const (
	JobCondition   = "jobId"
	MesoCondition  = "meso"
	LevelCondition = "level"
)

const (
	Equals = "="
	In     = "in"
)
```

Assert `parseStringConsts(src, 0)` returns `[]string{"jobId","meso","level"}` and
`parseStringConsts(src, 1)` returns `[]string{"=","in"}` — declaration order, comments
and blank lines ignored.

**`TestParseOperationCases`** — feed this literal Go source:

```go
package script

func (e *OperationExecutor) ExecuteOperation(f field.Model, characterId uint32, op operation.Model) error {
	switch op.Type() {
	case "field_effect":
		return nil
	case "lock_ui":
		return nil
	case "unlock_ui":
		return nil
	default:
		return nil
	}
}
```

Assert `parseSwitchCases(src, "ExecuteOperation")` returns
`[]string{"field_effect","lock_ui","unlock_ui"}` — the `default` arm contributes
nothing.

**`TestParseSwitchCasesMissingFunc`** — the same source, asking for
`"NoSuchFunction"`. Assert it returns an error whose message contains
`ExecuteOperation`-style text: exactly `function NoSuchFunction not found`.

**`TestRenderIsDeterministic`** — call the renderer twice with the same inputs and
assert byte equality, so `--check` cannot flap.

**`TestRenderRequiresAllOfBlockPerOperation`** — render with operations
`[]string{"field_effect","play_sound"}` against a hand-written `allOf` fixture
containing only a `field_effect` block. Assert the render returns an error whose
message is exactly `operation "play_sound" has no allOf param block in the schema template`.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd tools/gen-map-action-schema && GOWORK=off go test ./... -v`
Expected: the module does not exist yet.

- [ ] **Step 3: Create the module**

`tools/gen-map-action-schema/go.mod`:

```
module github.com/Chronicle20/atlas/tools/gen-map-action-schema

go 1.27.0
```

No dependencies beyond the standard library (`go/ast`, `go/parser`, `go/token`,
`encoding/json`, `os`, `path/filepath`, `sort`, `strconv`, `strings`).

Add it to the workspace beside the two precedents. Both `libs/atlas-kafka/gen`
(`go.work:9`) and `tools/catalog-lint` (`go.work:97`) **are** listed in `go.work`, and
are additionally invoked with `GOWORK=off` at run time so the generator resolves against
its own `go.mod` rather than the workspace. Do the same here:

Run:
```bash
grep -n "catalog-lint" go.work
```
Expected: one hit at the end of the `use (` block. Add
`\t./tools/gen-map-action-schema` alongside it, keeping the block's existing ordering.

- [ ] **Step 4: Write the generator**

`tools/gen-map-action-schema/main.go` implements, at minimum:

```go
// parseStringConsts returns the string literal values of the nth top-level
// const block in src, in declaration order.
func parseStringConsts(src string, blockIndex int) ([]string, error)

// parseSwitchCases returns the string literal case values of the first
// type-less switch statement in the named function, in declaration order.
// The default arm contributes nothing.
func parseSwitchCases(src, funcName string) ([]string, error)

// render produces the complete schema JSON from the parsed inputs and the
// hand-written allOf template. It errors when an operation has no allOf block.
func render(conditionTypes, operators, operations []string, allOf json.RawMessage) ([]byte, error)

func main()  // no args => write; --check => diff and exit 1 on drift
```

`main` reads, relative to `git rev-parse --show-toplevel`:
- `libs/atlas-saga/validation.go` — const block 0 for condition types, block 1 for operators
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — `ExecuteOperation`'s switch for operations
- `services/atlas-map-actions/docs/map_script_schema.json` — the existing file, from which the hand-written `definitions.operation.allOf` array is read back and re-emitted verbatim (this is the only hand-maintained region)

Condition-type ordering: `map_id` first (it is the sole locally-evaluated type,
`evaluator.go:33`), then the saga constants in declaration order. Emit the schema
with `json.MarshalIndent(v, "", "  ")` and a trailing newline so the checked-in file
and the generated buffer are byte-comparable.

`--check` mirrors `tools/gen-tenant-tables.sh`'s tail exactly: diff the generated
buffer against the checked-in file, print the unified diff to stderr, and exit 1 with
`gen-map-action-schema: services/atlas-map-actions/docs/map_script_schema.json is stale; run tools/gen-map-action-schema.sh and commit`.

- [ ] **Step 5: Write the wrapper**

`tools/gen-map-action-schema.sh`, mode `0755`:

```bash
#!/usr/bin/env bash
# Regenerate services/atlas-map-actions/docs/map_script_schema.json from Go
# source: the condition-type and operator constants in libs/atlas-saga/
# validation.go, and the operation cases in atlas-map-actions'
# ExecuteOperation switch. task-290 design D4 / PRD FR-1.1, FR-1.2, FR-1.5,
# FR-3.0.
#
#   gen-map-action-schema.sh           rewrite the schema in place
#   gen-map-action-schema.sh --check   exit 1 with a diff on drift; writes nothing
#
# The generator is its own module and is deliberately outside go.work, so it
# is invoked with GOWORK=off (same posture as tools/gen-topics.sh).
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT/tools/gen-map-action-schema"
GOWORK=off exec go run . "$@"
```

- [ ] **Step 6: Run the generator tests, then regenerate**

Run:
```bash
cd tools/gen-map-action-schema && GOWORK=off go test ./... -v && cd -
chmod +x tools/gen-map-action-schema.sh
./tools/gen-map-action-schema.sh
git diff --stat services/atlas-map-actions/docs/map_script_schema.json
```
Expected: tests PASS; the schema diff shows the condition enum growing from 5 to 38
entries, `job`→`jobId` and `quest_status`→`questStatus` gone, and `in` added to the
operator enum.

- [ ] **Step 7: Add the `spawn_monster` params to the hand-written `allOf` block**

Task A5 added two params. In `map_script_schema.json`'s `spawn_monster` `allOf` block
(currently lines 158-192), remove `"required": ["monsterId"]` and add the two new
params plus the mutual exclusion:

```json
          "then": {
            "properties": {
              "params": {
                "type": "object",
                "oneOf": [
                  { "required": ["monsterId"], "not": { "required": ["monsterIds"] } },
                  { "required": ["monsterIds"], "not": { "required": ["monsterId"] } }
                ],
                "properties": {
                  "monsterId": {
                    "type": "string",
                    "description": "Monster ID to spawn. Mutually exclusive with monsterIds."
                  },
                  "monsterIds": {
                    "type": "string",
                    "description": "Comma-separated monster IDs; one is chosen uniformly at execution. Mutually exclusive with monsterId."
                  },
                  "spawnIfAbsent": {
                    "type": "string",
                    "enum": ["true", "false"],
                    "description": "When \"true\", atlas-monsters suppresses the spawn if a monster of this template is already on the field instance."
                  },
                  "x": { "type": "string", "description": "X coordinate for spawn position" },
                  "y": { "type": "string", "description": "Y coordinate for spawn position" },
                  "count": { "type": "string", "description": "Number of monsters to spawn (default: 1)" },
                  "mapId": { "type": "string", "description": "Map ID to spawn in (defaults to current map)" }
                }
              }
            }
          }
```

Then re-run `./tools/gen-map-action-schema.sh` (which re-emits this block verbatim)
and confirm `./tools/gen-map-action-schema.sh --check` exits 0.

- [ ] **Step 8: Verify the generator catches drift**

Run:
```bash
python3 - <<'PY'
import json,io
p='services/atlas-map-actions/docs/map_script_schema.json'
d=json.load(open(p))
d['definitions']['condition']['properties']['type']['enum'].append('bogus')
json.dump(d,open(p,'w'),indent=2)
open(p,'a').write('\n')
PY
./tools/gen-map-action-schema.sh --check; echo "exit=$?"
git checkout -- services/atlas-map-actions/docs/map_script_schema.json
./tools/gen-map-action-schema.sh --check; echo "exit=$?"
```
Expected: first `--check` prints a diff and reports `exit=1`; after the checkout the
second reports `exit=0`.

- [ ] **Step 9: Commit**

```bash
git add tools/gen-map-action-schema/ tools/gen-map-action-schema.sh \
        services/atlas-map-actions/docs/map_script_schema.json
git commit -m "feat(tools): generate the map-action schema enums from Go source"
```

---

## Task A9: retrofit `spawnIfAbsent` onto the two seeded spawn documents

PRD FR-2.3. Two of the nine seeded documents carry an unguarded `spawn_monster`:

- `map-actions/onUserEnter/map-108010301.json` — five rules
  (`spawn_archer_test`, `spawn_warrior_test`, `spawn_mage_test`, `spawn_thief_test`,
  `spawn_pirate_test`), each one `spawn_monster` operation. Cosmic's
  `108010301.js` guards every one of these through its local `spawnMob` helper, which
  checks `map.getMonsterById(id) != null` before spawning; Atlas dropped the guard.
- `map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json` — one rule
  (`spawn_space_gaga`). Cosmic's `spaceGaGa_sMap` guards via `resetEnteredScript()`,
  which Atlas does not support and `/convert-map` grants an explicit exception; the
  `spawnIfAbsent` guard is the closest available equivalent and is required for
  Task A10's catalog-lint rule to pass.

Both must change in **all 11** version roots — 22 files.

### Files

- `deploy/seed/gms/12_1/map-actions/onUserEnter/map-108010301.json` — add `"spawnIfAbsent": "true"` to all five operations
- `deploy/seed/gms/12_1/map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json` — add it to the one operation
- …and the identical pair under `gms/48_1`, `gms/61_1`, `gms/72_1`, `gms/79_1`, `gms/83_1`, `gms/84_1`, `gms/87_1`, `gms/92_1`, `gms/95_1`, `jms/185_1` — 22 files total

This is the one place in this plan where a mechanical repo-wide sweep is correct: the
11 copies are byte-identical, so edit the `gms/83_1` pair and copy them over the
other ten roots.

- [ ] **Step 1: Confirm the 11 roots are byte-identical before touching them**

Run:
```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  diff -q deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json \
          deploy/seed/$r/map-actions/onUserEnter/map-108010301.json
  diff -q deploy/seed/gms/83_1/map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json \
          deploy/seed/$r/map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json
done
echo "identical=$?"
```
Expected: no diff output, `identical=0`. If any root differs, STOP and report — the
replication invariant Task A10 is about to enforce is already broken.

- [ ] **Step 2: Edit the `gms/83_1` pair**

In `deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json`, each of the
five `spawn_monster` operations' `params` object becomes (keys stay alphabetically
sorted, 2-space indent):

```json
              "params": {
                "monsterId": "9001002",
                "spawnIfAbsent": "true",
                "x": "188",
                "y": "20"
              },
```

with `monsterId` per rule: `spawn_archer_test` `9001002`, `spawn_warrior_test`
`9001000`, `spawn_mage_test` `9001001`, `spawn_thief_test` `9001003`,
`spawn_pirate_test` `9001008`. `x`/`y` stay `"188"`/`"20"` on all five.

In `deploy/seed/gms/83_1/map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json`, the
single operation becomes:

```json
              "params": {
                "monsterId": "9300331",
                "spawnIfAbsent": "true",
                "x": "-28",
                "y": "0"
              },
```

- [ ] **Step 3: Replicate to the other ten roots**

Run:
```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json \
     deploy/seed/$r/map-actions/onUserEnter/map-108010301.json
  cp deploy/seed/gms/83_1/map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json \
     deploy/seed/$r/map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json
done
git status --short deploy/seed/
```
Expected: exactly 22 modified files.

- [ ] **Step 4: Validate against the generated schema**

Run:
```bash
./tools/gen-map-action-schema.sh --check
grep -c spawnIfAbsent deploy/seed/*/*/map-actions/onUserEnter/map-108010301.json | sort -u
grep -c spawnIfAbsent deploy/seed/*/*/map-actions/onFirstUserEnter/map-spaceGaGa_sMap.json | sort -u
```
Expected: `--check` exits 0; the first `grep -c` reports `5` for all 11 files and the
second reports `1` for all 11 (the `sort -u` collapses to one distinct count line
each, so any root out of step shows up as a second line).

- [ ] **Step 5: Commit**

```bash
git add deploy/seed/
git commit -m "fix(seed): guard the two seeded map-action spawns with spawnIfAbsent"
```

---

## Task A10: catalog-lint enforces the map-action seed invariants

Design §6, PRD FR-1.6 and FR-2.2. `tools/catalog-lint` already declares both
map-action subdomains (`tools/catalog-lint/subdomains.go:18-19`) and already opens and
parses every seed JSON (`main.go:52-101`), so three checks cost one extra decode:

1. **Replication (FR-1.6).** Every map-action document must be byte-identical across
   all 11 version roots — no partial replication, and no root missing a document
   another root has.
2. **`spawnIfAbsent` (FR-2.2).** Every `spawn_monster` operation must carry
   `"spawnIfAbsent": "true"`. Cosmic guards every map spawn; a converted document
   that omits the guard stacks duplicates on re-entry.
3. **Schema validity (design §6).** Every map-action document's `attributes` must
   validate against `services/atlas-map-actions/docs/map_script_schema.json`.

`subdomainRule` (`subdomains.go:5-9`) carries only `{path, typ, pattern}`, so these
checks cannot be expressed as rule-table rows. Add them as a `typ == "map-action"`
special case inside `lint()`, gathering documents on the first walk and asserting
after it.

### Files

- `tools/catalog-lint/main.go` — gather map-action docs during the walk; add the three checks after it
- `tools/catalog-lint/mapactions.go` — **new file**; the three checks, kept out of `main.go`
- `tools/catalog-lint/go.mod` — add the JSON-schema validator dependency
- `tools/catalog-lint/mapactions_test.go` — **new file**
- `tools/catalog-lint/testdata/bad/map-action-unreplicated/` — **new fixture tree**
- `tools/catalog-lint/testdata/bad/map-action-unguarded-spawn/` — **new fixture tree**
- `tools/catalog-lint/testdata/bad/map-action-schema-invalid/` — **new fixture tree**
- `tools/catalog-lint/testdata/good/` — extend with a valid replicated map-action pair
- `services/atlas-map-actions/docs/map_script_schema.json` — read-only input

Module root: `tools/catalog-lint` (own module — run `go test` from inside it).

Patterns to copy: `tools/catalog-lint/main_test.go:10-18` (`buildLint`) and
`main_test.go:29-35` (`TestLint_IDMismatchExitsNonZero`) are the exact test shape —
build the binary, run it as a subprocess against a `testdata/` tree, assert on the
exit code. The existing fixtures at `tools/catalog-lint/testdata/good/gms/83_1/` and
`testdata/bad/id-mismatch/gms/83_1/` show the minimum tree: a `CATALOG_REVISION` file
plus the subdomain directory.

**Interfaces:**
- Consumes: `services/atlas-map-actions/docs/map_script_schema.json` as generated by Task A8; the `spawnIfAbsent` param as landed by Task A9.
- Produces:
  - `func checkMapActions(root string, docs []mapActionDoc, schemaPath string) []string` in `mapactions.go` — returns one error string per violation, empty when clean.
  - `type mapActionDoc struct { path, region, version, hook, id string; raw []byte }`.

- [ ] **Step 1: Choose and vendor the schema validator**

`catalog-lint` is a standalone module. Check whether a JSON-schema validator is
already used anywhere in this repo before adding one:

Run: `grep -rn 'jsonschema\|santhosh-tekuri\|xeipuuv' --include=go.mod --include=go.sum . | head -20`

If a validator is already in use, use that one. If not, use
`github.com/santhosh-tekuri/jsonschema/v6` (draft-07 support, which
`map_script_schema.json:2` declares). Add it to `tools/catalog-lint/go.mod` and run
`GOWORK=off go mod tidy` from `tools/catalog-lint`.

- [ ] **Step 2: Write the failing tests**

`tools/catalog-lint/mapactions_test.go` — new file, package `main`. Copy
`buildLint` usage from `main_test.go:10-27`.

Fixture trees to create first (each needs a `CATALOG_REVISION` in every version dir):

`testdata/good/` — add `gms/83_1/map-actions/onUserEnter/map-t.json` and an
identical `gms/84_1/map-actions/onUserEnter/map-t.json`, plus
`gms/84_1/CATALOG_REVISION`. Content:

```json
{
  "data": {
    "attributes": {
      "description": "fixture",
      "rules": [
        {
          "conditions": [{"operator": "=", "type": "map_id", "value": "100"}],
          "id": "r1",
          "operations": [{"params": {"monsterId": "1", "spawnIfAbsent": "true", "x": "0", "y": "0"}, "type": "spawn_monster"}]
        }
      ],
      "scriptName": "t"
    },
    "id": "t",
    "type": "map-action"
  }
}
```

`testdata/bad/map-action-unreplicated/` — the same document under
`gms/83_1/.../map-t.json` and a **differing** copy under `gms/84_1/.../map-t.json`
(change `description` to `"drifted"`).

`testdata/bad/map-action-unguarded-spawn/` — one root, one document whose
`spawn_monster` params omit `spawnIfAbsent`.

`testdata/bad/map-action-schema-invalid/` — one root, one document whose condition
`type` is `"quest_status"` (a name the generated enum no longer contains).

Tests:

| test function | fixture | expect |
|---|---|---|
| `TestLint_MapActionGoodTreeExitsZero` | `testdata/good` | exit 0 |
| `TestLint_MapActionUnreplicatedExitsNonZero` | `testdata/bad/map-action-unreplicated` | exit != 0 |
| `TestLint_MapActionUnguardedSpawnExitsNonZero` | `testdata/bad/map-action-unguarded-spawn` | exit != 0 |
| `TestLint_MapActionSchemaInvalidExitsNonZero` | `testdata/bad/map-action-schema-invalid` | exit != 0 |

Also add a direct unit test on `checkMapActions` asserting the exact message strings:

| scenario | expected message |
|---|---|
| unreplicated | `map-actions/onUserEnter/map-t.json: differs between gms/83_1 and gms/84_1` |
| missing from a root | `map-actions/onUserEnter/map-t.json: present in gms/83_1, missing from gms/84_1` |
| unguarded spawn | `<path>: rule "r1" operation 1: spawn_monster requires "spawnIfAbsent": "true"` |
| schema invalid | `<path>: schema: /rules/0/conditions/0/type: value must be one of the enumerated values` (assert with `strings.Contains` on `schema:` and the JSON pointer, since the validator's wording is its own) |

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd tools/catalog-lint && GOWORK=off go test ./... -v`
Expected: compile failure — `checkMapActions` undefined.

- [ ] **Step 4: Gather map-action documents during the walk**

In `main.go`'s walk (`main.go:52-101`), after the existing type/id checks succeed,
append to a slice declared alongside `errs`:

```go
	var mapActionDocs []mapActionDoc
```

```go
		if rule.typ == "map-action" {
			mapActionDocs = append(mapActionDocs, mapActionDoc{
				path:    path,
				region:  parts[0],
				version: parts[1],
				hook:    parts[len(parts)-2],
				id:      env.Data.ID,
				raw:     b,
			})
		}
```

and after the walk, before the `len(errs) > 0` check (`main.go:103`):

```go
	errs = append(errs, checkMapActions(root, mapActionDocs, mapActionSchemaPath())...)
```

`mapActionSchemaPath()` resolves `services/atlas-map-actions/docs/map_script_schema.json`
from `git rev-parse --show-toplevel`. When the schema cannot be found — which is the
case for the `testdata/` fixture trees run outside the repo, if that turns out to be
so — the schema check is **skipped with an explicit stderr note**, never silently.
Confirm the behavior you get and encode it; do not leave it ambiguous.

- [ ] **Step 5: Implement the three checks**

`tools/catalog-lint/mapactions.go` — new file. `checkMapActions`:

1. **Replication.** Group docs by `hook + "/" + filepath.Base(path)`. Collect the set
   of `region/version` roots seen anywhere in the tree (from the CATALOG_REVISION scan
   in `main.go`, or by re-reading the top two directory levels). For each group:
   - every root must have the document — otherwise
     `<rel>: present in <r1>, missing from <r2>`
   - every copy's `raw` must be `bytes.Equal` to the first — otherwise
     `<rel>: differs between <r1> and <r2>`

   Sort roots before comparing so the message names them deterministically.

2. **`spawnIfAbsent`.** Unmarshal `data.attributes` into a minimal local struct
   (`rules[].id`, `rules[].operations[].type`, `rules[].operations[].params`). For each
   operation with `type == "spawn_monster"` whose `params["spawnIfAbsent"] != "true"`,
   emit `<path>: rule %q operation %d: spawn_monster requires "spawnIfAbsent": "true"`
   with a 1-based operation index.

3. **Schema.** Compile `map_script_schema.json` once, validate each document's
   `data.attributes` object, and emit `<path>: schema: <validator error>` per failure.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd tools/catalog-lint && GOWORK=off go test ./... -v`
Expected: PASS.

- [ ] **Step 7: Run the linter against the real seed tree**

Run:
```bash
cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed; echo "exit=$?"
```
Expected: `exit=0`. If it reports violations, they are real defects in the seeded 99
documents — fix them here rather than weakening the check, and say so in the commit
body.

- [ ] **Step 8: Commit**

```bash
git add tools/catalog-lint/
git commit -m "feat(catalog-lint): enforce map-action replication, spawn guards and schema validity"
```

---

## Task A11: wire catalog-lint and the schema generator into `tools/verify.sh`

Design F5: `catalog-lint` runs only from `.github/workflows/catalog-lint.yml` — where
it is additionally advisory (`|| true`) on `push` — and has **zero** references in
`tools/verify.sh`. That is a script/CI divergence that already exists, independent of
this task; closing it is what makes Task A10's checks a gate rather than a suggestion.

The schema drift `--check` from Task A8 needs the same treatment.

### Files

- `tools/verify.sh` — two new `touched`-gated blocks, placed beside the existing generator `--check` blocks (currently lines 865-887)
- `.github/workflows/catalog-lint.yml` — read-only reference for the path filter to mirror

- [ ] **Step 1: Read the existing block and the workflow's path filter**

Run:
```bash
sed -n '860,895p' tools/verify.sh
cat .github/workflows/catalog-lint.yml
```

The `touched '<regex>'` helper gates each block on paths changed since the merge base.
Mirror the workflow's `paths:` list exactly for the catalog-lint block.

- [ ] **Step 2: Add the two blocks**

Insert immediately after the existing topic-manifest block (which currently ends at
`tools/verify.sh:886`, the `fi` after the `skip "topic manifest drift ..."` line):

```bash
if touched '^(deploy/seed/|tools/catalog-lint/|libs/atlas-seeder/|services/atlas-map-actions/docs/map_script_schema\.json)'; then
    step "catalog lint"            bash -c 'cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed'
    step "catalog lint tests"      bash -c 'cd tools/catalog-lint && GOWORK=off go test ./...'
else
    skip "catalog lint (no seed, catalog-lint, seeder or map-action schema change)"
fi

if touched '^(libs/atlas-saga/validation\.go|services/atlas-map-actions/atlas\.com/map-actions/script/(executor|evaluator)\.go|services/atlas-map-actions/docs/map_script_schema\.json|tools/gen-map-action-schema)'; then
    step "map-action schema drift"      ./tools/gen-map-action-schema.sh --check
    step "map-action schema gen tests"  bash -c 'cd tools/gen-map-action-schema && GOWORK=off go test ./...'
else
    skip "map-action schema drift (no saga condition, executor, schema or generator change)"
fi
```

- [ ] **Step 3: Confirm both blocks actually run on this branch**

Every path in both regexes has been touched by Tasks A1-A10, so neither block should
skip.

Run: `./tools/verify.sh --quick 2>&1 | grep -E 'catalog lint|map-action schema'`
Expected: four `step` lines, no `skip` lines. If a `skip` appears, the regex does not
match a path this branch changed — fix the regex, do not adjust the expectation.

- [ ] **Step 4: Confirm the gate actually fails on drift**

Run:
```bash
python3 - <<'PY'
import json
p='services/atlas-map-actions/docs/map_script_schema.json'
d=json.load(open(p))
d['definitions']['condition']['properties']['type']['enum'].append('bogus')
json.dump(d,open(p,'w'),indent=2); open(p,'a').write('\n')
PY
./tools/verify.sh --quick 2>&1 | tail -30; echo "exit=$?"
git checkout -- services/atlas-map-actions/docs/map_script_schema.json
```
Expected: a non-zero exit naming the `map-action schema drift` step. Quote the actual
output in the commit body — do not claim it from memory.

- [ ] **Step 5: Commit**

```bash
git add tools/verify.sh
git commit -m "chore(verify): gate on catalog-lint and map-action schema drift"
```

---

## Task A12: correct the `/convert-map` command contract

PRD FR-1.3 and FR-1.4, design F7. `.claude/commands/convert-map.md` currently tells
the agent to write **bare-schema** JSON to
`services/atlas-map-actions/scripts/map/<hook>/<name>.json` — a path that does not
exist — using condition names (`job`, `quest_status`) the aggregator rejects, with no
mention of the quest-status value shift. Every conversion in Plan B and Plan C is
driven by this document, so it is corrected before either runs.

### Files

- `.claude/commands/convert-map.md` — the whole output-contract section
- `services/atlas-map-actions/docs/map_script_schema.json` — read-only; the generated enums are now the authority the command must point at
- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json` — read-only; the formatting exemplar the command must cite

- [ ] **Step 1: Read the current command doc in full**

Run: `cat .claude/commands/convert-map.md`

Note every place it names an output path, an envelope shape, a condition type, or an
operation type, so none is left stale.

- [ ] **Step 2: Correct the output contract**

The command must state, in its own words but with these exact facts:

- **Path.** `deploy/seed/<client>/<version>/map-actions/<hook>/map-<scriptName>.json`,
  where `<hook>` is `onUserEnter` or `onFirstUserEnter`, replicated **byte-identically**
  to all 11 version roots: `gms/12_1`, `gms/48_1`, `gms/61_1`, `gms/72_1`, `gms/79_1`,
  `gms/83_1`, `gms/84_1`, `gms/87_1`, `gms/92_1`, `gms/95_1`, `jms/185_1`.
  `tools/catalog-lint` fails on a partial replication (Task A10).
- **Envelope.** JSON:API:
  `{"data": {"type": "map-action", "id": "<scriptName>", "attributes": {"scriptName": ..., "description": ..., "rules": [...]}}}`.
  **No `scriptType` attribute** — the hook is derived from the directory
  (`script/subdomain_on_user_enter.go`, `script/subdomain_on_first_user_enter.go`).
- **Formatting.** 2-space indent, object keys sorted alphabetically at every level,
  trailing newline. Exemplar:
  `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json`.
- **Condition names.** The authority is
  `services/atlas-map-actions/docs/map_script_schema.json`'s generated enum, which is
  derived from `libs/atlas-saga/validation.go`. `jobId` not `job`; `questStatus` not
  `quest_status`. Delete every occurrence of `job` and `quest_status` from the command.
- **Quest-status shift (FR-1.4).** Cosmic's `QuestStatus` is
  `UNDEFINED(-1), NOT_STARTED(0), STARTED(1), COMPLETED(2)`; Atlas' aggregator is
  `UNDEFINED=0, NOT_STARTED=1, STARTED=2, COMPLETED=3`. **Every ported
  `getQuestStatus(x) == n` is emitted as `n + 1`.** State this with the worked example:
  Cosmic `getQuestStatus(2175) == 1` becomes
  `{"type": "questStatus", "operator": "=", "referenceId": "2175", "value": "2"}`.
- **`spawnIfAbsent` (FR-2.2).** Cosmic guards every map spawn
  (`map.getMonsterById(id) != null`, `containsNPC`, `countMonster(...) == 0`). Every
  emitted `spawn_monster` operation sets `"spawnIfAbsent": "true"`. catalog-lint
  fails otherwise (Task A10).
- **Operation names.** The authority is the same generated schema, derived from
  `ExecuteOperation`'s switch. An operation not in that enum does not exist; the
  executor now **errors** on it rather than ignoring it (Task A4).
- **`resetEnteredScript()` exception.** Unchanged and still explicit: Atlas has no
  equivalent, and a script whose only guard is `resetEnteredScript()` converts with
  `spawnIfAbsent` where it spawns, and with the omission noted in the document's
  `description` otherwise.

- [ ] **Step 3: Check the sibling command docs for the same defects**

The portal, reactor, quest and NPC conversion commands share this shape and may carry
the same wrong path or wrong condition names.

Run:
```bash
grep -ln 'scripts/map\|"quest_status"\|"job"' .claude/commands/*.md
```
If any sibling matches, report it — do **not** fix it in this task. It is out of this
plan's scope and belongs on its own branch.

- [ ] **Step 4: Verify the corrected doc against a real seed**

Take the corrected contract and check it describes
`deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json` exactly: envelope,
key order, indent, absence of `scriptType`, `gender` condition name.

Run: `cat deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json`

- [ ] **Step 5: Commit**

```bash
git add .claude/commands/convert-map.md
git commit -m "docs(convert-map): correct the output path, envelope, condition names and quest-status shift"
```

---

## Plan A completion gate

- [ ] **Run the full verification gate**

Run: `./tools/verify.sh`
Expected: exit 0. `--quick` and `--no-docker` do **not** satisfy this.

- [ ] **Confirm every Plan A requirement has a landed change**

| Requirement | Task |
|---|---|
| FR-1.1 condition names | A8 (generated enum drops `job`/`quest_status`) |
| FR-1.2 enum breadth + `in` operator | A8 |
| FR-1.3 `/convert-map` path and envelope | A12 |
| FR-1.4 quest-status shift documented | A12 |
| FR-1.5 schema/Go drift check | A8 + A11 |
| FR-1.6 replication check | A10 + A11 |
| FR-2.1 `spawnIfAbsent` param | A5 + A6 + A7 |
| FR-2.2 every spawn guarded | A10 (lint rule) |
| FR-2.3 `108010301` retrofit | A9 (plus `spaceGaGa_sMap`) |
| FR-3.0 no silent no-ops | A4 + A8 |
| Design F1 condition truncation | A1 + A2 + A3 |
| Design F3 spawn instance defect | A5 |
| Design G14a `map_id` range operators | A3 |
| Design D9 / G7 `monsterIds` | A5 |
| Design F5 catalog-lint absent from verify.sh | A11 |

Plan B ([plan-b.md](plan-b.md)) and Plan C ([plan-c.md](plan-c.md)) may not start
until this gate is green.
