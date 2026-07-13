# Task 169 — Phase 4a close (mechanical churn guards + CI wiring)

Scope: T4.1 (gate-lint), T4.3 (export non-destructive default), T4.5
(doc-freshness), T4.6 (CI wiring). T4.2 (gate registry) and T4.4
(manifest/critic) are deliberately deferred to a separate agent.

Every guard is TDD and proven in BOTH directions (fails on a seeded violation,
passes on the clean tree).

---

## T4.3 — Export non-destructive default (FR-3.2)

- **CLI surface:** `packet-audit export` — default now refuses to overwrite an
  existing `--output` when the fresh harvest DIFFERS: writes `<output>.new` + an
  `added/removed/changed` function-key summary to stderr and exits non-zero
  (code 4), leaving the committed file byte-unchanged. `--force` restores the
  old overwrite. `--splice <FName>` merges a single harvested entry
  (VERIFYING_A_PACKET.md §10 surgical path), preserving every other entry
  byte-for-byte.
- **Code:** `tools/packet-audit/cmd/export.go` (guard + `writeExportFile`,
  `summarizeExportDelta`), `internal/idasrc/export.go` (`SpliceExport` — typed
  round-trip so untouched entries re-marshal identically), flags in
  `cmd/root.go`.
- **Fires-on-violation proof** (`cmd/export_test.go`):
  - `TestExportRefusesDifferingOverwrite`: differing harvest, no `--force` →
    exit≠0, committed file byte-unchanged, `<output>.new` written, "changed" in
    the stderr summary.
  - `TestExportForceOverwrites`: `--force` overwrites, no `.new` sidecar.
  - `TestExportIdenticalIsIdempotent`: identical re-harvest succeeds (exists but
    equal → no refusal).
  - `TestExportSpliceMergesOneEntry`: `--splice` updates only the target entry;
    the untouched entry stays byte-identical.
- **CI:** none — the export guard is a runtime behavior of a human-run command;
  CI never harvests. Documented as such in PROCESS.md.

## T4.5 — Doc-freshness lint (FR-2.3)

- **CLI surface:** `packet-audit doc-freshness --check`.
- **Ground truth compared:** parses the `packet-process-facts` fenced block in
  `docs/packets/PROCESS.md` and asserts:
  - `version_count` / `version_keys` vs `matrix.VersionKeys`
  - `dispatcher_lint_baseline_families` vs
    `docs/packets/dispatcher-lint-baseline.yaml` (`exempt_families`)
  - `family_cap_dispatchers` vs `docs/packets/evidence/families.yaml`
    (`dispatchers`)
  - `ci_gates` vs `.github/workflows/packet-matrix.yml` (both directions over
    the known gate map)
- **Code:** `tools/packet-audit/cmd/doclint.go`; dispatch in `cmd/root.go`.
- **Fires-on-violation proof** (`cmd/doclint_test.go`):
  - `TestDocFreshnessRealTreePasses`: real tree → exit 0.
  - `TestDocFreshnessDetectsVersionCountDrift`: `version_count: 9`→`5` copy →
    exit≠0, "version_count" on stderr.
  - `TestDocFreshnessDetectsVersionKeyDrift`: `- gms_v84`→`- gms_v85` copy →
    exit≠0.
  - `TestDocFreshnessDetectsMissingCIGate`: workflow copy with `operations
    --check` removed → exit≠0, "ci_gates" on stderr.
- **CI:** blocking step `doc-freshness check` added to `packet-matrix.yml`.

## T4.1 — Gate-idiom lint (FR-3.1a)

- **CLI surface:** `packet-audit gate-lint` (report-only, exit 0) /
  `gate-lint --check` (exit≠0 on any hit).
- **What it flags:** raw `MajorVersion()` comparisons (`>`/`>=`/`<`/`<=`, either
  operand order) against a client-version boundary `{61,72,79,83,84,87,95}` on
  wire-encode paths in `libs/atlas-packet` — the documented `>83` off-by-one
  footgun class (`bug_majorversion_gt83`). Narrow: base-version gates (`>12`,
  `>28`), `Region()` checks, and non-boundary constants are ignored. Inline
  `//gate-lint:allow <reason>` suppresses a site. `_test.go` skipped.
- **Code:** `tools/packet-audit/cmd/gatelint.go`; dispatch in `cmd/root.go`.
- **Fires-on-violation proof** (`cmd/gatelint_test.go`):
  - `TestGateLintFlagsBoundaryComparisons`: seeded `MajorVersion() > 83` → 1 hit
    at boundary 83; `MajorAtLeast(87)` clean; `> 12` not flagged;
    `//gate-lint:allow` line not flagged.
  - `TestGateLintFormsAndTestSkip`: number-on-left (`87 <= MajorVersion()`) and
    `< 79` both flagged; a `_test.go` boundary comparison is ignored.
- **Real-tree hit count: 220.** The established codebase idiom
  (`t.Region()=="GMS" && t.MajorVersion() >= N`) is used at ~220 legitimate
  sites; allowlisting them all would be pure churn.
- **Decision: REPORT-ONLY, NOT a blocking CI gate.** Wiring it blocking would
  demand ~220 `//gate-lint:allow` annotations. It ships as a manual/report tool
  (`--check` for targeted use + the fires-on-violation test). This is the honest
  call — not a fabricated clean pass. Documented in PROCESS.md.

## T4.6 — CI wiring

- Added one blocking step to `.github/workflows/packet-matrix.yml`:
  `doc-freshness check` → `go run ./tools/packet-audit doc-freshness --check`.
- The **family-cap guard (FR-5.1)** is already folded into
  `collectDispatcherViolations` → covered by the existing `dispatcher lint`
  step (comment updated to say so).
- `gate-lint` NOT wired (report-only, 220 legit hits). Export guard NOT wired
  (runtime behavior, CI never harvests). Both documented in PROCESS.md's CI-gate
  section.
- PROCESS.md's CI-gate prose list + `packet-process-facts` block updated to
  include `doc-freshness-check` (now 6 gates); the gate map in `doclint.go`
  updated to match, so doc-freshness cross-checks itself.

---

## Verification

All from the worktree root:

- `go test -race ./tools/packet-audit/...` → 0
- `go vet ./tools/packet-audit/...` → 0
- `go build ./tools/packet-audit/...` → 0
- Existing gates all exit 0: `matrix --check`, `operations --check`,
  `fname-doc --check`, `dispatcher-lint`, and the new `doc-freshness --check`.
- Adding the new `.go` files changed the packet-audit ToolSHA, so STATUS.md /
  status.json were regenerated (tool-SHA line ONLY — zero matrix cells, zero
  count changes) and committed. Tree is clean (no dirty STATUS.md).
- No `go.mod` touched → no docker bake required.

## Commits (branch `task-169-packet-process-consistency`)

1. `feat(packet-audit): non-destructive export default + --force/--splice` (T4.3)
2. `feat(packet-audit): doc-freshness lint for PROCESS.md facts` (T4.5)
3. `feat(packet-audit): gate-idiom lint for raw version boundaries` (T4.1)
4. `ci(packet-audit): wire doc-freshness gate; note family-cap + report-only guards` (T4.6)
5. `chore(packet-audit): refresh matrix tool-SHA after P4 tool changes`
