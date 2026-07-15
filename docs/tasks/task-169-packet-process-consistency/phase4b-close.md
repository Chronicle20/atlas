# Task 169 — Phase 4b close (boundary-fixture check, completeness critic, gate-lint re-measure)

Scope: T4.2 (gate registry + boundary-fixture check), T4.4 (coverage manifest +
completeness critic), T4.1b (gate-lint narrowing re-measure). Deferred from
phase 4a. Pure tooling/docs/CI — no wire/registry/template/matrix-cell change.

---

## T4.1b — Gate-lint narrowing re-measure (FR-3.1a)

Phase 4a's gate-lint flagged all four operators (`>`/`>=`/`<`/`<=`) at a
boundary and hit **220** sites, ~185 of which were the CORRECT
`Region()=="GMS" && MajorVersion()>=N` idiom — pure noise — so it shipped
report-only.

**Narrowed** (`tools/packet-audit/cmd/gatelint.go`) to the genuinely
off-by-one-prone forms only, the exact `>83` footgun shape:
- right form: `MajorVersion() > N` and `MajorVersion() <= N`
- left form: `N < MajorVersion()` and `N >= MajorVersion()`

The correct idioms `>= N` / `< N` (and their left twins `<= N` / `> N`) are no
longer flagged — they split at N-1/N and don't re-bucket version N.

**Real-tree hit count: 220 → 35.** Breakdown of the 35: `>87`×16, `>61`×9,
`<=87`×6, `<=95`×2, `<=61`×2 (left-form: 0).

**Decision: keep REPORT-ONLY.** Every one of the 35 is a task-113
code-gate-audit VERIFIED-CORRECT gate whose boundary happens to sit between two
adjacent version columns (e.g. `>87` == `>=95` today because no v88..v94 GMS
column exists) — 0 are bugs. Making it blocking would demand an
`//gate-lint:allow` annotation on each of those 35 wire-source files, which is
out of scope for a pure-tooling phase (no wire-source edits). The narrowing is
still a real win: the report is now 35 latent-footgun sites worth a human glance
instead of 220 lines of idiom noise. Documented in PROCESS.md; the tool comment
records the blocking-follow-up path (35 inline allows or an allowlist file).

Both-directions proof (`cmd/gatelint_test.go`, updated):
- `TestGateLintFlagsBoundaryComparisons`: `> 83` flagged; `MajorAtLeast(87)`,
  `> 12` (non-boundary), and now `>= 95` / `< 87` (correct idioms) all clean.
- `TestGateLintFormsAndTestSkip`: footgun `<= 87` and left-twin `83 >=
  MajorVersion()` flagged; correct `87 <= MajorVersion()` and `< 79` clean;
  `_test.go` skipped.

## T4.2 — Gate registry + boundary-fixture check (FR-3.1b)

**`docs/packets/gates.yaml`** — 19-gate seed extracted from task-113's
`code-gate-audit.md`, spanning **all 7 adjacent version boundaries**:

| Boundary | Gates |
|---|---|
| v48/v61 | CharacterSpawn new-year-card `>=61` |
| v61/v72 | WorldCharacterListRequest socketAddr `>=72`; CreateCharacter base-stat `<=61` |
| v72/v79 | CreateCharacter jobIndex `>=73` (the intra-legacy discriminator) |
| v79/v83 | CharacterSpawn 2nd-effect byte `>=83` |
| v83/v84 | CharacterAttackMeleeRequest DR-block `>=84`; MonsterMovementRequest `>=84` |
| v84/v87 | ChatWhisper `>=87`; AllCharacterListRequest `>=87`; CharacterInfo `>=87`; MonsterMovement `>=87` |
| v87/v95 | CharacterSpawn/ItemUpgrade/CharacterExpression/CharacterViewAllCharacters/CharacterList `>87`; CharacterInfo `<=87`; FieldSetField `>=95`; CharacterDamage `>=95` |

Schema per entry: `packet`, `direction`, `field`, `boundary`,
`lower_version_key`, `upper_version_key`, optional `expect: partial` + `reason`.
A header comment documents the authoring contract (how to add a row, how to mark
a real coverage gap `partial` without fabricating a fixture).

**`packet-audit gate-check`** (`cmd/gatecheck.go`) reads gates.yaml + status.json
(via `matrix.LoadMatrix`) and, per gate, asserts SOME matrix row for
`(packet, direction)` is `verified` at BOTH the lower and upper version key
(EXISTS-a-verified-row semantics, so packets that appear as multiple op/sub-struct
rows resolve gracefully). Config errors (unknown version key, typo'd packet,
`partial` without reason) fail too. Default reports (exit 0); `--check` exits 1
on any failing `full` gate.

**Both-directions proof** (`cmd/gatecheck_test.go`):
- `TestGateCheckBothDirections`: both-verified → exit 0; upper side `incomplete`
  → exit≠0 and names `gms_v95`.
- `TestGateCheckPartial`: `expect: partial` + reason with one side unpinned →
  passes; `partial` without a reason → config-error fail.
- `TestGateCheckUnknownPacket`: a typo'd packet → fail ("no matrix row").
- `TestGateCheckRealTreePasses`: the committed gates.yaml is green against the
  real status.json.

**Real-tree state: GREEN.** All 19 seeded gates are both-sides-verified (`0`
partial). No fabricated fixtures; the seed was chosen from packets already
verified on both straddling columns.

**CI decision: BLOCKING.** Because the seed is green, wired a blocking
`gate-check` step into `.github/workflows/packet-matrix.yml` (after
doc-freshness, before matrix). PROCESS.md's CI-gate prose list (now 7) + the
`packet-process-facts` `ci_gates` block + `doclint.go`'s `ciGateWorkflowSubstr`
map were all updated to include `gate-check`, so doc-freshness cross-checks it
(verified: "9 versions, 7 CI gates").

## T4.4 — Coverage manifest + completeness critic (FR-3.3)

**Schema** — added a "Coverage manifest (packet tasks)" section to
`docs/packets/PROCESS.md`: `docs/tasks/<task>/coverage-manifest.yaml` =
`{ops, versions, fields, out_of_scope}`. `ops` accepts an op name OR a packet
path; `out_of_scope` whitelists intentional incidental touches.

**Agent** — `.claude/agents/packet-completeness-critic.md`, read-only. Concrete
diff logic specified against real inputs:
- diff base `BASE=$(git merge-base origin/main HEAD)`.
- **CHANGED-BUT-UNCLAIMED** (the class-8 scope hole): touched codecs via
  `git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$'`;
  touched version-gates via
  `git diff $BASE...HEAD | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`
  (called out as the higher-severity subclass); matrix delta via
  `git diff $BASE...HEAD -- docs/packets/audits/status.json` parsing
  `cells[...].state` transitions. Anything not in `ops`/`out_of_scope` is flagged.
- **CLAIMED-BUT-UNVERIFIED**: for each `op × version`, read the final HEAD
  `status.json` cell; not `verified` → flag (`partial`/`incomplete` never
  satisfy a claim).
- Writes `docs/tasks/<task>/completeness-critic.md`; mutates nothing; a missing
  manifest is the top finding.

**How it catches scope holes**: the off-by-one/reshift class lands when a change
moves a codec or gate the task never declared and no fixture pinned. gate-lint
(shape) and gate-check (fixture pairs) are the mechanical guards; the critic is
the SEMANTIC guard — it fails review when the diff and the declared manifest
disagree, so a silent gate move can't ride along unclaimed.

**Wired** into the pre-PR review flow via `docs/superpowers-integration.md`
(Packet Work section) as the packet-specific review companion, run alongside the
guideline reviewers.

---

## Verification (from the worktree root)

- `go build ./tools/packet-audit/...` → ok
- `go vet ./tools/packet-audit/...` → ok
- `go test -race ./tools/packet-audit/...` → all ok
- All existing gates exit 0: `matrix --check`, `operations --check`,
  `fname-doc --check`, `dispatcher-lint`, `doc-freshness --check`, and the new
  `gate-check --check`.
- Tree clean (no dirty STATUS.md/status.json). The gate-check subcommand added
  .go files → the committed status.json/STATUS.md tool-SHA line was refreshed
  against the FINAL HEAD (tool-SHA line only; zero matrix cells, zero counts).
- No `go.mod` touched → no docker bake required.

## Commits (branch `task-169-packet-process-consistency`)

1. `refactor(packet-audit): narrow gate-lint to off-by-one-prone forms` (T4.1b)
2. `feat(packet-audit): gate-check boundary-fixture pairs + gates.yaml` (T4.2)
3. `feat(packet-audit): coverage manifest schema + completeness-critic` (T4.4)
4. `chore(packet-audit): refresh matrix tool-SHA after P4b tool changes`
