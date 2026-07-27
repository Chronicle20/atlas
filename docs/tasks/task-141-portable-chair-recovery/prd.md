# Portable Chair Recovery Tick — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

Chairs are functionally implemented in Atlas: sitting on fixed map seats and portable chairs works end-to-end (`USE_CHAIR` STATUS 529, `SHOW_CHAIR` 262, `CANCEL_CHAIR` 272, backed by `atlas-chairs` sit-state tracking and `atlas-channel` socket handlers). However, the serverbound packet `STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST` (`CWvsContext::TryRecovery`) is unimplemented in every version column (STATUS row 562: gms_v83 `0x04A` ❌, gms_v84 `0x04A` ❌, gms_v87 `0x04D` ❌, gms_v95 `0x050` ❌, jms_v185 `0x042` ❌).

The client sends this packet on its own timer while the character sits on a portable chair that carries recovery stats. Because no handler exists, recovery-stat chairs (e.g. The Relaxer, 03010000, `recoveryHP=50`) never heal — the flagship purpose of most cash-shop and reward chairs is silently broken. WZ data confirms the scale: in the local v83-era dump (`Item.wz/Install/0301.img.xml`), **238 chairs carry `info/recoveryHP` and 178 also carry `info/recoveryMP`** (e.g. 03010136 has both at 60).

This task implements the packet decode, socket handler, validation, and stat application so recovery chairs heal HP and MP on all five supported versions, with the server — not the client — as the authority on whether a tick is legitimate.

## 2. Goals

Primary goals:
- Implement `STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST` decode + handler for all five versions: gms_v83, gms_v84, gms_v87, gms_v95, jms_v185.
- Recovery-stat portable chairs restore HP and MP while the character is seated.
- Server-side validation: a tick is honored only when the character is actually seated on a portable chair whose item data carries recovery stats, and only at the expected client cadence (rate-limited).
- Promote STATUS row 562 from ❌ to ✅ for all five versions via the standard packet-verifier flow (byte fixtures + evidence records).

Non-goals:
- Fixed map seats (`chairType=fixed`) — map seats have no recovery stats; only portable (301xxxx setup items) are in scope.
- Any clientbound writer work beyond what the existing stat-change path already emits (`CHANGE_HP`/`CHANGE_MP` through atlas-character drives the existing stat-update packets). If IDA review reveals the client requires a dedicated response packet, that becomes a design-phase finding, not new PRD scope.
- Chair cosmetics, chair inventory, or towel/chair gachapon behavior.
- Big-Bang+ recovery mechanics (jms_v185 uses the same TryRecovery request; anything structurally different is handled per-version during verification).

## 3. User Stories

- As a player sitting on a recovery chair, I want my HP/MP to tick up at the normal client cadence so that the chair does what its tooltip says.
- As a player on any supported tenant version (v83/v84/v87/v95/jms185), I want chair recovery to behave identically so version choice doesn't change gameplay.
- As an operator, I want the server to reject recovery requests from characters who are not seated (or who spam the packet) so hacked clients cannot use it as a free heal.

## 4. Functional Requirements

### 4.1 Packet decode (libs/atlas-packet)

- FR-1.1: Add a `TryRecovery` (working name; final name matches audit conventions) serverbound codec in `libs/atlas-packet/character/serverbound/`, following the existing `chair_portable.go` pattern (`Operation()`, `Decode`, `Encode`, `String`).
- FR-1.2: The packet body MUST be derived from IDA decompilation of `CWvsContext::TryRecovery` per version — never from other server implementations or memory. Structural differences between versions, if any, are handled via reader options as done elsewhere in the lib.
- FR-1.3: Byte-fixture tests with `packet-audit:verify` markers for every version in scope, per `docs/packets/audits/VERIFYING_A_PACKET.md` (evidence record pinned, matrix regenerated).

### 4.2 Socket handler (atlas-channel)

- FR-2.1: New handler (pattern: `character_chair_portable.go`) that decodes the packet and forwards a recovery request for the session's character.
- FR-2.2: Handler registered in the tenant socket-handler seed templates for **all five versions** at the correct per-version opcode (0x04A/0x04A/0x04D/0x050/0x042), each entry with an explicit validator (`LoggedInValidator`) — a missing validator silently drops the handler.
- FR-2.3: Live tenant configurations for existing tenants MUST be patched (seed templates apply only at tenant creation) and atlas-channel restarted, per the established rollout gotcha.

### 4.3 Chair recovery data (atlas-data)

- FR-3.1: `setup/reader.go` additionally parses `info/recoveryMP` (default 0), alongside the existing `recoveryHP`.
- FR-3.2: `recoveryMP` is exposed on the setups REST resource (`/data/setups`, `rest.go`) like `recoveryHP`.
- FR-3.3: Rollout note: the new field only exists for data ingested after the change. The plan must state how existing tenants get the field (re-ingest or canonical baseline re-publish) — silently serving 0 for all chairs on live tenants is not acceptable as the end state.

### 4.4 Recovery validation and application

- FR-4.1: A recovery tick is honored only if ALL of the following hold (server-authoritative):
  1. The character has an active sit registration in `atlas-chairs` with `chairType=portable`.
  2. The seated chair's setup item data has `recoveryHP > 0` or `recoveryMP > 0`.
  3. The tick respects the rate limit (FR-4.3).
- FR-4.2: A valid tick applies `recoveryHP` via the existing `CHANGE_HP` command and `recoveryMP` via `CHANGE_MP` to atlas-character (amounts straight from item data; clamping to max HP/MP is atlas-character's existing responsibility).
- FR-4.3: Rate limiting: the server tracks the last honored tick per character and ignores requests arriving faster than the client's real send interval. The exact interval MUST be IDA-verified from the client's TryRecovery timer during design (do not assume a folklore value); the server allows a small tolerance below it for clock skew.
- FR-4.4: Invalid ticks (not seated, no recovery stats, rate exceeded) are dropped with a log line at debug/warn; they MUST NOT disconnect the player and MUST NOT apply any stat change.
- FR-4.5: Standing up (`CANCEL_CHAIR` / sit-state cleared) ends recovery: subsequent ticks fail FR-4.1(1). No server-side ticker exists — recovery is purely client-request-driven.

### 4.5 Ownership of the validation logic

- FR-5.1: Sit-state truth lives in `atlas-chairs`; the design phase decides whether the handler in atlas-channel queries atlas-chairs (REST) and applies stats itself, or emits a command consumed by atlas-chairs which orchestrates validation + `CHANGE_HP`/`CHANGE_MP` emission. Requirement: exactly one service owns the validation decision, and the sit-state check must not be duplicated in atlas-channel.

## 5. API Surface

- `GET /data/setups` and `GET /data/setups/{id}` (atlas-data, JSON:API `setups` resource): response gains `recoveryMP` attribute (uint32, default 0). Additive, non-breaking.
- New Kafka command/topic only if the design places validation in atlas-chairs (e.g. a `RECOVERY` command on the existing chair command topic). No new public REST endpoints.
- No new clientbound packets expected (stat updates flow through the existing atlas-character stat-change events); confirmed or corrected during IDA design review.

## 6. Data Model

- No new persistent entities. Sit state already lives in the atlas-chairs registry.
- New transient state: last-honored-recovery-tick timestamp per (tenant, characterId) for rate limiting — registry/in-memory alongside the existing chair registry; cleared when the sit registration is cleared. Multi-tenant keyed like the existing registry.
- atlas-data setup model gains `RecoveryMP uint32` (reader, registry, REST model). Requires re-ingest/baseline re-publish for existing tenants (FR-3.3).

## 7. Service Impact

| Component | Change |
|---|---|
| `libs/atlas-packet` | New serverbound codec + per-version byte fixtures (FR-1.*) |
| `services/atlas-channel` | New socket handler + registration wiring (FR-2.1) |
| `services/atlas-data` | `recoveryMP` parse + REST exposure (FR-3.*) |
| `services/atlas-chairs` | Recovery validation/orchestration + tick rate-limit state (FR-4.*, placement per FR-5.1) |
| Tenant seed templates | Handler entry with validator at per-version opcode, all 5 versions (FR-2.2) |
| Live tenant configs | Post-merge patch + channel restart (FR-2.3) |
| `docs/packets/audits/` | Evidence records, fixtures, STATUS row 562 → ✅ ×5 (FR-1.3) |

## 8. Non-Functional Requirements

- Multi-tenancy: all lookups, registries, and Kafka messages tenant-scoped via `tenant.MustFromContext`; rate-limit state keyed per tenant.
- Anti-cheat: server-authoritative validation per FR-4.1/4.3; no trust in client-supplied recovery amounts (amounts come from item data only — if the packet body carries client-side values, they are ignored for stat application).
- Performance: one registry lookup + one item-data lookup per tick per seated character; item recovery stats should be resolved via the existing cached data-service access patterns, not a fresh REST call per tick if a cache exists.
- Observability: debug log per honored tick; warn (rate-limited) log for rejected ticks with reason.
- Verification gates: `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake` for every touched service; `tools/redis-key-guard.sh`; `packet-audit` matrix checks clean.

## 9. Open Questions

1. Exact packet body of `CWvsContext::TryRecovery` per version (field order/types) — resolved by IDA during design/verification; no assumption made here.
2. Exact client send interval for the recovery timer (needed for FR-4.3's rate limit) — IDA-verify; if it differs per version, the limit is per-version config-resolved rather than hard-coded.
3. Whether the client expects any acknowledgment beyond the normal stat-change packet (design-phase IDA finding; current assumption: no).
4. FR-5.1 placement: atlas-channel-applies vs atlas-chairs-orchestrates — design-phase decision.
5. Mechanism for backfilling `recoveryMP` on existing tenants (re-ingest vs canonical baseline re-publish) — plan-phase decision per FR-3.3.

## 10. Acceptance Criteria

- [ ] Serverbound codec exists in `libs/atlas-packet` with IDA-derived decode and byte-fixture tests for gms_v83, gms_v84, gms_v87, gms_v95, jms_v185.
- [ ] STATUS row `STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST` shows ✅ for all five versions after matrix regeneration; evidence records committed.
- [ ] Handler registered (with validator) in all five seed templates at the per-version opcode; live tenant patch steps documented in the plan.
- [ ] `GET /data/setups` returns `recoveryMP`; reader test covers a chair with both stats (03010136-style) and an HP-only chair.
- [ ] Sitting on an HP-recovery chair on the v83 test tenant visibly restores HP at the client cadence; an HP+MP chair restores both.
- [ ] Recovery requests while not seated, or above the rate limit, are dropped without stat change or disconnect (covered by unit tests on the validation path).
- [ ] Standing up stops recovery (subsequent tick rejected).
- [ ] All verification gates in §8 pass; no `// TODO` stubs.
