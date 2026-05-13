# Atlas-Packet v95 Audit & Wire Correctness — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-08

---

## 1. Overview

Atlas's packet library (`libs/atlas-packet`) was written against MapleStory GMS v83 wire layouts and accreted ad-hoc version branches over time. The `template_gms_95_1.json` tenant template exists and the test context already lists `GMS v95` (`libs/atlas-packet/test/context.go:21`), but the encoder/decoder bodies have not been systematically validated against the v95 client's actual wire shape. The spike at `docs/packets/spike-login-v95.md` audited six representative login packets against the v95 IDA decompile (md5 `3c71fd8872d5efbe16183ae8c51f887d`) and surfaced concrete wire-level drift, including one field-width misalignment in `AuthSuccess` that breaks every subsequent field for v95, a structural protocol change in the login handshake, and a latent multi-world bug in `ServerListEntry`.

This task makes atlas-packet wire-correct for GMS v95 across the full library and delivers a reusable audit pipeline so future versions (v87, v92, v111, JMS v185) can be enumerated the same way. The strategy is version-conditional fields inside encoders/decoders, driven by `tenant.MustFromContext(ctx).MajorVersion()` and `.Region()` — the pattern already in use in files like `auth_success.go` and `request.go`.

Two v95 client variants are supported in scope: a **modified v95** client whose wire shape is largely additive over v83 (the easy path; covers most drift the spike found), and **stock Nexon v95**, which introduces Nexon-passport authentication and structural reorderings in serverbound handshake packets. The encoder design admits both via tenant configuration; stock-Nexon validation infrastructure (passport validation, partner-code handling) may ship in this task or split into a sibling task, see open questions.

---

## 2. Goals

### Primary goals
- **Wire-level v95 correctness** for every packet currently defined in `libs/atlas-packet/**` — encoders match the v95 IDA decompile byte-for-byte for GMS-v95 tenants; decoders consume the entire wire frame without leftover or short reads.
- **Reusable audit pipeline** committed under `tools/packet-audit/` that ingests the operation CSV, queries IDA via MCP (or accepts a pre-exported IDA artifact), reads `template_gms_*.json` for the writer/handler mapping, and emits per-packet diff reports against `libs/atlas-packet/**`. Output is markdown reports; never auto-mutates `.go` files.
- **Sub-struct audit coverage** for the shared types that participate in many packets — at minimum `CharacterStat`, `AvatarLook`, `ChannelLoad`, `AttackInfo`, `Asset`, and any others the audit pipeline flags as referenced by ≥3 packets.
- **Dual-client-target design** — the encoder pattern admits both modified-v95 (additive drift) and stock-Nexon-v95 (Nexon passport + field reorder) without forcing one to be a special case in the other's code path. A tenant flag selects behavior.
- **Round-trip test coverage** for every audited packet across every version listed in the test context (v12, v83, v87, v92, v95, v111, JMS v185), using `libs/atlas-packet/test/roundtrip.go`.

### Non-goals
- Atlas-side runtime / business-logic changes (handlers, processors, Kafka flows) beyond what's required to bridge a renamed or reshaped wire field to the existing service contract. New service features are out of scope.
- Other game versions' *new* correctness work — v87/v92/v111/JMS v185 are explicitly *not* targets for fix-up in this task. The audit pipeline produces reports for those versions as a side effect, but no encoder changes for non-v95 versions ship here unless a v95 fix would regress them (in which case the corresponding non-v95 branch is preserved).
- WZ data / game-content updates.
- Client distribution, client patching, anti-cheat changes.
- The legacy `legacy-atlas-packet-improvements` Phase 1–4 work (ResolveCode unification, foreign effect concretization, inventory change move-into-lib, no-op decode completion). Adjacent but separately scoped; this task assumes that work either lands first or runs in parallel without conflict.

---

## 3. User Stories

- As an **atlas-channel/atlas-login operator**, I want to add a tenant configured with `region=GMS, majorVersion=95` and serve a v95 client end-to-end without manual packet patches, so that v95 support is a configuration concern, not a code-fork concern.
- As an **atlas backend developer adding v87 (or another version) support next quarter**, I want to run `tools/packet-audit` against the v87 IDA + `template_gms_87_1.json` and get a per-packet markdown report telling me which encoders need work, so that I am not re-doing the v95 spike from scratch.
- As a **code reviewer**, I want a checked-in audit report alongside any encoder change that cites the IDA function and field-by-field comparison, so that I can verify the change against the source-of-truth rather than the author's claim.
- As an **on-call engineer triaging "client disconnects on login"**, I want the audit pipeline's report output to be readable enough to localize a field misalignment without me running IDA myself, so that v95-specific regressions are debuggable from the report alone.
- As a **multi-world tenant administrator**, I want the world-id field in `ServerListEntry` to reflect the configured world id, not a hardcoded `1`, so that worlds 2+ render correctly in the client.

---

## 4. Functional Requirements

### 4.1 Audit pipeline (`tools/packet-audit/`)

- **Inputs:**
  - `docs/packets/MapleStory Ops - ClientBound.csv` + `... - ServerBound.csv` (`FName` ↔ per-version opcode mapping).
  - One or more `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json` files (`writer`/`handler` name ↔ opcode mapping for a specific tenant version).
  - IDA decompiles — either via the live `mcp__ida-pro__*` tool set when the user is connected to an IDB, or via a pre-exported JSON artifact format the tool also supports (so CI / non-IDA contributors can run a diff without IDA).
  - `libs/atlas-packet/**/*.go` (current Atlas implementation).

- **Resolver behavior:**
  - For each CSV row with a non-zero opcode at the target version, resolve `FName` → IDA function address.
  - Decompile, extract the ordered sequence of `CInPacket::Decode{1,2,4,8,Buffer,Str}` (clientbound) or `COutPacket::Encode{1,2,4,8,Buffer,Str}` (serverbound) calls, normalized into a primitive field list (e.g. `[byte, byte, int32, str, ...]`).
  - For each writer/handler name in the chosen template, locate the corresponding atlas-packet `.go` file by walking the **full** `libs/atlas-packet/**/*.go` tree (writers can live outside their nominal domain — e.g. `CharacterList` is under `character/clientbound/list.go`, not `login/`).
  - Extract atlas-packet's wire sequence by static analysis of `Encode`/`Decode` method bodies (`w.Write*` / `r.Read*` calls, with awareness of `if t.Region() == ... { ... }` and `if t.MajorVersion() > N { ... }` branches; each branch is enumerated as a separate variant for the tenant context that selects it).

- **Output:**
  - One markdown report per packet under `docs/packets/audits/<region>_v<major>/<writer-or-handler-name>.md`, structured similarly to the spike report's per-packet sections (header → v95 wire table → atlas diff table → drift summary).
  - One roll-up report at `docs/packets/audits/<region>_v<major>/SUMMARY.md` categorizing findings: ✅ matches, ⚠️ minor, ❌ blocker, 🔍 deferred (e.g. recursive sub-struct).
  - Machine-readable JSON sidecar (`<name>.json`) per packet with the field arrays and verdicts, so downstream tooling can grep / aggregate.

- **What the pipeline does NOT do:**
  - Never writes to `libs/atlas-packet/**/*.go`. All fixes are human-applied.
  - Never modifies templates or CSVs (those are inputs).

### 4.2 Sub-struct (recursive) audit

- The pipeline recurses into atlas-packet sub-models invoked from a top-level `Encode`/`Decode` (e.g. `model.CharacterListEntry.Encode`, `model.Asset.Encode`). For each sub-model:
  - Locate the corresponding C++ struct decoder in IDA (e.g. `GW_CharacterStat::Decode`, `AvatarLook::Decode`) — by name lookup or by following the call site in the parent function's decompile.
  - Produce its own per-struct audit report under `docs/packets/audits/<region>_v<major>/_substruct/<StructName>.md`.
- Sub-structs referenced by ≥3 distinct top-level packets get a dedicated audit report; sub-structs referenced once are reported inline in the parent's audit.

### 4.3 Version-conditional encoder pattern

- The standard pattern is `if t.MajorVersion() >= N { ... }` / `if t.Region() == "X" { ... }` blocks inside the existing `Encode`/`Decode` method bodies, as already used in `auth_success.go`, `request.go`, `list.go`, `server_list_entry.go`, etc.
- A small helper API may be added under `libs/atlas-packet/version/` to standardize comparison operators (e.g. `version.AtLeast(t, 95)`, `version.Between(t, 87, 95)`, `version.Region(t) == version.GMS`) so encoders read uniformly. Strictly optional — inline checks remain legal — but if introduced, all newly touched encoders use it for consistency.
- **Branch nesting limit:** any single packet whose `Encode` method exceeds 3 levels of nested version/region branching is a candidate for a per-version sub-file split (see 4.4). The pipeline's report flags candidates; the design phase decides per-packet.

### 4.4 Structural drift handling (stock-Nexon v95 path)

- For packets where v95 reorders, replaces, or removes fields (not purely appends them) — `LoginHandle.Request` is the spike-confirmed example — the encoder admits one of two implementations, chosen at design time per packet:
  - **(a) Two-arm conditional inside the same file**, when the structural change is small (≤4 lines of distinct logic).
  - **(b) Sibling per-version file**, e.g. `libs/atlas-packet/login/serverbound/request_v95.go`, dispatched from a shared entry point that reads tenant version once. Used when (a) would push branch nesting past the limit in 4.3.
- A **client-variant tenant flag** distinguishes stock-Nexon-v95 from modified-v95. Proposed flag location: `template_gms_95_*.json` socket-level config, surface name `clientVariant: "stock" | "modified"` (default `"modified"` for back-compat with how the existing template appears to be configured today). Encoders reference this flag the same way they reference `MajorVersion` / `Region`.

### 4.5 Concrete fixes from the spike

The following spike-surfaced bugs ship with this task:

| Bug | File | Severity |
|-----|------|----------|
| `AuthSuccess` field-7 width: writes `byte` where v95 expects `int16` (subGradeCode + testerAccount) | `libs/atlas-packet/login/clientbound/auth_success.go` | Blocker for v95 |
| `LoginHandle.Request` field shape on stock v95 (passport, partnerCode) | `libs/atlas-packet/login/serverbound/request.go` | Blocker for stock-v95; deferred under modified-v95 |
| `ServerListEntry` per-channel `WriteByte(1)` hardcoded; should be `byte(m.worldId)` | `libs/atlas-packet/login/clientbound/server_list_entry.go` | Cross-version bug; ships regardless |
| Field-label corrections (semantic-only): `clientId` → `dwCharacterID` in `ServerIP`; `hwid` → `MachineId` in `Request`; `hwid` → `MacWithHDDSerial` in `CharacterSelect`; "quiet ban timestamp" → "chatUnblockDate" in `AuthSuccess` | various | Cosmetic; ship with the audit pass that touches each file |

The full bug list ships from the audit-pipeline output. The four above are the minimum guaranteed-included items.

### 4.6 Phasing requirement

Given total scope (estimated 250+ packets across login + channel + serverbound + sub-structs), the task is phased; each phase is independently mergeable and produces a usable audit artifact. The plan phase (`/plan-task`) refines specifics; phasing structure is non-negotiable:

1. **Phase A — Tooling foundation.** Audit pipeline scaffolding, CSV + template parsers, IDA-MCP shim with a fallback exported-JSON format, static-analysis extractor for atlas-packet `Encode`/`Decode`, markdown + JSON report writers. Phase exits when running against the spike's 6 packets reproduces the spike report's findings within tolerance.
2. **Phase B — Login domain.** Full audit pass + fixes for `libs/atlas-packet/login/**` (both client- and serverbound). Includes `AuthSuccess` width fix and `ServerListEntry` multi-world fix. Phase exits when every login packet reports ✅ or has an opened follow-up issue, and v95 round-trip tests pass for the login domain.
3. **Phase C — Sub-structs.** Recursive audit of `CharacterStat`, `AvatarLook`, `Asset`, `ChannelLoad`, `AttackInfo`, plus any others the pipeline identifies. Each sub-struct gets a dedicated audit report. Fixes ship with this phase; phase exits when sub-struct round-trip tests pass for every version in the test context.
4. **Phase D — Channel domain clientbound.** All server→client packets in `libs/atlas-packet/{character,inventory,monster,drop,field,pet,...}/clientbound/**`. The bulk of user-visible drift.
5. **Phase E — Serverbound (channel).** All client→server handlers in channel domain.
6. **Phase F — Stock-Nexon variant.** Implements the `clientVariant: "stock"` paths for the small set of structurally-different packets (login handshake, character select PIC variants). Includes any Nexon-passport integration plumbing required. *May spin out to a separate task* — flagged in open questions.

---

## 5. API Surface

This task does not add user-facing REST APIs. Internal APIs that change:

### 5.1 `tools/packet-audit` CLI

```
packet-audit \
  --csv-clientbound  docs/packets/MapleStory\ Ops\ -\ ClientBound.csv \
  --csv-serverbound  docs/packets/MapleStory\ Ops\ -\ ServerBound.csv \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       mcp                  # or: --ida-source export.json
  --output           docs/packets/audits/gms_v95
```

Exit codes: `0` clean, `1` any ❌ blocker, `2` ⚠️ warnings only, `3` runtime error.

### 5.2 Optional `libs/atlas-packet/version/` helper

```go
// version.AtLeast reports whether t.MajorVersion() >= n.
func AtLeast(t tenant.Model, n int) bool

// version.Region returns the canonical region constant.
func Region(t tenant.Model) Region

// version.IsStockClient reports whether the tenant config marks this version as a stock-client deployment.
func IsStockClient(t tenant.Model) bool
```

Optional — encoders may keep inline `t.MajorVersion()` checks. If introduced, new code uses it; old code is not bulk-migrated as part of this task.

### 5.3 Template schema addition

`template_<region>_<major>_<minor>.json` gains an optional top-level field:

```json
{
  "region": "GMS",
  "majorVersion": 95,
  "minorVersion": 1,
  "clientVariant": "stock" | "modified",
  ...
}
```

Default `"modified"` if absent. atlas-configurations REST handler and Go model are extended; existing templates are not modified.

---

## 6. Data Model

No database changes. Configuration-only:

- `atlas-configurations` template model gains the `clientVariant` field. Loader treats missing-or-empty as `"modified"`. No migration; templates are JSON files loaded at runtime.
- atlas-packet does not persist anything. Per-tenant version is already in `tenant.Model`.

---

## 7. Service Impact

- **`libs/atlas-packet`** — primary. Most files in `clientbound/` and `serverbound/` subtrees touched by audit + fix passes. New optional `version/` package. Round-trip tests extended.
- **`tools/packet-audit/`** — new Go CLI tool (this task creates the directory).
- **`services/atlas-login`** — consumes login packets. May need adapter-layer adjustments where field semantics changed (e.g. `Request.Password()` is in fact the password under modified-v95 but a different field under stock-v95). Adapter wires the right method to the right downstream service call based on tenant flag.
- **`services/atlas-channel`** — consumes channel packets. Same pattern as atlas-login for any field-semantic shifts surfaced by Phase D/E.
- **`services/atlas-configurations`** — adds `clientVariant` to the template model + REST handler + provider/transform layer. Includes mock update per project convention.
- **`docs/packets/`** — populated with `audits/<region>_v<major>/` per the audit pipeline. The spike report at `docs/packets/spike-login-v95.md` is preserved as the rationale-of-record for the strategy.
- **CI** — a new check runs `packet-audit` against the checked-in IDA JSON export and fails the build on new ❌ blockers introduced by a PR. Details refined during planning.

---

## 8. Non-Functional Requirements

### 8.1 Performance
- The audit pipeline must complete a full library scan against one version in <10 minutes on developer hardware (excluding IDA MCP round-trip latency, which is bound by user setup). The JSON-fallback path must complete in <2 minutes.
- Encoder hot paths: any new `version.AtLeast` helper must be inlinable. No reflection, no map lookups per encode call.

### 8.2 Security
- No user-controlled input reaches the encoder logic — version comes from tenant config, not request data.
- Stock-Nexon-v95 passport handling, if implemented in this task, must validate the passport token before trusting any field derived from it. Validation surface needs threat-modelling during planning (TBD what Nexon backend integration looks like — see open questions).

### 8.3 Observability
- Encoder version-branch coverage is logged at DEBUG level when a branch is selected: e.g. `WithField("packet", "AuthSuccess").WithField("variant", "gms_v95_modified").Debug("encoded")`. Optional — adds noise; planning decides which packets warrant it.
- Audit pipeline JSON output is structured so it can be ingested into a CI dashboard (per-packet match/warn/fail counts trend over time).

### 8.4 Multi-tenancy
- Every encoder reads version+region from `tenant.MustFromContext(ctx)` — no global state, no version flags that leak between tenants. Already the case; this requirement is to prevent regression.
- `clientVariant` flag is per-tenant. A single Atlas deployment must be able to serve modified-v95 and stock-v95 tenants concurrently on the same channel server.

### 8.5 Testing
- Round-trip tests run for every audited packet across every version in `libs/atlas-packet/test/context.go`. Existing `roundtrip.go` is extended if needed to iterate contexts.
- Audit pipeline has unit tests against fixture CSVs, fixture templates, and a fixture IDA-export JSON.
- A regression test ensures that for any tenant context where the audit pipeline reports ✅, round-trip succeeds with no leftover/short bytes.

---

## 9. Open Questions

1. **Stock-Nexon-v95 passport validation infrastructure** — does the integration with Nexon's authentication backend ship in this task (Phase F), or split into a sibling task that this one unblocks? Hinges on whether Atlas has any prior Nexon integration to build on, or starts from zero. *Recommendation: split if no prior integration exists.*
2. **IDA-MCP availability for CI** — the live `mcp__ida-pro__*` tool depends on a user's IDA Pro install. CI cannot run IDA. The pipeline supports a JSON-exported-from-IDA fallback; **who maintains the export and how often is it refreshed?** Options: (a) checked into repo as `docs/packets/ida-exports/gms_v95.json`, refreshed manually; (b) pulled from a secondary CI artifact store; (c) generated by a maintainer-only nightly job. Planning to decide.
3. **`legacy-atlas-packet-improvements` overlap** — that task's Phase 4 ("Implement Decode for solvable no-op packets") and the v95 audit's encoder-completeness checks overlap on the same files. **Coordinate sequencing or merge?** Likely just need awareness; concrete decision in planning.
4. **Field rename strategy** — fixes like "`clientId` → `dwCharacterID`" rename public model fields. **Do these go in a single rename PR per phase, or batched per audit pass?** Affects review burden vs. PR atomicity.
5. **Test-context expansion** — the audit pipeline will surface drift for v87, v92, v111, JMS v185 even though they're non-goals for fixes. **Do we open follow-up issues for each non-v95 finding, or commit the reports as informational-only?** Default: informational-only; fixes filed only on explicit request.
6. **Audit-pipeline placement** — `tools/packet-audit/` is the proposal. Alternatives: under `libs/atlas-packet/internal/audit/` (keeps it close to the audited code, but conflates lib and tool); standalone repo. Default to `tools/` per existing project convention (cf. `tools/task-numbers.sh`).
7. **Branch-nesting threshold (4.3 limit)** — proposed at 3 levels. Is that the right number, or should it be a "warning at 3, error at 4" gradient? Planning calibrates against measured branch depth in the existing code.

---

## 10. Acceptance Criteria

The task is "done" when **all** of:

### Tooling
- [ ] `tools/packet-audit/` builds with `go build ./tools/packet-audit/...` and `go test` is clean.
- [ ] Running `tools/packet-audit --template <gms_v95> --ida-source <exported>` against the checked-in IDA export reproduces the six per-packet findings in `docs/packets/spike-login-v95.md` (within field-name normalization tolerance).
- [ ] `packet-audit` writes per-packet markdown reports + JSON sidecars under `docs/packets/audits/gms_v95/`.
- [ ] CI fails on a PR that introduces a new ❌ blocker finding without an accompanying audit-report update justifying the change.

### Wire correctness
- [ ] Every packet under `libs/atlas-packet/login/**` has an audit report at `docs/packets/audits/gms_v95/<name>.md` and the report's verdict is ✅ or ⚠️ (no remaining ❌ for modified-v95).
- [ ] Every packet under `libs/atlas-packet/{character,inventory,monster,drop,field,pet,reactor,quest,party,guild,buddy,chat,messenger,note,pet,merchant,interaction,fame,storage,cash,ui,socket}/**` has an audit report and verdict ✅ or ⚠️ for modified-v95.
- [ ] Spike-surfaced bugs in 4.5 are fixed; PR diffs reference their respective audit reports.
- [ ] Sub-structs `CharacterStat`, `AvatarLook`, `Asset`, `ChannelLoad`, `AttackInfo` (and any others ≥3-referenced) have dedicated audit reports under `_substruct/`.

### Testing
- [ ] `go test -race ./libs/atlas-packet/...` passes.
- [ ] Round-trip tests iterate over every version context in `libs/atlas-packet/test/context.go` and pass.
- [ ] `go vet ./libs/atlas-packet/...` clean.
- [ ] Docker builds clean for `atlas-login`, `atlas-channel`, `atlas-configurations` (touched-shared-lib gate per CLAUDE.md).

### Configuration
- [ ] `clientVariant` field added to atlas-configurations template model, REST handler, provider, transform layer, and mock; default behavior is `"modified"`.
- [ ] `template_gms_95_1.json` either retains its existing implicit-modified behavior or is explicitly set to `clientVariant: "modified"`.

### Documentation
- [ ] `docs/packets/spike-login-v95.md` and the CSVs are preserved on the task branch.
- [ ] A `tools/packet-audit/README.md` explains how to run the pipeline, where reports land, and how to refresh the IDA export.
- [ ] `docs/packets/audits/gms_v95/SUMMARY.md` lists every packet, its verdict, and links to its per-packet report.

### Stock-Nexon-v95
- [ ] EITHER Phase F ships in this task (LoginHandle stock variant, character-select PIC variants, passport-validation plumbing or a clearly-stubbed integration), OR a sibling task is filed with a PRD that this task explicitly enables.
- [ ] In either case, the encoder pattern admits the stock variant via the `clientVariant` flag; no v95-modified code path is special-cased to be the "default".

---

## Appendix A — Spike artifacts on this branch

- `docs/packets/spike-login-v95.md` — full six-packet audit narrative.
- `docs/packets/MapleStory Ops - ClientBound.csv`, `... - ServerBound.csv` — CSV inputs.
- (Future) `docs/packets/ida-exports/gms_v95.json` — IDA decompile field-sequence export, generated by maintainer + checked in (per open question 2 resolution).
