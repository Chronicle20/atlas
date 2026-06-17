# Backend Audit — task-096 CField packet family

- **Service Path:** `services/atlas-channel/atlas.com/channel`, `libs/atlas-packet/field`, `tools/packet-audit`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-15
- **Build:** PASS (atlas-packet, atlas-channel, packet-audit)
- **Tests:** PASS (`field/...`, channel `socket/...`, all `tools/packet-audit/...`); 0 failed
- **Vet:** PASS (all three modules)
- **Overall:** NEEDS-WORK (build/test/vet all green; one mislabeled wire-format change + one uncovered grader branch — both non-blocking)

## Scope & Classification

This change is a **packet-codec + socket-wiring** change, not a DDD domain service. There are no
domain packages (`model.go`/`builder.go`/`entity.go`/`processor.go`/`administrator.go`/`rest.go`/
`resource.go`) introduced. Most of the DOM-* checklist (builder/entity/Make/Transform/JSON:API/
provider/multitenancy/HTTP-status/administrator) is **N/A by construction** — the checklist targets
REST domain services, and this is `atlas-packet` wire codecs + channel socket handlers/writers.

Changed surface:
- 50 new clientbound codecs + 15 new serverbound codecs in `libs/atlas-packet/field/` (each with a sibling `_test.go`).
- 3 relocated codecs `chat → field` (`multi`, `whisper` clientbound; `general` serverbound).
- 14 new channel socket handlers (`socket/handler/*.go`), decode-and-log only.
- 48 new channel socket writers (`socket/writer/*.go`), `Body()` seam helpers.
- 2 modified relocated chat handlers (`character_chat_general.go`, `character_chat_whisper.go`).
- `main.go` writer/handler registration; consumer wiring.
- 5 seed templates (`atlas-configurations/seed-data/templates/template_*.json`).
- `tools/packet-audit/{cmd,internal}` matrix-grader op-identity fix (commit `baa937176`).

## Build & Test Results

Run against the worktree (`.worktrees/task-096-cfield-packet-family`):

| Module | build | vet | test |
|--------|-------|-----|------|
| `libs/atlas-packet` (`field/...`) | PASS | PASS | PASS (`clientbound`, `serverbound` ok) |
| `services/atlas-channel/.../channel` (`socket/...`) | PASS | PASS | PASS (`handler`, `model`, `writer` ok) |
| `tools/packet-audit` | PASS | PASS | PASS (all 13 packages ok) |

## Domain Checklist Results

No domain (`model.go`) or sub-domain (`resource.go` action-event) packages were introduced.
The DOM-* items that have a meaningful interpretation for codec/wiring code are reported below;
the rest are N/A (no DDD domain surface).

### libs/atlas-packet/field (codec package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01..05 | builder/ToEntity/Make/Transform/TransformSlice | N/A | No domain model; codecs use immutable struct + constructor + getters + Encode/Decode (e.g. `field/clientbound/foothold_info.go:36-76`). |
| DOM-06 | Codec accepts `FieldLogger` | PASS | All Encode/Decode signatures take `logrus.FieldLogger`, not `*logrus.Logger` (`foothold_info.go:89,130`; `whisper.go:34,44`). Grep for `*logrus.Logger` in new codecs: zero. |
| DOM-11 | Lazy provider eval | N/A | No DB/provider layer in codecs. |
| DOM-12 | No `os.Getenv()` | PASS | Grep across new codecs + handlers: zero matches. |
| DOM-18/19 | JSON:API / flat request models | N/A | Binary socket codecs, not REST. |
| DOM-20 | Table-driven tests | PASS | Every new codec has a sibling `_test.go` (50/50 clientbound, 15/15 serverbound) using version-table `RoundTrip` + `packet-audit:verify` markers (e.g. `whisper_test.go:9-13,87-98`). |
| DOM-21 | No atlas-constants duplication | PASS | No new shared-type declarations. The two `type` decls under `field/clientbound/` (`clock.go:14 ClockType`, `kite_destroy.go:14 KiteDestroyAnimationType`) are pre-existing, out of this diff. New codecs use raw `uint32`/`byte` at the wire boundary (`whisper.go:56 channelId byte`, `:140 mapId uint32`) — consistent with the package's existing codec convention; no `_map.Id`/`channel.Id` redeclaration. See INFO-1. |
| DOM-22 | Dockerfile lib mentions | N/A | No new `libs/atlas-X` direct require added; `atlas-packet` already wired. |
| DOM-23 | Kafka topic naming | N/A | No new Kafka topics introduced by this change. |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS / N/A | New handler tests are decode-and-log; no emit path. The relocated `character_chat_general.go` calls `message.NewProcessor(...).GeneralChat(...)` but that path is unchanged by this task (import-only relocation) and its test posture is inherited, not introduced here. |

### services/atlas-channel socket/handler (decode-and-log handlers)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Logic not in handler | PASS (by design D2) | Handlers decode then `l.Debugf` only — no business logic to misplace (`request_foothold_info.go:13-19`, `admin_command.go:13-19`). Behavior deliberately deferred. |
| SUB-02 | No `db.Create`/`db.Save` in handler | PASS | Grep across all 14 new handlers: zero matches. |
| SUB-04 | No manual JSON parsing | PASS | Grep for `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` across new handlers: zero. Socket handlers decode via the codec's `Decode(l,ctx)(r,ro)`. |
| — | `d.Logger()` / no StandardLogger | PASS | Handlers receive `l logrus.FieldLogger` from the registration plumbing; grep for `logrus.StandardLogger` in new handlers: zero. |

### services/atlas-channel socket/writer (seam writers)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Writer `Body` helpers | PASS (intentional seam D3) | Uncalled `Body()` helpers are documented seams (`snowball_touch.go:8-15` carries an explicit "documented seam (IMPLEMENTING_A_PACKET D2), not dead code" comment). The 3 wired writers (snowball_touch/stalk_result/admin_result) and all 48 are registered in `main.go` (`main.go:702-715`, handlers `:798-802`). |

### main.go wiring

| Check | Status | Evidence |
|-------|--------|----------|
| New writers registered | PASS | `main.go:702-704,713-715` register `SnowballTouchWriter`, `StalkResultWriter`, `AdminResultWriter`, `MtsOperation2Writer`, `MtsOperationWriter`, `FootholdInfoWriter`. |
| New handlers registered | PASS | `main.go:798-802` register `RequestFootholdInfoHandle`, `WeddingActionHandle`, `AdminCommandHandle`, etc. → codecs are not dead. |

### Relocation hygiene (chat → field)

| Check | Status | Evidence |
|-------|--------|----------|
| Moved files deleted from source | PASS | `git diff --name-status` shows `D libs/atlas-packet/chat/{clientbound/multi,clientbound/whisper,serverbound/general}.go` (+ their tests). |
| No dead code left behind | PASS | Remaining `chat/clientbound/general.go`, `chat/clientbound/world_message.go`, `chat/serverbound/{multi,whisper}.go` are still referenced (e.g. `socket/handler/character_chat_multi.go`, `character_chat_whisper.go`, `socket/writer/world_message.go`). No orphaned symbols. |
| `multi`/`general` are pure moves | PASS | Rename similarity `R100` (byte-identical apart from package/import). |
| `whisper` is a pure move | **FAIL (mislabeled)** | Rename similarity `R097` — NOT a pure move. See FINDING-1. |

## Security Review

Not an auth/token/authorization service. SEC-01..04 N/A. Note: the admin-* serverbound handlers
(`admin_chat`, `admin_command`, `admin_log`) are decode-and-log only and take no privileged action,
so no authorization gap is introduced by this change (behavior deferred per D2). When these are
later given behavior, an authorization check on the GM/admin command path will be required — out of
scope for this audit.

## Summary

- **PASS:** 16 applicable checks (build/vet/test gate, codec FieldLogger, no env, table-driven tests
  with evidence markers, DOM-21 no-duplication, handler hygiene SUB-01/02/04, writer seams, main.go
  wiring, relocation deletes + no dead code, multi/general pure moves).
- **N/A:** the DDD DOM-* items (01–05, 11, 18–19, 22–23) — no domain/REST surface.
- **FAIL/WARN:** 2 findings, both non-blocking.

### Blocking (must fix)
- None. Build, vet, and tests are clean across all three modules; no anti-pattern violations.

### Non-Blocking (should fix)
- **FINDING-1 (whisper relocation is NOT move-not-rewrite):** `libs/atlas-packet/field/clientbound/whisper.go`
  diverges from the chat original (`R097`, not `R100`). The `WhisperFindResultMap` x/y fields were
  widened `int16 → int32` and the wire ops changed `WriteShort/ReadUint16 → WriteInt/ReadUint32`
  (`whisper.go:142-143,150,158-159,172-176,188-190`). The task brief labels all three chat→field
  codecs as "RELOCATED ... (move-not-rewrite)"; for whisper that is inaccurate — this is a wire-format
  change folded into the relocation. It IS covered: a round-trip test asserts x/y survive
  (`whisper_test.go:112-133`) and carries `packet-audit:verify` markers pinned to client addresses
  (`whisper_test.go:87-91`, v83 `0x53228e` … jms `0x56f4df`). Treated non-blocking because the change
  is test-and-evidence-backed, but it should be called out explicitly in the PR description /
  deploy-notes rather than hidden under "move-not-rewrite," since it changes the bytes on the wire for
  the find-result-map whisper reply.
- **FINDING-2 (matrix-grader op-identity branch has no regression test):** `tools/packet-audit/internal/matrix/build.go:80-100`
  adds the `RoutedNames` op-identity guard (rejects raw-opcode coincidences in the cross-version
  `routedElsewhere` signal). No test in this diff sets `Inputs.RoutedNames` — `grep -L RoutedNames`
  over `internal/matrix/grade_test.go` and `cmd/matrix_test.go` confirms neither populates it, and
  `git diff --name-status` shows no `*_test.go` added/changed under `tools/packet-audit`. The grading
  package's existing tests pass, but the specific false-positive-fix path (`build.go:94-99`) is
  uncovered. The brief acknowledges this as a known gap. Add a `grade_test.go` case that populates
  `RoutedNames` with a colliding opcode in one version and asserts the unrelated version is NOT graded
  `Conflict`.

### Informational (no action required)
- **INFO-1 (DOM-21 wire-boundary primitives):** new codecs carry `mapId uint32` / `channelId byte`
  rather than `_map.Id` / `channel.Id` (`whisper.go:56,140`; `foothold_info.go`). This is not a DOM-21
  violation — no shared type is redeclared, and it matches the established convention for low-level
  `atlas-packet/field` codecs (only higher-level codecs like `set_field.go`/`warp_to_map.go` import
  atlas-constants id types). The whisper field types were inherited from the chat original, not newly
  authored. No change required.
</content>
</invoke>
