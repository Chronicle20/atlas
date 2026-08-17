# Backend Audit — commit ccde9408f (world-transfer client-crash fix)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Commit:** ccde9408f43f8741510bd4ac4e0f004ac0aed8f7
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-16
- **Build:** PASS
- **Tests:** PASS (`atlas-channel/world`, `atlas-channel/socket/handler`, `go test -count=1`)
- **Overall:** NEEDS-WORK

## Scope

Files changed by ccde9408f:
- `services/atlas-channel/atlas.com/channel/world/processor.go`
- `services/atlas-channel/atlas.com/channel/world/requests.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible_test.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible_test.go`

Also read (not scored, load-bearing for correctness): `libs/atlas-rest/requests/paged.go` (`DrainProvider`), `services/atlas-channel/atlas.com/channel/world/{model,rest,builder}.go`, `libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result.go` (wire layout the test decodes against), `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go` (POP_UP/PINK_TEXT resolution).

## Build & Test Results

```
$ go build ./...          # atlas-channel/atlas.com/channel — clean, no output
$ go test ./world/... ./socket/handler/... -count=1
ok  	atlas-channel/world	0.004s
ok  	atlas-channel/socket/handler	0.734s
```

## Findings

### EXT-02 — FAIL: no httptest-backed integration test for the new `AllProvider`/`GetAll`

`world/processor.go:47-53` adds `AllProvider()`/`GetAll()`, a new external HTTP client path through `requests.DrainProvider[RestModel, Model]` (`world/requests.go:28-30` builds the bare `worldsUrl()` it pages against). Per the External HTTP Client Checklist this triggers EXT-02: an `httptest.NewServer`-backed test serving a representative fixture (including pagination `meta`) and asserting `GetAll()` returns a populated `[]Model`.

No such test exists. `services/atlas-channel/atlas.com/channel/world/` contains only `builder_test.go` (confirmed via `find services/atlas-channel/atlas.com/channel/world -name '*_test.go'` → single hit). The only place `GetAll`/`AllProvider` is exercised is through `checkPossibleWorldsFunc`, which the handler tests stub out entirely (`cash_shop_check_name_change_possible_test.go:129-134`: `checkPossibleWorldsFunc = func(...) { return env.worlds, env.worldsErr }`). That is exactly the case EXT-02 calls out as insufficient — a seam/mock bypasses the unmarshal path — so the real integration (page-param construction, JSON:API `Extract` decode, `DrainProvider`'s page-2..last loop and empty-page/no-envelope termination) is never exercised by this change. `RestModel` already carries the required `SetToManyReferenceIDs`/`GetReferencedIDs` stubs (`world/rest.go:44,56-68`) so EXT-01 is satisfied, but that only prevents an api2go crash — it doesn't substitute for the missing round-trip test.

**This is the one blocking finding.** Everything else below passes or is a non-blocking note.

### DOM-21 — PASS: no atlas-constants duplication

The change reuses `world.Id` (`github.com/Chronicle20/atlas/libs/atlas-constants/world`) throughout — `world/processor.go:8,14,16`, `handler/cash_shop_check_transfer_world_possible.go:13`. `transferWorldNameList` (`cash_shop_check_transfer_world_possible.go:172-194`) indexes a `[]string` by `int(w.Id())`; no new id/classification type, enum, or numeric constant is declared. `grep -rn "AllProvider\|GetAll" libs/atlas-constants/` returns nothing, so there is no existing shared equivalent being reinvented.

### Package-level mutable var test seam (`checkPossibleWorldsFunc`) — WARN, not a violation

`cash_shop_check_transfer_world_possible.go:33-35` adds `var checkPossibleWorldsFunc = func(l logrus.FieldLogger, ctx context.Context) ([]channelworld.Model, error) { return channelworld.NewProcessor(l, ctx).GetAll() }`, swapped in tests via `newCheckPossibleHandlerEnv` (`cash_shop_check_name_change_possible_test.go:129-134`) with `t.Cleanup` restoring the original.

This is judged **consistent with established package convention, not a new violation**: `cash_shop_check_name_change_possible.go:24` (`checkPossibleAccountGetByIdFunc`) and `:28` (`checkPossibleRecordPicAttemptFunc`) already used the identical package-var-swap-with-cleanup pattern *before* this commit — the new seam is the third instance of a pre-existing idiom in the same file, not an invented pattern this diff introduces.

It does deviate from the canonical DI pattern the guidelines document elsewhere (`testing-guide.md` "Mock Implementation Pattern" — interface + `ProcessorMock` struct with injectable `*Func` fields in a `mock/` package). No checklist item or anti-pattern entry explicitly forbids the package-var seam, so this is **not scored as a FAIL**, but it is worth naming: three of these vars now exist in one file with no interface behind them, which means nothing enforces that a swapped function's signature still matches what a real caller would produce, and the pattern is not safe under `t.Parallel()` (not currently used in this package — verified via `grep -rn "t.Parallel" services/atlas-channel/atlas.com/channel/socket/handler/*.go`, zero hits — so no live race today, but the pattern would break silently if parallelism were added later without also switching to per-test injection).

### Error handling / logging on the new REST path — PASS

`cash_shop_check_transfer_world_possible.go:128-133`: `wErr != nil` is logged with `l.WithError(wErr).Errorf(...)` and characterId context, then refuses via `CheckTransferWorldPossibleUnknownError` rather than propagating a bare 500-equivalent or silently degrading to ALLOWED. `:135-138` separately handles the "list produced but empty" case with its own `Errorf` and the same refusal — this is not silent degradation (DOM-28-style requirement, satisfied even though DOM-28 targets decorators specifically): a fetch failure or an empty result is loudly logged and drives an explicit refusal branch, never a quiet fallback to the crash-causing empty-list ALLOWED. `transferWorldNameList` itself additionally warns per missing id (`:188-192`) rather than swallowing the gap.

### World-list fetch: handler vs. processor placement — PASS

The actual fetch (`channelworld.NewProcessor(l, ctx).GetAll()`, `cash_shop_check_transfer_world_possible.go:34`) is correctly delegated to the `world` package's processor — the handler does not call `world`'s provider/request functions directly, satisfying the resource→processor layering rule (`anti-patterns.md` "Handlers Calling Providers Directly"; `file-responsibilities.md` `resource.go` "Delegate ALL business logic to processors"). Note this package is a socket packet handler (`socket/handler`), not a JSON:API `resource.go` domain package, so the DOM-13/14/15 checks apply by analogy rather than literally — no `db.Create/Save/Delete` or provider calls appear in the handler file (`grep` confirms none).

`transferWorldNameList`'s id-indexing/gap-handling logic (`:172-194`) stays in the handler file rather than moving into `world/processor.go`. This is judged appropriate, not a violation: the rendering rule (index-by-id, blank-fill-for-gaps) is a property of *this one wire packet's* client-side combo-box behavior (documented via the IDA chain at `:153-171`), not a general `world` domain operation — folding it into the shared `world` processor would leak cash-shop/packet-specific semantics into a package every other consumer of `world.Model` shares.

### Test quality — PASS

`cash_shop_check_transfer_world_possible_test.go` decodes the actual wire bytes rather than only checking whether *something* was announced. `decodeAllowedWorldNames` (`:180-209`) reads the ALLOWED body starting at offset 10, matching the real codec layout verified against `libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result.go` (characterId(4) + result(1) + birthDate(4) + hasWorldList(1) = 10-byte header before the count/entries). `TestTransferWorldPossibleAllowedCarriesTheWorldNameList` (`:227-246`) asserts the decoded names against a concrete `["Scania","Bera"]`, not just non-empty. `TestTransferWorldNameListIsIndexedByWorldId` (`:254-279`) is table-driven via `t.Run` subtests and directly exercises `transferWorldNameList` for out-of-order input, gapped ids, and the empty case — all against `reflect.DeepEqual`, not loose length checks. `TestTransferWorldPossibleWorldListFailureRefusesRatherThanCrashing` (`:283-305`) covers both the lookup-error and empty-list refusal arms and asserts the exact `0x2F` UNKNOWN_ERROR byte. `TestWorldTransferStorageWarningUsesPopUpNotPinkText` (`:313-322`) decodes the actual mode byte (`storageWarningModeByte`, `cash_shop_check_name_change_possible_test.go` — helper reads `body[0]`) rather than only checking "a warning was announced," which is exactly what distinguishes this from a rename-only test. These assertions reach real behavior, not just call-happened booleans.

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in processor.go | PASS | `AllProvider`/`GetAll` added to `world/processor.go:43-53`, alongside the existing `ByIdModelProvider`/`GetById`. No `ProcessorImpl` method appears outside `processor.go`. |
| FILE-03 | Cross-service request funcs in requests.go | PASS | `worldsUrl()`, `WorldsList` const added to `world/requests.go:13-16,28-30`. |
| FILE-06 | No package-named catch-all file | PASS | Neither `world/` nor the handler package gained a `<pkg>.go`/collapsed file; new code landed in the file the responsibility table designates. |

## External HTTP Client Checklist (`world` package → atlas-world)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API target implements relationship interfaces | PASS | `world/rest.go:44` `SetToManyReferenceIDs`, `:56-68` `GetReferencedIDs`/`GetReferencedStructs` — pre-existing, unaffected by this diff, still present. |
| EXT-02 | httptest-backed integration test exists | **FAIL** | No test file under `world/` exercises `AllProvider`/`GetAll`/`DrainProvider` against a real HTTP fixture; only `builder_test.go` exists in the package. Handler-level tests stub the seam instead. |
| EXT-03 | Errors distinguish 404 from other failures | N/A | `GetAll` is a list fetch with no single-resource "not found" domain state; the handler treats any error (transport, decode, or otherwise) as an undifferentiated refusal (`UNKNOWN_ERROR`), which is the correct conservative behavior for a client-crash-prevention gate, not a hidden-deploy-bug case EXT-03 targets. |
| EXT-04 | Service URL not hardcoded | PASS | `worldsUrl()` = `getBaseRequest() + WorldsList` = `requests.RootUrl("WORLDS") + "worlds"` (`world/requests.go:18-20,28-30`). |

## Domain Checklist Results (relevant subset — `world` package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts FieldLogger | PASS | `NewProcessor(l logrus.FieldLogger, ctx context.Context)` — `world/processor.go:25` (pre-existing, unchanged by diff, still correct). |
| DOM-11 | Providers use lazy evaluation | PASS | `AllProvider()` returns a `model.Provider[[]Model]` (curried, not eagerly executed) — `world/processor.go:47-49`; `GetAll()` (`:51-53`) is the only place it is invoked. |
| DOM-21 | No atlas-constants duplication | PASS | See Findings above. |

## Not evaluable from the diff

- Whether `atlas-world`'s `/api/worlds` endpoint actually emits the `meta.page.last` envelope `DrainProvider` depends on to terminate its page loop correctly (`libs/atlas-rest/requests/paged.go:128-134`) — would need to read atlas-world's `resource.go`/pagination wiring, out of this diff's scope.
- Whether any other atlas-channel consumer already held a `world.Processor` mock that needed updating for the two new interface methods — grepped for `world.Processor` usage outside the `world` package itself (none found) and confirmed via `go build ./...` (clean), which is sufficient to rule this out without reading further.

## Summary

### Blocking (must fix)
- EXT-02: `world` package's new `AllProvider`/`GetAll` external HTTP client path has no httptest-backed integration test — the handler-level seam stub does not substitute for it.

### Non-Blocking (should fix)
- The `checkPossibleWorldsFunc` package-var test seam is consistent with two pre-existing siblings in the same file and is not a guideline violation, but all three seams sit outside the documented Mock/interface DI pattern (`testing-guide.md`) and are not `t.Parallel()`-safe; worth standardizing if this package's tests are ever parallelized.
