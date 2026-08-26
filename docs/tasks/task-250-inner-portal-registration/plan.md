# Inner-Portal Registration (`USE_INNER_PORTAL`) — Implementation Plan

Version: v1
Status: Draft
Created: 2026-08-21
PRD: [prd.md](prd.md) · Design: [design.md](design.md) · Context: [context.md](context.md)

---

## Sequencing

```
Task 1 (derive v83/v84/v92 + structures)  ─┬─> Task 3 (codec) ──────────────┐
Task 2 (⬜ columns + threshold)           ─┘                                 │
                                                                             │
Task 4 (data/portal cache + accessors) ─┐                                    │
Task 5 (position registry + session)   ─┼─> Task 8 (portal.EnterInner) ──> Task 9 (handler + main.go)
Task 6 (movement.TeleportCharacter)    ─┘                                    │
Task 7 (atlas-character fh preserve)   ── independent                        │
                                                                             │
Task 10 (seed templates) ──┬─────────────────────────────────────────────────┤
Task 11 (packet-audit wiring) ─────────────────────────────────────────────> Task 12 (export splice, evidence, matrix)
```

Tasks 4, 5, 6, 7, 10, 11 have no dependency on each other and may run in any
order. Tasks 1 and 2 need a live IDA-MCP session and must complete before
Task 3 and Task 8 respectively.

**Matrix-promotion path (design §0 / VERIFYING_A_PACKET §9).** This op is
tier-0 (`status.json` `"tier1": false`). Grading
(`tools/packet-audit/internal/matrix/grade.go:198-210`) promotes a tier-0 cell
to ✅ on **`packet-audit:verify` marker + fresh evidence record with no audit
report**, provided the version registry entry declares `packet:`. That is the
path this plan takes — it avoids generating six audit reports. `matrix --check`
explicitly exempts registry-declared packets from its "dangling evidence"
failure (`tools/packet-audit/cmd/matrix.go:158-184`). If a cell fails to
promote on that path, escalate to the report path (§9 of the playbook) rather
than hand-editing the matrix.

---

## Task 1: Derive `CUserLocal::TryRegisterTeleport` for gms_v83, gms_v84, gms_v92 and record all six structures

Design §1 confirms the field order on gms_v95 (`0x913690`), gms_v87
(`0x9da037`) and jms_v185 (`0xa2218f`) — all three byte-identical. The v83,
v84 and v92 IDBs carry **no `TryRegisterTeleport` symbol**; this task locates
their send sites by caller-walk and records every version's derivation.

### Files

- `docs/tasks/task-250-inner-portal-registration/structures/gms_v83.md` — new file
- `docs/tasks/task-250-inner-portal-registration/structures/gms_v84.md` — new file
- `docs/tasks/task-250-inner-portal-registration/structures/gms_v87.md` — new file
- `docs/tasks/task-250-inner-portal-registration/structures/gms_v92.md` — new file
- `docs/tasks/task-250-inner-portal-registration/structures/gms_v95.md` — new file
- `docs/tasks/task-250-inner-portal-registration/structures/jms_v185.md` — new file
- `docs/tasks/task-250-inner-portal-registration/design.md` — read-only; §1 carries the three confirmed derivations verbatim
- `docs/packets/registry/gms_v83.yaml`, `gms_v84.yaml`, `gms_v87.yaml`, `gms_v92.yaml`, `gms_v95.yaml`, `jms_v185.yaml` — read-only; the registry opcode to cross-check against

Docs-only task. No Go module is touched, so no `go build`/`go test`.

### Steps

- [ ] **Step 1: Adopt the IDBs.** `mcp__ida-pro__idb_list` to enumerate open
      sessions; pin the session id per version on every subsequent call
      (`database` argument). Never rely on "whichever IDB is active".

- [ ] **Step 2: Confirm the three named versions.** For gms_v95, gms_v87 and
      jms_v185, `mcp__ida-pro__func_query "*TryRegisterTeleport*"` and
      `mcp__ida-pro__decompile` the hit. Confirm, against the actual output:
      the `COutPacket::COutPacket(&pkt, N)` constant, and the encode sequence
      `Encode1, EncodeStr, Encode2, Encode2, Encode2, Encode2`.

      | version | expected export address | expected opcode constant |
      |---|---|---|
      | gms_v95 | `0x913690` | `113` (`0x071`) |
      | gms_v87 | `0x9da037` (ctor at `0x9da1c0`) | `104` (`0x068`) |
      | jms_v185 | `0xa2218f` (ctor at `0xa2230e`) | `96` (`0x060`) |

      A disagreement with this table or with the registry opcode is a
      **finding**: record it in the structure doc and stop; do not reconcile
      it silently.

- [ ] **Step 3: Caller-walk gms_v83, gms_v84, gms_v92.** `func_query
      "*TryRegisterTeleport*"` returns empty on these three. Procedure
      (design §7.1):
      1. Locate `CUserLocal::CheckPortal_Collision` (or
         `CPortalList::FindPortalByName`) via `func_query`.
      2. In its decompilation, find the call taking five arguments whose last
         is the literal `1` (`bForced`) and whose 3rd/4th are the portal's
         `pn` / `tn` string pointers — that is `TryRegisterTeleport`.
      3. `decompile` the callee; read the integer pushed into the
         `COutPacket::COutPacket` constructor.
      4. Cross-check that integer against the registry: gms_v83 `101`
         (`0x065`), gms_v84 `101` (`0x065`), gms_v92 `112` (`0x070`). A
         mismatch is a finding, not something to reconcile.
      5. `mcp__ida-pro__rename` the callee to
         `CUserLocal::TryRegisterTeleport` in each IDB, then
         `mcp__ida-pro__idb_save`. Task 12's export splice keys on that name,
         so an unnamed function here blocks that task — name it now (CLAUDE.md
         "producible prerequisite").

- [ ] **Step 4: Write one structure doc per version.** Each file carries a
      `## USE_INNER_PORTAL` section with:

      - the export address of `CUserLocal::TryRegisterTeleport`
      - the address of the `COutPacket` constructor and its opcode constant
      - the ordered field table, quoting the decompiled call per row:

        | # | field | width | client expression |
        |---|---|---|---|
        | 1 | `fieldKey` | `byte` | `Encode1(get_field()->m_bFieldKey)` |
        | 2 | `portalName` | ascii string (u16 len + bytes) | `EncodeStr(sPortalName)` |
        | 3 | `x` | `int16` | `Encode2(this->GetPos()->x)` |
        | 4 | `y` | `int16` | `Encode2(this->GetPos()->y)` |
        | 5 | `targetX` | `int16` | `Encode2(destPortal->ptPos.x)` |
        | 6 | `targetY` | `int16` | `Encode2(destPortal->ptPos.y)` |

      - an explicit **per-version delta** line: either "no delta vs gms_v95"
        or the exact divergence found.

- [ ] **Step 5: Record the gate decision.** If every one of the six versions
      is byte-identical, state in each doc that **no `MajorAtLeast` gate is
      required** and why. If any version diverges, state which field, from
      which major version, and which `MajorAtLeast(N)` boundary Task 3 must
      use. A raw `> N` comparison is not acceptable (PRD FR-2.3).

---

## Task 2: Re-derive the ⬜ columns and derive the entry-distance threshold

PRD FR-6.2 / Q9.4 forbid inheriting the ⬜ by assumption. PRD FR-4.7 / Q9.3
require the distance threshold to be a documented, derived constant rather
than a chosen number.

### Files

- `docs/tasks/task-250-inner-portal-registration/version-coverage.md` — new file
- `docs/tasks/task-250-inner-portal-registration/coverage-manifest.yaml` — new file
- `docs/tasks/task-250-inner-portal-registration/structures/gms_v95.md` — created by Task 1; append the `## Threshold derivation` section
- `docs/packets/MapleStory Ops - ServerBound.csv` — read-only; line 237 is the `USE_INNER_PORTAL` row
- `docs/packets/audits/support/gms_v48.md` — read-only; line 798 is the current `n-a` record (siblings: `gms_v61.md`, `gms_v72.md`, `gms_v79.md`)
- `docs/packets/registry/gms_v48.yaml`, `gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml` — read-only

Docs-only task. No Go module is touched.

### Steps

- [ ] **Step 1: Record the gms_v12 finding.** The ServerBound CSV has a
      `GMS v12` column holding `0x000` — the absent sentinel — on the
      `USE_INNER_PORTAL` row. Quote the CSV line in `version-coverage.md` and
      record gms_v12 as **confirmed absent**:
      `services/atlas-configurations/seed-data/templates/template_gms_12_1.json`
      gets no route.

- [ ] **Step 2: Derive v48 / v61 / v72 / v79.** These four have no CSV column
      *and* no registry entry — absence is currently inferred, not derived.
      For each: `mcp__ida-pro__idb_open` the IDB (`GMS_v48_1_DEVM.exe.i64`,
      `GMS_v61.1_U_DEVM.exe.i64`, `GMS_v72.1_U_DEVM.exe.i64`,
      `GMS_v79_1_DEVM.exe.i64`), `func_query "*TryRegisterTeleport*"`, and on
      a miss run the Task 1 Step 3 caller-walk from
      `CheckPortal_Collision` / `FindPortalByName`.

      Record per version, in `version-coverage.md`: the IDB session id, the
      queries run, and the outcome — **absent** (with the evidence that makes
      it absent: no send site reachable from the portal-collision path) or
      **present** (with the export address and the `COutPacket` opcode).

      If any version is **present**, it joins the Task 1 / Task 10 / Task 12
      in-scope list in full — registry entry, template route, codec coverage,
      fixture — and this plan must be amended before those tasks run. Say so
      explicitly in `version-coverage.md`.

- [ ] **Step 3: Derive the entry-distance threshold.** Decompile
      `CUserLocal::CheckPortal_Collision` (gms_v95 `0x919a10`) and read the
      portal collision-rect half-extents the client tests the avatar's
      position against before it calls `TryRegisterTeleport`. Append to
      `structures/gms_v95.md`:

      ```
      ## Threshold derivation

      Collision rect half-extents (CUserLocal::CheckPortal_Collision @0x919a10):
        halfWidth  = <derived>
        halfHeight = <derived>
      Diagonal bound: ceil(sqrt(halfWidth^2 + halfHeight^2)) = <derived>
      Movement-latency margin: <stated value> map units, because <stated reason>
      maxPortalEntryDistance = <integer>   # map coordinate units
      ```

      The final line MUST be a single integer. Task 8 copies that integer into
      the Go constant verbatim and cites this section — so leaving it
      unresolved blocks Task 8.

- [ ] **Step 4: Write `coverage-manifest.yaml`.** Declare the op × version
      cells this task claims, for `packet-completeness-critic` to diff
      against. Include only the versions Steps 1–2 confirm as present. The
      shape is `ops` / `versions` / `fields`, copied from
      `docs/tasks/task-230-scripted-items/coverage-manifest.yaml`:

      ```yaml
      # coverage-manifest
      ops:
        - USE_INNER_PORTAL
      versions:
        - gms_v83
        - gms_v84
        - gms_v87
        - gms_v92
        - gms_v95
        - jms_v185
      fields:
        - "portal/serverbound/PortalInnerPortal: <the gate finding Task 1 Step 5 recorded>"
      ```

      Record the gms_v12 / v48 / v61 / v72 / v79 absences in
      `version-coverage.md`, not in the manifest — the manifest declares
      claimed cells only.

      The packet id is `portal/serverbound/PortalInnerPortal`, not
      `.../InnerPortal`: `qualifiedWriterName`
      (`tools/packet-audit/cmd/run.go:223-228`) prepends the title-cased
      package to the struct name whenever the candidate carries a `pkg`, the
      same way struct `Script` in package `portal` yields
      `portal/serverbound/PortalScript`. Design §2's "matrix codec identifier
      becomes `portal/serverbound/InnerPortal`" is superseded by this.

---

## Task 3: `InnerPortal` codec in `libs/atlas-packet`

### Files

- `libs/atlas-packet/portal/serverbound/inner_portal.go` — new file
- `libs/atlas-packet/portal/serverbound/inner_portal_test.go` — new file
- `libs/atlas-packet/portal/serverbound/script.go` — read-only; the sibling codec to copy shape from
- `libs/atlas-packet/portal/serverbound/script_test.go` — read-only; round-trip test shape
- `docs/tasks/task-250-inner-portal-registration/structures/gms_v95.md` — new file, created read-only by Task 1; the derived field order

Module root: `libs/atlas-packet`. Run `go build ./... && go test ./...` there.

### Steps

- [ ] **Step 1: Write the failing test.**

`inner_portal_test.go`, package `serverbound`. Two test funcs. Imports,
`t.Run` wrappers and `pt` helpers are exactly as in `script_test.go:1-10`.

**`TestInnerPortalGoldenBytes`** — table-driven over the six in-scope
versions. All six are byte-identical (design §1.3), so every row expects the
same 13 bytes; the rows exist so a future divergence fails loudly on the
version that diverges.

Fixture value (shared by every row):
`InnerPortal{fieldKey: 1, portalName: "sp", x: 100, y: 200, targetX: 300, targetY: -50}`

Expected bytes, per field (`WriteAsciiString` is a `uint16` LE length followed
by the encoded bytes — `libs/atlas-socket/response/writer.go:82-92`):

| offset | bytes | field |
|---|---|---|
| 0 | `0x01` | `fieldKey` = 1 (`Encode1`) |
| 1–2 | `0x02 0x00` | `portalName` length 2 (LE u16) |
| 3–4 | `0x73 0x70` | `"sp"` |
| 5–6 | `0x64 0x00` | `x` = 100 (LE i16) |
| 7–8 | `0xC8 0x00` | `y` = 200 |
| 9–10 | `0x2C 0x01` | `targetX` = 300 |
| 11–12 | `0xCE 0xFF` | `targetY` = -50 |

Full expected slice:
`[]byte{0x01, 0x02, 0x00, 0x73, 0x70, 0x64, 0x00, 0xC8, 0x00, 0x2C, 0x01, 0xCE, 0xFF}`

| subtest name | context | expected |
|---|---|---|
| `gms_v83` | `pt.CreateContext("GMS", 83, 1)` | the 13 bytes above |
| `gms_v84` | `pt.CreateContext("GMS", 84, 1)` | the 13 bytes above |
| `gms_v87` | `pt.CreateContext("GMS", 87, 1)` | the 13 bytes above |
| `gms_v92` | `pt.CreateContext("GMS", 92, 1)` | the 13 bytes above |
| `gms_v95` | `pt.CreateContext("GMS", 95, 1)` | the 13 bytes above |
| `jms_v185` | `pt.CreateContext("JMS", 185, 1)` | the 13 bytes above |

Use `pt.Encode(t, ctx, input.Encode, nil)` (`libs/atlas-packet/test`) to
produce the bytes and `bytes.Equal` to compare. Then decode the same 13 bytes
back through `(&InnerPortal{}).Decode` and assert every accessor
field-by-field: `FieldKey() == 1`, `PortalName() == "sp"`, `X() == 100`,
`Y() == 200`, `TargetX() == 300`, `TargetY() == -50`.

**`TestInnerPortalRoundTrip`** — `for _, v := range pt.Variants`, subtest
`v.Name`, exactly the loop in `script_test.go:12-36`. Same fixture. Assert via
`pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)` (which already fails
on unconsumed trailing bytes, satisfying FR-2.4's no-trailing-byte tolerance)
plus the six field-by-field equality checks above.

Do **not** add `// packet-audit:verify` markers yet — Task 12 adds them once
the evidence records exist. A marker with no evidence downgrades the cell to
`incomplete` with note "byte-test marker present but no fresh evidence record"
(`grade.go:211-218`).

- [ ] **Step 2: Implement the codec.**

`inner_portal.go`, package `serverbound`, structurally a copy of `script.go`
with six fields:

```go
const InnerPortalHandle = "InnerPortalHandle"

// InnerPortal - the client's in-map ("inner") portal teleport registration.
// The client performs the move locally and reports it; fields 3/4 are where
// the character stood BEFORE the teleport and fields 5/6 are what the
// CLIENT's own WZ says the destination portal's position is. Neither is
// adopted as authoritative — see services/atlas-channel/.../portal.EnterInner.
// packet-audit:fname CUserLocal::TryRegisterTeleport
type InnerPortal struct {
	fieldKey   byte
	portalName string
	x          int16
	y          int16
	targetX    int16
	targetY    int16
}
```

Accessors `FieldKey() byte`, `PortalName() string`, `X() int16`, `Y() int16`,
`TargetX() int16`, `TargetY() int16` — all value receivers.
`Operation() string` returns `InnerPortalHandle`. `String()` in the
`script.go:44` format, extended with `targetX`/`targetY`.

`Encode` writes, in order: `WriteByte(fieldKey)`, `WriteAsciiString(portalName)`,
`WriteInt16(x)`, `WriteInt16(y)`, `WriteInt16(targetX)`, `WriteInt16(targetY)`.
`Decode` reads the mirror sequence with `ReadByte`, `ReadAsciiString`,
`ReadInt16` ×4. Both are unconditional — no version branch — unless Task 1
Step 5 recorded a delta, in which case gate that field with the repo's
`MajorAtLeast` idiom and cite the structure doc in a comment.

- [ ] **Step 3: Verify.** `go build ./... && go test ./...` from
      `libs/atlas-packet`.

---

## Task 4: `data/portal` — model accessors and a per-map tenant-scoped cache

Two defects block the design as written. `data/portal.Model` exposes only
`Id()` — `Target()`, `TargetMapId()`, `X()` and `Y()` do not exist. And
`GetInMapByName` is an uncached REST GET per call
(`data/portal/requests.go:12-13`), while `EnterInner` needs two lookups per
packet and inner portals fire in bursts (PRD NFR-Performance).

### Files

- `services/atlas-channel/atlas.com/channel/data/portal/model.go` — add the missing accessors
- `services/atlas-channel/atlas.com/channel/data/portal/requests.go` — add the whole-list request
- `services/atlas-channel/atlas.com/channel/data/portal/processor.go` — add the cache; `GetInMapByName` filters it
- `services/atlas-channel/atlas.com/channel/data/portal/processor_test.go` — new file
- `services/atlas-channel/atlas.com/channel/data/map/processor.go` — read-only; lines 34-73 are the cache pattern to copy
- `services/atlas-channel/atlas.com/channel/data/portal/mock/processor.go` — read-only; the `Processor` interface is unchanged, so this file needs no edit

Module root: `services/atlas-channel/atlas.com/channel`. Run `go build ./... && go test ./...` there.

### Steps

- [ ] **Step 1: Write the failing test.**

`processor_test.go`, package `portal`. The cache is package-level state, so
each test must key on a fresh `tenant`/`mapId` pair to stay independent —
mirror how `data/map` tests isolate (or generate a fresh `uuid.New()` tenant
per test).

Stub the REST leg by pointing the request at an `httptest.Server` through the
tenant/service-URL context the `libs/atlas-rest/requests` pipeline reads.
Patterns to copy: whatever `data/map` or a sibling `data/*` package already
uses to stand up a fake `atlas-data`; if no such helper exists in this module,
introduce the seam as a package-level `var requestInMapFn = requestInMap` and
swap it in the test (precedent: `monsterByIdFn`,
`services/atlas-channel/atlas.com/channel/movement/processor.go:133-136`).

| test func | scenario | assertion |
|---|---|---|
| `TestGetInMapByName_CachesWholeList` | two `GetInMapByName` calls, same tenant + mapId, different portal names | exactly **1** REST fetch; both calls return the matching portal |
| `TestGetInMapByName_TenantScoped` | same mapId, two distinct tenants | **2** REST fetches; neither tenant sees the other's data |
| `TestGetInMapByName_NotFound` | cached list has no portal with that name | returns a non-nil error; performs no second fetch |
| `TestModelAccessors` | `Extract` a `RestModel{Name:"sp", Target:"tp", Type:2, X:100, Y:200, TargetMapId:104040000, ScriptName:""}` with `Id:"7"` | `Id()==7`, `Name()=="sp"`, `Target()=="tp"`, `Type()==2`, `X()==100`, `Y()==200`, `TargetMapId()==104040000`, `ScriptName()==""` |

- [ ] **Step 2: Add the accessors.** In `model.go`, add value-receiver
      accessors for every field the struct already carries: `Name()`,
      `Target()`, `Type() uint8`, `X() int16`, `Y() int16`,
      `TargetMapId() _map.Id`, `ScriptName()`. Field names and types are
      already fixed by `Extract` in `data/portal/rest.go`.

- [ ] **Step 3: Add the whole-list request.** In `requests.go`, alongside the
      existing `portalsInMap` / `portalsByName` constants, add:

      ```go
      func requestInMap(ctx context.Context, mapId _map.Id) requests.Request[[]RestModel]
      ```

      built from `portalsInMap` (`data/maps/%d/portals`) exactly as
      `requestInMapByName` is built from `portalsByName`. `atlas-data`
      already serves the unfiltered list — `get_map_portals` at
      `services/atlas-data/atlas.com/data/map/resource.go:37` (line 36 is the
      `?name=` variant the current code uses).

- [ ] **Step 4: Cache the list.** In `processor.go`, copy the cache shape from
      `data/map/processor.go:34-73` verbatim in structure — a `cacheKey{tenantId
      uuid.UUID; mapId _map.Id}`, a `sync.Map` value cache, a `sync.Map` of
      per-key load mutexes, double-checked load — but cache `[]Model` (the
      whole map's portal list) rather than one model. Carry the same
      "static WZ data, cached for the process lifetime, pod restart is the
      invalidation contract" comment, naming portals.

      `GetInMapByName` becomes: load the cached list, filter by `Name()`,
      return the first match or an error naming the map and the portal.
      `InMapByNameModelProvider` keeps its signature and returns
      `model.FixedProvider` over the filtered slice — the `Processor`
      interface does not change, so `data/portal/mock` and every existing
      caller compile untouched.

- [ ] **Step 5: Verify.** `go build ./... && go test ./...` from the module root.

---

## Task 5: `position` — the process-local last-known-position registry

PRD FR-4.4 wants the character's *last known* position. `session.Model` has no
coordinates and `character.Processor.GetById` is a REST call, which
NFR-Performance forbids on the socket read path. The channel already computes
an authoritative `(x, y)` in the movement fold; this task gives that value a
home.

The registry lives in its own package rather than in `movement` because
`session` must clear it on destroy and `movement` already imports `session` —
putting it in `movement` would create an import cycle.

### Files

- `services/atlas-channel/atlas.com/channel/position/registry.go` — new file
- `services/atlas-channel/atlas.com/channel/position/registry_test.go` — new file
- `services/atlas-channel/atlas.com/channel/session/processor.go` — clear on destroy
- `services/atlas-channel/atlas.com/channel/session/position_hook_test.go` — new file
- `services/atlas-channel/atlas.com/channel/character/chakra/registry.go` — read-only; the registry + `GetRegistry()` singleton pattern to copy
- `services/atlas-channel/atlas.com/channel/session/aran_combo_hook_test.go` — read-only; the destroy-hook test shape to copy

Module root: `services/atlas-channel/atlas.com/channel`.

### Steps

- [ ] **Step 1: Write the failing tests.**

`position/registry_test.go`, package `position`:

| test func | scenario | assertion |
|---|---|---|
| `TestRegistry_PutThenLookup` | `Put(t, 42, Position{X: 100, Y: 200})` then `Lookup(t, 42)` | returns `Position{100, 200}`, `ok == true` |
| `TestRegistry_LookupMiss` | `Lookup(t, 99)` with nothing stored | zero `Position`, `ok == false` |
| `TestRegistry_TenantIsolated` | `Put` for tenant A id 42; `Lookup` for tenant B id 42 | `ok == false` |
| `TestRegistry_PutOverwrites` | two `Put`s for the same key | `Lookup` returns the second value |
| `TestRegistry_Clear` | `Put` then `Clear(t, 42)` then `Lookup` | `ok == false` |

Tenant construction: copy `aranHookTestTenant` from
`session/aran_combo_hook_test.go` (a `tenant.Create(uuid.New(), "GMS", 83, 1)`
helper).

`session/position_hook_test.go`, package `session` — copy the file shape of
`aran_combo_hook_test.go` exactly:

| test func | scenario | assertion |
|---|---|---|
| `TestClearLastPositionOnDestroy_NonZeroCharacter_ClearsState` | `position.GetRegistry().Put(t, 42, ...)`, then `clearLastPositionOnDestroy(ctx, 42)` | `Lookup(t, 42)` returns `ok == false` |
| `TestClearLastPositionOnDestroy_ZeroCharacter_NoOp` | state stored for id 42; call with `characterId == 0` | id 42's entry survives |

- [ ] **Step 2: Implement the registry.** `position/registry.go`, package
      `position`. Copy the singleton + mutex shape from
      `character/chakra/registry.go:29-69`:

      ```go
      // Key scopes a last-known position to one character in one tenant.
      type Key struct {
          Tenant      tenant.Model
          CharacterId uint32
      }

      // Position is the authoritative (x, y) the channel last folded out of a
      // movement path, or wrote on an inner-portal teleport.
      type Position struct {
          X int16
          Y int16
      }

      type Registry struct {
          mutex   sync.RWMutex
          entries map[Key]Position
      }

      func GetRegistry() *Registry
      func (r *Registry) Put(t tenant.Model, characterId uint32, p Position)
      func (r *Registry) Lookup(t tenant.Model, characterId uint32) (Position, bool)
      func (r *Registry) Clear(t tenant.Model, characterId uint32)
      ```

      The package imports only `sync` and `libs/atlas-tenant` — nothing from
      `atlas-channel` — which is what keeps `session` free to import it.
      Document that in the package comment, because it is load-bearing.

      No TTL and no sweeper: entries are bounded by the characters connected
      to this pod and are removed on session destroy (Step 3). Say so in the
      `Registry` doc comment.

- [ ] **Step 3: Clear on session destroy.** In `session/processor.go`, add
      `clearLastPositionOnDestroy(ctx context.Context, characterId uint32)`
      immediately after `clearAranComboOnDestroy` (line 459), with the same
      `if characterId != 0` guard and the same doc-comment framing, and call
      it from `Destroy` beside the existing `clearAranComboOnDestroy(p.ctx,
      s.CharacterId())` call at line 417.

- [ ] **Step 4: Verify.** `go build ./... && go test ./...` from the module root.

---

## Task 6: `movement.TeleportCharacter` and the last-position write

### Files

- `services/atlas-channel/atlas.com/channel/movement/processor.go` — add `TeleportCharacter` to the interface and impl; write the position registry in `ForCharacter`
- `services/atlas-channel/atlas.com/channel/movement/mock/processor.go` — add `TeleportCharacterFunc`
- `services/atlas-channel/atlas.com/channel/movement/teleport_test.go` — new file
- `services/atlas-channel/atlas.com/channel/movement/producer.go` — read-only; `CommandProducer` is the emit used unchanged
- `services/atlas-channel/atlas.com/channel/position/registry.go` — new file, created read-only by Task 5

Module root: `services/atlas-channel/atlas.com/channel`.

### Steps

- [ ] **Step 1: Write the failing test.**

`movement/teleport_test.go`, package `movement`. Tenant + field helpers:
copy `newMovementTestTenant` / `movementTestField` from
`movement/processor_test.go` (same package — call them directly rather than
redefining).

| test func | scenario | assertion |
|---|---|---|
| `TestTeleportCharacter_WritesLastPosition` | `TeleportCharacter(f, 42, 300, -50)` | `position.GetRegistry().Lookup(t, 42)` returns `Position{300, -50}`, `ok == true` |
| `TestTeleportCharacter_NoClientboundBroadcast` | same call, with the writer.Producer set to a spy that records every announce | spy recorded **zero** announces |
| `TestForCharacter_WritesLastPosition` | `ForCharacter(f, 42, mv)` where `mv` folds to `(150, 250)` — build `mv` the way `movement/fold_test.go` builds one | `Lookup(t, 42)` returns `Position{150, 250}` |

`TeleportCharacter` publishes to Kafka in a `routine.Go`, so the two
position-registry assertions must wait for the write. Prefer making the
registry write **synchronous** — before the `routine.Go` — so the test needs
no polling; that is also the correct ordering, because `EnterInner`'s next
plausibility check should see the teleport immediately.

- [ ] **Step 2: Implement.**

Add to the `Processor` interface (`movement/processor.go:38-43`) and to
`ProcessorImpl`:

```go
// TeleportCharacter publishes an authoritative position for a character that
// relocated without a movement path — an inner portal. It emits the SAME
// COMMAND_TOPIC_CHARACTER_MOVEMENT command ForCharacter emits, so
// atlas-character remains the single position authority, and it emits NO
// clientbound broadcast: the client performed the teleport locally and its
// next MOVE carries the TELEPORT element that relays it to the field
// (design §4.4).
//
// fh is published as 0 — portal data carries no foothold and inventing one is
// not an option. atlas-character preserves the stored foothold on a zero fh
// (see services/atlas-character/.../character/processor.go Move).
TeleportCharacter(f field.Model, characterId uint32, x int16, y int16) error
```

Implementation: write `position.GetRegistry().Put(p.t, characterId,
position.Position{X: x, Y: y})` synchronously, then a single `routine.Go`
publishing
`producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandCharacterMovement)(CommandProducer(f, uint64(characterId), characterId, x, y, 0, 0))`
— the same call `ForCharacter` makes at `processor.go:71`, with `fh` and
`stance` zero. Log the publish failure exactly as `ForCharacter` does.

In `ForCharacter`'s second `routine.Go`, after the successful fold and before
the publish, add the same `position.GetRegistry().Put(p.t, characterId,
position.Position{X: ms.X, Y: ms.Y})`.

Add `TeleportCharacterFunc func(f field.Model, characterId uint32, x int16, y int16) error`
plus its nil-guarded method to `movement/mock/processor.go`, following that
file's existing four methods exactly.

- [ ] **Step 3: Verify.** `go build ./... && go test ./...` from the module root.

---

## Task 7: `atlas-character` — preserve the stored foothold on a zero-`fh` movement command

`temporalRegistry.Update` writes `fh` unconditionally
(`character/temporal_data.go:77-79`), so Task 6's `fh = 0` would clobber a
real foothold. The channel's own fold already states the correct rule —
"only when non-zero so we don't trample the spawn-time fh"
(`services/atlas-channel/.../movement/processor.go:288-294`) — and this task
applies it at the consumer so `Fh == 0` means "no foothold information" on
both sides of the topic.

This service is independent of every other task; it may run first.

### Files

- `services/atlas-character/atlas.com/character/character/temporal_data.go` — give `UpdatePosition` a `stance` parameter
- `services/atlas-character/atlas.com/character/character/processor.go` — `Move` routes on `fh == 0`
- `services/atlas-character/atlas.com/character/character/temporal_position_test.go` — new file
- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` — read-only; `handleMovementEvent` at line 409 calls `Move` and is unchanged

Module root: `services/atlas-character/atlas.com/character`.

### Steps

- [ ] **Step 1: Write the failing test.**

`character/temporal_position_test.go`, package `character`. The registry is a
process singleton reached via `GetTemporalRegistry()`; use a fresh
`uuid.New()` tenant per test so cases stay independent. Builder/tenant setup:
copy from an existing test in this package that touches
`GetTemporalRegistry()`; if none does, construct the tenant with
`tenant.Create(uuid.New(), "GMS", 83, 1)` and a `tenant.WithContext` context,
as `services/atlas-channel/.../session/aran_combo_hook_test.go` does.

| test func | seed state | call | assertion |
|---|---|---|---|
| `TestMove_ZeroFh_PreservesStoredFoothold` | `Update(ctx, t, 42, 10, 20, 77, 3)` | `Move(42, 300, -50, 0, 5)` | stored `x==300`, `y==-50`, **`fh==77`**, `stance==5` |
| `TestMove_NonZeroFh_OverwritesFoothold` | `Update(ctx, t, 42, 10, 20, 77, 3)` | `Move(42, 300, -50, 88, 5)` | stored `x==300`, `y==-50`, `fh==88`, `stance==5` |
| `TestMove_ZeroFh_NoPriorState_StoresZeroFh` | nothing stored for id 43 | `Move(43, 5, 6, 0, 1)` | stored `x==5`, `y==6`, `fh==0`, `stance==1` |

Read stored state back with `GetTemporalRegistry().GetById(ctx, t, id)` and
its `X()`, `Y()`, `Fh()`, `Stance()` accessors
(`character/temporal_data.go`).

- [ ] **Step 2: Implement.**

`UpdatePosition` (`temporal_data.go:73-76`) currently takes `(ctx, t,
characterId, x, y)` and has **no callers** — grep confirms the only reference
is its own definition. Give it a `stance byte` parameter and keep preserving
`existing.fh`:

```go
// UpdatePosition writes an authoritative (x, y, stance) while PRESERVING the
// stored foothold. Used for movement commands that carry fh == 0 — an
// inner-portal teleport publishes no foothold because portal data has none,
// and a zero there means "no foothold information", not "foothold zero".
// Mirrors the channel-side fold rule (atlas-channel movement/processor.go).
func (r *temporalRegistry) UpdatePosition(ctx context.Context, t tenant.Model, characterId uint32, x int16, y int16, stance byte)
```

Then in `processor.go:876-879`:

```go
func (p *ProcessorImpl) Move(characterId uint32, x int16, y int16, fh int16, stance byte) error {
	t := tenant.MustFromContext(p.ctx)
	if fh == 0 {
		GetTemporalRegistry().UpdatePosition(p.ctx, t, characterId, x, y, stance)
		return nil
	}
	GetTemporalRegistry().Update(p.ctx, t, characterId, x, y, fh, stance)
	return nil
}
```

- [ ] **Step 3: Verify.** `go build ./... && go test ./...` from
      `services/atlas-character/atlas.com/character`.

---

## Task 8: `portal.EnterInner` — validation and position adoption

### Files

- `services/atlas-channel/atlas.com/channel/portal/processor.go` — add `EnterInner`, the threshold constant, and the movement seam
- `services/atlas-channel/atlas.com/channel/portal/mock/processor.go` — add `EnterInnerFunc`
- `services/atlas-channel/atlas.com/channel/portal/processor_test.go` — new file
- `services/atlas-channel/atlas.com/channel/data/portal/mock/processor.go` — read-only; the portal-data mock the tests inject
- `services/atlas-channel/atlas.com/channel/movement/mock/processor.go` — read-only; the movement mock the tests inject (Task 6)
- `services/atlas-channel/atlas.com/channel/position/registry.go` — new file, created read-only by Task 5; the last-known-position read
- `docs/tasks/task-250-inner-portal-registration/structures/gms_v95.md` — new file, created read-only by Task 2; `## Threshold derivation` supplies the constant's integer

Module root: `services/atlas-channel/atlas.com/channel`.

### Steps

- [ ] **Step 1: Write the failing test.**

`portal/processor_test.go`, package `portal`. Construct `ProcessorImpl`
directly (same package) with `pd` set to a `data/portal/mock.ProcessorMock`
and the movement seam swapped to a `movement/mock.ProcessorMock` that records
its `TeleportCharacter` arguments.

Shared fixture for every case unless the row overrides it:

- field: `MapId() == 104040000`, channel/world from a helper copied from
  `session/aran_combo_hook_test.go`'s `aranHookTestField`
- character id `42`
- source portal `"sp"`: `x=100, y=200, target="tp", targetMapId=104040000`
- destination portal `"tp"`: `x=300, y=-50, target="sp", targetMapId=104040000`
- last-known position seeded via `position.GetRegistry().Put(t, 42,
  position.Position{X: 100, Y: 200})` (exactly on the source portal)
- packet claim: `claimedX=100, claimedY=200, claimedTargetX=300,
  claimedTargetY=-50`

`TestEnterInner` — table-driven, one subtest per row. Every refusal row
asserts **`TeleportCharacter` was not called**.

| subtest | mutation from the fixture | expect |
|---|---|---|
| `source portal unresolvable` | `GetInMapByNameFunc` returns an error for `"sp"` | refused |
| `source targetMapId is the sentinel` | source portal `targetMapId = map.EmptyMapId` | refused |
| `source targetMapId is a different map` | source portal `targetMapId = 104040001` | refused |
| `source target empty` | source portal `target = ""` | refused |
| `destination portal unresolvable` | `GetInMapByNameFunc` errors for `"tp"` | refused |
| `last known position beyond threshold` | registry seeded with `Position{X: 100 + maxPortalEntryDistance + 1, Y: 200}` | refused |
| `last known position at the threshold` | registry seeded with `Position{X: 100 + maxPortalEntryDistance, Y: 200}` | accepted |
| `claimed target disagrees with destination portal` | `claimedTargetX = 999` | refused |
| `last-position registry miss` | nothing seeded for character 42 | **accepted** (design §5.5: a miss must never refuse) |
| `happy path adopts server coordinates` | `claimedX=9999, claimedY=9999` (deliberately absurd, registry miss so the entry check is skipped) | `TeleportCharacter(f, 42, 300, -50)` — the **destination portal's** coordinates, never the packet's |

Every row returns `nil` from `EnterInner` (see Step 2). Refusal is asserted
via the mock, not the return value. To also assert the warning was logged, use
`github.com/sirupsen/logrus/hooks/test`'s `NewNullLogger()` and inspect
`hook.LastEntry().Level == logrus.WarnLevel` — the same helper
`libs/atlas-packet/test` already uses.

- [ ] **Step 2: Implement.**

Add to the `Processor` interface and `ProcessorImpl`:

```go
// EnterInner registers an inner-portal teleport: the client already moved
// itself inside the current field and reported it. Every coordinate the
// server adopts comes from ITS OWN portal data — the packet's claim is used
// only for plausibility comparison and logging (PRD FR-4.5).
//
// A refusal is a deliberate no-op, not a handler failure: it logs at warning,
// updates nothing, and returns nil. The character's next MOVE re-establishes
// position exactly as it does today, so a false positive degrades to current
// behaviour rather than to a broken portal (PRD FR-4.6). The only non-nil
// error this returns is one from the position publish itself.
EnterInner(f field.Model, characterId uint32, sourcePortalName string,
	claimedX int16, claimedY int16, claimedTargetX int16, claimedTargetY int16) error
```

Constant, placed at the top of `portal/processor.go`:

```go
// maxPortalEntryDistance bounds how far the character's last known position
// may be from the source portal for an inner-portal claim to be plausible.
// Unit: map coordinate units — the same units as portal x/y and character
// x/y. The value is DERIVED from the client's own portal collision rect, not
// chosen: see docs/tasks/task-250-inner-portal-registration/structures/gms_v95.md
// "## Threshold derivation". Do not change it without redoing that derivation.
const maxPortalEntryDistance = /* the integer recorded by Task 2 */
const maxPortalEntryDistanceSq = maxPortalEntryDistance * maxPortalEntryDistance
```

Movement seam, so `EnterInner` does not force a `writer.Producer` through the
six existing `portal.NewProcessor(l, ctx)` call sites:

```go
// newMovementProcessor is the movement seam for EnterInner. EnterInner's only
// movement call is TeleportCharacter, which publishes a Kafka command and
// emits no clientbound packet, so the writer.Producer a movement Processor
// carries is unused on this path — hence the nil. Package-level var so portal
// tests can inject movement/mock without a live producer (precedent: warpFunc
// in socket/handler/mystic_door_enter.go, monsterByIdFn in
// movement/processor.go).
var newMovementProcessor = func(l logrus.FieldLogger, ctx context.Context) movement.Processor {
	return movement.NewProcessor(l, ctx, nil)
}
```

`EnterInner` body, in this order — each refusal is `Warnf` + `return nil`:

1. `sp, err := p.pd.GetInMapByName(f.MapId(), sourcePortalName)`; on error
   warn with character id, `f.MapId()`, and the unresolved name (FR-4.2).
2. `if sp.TargetMapId().IsSentinel() || sp.TargetMapId() != f.MapId()` → warn
   with both map ids and refuse (FR-4.3). Use
   `_map.Id.IsSentinel()` (`libs/atlas-constants/map/model.go:39`); do not
   write the `999999999` literal.
3. `if sp.Target() == ""` → warn and refuse; a portal with no `tn` has no
   destination.
4. `dp, err := p.pd.GetInMapByName(f.MapId(), sp.Target())`; on error warn and
   refuse.
5. Entry proximity, **only when the registry has an entry** — a miss falls
   through without refusing (design §5.5):
   ```go
   if last, ok := position.GetRegistry().Lookup(tenant.MustFromContext(p.ctx), characterId); ok {
       dx, dy := int32(last.X)-int32(sp.X()), int32(last.Y)-int32(sp.Y())
       if dx*dx+dy*dy > maxPortalEntryDistanceSq { /* warn + refuse */ }
   }
   ```
   Widen to `int32` before squaring — `int16` products overflow. The warning
   carries character id, field, portal name, last known position, portal
   position, the computed squared distance and the threshold (FR-4.4).
6. `if claimedTargetX != dp.X() || claimedTargetY != dp.Y()` → warn and refuse
   (design §5.4(ii)). Log both pairs.
7. Accept: `Debugf` the acceptance (debug, not warn — a player pacing through
   a portal must not flood the log), then
   `return newMovementProcessor(p.l, p.ctx).TeleportCharacter(f, characterId, dp.X(), dp.Y())`.

Add `EnterInnerFunc` plus its nil-guarded method to
`portal/mock/processor.go`. That file's `var _ portal.Processor` assertion
means the mock must gain the method or the package stops compiling; also add
`WarpToPortalFunc` if the assertion already fails for it.

- [ ] **Step 3: Verify.** `go build ./... && go test ./...` from the module root.

---

## Task 9: `atlas-channel` handler and `handlerMap` registration

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/portal_inner.go` — new file
- `services/atlas-channel/atlas.com/channel/main.go` — register the handle beside line 901
- `services/atlas-channel/atlas.com/channel/socket/handler/portal_script.go` — read-only; the handler shape to copy

Module root: `services/atlas-channel/atlas.com/channel`.

### Steps

- [ ] **Step 1: Write the handler.** `portal_inner.go`, a direct analogue of
      `portal_script.go`:

      ```go
      func InnerPortalHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
          return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
              p := portal2.InnerPortal{}
              p.Decode(l, ctx)(r, readerOptions)
              l.Debugf("[%s] read [%s]", p.Operation(), p.String())

              _ = portal.NewProcessor(l, ctx).EnterInner(s.Field(), s.CharacterId(),
                  p.PortalName(), p.X(), p.Y(), p.TargetX(), p.TargetY())
          }
      }
      ```

      Tenant comes from `ctx` inside the processor; nothing is derived from
      packet contents (PRD FR-3.4).

- [ ] **Step 2: Register.** In `main.go`, beside
      `handlerMap[portal2.PortalScriptHandle] = handler.PortalScriptHandleFunc`
      (line 901), add
      `handlerMap[portal2.InnerPortalHandle] = handler.InnerPortalHandleFunc`.

- [ ] **Step 3: Verify.** `go build ./... && go test ./...` from the module root.

---

## Task 10: Route the opcode in the six in-scope seed templates

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` — `0x65`
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` — `0x65`
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` — `0x68`
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — `0x70`
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — `0x71`
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — `0x60`
- `docs/packets/TEMPLATE_CONVENTIONS.md` — read-only; the sorted-opcode rule

The same mechanical edit six times; no Go module is touched.

### Steps

- [ ] **Step 1: Insert each route.** Into each file's `socket.handlers` array,
      at its **sorted `opCode` position** (ascending; enforced by
      `tools/template-opcode-order-guard.sh`), insert:

      ```json
      {
        "opCode": "<per-file opcode above>",
        "validator": "LoggedInValidator",
        "handler": "InnerPortalHandle",
        "fname": "CUserLocal::TryRegisterTeleport",
        "services": ["channel"]
      }
      ```

      Match the surrounding file's exact formatting — the neighbouring
      `PortalScriptHandle` entry
      (`template_gms_83_1.json:884-891`) is the shape, including the expanded
      `"services"` array. `validator` is mandatory: a handlers entry without
      one is silently dropped by `BuildHandlerMap` and the op no-ops.

      Opcodes are the registry values, confirmed per file — gms_v83 `101`,
      gms_v84 `101`, gms_v87 `104`, gms_v92 `112`, gms_v95 `113`, jms_v185
      `96`. Do not copy one file's opcode into another.

- [ ] **Step 2: No route for the out-of-scope templates.**
      `template_gms_12_1.json`, `template_gms_48_1.json`,
      `template_gms_61_1.json`, `template_gms_72_1.json` and
      `template_gms_79_1.json` get **no** entry, per Task 2's findings. If
      Task 2 found an opcode on any of them, amend this plan first.

- [ ] **Step 3: Verify the guards.** From the repo root:

      ```
      tools/template-opcode-order-guard.sh
      tools/template-duplicate-binding-guard.sh
      tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_83_1.json
      ```

      Run `template-symbol-check.sh` once per edited template. It requires the
      literal `"InnerPortalHandle"` to exist in a `.go` file under
      `libs/atlas-packet` — Task 3 provides it, so run this after Task 3
      lands.

---

## Task 11: packet-audit linkage — `candidatesFromFName` case and registry `packet:` declarations

Two edits make the tooling able to resolve this op: the fname→codec candidate
(needed for export direction backfill and for any later report generation),
and the registry `packet:` link that lets a tier-0 cell promote on
marker + evidence with no audit report
(`tools/packet-audit/cmd/matrix.go:158-184`, `internal/matrix/grade.go:198-210`).

### Files

- `tools/packet-audit/cmd/run.go` — add the `CUserLocal::TryRegisterTeleport` case beside the existing portal case at line 2847
- `docs/packets/registry/gms_v83.yaml` — add `packet:` to the `USE_INNER_PORTAL` entry (line 2454)
- `docs/packets/registry/gms_v84.yaml` — line 3142
- `docs/packets/registry/gms_v87.yaml` — line 2570
- `docs/packets/registry/gms_v92.yaml` — line 2790
- `docs/packets/registry/gms_v95.yaml` — line 2825
- `docs/packets/registry/jms_v185.yaml` — line 2547

Module root for the Go edit: `tools/packet-audit` (or the repo root module
that owns it — build with `go build ./tools/packet-audit/...`).

### Steps

- [ ] **Step 1: Add the candidate case.** In `run.go`'s
      `candidatesFromFName`, in the `--- World: portal (serverbound) ---`
      block that already handles `CUserLocal::CheckPortal_Collision`
      (line 2839-2848), add:

      ```go
      // USE_INNER_PORTAL. The in-map ("inner") portal teleport registration
      // built by CUserLocal::TryRegisterTeleport (gms_v95 @0x913690, ctor
      // pushes 113 = 0x071). Wire per IDA: Encode1(fieldKey) +
      // EncodeStr(sourcePortalName) + Encode2(x) + Encode2(y) +
      // Encode2(destPortal.x) + Encode2(destPortal.y) — six fields, distinct
      // from CHANGE_MAP_SPECIAL's four. Matches atlas
      // portal/serverbound/inner_portal.go InnerPortal.
      case "CUserLocal::TryRegisterTeleport":
          return []candidate{{name: "InnerPortal", pkg: "portal", dir: csvpkg.DirServerbound}}
      ```

      With `pkg: "portal"`, `qualifiedWriterName` yields **`PortalInnerPortal`** —
      that is the report filename, the evidence filename stem and the
      `packet=` value everywhere downstream.

- [ ] **Step 2: Declare the packet in each registry.** In each of the six
      registry YAMLs, add a `packet:` line to the `USE_INNER_PORTAL` entry,
      between `fname:` and `provenance:` — the key order used by the existing
      declared entries (e.g. `gms_v95.yaml:228-233`):

      ```yaml
      - op: USE_INNER_PORTAL
        direction: serverbound
        opcode: <unchanged>
        fname: CUserLocal::TryRegisterTeleport
        packet: portal/serverbound/PortalInnerPortal
        provenance: csv-import
      ```

      Change nothing else on those entries — the opcode and fname stay as they
      are.

- [ ] **Step 3: Verify.** `go build ./tools/packet-audit/... && go test ./tools/packet-audit/...`,
      then `go run ./tools/packet-audit matrix --check` from the repo root to
      confirm the run introduces no new dangling/orphan/conflict lines
      mentioning `PortalInnerPortal` and does not raise the conflict count.
      Capture the pre-edit `matrix --check` output first so the comparison is
      against a real baseline, not a remembered one.

---

## Task 12: Splice the exports, pin evidence, add the verify markers, regenerate the matrix

This is the promotion step. It needs a live IDA-MCP session per version and
the renames Task 1 Step 3 saved.

### Files

- `docs/packets/ida-exports/gms_v83.json` — splice one entry
- `docs/packets/ida-exports/gms_v84.json` — splice one entry
- `docs/packets/ida-exports/gms_v87.json` — splice one entry
- `docs/packets/ida-exports/gms_v92.json` — splice one entry
- `docs/packets/ida-exports/gms_v95.json` — splice one entry
- `docs/packets/ida-exports/gms_jms_185.json` — splice one entry
- `libs/atlas-packet/portal/serverbound/inner_portal_test.go` — add the six `packet-audit:verify` markers
- `docs/packets/audits/STATUS.md` — regenerated, never hand-edited
- `docs/packets/audits/status.json` — regenerated
- `docs/packets/evidence/gms_v83/portal.serverbound.PortalInnerPortal.yaml` — new file (one per version, six total)
- `docs/packets/audits/VERIFYING_A_PACKET.md` — read-only; §7 pinning, §9 serverbound rules, §10 export hygiene

### Steps

- [ ] **Step 1: Splice each export.** Never overwrite a committed export — the
      harvest is not idempotent and a full re-export drifts ~150 unrelated
      entries. Use the surgical `--splice` path, once per version, with a
      one-line roster file naming only this fname:

      ```
      printf 'CUserLocal::TryRegisterTeleport\n' > /tmp/task250-roster.md
      go run ./tools/packet-audit export \
        --version gms_v95 \
        --output docs/packets/ida-exports/gms_v95.json \
        --prior-export "" --pending /tmp/task250-roster.md \
        --splice "CUserLocal::TryRegisterTeleport" \
        --descent-depth 12 \
        --ida-database <session id for THIS version's IDB>
      ```

      Repeat for `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, and `jms_v185`
      (whose export file is `gms_jms_185.json`). `--ida-database` is
      mandatory: without it the harvest silently targets whichever IDB the
      server considers active. Confirm each run prints
      `export: spliced ... (one entry merged, others preserved)` and that
      `git diff --stat` on the export shows a small delta, not a rewrite.

      If a splice fails with `delegate to COutPacket: not in export`, strip
      that one `{op: Delegate, ref: COutPacket}` call from the spliced entry —
      it is the packet constructor, not a wire read, and other versions' entries
      omit it (VERIFYING §10).

- [ ] **Step 2: Pin evidence per version.** Six runs:

      ```
      go run ./tools/packet-audit evidence pin \
        --packet portal/serverbound/PortalInnerPortal \
        --version gms_v95 \
        --ida "CUserLocal::TryRegisterTeleport" \
        --category TIER1-FIXTURE \
        --verifies "libs/atlas-packet/portal/serverbound/inner_portal_test.go#TestInnerPortalGoldenBytes"
      ```

      `--ida` is the fname as it keys the export's `functions` map; the tool
      resolves the address and the decompile hash itself, which is why Step 1
      must land first. Pass `--verifies` on the pin — do **not** hand-edit the
      generated
      `docs/packets/evidence/<version>/portal.serverbound.PortalInnerPortal.yaml`;
      the flag exists precisely so the record stays tool-written
      (`tools/packet-audit/cmd/evidence.go:31-35`).

- [ ] **Step 3: Add the verify markers.** Above `TestInnerPortalGoldenBytes`
      in `inner_portal_test.go`, one marker per in-scope version, `ida=` being
      that version's export address for the fname (Task 1's structure docs):

      ```go
      // packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v83 ida=0x<addr>
      // packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v84 ida=0x<addr>
      // packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v87 ida=0x9da037
      // packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v92 ida=0x<addr>
      // packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v95 ida=0x913690
      // packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=jms_v185 ida=0xa2218f
      ```

      A version with no opcode gets **no** marker — never a silent skip.

- [ ] **Step 4: Regenerate and confirm promotion.**

      ```
      go run ./tools/packet-audit matrix
      go run ./tools/packet-audit matrix --check
      go run ./tools/packet-audit fname-doc --check
      go run ./tools/packet-audit operations --check
      ```

      All must exit 0. Then confirm, by reading the regenerated file rather
      than assuming: the `USE_INNER_PORTAL` row in
      `docs/packets/audits/STATUS.md` shows ✅ on gms_v83, gms_v84, gms_v87,
      gms_v92, gms_v95 and jms_v185, and the four legacy columns still read
      the state Task 2 derived. A cell that does not promote is a failure to
      diagnose against `internal/matrix/grade.go`, never something to
      hand-edit into `STATUS.md`.

      `fname-doc --check` will report the `PortalInnerPortal` struct as
      "without an audit report, carries no fname" — that is reported, not
      failed (`tools/packet-audit/cmd/fnamedoc.go:126-138`), and the
      hand-written `// packet-audit:fname CUserLocal::TryRegisterTeleport`
      comment from Task 3 stays.

- [ ] **Step 5: Commit the artifacts together.** Exports, evidence YAMLs, the
      test with its markers, and the regenerated `STATUS.md` / `status.json`
      in one commit — a split leaves the matrix inconsistent with its inputs.

---

## Final verification (controller, after every task)

- [ ] Flagless `tools/verify.sh` exits 0 (not `--quick`, not `--no-docker` —
      only the flagless run performs the bake and `-race`).
- [ ] `go run ./tools/packet-audit matrix --check`, `fname-doc --check` and
      `operations --check` all exit 0.
- [ ] `packet-completeness-critic` run against
      `docs/tasks/task-250-inner-portal-registration/coverage-manifest.yaml`
      reports no CHANGED-BUT-UNCLAIMED and no CLAIMED-BUT-UNVERIFIED cells.
- [ ] `backend-guidelines-reviewer` over the changed Go packages in
      `services/atlas-channel`, `services/atlas-character` and
      `libs/atlas-packet`.
- [ ] Code review completed before the PR is opened.
