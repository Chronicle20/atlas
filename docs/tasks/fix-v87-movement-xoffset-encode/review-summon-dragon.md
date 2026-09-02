# Review — `890089fe3` + `9f845ab4d` (summon/dragon move-path re-serialization, movement writer `types`)

verdict: APPROVED_WITH_FINDINGS

Scope reviewed: `git diff 890089fe3~1..9f845ab4d` — 22 files, +3580/−80. Commits
`890089fe3` (`model.ReserializeMovePath` + summon/dragon `Encode` + tests) and
`9f845ab4d` (writer `options.types` across 11 seed templates, guard extension,
`TEMPLATE_CONVENTIONS.md`, template-driven tests). `4a108ee43` / `9b491d46e`
explicitly excluded and not read beyond `model/movement.go` at HEAD, which the
new code calls and whose contract the fix depends on.

Range matches the described work exactly; no scope mismatch.

## Brief compliance

### `brief-summon-dragon.md`

| Requirement | Status | Evidence |
|---|---|---|
| Re-serialize at ENCODE time inside the two clientbound codecs | PASS | `libs/atlas-packet/summon/clientbound/move.go:64`, `libs/atlas-packet/dragon/clientbound/move.go:61` — `w.WriteByteArray(model.ReserializeMovePath(l, ctx)(m.rawMovement, options))` |
| Kafka contract and the `[]byte` crossing it untouched | PASS | diff touches no file under `services/atlas-channel/.../kafka/`; `git diff --stat 890089fe3~1..9f845ab4d` lists only `libs/atlas-packet`, templates, `tools/`, `docs/` |
| Serverbound capture unchanged | PASS | `summon/serverbound/move.go` and `dragon/serverbound/move.go` absent from the diff stat |
| Decode side uses the tenant's own `types` from `options` | PASS | `model/move_path.go:58` passes the same `options` map into `m.Decode(...)`; resolution path is `resolveMovementPathAttr` (`model/movement.go:412`) |
| No cross-version translation | PASS | `move_path.go:56-60` uses one context/tenant for both decode and encode |
| Inherits the inbound/outbound gate split rather than adding version logic | PASS | `move_path.go` contains no `tenant` import and no version predicate; the split lives at `model/movement.go:81` / `:96` |
| Trailer question resolved without inventing a field | PASS | `move_path.go:71-76` copies `raw[consumed:]` verbatim; rationale documented at `move_path.go:27-34` — behaviour on `bPassive` versions is unchanged by construction, so the unestablished IDB answer is not depended on. That is the correct disposition of the brief's "STOP and report" clause. |
| Channel plumbing unchanged | PASS | no `services/atlas-channel` file in the diff |
| Test proving v87 drops the pair and v83/v92/v95/JMS are byte-unchanged | PASS | `summon/clientbound/move_reserialize_test.go:28` and `:55`; `dragon/clientbound/move_reserialize_test.go:26` and `:55`. Expected bytes come from `test.MovePathBlob` (`test/move_path.go:24`), a hand-built layout, not the encoder's own output — the pattern the brief asked for. |
| `cd libs/atlas-packet && go test ./...` | PASS | run: all packages `ok` |
| `packet-audit matrix --check`, `fname-doc --check`, `operations --check`, `dispatcher-lint` | PASS | all four run from repo root, exit 0 |

### `brief-writer-types.md`

| Requirement | Status | Evidence |
|---|---|---|
| Exact inventory of 23 writer tables added | PASS + superset | Machine-compared old vs new JSON per file. Added: gms_12 `SummonMove`; gms_48/61/72/79 `SummonMove`; gms_83/84 `SummonMove`+`DragonMove`; gms_87 `PetMovement`+`SummonMove`+`DragonMove`; gms_92 all six; gms_95 `SummonMove`+`DragonMove`; jms_185 `PetMovement`+`SummonMove`+`DragonMove`. Nothing removed, nothing changed. |
| Table copied from the matching handler IN THE SAME FILE, no reorder | PASS | Every writer `options.types` list at HEAD compares element-for-element equal (Python list equality, order-sensitive) to its same-file handler counterpart under the mapping `SummonMove←SummonMoveHandle`, `DragonMove←DragonMoveHandle`, `PetMovement←PetMovementHandle`, `CharacterMovement←CharacterMoveHandle`, `MoveMonster←MonsterMovementHandle`, `NPCAction←NPCActionHandle`, for all 11 files. Index positions therefore preserved. |
| No other JSON changed | PASS | Reparsed both revisions of all 11 templates, stripped writer `options.types`, compared serialized documents — identical for every file. No opcode, ordering or unrelated key changed. |
| gms_92 had no writer tables at all; six filled | PASS + correct | Confirmed at `890089fe3`: zero writer entries in `template_gms_92_1.json` carried `types`. All six existing movement writer entries now do. Filling `CharacterMovement`/`MoveMonster`/`NPCAction` goes beyond the brief's three-entry inventory but is *required* by the extended guard (every derived-carrier writer entry present in a template must carry the table) and fixes the same latent `fhFallStart` drop on gms_92 that the brief flags for pet movement. Correct and complete. |
| Guard extended to writers, carrier set derived not hardcoded | PASS | `tools/template-movement-types-guard.sh` — `derive_move_writers` walks `libs/atlas-packet/*/clientbound/*.go`, matching a `model.Movement` struct field or a `model.ReserializeMovePath` call, and reads the single `const ...Writer = "..."`. Live run derives exactly `CharacterMovement, DragonMove, MoveMonster, NPCAction, PetMovement, SummonMove`. Cross-checked against `grep -rln 'model\.Movement'`: the only clientbound file matching that grep and *not* derived is `character/clientbound/buff_cancel.go`, where the match is a comment mentioning `model.MovementAffectingMask` — correctly excluded. |
| Guard cannot pass vacuously | PASS | Two failure paths proven by execution against a staged copy at `/tmp/guardtest` (repo untouched): (a) pointing the derivation at an empty `libs/atlas-packet` prints `DERIVATION ERROR: only 0 movement-carrying writer(s) derived ... refusing to check writers vacuously` and exits 1 (floor of 4, script `derive_move_writers`); (b) a file that encodes a move path but declares ≠1 writer const is a hard `DERIVATION ERROR`. Per-template floors (`contrib_w == 0` → NO MOVE WRITERS, `< 3` → TOO FEW) catch a whole-section rename. |
| Guard fails when a table is removed | PASS | Removed `SummonMove`'s `types` from the staged `template_gms_87_1.json`: `MISSING TYPES: template_gms_87_1.json SummonMove (writer) (opCode 0xBE) has no non-empty options.types` / exit 1. Baseline exit 0: `OK: 60 move handlers and 60 move writers across 11 templates`. |
| Other template guards still pass | PASS with one pre-existing failure | `template-opcode-order-guard.sh` OK, `template-duplicate-binding-guard.sh` OK. `template-symbol-check.sh` (per-file) reports `DANGLING: ChatGeneralChat` on gms_72/79/92/95 and jms_185 — **pre-existing**, present identically at `890089fe3` and at `4a108ee43~1`. Not caused by this unit. |
| `TEMPLATE_CONVENTIONS.md` corrected | PASS | `docs/packets/TEMPLATE_CONVENTIONS.md` — rule renamed to "move entries", both carrier lists enumerated, the writer-absent failure mode documented, "copy from the matching handler in the same file, never across templates" recorded. |
| Prove the fix is LIVE, driven by the shipped template | PASS | `summon/clientbound/move_reserialize_test.go:111` and `dragon/clientbound/move_reserialize_test.go:109` call `test.TemplateWriterOptions(t, "template_gms_87_1.json", SummonMoveWriter/DragonMoveWriter)`, which reads the seed file off disk (`test/template_options.go:31-47`) and `t.Fatalf`s if the writer is not registered exactly once. Honesty check: `TestSummonMoveUnparseableBlobIsUnchanged` (`:84`) proves that with `nil` options the emitted bytes equal the capture verbatim; the template-driven test asserts equality with the *offset-free* layout, so deleting the template table flips it from `want=rebroadcast` to `got=captured` and it fails. The test cannot pass either way. |
| Pet `fhFallStart` defect covered | PASS | `pet/clientbound/movement_test.go:157-199` — template-driven, attr 15, expected bytes hand-written including `0x09 0x00` FhFallStart and *no* XOffset/YOffset. Fails if the template table is dropped (the `FALL_DOWN` name check would not fire). |
| Tree clean, work committed | PASS | `git status --porcelain` in the worktree is empty. |

## Adversarial checks requested

### 1. `ReserializeMovePath` fallback conditions — `model/move_path.go:51-70`

- `len(raw) < 5` (`:52`): the minimum header any version writes is `Encode2 x, Encode2 y, Encode1 count`. Correct floor. Versions with `startVx/startVy` need 9, but a 5..8-byte blob is caught downstream (`len(m.Elements) == 0` or `consumed >= len(raw)`); traced by hand for a 6-byte GMS 92 blob: reader stalls, `numElems = 0`, `:66` fires. Pass-through is the right failure mode — a sub-header blob is not a move path.
- `consumed <= 0 || consumed >= len(raw)` (`:63`): **cannot fire on a legitimate blob.** `CMovePath::Encode` always writes, after the fragment array, `Encode1 keypadLen` + keypad run + `Encode2 minX/minY/maxX/maxY` — ≥ 9 bytes — corroborated independently in `docs/tasks/task-088-player-summons/summon-wire-truth.md:133` and in `libs/atlas-packet/summon/serverbound/move.go:27-30`. `rawMovement` is the whole post-identity remainder (`summon/serverbound/move.go:75`), so the trailer is always present. A valid all-NORMAL path therefore leaves ≥9 unconsumed bytes and never trips the no-trailer heuristic. The condition doubles as the over-read detector (`request.Reader` clamps at end-of-buffer), which is why it is right to keep it.
- any `*Element` fallback (`:67-71`): the type assertion matches only the bare `*Element`; `*NormalElement` et al. embed it but are distinct pointer types, so the check does not over-fire. Conservative and correct — width is genuinely unknown for a `DEFAULT`-typed or out-of-table attr (e.g. index 21 in `template_gms_87_1.json` is `{"Name":"UNKNOWN","Type":"DEFAULT"}`), and re-encoding would truncate. Consequence worth knowing: a v87 path containing attr 21 is still rebroadcast verbatim, i.e. still carrying the pair. That is unavoidable without a width model for that attr and is the safe direction.
- Round-trip fidelity on the non-v87 versions is a real property, not an assumption: every element kind's `Encode` is field-for-field the inverse of its `Decode` (`model/movement.go:220-289` vs `:314-395`), including `StatChangeElement`'s single `BStat` byte and the `X = m.StartX` synthesis in `JumpElement`/`StartFallDownElement`.

### 2. Trailer handling — `model/move_path.go:71-76`

Offset arithmetic is correct. `consumed = r.Position()` is the absolute index after the element array (`request.Reader.Position()` returns `r.pos`, `libs/atlas-socket/request/reader.go:125`), the reader is created at offset 0 over the blob alone (`:57`), and `body` is the freshly encoded prefix. `out = body ++ raw[consumed:]` therefore cannot duplicate (the two slices are disjoint by construction — `body` replaces `raw[:consumed]`) and cannot truncate (the slice runs to `len(raw)`). The capacity hint `len(body)+len(raw)-consumed` matches the final length exactly. The client asymmetry is handled by not depending on it: the trailer is byte-identical in and out on every version, so `bPassive` behaviour is unchanged everywhere. `TestSummonMoveV87DropsElementOffsets` asserts `bytes.HasSuffix(got, test.MovePathTrailer)`.

### 3. Template diff — see the table above. Byte-identical to the same-file handler tables, ordering preserved, nothing else changed, gms_92 correctly and completely filled.

### 4. Extended guard — see the table above. Derivation cannot return empty or partial silently; removal of a table is proven to fail the guard.

### 5. End-to-end liveness — see the table above. Template-driven tests exist for summon, dragon and pet on `template_gms_87_1.json` and are honest. One residual, below (F1).

## Findings

### Non-blocking

**F1 — `services/atlas-configurations/atlas.com/configurations/seeder/seeder.go:144` — the seed-template edit does not reach an already-provisioned tenant.**
`importTemplate` is documented and implemented as CREATE-IF-ABSENT (`seeder.go:144`, `:157-164`, "Template already exists, skipping"). The live GMS 87.1 tenant on `atlas-main` — the deployment the diagnosis was written against — already has a template row, so shipping the new `options.types` in `seed-data` will not by itself make `ReserializeMovePath` classify a fragment there. An operator must update the stored configuration (API/UI, or delete-and-reseed) for the fix to take effect. The code is correct and the test proves the shipped template is correct; this is a rollout prerequisite that neither commit records anywhere. Worth stating in the PR body.

**F2 — `libs/atlas-packet/model/move_path.go:63` — a malformed, client-controlled blob can be re-encoded into a *fabricated* path rather than passed through.**
The documented contract at `:35-43` is "returned unchanged whenever it cannot be parsed with confidence", but the only over-read detector is `consumed >= len(raw)`, and `request.Reader` returns 0 without advancing on a short read. Concrete case, GMS 87 with the v87 table: `raw` = 5-byte header + one complete 18-byte NORMAL fragment + `count = 2` + one further byte (25 bytes total). The second fragment's attr byte is read at pos 23, then `ReadInt16` at pos 24 has one byte left and stalls; `consumed = 24 < 25`, so no fallback fires. The result is a 34-byte packet carrying a synthesized all-zero second fragment, broadcast to every observer. Pre-change the garbage went out verbatim, so this is not a regression in kind, but the guarantee in the doc comment is not actually enforced. A cheap tightening would be to require the fragment array to end on a plausible boundary (e.g. ≥9 trailer bytes remaining, matching the keypad+`m_rcMove` floor established above). Reported, not fixed.

**F3 — `libs/atlas-packet/model/move_path.go:59` — new outbound error-log source, multiplied by observer count.**
`m.Decode` is now invoked on the *writer* path, and `Movement.Decode` calls `logUnconfiguredMovementCode` (`model/movement.go:449`) — an `Errorf` with up to a 512-byte hex dump — once per unresolved element. `session.Announce` (`services/atlas-channel/atlas.com/channel/session/processor.go:250-271`) runs the encoder once per receiving session, so a summon whose path carries an out-of-table attr now emits that error once per element per observer, at movement-packet frequency. Summon/dragon blobs previously never reached this code. The per-observer decode+encode CPU cost is in line with what `CharacterMovement`/`MoveMonster` already pay, so only the log volume is new.

**F4 — stale "byte-faithfully" comments now contradict the implemented behaviour.** `services/atlas-channel/atlas.com/channel/socket/writer/summon.go:21`, `.../writer/dragon.go:16`, `.../socket/handler/summon_move.go:17`, `.../summon/processor.go:71`. `890089fe3` updated the two codec doc comments in `libs/atlas-packet` but left these four saying the blob is rebroadcast byte-faithfully, which is exactly the belief this branch exists to correct. Documentation only; the files are outside the diff.

**F5 — template-driven encoder coverage exists only for `template_gms_87_1.json`.** The six tables newly added to `template_gms_92_1.json` (including `CharacterMovement`/`MoveMonster`/`NPCAction`, which no prior test or guard exercised) are covered only by `template-movement-types-guard.sh`. That is adequate against *absence*; there is no assertion that gms_92's outbound `fhFallStart` is now emitted. Low value-to-cost, noted for completeness.

**F6 — `libs/atlas-packet/test/template_options.go:66-83` couples a `libs/atlas-packet` module test to the monorepo layout.** `repoRoot` walks upward until it finds `services/atlas-configurations/seed-data/templates`. Correct inside this checkout and it `t.Fatalf`s rather than skipping if the walk fails, which is the right failure mode; but the packet library's tests can no longer run against a bare module checkout. Deliberate trade-off given F-liveness is the point of the test; recorded, not objected to.

### Pre-existing, not attributable to this unit

`tools/template-symbol-check.sh` reports `DANGLING: ChatGeneralChat` for `template_gms_72_1.json`, `_79_1`, `_92_1`, `_95_1` and `template_jms_185_1.json`. Identical at `890089fe3` and at `4a108ee43~1`; this unit neither introduced nor worsened it.

## Not evaluable

1. **IDA claims.** Every `CMovePath::Encode`/`Decode` address, instruction count and read/write assertion in `move_path.go`, the tests and the templates' version reasoning is taken on the diagnosis's word — no IDB was consulted in this review. The *internal* consistency (which versions the tests treat as symmetric vs. asymmetric, and the gates in `movement.go`) does match the diagnosis table.
2. **Whether `bPassive` is 1 at `CSummonedPool::OnMove` / `CDragon::OnMove`.** The implementation makes the answer irrelevant by copying the trailer verbatim, which is the right call, but the question itself is still unanswered.
3. **Live-wire confirmation.** No capture from a v87 client was replayed through the new writer path; the v87 assertion rests on hand-built `test.MovePathBlob` bytes.

## Verdict rationale

Both briefs are satisfied requirement by requirement, with the template diff verified mechanically rather than by eye, the guard's failure modes proven by execution, and the liveness test shown to be honest. No blocking defect found. F1 is the one item that should reach the PR body, because the fix being *correct in the repo* is not the same as it being *live on `atlas-main`*.
