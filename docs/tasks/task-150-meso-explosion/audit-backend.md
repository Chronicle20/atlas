# Backend Audit — task-150 (Meso Explosion)

- **Scope:** Changed Go packages only (merge-base `cdfb71aa3` → HEAD `e43f2d883`)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-27
- **Build:** PASS
- **Tests:** All passed (0 failed)
- **Overall:** PASS

## Build & Test Results

```
cd libs/atlas-packet && go build ./...          -> clean
cd libs/atlas-packet && go test ./... -count=1  -> all packages ok (model, character/serverbound, etc.)
cd services/atlas-channel/atlas.com/channel && go build ./...          -> clean
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1 -> all packages ok, including
    ok atlas-channel/drop            0.029s
    ok atlas-channel/socket/handler  0.720s (fast — no unstubbed Kafka retry cost)
go vet ./model/... ./character/serverbound/...  (atlas-packet)          -> clean
go vet ./drop/... ./socket/handler/... ./kafka/message/drop/... (channel) -> clean
tools/goroutine-guard.sh (repo root)             -> exit 0
```

## Domain Checklist Results

None of the changed packages are new domain packages (no new `model.go`). This
diff modifies existing packages: `libs/atlas-packet/model` (attack_info.go,
damage_info.go), `libs/atlas-packet/character/serverbound` (test only),
`services/atlas-channel/.../drop` (processor.go, producer.go, mock),
`services/atlas-channel/.../kafka/message/drop` (kafka.go envelope),
`services/atlas-channel/.../socket/handler` (character_attack_common.go +
new character_attack_meso_explosion.go), and
`services/atlas-channel/.../data/skill/effect` (model.go accessor). The
File-Responsibilities and targeted DOM/anti-pattern checks below were run
against every touched file per the audit brief's focus areas.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplicate of atlas-constants types — meso skill id | PASS | `libs/atlas-constants/skill/constants.go:3179` defines `ChiefBanditMesoExplosionId = Id(4211006)` (pre-existing, not redeclared). `libs/atlas-packet/model/attack_info.go:8` imports `github.com/Chronicle20/atlas/libs/atlas-constants/skill` and uses `skill.ChiefBanditMesoExplosionId` at `attack_info.go:141` and `attack_info.go:300`. `services/atlas-channel/.../character_attack_common.go:27` imports the same package aliased `skill3` and uses `skill3.ChiefBanditMesoExplosionId` at `character_attack_common.go:539`. No bare `4211006` literal appears in any non-test production `.go` file — the only non-test hits are the constant definition itself and doc comments. |
| DOM-21b | Kafka command envelope mirrors atlas-drops without cross-service coupling | PASS | `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go:16-24` adds `TransactionId uuid.UUID` and `ConsumeCommandBody{DropId uint32}`, matching atlas-drops' own independently-defined `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:68-75` (`Command[E]` with `TransactionId` first field) and `:181-183` (`CommandConsumeBody{DropId uint32}`) field-for-field. Each service defines its own struct in its own package (no import of the other service's internal Go package) — a clean JSON-wire mirror, not a layering violation. |
| FILE-01 | Processor logic in processor.go | PASS | `drop.Processor` interface gains `ConsumeAll(f field.Model, dropIds []uint32) error` at `services/atlas-channel/atlas.com/channel/drop/processor.go:19` (interface) and the `ProcessorImpl` method at `processor.go:42-47`. No processor method or interface addition appears outside `processor.go`. |
| FILE-02/03 | Producer / requests placement | PASS | `ConsumeAllCommandProvider` lives in `services/atlas-channel/atlas.com/channel/drop/producer.go:19-33`, alongside the pre-existing `RequestReservationCommandProvider`/`SpawnMesoCommandProvider` in the same file. No REST client (`requests.go`) files were touched — not triggered. |
| FILE-06 | No package-named catch-all file | PASS | New file `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion.go` contains exactly one pure validation function (`validateMesoExplosion`) — a single-purpose utility, not a `<pkg>.go` collapse of ≥2 responsibilities. |
| DOM-26 | Goroutines via routine.Go | PASS | `grep -rnE '^\s*go (func|[A-Za-z_])'` over all changed non-test files in scope returns zero matches. `tools/goroutine-guard.sh` exits 0 from repo root. |
| DOM-20 | Table-driven tests | PASS | `character_attack_meso_explosion_test.go:24-37` (`TestValidateMesoExplosion`) uses `tests := []struct{...}` + `t.Run`. `attack_info_test.go` and `damage_info_test.go` iterate `pt.Variants` with `t.Run(v.Name, ...)`. |
| DOM-24 | Kafka producer stubbed in emit-path tests | PASS (no stub needed — no real emit path exercised) | `drop/producer_test.go:TestConsumeAllCommandProvider` calls `drop.ConsumeAllCommandProvider(txId, f, dropIds)()` directly — a pure `model.Provider[[]kafka.Message]` built via `producer.MessageProvider(model.FixedProvider(raws))`, with no network I/O. No test in the diff calls `drop.Processor.ConsumeAll` (the method that actually invokes `producer.ProviderImpl`), and `character_attack_meso_explosion_test.go` only exercises the pure `validateMesoExplosion` helper — `processAttack` itself (which would reach the real emit path) is not invoked by any new/changed test. `go test ./socket/handler/... -count=1` completed in 0.720s, confirming no unstubbed 42s retry cost was introduced. |
| Purity — `validateMesoExplosion` | No I/O | PASS | `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion.go:19-35` — the function only reads its `dropIds`/`fieldDrops`/`maxCount` arguments and a local `seen` map; no DB/HTTP/Kafka/logger calls. |
| Error-handling wiring in `processAttack` | Rejection → nil / infra failure → real error / CONSUME failure → log-and-continue | PASS | `character_attack_common.go:540-543`: `InMapModelProvider` failure returns `dErr` (the real error) unmodified. `character_attack_common.go:548-551`: `validateMesoExplosion` failure logs a `Warnf` and `return nil` — this is the top-level `func(s session.Model) error` closure, so it aborts before the HP/MP cost block (`:561-568`), the damage-processing loop (`:636-643`), the field broadcast (`:645-670`), and the drop-destruction block (`:687-693`) — confirmed no side effects follow. `character_attack_common.go:687-693`: `ConsumeAll` failure is logged via `l.WithError(cErr).Errorf(...)` and execution falls through to `return nil` at `:724` (function continues) — matches the documented at-least-once posture used identically for projectile emission at `:675-679`. |
| Model / accessor discipline | Private field + public getter | PASS | `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:54` — `attackCount uint32` is a pre-existing private field; this diff only adds the getter `AttackCount() uint32` at `model.go:161-163`, no new mutable state. |

## Sub-Domain Checklist Results

Not applicable — no new action-event (`resource.go`-without-`model.go`) package was added.

## Security Review

Not applicable — atlas-channel is not an auth/token service; this feature does not touch authentication/authorization code.

## Summary

### Blocking (must fix)
None.

### Non-Blocking (should fix)
None found within the scoped diff. (Out-of-scope note, not a finding: `character_attack_common.go` retains ~15 pre-existing `// TODO` markers for unrelated unimplemented attack effects (Bandit Steal, Homing Beacon, etc.) — none were added or touched by this task; the task's own `// TODO destroy Chief Bandit exploded mesos` marker at the old `:2740` position was correctly removed and replaced with the working implementation.)
