# Task 26 review — `atlas-channel` MAKER_RESULT writer / craft terminal-event consumer

Commit range reviewed: `61ff8cbd8..3d4391dad` (single commit `3d4391dad`).
Brief: `.superpowers/sdd/plan/task-26-brief.md` (body + ADDENDUM), contract:
`.superpowers/sdd/plan/task-26a-brief.md` `### Interfaces`.
Module: `services/atlas-channel/atlas.com/channel`.

## Scope confirmed

`git diff --stat 61ff8cbd8..3d4391dad`:

```
kafka/consumer/maker/consumer.go       | 274 +++++++
kafka/consumer/maker/consumer_test.go  | 324 +++++++
kafka/message/saga/kafka.go            |  17 +
main.go                                |   6 +
saga/model.go                          |   9 +
```

No changes to `services/atlas-configurations/seed-data/templates/` or
`deploy/k8s/...` — matches the report's claim (verified independently, see
below). Diff matches the brief's file list exactly (minus the templates/
deploy files, which the report explains and I confirmed are genuinely
no-ops for this commit). Scope matches the unit under review; no mismatch.

## The one finding I was asked to rule on

**Ruling: acceptable as shipped. Non-blocking.**

The FAILED-path discriminator is `e.Body.SagaType == SagaTypeInventoryTransaction
&& e.Body.CharacterId != 0` (`kafka/consumer/maker/consumer.go:212-217`). I
traced the orchestrator side by hand rather than trusting the report:

- `StatusEventFailedBody` (`kafka/message/saga/kafka.go:79-86`) carries no
  `Results` map at all — there is structurally no `kind` marker available on
  this path, unlike COMPLETED. This is a design fact from Task 26a's own
  Interfaces section ("Not provided, deliberately… a compensated or timed-out
  saga never reaches `CompletedStatusEventProvider`"), not something Task 26
  could have added without reopening 26a.
- `EmitSagaFailed` (`services/atlas-saga-orchestrator/.../saga/producer.go:222-253`)
  checks `MtsOperation`, then `MesoSackUse`/`PetNameTagUse`, then
  `craftManifestCharacterId(s) != 0` (a craft-only, `RecordCraftManifest`-step-guarded
  helper) *before* falling through to `ExtractCharacterCreationIds` (which
  returns 0 for every saga with no `CreateCharacter` step). I confirmed
  `InventoryTransaction` is shared by four saga families (craft, Duey,
  note-gift-forward, pet-destroy: `producer.go:245-253` comment, corroborated
  by `SagaTypeInventoryTransaction`'s doc comment in `kafka/message/saga/kafka.go`)
  and that only the craft arm routes a nonzero `CharacterId` for that shared
  type today. `craftManifestCharacterId`'s own test
  (`services/atlas-saga-orchestrator/.../saga/producer_test.go:262-266` region)
  independently pins this contract from the producer side.
- Therefore, on this branch, `SagaType == InventoryTransaction && CharacterId
  != 0` is true if and only if the saga is a craft. The discriminator is
  correct today.

It is, as the implementer flagged, **not an enforced invariant** — it is an
inference from `EmitSagaFailed`'s current arm ordering. A future task that
gives Duey/note-gift-forward/pet-destroy their own `CharacterId` routing
(the same fix pattern 26a used) would silently make `handleCraftFailed`
misroute that saga's failure into a `MAKER_RESULT` sent to the wrong
character's session, with no compiler or test signal until it happens in
production.

Why I am not blocking on it:
1. The base brief's Step 3 (soft-resolve, `FAILED`-arm generality) and the
   ADDENDUM both direct Task 26 not to re-raise or re-implement the FAILED-path
   gap (U1) — the fix belongs to `atlas-saga-orchestrator` (adding a `Results`-shaped
   marker to `StatusEventFailedBody`, mirroring the COMPLETED `kind` key), which
   is out of scope for a channel-side consumer task.
2. The risk is documented in three independent places that a future
   implementer touching either side will find: the `SagaTypeInventoryTransaction`
   doc comment, `handleCraftFailed`'s doc comment, and the report's "Issues or
   concerns" section — this is exactly the kind of assumption CLAUDE.md's
   grounding requirement asks to be surfaced rather than silently absorbed.
3. Blocking this task would not fix the asymmetry — it would only force the
   same unresolved gap to be re-litigated with no new information, since the
   real fix requires touching Task 26a's producer.

**Does `TestHandleCraftFailed_NonCraftInventoryTransactionIsANoOp` constrain
the risk, or only document it?** Only document it. The test
(`consumer_test.go:297-308`) constructs a `StatusEventFailedBody` with
`SagaType: InventoryTransaction, CharacterId: 0` directly and asserts the
handler is a no-op. It proves the `CharacterId == 0` branch of the guard
behaves correctly — but it hardcodes today's *known* value for a non-craft
saga's failure. It cannot and does not fail if `EmitSagaFailed` were changed
tomorrow to route a nonzero `CharacterId` for, say, Duey — the test would
still pass unchanged because it never asserts anything about the producer's
actual behavior, only about what this handler does with a value the test
chose by hand. So it is a regression guard for `handleCraftFailed`'s own
logic, not a tripwire for the cross-service assumption underlying it. This
matches the review prompt's suspicion.

Recommendation (non-blocking, for a follow-up task if another
`InventoryTransaction`-routed saga is ever added): give `StatusEventFailedBody`
an explicit discriminator analogous to the COMPLETED path's `Results["kind"]`,
then delete the `CharacterId != 0` heuristic.

## Other review points

1. **COMPLETED-path discriminator and arm selection** — PASS.
   `resultKind(e.Body.Results) != sagamsg.MakerCraftResultKind` gates entry
   (`consumer.go:110-112`), never `SagaType`. Arm selection is `manifest.Mode`
   read off the decoded `CraftManifestPayload`, not off `SagaType` or the step
   list (`consumer.go:150-161`), matching the ADDENDUM's correction #3 and the
   26a Interfaces mapping (1→CREATE, 2→CREATE_WITH_UPGRADE, 3→MONSTER_CRYSTAL,
   4→DISASSEMBLE) verbatim.

2. **Nested `craftManifest` decode** — PASS. `decodeCraftManifest`
   (`consumer.go:179-196`) returns `(zero, false)` on a `nil` map, a missing
   key, or an `Unmarshal` failure — never panics, and the caller
   (`handleCraftCompleted:134-139`) writes the FAILED arm on `!ok` rather than
   a bogus success arm. Covered by
   `TestDecodeCraftManifest_MissingOrMalformed` (nil map, missing key, scalar
   value) and `TestDecodeCraftManifest_TolerateJSONFloat64` (round-trip
   fidelity). `json.Marshal` on an already-JSON-decoded `map[string]any` value
   cannot realistically fail, so the marshal-error branch is defensive rather
   than reachable — acceptable.

3. **Every terminal path writes exactly one result** — PASS. Table cross-check:
   completed mode 1-4 → matching arm (`craftResultBody`); unrecognized mode →
   FAILED (`craftResultBody`'s `default`); undecodable manifest → FAILED
   (`handleCraftCompleted:134-139`); compensated/timed-out → FAILED
   (`handleCraftFailed`, single code path per the orchestrator's shared
   `EmitSagaFailed`, confirmed both `compensator.go` and `timer.go` call it).
   `TestUnknownTerminalStateStillWritesAResult` and
   `TestTerminalEventWritesCorrectArm`'s "unrecognized mode" case both assert
   this directly. No path found that returns without writing when the event is
   genuinely a craft terminal event for a session on this channel pod.
   (Early returns for tenant mismatch, `characterId == 0`, session-not-found,
   and cross-pod `ChannelId()` mismatch are all legitimate skips — the event
   is not this pod's to answer, mirroring the identical pattern in
   `kafka/consumer/saga/consumer.go:141-146,174-179,253-259`.)

4. **Soft-resolve on missing `operations` table/key** — assessed, not a
   defect, despite the brief's Step 3 language. `mts/consumer.go`'s
   `failNoticeOr`/`noticeFailReasonCode` idiom exists to soft-resolve a
   genuinely *optional*, per-tenant secondary table (`noticeFailReasons`) that
   not every MTS deployment configures. The MakerResult writer's `operations`
   table (`CREATE`/`CREATE_WITH_UPGRADE`/`MONSTER_CRYSTAL`/`DISASSEMBLE`) is
   not optional in the same sense: it is dispatcher-family-mandatory,
   generated uniformly across all eight supported version templates by
   `packet-audit operations` with no version gate
   (`docs/packets/dispatchers/maker_result.yaml`), and CI enforces completeness
   via `packet-audit operations --check` (I re-ran it: `operations check OK (0
   absent-writer note(s))`). `MakerResultFailedBody` itself never resolves a
   code at all (`libs/atlas-packet/character/maker_result_body.go:53-73`), so
   the FAILED arm this consumer writes on every fallback path is immune to the
   99-sentinel regardless. The residual risk (a tenant's runtime options
   diverging from its seed template) is a deployment-integrity concern shared
   by every other `WithResolvedCode` call site in the codebase, not something
   unique to or fixable within this consumer. Not a defect; the brief's
   generic Step-3 language does not fit this writer's actual table shape.

5. **In-flight guard not released here** — PASS, verified by grep across the
   full diff for `ReleaseInFlight`/`InFlight`/`guard`: no match in
   `kafka/consumer/maker/consumer.go` or anywhere else in this diff.
   `atlas-maker`'s own consumer retains sole ownership
   (`services/atlas-maker/.../kafka/consumer/saga/consumer.go:59,74`,
   unchanged by this commit).

6. **`resultKind`/`resultUint32` duplication** — non-blocking. Verified
   byte-for-byte identical to `kafka/consumer/saga/consumer.go`'s unexported
   versions. Acceptable: both are 3-8 line pure functions, unexported in the
   source package (Go package boundaries genuinely prevent reuse without a new
   shared internal package), and the repo already tolerates small per-package
   duplicated sentinels in this same family (e.g. `mtsSagaFailureReason` in
   `kafka/consumer/saga/consumer.go:82`). Not a blocking finding, but if a
   third package needs the same helpers, extraction becomes worth it.

7. **Gates re-run independently (not trusted from the report):**
   - `go build ./...` (module root) — clean, no output.
   - `go test ./... -count=1` — all packages `ok`/`[no test files]`, zero
     `FAIL` (grepped explicitly).
   - `tools/lint.sh --go services/atlas-channel/atlas.com/channel` →
     `0 issues. / lint.sh: OK`.
   - `gofmt -l` on all five changed files — no output (clean).
   - `go run ./tools/packet-audit operations --check` →
     `operations check OK (0 absent-writer note(s))`.
   All match the report's claims.

## Correct-as-shipped, not filed as defects (per the review brief's explicit list)

- `DisassembleMesoCostCharge` reported as `0` — verified `craftResultBody`'s
  mode-4 arm passes `manifest.MesoCost` straight through with no formula
  (`consumer.go:158`); the zero originates upstream in
  `services/atlas-maker/.../craft/processor.go:33-46`, out of this diff's
  scope.
- `NoItemAwarded` carried through verbatim to `MakerResultCreateBody`/
  `MakerResultCreateWithUpgradeBody`, no test asserts a `true` case — correct,
  no producer exists on this branch.
- No template or deploy/env changes in this commit — confirmed both are
  genuinely unnecessary here (writer entry pre-existing per `packet-audit
  --check`; `EVENT_TOPIC_SAGA_STATUS` already present in
  `deploy/k8s/base/env-configmap.yaml` and both overlays' `configMapGenerator`
  literals, grepped directly).

## Not evaluable

- The correctness of `atlas-saga-orchestrator`'s `EmitSagaFailed`/
  `craftManifestCharacterId` implementation itself is Task 26a's surface, not
  this unit's — I traced it only far enough to confirm the contract Task 26
  depends on, per the review brief's instruction to check whether the
  discriminator's grounding holds. I did not re-review 26a's own diff for
  defects outside that seam.
- Client-side behavior on receipt of `MAKER_RESULT` (whether the client
  actually unlocks its UI on every `nResult` value) is asserted only by
  brief-cited design references (`design §4.3.2 (C-1)`), not independently
  re-verified against client source/IDA in this review — outside this
  consumer's diff surface.

## Verdict

APPROVED_WITH_FINDINGS — one non-blocking design-debt finding (the FAILED-path
discriminator) and one non-blocking style note (helper duplication). No
blocking defects found; the brief's requirements are met, arm selection is
correct, no silent terminal path exists, the in-flight guard is not touched,
and all independently re-run gates pass clean.
