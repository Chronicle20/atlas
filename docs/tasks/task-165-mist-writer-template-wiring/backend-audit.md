# Backend Audit — task-165-mist-writer-template-wiring (Go surface)

- **Service Path:** `libs/atlas-packet` (field/clientbound package) and `services/atlas-channel/atlas.com/channel`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS
- **Tests:** all packages in `libs/atlas-packet` PASS (targeted `AffectedArea*` tests: 25/25 subtests PASS); `go vet ./...` clean on `services/atlas-channel/atlas.com/channel`
- **Overall:** PASS

## Scope

This audit covers only the Go files this branch actually touched, per the task brief:

- `libs/atlas-packet/field/clientbound/affected_area_created.go`
- `libs/atlas-packet/field/clientbound/affected_area_removed.go` (unchanged in this diff — read for context; `AffectedAreaRemovedWriter` const it defines is what `main.go` newly registers)
- `libs/atlas-packet/field/clientbound/affected_area_test.go`
- `services/atlas-channel/atlas.com/channel/main.go`

`git diff e0f5bd01d..a97540c8b -- '*.go'` confirms this is the complete Go delta (3 files, 197 insertions / 68 deletions): `affected_area_created.go` (+66/-8), `affected_area_test.go` (rewritten fixtures), `main.go` (+2).

Non-Go changes (tenant seed template JSON registering the two writer names across all 11 supported version templates, packet evidence records, coverage matrix, task docs) are context only, not audited — confirmed present via `git show c49d7dafb --stat` (8 templates) plus `template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_92_1.json` also carrying `AffectedAreaCreated` (11/11 supported versions wired).

## Build & Test Results

```
cd libs/atlas-packet && go build ./...          # exit 0, no output
cd libs/atlas-packet && go test ./... -count=1  # all "ok", no failures
cd services/atlas-channel/atlas.com/channel && go build ./...  # exit 0
cd services/atlas-channel/atlas.com/channel && go vet ./...    # exit 0
```

`go test ./field/clientbound/... -run AffectedArea -v` (11-version fixture matrix): `TestAffectedAreaCreatedWireShape` PASS, `TestAffectedAreaCreatedByteOutput` PASS across all 11 `t.Run` subtests (v12/v48/v61/v72/v79/v83/v84/v87/v92/v95/jms185), `TestAffectedAreaCreatedFields` PASS, `TestAffectedAreaRemovedByteOutput` PASS across all 11 subtests, `TestAffectedAreaRemoved_EncodeShape` PASS.

## Classification

`libs/atlas-packet/field/clientbound` is a packet-codec library package (no `model.go`, no `processor.go`, no REST/DB layer) — it does not fit the DOM/SUB domain checklist shape. `services/atlas-channel/.../main.go` is service wiring (writer registration list), not a domain package. The File Responsibilities checklist (Processor/RestModel/requests-in-separate-files) does not apply to either — there is no Processor, RestModel, or cross-service request code in this diff. Checks below are the ones from the guideline set that are actually mechanically applicable to a version-gated wire codec plus a registration-list edit, per the task's stated points of interest.

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client wire values (opcodes/dispatcher codes) are config-resolved, never Go literals | PASS | `grep -n "0x[0-9A-Fa-f][0-9A-Fa-f]\b" affected_area_created.go affected_area_removed.go` returns only `+0x30`/`+0x48` struct-offset references inside `//` comments (`affected_area_created.go:19-21,117-118,129-130,162`) documenting IDA memory offsets for engineering traceability — none are Go literals feeding wire output. `affected_area_test.go:78-86,238-246` `packet-audit:verify` tags carry `ida=0x...` addresses in comments only, same category. The wire body itself is built purely from struct fields (`m.nType`, `m.ownerId`, `m.skillId`, ...) written via `response.Writer` methods; no opcode byte is hardcoded into the encoder or into `main.go`. `main.go:719-720` registers the writer by symbolic Go const name (`fieldcb.AffectedAreaCreatedWriter`, `fieldcb.AffectedAreaRemovedWriter`), not a wire byte. |
| DOM-25 (gate resolution) | Version-dependent field presence/width resolved from `tenant.Model`, not a hardcoded table | PASS | `affected_area_created.go:113,119,122,126,132,134` — `t := tenant.MustFromContext(ctx)`, then `twoTimeWords`, `wideNType`, `hasOwnerId`, `trailing` are all derived from `t.IsRegion(...)`/`t.MajorAtLeast(...)` at encode time; no separate opcode/byte lookup table was introduced (this packet has no client-interpreted mode byte — it's a fixed-shape spawn/despawn packet whose only version variance is field width/presence, which is exactly what `MajorAtLeast`/`IsRegion` gating is for, not a DOM-25 config-table case). |
| Version gating | Uses `MajorAtLeast`/`IsRegion` idiom, not raw `MajorVersion() > N` | PASS | `affected_area_created.go:119,122,126,132,134` — all five gates use `t.IsRegion("GMS")` and `t.MajorAtLeast(N)`. `grep -n "MajorVersion()\s*[><=]" affected_area_created.go affected_area_removed.go` returns zero matches — the prior raw comparison `t.MajorVersion() >= 95` (visible in the pre-image via `git diff`, old line `v95Plus := t.Region() == "GMS" && t.MajorVersion() >= 95`) was replaced, not left alongside the new idiom. `libs/atlas-tenant/tenant.go:88,93` confirms `IsRegion(region string) bool` and `MajorAtLeast(v uint16) bool` are the real, exported methods being called (not local shims). |
| Multi-tenancy | Tenant obtained from context, not threaded or cached | PASS | `affected_area_created.go:113` — `t := tenant.MustFromContext(ctx)` is called fresh inside `Encode(l, ctx)`, using the `ctx` parameter passed at call time; no package-level or struct-level tenant field exists on `AffectedAreaCreated` (struct at `affected_area_created.go:55-70` holds only wire-data fields: `mistId`, `ownerId`, `nType`, `skillId`, `skillLevel`, `phase`, coordinates, `tStart`, `tEnd` — no tenant/context field). `affected_area_removed.go:35` `Encode(m AffectedAreaRemoved) ... (l logrus.FieldLogger, _ context.Context)` — context param present per the shared `Encode` signature even though this packet needs no tenant gating (body is a fixed 4-byte key across every supported version, per its own doc comment and the 11-version fixture parity in `affected_area_test.go:257-280`), so the unused `ctx` is correctly named `_` rather than a dead binding. |
| Immutability | Packet model fields are private with no mutating methods; `Encode` returns a pure closure over already-bound fields | PASS | `affected_area_created.go:56-69` — all fields (`mistId`, `ownerId`, `nType`, ...) are unexported (lowercase) on `AffectedAreaCreated`; only a constructor `NewAffectedAreaCreated` (`:72`) and read-only accessor methods (`:91-104`) exist — no setter mutates the struct after construction. `Encode` (`:111`) computes gate booleans (`twoTimeWords`, `wideNType`, `hasOwnerId`, `trailing`) once from `ctx`/`t`, then returns `func(options map[string]interface{}) []byte` (`:137`) that closes over `m` (receiver, value not pointer — `func (m AffectedAreaCreated) Encode`) and the precomputed gates; the closure only reads `m.*` fields and appends to a local `w`, it does not mutate `m` or package state. Receiver is by value (`m AffectedAreaCreated`, not `*AffectedAreaCreated`) at `:111`, so even if the closure were retained and called from multiple goroutines it operates on its own copy. |
| Closure `w` reuse | Encoder does not leak mutable state across repeated closure invocations | WARN (pre-existing pattern, not introduced by this branch) | `affected_area_created.go:112` — `w := response.NewWriter(l)` is created once in `Encode` and captured by the returned closure (`:137-169`), so if the returned func were invoked more than once the same `*Writer` buffer would be reused/appended-to rather than reset. This exact shape (`w` declared outside the returned closure) is unchanged by this diff — the pre-image (`git diff`) shows `w := response.NewWriter(l)` was already present before task-165 touched this file, and it is the standard idiom across the sibling `clientbound` writers in this package (each writer is invoked exactly once per call site in practice). Not a new defect introduced by this branch; flagged for completeness only, not counted as blocking. |
| DOM-26 | No bare `go` statements introduced | PASS | `grep -nE '^\s*go (func|[A-Za-z_])' affected_area_created.go affected_area_removed.go main.go` (scoped to changed lines) returns zero matches. No goroutines were introduced by this diff. |
| DOM-12 analog | No `os.Getenv` introduced | PASS | `grep -n "os.Getenv" affected_area_created.go affected_area_removed.go` returns zero matches. |
| DOM-21 | No duplication of atlas-constants types | PASS | The only new type is `trailingWidth int` (`affected_area_created.go:23-29`), a package-private 3-value enum (`trailingWidthWide/Byte/None`) that selects how many bytes of `tEnd` this specific packet writes — it is not a world/channel/character/map/job/skill/monster/item classification and has no equivalent in `libs/atlas-constants`. |
| Registration wiring | `main.go` registers both new writer consts by symbolic name | PASS | `services/atlas-channel/atlas.com/channel/main.go:719-720` — `fieldcb.AffectedAreaCreatedWriter,` and `fieldcb.AffectedAreaRemovedWriter,` added to the `produceWriters()` slice, referencing the exported consts declared at `affected_area_created.go:14` and `affected_area_removed.go:13`. Diff is additive-only (`+2` lines), no adjacent line touched or reordered incorrectly. |

## Security Review

Not applicable — this service surface is not auth-related.

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

- Closure `w` reuse: `libs/atlas-packet/field/clientbound/affected_area_created.go:112` declares the `*response.Writer` outside the returned closure, so a hypothetical repeated invocation of the same returned func would append to a stale buffer rather than start fresh. This is a pre-existing idiom shared by every sibling writer in the package (not introduced or altered by task-165) and is not blocking this branch, but is worth a fleet-wide follow-up if the socket layer ever starts caching/reusing `Encode`'s returned closure across calls.

### Noted, not a finding against this branch

- The mist Kafka consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go`) passes `phase = 0` while the client derives mist expiry from `phase * 100 + now`. This file is outside the Go diff for task-165 (confirmed by `git diff --stat` above) and is a known, already-tracked follow-up per the task brief — not audited here.

---

**Post-review correction (added by the controller after these audits ran).**
These audits were produced against the tree at commit `a97540c8b`, before the mist
consumer fix landed. Any statement in this document that the `phase = 0` defect in
`services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go` is a
pre-existing, out-of-scope follow-up is now **obsolete**: it was fixed in commit
`9b471311f` at the user's explicit direction, overriding the plan's original
"no consumer/producer changes" constraint. `phase` now carries `Duration / 100`
(clamped to int16 max, floored at 1) via the `mistPhase` helper. No wire byte moved.
