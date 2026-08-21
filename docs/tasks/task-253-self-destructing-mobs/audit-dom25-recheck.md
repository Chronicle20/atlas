# Backend Audit (Re-check) — DOM-25 fix, task-253

- **Service Paths:** `services/atlas-monsters`, `services/atlas-channel`, `services/atlas-configurations` (data-only)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Scope:** commit `6f986eb88` (range `77fc9bc64..6f986eb88`), per `.superpowers/sdd/plan/fix-dom25-brief.md`
- **Build:** PASS (both modules)
- **Tests:** PASS — `atlas-monsters` (all packages ok), `atlas-channel` (all packages ok)
- **Overall:** PASS — both prior blocking findings closed, no new violation found

## Build & Test Results

```
cd services/atlas-monsters/atlas.com/monsters && go build ./...   -> clean, no output
cd services/atlas-monsters/atlas.com/monsters && go test ./... -count=1
  ok atlas-monsters, atlas-monsters/monster (21.650s), atlas-monsters/monster/information (15.554s),
  atlas-monsters/kafka/consumer/monster, atlas-monsters/kafka/consumer/buff, etc. — all ok, no failures.

cd services/atlas-channel/atlas.com/channel && go build ./...    -> clean, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1
  ok across all packages incl. atlas-channel/socket/writer, atlas-channel (kafka consumer/monster covered
  via top-level `atlas-channel` package tests) — no failures.
```

## Finding (a) re-check — raw wire code on the Kafka contract

**CLOSED.**

- `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:66-72` — `DeathType*` are now string
  constants (`DeathTypeUnset = ""`, `DeathTypeDisappear = "DISAPPEAR"`, … `DeathTypeSelfDestruct =
  "SELF_DESTRUCT"`), following the `CatchCause*` pattern in the same file (comment at kafka.go:57-65
  cites it explicitly).
- `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:113` (`statusEventCreatedBody.DeathType
  string`) and `kafka.go:156` (`statusEventKilledBody.DeathType string`) — both fields are now `string`,
  no `byte` left.
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1875-1896` — new `deathTypeForAction`
  maps the WZ `selfDestruction.action` byte to its semantic key **at the point the byte is read**
  (`processor.go:695`, `processor.go:1866` — both call sites of `selfDestructFrom` go through it), not
  downstream. An action outside 0..5 is logged (`processor.go:1893`) and falls back to
  `DeathTypeFadeOut` (`processor.go:1894`) rather than inventing a key — matches the brief's explicit
  instruction.
- `services/atlas-monsters/atlas.com/monsters/monster/producer.go:31,164` — `destroyedStatusEventProvider`
  and `killedStatusEventProvider` now take `deathType string`.
- `services/atlas-monsters/atlas.com/monsters/monster/processor_catch.go:170` — the one other call site
  (`CatchCause`/claim path) passes `DeathTypeUnset` (now `""`), not a raw byte.
- Swept `grep -rn "DeathType"` over `services/atlas-monsters/.../monster/*.go` (non-test): no remaining
  `byte`-typed `DeathType` field or raw-value assignment anywhere in the package.
- Downstream mirror: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:196`
  (`StatusEventCreatedBody.DeathType string`) and `kafka.go:238` (`StatusEventKilledBody.DeathType
  string`) — both fields changed from `byte` to `string`, comments updated to describe the semantic key
  (kafka.go:189-195, kafka.go:230-236).
- Swept `grep -rl "deathType"` across `services/` outside `atlas-monsters`/`atlas-channel`: no other
  service consumes this field — no missed downstream contract holder.

## Finding (b) re-check — bare-cast instead of resolved lookup

**CLOSED.**

- `services/atlas-channel/atlas.com/channel/socket/writer/monster_destroy.go` (new file) — follows the
  `claim_result.go` shape exactly:
  - `DestroyMonsterCode string` type with doc comment naming the table (`monster_destroy.go:9-16`),
    mirroring `ClaimResultCode` at `claim_result.go:9-16`.
  - Six named constants `DestroyMonsterDisappear`..`DestroyMonsterSelfDestruct`
    (`monster_destroy.go:18-25`).
  - `DestroyMonsterBody(uniqueId, key)` routes through `atlas_packet.WithResolvedCode("operations",
    string(key), func(code byte) packet.Encoder {...})` (`monster_destroy.go:27-30`), same call shape as
    `ClaimResultNoticeBody` at `claim_result.go:29-33`.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:216-221`
  (`destroyCodeFor`) no longer bare-casts; it returns a `writer.DestroyMonsterCode`, with `""` mapped
  explicitly to `writer.DestroyMonsterFadeOut` (line 219-220).
- Both call sites route through it: `consumer.go:201` (DESTROYED, `destroyCodeFor(e.Body.DeathType)`)
  and `consumer.go:312` (KILLED, same). Both then call `writer.DestroyMonsterBody(uniqueId, code)`
  (`consumer.go:228`, `consumer.go:326`), not the old `monsterpkt.NewMonsterDestroy(...).Encode`.
- `libs/atlas-packet/monster/clientbound/destroy.go` is untouched (confirmed via `git diff
  77fc9bc64..6f986eb88 -- libs/atlas-packet` — no hunk), as the brief required.

### Seed templates — six-entry `operations` table

All 11 templates carry an identical six-entry table on the `DestroyMonster` writer (verified by
parsing each file's `socket.writers[].options.operations` where `writer == "DestroyMonster"`):

```
DISAPPEAR: 0, FADE_OUT: 1, BOMB: 2, DESTRUCT_BY_MISS: 3, SWALLOW: 4, SELF_DESTRUCT: 5
```

confirmed identical across `template_gms_12_1.json`, `_48_1`, `_61_1`, `_72_1`, `_79_1`, `_83_1`,
`_84_1`, `_87_1`, `_92_1`, `_95_1`, `template_jms_185_1.json` — 11/11, no template missing the table,
no value divergent. Example diff shape: `template_gms_12_1.json:746-754` (new `"options":
{"operations": {...}}` block inserted directly after the existing `"writer": "DestroyMonster"` /
`"fname"` keys, same position pattern as other resolved writers in the same file).

## Rolling-deploy zero-value property

**Verified, and asserted by a test.**

- `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:66` — `DeathTypeUnset = ""`, doc comment
  at kafka.go:63-65 states it is "a documentation alias for `\"\"` — the JSON zero value for a string
  field — not a distinct wire state."
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:218-220` —
  `destroyCodeFor` explicitly special-cases `deathType == ""` and returns `writer.DestroyMonsterFadeOut`
  before falling through to `writer.DestroyMonsterCode(deathType)`.
- Test assertion: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go`
  (post-fix `TestDestroyCodeFor`) has the explicit case `{"producer omitted the field", "",
  writer.DestroyMonsterFadeOut}` and calls `destroyCodeFor(tt.deathType)`, asserting equality against
  `writer.DestroyMonsterFadeOut` — this is a real assertion of the empty-string fallback, not merely
  "no error."
- Companion decode test `TestStatusEventKilledBodyDecodesDeathType` (same file) asserts that a JSON
  body omitting `deathType` decodes to `""` (`withoutField.DeathType != ""`), confirming the Go
  `encoding/json` zero-value path end to end, not just the resolver function in isolation.

## Wire-byte identity re-check

**Verified as a genuine byte-identity assertion, not a non-error check.**

- `services/atlas-channel/atlas.com/channel/socket/writer/monster_destroy_test.go`
  (`TestDestroyMonsterBodyResolvesCode`, lines 28-50) builds `DestroyMonsterBody(1234,
  tt.code)(l, reportTestContext(t))(destroyMonsterTestOptions)` and asserts `bytes.Equal(actual,
  tt.want)` against explicit byte slices:
  - ordinary death (`DestroyMonsterFadeOut`) → `[]byte{0xD2, 0x04, 0x00, 0x00, 0x01}` — trailing byte
    `0x01` matches the pre-existing `DestroyTypeFadeOut DestroyType = 1` constant in
    `libs/atlas-packet/monster/clientbound/destroy.go:29`.
  - self-destruct (`DestroyMonsterSelfDestruct`) → `[]byte{..., 0x05}` — matches
    `DestroyTypeSelfDestruct = 5` (`destroy.go:41`).
  - all four remaining codes (bomb=2, destruct-by-miss=3, swallow=4, disappear=0) also asserted, each
    matching the corresponding untouched `DestroyType` constant in `destroy.go`.
  - Leading four bytes `0xD2 0x04 0x00 0x00` are `1234` little-endian (`0x04D2`), matching
    `Destroy.Encode`'s `w.WriteInt(m.uniqueId)` at `destroy.go:90` followed by
    `w.WriteByte(byte(m.destroyType))` at `destroy.go:91` — this test exercises the real, unmodified
    codec via the new resolver, so byte-for-byte identity with pre-task-253 behaviour is directly
    demonstrated, not inferred.
- `destroyMonsterTestOptions` (`monster_destroy_test.go:13-22`) uses the same six values as the seed
  templates (`DISAPPEAR:0` … `SELF_DESTRUCT:5`), so the test table and the deployed config table agree.

## New violations introduced by the fix

None found.

| Check | Status | Evidence |
|---|---|---|
| FILE-* (new `monster_destroy.go` scoped correctly) | PASS | Single writer-body file in `socket/writer/`, same shape/package as `claim_result.go`; no domain logic added. |
| CONSTANTS reuse (`libs/atlas-constants`) | N/A | `grep -rl` over `libs/atlas-constants` for `DeathType\|DestroyType\|FADE_OUT\|SELF_DESTRUCT` returns no match — no pre-existing shared constant was bypassed. |
| Gate-lint raw-comparison (new site) | N/A | `libs/atlas-packet/monster/clientbound/destroy.go` untouched (confirmed via `git diff` — no hunk in that path); `destroyCodeFor`'s `deathType == ""` is a string-equality check on the new semantic-key contract, not a raw wire-code comparison bypassing resolution — no new site in `libs/atlas-packet`. |
| MESSAGING (contract change reaches every consumer) | PASS | `grep -rl "deathType"` across `services/` outside `atlas-monsters`/`atlas-channel` returns no match — no other service decodes this field. |
| TESTING (existing tests updated to match new contract) | PASS | `producer_test.go`, `self_destruct_test.go`, `self_destruct_dot_test.go`, `self_destruct_timer_test.go`, `consumer_test.go` all updated to assert the string key end to end (see wire-byte and rolling-deploy sections above). |
| Design doc D2.2 update (per brief, evidence not deleted) | PASS | `docs/tasks/task-253-self-destructing-mobs/design.md:96-109` (post-implementation update block) records the uniformity ruling and DOM-25/task-102/task-103 rationale; the pre-existing IDA evidence text above it is untouched (`git diff` shows a pure insertion, no deletion in §2.2). |

## Not evaluable from the diff

none

## Summary

### Blocking (must fix)
- none

### Non-Blocking (should fix)
- none
