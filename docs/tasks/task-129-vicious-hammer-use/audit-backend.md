# Backend Audit — task-129 (Vicious Hammer) Go changes

- **Worktree:** `.worktrees/task-129-vicious-hammer-use` (branch `task-129-vicious-hammer-use`, HEAD `ebce273b04`)
- **Base:** `git merge-base main HEAD` = `38d4d0ba2`
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Date:** 2026-07-03
- **Build:** PASS (all 4 changed modules)
- **Vet:** PASS (all 4 changed modules)
- **Tests:** PASS (all 4 changed modules)
- **Overall:** PASS — no blocking (Critical/Important) findings. 5 Minor advisories below.

> Note: an early run of my build/test/grep commands accidentally targeted the
> main repo checkout (on `main`) instead of the worktree; all results reported
> here were re-run inside the worktree and confirmed against the task-129 code.

## Phase 1 — Build & Test Gate (objective)

Run in the worktree:

| Module | build | vet | test |
|--------|-------|-----|------|
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS | PASS (`consumable`, `equipable` ok) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS (`socket/handler`, `socket/writer` ok) |
| `libs/atlas-packet` | PASS | — | PASS (`cash/serverbound`, `field/*` ok) |
| `libs/atlas-constants/item` | PASS | — | PASS |

Gate satisfied → proceeded to per-change checks.

## Scope classification

This feature is an **action-event / cross-service flow**, not a new domain
package. No package with a fresh `model.go` was added, so the full DOM-01..20
domain-scaffold checklist (builder/entity/rest/resource per new domain) does
not apply. The DOM checks that *do* apply to added code — logger typing,
handler→processor layering, no direct DB writes in handlers, shared-constant
reuse (DOM-21), config-resolved mode bytes, Kafka topic naming, producer stub
in emitting tests, plus SEC — were all evaluated and are reported below.

## PASS confirmations (positive evidence)

- **SEC (client-echoed token re-validated authoritatively).** The
  ITEM_UPGRADE_UPDATE confirm packet carries a client-echoed round-trip token.
  `socket/handler/item_upgrade_update.go:29` unpacks it
  (`unpackViciousHammerToken`) and forwards raw slots to
  `RequestViciousHammerUse`. The server never trusts the token: in
  `consumable/processor.go` `RequestViciousHammer` (added lines 918-956)
  re-fetches the character, requires a real item at `hammerSlot` in the **Cash**
  compartment whose classification is `item.ClassificationViciousHammer`
  (else `ViciousHammerErrorUnknown`), re-resolves the target via
  `resolveViciousHammerTarget`, and re-runs `validateViciousHammerUse`
  (WZ slots / cash / cap / Horntail). A forged token can only reference the
  caller's own valid hammer + eligible target — no escalation. **Anti-replay:**
  `ConsumeViciousHammer` (added lines 963-1002) re-validates cap against fresh
  state at execution time before mutating, so a replayed confirm past the cap
  fails with `ViciousHammerErrorCapReached`. SEC-* PASS.
- **No hard-coded dispatcher mode bytes.** All three arms resolve the mode from
  the tenant `operations` table via `atlas_packet.WithResolvedCode(...)` —
  `libs/atlas-packet/field/vicious_hammer_body.go:24,33,42`. The clientbound
  codecs write the resolved `mode` byte, never a literal. PASS.
- **Multi-tenancy guard on the event consumer.**
  `kafka/consumer/consumable/consumer.go` `handleViciousHammerConsumableEvent`
  uses `tenant.MustFromContext(ctx)` + `t.Is(sc.Tenant())` before writing to the
  session. PASS.
- **DOM-06/07 (FieldLogger).** Every new function/processor takes
  `logrus.FieldLogger` (`RequestViciousHammer`, `ConsumeViciousHammer`,
  `ViciousHammerError`, channel `RequestViciousHammerUse`, all handlers). No
  `*logrus.Logger` / `StandardLogger()`. PASS.
- **DOM-15 (no direct DB writes in handlers).** Channel handlers issue Kafka
  commands / call processors only; the authoritative equip mutation is
  `ep.ChangeStat(...)` inside the consumables reservation callback. PASS.
- **DOM-21 (shared types) — largely PASS.** New constant added correctly at
  `libs/atlas-constants/item/constants.go:106`
  (`ClassificationViciousHammer = Classification(557)`). Kafka bodies use
  `slot.Position` and `character.Id`
  (`kafka/message/consumable/kafka.go` both services). Producer branch uses the
  new constant. One raw-literal exception — see Minor #1.
- **No TODOs/stubs in landed code.** The stale
  `// TODO for v83 there is a trailing updateTime` was *removed* from
  `character_cash_item_use.go`; no new `TODO`/`panic`/501 introduced. (The only
  `// TODO consume vega scroll` in `processor.go` is pre-existing in
  `ConsumeScroll`, untouched.) PASS.
- **Dead-code cleanup.** The obsolete `socket/writer/vicious_hammer.go`
  (referenced the removed `NewViciousHammer` stub) was deleted. PASS.
- **Immutable model / builder.** `data/equipable` `cash bool` added as a private
  field + `Cash()` getter (`model.go`) + `Extract` mapping (`rest.go`);
  `AddHammersApplied` added as an `equipable.Change` delegating to the
  pre-existing `asset.ModelBuilder.AddHammersApplied`. PASS.
- **Curried consumer registration / producer.Provider pattern.** New command
  handler registered via `rf(t, message.AdaptHandler(message.PersistentConfig(...)))`;
  event producer follows `producer.SingleMessageProvider(key, value)` with
  `producer.CreateKey(int(characterId))`. PASS.

## Minor advisories (non-blocking)

### Minor #1 — DOM-21: raw `557` literal where the new constant exists
`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:488`
```go
if category == 557 {
```
`category` is `item.Classification`; this task added
`item.ClassificationViciousHammer = Classification(557)`. The same block was
refactored in this task to use the named `CashSlotItemTypeViciousHammer*`
return constants, and the enclosing function already uses
`item.ClassificationPet` / `item.ClassificationTeleportRock` / etc. elsewhere,
so the bare `557` is an inconsistency that the guideline's spirit targets.
Functionally correct. Fix: `if category == item.ClassificationViciousHammer {`.

### Minor #2 — magic numbers in the channel pre-check
`character_cash_item_use.go` `handleViciousHammerOpen` (~lines 509-513) uses raw
`target.HammersApplied() >= 2` and `ViciousHammerFailureBody(1)` /
`ViciousHammerFailureBody(2)`. These duplicate `maxHammersApplied` and the
`ViciousHammerError*` selectors defined in `consumable/processor.go`. They live
in different modules so cannot share directly, but the cap (`2`) and the notice
codes would be better as named constants (e.g. exported from the packet lib
next to the body funcs). Advisory only.

### Minor #3 — discarded `RegisterHandler` error (pre-existing pattern)
`consumable/processor.go` `RequestViciousHammer`:
```go
_, err = consumer.GetManager().RegisterHandler(...)   // err then overwritten
err = p.cpp.RequestReserve(...)
```
The registration error is silently overwritten by the next assignment. This is
verbatim the established `RequestScroll` pattern (same file, RequestScroll body
~line 572). Confirmed pre-existing convention; flagged for completeness only.

### Minor #4 — consume-after-mutate ordering (pre-existing scroll pattern)
`consumable/processor.go` `ConsumeViciousHammer` mutates the equip first
(`ep.ChangeStat(..., AddSlots(1), AddHammersApplied(1))` — atomic single
MODIFY_EQUIPMENT) and only then `cpp.ConsumeItem(...)` for the hammer. If
`ConsumeItem` fails it is only logged and the flow still emits the terminal
**success** event → the player could keep the hammer while the equip is already
upgraded. This mirrors `ConsumeScroll` exactly (ChangeStat → ConsumeItem, log
only on consume failure); with `ExecuteTransaction` being a project-wide no-op
there is no true cross-service atomicity, and the reserve/consume pattern is the
project's accepted best effort. The *mutation-failure* path is handled correctly
(cancel reservation + `ViciousHammerError` failure event). Consistent with
existing behavior; noted, not a regression.

### Minor #5 — emit paths untested
`consumable/vicious_hammer_test.go` and `equipable/processor_test.go` cover only
pure helpers (`viciousHammerErrorCode`, `AddHammersApplied`) and the channel
`vicious_hammer_token_test.go` covers the token round-trip. `RequestViciousHammer`
/ `ConsumeViciousHammer` (the reserve/consume/re-validation flow, including the
anti-replay cap re-check) have no test. Because no test exercises an emit path,
**DOM-24 is satisfied by avoidance** (no `producertest` stub is required), but
the highest-value server logic is unverified by tests.

## Summary

### Blocking (must fix)
- None. Build/vet/tests pass; no Critical/Important findings.

### Non-Blocking (should fix)
- Minor #1 (DOM-21): use `item.ClassificationViciousHammer` instead of raw `557`
  at `character_cash_item_use.go:488`.
- Minor #2: name the cap/error-code magic numbers in the channel pre-check.
- Minor #3: pre-existing discarded `RegisterHandler` error (mirrors RequestScroll).
- Minor #4: pre-existing consume-after-mutate ordering (mirrors ConsumeScroll).
- Minor #5: add coverage for the reserve/consume/re-validation flow.
