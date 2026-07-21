# Mob Spawn Move-Action (Stance) Byte — Design

> ## ⚠️ ROOT-CAUSE CORRECTION (supersedes the stance-byte design below)
>
> The stance-byte hypothesis in this document was **wrong**. Live v79 tracing on
> the ephemeral (FRAG/STANCE/SNAP-TRACE) proved the actual root cause:
> **`MonsterMovementHandle` (opcode `0xB4`) had no `types` option in the tenant
> socket config.** Without it, `movementPathAttrFromOptions` returns `"DEFAULT"`,
> no fragment matches NORMAL/TELEPORT, and every monster move fragment decodes as
> the 3-byte generic `Element` — so **`fh` and `stance` are never decoded**. `fh`
> stayed at the fold seed `0` → stored foothold frozen at spawn → mob drifts off it
> → re-enter snap fails → **fall-through**. `stance` stayed seed `0` → v79 ships
> stance 0 → `CMob::Init` sentinel AV → **freeze**. Both symptoms, one config gap.
> This is also the real v79-vs-v83 divergence: the v83 template had monster
> `types`; v79 did not (`monster.types == char.types` in every working template).
>
> **What actually shipped:** (1) add `types` to Monster/Pet/Summon move handlers
> across every template lacking it (copying each template's own
> `CharacterMoveHandle.types`); (2) fix a latent fold bug where Teleport/Jump/
> StartFallDown were matched by value but the decoder makes pointers (dead cases);
> (3) keep `ControlOnEnter` (spawn-then-control ordering). The fly-aware stance
> work, the `floor-to-5` guard, and all diagnostic tracing were removed. See
> `[[bug_mob_control_before_spawn_creates_mob]]` in project memory. The original
> design is retained below as the investigation record.

Task: task-179-mob-spawn-stance-byte
Status: Approved design
Created: 2026-07-18
Consumes: [`prd.md`](prd.md)

---

## 1. Summary

The clientbound spawn/control monster packets carry a one-byte "move action"
(stance). The v83 client treats `0`/`1` as a **sentinel** ("unresolved — please
compute the idle action") and routes through `CMob::OnResolveMoveAction`, which
null-derefs on bulk re-spawn (client access violation). A resolved byte
(`>= 2`) skips that path.

This design implements two fixes plus a shared computation:

1. **Fresh-spawn origin** (`atlas-monsters`): replace the hardcoded `5` at
   `Create` with a fly-aware idle stance derived from the mob template.
2. **Emit-boundary guard** (`atlas-channel`): at the two `NewMonster(...)` sites
   that feed `CMob::Init`, rewrite a `0`/`1` stance to the mob's fly-aware idle
   stance; pass `>= 2` through verbatim.
3. **One shared pure helper** for the encoding, in `libs/atlas-constants/monster`,
   used by both services so the formula has a single source of truth.

No packet layout change, no DB/Kafka schema change, no `atlas-data` change.

---

## 2. Client grounding (NFR-1)

The encoding and the sentinel path are taken from client `MapleStory_dump.exe`
(v83) and cited in the PRD (§8 NFR-1):

- `CMob::GetFineAction @0x671999` → `sub_671AFF` (`GetFineMoveDirAction`) →
  `sub_664D42`: `v8 = (moveAbility != 3) ? 2 : 6; return (a2 & 1) | (2*v8)`.
- Idle `actionIndex` is `2` for ground move-ability, `6` for fly move-ability.
- The emitted byte is `(actionIndex << 1) | facingBit` → ground `4/5`, fly `12/13`.
- Crash: `(byte & ~1) == 0` (i.e. `0` or `1`) routes `CMob::Init` into
  `CMob::OnResolveMoveAction` → `m_pvc` null-deref.

Fly-vs-ground is distinguished by **fly-animation presence** (no `moveAbility`
scalar exists in WZ). `atlas-data` already derives this
(`monster/reader.go:92-96`): `Flying = has "fly"`,
`Swimming = has "hover" || has "swim"`. For stance purposes
**`isFly = Flying || Swimming`** (swim = fly-in-water, same client branch).

> Grounding note for implementation: NFR-1 says these MUST be re-verified against
> the client, not taken on faith. The IDA re-verification of `sub_664D42` /
> `CMob::GetFineAction` / `CMob::Init` is a plan step (§9), not a design
> assumption. The design's correctness rests on the encoding above; if the
> re-verification contradicts it, the helper's constants change but the
> architecture does not.

### The shared encoding

```
actionIndex = isFly ? 6 : 2
facingBit   = (fixedStance != 0) ? (fixedStance & 1) : 0   // 0 = right, 1 = left
moveAction  = byte(actionIndex<<1) | facingBit
```

`fixedStance` is `atlas-data`'s `getFixedStance` output: `4`/`5` for `noFlip`
mobs (fixed facing from `stand/0/origin`), else `0`. It contributes **only the
facing bit**, never the action index (FR-2.2). So a `noFlip` **fly** mob emits
`12/13`, not the ground `4/5` that `FixedStance` alone would imply.

Precomputed truth table (the §10 acceptance vectors):

| isFly | fixedStance | actionIndex | facingBit | byte |
|---|---|---|---|---|
| false | 0 | 2 | 0 | **4** |
| false | 4 | 2 | 0 | **4** |
| false | 5 | 2 | 1 | **5** |
| true  | 0 | 6 | 0 | **12** |
| true  | 4 | 6 | 0 | **12** |
| true  | 5 | 6 | 1 | **13** |

Every output satisfies `(byte & ~1) != 0` — the crash invariant holds by
construction for all four `(isFly, fixedStance∈{0,4,5})` inputs.

---

## 3. Architecture

Three touch-points across two services plus one shared lib. `atlas-data` is
unchanged (already produces `flying`, `swimming`, `fixed_stance`).

```
atlas-data (unchanged)
  └─ REST monster model: flying, swimming, fixed_stance, animation_times
       │
       ├──────────────► atlas-monsters  monster/information client
       │                   (extend DTO+Extract to carry the 3 fields)
       │                         │
       │                   Create(): stance = monster.IdleMoveAction(isFly, fixedStance)
       │                         │        ── replaces hardcoded 5
       │                         ▼
       │                   registry.stance ──(Kafka spawn/move, unchanged)──┐
       │                                                                     │
       └──────────────► atlas-channel  monster/information client           │
                           (extend DTO+Extract; existing TTL cache reused)   │
                                 │                                           ▼
                                 │                             monster.Model.Stance()
                                 ▼                                           │
                    monster_spawn.go / monster_control.go emit sites:        │
                    stance := resolveStance(m.Stance(), isFly, fixedStance)  ◄┘
                    // if 0|1 → IdleMoveAction(isFly,fixedStance); else verbatim
                                 │
                                 ▼
                    packetmodel.NewMonster(..., stance, ...)  ── WriteByte moveAction
```

### 3.1 Shared helper — `libs/atlas-constants/monster` (resolves OQ-1)

New file `libs/atlas-constants/monster/stance.go`:

```go
package monster

// Move-action (stance) idle encoding, verified against v83 client
// CMob::GetFineAction @0x671999 → sub_664D42.
const (
	idleActionIndexGround = 2
	idleActionIndexFly    = 6

	FacingRight byte = 0
	FacingLeft  byte = 1
)

// IdleMoveAction returns the pre-resolved idle move-action byte the client
// would compute in CMob::OnResolveMoveAction, so the server never sends the
// 0/1 sentinel that crashes the spawn/control CMob::Init path.
//
// isFly is Flying || Swimming (swim = fly-in-water, same client branch).
// fixedStance is atlas-data's getFixedStance output (4/5 for noFlip mobs,
// else 0); it contributes only the facing bit, never the action index.
func IdleMoveAction(isFly bool, fixedStance uint32) byte {
	actionIndex := byte(idleActionIndexGround)
	if isFly {
		actionIndex = idleActionIndexFly
	}
	facingBit := FacingRight
	if fixedStance != 0 {
		facingBit = byte(fixedStance & 1)
	}
	return actionIndex<<1 | facingBit
}
```

**Why `libs/atlas-constants/monster` and not `libs/atlas-packet/model` (OQ-1):**

- Both `atlas-monsters` and `atlas-channel` already have `atlas-constants` as a
  **direct** `require` (verified: both `go.mod` line 6). Neither would need a new
  dependency edge.
- `atlas-monsters` does **not** import `atlas-packet` in any `.go` file (verified
  by grep — only a stray `replace` directive with no `require`). Homing the
  helper in `atlas-packet/model` would force a **new** `atlas-monsters →
  atlas-packet` dependency purely for one pure function — the wrong direction
  (a data/registry service pulling in the wire-codec lib).
- CLAUDE.md DOM-21 explicitly directs numeric-constant/type reuse through
  `atlas-constants` first; `libs/atlas-constants/monster` already exists and
  already holds monster-domain constants (`skill.go`, `status.go`,
  `temporary_stat.go`, `constants.go`). The idle-action index/facing encoding
  is exactly that kind of shared numeric constant.

The helper is pure (no I/O, no context) → trivially unit-testable in the lib
against the §10 vectors, independent of either service.

### 3.2 `atlas-monsters` — fresh-spawn origin (FR-3)

`monster/information` currently drops fly/fixed-stance data. Extend it:

- **DTO** (`information/rest.go`): add `Flying bool json:"flying"` and
  `Swimming bool json:"swimming"`. `FixedStance uint32 json:"fixed_stance"`
  already exists on the DTO but is dropped in `Extract`.
- **Domain model** (`information/model.go`): add unexported `flying`, `swimming`,
  `fixedStance` fields with getters `IsFly() bool` (returns `flying||swimming`)
  and `FixedStance() uint32`. Keep the immutable private-fields+getters pattern.
- **`Extract`**: map the three fields through.
- **`Create`** (`monster/processor.go:198`): the processor already calls
  `information.NewProcessor(...).GetById(input.MonsterId)` into `ma` for
  `ma.Hp()/ma.Mp()`. Replace the literal `5` in `CreateMonster(...)` with
  `monster.IdleMoveAction(ma.IsFly(), ma.FixedStance())`.

No new fetch — `ma` is already retrieved at `Create`. The registry `stance`
field, its JSON, and the Kafka `stance`/`ni` byte are all unchanged (FR-3.3).

The `NewMonster(f, uniqueId, 0, 0, 0, 0, 0, 0, 0, 0)` diff/placeholder at
`processor.go:1511` is out of scope (FR-5.2) — not a spawn emit.

### 3.3 `atlas-channel` — emit-boundary guard (FR-4)

Extend the channel `monster/information` client the same way:

- **DTO** (`monster/information/rest.go`): add `Flying`, `Swimming`,
  `FixedStance` (today only `Attacks`).
- **Model** (`monster/information/model.go`): add `flying`, `swimming`,
  `fixedStance` with `IsFly()` and `FixedStance()` getters.
- **`Extract`**: map them through.
- **Cache (OQ-2 — already solved):** the channel client already has a
  tenant-scoped read-through TTL cache (`information/cache.go` +
  `processor.go GetById`) with positive/negative caching and metrics. The new
  fields ride inside the same cached `Model`. **No new cache is added** — NFR-2
  is satisfied by the code already on the path. The guard calls
  `information.NewProcessor(l, ctx).GetById(monsterId)`, which is a cache hit
  after the first lookup per (tenant, monsterId).

**The guard.** A single narrow helper resolves the sentinel:

```go
// resolveSpawnStance rewrites the 0/1 idle sentinel to the mob's fly-aware
// idle stance so the client never resolves it during CMob::Init (crash path).
// Any stance >= 2 is emitted verbatim (FR-4.3).
func resolveSpawnStance(l, ctx, stance byte, monsterId uint32) byte {
	if stance & ^byte(1) != 0 { // stance >= 2
		return stance
	}
	ma, err := information.NewProcessor(l, ctx).GetById(monsterId)
	if err != nil {
		// Fail safe: never emit the crashing sentinel. Ground idle right
		// (4) is the conservative floor; log at debug (NFR-4).
		l.WithError(err).Debugf("stance guard: info lookup failed for monster [%d]; flooring 0/1→4", monsterId)
		return monster.IdleMoveAction(false, 0)
	}
	resolved := monster.IdleMoveAction(ma.IsFly(), ma.FixedStance())
	l.Debugf("stance guard: rewrote sentinel [%d]→[%d] for monster [%d] (isFly=%t)",
		stance, resolved, monsterId, ma.IsFly())
	return resolved
}
```

Call it at both emit sites, right before `NewMonster`:

- `socket/writer/monster_spawn.go:~51` (`SpawnMonsterWithEffectBody`) — the
  existing spawn-wire debug log (`monster_spawn.go:48`) already prints `stance`;
  it will print the resolved value, keeping NFR-4 observable.
- `socket/writer/monster_control.go:~51` (`ControlMonsterBody`, the
  `controlType > Reset` branch that constructs `mem`).

`m.Stance()` (channel `monster.Model`, `model.go:80`) supplies the input.

**Helper home for `resolveSpawnStance`:** it lives in package `writer` next to
its two callers (it needs `l`/`ctx` and the `information` processor — service
plumbing, not lib-pure). Only the pure `IdleMoveAction` formula is shared via
the lib. This keeps the lib free of service dependencies.

---

## 4. Data flow (two scenarios)

**Fresh spawn (fixes flyers + fresh-spawn crash surface):**
`Create` → `IdleMoveAction(isFly,fixedStance)` → registry `stance` (`12/13` fly,
`4/5` ground) → Kafka → channel `m.Stance()` = `12/13`/`4/5` → guard sees
`>= 2` → **verbatim** → wire. Flyers now animate correctly; no sentinel.

**Moved-then-idle (fixes the real 0-path crash):** client move-fragment carries
`BMoveAction = 0`/`1` → channel `movement/processor.go:287…302` sets
`ms.Stance = 0` → Kafka → `atlas-monsters` `Move` → registry `stance = 0`. On the
next spawn/control emit, channel `m.Stance() = 0` → guard rewrites to
`IdleMoveAction(isFly,fixedStance)` → wire is `4/5`/`12/13`, never `0/1`.

The two fixes are complementary, not redundant (OQ-3): `Create` fixes fresh
flyers at the origin; the guard is the backstop for any `0/1` that a
move-fragment (or a future upstream regression) introduces after spawn. They
never disagree because they operate on disjoint inputs — the guard only fires on
`0/1`, which `Create` never produces. Both derive facing from the same
`fixedStance` via the same helper, so even the facing bit agrees.

---

## 5. Alternatives considered

**A1 — Helper in `libs/atlas-packet/model` (co-located with the `WriteByte`).**
Rejected: forces a new `atlas-monsters → atlas-packet` dependency (monsters
doesn't import the wire lib today); wrong direction for a data service. The
byte's *encoder* lives there, but its *value computation* is domain data shared
by two services — `atlas-constants` is the established shared-constant home
(DOM-21). See §3.1.

**A2 — Fix only `atlas-monsters` `Create`, no channel guard.** Rejected: the
PRD's *real* crash is the post-movement `0/1` sentinel persisted from a client
move-fragment and re-emitted verbatim (`monster_spawn.go:51`). `Create` never
touches that path — the flat `5` never protected it. The emit-boundary guard is
the only place that catches every producer of a spawn/control stance (FR-4.1,
the §10 invariant). Origin-only would leave the AV latent.

**A3 — Fix only the channel guard, leave `Create`'s `5`.** Rejected: the guard's
`>= 2` pass-through would let the ground-idle `5` reach flyers unchanged
(`GetFineAction` silently downgrades → wrong fly animation, FR-2 / the
second defect). `Create` must emit `12/13` for fresh flyers; the guard only
rewrites `0/1`.

**A4 — Clamp the sentinel to a flat non-fly-aware `4` everywhere.** Rejected:
re-introduces the wrong-flyer-animation bug the PRD explicitly calls out. The
fly-aware resolution is a hard requirement (FR-4.2). (A flat `4` remains only as
the guard's *fail-safe* when the info lookup errors — never the normal path.)

**A5 — New dedicated channel template cache for fly-class.** Rejected as
redundant: the channel `monster/information` client already has exactly this
cache (OQ-2). Adding a second would duplicate TTL/metrics/negative-cache logic.

---

## 6. Testing strategy

**Lib unit tests** (`libs/atlas-constants/monster/stance_test.go`) — the §10
vectors, table-driven, plus the invariant sweep:

- `(ground, 0)→4`, `(ground, 4)→4`, `(ground, 5)→5`,
  `(fly, 0)→12`, `(fly, 4)→12`, `(fly, 5)→13`.
- Invariant: for `isFly ∈ {true,false}` × `fixedStance ∈ {0,4,5}`,
  assert `(IdleMoveAction(...) & ^byte(1)) != 0` (never `0`/`1`).

**`atlas-monsters`** — `Extract` maps `flying`/`swimming`/`fixed_stance` into the
model (getters return expected `IsFly()`/`FixedStance()`); a `Create`-level test
(or focused test on the stance argument) asserting a fresh fly mob's registry
`stance` is `12` and a fresh ground mob's is `4`/`5`, using the project Builder
pattern + the existing `information/mock` processor (no `*_testhelpers.go`).

**`atlas-channel`** — `resolveSpawnStance` table test with a mocked
`information` processor (existing `monster/information/mock`): fly mob input `0`
→ `12`, `1` → `13` (if fixedStance=5) else `12`; ground input `0` → `4`; and the
pass-through cases input `5` → `5`, `4` → `4`, `12` → `12` unchanged. Plus an
info-lookup-error case → fail-safe `4`. `Extract` field-mapping test mirrors the
monsters side.

**Cache-on-path (§10 checkbox):** verified by inspection — the guard calls
`GetById`, which is the cached read-through; the existing `cache_test.go` already
covers hit/miss/negative. No per-mob synchronous `atlas-data` fetch on the hot
path once warm.

**No-layout-change:** the packet `Encode` (`libs/atlas-packet/model/monster.go`)
is untouched — only the `moveAction` *value* differs. The existing
`movement_test.go`/model tests continue to pass unchanged; a byte-level diff of a
spawn encode with the same stance is identical.

---

## 7. Non-functional coverage

- **NFR-2 (perf):** guard reads the already-existing TTL cache; warm path = no
  network. Fresh-spawn `Create` reuses the `ma` it already fetched.
- **NFR-3 (tenancy):** the channel cache is keyed by `(tenant.Id(), monsterId)`
  (existing `cache.go`); no cross-tenant bleed. All fetches stay context-scoped.
- **NFR-4 (observability):** the guard logs sentinel rewrites at debug; the
  existing spawn-wire debug log already prints the (now-resolved) `stance`.
- **NFR-5 (version scope):** the fix is version-agnostic — it only changes the
  *value* of an existing byte. `packetmodel.Encode` version-gates (the
  `MajorVersion() > 12` temporary-stat prefix) are untouched; no version's
  field count/order changes.

---

## 8. Scope fences (from PRD §5)

- Live movement-broadcast (relay-to-other-clients) path: **unchanged** (FR-5.1).
- `processor.go:1511` zero-value diff construction: **out of scope** (FR-5.2).
- No endpoint, DB, migration, or Kafka schema change. REST read-model additions
  are additive (older/newer peers ignoring the fields are unaffected).

---

## 9. Open questions — resolved

- **OQ-1 (helper home):** `libs/atlas-constants/monster/stance.go`. Evidence:
  both services already `require` `atlas-constants`; `atlas-monsters` does not
  import `atlas-packet`; DOM-21 directs shared numeric constants here. (§3.1)
- **OQ-2 (channel cache):** reuse the existing `monster/information` TTL cache;
  add no new cache. (§3.3)
- **OQ-3 (double-resolution):** both are needed and cannot disagree — disjoint
  inputs, shared helper, shared facing derivation. (§4)
- **OQ-4 (noFlip flyer):** the formula is correct by construction — `fixedStance`
  contributes only the facing bit, `isFly` always drives `actionIndex` → a
  `noFlip` flyer emits `12/13` with fixed facing. Whether such a mob exists in
  the target catalogs is moot to correctness; the lib unit test covers
  `(fly, 5)→13` regardless. A concrete-catalog spot-check is a nice-to-have
  verification step, not a design dependency.

**Deferred to the implementation plan (not design assumptions):** the NFR-1 IDA
re-verification of `sub_664D42`/`CMob::GetFineAction`/`CMob::Init` against the
v83 client — a plan step that confirms (or corrects) the helper's constants
before the codec value is trusted.

---

## 10. Change inventory

| Location | Change |
|---|---|
| `libs/atlas-constants/monster/stance.go` (new) | `IdleMoveAction` + constants |
| `libs/atlas-constants/monster/stance_test.go` (new) | §10 vectors + invariant |
| `atlas-monsters .../monster/information/rest.go` | DTO: `+Flying,+Swimming`; `Extract` maps `flying/swimming/fixed_stance` |
| `atlas-monsters .../monster/information/model.go` | `+flying,+swimming,+fixedStance` + `IsFly()`,`FixedStance()` |
| `atlas-monsters .../monster/processor.go:198` | `5` → `monster.IdleMoveAction(ma.IsFly(), ma.FixedStance())` |
| `atlas-channel .../monster/information/rest.go` | DTO: `+Flying,+Swimming,+FixedStance`; `Extract` maps them |
| `atlas-channel .../monster/information/model.go` | `+flying,+swimming,+fixedStance` + `IsFly()`,`FixedStance()` |
| `atlas-channel .../socket/writer/monster_spawn.go` | call `resolveSpawnStance` before `NewMonster` |
| `atlas-channel .../socket/writer/monster_control.go` | call `resolveSpawnStance` before `NewMonster` |
| `atlas-channel .../socket/writer/*_test.go` (new/extended) | guard table tests |
| `atlas-data` | **none** |
| `libs/atlas-packet` | **none** (layout unchanged) |

Changed modules to verify (CLAUDE.md build rules): `libs/atlas-constants`,
`atlas-monsters`, `atlas-channel` — `go test -race`, `go vet`, `go build` each;
`docker buildx bake atlas-monsters` + `atlas-channel`; `tools/lint.sh --check`,
`redis-key-guard.sh`, `goroutine-guard.sh` from repo root.
