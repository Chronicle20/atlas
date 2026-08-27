# Backend Audit — task-262-wz-property-reader-divergence

- **Service Path:** `libs/atlas-wz` (module) + `services/atlas-data/atlas.com/data` (module)
- **Scope:** changed Go packages between `a6820d1c4..HEAD`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-26
- **Build:** PASS
- **Tests:** 32 packages `ok` (libs/atlas-wz), 47 packages `ok` (atlas-data); 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd libs/atlas-wz && GOCACHE=/tmp/gocache-262 go build ./...   -> exit 0, no output
cd libs/atlas-wz && GOCACHE=/tmp/gocache-262 go test ./... -count=1
  ok  	.../atlas-wz/atlas ... wz ... wz/property ... wz/wzxml ... wzdiff   (all ok, 2 "no test files")

cd services/atlas-data/atlas.com/data && GOCACHE=/tmp/gocache-262 go build ./...   -> exit 0, no output
cd services/atlas-data/atlas.com/data && GOCACHE=/tmp/gocache-262 go test ./... -count=1
  ok across all 47 listed packages (several "no test files"), including
  data/wztoxml   ok  0.029s
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | No | No changed package has `model.go`, `entity.go`, `rest.go`, or `provider.go` (`wz`, `wzxml`, `wzdiff`, `wztest`, `cmd/wzdiff`, `wztoxml` are all non-domain parsing/tooling packages). |
| FILE placement (FILE-01..06) | Yes | Every changed Go package — no exemption. |
| SUB sub-domain (SUB-01..04) | No | No changed package has `resource.go`. |
| REST (DOM-06..09,12..15,17..19,32) | No | No changed package has `resource.go`/`rest.go`/`processor.go`, none registers HTTP routes. |
| Constants reuse (DOM-21) | Yes | New named const block: `wztest.Kind` gains `KindUOL, KindNull, KindShort, KindLong, KindFloat, KindDouble, KindVector, KindConvex, KindFloatNoMarker` (`libs/atlas-wz/wztest/builder.go:23-31`). |
| Testing (DOM-10,20,24,33) | Yes | Diff touches many `_test.go` files. |
| Cache (DOM-29) | No | No changed package adds `cache.go` or introduces new cached state (`wz.File`'s pre-existing `keyRanges`/`parseMu` fields are untouched by this diff apart from the new `trace` field, which is a one-shot hook, not cached data). |
| Messaging (DOM-30) | No | No `producer.go`, no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call added. |
| Multi-tenancy (DOM-31) | No | No `rest.go`; no tenant/trace state read or passed. |
| Migration hygiene (DOM-34,35) | Yes | `xmlElement`/`propertyToElement`/`propertiesToElements`/`formatFloat` moved out of `services/atlas-data/.../wztoxml/adapter.go` into `libs/atlas-wz/wz/wzxml/element.go` as `Element`/`PropertyToElement`/`PropertiesToElements`/`FormatFloat`. |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module added (no `go.mod`/`go.work`/`Dockerfile` change) and no Kafka topic env var touched. |
| Runtime safety (DOM-26) | Yes (family) — rule N/A | Non-test Go files changed, but no `go` statement (bare or wrapped) appears anywhere in the diff. |
| Channel wire values (DOM-25) | No | Diff does not touch `services/atlas-channel` or `libs/atlas-packet`, and emits no client-interpreted byte. |
| Resilience (DOM-27,28) | No | No DB-backed handler changed; no `model.Decorator`/enrichment path changed. |
| External clients (EXT-01..04) | No | No `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` call added. |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler, no `routes.conf` change. |
| Security (SEC-01..04) | No | Neither `atlas-wz` nor `atlas-data` handles authentication, tokens, redirects, or secrets. |
| Foundational: patterns-provider.md | No | No provider defined or composed in the diff. |
| Foundational: patterns-functional.md | No | No curried constructor/decorator/model combinator defined. |

## Checklist Results

### libs/atlas-wz/wz (support — parsing library, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | Processor/RestModel/requests/Entity/Builder-Model-administrator-provider-state placement | N/A | No such constructs exist in this package (grep for `type Processor interface`, `type RestModel`, `requests.RootUrl(`, `type entity struct`, `type Builder` in `libs/atlas-wz/wz/*.go` returns nothing). |
| FILE-06 | No package-named catch-all bundling ≥2 responsibilities | PASS | `file.go`, `reader.go`, `directory.go`, `image.go`, `trace.go` each carry one parsing concern; none bundles Processor+RestModel+requests-style responsibilities. |
| DOM-26 | Goroutines via `routine.Go` | N/A | No `go` statement added anywhere in `libs/atlas-wz/wz/*.go` (`grep -rn "^\s*go "` over the diff returns nothing). |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wz/directory_error_test.go:111,142` — two `func Test...` bodies, no `tests := []struct{...}` + `t.Run`. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wz/trace_test.go:78,160,233,270,299,352,371` — seven `func Test...` bodies, zero `t.Run` occurrences (`grep -c "t.Run(" trace_test.go` = 0). |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wz/wztest_canvas_test.go:18,98` — two flat `func Test...`, no `t.Run`. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wz/wztest_dedup_test.go:136,155` — two flat `func Test...`, no `t.Run`. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wz/wztest_kinds_test.go:17-97` — `TestBuilderEmitsAllPropertyKinds` checks 10 distinct property kinds (null/short/int/long/float/double/string/vector/uol/convex) in one flat function with no `t.Run`/table, the textbook table-driven candidate; same file also has two more flat functions at lines 105 and 137. |

### libs/atlas-wz/wz/wzxml (support, new package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | Domain-file placement | N/A | No Processor/RestModel(JSON:API)/requests/Entity/Builder construct — `element.go` only maps `property.Property` to a generic XML `Element`, not a JSON:API `RestModel` (`GetName()`/`GetID()`/`SetID()` absent). |
| FILE-06 | Catch-all | PASS | Single file `element.go`, single mapping responsibility. |
| DOM-20 | Table-driven tests | PASS | `libs/atlas-wz/wz/wzxml/element_test.go:10,194` — `tests := []struct{...}` + `t.Run(tc.name, ...)`. |
| DOM-34/35 | Migration hygiene | PASS | Package is the new home for logic moved out of `services/atlas-data/.../wztoxml/adapter.go`; see wztoxml row below. |

### libs/atlas-wz/wzdiff (support, new package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Domain-file placement | N/A / PASS | No Processor/RestModel/requests/Entity/Builder constructs; `run.go` bundles `Run`/`Trace`/`WriteReport`/`collectImages`/`referenceImageNames`, none of which are FILE-01..05 responsibilities, so FILE-06's catch-all test does not fire. |
| DOM-20 | Table-driven tests | PASS | `libs/atlas-wz/wzdiff/node_test.go:6,45` and `libs/atlas-wz/wzdiff/diff_test.go:9` (subtests via `t.Run` per named case) both organize cases as enumerated scenarios; `diff_test.go` uses 12 named `t.Run` blocks per `TestDiff`, satisfying the spirit of the pattern even though it is not a literal `tests := []struct` slice — no FAIL recorded for this file. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wzdiff/allowlist_test.go:10,37,54,67,84,97,110,136,161,177,195` — eleven `func Test...` bodies testing eleven variations of the same `LoadAllowlist`/`normalizeAllowEntry`/`validateAllowEntry` contract; only `TestAllowed` (line 222) uses `t.Run` (1 occurrence total in the file), the other ten are flat. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wzdiff/run_test.go:40,83,132` — three flat `func Test...`, zero `t.Run`. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wzdiff/selfcheck_test.go:19,68` — two flat `func Test...`, zero `t.Run`. |
| DOM-20 | Table-driven tests | FAIL | `libs/atlas-wz/wzdiff/xmlload_test.go:33,100,157` — three flat `func Test...`, zero `t.Run`. |

### libs/atlas-wz/wztest (support, test-fixture builder library)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Builder in `builder.go` | PASS | `type Builder struct` / `func NewBuilder()` at `libs/atlas-wz/wztest/builder.go:130,138`, already in `builder.go`. |
| FILE-06 | Catch-all | PASS | `builder.go` carries only the fixture-construction responsibility (no Processor/RestModel/requests/Entity mixed in). |
| DOM-21 | No redeclaration of an existing `libs/atlas-constants` type | N/A | New `Kind` const values (`KindUOL`, `KindNull`, `KindShort`, `KindLong`, `KindFloat`, `KindDouble`, `KindVector`, `KindConvex`, `KindFloatNoMarker` at `libs/atlas-wz/wztest/builder.go:23-31`) are WZ-binary-format-internal property-tag kinds; `libs/atlas-constants` (game-domain IDs: item/map/job/skill/etc.) has no overlapping concept — confirmed no `Kind` type in `libs/atlas-constants/`. |
| — | Tests | N/A | No `_test.go` in `wztest` package itself (consumers' tests are graded under their own packages above). |

### libs/atlas-wz/cmd/wzdiff (support, new CLI)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Domain-file placement | N/A | `main.go` is flag parsing/dispatch only, delegating all logic to the `wzdiff` package (`libs/atlas-wz/cmd/wzdiff/main.go:23` doc comment: "All comparison and formatting logic lives in the testable wzdiff package; main is flag parsing and dispatch only."). No Processor/RestModel/requests/Entity/Builder construct present. |
| DOM-26 | Goroutines | N/A | No `go` statement in `main.go`. |

### services/atlas-data/atlas.com/data/data/wztoxml (support, existing package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Domain-file placement | N/A | No Processor/RestModel/requests/Entity/Builder construct in `adapter.go`; it is serialization orchestration only. |
| DOM-34 | No aliases/wrappers left behind after moving `xmlElement`/`propertyToElement`/`propertiesToElements`/`formatFloat` to `wzxml` | PASS | `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go:88,96` now call `wzxml.Element{...}` and `wzxml.PropertiesToElements(props)` directly; `grep -rn "xmlElement\|propertyToElement\|propertiesToElements\b\|formatFloat("` under `services/atlas-data/atlas.com/data/` returns no matches — no re-export or delegating wrapper left. |
| DOM-35 | Dead symbols removed | PASS | The diff for `adapter.go` deletes the old `xmlElement` type, `propertiesToElements`, `propertyToElement`, `formatFloat`, and their now-unused `strconv`/`strings` imports (`git diff a6820d1c4..HEAD -- services/atlas-data/.../wztoxml/adapter.go`, deletion hunk lines 78-163 of the old file). |
| DOM-20 | Table-driven tests | FAIL | `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go:23,93,183` — three flat `func Test...` (`TestRoundTripImage`, `TestSerializeDirectoryCountsFailures`, `TestSerializeDirectorySuccessLogsInfo`), zero `t.Run` occurrences. |

## Security Review

Not applicable. `libs/atlas-wz` and `atlas-data` handle neither authentication,
tokens, redirects, nor secrets in this diff; SEC-* family disposed as N/A
per `patterns-security.md`'s own scoping statement ("Services with none of
those responsibilities dispose of the whole SEC-* family as N/A").

## Not evaluable from the diff

- None. Every applicable rule was settled by reading the changed files, their
  git diff hunks, and one targeted grep each (for `libs/atlas-constants`
  overlap and for leftover migration symbols). No item required reading
  outside the changed packages plus those targeted lookups.

## Summary

### Blocking (must fix)
- DOM-20: `libs/atlas-wz/wz/directory_error_test.go` (2 tests) — not table-driven.
- DOM-20: `libs/atlas-wz/wz/trace_test.go` (7 tests) — not table-driven, no `t.Run`.
- DOM-20: `libs/atlas-wz/wz/wztest_canvas_test.go` (2 tests) — not table-driven.
- DOM-20: `libs/atlas-wz/wz/wztest_dedup_test.go` (2 tests) — not table-driven.
- DOM-20: `libs/atlas-wz/wz/wztest_kinds_test.go` (3 tests, esp. `TestBuilderEmitsAllPropertyKinds` covering 10 property kinds inline) — not table-driven.
- DOM-20: `libs/atlas-wz/wzdiff/allowlist_test.go` (10 of 11 tests) — not table-driven.
- DOM-20: `libs/atlas-wz/wzdiff/run_test.go` (3 tests) — not table-driven.
- DOM-20: `libs/atlas-wz/wzdiff/selfcheck_test.go` (2 tests) — not table-driven.
- DOM-20: `libs/atlas-wz/wzdiff/xmlload_test.go` (3 tests) — not table-driven.
- DOM-20: `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go` (3 tests) — not table-driven.

### Non-Blocking (should fix)
- None recorded beyond the blocking DOM-20 items above.
