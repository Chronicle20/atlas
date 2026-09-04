# Backend Audit — task-295-mob-escort-full-path-fix

- **Service Path:** libs/atlas-packet, services/atlas-channel (changed-package scope)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-04
- **Build:** PASS
- **Tests:** 3502 passed, 0 failed (libs/atlas-packet: 3014; atlas-channel/socket: 488)
- **Overall:** PASS

## Build & Test Results

```
cd libs/atlas-packet && go build ./...        # exit 0, no output
cd libs/atlas-packet && go test ./... -count=1 # all packages "ok", including
                                                 # monster/clientbound 0.087s
cd services/atlas-channel/atlas.com/channel && go build ./...  # exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./socket/... -count=1
  ok  atlas-channel/socket           0.031s
  ok  atlas-channel/socket/handler   2.592s
  ok  atlas-channel/socket/model     0.016s
  ok  atlas-channel/socket/writer    1.010s
```

## Scope

Range `25ba8d1cb..7ca937148`. Go/wire surfaces actually in the diff:

- `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go`
- `libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go`
- `services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go`

(The service-relative path is `services/atlas-channel/atlas.com/channel/socket/writer/...`,
not `services/atlas-channel/channel/socket/writer/...` as given in the task
brief — confirmed via `git log --all` returning nothing for the literal path
and `find` locating the file under the `atlas.com/channel` module root.)

Both changed Go packages are **support packages** — neither has a `model.go`;
they carry only wire-codec struct + `Encode`/`Decode` (packet lib) and a thin
body-builder wrapper (channel writer). No `resource.go`, `rest.go`,
`processor.go`, `entity.go`, `provider.go`, or `administrator.go` in either.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | No | Neither changed package has `model.go`/`entity.go`/`rest.go`/`provider.go`. |
| FILE placement (FILE-01..06) | Yes | Every changed Go package runs FILE-*, unconditionally. |
| SUB sub-domain (SUB-01..04) | No | Neither package has `resource.go`. |
| REST (DOM-06..09,12..15,17..19,32) | No | No `resource.go`/`rest.go`/`processor.go`, no HTTP route registration. |
| Constants reuse (DOM-21) | No | Diff modifies fields of two pre-existing types (`MobEscortWaypoint`, `MobEscortFullPath`); it declares no new type, no new named `const` block (`MobEscortFullPathWriter` predates this diff, confirmed no `const` line in the diff hunk), and the `attr==1`/`attr==2` literals are wire-format discriminators local to this one packet, not an item/inventory/weapon/id-width classification per the rule's own list. |
| Testing (DOM-10,20,24,33) | Yes (family) | Diff touches `mob_escort_full_path_test.go`. |
| Cache (DOM-29) | No | No `cache.go`, no cached processor state. |
| Messaging (DOM-30) | No | No `producer.go`, no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call. |
| Multi-tenancy (DOM-31) | No | No `rest.go`; tenant/trace state is not read or passed by the changed code (only a `context.Context` threaded to `Encode`/`Decode`, unused). |
| Migration hygiene (DOM-34,35) | No | Diff renames struct fields and doc comments in place; it does not move or extract symbols between a service and a `libs/atlas-*` module. |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module, no Kafka topic env var touched. |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed (`mob_escort_full_path.go` ×2). |
| Channel wire values (DOM-25) | Yes | Diff touches `services/atlas-channel` and `libs/atlas-packet`. |
| Resilience (DOM-27,28) | No | No DB-backed handler, no `model.Decorator`/enrichment path. |
| External clients (EXT-01..04) | No | No `requests.RootUrl`/`requests.*Request[T]` call. |
| Scaffolding (SCAFFOLD-01..09) | No | No new service, no new Writer/Handler registration (writer already registered pre-diff), no `routes.conf` change. |
| Security (SEC-01..04) | No | Not an auth/token/redirect/secret-handling surface. |
| patterns-provider.md (foundational) | No | No provider defined or composed. |
| patterns-functional.md (foundational) | No | No curried constructor/decorator/combinator defined. |

## Checklist Results

### monster/clientbound (support package — `libs/atlas-packet`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor lives in `processor.go` | N/A | No `type Processor interface`/`ProcessorImpl`/`NewProcessor(` anywhere in `mob_escort_full_path.go` — package holds only a wire struct. |
| FILE-02 | RestModel/Transform/Extract in `rest.go` | N/A | No `RestModel`, `Transform(`, `Extract(` in the file. |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` call. |
| FILE-04 | Entity/Migration/TableName in `entity.go` | N/A | No `type entity struct`, `Migration(`, `TableName()`. |
| FILE-05 | Builder/Model/administrator/provider placement | N/A | No `Builder`, no domain `Model` (this is a wire struct, not a DOM-domain `Model`), no writes, no `database.Query`. |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go:1-153` carries exactly one responsibility — the `MobEscortWaypoint`/`MobEscortFullPath` wire structs plus their `Encode`/`Decode` — matching the single-purpose packet-codec shape used throughout `libs/atlas-packet`. |
| DOM-25 | Client-interpreted wire values resolved from tenant table, not Go literal outside codec internals | PASS | `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go:116` and `:141` (`if wp.attr == 2`) are inside `libs/atlas-packet` codec internals, which the rule's pass criteria explicitly exempts ("No client wire code appears as a Go literal outside `libs/atlas-packet` codec internals"). The dispatch opcode itself is not a Go literal at all: `services/atlas-channel/atlas.com/channel/main.go:758` registers the writer by name (`monstercb.MobEscortFullPathWriter`), and the opcode→writer binding lives in the seed template, e.g. `services/atlas-configurations/seed-data/templates/template_gms_92_1.json:3808-3809` (`"opCode": "0x128"`, `"writer": "MobEscortFullPath"`), not in Go source. |
| DOM-26 | Every goroutine via `routine.Go`; bare `go` needs `//goroutine-guard:allow` | PASS | `grep -n "go \|goroutine" libs/atlas-packet/monster/clientbound/mob_escort_full_path.go` — no match; the file spawns no goroutines. |
| DOM-20 | Tests are table-driven (`tests := []struct{...}` + `t.Run`) | PASS | `libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go:63-68` iterates the shared `pt.Variants` table with `t.Run(v.Name, ...)`, the established idiom for this test suite's per-tenant round-trip coverage. |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | Test opens no GORM DB — pure byte-fixture + round-trip codec test. |
| DOM-24 | Test installs `producertest` stub when reaching an emit path | N/A | Test reaches no `AndEmit`/`message.Emit`/`producer.Produce` path. |
| DOM-33 | Interface change updates every mock | N/A | No `Processor`/`Provider`/`Administrator` interface method added/removed/re-signed in this diff. |
| Version-gating idiom (task brief) | `MajorAtLeast` used instead of raw numeric comparison | N/A | `grep -n "MajorAtLeast\|Major()\|>=\|<=" libs/atlas-packet/monster/clientbound/mob_escort_full_path.go` finds no version conditional at all — `Encode`/`Decode` (lines 106-153) contain zero branches on `context`/tenant version; the doc comment (lines 46, 64-68) and test comment (`mob_escort_full_path_test.go:10-12`) state the layout is byte-identical across all three routed versions, so no gate — raw or `MajorAtLeast` — exists to grade. Not a violation: there is nothing to gate. |

### socket/writer (support package — `services/atlas-channel`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | Processor/RestModel/requests/entity/builder placement | N/A | `services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go:1-27` contains only `MobEscortFullPathBody`, a thin `packet.Encode`-returning wrapper around `monsterpkt.NewMobEscortFullPath(...).Encode(...)`. None of the FILE-01..05 constructs are present. |
| FILE-06 | No catch-all file | PASS | Same file, single responsibility (body-builder wrapper), matching every sibling file in `socket/writer/`. |
| DOM-25 | Wire values resolved from config, not literal | PASS | `mob_escort_full_path.go:20-24` — the function takes the escort fields as typed parameters (`oldDestX int32`, `waypoints []monsterpkt.MobEscortWaypoint`, etc.) from its caller and forwards them to the codec; it contains no dispatcher/sub-op/message-code literal itself. |
| DOM-26 | Goroutine discipline | PASS | `grep -n "go \|goroutine" .../writer/mob_escort_full_path.go` — no match. |
| Struct immutability (task brief) | Unexported fields + accessor methods, no exported setters | PASS | `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go:26-30` (`MobEscortWaypoint{x,y,attr,stopDuration}`, all lowercase) and `:71-79` (`MobEscortFullPath{oldDestX,oldDestY,waypoints,currentDestIndex,hasStopDuration,stopDuration,stopIndefinitely}`, all lowercase); construction only via `NewMobEscortWaypoint` (`:33-35`) and `NewMobEscortFullPath` (`:81-91`); read-only accessors at `:37-40` and `:93-99`; no exported setter method on either type. |

## Not evaluable from the diff

None — the review surface (three changed Go files plus their direct callers/config, reached via targeted `grep`) was sufficient to settle every applicable rule.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.
