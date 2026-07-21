# task-181 v79 template audit — completeness verification

**Audit date:** 2026-07-20
**Branch:** task-181-v79-template-audit (verified via `git branch --show-current`)
**Worktree:** `.worktrees/task-181-v79-template-audit` (operated in-place; no files mutated other than this report)

## Scope

Deliverable set audited (no formal plan.md; per task instructions the deliverables are the claims in `docs/tasks/task-181-v79-template-audit/{audit.md,codec-defects.md,writer-routing.md,divergent-writer-recipe.md}`):

1. All 30 writers named in `writer-routing.md`/`codec-defects.md` (18 route-safe + 12 divergent) routed in `template_gms_79_1.json` AND showing ✅ in the coverage matrix for v79.
2. The 6 false-pass codec fixes corrected across ALL applicable versions (v79/83/84/87/95/jms), not just v79.
3. `matrix --check`, `dispatcher-lint`, and `go test -race ./field/... ./monster/carnival/...` clean.
4. Any claim in the docs not actually supported by the code/matrix.

## Result summary

| # | Deliverable | Verdict |
|---|---|---|
| 1 | 30/30 writers routed in template; matrix ✅ | **29/30 — FAIL on WeddingProgress** |
| 2 | 6 false-pass defects fixed across all 6 versions | **PASS** |
| 3 | Gates (matrix --check, dispatcher-lint, go test -race) | **PASS (all exit 0)** |
| 4 | Docs vs. code/matrix consistency | **One false claim found** (WeddingProgress "Routed" list implies ✅; matrix says n-a) |

---

## 1. Template routing — 30/30 present, opcodes correct, array sorted

Verified programmatically against `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`:

- All 30 writer names (18 route-safe + 12 divergent, enumerated from `writer-routing.md` lines 16–59) are present in `socket.writers` at their documented opcodes — exact match for every one (`FieldTransportState`@0x8D … `MtsOperation`@0x144).
- `socket.writers` (193 entries) is stable-sorted ascending by numeric opcode — confirmed programmatically (`ops[i] <= ops[i+1]` holds for the whole array).

Routing itself is 30/30 correct. This part of the deliverable is DONE.

## 2. Coverage-matrix promotion — 29/30 ✅, 1 silently NOT promoted

Cross-checked every one of the 30 writers' matrix rows in `docs/packets/audits/STATUS.md` / `status.json` for the `gms_v79` column:

- 29 of 30 show `state: verified` / ✅ for gms_v79, with a resolved opcode matching the template (spot-checked via grep + `status.json` walk).
- **`WeddingProgress` does NOT.** `status.json` op `WEDDING_PROGRESS`:
  ```json
  "gms_v79": { "state": "n-a", "opcode": -1 }
  ```
  `docs/packets/audits/STATUS.md:474` confirms the same (`v79#` empty, `v79` = ⬜).

**Root cause (file:line evidence):**
- The writer IS routed in the template: `template_gms_79_1.json` → `{"opCode":"0x12A","writer":"WeddingProgress"}`.
- The codec DOES carry v79 evidence: `libs/atlas-packet/field/clientbound/wedding_progress_test.go:11` — `// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v79 ida=0x55dfbb`, backed by a passing `TestWeddingProgressByteOutputV79` (verified: `go test -run TestWeddingProgressByteOutputV79 -v ./field/clientbound/...` → PASS).
- The IDA export has the resolved fname: `docs/packets/ida-exports/gms_v79.json` → `CField_Wedding::OnWeddingProgress` @0x55dfbb, 3 calls.
- **But `docs/packets/registry/gms_v79.yaml` has no `WEDDING_PROGRESS` clientbound op entry.** `grep -n -i wedding docs/packets/registry/gms_v79.yaml` returns only `WEDDING_PHOTO`, `WEDDING_GIFT_RESULT`, `WEDDING_CEREMONY_END`, `WEDDING_ACTION`, `WEDDING_TALK` — never `WEDDING_PROGRESS`. By contrast every other version's registry (`gms_v83.yaml:1573`, `gms_v84.yaml:2116`, `gms_v87.yaml:1678`, `gms_v95.yaml:1870`, `jms_v185.yaml:1760`) has the `WEDDING_PROGRESS` clientbound op entry. Without the registry entry the matrix tool cannot join the routed template opcode to a coverage-matrix row, so it silently defaults the cell to `n-a` (opcode -1) instead of surfacing a hole.

**Why this went unnoticed:** commit `f43fee685` ("test(task-181): pin gms_v79 byte fixtures for 18 routed writers") explicitly says *"WeddingProgress already had its v79 marker and was skipped"* — i.e. the agent that added registry entries for the other 17 route-safe writers deliberately skipped WeddingProgress because a test marker already existed, without checking that the marker alone doesn't populate the registry op needed for matrix promotion. `writer-routing.md`'s "Routed (18)" table (line 35) lists `WeddingProgress` alongside the other 17 with no caveat, implying equal completion status — that implication is false.

**Blast radius / gate coverage:** confirmed this gap is invisible to every existing CI gate — all ran clean despite the hole:
```
matrix --check        → exit 0 (n-a is a legal state; only 🟥/stale/fatal fails)
dispatcher-lint        → exit 0
fname-doc --check      → exit 0
operations --check     → exit 0
doc-freshness --check  → exit 0
gate-check --check     → exit 0
```
This is exactly the "false pass" pattern task-181 itself was created to hunt (per `codec-defects.md`'s framing) — found here inside the task's own deliverable, on the one writer whose registry step was skipped.

**Fix required:** add a `WEDDING_PROGRESS` clientbound op entry to `docs/packets/registry/gms_v79.yaml` (opcode 0x12A / 298 decimal, fname `CField_Wedding::OnWeddingProgress`, packet `field/clientbound/FieldWeddingProgress`, mirroring the v83 block at `gms_v83.yaml:1573-1577`), then regenerate `matrix` and confirm the cell promotes to ✅. This is a mechanical, single-entry fix — no RE work needed, all evidence already exists.

## 3. False-pass codec fixes (DEFECT-1..6) — verified fixed across ALL applicable versions

Per-defect matrix row check (`docs/packets/audits/STATUS.md`), all six versions (v79/v83/v84/v87/v95/jms) column-by-column:

| Defect | Op | v79 | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|---|---|
| DEFECT-1 SnowballState | SNOWBALL_STATE (line 410) | ✅ 0x103 | ✅ 0x119 | ✅ 0x120 | ✅ 0x12A | ✅ 0x152 | ✅ 0x131 |
| DEFECT-2 AriantArenaUserScore | ARIANT_ARENA_USER_SCORE (line 446) | ✅ 0x113 | ✅ 0x129 | ✅ 0x130 | ✅ 0x13A | ✅ 0x162 | ✅ 0x141 |
| DEFECT-3 ContiMove | CONTI_MOVE (line 247) | ✅ 0x08C | ✅ 0x094 | ✅ 0x097 | ✅ 0x09C | ✅ 0x0A4 | ✅ 0x091 |
| DEFECT-4 TournamentSetPrize | TOURNAMENT_SET_PRIZE (line 456) | ✅ 0x127 | ✅ 0x13D | ✅ 0x144 | ✅ 0x14E | ✅ 0x178 | ✅ 0x156 |
| DEFECT-5 Tournament | TOURNAMENT (line 454) | ✅ 0x125 | ✅ 0x13B | ✅ 0x142 | ✅ 0x14C | ✅ 0x176 | ✅ 0x154 |
| DEFECT-6 TournamentMatchTable | TOURNAMENT_MATCH_TABLE (line 455) | ✅ 0x126 | ✅ 0x13C | ✅ 0x143 | ✅ 0x14D | ✅ 0x177 | ✅ 0x155 |

All 36 cells (6 defects × 6 versions) show ✅. **PASS — this deliverable is fully and correctly done**, including the honestly-disclosed exception:

- `codec-defects.md`'s DEFECT-1 "residual" claim was independently verified: the jms evidence file `docs/packets/evidence/jms_v185/field.clientbound.FieldSnowballState.yaml` was last touched by a pre-task-181 commit (`0564037e4`), confirming it was *not* re-pinned by this task, while the v79/v83/v84/v87/v95 evidence files *were* touched by `05d47c70e` (this task's SnowballState fix commit). The jms **report** doc (`docs/packets/audits/jms_v185/FieldSnowballState.md`) still shows the stale 8-field layout, exactly as documented — an honest, correctly-scoped residual, not a hidden gap. The matrix cell itself is still ✅ (the evidence hash, not the stale prose report, is what the tool grades).
- The DEFECT-8 "cosmetic bug" claim (MONSTER_CARNIVAL_SUMMON's matrix row displaying `MonsterCarnivalMessage` in the Packet column due to shared-fname resolution) was verified present in STATUS.md line ~448 exactly as described, and confirmed not to affect per-op grading — both ops show independent ✅.

Also spot-verified DEFECT-7/8/9 (MonsterCarnivalStart/Summon/Message/Died/Leave — "codec already correct, route-only gap") and DEFECT-10 (MtsOperation 35-arm dispatcher family): all show ✅ for v79 in the matrix, and all 35 individual `FieldMtsResult*.json` per-arm reports under `docs/packets/audits/gms_v79/` show `Verdict: ✅`. jms_v185 correctly stays ⬜ for MTS_OPERATION/MTS_OPERATION2 (registry-absent — CITC op doesn't exist in the JMS client build), matching the documented "version-absent, not a gap" claim.

Byte-fixture test functions claimed in `codec-defects.md` were all confirmed to exist and pass (`TestSnowballStateByteOutputV79`, `TestContiMoveByteOutputV79` + `...Nullsub`, `TestTournamentByteOutputV79`, `TestTournamentMatchTableByteOutputV79`, `TestTournamentSetPrizeByteOutputV79` + `...NoItems`, `TestAriantArenaUserScoreByteOutputV79`, `TestMonsterCarnivalStartByteOutputV79`, `TestMonsterCarnivalSummonByteOutputV79`, `TestMonsterCarnivalMessageByteOutputV79`, `TestMonsterCarnivalDiedByteOutputV79`, `TestMonsterCarnivalLeaveByteOutputV79`).

## 4. Gate results (run fresh, from repo root / libs/atlas-packet)

| Gate | Command | Exit | Notes |
|---|---|---|---|
| Coverage matrix | `go run ./tools/packet-audit matrix --check` | **0** | one informational `note:` line (unrelated n-a evidence for USE_TELEPORT_ROCK×gms_v48); no failures |
| Dispatcher lint | `go run ./tools/packet-audit dispatcher-lint` | **0** | `dispatcher-lint: clean` |
| Byte-fixture tests | `cd libs/atlas-packet && go test -race -count=1 ./field/... ./monster/carnival/...` | **0** | all 5 packages `ok` |
| (extra, for context) fname-doc --check | | 0 | |
| (extra) operations --check | | 0 | |
| (extra) doc-freshness --check | | 0 | |
| (extra) gate-check --check | | 0 | |
| (extra) atlas-channel build+vet | `go build ./...` / `go vet ./...` | 0 / 0 | widened wrapper signatures compile clean |
| (extra) libs/atlas-packet build+vet | `go build ./...` / `go vet ./...` | 0 / 0 | |
| (extra) tools/packet-audit build+test | `go build ./...` / `go test ./...` | 0 / 0 | resolver change (COutPacket header-op skip) covered by `export_test.go` |

All gates the task instructed to run pass clean. **However, the WeddingProgress gap in §2 demonstrates these gates do not actually verify deliverable #1 ("matrix cell verified ✅") — they only catch conflicts/staleness, not silently-n-a cells that should be verified.** Passing gates ≠ deliverable complete; treat §2's finding as authoritative over the green gate exit codes.

## 5. Other integrity checks performed

- No stray/leaked report files from the "selective regen" recipe: `git status --short` is clean on this branch; diffed `docs/packets/audits/` file list against the known writer/dispatcher families — every changed file maps to one of the 30 writers or the 35 MtsOperation arms, except two **deletions** (`gms_v79/FieldAriantScore.{json,md}`), which is the pre-existing stray the recipe explicitly calls out for cleanup (confirmed via `git diff`: both files were removed, not added).
- `docs/packets/audits/STATUS.md` Conflicts section: `None.`
- Handler-side items in `audit.md` (§4 missing handlers, `OwlActionHandle`, `UiOpen`) are out of scope for this writer-routing/false-pass audit and were not re-verified here; they are documented as open/unresolved in the task's own docs already.

## Overall assessment

- **Plan/deliverable adherence:** MOSTLY_COMPLETE — 29/30 writers correctly routed *and* matrix-verified; the 6 false-pass defects are fully corrected and re-verified across all applicable versions; all instructed gates pass.
- **One real, unflagged gap:** WeddingProgress is routed in the template and has a passing byte-fixture with a v79 evidence marker, but is **not promoted in the coverage matrix** (`n-a`, opcode -1) because of a missing one-line registry entry in `docs/packets/registry/gms_v79.yaml`. `writer-routing.md` lists it in the "Routed (18)" table without qualification, which reads as a completion claim the matrix does not support. This is a false pass by the task's own definition, not merely an oversight worth a footnote.
- **Recommendation:** NEEDS_FIXES — add the missing `WEDDING_PROGRESS` registry entry for gms_v79 (mirrors the existing v83/v84/v87/v95/jms blocks, all evidence already exists), regenerate the matrix, and confirm the cell promotes to ✅ before treating this task as complete. All other deliverables are DONE.

## Action items

1. Add a `WEDDING_PROGRESS` clientbound op entry to `docs/packets/registry/gms_v79.yaml` — opcode 0x12A (298 decimal), `fname: CField_Wedding::OnWeddingProgress`, `packet: field/clientbound/FieldWeddingProgress`, provenance `ida-discovered`, ida address `0x55dfbb` — mirroring the v83 block (`docs/packets/registry/gms_v83.yaml:1573-1577`).
2. Regenerate the coverage matrix (`go run ./tools/packet-audit matrix`) and confirm `WEDDING_PROGRESS` × `gms_v79` promotes to ✅ in both `STATUS.md` and `status.json`.
3. Re-run `matrix --check` to confirm still clean, and correct the "Routed (18)" framing in `writer-routing.md` if it is to remain an accurate record (currently implies uniform completion that wasn't true until action item 1 lands).
