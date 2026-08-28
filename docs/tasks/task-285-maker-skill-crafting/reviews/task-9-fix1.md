# Review: Task 9 fix round 1 — MAKER_RESULT four-arm evidence coverage

**Commit under review:** `5d2a2ed52` (117-118 files, +5785-5787/-59)
**Fix brief:** `.superpowers/sdd/plan/task-9-brief-fix1.md`
**Prior blocking finding:** `.superpowers/sdd/plan/task-9-review.md` — ✅ was earned on
`MakerResultCreate` only, one of four non-degenerate mode arms.

## Verdict: APPROVED

## 1. Was any grading logic changed?

No. `git diff ab31624bd..5d2a2ed52 --stat` inside `tools/packet-audit/` shows **zero** changes
under `internal/matrix/` (confirmed by an explicit `git diff --stat -- internal/matrix/` — empty
output). `gradeCore`, `findReport`, `worstCandidateCell`, `registryDeclaresPacket` are byte-for-
byte unchanged.

The only code diff is `tools/packet-audit/cmd/run.go` (+11/-5): each of the five
`CUserLocal::OnMakerResult#<Arm>` candidate cases gained a `reportName: "MakerResult<Arm>"`
field. Read `run.go:209-216` (the `candidate.reportName` doc comment) and `run.go:137-138`
(`if c.reportName != "" { writerName = c.reportName }`): this only substitutes the report-file /
matrix-`WriterName` string used for `report.WritePacket` and the summary row; it does not touch
`locateAtlasFile`'s `pkg`-scoped struct resolution, the diff computation, or the verdict
(`report.Packet.Verdict` is computed before this line, from `worstRow(rows)` and the
flat-invalid/absent-feature reclassification — none of that is touched).

This is also not a new mechanism invented for this op: `reportName` already existed pre-commit
and is used by `SkillMacro` (`run.go:454`), `summon/Move|Attack|Damage` (`run.go:1202-1221`),
`CashItemGachaponResult/Button` (`run.go:2757-2768`), and `IncubatorResult` (`run.go:3953`) —
confirmed by `grep -n reportName run.go` before this commit's hunk. The fix reuses an existing,
narrow, report-naming-only lever. **Grading logic: untouched.**

## 2. Is the ✅ now genuinely arm-dependent? — reproduced myself

```
$ cd tools/packet-audit && go build -o /tmp/packet-audit .
$ /tmp/packet-audit matrix --check          # against committed tree
EXIT=0
```

Negative control, reproduced from the worktree root:
1. Backed up and deleted `docs/packets/evidence/gms_v87/character.clientbound.MakerResultDisassemble.yaml`.
2. `/tmp/packet-audit matrix -out-dir /tmp/audit-out` (default `-out-dir` writes to `docs/packets/audits`,
   so I redirected to `/tmp` to avoid touching the worktree per instructions).
3. Result: `MAKER_RESULT` row's `gms_v87` cell (`0x0E6`) flipped **✅ → ❌**. All other version
   cells for the row stayed ✅.
4. Restored the evidence file from backup, re-ran matrix into a second `/tmp` out-dir: the cell
   flipped back to ✅.
5. `git status --porcelain` after restore shows only the pre-existing, unrelated
   `docs/tasks/task-285-maker-skill-crafting/agent-ledger.tsv` modification (present before this
   test began, and outside this commit's diff) — no residual change from my test.

This confirms the claim: the row is graded worst-of across the four arms sharing the base fname,
and a single missing per-arm evidence record demonstrably holds the whole op row down. Before
this fix, per the original review, this same deletion changed nothing (the registry `packet:`
link made every candidate resolve to Create's fixture regardless of the other arms' state).

## 3. Does this match the GRADUATED precedent in `families.yaml`?

Read the GRADUATED comment blocks for `CCashShop::OnCashItemResult` (families.yaml:22-58),
`CITC::OnNormalItemResult` (60-76), `CShopDlg::OnPacket` (77-104), `CUIMessenger::OnPacket`
(105-112), `CMiniRoomBaseDlg::OnPacketBase` (113-129), `CTrunkDlg::OnPacket` (130-...). Every one
explicitly names **"synthetic #-suffix FNames in the per-version exports"** as the mechanism by
which per-arm audit reports and per-arm evidence exist for a mode-prefix dispatcher lacking a
seed-template route. That is precisely what this fix does for `CUserLocal::OnMakerResult#{Create,
CreateWithUpgrade,MonsterCrystal,Disassemble}` (added to all eight `docs/packets/ida-exports/*.json`,
additions-only, 209 lines each). Unlike the `OnClaimResult` analogy the prior review rejected
(a degenerate uniform-body claim that didn't hold up), this one is verified against the actual
GRADUATED text, not just asserted by resemblance.

## 4. Are the 32 new arm reports additions only, no unrelated churn?

`git show 5d2a2ed52 --diff-filter=A --name-only | grep audits/ | wc -l` → **64** (32 arms × {json,md}).
`git show 5d2a2ed52 --diff-filter=M --name-only | grep audits/` → only `STATUS.md` and
`status.json` (the two aggregate files, expected to change for any promotion). No other audit
report file appears as Modified or Deleted. `status.json`'s diff opens with only a `toolSha`
change plus the MAKER_RESULT op-row rewrite — consistent with "narrow regeneration of one op,"
not a wholesale re-run.

## 5. Rulings held

- `libs/atlas-packet/character/clientbound/maker_result_test.go:32-36` (comment) and the function
  body: `MakerResultFailed` carries **no** `packet-audit:verify` marker (grep for `MakerResultFailed`
  in the file only matches the doc comment, the test name, and constructor calls — no marker line).
- No `docs/packets/evidence/*/character.clientbound.MakerResultFailed.yaml` exists anywhere
  (`find docs/packets/evidence -iname '*MakerResultFailed*'` — empty).
- No `CUserLocal::OnMakerResult#Failed` entry was added to the per-version registry/export for
  grading purposes (the `run.go` `#Failed` candidate case existed pre-commit, untouched here, and
  carries no evidence/marker regardless).
- `docs/packets/dispatchers/maker_result.yaml:32-36` still documents "FAILED: no key" and was last
  touched by `ab31624bd` (Task 9), not this commit — confirmed via `git log -1` and
  `git show 5d2a2ed52 --stat -- docs/packets/dispatchers/maker_result.yaml` (empty diff).

## 6. Gates and build, re-run against the committed tree

```
matrix --check:        exit 0  (2 unrelated n-a notes, no errors)
operations --check:    exit 0  ("operations check OK (0 absent-writer note(s))")
fname-doc --check:     exit 0  ("fname-doc check OK (277 structs without an audit report carry no fname)")
dispatcher-lint:       exit 0  ("dispatcher-lint: clean")
doc-freshness --check: exit 0  ("doc-freshness: PROCESS.md packet-process-facts agree with the tool (10 versions, 7 CI gates).")
gate-check --check:    exit 0  ("gate-check: all 21 gate(s) have verified byte-fixtures on both straddling versions (1 partial-by-design).")
```

`docs/packets/audits/STATUS.md:334`:
```
| MAKER_RESULT | CUserLocal::OnMakerResult | character/clientbound/MakerResultCreate (T1) |  | ⬜ |  | ⬜ | 0x0C7 | ✅ | 0x0CB | ✅ | 0x0D9 | ✅ | 0x0DD | ✅ | 0x0E6 | ✅ | 0x0FA | ✅ | 0x0F8 | ✅ | 0x0E2 | ✅ |
```
⬜ on gms_v48/gms_v61, ✅ on all eight applicable versions (gms_v72/79/83/84/87/92/95, jms_v185),
no 🧩, no 🟥 — matches the brief's required end state exactly.

`go build ./... && go test ./...` in `libs/atlas-packet`: all packages `ok` (character/clientbound
included). Same in `tools/packet-audit`: all packages `ok`.

## Also-fix item (non-blocking, cheap)

`.superpowers/sdd/plan/task-9-report.md:266-268` now documents the fifth wrong brief hex
(`4021313` → correct `41 5C 3D 00`, brief had `41 5F 3D 00`), matching the shipped fixture. Done.

## Not evaluable

- No `ida-pro` access; decompiled ground truth for the four arms' per-version addresses
  (`0x86a1b6`/`0x86a1c0` etc.) is taken on the wire-derivation.md/pinning-report.md record, not
  independently re-derived from the binary. Declared limitation, per instructions, not spent as
  a finding.

## Pre-existing/flagged issue, not this commit's fault

The implementer's report notes latent audit-report drift on wholesale regeneration (e.g. gms_v95
`Action.json` resolving to a different Atlas file with a different verdict). This commit
deliberately avoided a wholesale regeneration and cherry-picked only the 32 new files — confirmed
above by the diff-filter check — so leaving that drift alone was the right call for this unit.
Out of scope here; tracked separately per the task instructions.

## Scope check

Reviewed: `5d2a2ed52` diff in full (`git show --stat`), `run.go` code diff, `families.yaml` diff,
one registry file diff, one evidence-file diff pair, `maker_result_test.go` diff, `status.json`
diff head, `dispatchers/maker_result.yaml` (read, contract file the diff depends on for the
FAILED-arm ruling and mode-numbering — genuinely load-bearing, not a survey). Ran all six gates
and both modules' build/test against the committed tree. Reproduced the negative control myself
rather than trusting the implementer's claim. No file was written into the worktree; `go clean
-cache` was never run.
