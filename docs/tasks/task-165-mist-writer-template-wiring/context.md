# task-165 Context — Mist Broadcast Writer Wiring

Companion to `plan.md`. Everything here was verified against source on 2026-07-10 during planning.

## What is broken (and what is not)

Mist broadcasts die in `session.Announce` with "writer not found" because the name→opcode chain is broken on **both** sides of the `BuildWriterProducer` intersection (`libs/atlas-opcodes/producer.go`):

1. `produceWriters()` in `services/atlas-channel/atlas.com/channel/main.go` never declares `AffectedAreaCreated` / `AffectedAreaRemoved`.
2. No seed template (`services/atlas-configurations/seed-data/templates/`) has a `socket.writers` entry for either name — confirmed zero matches across all seven templates.

Already correct, must NOT be changed: the codecs (`libs/atlas-packet/field/clientbound/affected_area_created.go` / `affected_area_removed.go`), the mist consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go` — announces via `fieldpkt.AffectedAreaCreatedWriter` at :63 and `...RemovedWriter` at :71), atlas-maps mist domain, atlas-monsters producers.

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/main.go` | `produceWriters()` — add 2 constants after `fieldcb.KiteDestroyWriter` (~line 716) |
| `libs/atlas-packet/field/clientbound/affected_area_created.go` | Frozen codec; constant `AffectedAreaCreatedWriter = "AffectedAreaCreated"` at :13; `mistKey()` at bottom; v95 gate `Region()=="GMS" && MajorVersion()>=95` at :92 |
| `libs/atlas-packet/field/clientbound/affected_area_test.go` | 147 lines; add `TestAffectedAreaCreatedByteOutput` after `TestAffectedAreaCreatedFields`; `TestAffectedAreaRemovedByteOutput` (~:92-134) is the exact pattern + marker reference |
| `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` | Add 2 `socket.writers` entries each, immediately before the `SpawnDoor` entry |
| `docs/packets/evidence/<ver>/field.clientbound.FieldAffectedAreaRemoved.yaml` | Shape reference for the 5 new Created evidence records |
| `docs/packets/audits/STATUS.md` | SPAWN_MIST row :367 (❌×5 → ✅×5), REMOVE_MIST :369 (already ✅×5) — regenerated, never hand-edited |
| `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json` | Authoritative read orders (key `CAffectedAreaPool::OnAffectedAreaCreated`); read-only |
| `docs/packets/audits/VERIFYING_A_PACKET.md` | Playbook; §7 evidence-pin command, §8 matrix regen |

## Pinned facts

- **Opcodes** (STATUS.md + registries): Created/Removed = v83 `0x111`/`0x112`, v84 `0x118`/`0x119`, v87 `0x122`/`0x123`, v95 `0x148`/`0x149`, jms185 `0x126`/`0x127`.
- **IDA addresses** (`CAffectedAreaPool::OnAffectedAreaCreated`): v83 `0x431a63`, v84 `0x4326ca`, v87 `0x432f3f`, v95 `0x437ec0`, jms `0x436572`.
- **Read order**: dwId(4) nType(4) dwOwnerId(4) nSkillID(4) nSLV(1) phase(2) rcArea DecodeBuf(16, abs LTRB) [tStart(4) — **gms_v95 ONLY**] tEnd(4). 39 bytes; 43 on v95.
- **jms has NO tStart** — design §3.3 ruling, backed by the jms export ("NO leading tStart (v95-only; absent in JMS like v83)"). The PRD FR-3 line saying "v95/jms" is corrected by the design.
- **Matrix baseline**: `go run ./tools/packet-audit matrix --check` exits **0** on this branch pre-change (verified 2026-07-10). Bar for all matrix steps is clean exit 0.
- **Template shape**: `{"opCode": "0xNNN", "writer": "<Name>"}`, no `options`. Templates are read from disk by the seeder (not embedded); seeder tests in `services/atlas-configurations/.../seeder/seeder_test.go` parse them, so `go test` validates the JSON.

## Decisions locked in design (do not relitigate)

- New `TestAffectedAreaCreatedByteOutput` mirroring the Removed byte-output test (option A) — NOT retrofitting markers onto the existing WireShape/Fields tests (length/offset asserts are the "passes on 1 byte" false-pass family), NOT per-version sibling files. Existing Created tests stay untouched.
- Any read-order/byte mismatch vs the codec = **STOP and escalate** (codec changes are out of scope). Same for unresolvable IDA function/address at evidence-pin time (unresolved-fname rule — never substitute).
- gms_92 / gms_12 reduced templates: out of scope (no registry/IDB).
- Live rollout = documented manual REST patch + channel restart (rejected: one-off migration script).
- Sequencing: fixtures/evidence/matrix (Tasks 1–3) BEFORE wiring (Tasks 4–5) so surprises fire before templates are touched.

## Gotchas for the executor

- **Stale export note**: the jms_v185 export entry's `notes` still says "STRUCTURAL DEFERRAL … matches NEITHER" — that predates the abs-RECT codec fix; the `_pending.md` entry was cleared. The `calls` list is authoritative. Do not edit the export.
- **Audit-record Verdict-2 tails**: `docs/packets/audits/<ver>/FieldAffectedAreaCreated.json` has trailing "atlas: extra" rows + `FlatInvalid: true` — flat-alignment artifact (4×WriteInt32 RECT vs one DecodeBuf(16)). Not a divergence; do not regenerate these.
- **Evidence `verifies:` is manual**: `packet-audit evidence pin` writes everything except the `verifies:` list — append it by hand (playbook §7).
- **`fieldcb` vs `fieldpkt`**: main.go aliases the package `fieldcb`; the consumer aliases it `fieldpkt`. Same package (`libs/atlas-packet/field/clientbound`).
- **Test comment style**: this test file labels constructor args as `/*name*/ value` (comment BEFORE the value) — 42 is ownerId, 7 is nType in the existing Fields test.
- **JSON:API envelope** for the live PATCH: `RegisterInputHandler` rejects bare bodies; wrap as `{"data": {"type": "tenants", "id": "<tenantId>", "attributes": {...}}}` (`tenants` per `GetName()` in `services/atlas-configurations/atlas.com/configurations/tenants/rest.go:24`).
- **Live-config REST surface** (verified in `services/atlas-configurations/atlas.com/configurations/tenants/resource.go:24-28`): `GET /configurations/tenants` (list), `GET|PATCH /configurations/tenants/{tenantId}`. The channel reads its config from the CONFIGURATIONS service (`requests.RootUrl("CONFIGURATIONS")` in `services/atlas-channel/atlas.com/channel/configuration/requests.go`); `socket` lives in the tenant configuration RestModel.
- **Config does not hot-reload**: restart atlas-channel after patching (projection updates config but writer maps build at startup).
- **gh CLI**: run as `env -u GH_TOKEN -u GITHUB_TOKEN gh …` (stored hosts.yml auth works; the env token is invalid). `deploy-env` label triggers the ephemeral deploy.
- **Bake target**: only `docker buildx bake atlas-channel` is mandatory (only its Go module changed; atlas-configurations changed JSON, atlas-packet changed a test file).
- **redis-key-guard**: run `tools/redis-key-guard.sh` from the worktree root WITHOUT a global `GOWORK=off` prefix.
- **E2E values from WZ/repo only**: the AREA_POISON mob and the player mist skill for FR-5 must be resolved from the tenant's data at execution time — 2121006 in the fixture is just an input value, not a verified e2e choice.

## Dependencies between tasks

Task 2 needs Task 1's test name (evidence `verifies:`). Task 3 needs Tasks 1+2 (markers + evidence). Task 5 re-runs the matrix after template edits. Task 6 gates everything. Task 8 needs Task 7's rollout.md and a deploy environment (Steps 3–6 are BLOCKED without one — report, don't fake).
