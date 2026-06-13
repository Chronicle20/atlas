# Backend guidelines audit — task-085 (packet-audit coverage matrix)

- **Scope:** `tools/packet-audit/**` (new packages: internal/opregistry, internal/seedcsv, internal/matrix, internal/evidence, internal/marker, internal/discover; cmd: registry.go, matrix.go, evidence.go, discover_ops.go; atlaspacket writer-name index) + `libs/atlas-packet/**` (test files only)
- **Guidelines Source:** backend-dev-guidelines skill (ai-guidance.md, anti-patterns.md, testing-guide.md)
- **Date:** 2026-06-12
- **Build:** PASS (both modules)
- **Tests:** packet-audit 13/13 packages ok; atlas-packet 58/58 packages ok; 0 failures; `-race` clean in both; `go vet` clean in both
- **Overall:** NEEDS-WORK (one dead-code FAIL; everything else passes or is N/A)

## Build & Test Results

- `tools/packet-audit`: `go build ./...` clean; `go test ./... -count=1` — 13 packages ok, 0 failures; `go test -race ./... -count=1` clean; `go vet ./...` clean.
- `libs/atlas-packet`: `go build ./...` clean; `go test ./... -count=1` — 58 packages ok, 0 failures; `go test -race ./... -count=1` clean; `go vet ./...` clean.
- Verified the test-only claim for libs/atlas-packet: `git diff main...HEAD --stat -- libs/atlas-packet | grep -v _test` returns only the summary line — all 31 changed files are `*_test.go` (180 insertions, 0 deletions, no production code touched).

## Scope classification

No `services/**` package changed. There are no domain packages (`model.go` + GORM entity), sub-domain packages, REST handlers, Kafka producers/consumers, or tenancy-aware DB code in this diff. The following checklist items are therefore **N/A**, stated explicitly per the audit mandate:

| IDs | Reason N/A |
|-----|------------|
| DOM-01..DOM-08, DOM-10, DOM-11, DOM-13..DOM-19 | No service domain package, no GORM, no REST, no processors/providers/administrators in scope |
| DOM-22 (Dockerfile lib mentions) | No service `go.mod` touched; `tools/packet-audit` is not a docker-built service |
| DOM-23 (Kafka topic naming) | No Kafka topics consumed or produced |
| DOM-24 (Kafka producer stub in tests) | Zero `AndEmit(` / `message.Emit(` / `producer.Produce(` matches across all 69 changed Go files |
| SUB-01..SUB-04 | No sub-domain packages |
| EXT-01..EXT-04 | No `requests.GetRequest`/`requests.RootUrl` calls; the only HTTP client is the pre-existing `internal/idasrc` MCP client to IDA (not an atlas service) |
| SCAFFOLD-01..08 | No new service; no new atlas-channel Writer/Handler constants registered |
| SEC-01..SEC-04 | Not an auth/token service; no secrets in diff (the `192.168.20.3` IDA-MCP default URL in `cmd/discover_ops.go:43` matches the pre-existing convention in `cmd/root.go:96,154,207,...` on main — not a new pattern and not a secret) |

## Applicable Checks

### tools/packet-audit (developer CLI — support code)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-09 (analog) | Errors handled, not discarded | PASS* | All loaders return wrapped errors (opregistry.go:64,69,72,75,79; seedcsv.go:50,101; evidence/evidence.go:58-68; matrix/load.go:48; matrix/evidence_input.go:14); cmd entry points check every error and exit 3 (cmd/matrix.go:76-248, cmd/registry.go:50-98, cmd/evidence.go:40-76, cmd/discover_ops.go:110-269). *Two ignored-error spots noted as non-blocking below. |
| DOM-12 | No `os.Getenv()` in new code | PASS | Zero matches in the six new internal packages and four new cmd files; the single `os.Getenv` in cmd/root.go:125 (`PACKET_AUDIT_GENERATED_AT`) pre-exists on main (this task's root.go diff only adds subcommand dispatch lines) |
| DOM-20 | Table-driven tests | PASS w/ WARN | New tests are table-driven where multi-case: opregistry_test.go (3 `t.Run` tables), cmd/matrix tests, and the new byte-fixture tests (sit_result_test.go uses `cases := []struct{...}` + `t.Run`). WARN: internal/matrix/grade_test.go:24-263 is one-function-per-scenario (16 `TestGrade*` functions) instead of a table — testing-guide.md line 18 says "Prefer table-driven tests" (preference, not MUST → non-blocking) |
| DOM-21 | No duplication of atlas-constants types | PASS | New types (`opregistry.Direction`, `Applicability`, `matrix.State`, `RowKind`, `Tiers`, `discover.Discovered`) are packet-audit domain concepts with no equivalent in `libs/atlas-constants/` (checked package index: asset, channel, character, field, inventory, invite, item, job, map, monster, point, skill, stat, world — no packet-direction/opcode/grading types). The new byte test `monster/clientbound/stat_test.go:9` correctly imports `atlas-constants/monster` (`monster.TemporaryStatTypeSpeed`, `monster.SkillTypeSlow`) instead of redeclaring |
| Dead code | anti-patterns.md "Leaving dead code after refactoring" | **FAIL** | `tools/packet-audit/internal/atlaspacket/registry.go:644-648`: `operationReturnLiteral` has **zero callers** anywhere in the module — including tests, despite its own comment claiming "backward-compatible wrapper used by tests" (`grep -rn "operationReturnLiteral(" tools/packet-audit` finds only the definition). Delete it, or fix the tests to actually use it |
| No panics in lib paths | Code-quality | PASS | Zero `panic(`/`log.Fatal`/`os.Exit` in the six new internal packages (grep verified); cmd files return exit codes through `Run` instead of exiting |
| Deterministic output | Code-quality (context.md D2) | PASS | STATUS.md/status.json generation sorts at every map boundary: `opregistry.AllOps` sorts (opregistry.go:178-183), sub-struct keys sorted (matrix/build.go:78-82), `lookupAnyVersion` sorts version keys (build.go:104-108), duplicate-claim IDA names sorted (build.go:141-145), export hashes sorted in render (render.go:29-32), evidence keys sorted in cmd (matrix.go:140-149), seed/apply entries sorted (registry.go:71-79, discover_ops.go:290-298). The unsorted map iterations that remain (`transitiveRecurseTypes` matrix.go:377-391, `Inputs.Reports` in build.go:64) feed order-insensitive consumers (set membership / per-packet-keyed maps with unique WriterName keys) |
| Test helpers | CLAUDE.md "no `*_testhelpers.go`" | PASS w/ WARN | No `*_testhelpers.go` files in the diff (grep verified). WARN: two test-only helpers live in production source instead of the test files — `marker.scanString` (marker.go:111-113, sole caller marker_test.go:37) and `seedcsv.LoadFromString` (seedcsv.go:41-43, sole caller seedcsv_test.go:88) |

### libs/atlas-packet (test-only changes)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Test-only claim | No production code changed | PASS | All 31 changed files match `_test`; +180/-0 lines |
| Builder-pattern test conventions | CLAUDE.md Test Helper Pattern | PASS | The 3 new byte-fixture tests construct inputs via existing public constructors only: `NewCharacterSit(17)`/`NewCharacterCancelSit()` (sit_result_test.go), `model.NewMonsterTemporaryStat()` + `AddStat` + `NewMonsterStatSet(5001, stat)` (stat_test.go), existing `pt.Variants`/`pt.CreateContext`/`pt.Encode` harness. No new test-only constructors introduced |
| DOM-20 | Table-driven | PASS | sit_result_test.go: `cases := []struct{...}` with `t.Run(v.Name+"/"+tc.name, ...)` across all tenant variants; stat_test.go is a single-fixture byte oracle (one case — table not applicable); disable_test.go change is comment-only |
| Marker comments | Well-formed `packet-audit:verify` | PASS | All added markers carry `packet= version= ida=` fields (e.g. buddy/clientbound/invite_test.go +4 markers, guild/clientbound/operation_test.go +2, cash/serverbound/shop_operation_gift_test.go +4); the tool's own `matrix --check` validates marker↔report address linkage, and the module's tests pass |
| Verification-over-memory | CLAUDE.md | PASS | Each new byte fixture cites its IDA export entry + address in the doc comment (sit_result_test.go: `CUserLocal::OnSitResult @ 0x959797`; stat_test.go: `CMob::OnStatSet @ 0x66c301`; disable_test.go: `CUserLocal::OnSetStandAloneMode @ 0x95ffa2`) |

## Summary

### Blocking (must fix)

- **Dead code:** `operationReturnLiteral` at `tools/packet-audit/internal/atlaspacket/registry.go:644-648` — never called by anything (its comment claims test usage that does not exist). anti-patterns.md: "Leaving dead code after refactoring". Delete the function (and its comment), or make a test exercise it.

### Non-Blocking (should fix)

- `tools/packet-audit/internal/opregistry/opregistry.go:143-153` — the doc comment for `AllOps` ("AllOps returns the union of (op, direction)…") is physically attached to `NewVersionFile`, so godoc renders the wrong documentation for both functions. Move the `AllOps` comment down to line 155.
- `tools/packet-audit/internal/marker/marker.go:44` — `rel, _ := filepath.Rel(root, path)` discards the error; on failure `Marker.File` is silently empty. Handle or document.
- `tools/packet-audit/cmd/matrix.go:168-172` — `atlaspacket.NewTypeRegistry` errors are swallowed (`regErr` checked only for nil); a genuine parse failure silently disables opaque-type tier-1 expansion. The comment only justifies the missing-dir case — at minimum log the error to stderr as a warning.
- `tools/packet-audit/internal/marker/marker.go:111` (`scanString`) and `tools/packet-audit/internal/seedcsv/seedcsv.go:41` (`LoadFromString`) — test-only helpers in production source; inline into the respective `_test.go` files.
- `tools/packet-audit/internal/opregistry/opregistry.go:92` — exported `VersionFile.ByFName` has no production caller (only opregistry_test.go exercises it). If it is the seam for the deferred Task 5.4 serverbound verification pass (discover_ops.go:85-90), say so in its doc comment; otherwise remove.
- `tools/packet-audit/internal/matrix/grade_test.go:24-263` — sixteen single-scenario `TestGrade*` functions; testing-guide.md prefers table-driven. Consider collapsing into a `gradeArgs`→`Cell` table.
- `tools/packet-audit/cmd/discover_ops.go:123` — `fmt.Sprintf("FAILED: not found in IDA")` has no format directives; use the string literal directly.

### Verdict

**NEEDS-WORK** — build, tests, vet, and race are clean in both changed modules; every applicable guideline check passes except the one dead-code FAIL above. All service-oriented checklist items (DOM REST/GORM/Kafka, SUB, EXT, SCAFFOLD, SEC) are N/A for this CLI-tool + test-only change and are recorded as such, not failed.

---

## Re-audit — 2026-06-13 (HEAD 2f0073af)

Second pass over the same branch after additional commits. Build (`go build ./...`) and tests (`go test ./... -count=1`, 13 packages) PASS; `go vet ./tools/packet-audit/...` clean.

**Scope correction.** The two-dot `origin/main..HEAD` diff surfaces Go changes in `services/atlas-pets`, `services/atlas-query-aggregator`, and `services/atlas-parties`. These are NOT on this branch — the merge-base is `3d5e40626` but `origin/main` (afb6224e) is ahead of it, so those edits belong to main, not task-085. The three-dot diff `origin/main...HEAD -- services/` shows only `atlas-configurations/seed-data/templates/*.json` (data, not Go). The reviewable Go surface is exactly `tools/packet-audit/**` + `libs/atlas-packet/**` test files, as the brief stated. The `libs/atlas-packet` test-only claim re-verified with three-dot diff: zero non-`_test.go` files changed.

### Resolved since first pass

- **Blocking dead-code FAIL — RESOLVED.** `operationReturnLiteral` no longer exists. The only related symbol is `operationReturnLiteralWithIdent` (registry.go:620), which has a live production caller at registry.go:214. The blocking item is cleared.
- **`AllOps` comment placement — RESOLVED.** Comment now sits directly above the function (opregistry.go:153 → func at :155).
- **`filepath.Rel` discarded error — RESOLVED.** marker.go:44-47 now captures the error and falls back to the absolute path with a comment.
- **`matrix.go` swallowed `regErr` — RESOLVED.** matrix.go:165-172 now logs a stderr warning when `NewTypeRegistry` fails ("opaque-type tier expansion skipped").
- **`discover_ops.go` no-directive `Sprintf` — RESOLVED.** No longer present.

### Still open (non-blocking)

- `tools/packet-audit/internal/opregistry/opregistry.go:92` — `VersionFile.ByFName` still has no production caller (only `opregistry_test.go`). Either the seam for deferred serverbound work (document why) or dead — `verify-serverbound` resolves via the IDA send-site path, not `ByFName`.
- `tools/packet-audit/internal/marker/marker.go:114` (`scanString`) and `tools/packet-audit/internal/seedcsv/seedcsv.go:41` (`LoadFromString`) — still test-only helpers living in production source; each has a single caller in the sibling `_test.go`. Inline them.
- `tools/packet-audit/internal/matrix/grade_test.go` — still sixteen single-scenario `TestGrade*` functions rather than one table (testing-guide.md preference, not a MUST).

### New files reviewed (not itemized in first pass)

- `cmd/verify_serverbound.go` — clean: injectable `idasrc.MCPClient` for tests, all errors checked, exit-3 on failure, pure core (`verifyServerboundRun`) split from flag parsing. Hardcoded `192.168.20.3:13337` is an overridable flag default matching pre-existing convention.
- `internal/discover/sendparse.go` — clean and deterministic: opcodes sorted + deduped, `strconv.ParseInt` errors handled, regex documented, integer-literal-only (non-literal args correctly skipped as unverifiable).

### Config glance (data, not Go)

- `docs/packets/registry/gms_v84.yaml` (new, generated) carries an explicit UNVERIFIED banner for clientbound opcodes ≥0x3F, consistent with the known v84 opcode-table shift. No false confidence asserted.
- `template_{gms_87,jms_185}_1.json` opcode wiring is config data validated by the tool's own `matrix --check` gate, which the passing test suite exercises.

### Re-audit verdict

**NEEDS-WORK → effectively PASS-with-nits.** The sole blocking item from the first pass is resolved and four of the non-blocking items were fixed. What remains is three minor non-blocking cleanups (one dead exported method, two test-helpers-in-prod-source, one table-test preference) — none block merge. Build, tests, and vet are clean.
