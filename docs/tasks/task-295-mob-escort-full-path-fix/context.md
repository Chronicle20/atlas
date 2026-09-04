# task-295 — Implementation Context

Companion to [`plan.md`](plan.md); spec is [`design.md`](design.md), PRD is [`prd.md`](prd.md).

## Key files

| Path | Role |
|---|---|
| `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go` | The codec being corrected. 129 lines; both structs, `Encode`, `Decode`, and the `packet-audit:fname` doc comment live here. |
| `libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go` | The byte fixture. Carries the `packet-audit:verify` markers the coverage matrix keys on — 2 today, 3 after Task 1. |
| `services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go` | The only non-test consumer, and itself uncalled. 21 lines. |
| `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` | `socket.writers`, 152 entries, sorted by `opCode`. Missing the `0x128` row — that gap is Task 3. |
| `docs/packets/registry/gms_v92.yaml:1581-1585` | `MOB_ESCORT_FULL_PATH`, opcode 296, `fname: CMob::OnEscortFullPath`. Already present — read-only. |
| `docs/packets/evidence/{gms_v92,gms_v95,jms_v185}/monster.clientbound.MonsterMobEscortFullPath.yaml` | Evidence records. v95 and jms exist; v92 is new. Tool-written only. |
| `docs/packets/audits/STATUS.md:457` / `status.json` `rows[741]` | The coverage row. `gms_v92` is `{"state":"incomplete","note":"tier-1 without fixture; verdict 🔍","opcode":296}` today. |
| `docs/packets/ida-exports/{gms_v92,gms_v95,gms_jms_185}.json` | Confirmed to contain `CMob::OnEscortFullPath` at `0x6374c0` / `0x643d90` / `0x6efa01`. `matrix.ExportPath` maps `jms_v185` → `gms_jms_185.json` (`tools/packet-audit/internal/matrix/model.go:18-23`). |

## Module roots

- `libs/atlas-packet` — Task 1's `go build`/`go test` cwd.
- `services/atlas-channel/atlas.com/channel` — Task 2's cwd.
- Repo root — Tasks 3 and 4 (`go run ./tools/packet-audit …`).
- `tools/packet-audit` — its own module; `go test ./...` must run from inside it.

## Decisions carried from the design

- **`mode` is deleted.** The first wire `int32` is `count` (the `ZArray::_Alloc` bound in all three handlers). The old `mode` field never had a wire counterpart, so the codec loses one field and gains two (`oldDestX`, `oldDestY`).
- **Renames beyond PRD §5.** `tail`→`currentDestIndex`, `hasArrive`→`hasStopDuration`, `arriveDelay`→`stopDuration`, `hasReset`→`stopIndefinitely`, waypoint `kind`→`attr`, `extra`→`stopDuration`. Design §3 states this deviation and its FR-2 basis; a reviewer reading it as scope creep should be pointed at §3.
- **No version gate.** Byte-identical across the three routed versions.
- **The re-pinned v95/jms evidence hashes do not change**, because `evidence.FunctionHash` reads the checked-in export and no export changes here. An unchanged file is the expected outcome of Step 1 of Task 4, not a skipped step.
- **The fixture gains an `attr == 2` waypoint** so the conditional per-waypoint `stopDuration` is exercised. The current golden encodes only `kind=1` waypoints, which is how a whole conditional branch stayed unasserted.

## Dependencies and ordering

- **Tasks 1 and 2 must run back to back.** Task 1 changes the constructor arity, so `services/atlas-channel` does not compile between the two commits. Do not run `tools/verify.sh` or a repo-wide build after Task 1 alone. Task 1's own verification is deliberately scoped to `libs/atlas-packet`.
- **Task 3 must precede Task 4's matrix run.** Adding the third `packet-audit:verify` marker (Task 1) while the `gms_v92` template row is still missing can turn the cell into a template-wiring-gap conflict rather than promoting it (`tools/packet-audit/cmd/matrix_test.go:126-197`).
- Task 4 Step 5 runs the six packet gates from `.github/workflows/packet-matrix.yml`. `tools/verify.sh` does **not** run them, so a green `verify.sh` alone would not catch a matrix or fname-doc regression on this branch.

## Sizing notes

All four tasks are within the ≤6-file / single-service budget. Task 4 lists six files but four of them are written by tooling, not edited — the implementer's hand-edit surface there is zero.

Tasks 1 and 2 could have been one atomic task (design §9 calls steps 1-3 "one logical unit" and the branch a two-commit branch). They are split anyway because they sit in different modules, and the split keeps each implementer's verification module-local. The cost is the transient non-compiling repo state between them, called out above.
