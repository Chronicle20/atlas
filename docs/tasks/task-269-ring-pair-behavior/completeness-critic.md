# Packet completeness critic — task-269-ring-pair-behavior

Diff base: `BASE=32d55cb2164866063fe336256128621443c6d077` (= `git merge-base origin/main HEAD`, matches the prompt's `32d55cb21`), `HEAD=e5f7cf047` (matches the prompt's `e5f7cf0`).

**Verdict: 1 finding (CHANGED-BUT-UNCLAIMED, inherited-branch), 0 CLAIMED-BUT-UNVERIFIED.** Task-269's own codec/gate/matrix work is fully claimed and fully verified. The one finding is a scope-boundary artifact of stacking on unmerged task-240, not a task-269 defect.

## Preliminary rulings requested by the dispatch prompt

**1. Manifest schema deviation (`versions:` as a list of single-key per-op mappings).** I read it correctly and it did not impair the audit. `coverage-manifest.yaml:70-100` gives four `- <packet>: [v1, v2 (n-a), ...]` entries, one per claimed op, rather than one flat scalar list applied uniformly. I parsed each entry as its own op→version-list pair (`character/clientbound/CharacterSpawn`: all 10 versions; `CharacterInfo`: all 10; `CharacterAppearanceUpdate`: 9, `gms_v48` n-a; `character/CharacterData`: 9, `gms_v48` n-a) and diffed each op's declared list against its own final `status.json` cells (Step 3 below) — the per-op granularity this form provides is exactly what let me confirm `CharacterAppearanceUpdate`/`CharacterData` are held to a 9-version bar while `CharacterSpawn`/`CharacterInfo` are held to 10, which a flat list could not have expressed. No impairment; if anything the mapping form gave more precise coverage than PROCESS.md's flat-list example would allow me to check.

**2. Ruling 1 (WITHDRAWN as never-applicable) — CONFIRMED from source.**
- `tools/packet-audit/internal/evidence/hash.go:14-38` (`FunctionHash(exportPath, fname string)`): reads `exportPath` (an on-disk IDA export JSON), unmarshals `file.Functions[fname]`, canonicalizes, and hashes it. The only input is the IDA export.
- `tools/packet-audit/internal/matrix/evidence_input.go:30`: `h, err := evidence.FunctionHash(exp, r.IDA.Function)` where `exp := exportPaths[r.Version]` (line 21, an on-disk export path) is compared at line 39 (`fresh := h == r.IDA.DecompileSHA256`) against the record's pinned hash. Both sides of the freshness check are export-vs-pinned-export; `libs/atlas-packet` Go source is never read by this path.
- `grep -c "packet-audit" tools/verify.sh` → `0` (confirmed live). `.github/workflows/packet-matrix.yml` is the only place `packet-audit` runs (confirmed: `go test ./...` at line 28, `fname-doc --check` at line 32, etc.).

Ruling 1's assertion is correct: a Go encoder change on this branch cannot stale a `decompile_sha256` by construction. No stale-evidence problem exists that the manifest is hiding.

## CHANGED-BUT-UNCLAIMED

| kind | file/packet | evidence | recommendation |
|---|---|---|---|
| codec | `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go` (new file, +62 lines) | `git log --oneline $BASE..HEAD -- libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go` → single hit `72e9be490 feat(channel): implement BUY_PACKAGE and route BUY_OTHER_PACKAGE`, a commit that belongs to `docs/tasks/task-240-cash-shop-stub-operations/` (confirmed: `b554b6d7b plan(task-240-cash-shop-stub-operations)` is reachable in the same range and `docs/tasks/task-240-cash-shop-stub-operations/` exists as its own task folder). `72e9be490` is not reachable from any task-269 commit boundary (task-269's own range starts at `43d164c25 spec(task-269): initial PRD for ring-pair-behavior`, well after task-240's commits in the log). `cash/serverbound` is neither in `coverage-manifest.yaml`'s `ops:`/`fields:` nor its `out_of_scope:`. | Not a task-269 defect — task-269 is stacked on unmerged task-240 (`git merge-base --is-ancestor 72e9be490 origin/main` fails: not yet in `origin/main`). No manifest action needed on *this* branch; task-240 owns its own coverage-manifest and review. Flagging here only because the mechanical diff range (`merge-base origin/main HEAD`) can't distinguish "task-269's own work" from "unmerged prerequisite branch bleed-through." Re-run this critic once task-240 merges to main and task-269 rebases, so the diff range narrows to task-269's own commits only. |
| matrix | `cash/serverbound/CashShopOperationBuyOtherPackage` (new row, `gms_v95` → `verified`) | `git diff $BASE...HEAD -- docs/packets/audits/status.json` shows a wholly new row at status.json (added block, `+62` lines) for this packet, same commit `72e9be490`. | Same as above — task-240's own matrix delta, not task-269's. No manifest action needed on task-269's branch. |

No other `libs/atlas-packet/**/*.go` (non-test) file outside task-269's declared scope was touched: the full non-test codec touch-set is `appearance_update.go`, `info.go`, `spawn.go` (all `character/clientbound`, claimed), `character/data.go` (claimed via `character/CharacterData`), and `model/ring.go` (a new shared struct file, not itself a status.json packet, but named explicitly in `coverage-manifest.yaml:103-104`'s `fields:` notes as the implementation of the RingSet/RingRecords blocks for the four claimed ops — treated as adequately declared, not a separate finding).

No version-gate change (`MajorVersion`/`MajorAtLeast`/`IsRegion`/`Region()`) landed outside a claimed dir. Full attributed list (`git diff $BASE...HEAD -- libs/atlas-packet` filtered to those tokens, headers reattached):
- `character/clientbound/appearance_update.go` — new `hasTrailingCompletedSetItemId` gate (`t.IsRegion("GMS") && t.MajorAtLeast(87)`) — claimed (`character/clientbound/CharacterAppearanceUpdate`).
- `character/data.go` — removed the old legacy gate `(t.Region()=="GMS" && t.MajorVersion()>28) || t.Region()=="JMS"` (replaced by the ring-aware path) — claimed (`character/CharacterData`).
- `model/ring.go` — same legacy-boundary gate, now centralized in the shared ring codec — covered by the `fields:` note citing `model/ring.go` for both claimed ops.
- All other gate-token hits are in `_test.go` files (comments/fixture setup), not production gates.

Matrix delta outside the above two task-240 rows: only two other cell transitions, both inside claimed rows — `character/clientbound/CharacterInfo` `gms_v92`: `incomplete` → `verified` (confirmed via row context at `docs/packets/audits/status.json` line ~6893, `"op": "CHAR_INFO"`), and `character/clientbound/CharacterSpawn` `gms_v92`: `incomplete` → `verified` (row context `"op": "SPAWN_PLAYER"` at line ~16800). Both are `character/clientbound/*` packets explicitly in `coverage-manifest.yaml:45,76` (`CharacterSpawn`) and `:51,80` (`CharacterInfo`), each declaring `gms_v92` in its version list. `field/clientbound/FieldSetField` (the row carrying `CharacterData`'s opaque span) shows **zero** cell diffs (`git diff $BASE...HEAD -- docs/packets/audits/status.json | grep -c FieldSetField` → `0`), consistent with the manifest's claim (`coverage-manifest.yaml:109-111`) that this branch doesn't move that row's disposition.

## CLAIMED-BUT-UNVERIFIED

None. Checked every `claimedOps` cell against the final (HEAD) `status.json`:

| op (packet) | versions checked | result |
|---|---|---|
| `character/clientbound/CharacterSpawn` | all 10 (`gms_v48`…`jms_v185`) | all `verified` |
| `character/clientbound/CharacterInfo` | all 10 | all `verified` |
| `character/clientbound/CharacterAppearanceUpdate` | 8 declared-verified (`gms_v61`…`gms_v87`, `gms_v95`, `jms_v185`; `gms_v48` manifest-declared n-a, `gms_v92` manifest-declared excluded) | all 8 `verified`; `gms_v48`/`gms_v92` correctly excluded from the claim (not checked as claims) |
| `character/CharacterData` | not its own `status.json` row (opaque span inside `field/clientbound/FieldSetField`/`SET_FIELD`); manifest documents this explicitly (`coverage-manifest.yaml:65-68,109-111`) rather than claiming a row-level `verified` state that doesn't exist for this packet. Not flagged as CLAIMED-BUT-UNVERIFIED since the manifest itself discloses the row doesn't carry per-packet cells and cites the byte-fixture tests (Tasks 4/5/6/10/11) as the actual verification mechanism instead. | n/a — correctly disclosed, not silently claimed |

Tree state: `git status --short` shows only pre-existing untracked files (`docs/tasks/task-269-ring-pair-behavior/{agent-ledger.tsv,audit-tasks-1-8.md,reviews/}`), none created or modified by this audit. No codec, registry, template, `gates.yaml`, evidence record, `STATUS.md`, or `status.json` was touched.
