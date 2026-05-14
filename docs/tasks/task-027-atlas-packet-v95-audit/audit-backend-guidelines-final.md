# Backend Audit — task-027-atlas-packet-v95-audit (Final Pre-Merge)

- **Worktree:** `.worktrees/task-027-atlas-packet-v95-audit`
- **Branch:** `task-027-atlas-packet-v95-audit`
- **Base SHA:** `c2b7e5eaec63cee7fe689f92e694d7ad9362a1f8`
- **HEAD SHA:** `4a12a0376cc5e571c4b74fa48c6def1ff227f575`
- **Prior audit point:** `0e937b165`
- **Guidelines source:** `.claude/skills/backend-dev-guidelines/`
- **Date:** 2026-05-13
- **Overall:** NEEDS-WORK (1 non-blocking finding)

## Phase 1 — Build & Test

| Module | `go build` | `go test ./... -count=1` | `go vet` | `go test -race` |
|---|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS (all `ok`) | PASS | PASS |
| `libs/atlas-tenant` | PASS | PASS (`ok`) | PASS | PASS |
| `services/atlas-login/atlas.com/login` | PASS | PASS (`maps/location`, `world`; rest no test files) | one PRE-EXISTING warning in `socket/init.go:39` (not touched by this task) | n/a (no changes) |
| `services/atlas-configurations/atlas.com/configurations` | PASS | PASS | PASS | n/a (no changes) |
| `tools/packet-audit` | PASS | PASS (all packages `ok`) | PASS | PASS |

Build/test gate: PASS.

## Phase 2 — Scope Classification

This branch is **not** a domain-service feature; it is:

1. Wire-encoding bug fixes in `libs/atlas-packet/` (an encoder library, not a service).
2. A maintainer CLI under `tools/packet-audit/` (no Kafka/REST/tenant runtime — task brief explicitly exempts).
3. A data-only `template_gms_95_1.json` opcode-table fix in `seed-data/`.
4. A one-line caller update in `services/atlas-login/.../socket/writer/server_list.go` to absorb the new `NewServerListEntry` signature.
5. A `clientVariant` YAGNI revert in `libs/atlas-tenant/` + `services/atlas-configurations/templates/rest.go` (and tests).

The DOM-/SUB-/SEC-* mechanical checklists target services with `model.go`/`processor.go`/`resource.go`/`administrator.go` layering. No such files were added or modified by this branch. The applicable guideline-level checks are:

- Immutable model + builder pattern (`patterns-functional.md`)
- Multi-tenancy via context (`patterns-multitenancy-context.md`)
- Dead-code cleanup after refactor (`anti-patterns.md` line 35, `ai-guidance.md` lines 173–175)
- Functional purity / no manual JSON envelope decoding

These are evaluated below.

## Phase 3 — Per-Area Findings

### Area A — `libs/atlas-packet/` wire fixes

| Check | Result | Evidence |
|---|---|---|
| `CharacterStatistics` HP/MP fields gated on `GMS && >=95` (int32) else int16 | PASS | `libs/atlas-packet/model/character_statistics.go:113-123` encode; `:189-199` decode |
| `AuthPermanentBan` trailing reason+timestamp omitted on `GMS && >=95` | PASS | `libs/atlas-packet/login/clientbound/auth_permanent_ban.go:41-44` encode; `:59-62` decode |
| `ServerStatusRequest` worldId widens to int16 on `GMS && >=95` | PASS | `libs/atlas-packet/login/serverbound/server_status_request.go:36-40` encode; `:48-52` decode |
| `Request` (LoginHandle) trailing byte gated on `GMS && >=95` | PASS | `libs/atlas-packet/login/serverbound/request.go:64-66` encode; `:80-82` decode |
| `ServerListEntry` uses `byte(m.worldId)` for per-channel worldId byte (was incorrectly tied to channel index in prior code) | PASS | `libs/atlas-packet/login/clientbound/server_list_entry.go:75` |
| `WorldBalloon` model has immutable fields + Write/Read | PASS | `libs/atlas-packet/model/world_balloon.go:12-36` (private fields, `New*` constructor, getters, encoder uses `(w *response.Writer)`) |
| `ServerListEntry` balloons threaded through encoder/decoder under correct guard | PASS | `:80-85` encode; `:123-129` decode (`(GMS && >12) || JMS` matches the world-balloon block's pre-v62 absence in stock client) |
| All wire-fix files have round-trip tests across `pt.Variants` (v28, v83, v95, JMS v185) | PASS | `auth_permanent_ban_test.go`, `server_status_request_test.go`, `auth_success_test.go`, `server_list_entry_test.go:14-124`, `model/character_statistics_test.go` (covered by existing tests under model pkg) |
| `pt.Variants` includes v95 | PASS | `libs/atlas-packet/test/context.go:21` |
| All Encode/Decode methods use `tenant.MustFromContext(ctx)` for region/version gating | PASS | All 4 fixed files; e.g. `server_status_request.go:34, :46`; `request.go:56, :72` |

**Sub-finding — context handling.** All region/version conditionals route through `tenant.MustFromContext(ctx)`. No file extracts `MajorVersion` via raw `ctx.Value(...)` or via a global. Consistent with `patterns-multitenancy-context.md`.

**Sub-finding — immutability.** `WorldBalloon` and `CharacterStatistics` use private fields + accessor methods + factory constructors (`NewWorldBalloon`, `NewCharacterStatistics`). The Write/Read pair on `WorldBalloon` cleanly separates encode/decode side. `world_balloon.go:32-36`'s `Read` mutates via pointer receiver, consistent with the rest of the lib's decode convention.

### Area B — `libs/atlas-tenant/` clientVariant revert

| Check | Result | Evidence |
|---|---|---|
| `clientVariant` field removed from `Model` | PASS | `libs/atlas-tenant/tenant.go:10-15` (4 fields only) |
| `ClientVariant()` accessor removed | PASS | grep for `ClientVariant` in `libs/atlas-tenant/`: zero hits |
| `CreateWithVariant` removed | PASS | `libs/atlas-tenant/processor.go` no longer contains it (see git diff) |
| `MarshalJSON`/`UnmarshalJSON` no longer carry `clientVariant` field | PASS | `tenant.go:33-64` |
| `Is()` no longer compares variant | PASS | `tenant.go:66-80` |
| `String()` no longer formats variant | PASS | `tenant.go:82-85` |
| Variant-related tests removed | PASS | `tenant_test.go` retains only `TestSerialization` |
| No leftover variant references in the repo | PASS | `grep -r 'ClientVariant\|clientVariant\|VariantOf\|IsStock\|ClientVariantKey\|CreateWithVariant\|decodeStock\|RequestStock'` across `libs/`, `services/`, `tools/`: zero hits (only design/plan docs mention removed names, which is intended history) |

### Area C — `libs/atlas-packet/version/` — DEAD CODE (only finding)

The `version` package (`version.go` + `version_test.go`) was introduced in commit `fd4eec27a` as scaffolding for the `clientVariant` system. The variant-specific helpers (`IsStock`, `VariantOf`) were stripped in the `7fb32b5c0` revert, but the remaining helpers were left behind:

- `version.GMS`, `version.JMS` constants — zero callers
- `version.Region` type — zero callers
- `version.RegionOf(tenant.Model)` — zero callers
- `version.AtLeast(tenant.Model, n)` — zero callers
- `version.LessThan(tenant.Model, n)` — zero callers
- `version.Between(tenant.Model, lo, hi)` — zero callers

Verification:

```
grep -rn 'version\.AtLeast\|version\.LessThan\|version\.RegionOf\|version\.Between\|version\.GMS\|version\.JMS' libs/ services/ tools/
# → zero hits

grep -rn 'libs/atlas-packet/version\|atlas-packet/version' libs/ services/ tools/
# → zero hits (no package is imported)
```

The wire-fix code uses inline `t.Region() == "GMS" && t.MajorVersion() >= 95` rather than `version.AtLeast(t, 95)`, so the helpers serve no purpose and won't be exercised in the future without a deliberate switch.

**Verdict:** WARN / non-blocking. Violates:

- `.claude/skills/backend-dev-guidelines/resources/anti-patterns.md:35` — "Leaving dead code after refactoring: Unused constants/structs/functions clutter the codebase and cause confusion."
- `.claude/skills/backend-dev-guidelines/resources/ai-guidance.md:173-175` — "After extracting code to a shared library, review every modified service file for symbols that are no longer referenced... delete. Do not leave dead code behind."

**Remediation:** delete `libs/atlas-packet/version/version.go` and `libs/atlas-packet/version/version_test.go`. The package directory itself should also be removed.

### Area D — `tools/packet-audit/` maintainer CLI

Per task brief, DOM/SUB/REST/Kafka/tenant guidelines don't apply. Reviewed for code-quality:

| Check | Result | Evidence |
|---|---|---|
| `TypeRegistry` walks `libs/atlas-packet/` without panicking on broken files | PASS | `tools/packet-audit/internal/atlaspacket/registry.go:50-52` (returns `nil` on parse error, continues walk) |
| Encode wins over Write when both are present on a type | PASS | `registry.go:97-114` |
| `Flatten`/`FlattenWithRegistry` recursion does not infinitely loop on self-referential types | PASS conditionally — `tools/packet-audit/internal/diff/diff.go:127-146` recurses unconditionally on `KindRecurse`/`KindRepeat`; relies on absence of cycles in atlas-packet structs. No depth bound. Currently safe because atlas-packet has no cyclic struct chains, but this is brittle. Filing as informational, not a violation. |
| Range-var → field-type binding has scoped lifecycle | PASS | `analyzer.go:218-233`: bind on entry, delete on exit. `collectSub` clones the map on `:398-401` so nested loops don't pollute the parent. |
| `conjoin` preserves outer guards when text-reparse fails (the previously flagged regression) | PASS | `analyzer.go:537-564`: when `ParseGuard` of the joined text fails, builds an `eval` closure that ANDs the original eval funcs directly. This fixes the prior bug where un-parseable guards (e.g. `len(x) > 0`) silently collapsed to the last guard. |
| `WritePaddedString`/`ReadPaddedString` recognized as `EncodeBuf` | PASS | `analyzer.go:464-470` |
| `WriteKeyValue` decomposes into byte+int32 | PASS | `analyzer.go:301-307` |
| `WriteByteArray(sub.Encode(...)(...))` recognized as `KindRecurse` | PASS | `analyzer.go:487-513` |
| `candidatesFromFName` switch is alphabetized only by group, not strictly — 27 FNames covered | NOTE only | `cmd/run.go:131-197` |
| Test coverage for the new registry behavior | PASS | `tools/packet-audit/internal/atlaspacket/registry_test.go:9-38` exercises `HasType`, `Calls`, `FieldType`. |

### Area E — `services/atlas-configurations/atlas.com/configurations/templates`

| Check | Result | Evidence |
|---|---|---|
| `RestModel` retains JSON:API interface methods (`GetName`, `GetID`, `SetID`) | PASS | `templates/rest.go:24-35` |
| No nested `Data/Type/Attributes` envelope in REST struct | PASS | `templates/rest.go:11-22` (flat) |
| `RestModel.GetName() == "templates"` | PASS | `templates/rest.go:25` |
| `Id` field tagged `json:"-"` | PASS | `templates/rest.go:12` |
| `validation_error.go` is unchanged (no dependencies on removed `validateClientVariant`) | PASS | confirmed by reading file — 35 lines, no variant helper present |
| `rest_test.go` no longer references removed `validateClientVariant` or `ClientVariant` field | PASS | `rest_test.go:1-154` — three variant tests cleanly removed |

### Area F — `services/atlas-login/.../socket/writer/server_list.go`

| Check | Result | Evidence |
|---|---|---|
| Caller updated for new `NewServerListEntry(..., nil)` signature | PASS | `services/atlas-login/atlas.com/login/socket/writer/server_list.go:23` passes `nil` for balloons (login service doesn't have a balloon source yet; this is intentional per task scope) |
| No other call sites of `NewServerListEntry` exist in services/* | PASS | `grep -r NewServerListEntry services/`: only the one site in `server_list.go:23` |

### Area G — Security (SEC-*)

Not applicable — this branch touches no auth/JWT/redirect logic. The packet-level region/version gating is not a security boundary.

## Summary

### Blocking
None.

### Non-Blocking (must fix before merge per dead-code policy)

1. **DEAD-01** — `libs/atlas-packet/version/version.go` + `libs/atlas-packet/version/version_test.go` are completely unreferenced after the `clientVariant` revert. Delete the package. Evidence:
   - `libs/atlas-packet/version/version.go:1-23` — defines `Region`, `GMS`, `JMS`, `RegionOf`, `AtLeast`, `LessThan`, `Between`.
   - Zero callers in `libs/`, `services/`, `tools/` (verified via `grep -rn 'version\.AtLeast\|version\.LessThan\|version\.RegionOf\|version\.Between\|version\.GMS\|version\.JMS'`).
   - Zero imports of the package path (verified via `grep -rn 'libs/atlas-packet/version'`).
   - Violates `anti-patterns.md:35` and `ai-guidance.md:173-175`.

### Informational

- The `Flatten`/`FlattenWithRegistry` recursion in `tools/packet-audit/internal/diff/diff.go:127-146` has no depth bound. Safe today (no cyclic types in `libs/atlas-packet/`), but a hostile struct definition would stack-overflow the audit run. A 32-level bound would be cheap insurance. Not a violation of any current guideline — noted for future hardening.

## Audit Verdict Rationale

Build is clean, tests are clean (including `-race`), `go vet` is clean on every changed module. All wire-fix code passes the guideline checks for immutability, tenant-context gating, and test coverage. The `clientVariant` revert is thorough (zero residual symbol references; tests cleanly removed). The only finding is leftover dead code in `libs/atlas-packet/version/`, which is a clearly-documented anti-pattern in the repo's own guidelines. Once the `version` package is deleted (or its helpers wired into actual callers), this branch is mergeable.

Default verdict per skill rubric: NEEDS-WORK due to one FAIL (DEAD-01 on `anti-patterns.md:35`). Promote to PASS after deletion.
