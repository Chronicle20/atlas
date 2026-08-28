# Task 274 — Implementation Context

Companion to `plan.md`. Records the key files, the decisions the plan is built on, and the facts that were verified against the code rather than assumed.

---

## Key files, by module

### `libs/atlas-socket` (module root `libs/atlas-socket`)

| File | Role |
|---|---|
| `server.go:87-97` | the unexported `config` struct — gains `tracer PacketTracer` |
| `server.go:325-334` | `handle` — the single inbound emission site |
| `opts.go:55-59` | `SetHandlers` — the configurator shape `SetPacketTracer` copies |
| `server_test.go:37-73` | `serve(t, ops)` — builds `&config{...}` as a literal in an internal test; the new tests call `handle` directly with the same literal |
| `request/request.go:13` | `type Request []byte` — so `p` in `handle` is already the full plaintext frame including opcode bytes |
| `writer/writer.go:15-24` | `MessageGetter` — writes the opcode as the first 1-2 bytes of the outbound buffer, little-endian |
| `trace/` | **new** — the formatter package |

`go.mod` already requires `github.com/google/uuid v1.6.0` and `github.com/sirupsen/logrus v1.10.1`; the new `trace` subpackage lives in the same module and needs no dependency change.

### `libs/atlas-opcodes` (module root `libs/atlas-opcodes`)

| File | Role |
|---|---|
| `config.go:6-7` | `ServiceLogin = "login"`, `ServiceChannel = "channel"` |
| `config.go:11-21` | `HandlerConfig{OpCode, Validator, Handler, Options, Services}` |
| `config.go:36-46` | `appliesToService` — the service filter `BuildHandlerNames` reuses |
| `producer.go:58-87` | `BuildHandlerMap` — opcode parse at `:77`, map assignment at `:84` |
| `producer_test.go:24-51` | `warnContaining` / `warnCount` helpers over `test.NewNullLogger()` |
| `width.go` | **new** — `OpCodeSize` / `OpReadWriterFor` |

`go.mod:6` and `:17` already require and `replace` `libs/atlas-socket`, so `OpReadWriterFor` returning `socket.OpReadWriter` adds no dependency.

### `atlas-configurations` (module root `services/atlas-configurations/atlas.com/configurations`)

| File | Role |
|---|---|
| `tenants/rest.go:12-31` | `RestModel` — `CashShop` at `:22`, `MapleLife` at `:23`; `Diagnostics` follows the same non-pointer, no-`omitempty` shape |
| `tenants/cashshop/rest.go:1-11` | the sibling-package template |
| `tenants/processor.go:124-176` | `UpdateById` — `json.Marshal(input)` at `:147`, `enqueueTenantStatus(db, tenantId, sanitized)` at `:174` |
| `tenants/processor.go:187-238` | `Create` — same pattern at `:193` / `:232` |
| `tenants/rest_test.go:14-64` | `TestTenantRestModelCarriesMapleLife` — the exact round-trip template |
| `tenants/processor_test.go:53-79, :696-718, :757-786` | DB fixture, outbox-envelope decode helpers, and the update-path outbox test |

### `atlas-channel` (module root `services/atlas-channel/atlas.com/channel`)

| File | Role |
|---|---|
| `configuration/tenant/rest.go:10-20` | the projection mirror — gains `Diagnostics` |
| `configuration/registry.go:12-16` | `configMu`, `serviceConfig`, `tenantConfig` |
| `configuration/registry.go:68-79` | `GetTenantConfig` — the blocking version `TracePacketsEnabled` must not imitate |
| `configuration/registry.go:81-107` | `PublishSnapshot` and the doc comment promising the apply-loop call site |
| `configuration/projection/loop.go:86` | `a.State.Snapshot()` — the publish goes on the next line |
| `configuration/projection/apply.go:28-34` | `ListenerConfig{IPAddress, Port, Region, MajorVersion, MinorVersion}` |
| `configuration/projection/projection_test.go:92-149` | `TestComputeOps_AddRemovePortChangeUnchanged` |
| `configuration/registry_test.go:14-43` | the `done chan result` + `select`/`time.After` non-blocking idiom |
| `socket/init.go:83-159` | `CreateSocketService`; the `socket.Serve` configurator list is at `:114-140`, `t := sc.Tenant()` at `:98` |
| `session/processor.go:249-281` | `Announce`; `t := tenant.MustFromContext(ctx)` already present at `:257` |
| `session/processor.go:366-380` | `Create` — the sole `WriteHello` caller, at `:375` |
| `session/model.go:143-145` | `WriteHello` |
| `session/processor_test.go:661-712` | `TestAnnounce_StartsSpan` — the only Announce test in the repo |
| `main.go:429-431` | the duplicated opcode-width block |
| `main.go:646, :651` | `handlerProducer(...)` and the `CreateSocketService` call inside `buildListener` |

### `atlas-login` (module root `services/atlas-login/atlas.com/login`)

Same layout, different lines: `configuration/tenant/rest.go:10-20`; `configuration/registry.go:67-78` (`GetTenantConfig`), `:88-106` (`PublishSnapshot`); `configuration/projection/loop.go:66` (snapshot); `configuration/projection/apply.go:25-30` (`ListenerConfig` — **no `IPAddress`**); `configuration/projection/projection_test.go:84-135`; `configuration/registry_test.go:1-45`; `socket/init.go:37-83` (`socket.Run` list at `:62-70`, `t := tenant.MustFromContext(ctx)` at `:38`); `session/processor.go:209-225` (`Announce`), `:160-172` (`Create`, `WriteHello` at `:167`); `session/model.go:110-112`; `main.go:242-303` (`buildListener`), `:272-274` (width block), `:277` (`hp`), `:300` (`CreateSocketService`).

### `atlas-ui` (working dir `services/atlas-ui`)

| File | Role |
|---|---|
| `src/App.tsx:441-471` | the tenant route subtree — routes are centralized `<Route>` elements, not files |
| `src/pages/TenantsMapleLifePage.tsx:1-72` | the modern page template |
| `src/pages/tenants-properties-form.tsx:24-38, :48-57, :165-184` | Zod schema, `useForm`, and the `<Switch>` inside a `FormField` |
| `src/services/api/tenants.service.ts:76-134` | `TenantConfigAttributes` |
| `src/services/api/tenants.service.ts:305-322` | `updateTenantConfiguration` — shallow-merges a `Partial` over the whole cached attributes and PATCHes the document |
| `src/components/features/tenants/TenantDetailLayout.tsx:22-36` | `sidebarNavItems` |
| `src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx:1-88` | `vi.mock` + `renderAt` helper + the Maple Life nav-item tests |
| `src/pages/__tests__/tenants-mts-config-form.test.tsx:1-94` | the newer `vi.hoisted` + `userEvent` form-page test shape |

---

## Decisions

**Nested `diagnostics.tracePackets`, not a flat boolean.** It matches the tenant document's existing shape (`cashShop`, `mapleLife` are both nested sub-objects backed by their own packages), namespaces a dangerous operational switch away from gameplay settings, and leaves room for the deferred filters (opcode allowlist, direction filter, per-session scope) without a JSON migration. Cost: three small mirror structs and one TypeScript interface.

**Inbound is traced in `libs/atlas-socket`'s `handle`, not in each service's `AdaptHandler`.** This deviates from PRD FR-3.1 and is recorded in design §2 (OQ-2) and §11. The library site wins on every requirement: the opcode is already parsed there (`AdaptHandler` receives only the handler *name*), the unhandled-opcode case (FR-3.4) becomes the same code path instead of a second differently-shaped one, and it additionally traces packets arriving for a session that has already left the registry — which `AdaptHandler`'s `sp.IfPresentById` wrapper silently drops. The handler name is supplied to the library as an `op → name` map from the new `opcodes.BuildHandlerNames`.

**`libs/atlas-opcodes` gains two additive helpers.** PRD §7 expected it unchanged. `BuildHandlerNames` is required by the decision above; `OpCodeSize` / `OpReadWriterFor` exist because the opcode-width rule is currently duplicated verbatim at `atlas-channel/main.go:429-431` and `atlas-login/main.go:272-274`, and the tracer needs it to render the opcode at the right hex width — without lifting it, the tracer becomes a third copy. Both are additive; no signature in the library changes.

**A prerequisite defect is fixed inside this task.** `configuration.PublishSnapshot` has **no production caller at all in `atlas-channel`** — `loop.go` imports the `configuration` package only for the `*configuration.RestModel` type — so `serviceConfig` is permanently nil, `readyCh` is never closed, and every `GetServiceConfig()` caller blocks the full 60 s `readyTimeout` and returns `ErrNotReady`. In `atlas-login` it is called exactly once at `main.go:100`, freezing the snapshot at startup. `registry.go`'s own doc comment already promises the apply-loop call site; it was never written. FR-2.3 ("a change takes effect on the next packet, no restart") cannot hold without it, so this is a prerequisite, not scope creep. Fixing it also repairs `atlas-channel`'s `session/task.go` `GetServiceConfig()` call.

**A package-level non-blocking accessor, not threaded state.** Threading `*projection.State` into the emission sites works for the inbound closure but not for `session.Announce`, which is a package-level curried function invoked from several hundred call sites with a fixed `(l)(ctx)(wp)(name)(encoder)` shape. Widening it would be a repo-scale mechanical change for no benefit over `configuration.TracePacketsEnabled`.

**The outbound opcode is read off the payload, not from a registry.** `writer.MessageGetter` writes it as the first 1-2 bytes; the only unknown is the width, which `opcodes.OpCodeSize` now answers. The alternative — a per-tenant `writerName → opcode` registry — needs a new registry in `atlas-login` (which has no equivalent today) plus registration and eviction lifecycle in both services, all for a value already sitting in byte 0.

**A dedicated Diagnostics page, not the Global Properties tab.** The credential-exposure warning needs room to be prominent; on the properties tab it competes with Region/Major/Minor/Uses PIN and reads as boilerplate. The page maps 1:1 onto the `diagnostics` config subtree, so the deferred filters extend the page rather than bloating properties.

**No seed-data change.** `services/atlas-configurations/seed-data/` contains only `templates/`, and FR-1.7 puts templates out of scope. There is no tenant seed file to edit; the zero value covers every existing tenant row.

---

## Corrections to the design doc, applied in the plan

1. **atlas-ui is Vite + React Router, not Next.js App Router.** The design's OQ-4 implies a `app/tenants/[id]/diagnostics/page.tsx` file-based route. The real repo centralizes `<Route>` elements in `src/App.tsx:441-471` and puts named-export page components under `src/pages/`. Task 9 adds one `<Route>` and one page file, not a directory tree.
2. **Neither consumer mirror carries `cashShop` or `mapleLife` today.** `atlas-channel/configuration/tenant/RestModel` and its login twin stop at `Worlds`; there are no sibling mirror packages for those sub-objects. `Diagnostics` will be the first extra sub-object either mirror carries, so there is no in-service precedent to copy — Task 4's `atlas-configurations` package is the template.
3. **`BuildHandlerMap` lives in `producer.go`, not `config.go`/`registry.go`**, and the opcode parse the design pins at `producer.go:84` is actually at `:77` (`:84` is the map assignment). `BuildHandlerNames`'s home is `producer.go`.
4. **`CreateSocketService` must be widened.** The design says the `op → name` map is "passed in from `main.go`'s `AddBody`", but neither service's `CreateSocketService` takes such a parameter today and the map is not otherwise reachable inside it. Both signatures gain a trailing `names map[uint16]string`. (`AddBody` is also the *type*; the function is `buildListener` in both `main.go` files.)
5. **`atlas-login`'s `Announce` has no `tenant.MustFromContext` call and no OTel span today.** The design says login "adds the same line" — correct, but it is a net-new addition, and the login-side Announce test cannot copy channel's span-stub scaffolding.

---

## Verified non-issues

- **No import cycle** from either service's `socket` or `session` package importing `configuration`: `configuration` imports only `configuration/tenant` and `configuration/task` (`registry.go:4`, `rest.go:4`). `session/task.go` already calls `configuration.GetServiceConfig()`.
- **FR-1.6 falls out for free.** `ComputeOps` diffs only `desired{Key, Cfg}`, and `ListenerConfig` holds no diagnostics field in either service, so a `Diagnostics`-only change produces an identical `desired` map and emits zero ops. Tasks 5 and 6 pin this with a test rather than relying on it.
- **The write path needs no change.** `UpdateById` and `Create` both `json.Marshal` the whole `RestModel` into `Entity.Data` and enqueue `sanitized := input` onto the outbox, so the new field is carried by construction — no per-field enumeration to update.
- **The hex-dump format was validated against the design's own worked example.** Rendering the design §5 sample bytes with the plan's stated algorithm reproduces its two sample lines character for character.

---

## Task sizing

Twelve tasks. Nothing was deliberately left oversized; the two that would have been were split:

- Channel and login emission each split into **a** (inbound: tracer closure, `socket/init.go`, `main.go`) and **b** (outbound: `session/trace.go`, `Announce`, `WriteHello`, tests). Unsplit, each would have touched seven files.
- Tasks 5 and 6 are the same change in two services and are deliberately *not* merged: they are separate modules with separate `go test` roots, and login's `ListenerConfig` differs (no `IPAddress`), so the test fixtures differ.

Task 10 (docs) touches two services. It is documentation only — no code, no build, no tests — so the >1-service warning does not carry the review risk the rule targets.

Task ordering and dependencies:

```
1 (trace pkg) ──┐
2 (socket seam)─┼─→ 7a ─→ 7b      (atlas-channel)
3 (opcodes) ────┤    ↑
4 (configs) ────┤    │
5 (channel cfg)─┴────┘
6 (login cfg) ──────→ 8a ─→ 8b     (atlas-login)
4 ──────────────────→ 9  (atlas-ui)
7b, 8b ─────────────→ 10 (docs)
```

Tasks 1, 2, 3, 4 are mutually independent and can run in any order. Task 5 depends on nothing but is needed by 7a/7b; Task 6 likewise for 8a/8b.
