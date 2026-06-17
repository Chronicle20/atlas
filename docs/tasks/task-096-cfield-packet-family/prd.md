# CField Map/Field Packet Family — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-14
---

## 1. Overview

The `tools/packet-audit` coverage matrix (`docs/packets/audits/STATUS.md`) tracks every
MapleStory wire packet × supported client version (gms_v83, gms_v84, gms_v87, gms_v95,
jms_v185). task-092 closed out the MOB/MONSTER family. This task addresses the next
coherent owner-class family: **`CField` — the map/field packets** (map transfer, field
chat, field-object/obstacle state, quests/timers, GM events, MTS, and the field-based
mini-games: snowball, coconut, guild boss, tournament, wedding, Ariant arena, sheep
ranch, pyramid/massacre, witch tower, conti-move).

There are **75 `CField*` ops with at least one ❌ cell** (318 ❌ cells), split as 45
core `CField::` ops + 30 across the `CField_*` mini-game subclasses (see
`structures/cfield-ops.md` for the full grouped work-list). The batch covers all 75.

**Critical framing — this is verify-existing AND implement-new, not all greenfield.**
A material fraction of these ops are *already implemented but unverified*: the
`libs/atlas-packet/chat/clientbound/` package (general/multi/whisper/world_message) and
`libs/atlas-packet/field/clientbound/` package (effect, effect_weather, clock,
affected_area, kite_destroy, …) already exist and several have wired channel handlers.
They show ❌ only because no per-op codec is linked to the registry op in
`candidatesFromFName`, not because the codec is missing (the exact trap task-092 flagged
— see `docs/packets/IMPLEMENTING_A_PACKET.md` Step 0). Every op MUST be triaged
verify-vs-implement before any codec is written, to avoid shipping duplicates.

## 2. Goals

Primary goals:
- Drive every applicable `CField*` cell in the coverage matrix to ✅ **verified**, with
  genuine version-absences marked ⬜ **n/a** (VERSION-ABSENT, IDB-evidenced).
- For each op: a linked codec (verify the existing one, or add a new one), a byte-fixture
  test with a `// packet-audit:verify` marker per applicable version, pinned tier-1
  evidence, an audit report, and a route in each applicable seed template.
- `go run ./tools/packet-audit matrix --check` exits 0 with zero new conflicts.

Non-goals:
- Gameplay behavior (acting on decoded serverbound packets; orchestrating minigames,
  weddings, tournaments, MTS sales). Handlers decode-and-log only; clientbound writers
  are intentional uncalled seams (design D2 from task-092).
- Live-tenant config changes (seed templates only; live PATCH/rollout is documented in
  `deploy-notes.md`, not applied).
- Non-`CField` families (CWvsContext, CUserLocal, CCashShop, etc. — separate batches).
- Reclassifying a live cell to ⬜ to "close" it — only genuine IDB-evidenced absence.

## 3. User Stories

- As an Atlas maintainer, I want every CField map/field packet decoded/encoded faithfully
  per version so map transitions, field chat, field-object state, and minigames behave
  correctly across all supported clients.
- As a packet-coverage owner, I want the CField family fully ✅/⬜ in the matrix so the
  remaining ❌ backlog shrinks by one whole owner-class family with no silent gaps.
- As the next implementer, I want each op triaged verify-vs-implement so existing codecs
  are reused (not duplicated) and only genuinely-missing packets get new codecs.

## 4. Functional Requirements

### Phase 0 — Triage (per op, before any codec work)
For each of the 75 ops, classify into:
- **(A) already implemented → verify only**: a codec exists (in `field/`, `chat/`, or
  elsewhere) and a channel handler/writer may be wired. Work = link it (`candidatesFromFName`
  case → existing struct, or a thin wrapper for a shared model), add marker + evidence +
  report, ensure the seed template routes it. NO new codec.
- **(B) missing → implement-new**: follow the `IMPLEMENTING_A_PACKET.md` four-step recipe
  (derive → model+codec → wire → verify).
- **(C) unresolved / registry-gap**: the `IDA_0X09x/0Ax/0Bx` rows (fname `OnStalkResult`/
  `OnFootHoldInfo`, no clean op-name) and any `(no fname)` CField rows. Resolve the
  op-name/fname against the IDB first (registry correction), or mark VERSION-ABSENT with
  IDB evidence. Treat per the producible-prerequisite-vs-genuine-blocker bar in
  `VERIFYING_A_PACKET.md` — derive the name; only defer on a true hard blocker.

### Per-op work (recipe by classification)
- **Derivation (IDA):** decompile the `CField`/`CField_*` send/recv function per applicable
  version; record the ordered field list + per-version deltas + export address into
  `structures/<version>.md#<OP>`. Use the IDA-harvest subagent workflow (one IDB at a time;
  `select_instance` is shared global state). jms uses the clean `*_U_DEVM` build, not the
  SMC retail dump.
- **Codec:** immutable struct in the owner-class package (`field/{clientbound,serverbound}`;
  chat ops stay in `chat/`), version-branched on `tenant.MustFromContext(ctx)`
  (`t.Region()`, `t.MajorAtLeast(n)`); reuse shared `model/` sub-structs.
- **Wire:** clientbound → writer `Body` + `produceWriters()`; serverbound → handler
  (decode+log) + `produceHandlers()`; route the per-version opcode in every applicable
  seed template with a `validator` (LoggedInValidator default).
- **Verify:** round-trip across `pt.Variants` + golden-byte for the baseline; one
  `packet-audit:verify` marker per applicable version; pin tier-1 evidence; generate the
  audit report (root run with `-ida-source <export>`); regenerate the matrix.

### Direction & version handling
- Both directions: clientbound `On*` writers dominate; a few serverbound `Send*`
  (e.g. `CField::SendChatMsg`/`SendChatMsgSlash` for GENERAL_CHAT/ADMIN_CHAT/SUE_CHARACTER).
- Per-version opcodes come from `docs/packets/registry/<version>.yaml` (read per file; the
  COutPacket/recv-dispatch opcode is ground truth, registry csv values can be off-by-one).
- Genuine version-absence (op not in a version's dispatcher/registry) → ⬜ with evidence.

### Export / report mechanics (reuse task-092 findings)
- The export is NOT idempotent — never overwrite a committed export; surgically splice
  only the needed fname entries (`VERIFYING_A_PACKET.md` §10). Strip the misclassified
  `COutPacket`-ctor `Delegate` artifact when report-gen descent fails on it.
- A `routedElsewhere && !routed` conflict = a real template-wiring gap; route it (if the
  tenant should support it) or don't claim the cell.

## 5. API Surface

N/A — wire packets, not REST. The "surface" is the per-version opcode set per op, sourced
from `docs/packets/registry/<version>.yaml` and produced into `deploy-notes.md` (per-version
handler/writer opcode tables in live-tenant PATCH shape, as in task-092).

## 6. Data Model

No DB entities. Codec structs (immutable, private fields + getters + `Operation()`/`String()`/
`Encode`/`Decode`) live in `libs/atlas-packet/field/{clientbound,serverbound}` (and existing
`chat/` for chat ops). Shared sub-structs reused from `libs/atlas-packet/model/`. No `go.mod`
change expected (atlas-packet is a workspace member).

## 7. Service Impact

- `libs/atlas-packet` — new/linked codecs under `field/` (+ `chat/` for chat ops) and `_test.go`.
- `services/atlas-channel` — `socket/writer/`, `socket/handler/`, `main.go` registrations.
- `services/atlas-configurations` — `seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` routes (kept ascending by opCode).
- `docs/packets` — `registry/*.yaml` (fname/opcode corrections), `ida-exports/*.json` (surgical splices), `evidence/`, `audits/STATUS.md`+`status.json`+per-writer reports.
- `docs/tasks/task-096-cfield-packet-family` — design/plan/structures/deploy-notes/audit artifacts.

## 8. Non-Functional Requirements

- **Multi-tenancy:** every version delta gated via tenant ctx; never hardcode opcodes/bytes.
- **Verification gates (CLAUDE.md):** `go test -race ./...` + `go vet ./...` clean in every
  changed module; `go build ./...` for atlas-channel/atlas-configurations; `GOWORK=off`
  `tools/redis-key-guard.sh` clean; `docker buildx bake` only if a `go.mod` changes (not
  expected); `packet-audit matrix --check` exit 0.
- **Validator-mandatory:** every `socket.handlers` entry carries a validator (BuildHandlerMap
  silently drops validator-less entries).
- **No knowingly-wrong codec:** SMC/undecompilable/runtime-config-gated/genuinely-absent ops
  are documented (⬜ or honest ❌), never faked.
- **Observability:** decode+log handlers log `[Op] read [String()]` at debug.

## 9. Open Questions

- **Unresolved foothold/stalk rows** (`IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1`, fname
  `OnStalkResult`/`OnFootHoldInfo`): are these real per-version CField ops needing op-name
  resolution, mode-dispatch sub-ops, or version-absent cruft? Resolve in Phase 0/1.
- **Chat ops package placement**: MULTICHAT/WHISPER/GENERAL_CHAT/SPOUSE_CHAT/ADMIN_CHAT/
  SUE_CHARACTER are `CField`-owned but the existing codecs live in `chat/`. Keep them in
  `chat/` (link there) rather than moving to `field/` — confirm during design.
- **Minigame value/coverage**: the `CField_*` minigames (tournament, wedding, snowball,
  etc.) are low-traffic, self-contained sub-features — verify their wire shape but expect
  some VERSION-ABSENT cells (several are GMS-event-only).
- **MTS_OPERATION/MTS_OPERATION2** (`OnCharacterSale`) — two ops, one fname; confirm
  whether they are two modes of one structure or two distinct packets.

## 10. Acceptance Criteria

- [ ] Every applicable `CField*` cell in `STATUS.md` is ✅; genuine absences are ⬜ with
      VERSION-ABSENT evidence; zero 🟥 conflicts attributable to this task.
- [ ] `go run ./tools/packet-audit matrix --check` exits 0 (no new orphan/dangling/stale/
      drift lines mentioning a CField packet).
- [ ] Each verified op has: linked codec + `_test.go` (round-trip + golden) + verify
      marker(s) + pinned evidence + audit report + seed-template route(s).
- [ ] No duplicate codec for an already-implemented op (Phase-0 triage honored).
- [ ] `go test -race`/`go vet` clean (libs/atlas-packet, atlas-channel, tools/packet-audit);
      redis-key-guard clean; seed-template JSON valid + handler/writer arrays ascending by opCode.
- [ ] `deploy-notes.md` with per-version opcode tables + rollout checklist.
- [ ] Code review (plan-adherence + backend-guidelines) green before PR.
