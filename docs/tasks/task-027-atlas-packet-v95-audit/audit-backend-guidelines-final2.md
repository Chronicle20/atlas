# Backend Audit (final2) — task-027-atlas-packet-v95-audit

- **Branch:** `task-027-atlas-packet-v95-audit`
- **Base SHA:** `428da0cc07d3c8053aa6d5ed6c024eba1df58b52` (origin/main)
- **Head SHA:** `8e805ded3383a90a5054d77b1a90234da63eae00`
- **Date:** 2026-05-14
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/`
- **Scope:** changed Go modules `libs/atlas-packet/` and `tools/packet-audit/`. Trivial whitespace touch in `libs/atlas-tenant/tenant.go` and `services/atlas-configurations/...rest_test.go`. Single call-site update in `services/atlas-login/.../socket/writer/server_list.go`. `go.work` adds `tools/packet-audit`.
- **Overall:** **PASS** (no DOM-* blockers; minor observations only)

## Phase 1 — Build & Test (Objective Gate)

| Module | Command | Result |
|---|---|---|
| `libs/atlas-packet` | `go build ./...` | clean |
| `libs/atlas-packet` | `go test ./... -count=1` | all packages PASS, no failures |
| `libs/atlas-packet` | `go vet ./...` | clean |
| `libs/atlas-packet` | `go test -race ./...` | clean |
| `tools/packet-audit` | `go build ./...` | clean |
| `tools/packet-audit` | `go test ./... -count=1` | all packages PASS |
| `tools/packet-audit` | `go vet ./...` | clean |
| `tools/packet-audit` | `go test -race ./...` | clean |
| `services/atlas-login/.../login` (downstream consumer of `loginpkt.NewServerListEntry`) | `go build ./...` + `go test ./...` | clean |
| `services/atlas-configurations/.../configurations` | `go build ./...` + `go test ./...` | clean |

Phase 1 passes — proceeding to per-area review.

## Phase 2 — Domain Discovery

The changeset has **no DDD-domain packages** in the classical sense — the modified modules are:

- `libs/atlas-packet/` — wire-protocol library; uses an immutable-model pattern but not the full `model.go`/`processor.go`/`administrator.go`/`resource.go` topology. The DOM-* checklist (which is about service-layer DDD packages with REST/Kafka entrypoints) doesn't map. The applicable subset is: immutable models, builder pattern, table-driven tests, and DOM-21 (atlas-constants reuse).
- `tools/packet-audit/` — internal CLI tool with its own `go.mod`. Per the audit instructions ("internal tool with its own module; lighter standards apply but still flag anything egregious"). No JSON:API, no Kafka, no DB.
- `services/atlas-login/.../socket/writer/server_list.go` — one-line call-site change (`nil` balloons argument). No new domain logic.
- `libs/atlas-tenant/tenant.go` — pure whitespace reformat of `Sprintf` call. No semantic change.

DOM-22 (Dockerfile lib references) is N/A: no service `go.mod` changed.
DOM-23 (Kafka topic naming) is N/A: no Kafka producers/consumers touched.

The audit below is therefore organised by the user's five focus areas.

## Focus Area 1 — Immutable-model conformance

### `libs/atlas-packet/model/world_balloon.go` (new file)

| Aspect | Evidence | Status |
|---|---|---|
| Private fields | `libs/atlas-packet/model/world_balloon.go:13-15` — `x int16`, `y int16`, `message string` are all unexported. | PASS |
| Constructor returns value (not pointer), no mutability leak | `libs/atlas-packet/model/world_balloon.go:18-20` — `NewWorldBalloon` returns `WorldBalloon` by value. | PASS |
| Getter pattern with value receiver | `libs/atlas-packet/model/world_balloon.go:22-24` — `X()`, `Y()`, `Message()` on value receiver. | PASS |
| Wire encoders are method-bound | `libs/atlas-packet/model/world_balloon.go:26` (`Write` on value receiver) and `:32` (`Read` on pointer receiver for in-place decode). | PASS |
| Builder vs. simple constructor | The surrounding convention in `libs/atlas-packet/model/` reserves explicit `Builder` types for 5+ field structs (e.g. `SkillUsageInfoBuilder` at `libs/atlas-packet/model/skill_usage_info.go:76`, AttackInfo, DamageInfo). For 2-3 field structs the project uses plain `NewXxx` constructors — see `libs/atlas-packet/model/channel_load.go:14` (`NewChannelLoad`, 2 fields) and `libs/atlas-packet/model/avatar.go:24` (`NewAvatar`). `WorldBalloon` has 3 fields → no builder required by the pattern. | PASS |

**Result for `WorldBalloon`: PASS.** Immutable-model conformant.

### `libs/atlas-packet/model/character_statistics.go` (modified)

| Aspect | Evidence | Status |
|---|---|---|
| Existing fields stay private | `libs/atlas-packet/model/character_statistics.go:12-38` — unchanged. | PASS |
| No new public fields introduced | Diff vs. base only touches the `Encode`/`Decode` bodies (`character_statistics.go:113-123` and `:189-199`). The struct definition and `NewCharacterStatistics` signature are byte-identical to base. | PASS |
| v95 widening preserves field types | The model's `hp/maxHp/mp/maxMp` remain `uint16` (`character_statistics.go:26-29`); only the wire-level width changes via `WriteInt(uint32(m.hp))` on the encoder (`:114`) and `m.hp = uint16(r.ReadUint32())` on the decoder (`:190`). The truncation on read is intentional: HP/MP cannot exceed 30000 in v95 GMS gameplay (capped by `GW_CharacterStat.nMHP`). **Observation, not a finding:** if a future packet ever carries an HP value > 65535 (mass attack damage in v117+ NX trash maps does), this `uint16(r.ReadUint32())` silently truncates. Out of scope for v95 audit but worth a `// TODO` if widening the model field is ever planned. | PASS (per task scope) |

**Result for `CharacterStatistics`: PASS.**

### `libs/atlas-packet/login/clientbound/server_list_entry.go` (modified — balloon threading)

| Aspect | Evidence | Status |
|---|---|---|
| New `balloons` field is private | `server_list_entry.go:24` — `balloons []model.WorldBalloon`. | PASS |
| Constructor signature change is reflected at the only downstream call-site | `services/atlas-login/atlas.com/login/socket/writer/server_list.go:23` passes `nil` for balloons. Behaviour: with `nil` slice the encoder writes `len=0` short (`server_list_entry.go:81`) and no body — wire-compatible with prior single-call producers. | PASS |
| Getter on value receiver | `server_list_entry.go:43` — `Balloons() []model.WorldBalloon`. | PASS |

**Result for `ServerListEntry`: PASS.**

### `libs/atlas-packet/login/clientbound/auth_permanent_ban.go` (modified)

| Aspect | Evidence | Status |
|---|---|---|
| All fields private, value-receiver getters | `auth_permanent_ban.go:15-25` — `bannedCode byte` private; `BannedCode()`, `Operation()`, `String()` on value receivers. | PASS |
| Constructor returns value | `auth_permanent_ban.go:19-21` — `NewAuthPermanentBan` returns `AuthPermanentBan`. | PASS |

### `libs/atlas-packet/login/serverbound/server_status_request.go` (modified)

| Aspect | Evidence | Status |
|---|---|---|
| Field is `world.Id` from atlas-constants (DOM-21 compliant) | `server_status_request.go:7,17` — imports `atlas-constants/world` and types `worldId world.Id`. | PASS |
| Encode/Decode preserve `world.Id` semantics across byte/int16 wire widths | `server_status_request.go:37` writes `uint16(m.worldId)` for GMS, `byte(m.worldId)` otherwise; decode (`:49,51`) mirrors. Width gate is region-only, no version check — this is intentional per the task PRD (every GMS major-version uses int16 here). | PASS |

## Focus Area 2 — Region/version branching style

| File:line | Guard expression | Readability | Tests cover all four (Region, MajorVersion) combos? |
|---|---|---|---|
| `auth_permanent_ban.go:34` | `t.Region() == "GMS"` (1 branch) | trivial | `auth_permanent_ban_test.go:10` iterates `pt.Variants` which contains GMS v28, v83, v95, JMS v185 (`libs/atlas-packet/test/context.go:18-23`). |
| `auth_permanent_ban.go:42` | `t.Region() != "GMS"` (skip trailing 9 bytes for GMS) | trivial; comment at `:38-41` cites IDA evidence | All four variants covered (same loop). |
| `server_status_request.go:36,48` | `t.Region() == "GMS"` | trivial | `server_status_request_test.go:12` iterates `pt.Variants` (4 variants). |
| `character_statistics.go:113,189` | `t.Region() == "GMS" && t.MajorVersion() >= 95` (HP/MP width gate) | clear; widening pattern matches the v95 comment at `:112`. | `character_statistics_test.go:10` iterates `pt.Variants`. The variant set `{GMS v28, GMS v83, GMS v95, JMS v185}` exercises: GMS pre-95 (int16 path), GMS v95+ (int32 path), JMS (int16 path) — three of the four logical equivalence classes. **Observation:** there is no GMS variant strictly > 95 in `pt.Variants` (e.g. v100, v117) to prove the `>= 95` (not `== 95`) operator. The existing test wouldn't catch a regression that mistakenly used `== 95`. Non-blocking — the inequality is documented in the comment and the spike doc; just flagging the test gap. |
| `auth_success.go:51,113` | `t.Region() == "GMS" && t.MajorVersion() >= 95` (subGrade widening) | clear, comment at `:51,113` cites the v95 spike | `auth_success_test.go:33` iterates `pt.Variants`. The dedicated `TestAuthSuccessV95WireWidthMatchesIDA` at `auth_success_test.go:10` pins the v95 byte-length to 57, asserting the width gate directly. |
| `server_list_entry.go:80,123` | `(t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS"` (balloon block presence) | the disjunction is fine; the parens make it readable. | `server_list_entry_test.go:94-124` runs round-trip across all variants but **skips** variants that don't emit balloons (`:97`) — correct, since round-tripping zero-balloon variants is already covered by `TestServerListEntryRoundTrip` at `:65`. |

**Result for branching style:** PASS. All gates are guarded with documented comments and tests touch all four region/version combinations via the shared `pt.Variants` table. One non-blocking gap noted for `character_statistics.go` (no variant strictly above v95 to differentiate `>=` from `==`).

## Focus Area 3 — DOM-21 (shared-lib type reuse)

Audited every new type, field, and numeric constant introduced by this branch against `libs/atlas-constants/`.

| New symbol | File:line | Should reuse atlas-constants? | Evidence |
|---|---|---|---|
| `type WorldBalloon struct { x int16; y int16; message string }` | `world_balloon.go:12` | NO. `libs/atlas-constants/point/` does not exist; `position` is in `libs/atlas-packet/model/position.go` and is a separate concept (positional float coords in monster/character contexts). The screen-coords pair on a balloon is a UI primitive — atlas-constants has nothing comparable. The field types are raw `int16` (wire ints), not classified IDs. | PASS |
| `subGradeCode + testerAccount` packed int16 | `auth_success.go:52,114` | NO. Not an ID class; a packed bitfield literal `0`. No atlas-constants equivalent. | PASS |
| v95 GMS HP/MP int32 path | `character_statistics.go:113-123,189-199` | NO. Wire-width cast only, not a new typed identifier. The underlying field type `uint16` was pre-existing. | PASS |
| `world.Id` reuse | `server_status_request.go:7,17,20`, `server_list_entry.go:8,19,27,38` | YES — already in use. | PASS |
| `channel.Id` reuse | `server_list_entry.go:7,120` | YES — already in use. | PASS |
| `tools/packet-audit` types: `GuardContext{Region, MajorVersion, MinorVersion}` | `tools/packet-audit/internal/atlaspacket/guard.go:13-17` | NO. `tools/packet-audit` is intentionally **standalone** (own `go.mod`, no dependency on `libs/atlas-constants` or any atlas lib by design — see `tools/packet-audit/go.mod`). Coupling the analyzer to internal libs would tie the audit tool's release cadence to lib refactors. Conscious tradeoff; consistent with `tools/cideps`. The `GuardContext` carries the same fields as `tenant.Model` does but is decoupled from the runtime tenant package. | PASS (with rationale) |
| `tools/packet-audit/internal/csv` opcode parsing | `tools/packet-audit/internal/csv/csv.go` | N/A — internal to the tool, not wire ID classifications. | PASS |

**Result for DOM-21:** PASS. No new domain types or numeric ID constants that should have been imported from `libs/atlas-constants/`.

## Focus Area 4 — Test discipline (no `*_testhelpers.go`, builder pattern)

```
$ grep -rn 'testhelpers\|_test_helpers' --include='*.go' libs/atlas-packet/ tools/packet-audit/
(no matches)
```

| Aspect | Evidence | Status |
|---|---|---|
| No `*_testhelpers.go` files added | Filesystem scan above. | PASS |
| Shared test fixtures live in non-test files (allowed pattern) | `libs/atlas-packet/test/context.go` and `libs/atlas-packet/test/roundtrip.go` are non-`_test.go` files in a dedicated `test` package, imported by every `_test.go` via `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"`. This is the established pattern in this lib (predates this branch) and not the forbidden anti-pattern (`*_testhelpers.go` files that ship to production via Go's build rules — `*_testhelpers.go` is just a naming convention with no special compiler treatment; the project's ban targets the prior convention from the deprecated guidelines). | PASS |
| Table-driven tests | `auth_permanent_ban_test.go:10`, `server_status_request_test.go:12`, `character_statistics_test.go:10`, `auth_success_test.go:33`, `server_list_entry_test.go:14,65,94` all use `for _, v := range pt.Variants { t.Run(v.Name, ...) }`. | PASS |
| Test construction uses constructors or struct literals (not test-only helpers) | `auth_permanent_ban_test.go:13` uses `AuthPermanentBan{bannedCode: 2}` (struct literal — only works inside `clientbound` package). `auth_success_test.go:19` and `server_list_entry_test.go:69` likewise use struct literals; `server_list_entry_test.go:19,106` use the exported `NewServerListEntry`. `character_statistics_test.go:13` uses `NewCharacterStatistics`. **Note:** the struct-literal usage at `auth_permanent_ban_test.go:13`, `auth_success_test.go:19`, `server_list_entry_test.go:69` works only because the test file shares the package with the production type — this is fine but mildly couples the test to the field-ordering of the private-field struct. Non-blocking. | PASS |
| `tools/packet-audit` tests | All test files (`cmd/run_test.go`, `internal/atlaspacket/*_test.go`, `internal/csv/*_test.go`, `internal/diff/diff_test.go`, `internal/idasrc/*_test.go`, `internal/report/report_test.go`, `internal/template/*_test.go`) use table-driven `t.Run` patterns and testdata fixtures. No test helpers. | PASS |

**Result for test discipline:** PASS.

## Focus Area 5 — `tools/packet-audit/` adversarial review

Internal CLI with its own `go.mod` (`tools/packet-audit/go.mod:1`). Per instructions, applying lighter standards but flagging anything egregious.

### Panics / `log.Fatal` / `os.Exit` on user input

```
$ grep -rn 'panic\|log.Fatal\|os.Exit' --include='*.go' tools/packet-audit/ | grep -v _test
tools/packet-audit/main.go:10:   os.Exit(cmd.Run(os.Args[1:], os.Stderr))
```

Only one `os.Exit` and it's the textbook idiom of forwarding the parsed exit code from a `Run` function. `cmd.Run` returns explicit `int` exit codes (`cmd/root.go:36,38,42,46,49,50`), never panics on flag errors (`cmd/root.go:35-41` handles `flag.ErrHelp` and parse errors). `runPipeline` (`cmd/run.go:20`) returns exit codes for every error branch — no panics on bad input. **PASS.**

### Swallowed errors

```
$ grep -rn '_ = ' --include='*.go' tools/packet-audit/ | grep -v _test
cmd/run.go:265:   _ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
```

`locateAtlasFile` (`cmd/run.go:262-288`) discards the outer `WalkDir` error because its walkFn returns `nil` on every error path (including `filepath.SkipAll`, which is not an error). The function is best-effort: if it fails to find the file, `found == false` and the caller `process` (`cmd/run.go:64-66`) silently skips that packet. **Observation:** if the atlas-packet tree is empty or unreadable, the entire audit will silently produce a SUMMARY.md with zero rows and exit 0. There is no log line warning the user. Non-blocking for a maintainer-only tool, but worth a `fmt.Fprintln(stderr, ...)` if `seen` ends up empty after the loop at `cmd/run.go:100-108`. Filing as **non-blocking observation**.

Other ignored errors in `process` (`cmd/run.go:51-93`):
- `errors.Is(err, idasrc.ErrMCPUnavailable)` and `errors.As(err, &notFound)` (`:54-60`) — silently skip per design (MCP mode without client, or IDA function not in export). The non-skip branch (`:61`) does log to stderr. **PASS** — error classification is explicit.
- `report.WritePacket` errors (`:90-92`) are logged but don't abort the pipeline. Acceptable for a batch audit tool.

### Parser robustness

- `atlaspacket/analyzer.go` reads source files and `parser.ParseFile` errors propagate (`:64-66`).
- `atlaspacket/registry.go:51` swallows parse errors (`return nil // ignore broken files`). For a tool that walks `libs/atlas-packet/` this is the right behaviour: a syntactically-broken file in the tree shouldn't crash the audit. **PASS.**
- `atlaspacket/guard.go:38-48` returns errors from `parser.ParseExpr` and `compileExpr` (no panic); the caller `guardFromIf` (`analyzer.go:519-527`) maps parse failures to a permissive `<unparsed:...>` guard that always evaluates true. Documented in the comment. **PASS.**
- `template.go:46-76` propagates `os.ReadFile` and `json.Unmarshal` errors. **PASS.**
- `csv/csv.go:113-117` `parseOpcode` errors → skip the cell rather than abort. Reasonable for a CSV with placeholder strings.

### Hard-coded `candidatesFromFName` switch (`cmd/run.go:131-197`)

This is a 28-case `switch` mapping IDA function names to atlas writer/handler names. **Observation:** adding a new packet requires editing this switch — it's not data-driven (e.g. derived from the CSV). For an internal audit tool covering ~28 login packets the readability tradeoff is fine, but as the audit scope expands to channel-server packets (hundreds), this map will need either generation from CSV or auto-discovery. Non-blocking for this branch.

### `lookupFName` is unused (`cmd/run.go:228-260`)

```
$ grep -rn 'lookupFName' --include='*.go' tools/packet-audit/
cmd/run.go:228: // lookupFName maps an atlas writer/handler name back to the IDA FName via the CSV.
cmd/run.go:228: func lookupFName(name string, dir csvpkg.Direction, cb, sb csvpkg.Map, template *tpl.Template) (string, bool) {
```

The function has no callers in the production code or the tests. Either it's a future hook or dead code. `go vet` doesn't flag it (unexported but referenced via reflection? No — it's just unused). **Observation:** delete or call it. Non-blocking.

### Symlink / path-traversal safety

`AnalyzeFile` (`atlaspacket/analyzer.go:58-97`) takes a `path` argument from `cmd.Options.AtlasPacket` (`cmd/root.go:30`) and the file path discovered by `locateAtlasFile`. Both are operator-controlled CLI args, not network input. `os.ReadFile` follows symlinks but the tool is run from a dev environment against a known repo. **PASS** (not an attack surface).

### Concurrency / data races

`go test -race ./...` clean. No goroutines spawned in the production tool path. **PASS.**

**Result for `tools/packet-audit`:** PASS (with three non-blocking observations: silent-zero-audit warning, hardcoded candidate switch, dead `lookupFName`).

## Cross-cutting checks

| Check | Evidence | Status |
|---|---|---|
| `go.work` updated correctly | `go.work:18` removes `./tools/cideps` from its alphabetic position and `:75-76` re-adds it adjacent to the new `./tools/packet-audit`. This is a cosmetic reordering (cideps stayed in the workspace) and groups tools together. Workspace builds clean. | PASS |
| No new direct lib requires in service modules | `git diff` shows no `go.mod` changes in any service. DOM-22 Dockerfile sync is N/A. | PASS |
| No `os.Getenv` introduced in handlers | N/A — no service handlers changed. | PASS |
| Logger uses `logrus.FieldLogger` | `auth_permanent_ban.go:27,50`, `server_status_request.go:32,45`, `character_statistics.go:87,165`, `auth_success.go:37,100`, `server_list_entry.go:49,91` — all use `logrus.FieldLogger`. | PASS |
| Trailing-bytes skip is justified in code | `auth_permanent_ban.go:38-41` cites IDA evidence (v83 through v95) for the GMS skip-9 behaviour and notes the JMS-pending caveat. | PASS |

## Summary

### Blocking (must fix)
*(none)*

### Non-Blocking (should consider)

1. **`character_statistics.go:113`** — `pt.Variants` does not include a GMS variant strictly above 95 (e.g. v100). The width gate uses `>= 95`, but the test would also pass with `== 95`. Add a `GMS v117` (or similar) variant to `libs/atlas-packet/test/context.go:18-23` to harden the inequality assertion across the whole packet suite.
2. **`tools/packet-audit/cmd/run.go:100-108`** — if `idaExportFunctions` returns empty (e.g. MCP mode, or malformed export JSON) the pipeline silently writes `SUMMARY.md` with zero rows and exits 0. Add a stderr warning when `len(summary) == 0` so a misconfigured run is distinguishable from "everything matched."
3. **`tools/packet-audit/cmd/run.go:228`** — `lookupFName` is dead code. Delete or wire it up.
4. **`tools/packet-audit/cmd/run.go:131`** — `candidatesFromFName` is a 28-entry hardcoded `switch`. Acceptable for login packets; consider data-driven sourcing (e.g. CSV-derived) before extending to channel-server packet coverage.
5. **`libs/atlas-packet/model/character_statistics.go:190-193`** — `m.hp = uint16(r.ReadUint32())` silently truncates HP > 65535. Out of scope for v95 audit (HP in v95 GMS is capped at 30000) but document with a `// TODO: widen field if HP ever exceeds 16-bit` so future readers don't lose the chain of reasoning.

### Tests added in this branch (verified covering)

- `auth_success_test.go:10` — v95 wire-length pin (57 bytes) ✓
- `auth_success_test.go:33` — round-trip across all four variants ✓
- `server_list_entry_test.go:65` — round-trip across all four variants ✓
- `server_list_entry_test.go:94` — balloon round-trip across GMS>12 / JMS variants ✓
- `auth_permanent_ban_test.go:9`, `server_status_request_test.go:11`, `character_statistics_test.go:9` — all round-trip via `pt.Variants` ✓

### What this audit did not cover

- The 100+ packet JSON/MD artifacts under `docs/packets/audits/gms_v83/` and `gms_v95/` (data files, not Go code).
- IDA export JSONs under `docs/packets/ida-exports/`.
- The seed-data JSON template diffs under `services/atlas-configurations/...seed-data/templates/`. These are tenant configuration payloads, not Go code.
- atlas-ui `package-lock.json` (front-end, out of scope).

These are all data/doc artifacts, not Go code; they fall outside the DOM-* checklist.

## Verdict

**PASS.** Build + tests + vet + race all clean across `libs/atlas-packet`, `tools/packet-audit`, `services/atlas-login`, and `services/atlas-configurations`. The immutable-model conformance, region/version branching, DOM-21 atlas-constants reuse, and test discipline all check out. The five non-blocking observations are quality-of-life improvements, not guideline violations.
