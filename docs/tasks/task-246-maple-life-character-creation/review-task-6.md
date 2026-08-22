# Review: Task 6 — Maple Life serverbound codecs (543 sub-body, duplicate probe)

Range reviewed: `a79ead298..ddef3d665` (7 commits, one implement/revert pair:
`b130b951b` → `051956cd9`). Both `.superpowers/sdd/plan/task-6-brief.md`
(controller addendum) and `task-6-brief-cont.md` (controller ruling) read.
`task-6-report.md` read and cross-checked against the commits rather than
trusted at face value, per the reconstruction caveat.

## Scope confirmed

The range delivers exactly what both briefs specify: `item_use_maple_life.go`
+ test (the `USE_CASH_ITEM` 543 sub-body), `maplelife/serverbound/check_name.go`
+ test (the duplicate-name probe, gms_v83=256/v87=270/v92=301/v95=311),
`use.go` correctly not written, evidence records, export splices, `derivation.md`
PENDING fixes, an implement-then-revert of an analyzer change, and the Option 1
fallback (evidence deletion + `candidatesFromFName` removal + `docs/TODO.md`
entry). No scope drift found — the diff matches the brief.

## Findings

### 1. gms_v84 out of scope — PASS

No v84 marker, fixture, evidence record, or matrix cell for either op.
`item_use_maple_life_test.go` and `serverbound_test.go` both state "gms_v84 is
VERSION-ABSENT ... no marker, no fixture" and the loops (`inScope`,
`imlFixtureVersions`, `mlcnFixtureVersions`) only ever list v83/v87/v92/v95.
`status.json`'s `MaplelifeCheckName` row shows `gms_v84: {"state": "n-a",
"opcode": -1}` (`docs/packets/audits/status.json:37851-37854`) — derived, not
chosen. `git diff a79ead298..ddef3d665 -- docs/packets/ida-exports/gms_v84.json`
is empty — the file wasn't touched this range.

### 2. The doubled `update_time` finding — PASS, codec matches the conclusion

`item_use_maple_life.go:1-95`'s doc comment quotes the raw-disassembly
addresses from the brief verbatim (v83 `0x7d7a4d`/`0x7d7a56`; v87
`0x82e46b`/`0x82e474` leading, `0x82e4fd`/`0x82e506` trailing) and concludes the
trailing write is unconditional on every version. The `Encode`/`Decode` bodies
(`item_use_maple_life.go:126-153`) confirm this: `w.WriteInt(m.updateTime)` /
`m.updateTime = r.ReadUint32()` run unconditionally, with no
`if !m.updateTimeFirst` gate — unlike `item_use_incubator.go:34-46`'s
`if !m.updateTimeFirst { … }`, which the doc comment explicitly contrasts
against.

`TestItemUseMapleLifeUpdateTimeTrailing`
(`item_use_maple_life_test.go:103-127`) is not vacuous: it round-trips both
`updateTimeFirst=true` and `updateTimeFirst=false` and asserts `UpdateTime()
== 999` in both cases. If the codec had been (wrongly) gated on
`updateTimeFirst`, this test would fail for `true`. It genuinely pins the
"unconditional" claim.

`WriteInt(uint32)` / `ReadUint32()` signatures
(`libs/atlas-socket/response/writer.go:36`, `libs/atlas-socket/request/reader.go:96`)
match the `updateTime uint32` field type — no silent truncation.

The four `decompile_sha256` values quoted in `derivation.md` §2 for
gms_v83/v87/v92/v95 (`derivation.md` diff hunks) match the corresponding
`docs/packets/evidence/gms_v*/cash.serverbound.CashItemUseMapleLife.yaml`...
**except those evidence files were deleted under Option 1** (see finding 5) —
the sha values now live only in `derivation.md`'s prose and the (deleted)
evidence git history, not in a currently-checked-in evidence record. That is
the expected consequence of the Option 1 fallback and not itself a defect,
but it does mean the sha pins are no longer independently re-verifiable from
a live evidence file — noted as non-blocking below.

### 3. Option 2 → revert → Option 1 — PASS, revert is clean

`git diff a79ead298..ddef3d665 -- tools/packet-audit/internal/atlaspacket/analyzer.go`
is empty — the net change across the whole range to the file Option 2 touched
is zero, confirming `051956cd9` fully undoes `b130b951b` with no residue.

`git diff a79ead298..ddef3d665 -- docs/packets/audits/status.json` touches
exactly one `"packet":` block (`maplelife/serverbound/MaplelifeCheckName`,
promoting v83/v87/v92/v95 from `incomplete`/`"no audit report"` to `verified`)
plus the `toolSha`/`exportHashes` header. `grep -c '"packet":'` on that diff
returns `1`. `grep -n CashItemUseMapleLife docs/packets/audits/status.json
docs/packets/audits/STATUS.md` returns nothing — no dangling or promoted row
for the unlinked cell. The seven already-verified `CashItemUse*` sibling rows
(MapleTV, Megaphone, PetNameTag, PointReset, SuperMegaphone, TeleportRock,
TripleMegaphone) and the ~15 packets sharing the guard shape
(`chat/serverbound` Whisper, `cash/serverbound` `shop_operation_buy*`) show no
diff at all across the range — `git diff a79ead298..ddef3d665 --stat -- libs/atlas-packet/chat libs/atlas-packet/cash/serverbound/shop_operation_buy*`
is empty, confirming those files were only observed, not edited, matching the
report's self-review claim.

### 4. Option 1's fallback obligations — PASS

- The four `cash.serverbound.CashItemUseMapleLife.yaml` evidence records are
  gone: `find docs/packets/evidence -iname "*CashItemUseMapleLife*"` returns
  nothing.
- The `SendCreateNewCharacter` `candidatesFromFName` case is removed;
  `tools/packet-audit/cmd/run.go`'s diff shows only the two new cases added
  this range (`SendCheckDuplicateIDPacket` for `MaplelifeCheckName`, and the
  `SendCreateNewCharacter` case is present only as a doc comment explaining
  why it's *not* a case — no dead/orphaned case left).
- `docs/TODO.md` carries a new entry under the existing "Tooling defects found
  in `tools/packet-audit`" list (`docs/TODO.md` diff, +21 lines) that
  correctly names the root cause: `guardFromIf` re-prints and reparses the
  `if` condition's AST text via `ParseGuard`, which only compiles
  `t.Region()`/`t.MajorVersion()`/`t.MinorVersion()` comparisons — a bare call
  to a named boolean helper (`UpdateTimeFirst(t)`) fails to parse and falls
  back to an always-true guard. It also names the concrete blast-radius
  evidence (`whisperHasUpdateTime`, `legacyGMS`/`buyOmitsCurrency`) and
  proposes a scoped-fix direction. This satisfies the "a fallback that leaves
  no record is not acceptable" bar.
- The dropped test markers (`ddef3d665`) are exactly the four
  `packet-audit:verify packet=cash/serverbound/CashItemUseMapleLife ...`
  lines that pointed at the now-deleted evidence/matrix row — `git show
  ddef3d665` confirms only those four comment lines were removed, and no
  other marker in the file was touched. No live cell depended on them (the
  cell was never linked into the matrix to begin with).

### 5. Promotion / unlink end states — PASS

`MaplelifeCheckName` fully promotes: `status.json:37847-37865` shows
`gms_v83/v87/v92/v95` all `"state": "verified"` with the correct opcodes
(256/270/301/311), `gms_v84: "n-a"`. Its four audit reports
(`docs/packets/audits/gms_v{83,87,92,95}/MaplelifeCheckName.json`) are present
and clean (`"Verdict": 0`, `"FlatInvalid": false`).

`CashItemUseMapleLife` is unlinked: codec + tests present
(`item_use_maple_life.go`, `_test.go`), no evidence records, no
`candidatesFromFName` case, no matrix row — the same state as its
already-audited-but-unlinked `item_use_*.go` siblings, as intended.

### 6. `MajorAtLeast` idiom — N/A, correctly so

Neither new file introduces a version-gated boundary — both derivations found
an identical wire shape across all four in-scope versions ("IDENTICAL SHAPE ...
on every in-scope version" appears in both files' doc comments), so there is
no raw `> N` comparison to flag, and none was introduced. `item_use.go`
(pre-existing `UpdateTimeFirst`, using `MajorVersion() >= 87`, not the
`MajorAtLeast` idiom) was not touched by this range — out of this unit's
surface, and not a new-code concern.

### 7. `derivation.md` §2/§5/§6 PENDING fixes — PASS

§2/§6 sha256 values are freshly harvested this pass and match the evidence
YAMLs 1:1 (spot-checked all four `MaplelifeCheckName` records against
`derivation.md`'s quoted values — exact match). §5's four values were
confirmed **not** re-derived: `grep decompile_sha256` on the four
`docs/packets/evidence/gms_v*/maplelife.clientbound.MapleLifeError.yaml`
files (Task 5's pinned records) returns exactly the four values `derivation.md`
§5 quotes, byte-for-byte.

## Not evaluable

- The live IDA re-decompilation and raw-disassembly claims (addresses,
  instruction sequences) cannot be independently re-verified without IDA
  session access from this review; they are taken on the strength of the
  doc-comment citations, evidence-file cross-references, and internal
  consistency (test assertions matching the claimed shape), which is the
  strongest check available from the diff surface alone.
- Whether extending `guardFromIf` really is the architecturally correct fix
  (vs. some other design) is a judgment call outside this unit's diff; the
  TODO entry documents the finding, which is what was required.

## Non-blocking notes

- The `cash.serverbound.CashItemUseMapleLife.yaml` evidence records that
  carried the `decompile_sha256` values now quoted in `derivation.md` §2 were
  deleted as part of Option 1. The sha values are preserved in `derivation.md`'s
  prose and in git history, but a future reader of `derivation.md` §2 can no
  longer click through to a live evidence file to re-verify them — this is
  the expected shape of "unlinked, same as its siblings" and not a defect,
  but worth flagging since it's slightly unusual for a PENDING-fix commit to
  end with its own sourced values no longer independently checked-in.

## Verdict rationale

Every requirement in the review brief was checked against the actual diff
(not the report's prose) and passed: the four-version scope, the doubled-
`update_time` codec/test correctness, the revert's completeness (zero net
diff to `analyzer.go`, zero unrelated row movement), the Option 1 fallback's
three obligations (evidence deletion, case removal, TODO entry), the intended
promote/unlink end states, and the §5 non-rederivation. No blocking defect
found.
