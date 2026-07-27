# Mob Spawn Move-Action (Stance) Byte — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-18
---

## 1. Overview

The clientbound spawn-monster and control-monster packets each carry a one-byte
"move action" (a.k.a. stance) per mob. In the v83 client this byte is consumed
during `CMob::Init`. A value of `0` or `1` is a **sentinel** meaning "unresolved —
client, please compute the idle action," which routes the client through
`CMob::OnResolveMoveAction`. On a bulk re-spawn that path dereferences a vector
controller (`m_pvc`) that is not yet wired up → null-deref → client access
violation (crash). Sending an already-resolved byte (`>= 2`) skips that path
entirely and is safe.

The move-action byte is not a magic constant. The client encodes it as
`(actionIndex << 1) | facingBit`, where `actionIndex` is the mob's idle animation
index selected purely from the mob's move-ability class: ground mobs idle at
`actionIndex 2`, fly/swim mobs idle at `actionIndex 6`. The correct emitted byte
is therefore `4/5` for ground mobs and `12/13` for fly/swim mobs (right/left
facing).

Two defects exist in the current Atlas implementation:

1. **Crash (the real 0-path).** `atlas-monsters` seeds a fresh spawn with a flat
   `5` (`monster/processor.go:198`), which is safe on the *initial* spawn. But
   after the mob moves, the stored stance is overwritten by the raw client-supplied
   move-action byte with no floor
   (`atlas-channel .../movement/processor.go:287,293,296,302` → Kafka `stance` →
   `atlas-monsters` `Move` → registry `stance`). The v83 client legitimately sends
   the `0`/`1` idle sentinel in its move fragments; that sentinel is persisted and
   later **re-emitted verbatim** on the next spawn / control-assignment
   (`monster_spawn.go:51`, `monster_control.go:51`), reproducing the
   `CMob::Init → OnResolveMoveAction` crash. The flat `5` never protected the
   post-movement case.
2. **Wrong flyer animation.** The flat `5` is ground-idle. Fly/swim mobs spawned
   fresh receive `5` instead of `12/13`; the client's `GetFineAction` silently
   downgrades the missing action, so flyers animate incorrectly.

This feature emits a correct, pre-resolved, fly-aware move-action byte so the
client never resolves the stance during `CMob::Init`, and guarantees the emitted
byte is never the `0`/`1` sentinel on any spawn/control packet.

## 2. Goals

Primary goals:

- Eliminate the client AV crash: **no spawn or control packet ever carries a
  move-action byte of `0` or `1`** for a spawned mob (`(byte & ~1) != 0`).
- Fresh mob spawns carry a fly-aware idle stance: ground mobs `4/5`, fly/swim
  mobs `12/13`, computed from the mob's template — matching what the client's
  own resolver would produce.
- The `0`/`1` sentinel that a client move-fragment can introduce is resolved to
  the mob's correct fly-aware idle stance before it is emitted, at the final gate
  before the wire.

Non-goals:

- Rewriting mid-action / pre-resolved stances the server already sends correctly.
  Only the `0`/`1` idle sentinel and the flat-`5`-for-flyers cases are in scope.
- Changing the live movement-broadcast (relay-to-other-clients) path. The crash
  is specific to the spawn/control `CMob::Init` path; live move packets are
  handled in the client's move context and are out of scope.
- Any change to the spawn/control **packet layout** — same fields, same order,
  only the value of the existing `moveAction` byte changes.
- Modelling a full `moveAbility` enum. Only the FLY-vs-not distinction that the
  client's idle resolver uses is required.

## 3. User Stories

- As a **player**, I want mobs to spawn without crashing my client when I enter or
  re-enter a map, so that bulk re-spawns are safe.
- As a **player**, I want flying and swimming mobs to animate in their correct
  idle pose on spawn, so that the world looks right.
- As a **server operator**, I want the invariant "spawn/control stance is never a
  sentinel" enforced at the emit boundary, so that no future upstream regression
  can reach the client with a crashing byte.

## 4. Functional Requirements

### FR-1 — Fly class derivation (data)

- FR-1.1: A mob is **fly-class** iff it has a fly-family animation. `atlas-data`
  already derives this at `monster/reader.go:92-96`:
  `Flying = hasAnimation("fly")`, `Swimming = hasAnimation("hover") || hasAnimation("swim")`.
- FR-1.2: For stance purposes, **`isFly = Flying || Swimming`** (swim mobs are
  fly-in-water and use the same client branch → `actionIndex 6`).
- FR-1.3: `atlas-data` already exposes `flying`, `swimming`, `fixed_stance`, and
  `animation_times` on its monster REST model (`monster/rest.go:28-38`). No new
  `atlas-data` field is required.

### FR-2 — Idle stance computation

- FR-2.1: Define the encoding and constants (client-verified):
  - `actionIndex = isFly ? 6 : 2`
  - `moveActionByte = (actionIndex << 1) | facingBit`
  - `facingBit`: `0 = right`, `1 = left`.
  - Precomputed: ground `4` (right) / `5` (left); fly `12` (right) / `13` (left).
- FR-2.2: **Facing policy (decided):** honor `FixedStance` when the mob is a
  `noFlip` mob, otherwise default to facing **right**.
  - `atlas-data`'s `getFixedStance` returns `4` or `5` for `noFlip` mobs (fixed
    facing derived from `stand/0/origin`), else `0`.
  - When `FixedStance != 0`, take the facing bit from it: `facingBit = FixedStance & 1`.
  - When `FixedStance == 0`, `facingBit = 0` (right).
  - `actionIndex` is always driven by `isFly` (FR-1.2). A `noFlip` **fly** mob
    therefore emits `12/13` with the fixed facing, not the ground `4/5` that
    `FixedStance` alone would imply — `FixedStance` contributes only the facing
    bit, never the action index.
- FR-2.3: Provide a single pure helper `idleMoveAction(isFly bool, fixedStance uint32) byte`
  (exact location per the design phase) that implements FR-2.1/FR-2.2 and is unit
  tested against the acceptance vectors in §10.

### FR-3 — Fresh-spawn origin (atlas-monsters)

- FR-3.1: Replace the hardcoded `5` at `atlas-monsters .../monster/processor.go:198`
  with `idleMoveAction(isFly, fixedStance)` computed from the mob's template.
- FR-3.2: `atlas-monsters` fetches template data via its
  `monster/information` client (root `DATA`, `data/monsters/{id}`). Extend that
  client's REST/domain model to carry `flying`, `swimming`, and `fixed_stance`
  (it already fetches `animation_times` and `fixed_stance` in the DTO but drops
  them in `Extract`; `flying`/`swimming` are not yet on the DTO). Wire them through
  so `Create` has `isFly` and `fixedStance` available.
- FR-3.3: The computed stance is stored in the registry `stance` field exactly as
  today (no schema change) and propagates unchanged to `atlas-channel`.

### FR-4 — Sentinel guard at the emit boundary (atlas-channel)

- FR-4.1: At the final gate before the wire — the two `NewMonster(...)` sites that
  feed `CMob::Init`: `socket/writer/monster_spawn.go:51` and
  `socket/writer/monster_control.go:51` — if the stance to be emitted is `0` or
  `1`, resolve it to the mob's fly-aware idle stance instead of emitting the
  sentinel.
- FR-4.2: The resolution MUST be fly-aware: the guard needs the mob's `isFly` and
  `fixedStance`. `atlas-channel` already has a `monster/information` client hitting
  `DATA` (`.../monster/information/`, currently extracting only `Attacks`); extend
  it to fetch `flying`/`swimming`/`fixed_stance` so the guard produces `12/13` for
  flyers rather than blindly clamping to a ground value.
- FR-4.3: The guard MUST be **narrow**: it only rewrites `0`/`1`. Any stance
  `>= 2` (including legitimate mid-action stances and the fresh `4/5`/`12/13`
  from FR-3) is emitted verbatim.
- FR-4.4: Template fly-class lookups on the spawn/control hot path MUST be cached
  (see NFR-2) so the guard does not add a synchronous `atlas-data` round-trip per
  mob per spawn.

### FR-5 — Scope fences

- FR-5.1: The live movement-broadcast path (relaying a controller's move to other
  clients) is unchanged.
- FR-5.2: The zero-value `NewMonster(f, uniqueId, 0, 0, ...)` at
  `atlas-monsters .../processor.go:1511` is a diff/placeholder construction, not a
  spawn emit; it is out of scope.

## 5. API Surface

No new endpoints. Modified REST **read models only** (additive, backward
compatible):

- `atlas-monsters` `monster/information` client DTO (`information/rest.go`): add
  `Flying bool json:"flying"`, `Swimming bool json:"swimming"`; map `flying`,
  `swimming`, `fixed_stance` through `Extract` into the domain
  `information.Model` (which today drops `fixed_stance` and lacks fly flags).
- `atlas-channel` `monster/information` client model
  (`monster/information/rest.go` + `model.go`): add `flying`, `swimming`,
  `fixed_stance` extraction (today only `Attacks`).

Producer of these fields (`atlas-data`) is already complete — no change. All
additions are additive JSON fields; older/newer peers that ignore them are
unaffected.

## 6. Data Model

- No database schema change. No migration.
- No Kafka message schema change (`stance`/`ni` byte already exists end-to-end).
- The monster registry `stance` field (`atlas-monsters .../registry.go:46`) is
  unchanged; only the value written into it at `Create` time changes.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-data` | **None.** Already derives and exposes `flying`, `swimming`, `fixed_stance`, `animation_times`. |
| `atlas-monsters` | Extend `monster/information` DTO + `Extract` to carry `flying`/`swimming`/`fixed_stance`; replace hardcoded `5` at `Create` (`processor.go:198`) with `idleMoveAction(...)`. Add unit tests for the helper. |
| `atlas-channel` | Extend `monster/information` client to fetch fly fields (with caching); add the narrow `0/1 → fly-aware idle` guard at `monster_spawn.go` and `monster_control.go` emit sites. |
| `libs/atlas-packet` | **None** — layout unchanged; `moveAction` still `WriteByte` at `model/monster.go:521`. Optional home for the shared `idleMoveAction` helper/constants if both services should share it (design decision). |

## 8. Non-Functional Requirements

- **NFR-1 (Correctness / grounding):** Encoding `(actionIndex << 1) | facing` and
  the two-way idle branch `v8 = (moveAbility != 3) ? 2 : 6; return (a2 & 1) | (2*v8)`
  are from client `MapleStory_dump.exe` (v83): `CMob::GetFineAction @0x671999` via
  `sub_671AFF` (`GetFineMoveDirAction`) → `sub_664D42`. The crash is the
  `(byte & ~1) == 0` sentinel path into `CMob::OnResolveMoveAction`. These MUST be
  re-verified against the client during design/implementation, not taken on faith.
  Fly/ground is distinguished by fly-animation presence (no `moveAbility`/`flySpeed`
  scalar exists in WZ).
- **NFR-2 (Performance):** Spawn and control are hot paths (bulk map-enter emits
  one spawn per mob). Fly-class template lookups in `atlas-channel` MUST be cached
  per monster template id (the guard runs per emit); no per-mob synchronous
  `atlas-data` fetch on the spawn path.
- **NFR-3 (Multi-tenancy):** All template fetches remain tenant-scoped via the
  existing context/tenant plumbing; no cross-tenant cache bleed for fly-class
  lookups.
- **NFR-4 (Observability):** The existing spawn-wire debug log
  (`monster_spawn.go:48`) already prints `stance`. Ensure the guard's resolution
  is observable (e.g. log when a `0`/`1` sentinel is rewritten, at debug level) so
  the fix can be confirmed against live traffic.
- **NFR-5 (Version scope):** The sentinel crash is latent in v79/v83/v95 alike;
  the fix (never emit `0`/`1`, emit fly-aware idle) is version-agnostic and must
  not regress any version's spawn/control encoding.

## 9. Open Questions

- **OQ-1:** Where should the shared `idleMoveAction` helper + stance constants
  live so both `atlas-monsters` and `atlas-channel` use one definition without
  breaking service boundaries? Candidates: `libs/atlas-packet/model` (co-located
  with the byte it feeds) or `libs/atlas-constants`. Resolve in design; prefer a
  straightforward shared home over duplicating the formula in two services.
- **OQ-2:** Caching strategy/TTL for the `atlas-channel` fly-class template lookup
  (NFR-2) — reuse an existing template cache pattern in the channel
  `monster/information` processor, or add one. Resolve in design.
- **OQ-3:** Should `atlas-monsters` `Create` and the `atlas-channel` guard both
  compute independently, or should `atlas-monsters` always emit a correct
  non-sentinel value and the channel guard remain a pure backstop? (Both are
  needed — Create fixes fresh flyers, the guard fixes moved-then-idle sentinels —
  but confirm no double-resolution disagreement, e.g. facing.)
- **OQ-4:** `noFlip` fly mobs: confirm that combining `FixedStance` facing with the
  fly `actionIndex 6` (→ `12/13`) is correct against a concrete `noFlip` flyer, or
  whether any such mob exists in the target catalogs (may be vacuously moot).

## 10. Acceptance Criteria

- [ ] A pure helper computes the idle move-action byte and passes these vectors:

  | Mob | Class | hasFlyAnimation | FixedStance | Expected (right / left) |
  |---|---|---|---|---|
  | 0100100 (snail) | ground | no | 0 | 4 / — (right default) |
  | 2300100 | fly | yes | 0 | 12 / — (right default) |
  | 7130020 | swim | yes | 0 | 12 / — (right default) |
  | ground `noFlip` (FixedStance=5) | ground | no | 5 | 5 (facing from FixedStance) |
  | ground `noFlip` (FixedStance=4) | ground | no | 4 | 4 (facing from FixedStance) |

- [ ] Invariant test: for every code path that produces a spawn/control stance,
  the emitted byte satisfies `(byte & ~1) != 0` (never `0` or `1`).
- [ ] `atlas-monsters` `Create` no longer hardcodes `5`; a fresh fly/swim mob's
  registry `stance` is `12` (or `13` per facing), a fresh ground mob's is `4`/`5`.
- [ ] `atlas-channel` spawn (`monster_spawn.go`) and control
  (`monster_control.go`) emit sites rewrite a `0`/`1` input stance to the mob's
  fly-aware idle stance, and pass `>= 2` stances through unchanged (unit tested
  with a fly mob and a ground mob).
- [ ] The fly-class template lookup on the channel spawn/control path is cached
  (no per-mob synchronous `atlas-data` fetch); verified by test or by inspection.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every
  changed module (`atlas-monsters`, `atlas-channel`, and `libs/atlas-packet` if
  the helper lands there).
- [ ] `docker buildx bake atlas-monsters` and `docker buildx bake atlas-channel`
  succeed from the worktree root (mandatory per CLAUDE.md build rules).
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, and
  `tools/goroutine-guard.sh` clean from the repo root.
- [ ] No packet layout change: a byte-level diff of the spawn/control encoding
  shows only the `moveAction` value differs, never field count/order.
