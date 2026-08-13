# Backend Audit — task-188-v48-template-completeness

- **Service Path:** libs/atlas-packet, services/atlas-configurations/atlas.com/configurations, tools/packet-audit (no single service module — libs/tools change)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS
- **Tests:** All packages pass (see per-module results below); 0 failed
- **Overall:** NEEDS-WORK

## Scope Note

No `plan.md`/`design.md` exists for this task (per the run instructions this is
expected and not itself a finding). Reviewed via `git diff origin/main...HEAD`
against the 32 changed `.go` files plus the `template_gms_48_1.json` /
`docs/packets/registry/gms_v48.yaml` data changes that motivate them.

None of the changed packages are DOM/SUB domain packages (model.go / resource.go
CRUD) — they are packet codec libraries (`libs/atlas-packet/<family>/{clientbound,serverbound}`)
plus one shared model file (`libs/atlas-packet/model/monster.go`) and a config
generation tool (`tools/packet-audit/cmd`). All DOM-01..24, DOM-27, DOM-28,
SUB-*, FILE-*, EXT-*, and SCAFFOLD-* checks are marked N/A per the run
instructions. DOM-25 (client-wire-value config-resolution), the version-gating
idiom check, and DOM-26 (goroutine guard) were fully applied as instructed.

## Build & Test Results

```
$ cd libs/atlas-packet && go build ./...          -> clean, no output
$ cd libs/atlas-packet && go test ./... -count=1   -> ok, all packages (no failures)

$ cd services/atlas-configurations/atlas.com/configurations && go build ./...          -> clean
$ cd services/atlas-configurations/atlas.com/configurations && go test ./... -count=1   -> ok, all packages
  (socket/corpus_test.go: TestValidate_AcceptsEverySeedTemplate now asserts total==3051,
   i.e. 2993 + 58 gms_v48 handler/writer entries — matches the task's stated FR)

$ cd tools/packet-audit && go build ./...          -> clean
$ cd tools/packet-audit && go test ./... -count=1   -> ok, all packages
```

`go.work` is a single workspace covering all `libs/*` and `services/*/atlas.com/*`
modules plus `tools/packet-audit`, so `go build`/`go test` per-module above also
exercises cross-module compatibility within the workspace.

Guard scripts run tree-wide (repo root), relevant to this diff:

```
$ ./tools/goroutine-guard.sh          -> exit 0, no bare `go` statements outside
                                          libs/atlas-routine / justified sites
$ ./tools/template-opcode-order-guard.sh       -> OK: 22 template arrays in ascending opcode order
$ ./tools/template-duplicate-binding-guard.sh  -> OK: no duplicate (name, opCode) bindings
$ ./tools/template-movement-types-guard.sh     -> OK: 54 move handlers, valid movement types tables
```

## Domain Checklist Results

N/A — no package under the 32 changed files has a `model.go` (domain) or a
bare `resource.go` (sub-domain). DOM-01..24, DOM-27, DOM-28 are N/A for this
diff's scope.

### DOM-25 — Client-interpreted byte values are config-resolved

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | New/changed writers with client-interpreted sub-op codes are config-resolved | PASS | `services/atlas-configurations/seed-data/templates/template_gms_48_1.json` — new `ShopScannerResult` writer entry (opCode `0x39`) carries `options.operations: {RESULT: 6, HOT_LIST: 7}`, matching `libs/atlas-packet/merchant/shop_scanner_body.go:37,43` (`atlas_packet.WithResolvedCode("operations", ...)`). New `MessengerOperation` writer entry (opCode `0xEE`) carries a full 9-arm `options.operations` table matching `libs/atlas-packet/messenger/operation_body.go`'s `WithResolvedCode("operations", ...)` call sites. Spot-checked entries requiring a table (`IncubatorResult`, `ShopScannerResult`, `MessengerOperation`) all carry one; entries whose writer has no `WithResolvedCode` call in the codebase (e.g. `MiniRoom`, `PartyMemberHP`, `GuildNameChanged`) correctly carry none. |
| DOM-25 | Version-gated field presence is not itself a client wire code | N/A | The `has*`/`mob*` helper functions added in this diff (`hasIncStorageCurrency`, `hasWorldBalloons`, `mobHasTeamAndEffectItem`, `hasEnabledFlag`, `hasPetSpawnLead`, `hasReactorName`) gate whether a *field* is present on the wire, not which value a client-side lookup switch resolves to — DOM-25 does not apply to field-presence gating. |

### DOM-26 — Goroutines spawned via routine.Go

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | No bare `go` statements in changed packages | PASS | `./tools/goroutine-guard.sh` exits 0 tree-wide; `grep -rnE '^\s*go (func|[A-Za-z_])' --include='*.go'` on the 32 changed files returns no matches outside test files, and none of the changed `.go` files spawn goroutines at all. |

### Version-gating idiom check (raw literal comparisons banned; `MajorAtLeast`/`MajorAtMost` idiom required)

`libs/atlas-tenant/tenant.go` exposes `MajorAtLeast(v)`, `MajorAtMost(v)`, and
`MajorInRange(lo, hi)`. The task's own convention, used correctly six times in
this diff (`hasIncStorageCurrency`, the two new `SET_FIELD` `MajorAtLeast(61)`
guards, `hasWorldBalloons`, `mobHasTeamAndEffectItem`, `hasPetSpawnLead`,
`hasReactorName`), is to express version boundaries via these methods, never
via raw `t.MajorVersion() > N` / `>= N` / `<= N` comparisons.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| VER-01 | `set_field.go` damage-seed-count gate uses the idiom | **FAIL** | `libs/atlas-packet/field/clientbound/set_field.go:74` and `:126` — `if (t.Region() == "GMS" && t.MajorVersion() > 28) \|\| t.Region() == "JMS"` is a raw `>` literal comparison, newly split out into its own `if` block by this diff (previously it shared a block with the `nNotifierCheck` gate, which this diff correctly rewrote to `t.IsRegion("GMS") && t.MajorAtLeast(61)` two lines above). The sibling condition on the very same two lines proves the idiom was known and available; the damage-seed gate was left on the banned raw-literal form instead of being expressed as `t.IsRegion("GMS") && t.MajorAtLeast(29)`. |
| VER-02 | `model/monster.go` legacy-v12-tail gate uses the idiom | **FAIL** | `libs/atlas-packet/model/monster.go:537` — `func mobLegacyV12Tail(t tenant.Model) bool { return t.Region() == "GMS" && t.MajorVersion() <= 12 }` uses a raw `<=` literal comparison. `tenant.Model.MajorAtMost(v)` exists specifically for this shape (`libs/atlas-tenant/tenant.go:98`) and is unused anywhere in `libs/atlas-packet`; this should be `t.IsRegion("GMS") && t.MajorAtMost(12)`. |
| VER-03 | `npc/clientbound/spawn.go` `hasEnabledFlag` uses the idiom | **FAIL** | `libs/atlas-packet/npc/clientbound/spawn.go:35` — `return t.Region() != "GMS" \|\| t.MajorVersion() >= 61` is a raw `>=` literal comparison. The equivalent boundary is expressed correctly elsewhere in the very same diff as `t.IsRegion("GMS") && t.MajorAtLeast(61)` (e.g. `pet/serverbound/spawn.go`'s `hasPetSpawnLead`, `reactor/clientbound/spawn.go`'s `hasReactorName`, `login/clientbound/server_list_entry.go`'s `hasWorldBalloons`). This should read `!t.IsRegion("GMS") \|\| t.MajorAtLeast(61)`. |
| VER-04 | No wire-byte regression to an already-verified version | PASS | The three genuinely gated codecs that changed *existing* branch conditions (`effect_weather.go`'s legacy-shape gate moved from `< 61` to `== 61`; `shop_operation_increase_storage.go` gains a v48-only omission; `set_field.go`/`model/monster.go`/`npc/clientbound/spawn.go`/`pet/serverbound/spawn.go`/`reactor/clientbound/spawn.go` add a `v48`-excluded arm to a previously-unconditional write) all move only the **v48** boundary (or, for `effect_weather.go`, the v61 boundary while leaving v48+ on `encodeGMS`, which the comment documents as a correction of a prior mis-citation, not a new regression) — none of the changes narrow or alter the wire shape for v72/v79/v83/v84/v87/v95/JMS, which is exactly the "must never alter an already-verified version's bytes" constraint. Round-trip and byte-fixture tests for the untouched versions (`TestSetFieldByteOutputV61`, `TestServerListEntryRoundTrip`, `TestSpawnRoundTrip`, etc.) still pass, confirming no regression. |

Findings VER-01..03 are cosmetic-idiom violations, not wire-correctness bugs
— the encoded bytes are identical whether expressed as `> 28`/`<= 12`/`>= 61`
or via the `MajorAtLeast`/`MajorAtMost` helpers, and all affected tests pass.
They are flagged because the raw-literal form is explicitly banned and the
correct idiom was demonstrably known and used on adjacent lines of the same
diff, making the inconsistency avoidable.

## Testing-Guide / Anti-Pattern Checks

| Check | Status | Evidence |
|-------|--------|----------|
| Table-driven tests where applicable | PASS (convention-consistent with `docs/packets/IMPLEMENTING_A_PACKET.md` / `VERIFYING_A_PACKET.md`) | New tests follow the established per-version byte-fixture pin convention (`TestXxxBytesV48`, one function per packet × version cell, each carrying a `packet-audit:verify` marker) rather than `t.Run` tables — this is the packet-audit playbook's documented pattern, not the generic DOM-20 domain-CRUD convention, and the playbook is the more specific guideline for this file type. |
| Byte-fixture correctness / no invented values | PASS | Every new/changed fixture test (`door/clientbound/v48_test.go`, `field/clientbound/kite_v48_test.go`, `field/clientbound/set_field_test.go`, `login/clientbound/login_v48_test.go`, `login/clientbound/server_list_entry_v48_test.go`, `monster/clientbound/{v48_test.go,mob_affected_v48_test.go}`, `monster/serverbound/mob_drop_pickup_v48_test.go`, `npc/clientbound/spawn_v48_test.go`, `pet/serverbound/v48_test.go`, `reactor/clientbound/v48_test.go`, `inventory/serverbound/scroll_use_v48_test.go`, `guild/serverbound/operation_agreement_response_v48_test.go`, `cash/serverbound/v48_test.go`) cites an IDA address and, where the layout is non-obvious, the specific `Decode*` call sequence backing each field — consistent with CLAUDE.md "Verification Over Memory". No un-cited magic byte sequences found. |
| Kafka producer stubbing (DOM-24) | N/A | None of the 32 changed files call `AndEmit`, `message.Emit`, or `producer.Produce` — packet codecs have no Kafka interaction. |
| No manual JSON parsing / handler layering anti-patterns | N/A | No `resource.go`/handler files in scope. |
| `go test ./... -count=1` clean, no cache | PASS | Ran with `-count=1` in all three modules; see Build & Test Results above. |

## File Responsibilities Checklist

N/A — none of the 32 changed files belong to a package with `model.go`,
`resource.go`, `processor.go`, `rest.go`, `requests.go`, or `entity.go`
responsibilities; they are packet-codec files (`<op>.go` per family/direction
package, following that subsystem's own established one-codec-per-file
convention) and one CLI tool package. FILE-01..06 do not apply to this file
type.

## External HTTP Client Checklist

N/A — no package in this diff calls `requests.GetRequest[T]` / `PostRequest[T]`
against another atlas service.

## Service Scaffolding Checklist

N/A — no new `services/atlas-<service>/` directory, and no new `Writer` /
`Handler` constant is registered in `services/atlas-channel/atlas.com/channel/main.go`
in this diff (the diff wires *existing* Go writer/handler constants into the
v48 seed template's `writers[]`/`handlers[]` arrays; it does not add new Go
constants to atlas-channel's dispatch tables).

## Security Review

N/A — not an authentication/authorization/token-management service.

## Summary

### Blocking (must fix)
- VER-01: `libs/atlas-packet/field/clientbound/set_field.go:74,126` — raw `t.MajorVersion() > 28` should be `t.IsRegion("GMS") && t.MajorAtLeast(29)`.
- VER-02: `libs/atlas-packet/model/monster.go:537` — raw `t.MajorVersion() <= 12` should be `t.IsRegion("GMS") && t.MajorAtMost(12)`.
- VER-03: `libs/atlas-packet/npc/clientbound/spawn.go:35` — raw `t.MajorVersion() >= 61` should be `t.IsRegion("GMS") && t.MajorAtLeast(61)` (with the outer `!=`/`||` restructured accordingly).

### Non-Blocking (should fix)
- None identified beyond the three VER-* items above (which are style/idiom, not wire-correctness, defects — all builds and tests pass and no already-verified version's bytes changed).
