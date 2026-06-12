# Task-083 Context — GMS v84 Tenant Support

Companion to `plan.md`. Key files, decisions, and dependencies an implementer needs before starting. All paths relative to the `task-083-gms-v84-tenant-support` worktree root.

## What this task is (and is not)

- **Is:** Add GMS v84.1 as a *new* tenant version alongside v83.1. No migration, no new entity/service/REST endpoint. v84 reuses the existing tenant + configuration + WZ model.
- **Is not:** Removing/deprecating v83; any non-GMS region or non-84.1 GMS version; new IDA tooling; atlas-ui changes (unless a hard provisioning blocker — OQ-5); full v84 feature parity (bar is "basic playthrough works").
- **Done means:** the live v84 playthrough passes AND v83 regression passes (design §1 — completion is coupled to a running cluster + a real v84 client, not a merge).

## The single source of truth

`docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (created in Task A0). Every opcode value, predicate range, and `usesPin` decision in code or template **must cite a row in this doc**. "Same as v83" is a finding with evidence, never a default (FR-1.3).

## Hard dependency order

```
A (delta) ─┬─▶ B (Go: helpers + audit)
           ├─▶ C (template)
           └    D (WZ ingest)  ── independent of B/C, still needs nothing from A
A+B+C+D ───▶ E (provisioning + live E2E, blocking)
```
B, C, D can run in parallel once A's deliverables (opcode maps + usesPin) exist. F (build gates) runs after B+C land.

## Key files & symbols (verified against source)

### Version identity & helpers (Component B)
- `libs/atlas-tenant/tenant.go` — `Model{ id, region string, majorVersion uint16, minorVersion uint16 }`, getters only, immutable. **No comparison helpers exist today** — every call site hand-rolls `MajorVersion() op N`. Task B1 adds `IsRegion`, `MajorAtLeast`, `MajorAtMost`, `MajorInRange`.
- Module name is `atlas-tenant` (the lib). It is consumed by many services → expect multiple `docker buildx bake` targets in Phase F.

### Correction sites (verified line numbers)
- `services/atlas-character/atlas.com/character/character/processor.go:1336` — `p.t.Region() == "GMS" && p.t.MajorVersion() == 83` gating Beginner/Noblesse/Legend auto-AP. **The one unambiguous bug** (exact `==83` excludes v84; inline TODO admits range undefined). Task B3.
- `services/atlas-account/atlas.com/account/account/processor.go:165` — `p.t.Region() == "GMS" && p.t.MajorVersion() > 83` sets default gender `10` (UI-choose). Already fires for v84; migration is behavior-identical (`MajorAtLeast(84)`). Task B4.
- Full enumeration: `grep -rn 'Region()\|MajorVersion()\|MinorVersion()' services/ libs/ --include='*.go' | grep -v _test.go | grep -E '==|!=|>=|<=|>|<'` → ~302 raw hits (design cited ~410 sites / ~42 unique predicates including non-comparison usages). Task B2 classifies every one.

### Socket config templates (Component C)
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` — the **complete** GMS template (the base to copy; 110 KB). Other GMS templates (12/87/92/95) are partial. JMS 185 also present.
- Template shape: top-level `region`, `majorVersion`, `minorVersion`, `usesPin` (all GMS = `false` today); `socket.handlers[] = {opCode, validator, handler}`; `socket.writers[] = {opCode, writer}`; non-socket `characters`/`npcs`/`worlds`/`cashShop`.
- **Seeder** `services/atlas-configurations/atlas.com/configurations/seeder/seeder.go` — `DefaultConfig()` reads `SEED_DATA_PATH` (default `/seed-data`), `SEED_ENABLED` (default true). Discovers every `*.json` under `templates/` and **skips any whose `(region, major, minor)` already exists** (`importTemplate` → `templateExists`). So adding `template_gms_84_1.json` is idempotent and cannot mutate the v83 row — **FR-2.3 holds by construction.**

### Handler/validator/writer name resolution (the FR-2.4 gate)
- Template name strings resolve to Go funcs via these registry builders (verified):
  - Login: `services/atlas-login/atlas.com/login/main.go` — `produceHandlers()` (line ~384), `produceValidators()` (~415), `produceWriters()` (writerList).
  - Channel: `services/atlas-channel/atlas.com/channel/main.go` — `produceHandlers()` (~688), `produceValidators()` (~764), `produceWriters()` (~586).
- Map keys are **string constants whose value equals the constant name**, e.g. `const LoginHandle = "LoginHandle"` in `libs/atlas-packet/login/serverbound/request.go`; `const NoOpValidator = "NoOpValidator"` / `LoggedInValidator` in `services/atlas-login/atlas.com/login/socket/handler/handle.go`. So the template uses the same strings.
- **Gate strategy (Task C2):** a name is dangling iff its quoted string literal appears **nowhere** in `services/atlas-login` + `services/atlas-channel` + `libs/atlas-packet` Go source. `tools/template-symbol-check.sh` greps for exactly that. Sanity-check it against v83 first (must pass).

### WZ data (Component D)
- Resolution: `services/atlas-data/atlas.com/data/data/runwz.go` — key format `<scope>/regions/<region>/versions/<major>.<minor>/<archive>` (e.g. `regions/GMS/versions/84.1/...`). Ingest is **operational, not code** — a k8s Job (`MODE=ingest`) reads `SCOPE/REGION/MAJOR_VERSION/MINOR_VERSION`; the REST `JobCreator` renders it from the `atlas-data-ingest-job-template` ConfigMap. No atlas-data code change expected.

## Decisions locked in design (do not re-litigate)

1. **Boundary predicates** → add `tenant.Model` range/comparison helpers and migrate touched sites (NOT named capability flags, NOT a central capability registry). Region stays a separate predicate. Inline comments name the capability a range encodes.
2. **Delta depth** → full v84 opcode-table dump (both directions) diffed against v83 (anchor) + v95 (tie-breaker). Packet *structure* exhaustive for in-scope flows, spot-checked elsewhere.
3. **Migration scope** → only sites that are (a) wrong for v84 or (b) on an in-scope flow. Correct-but-out-of-scope sites are recorded `unchanged (correct)` and left as-is (avoid needless v83-regression surface). Helpers will already exist if a future task wants a blanket migration.
4. **Live E2E blocks done.**

## Gotchas / prior-art memory to honor

- `bug_npc_msgtype_hardcoded_vs_config` — inbound handlers must **reverse-resolve the message-type table the writers use**; never hardcode enum bytes in Go. Relevant to FR-2.2 message-type table in the template.
- `bug_new_opcodes_not_in_live_tenant_config` — handler/writer bindings are **not hot-reloaded**; existing tenants don't pick up new opcodes. A fresh v84 tenant seeds at creation, but document the channel/login restart sequence (OQ-6 / Task E1 Step 4).
- `reference_atlas_maps_spawn_cache` — after v84 WZ ingest, `DEL atlas:maps:spawn:*` and DELETE affected monsters, else stale 83-era cache masks v84 data (Task D2 Step 3).
- `reference_atlas_data_wz_inspection` — three verified ways to inspect WZ/consumable data (local XML dump, live `GET /api/data/...` with version headers via throwaway curl pod, MinIO `atlas-wz`).
- `reference_rediskeyguard_invariant` — run `tools/redis-key-guard.sh` with `GOWORK=off`.
- `reference_ida_harvest_subagents` / `reference_ida_mcp_new_api` — one IDB loaded per instance; user switches; `select_instance(port)` to target v83/v84/v95. New API: batch tools, `structuredContent`+`isError`, demangled `Class::Method` → "Not found".
- `project_packet_audit_exporter_real_decompile_gaps` — real Hex-Rays output has alias sets, unnamed `sub_XXXX` descent, switch-expr, etc. The v84 IDB is partially named (OQ-7) → expect low-confidence opcodes; flag them and treat as first E2E suspects.
- `reference_observability` — k8s + Grafana MCP for live diagnosis during the playthrough; canonical failure signature `unhandled message op 0xXX` at info.

## Open questions resolved by which task

| OQ | Resolved by |
|---|---|
| OQ-1 usesPin | Task A4 Step 3 → encoded in template C1 Step 2 |
| OQ-2 login parity | Task A4 (exhaustive in-scope structure delta) |
| OQ-3 boundary semantics | Task A behavior evidence → Tasks B3/B5 ranges |
| OQ-4 WZ structural diffs | Task D2 (representative-set verification) |
| OQ-5 UI blocker | Task E1 (confirm REST/seed path sufficient; escalate only if UI version field is a hard blocker) |
| OQ-6 live-config restart | Task E1 Step 4 (documented working restart sequence) |
| OQ-7 IDB naming gaps | Tasks A1–A3 flag per-opcode low-confidence; first E2E suspects |

## Build & verification gates (Phase F — mandatory)

Per CLAUDE.md: `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-<svc>` for every service with a touched `go.mod` (atlas-tenant is widely consumed → multiple targets); `GOWORK=off tools/redis-key-guard.sh` clean; plus the `tools/template-symbol-check.sh` gate on the v84 template.
