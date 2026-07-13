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

---

# Post-rebase + v79 re-audit

- **Date:** 2026-07-13
- **Branch:** `task-129-vicious-hammer-use`
- **Base:** `origin/main` (three-dot Go diff)
- **Build:** PASS — `go build ./...` clean in `atlas-consumables` and `atlas-channel`
- **Vet:** PASS — `go vet ./...` clean in both services
- **Tests:** PASS — `consumable`, `equipable`, `data/equipable`, `socket/handler` packages green (`-count=1`)
- **Overall:** NEEDS-WORK — one **Important** DOM-25 finding (previously under-classified as "Minor #2 magic numbers"). Everything else re-confirms PASS or the prior Minor advisories.

## Verified since the pre-rebase pass

- **Gen3 interface/impl/mock consistency — PASS.** `RequestViciousHammer` is on the consumables `Processor` interface (`consumable/processor.go:61`), impl (`processor.go:925`), and mock (`mock/processor.go:92`); `var _ Processor = (*ProcessorImpl)(nil)` at `processor.go:85`, `var _ consumable.Processor = (*ProcessorMock)(nil)` at `mock/processor.go:28`. Channel side identical: `RequestViciousHammerUse` on interface (`consumable/processor.go:18`), impl (`:46`), mock (`mock/processor.go:36`), assertions at `processor.go:34` / `mock/processor.go:18`.
- **`NewProcessor(l, ctx).(*ProcessorImpl)` type assertion (`consumable/processor.go:1011`) — PASS (safe, advisory only).** `NewProcessor` (`processor.go:73`) has exactly one implementation (`*ProcessorImpl`), so the unchecked assertion cannot panic. It exists because `ConsumeViciousHammer` (a standalone `ItemConsumer` factory) needs the impl-only helpers `ViciousHammerError` + `validateViciousHammerUse`, which depend on `p.cpp`/`p.l`/`p.ctx`. It is the ONLY `.(*ProcessorImpl)` site in the file — sibling `Consume*` factories (`ConsumeScroll` `:627`, `ConsumeStandard` `:337`, etc.) use `p := NewProcessor(l, ctx)` because they only need interface methods (`ConsumeError`, `FailScroll`). No guideline forbids the assertion; the cleaner-but-not-required alternative is to promote `ViciousHammerError` to the interface. Advisory, not a finding.
- **Dropped `RegisterHandler` error (`processor.go:991`→`993`) — PASS (mirrors sibling).** `_, err = ...RegisterHandler(...)` then `err = p.cpp.RequestReserve(...)` overwrites the first `err`. This is byte-for-byte the established `RequestScroll` idiom (`processor.go:590`→`592`) and the base reserve flow (`:234`→`:236`). Pre-existing convention, not a rebase regression.
- **DOM-21 — PASS.** `ClassificationViciousHammer = Classification(557)` was added to the shared lib (`libs/atlas-constants/item/constants.go:106`) and consumed via `item.GetClassification(...) != item.ClassificationViciousHammer` (`processor.go:958`). No service-local redeclaration. (The bare `if category == 557` at `character_cash_item_use.go:511` is an unchanged pre-existing line — reaffirms prior Minor #1; could now use the named constant.)
- **DOM-24 — PASS by avoidance.** The three changed/new test files (`consumable/vicious_hammer_test.go`, `equipable/processor_test.go`, `socket/handler/vicious_hammer_token_test.go`) contain no direct or transitive emit call sites; no `producertest` stub required.
- **Mode byte config-resolution — PASS.** The version-variable dispatcher mode (v83 61/62 vs v95 65/66) is resolved via `atlas_packet.WithResolvedCode("operations", ...)` in `libs/atlas-packet/field/vicious_hammer_body.go:24,33,42`. Correctly NOT hard-coded.

## FAIL — DOM-25: client notice/fail-reason code threaded raw from a domain service

**Status: FAIL (Important).** The dispatcher *mode* byte is config-resolved (good), but the *failure notice selector* (`errorCode` 1/2/3 — the value the client's `CUIItemUpgrade::ShowResult` switch maps to a message string: 1 = "not upgradable", 2 = "cap reached", 3 = "Horntail Necklace") is a client wire code that is (a) written as Go literals in channel handler code and (b) emitted raw by a domain service in a Kafka event. Both are exactly what DOM-25(a) and DOM-25(c) prohibit.

Evidence:
- **DOM-25(c) — domain service emits a client byte, not a semantic key.** `ViciousHammerBody.ErrorCode uint32` (`services/atlas-consumables/.../kafka/message/consumable/kafka.go:98`) is populated with the raw client selector by `ViciousHammerEventProvider(..., errorCode uint32)` (`consumable/producer.go:45,52`) and the `ViciousHammerError*` constants at `consumable/processor.go:885-889`. atlas-consumables is a domain service; per DOM-25(c) it must emit a SEMANTIC key (the WishOrigin/FailReason pattern), e.g. `"NOT_UPGRADABLE"`/`"CAP_REACHED"`/`"HORNTAIL"`, and let the channel resolve to the wire code. The sibling `ScrollBody` (`kafka.go:88-93` in the same file family) carries only semantic booleans — the hammer path diverges by shipping a client code.
- **DOM-25(a) — client wire codes as Go literals outside packet codec internals.** `character_cash_item_use.go:530` `announce(fieldpkt.ViciousHammerFailureBody(1))` and `:534` `announce(fieldpkt.ViciousHammerFailureBody(2))` pass bare notice selectors into a `*Body(...)` arg from a channel socket handler.
- **Channel consumer forwards the raw code unresolved.** `kafka/consumer/consumable/consumer.go:127` `fieldpkt.ViciousHammerFailureBody(e.Body.ErrorCode)` threads the domain-emitted byte straight into the packet with no `WithResolvedCode`/soft-resolver step.

Why this is the DOM-25 class and not a cosmetic "magic number":
- DOM-25 names "notice/fail reason code — any byte the client feeds through a lookup switch" as in-scope, and part (c) explicitly calls a "client code in a Kafka event produced by a domain service" a finding. The rule cites the **task-102 NoticeFailReason** precedent — the same shape (a notice selector resolved via a soft resolver + tenant writer-options table), which is the documented fix pattern (`failNoticeOr`/`noticeFailReasons`).
- The code comment justifies the literals as version-stable (1/2/3 identical in v83/v95). DOM-25's stated ruling is that "'version-stable (IDA-verified identical)' does NOT exempt it." So stability is not a defense here.

Practical-risk note (for the fix-prioritization, not an exemption): the mode byte — the genuinely version-divergent value — IS resolved, and jms is version-absent, so the immediate wire-break risk is low. But the rule as written fails on the raw notice codes, and the prior pass mis-filed this as advisory "Minor #2." Recommended fix: define semantic `EventTypeViciousHammer` fail-reason keys (string constants) in atlas-consumables, emit those, and resolve them to the client selector on the channel (soft resolver with a bare-arm `default` → unknown), matching the WishOrigin/FailReason precedent.

## Summary of FAILs

- **DOM-25 (Important):** Vicious-hammer failure notice selector (1/2/3) is a client wire code emitted raw by the domain service (`atlas-consumables kafka/message/consumable/kafka.go:98`, `producer.go:45`) and passed as Go literals in the channel handler (`character_cash_item_use.go:530,534`) / forwarded unresolved by the consumer (`consumer.go:127`). Domain services must emit semantic fail-reason keys and let the channel resolve them (WishOrigin/FailReason pattern; task-102 NoticeFailReason precedent). Stability of the codes does not exempt.

No other new findings. Prior Minor advisories (#1 bare 557, #2 magic numbers — now elevated to the DOM-25 FAIL above, #3 discarded RegisterHandler err, #4 consume-after-mutate ordering, #5 emit paths untested) stand as previously recorded.

### Resolution — DOM-25 errorCode (2026-07-13)

FIXED. The failure notice selector is now config-resolved end-to-end, no client
wire byte survives as a Go literal:

- `atlas-consumables` emits a SEMANTIC reason (`ViciousHammerReason`:
  `NOT_UPGRADABLE` / `CAP_REACHED` / `HORNTAIL` / `UNKNOWN`, `""` = eligible) in
  the Kafka event `ViciousHammerBody.Reason` — the `ErrorCode uint32` field is
  gone (`consumable/processor.go`, `producer.go`,
  `kafka/message/consumable/kafka.go`).
- `libs/atlas-packet/field/vicious_hammer_body.go` `ViciousHammerFailureBody`
  now takes a `ViciousHammerFailureReason` and resolves BOTH the dispatcher mode
  (`operations`/FAILURE) AND the notice byte (`errorCodes`/<reason>) from the
  tenant writer config via `ResolveCode`. New test
  `vicious_hammer_body_test.go` covers the resolution + the unconfigured-reason
  99 degrade.
- `atlas-channel` no longer hardcodes literals: the pre-check arm passes
  `fieldpkt.ViciousHammerReasonNotUpgradable` / `...CapReached`, and the result
  consumer forwards `e.Body.Reason`.
- All 5 GMS templates (v79/83/84/87/95) carry an `errorCodes` table on the
  `ViciousHammer` writer (`UNKNOWN`0 / `NOT_UPGRADABLE`1 / `CAP_REACHED`2 /
  `HORNTAIL`3). Values are version-stable but now tenant-config-resolved
  (DOM-25 — "version-stable never exempts").

Verified: `go build`/`vet`/`test -race` clean in atlas-packet, atlas-consumables,
atlas-channel; `matrix --check`, `dispatcher-lint`, `operations --check`,
redis-key-guard, goroutine-guard all clean.
