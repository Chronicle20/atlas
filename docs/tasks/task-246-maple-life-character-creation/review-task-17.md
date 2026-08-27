# Review: Task 17 — Carry `ap` and `sp` on the character-creation saga contract

Commit range: `677b62c51..2f7faecf7` (single commit `2f7faecf7`).
Brief: `.superpowers/sdd/plan/task-17-brief.md`

## Scope confirmed

`git show --stat 2f7faecf7` touches exactly the 6 named sites (plus 3 test
files: `libs/atlas-saga/payloads_test.go`, `character/producer_test.go`,
`saga/handler_test.go`, `saga/processor_test.go`). No file under
`services/atlas-login/` appears in the diff — the `atlas-login`-not-touched
constraint holds.

```
 libs/atlas-saga/payloads.go                                       |  9 +++++
 libs/atlas-saga/payloads_test.go                                  | 43 ++++++++++++
 .../saga-orchestrator/character/mock/processor.go                 |  6 +--
 .../saga-orchestrator/character/processor.go                      |  6 +--
 .../saga-orchestrator/character/producer.go                       |  4 +-
 .../saga-orchestrator/character/producer_test.go                  | 29 ++++++++-
 .../kafka/message/character/kafka.go                               |  2 +
 .../saga-orchestrator/saga/handler.go                              |  2 +-
 .../saga-orchestrator/saga/handler_test.go                         |  2 +-
 .../saga-orchestrator/saga/processor_test.go                       |  4 +-
```

## 1. Propagation completeness

Traced `ap`/`sp` through all six named hops, field-for-field against the
`Gm`/`Meso` pattern:

1. `libs/atlas-saga/payloads.go:1044-1049` — `CharacterCreatePayload` gains
   `AP uint16 \`json:"ap,omitempty"\`` and `SP string \`json:"sp,omitempty"\``,
   immediately after `Meso`, same struct-tag style as `Gm`/`Meso`. PASS.
2. `kafka/message/character/kafka.go:~205-206` — `CreateCharacterCommandBody`
   gains `AP uint16 \`json:"ap,omitempty"\`` / `SP string \`json:"sp,omitempty"\`
   with matching tags. PASS.
3. `character/producer.go:250,276-277` — `RequestCreateCharacterProvider`
   appends `ap uint16, sp string` params and assigns `AP: ap, SP: sp` in the
   body literal. PASS.
4. `character/processor.go:46,214-215` — `Processor` interface member and
   `ProcessorImpl.RequestCreateCharacter` both append `ap uint16, sp string`
   and the impl forwards both into `RequestCreateCharacterProvider(...)`.
   PASS.
5. `character/mock/processor.go:41,250-252` — `ProcessorMock.RequestCreateCharacterFunc`
   field type and `RequestCreateCharacter` method both widened to 21 params,
   and the method forwards `ap, sp` into the stored func. PASS.
6. `saga/handler.go:1929` — `handleCreateCharacter` now passes
   `payload.AP, payload.SP` as the trailing two arguments to
   `h.charP.RequestCreateCharacter(...)`. PASS.

No site handles `Meso` but drops `ap`/`sp` — the six-hop chain is intact end
to end. Nothing downstream of `RequestCreateCharacterProvider` currently
reads `AP`/`SP` off the command body (that's Task 18/22 by design), which is
consistent with the task's stated boundary.

## 2. Omitted-case behaviour proven

`TestCharacterCreatePayloadCarriesApAndSp`
(`libs/atlas-saga/payloads_test.go:44-86`) has three sub-cases exactly per
the brief's table:

- both set → asserts `"ap":12` and `"sp":"3,0,0,0,0,0,0,0,0,0"` present in
  the marshaled JSON (`payloads_test.go:52-58`).
- round trip → unmarshal, assert `AP == 12`, `SP == "3,0,0,0,0,0,0,0,0,0"`
  (`payloads_test.go:60-68`).
- zero value → `CharacterCreatePayload{}` marshaled, asserts `"ap"` and
  `"sp"` substrings are **absent** (`payloads_test.go:70-85`). This is the
  test that would fail if `omitempty` were dropped from either tag (the
  zero-value uint16/string would then always serialize as `"ap":0` /
  `"sp":""`), so it genuinely proves the additive guarantee rather than just
  asserting it.

On the orchestrator side, `TestRequestCreateCharacterProviderCarriesApAndSp`
(`character/producer_test.go:31-56`) is a populated-case round trip only
(ap=12, sp set) — it does not itself prove the omitted-case, but the
existing `TestRequestCreateCharacterProvider_GmMeso` call site was updated to
pass `0, ""` for the new trailing params
(`character/producer_test.go:16`, diff: `..., 2, 12345, 0, ""`), and that
test's separate assertions on the produced body still pass with those zero
values — consistent with (not itself a proof of) `omitempty` continuing to
suppress the fields on the wire. The `libs/atlas-saga` test is the one that
actually exercises the JSON `omitempty` contract; that satisfies the brief's
bar since `CharacterCreatePayload` is the JSON-tagged type in this chain
(`CreateCharacterCommandBody` carries the same tags but is never asserted for
omission — non-blocking, noted below).

## 3. `SP` stays a `string`

`libs/atlas-saga/payloads.go`: `SP string` (json:"sp,omitempty"). `kafka.go`:
`SP string`. `producer.go`: `sp string` param, `SP: sp` assignment.
`processor.go`/`mock/processor.go`: `sp string` param throughout.
`handler.go`: `payload.SP` (already typed `string` on the payload). No site
introduces a numeric SP representation. PASS — matches design §11 A2/A4 and
the Task 19 schema.

## 4. Out-of-brief mechanical test widenings

- `saga/handler_test.go:866` — only the `RequestCreateCharacterFunc` literal
  signature widened (`gm int, meso uint32` → `..., ap uint16, sp string`);
  the function body and its `assert.Equal` calls are untouched in the diff.
  PASS — mechanical.
- `saga/processor_test.go:82,278` — same widening in two closures
  (`TestCharacterCreationSagaIntegration`,
  `TestCharacterCreationSagaCompensation`); both closure bodies (`return
  tt.characterCreationResult` and the compensation-failure branch) are
  unchanged in the diff. PASS — mechanical.

Both match the implementer's claim: signature-only, no assertion behaviour
changed.

## Verification

Ran module-local build/test per the brief (not `tools/verify.sh`):

```
cd libs/atlas-saga && go build ./... && go test ./...
→ ok  github.com/Chronicle20/atlas/libs/atlas-saga

cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...
→ all packages ok (saga, saga/mock, character, etc.)
```

Both exit 0, consistent with the report.

## Out of scope, correctly not raised

- `RequestCreateCharacterProvider`'s 21-parameter positional signature — the
  brief itself calls this out as noted-and-declined; not a finding here.
- No caller yet sets non-zero `ap`/`sp` — Task 22's job by design; not a
  finding here.

## Findings

None blocking.

Non-blocking:
- `CreateCharacterCommandBody.AP`/`SP` `omitempty` tags are asserted nowhere
  directly (only `CharacterCreatePayload`'s are). Since the wire shape of
  `CreateCharacterCommandBody` is produced by `json.Marshal` through the same
  Go `encoding/json` semantics and the struct tags are visually identical to
  `CharacterCreatePayload`'s, this is very low risk, but a future task adding
  a consumer of this body should not assume the omission is test-proven at
  that layer specifically.

## Not evaluable

None — the full six-site chain plus the two out-of-brief test files were all
within the diff and were traced directly.
