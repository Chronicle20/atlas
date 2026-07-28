# Task 169 — Phase 2 close (matrix expressiveness & visibility)

All Phase 2 tasks complete except T2.6 (stretch, deliberately skipped to protect
the phase). Every behavioral change was TDD (failing test first). Branch stayed
`task-169-packet-process-consistency` throughout. Final gate green.

## Per-task outcomes

| Task | Status | Summary |
|------|--------|---------|
| T2.0 | done | ToolSHA now hashes only the tool's committed `.go` blobs (excludes README/docs/testdata), sorted by path — deterministic + order-stable. Editing tool docs no longer invalidates the matrix. |
| T2.1 | done | Sub-struct `n-a` disposition from `_unimplemented.json` threaded into `matrix.Build`. THE one count-moving change (see delta below). |
| T2.2 | done | The three distinct `StatePartial` notes render as `🟡ᶠ / 🟡ᵈ / 🟡ᵖ` in STATUS.md + a legend row. Render-only; `status.json` and totals unchanged. |
| T2.3 | done | `packet-audit support-summary` writes `docs/packets/audits/support/<version>.md` for all 9 versions (totals, verified%, gap table n-a/unverified/conflict). Deterministic; totals match status.json. |
| T2.4 | done | `packet-audit status <version>` prints the version summary + open-gap + stale-evidence rows to stdout. Read-only; writes nothing. Reuses the T2.3 aggregation (`matrix.Summarize`). |
| T2.5 | done | `dispatcher-lint` family-cap guard (FAM-CAP): every `dispatchers/*.yaml` family must be discrete-implemented (has `#`-suffixed case arms in run.go) or families.yaml/baseline-listed. |
| T2.6 | skipped | Per-arm rollup — stretch; skipped per instructions (non-blocking). |

## T2.1 count delta (AC-3 documented reclassification)

9 sub-struct cells reclassified `incomplete` (❌) → `n-a` (⬜). All other cells
byte-identical to `baseline-counts.md`.

| version | ❌ before → after | ⬜ before → after | verified% before → after |
|---------|-------------------|-------------------|--------------------------|
| v48 | 163 → 156 (−7) | 627 → 634 (+7) | 50.2% → 51.2% |
| v79 | 217 → 215 (−2) | 417 → 419 (+2) | 46.6% → 46.8% |
| v61, v72, v83, v84, v87, v95, JMS185 | unchanged | unchanged | unchanged |

`✅ / 🧩 / 🟡 / 🟥` unchanged in every version. Matching rule + the exact 9
cells: `phase2-substruct-delta.md`. Safe by construction — only explicit
`packet:` paths and suffix-qualified (`#`) fnames resolve; bare base-fname +
numeric-case dispatcher-arm dispositions are excluded (they collide with
implemented sibling structs' IDANames).

## Fires-on-violation proofs

- **T2.0** `TestHashGoTreeEntriesStableAcrossNonGoChange`: SHA stable across a
  README blob change, moves on a `.go` change, order-stable, testdata excluded.
- **T2.1** `TestSubStructDispositionedIsNA` / `…UndispositionedIsIncomplete` /
  `TestResolveUnimplemented` (bare base fname must NOT resolve → would downgrade
  an implemented sibling like `login/clientbound/AuthSuccess`).
- **T2.2** `TestPartialNotesRenderDistinctly`: 3 notes → 3 distinct glyphs;
  🟡 total still counts all partials as one class.
- **T2.4** `TestStatusRunPrintsSummary` / `…UnknownVersion` against a fixture.
- **T2.5** `TestFamilyCapPhantomFailsDiscretePasses` (phantom → 1 FAM-CAP
  violation), `…FamiliesListedPasses`, `…MissingFnameFails`, `…RealTreeClean`.
  End-to-end: seeding `dispatchers/zzz_test_family.yaml` made
  `dispatcher-lint` exit 1 with a FAM-CAP violation naming
  `CPhantom::OnFakeResult`; removing it returned exit 0. (Phantom NOT committed.)

## Final gate (all at branch tip)

- `go test ./tools/packet-audit/...` — pass
- `go vet ./tools/packet-audit/...` — clean
- `go build ./tools/packet-audit/...` — ok
- `matrix --check` — exit 0
- `operations --check` / `fname-doc --check` / `dispatcher-lint` — exit 0

## ToolSHA / regeneration ordering note

`toolTreeSHA` reads `git ls-tree -r HEAD tools/packet-audit`, so a matrix
regenerated in the same commit as a `.go` change bakes the PRE-commit HEAD's
SHA. A final docs-only commit (`chore: regenerate matrix at branch tip`)
rebakes the SHA against the committed sources; since it touches no `.go`, the
SHA is now stable and `matrix --check` is clean at the tip.

## Commits

- `387fc3053` T2.0 ToolSHA hashes only .go sources
- `3fabd9376` T2.1 sub-struct n-a disposition (the one count-moving change)
- `d0d304a09` T2.2 disambiguate partial glyphs
- `7d9b19deb` T2.3 + T2.4 support-summary + status query
- `3a81f6fe9` T2.5 family-cap guard in dispatcher-lint
- `7b805a444` regenerate matrix at branch tip
