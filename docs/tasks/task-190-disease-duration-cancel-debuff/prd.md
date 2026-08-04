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

**FR-2.1** A serverbound `CANCEL_DEBUFF` codec MUST be added under `libs/atlas-packet`,
with both `Encode` and `Decode`, following `docs/packets/IMPLEMENTING_A_PACKET.md`.

**FR-2.2** The wire body MUST be derived from the client binary
(`CWvsContext::CheckTemporaryStatDuration`), not inferred from the opcode table or from
sibling packets. The field layout is **unknown at spec time** and is an explicit
design-phase deliverable. Do not guess a mask width or field order.

**FR-2.3** Version coverage is **ten** client versions. Opcodes already in the registry:

| Version | Opcode | Source |
|---|---|---|
| gms_v83 | `0x63` (99) | `docs/packets/registry/gms_v83.yaml:2326` |
| gms_v84 | `0x63` (99) | registry |
| gms_v87 | `0x66` (102) | registry |
| gms_v95 | `0x6F` (111) | registry |
| jms_v185 | `0x5E` (94) | registry |

Opcodes NOT yet known and requiring IDB discovery: **gms_v48, gms_v61, gms_v72, gms_v79**
(matrix shows `⬜` — never discovered) and **gms_v92** (no `docs/packets/registry/gms_v92.yaml`
exists at all; per project convention v92 is a template/wire target rather than a coverage-matrix
column, so it needs the opcode but not a matrix cell).

**FR-2.4** If a version genuinely lacks the opcode after IDB discovery, it MUST be recorded
as `n-a` with evidence per the matrix's n-a consistency gate — never silently skipped and
never given a guessed opcode.

**FR-2.5** A handler MUST be added to `atlas-channel` under
`socket/handler/`, modelled on the existing `character_buff_cancel.go`. It decodes the
request, maps the wire stat representation to `TemporaryStatType` names, and dispatches.

**FR-2.6** The handler MUST honor the request by cancelling the named stat(s), via the
existing end-to-end path — no new plumbing is required:

```
atlas-channel buff.Processor.CancelByTypes(f, characterId, types)   // processor.go:73
  → COMMAND_TOPIC_CHARACTER_BUFF
    → atlas-buffs kafka/consumer/character/consumer.go:90
      → character.Processor.CancelByStatTypes                        // processor.go:131
        → Registry.CancelByStatTypes                                 // registry.go:233
```

**FR-2.7** The handler MUST be routed in **all ten** templates under
`services/atlas-configurations/seed-data/templates/` at each version's opcode, with a
validator (a handler entry without a validator is silently dropped), inserted at its
sorted `opCode` position per `docs/packets/TEMPLATE_CONVENTIONS.md`.

**FR-2.8** Routing the seed templates does NOT reach already-provisioned tenants. A
live-tenant backfill procedure MUST be documented in the task folder, following the
precedent of `docs/tasks/task-153-corsair-battleship/backfill.md`.

**FR-2.9** The handler MUST be resilient to a hostile or buggy client: a cancel naming a
stat the character does not have is a no-op, not an error; an unparseable body is logged
and dropped without terminating the session; and the handler MUST NOT be a vector for
cancelling *beneficial* buffs. See NFR-2.

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

**Kafka — no new topics or message types.** `CANCEL_DEBUFF` reuses the existing
`COMMAND_TOPIC_CHARACTER_BUFF` cancel-by-stat-types command already consumed at
`services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go:90`.

**Socket — one new serverbound packet** per §FR-2, opcodes per FR-2.3. No clientbound
packet is added: the resulting `CANCEL_TEMPORARY_STAT` broadcast already flows through the
existing buff consumer on the `EXPIRED`/`CANCELLED` event.

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
| **atlas-channel** | FR-2.5 new `CANCEL_DEBUFF` handler; registration in the socket handler map. |
| **libs/atlas-packet** | FR-2.1/2.2 new serverbound codec with version gates. |
| **atlas-configurations** | FR-2.7 handler + validator entries in all ten templates. |
| **atlas-buffs** | No code change expected — `CancelByStatTypes` already exists. Re-verify the ms interpretation is unchanged. |
| **atlas-consumables, atlas-summons, atlas-messages, atlas-saga-orchestrator** | FR-1.5 audit only; changes only if the audit finds a defect. |

## 8. Non-Functional Requirements

**NFR-1 — Hot path cost.** `CANCEL_DEBUFF` arrives at up to ~30/sec per wedged client
today. Once diseases have correct durations that rate collapses, but the handler MUST
still be cheap: no synchronous cross-service REST call per packet. The Kafka emit path in
FR-2.6 satisfies this.

**NFR-2 — Security: no beneficial-buff cancellation.** `CANCEL_DEBUFF` is client-initiated
and therefore untrusted. The handler MUST restrict cancellation to disease/debuff
temporary stats — the design phase defines the allow-list, anchored on the `Disease()`
predicate already present in `libs/atlas-packet/model/character_temporary_stat.go`
(`newAndIncDiseased`, lines ≈97-250). A client MUST NOT be able to use this opcode to
strip a debuff the server still considers active *if* the server's remaining duration is
materially in the future — the design phase decides whether to gate on server expiry or
honor unconditionally, and MUST record the reasoning. Honoring unconditionally is the
default per the scoping decision; the gate is the fallback if the design finds an abuse
vector.

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

**9.2 — Server-expiry gate (NFR-2).** Honor `CANCEL_DEBUFF` unconditionally (scoping
decision, chosen for self-healing) versus validate against server-side remaining duration.
Needs the IDB read from FR-2.2 to know whether the client sends enough information to
correlate. Resolve in design.

**9.3 — `USER_CALC_DAMAGE_STAT_SET_REQUEST` (v83 `0x6C`,
`CWvsContext::OnTemporaryStatReset`)** is also unhandled and was observed in the same live
capture, firing immediately after each mount apply/expire. Explicitly out of scope here, but
it is adjacent enough that the design phase should confirm it is not part of the same
recovery handshake before this task closes.

**9.4 — v92 matrix representation.** v92 has a seed template but no registry file and no
coverage-matrix column. Confirm during design whether this task should create
`docs/packets/registry/gms_v92.yaml`, or record the opcode by another means.

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

- [ ] Codec exists in `libs/atlas-packet` with `Encode` and `Decode`, layout derived from
      the IDB with the evidence recorded.
- [ ] Handler registered in atlas-channel; cancels only disease-class stats (NFR-2).
- [ ] Routed with a validator, at the sorted `opCode` position, in all ten templates;
      `tools/template-opcode-order-guard.sh` clean.
- [ ] Every version resolved: opcode wired, or `n-a` with evidence. No guessed opcodes.
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
