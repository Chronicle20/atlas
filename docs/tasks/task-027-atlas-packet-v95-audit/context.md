# Context — Atlas-Packet v95 Audit

Companion to `plan.md`. Captures the key files, decisions, and dependencies the implementer needs without re-reading the full PRD/design.

---

## Source artifacts (read these first)

- `docs/tasks/task-027-atlas-packet-v95-audit/prd.md` — full requirements, six bug list (PRD 4.5), phasing rationale.
- `docs/tasks/task-027-atlas-packet-v95-audit/design.md` — AST-analyzer design (§3), encoder pattern (§5), template/tenant plumbing (§6), phasing artifacts (§8).
- `docs/packets/spike-login-v95.md` — six-packet manual audit. The Phase A exit criterion is reproducing this report's findings via the tool.
- `docs/packets/MapleStory Ops - ClientBound.csv`, `... - ServerBound.csv` — `FName ↔ {version → opcode}` mapping. Already on the task branch.

---

## Key existing code references

| Concern | File | Notes |
|---|---|---|
| Tenant model + accessors | `libs/atlas-tenant/tenant.go:10-31` | Private fields, getters. `MajorVersion()` returns `uint16`. Add `clientVariant string` here. |
| Tenant constructor | `libs/atlas-tenant/processor.go:30` | `Create(id, region, major, minor) (Model, error)`. Keep back-compat; add `CreateWithVariant` sibling. |
| Tenant context | `libs/atlas-tenant/processor.go:80-89` | `MustFromContext` / `WithContext` — leave alone. |
| Canonical version-conditional encoder | `libs/atlas-packet/login/clientbound/auth_success.go:37-94` | Shape every new encoder mirrors. |
| Spike-confirmed multi-world bug | `libs/atlas-packet/login/clientbound/server_list_entry.go:73` | `w.WriteByte(byte(x.ChannelId() - 1))` is fine; the bug is `w.WriteByte(1)` on line 72 — should be `byte(m.worldId)`. Verify against spike report Packet 3. |
| Spike-confirmed width bug | `libs/atlas-packet/login/clientbound/auth_success.go:64` | `w.WriteInt(1)` ("nNumOfCharacter") is fine — the actual width drift the spike found is one of the later byte fields. Re-read spike Packet 1 for exact field index before fixing. |
| Round-trip harness | `libs/atlas-packet/test/roundtrip.go:12-24` | Asserts `reader.Available() == 0` after decode. This is the "wire complete" oracle. |
| Tenant variants for tests | `libs/atlas-packet/test/context.go:18-23` | Add `clientVariant` field to `TenantVariant` when Task 14 lands; iterate over the cross-product. |
| Existing roundtrip test pattern | `libs/atlas-packet/login/clientbound/auth_success_test.go:9-37` | `for _, v := range pt.Variants { t.Run(v.Name, ...) }` — copy this shape for new tests. |
| Template REST model | `services/atlas-configurations/atlas.com/configurations/templates/rest.go:11-22` | Add `ClientVariant string \`json:"clientVariant"\``. |
| Template entity | `services/atlas-configurations/atlas.com/configurations/templates/entity.go:14-20` | JSON blob in `Data` column — **no DB migration required** when adding fields. |
| Template seed file | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` | Anchor `clientVariant: "modified"` here once schema lands. |

---

## Critical decisions locked in design

- **Encoder pattern:** inline version conditionals in existing `Encode`/`Decode` bodies, optionally calling helpers from a new `libs/atlas-packet/version/` package. **No** "one file per (packet × version)" split unless a packet exceeds 3 levels of branch nesting (lint warning) or 4 levels (hard fail).
- **Sibling per-version file** (e.g. `request_stock.go`) only for whole-packet structural rewrites — stock-Nexon `LoginHandle.Request` is the canonical example. Tail-only widenings stay inline.
- **`clientVariant` flag** lives at the **template root** (`template_*.json`), propagates to `tenant.Model.ClientVariant()`, defaults to `"modified"` when absent.
- **AST analyzer (`tools/packet-audit/internal/atlaspacket`)** is the load-bearing component. It walks Go AST, collects `w.WriteX` / `r.ReadX` calls, parses `*ast.IfStmt` guards into a small predicate DSL (`regionEq`, `majorGE`, `majorLT`, `variantEq` + AND/OR/NOT), records sub-struct calls as recursion markers, and treats `for` loops as `repeat(count, body)`.
- **IDA source:** dual implementation behind `FieldSource` interface. `MCPSource` calls `mcp__ida-pro__*` live; `ExportSource` reads a checked-in JSON file. CI always uses `ExportSource`. Export file at `docs/packets/ida-exports/gms_v95.json` is the canonical artifact.
- **Round-trip cross-check** (design §3.4): Phase A's exit gate runs both the AST analyzer (Option C) and a byte-sniffer (Option B) on the spike's 6 packets and asserts field-list agreement. This catches analyzer bugs without hand-eyeballing.
- **Templates are JSON blobs** in the `templates` GORM table (see `entity.go` above). Adding `clientVariant` to `RestModel` requires **no** database migration — the field round-trips through `Entity.Data`.

---

## Decisions deferred to execution time

- **Phases C–E** (sub-structs, channel clientbound, channel serverbound) are scoped *after* Phase A ships, because the audit pipeline's output determines the actual workload. The plan documents this as a checkpoint, not a task list.
- **Phase F (stock-Nexon v95)** ships as a sibling task per design §9.1 unless Phase A–E come in under budget. This plan delivers only the `clientVariant` plumbing (template field, tenant accessor, encoder slot) — not Nexon-passport validation.
- **Open question 3** (`legacy-atlas-packet-improvements` overlap): the audit pipeline's `🔍 manual review` verdict naturally captures no-op-decode packets. No coordination work required up front.

---

## Workflow notes for the implementer

1. **Verify cwd before every commit** — you're in a worktree at `.worktrees/task-027-atlas-packet-v95-audit/`. The branch is `task-027-atlas-packet-v95-audit`. `git rev-parse --show-toplevel` and `git branch --show-current` must agree.
2. **Run `go test -race ./libs/atlas-packet/...` after every encoder edit.** A v95 fix that regresses v83 is the worst case; the round-trip suite is the canary.
3. **Run `go vet ./libs/atlas-packet/...` before each commit.**
4. **No service-Dockerfile changes are anticipated** for Phase A or Phase B login. The `atlas-packet` and `atlas-tenant` lib changes are visible to atlas-login / atlas-channel via existing `go mod edit -replace` lines in their Dockerfiles. Re-verify per CLAUDE.md's "Build & Verification" section *only if* you touch a service Dockerfile or service `go.mod`.
5. **The audit pipeline produces reports; never auto-mutates `.go` files.** Every encoder fix is a hand edit anchored to a freshly-generated audit report.

---

## External dependencies / open questions resolved

- IDA-MCP availability — CI uses checked-in `ExportSource` JSON. Refresh is a maintainer task (`make audit-export-v95`).
- Nexon-passport validation — split to sibling task; this plan stubs the validator.
- Field renames (e.g. `clientId → dwCharacterID`) — bundled with the audit-report PR that surfaces them.
- Non-v95 findings (v87, v92, v111, JMS v185) — committed as informational-only reports; no follow-up issues filed.
