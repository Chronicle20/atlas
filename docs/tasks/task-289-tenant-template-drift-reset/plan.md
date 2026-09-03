# Tenant Template Drift Detection & Reset — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Report, per tenant configuration, which sections diverge from the template it derives from, and let an operator reset the tenant back to that baseline — whole-document or per-section — without re-rolling the tenant UUID.

**Architecture:** A new dependency-free `drift` package in `atlas-configurations` canonicalizes either package's `RestModel` into `map[string]json.RawMessage`, prunes empty values, and hashes per section plus an aggregate. `tenants.Processor` gains `WithTemplates` so the read path resolves each distinct `(region, major, minor)` baseline once per request and decorates a new `tenants.ViewRestModel`. A new `POST /configurations/tenants/{tenantId}/reset` merges baseline sections over the stored document and writes through the existing `update` transaction (history-before-write + `AuthorizeWrite`) plus the existing status outbox. The UI adds a drift column, a header drift summary, and a reusable `TenantResetButton` mounted whole-document and per-section.

**Tech Stack:** Go 1.27 (`services/atlas-configurations/atlas.com/configurations`), GORM + sqlite for tests, gorilla/mux, api2go JSON:API; React 19 + TypeScript + TanStack Query + Vitest + Testing Library (`services/atlas-ui`).

**Spec:** `docs/tasks/task-289-tenant-template-drift-reset/design.md` (PRD: `prd.md`, decisions: `decisions.md`)

## Global Constraints

- **Section names are exactly:** `properties`, `socket`, `characters`, `npcs`, `cashShop`, `mapleLife`. `properties` replaces the PRD's literal `usesPin` (design OQ-1). No alias is accepted — `sections: ["usesPin"]` is a `400`.
- **Never compared:** `id`, `environment`, `region`, `majorVersion`, `minorVersion`, `worlds`, `diagnostics` (FR-2.4). `Excluded` lists all but `id`, which carries `json:"-"` on both models.
- **`templates.Revision` is not touched, not moved, not re-expressed.** `services/atlas-configurations/atlas.com/configurations/templates/revision.go` must be byte-identical at the end of this plan except for one added comment (Task 3). Its tests must pass unmodified.
- **`drift` imports nothing from `templates` or `tenants`.** It operates on marshaled JSON only.
- **Module root for every Go task:** `services/atlas-configurations/atlas.com/configurations` (run `go build ./...` / `go test ./...` from there).
- **Frontend cwd:** `services/atlas-ui` (`npm test`, `npm run lint`, `npm run build`).
- Go module path prefix is `atlas-configurations/…` (see any existing import block).
- Reset HTTP statuses: `400` unknown section, `403` cross-environment, `404` tenant not found, `409` no baseline, `422` validation failure.
- Preserve existing line endings. Never edit files under the main repo — all work happens in this worktree.

---

## Design deviations recorded at plan time

Two facts were established from the code during planning that the design assumed otherwise. Both are handled by tasks in this plan.

1. **The two preset models are NOT key-identical.** `tenants/characters/preset.Attributes` carries `AP uint16 \`json:"ap"\`` and `SP string \`json:"sp"\`` (`tenants/characters/preset/rest.go:46-47`); `templates/characters/preset.Attributes` has neither (`templates/characters/preset/rest.go:27-46`). Every tenant preset therefore serializes `"ap":0,"sp":""` where the template never does. `0` and `""` are values, not absences, so pruning cannot erase them, and 10 of 11 shipped seed templates carry 10–16 presets. Left alone, `characters` drift would be permanently `true` for essentially every tenant — the exact NFR-5 failure this task exists to prevent. **Task 1 adds `AP`/`SP` to the templates-side model.**
2. **No tenant `cashShop` or `npcs` page exists.** `src/App.tsx:441-480` routes only `handlers`, `worlds`, `writers`, `properties`, `character/templates`, `character/presets`, `character/maple-life`, `mts-config`, `diagnostics`. FR-6.4's per-section reset is therefore mounted on the pages that exist (properties → `properties`, handlers + writers → `socket`, character templates + presets → `characters`, maple-life → `mapleLife`). `cashShop` and `npcs` remain resettable via the API and via the whole-document header reset. **No new editing pages are created; that is outside this task.**

---

## Task 1: Preset model parity (`ap` / `sp`)

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/characters/preset/rest.go` — add `AP` and `SP` to `Attributes`
- Test: `services/atlas-configurations/atlas.com/configurations/templates/characters/preset/rest_test.go` — new file

**Module root:** `services/atlas-configurations/atlas.com/configurations`

Patterns to copy: `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/rest.go:42-47` (the exact field declarations and their comment); `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/rest_test.go` (test file shape in the same package family).

**Interfaces:**
- Consumes: nothing.
- Produces: `templates/characters/preset.Attributes` gains `AP uint16` (`json:"ap"`) and `SP string` (`json:"sp"`). Task 4's cross-type equality test depends on this.

**Why this is safe for task-201:** both sides of `templates.Revision` marshal the same `templates.RestModel` struct — the shipped-catalog side (`LoadCatalog` parses the seed JSON into `RestModel`) and the stored side (`Make` unmarshals the row into `RestModel`). Adding a field changes both hashes identically, so `SeedDrift` equality is unchanged. No test in `templates/` asserts a literal hash; `revision_test.go` only compares revisions to each other.

- [ ] **Step 1: Write the failing test**

New file `templates/characters/preset/rest_test.go`, package `preset`. No fixtures or helpers needed — this is a pure marshal test.

`TestAttributesCarriesAPAndSP` — marshal a zero `Attributes` and assert both keys are present, so the tenant and template preset documents have the same key set.

| assertion | exact expected |
|---|---|
| `json.Marshal(Attributes{})` contains | `"ap":0` |
| `json.Marshal(Attributes{})` contains | `"sp":""` |
| `json.Marshal(Attributes{AP: 5, SP: "61,0,0"})` contains | `"ap":5` |
| `json.Marshal(Attributes{AP: 5, SP: "61,0,0"})` contains | `"sp":"61,0,0"` |

Use `strings.Contains(string(b), want)` with `t.Errorf("marshaled Attributes missing %q: %s", want, b)`. Imports: `encoding/json`, `strings`, `testing`.

- [ ] **Step 2: Run the test and confirm it fails**

Run from `services/atlas-configurations/atlas.com/configurations`:

```bash
go test ./templates/characters/preset/ -run TestAttributesCarriesAPAndSP -v
```

Expected: FAIL — the marshaled JSON has no `ap` or `sp` key.

- [ ] **Step 3: Add the fields**

In `templates/characters/preset/rest.go`, inside `Attributes`, between `Stats StatBlock \`json:"stats"\`` and `DefaultName string \`json:"defaultName"\``:

```go
	// AP and SP are the unspent ability/skill points the created character
	// should start with. They exist here solely so this model's JSON key set
	// matches tenants/characters/preset.Attributes: the drift comparison
	// (task-289) hashes the marshaled document, and a key present on one
	// side only would report permanent `characters` drift for every tenant.
	// Zero (the marshaled default) reproduces prior behaviour for every
	// existing producer -- neither the admin preset UI nor any shipped seed
	// file sets these today.
	AP uint16 `json:"ap"`
	SP string `json:"sp"`
```

- [ ] **Step 4: Run the tests and confirm they pass**

```bash
go test ./templates/... ./tenants/... 
```

Expected: PASS, including every pre-existing `templates` test unmodified (`TestRevisionIgnoresId`, `TestRevisionNormalizesSocket`, `TestRevisionIsStableLowercaseHex`, the `shipped_test.go` corpus tests).

- [ ] **Step 5: Commit**

```bash
git add templates/characters/preset/rest.go templates/characters/preset/rest_test.go
git commit -m "fix(configurations): give the template preset model the tenant model's ap/sp keys"
```

---

## Task 2: The `drift` package — canonicalization and pruning

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/drift/doc.go` — package doc, `Doc`, `Excluded`, `Named`, `Properties`
- Create: `services/atlas-configurations/atlas.com/configurations/drift/canonical.go` — `Canonicalize`, `prune`
- Create: `services/atlas-configurations/atlas.com/configurations/drift/canonical_test.go`

**Module root:** `services/atlas-configurations/atlas.com/configurations`

Patterns to copy: `services/atlas-configurations/atlas.com/configurations/scope/scope.go:1-19` (package-doc style — a paragraph that states the policy and names the alternative it is not). Test style: table-driven with `t.Run`, `t.Fatalf`/`t.Errorf`, no external assertion library — see `services/atlas-configurations/atlas.com/configurations/templates/revision_test.go:9-54`.

**Interfaces:**
- Consumes: nothing. `drift` imports only the standard library.
- Produces:
  ```go
  package drift

  type Doc map[string]json.RawMessage

  var Excluded = []string{"environment", "region", "majorVersion", "minorVersion", "worlds", "diagnostics"}
  var Named = []string{"socket", "characters", "npcs", "cashShop", "mapleLife"}
  const Properties = "properties"

  func Canonicalize(v any) (Doc, error)
  ```

- [ ] **Step 1: Write the failing test**

New file `drift/canonical_test.go`, package `drift`. Imports: `encoding/json`, `testing`. Helper used by every case:

```go
func canon(t *testing.T, raw string) Doc {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	d, err := Canonicalize(v)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	return d
}

func keys(d Doc) []string { /* sorted key slice for comparison */ }
```

`TestCanonicalizeDropsExcluded` — table-driven, one subtest per row:

| subtest | input JSON | expected key set |
|---|---|---|
| `DropsEveryExcludedKey` | `{"environment":"main","region":"GMS","majorVersion":83,"minorVersion":1,"worlds":[{"name":"w"}],"diagnostics":{"tracePackets":true},"usesPin":true}` | `["usesPin"]` |
| `KeepsComparableSections` | `{"usesPin":false,"socket":{"handlers":[{"opCode":1}]},"characters":{"templates":[{"gender":0}]},"npcs":[{"npcId":1}],"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":1}]}},"mapleLife":{"looks":[{"gender":0}]}}` | `["cashShop","characters","mapleLife","npcs","socket","usesPin"]` |

`TestCanonicalizePrunesEmptiness` — table-driven; each case asserts two inputs canonicalize to byte-identical `json.Marshal(Doc)`:

| subtest | input A | input B |
|---|---|---|
| `NullEqualsEmptyArrayAtTopLevel` | `{"npcs":null,"usesPin":true}` | `{"npcs":[],"usesPin":true}` |
| `NullEqualsAbsentAtTopLevel` | `{"npcs":null,"usesPin":true}` | `{"usesPin":true}` |
| `NestedNullEqualsNestedEmpty` | `{"cashShop":{"commodities":null},"usesPin":true}` | `{"cashShop":{"commodities":[]},"usesPin":true}` |
| `EmptyObjectPrunedRecursively` | `{"cashShop":{"commodities":{}},"usesPin":true}` | `{"usesPin":true}` |
| `NestedEmptyCollapsesParent` | `{"cashShop":{"surprise":{"boxTemplateIds":[]}},"usesPin":true}` | `{"usesPin":true}` |

`TestCanonicalizeKeepsFalsyValues` — table-driven; each case asserts the two inputs canonicalize to **different** marshaled bytes, because `false`/`0`/`""` are values, not absences:

| subtest | input A | input B |
|---|---|---|
| `FalseIsNotPruned` | `{"usesPin":false}` | `{}` |
| `ZeroIsNotPruned` | `{"majorRank":0}` | `{}` |
| `EmptyStringIsNotPruned` | `{"note":""}` | `{}` |

`TestCanonicalizeContentDiffersFromEmpty` — `{"npcs":[{"npcId":1}]}` and `{"npcs":[]}` must produce different marshaled bytes: pruning erases ways of writing nothing, never a real divergence.

`TestCanonicalizeIsDeterministic` — build one `Doc` from `{"socket":{"handlers":[]},"usesPin":true,"npcs":[{"npcId":9}]}` and another from the same object with the keys written in a different order; `json.Marshal` of both `Doc`s must be byte-identical (a `map[string]json.RawMessage` marshals key-sorted).

- [ ] **Step 2: Run the test and confirm it fails**

```bash
go test ./drift/ -v
```

Expected: FAIL to build — `undefined: Canonicalize`, `undefined: Doc`.

- [ ] **Step 3: Write `drift/doc.go`**

```go
// Package drift computes the comparable revision of a configuration
// document. It is a third package that neither `templates` nor `tenants`
// owns, and it imports neither: making one domain package reach into the
// other's hashing internals is exactly the cross-boundary call the repo
// convention warns about, and the direction chosen would decide by
// accident which package owns a policy that belongs to both.
//
// drift operates on MARSHALED JSON, never on either package's Go types.
// templates.maplelife.RestModel and tenants.maplelife.RestModel are
// distinct Go types with identical JSON tags; the hash never learns the
// difference, which is what makes one definition genuinely serve both
// sides (FR-2.1, FR-2.6).
//
// A drift hash is NOT comparable with templates.Revision. Revision hashes
// the STRUCT, in field-declaration order; Aggregate hashes a MAP, in
// key-sorted order. Same document, different bytes, different SHA-256.
// Each is only ever compared against itself:
//
//	template vs shipped seed file  -> templates.Revision, both sides
//	tenant vs baseline template    -> drift.Aggregate,   both sides
//
// Crossing them produces a flag that is permanently true for every row.
package drift

import "encoding/json"

// Doc is a canonical, package-neutral view of a configuration document:
// the comparable top-level keys, each holding its already-marshaled value.
// json.Marshal of a Doc emits keys in sorted order, so a Doc has exactly
// one serialization regardless of how it was built.
type Doc map[string]json.RawMessage

// Excluded names the keys that never participate in drift or reset
// (FR-2.4). "id" is deliberately absent: it carries json:"-" on both
// models, so it cannot be produced, and listing it would imply the
// marshal is untrusted.
var Excluded = []string{
	"environment",
	"region",
	"majorVersion",
	"minorVersion",
	"worlds",
	"diagnostics",
}

// Named lists the sections that get their own key in the drift report.
// Everything comparable and NOT named falls into Properties by
// subtraction, so a new top-level field participates in drift with no
// edit here (FR-2.7).
//
// The cost of that default: adding a new top-level SECTION (an object)
// without adding it here silently folds it into Properties. It still
// drifts and still resets -- it just does not get its own flag. If you
// add a section, add it here.
var Named = []string{
	"socket",
	"characters",
	"npcs",
	"cashShop",
	"mapleLife",
}

// Properties is the residual section: every comparable key not in Named.
// Today that is exactly "usesPin". It supersedes the PRD's literal
// "usesPin" section name (design OQ-1): an enumerated scalar section
// would leave a future top-level scalar in the aggregate hash but in no
// named section -- an indicator that says "something is wrong, and I will
// not tell you what".
const Properties = "properties"
```

- [ ] **Step 4: Write `drift/canonical.go`**

```go
package drift

import "encoding/json"

// Canonicalize marshals v, drops the Excluded keys, and prunes every
// empty value, returning the comparable document.
//
// Pruning recursively removes any object key whose value is null, an
// empty array, or an empty object AFTER recursing into it. This collapses
// the whole nil-vs-empty false-positive class in one generic rule rather
// than one normalizer per slice field: `npcs: null` (a seed file that
// omits the key) is the same document as `npcs: []` (what the UI sends
// after a round trip through `?? []`), at any depth, without naming the
// field.
//
// Pruning cannot hide a real divergence: if one side has content and the
// other is empty, the non-empty side keeps its key and the hashes differ.
// It only erases distinctions between WAYS OF WRITING NOTHING. false, 0
// and "" are NOT pruned -- they are values, not absences.
func Canonicalize(v any) (Doc, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}

	out := make(Doc, len(top))
	for k, raw := range top {
		if isExcluded(k) {
			continue
		}
		pruned, empty, err := prune(raw)
		if err != nil {
			return nil, err
		}
		if empty {
			continue
		}
		out[k] = pruned
	}
	return out, nil
}

func isExcluded(k string) bool {
	for _, e := range Excluded {
		if e == k {
			return true
		}
	}
	return false
}

// prune returns raw with empty descendants removed, and reports whether
// the result is itself empty (null, [], or {}).
func prune(raw json.RawMessage) (json.RawMessage, bool, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false, err
	}
	switch t := v.(type) {
	case nil:
		return nil, true, nil
	case map[string]any:
		obj := make(map[string]json.RawMessage, len(t))
		for k, child := range t {
			cb, err := json.Marshal(child)
			if err != nil {
				return nil, false, err
			}
			pruned, empty, err := prune(cb)
			if err != nil {
				return nil, false, err
			}
			if empty {
				continue
			}
			obj[k] = pruned
		}
		if len(obj) == 0 {
			return nil, true, nil
		}
		b, err := json.Marshal(obj)
		return b, false, err
	case []any:
		if len(t) == 0 {
			return nil, true, nil
		}
		arr := make([]json.RawMessage, 0, len(t))
		for _, child := range t {
			cb, err := json.Marshal(child)
			if err != nil {
				return nil, false, err
			}
			pruned, empty, err := prune(cb)
			if err != nil {
				return nil, false, err
			}
			if empty {
				// An element that prunes to nothing is still an
				// element: the array's LENGTH is content. Keep it
				// as the empty object it canonicalizes to.
				pruned = json.RawMessage("{}")
			}
			arr = append(arr, pruned)
		}
		b, err := json.Marshal(arr)
		return b, false, err
	default:
		// A scalar: false, 0 and "" are values, not absences.
		return raw, false, nil
	}
}
```

- [ ] **Step 5: Run the tests and confirm they pass**

```bash
go test ./drift/ -v
```

Expected: PASS, every subtest.

- [ ] **Step 6: Commit**

```bash
git add drift/doc.go drift/canonical.go drift/canonical_test.go
git commit -m "feat(configurations): add the drift package's canonical document view"
```

---

## Task 3: `drift` hashing, comparison, merge, and section validation

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/drift/revision.go` — `Sections`, `Aggregate`, `Compare`
- Create: `services/atlas-configurations/atlas.com/configurations/drift/merge.go` — `Merge`, `ValidateSections`, `ErrUnknownSection`
- Create: `services/atlas-configurations/atlas.com/configurations/drift/revision_test.go`
- Create: `services/atlas-configurations/atlas.com/configurations/drift/merge_test.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/revision.go` — add the non-comparability comment only (no code change)

**Module root:** `services/atlas-configurations/atlas.com/configurations`

Patterns to copy: `services/atlas-configurations/atlas.com/configurations/templates/revision.go:32-43` (sha256 → `hex.EncodeToString(sum[:])`); `services/atlas-configurations/atlas.com/configurations/scope/scope.go:21-24` (sentinel error declaration + the comment naming the HTTP status it maps to).

**Interfaces:**
- Consumes: `Doc`, `Named`, `Properties` from Task 2.
- Produces:
  ```go
  func Sections(d Doc) (map[string]string, error)
  func Aggregate(d Doc) (string, error)
  func Compare(base, stored Doc) (agg bool, per map[string]bool, err error)
  func Merge(stored, base Doc, sections []string) (Doc, error)
  func ValidateSections(sections []string) error
  var ErrUnknownSection = errors.New("unknown section")
  ```
  `Sections` and `Compare`'s `per` are always fully populated: all six keys, every time.

- [ ] **Step 1: Write the failing tests**

New file `drift/revision_test.go`, package `drift`. Reuse the `canon` helper from `canonical_test.go` (same package — do not redeclare it).

`TestSectionsAlwaysCarriesSixKeys` — `canon(t, "{}")` → `Sections` returns exactly the keys `properties`, `socket`, `characters`, `npcs`, `cashShop`, `mapleLife`. A client never distinguishes "absent" from "false".

`TestSectionsIsHexSHA256` — for `canon(t, '{"usesPin":true}')`, every returned value is 64 characters and matches `^[0-9a-f]{64}$`.

`TestPropertiesIsDefinedBySubtraction` — table-driven:

| subtest | doc A | doc B | expectation |
|---|---|---|---|
| `UsesPinLandsInProperties` | `{"usesPin":true}` | `{"usesPin":false}` | `properties` hashes differ; the other five are equal |
| `UnknownScalarLandsInProperties` | `{"usesPin":true,"enableAutoRegister":true}` | `{"usesPin":true}` | `properties` hashes differ; the other five are equal |
| `NamedSectionDoesNotLeakIntoProperties` | `{"usesPin":true,"npcs":[{"npcId":1}]}` | `{"usesPin":true,"npcs":[{"npcId":2}]}` | `npcs` hashes differ; `properties` equal |

`TestAggregateChangesWithAnySection` — the aggregate of `{"usesPin":true,"npcs":[{"npcId":1}]}` differs from that of `{"usesPin":true,"npcs":[{"npcId":2}]}` and from that of `{"usesPin":false,"npcs":[{"npcId":1}]}`.

`TestCompareIsolatesOneSection` — `base = canon(t, '{"usesPin":true,"socket":{"handlers":[{"opCode":1}]},"npcs":[{"npcId":1}]}')`, `stored` identical except `socket.handlers[0].opCode` is `2`. Assert `agg == true`, `per["socket"] == true`, and `per` is `false` for `properties`, `characters`, `npcs`, `cashShop`, `mapleLife`.

`TestCompareIdenticalDocsReportNoDrift` — `base` and `stored` from the same JSON string → `agg == false` and all six `per` values `false`.

New file `drift/merge_test.go`, package `drift`.

`TestValidateSections` — table-driven, one subtest per case:

| subtest | input | expect |
|---|---|---|
| `NilIsWholeDocument` | `nil` | `nil` error |
| `EmptyIsWholeDocument` | `[]string{}` | `nil` error |
| `EveryNamedSection` | `[]string{"properties","socket","characters","npcs","cashShop","mapleLife"}` | `nil` error |
| `RejectsWorlds` | `[]string{"worlds"}` | `errors.Is(err, ErrUnknownSection)` |
| `RejectsDiagnostics` | `[]string{"diagnostics"}` | `errors.Is(err, ErrUnknownSection)` |
| `RejectsRegion` | `[]string{"region"}` | `errors.Is(err, ErrUnknownSection)` |
| `RejectsId` | `[]string{"id"}` | `errors.Is(err, ErrUnknownSection)` |
| `RejectsEnvironment` | `[]string{"environment"}` | `errors.Is(err, ErrUnknownSection)` |
| `RejectsUsesPinAlias` | `[]string{"usesPin"}` | `errors.Is(err, ErrUnknownSection)` — no alias for `properties` |
| `RejectsGibberish` | `[]string{"socket","nonsense"}` | `errors.Is(err, ErrUnknownSection)` |

The error message must name the offending section: `t.Errorf` if `err.Error()` does not contain the rejected name.

`TestMerge` — table-driven. Build `stored` and `base` with `canon`, call `Merge`, compare `json.Marshal(got)` against `json.Marshal(canon(t, want))`.

| subtest | stored | base | sections | want |
|---|---|---|---|---|
| `EmptySectionsReplacesEverything` | `{"usesPin":false,"npcs":[{"npcId":9}],"socket":{"handlers":[{"opCode":2}]}}` | `{"usesPin":true,"npcs":[{"npcId":1}],"socket":{"handlers":[{"opCode":1}]}}` | `nil` | `{"usesPin":true,"npcs":[{"npcId":1}],"socket":{"handlers":[{"opCode":1}]}}` |
| `NamedSectionOnly` | `{"usesPin":false,"npcs":[{"npcId":9}],"socket":{"handlers":[{"opCode":2}]}}` | `{"usesPin":true,"npcs":[{"npcId":1}],"socket":{"handlers":[{"opCode":1}]}}` | `["socket"]` | `{"usesPin":false,"npcs":[{"npcId":9}],"socket":{"handlers":[{"opCode":1}]}}` |
| `ReplacementNotFieldMerge` | `{"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":9,"hours":2}]},"surprise":{"boxTemplateIds":[7]}}}` | `{"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":1,"hours":1}]}}}` | `["cashShop"]` | `{"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":1,"hours":1}]}}}` — `surprise` is gone, not merged |
| `PropertiesReplacesResidualScalars` | `{"usesPin":false,"npcs":[{"npcId":9}]}` | `{"usesPin":true,"npcs":[{"npcId":1}]}` | `["properties"]` | `{"usesPin":true,"npcs":[{"npcId":9}]}` |
| `PropertiesAddsKeyPresentOnlyInBase` | `{"usesPin":false}` | `{"usesPin":true,"enableAutoRegister":true}` | `["properties"]` | `{"usesPin":true,"enableAutoRegister":true}` |
| `PropertiesRemovesKeyPresentOnlyInStored` | `{"usesPin":false,"legacyFlag":true}` | `{"usesPin":true}` | `["properties"]` | `{"usesPin":true}` |
| `NamedSectionAbsentInBaseIsRemoved` | `{"usesPin":true,"npcs":[{"npcId":9}]}` | `{"usesPin":true}` | `["npcs"]` | `{"usesPin":true}` |
| `UnrequestedSectionsUntouched` | `{"usesPin":false,"mapleLife":{"looks":[{"gender":1}]}}` | `{"usesPin":true,"mapleLife":{"looks":[{"gender":0}]}}` | `["properties"]` | `{"usesPin":true,"mapleLife":{"looks":[{"gender":1}]}}` |

`TestMergeIsIdempotent` — apply `Merge(stored, base, nil)` twice; `json.Marshal` of both results is byte-identical.

`TestMergeRejectsUnknownSection` — `Merge(stored, base, []string{"worlds"})` returns an error satisfying `errors.Is(err, ErrUnknownSection)`.

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
go test ./drift/ -v
```

Expected: FAIL to build — `undefined: Sections`, `undefined: Aggregate`, `undefined: Compare`, `undefined: Merge`, `undefined: ValidateSections`, `undefined: ErrUnknownSection`.

- [ ] **Step 3: Write `drift/revision.go`**

```go
package drift

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// hash is the one hashing primitive: lowercase hex SHA-256 over bytes.
func hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sectionDoc returns the sub-document a section name selects. A Named
// section is the single key of that name (absent -> an empty Doc, so a
// missing section still has a stable hash). Properties is every
// comparable key not in Named.
func sectionDoc(d Doc, section string) Doc {
	out := Doc{}
	if section == Properties {
		for k, v := range d {
			if !isNamed(k) {
				out[k] = v
			}
		}
		return out
	}
	if v, ok := d[section]; ok {
		out[section] = v
	}
	return out
}

func isNamed(k string) bool {
	for _, n := range Named {
		if n == k {
			return true
		}
	}
	return false
}

// All returns every section name in report order: Properties first, then
// Named. The result is always the full set, so a caller never has to
// distinguish an absent key from a false one.
func All() []string {
	out := make([]string, 0, len(Named)+1)
	out = append(out, Properties)
	out = append(out, Named...)
	return out
}

// Sections returns the per-section hex SHA-256 of d. The map is ALWAYS
// fully populated -- every name All() returns is present, even when the
// section is absent from d.
func Sections(d Doc) (map[string]string, error) {
	out := make(map[string]string, len(Named)+1)
	for _, name := range All() {
		b, err := json.Marshal(sectionDoc(d, name))
		if err != nil {
			return nil, err
		}
		out[name] = hash(b)
	}
	return out, nil
}

// Aggregate returns one hex SHA-256 over the whole comparable document.
// NOT comparable with templates.Revision -- see the package doc.
func Aggregate(d Doc) (string, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return hash(b), nil
}

// Compare reports whether stored diverges from base, in aggregate and per
// section. per is always fully populated.
func Compare(base, stored Doc) (bool, map[string]bool, error) {
	ba, err := Aggregate(base)
	if err != nil {
		return false, nil, err
	}
	sa, err := Aggregate(stored)
	if err != nil {
		return false, nil, err
	}
	bs, err := Sections(base)
	if err != nil {
		return false, nil, err
	}
	ss, err := Sections(stored)
	if err != nil {
		return false, nil, err
	}

	per := make(map[string]bool, len(bs))
	for _, name := range All() {
		per[name] = bs[name] != ss[name]
	}
	return ba != sa, per, nil
}
```

- [ ] **Step 4: Write `drift/merge.go`**

```go
package drift

import (
	"errors"
	"fmt"
)

// ErrUnknownSection is returned by ValidateSections for a name that is
// not a comparable section. The reset handler maps it to 400.
//
// There is deliberately no alias: "usesPin" is rejected like any other
// unknown name. An alias would be a permanent second name for a section,
// existing only to paper over an earlier draft.
var ErrUnknownSection = errors.New("unknown section")

// ValidateSections rejects any name that is not comparable. nil and an
// empty slice mean "the whole document" and are always valid.
func ValidateSections(sections []string) error {
	for _, s := range sections {
		if s != Properties && !isNamed(s) {
			return fmt.Errorf("%w: %q", ErrUnknownSection, s)
		}
	}
	return nil
}

// Merge returns stored with the requested sections replaced WHOLESALE by
// base's. A section is replaced key-for-key at the top level, never
// field-merged: "restore to baseline" means the section looks like the
// baseline, not like a union.
//
// nil or empty sections means every comparable section.
//
// Properties is computed over base UNION stored, so a key present on only
// one side is still handled: present in base and not stored -> added;
// present in stored and not base -> removed. Both are correct "restore to
// baseline" outcomes.
func Merge(stored, base Doc, sections []string) (Doc, error) {
	if err := ValidateSections(sections); err != nil {
		return nil, err
	}
	if len(sections) == 0 {
		sections = All()
	}

	out := make(Doc, len(stored))
	for k, v := range stored {
		out[k] = v
	}

	for _, s := range sections {
		if s == Properties {
			for k := range stored {
				if !isNamed(k) {
					delete(out, k)
				}
			}
			for k, v := range base {
				if !isNamed(k) {
					out[k] = v
				}
			}
			continue
		}
		if v, ok := base[s]; ok {
			out[s] = v
		} else {
			delete(out, s)
		}
	}
	return out, nil
}
```

- [ ] **Step 5: Add the non-comparability comment to `templates/revision.go`**

No code change. Insert immediately above `func Revision(rm RestModel) (string, error) {` (currently line 32), after the existing comment block:

```go
// This hash is NOT comparable with drift.Aggregate. Revision hashes the
// STRUCT, in field-declaration order; drift.Aggregate hashes a MAP, in
// key-sorted order. Same document, different bytes. Each is only ever
// compared against itself -- Revision for template-vs-shipped-seed-file,
// drift for tenant-vs-baseline-template. Crossing them produces a flag
// that is permanently true for every row.
```

- [ ] **Step 6: Run the tests and confirm they pass**

```bash
go test ./drift/ -v
go test ./templates/...
```

Expected: PASS. The `templates` tests are unmodified and must stay green — that is the task-201 guard.

- [ ] **Step 7: Commit**

```bash
git add drift/revision.go drift/merge.go drift/revision_test.go drift/merge_test.go templates/revision.go
git commit -m "feat(configurations): add drift section hashing, comparison, and merge"
```

---

## Task 4: Cross-type equality between the two `RestModel`s

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/drift/crosstype_test.go` — new test file, package `drift_test`
- Test: same file

**Module root:** `services/atlas-configurations/atlas.com/configurations`

This test lives in an **external** test package (`package drift_test`) because it imports `templates` and `tenants` — the `drift` package itself must keep importing neither. Both domain packages may import `drift`; `drift`'s external test importing them creates no cycle.

Patterns to copy: `services/atlas-configurations/atlas.com/configurations/tenants/rest_test.go` (round-trip JSON fixture style in this service).

**Interfaces:**
- Consumes: `drift.Canonicalize`, `drift.Aggregate`, `drift.Sections` (Tasks 2–3); `templates.RestModel`, `tenants.RestModel`; Task 1's `ap`/`sp` parity.
- Produces: nothing consumed by later tasks. This is the regression guard for FR-2.6.

- [ ] **Step 1: Write the failing test**

New file `drift/crosstype_test.go`:

```go
package drift_test
```

Imports: `atlas-configurations/drift`, `atlas-configurations/templates`, `atlas-configurations/tenants`, `encoding/json`, `testing`.

`TestIdenticalDocumentsHashIdenticallyAcrossPackages` — the test that fails if someone adds a field to one model and not the other, which is the whole reason it exists.

1. Declare one JSON literal `doc` covering **every comparable section with content** (this is what makes the test load-bearing):

```json
{
  "region": "GMS",
  "majorVersion": 83,
  "minorVersion": 1,
  "usesPin": true,
  "socket": {"handlers": [{"opCode": 1, "validator": "v", "handler": "h"}]},
  "characters": {
    "templates": [{"jobIndex": 0, "subJobIndex": 0, "gender": 0, "mapId": 40000}],
    "presets": [{"id": "11111111-1111-1111-1111-111111111111", "attributes": {"name": "n", "jobId": 100, "level": 10, "stats": {"str": 4, "dex": 4, "int": 4, "luk": 4, "hp": 50, "mp": 5}}}]
  },
  "npcs": [{"npcId": 9000, "impl": "shop"}],
  "cashShop": {"commodities": {"hourlyExpirations": [{"templateId": 1, "hours": 2}]}},
  "mapleLife": {"looks": [{"gender": 0, "faces": [20000], "hairs": [30000], "hairColors": [0], "skinColors": [0]}]}
}
```

Verify the field names against `templates/socket/rest.go`, `templates/characters/template/rest.go`, `templates/characters/preset/rest.go`, `templates/npcs/rest.go`, `templates/cashshop/rest.go` and `templates/maplelife/rest.go` before writing them; if a name in the literal above does not exist on the model, correct the literal, not the model.

2. `json.Unmarshal` it into a `templates.RestModel` and into a `tenants.RestModel`. `t.Fatalf` on error.
3. `drift.Canonicalize` each.
4. Assert `drift.Aggregate(tmplDoc) == drift.Aggregate(tenantDoc)`, with `t.Errorf("aggregate differs across packages:\n templates=%s\n tenants  =%s", a, b)`.
5. Assert `drift.Sections` of each are equal for all six names, reporting each mismatch as `t.Errorf("section %q differs: templates=%s tenants=%s", name, a, b)`.

`TestTenantOnlyDiagnosticsDoNotAffectHashes` — unmarshal the same literal into a `tenants.RestModel`, set `Diagnostics.TracePackets = true`, canonicalize, and assert the aggregate is unchanged from step 4's tenant aggregate. `diagnostics` is tenant-only and must never participate.

`TestWorldsDoNotAffectHashes` — same shape, but populate `Worlds` on the tenant model (one entry, `Name: "w"`), and assert the aggregate is unchanged. `worlds` is tenant-owned (D3).

- [ ] **Step 2: Run the test and confirm it fails**

```bash
go test ./drift/ -run 'TestIdenticalDocumentsHashIdenticallyAcrossPackages' -v
```

Expected before Task 1's fix: FAIL on the `characters` section, because the tenant preset serializes `ap`/`sp` and the template preset does not. Since Task 1 already landed, expected here: PASS on first run — which is the point. If it fails, the two models have diverged again and the fix belongs in the models, not in `drift`.

- [ ] **Step 3: Run the full package and confirm everything passes**

```bash
go test ./drift/ -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add drift/crosstype_test.go
git commit -m "test(configurations): pin cross-package drift hash equality"
```

---

## Task 5: `tenants.ViewRestModel` and baseline-decorated read paths

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/rest.go` — add `ViewRestModel`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/processor.go` — add `WithTemplates`, `makeView`, `ViewByIdProvider`, `AllViewProvider`, `ErrNoBaselineTemplate`, `ErrTenantNotFound`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/mock/processor.go` — add the new interface members
- Test: `services/atlas-configurations/atlas.com/configurations/tenants/view_test.go` — new file

**Module root:** `services/atlas-configurations/atlas.com/configurations`

Patterns to copy:
- `services/atlas-configurations/atlas.com/configurations/templates/rest.go:46-66` (the `ViewRestModel` split and the comment explaining why the write model stays untouched)
- `services/atlas-configurations/atlas.com/configurations/templates/processor.go:74-127` (`WithCatalog` degradation comment, `makeView`, the three view providers)
- `services/atlas-configurations/atlas.com/configurations/templates/processor.go:22-31` (sentinel error block)
- `services/atlas-configurations/atlas.com/configurations/tenants/processor_test.go:28-99` (sqlite test entity + `setupTestDB` + `testLogger` + `createTestRestModel`) — **already in the package, do not redeclare**

**Interfaces:**
- Consumes: `drift.Canonicalize`, `drift.Compare`, `drift.Aggregate`, `drift.All` (Tasks 2–3); `templates.Processor.GetByRegionAndVersion(region string, majorVersion, minorVersion uint16) (templates.RestModel, error)`.
- Produces:
  ```go
  type ViewRestModel struct {
      RestModel
      BaselineTemplateId string          `json:"baselineTemplateId"`
      BaselineRevision   string          `json:"baselineRevision"`
      StoredRevision     string          `json:"storedRevision"`
      TemplateDrift      bool            `json:"templateDrift"`
      SectionDrift       map[string]bool `json:"sectionDrift"`
  }

  // on Processor:
  WithTemplates(tp templates.Processor) Processor
  ViewByIdProvider(id uuid.UUID) model.Provider[ViewRestModel]
  AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]]

  var ErrTenantNotFound = errors.New("tenant not found")
  var ErrNoBaselineTemplate = errors.New("no baseline template")
  ```

**Direction note:** `tenants` importing `templates` introduces no cycle — `templates` imports nothing from `tenants` (verified), and `tenants/characters/rest.go:4` already imports `atlas-configurations/templates/characters/template`.

- [ ] **Step 1: Write the failing test**

New file `tenants/view_test.go`, package `tenants`. It needs a template row in the same sqlite DB, so add a local sqlite-compatible template entity and helper — the `templates` package's own `testEntity` is unexported and unreachable from here:

```go
// testTemplateEntity is a SQLite-compatible mirror of templates.Entity,
// mapped to the same table name so templates.Processor's real queries
// find the rows this test seeds. Copied from
// templates/processor_test.go:22-33.
type testTemplateEntity struct {
	Id           uuid.UUID       `gorm:"type:text;primaryKey"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:text;not null"`
	Environment  string          `gorm:"not null;default:''"`
}

func (testTemplateEntity) TableName() string { return "templates" }

// setupViewTestDB extends setupTestDB (processor_test.go:55) with the
// templates table, so one DB serves both processors.
func setupViewTestDB(t *testing.T) *gorm.DB { /* setupTestDB(t) + AutoMigrate(&testTemplateEntity{}) */ }

// seedTemplate writes a template row directly and returns its id.
func seedTemplate(t *testing.T, db *gorm.DB, region string, major, minor uint16, mutate func(*templates.RestModel)) uuid.UUID
```

`seedTemplate` builds a `templates.RestModel{Region: region, MajorVersion: major, MinorVersion: minor, UsesPin: true}`, applies `mutate` if non-nil, marshals it, and inserts a `testTemplateEntity` with a fixed `uuid.New()`.

A matching `seedTenant(t, db, p Processor, region string, major, minor uint16, mutate func(*RestModel)) uuid.UUID` builds via `createTestRestModel` (processor_test.go:92), applies `mutate`, and calls `p.Create`.

`TestView` — table-driven where practical, `t.Run` per case. Processor under test: `NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))`.

| subtest | setup | expected |
|---|---|---|
| `NoBaselineReportsUnknownNotDrift` | tenant at `GMS 83.1`, no template row | `BaselineTemplateId == ""`, `BaselineRevision == ""`, `StoredRevision != ""`, `TemplateDrift == false`, all six `SectionDrift` values `false`, `err == nil` |
| `IdenticalDocumentsReportNoDrift` | template at `GMS 83.1` with `UsesPin: true`; tenant created from the template's `RestModel` re-marshaled into a `tenants.RestModel` | `BaselineTemplateId == templateId.String()`, `BaselineRevision == StoredRevision`, `TemplateDrift == false`, all six flags `false` |
| `PropertiesEditFlipsOnlyProperties` | as above, then tenant `UsesPin` flipped to `false` | `TemplateDrift == true`, `SectionDrift["properties"] == true`, the other five `false` |
| `NpcsEditFlipsOnlyNpcs` | as above, tenant `NPCs = []npcs.RestModel{{NpcId: 9000, Impl: "shop"}}` | `TemplateDrift == true`, `SectionDrift["npcs"] == true`, the other five `false` |
| `WorldsEditFlipsNothing` | as `IdenticalDocumentsReportNoDrift`, then tenant `Worlds` populated with one entry | `TemplateDrift == false`, all six flags `false` |
| `DiagnosticsEditFlipsNothing` | as above, then tenant `Diagnostics.TracePackets = true` | `TemplateDrift == false`, all six flags `false` |
| `SectionDriftAlwaysCarriesSixKeys` | any of the above | `len(SectionDrift) == 6` and each of `properties`, `socket`, `characters`, `npcs`, `cashShop`, `mapleLife` is present |
| `UnwiredTemplatesProcessorDegrades` | `NewProcessor(...)` with **no** `WithTemplates`, tenant + matching template both present | no panic, `BaselineTemplateId == ""`, `TemplateDrift == false`, all flags `false` |

`TestAllViewProviderResolvesEachBaselineOnce` — the FR-3.4 counting test:

- Seed one template at `GMS 83.1` and one at `GMS 84.1`.
- Seed **six** tenants: four at `GMS 83.1`, two at `GMS 84.1`.
- Wrap the templates processor in a counting stub declared in this file:

```go
// countingTemplates counts GetByRegionAndVersion calls so FR-3.4 is
// ENFORCED rather than described: a per-row lookup on a paged list is not
// acceptable (NFR-1).
type countingTemplates struct {
	templates.Processor
	calls int
}

func (c *countingTemplates) GetByRegionAndVersion(region string, major, minor uint16) (templates.RestModel, error) {
	c.calls++
	return c.Processor.GetByRegionAndVersion(region, major, minor)
}
```

- Call `AllViewProvider(model.Page{Number: 1, Size: 250})()`.
- Assert `len(paged.Items) == 6` and `stub.calls == 2` — once per distinct `{region, major, minor}` key, not once per row. `t.Errorf("GetByRegionAndVersion called %d times, want 2 (one per distinct region/version key)", stub.calls)`.

- [ ] **Step 2: Run the test and confirm it fails**

```bash
go test ./tenants/ -run 'TestView|TestAllViewProviderResolvesEachBaselineOnce' -v
```

Expected: FAIL to build — `undefined: ViewRestModel`, `WithTemplates`, `AllViewProvider`.

- [ ] **Step 3: Add `ViewRestModel` to `tenants/rest.go`**

Append to the file:

```go
// ViewRestModel is the READ-ONLY projection of a tenant configuration:
// RestModel plus the five computed drift attributes. It is a separate
// type for the same reason templates.ViewRestModel is (see its comment):
// Create persists json.Marshal(input) verbatim, so any field with a JSON
// tag on RestModel would be written INTO the stored document, read back
// by Make, and folded into the next revision -- self-reference and
// permanent phantom drift. Keeping the write model untouched means that
// failure class does not exist rather than being defended against.
//
// encoding/json flattens anonymous embedded structs, so the wire shape is
// exactly RestModel's attributes plus five keys, and sparse fieldsets
// (?fields[tenants]=...) keep working. GetName / GetID / SetID promote
// from the embedded RestModel.
//
// SectionDrift is a map, not a struct: a struct would have to be edited
// every time a section is added, which is the FR-2.7 trap one level up.
// The map is ALWAYS fully populated -- all six keys present, all false
// when no baseline resolved -- so a client never distinguishes "absent"
// from "false".
//
// The PATCH path still binds RestModel, so these five are ignored on
// write by omission rather than by code (FR-3.3).
type ViewRestModel struct {
	RestModel
	BaselineTemplateId string          `json:"baselineTemplateId"`
	BaselineRevision   string          `json:"baselineRevision"`
	StoredRevision     string          `json:"storedRevision"`
	TemplateDrift      bool            `json:"templateDrift"`
	SectionDrift       map[string]bool `json:"sectionDrift"`
}
```

- [ ] **Step 4: Extend `tenants/processor.go`**

Add to the imports: `atlas-configurations/drift`, `atlas-configurations/templates`, `errors`, `fmt`.

Add the sentinel block above `type Processor interface`:

```go
// Sentinel errors the reset handler maps to HTTP statuses.
// server.WriteErrorResponse maps everything to 500, so the handler
// switches on these and writes the JSON:API error document itself --
// the same arrangement templates/processor.go:22-31 uses.
var (
	// ErrTenantNotFound wraps gorm.ErrRecordNotFound for a tenant id -> 404.
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrNoBaselineTemplate means no template resolves for the tenant's
	// region/version in the caller's environment or its baseline -> 409.
	// There is nothing to reset to.
	ErrNoBaselineTemplate = errors.New("no baseline template")
)
```

Add to `Processor`:

```go
	WithTemplates(tp templates.Processor) Processor
	ViewByIdProvider(id uuid.UUID) model.Provider[ViewRestModel]
	AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]]
```

Add `templates templates.Processor` to `ProcessorImpl`, and:

```go
// WithTemplates injects the templates processor used to resolve a
// tenant's baseline. Unset means "no baseline for anything": every row
// reports the FR-1.3 unknown state, so an un-wired processor degrades
// safely rather than nil-panicking. Same contract as
// templates.WithCatalog.
//
// Direction is tenants -> templates and never the reverse; templates
// imports nothing from tenants, so there is no cycle.
func (p *ProcessorImpl) WithTemplates(tp templates.Processor) Processor {
	p.templates = tp
	return p
}

// baselineFor resolves the template a tenant's region/version derives
// from. Lookup goes through templates.Processor.GetByRegionAndVersion,
// which carries the overlay/baseline environment fallback for free --
// that IS FR-1.1, and re-implementing the query here would be a second
// definition of visibility.
//
// Any failure degrades to "no baseline" (ok == false), never an error to
// the caller: a read must not 404 or 500 because a template is missing or
// the templates table hiccupped (FR-1.4).
func (p *ProcessorImpl) baselineFor(region string, majorVersion uint16, minorVersion uint16) (templates.RestModel, bool) {
	if p.templates == nil {
		return templates.RestModel{}, false
	}
	rm, err := p.templates.GetByRegionAndVersion(region, majorVersion, minorVersion)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			p.l.WithError(err).WithFields(logrus.Fields{
				"region":       region,
				"majorVersion": majorVersion,
				"minorVersion": minorVersion,
			}).Warn("Unable to resolve baseline template; reporting no baseline")
		}
		return templates.RestModel{}, false
	}
	return rm, true
}

// decorate builds the view for one tenant against an already-resolved
// baseline. baselineOk == false is the FR-1.3 unknown state: empty
// revisions, empty id, and every flag false -- an unknown, never a true.
func decorate(rm RestModel, baseline templates.RestModel, baselineOk bool) (ViewRestModel, error) {
	stored, err := drift.Canonicalize(rm)
	if err != nil {
		return ViewRestModel{}, err
	}
	storedRev, err := drift.Aggregate(stored)
	if err != nil {
		return ViewRestModel{}, err
	}

	v := ViewRestModel{
		RestModel:      rm,
		StoredRevision: storedRev,
		SectionDrift:   make(map[string]bool, len(drift.Named)+1),
	}
	for _, name := range drift.All() {
		v.SectionDrift[name] = false
	}
	if !baselineOk {
		return v, nil
	}

	base, err := drift.Canonicalize(baseline)
	if err != nil {
		return ViewRestModel{}, err
	}
	baseRev, err := drift.Aggregate(base)
	if err != nil {
		return ViewRestModel{}, err
	}
	agg, per, err := drift.Compare(base, stored)
	if err != nil {
		return ViewRestModel{}, err
	}

	v.BaselineTemplateId = baseline.Id
	v.BaselineRevision = baseRev
	v.TemplateDrift = agg
	v.SectionDrift = per
	return v, nil
}

func (p *ProcessorImpl) makeView(rm RestModel) (ViewRestModel, error) {
	baseline, ok := p.baselineFor(rm.Region, rm.MajorVersion, rm.MinorVersion)
	return decorate(rm, baseline, ok)
}

func (p *ProcessorImpl) ViewByIdProvider(id uuid.UUID) model.Provider[ViewRestModel] {
	return model.Map(p.makeView)(p.ByIdProvider(id))
}

// AllViewProvider builds the page in TWO EXPLICIT PHASES rather than
// resolving a baseline inside ParallelMap (FR-3.4, NFR-1):
//
//  1. read the page,
//  2. collect the distinct {region, major, minor} keys -- realistically
//     1-3 per page -- and resolve each baseline ONCE, serially,
//  3. decorate every row from that map.
//
// A cache consulted from inside ParallelMap would need a mutex and would
// still race two goroutines into the same query. Phase separation makes
// "once per distinct key per request" a property of the control flow
// instead of a property of a lock.
func (p *ProcessorImpl) AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]] {
	return func() (model.Paged[ViewRestModel], error) {
		paged, err := p.AllProvider(page)()
		if err != nil {
			return model.Paged[ViewRestModel]{}, err
		}

		type key struct {
			region string
			major  uint16
			minor  uint16
		}
		type resolved struct {
			rm templates.RestModel
			ok bool
		}
		baselines := make(map[key]resolved)
		for _, rm := range paged.Items {
			k := key{rm.Region, rm.MajorVersion, rm.MinorVersion}
			if _, seen := baselines[k]; seen {
				continue
			}
			b, ok := p.baselineFor(k.region, k.major, k.minor)
			baselines[k] = resolved{rm: b, ok: ok}
		}

		items := make([]ViewRestModel, 0, len(paged.Items))
		for _, rm := range paged.Items {
			b := baselines[key{rm.Region, rm.MajorVersion, rm.MinorVersion}]
			v, err := decorate(rm, b.rm, b.ok)
			if err != nil {
				return model.Paged[ViewRestModel]{}, err
			}
			items = append(items, v)
		}

		return model.Paged[ViewRestModel]{
			Items:      items,
			Total:      paged.Total,
			PageNumber: paged.PageNumber,
			PageSize:   paged.PageSize,
		}, nil
	}
}
```

Before writing the final `model.Paged[ViewRestModel]{...}` literal, read `model.Paged`'s field list in the `atlas-model` module and use its actual field names. Find it with:

```bash
go doc github.com/Chronicle20/atlas/libs/atlas-model/model.Paged
```

- [ ] **Step 5: Extend `tenants/mock/processor.go`**

Add fields and methods following the file's existing shape exactly (nil func → zero-value return):

```go
	WithTemplatesFunc     func(tp templates.Processor) tenants.Processor
	ViewByIdProviderFunc  func(id uuid.UUID) model.Provider[tenants.ViewRestModel]
	AllViewProviderFunc   func(page model.Page) model.Provider[model.Paged[tenants.ViewRestModel]]
```

```go
func (m *ProcessorMock) WithTemplates(tp templates.Processor) tenants.Processor {
	if m.WithTemplatesFunc != nil {
		return m.WithTemplatesFunc(tp)
	}
	return m
}

func (m *ProcessorMock) ViewByIdProvider(id uuid.UUID) model.Provider[tenants.ViewRestModel] {
	if m.ViewByIdProviderFunc != nil {
		return m.ViewByIdProviderFunc(id)
	}
	return model.FixedProvider(tenants.ViewRestModel{})
}

func (m *ProcessorMock) AllViewProvider(page model.Page) model.Provider[model.Paged[tenants.ViewRestModel]] {
	if m.AllViewProviderFunc != nil {
		return m.AllViewProviderFunc(page)
	}
	return model.FixedProvider(model.Paged[tenants.ViewRestModel]{})
}
```

Add `"atlas-configurations/templates"` to the mock's imports.

- [ ] **Step 6: Run the tests and confirm they pass**

```bash
go build ./...
go test ./tenants/... ./templates/... ./drift/...
```

Expected: PASS, including `TestAllViewProviderResolvesEachBaselineOnce` with `calls == 2`.

- [ ] **Step 7: Commit**

```bash
git add tenants/rest.go tenants/processor.go tenants/mock/processor.go tenants/view_test.go
git commit -m "feat(configurations): compute tenant template drift on the read path"
```

---

## Task 6: `ResetById`

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/tenants/reset.go` — `ResetById`
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/processor.go` — add `ResetById` to the `Processor` interface
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/mock/processor.go` — add `ResetByIdFunc` / `ResetById`
- Test: `services/atlas-configurations/atlas.com/configurations/tenants/reset_test.go` — new file

**Module root:** `services/atlas-configurations/atlas.com/configurations`

Patterns to copy:
- `services/atlas-configurations/atlas.com/configurations/templates/processor.go:239-297` (`ReseedById`: entity-first lookup, sentinel wrapping, best-effort before-revision, `update` inside `ExecuteTransaction`, structured info log)
- `services/atlas-configurations/atlas.com/configurations/tenants/processor.go:158-176` (the `ExecuteTransaction` body: `update` then re-read then `enqueueTenantStatus` — copy it byte-for-byte in intent)
- `services/atlas-configurations/atlas.com/configurations/tenants/view_test.go` (Task 5's `setupViewTestDB`, `seedTemplate`, `seedTenant` — already in the package, reuse them)
- `services/atlas-configurations/atlas.com/configurations/tenants/processor_test.go:342-371` (history-count assertion), `:713-724` (`latestOutboxTenantEnvelope`)

**Interfaces:**
- Consumes: `drift.ValidateSections`, `drift.Canonicalize`, `drift.Merge`, `drift.ErrUnknownSection` (Tasks 2–3); `ErrTenantNotFound`, `ErrNoBaselineTemplate`, `decorate`, `baselineFor`, `ViewRestModel` (Task 5); `update` (`administrator.go:58`), `enqueueTenantStatus` (`processor.go:33`), `byIdEntityProvider` (`provider.go:23`), `socketValidate` (`processor.go:244`), `validationFailureError` (`validation_error.go:14`).
- Produces: `ResetById(tenantId uuid.UUID, sections []string) (ViewRestModel, error)`.

- [ ] **Step 1: Write the failing test**

New file `tenants/reset_test.go`, package `tenants`. Reuse `setupViewTestDB`, `seedTemplate`, `seedTenant`, `testLogger`, `createTestRestModel`, `latestOutboxTenantEnvelope` — all already in the package.

`TestResetById` — `t.Run` per case; processor is `NewProcessor(l, ctx, db).WithTemplates(templates.NewProcessor(l, ctx, db))`.

| subtest | setup | expected |
|---|---|---|
| `WholeDocumentClearsAllDrift` | template `GMS 83.1` `UsesPin: true`, tenant cloned then mutated: `UsesPin: false`, `NPCs` populated | returned `ViewRestModel.TemplateDrift == false`, all six `SectionDrift` false; stored `UsesPin == true`; stored `NPCs` matches the template's |
| `ScopedResetLeavesOtherSectionsDrifted` | as above, both `properties` and `npcs` drifted | `ResetById(id, []string{"npcs"})` → `SectionDrift["npcs"] == false`, `SectionDrift["properties"] == true`, `TemplateDrift == true`; stored `UsesPin` still `false` |
| `PreservesTheFR44Set` | tenant given `Worlds: []worlds.RestModel{{Name: "w0", ServerMessage: "keep"}}` and `Diagnostics.TracePackets = true`, plus drift in `properties` | after `ResetById(id, nil)`: stored `Id`, `Region`, `MajorVersion`, `MinorVersion`, `Environment`, `Worlds` and `Diagnostics` are byte-for-byte what they were before (compare `json.Marshal` of each field), and `UsesPin` came from the template |
| `Idempotent` | drifted tenant | call `ResetById(id, nil)` twice; the `Entity.Data` bytes after call 1 and call 2 are identical (`bytes.Equal`) |
| `WritesHistoryBeforeTheChange` | drifted tenant with `UsesPin: false` | after one reset, `count(tenant_history where tenant_id = id) == 1` and the single history row's `Data` unmarshals to a `RestModel` whose `UsesPin` is `false` — the PRE-reset document |
| `EnqueuesStatusOutbox` | `t.Setenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS", "tenant-status")`, drifted tenant | after reset, `latestOutboxTenantEnvelope(t, db, "tenant-status")` decodes and its tenant config carries the post-reset `UsesPin` (the template's value) |
| `UnknownSectionRejected` | any tenant | `ResetById(id, []string{"worlds"})` → `errors.Is(err, drift.ErrUnknownSection)`; the stored row is unchanged |
| `UnknownSectionRejectedBeforeIO` | **no** tenant row at all, random uuid | `ResetById(uuid.New(), []string{"nonsense"})` → `errors.Is(err, drift.ErrUnknownSection)`, NOT `ErrTenantNotFound`. A 400 for a typo must not depend on the tenant existing. |
| `MissingTenantIs404` | no row | `ResetById(uuid.New(), nil)` → `errors.Is(err, ErrTenantNotFound)` |
| `NoBaselineIs409` | tenant at `GMS 99.9`, no matching template | `errors.Is(err, ErrNoBaselineTemplate)` |
| `CrossEnvironmentIs403` | tenant created under `ctx` with environment `"main"`, reset attempted under a processor whose ctx carries environment `"other"` | `errors.Is(err, scope.ErrCrossEnvironmentWrite)` |

For the cross-environment case, copy the environment-context construction from `tenants/processor_test.go:609-675` (`TestProcessor_Create_IgnoresClientSuppliedEnvironment` / `TestProcessor_UpdateById_IgnoresClientSuppliedEnvironment`) — read those exact lines and reuse whatever they use to put an `env.Id` into the context.

`TestResetById_PersistsBaselinePresetIdsVerbatim` — the §3.6 trap:

- Seed a template whose `Characters.Presets` holds one preset with an **empty** `Id` and a valid `Attributes` (`JobId: 100`, `Level: 10`, a non-empty `Name`).
- Reset the tenant with `nil` sections, using a processor that has `WithValidator(preset.NewValidator(mock.ProcessorMock{...}))` wired — copy the validator wiring from `tenants/processor_test.go:519-559` (`TestUpdateById_AssignsMissingPresetId`).
- Assert the stored tenant's preset `Id` is still `""` — the validator ran for detection but its mutation was discarded — and that the returned `ViewRestModel.SectionDrift["characters"] == false`.

`TestResetById_ValidationFailureIsNotPersisted` — seed a template whose socket document violates `socketValidate` (copy an invalid fixture from `tenants/socket_validation_test.go`), reset, assert the error is a `*validationFailureError` (`errors.As`) and the stored tenant row is byte-identical to what it was before.

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
go test ./tenants/ -run TestResetById -v
```

Expected: FAIL to build — `undefined: ResetById`.

- [ ] **Step 3: Add `ResetById` to the `Processor` interface**

In `tenants/processor.go`, add to `type Processor interface`:

```go
	ResetById(tenantId uuid.UUID, sections []string) (ViewRestModel, error)
```

- [ ] **Step 4: Write `tenants/reset.go`**

```go
package tenants

import (
	"atlas-configurations/drift"
	"atlas-configurations/tenants/characters/preset"
	"atlas-configurations/tenants/socket"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// ResetById replaces the tenant's stored content, for the requested
// scope, with its baseline template's (FR-4.1). nil or empty sections
// means every comparable section.
//
// It deliberately does NOT go through UpdateById. UpdateById runs the
// preset validator, which assigns a fresh uuid to any preset with an
// empty Id (preset/validator.go:36-47); persisting that output would make
// the tenant differ from the template the instant it was reset, and
// FR-4.10 would fail intermittently -- on exactly the templates that were
// hand-authored rather than round-tripped through a PATCH. This is the
// same trap templates.ReseedById documents at processor.go:152-160,
// arriving from a different direction.
//
// Resolution: run the validator for DETECTION, discard its MUTATION. The
// merged document is persisted verbatim. Consequence, accepted: a
// baseline preset with an empty id is persisted with an empty id, and the
// next ordinary PATCH assigns one -- at which point the tenant genuinely
// has drifted and the flag correctly says so.
func (p *ProcessorImpl) ResetById(tenantId uuid.UUID, sections []string) (ViewRestModel, error) {
	// Validate section names FIRST, before any I/O: a 400 for a typo
	// must not depend on the tenant existing (FR-4.3).
	if err := drift.ValidateSections(sections); err != nil {
		return ViewRestModel{}, err
	}

	e, err := byIdEntityProvider(p.ctx)(tenantId)(p.db)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Scoped, so a tenant in another environment is a 404, not
			// a 403: a caller who cannot read the row learns nothing
			// about it.
			return ViewRestModel{}, fmt.Errorf("%w: %s", ErrTenantNotFound, tenantId)
		}
		return ViewRestModel{}, err
	}

	// Resolve the baseline from the ENTITY's region/version columns, not
	// the document's: the lookup key must come from the row, so a
	// document/column mismatch can never rewrite the key (same reasoning
	// as templates.ReseedById).
	baseline, ok := p.baselineFor(e.Region, e.MajorVersion, e.MinorVersion)
	if !ok {
		return ViewRestModel{}, fmt.Errorf("%w for %s %d.%d", ErrNoBaselineTemplate, e.Region, e.MajorVersion, e.MinorVersion)
	}

	storedModel, err := Make(e)
	if err != nil {
		return ViewRestModel{}, err
	}

	storedDoc, err := drift.Canonicalize(storedModel)
	if err != nil {
		return ViewRestModel{}, err
	}
	baseDoc, err := drift.Canonicalize(baseline)
	if err != nil {
		return ViewRestModel{}, err
	}
	beforeRevision, err := drift.Aggregate(storedDoc)
	if err != nil {
		// Best-effort, as ReseedById is: the point of a reset is to
		// repair a row, so a row too broken to hash must still be
		// repairable.
		p.l.WithError(err).WithField("tenantId", tenantId.String()).Warn("Unable to compute pre-reset revision")
		beforeRevision = ""
	}

	mergedDoc, err := drift.Merge(storedDoc, baseDoc, sections)
	if err != nil {
		return ViewRestModel{}, err
	}
	mergedBytes, err := json.Marshal(mergedDoc)
	if err != nil {
		return ViewRestModel{}, err
	}

	// Unmarshaling into RestModel discards any key the template document
	// has and the tenant model does not. There are none today; if one is
	// ever added the tenant simply does not gain a field it has no code
	// to use.
	var merged RestModel
	if err := json.Unmarshal(mergedBytes, &merged); err != nil {
		return ViewRestModel{}, err
	}

	// Re-apply everything the merge never carried. FR-4.4 is enforced
	// twice: once because Canonicalize dropped these keys so Merge could
	// not see them, and once here from the stored row.
	merged.Id = storedModel.Id
	merged.Environment = storedModel.Environment
	merged.Region = e.Region
	merged.MajorVersion = e.MajorVersion
	merged.MinorVersion = e.MinorVersion
	merged.Worlds = storedModel.Worlds
	merged.Diagnostics = storedModel.Diagnostics

	merged.Socket = socket.Normalize(merged.Socket)
	if err := p.validateReset(merged); err != nil {
		return ViewRestModel{}, err
	}

	data, err := json.Marshal(merged)
	if err != nil {
		return ViewRestModel{}, err
	}

	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		// update gives history-before-write (FR-4.7) and AuthorizeWrite
		// (FR-4.6) with no new code.
		if err := update(p.ctx, tenantId, e.Region, e.MajorVersion, e.MinorVersion, data)(db); err != nil {
			return err
		}
		// Environment is server-owned: re-read the persisted row rather
		// than trusting the merged document, byte-identical to what
		// UpdateById does (processor.go:162-175).
		persisted, err := byIdEntityProvider(p.ctx)(tenantId)(db)()
		if err != nil {
			return err
		}
		sanitized := merged
		sanitized.Environment = persisted.Environment
		return enqueueTenantStatus(db, tenantId, sanitized)
	})
	if err != nil {
		return ViewRestModel{}, err
	}

	afterRevision, aErr := drift.Aggregate(mergedDoc)
	if aErr != nil {
		afterRevision = ""
	}
	// NFR-6: the change must be reconstructable from logs alone.
	p.l.WithFields(logrus.Fields{
		"tenantId":           tenantId.String(),
		"baselineTemplateId": baseline.Id,
		"sections":           sections,
		"beforeRevision":     beforeRevision,
		"afterRevision":      afterRevision,
	}).Info("Tenant configuration reset to baseline template")

	return p.ViewByIdProvider(tenantId)()
}

// validateReset runs the same validators a PATCH runs (FR-4.9) but
// DISCARDS the preset validator's mutation. Validate assigns a fresh uuid
// to any preset with an empty Id and returns the mutated slice; we hand
// it a copy and keep only the errors, so the merged document is persisted
// verbatim.
func (p *ProcessorImpl) validateReset(merged RestModel) error {
	issues := socketValidate(merged.Socket)

	var presetErrs []preset.ValidationError
	if p.validator != nil {
		// Validate mutates in place, so copy the slice. RestModel
		// elements are values, and only Id is assigned, so a shallow
		// copy is sufficient.
		scratch := make([]preset.RestModel, len(merged.Characters.Presets))
		copy(scratch, merged.Characters.Presets)
		_, presetErrs = p.validator.Validate(p.ctx, scratch)
	}

	if len(issues) > 0 || len(presetErrs) > 0 {
		return &validationFailureError{errors: presetErrs, socketIssues: issues}
	}
	return nil
}
```

- [ ] **Step 5: Extend the mock**

In `tenants/mock/processor.go`:

```go
	ResetByIdFunc func(tenantId uuid.UUID, sections []string) (tenants.ViewRestModel, error)
```

```go
func (m *ProcessorMock) ResetById(tenantId uuid.UUID, sections []string) (tenants.ViewRestModel, error) {
	if m.ResetByIdFunc != nil {
		return m.ResetByIdFunc(tenantId, sections)
	}
	return tenants.ViewRestModel{}, nil
}
```

- [ ] **Step 6: Run the tests and confirm they pass**

```bash
go build ./...
go test ./tenants/... ./templates/... ./drift/...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tenants/reset.go tenants/processor.go tenants/mock/processor.go tenants/reset_test.go
git commit -m "feat(configurations): add tenant configuration reset to baseline template"
```

---

## Task 7: Reset route, handler, and the view-model read/create surfaces

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/resource.go` — `viewProcessor` helper, the reset route + handler, `writeJSONAPIError`, and switch GET/GET-by-id/POST to `ViewRestModel`
- Test: `services/atlas-configurations/atlas.com/configurations/tenants/resource_reset_test.go` — new file

**Module root:** `services/atlas-configurations/atlas.com/configurations`

Patterns to copy:
- `services/atlas-configurations/atlas.com/configurations/templates/resource.go:37-42` (`viewProcessor`), `:188-236` (`writeJSONAPIError` and the sentinel switch in `handleReseedConfigurationTemplate`), `:44-80` (create reading back through the view provider)
- `services/atlas-configurations/atlas.com/configurations/tenants/resource.go:77-113` (the synthesized tenant context for the validator — copy it verbatim into the reset handler)
- `services/atlas-configurations/atlas.com/configurations/templates/resource_reseed_test.go:1-60` (route-level test shape) and `:167-191` (`assertJSONAPIErrorDocument` — **copy this helper into the tenants test file**, it is unexported and lives in `templates`)
- `services/atlas-configurations/atlas.com/configurations/tenants/resource_paginate_test.go:13-31` (`testServerInformation`, request helper — already in the package, reuse)

**Interfaces:**
- Consumes: `ResetById` (Task 6), `ViewByIdProvider` / `AllViewProvider` / `WithTemplates` (Task 5), `drift.ErrUnknownSection`, `scope.ErrCrossEnvironmentWrite`.
- Produces: `POST /configurations/tenants/{tenantId}/reset`; GET list, GET by id and POST create now marshal `ViewRestModel`.

- [ ] **Step 1: Write the failing test**

New file `tenants/resource_reset_test.go`, package `tenants`. Copy `assertJSONAPIErrorDocument` from `templates/resource_reseed_test.go:167-191` verbatim (adjusting nothing). Router construction: `router := mux.NewRouter(); InitResource(testServerInformation{})(db)(router, l)`. DB: Task 5's `setupViewTestDB`.

Request helper for this file:

```go
func doReset(t *testing.T, router *mux.Router, id string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(http.MethodPost, "/configurations/tenants/"+id+"/reset", r)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}
```

`TestResetEndpoint` — `t.Run` per case:

| subtest | body | setup | expected |
|---|---|---|---|
| `AbsentBodyIsWholeDocument` | `""` | drifted tenant, matching template | `200`; response `data.attributes.templateDrift == false`; every value in `data.attributes.sectionDrift` is `false`; `data.attributes.socket` is present (the embedded `RestModel` flattened) |
| `EmptyObjectIsWholeDocument` | `{}` | same | `200`, same assertions |
| `AbsentSectionsKeyIsWholeDocument` | `{"data":{"type":"tenants","attributes":{}}}` | same | `200`, same assertions |
| `EmptySectionsArrayIsWholeDocument` | `{"data":{"type":"tenants","attributes":{"sections":[]}}}` | same | `200`, same assertions |
| `ScopedSections` | `{"data":{"type":"tenants","attributes":{"sections":["properties"]}}}` | tenant drifted in `properties` and `npcs` | `200`; `sectionDrift.properties == false`, `sectionDrift.npcs == true`, `templateDrift == true` |
| `UnknownSectionIs400` | `{"data":{"type":"tenants","attributes":{"sections":["worlds"]}}}` | any tenant | `400` + `assertJSONAPIErrorDocument(t, rr, "400")` |
| `UsesPinAliasIs400` | `{"data":{"type":"tenants","attributes":{"sections":["usesPin"]}}}` | any tenant | `400` + error document |
| `MalformedBodyIs400` | `{not json` | any tenant | `400` + error document |
| `UnknownTenantIs404` | `""` | random uuid, no row | `404` + `assertJSONAPIErrorDocument(t, rr, "404")` |
| `NoBaselineIs409` | `""` | tenant at `GMS 99.9`, no template | `409` + `assertJSONAPIErrorDocument(t, rr, "409")` |
| `InvalidUUIDIs400` | `""` | path id `"not-a-uuid"` | `400` |

`TestGetTenantCarriesComputedAttributes` — seed a template and a matching tenant, `GET /configurations/tenants/{id}`, assert `200` and that `data.attributes` contains all five of `baselineTemplateId`, `baselineRevision`, `storedRevision`, `templateDrift`, `sectionDrift`, plus `socket` (proving the embedded model still flattens).

`TestGetTenantsListCarriesComputedAttributes` — same but over `GET /configurations/tenants`, asserting the five keys on `data[0].attributes`.

`TestCreateTenantReturnsViewModel` — `POST /configurations/tenants` with a valid body and a matching template seeded, assert `201` and the five computed keys are present on the response, with `templateDrift == false` when the body was cloned from the template (this is the Go half of FR-5.2).

`TestPatchIgnoresComputedAttributes` — `PATCH /configurations/tenants/{id}` with a body whose `attributes` carries `templateDrift: true`, `storedRevision: "deadbeef"` and `sectionDrift: {"socket": true}` alongside valid fields: assert `204`, then read the raw `Entity.Data` from the DB and assert it contains none of the strings `baselineTemplateId`, `baselineRevision`, `storedRevision`, `templateDrift`, `sectionDrift`, and that a subsequent `GET` shows no phantom drift.

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
go test ./tenants/ -run 'TestResetEndpoint|TestGetTenant|TestCreateTenantReturnsViewModel|TestPatchIgnores' -v
```

Expected: FAIL — `404` on the reset route (not registered), and the computed attributes absent from the GET responses.

- [ ] **Step 3: Register the route and add the helpers**

In `tenants/resource.go`, add to `InitResource`'s subrouter, after the DELETE line:

```go
			r.HandleFunc("/{tenantId}/reset", rest.RegisterHandler(l)(si)("reset_configuration_tenant", handleResetConfigurationTenant(db))).Methods(http.MethodPost)
```

`rest.RegisterHandler`, not `rest.RegisterInputHandler[T]`: the body is optional, and `RegisterInputHandler` requires a JSON:API envelope. The handler decodes the body itself and tolerates `io.EOF` as "empty".

Add the helpers:

```go
// viewProcessor is the read/reset processor: the ordinary processor with
// the templates processor attached, so a baseline is resolvable. The
// write paths (Create, UpdateById, DeleteById) deliberately do NOT need
// it -- mirrors templates/resource.go:37-42.
func viewProcessor(d *rest.HandlerDependency, db *gorm.DB) Processor {
	return NewProcessor(d.Logger(), d.Context(), db).
		WithTemplates(templates.NewProcessor(d.Logger(), d.Context(), db))
}

// writeJSONAPIError emits the same document shape validationFailureError
// renders, for the statuses server.WriteErrorResponse cannot express (it
// maps everything to 500/503). Copied from templates/resource.go:188-202.
func writeJSONAPIError(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{{
			"status": strconv.Itoa(status),
			"title":  title,
			"detail": detail,
		}},
	})
}

// resetRequest is the reset endpoint's optional body. An absent body,
// `{}`, an absent `sections` key and `sections: []` are all equivalent
// and mean "every comparable section" (FR-4.2).
type resetRequest struct {
	Data struct {
		Attributes struct {
			Sections []string `json:"sections"`
		} `json:"attributes"`
	} `json:"data"`
}

func parseResetSections(r *http.Request) ([]string, error) {
	var body resetRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if errors.Is(err, io.EOF) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return body.Data.Attributes.Sections, nil
}
```

Add imports: `atlas-configurations/templates`, `io`, `strconv`, `atlas-configurations/drift`, `atlas-configurations/scope`.

- [ ] **Step 4: Write the reset handler**

```go
func handleResetConfigurationTenant(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				sections, err := parseResetSections(r)
				if err != nil {
					writeJSONAPIError(w, http.StatusBadRequest, "malformed request body", "The reset request body could not be decoded: "+err.Error())
					return
				}

				// The validator's atlas-data calls are tenant-scoped and
				// atlas-configurations takes no tenant headers, so
				// synthesize a tenant context exactly as the PATCH path
				// does (resource.go:81-93) -- from the URL tenant id and
				// the STORED row's region/version, which the processor
				// re-reads. Without it the atlas-data-backed preset rules
				// silently skip.
				ctx := d.Context()
				stored, gErr := NewProcessor(d.Logger(), ctx, db).GetById(tenantId)
				if gErr == nil {
					if t, terr := tenantlib.Create(tenantId, stored.Region, stored.MajorVersion, stored.MinorVersion); terr == nil {
						ctx = tenantlib.WithContext(ctx, t)
					} else {
						d.Logger().WithError(terr).Warn("Unable to construct tenant model for reset; preset validation will skip atlas-data lookups.")
					}
				}

				p := NewProcessor(d.Logger(), ctx, db).
					WithTemplates(templates.NewProcessor(d.Logger(), ctx, db)).
					WithValidator(preset.NewValidator(data.NewProcessor(d.Logger())))

				view, err := p.ResetById(tenantId, sections)
				if err == nil {
					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(view)
					return
				}

				switch {
				case errors.Is(err, drift.ErrUnknownSection):
					writeJSONAPIError(w, http.StatusBadRequest, "unknown section", err.Error())
				case errors.Is(err, scope.ErrCrossEnvironmentWrite):
					writeJSONAPIError(w, http.StatusForbidden, "cross-environment write", err.Error())
				case errors.Is(err, ErrTenantNotFound):
					writeJSONAPIError(w, http.StatusNotFound, "tenant not found", "No configuration tenant exists with id "+tenantId.String()+".")
				case errors.Is(err, ErrNoBaselineTemplate):
					writeJSONAPIError(w, http.StatusConflict, "no baseline template", "No configuration template resolves for this tenant's region and version, so there is nothing to reset to.")
				default:
					var ve *validationFailureError
					if errors.As(err, &ve) {
						// 422, not the PATCH path's 400: a validation
						// failure here means "the server's own baseline
						// is unprocessable", not "your body is bad". The
						// request was fine.
						w.Header().Set("Content-Type", "application/vnd.api+json")
						w.WriteHeader(http.StatusUnprocessableEntity)
						_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
						return
					}
					d.Logger().WithError(err).Errorf("Unable to reset configuration tenant.")
					server.WriteErrorResponse(d.Logger())(w)(err)
				}
			}
		})
	}
}
```

`ResetById` must return a 200 with a body, so no `WriteHeader` call precedes `MarshalResponse`.

- [ ] **Step 5: Switch the read and create surfaces to `ViewRestModel`**

Three edits in `tenants/resource.go`, each additive on the wire:

1. `handleGetConfigurationTenants` — replace `NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)()` with `viewProcessor(d, db).AllViewProvider(page)()`, and `MarshalPaginatedResponse[[]RestModel]` with `MarshalPaginatedResponse[[]ViewRestModel]`.
2. `handleGetConfigurationTenant` — replace `.GetById(tenantId)` with `viewProcessor(d, db).ViewByIdProvider(tenantId)()`, and `MarshalResponse[RestModel]` with `MarshalResponse[ViewRestModel]`.
3. `handleCreateConfigurationTenant` — after `Create` succeeds and the `Location` header is set, replace the `input.Id = …` + `MarshalResponse[RestModel](…)(input)` pair with a read-back through the view provider, exactly as `templates/resource.go:61-77` does:

```go
			// Read back through the view provider so POST returns exactly
			// what a subsequent GET returns -- same attributes, same
			// computed drift. It also means the onboarding flow can
			// assert FR-5.2 from the create response.
			view, err := viewProcessor(d, db).ViewByIdProvider(tenantId)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to read back created configuration tenant.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			w.Header().Set("Location", "/configurations/tenants/"+tenantId.String())

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(view)
```

- [ ] **Step 6: Run the tests and confirm they pass**

```bash
go build ./...
go test ./...
```

Expected: PASS, including the pre-existing `tenants/resource_paginate_test.go` and `tenants/resource_no_content_test.go` — if either now fails on an attribute assertion, read it before changing it: the added keys are additive and existing assertions should still hold.

- [ ] **Step 7: Commit**

```bash
git add tenants/resource.go tenants/resource_reset_test.go
git commit -m "feat(configurations): expose tenant drift attributes and the reset endpoint"
```

---

## Task 8: Service documentation

**Files:**
- Modify: `services/atlas-configurations/docs/domain.md:117-158` — the Tenants section
- Modify: `services/atlas-configurations/docs/rest.md` — the tenants endpoints (`:201-340`) plus a new reset section

Patterns to copy: `services/atlas-configurations/docs/domain.md:96-114` (the Templates Processors block, including how `ReseedById` and the view providers are worded); `services/atlas-configurations/docs/rest.md:171-198` (the `POST …/reseed` endpoint entry, including its error table); `docs/rest.md:22` (the sentence pattern "Each resource's attributes include … in addition to the … own fields").

**Interfaces:**
- Consumes: everything Tasks 2–7 produced.
- Produces: nothing.

- [ ] **Step 1: Update `docs/domain.md`**

Add a `drift` block to the Templates section's Processors list area — actually place it as its own subsection between `## Templates` and `## Tenants`, styled like the existing `**templates.Catalog**` block:

```markdown
**drift**
- The one definition of a comparable revision, shared by both sides of every tenant-vs-template comparison. Operates on marshaled JSON, never on either package's Go types, so structurally identical documents in distinct Go types hash identically
- Sections: `properties` (every comparable key not otherwise named — today `usesPin`), `socket`, `characters`, `npcs`, `cashShop`, `mapleLife`
- Excluded from every revision: `id`, `environment`, `region`, `majorVersion`, `minorVersion`, `worlds`, `diagnostics`
- Empty values (`null`, `[]`, `{}`) are pruned recursively before hashing, so a document that omits a collection and one that carries an empty collection are the same document. `false`, `0` and `""` are values and are never pruned
- A `drift` hash is not comparable with `templates.Revision`: one hashes a struct in field order, the other a map in key order
```

In the Tenants section, add to Core Models:

```markdown
**ViewRestModel**
- Embeds `RestModel` and adds five read-only computed attributes: `BaselineTemplateId`, `BaselineRevision`, `StoredRevision`, `TemplateDrift`, `SectionDrift`
- `SectionDrift` is a map keyed by section name, always fully populated with all six keys
- Read paths only. The write model bound by POST/PATCH is unchanged, so the computed attributes are ignored on write by omission
```

Add to Invariants:

```markdown
- A tenant's baseline is the template row matching its `(region, majorVersion, minorVersion)` as it stands at read time, resolved through the templates overlay/baseline environment fallback. No baseline resolves to the unknown state — empty revisions, empty baseline id, every drift flag `false` — never to `true`
- Baseline resolution never fails a read: a missing or unreadable template yields the unknown state, not a 404 or 500
- A reset writes only the `tenants` row. The tenant's `id`, `region`, `majorVersion`, `minorVersion`, `environment`, `worlds` and `diagnostics` survive verbatim at every scope
- A reset records history before writing and enqueues the tenant status outbox message in the same transaction, identically to an update
- A reset validates the merged document with the same validators an update runs, but discards the preset validator's id-assignment mutation, so the persisted document is byte-identical to the baseline's content
```

Add to the `tenants.Processor` list:

```markdown
- `ResetById` - Replaces the tenant's stored content, for the requested sections, with its baseline template's (creates history record, enqueues status)
- `AllViewProvider` / `ViewByIdProvider` - Read paths that additionally compute `BaselineTemplateId`, `BaselineRevision`, `StoredRevision`, `TemplateDrift` and `SectionDrift` against the resolved baseline template. The list resolves each distinct region/version baseline once per request
```

- [ ] **Step 2: Update `docs/rest.md`**

1. `GET /api/configurations/tenants` (§201) — append to its Response Model paragraph: *"Each resource's attributes include `baselineTemplateId`, `baselineRevision`, `storedRevision`, `templateDrift`, and `sectionDrift` in addition to the tenant's own fields."*
2. `GET /api/configurations/tenants/{tenantId}` (§227) — same sentence for the single resource.
3. `POST /api/configurations/tenants` (§250) — note that the `201` response is the tenant's view model, carrying the same five computed attributes.
4. `PATCH /api/configurations/tenants/{tenantId}` (§288) — note that a body carrying the five computed attributes succeeds and does not persist them.
5. Add a new section immediately after `DELETE /api/configurations/tenants/{tenantId}` (§317-340), styled exactly like the reseed entry at `:171-198`:

```markdown
### POST /api/configurations/tenants/{tenantId}/reset

Replaces the tenant's stored content, for the requested scope, with its baseline template's.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| tenantId | UUID | path | yes |

**Request Model**

Optional. An absent body, `{}`, an absent `sections` key, and an empty `sections` array are all equivalent and mean "every comparable section".

```json
{ "data": { "type": "tenants", "attributes": { "sections": ["socket"] } } }
```

Valid section names: `properties`, `socket`, `characters`, `npcs`, `cashShop`, `mapleLife`.

**Response Model**

Single `tenants` resource in its view shape (200 OK), including the five computed attributes.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format, a malformed request body, or a `sections` entry that is not a comparable section (JSON:API `errors` array) |
| 403 | The caller's environment does not match the tenant row's environment (JSON:API `errors` array) |
| 404 | No configuration tenant exists with the given ID, or it is not visible to the caller (JSON:API `errors` array) |
| 409 | No baseline template resolves for the tenant's region and version (JSON:API `errors` array) |
| 422 | The baseline content fails tenant validation (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
| 500 | Database error |
```

- [ ] **Step 3: Verify the docs match the code**

```bash
grep -n "reset" services/atlas-configurations/docs/rest.md
grep -n "sectionDrift\|ResetById\|drift" services/atlas-configurations/docs/domain.md
```

Confirm every section name, status code and attribute name above matches what Tasks 2–7 actually implemented.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/docs/domain.md services/atlas-configurations/docs/rest.md
git commit -m "docs(configurations): document tenant drift attributes and the reset endpoint"
```

---

## Task 9: Frontend types and the two hygiene deletions

**Files:**
- Modify: `services/atlas-ui/src/services/api/tenants.service.ts` — extend `TenantConfigAttributes`, add `resetTenantConfiguration`, strip computed keys in `updateTenantConfiguration`
- Modify: `services/atlas-ui/src/lib/utils/config-export.ts:74-83` — extend the delete block
- Test: `services/atlas-ui/src/services/api/__tests__/tenants.service.test.ts` — extend
- Test: `services/atlas-ui/src/lib/utils/__tests__/config-export.test.ts` — extend (create if absent)

**Working directory:** `services/atlas-ui`

Patterns to copy: `services/atlas-ui/src/types/models/template.ts:112-164` (how the template side documents its computed attributes as optional with a "computed server-side; ignored on write" comment); `services/atlas-ui/src/services/api/templates.service.ts:365-376` (`reseed`'s shape — **but not its `skipTenantHeaders: true`**, which is template-specific because templates are global; tenant calls carry normal tenant headers); `services/atlas-ui/src/services/api/__tests__/tenants.service.test.ts:7-18` (the `vi.mock("@/lib/api/client")` shape) and `:52-71` (`seededConfig` fixture builder).

**Note on file location:** `TenantConfigAttributes` is declared in `src/services/api/tenants.service.ts:76-135`, not in `src/types/models/tenant.ts` — that file is a pure re-export barrel. Edit the service file.

**Interfaces:**
- Consumes: the wire shape from Task 7.
- Produces:
  ```ts
  interface TenantSectionDrift { [section: string]: boolean }

  // added to TenantConfigAttributes, all optional:
  baselineTemplateId?: string;
  baselineRevision?: string;
  storedRevision?: string;
  templateDrift?: boolean;
  sectionDrift?: TenantSectionDrift;

  export type TenantResetSection = "properties" | "socket" | "characters" | "npcs" | "cashShop" | "mapleLife";

  // on tenantsService:
  reset(id: string, sections?: TenantResetSection[], options?: ServiceOptions): Promise<TenantConfig>
  ```

- [ ] **Step 1: Write the failing tests**

Extend `src/services/api/__tests__/tenants.service.test.ts`, following the existing `vi.mock("@/lib/api/client")` + `seededConfig()` shape already in that file.

`describe("reset")`:

| test | call | assert |
|---|---|---|
| `posts with no body for a whole-document reset` | `tenantsService.reset("t1")` | `post` called with `"/api/configurations/tenants/t1/reset"` and `undefined` body |
| `posts the sections envelope when scoped` | `tenantsService.reset("t1", ["socket"])` | `post` called with `"/api/configurations/tenants/t1/reset"` and `{ data: { type: "tenants", attributes: { sections: ["socket"] } } }` |
| `does not skip tenant headers` | `tenantsService.reset("t1")` | the options object passed to `post` has no `skipTenantHeaders: true` — tenants are tenant-scoped, unlike templates |

`describe("updateTenantConfiguration computed-key hygiene")`:

| test | setup | assert |
|---|---|---|
| `strips the five computed keys from the PATCH body` | a `TenantConfig` whose `attributes` carries `baselineTemplateId: "b"`, `baselineRevision: "r"`, `storedRevision: "s"`, `templateDrift: true`, `sectionDrift: { socket: true }` plus `usesPin: false` | the object passed to `patch` has `data.attributes` with none of those five keys, and still has `usesPin` and every other real key |
| `still applies the caller's updates` | same, with `{ usesPin: true }` as `updatedAttributes` | `data.attributes.usesPin === true` |

Extend (or create) `src/lib/utils/__tests__/config-export.test.ts`:

| test | input | assert |
|---|---|---|
| `strips the tenant computed keys` | a tenant attributes object carrying all five computed keys plus `usesPin`, `socket`, `worlds` | `toConfigExportPayload` output has none of `baselineTemplateId`, `baselineRevision`, `storedRevision`, `templateDrift`, `sectionDrift`, and still has `usesPin`, `socket`, `worlds` |
| `still strips the template computed keys` | a template attributes object with `shippedRevision`, `storedRevision`, `seedDrift` | none of the three survive (the existing behaviour, unchanged) |

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
npm test -- src/services/api/__tests__/tenants.service.test.ts src/lib/utils/__tests__/config-export.test.ts
```

Expected: FAIL — `tenantsService.reset is not a function`, and the computed keys survive both the PATCH body and the export payload.

- [ ] **Step 3: Extend `TenantConfigAttributes` and add the reset call**

In `src/services/api/tenants.service.ts`, add above `interface TenantConfigAttributes`:

```ts
/**
 * The section names the reset endpoint accepts. `properties` is the
 * residual section — every comparable top-level key not claimed by one of
 * the five named sections, which today is exactly `usesPin`. The server
 * rejects `usesPin` as a name: there is no alias.
 */
export type TenantResetSection =
  | "properties"
  | "socket"
  | "characters"
  | "npcs"
  | "cashShop"
  | "mapleLife";
```

Add to `TenantConfigAttributes`, after `diagnostics`:

```ts
  /**
   * Computed server-side (task-289); never persisted and ignored on write.
   * Optional so a response from an older backend still type-checks — read
   * `templateDrift === true`, never a truthy check.
   */
  baselineTemplateId?: string;
  baselineRevision?: string;
  storedRevision?: string;
  templateDrift?: boolean;
  /** Always fully populated by a current backend: all six section keys. */
  sectionDrift?: Record<string, boolean>;
```

Add the service method alongside the other config methods:

```ts
  /**
   * Resets a tenant configuration to its baseline template. Omit
   * `sections` for the whole document.
   *
   * Unlike templatesService.reseed this does NOT set
   * `skipTenantHeaders` — templates are global, tenant configurations are
   * not, and the reset must carry the ordinary tenant headers.
   */
  async reset(
    id: string,
    sections?: TenantResetSection[],
    options?: ServiceOptions,
  ): Promise<TenantConfig> {
    const body =
      sections && sections.length > 0
        ? { data: { type: "tenants", attributes: { sections } } }
        : undefined;
    const config = await api.post<TenantConfig>(
      `${CONFIG_PATH}/${id}/reset`,
      body,
      options,
    );
    return sortTenantConfig(config);
  },
```

Check `api.post`'s actual return typing in `src/lib/api/client.ts` before writing this — if it returns the JSON:API envelope rather than the resource, follow whatever `getTenantConfigurationById` (`:267-276`) does to unwrap.

Add the strip in `updateTenantConfiguration`, replacing the `attributes` line:

```ts
  async updateTenantConfiguration(tenant, updatedAttributes, options?) {
    // The five computed attributes are read-only and server-owned. The
    // server ignores them (they are absent from the bound write model),
    // so this is hygiene rather than a fix — it keeps request bodies
    // honest instead of echoing a hash of the document back at the
    // service that produced it.
    const {
      baselineTemplateId: _baselineTemplateId,
      baselineRevision: _baselineRevision,
      storedRevision: _storedRevision,
      templateDrift: _templateDrift,
      sectionDrift: _sectionDrift,
      ...writable
    } = tenant.attributes;

    const input: UpdateTenantConfigInput = {
      data: {
        id: tenant.id,
        type: "tenants",
        attributes: { ...writable, ...updatedAttributes },
      },
    };
    await api.patch<void>(`${CONFIG_PATH}/${tenant.id}`, input, options);
    return {
      ...tenant,
      attributes: { ...tenant.attributes, ...updatedAttributes },
    };
  },
```

If the lint config rejects unused destructured bindings even with the `_` prefix, use `delete` on a shallow copy instead — match whatever the repo's eslint config already permits.

- [ ] **Step 4: Extend the export delete block**

In `src/lib/utils/config-export.ts`, extend the existing delete block (currently lines 81-83):

```ts
  delete out.shippedRevision;
  delete out.storedRevision;
  delete out.seedDrift;
  // task-289's tenant-side equivalents. Deleted unconditionally: `out` is
  // an untyped record, and deleting a key the template path never had is
  // a no-op, so this needs no branch on `kind`. `storedRevision` is
  // deliberately shared — one delete covers both sides.
  delete out.baselineTemplateId;
  delete out.baselineRevision;
  delete out.templateDrift;
  delete out.sectionDrift;
```

- [ ] **Step 5: Run the tests and confirm they pass**

```bash
npm test -- src/services/api/__tests__/tenants.service.test.ts src/lib/utils/__tests__/config-export.test.ts
npx tsc --noEmit
```

Expected: PASS, no type errors.

- [ ] **Step 6: Commit**

```bash
git add src/services/api/tenants.service.ts src/lib/utils/config-export.ts src/services/api/__tests__/tenants.service.test.ts src/lib/utils/__tests__/config-export.test.ts
git commit -m "feat(ui): type the tenant drift attributes and add the reset service call"
```

---

## Task 10: The reset mutation hook and the onboarding clone fix

**Files:**
- Modify: `services/atlas-ui/src/lib/hooks/api/useTenants.ts` — add `useResetTenantConfiguration`
- Modify: `services/atlas-ui/src/services/api/onboarding.service.ts:106-118` — copy `mapleLife`
- Test: `services/atlas-ui/src/lib/hooks/api/__tests__/useTenants.reset.test.tsx` — new file
- Test: `services/atlas-ui/src/services/api/__tests__/onboarding.service.test.ts` — extend (create if absent)

**Working directory:** `services/atlas-ui`

Patterns to copy: `services/atlas-ui/src/lib/hooks/api/useTemplates.ts:351-373` (`useReseedTemplate` — the invalidate-on-success-only comment and the three `invalidateQueries` calls); `services/atlas-ui/src/lib/hooks/api/useTenants.ts:289-344` (`useUpdateTenantConfiguration`'s invalidation set: `configDetail(id)`, `configLists()`, `socketKeys.all`); `services/atlas-ui/src/lib/hooks/api/__tests__/useTenants.socketInvalidation.test.tsx` (an existing hook test in this exact area — copy its `QueryClientProvider` wrapper and `renderHook` setup).

**Interfaces:**
- Consumes: `tenantsService.reset` and `TenantResetSection` (Task 9); `tenantKeys` (`useTenants.ts:29-44`), `socketKeys` (`src/lib/hooks/api/socketKeys.ts`).
- Produces:
  ```ts
  export function useResetTenantConfiguration(): UseMutationResult<
    TenantConfig,
    Error,
    { id: string; sections?: TenantResetSection[] }
  >;
  ```

- [ ] **Step 1: Write the failing tests**

New file `src/lib/hooks/api/__tests__/useTenants.reset.test.tsx`, copying the provider wrapper and `renderHook` shape from `useTenants.socketInvalidation.test.tsx`. Mock `@/services/api` (or the exact module `useTenants.ts` imports `tenantsService` from) with `vi.mock`.

| test | action | assert |
|---|---|---|
| `calls the service with the id and sections` | `mutateAsync({ id: "t1", sections: ["socket"] })` | `reset` called with `("t1", ["socket"])` |
| `omits sections for a whole-document reset` | `mutateAsync({ id: "t1" })` | `reset` called with `("t1", undefined)` |
| `invalidates the detail, the list, and socket on success` | successful `mutateAsync({ id: "t1" })` | `invalidateQueries` spy called with `tenantKeys.configDetail("t1")`, with `tenantKeys.configLists()`, and with `socketKeys.all` |
| `invalidates nothing on failure` | `reset` rejects | `invalidateQueries` not called — a failed reset changed nothing server-side |

Extend `src/services/api/__tests__/onboarding.service.test.ts` (create it if absent, copying the `vi.mock("@/lib/api/client")` shape from `tenants.service.test.ts:7-18`):

`TS half of FR-5.2` — `onboardTenant` must put **every comparable key** into the POST body, so the Go half's premise stays true.

| test | setup | assert |
|---|---|---|
| `copies mapleLife from the template` | a template whose `attributes.mapleLife` is `{ looks: [...], classes: [...] }` | the attributes object passed to `createTenantConfiguration` has `mapleLife` deep-equal to the template's |
| `copies every comparable section` | a template with all of `usesPin`, `socket`, `characters`, `npcs`, `cashShop`, `mapleLife`, `worlds` populated | the created attributes carry all seven keys, each deep-equal to the template's |

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
npm test -- src/lib/hooks/api/__tests__/useTenants.reset.test.tsx src/services/api/__tests__/onboarding.service.test.ts
```

Expected: FAIL — `useResetTenantConfiguration is not a function`, and `mapleLife` absent from the created attributes.

- [ ] **Step 3: Add the hook**

In `src/lib/hooks/api/useTenants.ts`, after `useUpdateTenantConfiguration`:

```ts
/**
 * Resets a tenant configuration to its baseline template — whole
 * document when `sections` is omitted, otherwise exactly those sections.
 *
 * Invalidates on SUCCESS ONLY: a failed reset changed nothing
 * server-side, so there is nothing stale to refetch. Mirrors
 * useReseedTemplate (useTemplates.ts:351-373).
 *
 * socketKeys.all is invalidated because a socket reset changes what the
 * socket matrix and the handlers/writers grids show, and none of those
 * clear on their own.
 */
export function useResetTenantConfiguration(): UseMutationResult<
  TenantConfig,
  Error,
  { id: string; sections?: TenantResetSection[] }
> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, sections }) => tenantsService.reset(id, sections),
    onSuccess: (_data, { id }) => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.configDetail(id) });
      queryClient.invalidateQueries({ queryKey: tenantKeys.configLists() });
      queryClient.invalidateQueries({ queryKey: socketKeys.all });
    },
  });
}
```

Import `TenantResetSection` from wherever Task 9 exported it, and add it to the file's type re-export block (`useTenants.ts:382-387`) if that block is how consumers reach these types.

- [ ] **Step 4: Fix the onboarding clone**

In `src/services/api/onboarding.service.ts`, add to the `configAttributes` object, after the `cashShop` conditional spread:

```ts
        // mapleLife is a non-pointer struct on both Go models, so it is
        // always present in a template response. Copied unconditionally
        // rather than behind the cashShop-style `!== undefined` guard: a
        // conditional would reintroduce exactly the omission being fixed
        // the first time a template happened to serialize it as absent,
        // and a tenant missing mapleLife reports drift the moment it is
        // created (D5, FR-5.1).
        mapleLife: template.attributes.mapleLife,
```

If `TenantConfigAttributes.mapleLife` is optional and `TemplateAttributes.mapleLife` is too, this assignment type-checks; confirm with `npx tsc --noEmit` rather than assuming.

- [ ] **Step 5: Run the tests and confirm they pass**

```bash
npm test -- src/lib/hooks/api/__tests__/useTenants.reset.test.tsx src/services/api/__tests__/onboarding.service.test.ts
npx tsc --noEmit
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add src/lib/hooks/api/useTenants.ts src/services/api/onboarding.service.ts src/lib/hooks/api/__tests__/useTenants.reset.test.tsx src/services/api/__tests__/onboarding.service.test.ts
git commit -m "feat(ui): add the tenant reset mutation and copy mapleLife when cloning a template"
```

---

## Task 11: `TenantResetButton`

**Files:**
- Create: `services/atlas-ui/src/components/features/tenants/TenantResetButton.tsx`
- Test: `services/atlas-ui/src/components/features/tenants/__tests__/TenantResetButton.test.tsx` — new file

**Working directory:** `services/atlas-ui`

Patterns to copy: `services/atlas-ui/src/components/features/templates/TemplateReseedButton.tsx` — **the whole file**. This component is that one re-cut; keep every convention its comments establish:
- the `Tooltip` root is always mounted and only `TooltipContent` is gated on `disabled` (its lines 84-100), so the button's DOM node is never remounted and focus is never dropped;
- `AlertDialogCancel` renders **first** in the footer (its lines 113-116), so Enter never fires the destructive action;
- `AlertDialogAction` calls `e.preventDefault()` then `void onConfirm()`;
- errors route through `createErrorFromUnknown(e, "Reset failed").message` into `toast.error`, the dialog stays **open**, and the displayed tenant is untouched.

Test pattern to copy: `services/atlas-ui/src/components/features/templates/__tests__/TemplateReseedButton.test.tsx` — the whole file, including its `vi.mock` of the hooks module (not the service), the `toastError`/`toastSuccess` spies from a `vi.mock("sonner", …)`, the fixture builder, the `mockHooks({ data, mutateAsync })` helper, `beforeEach(() => vi.clearAllMocks())`, and `userEvent.setup()`.

**Interfaces:**
- Consumes: `useTenantConfiguration` (`useTenants.ts:229-243`), `useResetTenantConfiguration` (Task 10), `TenantResetSection` (Task 9), `createErrorFromUnknown` (`@/types/api/errors`).
- Produces:
  ```tsx
  interface TenantResetButtonProps {
    id: string | undefined;
    /** Omit for a whole-document reset. */
    sections?: TenantResetSection[];
    /** Human label for the scoped copy, e.g. "socket handlers and writers". */
    sectionLabel?: string;
  }
  export function TenantResetButton(props: TenantResetButtonProps): JSX.Element
  ```

- [ ] **Step 1: Write the failing test**

New file `src/components/features/tenants/__tests__/TenantResetButton.test.tsx`, mocking `@/lib/hooks/api/useTenants` with `{ useTenantConfiguration: vi.fn(), useResetTenantConfiguration: vi.fn() }` and `sonner` as the template test does.

Fixture: `tenantConfig(attrs)` returning `{ id: "t1", type: "tenants", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1, usesPin: true, ...attrs } }`.

| test | setup | assert |
|---|---|---|
| `is disabled with an explanatory tooltip when no baseline resolves` | `tenantConfig({ baselineTemplateId: "" })` | the trigger button has the `disabled` attribute; hovering it shows tooltip text mentioning that no template resolves for this tenant's region and version |
| `is disabled when baselineTemplateId is absent` | `tenantConfig({})` — an older backend | button disabled (`Boolean(undefined)` is `false`, same branch) |
| `is enabled when a baseline resolves` | `tenantConfig({ baselineTemplateId: "b1" })` | button not disabled |
| `Cancel renders before the destructive action` | enabled, dialog opened | within the dialog, the Cancel button's index in `within(dialog).getAllByRole("button")` is lower than the confirm button's — Enter must never fire the destructive action |
| `Cancel closes without calling the mutation` | open dialog, click Cancel | `mutateAsync` not called; dialog closes |
| `confirm calls the mutation with no sections for the whole document` | no `sections` prop | `mutateAsync` called with `{ id: "t1", sections: undefined }` |
| `confirm calls the mutation with the scoped sections` | `sections={["socket"]}` | `mutateAsync` called with `{ id: "t1", sections: ["socket"] }` |
| `a failure toasts the server detail and leaves the dialog open` | `mutateAsync` rejects with an error whose message is `"baseline is unprocessable"` | `toast.error` called with a string containing `"baseline is unprocessable"`; the dialog is still in the document |
| `a success toasts and closes` | `mutateAsync` resolves | `toast.success` called; the dialog is removed |
| `the whole-document dialog copy states all three facts` | no `sections` prop, dialog open | the dialog text mentions that UI edits will be lost, that the tenant's id, region, version, world configuration and diagnostics are unchanged, and that no game data is affected |
| `the scoped dialog copy names the section` | `sections={["socket"]} sectionLabel="socket handlers and writers"` | the dialog text contains `"socket handlers and writers"` |

- [ ] **Step 2: Run the test and confirm it fails**

```bash
npm test -- src/components/features/tenants/__tests__/TenantResetButton.test.tsx
```

Expected: FAIL — the module does not exist.

- [ ] **Step 3: Write the component**

Read `src/components/features/templates/TemplateReseedButton.tsx` in full and transpose it. The differences from that file, and only these:

- Props gain `sections?: TenantResetSection[]` and `sectionLabel?: string`.
- `const query = useTenantConfiguration(id ?? "")`, `const reset = useResetTenantConfiguration()`.
- `const hasBaseline = Boolean(query.data?.attributes.baselineTemplateId)`, `const disabled = !id || !query.data || !hasBaseline`.
- `await reset.mutateAsync({ id, sections })`.
- Trigger label: `"Reset to template"` when `sections` is absent, `` `Reset ${sectionLabel ?? "this section"}` `` when present.
- Disabled tooltip: *"No configuration template resolves for this tenant's region and version, so there is nothing to reset to."*
- Dialog title: `"Reset to template?"` / `` `Reset ${sectionLabel}?` ``.
- Dialog description, whole document: *"This replaces every comparable section of this tenant's configuration with its template's. Edits you have made through the UI to those sections will be lost. The tenant's id, region, version, world configuration and diagnostics are unchanged, and no game data — accounts, characters, inventories — is affected."*
- Dialog description, scoped: the same three facts with the first sentence naming `sectionLabel`.
- Confirm button label: `"Reset Tenant"` / `` `Reset ${sectionLabel}` ``, `"Resetting..."` with the `Loader2` spinner while pending.

Keep the `RotateCcw` icon, the `variant="outline" size="sm"` trigger, the always-mounted `Tooltip` root, the `Cancel`-first footer, `buttonVariants({ variant: "destructive" })` on the action, and the `createErrorFromUnknown` error path — all verbatim in structure.

- [ ] **Step 4: Run the tests and confirm they pass**

```bash
npm test -- src/components/features/tenants/__tests__/TenantResetButton.test.tsx
npx tsc --noEmit
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/components/features/tenants/TenantResetButton.tsx src/components/features/tenants/__tests__/TenantResetButton.test.tsx
git commit -m "feat(ui): add TenantResetButton"
```

---

## Task 12: List drift column and detail-header drift summary

**Files:**
- Modify: `services/atlas-ui/src/pages/tenants-columns.tsx` — add the drift column and a drift-source parameter to `getColumns`
- Modify: `services/atlas-ui/src/pages/TenantsPage.tsx:49,146-149` — call `useTenantConfigurations()` and pass the map in
- Modify: `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx:40-49` — drift summary + whole-document reset button in the header
- Test: `services/atlas-ui/src/pages/__tests__/tenants-columns.test.tsx` — extend
- Test: `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx` — extend

**Working directory:** `services/atlas-ui`

Patterns to copy: `services/atlas-ui/src/pages/templates-columns.tsx:58-82` (the `seedDrift` cell — read it and mirror it exactly, including the `!== true` guard and the `variant="secondary"` badge with its NFR-4 comment); `services/atlas-ui/src/pages/__tests__/templates-columns.test.tsx:7-17` (the `driftCell(attributes)` helper that calls `column.cell({ row } as never)` directly).

**Why the list needs a second source:** `TenantsPage.tsx:49` renders **registry** tenants from `useTenant()` (`@/context/tenant-context`), which are `TenantBasic` and carry no configuration attributes at all. `templates-columns.tsx`'s pattern works only because `TemplatesPage` sources `Template[]` end to end. The page therefore calls `useTenantConfigurations()` (already exists, `useTenants.ts:212`) and passes a `Map<string, TenantConfig>` into `getColumns`. A registry tenant with no configuration row renders nothing — that gap is visible elsewhere already and is not this task's to solve.

**Interfaces:**
- Consumes: `useTenantConfigurations` (`useTenants.ts:212-224`), `useTenantConfiguration` (`:229-243`), `TenantResetButton` (Task 11), `TenantConfig`.
- Produces: `getColumns({ onDelete?, onRename?, configs? }: ColumnProps)` where `configs?: Map<string, TenantConfig>`.

- [ ] **Step 1: Write the failing tests**

Extend `src/pages/__tests__/tenants-columns.test.tsx` with a `driftCell` helper copied from the templates version (`templates-columns.test.tsx:7-17`), adapted to pass a `configs` map into `getColumns` and to look up the `templateDrift` column by `id`.

| test | configs map entry for the row | assert |
|---|---|---|
| `renders a badge when the tenant has drifted` | `{ attributes: { templateDrift: true, sectionDrift: { socket: true, properties: false, characters: false, npcs: false, cashShop: false, mapleLife: false } } }` | the cell renders text "Differs from template" |
| `renders nothing when the tenant has not drifted` | `{ attributes: { templateDrift: false, … } }` | the cell renders nothing |
| `renders nothing when templateDrift is absent` | `{ attributes: {} }` — older backend | the cell renders nothing; strictly `=== true`, never truthy-ish |
| `renders nothing when the tenant has no configuration row` | empty map | the cell renders nothing |

Extend `src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx` (it already mocks `useTenantConfiguration` and `ConfigExportButton`; add a mock for `TenantResetButton` or let it render with the hooks mocked):

| test | `useTenantConfiguration` returns | assert |
|---|---|---|
| `names the diverging sections in the header` | `templateDrift: true`, `sectionDrift: { properties: false, socket: true, characters: true, npcs: false, cashShop: false, mapleLife: false }` | the header text names `socket` and `characters` and does not name `npcs` |
| `renders no drift summary when nothing has drifted` | `templateDrift: false`, all `sectionDrift` false | no drift text in the header |
| `renders no drift summary when templateDrift is absent` | `attributes: {}` | no drift text |
| `mounts the whole-document reset button` | any | a button whose accessible name matches `/reset to template/i` is in the header |

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
npm test -- src/pages/__tests__/tenants-columns.test.tsx src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx
```

Expected: FAIL — no `templateDrift` column, no drift text, no reset button.

- [ ] **Step 3: Add the column**

In `src/pages/tenants-columns.tsx`, extend `ColumnProps` (lines 13-16) with `configs?: Map<string, TenantConfig>` and add the column after `attributes.minorVersion`, before `actions`:

```tsx
    {
      id: "templateDrift",
      header: "Template",
      cell: ({ row }) => {
        // Strictly `=== true`: the attribute is optional (an older API,
        // or a tenant with no configuration row at all) and `undefined`
        // must read as "no badge", never as truthy-ish. FR-1.3 falls out
        // of the server contract — templateDrift is false whenever
        // baselineTemplateId is empty.
        const config = configs?.get(row.original.id);
        if (config?.attributes.templateDrift !== true) return null;
        return (
          <Tooltip>
            <TooltipTrigger asChild>
              {/*
                `secondary`, not `destructive`: drift is advisory and
                relative to a template row that is itself mutable
                (NFR-4). A tenant an operator edited on purpose is not in
                an error state.
              */}
              <Badge variant="secondary">Differs from template</Badge>
            </TooltipTrigger>
            <TooltipContent>
              Differs from the template this tenant derives from
            </TooltipContent>
          </Tooltip>
        );
      },
    },
```

Add the `Badge` and `Tooltip*` imports, copying them from `templates-columns.tsx`'s import block.

- [ ] **Step 4: Wire the page**

In `src/pages/TenantsPage.tsx`:

```tsx
  // The registry tenants this page lists carry no configuration
  // attributes, so drift needs a second source. useTenantConfigurations
  // is already cached elsewhere in the app; a tenant with no
  // configuration row simply renders no badge.
  const { data: configs } = useTenantConfigurations();
  const configsById = useMemo(
    () => new Map((configs ?? []).map((c) => [c.id, c])),
    [configs],
  );
```

and pass it at the `getColumns` call (line 146-149):

```tsx
  const columns = getColumns({
    onDelete: openDeleteDialog,
    onRename: openRenameDialog,
    configs: configsById,
  });
```

Add `useMemo` and `useTenantConfigurations` imports. If `getColumns` is called inside a `useMemo` already, add `configsById` to its dependency array.

- [ ] **Step 5: Add the header summary and reset button**

In `src/components/features/tenants/TenantDetailLayout.tsx`, inside the existing `div.flex.items-start.justify-between` (lines 40-49), beside `ConfigExportButton`:

```tsx
        <div className="flex items-center gap-2">
          {driftedSections.length > 0 && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="secondary">
                  Differs from template: {driftedSections.join(", ")}
                </Badge>
              </TooltipTrigger>
              <TooltipContent>
                These sections diverge from the template this tenant derives
                from. Drift is advisory — a template edit makes its tenants
                report drift with no tenant-side change.
              </TooltipContent>
            </Tooltip>
          )}
          <ConfigExportButton kind="tenant" id={id} />
          <TenantResetButton id={id} />
        </div>
```

with, above the return:

```tsx
  // Strictly `=== true` per key, so an older backend that omits
  // sectionDrift renders nothing rather than six false positives.
  const sectionDrift = tenantQuery.data?.attributes.sectionDrift;
  const driftedSections =
    tenantQuery.data?.attributes.templateDrift === true && sectionDrift
      ? Object.keys(sectionDrift).filter((k) => sectionDrift[k] === true)
      : [];
```

- [ ] **Step 6: Run the tests and confirm they pass**

```bash
npm test -- src/pages/__tests__/tenants-columns.test.tsx src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx
npx tsc --noEmit
npm run lint
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add src/pages/tenants-columns.tsx src/pages/TenantsPage.tsx src/components/features/tenants/TenantDetailLayout.tsx src/pages/__tests__/tenants-columns.test.tsx src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx
git commit -m "feat(ui): show tenant template drift on the list and the detail header"
```

---

## Task 13: Per-section reset buttons on the section pages

**Files:**
- Modify: `services/atlas-ui/src/pages/tenants-properties-form.tsx:185-187` — `properties`
- Modify: `services/atlas-ui/src/pages/TenantsHandlersPage.tsx` — `socket`
- Modify: `services/atlas-ui/src/pages/TenantsWritersPage.tsx` — `socket`
- Modify: `services/atlas-ui/src/pages/TenantsCharacterTemplatesPage.tsx:47-51` — `characters`
- Modify: `services/atlas-ui/src/pages/TenantsCharacterPresetsPage.tsx:63-65` — `characters`
- Modify: `services/atlas-ui/src/pages/TenantsMapleLifePage.tsx:68-70` — `mapleLife`
- Test: `services/atlas-ui/src/pages/__tests__/TenantsSectionReset.test.tsx` — new file

**Working directory:** `services/atlas-ui`

Patterns to copy: `services/atlas-ui/src/pages/TenantsCharacterTemplatesPage.tsx:47-51` (the `<TenantDetailLayout>` wrapper each section page returns — the button goes inside it, above the editor).

**Mount decision:** the button goes into each **page's** JSX inside `<TenantDetailLayout>`, in a right-aligned row above the editor. It does **not** go into `DetailActionBar`: that is a single shared instance rendered once by `TenantDetailLayout:58`, takes no children, and is populated by `useRegisterDetailActionBar({ dirty, isSaving, onSave, onDiscard })` from inside three editor components. Adding a reset trigger there would mean editing all three editors for no gain and coupling a destructive action to the save bar's dirty state.

**Sections with no page:** there is no tenant `cashShop` or `npcs` editing page (`src/App.tsx:441-480` routes only handlers, worlds, writers, properties, character/templates, character/presets, character/maple-life, mts-config, diagnostics). Those two sections stay resettable via the API and via the header's whole-document reset; creating editing pages for them is outside this task.

**Handlers and Writers are two pages over one section:** both mount `sections={["socket"]}` and both use the label `"socket handlers and writers"`, so an operator resetting from the Writers page is not surprised to find handlers reverted too. The same holds for the two `characters` pages.

**Interfaces:**
- Consumes: `TenantResetButton` (Task 11).
- Produces: nothing.

- [ ] **Step 1: Write the failing test**

New file `src/pages/__tests__/TenantsSectionReset.test.tsx`. Mock `@/components/features/tenants/TenantResetButton` with a spy component that records its props and renders a `data-testid="tenant-reset-button"` element carrying `data-sections` and `data-label`. Mock `useTenantConfiguration`/`useUpdateTenantConfiguration` and the editors so each page renders without a real query client, following the mocking shape in `src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx:8-18`. Render each page inside `MemoryRouter` + `Routes` at its route path, as that test does.

| test | page | assert on the spy's props |
|---|---|---|
| `properties page resets properties` | `TenantsPropertiesPage` | `sections` deep-equals `["properties"]`; `sectionLabel` is `"global properties"` |
| `handlers page resets socket` | `TenantsHandlersPage` | `sections` deep-equals `["socket"]`; `sectionLabel` is `"socket handlers and writers"` |
| `writers page resets socket` | `TenantsWritersPage` | `sections` deep-equals `["socket"]`; `sectionLabel` is `"socket handlers and writers"` |
| `character templates page resets characters` | `TenantsCharacterTemplatesPage` | `sections` deep-equals `["characters"]`; `sectionLabel` is `"character templates and presets"` |
| `character presets page resets characters` | `TenantsCharacterPresetsPage` | `sections` deep-equals `["characters"]`; `sectionLabel` is `"character templates and presets"` |
| `maple life page resets mapleLife` | `TenantsMapleLifePage` | `sections` deep-equals `["mapleLife"]`; `sectionLabel` is `"Maple Life configuration"` |

Add one negative test: `maple life page renders no reset button on an unsupported client` — `TenantsMapleLifePage`'s early-return branch (its lines 20-33) renders explanatory text and no configuration section, so `queryByTestId("tenant-reset-button")` is null there.

- [ ] **Step 2: Run the test and confirm it fails**

```bash
npm test -- src/pages/__tests__/TenantsSectionReset.test.tsx
```

Expected: FAIL — no reset button on any page.

- [ ] **Step 3: Mount the button on each page**

For the five pages that return `<TenantDetailLayout>…</TenantDetailLayout>`, insert immediately inside the layout, above the editor:

```tsx
      <div className="flex justify-end">
        <TenantResetButton
          id={id}
          sections={["socket"]}
          sectionLabel="socket handlers and writers"
        />
      </div>
```

substituting per the table in Step 1. Add the `TenantResetButton` import to each.

`TenantsHandlersPage.tsx` and `TenantsWritersPage.tsx` are 10-line wrappers around `<DefinitionGridPage kind=… scope="tenant" />`; read them first. If wrapping them in a fragment with the button above the grid disturbs the grid's flex layout, place the button instead in `DefinitionGridPage`'s existing right-aligned row at `src/components/features/socket/DefinitionGridPage.tsx:252-262`, gated on `scope === "tenant"` only (not on `ancestor`), taking the tenant id from the same source that row already uses — and adjust the test to render `DefinitionGridPage` accordingly.

For `tenants-properties-form.tsx`, the button goes into the existing right-aligned row at lines 185-187, before `<Button type="submit">Save</Button>`:

```tsx
        <div className="flex flex-row gap-2 justify-end">
          <TenantResetButton
            id={id}
            sections={["properties"]}
            sectionLabel="global properties"
          />
          <Button type="submit">Save</Button>
        </div>
```

Read the file to find how it reaches the tenant id — if it is not already in scope, thread it in from `TenantsPropertiesPage.tsx` as a prop rather than calling `useParams` a second time.

- [ ] **Step 4: Run the tests and confirm they pass**

```bash
npm test
npx tsc --noEmit
npm run lint
npm run build
```

Expected: PASS across the whole frontend suite.

- [ ] **Step 5: Commit**

```bash
git add src/pages/tenants-properties-form.tsx src/pages/TenantsHandlersPage.tsx src/pages/TenantsWritersPage.tsx src/pages/TenantsCharacterTemplatesPage.tsx src/pages/TenantsCharacterPresetsPage.tsx src/pages/TenantsMapleLifePage.tsx src/pages/__tests__/TenantsSectionReset.test.tsx
git commit -m "feat(ui): add per-section reset actions to the tenant section pages"
```

---

## Task 14: Full verification gate

**Files:**
- Read-only: `tools/verify.sh`

**Interfaces:**
- Consumes: every prior task.
- Produces: a green gate.

- [ ] **Step 1: Run the flagless gate**

From the worktree root:

```bash
tools/verify.sh
```

Per CLAUDE.md, the **flagless** run must exit 0 before this branch is "done". `--quick` / `--no-docker` also exit 0 but skip the bake and `-race`, and do not count.

- [ ] **Step 2: Fix any failure and re-run**

If the gate fails, read the first failing block only, fix it in the task that owns it, and re-run the flagless gate. Do not proceed on a partial or flagged run.

- [ ] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix(task-289): address verification gate findings"
```

(Skip if the gate was green with no changes.)
