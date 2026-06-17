# CField Map/Field Packet Family — Byte-Plumbing Batch 2 — Design

Status: Approved (design phase)
Created: 2026-06-14
PRD: `docs/tasks/task-096-cfield-packet-family/prd.md`
Precedent: task-092 (MOB/MONSTER family) — same four-layer recipe, codified in
`docs/packets/IMPLEMENTING_A_PACKET.md` / `docs/packets/VERIFYING_A_PACKET.md`.

---

## 1. Scope & Locked Decisions

This task drives every applicable `CField*` map/field coverage-matrix cell to ✅ **verified**
(or ⬜ **n/a** with IDB-evidenced version-absence) across the five supported versions
(gms_v83, gms_v84, gms_v87, gms_v95, jms_v185). The family is **75 ops with at least one ❌
cell / 318 ❌ cells**: 45 core `CField::` ops + 30 across the `CField_*` minigame subclasses
(`structures/cfield-ops.md`). **Gameplay behavior is out of scope** — handlers decode-and-log,
clientbound writers are registered uncalled seams (the task-092 D2 convention, reaffirmed below).

**Critical framing carried from the PRD: this is verify-existing AND implement-new.** A
material fraction of these ops already have codecs (`libs/atlas-packet/field/*`,
`libs/atlas-packet/chat/*`) and some have wired channel handlers; they read ❌ only because no
per-op codec is linked to the registry op in `candidatesFromFName`, not because a codec is
missing (`IMPLEMENTING_A_PACKET.md` Step 0). Every op is triaged verify-vs-implement *before*
any codec is written so we never ship a duplicate.

Four decisions were settled with the user before this document:

| # | Decision | Resolution |
|---|----------|-----------|
| D1 | Batch packaging — split minigames out, or one batch? | **One task / one branch / one PR for all 75 ops, with cluster-gated commits** (core CField first, then each `CField_*` minigame block). One whole owner-class family closed; reviews stay bounded. |
| D2 | Per-op deliverable scope | **Codec + route + byte-test only** (task-092 D2). Serverbound handlers decode-and-log; clientbound writers are registered seams with no emitter. No gameplay/minigame orchestration, no producer triggers. |
| D3 | Chat-op package placement (`GENERAL_CHAT`, `MULTICHAT`, `WHISPER`, `SPOUSE_CHAT`, `ADMIN_CHAT`, `SUE_CHARACTER`, slash variants) | **Relocate the `CField::`-owned chat codecs from `chat/` into `field/{clientbound,serverbound}`** (move, not rewrite). Non-`CField` chat codecs stay in `chat/`. See §5. |
| D4 | Phase-0 triage rigor | **Triage is a committed artifact** (`structures/triage.md`, an A/B/C table per op) produced before codec work; unresolved fnames are **derived from the IDB (name them), not deferred** — defer only on a true hard blocker (`VERIFYING_A_PACKET.md` producible-prerequisite bar). |

**Why D3 is the right call (not just churn for its own sake):** `field/` is a **tier-1 prefix**
in `docs/packets/evidence/tiers.yaml`; `chat/` is **not**. Relocating the CField chat codecs
into `field/` brings them under the same tier-1 prefix as the rest of this family, so the
flat-diff verdict is advisory and the byte-fixture test is authoritative — exactly the regime
the other 69 CField ops live under. Leaving them in `chat/` would make six members of this
family verify under a different (non-tier-1-prefix) regime than their siblings.

---

## 2. Architecture

Identical four-layer shape to task-092 — only the byte layouts and per-version applicability
differ. Each layer has one responsibility and a defined seam to the next.

```
┌─ libs/atlas-packet/field/<dir>/<op>.go (chat ops relocate here too) ────┐
│  Immutable model (private fields + getters + constructor).             │
│  Operation() string  → writer/handler NAME (the wiring key)            │
│  Encode(l, ctx)(opts) []byte   — clientbound payload (no opcode)       │
│  Decode(l, ctx)(r, opts)       — serverbound parse                     │
│  Version-branches on tenant.MustFromContext(ctx) (Region/MajorAtLeast) │
│  Shared sub-structs reused from libs/atlas-packet/model/               │
└───────────────┬─────────────────────────────────────────────────────────┘
                │  Operation() name
┌───────────────▼─ services/atlas-channel ───────────────────────────────┐
│  clientbound:  produceWriters() += NAME ; socket/writer/<op>.go Body   │
│  serverbound:  produceHandlers()[NAME] = <op>Func (decode + log) ;     │
│                validator = LoggedInValidator (NoOp only for conn-level) │
└───────────────┬─────────────────────────────────────────────────────────┘
                │  NAME ↔ opcode binding (per version)
┌───────────────▼─ services/atlas-configurations seed templates (×5) ─────┐
│  socket.handlers[] += {opCode, validator, handler:NAME}  (asc by op)   │
│  socket.writers[]  += {opCode, writer:NAME}              (asc by op)   │
│  opCode value comes from docs/packets/registry/<version>.yaml          │
└───────────────┬─────────────────────────────────────────────────────────┘
                │  applicability (registry ⇄ template ⇄ code)
┌───────────────▼─ tools/packet-audit matrix ────────────────────────────┐
│  scans verify markers + evidence + templates → grades each cell        │
│  `matrix` regenerates STATUS.md/status.json ; `matrix --check` gates   │
└──────────────────────────────────────────────────────────────────────────┘
```

**Runtime data-flow asymmetry is intentional and unchanged from task-092:**

- **Serverbound** ops are live end-to-end: client sends → channel handler decodes + logs
  (`l.Debugf("[%s] read [%s]", p.Operation(), p.String())`), no "unhandled op" warning, no
  action. Genuinely useful; not dead code.
- **Clientbound** ops get a registered writer with **no emitter** in this task — the `Body`
  helper + `produceWriters()` entry exist so a future behavior task calls
  `session.Announce(...)(NAME)(Body(...))` without re-wiring. The one knowingly-landed uncalled
  seam, already documented in `IMPLEMENTING_A_PACKET.md` so reviewers don't flag it.

What the matrix grader needs for ✅ (tier-1, since `field/` is a tier-1 prefix): an immutable
codec with both `Encode` and `Decode`; a per-version byte-fixture test with a
`// packet-audit:verify packet=… version=… ida=0x…` marker; a pinned evidence record whose
`decompile_sha256` matches the current IDA export. Seed-template routes + channel registration
aren't required by the grader but are required by PRD acceptance and keep
registry⇄template⇄code applicability conflict-free (a missing/extra route is a `conflict`,
a `--check` blocker).

---

## 3. Phase 0 — Triage (committed artifact, D4)

Before any codec is touched, produce `structures/triage.md`: one row per op classifying it
**A / B / C** with file:line / IDB evidence. This is the load-bearing anti-duplication step and
the first commit of the task.

| Class | Meaning | Work |
|---|---|---|
| **A — verify-only** | A codec already exists (in `field/`, `chat/`, or elsewhere) and may have a wired handler/writer. | Link it (`candidatesFromFName` case → existing struct, or a thin wrapper for a shared model), add marker + evidence + report, ensure each seed template routes it. **No new codec.** For CField-owned chat codecs, A-work also includes the §5 relocation. |
| **B — implement-new** | No codec exists. | Full `IMPLEMENTING_A_PACKET.md` four-step recipe (derive → model+codec → wire → verify). |
| **C — unresolved / registry-gap** | `IDA_0X09x/0Ax/0Bx` rows (fname `OnStalkResult`/`OnFootHoldInfo`/`OnRequestFootHoldInfo`, no clean op-name), `(no fname)` rows, and the ambiguous `MTS_OPERATION/2`, `USE_DOOR`, `GUILD_OPERATION`, minigame `Update`/`BasicActionAttack`/`Init` rows. | **Resolve the op-name/fname against the IDB first** (registry correction, provenance `manual`/`ida-discovered` with citation), then re-classify to A or B. Mark ⬜ VERSION-ABSENT only with IDB evidence of a genuine dispatcher gap. Derive the name — defer only on a true hard blocker (e.g. SMC-only / undecompilable). |

The triage table records, per op: direction (CB/SB/?), owner class, current codec path (if any),
classification, and the resolution note for C-rows. The execution plan (Phase 3) consumes this
table directly as its work-list. **Honoring this table is an acceptance criterion** — no op is
implemented without a triage row, and no duplicate codec is created for an A-row.

### C-row resolution targets (PRD §9 open questions)

- **Foothold/stalk cluster** (`IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1`): decompile
  `OnStalkResult` / `OnFootHoldInfo` / `OnRequestFootHoldInfo` per applicable version, derive
  the real op-name + structure, and decide whether each `IDA_0x…` row is (a) a mislabeled alias
  of a named CField op already in the list (collapse into it), (b) a distinct per-version op
  needing its own codec (B), or (c) genuinely version-absent (⬜). Resolve in triage; do not
  carry `IDA_0x…` placeholders into code.
- **`MTS_OPERATION` / `MTS_OPERATION2`** (both `OnCharacterSale`): decompile `OnCharacterSale`
  to determine whether they are two **modes** of one structure (one model, mode-branched, two
  registry ops → two routes) or two **distinct** packets (two models). Record the verdict.
- **`USE_DOOR`** (`CField::TryEnterTownPortal`) and **`GUILD_OPERATION`**
  (`CField::InputGuildName`, already `PKT`): resolve direction (the worklist marks them `?`) and
  whether an existing codec covers them (GUILD_OPERATION is `PKT` → likely A).
- **Minigame `?` rows** (`SNOWBALL`/`LEFT_KNOCKBACK`/`COCONUT`/`GUILD_BOSS`/`CONTI_MOVE` via
  `Update`/`BasicActionAttack`/`Init`): these fnames are state/update methods, not packet
  send/recv sites — derive the actual send-site fname (or confirm serverbound recv) before
  classifying.

---

## 4. Per-Op Recipe

The recipe is already documented verbatim in `docs/packets/IMPLEMENTING_A_PACKET.md` (created by
task-092) and `docs/packets/VERIFYING_A_PACKET.md`. task-096 **follows** it; it does not
re-author it. Summary of the four steps with CField-specific notes:

1. **Derive (IDA).** `select_instance(port)` then `decompile` the registry `fname`, descending
   into helper reads/writes. Record the ordered field list (`Decode1/2/4/Str/Buffer`) + every
   per-version delta + the export address into `structures/<version>.md#<OP>`. Use the IDA-harvest
   subagent workflow — **one IDB at a time** (`select_instance` is shared global state); batch
   all ops for a given IDB before switching. Ports: **v83=13337, v87=13338, v95=13339,
   jms=13340, v84=13341.** jms uses the clean `*_U_DEVM` build, never the SMC retail dump.
   Guards: confirm every `fname` against the IDB before coding (stale-registry-fname class, per
   task-085/092); the COutPacket opcode is ground truth (registry csv can be off-by-one);
   **v84 ≡ v83** below the shifted opcode-table region — use `MajorAtLeast(87)` gates, never `>83`,
   and do not invent v84 deltas the IDB doesn't show.
2. **Model + codec.** Immutable struct in `field/{clientbound,serverbound}` (relocated chat ops
   included), private fields + getters + constructor, `Operation()`/`String()`/`Encode`/`Decode`.
   Per-version variants live **inside** `Encode`/`Decode` via `tenant.MustFromContext(ctx)`
   (`t.Region()`, `t.MajorAtLeast(n)`), not as separate types unless a version diverges enough to
   warrant its own model. Reuse `libs/atlas-packet/model/` sub-structs (`Position`, `Movement`,
   etc.) — never re-derive them.
3. **Wire.** clientbound → `produceWriters() += NAME` + `socket/writer/<op>.go` `Body` helper;
   serverbound → `produceHandlers()[NAME] = …Func` + `socket/handler/<op>.go` decode+log. Route
   the per-version opcode in **every applicable** seed template with a **validator**
   (`LoggedInValidator` default; `NoOpValidator` only for connection-level). A validator-less
   handler entry is silently dropped by `BuildHandlerMap` — mandatory, not cosmetic. Keep
   `handlers`/`writers` arrays ascending by opCode.
4. **Verify.** `_test.go` round-trip across `test.Variants` + a golden-byte assertion for the
   v83 baseline on any mask/mode-driven packet (round-trip proves symmetry, not byte-exactness
   vs the client). One `// packet-audit:verify` marker per applicable version. Pin tier-1
   evidence (`packet-audit evidence pin …`, then add `verifies:`). Regenerate
   (`packet-audit matrix`) and gate (`matrix --check` exit 0). Export is **non-idempotent** —
   surgically splice only needed fname entries (`VERIFYING_A_PACKET.md` §10), strip the
   misclassified `COutPacket`-ctor `Delegate` artifact when report-gen descent fails on it. A
   `routedElsewhere && !routed` conflict = a real template-wiring gap; route it (if the tenant
   should support it) or don't claim the cell.

---

## 5. Chat-Op Relocation Plan (D3)

The six CField-owned chat ops have existing codecs under `libs/atlas-packet/chat/`. They move to
`field/`; this is a **move-not-rewrite**, executed as its own early commit (after triage, before
new codec work) so any regression is isolated and bisectable.

**Move rule (precise):** a chat codec file moves to `field/<dir>/` **iff every registry op it
serves is `CField::`-owned.** Confirmed owners from `registry/gms_v83.yaml`:

| Codec (current) | Serves | Owner | Action |
|---|---|---|---|
| `chat/serverbound/general.go` | `GENERAL_CHAT` | `CField::SendChatMsg` | **move** → `field/serverbound/` |
| `chat/clientbound/multi.go` | `MULTICHAT` | `CField::OnGroupMessage` | **move** → `field/clientbound/` |
| `chat/clientbound/whisper.go` | `WHISPER` | `CField::OnWhisper` | **move** → `field/clientbound/` |
| `chat/{clientbound}/world_message*.go` | `BROADCAST_MSG` etc. | `CWvsContext::OnBroadcastMsg` | **stay** in `chat/` (not CField) |
| (chat ops with `CUser::OnChat` / `CUserLocal::OnChatMsg` owners — e.g. `CHATTEXT`) | — | `CUser*` | **stay** in `chat/` (not CField) |
| `SPOUSE_CHAT` (`OnCoupleMessage`), `ADMIN_CHAT`/`SUE_CHARACTER`/slash variants (`SendChatMsgSlash`) | new | `CField::` | **implement new** in `field/` |

A file that is **shared** across a CField op and a non-CField op is **not** moved (would orphan
the non-CField owner's package convention) — it stays and is **linked** in place. Phase-0 triage
confirms, per file, that no non-CField op depends on it before the move; any shared file is
recorded as a "link-in-place" exception in `triage.md`.

**Mechanics of the move:**
- `git mv` the `.go` and its `_test.go` together; rename the package clause to the destination
  package; keep the exported `Operation()` NAME constants byte-identical (the wiring key must not
  change). Preserve existing `// packet-audit:verify` markers and evidence references.
- Repoint every importer — primarily `services/atlas-channel/socket/writer/*` and
  `socket/handler/*` and `main.go` `produceWriters()/produceHandlers()` — from the `chat/…`
  import path to `field/…`. `go build ./...` + `go test -race ./...` for atlas-channel and
  atlas-packet must stay green across the move commit.
- After the move, re-run `packet-audit matrix` — any cell already ✅ via the old path must remain
  ✅ (evidence `verifies:` may need its packet id updated to the new dotted path; treat a
  drop-to-❌ as a move regression to fix in the same commit, never as an accepted loss).

`SUE_CHARACTER_RESULT` (`CWvsContext::OnSueCharacterResult`) is **not** in this family's
work-list and is **not** touched.

---

## 6. Cluster Breakdown & Sequencing

All 75 ops, grouped for cluster-gated commits (D1). Each cluster is independently committable
(codec/relocation + wiring + tests + evidence + regenerated STATUS.md), keeping reviews bounded
even though all 75 land in one PR. Sequence proves the recipe on the simplest, highest-value ops
first, then the self-contained minigames.

0. **Triage** (`structures/triage.md`) — the A/B/C table for all 75 ops + C-row resolutions
   (§3). First commit; no code.
1. **Chat relocation** (§5) — move the 3 CField-owned chat codecs to `field/`, repoint importers,
   add the 3 new chat ops (`SPOUSE_CHAT`, `ADMIN_CHAT`/slash family, `SUE_CHARACTER`). Lands the
   relocation churn in isolation, on already-tested code, before new derivation work.
2. **Core CField — transfer/blocked/obstacle/quest/clock (≈20 ops)** —
   `BLOCKED_MAP`/`BLOCKED_SERVER` (transfer-req-ignored), `FIELD_OBSTACLE_*`,
   `SET_QUEST_CLEAR`/`SET_QUEST_TIME`, `STOP_CLOCK`, `SET_OBJECT_STATE`, `FORCED_MAP_EQUIP`,
   `SUMMON_ITEM_INAVAILABLE`, `FOOTHOLD_INFO`, `GMEVENT_INSTRUCTIONS`/`OX_QUIZ`/`PLAY_JUKEBOX`.
   Mostly fixed-width scalars; verifies many existing `field/` codecs (A-rows).
3. **Core CField — boss timers / events / admin / MTS (≈12 ops)** — `ZAKUM_SHRINE`,
   `HORNTAIL_CAVE`, `WITCH_TOWER_SCORE_UPDATE`, `ARIANT_RESULT` (`OnWarnMessage`),
   `ADMIN_RESULT`, `VICIOUS_HAMMER` (`OnItemUpgrade`), `MTS_OPERATION`/`MTS_OPERATION2`,
   `MATCH_TABLE`/`SLIDE_REQUEST`/`ADMIN_COMMAND`/`ADMIN_LOG` (slash variants),
   `USE_DOOR`, `GUILD_OPERATION`. Includes the MTS/door/guild C-row resolutions.
4. **Foothold/stalk C-cluster** — the `IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1` + `IDA_0X169`
   rows, after their §3 resolution (collapse-into-named-op / new-codec / ⬜). Most ⬜
   VERSION-ABSENT justifications live here.
5. **Minigames (the `CField_*` subclasses, ≈30 ops)** — landed as one block, sub-grouped per
   subclass: SnowBall (6), Tournament (5), Wedding (4), Coconut (3), GuildBoss (3), ContiMove (2),
   AriantArena (2), Battlefield/sheep-ranch (2), Massacre/MassacreResult (2), Witchtower (1).
   Low-traffic, self-contained; expect several ⬜ VERSION-ABSENT cells (GMS-event-only).

> Cluster op-counts are indicative; the authoritative per-op list and per-version applicability
> come from `structures/triage.md` (Phase 0), not from this section.

---

## 7. Version Handling & VERSION-ABSENT Policy

- Per-version opcodes come from `docs/packets/registry/<version>.yaml`, read per file. The
  COutPacket/recv-dispatch opcode is ground truth; registry csv values can be off-by-one.
- A version where an op is genuinely absent from the dispatcher/registry → ⬜ **n/a** with an
  IDB-evidenced justification recorded in the evidence/triage record. **Never** reclassify a
  live cell to ⬜ to "close" it (PRD non-goal). The minigames are the main source of legitimate
  ⬜ cells (several are GMS-event-only; jms may lack them).
- `MajorAtLeast(87)` gates only (never `>83`), so v84 takes the v83 path where structures match.

---

## 8. Operational Rollout

Seed templates apply only at tenant **creation**; existing tenants don't auto-pick-up new
opcodes (`bug_new_opcodes_not_in_live_tenant_config`). This task lands **seed templates only**;
the live PATCH/rollout is **documented, not applied** (PRD non-goal). Produce
`deploy-notes.md` with, per version: the new `socket.handlers` (with validator) + `socket.writers`
entries in live-tenant PATCH shape (per-version opcode tables, as in task-092), plus the checklist:

1. PATCH each live v83/v84/v87/v95/jms tenant config with the new entries.
2. **Restart `atlas-channel`** — the handler/writer map is built once at startup; the config
   projection does not hot-reload handlers/writers.
3. Post-deploy checks: `grep "Unable to locate validator"` == 0; no new error/fatal; serverbound
   ops no longer emit "unhandled message op 0xXX".

---

## 9. Verification Gates (CLAUDE.md)

- `go test -race ./...` clean in every changed module (`libs/atlas-packet`, `atlas-channel`,
  `atlas-configurations`, plus `tools/packet-audit` if touched).
- `go vet ./...` clean; `GOWORK=off tools/redis-key-guard.sh` clean from repo root.
- `go build ./...` clean for `atlas-channel` and `atlas-configurations`.
- **No `go.mod` touched** (`libs/atlas-packet` is already a workspace member; no new lib) → no
  `Dockerfile` COPY / `go.work` edit, and `docker buildx bake` only if a `go.mod` changes (not
  expected). Confirm with `git diff --name-only -- '**/go.mod'` before claiming done.
- `go run ./tools/packet-audit matrix --check` exits 0; every targeted CField cell is ✅ (or ⬜
  with IDB evidence); zero `conflict` cells attributable to this task; no orphan/dangling/stale/
  drift line mentioning a CField packet.
- Seed-template JSON valid; `handlers`/`writers` arrays ascending by opCode.
- Code review: `plan-adherence-reviewer` + `backend-guidelines-reviewer`, green before PR.

---

## 10. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Duplicate codec for an already-implemented op | Phase-0 triage (§3) is a committed gate; every op has an A/B/C row; A-rows forbid new codecs. |
| Chat relocation regresses already-✅ cells | §5 move-not-rewrite, `git mv` codec+test together, NAME constants unchanged; isolated commit; `matrix` re-run must keep cells ✅; shared files link-in-place, not moved. |
| `IDA_0x09x` foothold/stalk rows carried into code as placeholders | §3 C-row resolution derives real op-names from `OnStalkResult`/`OnFootHoldInfo` before any codec; no `IDA_0x…` reaches code. |
| MTS_OPERATION/2 modeled wrong (two modes vs two packets) | Decompile `OnCharacterSale` in triage; record the verdict before coding. |
| Minigames padded with false ⬜ to "close" cells | ⬜ only with IDB dispatcher-absence evidence; never to close a live cell (§7). |
| v84 deltas invented where v84≡v83 | `MajorAtLeast(87)` gates only; derive strictly from the v84 IDB (port 13341). |
| 75 ops × ≤5 versions is large for one PR | Cluster-gated commits (§6); each cluster independently reviewable; `matrix --check` green per cluster. |
| Validator-less handler entry silently dropped | Every `socket.handlers` entry carries a validator (`LoggedInValidator` default); checked in review + `Unable to locate validator`==0 post-deploy. |
| Export non-idempotency corrupts a committed export | Surgical splice only (`VERIFYING_A_PACKET.md` §10); strip the `COutPacket`-ctor `Delegate` artifact; never overwrite a committed export. |

---

## 11. Out of Scope

- All gameplay behavior: acting on decoded serverbound packets; orchestrating minigames,
  weddings, tournaments, MTS sales; door/transfer side effects. Deferred to behavior tasks.
- Producer triggers / serverbound action stubs in any service (D2 — none land).
- Live-tenant config PATCH/rollout (documented in `deploy-notes.md`, not applied).
- Non-`CField` packet families (CWvsContext, CUserLocal, CCashShop, etc. — separate batches);
  `SUE_CHARACTER_RESULT` and the `CUser*`/`CWvsContext` chat codecs that stay in `chat/`.
- Re-authoring `IMPLEMENTING_A_PACKET.md` (task-092 already produced it; task-096 follows it).
- Reclassifying any live cell to ⬜ to "close" it (only genuine IDB-evidenced absence).

---

## 12. Open-Question Resolution Summary (PRD §9)

| PRD Open Q | Resolution |
|---|---|
| Unresolved foothold/stalk rows (`IDA_0X098/09C/09D/0A4/0AA/0AC/0B0/0B1`) | §3 C-row: decompile `OnStalkResult`/`OnFootHoldInfo`/`OnRequestFootHoldInfo`, resolve to collapse / new-codec / ⬜ before coding; no `IDA_0x…` in code. |
| Chat-ops package placement | **D3 — relocate the CField-owned chat codecs into `field/`** (gains tier-1 prefix), §5; non-CField chat codecs stay in `chat/`. |
| Minigame value/coverage | §6 cluster 5: verify wire shape; expect IDB-evidenced ⬜ VERSION-ABSENT cells (GMS-event-only). |
| `MTS_OPERATION`/`MTS_OPERATION2` (one fname, two ops) | §3 C-row: decompile `OnCharacterSale` to decide two-modes-of-one vs two-distinct; record verdict before coding. |
