# Backend Audit — task-105 party+buddy dispatcher-family migration

- **Scope:** Go diff `main..HEAD` — `libs/atlas-packet/{party,buddy}/clientbound` + `services/atlas-channel` call-sites
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-23
- **Build:** PASS (`go build ./...` clean in `libs/atlas-packet` and `services/atlas-channel/atlas.com/channel`)
- **Vet:** PASS (`go vet` clean on touched packages in both modules)
- **Tests:** PASS (`go test ./party/... ./buddy/...` ok; `socket/handler` ok; consumers have no test files — encode/decode covered in the lib)
- **Overall:** PASS

## Applicability note

This change touches `libs/atlas-packet` (packet codec structs) and atlas-channel
Kafka-consumer / socket-handler call-sites. It introduces NO new domain package
(`model.go`), no REST resource, no GORM administrator/provider, no Kafka emit in
the changed code. Most DOM-* checks (builder/ToEntity/Make/Transform/JSON:API/
administrator/provider/RegisterInputHandler/DB-write) are therefore N/A. The
relevant gates are: build/test (Phase 1), immutable-struct + no-stub conventions,
encode/decode symmetry, config-resolved mode bytes, version-gating idiom,
service-boundary, DOM-21 const reuse, and DOM-24 (no unstubbed emit in tests).

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Phase 1 | Build | PASS | both modules `go build ./...` clean |
| Phase 1 | Tests | PASS | `libs/atlas-packet/{party,buddy}/...` ok; `atlas-channel/socket/handler` ok |
| DOM-06/07 | Encoders take `logrus.FieldLogger` | PASS | error.go all `Encode(l logrus.FieldLogger, ...)`; call-sites pass `l` through `session.Announce(l)` |
| DOM-21 | Reuse existing consts/types | PASS | party dispatch keys on `partycb.PartyOperation*` (operation_body.go:27-41); buddy map keys on pre-existing `buddylist2.StatusEventError*` (kafka/message/buddylist/kafka.go:39, unchanged by branch); no shared type redeclared |
| DOM-24 | Kafka producer stubbed in emitting tests | N/A | lib tests are pure encoders; no `AndEmit`/`message.Emit`/`producer.Produce` in changed test files |
| Service boundary | atlas-channel must not import atlas-parties internals | PASS | party consumer (consumer.go:25,455-490) keys `partyErrorBody` switch on `partycb.PartyOperation*` consts from `libs/atlas-packet/party/clientbound`, NOT any atlas-parties import |
| Mode bytes config-resolved | No hard-coded operation mode bytes | PASS | every body func wraps `atlas_packet.WithResolvedCode("operations", PartyOperation*/BuddyOperation*, ...)`; `WriteByte(m.mode)` only writes the resolved mode; `WriteByte(0)` (buddy error.go:144/175/206/237) is the no-name extra-byte payload, not a mode byte |
| Encode/Decode symmetry | Decode mirrors Encode | PASS | each struct has both; round-trip test `TestBuddyErrorRoundTrip` (error_test.go:152) asserts clean RoundTrip per region; mode-only + {mode,name} arms mirror exactly |
| Version-gating idiom | `tenant.MustFromContext(ctx).IsRegion("GMS")` correct & consistent | PASS | buddy error.go:138-139 etc.; matches pre-existing idiom in `buddy/clientbound/invite.go:44`. Plain `IsRegion("GMS")` (not `&& MajorAtLeast`) is correct here — buddy mode table is byte-identical across all GMS majors; only JMS diverges (documented + IDA-verified) |
| Immutable structs / no setters | PASS | all error structs are private-field value types built via `New*(...)` ctors; no exported mutable fields, no fluent-mutation leak |
| Test fixtures assert real bytes | PASS | party error_test.go:150 `[]byte{mode,0x03,0x00,'B','o','b'}` (length-prefixed AsciiString); buddy error_test.go:137 `[]byte{c.mode,0x00}` GMS vs `[]byte{c.mode}` JMS |
| No `*_testhelpers.go`; Builder pattern | PASS | no testhelpers files added; tests use ctor `New*` + table-driven `t.Run` |
| No `// TODO`/stub/501 in landed lines | PASS | only TODO is invite/consumer.go:154 (buddy rejection), pre-existing from commit 2c43c0e9e — NOT in this branch's diff |
| Old catch-all removed | PASS | `PartyErrorBody`, buddy `Error` struct / `NewBuddyError` fully gone (grep returns NONE across libs + atlas-channel) |
| Operation keys resolve against seed templates | PASS | spot-checked v83/jms templates: `ALREADY_HAVE_JOINED_A_PARTY_1`, `UNABLE_TO_HAND_OVER_..._WITHIN_THE`, `AS_A_GM_..._A_PARTY`, `UNABLE_TO_FIND_THE_CHARACTER`, `BUDDY_LIST_FULL`, `UNKNOWN_ERROR_4`, jms `UNKNOWN_ERROR` all present |

## Findings

### Critical
None.

### Important
None.

### Minor (non-blocking, pre-existing — NOT introduced by this branch)

1. **party_operation.go:97-101 — missing `return` after `PartyUnableToFindCharacterBody` announce.**
   When `character.GetByName` fails, the handler announces the "unable to find
   character" notice but does NOT `return`; control falls through to the
   `GetByCharacterId(...)(cs.Id())` call with a zero-value `cs`, producing a
   second (UnableToFindInChannel) notice and an error log. The branch only
   swapped the body-func call (`PartyErrorBody(...)` → `PartyUnableToFindCharacterBody()`);
   the control-flow defect predates this task (verified via `git diff`). Out of
   scope to block, but a `return` after line 100 is the correct fix.

2. **invite/consumer.go:154 — `// TODO send rejection to requesting character`.**
   Pre-existing (commit 2c43c0e9e "Initiate buddy invitation"), not added by this
   branch. The CLAUDE.md no-TODO rule applies to *landed* new TODOs; this one is
   untouched by the diff. Flagged for visibility only.

## Bottom line

PASS. The migration is mechanically clean: 15 party + 9 buddy discrete per-mode
structs each map 1:1 to a config-resolved operation key (no hard-coded mode
bytes), encode/decode are symmetric and round-trip-tested, the version-gated
buddy extra-byte idiom is correct and consistent with the established pattern,
the dispatch maps reuse existing key consts without crossing the
atlas-channel↔atlas-parties service boundary, the legacy catch-all is fully
removed, and no new TODO/stub was landed. Build, vet, and tests are green in
both touched modules. The two Minor items are pre-existing and out of scope.
