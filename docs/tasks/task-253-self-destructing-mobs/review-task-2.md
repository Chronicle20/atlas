# Review: Task 2 — dead-type constants and version-gated trailing int32 (MonsterDestroy)

Range reviewed: `1adaf6c9c..81c719bec` (single commit `81c719bec`).
Brief: `.superpowers/sdd/plan/task-2-brief.md`
Report: `.superpowers/sdd/plan/task-2-report.md`

## Scope confirmed

`git diff --stat 1adaf6c9c..81c719bec` shows exactly two files touched:

```
libs/atlas-packet/monster/clientbound/destroy.go      | 49 ++++++++++++++----
libs/atlas-packet/monster/clientbound/destroy_test.go | 60 ++++++++++++++++++++++
```

This matches the brief's file list exactly. `libs/atlas-packet/reactor/clientbound/spawn.go` and `libs/atlas-packet/test/context.go` show zero diff lines in this range (confirmed via `git diff` filtered to those two paths) — the read-only reference files were not touched.

## Spec compliance vs brief

1. **Constants** (`libs/atlas-packet/monster/clientbound/destroy.go:24-40`) — `DestroyTypeDisappear=0`, `DestroyTypeFadeOut=1`, `DestroyTypeBomb=2`, `DestroyTypeDestructByMiss=3`, `DestroyTypeSwallow=4`, `DestroyTypeSelfDestruct=5` all present with the exact doc comments and IDA addresses given verbatim in the brief's Step 3 code block. No address was invented; every one traces to a value the brief supplied. PASS.

2. **Gate function** (`destroy.go:42-49`) — `hasSwallowCharacterId(t tenant.Model) bool` returns `(t.IsRegion("GMS") && t.MajorAtLeast(92)) || t.Region() == "JMS"`, matching the brief's Step 3 code verbatim, and uses the `MajorAtLeast(N)` idiom (never a raw `>`/`<=`). PASS.

3. **Encode/Decode gating** (`destroy.go:87-108`) — both take `t := tenant.MustFromContext(ctx)` and gate the trailing int32 on `m.destroyType == DestroyTypeSwallow && hasSwallowCharacterId(t)`, matching Step 4 verbatim. PASS.

4. **`NewMonsterDestroyBySwallow` doc comment** (`destroy.go:63-68`) — updated to state "The trailing field is version-gated to GMS >= 92 / JMS; see hasSwallowCharacterId." Matches Step 4's instruction. PASS.

5. **Import** — `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` added (`destroy.go:11`). PASS.

6. **Tests** (`destroy_test.go`) —
   - `TestMonsterDestroySwallowGate` (lines ~99-124): table-driven over all 10 rows from the brief's table, with exact byte sequences matching (`89 13 00 00 04` for v48..v87, `89 13 00 00 04 39 30 00 00` for v92/v95/jms_v185). Verified this is a real discriminating fixture — v87 expects 5 bytes, v92 expects 9 bytes, straddling the gate boundary exactly where the brief places it. Each subtest also does `test.RoundTrip(t, ctx, input.Encode, out.Decode, nil)` for decode-symmetry / no-unconsumed-byte checking, matching brief instructions. PASS.
   - `TestMonsterDestroyNonSwallowTypesAreFiveBytes` (lines ~126-139): iterates `test.Variants × {FadeOut, DestructByMiss, SelfDestruct}`, asserts exactly 5 bytes `89 13 00 00 <dt>`, subtest name `fmt.Sprintf("%s/type%d", v.Name, dt)` — matches brief exactly. PASS.
   - The 9 `packet-audit:verify` markers (lines ~89-97) match the brief's list verbatim (same packet/version/ida triples), placed immediately above `TestMonsterDestroySwallowGate` as instructed. PASS.

7. **Build/tests actually pass** — ran `cd libs/atlas-packet && go build ./... && go test ./monster/clientbound/... -v`: all tests including pre-existing `TestMonsterDestroy`, `TestMonsterDestroyBytesV79`, `TestMonsterDestroyBytesV72`, `TestMonsterDestroyBytesV61`, `TestMonsterDestroyBySwallow` pass. PASS.

8. **gate-lint** — ran `go run ./tools/packet-audit gate-lint --check 2>&1 | grep -i destroy` — zero output, confirming no raw version comparison was introduced in this file. (Repo-wide `gate-lint --check` still fails on 38 pre-existing violations elsewhere, correctly out of this task's scope and correctly disclosed in the report rather than silently swallowed.) PASS.

## Global constraints check

- `MajorAtLeast(N)` idiom used, no raw `>`/`<=` — confirmed by reading `destroy.go:48` and by the scoped gate-lint grep. PASS.
- No invented IDA address: every address in the new comments (`destroy.go:25-40`, `42-47`) is present verbatim in the brief's Step 3 code block. Cross-checked line by line — no extra address was added that the brief did not supply. PASS.
- `libs/atlas-constants/` checked before introducing `DestroyType` — confirmed no existing `DestroyType`/`DeadType` constant exists there (`grep -ril` returned nothing); this is correctly a packet-local wire enum, not a shared domain constant. PASS.
- `spawn.go` / `test/context.go` unchanged — confirmed via scoped `git diff`, zero lines. PASS.
- Byte fixtures discriminate gated vs ungated: `TestMonsterDestroySwallowGate`'s `gms_v87` (5 bytes) vs `gms_v92` (9 bytes) genuinely differ across the exact gate boundary — this is not a fixture that would pass under both shapes. PASS.
- Immutable packet model, no stubs, no `// TODO`: confirmed by full read of `destroy.go` — no TODO/stub, and `Destroy` fields remain unexported with constructor functions, consistent with the pre-existing immutable-model pattern. PASS.
- Line endings: `grep -c $'\r'` on both files returns 0 for each — no CRLF introduced, consistent with a pre-existing LF-only file (edits were surgical, not full-file rewrites). PASS.

## Task quality

- The report accurately describes RED/GREEN evidence, though the RED step is described from pre-edit-file inspection rather than a captured failing-test run transcript; this is a minor process gap (the brief's Step 2 asked for an actual `go test -v` failing run) but the described failure mode is correct and verifiable by inspection of the prior file, and the final GREEN state was independently reproduced by this reviewer.
- The report's disclosure of the pre-existing repo-wide gate-lint violations (out of scope) rather than silently omitting it is good practice and matches repo conventions on scope discipline.
- Commit message and commit contents match the brief's Step 7 instruction (only the two intended files staged/committed).

## Not evaluable

- None. The full surface named in the brief (constants, gate function, Encode/Decode, tests, markers, gate-lint, build) was reviewable within this diff and its two reference files, and was reviewed directly.

## Verdict

Both spec compliance and task quality are satisfied with no blocking defects found.
