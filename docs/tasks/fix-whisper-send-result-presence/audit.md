# Backend Audit — atlas-channel (fix-whisper-send-result-presence)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Audit range:** `62439e69b..2633d67d3` (task-238's own commits, everything at or before
  `62439e69b`, are out of scope — already audited on task-238's branch)
- **Files in scope:** `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go`,
  `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper_test.go`
- **Build:** PASS
- **Tests:** PASS (0 failed)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...
(clean, no output)

$ go test ./socket/handler/... -run TestWhisperChat -v -count=1
=== RUN   TestWhisperChat_Decision
    --- PASS x6 subtests (unresolvable name, never logged in, offline, in cash shop,
        in field, infrastructure error fails open)
=== RUN   TestWhisperChat_ProduceFailure
--- PASS
ok  	atlas-channel/socket/handler	0.014s

$ go test ./... -count=1
All packages report `ok` or `[no test files]`; zero FAIL lines across the module.
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | No | Package `handler` has none of `model.go`/`entity.go`/`rest.go`/`provider.go` |
| FILE placement (FILE-01..06) | Yes | Any changed Go package — unconditional; opened `file-responsibilities.md` |
| SUB (SUB-01..04) | No | No `resource.go` in the changed package |
| REST (DOM-06..09,12..15,17..19,32) | No | No `resource.go`/`rest.go`/`processor.go`, no HTTP route registration in the two changed files |
| Constants reuse (DOM-21) | Yes | Diff declares `type whisperOutcome struct` — a new type; opened `anti-patterns.md` |
| Testing (DOM-10,20,24,33) | Yes | Diff touches `character_chat_whisper_test.go`; opened `testing-guide.md` |
| Cache (DOM-29) | No | No `cache.go`, no cached processor/struct state |
| Messaging (DOM-30) | No | File calls `message.NewProcessor(...).WhisperChat(...)` (a sibling package's processor method), not `AndEmit`/`message.Emit`/`producer.ProviderImpl` directly |
| Multi-tenancy (DOM-31) | No | No `rest.go`; `ctx` is passed through to callees, not read for tenant/trace state, and no tenant/trace value crosses a REST model, request body, or path/query param |
| Migration hygiene (DOM-34,35) | No | No symbol moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No `libs/atlas-*` module added, no topic env var added/renamed |
| Runtime safety (DOM-26) | Yes (checked, no findings) | Non-test Go file changed; grepped for bare `go ` statements — none found |
| Channel wire values (DOM-25) | Yes | Diff touches `services/atlas-channel` socket handler code; opened `anti-patterns.md` |
| Resilience (DOM-27,28) | No | Not a DB-backed HTTP handler; no `model.Decorator` touched |
| External clients (EXT-01..04) | No | File calls sibling in-service packages (`character.NewProcessor`, `message.NewProcessor`, `location.Get`), not `requests.RootUrl`/`GetRequest[T]`/`PostRequest[T]` directly |
| Scaffolding (SCAFFOLD-01..09) | No | No new service, no new channel Writer/Handler registration, no `routes.conf` change |
| Security (SEC-01..04) | No | No auth/token/redirect/secret surface |
| patterns-functional.md (foundational) | Yes | `produceWhisperChatResult` / `produceFindResultBody` are curried constructors returning `model.Operator[session.Model]` |
| patterns-provider.md (foundational) | No | No provider composed or defined in the diff |

## Checklist Results

### handler (support package — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/methods live in `processor.go` | N/A | No `Processor`/`ProcessorImpl`/`NewProcessor(` declared in either changed file |
| FILE-02 | `RestModel`/`Transform`/`Extract`/JSON:API methods live in `rest.go` | N/A | No `RestModel` type or `Transform`/`Extract`/`GetName`/`GetID`/`SetID` declared |
| FILE-03 | Cross-service request functions live in `requests.go` | N/A | No `requests.RootUrl(`/`GetRequest[`/`PostRequest[`/`getBaseRequest(` in either file |
| FILE-04 | Entity struct/`Migration`/`TableName` live in `entity.go` | N/A | No entity/migration/table-name symbols declared |
| FILE-05 | Builder/Model/administrator writes/provider readers placed per file table | N/A | No `Builder`, domain `Model`, `Create*`/`Update*`/`Delete*` write, or `database.Query`/`SliceQuery` reader declared |
| FILE-06 | No package-named catch-all file carrying ≥2 responsibilities | PASS | `character_chat_whisper.go` (not `handler.go`) holds only handler funcs, package-level seam vars, and pure decision helpers (`whisperDecision`, `findDecision`, `produceWhisperChatResult`, `produceFindResultBody`) — none of FILE-01..05's responsibilities (Processor/RestModel/requests/Entity/Builder-Model-administrator-provider) are present, so there is nothing to collapse |
| DOM-21 | No redeclaration of a type/helper/constant that exists in `libs/atlas-constants/` | PASS | `type whisperOutcome struct{ deliverable bool; branch string; err error }` (character_chat_whisper.go:172-176) is a page-local decision-result type with no equivalent in `libs/atlas-constants/` (grepped, none); it wraps `characterconst.PresenceState`/`fieldconst.Model` from atlas-constants rather than redeclaring them |
| DOM-25 | Client-interpreted wire bytes resolved from a tenant writer-options table, never a Go literal | **FAIL** | character_chat_whisper.go:233 and :237: `fieldcb.NewWhisperSendResult(0x0A, targetName, false)` — the `0x0A` mode byte is a hardcoded Go literal, not resolved via `WithResolvedCode(...)` or an equivalent tenant writer-options table. Both call sites are new code introduced by this diff (the pre-diff code at the old single call site, `62439e69b`'s line 35, used the identical literal, but that line is baseline, not in scope — the diff nonetheless reproduces the same non-compliant literal into two new call sites rather than fixing it). Per `anti-patterns.md`'s task-103 uniformity ruling, "the value is version-stable" does not exempt it |
| DOM-26 | Every goroutine spawned via `routine.Go(l, ctx, fn)` | PASS (no findings) | `grep -nE '^\s*go (func|[A-Za-z_])' character_chat_whisper.go character_chat_whisper_test.go` — no matches; no goroutines added |
| DOM-20 | Tests are table-driven | PASS | character_chat_whisper_test.go:517-524 `cases := []struct{...}` and :579 `for _, c := range cases { t.Run(c.name, ...) }` (`TestWhisperChat_Decision`) |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | No `gorm.Open()`/direct DB bootstrap in either changed file |
| DOM-24 | Test package reaching an emit path installs `producertest` or injects a no-op producer per test | N/A | The only emit-reaching call in this arm is `message.NewProcessor(l, ctx).WhisperChat(...)` (called via the `produceWhisperChatFunc` seam, character_chat_whisper.go:52-54), which itself calls `producer.ProviderImpl(...)` one hop down (`message/processor.go:84`). Every subtest in `TestWhisperChat_Decision` and `TestWhisperChat_ProduceFailure` overrides `produceWhisperChatFunc` with a stub before calling `dispatchWhisperChat` (character_chat_whisper_test.go:585-590, 629-633) and restores it via `t.Cleanup`, so the real `WhisperChat`/`producer.ProviderImpl` call is never reached by any test in this file — confirmed empirically: `go test ./socket/handler/... -run TestWhisperChat -v` completes in 0.014s with no live Kafka broker, well under the ~42s a reached-but-unstubbed emit path would cost per the DOM-24 procedure note. Because the emit path is never actually reached (the seam substitution happens strictly before the call, not after), DOM-24's own trigger — "reaches an emit path... directly or transitively" — does not fire, so there is nothing for `producertest`/`WithProducer(...)` to be graded against |
| DOM-33 | Interface change updates every mock | N/A | No method added/removed/re-signed on any `Processor`/`Provider`/`Administrator` interface — `produceWhisperChatFunc` and the find seams are package-level `var`s, not interface methods |

## Seam pattern assessment (requested focus)

`produceWhisperChatFunc` (character_chat_whisper.go:49-54) is a package-level mutable
`var` wrapping `message.NewProcessor(l, ctx).WhisperChat(...)`, added by this diff and
following the exact shape of the four pre-existing seams in the same file
(`findCharacterByNameFunc`, `findLocalSessionFunc`, `findCharacterLocationFunc`, all
predating this diff and out of scope per the task-238 boundary) plus the file's own
cited precedent, `checkNameChangeValidityFunc` in
`cash_shop_check_name_change.go` (character_chat_whisper.go:41-44 comment).

No rule in `audit-checklist.md` (DOM-*, FILE-*, SUB-*, EXT-*, SCAFFOLD-*, SEC-*), and
neither foundational document (`patterns-functional.md`, `patterns-provider.md`),
addresses package-level mutable function-var seams as a category — grepping both
foundational documents and every detail document linked from the checklist for "seam"
returns zero hits. This is a real gap in rule coverage, not a disposition I can grade
PASS/FAIL against a specific rule; the pattern is neither sanctioned nor prohibited by
name anywhere in the current checklist.

What IS gradeable, and passes:
- It is not an interface, so DOM-33 does not apply to it.
- It is not a `Processor`/`Provider`/`Administrator` construction site, so DOM-06/DOM-11/
  patterns-provider.md's "Providers do NOT receive tenantId" rule do not apply.
- No test in the file uses `t.Parallel()` (grepped, zero hits), so the shared mutable
  package-level var carries no observed cross-test race risk, and every override is
  paired with a `t.Cleanup` restore (character_chat_whisper_test.go:590, 633,
  and pre-existing 115, 121, 128 for the find seams).
- As documented under DOM-24 above, the practical effect of this seam is that it
  fully satisfies the *intent* of DOM-24 (test never reaches the real Kafka-emitting
  call), even though the mechanism is neither of the two enumerated in DOM-24's pass
  criteria (`producertest.InstallNoop()` or a `WithProducer(...)` builder method). A
  service-local `noopWriter`/`testkafka` helper is explicitly called out as
  insufficient by the testing guide, but that helper still exercises real
  `ConfigWriterFactory`/producer wiring; this seam skips the production call
  entirely before it is ever made, which is a stronger guarantee for this
  narrow purpose than the disallowed helper, but is a different mechanism than
  either of the two the guide names.

Conclusion: this is a design choice with no numbered-rule verdict available, not a
finding. Flagging it here rather than silently passing it, per the audit's own
instruction not to enforce a rule that does not exist in the checklist.

## Not evaluable from the diff

None. Both changed files were read in full; every checklist item whose trigger fired
was settled from the diff plus one targeted read of `message/processor.go:84` (the
one-hop call `WhisperChat` makes into `producer.ProviderImpl`) and a grep of
`libs/atlas-packet/field/clientbound/whisper.go` to confirm what the `0x0A` argument
to `NewWhisperSendResult` represents.

## Summary

### Blocking (must fix)
- DOM-25: character_chat_whisper.go:233 and :237 — `fieldcb.NewWhisperSendResult(0x0A, targetName, false)` hardcodes the client-interpreted mode byte as a Go literal instead of resolving it from a tenant writer-options table.

### Non-Blocking (should fix)
- (none)
