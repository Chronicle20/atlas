# Backend Audit — atlas-channel (fix/monster-card-pickup-feedback)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Scope:** diff 75d730bc3..f64eea592 — `kafka/consumer/drop/consumer.go`, `kafka/consumer/drop/consumer_test.go`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-05
- **Build:** PASS (`go build`/`go vet`/`go test` clean in drop package; gofmt clean)
- **Tests:** drop package PASS (`ok atlas-channel/kafka/consumer/drop`)
- **Overall:** PASS

## Scope note

This is a sub-domain Kafka consumer package (`resource.go`-less; no `model.go`), not a REST
domain package. Most DOM-* REST checks (DOM-01..05, 08, 17..19, builder/entity/Transform) are
N/A — the package has no model, entity, REST layer, builder, or HTTP handlers. The audit applies
the items that are actually triggered by the changed lines, per the focus brief. Pre-existing
handler structure outside the diff is not re-litigated.

## Per-item results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Handler/consumer uses `logrus.FieldLogger`, not `*logrus.Logger` | PASS | `consumer.go:139-140` handler closure takes `l logrus.FieldLogger`; `isConsumedOnPickupCard` is logger-free |
| DOM-21 | No reinvented atlas-constants type / magic number | PASS | `consumer.go:43-45` uses `item.GetClassification(item.Id(itemId)) == item.ClassificationConsumableMonsterCard`; constant defined at `libs/atlas-constants/item/constants.go:43` (`Classification(238)`), helper at `constants.go:126`. No literal `238`/`/10000` reinvented in the service. Type conversion `item.Id(itemId)` valid (`Id uint32`, `constants.go:5`) |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS (N/A) | `consumer_test.go` exercises only the pure predicate `isConsumedOnPickupCard`; no `AndEmit`/`message.Emit`/`producer.Produce` and no consumer-handler invocation (grep exit 1). No producer path reached, so no stub required |
| — | Reuse existing enable-actions packet (not bespoke) | PASS | `consumer.go:175` uses `statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode` via `statpkt.StatChangedWriter` — byte-identical idiom to the compartment consumer `compartment/consumer.go:78` and `:132-135` (`enableActions`). `NewStatChanged(updates, exclRequestSent)` signature confirmed at `libs/atlas-packet/stat/clientbound/changed.go:34` |
| — | Error handling / logging consistency | PASS | `consumer.go:170-173` checks `err`, logs via `l.WithError(err).Errorf(...)` with characterId + dropId, returns `err` — matches the sibling status-message branch (`consumer.go:162-166`). The callback return is discarded by `IfPresentByCharacterId` in both branches, so semantics are consistent |
| — | Goroutine usage matches existing pattern | PASS | New branch lives inside the existing `go func()` + `session.NewProcessor(l, ctx).IfPresentByCharacterId(...)` block (`consumer.go:151-152`); no new goroutine introduced; the second goroutine (DropDestroy, `consumer.go:170-180`) is unchanged |
| — | Predicate pure + unit-tested; table-driven | PASS | `isConsumedOnPickupCard` (`consumer.go:43-45`) is side-effect-free. `consumer_test.go:9-31` is a `cases := []struct{...}` + `t.Run` table test covering card lo/mid/hi (2380000/2380001/2389999→true) and use/etc/equip/zero→false |
| — | No `*_testhelpers.go`; conventions respected | PASS | Single `consumer_test.go` added; no test-only constructor file; `package drop` internal test |
| — | Branch-ordering correctness | PASS | New card branch precedes the meso/equip/stackable branch but is itemId-classification-gated. Meso pickups carry `ItemId == 0` → `GetClassification(0) == 0 ≠ 238` → predicate false (covered by the `itemId: 0 → false` case, `consumer_test.go:18`), so meso/coin pickups correctly fall through to the generic message. Upper boundary 2389999→true, 2390000 would be 239 (excluded) — classification math is floor(itemId/10000), correct |

## Summary

### Critical
- None.

### Important
- None.

### Notes (non-blocking, not findings)
- DropDestroy goroutine (`consumer.go:170-180`) is unchanged by this diff — not in scope.
- `tools/redis-key-guard.sh` local failure is a pre-existing cross-module go.sum artifact (reproducible on clean main, no redis diagnostics); atlas-channel passes the guard clean. Not attributable to this change.

**Verdict: PASS** — the change uses the shared `libs/atlas-constants/item` classification (no magic
number, DOM-21 satisfied), reuses the established `StatChanged(excl=true)` enable-actions packet
rather than inventing one, keeps error-handling/logging/goroutine structure consistent with the
surrounding handler and the compartment consumer, and adds a pure, table-driven unit test. No
blocking or non-blocking guideline violations found in the diff.
