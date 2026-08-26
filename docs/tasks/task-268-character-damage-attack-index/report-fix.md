# Fix report — CharacterDamage truncation for mob attack index >= 1

Unit: `bug-character-damage-attack-index-truncation.md`
Agent: `task-implementer` (sonnet) · Status: **DONE** · Commit: `3c77762b7`

(Recorded by the controller: the implementer could not write this file itself — its
Write call was refused — so it returned the content and this is the transcription.
The commit and diff below were verified directly against the worktree, not taken on
the agent's word.)

## Change

`libs/atlas-packet/character/clientbound/damage.go`, both sides, verified in the diff:

```diff
-		if m.attackIdx == model.DamageTypePhysical || m.attackIdx == model.DamageTypeMagic {
+		if m.attackIdx >= model.DamageTypePhysical {
```

at line 52 (`Encode`) and line 75 (`Decode`). `damage.go:56` (the `>= 95` bGuard gate,
open item #1) was left untouched as instructed.

Diff stat, `main..3c77762b7`: 5 files, +179 / -2.

| File | Change |
|---|---|
| `character/clientbound/damage.go` | the predicate, both sides (+2 / -2) |
| `character/clientbound/damage_test.go` | `TestCharacterDamageMobAttackIndexIncludesBlock` (round trip across `test.Variants` at `attackIdx = 1`, plus a v83 byte-exact assertion), `TestCharacterDamageCounterOmitsBlock` (v83 byte-exact, `DamageTypeCounter = -2` still omits) |
| `character/clientbound/v61_test.go`, `v72_test.go`, `v79_test.go` | matching `...MobAttackIndex` and `...CounterOmitsBlock` byte fixtures per version |

Per-version fixtures reuse each file's existing physical-damage instruction addresses —
same code path, same client guard, only the `attackIdx` byte differs. No IDA addresses
were invented.

## Verification (implementer, module-local)

- RED: with `damage.go` stashed and the new tests kept,
  `TestCharacterDamageMobAttackIndexIncludesBlock` and the three
  `TestCharacterDamageByteOutputV{61,72,79}MobAttackIndex` fail. Notably the *round-trip*
  sub-tests still pass with the bug in place — which is precisely why the original
  suite never caught this. Only the byte-exact assertions fail.
- GREEN: `go test ./character/clientbound/... -count=1 -run TestCharacterDamage` — 12/12.
- Module: `go build ./... && go test ./...` in `libs/atlas-packet` — all ok.

Repo-wide `tools/verify.sh` was deliberately not run by the implementer; it runs
separately. **This report does not constitute the gate verdict.**

## Open item #2 — sweep result

Grep over `libs/atlas-packet` non-test sources for `DamageType*` comparisons:

```
model/damage_taken_info.go:19-23    const declarations
model/damage_taken_info.go:122      >= DamageTypePhysical   (already correct)
model/damage_taken_info.go:174      >= DamageTypePhysical   (already correct)
character/clientbound/damage.go:52  >= DamageTypePhysical   (fixed here)
character/clientbound/damage.go:75  >= DamageTypePhysical   (fixed here)
```

No other site re-derives the predicate as an equality pair. Nothing further to fix.

## Coverage matrix

`go run ./tools/packet-audit matrix` and `matrix --check` both exit 0 with no diff to
`docs/packets/audits/STATUS.md` / `status.json`. No fixture bytes changed and the
`DAMAGE_PLAYER` row was already ✅, so there is nothing to regenerate.

## Still open

- Open item #1 in the bug file: the `>= 95` stance-byte gate vs the v87 export. Needs
  the v87 IDB; out of scope here by ruling.
- Live re-test: a mob using attack index >= 1 while a second character stands in the
  same map. Not yet performed.
