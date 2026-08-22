# Review — Task 3: Registry hygiene, `JMS_SLASH_COMMAND` split for `CUICharacterSaleDlg::SendCheckDuplicateIDPacket`

Range reviewed: `2f2e5d8c7..a4f9a4172`
Diff stat: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`,
`docs/packets/registry/gms_v83.yaml`, `docs/packets/registry/gms_v87.yaml`,
`docs/packets/registry/gms_v92.yaml`, `docs/packets/registry/gms_v95.yaml`
(6 files, +168/-67).

## Controller rulings

1. **SPLIT, not rename** — ✅. `docs/packets/registry/jms_v185.yaml` does not appear
   anywhere in `git diff --stat 2f2e5d8c7..a4f9a4172` (confirmed by direct stat run).
   `jms_v185.yaml:3643` still carries `op: JMS_SLASH_COMMAND`,
   `fname: CUICharacterSaleDlg::SendCheckDuplicateIDPacket`, opcode 271, untouched.
   `STATUS.md`'s regenerated `JMS_SLASH_COMMAND` row now shows `n-a`/⬜ on every GMS
   column and retains only jms_v185 `0x10F ❌`, consistent with the row simply losing
   its GMS siblings after the split (not an edit to the JMS file itself).
2. **gms_v84 out of scope, VERSION-ABSENT** — ✅. `docs/packets/registry/gms_v84.yaml`
   does not appear in the diff. `gms_v84.yaml` still carries only the unrelated,
   pre-existing `CLogin::SendCheckDuplicateIDPacket` (ordinary login-socket probe,
   opcode 8, §6.2) — confirmed by grep; no `MAPLELIFE_CHECK_NAME` row was added.

## Requirement-by-requirement (brief Step 1 — the six registry files)

| File | Expected (§6.1/§6.3) | Actual in diff | ✅/❌ |
|---|---|---|---|
| `gms_v83.yaml` | add `MAPLELIFE_CHECK_NAME` opcode 256 @0x7d75ab | added exactly this, `provenance: ida-discovered`, note cites §6.1/§6.2/§6.3 | ✅ |
| `gms_v84.yaml` | no row (VERSION-ABSENT) | untouched | ✅ |
| `gms_v87.yaml` | rename `JMS_SLASH_COMMAND`→`MAPLELIFE_CHECK_NAME`, opcode 270 unchanged @0x82e04d | renamed exactly, provenance updated, address added | ✅ |
| `gms_v92.yaml` | add `MAPLELIFE_CHECK_NAME` opcode 301 @0x756250 | added exactly this, note explicitly disclaims collision with clientbound 301 `MOB_ATTACKED_BY_MOB` | ✅ |
| `gms_v95.yaml` | rename, opcode 311 unchanged @0x777d20 | renamed exactly, provenance updated, address added | ✅ |
| `jms_v185.yaml` | keep as-is, §6.3 unresolved | untouched | ✅ |

Every `opcode:`/`ida.address:` pair written was cross-checked directly against the
§6.1 table (`docs/tasks/task-246-maple-life-character-creation/derivation.md:766-771`):
256/0x7d75ab, 270/0x82e04d, 301/0x756250, 311/0x777d20 — all four match exactly, no
invented values. `provenance: ida-discovered` and an `ida: {address: ...}` block match
the established convention used elsewhere in the same files (e.g. `CLAIM_RESULT` in
`gms_v95.yaml`, which also uses `provenance: ida-discovered`). Each `note:` cites
`derivation.md §6` as required.

## Step 3 — no other op's coverage cell flipped state

`git diff` on `status.json` shows dozens of raw `"state"` line changes, but that is a
false signal from list-insertion reflow, not real content drift. I keyed both
`status.json` snapshots by `(op, fname)` in Python and diffed structurally
(`/tmp/old_status.json` = `2f2e5d8c7`, `/tmp/new_status.json` = `a4f9a4172`):

```
added rows: {('MAPLELIFE_CHECK_NAME', None)}
removed rows: set()
changed rows (excluding add/remove): [('JMS_SLASH_COMMAND', None)]
```

Only the two ops this task touched changed. ✅ — matches the report's claim, verified
independently rather than trusted.

## Regeneration — full vs. partial/misdirected

The `-*-dir` flag workaround the implementer used for the brief's broken `cd
tools/packet-audit && go run . matrix` instruction correctly redirected the tool's
*flagged* inputs/outputs (registry, evidence, exports, templates, audits) — confirmed:
`exportHashes` in `status.json` is byte-identical before/after (`old['exportHashes'] ==
new['exportHashes']` → `True`), and the STATUS.md coverage percentages/row content are
internally consistent with a real full pass over all 10 versions.

**However, one artifact is corrupted.** `matrix.go:491-497`'s `toolTreeSHA()` runs `git
ls-tree -r HEAD tools/packet-audit` **unconditionally relative to process cwd** — it has
no flag override, unlike the other five path-configurable inputs. The committed
`STATUS.md:7` / `status.json.toolSha` now reads
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`, which is exactly
`sha256("")` — confirmed by direct computation. Reproduced the exact failure mode: `cd
tools/packet-audit && git ls-tree -r HEAD tools/packet-audit` returns 0 entries (the
path resolves to the nonexistent `tools/packet-audit/tools/packet-audit` relative to
that cwd), vs. 244 entries run from the repo root. The prior, correct toolSha was
`95cea500fa2e248416bb29cd82bf32159e955a6e849bfed26109deade89a64ad`.

This means the regeneration **was** run from `tools/packet-audit` as cwd (as the
implementer's report states), and while the explicit `-*-dir` flags correctly
compensated for every flagged path, they could not compensate for this one hardcoded,
unflagged internal command. The result is a factually wrong, uninformative hash
(`sha256("")`) committed as the tool-provenance field of the audit's ground-truth
artifact — this directly fails the task's own instruction to confirm the regenerated
audits are "consistent with a correct full regeneration and not a partial or
misdirected one." It is misdirected in exactly the one place the implementer's
workaround didn't reach, and the implementer's report does not mention noticing it (the
three check commands don't validate toolSha, so nothing failed loudly).

**Blocking**: `docs/packets/audits/STATUS.md:7` and `docs/packets/audits/status.json`
(`toolSha` field) must be regenerated from repo-root cwd (or with `git -C` fixed inside
`toolTreeSHA()`) before this lands — this task's own diff must not carry a corrupted
provenance hash into the shared audit trail that later tasks in this plan depend on for
drift detection.

## fname-doc drift — independently re-verified

Ran `fname-doc --check` against the review worktree (post-task) and against a scratch
worktree pinned at the exact pre-task base commit `2f2e5d8c7` (not a stash — a real
detached-HEAD checkout, to remove any doubt about stash fidelity):

```
# both pre- and post-task:
fname-doc DRIFT: ../../libs/atlas-packet/character/serverbound/skill_macro.go SkillMacro (want "CWvsContext::OnMacroSysDataInit")
fname-doc DRIFT: ../../libs/atlas-packet/summon/serverbound/move.go Move (want "CSummonedPool::OnMove")
fname-doc: 2 drift, 0 missing — run `packet-audit fname-doc` to fix
```

Identical in both trees. Genuinely pre-existing, unrelated to `JMS_SLASH_COMMAND`,
`MAPLELIFE_CHECK_NAME`, or any GMS probe opcode touched by this task. ✅ not this task's
problem.

`operations --check` re-run independently: `operations check OK (0 absent-writer
note(s))` — exit 0. ✅

## Other conventions

- No CRLF present in any touched registry file before or after (`grep -c $'\r'` → 0 in
  both); no line-ending normalization concern.
- No literal home/absolute paths introduced in any committed file.
- Commit message (`fix(packets): resolve the JMS_SLASH_COMMAND row for
  CUICharacterSaleDlg::SendCheckDuplicateIDPacket`) matches the brief's Step 4 exactly.

## Not evaluable

- Whether the brief's Step 2 command literally as written (`cd tools/packet-audit && go
  run . matrix` with no flags) is itself a pre-existing tooling defect worth fixing is
  outside this task's file scope (brief lists only the six registry files +
  derivation.md as in-scope files) — noted as a real gap but not this diff's problem to
  fix. Flagging the toolSha bug above is different: that bug's *symptom* landed inside
  this diff's own regenerated artifacts, which is squarely in scope.

## Summary

Spec compliance: 6/6 registry-file requirements ✅, both controller rulings honored ✅,
no unrelated op flipped state ✅, fname-doc drift independently confirmed pre-existing
✅, no invented opcodes ✅. Task quality: one genuine defect — the regenerated
`STATUS.md`/`status.json` carry a corrupted `toolSha` (`sha256("")`) caused by an
un-flagged internal `git ls-tree` call resolving against the wrong cwd, which the
implementer's own verification did not catch because none of the three required check
commands validates that field.
