# task-250 — Implementation Context

Companion to [plan.md](plan.md). What an implementer or reviewer needs that the
per-task briefs do not carry.

---

## 1. Where the work lands

| Area | Module root for `go build ./... && go test ./...` |
|---|---|
| Codec | `libs/atlas-packet` |
| Channel (data/portal, position, movement, portal, handler, main) | `services/atlas-channel/atlas.com/channel` |
| Character (foothold preservation) | `services/atlas-character/atlas.com/character` |
| packet-audit tooling | repo root (`go build ./tools/packet-audit/...`) |
| Templates, registry, exports, evidence, matrix | no Go module — guard scripts and `packet-audit` |

Three services plus a library plus the audit tooling. That spread is why the
plan is twelve tasks rather than four.

---

## 2. Decisions this plan makes that the design did not

These are corrections and gap-fills found by reading the code. Each one is a
place where following `design.md` literally would fail.

**2.1 The matrix packet id is `portal/serverbound/PortalInnerPortal`, not
`.../InnerPortal`.** Design §2 states the latter.
`qualifiedWriterName(pkg, name)` (`tools/packet-audit/cmd/run.go:223-228`)
prepends the title-cased package whenever the candidate carries a `pkg`, which
is why struct `Script` in package `portal` is already
`portal/serverbound/PortalScript` in `status.json`. The id drives the evidence
filename, the marker `packet=` value and the registry `packet:` link — getting
it wrong means no cell promotes.

**2.2 Promotion goes through the no-report path.** The op is tier-0
(`"tier1": false` in `status.json`). `grade.go:198-210` promotes a tier-0 cell
on **marker + fresh evidence with no audit report**, provided the version's
registry entry declares `packet:`; `matrix.go:158-184` exempts exactly that
case from the dangling-evidence failure. Six audit reports would otherwise
need generating against the exports, with the `COutPacket`-delegate descent
failure mode that comes with them. Task 11 Step 2 is what makes this path
legal — without the `packet:` lines the evidence records are dangling and
`matrix --check` fails.

**2.3 `data/portal.Model` has only `Id()`.** Design §5 calls `pm.TargetMapId()`,
`pm.Target()`, `pm.X()` and `pm.Y()`; none exist
(`services/atlas-channel/atlas.com/channel/data/portal/model.go`). Task 4 adds
them. The fields are already populated by `Extract` in `rest.go`.

**2.4 The last-position registry lives in its own `position` package, not in
`movement`.** Design §5.5 puts it in `movement`. But `session` must clear it on
destroy, and `movement` already imports `session` — that is an import cycle.
A standalone `atlas-channel/position` package importing only `sync` and
`libs/atlas-tenant` is importable by `session`, `movement` and `portal` alike.
Verified no cycle: `go list -deps ./movement | grep atlas-channel/portal`
returns nothing, so `portal → movement` is also safe.

**2.5 `EnterInner` reaches `movement` through a package-level seam, not a
threaded `writer.Producer`.** `movement.NewProcessor` takes a
`writer.Producer`; `portal.NewProcessor(l, ctx)` does not, and there are six
call sites across three packages (`socket/handler/{portal_script,map_change,
owl_warp,mystic_door_enter}.go`, `skill/handler/resurrection/resurrection.go`).
`TeleportCharacter` emits no clientbound packet and never touches `p.wp`, so
the seam constructs the movement processor with a nil producer. That is the
dominant idiom in this codebase for exactly this situation — `warpFunc`
(`socket/handler/mystic_door_enter.go`), `warpToPosition`
(`skill/handler/resurrection/resurrection.go`), `monsterByIdFn`
(`movement/processor.go:133-136`) — and it keeps the portal tests free of a
live producer.

**2.6 `temporalRegistry.UpdatePosition` already exists and has no callers.**
Design §4.3 says to add it. `services/atlas-character/.../character/temporal_data.go:73`
already defines `UpdatePosition(ctx, t, characterId, x, y)` preserving `fh` and
`stance`, and nothing calls it. Task 7 adds the `stance` parameter the design
specifies rather than introducing a second method.

**2.7 Refusals return `nil`.** `EnterInner`'s refusal branches log at warning
and return `nil`; the only non-nil error is from the position publish. PRD
FR-4.6 makes a refusal a deliberate no-op, and the handler ignores the return
anyway (`_ =`, matching `PortalScriptHandleFunc`). Tests assert refusal through
the movement mock, not the return value.

**2.8 The distance threshold's value is produced by Task 2, not by this plan.**
`maxPortalEntryDistance` is derived from the client's portal collision rect in
`CUserLocal::CheckPortal_Collision`. Task 2 writes a single integer into
`structures/gms_v95.md` under `## Threshold derivation`; Task 8 copies it
verbatim and cites the section. Task 8's test table is written relative to the
constant (`threshold` accepted, `threshold + 1` refused), so it stays exact
whatever the derived number turns out to be.

---

## 3. Key files by role

**Patterns to copy**

| Pattern | Reference |
|---|---|
| Serverbound codec shape | `libs/atlas-packet/portal/serverbound/script.go` |
| Round-trip test shape | `libs/atlas-packet/portal/serverbound/script_test.go:12-36` |
| Tenant-scoped process-lifetime cache | `services/atlas-channel/atlas.com/channel/data/map/processor.go:34-73` |
| Singleton registry + `GetRegistry()` | `services/atlas-channel/atlas.com/channel/character/chakra/registry.go:29-69` |
| Session-destroy hook + its test | `services/atlas-channel/atlas.com/channel/session/processor.go:449-462`, `session/aran_combo_hook_test.go` |
| Handler shape | `services/atlas-channel/atlas.com/channel/socket/handler/portal_script.go` |
| Template handlers entry | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json:884-891` |
| fname→codec candidate | `tools/packet-audit/cmd/run.go:2839-2848` |
| Registry `packet:` declaration | `docs/packets/registry/gms_v95.yaml:228-233` |
| Coverage manifest | `docs/tasks/task-230-scripted-items/coverage-manifest.yaml` |

**Read-only ground truth**

- `docs/packets/registry/<version>.yaml` — the per-version opcode. Never copy
  one version's opcode into another; the v84 table is shifted relative to v83.
- `docs/packets/MapleStory Ops - ServerBound.csv:237` — the `USE_INNER_PORTAL`
  row, including the `0x000` absent sentinel in the `GMS v12` column.
- `docs/packets/audits/STATUS.md:640` — the row this task promotes.
- `libs/atlas-constants/map/model.go:39` — `Id.IsSentinel()` / `EmptyMapId`.
  Do not write `999999999`.

---

## 4. Opcode table (from the registry, confirmed per file)

| Version | Template | Registry opcode | Template `opCode` |
|---|---|---|---|
| gms_v83 | `template_gms_83_1.json` | 101 | `"0x65"` |
| gms_v84 | `template_gms_84_1.json` | 101 | `"0x65"` |
| gms_v87 | `template_gms_87_1.json` | 104 | `"0x68"` |
| gms_v92 | `template_gms_92_1.json` | 112 | `"0x70"` |
| gms_v95 | `template_gms_95_1.json` | 113 | `"0x71"` |
| jms_v185 | `template_jms_185_1.json` | 96 | `"0x60"` |

None of those opcodes currently appears in the corresponding template's
`socket.handlers` array, so there is no collision to resolve — verified by
grep across all six files.

`template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_61_1.json`,
`template_gms_72_1.json` and `template_gms_79_1.json` get no route unless
Task 2 finds an opcode, in which case the plan is amended before Task 10 runs.

---

## 5. Dependencies and IDA requirements

Tasks 1, 2 and 12 need a live IDA-MCP session. Task 1 also **renames** the
unnamed send site in the v83, v84 and v92 IDBs to
`CUserLocal::TryRegisterTeleport` and saves — Task 12's `--splice` keys on that
name, so skipping the rename blocks promotion on three versions. That is a
producible prerequisite, not a blocker (CLAUDE.md "Evidence & grounding").

Everything else is repo-local. Tasks 4, 5, 6, 7, 10 and 11 have no
inter-dependencies and can run in any order or concurrently.

Task 10's `tools/template-symbol-check.sh` leg requires the literal
`"InnerPortalHandle"` to already exist in `libs/atlas-packet`, so run that
verification after Task 3 has landed.

---

## 6. Task sizing

Every task is at or under six files and confined to one module, with two
deliberate exceptions:

- **Task 1** writes six structure docs. They are the same document six times
  with different addresses; there is no discovery cost per file after the
  first, and splitting would duplicate the IDB adoption overhead.
- **Task 11** touches seven files (one Go file plus six registry YAMLs). The
  six YAML edits are one identical three-token insertion each.
- **Task 12** touches six exports plus six evidence records. It is one
  command loop over six versions, not six investigations — but it is the most
  likely task to hand back `PARTIAL`, because each version needs its own IDB
  session and a splice can fail on the `COutPacket`-delegate artifact. A
  continuation brief for the remaining versions is the expected split point.

---

## 7. Risks carried forward from the design

| Risk | Where it is handled |
|---|---|
| v83/v84/v92 layout diverges from the three confirmed versions | Task 1 Step 5 records the delta and names the `MajorAtLeast` boundary; Task 3 gates on it |
| Portal data becomes process-lifetime static | Accepted, identical to `data/map`; the cache comment must say so |
| `fh = 0` clobbers a stored foothold | Task 7, with a test on both branches |
| A tight threshold refuses legitimate portal use | Derived from the client's own collision rect; a refusal is a no-op, so a false positive degrades to today's behaviour |
| `atlas-character` was not in the PRD's Service Impact table | Design §4.3 states the addition; Task 7 is the whole of it |
| A cell fails to promote in Task 12 | Diagnose against `internal/matrix/grade.go`; never hand-edit `STATUS.md` |
