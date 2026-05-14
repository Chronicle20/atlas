# Backend Guidelines Audit — task-027-atlas-packet-v95-audit

- **Date:** 2026-05-13
- **Branch:** `task-027-atlas-packet-v95-audit`
- **Base SHA:** `c2b7e5eaec63cee7fe689f92e694d7ad9362a1f8`
- **HEAD SHA:** `0e937b165c5f24ddd928f2ec095ea0b45d1037c9`
- **Reviewer mindset:** adversarial — default FAIL, every PASS must cite file:line

## Build & Test Gate (Phase 1)

| Module | `go build ./...` | `go test ./... -count=1` |
|--------|------------------|--------------------------|
| `libs/atlas-tenant/` | PASS | PASS |
| `libs/atlas-packet/` | PASS | PASS |
| `services/atlas-configurations/atlas.com/configurations` | PASS | PASS |
| `tools/packet-audit/` | PASS | PASS |

Gate: PASS. Proceeding to Phase 2.

## Scope of Audit

Per the task brief, the changed Go areas under audit are:

1. `libs/atlas-tenant/` — new `clientVariant` field, accessor, `CreateWithVariant`, JSON round-trip
2. `libs/atlas-packet/version/` — new helper package
3. `libs/atlas-packet/test/context.go` — new `CreateContextWithVariant`
4. `libs/atlas-packet/login/clientbound/auth_success.go` + `server_list_entry.go` — wire fixes
5. `libs/atlas-packet/login/serverbound/request.go` + `request_stock.go` — Passport accessor + variant dispatch stub
6. `services/atlas-configurations/atlas.com/configurations/templates/rest.go` — new `ClientVariant` REST field
7. `services/atlas-configurations/atlas.com/configurations/templates/validation_error.go` — unwired validator helper
8. `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — seed update

The `tools/packet-audit/` CLI is a maintainer tool, not a runtime service; DOM/SUB checks for Kafka/REST/tenant context do not apply. It is excluded from the per-domain checklist by the task brief.

The `templates` package is a **blob-storage configuration package** (RestModel is persisted directly as a JSON blob in `entity.Data` via `json.Marshal(input)` in `processor.go:73`). It does NOT follow the standard `model.go / builder.go / Transform / TransformSlice` DDD layout, and the changes here do not introduce that pattern. The applicable DOM checks are limited to what this diff touches: REST shape, validation, and handler discipline. Full DDD-layout checks (DOM-01 through DOM-05) are N/A for the templates package as a whole and were not in scope for this task.

## Phase 3: Per-Domain Mechanical Checks

### Package: `libs/atlas-tenant`

Library package, not a service domain. Applicable checks are functional-correctness only.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| TEN-01 | `clientVariant` is private with accessor | PASS | `libs/atlas-tenant/tenant.go:15` (private field), `libs/atlas-tenant/tenant.go:34-39` (accessor `ClientVariant()`) |
| TEN-02 | Accessor defaults empty string to `"modified"` | PASS | `libs/atlas-tenant/tenant.go:35-38` returns `"modified"` when field is empty |
| TEN-03 | `Is()` compares clientVariant | PASS | `libs/atlas-tenant/tenant.go:91-93` |
| TEN-04 | JSON round-trips `clientVariant` | PASS | `libs/atlas-tenant/tenant.go:47,53,63,74` and verified by `tenant_test.go:30-43` |
| TEN-05 | `CreateWithVariant` constructor exists | PASS | `libs/atlas-tenant/processor.go:35-42` |
| **TEN-06** | **`WithContext` / `FromContext` propagate `clientVariant`** | **FAIL** | `libs/atlas-tenant/processor.go:99-106` (`WithContext`) writes only `ID`, `Region`, `MajorVersion`, `MinorVersion` to ctx — `clientVariant` is dropped. `processor.go:65-87` (`FromContext`) never reads it. **Empirically reproduced:** a tenant constructed with `CreateWithVariant(..., "stock")` returns `variant=stock` before `WithContext`, but `variant=modified` after `MustFromContext(WithContext(ctx, t))`. This is the load-bearing failure for the entire variant-dispatch design (see TEN-07, DOM-VAR-01 below). |
| **TEN-07** | **No production caller of `CreateWithVariant`** | **FAIL (dead-code class)** | `grep -rn "CreateWithVariant" services/ libs/` returns only `libs/atlas-packet/test/context.go:31` (test helper) and `libs/atlas-tenant/tenant_test.go`. The new constructor exists, but no real service ever builds a tenant from the new templates `clientVariant` REST field — `services/atlas-world/atlas.com/world/channel/processor.go:147` still calls `tenant.Create(...)` and `services/atlas-monsters/atlas.com/monsters/monster/registry.go:172` likewise. Combined with TEN-06, the variant is never wired end-to-end. Anti-patterns.md: "Leaving dead code after refactoring." |

### Package: `libs/atlas-packet/version` (new)

Helper package; no model.go, builder.go, entity.go, or rest.go expected.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| VER-01 | `version.go` exists with documented helpers | PASS | `libs/atlas-packet/version/version.go:1-41` |
| VER-02 | `RegionOf`, `AtLeast`, `LessThan`, `Between`, `VariantOf`, `IsStock` defined | PASS | `libs/atlas-packet/version/version.go:21,23,24,26-29,33-38,40` |
| VER-03 | `VariantOf` defaults to `Modified` for back-compat | PASS | `libs/atlas-packet/version/version.go:33-38` |
| VER-04 | Structural type-assertion seam is documented | PASS | `libs/atlas-packet/version/accessor.go:5-11` includes a clear "Task 14 will replace" comment and reason |
| VER-05 | Table-driven tests for helpers | PARTIAL PASS | `libs/atlas-packet/version/version_test.go:16-38` exercises `AtLeast`, `Between`, `RegionOf` only. `VariantOf` and `IsStock` are NOT directly tested in this file — they only get exercised transitively via `request_stock_test.go`, which (see DOM-VAR-01) does not actually assert dispatch. testing-guide.md "Table-driven tests" — only PARTIAL coverage of the new helpers. Recommend FAIL-leaning WARN. |

### Package: `libs/atlas-packet/login/clientbound`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| AS-01 | v95 field-7 width fix (byte → int16) | PASS | `libs/atlas-packet/login/clientbound/auth_success.go:51-55` (encode) and `:113-117` (decode), guarded by `t.Region() == "GMS" && t.MajorVersion() >= 95` |
| AS-02 | Test asserts wire length matches spike | PASS | `libs/atlas-packet/login/clientbound/auth_success_test.go:16,28-30` (`wantLen = 57`, fails if mismatch) |
| AS-03 | Round-trip test across variants | PASS | `libs/atlas-packet/login/clientbound/auth_success_test.go:33-62` iterates `pt.Variants` |
| SLE-01 | Channel-loop writes `worldId`, not hardcoded `1` | PASS | `libs/atlas-packet/login/clientbound/server_list_entry.go:72` writes `byte(m.worldId)`; previously was `WriteByte(1)` per task brief. Decoder at `:115` reads the byte but discards it (still `_ = r.ReadByte()`). |
| SLE-02 | Top-level worldId field uses model value | PASS | `libs/atlas-packet/login/clientbound/server_list_entry.go:50` writes `byte(m.worldId)` |
| **SLE-03** | **Decoder asserts the world-id byte equals `m.worldId`** | **FAIL (WARN)** | `libs/atlas-packet/login/clientbound/server_list_entry.go:115` reads the per-channel world id and discards it with `_ = r.ReadByte()`. The test at `server_list_entry_test.go:53-56` does check it in an inline-parse path, but the package's own `Decode` does not populate or verify it. A latent regression here (e.g., re-introducing the `1` constant on encode) would not break the package's round-trip test because the decode discards. WARN, not blocking. |

### Package: `libs/atlas-packet/login/serverbound`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| REQ-01 | `Passport()` accessor exists | PASS | `libs/atlas-packet/login/serverbound/request.go:36-38` |
| REQ-02 | `Decode` dispatches by variant | PARTIAL | `libs/atlas-packet/login/serverbound/request.go:77-83` reads `version.IsStock(t)` and routes to `decodeStock` vs `decodeModified`. Code path exists, but see DOM-VAR-01. |
| REQ-03 | `decodeStock` is clearly stubbed | PASS | `libs/atlas-packet/login/serverbound/request_stock.go:10-20` explicitly documents this is a slot/stub for sibling task, body is `_ = r` |
| **DOM-VAR-01** | **Variant-dispatch test actually exercises the chosen branch** | **FAIL** | `libs/atlas-packet/login/serverbound/request_stock_test.go:10-23` claims to test stock-variant dispatch. Two failures: (1) the test only asserts `dec != nil` and `r.Passport() == ""`, both of which are true for `decodeModified` as well — the test passes regardless of which branch ran; (2) per TEN-06, `pt.CreateContextWithVariant("GMS", 95, 1, "stock")` round-trips through `tenant.WithContext` which silently drops the variant, so `version.IsStock(t)` in `request.go:79` returns false and the **modified** path actually executes. The test gives false confidence: it would pass even if `decodeStock` were `panic("never called")` (it isn't, because the dispatch never reaches it). Block of consequence: any later code that assumes "stock path is reachable from ctx-stamped tenant" will silently use modified decoding. |

### Package: `services/atlas-configurations/atlas.com/configurations/templates`

Blob-storage config package; full DDD checklist (DOM-01..05) is N/A for this layout. Only checks that touch the diff are applied.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| TEMPL-01 | New REST field has flat JSON tag | PASS | `services/atlas-configurations/atlas.com/configurations/templates/rest.go:17` (`ClientVariant string \`json:"clientVariant,omitempty"\``) |
| TEMPL-02 | Field round-trips through marshal/unmarshal | PASS | `services/atlas-configurations/atlas.com/configurations/templates/rest_test.go:156-180` |
| TEMPL-03 | JSON:API interface implemented (DOM-18) | PASS (pre-existing) | `services/atlas-configurations/atlas.com/configurations/templates/rest.go:25-36` |
| TEMPL-04 | Flat request model, no Data/Type/Attributes envelope (DOM-19) | PASS | `services/atlas-configurations/atlas.com/configurations/templates/rest.go:11-23` |
| TEMPL-05 | POST/PATCH use `RegisterInputHandler[T]` (DOM-08) | PASS (pre-existing) | `services/atlas-configurations/atlas.com/configurations/templates/resource.go:23,27` |
| TEMPL-06 | Handlers pass `d.Logger()` (DOM-07) | PASS | `services/atlas-configurations/atlas.com/configurations/templates/resource.go:36,64,84,102,121,145` |
| TEMPL-07 | Domain error → HTTP status mapping (DOM-17) | PASS (pre-existing for validation) | `services/atlas-configurations/atlas.com/configurations/templates/resource.go:125-134` maps `validationFailureError` to 400 |
| **TEMPL-08** | **`validateClientVariant` is wired into the validation entry point** | **FAIL** | `services/atlas-configurations/atlas.com/configurations/templates/validation_error.go:39-45` defines `validateClientVariant`. `grep -rn validateClientVariant services/ libs/` shows only **the test file** (`rest_test.go:184,185,188,189`) calls it — `processor.go:72,105` (`Create`, `UpdateById`) never invoke it. The plan explicitly says (plan.md:3200): *"Wire it into whatever the existing validation entry point is."* The comment in `validation_error.go:36-38` admits the helper is unwired: *"Kept as an unexported helper for future REST validation hooks."* Consequence: a `PATCH /configurations/templates/<id>` body of `{"clientVariant":"bogus", ...}` is accepted and persisted; only later consumers (`version.VariantOf`) silently coerce `"bogus"` to its own ClientVariant string, with no upstream rejection. anti-patterns.md "Missing validation — allows invalid domain states." Plan-adherence and DOM-correctness FAIL. |
| TEMPL-09 | Seed data includes `clientVariant` | PASS | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:6` (`"clientVariant": "modified"`) |
| TEMPL-10 | Mock processor updated (per project convention) | N/A | `find services/atlas-configurations -name 'mock*.go'` returns nothing for templates; no mock to update. |

### Package: `libs/atlas-packet/test`

Test-helper package. Only one relevant check.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| **TST-01** | **`CreateContextWithVariant` actually round-trips the variant** | **FAIL** | `libs/atlas-packet/test/context.go:30-33` calls `tenant.CreateWithVariant(...)` then `tenant.WithContext(...)`. Because `WithContext` does NOT serialize `clientVariant` into the context (see TEN-06), any test that retrieves the tenant via `MustFromContext` later in the encode/decode pipeline will see `ClientVariant() == "modified"`. The helper's *name* implies it sets up a variant-aware ctx; the *behavior* silently drops the variant. This is the upstream cause of DOM-VAR-01. |

## Phase 4: Security Review

The changed packages do not handle authentication, authorization, or token management directly. The `Passport` accessor (`request.go:36-38`) is a payload-field accessor, not a security boundary — actual passport validation is explicitly deferred to a sibling task per `request_stock.go:11-14`. No JWT, no token revocation, no redirect logic, no hardcoded secrets observed in the diff.

| ID | Check | Status |
|----|-------|--------|
| SEC-01 | JWT validation uses verified parsing | N/A — no JWT code in diff |
| SEC-02 | Token revocation checks validated tokens | N/A |
| SEC-03 | No open redirect | N/A |
| SEC-04 | No hardcoded secrets | PASS (grep confirms none in diff) |

## Summary

### Blocking (must fix before merge)

- **TEN-06** — `tenant.WithContext` / `FromContext` in `libs/atlas-tenant/processor.go:65-87,99-106` do not propagate `clientVariant`. Empirically demonstrated: a tenant constructed with `CreateWithVariant(..., "stock")` is silently downgraded to `"modified"` after a single `WithContext` → `MustFromContext` round-trip. This kills the entire variant-dispatch design at runtime.
- **TST-01** — `libs/atlas-packet/test/context.go:30-33` builds a variant-aware tenant but immediately throws away the variant via `tenant.WithContext`. Tests calling `CreateContextWithVariant("..., stock")` get a "modified" tenant downstream.
- **DOM-VAR-01** — `libs/atlas-packet/login/serverbound/request_stock_test.go:10-23` does not actually verify that `decodeStock` ran. The only assertions (`dec != nil`, `r.Passport() == ""`) are also true for `decodeModified`. Combined with TEN-06, the stock path is **never** exercised by the test suite despite the test's name.
- **TEMPL-08** — `validateClientVariant` (`services/atlas-configurations/atlas.com/configurations/templates/validation_error.go:39-45`) is defined but never invoked by `Create` (`processor.go:72`) or `UpdateById` (`processor.go:105`). Invalid `clientVariant` values are silently accepted and persisted. Plan explicitly required wiring; the implementer left a "future hooks" comment instead.

### Non-Blocking (should fix)

- **TEN-07** — No production caller of `CreateWithVariant`. The new templates `clientVariant` REST field round-trips through storage but no service that constructs a `tenant.Model` from a template ever uses the new constructor (e.g., `services/atlas-world/atlas.com/world/channel/processor.go:147` still calls `tenant.Create`). End-to-end the variant is unconsumed regardless of TEN-06.
- **VER-05** — `libs/atlas-packet/version/version_test.go` has no direct unit test for `VariantOf` or `IsStock`. They are only exercised transitively through `request_stock_test.go`, which itself is broken (DOM-VAR-01). Add a direct table-driven test.
- **SLE-03** — `libs/atlas-packet/login/clientbound/server_list_entry.go:115` discards the per-channel world-id byte in `Decode`. The encode-side fix (`:72`) won't be caught by any future regression that re-hardcodes a constant, because the package's own round-trip test cannot detect it. The accompanying `server_list_entry_test.go:53-56` does check via inline parsing, but the production decoder should match.

### Overall Status

**NEEDS-WORK.** Build and tests pass mechanically, but the variant-dispatch path that this entire task is built around does not work in practice: the variant is set by `CreateWithVariant` and then silently dropped by `WithContext`, and the one test that purports to verify dispatch does not actually verify it. The configurations-side validator is defined but unwired. Four blocking issues, three non-blocking. None of these are matters of style — all four blocking issues are reproducible behavioral defects.
