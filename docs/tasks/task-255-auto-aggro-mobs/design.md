# Auto-Aggro / First-Attack Mobs — Design

Task: task-255-auto-aggro-mobs
PRD: [prd.md](prd.md)
Status: Draft for review
Created: 2026-08-21

---

## 0. Summary

`AUTO_AGGRO` is `CMob::ApplyControl`. Every client that can see a mob whose
template carries `bFirstAttack` **or** `bPickUpDrop` sends it at most once per
second, carrying the mob's object id and a client-computed proximity score. It
is a **request to become the mob's controller**, not a request from the
controller to flip a flag.

That single fact — derived from the binary, not assumed — is the difference
between this design and the PRD's provisional shape. Two PRD requirements are
revised as a result (§9); everything else stands.

The implementation is smaller than the PRD anticipated, because the two hardest
pieces already exist:

- `ProcessorImpl.startControl(uniqueId, controllerId, forceAggro=true)` in
  atlas-monsters already performs the atomic controller-transfer-with-aggro that
  Monster Magnet's `FORCE_CONTROL` uses.
- `handleStatusEventAggroChanged` in atlas-channel already re-issues
  `StartControlMonsterBody(m, aggro)` on `AGGRO_CHANGED`, and
  `StatusEventStartControlBody` already carries `ControllerHasAggro`. FR-6.1 and
  FR-6.2 are **already satisfied by shipped code**; this task adds tests, not
  behaviour, for them.

What is genuinely new: the codec, the channel handler plus its rate gate, a
`SET_AGGRO` command, a `firstAttack` accessor, and an aggro **lease** so an
auto-aggro'd mob with no damage entries eventually goes passive.

---

## 1. Client derivation (FR-1.2, FR-2.1, FR-2.2)

### 1.1 The send site

`CMob::ApplyControl(int tCur)` — called unconditionally from `CMob::Update`
(v95 `0x654bdc`), immediately before the `TryPickUpDrop` branch:

```c
if ( (fuse(m_pTemplate->bPickUpDrop) || fuse(m_pTemplate->bFirstAttack))
  && tCur - fuse(m_tLastApplyCtrl) >= 1000 )
{
    m_tLastApplyCtrl = tCur;
    dx = |CUserLocal.pos.x - this->GetPos().x| / 10;
    dy = |CUserLocal.pos.y - this->GetPos().y| / 3;
    n  = dx + dy;
    if ( (CUserLocal.<moveActionField> & 0xFFFFFFFE) == 0x12 )   // nMoveAction 18/19
        n += 100;
    COutPacket(oPacket, <opcode>);
    Encode4( fuse(m_dwMobID, m_dwMobID_CS) );   // plaintext mob object id
    Encode4( n );
    SendPacket(oPacket);
}
```

Three consequences the PRD could not know:

1. **Any client sends it**, controller or not. There is no controller test in
   `ApplyControl` and none at the `CMob::Update` call site.
2. **`bPickUpDrop` alone triggers it.** A passive mob that picks up drops sends
   `AUTO_AGGRO` too. `firstAttack` must be checked server-side (it already is,
   FR-4.2 step 5) or drop-picking mobs would become aggressive.
3. **The second field is a proximity score**, computed with exactly the divisors
   `CMob::TryFirstAttack` uses for its own chase test
   (`v29 = |dx|/10 + |dy|/3; ... || v29 <= 40`, v95 `0x6482f0`). The client's own
   "close enough to chase" bar is therefore `score <= 40`, and we adopt it.

`CMob::TryFirstAttack` — which the PRD named as the sender — does **not** send
anything; it calls `CMob::ChaseTarget`. The chase is client-side AI. The server's
aggro flag drives the control packet's aggro byte, which is what arms that AI on
the controlling client.

### 1.2 Per-version addresses and opcodes

All ten binaries encode the identical two `Encode4` body. **There is no version
divergence, so no `MajorAtLeast` gate is required** (FR-1.4 is satisfied
vacuously, and adding a gate would be noise).

| Version | IDB | `CMob::ApplyControl` | Opcode (dec / hex) | Registry today |
|---|---|---|---|---|
| gms_v48 | `GMS_v48_1_DEVM.exe` | `0x551c79` | 130 / `0x082` | **`n-a`** ← wrong |
| gms_v61 | `GMS_v61.1_U_DEVM.exe` | `0x5ccf1c` | 156 / `0x09C` | **`n-a`** ← wrong |
| gms_v72 | `GMS_v72.1_U_DEVM.exe` | `0x61d358` | 179 / `0x0B3` | **`n-a`** ← wrong |
| gms_v79 | `GMS_v79_1_DEVM.exe` | `0x63d0e6` | 181 / `0x0B5` | **`n-a`** ← wrong |
| gms_v83 | `MapleStory_dump.exe` | `0x66e146` | 189 / `0x0BD` | 189 ✔ |
| gms_v84 | `GMS_v84.1_U_DEVM` | `0x684492` | **194 / `0x0C2`** | **189** ← wrong |
| gms_v87 | `GMSv87_4GB.exe` | `0x6a9061` | 201 / `0x0C9` | 201 ✔ |
| gms_v92 | `GMS_v92_1_DEVM.exe` | `0x636320` | 221 / `0x0DD` | 221 ✔ |
| gms_v95 | `GMS_v95.0_U_DEVM.exe` | `0x640d20` | 228 / `0x0E4` | 228 ✔ |
| jms_v185 | `MapleStory_dump_SCY.exe` | `0x6eba3c` | 195 / `0x0C3` | 195 ✔ |

Five of these functions were unnamed in their IDBs (v61, v72, v79, v84, v92);
they have been renamed to `?ApplyControl@CMob@@IAEXJ@Z` and the IDBs saved, so
the next reader finds them by name.

### 1.3 Registry corrections this task must make

Both are *corrections of unverified csv-import inheritance*, grounded in the
binary — not new opinions.

- **gms_v84 `AUTO_AGGRO` opcode 189 → 194.** The entry's own note says it was
  "seeded from the v83 CSV column — the CSVs have no v84 column". The v84 binary
  sends `0xC2`. Cross-check: v84 `MOB_DROP_PICKUP_REQUEST` is `ida-discovered`
  as 195, and the v84 binary's `CMob::SendDropPickUpRequest` (`0x684cdf`) does
  encode 195 — so 194/195 are an adjacent pair and 189 is simply stale. Bump
  `provenance` to `ida-discovered` with a note citing `0x684492`.
- **gms_v48 / v61 / v72 / v79 `n-a` → real opcodes.** FR-2.2 asked us to confirm
  the `n-a` rather than inherit it. It is wrong on all four: every one of those
  clients has `CMob::ApplyControl` and sends the packet. These four are the CSV's
  blind spot (the CSV has no column before GMS v12/v83), not a client absence.
  **No `feature-na-evidence.yaml` entry is written** — nothing is `n-a`.

Net: `AUTO_AGGRO` is routed and verified on **ten** version columns, not six.
`gms_12_1` remains out of scope (task-175 owns it; no registry column).

---

## 2. Architecture decision 1 — what `SET_AGGRO` actually does

The PRD's FR-4.2 gates on "the requesting character is the monster's **current
controller**". Under the real client behaviour that gate rejects the common case:
Atlas elects controllers by **least-loaded count**, not by proximity
(`getControllerCandidate`, processor.go:306), so the player who walks into an
aggressive mob's range is frequently not its controller.

Three candidate semantics:

| | Behaviour | Verdict |
|---|---|---|
| **A. Aggro-only** (PRD-literal) | Non-controller requests dropped; controller requests set the flag. | Rejected. Silently no-ops for a large fraction of real encounters — the mob ignores the player who is actually standing next to it, which is the exact bug this task exists to fix. |
| **B. Control-transfer** (client-literal) | Every qualifying request transfers control to the requester with aggro. | Rejected alone. With N clients seeing the mob and each sending 1/s, control thrashes between them once per second, and every thrash is a STOP+START control pair on the wire. |
| **C. Hybrid** ✅ | Requester is controller → flip aggro in place. Requester is not the controller → transfer control **with** aggro, but only if the current controller does not already hold aggro. | Recommended. |

**C is the design.** Its anti-thrash rule reads naturally: *the first player to
get close enough owns the mob until it de-aggros.* Once aggro is held, later
requests from other clients are dropped, so the 1/s multi-client fan-in
collapses to at most one accepted request per mob per aggro episode.

The transfer path is **not new code**: it is
`startControl(uniqueId, characterId, forceAggro=true)`, the same call
`ForceControl` makes for Monster Magnet, which already emits `START_CONTROL`
with `ControllerHasAggro: true`, already skips the redundant stop/start when the
target is the current controller, and already excludes GM-hidden characters.

---

## 3. Architecture decision 2 — the proximity gate

The payload's score is the client's own chase metric. The server applies
`score <= AutoAggroProximityThreshold` with the threshold defined as **40**,
the constant `CMob::TryFirstAttack` compares against (v95 `0x6482f0`).

This buys three things at zero cost:

- Aggro is granted only when the client itself considers the player chase-close,
  so a mob three screens away does not turn hostile.
- It gives the aggro **lease** (§4) a natural expiry: a player who walks away
  keeps sending requests, but with `score > 40`, so the lease is not refreshed.
- The `+100` bias for `nMoveAction` 18/19 pushes that state out of range
  automatically. We do **not** claim to know which state that is — the low bit
  is the facing flag, so it is move-action index 9 — and nothing in the design
  depends on identifying it.

Caveat, stated rather than papered over: `TryFirstAttack` also chases when
`IsTargetInAttackRange(...)` is true regardless of score. Attack range is far
tighter than a score of 40, so `score <= 40` is the looser bar in practice and no
real chase is excluded. The threshold is a named constant so a live-test finding
can move it without a redesign.

---

## 4. Architecture decision 3 — how auto-aggro releases (FR-6.3, OQ3)

**Today it never would.** `MonsterAggroDecayTask.Run` skips any monster with
`len(entries) == 0` (aggro_task.go), and `DecayDamageEntries` only flips
`ControllerHasAggro` off when a non-empty entry list decays to empty. Auto-aggro
writes no damage entry (FR-4.5), so nothing in the existing sweep would ever
touch it.

Leaving it set is not harmless: `startControl`'s re-pick gate and `UseSkill`'s
`ControllerHasAggro` gate both read the flag, so every mob a player ever walked
past would keep making skill decisions forever, casting at nobody.

**Design: an aggro lease.**

- `storedMonster` gains `AggroRefreshedMs int64` (Redis registry state — no DB,
  no migration; consistent with `LastDamageTakenMs` alongside it).
- Every accepted `SET_AGGRO` stamps it, including the idempotent
  already-aggro'd-by-the-same-controller case. The refresh is the *only* work
  that case does — no state transition, no event (FR-4.3 preserved: no second
  `AGGRO_CHANGED`).
- `MonsterAggroDecayTask` gains one branch: for a monster with **no damage
  entries** that holds aggro and whose lease is older than
  `AutoAggroLeaseTtl = 15s`, clear the flag and emit `AGGRO_CHANGED(false)`,
  reusing the existing `aggroChangedStatusEventProvider` emit the sweep already
  performs. Damage-driven aggro (entries present) keeps its existing path
  untouched, and a mob that is both damaged and auto-aggro'd is governed by the
  damage path — entries present means the lease branch does not apply.
- Bosses stay excluded, as they are today.

The channel refreshes at most once per `AutoAggroRefreshInterval = 5s` per
(character, mob) (§5.2), so the 15s TTL tolerates two missed refreshes before
releasing.

Alternative considered and rejected: no release at all ("real GMS leaves the flag
set"). Faithful to the client but wrong for Atlas, because Atlas's skill-decision
loop reads the flag and the client's does not.

---

## 5. Architecture decision 4 — Dark Sight (FR-7.3, OQ4)

**No server-side check is added.** Grounded, not assumed:

- `CMob::ApplyControl` has no dark-sight gate, so a hidden character's client
  does send `AUTO_AGGRO`. A gate placed here would be the only thing stopping it.
- But it does not need stopping. Task-231's client analysis
  (`client-analysis-dark-sight-touch-damage.md`) established that mob attack
  *arming* is gated client-side on `!CUser::IsDarkSight && !CUser::IsSneak`
  inside `CMob::Update` — under Dark Sight `IsTargetInAttackRange` returns 0,
  `GenerateMovePath`'s attack branch is never taken, and `DoAttack` never runs.
  The suppression is "the mob never arms an attack against an invisible player,"
  which holds whether or not the server flag is set.
- The alternative — a per-packet buff lookup — costs a REST call on a
  once-per-second-per-mob path to prevent a flag flip with no observable effect.

GM hide is separately and already handled: the shared control path excludes
GM-hidden characters via `hiddenFn`, and `RelinquishControlOnHide` actively
strips their control.

If live testing (§10) shows an aggressive mob chasing a dark-sighted Rogue, the
fix is a channel-side gate using the session's buff state — recorded here as the
known escalation, not pre-built.

---

## 6. Component design

### 6.1 Codec — `libs/atlas-packet/monster/serverbound/auto_aggro.go`

```go
const AutoAggroHandle = "AutoAggro"

type AutoAggro struct {
    mobId    uint32  // plaintext CMob::m_dwMobID (the client fuses before encoding)
    distance uint32  // |dx|/10 + |dy|/3, +100 in nMoveAction 18/19
}
```

Immutable, both `Encode` and `Decode`, `Operation()`, `String()`, and a
`packet-audit:fname CMob::ApplyControl` marker. The doc comment carries the §1.2
address table.

Naming note against FR-1.3: the sibling `MobDropPickupRequest` calls its first
field `mobCrc`. That name is a misnomer inherited from the send site's
`_ZtlSecureFuse` call — **fuse recovers the plaintext value**; the wire carries
the mob object id verbatim, which is why `MobDropPickupRequest`'s consumers can
treat it as one. This codec names the field `mobId` and says so in the comment
rather than propagating the misnomer. Renaming the sibling is out of scope.

### 6.2 Channel handler — `socket/handler/auto_aggro.go`

Registered in `main.go` as `handlerMap[monstersb.AutoAggroHandle]`.

Order of cheap local checks (FR-3.3 — none of these is the authority):

1. session has a character and a field;
2. `p.Distance() <= monster.AutoAggroProximityThreshold`;
3. the mob is in the live mirror and its `Field` matches the session's field;
4. the rate gate (§6.3) admits it.

Then `monster.NewProcessor(l, ctx).SetAggro(f, p.MobId(), characterId)`, which
emits `SET_AGGRO` on `COMMAND_TOPIC_MONSTER` via a new
`SetAggroCommandProvider` mirroring `ForceControlCommandProvider`.

Any failed check is a `Debugf` drop with the mob id, the character id, and which
check failed. No response packet — `AUTO_AGGRO` has no client-visible failure
path (FR-3.4).

### 6.3 Rate gate — `monster/auto_aggro_gate.go` (channel)

A per-pod, tenant-scoped map keyed `(characterId, mobId)` holding the last
forward time, swept on the same shape as `LiveMirror`'s staleness sweeper.

- Mirror says the mob is **not** aggro'd → forward, subject to a 1s floor (the
  client already self-throttles to 1s; the floor guards a modified client).
- Mirror says the mob **is** aggro'd → forward at most once per
  `AutoAggroRefreshInterval` (5s). This is the lease refresh, and it is why the
  gate throttles rather than suppresses.

Steady state cost: one command per aggro'd mob per 5s from its controller, plus a
bounded trickle from other nearby clients that atlas-monsters drops on the
already-aggro'd rule. Without this gate a 20-mob, 10-player map would produce
~200 commands/s.

`LiveEntry` gains `ControlCharacterId uint32`, maintained by the existing
start/stop-control handlers, so the gate can prefer "am I the controller"
without a REST call. It is an optimisation, never an authority.

### 6.4 Command — `SET_AGGRO`

Added symmetrically to both sides of `COMMAND_TOPIC_MONSTER`
(`atlas-channel/.../kafka/message/monster/kafka.go` and
`atlas-monsters/.../kafka/consumer/monster/kafka.go`, which carry
edit-both-together comments):

```
type: "SET_AGGRO"
<existing envelope: worldId, channelId, mapId, instance, monsterId>
body: { characterId: uint32 }
```

The proximity score is deliberately **not** on the command. It is a channel-side
admission criterion; carrying it into atlas-monsters would put a client-computed
number into the authoritative service for no decision it makes.

Consumer arm `handleSetAggroCommand` mirrors `handleForceControlCommand`
exactly, including its `PersistentConfig` registration.

### 6.5 `ProcessorImpl.SetAggro(uniqueId, characterId uint32) error`

Every rejection is a logged drop returning `nil` — never an error — for the same
reason `ForceControl` does it: a stale client-driven target must not wedge the
consumer.

Gates, in order:

1. **monster exists** — `GetById` fails → drop.
2. **monster is alive** — `Hp() > 0` → else drop.
3. **template is aggressive** — `information.NewProcessor(l, tctx).GetById(m.MonsterId()).FirstAttack()`.
   Lookup error or cache miss → **deny** (FR-5.2: uncertainty denies). This is
   the gate that rejects `bPickUpDrop`-only mobs (§1.1 consequence 2). It goes
   through the existing read-through cache and is injected as a lookup func in
   the `bossLookupFn` style so tests do not need HTTP.
4. **character is in the monster's field** — `inFieldFn`.
5. **arbitration**:
   - `m.ControlCharacterId() == characterId` → refresh the lease. If
     `ControllerHasAggro` was already true: **no state change, no emit** (FR-4.3).
     Otherwise flip it true and emit `AGGRO_CHANGED(characterId, true)`.
   - else if `m.ControllerHasAggro()` → drop (anti-thrash, §2).
   - else → `startControl(uniqueId, characterId, forceAggro=true)`, which stamps
     the lease and emits `START_CONTROL` carrying `ControllerHasAggro: true`.
     GM-hidden exclusion happens inside that path.

**No damage entry is created on any path** (FR-4.5). Auto-aggro confers no drop
ownership and no kill credit.

Registry support: a new `SetAggro(t, uniqueId, characterId)` atomic update
returning `{Monster, Changed bool}` so the emit decision is made from the same
atomic result — the shape `ClearSummary`/`DecaySummary` already establish.

### 6.6 `information.Model.FirstAttack()`

`firstAttack bool` field, `FirstAttack()` accessor, mapped from the existing
`RestModel.FirstAttack` (`first_attack`) in `Extract`, plus a
`ModelBuilder.SetFirstAttack` for tests. Four-line change; `atlas-data` already
emits the field.

---

## 7. Data flow

```
client (any, ≤1/s)  ──AUTO_AGGRO{mobId, score}──▶  atlas-channel handler
                                                     │ score>40 / not in mirror / rate-gated → drop
                                                     ▼
                                    COMMAND_TOPIC_MONSTER  SET_AGGRO{characterId}
                                                     ▼
                                          atlas-monsters SetAggro
                        ┌──────────────────────┴──────────────────────┐
        controller & already aggro'd                        not controller & unaggro'd
        → stamp lease, no emit                              → startControl(forceAggro)
                                                            → START_CONTROL{aggro:true}
        controller & unaggro'd
        → flip, AGGRO_CHANGED{true}
                                                     ▼
                       atlas-channel consumer: mirror update + StartControlMonsterBody(m, 1)
                                                     ▼
                                   controlling client arms CMob hostile AI

  ... 15s with no refresh and no damage entries ...
  MonsterAggroDecayTask → AGGRO_CHANGED{false} → StartControlMonsterBody(m, 0)
```

---

## 8. Error handling, observability, multi-tenancy

- **Every gate logs at debug** with the monster unique id, the requesting
  character id, and the failing gate name — so "cheating client" vs "broken
  controller bookkeeping" is separable from pod logs alone.
- **No error returns from the consumer arm.** Drops, not retries.
- **Tenancy**: the information lookup builds `tenant.WithContext(ctx, t)` exactly
  as `bossLookupFn` does — the upstream `atlas-data` middleware rejects requests
  without `TENANT_ID`. The gate map, the mirror, and the registry are all
  tenant-keyed already.
- **Performance**: the idempotent path costs one registry read/stamp and nothing
  else — no Kafka emit, no template lookup ordering issue (the template lookup
  is cached and precedes arbitration; if that ever shows up in a profile, moving
  gate 3 after gate 5's idempotent short-circuit is a safe local change).
- **Monster Magnet invariant preserved**: `CLEAR_AGGRO` still clears an
  auto-aggro'd mob (it clears the flag unconditionally when set), and the
  `CLEAR_AGGRO`-strictly-before-`FORCE_CONTROL` ordering is untouched. Clearing
  also makes the lease moot — no entries, no aggro, nothing for the sweep to do.

---

## 9. Deltas from the PRD

| PRD | Change | Why |
|---|---|---|
| FR-4.2 step 4: requester must be the **current controller** | Replaced by the §2 hybrid: controller → flip; non-controller → transfer with aggro unless someone already holds aggro | The client sends from any client, and Atlas elects controllers by load, not proximity. The literal gate would no-op the common case. |
| FR-2.1: six routed versions | **Ten** routed versions | v48/v61/v72/v79 `n-a` is wrong — all four binaries send it (§1.2). |
| FR-2.2: record grounded `n-a` in `feature-na-evidence.yaml` | Not applicable | Nothing is `n-a`. |
| — (not in PRD) | gms_v84 opcode 189 → **194** | csv-import inheritance contradicted by the v84 binary. |
| FR-1.4: version-gate divergence with `MajorAtLeast` | No gate | All ten versions are byte-identical. |
| FR-6.1 / FR-6.2 | Already implemented; this task adds tests only | `handleStatusEventAggroChanged` and `StatusEventStartControlBody` already do it. |
| FR-6.3 / OQ3 | Aggro **lease** (§4) | Nothing else would ever release it, and a permanently-aggro'd mob keeps making skill decisions. |
| FR-7.3 / OQ4 | No server-side Dark Sight check | Client gates attack arming (task-231); a per-packet buff lookup buys nothing. |
| — (not in PRD) | `firstAttack` gate also rejects `bPickUpDrop`-only mobs | The packet fires for drop-pickers too. |
| OQ6 (aggro on spawn) | Confirmed: leave spawn passive | The client sends `AUTO_AGGRO` on its next tick, ≤1s later. |
| OQ5 | Confirmed: gms_v92 in scope, gms_12_1 out | v92 has a registry column and a template; gms_12_1 has neither. |

Acceptance criteria affected: the STATUS.md row must be verified on **ten**
columns; the `feature-na-evidence.yaml` criterion is void; the FR-4.2 "not the
controller" test case asserts **transfer**, not rejection; two new cases cover
anti-thrash and lease release.

---

## 10. Testing

**Packet** — byte-fixture tests per version (ten), each with a
`packet-audit:verify` marker, plus an `Encode`/`Decode` round-trip.

**Channel handler** — table-driven: emits `SET_AGGRO` with the sender's character
id for a valid packet; emits nothing when the mob is absent from the mirror,
when the mirror field differs, when `score > 40`, and when the rate gate is
closed; forwards a refresh once the 5s interval elapses.

**Monsters `SetAggro`** — one case per gate: unknown monster, dead monster,
passive template, information-lookup error (denies), character not in field.
Then arbitration: controller-without-aggro flips and emits once;
controller-with-aggro emits nothing but stamps the lease (FR-4.3);
non-controller against an unaggro'd mob transfers control with
`ControllerHasAggro: true`; non-controller against an aggro'd mob drops. Plus
`DamageEntries` untouched on every success path (FR-4.5).

**Lease** — decay task releases a damage-entry-free aggro'd monster past TTL and
emits `AGGRO_CHANGED(false)`; does not release one inside TTL; does not touch a
monster with damage entries; bosses still skipped.

**Lifecycle** — control packet after `AGGRO_CHANGED(true)` carries `aggro = 1`;
aggro survives a controller handover; the Monster Magnet ordering test still
passes.

**Information** — `FirstAttack()` returns the DTO's `first_attack`.

**Manual (live channel)** — walk past a Jr. Necki (aggressive) → mob turns and
attacks unprovoked; walk past a Ribbon Pig (passive) → unchanged; walk away →
mob goes passive within ~15s; Dark Sight past an aggressive mob → not attacked.

---

## 11. Packet coverage artifacts

- Registry: ten `AUTO_AGGRO` entries updated — four `n-a` → real opcodes, one
  corrected (v84), all off `csv-import` to `ida-discovered` with the §1.2
  address in the note.
- Seed templates: `AutoAggro` routed in all ten applicable templates
  (`template_gms_{48,61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`)
  with `LoggedInValidator`, `fname: CMob::ApplyControl`, `services: [channel]`.
  `template_gms_12_1.json` is untouched.
- `docs/packets/audits/status.json` / `STATUS.md` regenerated; ten evidence
  records pinned. A cell that does not promote is a failure, not a claim.
- `packet-audit` matrix / fname-doc / operations `--check` must exit 0.
- `docs/tasks/task-255-auto-aggro-mobs/coverage-manifest.yaml` declares the ten
  `AUTO_AGGRO` op×version cells and nothing else, for
  `packet-completeness-critic`.

---

## 12. Risks

- **Command volume.** The rate gate (§6.3) is the only thing between a dense
  aggressive map and a Kafka storm. It is the piece to watch in live testing; the
  two constants (`AutoAggroRefreshInterval`, `AutoAggroLeaseTtl`) are the dials.
- **Threshold fidelity.** `40` is the client's constant, but `IsTargetInAttackRange`
  can chase outside it (§3). If live testing shows mobs failing to aggro at the
  edge, raise the threshold — but not past 100, or the `nMoveAction` 18/19 bias
  stops suppressing.
- **Control churn on contested mobs.** Anti-thrash binds aggro to the first
  claimant for the lease's duration; a second player closer to the mob waits up
  to 15s. Accepted: the alternative is per-second control thrash.
- **v84 opcode correction blast radius.** Correcting 189 → 194 changes only the
  `AUTO_AGGRO` row. Confirmed: no other serverbound entry in
  `docs/packets/registry/gms_v84.yaml` claims 189 or 194, and this task adds the
  first routing for either.
