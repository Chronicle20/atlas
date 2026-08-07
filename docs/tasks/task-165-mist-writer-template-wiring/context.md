# task-165 Context — Mist Broadcast Writer Wiring

Companion to `plan.md`. Originally verified 2026-07-10; **re-verified against source 2026-08-07** after the branch was merged with main and rescoped to all supported versions (PRD v2 / design rev 2026-08-07).

## What is broken (and what is not)

Mist broadcasts die in `session.Announce` with "writer not found" because the name→opcode chain is broken on **both** sides of the `BuildWriterProducer` intersection (`libs/atlas-opcodes/producer.go`):

1. `produceWriters()` in `services/atlas-channel/atlas.com/channel/main.go` never declares `AffectedAreaCreated` / `AffectedAreaRemoved`.
2. No seed template (`services/atlas-configurations/seed-data/templates/`) has a `socket.writers` entry for either name — confirmed zero matches across **all eleven** templates (2026-08-07).

Already correct, must NOT be changed: the codecs (`libs/atlas-packet/field/clientbound/affected_area_created.go` / `affected_area_removed.go`), the mist consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go` — announces via `fieldpkt.AffectedAreaCreatedWriter` at :64 and `...RemovedWriter` at :72), atlas-maps mist domain, atlas-monsters producers.

## What changed while this branch sat idle (254 commits)

The branch was cut when the repo tracked five versions and had seven templates. It now tracks nine matrix columns and eleven templates. Four facts the original docs got wrong as a result:

1. **v61/v72/v79 columns exist** (task-113 legacy bring-up). SPAWN_MIST is already ✅ there; **REMOVE_MIST is only 🟡ᶠ** — the verification gap moved rather than shrank.
2. **The v79 SPAWN_MIST ✅ is soft.** Its evidence points at `TestAffectedAreaCreatedWireShape`, whose only v79 assertion is `if len(b) != 39` — a length check with no byte values pinned. This is the exact false-pass family the v1 design rejected as "approach B" for the new cells, while it was already load-bearing for an existing one.
3. **gms_92 and gms_12 now have packet-named IDBs** (~950 and ~230 functions). The v1 non-goal "no registry or IDB to source opcodes from" is stale.
4. **v48's ⬜ is an artifact, not a ruling.** `docs/packets/ida-exports/gms_v48.json` carries `{"unresolved": true, "comment": "function not found in IDB"}` for both mist functions, and the v48 registry self-describes as a "clean PARTIAL" with deep pool leaves deferred. Nothing asserts absence — and per `feature-na-evidence.yaml`'s own bar, a failed name search is explicitly not evidence.

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/main.go` | `produceWriters()` at :608 — add 2 constants after `fieldcb.KiteDestroyWriter` (:718) |
| `libs/atlas-packet/field/clientbound/affected_area_created.go` | Frozen codec; `AffectedAreaCreatedWriter = "AffectedAreaCreated"` at :13; `mistKey()` at bottom; v95 gate `Region()=="GMS" && MajorVersion()>=95` at :92 |
| `libs/atlas-packet/field/clientbound/affected_area_removed.go` | Frozen codec; **version-independent** — `Encode(l, _ context.Context)` at :35 writes one LE uint32 on every version |
| `libs/atlas-packet/field/clientbound/affected_area_test.go` | Consolidate 3 Created tests into one table-driven `TestAffectedAreaCreatedByteOutput` (8 versions); extend `TestAffectedAreaRemovedByteOutput` (:184) with 3 rows |
| `services/atlas-configurations/seed-data/templates/template_*.json` | 8 Tier A templates get 2 `socket.writers` entries each, before the `SpawnDoor` entry; 3 Tier B templates conditional on discovery |
| `docs/packets/evidence/<ver>/field.clientbound.FieldAffectedArea*.yaml` | 8 new records + 3 `verifies:` re-points |
| `docs/packets/audits/STATUS.md` | SPAWN_MIST :346, REMOVE_MIST :347 — regenerated, never hand-edited |
| `docs/packets/ida-exports/*.json` | Authoritative read orders (key `CAffectedAreaPool::OnAffectedArea{Created,Removed}`); read-only |
| `docs/packets/TEMPLATE_CONVENTIONS.md` | Ascending-opcode-order rule; `tools/template-opcode-order-guard.sh` enforces it in CI |
| `docs/packets/feature-na-evidence.yaml` | The n-a bar, if Tier B discovery yields Outcome B for gms_48 |
| `docs/packets/audits/VERIFYING_A_PACKET.md` | Playbook; §7 evidence-pin command, §8 matrix regen |

## Pinned facts

- **Tier A opcodes** (Created/Removed, from `docs/packets/registry/<ver>.yaml`): v61 `0x0D2`/`0x0D3`, v72 `0x0F3`/`0x0F4`, v79 `0x0FB`/`0x0FC`, v83 `0x111`/`0x112`, v84 `0x118`/`0x119`, v87 `0x122`/`0x123`, v95 `0x148`/`0x149`, jms185 `0x126`/`0x127`.
- **Band layout is uniform in all eight**: `DropDestroy` at (SPAWN_MIST − 4), 2-slot gap, `SpawnDoor` at (+2), `RemoveDoor` at (+3). Both new entries always slot immediately before `SpawnDoor`.
- **Created IDA addresses**: v61 `0x423edc`, v72 `0x42e36c`, v79 `0x42e7fc`, v83 `0x431a63`, v84 `0x4326ca`, v87 `0x432f3f`, v95 `0x437ec0`, jms `0x436572`.
- **Removed IDA addresses**: v61 `0x4246b0`, v72 `0x42ec4e`, v79 `0x42f0de`, v83 `0x43234d`, v84 `0x432fb4`, v87 `0x43388c`, v95 `0x4360a0`, jms `0x436eda`.
- **Created read order**: dwId(4) nType(4) dwOwnerId(4) nSkillID(4) nSLV(1) phase(2) rcArea DecodeBuf(16, abs LTRB) [tStart(4) — **gms_v95 ONLY**] tEnd(4). 39 bytes; 43 on v95.
- **Removed read order**: a single `Decode4 dwId` on every version — so extending its fixture to v61/v72/v79 is three table rows, not new derivation.
- **jms has NO tStart** — backed by the jms export ("NO leading tStart (v95-only; absent in JMS like v83)"). The v1 PRD's "v95/jms" phrasing was wrong and is corrected in both PRD v2 and the design.
- **Matrix baseline**: `go run ./tools/packet-audit matrix --check` exits **0** on this branch post-merge (re-verified 2026-08-07). Bar for all matrix steps is clean exit 0.
- **Template shape**: `{"opCode": "0xNNN", "writer": "<Name>"}`, no `options`. Templates are read from disk by the seeder (not embedded); seeder tests parse them, so `go test` validates the JSON.
- **IDA sessions** (`idb_list`, 2026-08-07): v48, v61, v72, v79, v83, v84, v87, v92, v95, jms185 open. **No v12 session** — open it in Task 8. Match sessions by binary NAME, never a remembered port (the instance set rotates).

## Decisions locked in design (do not relitigate)

- **No derived-unverified opcodes.** A template is wired only when its opcode was read from that version's IDB. The task-088 summon precedent (interpolating `0xC2–0xC7` for v92, `0xAF–0xB4` for v12 and flagging them "confirm against capture") is explicitly rejected here — a wrong opcode silently drops the packet, i.e. reproduces the bug being fixed. Tier B Outcome C is "leave unwired and document", never "guess".
- One table-driven `TestAffectedAreaCreatedByteOutput` covering all 8 Tier A versions, replacing the per-version `...V61`/`...V72` tests. `WireShape`/`Fields` stay as guards but **carry no verify markers** — the v79 marker moves off `WireShape`.
- Any read-order/byte mismatch vs the codec on an already-✅ version = **STOP and escalate**. Tier B may add an *additive* version gate (`MajorAtLeast` idiom, never raw `> N`) but must not move a byte on any ✅ version.
- Unresolvable IDA function/address at evidence-pin time = stop-and-ask (unresolved-fname rule — never substitute a hash or address).
- Live rollout = documented manual REST patch + channel restart (rejected: one-off migration script).
- Sequencing: Tier A fixtures/evidence/matrix (Tasks 1–4) BEFORE wiring (Tasks 5–6), then Tier B (Task 8). Task 7 is an explicit shippable checkpoint — Tier B stalling must not block the eight ready versions.
- Standing up full packet-audit columns for gms_92/gms_12 is out of scope; the task needs two opcodes, not a version bring-up.

## Gotchas for the executor

- **Stale export note**: the jms_v185 export entry's `notes` still says "STRUCTURAL DEFERRAL … matches NEITHER" — that predates the abs-RECT codec fix; the `_pending.md` entry was cleared. The `calls` list is authoritative. Do not edit the export.
- **Audit-record Verdict-2 tails**: `docs/packets/audits/<ver>/FieldAffectedAreaCreated.json` has trailing "atlas: extra" rows + `FlatInvalid: true` — flat-alignment artifact (4×WriteInt32 RECT vs one DecodeBuf(16)). Not a divergence; do not regenerate these.
- **`unresolved: true` in the v48 export is not absence evidence** — it is the failed name lookup that made this task believe v48 was n-a. Treat it as "undiscovered".
- **Evidence `verifies:` is manual**: `packet-audit evidence pin` writes everything except the `verifies:` list — append it by hand (playbook §7).
- **`fieldcb` vs `fieldpkt`**: main.go aliases the package `fieldcb`; the consumer aliases it `fieldpkt`. Same package (`libs/atlas-packet/field/clientbound`).
- **Test comment style**: this file labels constructor args as `/*name*/ value` (comment BEFORE the value) — 42 is ownerId, 7 is nType in the existing Fields test.
- **Opcode literal width varies by template** (`0x0D2` vs `0xD2`) — copy the neighbouring `SpawnDoor` entry's style. The order guard parses the value, not the width.
- **Template guards are CI jobs**: `tools/template-opcode-order-guard.sh` and `tools/template-movement-types-guard.sh` must both exit 0 after any template edit.
- **JSON:API envelope** for the live PATCH: `RegisterInputHandler` rejects bare bodies; wrap as `{"data": {"type": "tenants", "id": "<tenantId>", "attributes": {...}}}` (`tenants` per `GetName()` in `services/atlas-configurations/atlas.com/configurations/tenants/rest.go:24`).
- **Live-config REST surface**: `GET /configurations/tenants` (list), `GET|PATCH /configurations/tenants/{tenantId}`. The channel reads its config from the CONFIGURATIONS service (`requests.RootUrl("CONFIGURATIONS")`); `socket` lives in the tenant configuration RestModel.
- **Config does not hot-reload**: restart atlas-channel after patching (projection updates config but writer maps build at startup).
- **gh CLI**: run as `env -u GH_TOKEN -u GITHUB_TOKEN gh …` (stored hosts.yml auth works; the env token is invalid). `deploy-env` label triggers the ephemeral deploy.
- **Bake target**: only `docker buildx bake atlas-channel` is mandatory (only its Go module changed; atlas-configurations changed JSON, atlas-packet changed a test file).
- **redis-key-guard**: run `tools/redis-key-guard.sh` from the worktree root WITHOUT a global `GOWORK=off` prefix.
- **E2E values from WZ/repo only**, and **skill ids are version-specific** — the AREA_POISON mob and the player mist skill must be resolved from each tenant's own data at execution time. 2121006 in the fixture is an input value, not a verified e2e choice, and is not assumed valid on v61/v48.
- **IDA discipline for Task 8**: name symbols as you reverse them; disasm is authoritative over Hex-Rays for SEH functions; never rename on adjacency alone; align dispatchers by structure, never by opcode number (drift is non-uniform on every legacy pair).

## Dependencies between tasks

Task 3 needs Tasks 1+2 (test names for `verifies:`). Task 4 needs Tasks 1–3 (markers + evidence). Task 6 re-runs the matrix after template edits. Task 7 is the Tier A checkpoint. Task 8 (Tier B) is independent of Tasks 1–7 and abortable — its outcome determines whether Tasks 10/11 cover 8 or up to 11 versions. Task 9 gates everything. Task 10 must be written after Task 8 so its version table reflects what was actually wired. Task 11 Steps 3–6 need a deploy environment (BLOCKED without one — report, don't fake).
