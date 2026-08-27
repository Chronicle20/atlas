# Task 14 review — CharacterAppearanceUpdate / CharacterData ring coverage

Range reviewed: `ccfd35e81..9df12f9d0` (3 files, +124/-25):
- `libs/atlas-packet/character/clientbound/appearance_update_test.go`
- `libs/atlas-packet/field/clientbound/set_field_test.go`
- `docs/packets/evidence/gms_v72/character.clientbound.CharacterAppearanceUpdate.yaml`

## Priority 1 — Ruling 1

**Conclusion: Ruling 1's premise is false. It should be recorded as WITHDRAWN
(never-applicable), not "discharged."**

1. `decompile_sha256` hashes the IDA export's JSON entry for the cited
   function — nothing about Atlas's own encoder output. Confirmed at
   `tools/packet-audit/internal/evidence/hash.go:14-38`
   (`FunctionHash(exportPath, fname)` reads `docs/packets/ida-exports/<version>.json`'s
   `functions[fname]`, canonicalizes the JSON, and SHA-256s it) and
   `tools/packet-audit/internal/matrix/evidence_input.go:29-30`
   (`h, err := evidence.FunctionHash(exp, r.IDA.Function); fresh := h == r.IDA.DecompileSHA256`).
   The only way to stale this hash is for the *client decompile* to change —
   i.e. a new/edited IDA export file — never a change to Atlas's Go encoder.
   Task 6 changed `appearance_update.go`'s Go code, not
   `docs/packets/ida-exports/*.json`, so it structurally cannot have staled
   any `CharacterAppearanceUpdate` evidence hash. Ruling 1's premise
   ("Task 6's byte change staled the pinned evidence") conflates the wire
   codec with the citation target.

2. `grep -c "packet-audit" tools/verify.sh` → `0`. The flagless `tools/verify.sh`
   contains zero references to `packet-audit`, and none of its sourced guard
   scripts (`tools/*.sh`, enumerated and checked) reference it either. The
   packet matrix check runs only in CI, via `.github/workflows/packet-matrix.yml`
   (`cd tools/packet-audit && go test ./...` plus a separate fname-doc-check
   step). So even if the premise in point 1 were true, it would never have
   "surfaced for the first time at the branch-end flagless `tools/verify.sh`"
   — that gate does not run this check at all.

3. Ran `go run ./tools/packet-audit matrix --check`:
   ```
   note	n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
   note	n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
   ```
   Exit code 0. No hash-drift, dangling, or citation-unresolvable problem
   reported for any packet. Cross-checked `docs/packets/audits/STATUS.md`
   directly: `UPDATE_CHAR_LOOK` (line 264) and `SET_FIELD` (line 180) both show
   ✅ at all eight target-version columns and ⬜ (n-a, correctly not attempted)
   at gms_v48. All 8 evidence YAMLs under `docs/packets/evidence/*/character.clientbound.CharacterAppearanceUpdate.yaml`
   were inspected directly; each carries a `decompile_sha256` and `matrix --check`
   accepts all eight as fresh.

The implementer's report already states the honest finding ("no stale-hash
drift was actually present at re-pin time") but still frames Ruling 1 as
"discharges here," which is the wrong disposition per the controller's own
distinction. This is a **process/bookkeeping finding, not a code defect** —
non-blocking, but it should be corrected in whatever tracks Ruling 1's status
so the branch-end verify.sh is not treated as carrying a live risk it never
carried.

## Priority 2 — frame-shrink predates the range

Confirmed by history, not by trusting the report:
- `git merge-base --is-ancestor 3c4d49bcc ccfd35e81` and same for `00dc42a52`
  both succeed — both commits (`fix(atlas-packet): correct CharacterAppearanceUpdate
  frame and drive its ring blocks` and `fix(atlas-packet): restore gms_v87/v95
  trailing completed-set int...`) are ancestors of the range start, i.e. Task 6
  itself, not this task.
- `git show --stat 3c4d49bcc` touches `appearance_update.go`, `appearance_update_test.go`,
  `v61_test.go`, `v72_test.go` — this is where the trailing-int removal and
  literal frame re-derivation for v61/v72 actually happened.
- The `ccfd35e81..9df12f9d0` diff does not touch `baseFrameHex`, `v61_test.go`,
  or `v72_test.go` at all (confirmed via `git diff --stat`), and does not
  perform any literal-minus-4-bytes arithmetic — it only replaces the
  `EncodeField`-on-want-side calls with new hex constants
  (`ringEmptyHex`/`ringPopulatedGMSHex`/`ringPopulatedJMSHex`), leaving
  `baseFrameHex` and the trailing-int gating (`hasTrailingCompletedSetItemId`)
  untouched. Step 1's forbidden shortcut was not taken.

## Priority 3 — A3 tautology retirement, fault-injected

Confirmed the six shared-fixture versions (`v79/v83/v84/v87/v95/jms_v185`,
via `runAppearanceUpdateCases`) now build `want` from captured hex literals
(`ringEmptyHex`, `ringPopulatedGMSHex`, `ringPopulatedJMSHex`), not
`model.RingSet.EncodeField` on both sides
(`appearance_update_test.go:63-155`).

Fault injection 1 — `model/ring.go:74`, `encodePair`'s nil-arm
`w.WriteByte(0)` → `w.WriteByte(9)`:
- `go test ./character/clientbound/...` went RED on every
  `TestCharacterAppearanceUpdateByteOutput*` test (v61, v72, v79, v83, v84,
  v87, v95, JMS) — all eight columns caught the perturbation.
- `TestSetFieldRoundTripPopulatedRings` in the same run stayed green. This is
  **correct, not a coverage gap**: `CharacterData`'s ring records go through
  `model.RingRecords.EncodeRecords`/`DecodeRecords` (a structurally different
  count-prefixed record-list codec, `model/ring.go:263-283`), never through
  `RingSet.EncodeField`'s nil-arm flag byte. This is exactly Rulings 10-13's
  deliberate non-unification of the AVATAR block (`RingSet`) and the RECORD
  block (`RingRecords`).
- Reverted (`git checkout -- libs/atlas-packet/model/ring.go`); reran both
  packages green.

Fault injection 2 — targeting `CharacterData`'s own codec, `model/ring.go`
`MarriageRecord.encode`, swapped the `GroomId`/`BrideId` write order (decode
order left unchanged, breaking encode/decode symmetry):
- `TestSetFieldRoundTripPopulatedRings` went RED on 11 of 12 variants (all
  except the legacy `GMS_v28` case, which correctly doesn't exercise the
  marriage arm at all).
- Reverted; reran, green.

`git status --porcelain` confirmed clean after each revert (no residual diff
to `model/ring.go`). Both fixtures are live coverage against their respective
codec paths — the earlier both-sides tautology is genuinely retired for
`CharacterAppearanceUpdate`, and the new `CharacterData` populated-Rings case
independently proves live for its own (deliberately separate) codec.

**Non-blocking finding** — stale comment: `appearance_update_test.go:278-280`
(inside the JMS test's doc comment) still reads:

> the ring block wire for JMS carries the extra per-arm entry-count int
> (model.RingSet.EncodeField, isJMS branch), which runAppearanceUpdateCases
> builds via the production codec rather than re-transcribing here.

This is no longer true — `runAppearanceUpdateCases` now builds the JMS ring
bytes from the `ringPopulatedJMSHex`/`ringEmptyHex` captured literals, not by
calling the production codec, which is the entire point of this task's A3
retirement. The comment directly contradicts the file's own new doc comment
at lines 55-62 and could mislead a future maintainer into believing the
tautology still exists on the JMS path. Should be corrected to say the
literal is independently derived, matching the v79/v83/v84/v87/v95 comments.

## Priority 4 — scope and CharacterData discipline

- Codec files not touched: `git diff ccfd35e81..9df12f9d0 -- appearance_update.go set_field.go model/ring.go` is empty. Confirmed no wire-format change riding along with the test-only diff.
- `TestSetFieldRoundTripPopulatedRings` (`set_field_test.go:316-373`, pure
  append, `+73/-0`) is a round-trip test, not a byte-literal fixture — it
  invents no per-field decompile citation, and its doc comment
  (`set_field_test.go:305-315`) states plainly that "CharacterData carries no
  per-field decompile citation of its own" and that the ring records are
  "verified here via the shared production codec... rather than a per-field
  decompile line," consistent with the existing `set_field_test.go:33-34`
  OPAQUE_LEDGER caveat.
- Ruling 10-13 compliance: the new `CharacterData` test uses
  `model.RingRecords{Couple: []model.CoupleRecord{...}, Friend:
  []model.FriendRecord{...}, Marriage: []model.MarriageRecord{...}}` with
  `GroomId`/`BrideId` field names (the RECORD block) — correctly NOT the
  `CharacterAppearanceUpdate` AVATAR block's `RingSet`/`PairRing`/
  `MarriageRing`/`MarriageCharacterId`/`PartnerCharacterId`. No cross-wiring
  found.
- Ruling 5 (marriage avatar arm's own-id-first ordering) is not exercised by
  this diff at all — `runAppearanceUpdateCases`'s populated case only
  populates the couple arm (matching the existing spawn_test.go precedent
  this file inherited); no marriage-arm literal was added or touched, so
  Ruling 5 is neither violated nor newly tested here.
- Marker sweep: `grep packet-audit:verify` across
  `appearance_update_test.go`/`v61_test.go`/`v72_test.go` shows exactly 8
  `CharacterAppearanceUpdate` markers (no duplicates), and
  `set_field_test.go` shows exactly 8 `FieldSetField` markers. `git diff` of
  the range contains zero `packet-audit:verify` lines — no marker was added,
  moved, or duplicated, as required.
- `docs/packets/audits/STATUS.md` / `status.json`: zero diff in range
  (`git diff ccfd35e81..9df12f9d0 -- docs/packets/audits/` is empty) —
  matrix output was already current; no cell regressed.
- `docs/tasks/task-269-ring-pair-behavior/`: zero diff in range
  (`git diff --stat ... -- docs/tasks/` empty) — nothing under that path was
  committed by this task.
- `tools/lint.sh --go libs/atlas-packet` → `0 issues.` / `lint.sh: OK`.

## Verdict rationale

No blocking defects found. Task 14 does the substantive work it claims: A3's
tautology is genuinely retired and independently fault-injection-verified on
both codec paths, Step 1's frame-shrink is confirmed pre-existing (not
skipped, not shortcut), the evidence re-pin is confirmed byte-identical
except cosmetic YAML formatting, and scope discipline (no codec edits, no new
markers, correct RECORD vs AVATAR block usage, no docs/tasks commits) all
hold up under inspection rather than just trusting the report.

Two non-blocking items:
1. Ruling 1 should be recorded as **withdrawn as never-applicable**, not
   "discharged" — the premise (Task 6's Go-code byte change could stale an
   IDA-decompile hash) is structurally false, and the guard it worried about
   was never in `tools/verify.sh`'s flagless path to begin with.
2. Stale/contradictory comment at `appearance_update_test.go:278-280`
   claiming the JMS ring block is still built via the production codec,
   when it is now a captured literal.

## Not evaluable

None — the full diff surface (3 files) was read, the cited codec contracts
(`model/ring.go`, `tools/packet-audit/internal/evidence/hash.go`,
`tools/packet-audit/internal/matrix/evidence_input.go`, `tools/verify.sh`)
were read and the relevant commands (`matrix --check`, targeted `go test`,
`lint.sh`, fault-injection reverts) were run directly.

---

verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-269-ring-pair-behavior/reviews/task-14.md
scope_confirmed: reviewed the full diff `ccfd35e81..9df12f9d0` (appearance_update_test.go, set_field_test.go, gms_v72 evidence YAML); traced into model/ring.go, appearance_update.go, set_field.go (confirmed untouched), tools/packet-audit/internal/{evidence,matrix} for Ruling 1, and tools/verify.sh for the verify-gate claim
blocking: 0
non_blocking: 2
  - libs/atlas-packet/character/clientbound/appearance_update_test.go:278-280 — stale comment claims the JMS ring block is "built via the production codec" when this task changed it to a captured hex literal (ringPopulatedJMSHex/ringEmptyHex); contradicts the file's own new doc comment at lines 55-62.
  - Ruling 1 (docs/tasks/task-269-ring-pair-behavior tracking) — should be recorded as withdrawn/never-applicable rather than "discharged": decompile_sha256 hashes the IDA export JSON only (tools/packet-audit/internal/evidence/hash.go:14-38), so Task 6's Go-encoder byte change could never have staled it, and tools/verify.sh (flagless) never invokes packet-audit at all (0 matches for "packet-audit"; the check runs only in .github/workflows/packet-matrix.yml).
not_evaluable: 0
