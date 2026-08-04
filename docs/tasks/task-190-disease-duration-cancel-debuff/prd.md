# Disease Duration Units & CANCEL_DEBUFF — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-04
---

## 1. Overview

Two defects combine to wedge a client's temporary-stat state until the attacking
monster stops re-casting. Neither is new; both live in `main` today.

**Defect A — mob-skill disease durations are emitted in seconds into a
milliseconds field.** `atlas-buffs` has interpreted the `COMMAND_TOPIC_CHARACTER_BUFF`
`duration` field as milliseconds since task-054 (`197324e40`, 2026-05-03), but two
producers were never migrated and still send raw WZ `time` in seconds. Every one of
the 13 mob-applied diseases (SEAL, DARKNESS, WEAKEN, STUN, CURSE, POISON, SLOW,
SEDUCE, CONFUSE, UNDEAD, STOP_PORTION, STOP_MOTION, FEAR) therefore expires roughly
1000× too early — a 15-second Slow becomes a 15-millisecond buff.

**Defect B — the client's recovery path is unimplemented.** Because the buff is
already expired by the time the `SET_TEMPORARY_STAT` packet is encoded
(`libs/atlas-packet/model/character_temporary_stat.go:666` computes
`et := int32(v.ExpiresAt().Sub(time.Now()).Milliseconds())`, which is ≤ 0), the client
receives a stat that is born expired and calls `CWvsContext::CheckTemporaryStatDuration`,
sending serverbound `CANCEL_DEBUFF` roughly 30×/second to ask the server to drop it.
`CANCEL_DEBUFF` has **no codec, no handler, and no template routing anywhere in the
repo** — `atlas-channel` logs `Read a unhandled message with op 0x63` and does nothing.
The client's local disease state is never cleared, and it wedges.

**Observed live** (`atlas-pr-1138`, GMS 83.1, 2026-08-04): mob 7130002 cast skill 126
at 22:53:10.242 emitting `{"sourceId":126,"duration":15,"changes":[{"type":"SLOW","amount":80}]}`.
123 ms later the client began the `0x63` loop, which ran to 22:56:54.319 — ~1,500
unhandled packets over 3m44s, ending 60 ms after the last server-side expiry. During
the loop the client stopped sending `CharacterBuffCancel` and attack packets entirely.

Defect B is the more consequential of the two: it makes **any** server/client
temporary-stat disagreement unrecoverable, not just this one. Defect A is merely the
trigger that exposed it.

## 2. Goals

Primary goals:

- Mob-skill and mist-applied diseases last their authored WZ duration on the client and
  the server.
- The `COMMAND_TOPIC_CHARACTER_BUFF` `duration` unit contract is enforced mechanically,
  not by comment, so it cannot silently regress a fourth time (see §9.1).
- A client that believes a temporary stat has expired can tell the server so, and the
  server honors it — closing the wedge class of bug permanently.
- No double-conversion regressions in the call sites that currently do their own ×1000.

Non-goals:

- Rebalancing any disease's gameplay effect, magnitude, or proc rate. Durations move to
  their authored WZ values; nothing is retuned.
- The `MountBuffDuration = math.MaxInt32` question (`skill/handler/mount.go:29`) and any
  client-side tick-overflow behavior it may cause. Separate concern, separate task.
- Server-authoritative disease immunity/resistance rules beyond what
  `atlas-buffs character/immunity.go` already implements.
- Re-verifying packet-coverage-matrix cells unrelated to `CANCEL_DEBUFF`.
- Implementing `USER_CALC_DAMAGE_STAT_SET_REQUEST` (v83 `0x6C`,
  `CWvsContext::OnTemporaryStatReset`), also unhandled and adjacent. Noted in §9.3.

## 3. User Stories

- As a player, I want a monster's Slow/Seal/Stun to last the few seconds it is supposed
  to and then end, so combat behaves as designed.
- As a player, I do not want a debuff to leave my client unable to use skills or dismount
  until the monster loses interest in me.
- As a server operator, I want a desynced temporary stat to self-heal from the client's
  cancel request rather than requiring the player to relog.
- As an engineer, I want the seconds-vs-milliseconds contract to fail a test when I get it
  wrong, rather than shipping and being found by a player three months later.

## 4. Functional Requirements

### FR-1 — Duration unit correction

**FR-1.1** `services/atlas-data/atlas.com/data/mobskill/reader.go:66` MUST convert WZ
`time` from seconds to milliseconds, matching the convention already documented and
applied at `services/atlas-data/atlas.com/data/skill/reader.go:170-174`. After this
change `mobskill.RestModel.Duration` and `mobskill.Model.Duration()` are milliseconds.

**FR-1.2** The three existing consumers of `mobskill.Model.Duration()` that currently
perform their own seconds→ms math MUST be corrected in the same change, or they will
double-convert. These are not optional follow-ups; shipping FR-1.1 alone silently breaks
two of them:

| Site | Today | Required | Consequence if missed |
|---|---|---|---|
| `atlas-monsters .../monster/processor.go:1068` (`buildMistCreateBody`) | `durMs := int64(sd.Duration()) * int64(time.Second/time.Millisecond)` | drop the `* 1000` | 1000× inflation is masked by the `MistDurationCapMs` (60 000) clamp, silently pinning **every** mob-cast mist to exactly 60 s |
| `atlas-monsters .../monster/processor.go:1105` (`executeStatBuff`) | `time.Duration(sd.Duration()) * time.Second` | `* time.Millisecond` | monster self-buffs/immunity/reflect last ~1000× too long (a 20 s immunity becomes ~5.5 h) |
| `atlas-maps .../tasks/mist_tick.go:86` | `Duration: int32(m.DiseaseDuration() / time.Second)` | `/ time.Millisecond` | mist DoT re-apply stays ~1000× too short |

**FR-1.3** `atlas-monsters .../monster/processor.go:1242` (`executeDebuff`) requires no
edit — it forwards `sd.Duration()` verbatim and becomes correct once FR-1.1 lands. The
implementation MUST NOT add a second conversion here.

**FR-1.4** The stale comment at `atlas-maps .../tasks/mist_tick.go:81-86` asserting
*"atlas-buffs treats Duration as SECONDS (buff.NewBuff multiplies by time.Second)"* MUST be
replaced. It documents the pre-task-054 contract and is the direct cause of the FR-1.2
mist entry. The replacement MUST state the ms contract and note that this change reverses
commit `11e07dfa7`, so a future reader does not flip it back a third time.

**FR-1.5** All **seven** services producing onto `COMMAND_TOPIC_CHARACTER_BUFF` MUST be
audited for the same defect, with the finding recorded per service (correct / fixed /
n-a). Only two are known-broken; the remaining five are unverified:

| Service | Status entering this task |
|---|---|
| atlas-monsters | **broken** (FR-1.2/1.3) |
| atlas-maps | **broken** (FR-1.2) |
| atlas-channel | unverified |
| atlas-consumables | believed correct (fixed in task-140, `88d270bf1`) — re-verify |
| atlas-summons | unverified |
| atlas-messages | unverified |
| atlas-saga-orchestrator | unverified |

**FR-1.6** Tests that pin the seconds contract MUST be updated, not deleted, and their
comments corrected: `atlas-monsters .../monster/processor_test.go` (≈`:894`, `:1179`,
`:1236`) and `atlas-maps .../tasks/mist_tick_test.go` (≈`:163`). Line numbers are
indicative; the implementation must locate them by content.

### FR-2 — CANCEL_DEBUFF codec, handler, and routing

> **IDB-resolved 2026-08-04.** The open questions this section originally deferred to the
> design phase have been answered by reading eight client binaries. See
> `investigation.md` §8 for the decompilation evidence.

**FR-2.1** A serverbound `CANCEL_DEBUFF` codec MUST be added under `libs/atlas-packet`,
with both `Encode` and `Decode`, following `docs/packets/IMPLEMENTING_A_PACKET.md`.

**FR-2.2 — RESOLVED: the packet has an empty body.** `CWvsContext::CheckTemporaryStatDuration`
constructs `COutPacket(opcode)` and calls `SendPacket` with **no intervening encode calls**
on every version examined (v72, v79, v83, v84, v87, v92, v95, jms185). The client computes
its locally-expired stat mask via `SecondaryStat::CheckByTime` and then does **not** transmit
it. `CANCEL_DEBUFF` is a bare "re-evaluate my temporary stats" nudge carrying zero payload.

The codec is therefore opcode-only. It MUST NOT invent a stat mask, skill id, or any other
field. The design phase MUST NOT "improve" on this.

**FR-2.3 — Version coverage is ten clients. Opcodes below are IDB-derived, not inferred:**

| Version | Opcode | Self-throttles? | Evidence |
|---|---|---|---|
| gms_v48 | `0x4E` (78) | no | `0x71b126` |
| gms_v61 | `0x5B` (91) | no | `0x84374e` |
| gms_v72 | `0x62` (98) | no | `sub_91914F` (renamed `CWvsContext::CheckTemporaryStatDuration`) |
| gms_v79 | `0x61` (97) | no | `sub_96AD48` (renamed) |
| gms_v83 | `0x63` (99) | no | `0xa20935` — matches registry |
| gms_v84 | `0x63` (99) | no | `sub_A6BD3A` (renamed) — matches registry |
| gms_v87 | `0x66` (102) | no | `0xab7fd7` — matches registry |
| gms_v92 | `0x6E` (110) | **yes** | `sub_9C7A70` (renamed) |
| gms_v95 | `0x6F` (111) | **yes** | `0x9f2d30`, PDB-named `m_tLastStatResetRequest` |
| jms_v185 | `0x5E` (94) | **yes** | `0xb0783e` |

The five registry values (v83/84/87/95/jms185, all `provenance: csv-import`) are confirmed
correct. **v48, v61, v72, v79 and v92 are newly discovered** and MUST be added to the
registry. All ten versions are resolved; none is `n-a`.

**FR-2.3.2 — `0x63` is not stable across versions.** It means `CANCEL_DEBUFF` at v83/v84,
but at v61 the same byte is the calc-damage-stat request emitted by `OnTemporaryStatReset`.
Opcodes MUST come from tenant config per version (DOM-25); a hard-coded `0x63` would
mis-route on v61.

**FR-2.3.1 — Send-rate divergence is load-bearing.** All versions gate on
`tick - m_tLastStatResetRequest > 200`. **v92, v95 and jms185 assign the anchor before
sending; v72, v79, v83, v84 and v87 never assign it anywhere in this function.** On those
five the guard latches open 200 ms after the last temporary-stat *change* and the client
then sends once per frame, indefinitely — exactly the ~30 ms spacing and ~1,500 packets
observed live. Any rate-limit design (NFR-2) MUST assume the unthrottled case.

**FR-2.4 — RESOLVED.** All ten opcodes are established from IDBs; no version is `n-a` and
no opcode is guessed. Nothing is deferred here.

**FR-2.4.1 — The clientbound reply mask is 8 bytes at v48 and 16 bytes everywhere else.**
v48 `OnTemporaryStatReset` does `CInPacket::DecodeBuffer(a2, &v8, 8u)`; v61 and every later
version do `DecodeBuffer(…, 16)` (UINT128). This is exactly the split the existing
`legacyGmsMask(t)` gate already implements in
`libs/atlas-packet/model/character_temporary_stat.go`, so the clientbound writer needs **no
change** for either arm. Confirm during design rather than re-deriving.

**FR-2.5** A handler MUST be added to `atlas-channel` under `socket/handler/`. Because the
body is empty it performs no decode beyond the opcode; its entire job is to trigger FR-2.6.

**FR-2.6 — RESOLVED: the server reconciles, it does not cancel-by-name.** The original
design (map wire stat names → `CancelByTypes`) is **impossible** — there are no names on the
wire. The correct response is to re-evaluate the character's buffs and tell the client which
ones are actually gone:

```
CANCEL_DEBUFF (empty)
  → atlas-channel handler
    → COMMAND_TOPIC_CHARACTER_BUFF: per-character reconcile command   [NEW]
      → atlas-buffs: per-character expiry sweep                        [NEW, see FR-2.6.1]
        → existing EXPIRED event for each lapsed buff
          → existing atlas-channel buff consumer
            → existing clientbound CharacterBuffCancel writer
               (libs/atlas-packet/character/clientbound/buff_cancel.go —
                encodes cts.EncodeMask, the 16-byte UINT128 the client's
                OnTemporaryStatReset decodes via DecodeBuffer(…, 16))
```

No new clientbound packet is required: `CWvsContext::OnTemporaryStatReset` (v83 clientbound
opcode 33) is already implemented and already encodes the mask the client expects.

**FR-2.6.1** `atlas-buffs` currently exposes only a fleet-wide sweep
(`character/processor.go:190 ExpireBuffs()`, driven by `tasks/expiration.go` on a poll
interval). A **per-character** variant MUST be added so a single client's nudge does not
force a fleet-wide sweep. The new Kafka command type is the only new message contract in
this task.

**FR-2.7** The handler MUST be routed in **all ten** templates under
`services/atlas-configurations/seed-data/templates/`, with a validator (a handler entry
without a validator is silently dropped), inserted at its sorted `opCode` position per
`docs/packets/TEMPLATE_CONVENTIONS.md`.

**FR-2.8** Routing the seed templates does NOT reach already-provisioned tenants. A
live-tenant backfill procedure MUST be documented in the task folder, following the
precedent of `docs/tasks/task-153-corsair-battleship/backfill.md`.

**FR-2.9** The handler MUST be resilient: a reconcile for a character with nothing expired
is a no-op that emits nothing, and MUST NOT produce a clientbound packet (an empty-mask
`CharacterBuffCancel` would be pointless traffic on an already-hot path).

**FR-2.10 — Rate limiting is mandatory, not optional.** See NFR-2.

### FR-3 — Contract guard (anti-regression)

**FR-3.1** The `COMMAND_TOPIC_CHARACTER_BUFF` `duration` unit MUST be stated once, in a
single authoritative location, and every emitter's code MUST reference it rather than
restating it in prose. Candidate home: the shared Kafka message definition for the buff
APPLY command.

**FR-3.2** A mechanical guard MUST make a seconds-valued emitter fail CI. The design phase
selects the mechanism; at minimum a test asserting the end-to-end duration for a known
mob skill (authored WZ seconds → observed expiry) for at least one disease. A `tools/*.sh`
guard in the style of `tools/redis-key-guard.sh` is an acceptable alternative if the design
finds a reliably greppable signature.

**FR-3.3** The guard MUST cover the mist path as well as the direct-disease path, since
those are the two that diverged.

## 5. API Surface

No REST endpoints are added, modified, or removed.

**Kafka — one new command type, no new topics.** A per-character reconcile/expire command
on the existing `COMMAND_TOPIC_CHARACTER_BUFF` (FR-2.6.1). The originally-planned reuse of
the cancel-by-stat-types command is **not** viable — FR-2.2 established there are no stat
types on the wire to forward.

**Socket — one new serverbound packet**, opcode-only body, per §FR-2. No clientbound packet
is added: `CWvsContext::OnTemporaryStatReset` is already implemented as the
`CharacterBuffCancel` writer and already encodes the 16-byte mask the client decodes, and
the resulting broadcast already flows through the existing buff consumer on the
`EXPIRED`/`CANCELLED` event.

**Behavioral change to existing REST/Kafka surfaces:** `GET /mobskills/...` (atlas-data)
begins returning `duration` in milliseconds rather than seconds. This is a breaking change
to that field's meaning; FR-1.2 enumerates every known consumer. The design phase MUST
confirm no consumer outside this repo depends on it.

## 6. Data Model

No new entities, tables, columns, or migrations.

One field changes meaning without changing type:
`mobskill.RestModel.Duration` / `mobskill.Model.Duration()` (`uint32`) goes from **seconds**
to **milliseconds**.

**Re-ingest required.** WZ-derived data is ingested, not parsed per request
(see the known-issue precedent: atlas-data effects are ingested, not re-parsed). A code
change in `mobskill/reader.go` does NOT retroactively fix already-ingested rows. The
deployment procedure MUST include a mob-skill re-ingest, and the acceptance criteria MUST
verify post-ingest values, not just post-deploy ones.

**Range check.** WZ mob-skill `time` values are small (the observed skill 126 level 2 is
`15`). ×1000 keeps them far inside `uint32`. The design phase MUST confirm the maximum
authored `time` across the ingested WZ set does not overflow after conversion.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-data** | FR-1.1 ×1000 in `mobskill/reader.go:66`. Requires re-ingest (§6). |
| **atlas-monsters** | FR-1.2 remove `*1000` at `processor.go:1068`, flip `:1105` to `time.Millisecond`. FR-1.3 leave `:1242` alone. FR-1.6 tests. |
| **atlas-maps** | FR-1.2 flip `mist_tick.go:86` to `/ time.Millisecond`. FR-1.4 replace the stale comment. FR-1.6 tests. |
| **atlas-channel** | FR-2.5 new `CANCEL_DEBUFF` handler (empty-body) + FR-2.10 per-character rate limit; registration in the socket handler map. |
| **libs/atlas-packet** | FR-2.1 new opcode-only serverbound codec. No clientbound change — `buff_cancel.go` already carries the mask. |
| **atlas-configurations** | FR-2.7 handler + validator entries in every template whose opcode is established (8 confirmed; v48/v61 pending FR-2.4). |
| **atlas-buffs** | FR-2.6.1 per-character expiry sweep + its consumer arm. Re-verify the ms interpretation is unchanged. |
| **docs/packets/registry** | Add `CANCEL_DEBUFF` to `gms_v72.yaml` (`0x62`) and `gms_v79.yaml` (`0x61`); record v92 `0x6E` (§9.4). |
| **atlas-consumables, atlas-summons, atlas-messages, atlas-saga-orchestrator** | FR-1.5 audit only; changes only if the audit finds a defect. |

## 8. Non-Functional Requirements

**NFR-1 — Hot path cost.** Measured live at ~30/sec sustained per wedged client on v83.
Once diseases have correct durations that rate collapses, but the handler MUST still be
cheap: no synchronous cross-service REST call per packet. The Kafka emit path in FR-2.6
satisfies this, subject to NFR-2.

**NFR-2 — Security: request amplification, NOT buff theft.** The original concern
(a client naming a beneficial buff to strip it) is **void** — FR-2.2 establishes the packet
carries no parameters, so a client cannot name anything. Honoring it unconditionally is
provably safe: the worst a client can assert is "please re-check me," and the server
answers only with buffs that have genuinely lapsed against server-side `expiresAt`.

The real risk is **amplification**. Per FR-2.3.1, v72–v87 clients do not self-throttle and
will emit at frame rate. Unbounded, each packet becomes a Kafka command and a registry
sweep, so one wedged or hostile client can generate thousands of messages per minute.
The handler MUST rate-limit per character. The client's own 200 ms interval is the natural
floor and MUST be treated as a lower bound the server enforces independently — never as
something the server can rely on the client to honor.

**NFR-2.1** A reconcile that finds nothing expired MUST NOT emit a clientbound packet
(FR-2.9), so a steady-state nudge costs at most one suppressed sweep.

**NFR-3 — Multi-tenancy.** All new code paths resolve tenant from context
(`tenant.MustFromContext`). Wire values (opcodes, any stat-mask constants) come from tenant
configuration, never hard-coded (DOM-25).

**NFR-4 — Observability.** A dropped or rejected `CANCEL_DEBUFF` MUST log at a level that
makes a recurrence diagnosable, and MUST NOT log per-packet at `info` for the normal case —
the current `Read a unhandled message with op 0x63` at info produced ~1,500 lines in under
four minutes.

**NFR-5 — Backward compatibility.** A client on a version where `CANCEL_DEBUFF` resolves
to `n-a` MUST be unaffected. Tenants whose live socket config has not been backfilled
(FR-2.8) MUST degrade to today's behavior, not error.

## 9. Open Questions

**9.1 — Guard mechanism (FR-3.2).** Test-based, `tools/*.sh` lint-based, or both? A grep
guard is brittle (the defect is a *missing* multiplication, which has no signature); a test
is more reliable but only covers paths it exercises. Resolve in design.

**9.2 — RESOLVED (IDB, 2026-08-04).** The server-expiry gate question is moot: the packet
carries no parameters, so there is nothing to validate and nothing a client can assert.
Honor unconditionally, subject to the rate limit in NFR-2.

**9.3 — RESOLVED (IDB, 2026-08-04): `0x6C` IS part of the same handshake — and is
deliberately still out of scope.** `CWvsContext::OnTemporaryStatReset` ends with:

```c
if ( IsCalcDamageStat(mask) ) { COutPacket::COutPacket(v29, 0x6C); SendPacket(...); }
```

So `USER_CALC_DAMAGE_STAT_SET_REQUEST` is sent by the client *in reaction to* a
temporary-stat reset that touched a calc-damage stat — it is the tail of the handshake
this task implements, also empty-bodied. It explains the live `0x6C` observed right after
each mount apply/expire (the mount buff carries WEAPON_DEFENSE / MAGIC_DEFENSE).

It stays out of scope on evidence rather than assumption: unlike `CANCEL_DEBUFF`, it is
**one-shot per reset, not a loop**, so leaving it unhandled cannot wedge a client. The cost
is a possibly-stale client-side damage-range display. Implementing FR-2 will make `0x6C`
fire *more often* than today, so this should be filed as a follow-up task rather than
forgotten.

**9.4 — PARTIALLY RESOLVED.** The v92 opcode is now known (`0x6E`, FR-2.3). What remains is
purely a bookkeeping decision: create `docs/packets/registry/gms_v92.yaml`, or record the
opcode by another means. v72 and v79 append to existing registry files with no such
question.

**9.6 — Benign over-send, not a desync. Out of scope.** The clientbound `CharacterBuffCancel`
encoder writes `tSwallowBuffTime` unconditionally
(`libs/atlas-packet/character/clientbound/buff_cancel.go`), but the client reads that
trailing byte only when the mask contains a movement-affecting stat — the v83 predicate
`sub_77DC78` is named `SecondaryStat::IsMovementAffectingStat` in the v61 IDB. Since packets
are length-framed, a mask with no movement-affecting stat simply leaves the extra byte
unread. So this is an over-send worth tidying for correctness, **not** the latent desync it
first looked like. Recorded so a future reader doesn't re-raise it as a bug.

**9.7 — Follow-up scope for the `0x6C` task (§9.3).** The calc-damage-stat request opcode is
version-specific and only three are known so far: v48 `0x56` (86), v61 `0x63` (99), v83
`0x6C` (108). The remaining seven need the same per-version IDB pass. Note the v61 collision
called out in FR-2.3.2.

**9.5 — Cross-repo consumers of atlas-data `mobskill.duration`.** §5 flags the field's
meaning change. Unverified whether anything outside this repository reads it.

## 10. Acceptance Criteria

**Duration correctness**

- [ ] `mobskill/reader.go:66` converts seconds → milliseconds.
- [ ] `processor.go:1068` no longer multiplies by 1000; `processor.go:1105` uses
      `time.Millisecond`; `mist_tick.go:86` divides by `time.Millisecond`.
- [ ] `processor.go:1242` is unchanged.
- [ ] The stale seconds comment in `mist_tick.go` is replaced and explicitly notes it
      reverses `11e07dfa7`.
- [ ] All seven `COMMAND_TOPIC_CHARACTER_BUFF` producers audited, each recorded
      correct / fixed / n-a in the task folder.
- [ ] Tests pinning the seconds contract updated (not deleted) with corrected comments.
- [ ] A mob-cast mist created after the change lasts its authored duration and is **not**
      clamped to exactly 60 s.
- [ ] A monster self-buff / immunity lasts its authored duration, not ~1000× longer.

**CANCEL_DEBUFF**

- [ ] Codec exists in `libs/atlas-packet`, **opcode-only body** — no invented fields.
- [ ] Handler registered in atlas-channel; triggers a per-character reconcile, does not
      attempt cancel-by-name.
- [ ] Per-character rate limit enforced server-side, independent of the client's 200 ms
      interval (NFR-2). Demonstrated against an unthrottled v72–v87-style send rate.
- [ ] A reconcile finding nothing expired emits no clientbound packet (FR-2.9).
- [ ] `atlas-buffs` per-character expiry sweep added; fleet-wide sweep unchanged.
- [ ] Routed with a validator, at the sorted `opCode` position, in all ten templates;
      `tools/template-opcode-order-guard.sh` clean.
- [ ] Opcodes resolved from tenant config per version, never hard-coded — a `0x63` literal
      would mis-route on v61 (FR-2.3.2).
- [ ] `CANCEL_DEBUFF` added to the v48 (`0x4E`), v61 (`0x5B`), v72 (`0x62`) and v79 (`0x61`)
      registry files; v92 (`0x6E`) recorded per the §9.4 decision.
- [ ] Live-tenant backfill procedure documented in the task folder.

**Guard**

- [ ] The duration unit contract is stated in exactly one authoritative place.
- [ ] A deliberately-reintroduced seconds emitter fails CI — demonstrated, not asserted.
- [ ] The guard covers both the direct-disease and mist paths.

**End-to-end (live tenant, GMS 83.1)**

- [ ] A mob Slow lasts its authored WZ duration on the client and ends on its own.
- [ ] Zero `Read a unhandled message with op 0x63` lines during a sustained debuff fight.
- [ ] A client that locally expires a stat the server still holds recovers without relog.
- [ ] The originally-reported wedge does not reproduce: skills remain usable and the mount
      buff can be cancelled throughout a debuff fight.

**Build & verification** (per CLAUDE.md)

- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module.
- [ ] `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`,
      `tools/template-opcode-order-guard.sh`, `tools/skill-job-id-guard.sh` all clean.
- [ ] Code review run before the PR is opened.
