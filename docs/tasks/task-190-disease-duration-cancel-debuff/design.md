# Disease Duration Units & CANCEL_DEBUFF — Design

Version: v1
Status: Draft
Created: 2026-08-04
Inputs: [`prd.md`](prd.md) (approved), [`investigation.md`](investigation.md)

---

## 1. Shape of the work

The PRD carries three requirement families. They are **independently
shippable but land together**, and their risk profiles are very different:

| Family | Risk | Value | Nature |
|---|---|---|---|
| FR-1 duration units | low (arithmetic, 4 sites) | high — fixes the live bug | correction |
| FR-2 `CANCEL_DEBUFF` | medium (new codec + new Kafka command + 10 templates) | closes the *class* | new capability |
| FR-3 contract guard | low | prevents flip #4 | infrastructure |

FR-1 alone stops the reported wedge. FR-2 alone does not fix any duration.
The order that minimises risk is **FR-1 → FR-3 → FR-2**: land the arithmetic
correction with its guard first, so the guard is proven against the very
defect it exists to catch, then build the recovery path on a tree that is
already unit-correct.

---

## 2. Architecture at a glance

### 2.1 FR-1 — the unit correction

```
Skill.wz MobSkill.img  time=15 (seconds)
   │
   ▼  atlas-data mobskill/reader.go:66   ← ×1000 HERE (only conversion point)
mobskill.RestModel.Duration = 15000 (ms)
   │
   ├─► atlas-monsters executeDebuff        forwards verbatim         [no edit]
   ├─► atlas-monsters buildMistCreateBody  drop the ×1000            [edit]
   ├─► atlas-monsters executeStatBuff      Second → Millisecond      [edit]
   └─► atlas-ui MonsterSkillChip           render ms as seconds      [edit — new, §8.1]

atlas-maps mist_tick.go   DiseaseDuration() → .Milliseconds()        [edit]
```

The design's single organising rule: **there is exactly one seconds→ms
conversion in the whole system, and it lives in `mobskill/reader.go`.** Every
other site forwards. That mirrors what `skill/reader.go:194-198` already does
for skill effects and is the reason the skill path never drifted the way the
mob-skill path did.

### 2.2 FR-2 — the recovery handshake

```
client: SecondaryStat::CheckByTime finds a locally-expired stat
   │  serverbound CANCEL_DEBUFF (empty body, opcode from tenant config)
   ▼
atlas-channel  CancelDebuffHandleFunc
   │  ├─ statreset registry: per-(tenant,character) throttle  ── drop ──► Debug log, return
   │  └─ pass
   ▼  COMMAND_TOPIC_CHARACTER_BUFF  type=EXPIRE  body={}      [NEW command type]
atlas-buffs  handleExpire → Processor.ExpireForCharacter      [NEW]
   │  Registry.GetExpired(ctx, characterId)   ← already exists, already prunes
   │  nothing expired → emit nothing                          (FR-2.9 / NFR-2.1)
   ▼  EVENT_TOPIC_CHARACTER_BUFF_STATUS  type=EXPIRED  (one per lapsed buff)
atlas-channel  handleStatusEventExpired                       [unchanged]
   ▼  clientbound CharacterBuffCancel (cts.EncodeMask, 16-byte UINT128)
client: CWvsContext::OnTemporaryStatReset clears the stat, stops nudging
```

Everything downstream of the new Kafka command already exists and is
exercised in production today by `CANCEL` / `CANCEL_ALL` / `CANCEL_BY_TYPES`.
The genuinely new code is: one codec, one handler, one throttle registry, one
command type, one processor method.

---

## 3. Design decisions

### D1 — Reconcile via a new per-character Kafka command (not sync REST)

**Chosen.** atlas-channel emits `EXPIRE` on the existing
`COMMAND_TOPIC_CHARACTER_BUFF`; atlas-buffs owns the decision about what has
lapsed.

*Alternatives considered:*

- **Synchronous REST** — the handler could `GET /characters/{id}/buffs`,
  compute expiry channel-side, and emit `CANCEL` per lapsed buff. Rejected:
  violates NFR-1 (a cross-service HTTP call on a ≤30/s hot path), and it is
  racy — the channel would be re-deriving state atlas-buffs already owns, and
  a buff that lapses between the GET and the CANCEL would be double-cancelled.
- **Do nothing; rely on the existing 10 s fleet sweep**
  (`tasks/expiration.go`, `NewExpiration(l, 10000)`). Rejected: a client that
  disagrees with the server still waits up to 10 s, and the disagreement
  class the PRD wants closed (server holds a buff the client already dropped)
  is only *sometimes* an expiry — but where it is not, nothing we can send
  would help either, so the sweep is the right primitive, just not at the
  right latency.

*Why the existing sweep is the right primitive:* `Registry.GetExpired`
(registry.go:186) already does prune-and-return for one character. The new
processor method is the existing `ExpireBuffs()` loop body with the
`GetCharacters` range removed. No new expiry semantics are invented.

**Command type name: `EXPIRE`,** body `ExpireCommandBody struct{}`. Considered
`RECONCILE`; rejected because "reconcile" implies a two-way diff, and the
server does not diff — it prunes what has genuinely lapsed against
server-side `expiresAt` and announces it. `EXPIRE` says what happens.

**WorldId** rides in the existing `Command[E]` envelope (the channel knows the
session's world). The fleet sweep reads world from the registry model; the
per-character path uses the command's, which is authoritative for a live
session.

### D2 — Rate limit: in-process, per (tenant, character), in atlas-channel

**Chosen.** A new `atlas-channel/character/statreset` package: a
`sync.Once` + `sync.RWMutex` registry of `map[Key]time.Time`, exactly the
shape of `atlas-channel/shopscanner/registry.go`.

```go
func (r *Registry) Allow(t tenant.Model, characterId uint32, now time.Time) bool
func (r *Registry) ClearCharacter(t tenant.Model, characterId uint32)
```

`Allow` returns true and records `now` when there is no entry or the entry is
older than the window; false otherwise. First nudge after a quiet period
always passes, so recovery latency is one packet, not one window.

*Window: 1000 ms.* Reasoning:

- The client's own floor is 200 ms (`tick - m_tLastStatResetRequest > 200`),
  and per FR-2.3.1 five of ten versions never advance that anchor, so the
  client floor must be treated as **advisory only**. The server's window must
  be an independent bound, not a mirror.
- 1 s caps a wedged or hostile client at 1 Kafka command/sec — a ~30×
  reduction against the measured live rate — while still recovering 10×
  faster than the fleet sweep.
- It is a `const`, not env-configurable. Adding a tunable adds deployment
  surface for a value with no known reason to vary per tenant. Deliberate
  non-goal; revisit if a real workload disagrees.

*Correctness of in-process state:* a character's socket session lives on
exactly one atlas-channel pod, so a per-pod map is not a partial view — it is
the whole view. On reconnect to a different pod the entry is absent and the
first nudge passes, which is the desired behaviour anyway.

*Eviction:* `ClearCharacter` is called from the socket destroyer in
`socket/init.go:49-55`, alongside the existing
`shopscanner.GetRegistry().ClearCharacter(t, s.CharacterId())`. Without this
the map leaks one entry per character ever seen by the pod.

*Alternatives considered:*

- **Redis-backed throttle via `libs/atlas-redis`.** Rejected: a network
  round-trip per packet on the path whose whole purpose is to be cheap
  (NFR-1). Cross-pod coordination buys nothing — see above.
- **Throttle in atlas-buffs (dedupe the command).** Rejected: the Kafka
  message has already been produced by then, so the amplification NFR-2 is
  about has already happened. The throttle must sit before the emit.
- **Rely on the client's 200 ms.** Explicitly rejected by NFR-2 and by the
  IDB evidence: five versions latch the guard permanently open.

### D3 — Codec: opcode-only, version-invariant, no gates

`libs/atlas-packet/character/serverbound/cancel_debuff.go`:

```go
const CancelDebuffHandle = "CancelDebuffHandle"

// CancelDebuff - CWvsContext::CheckTemporaryStatDuration
type CancelDebuff struct{}
```

`Encode` returns `[]byte{}`; `Decode` reads nothing. This is the exact shape
of the existing `ChalkboardClose` (serverbound/chalkboard_close.go), which is
the precedent for an empty-body serverbound codec including its byte-fixture
test style.

Placement in `character/serverbound` matches its sibling `BuffCancelRequest`
(CANCEL_BUFF) even though the client function lives on `CWvsContext` — the
packet is about the character's temporary stats.

**No version gates.** The body is empty on all ten clients (investigation.md
§8.1), so there is nothing to gate. The v48 three-argument
`COutPacket::COutPacket(v5, 78, 0)` constructor is a client-side construction
detail with no wire consequence. The `legacyGmsMask(t)` split (8-byte at v48,
16-byte at v61+) belongs to the *clientbound* reply and is already
implemented in `libs/atlas-packet/model/character_temporary_stat.go` — FR-2.4.1
is confirmed, and this task changes no clientbound encoder.

**FR-2.2 is honoured literally: no invented fields.** The client computes the
expired mask and discards it. The codec must not grow a stat mask, a skill id,
or a tick.

### D4 — Handler: thin, throttle-first

`services/atlas-channel/atlas.com/channel/socket/handler/character_cancel_debuff.go`

```
decode (no-op) → statreset.Allow? → no: Debug, return
                                  → yes: buff.NewProcessor(l,ctx).Expire(s.Field(), s.CharacterId())
```

`Expire` is a new method on the existing `atlas-channel/character/buff`
`Processor` interface, one line beside `CancelByTypes`, producing the new
command. Registration: `handlerMap[charsb.CancelDebuffHandle] =
handler.CancelDebuffHandleFunc` in `main.go`, in the block at ~:880.

Opcodes are never referenced in Go — the handler is bound by *name* through
the tenant socket config, so FR-2.3.2's v61 `0x63` collision cannot arise
(DOM-25). This is a property of the existing dispatch mechanism, not something
this task has to add.

### D5 — Contract statement (FR-3.1): one authority, pointers elsewhere

The seven producers each keep their **own copy** of the message contract
(`kafka/message/.../kafka.go`) — that is the established convention and this
task does not change it. So "stated once" resolves to:

- **The authority** is
  `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`,
  on `ApplyCommandBody.Duration`. atlas-buffs is the *consumer* that defines
  the unit (`buff.NewBuff` → `expiresAt = now + duration*time.Millisecond`),
  so the unit is its property to declare.
- **Every producer copy** carries a one-line pointer comment, not a restatement:
  `// milliseconds — contract owner: atlas-buffs kafka/message/character/kafka.go (task-190)`.
- The stale comment at `mist_tick.go:81-86` is replaced per FR-1.4, and its
  replacement explicitly records that this change reverses `11e07dfa7` so the
  contract is not flipped a fourth time.

Prose alone is what failed three times. It is necessary but not sufficient —
hence D6.

### D6 — Contract guard (FR-3.2, resolves Open Question 9.1): analyzer **and** tests

**Both, with distinct jobs.** They catch different halves of the defect:

| Failure mode | Example | Caught by |
|---|---|---|
| *Double* conversion (a scaling factor added) | `Duration: int32(d / time.Second)` | analyzer |
| *Missing* conversion (nothing added) | `Duration: int32(sd.Duration())` when the source is seconds | tests |

A grep-style guard cannot see a missing multiplication — it has no signature.
A test cannot cover a path nobody wrote yet. Neither alone satisfies "a
deliberately-reintroduced seconds emitter fails CI — demonstrated."

**The analyzer.** `tools/buffdurationguard` + `tools/buff-duration-guard.sh`,
built exactly like `tools/rediskeyguard` / `tools/goroutineguard` (build once,
run over every module under `services/` and `libs/`, non-empty diagnostics →
non-zero exit; a self-test on fixtures like goroutineguard's).

It anchors on **structural fingerprints, not type names**, because the body
struct is duplicated under seven different local names:

- **BD-1 (buff APPLY body)** — a composite literal whose struct has fields
  with json tags `sourceId`, `duration` and `changes`. The expression assigned
  to the `duration` field must not contain `time.Second`, `time.Minute`,
  `time.Hour`, or the integer literal `1000`.
- **BD-2 (mist create body)** — a composite literal whose struct has json tags
  `diseaseDuration` and `tickIntervalMs`. Same ban on the expressions assigned
  to `diseaseDuration` and `duration`.

Escape hatch: `//buffdurationguard:allow <reason>` on the offending line,
mirroring `//goroutine-guard:allow`. A justified site stays legal and visible.

Verified against the post-fix tree: `mist_tick.go` becomes
`int32(m.DiseaseDuration().Milliseconds())` (clean), `buildMistCreateBody`
becomes `durMs := int64(sd.Duration())` (clean), and
`MistDurationCapMs int64 = 60_000` is a const declaration, not a flagged
literal. Verified against the *pre-fix* tree: BD-1 flags
`mist_tick.go:86`, BD-2 flags `processor.go:1068`. The guard demonstrably
catches the historical defect — that is the demonstration the acceptance
criterion asks for.

**The tests.** Unit tests on pure functions at the end of each path:

- `atlas-data mobskill.readLevel` — `time=15` ⇒ `Duration == 15000`.
- `atlas-monsters buildMistCreateBody` — already pure; assert ms passthrough
  and that the 60 000 clamp now bites only above 60 s.
- `atlas-monsters` disease path — assert `executeDebuff` forwards ms verbatim
  (extract the duration computation into a pure helper if the emit is not
  otherwise testable).
- `atlas-maps applyDiseaseCommandProvider` — assert ms passthrough.

This covers both requirement halves: FR-3.3's "direct-disease *and* mist" is
satisfied by BD-1+BD-2 in the analyzer and by the four tests.

*Alternative considered — a typed unit (`type DurationMs int32`).* Rejected:
Go named types do not prevent unit errors, because `DurationMs(15)` compiles
just as happily as `DurationMs(15000)`. It would add churn across seven
services and enforce nothing. Only the analyzer actually fails a build.

### D7 — v92 registry bookkeeping (resolves Open Question 9.4)

**Do not create `docs/packets/registry/gms_v92.yaml`.**

Registry files are matrix-column artifacts: `docs/packets/PROCESS.md` states
the matrix tracks **9** versions, and each has a registry *and* an IDA export
*and* an audit dir. v92 has a seed template but is not a column. A lone,
half-populated v92 registry would look like a column to a reader and creates a
gratuitous question for `matrix --check` / `doc-freshness --check`.

Record v92's `0x6E` where it is actually consumed:

1. `template_gms_92_1.json` — the handler entry itself (slot verified free, §5).
2. `investigation.md` §8.2 — already records it with the IDB evidence.
3. `backfill.md` — the live-tenant v92 row.

If v92 is ever brought up as a matrix column, `/bringup-version` builds its
registry wholesale; this task should not pre-seed a fragment of it.

### D8 — gms_12 is not routed, and that is not a gap

The PRD scopes "ten templates". There are eleven. `template_gms_12_1.json`
carries **24 handlers** — login, character select, map/channel change, move,
chat, inventory move, info request, monster movement, NPC action, summon.
It routes no buff handler, no skill handler, and no attack handler at all.
`CANCEL_DEBUFF` is outside that template's entire feature surface, so it is
deliberately excluded on evidence, not deferred. No v12 IDB read is needed to
justify this: even a known opcode would have nothing to reconcile.

### D9 — matrix truth: this task corrects four false `n-a` cells

`docs/packets/audits/status.json` currently records `CANCEL_DEBUFF` as
`state: "n-a", opcode: -1` for gms_v48, v61, v72, v79 — not merely
"unverified". The IDB pass (investigation.md §8.2) proves the packet exists on
all four. Adding the registry entries is therefore not bookkeeping; it is
**correcting a wrong matrix assertion**, and the n-a consistency gate will
only let the state change once the registry says the opcode exists.

Registry insertions are conflict-free — verified by scanning each file for an
existing serverbound entry at the target opcode:

| Registry | opcode | existing serverbound entry |
|---|---|---|
| `gms_v48.yaml` | 78 (`0x4E`) | none |
| `gms_v61.yaml` | 91 (`0x5B`) | none |
| `gms_v72.yaml` | 98 (`0x62`) | none |
| `gms_v79.yaml` | 97 (`0x61`) | none |

Each new entry carries `fname: CWvsContext::CheckTemporaryStatDuration`
(satisfying `fname-doc --check`) and a provenance recording the IDB address
from investigation.md §8.2 — these are IDB-derived, not `csv-import`.

The nine matrix cells (four n-a→verified, five incomplete→verified) each go
through the standard single-cell procedure: `/verify-packet` +
`packet-verifier` per `docs/packets/audits/VERIFYING_A_PACKET.md`. v92 gets a
round-trip test but no matrix marker — it is not a column.

---

## 4. Component inventory

### atlas-data
- `mobskill/reader.go:66` — `m.Duration = uint32(node.GetIntegerWithDefault("time", 0)) * 1000`, with a comment mirroring `skill/reader.go:194-198`.
- `mobskill/` — new `readLevel` unit test.

### atlas-monsters
- `monster/processor.go:1068` (`buildMistCreateBody`) — drop `* int64(time.Second/time.Millisecond)`.
- `monster/processor.go:1105` (`executeStatBuff`) — `time.Second` → `time.Millisecond`.
- `monster/processor.go:1242` (`executeDebuff`) — **no edit** (FR-1.3).
- `monster/processor_test.go` — update the tests pinning the seconds contract (≈`:894`, `:1179`, `:1236`), located by content, with corrected comments.

### atlas-maps
- `tasks/mist_tick.go:81-86` — replace the stale seconds comment (FR-1.4, naming `11e07dfa7`) and change the value to `int32(m.DiseaseDuration().Milliseconds())`.
- `tasks/mist_tick_test.go` (≈`:163`) — update, not delete.

### atlas-ui — **new, see §8.1**
- `src/components/features/monsters/MonsterSkillChip.tsx:114` — render the now-ms `duration` as seconds instead of `${a.duration}s`.

### libs/atlas-packet
- `character/serverbound/cancel_debuff.go` — new opcode-only codec.
- `character/serverbound/cancel_debuff_test.go` — round-trip across `pt.Variants` (which already includes v48/61/72/79/83/84/87/92/95/jms185) plus per-version empty-body byte fixtures carrying `packet-audit:verify` markers for the nine matrix versions.

### atlas-channel
- `character/statreset/registry.go` (+ test) — the throttle.
- `character/buff/processor.go` — `Expire(f field.Model, characterId uint32) error` on the `Processor` interface and impl.
- `character/buff/producer.go` — `ExpireCommandProvider`.
- `kafka/message/buff/kafka.go` — `CommandTypeExpire`, `ExpireCommandBody`.
- `socket/handler/character_cancel_debuff.go` (+ test) — the handler.
- `socket/init.go:49-55` — `statreset.GetRegistry().ClearCharacter(...)` in the destroyer.
- `main.go` (~:880) — handler map registration.

### atlas-buffs
- `kafka/message/character/kafka.go` — `CommandTypeExpire`, `ExpireCommandBody`, and the FR-3.1 authoritative unit comment on `ApplyCommandBody.Duration`.
- `kafka/consumer/character/consumer.go` — `handleExpire` + its `rf(...)` registration.
- `character/processor.go` — `ExpireForCharacter(worldId, characterId) error`; refactor the shared body out of `ExpireBuffs()` so both call one helper. Fleet sweep behaviour unchanged.

### atlas-configurations
- All ten templates gain a `CancelDebuffHandle` entry with
  `"validator": "LoggedInValidator"` and `"services": ["channel"]`, at the
  sorted `opCode` position (`tools/template-opcode-order-guard.sh`).

### docs/packets
- `registry/gms_v48.yaml`, `gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml` — new `CANCEL_DEBUFF` entries.
- `audits/status.json` + `STATUS.md` — regenerated after the nine cells verify.

### tools
- `tools/buffdurationguard/` (analyzer + fixtures) and `tools/buff-duration-guard.sh`.
- CLAUDE.md Build & Verification list gains item 11.
- The packet-matrix / pr-validation workflow gains the guard job alongside the existing guards.

### task folder
- `producer-audit.md` (FR-1.5), `backfill.md` (FR-2.8).

---

## 5. Template opcode slots — all ten verified free

Scanned every template's `handlers` array for an existing entry at the target
opcode. All ten are unoccupied, so each is a clean sorted insertion:

| Template | opCode | occupied? | nearest neighbours |
|---|---|---|---|
| `template_gms_48_1.json` | `0x4E` | free | `0x4C`, `0x51` |
| `template_gms_61_1.json` | `0x5B` | free | `0x59`, `0x5C` |
| `template_gms_72_1.json` | `0x62` | free | `0x61`, `0x63` |
| `template_gms_79_1.json` | `0x61` | free | `0x60`, `0x62` |
| `template_gms_83_1.json` | `0x63` | free | `0x62`, `0x64` |
| `template_gms_84_1.json` | `0x63` | free | `0x62`, `0x64` |
| `template_gms_87_1.json` | `0x66` | free | `0x65`, `0x67` |
| `template_gms_92_1.json` | `0x6E` | free | `0x6C`, `0x6F` |
| `template_gms_95_1.json` | `0x6F` | free | `0x6E`, `0x72` |
| `template_jms_185_1.json` | `0x5E` | free | `0x5D`, `0x5F` |

---

## 6. FR-1.5 producer audit — preliminary findings

All seven `COMMAND_TOPIC_CHARACTER_BUFF` producers were traced during design.
Execution must re-confirm each with file:line and record the result in
`producer-audit.md`; these are the leads, not the record.

| Service | Finding | Evidence |
|---|---|---|
| **atlas-monsters** | **broken** — two sites double-convert after FR-1.1, one is fixed by it | `monster/processor.go:1068`, `:1105`, `:1242` |
| **atlas-maps** | **broken** — divides ms back to seconds | `tasks/mist_tick.go:86` |
| atlas-channel | correct | every `buff.Apply` duration is `effect.Model.Duration()` (ms since task-054) — `skill/handler/common.go:162`, `mysticdoor.go:127`. `HideBuffDuration` / `MountBuffDuration` are `math.MaxInt32` sentinels, unit-agnostic. |
| atlas-consumables | correct | `consumable/processor.go:150`, `:214` already document and use ms (task-140, `88d270bf1`) |
| atlas-summons | correct | `data/skill/effect/model.go:41-43` — "duration in milliseconds"; forwarded unchanged to `buff/producer.go:83` |
| atlas-messages | correct | `buff/processor.go:42` uses `effect.Duration()` (ms) |
| atlas-saga-orchestrator | **n-a** | produces only `CANCEL_ALL` (`buff/processor.go:49-51`); no `APPLY`, no duration field |

---

## 7. Error handling & failure modes

| Failure | Behaviour |
|---|---|
| Nudge arrives for a character with nothing expired | `GetExpired` returns empty ⇒ nothing put on the buffer ⇒ `message.Emit` iterates an empty map and emits nothing. FR-2.9 / NFR-2.1 hold **structurally**, not by an explicit guard. |
| Nudge within the throttle window | Dropped, Debug log. Not an error. |
| Tenant socket config not backfilled (NFR-5) | The opcode has no handler ⇒ `libs/atlas-socket/server.go:282` logs the unhandled op at Info, exactly as today. Degrades to current behaviour; does not error. |
| atlas-buffs unreachable / Kafka down | The producer call returns an error; the handler logs and returns. The 10 s fleet sweep remains the backstop. No session impact. |
| Registry Redis read fails inside `GetExpired` | Returns an empty slice (existing behaviour, registry.go:189-192) ⇒ nudge is a no-op, client retries next window. |
| Client on a version with no routed handler | Unchanged from today (NFR-5). |

**Residual, out of scope:** `AdaptHandler` opens an OTel span *before* the
handler runs, so a nudge flood still costs one span per packet even when
throttled. That cost exists today for every unhandled packet and is a
property of the shared dispatch layer, not something this task introduces.

**Residual, out of scope:** implementing FR-2 makes `0x6C`
(`USER_CALC_DAMAGE_STAT_SET_REQUEST`) fire *more* often, since it is the tail
of this handshake (PRD §9.3). It is one-shot per reset and cannot wedge a
client. It should be filed as a follow-up task at PR time, not silently
dropped.

---

## 8. Findings that amend the PRD

### 8.1 atlas-ui is an unlisted consumer of `mobskill.duration` (resolves Open Question 9.5, partially)

`services/atlas-ui/src/components/features/monsters/MonsterSkillChip.tsx:114`:

```ts
if (a.duration > 0) rows.push({ label: "Duration", value: `${a.duration}s` });
```

After FR-1.1 this renders **"15000s"** for a 15-second skill. The PRD's §7
Service Impact table does not list atlas-ui. It must: the fix is to divide by
1000 for display (or format ms→s), in the same change, or the data browser
ships visibly wrong.

This is the only in-repo consumer of the field outside the three the PRD
enumerates (`mob-skills.service.ts` types it, `useMobSkillData.ts` only reads
names). Consumers *outside* this repository remain unverified — the endpoint
is an internal service API with no published contract, and the design proceeds
on the assumption that the in-repo set is complete.

Consequence: atlas-ui verification (`npm run build` — not just vitest) joins
the acceptance checklist.

### 8.2 The mist cap changes meaning, and the acceptance test must account for it

`MistDurationCapMs = 60_000` currently clamps a 1000×-inflated value, so
**every** mob mist is pinned to exactly 60 s. After FR-1.2 the clamp applies
to real milliseconds — correct behaviour, but it means a mob mist whose
authored `time` exceeds 60 s will *still* be 60 s. The acceptance criterion
"lasts its authored duration and is not clamped to exactly 60 s" is only
testable against a mist skill with authored `time ≤ 60`. Execution must pick
such a skill and say which one.

### 8.3 Four matrix cells are `n-a`, not blank

See D9. Materially different from the `⬜` shown in the rendered STATUS.md:
the matrix currently *asserts* the packet does not exist on v48/61/72/79.

### 8.4 Overflow bounds (PRD §6 range check)

`RestModel.Duration` is `uint32`; `executeDebuff` narrows to `int32`. The
binding constraint is therefore int32: overflow needs an authored
`time > 2_147_483` seconds ≈ **24.8 days**. No mob skill approaches this.
Execution confirms with a max query over the ingested mob-skill rows before
the change and records the number rather than asserting safety.

---

## 9. Testing strategy

**Unit — FR-1 (the four pure-function tests in D6).** These are the durable
record of the contract.

**Unit — FR-2:**
- Codec: round-trip across `pt.Variants`, plus per-version empty-body byte
  fixtures with `packet-audit:verify` markers (nine matrix versions).
- Throttle: table-driven over `Allow` — first call passes, second inside the
  window drops, one after the window passes, distinct characters and distinct
  tenants do not share state, `ClearCharacter` resets.
- Handler: with a stubbed buff processor — a throttled nudge produces no
  command; an allowed one produces exactly one.
- `ExpireForCharacter`: nothing expired ⇒ zero messages; N expired ⇒ N
  `EXPIRED` events; the fleet sweep still behaves identically after the
  refactor (its existing tests must pass unchanged).

**Analyzer:** self-test fixtures containing (a) the historical defective forms
from `mist_tick.go:86` and `processor.go:1068`, which must be diagnosed, and
(b) the corrected forms plus an annotated allow-site, which must not be.

**Guards & build (per CLAUDE.md):** `go test -race`, `go vet`, `go build` in
every changed module; `docker buildx bake` for every service whose `go.mod`
was touched; `redis-key-guard`, `goroutine-guard`, `lint.sh --check`,
`template-opcode-order-guard`, `skill-job-id-guard`, and the new
`buff-duration-guard`; `packet-audit matrix --check` / `fname-doc --check`.
atlas-ui: `npm run build`.

**End-to-end (live tenant, GMS 83.1)** — the PRD §10 list. The two that
matter most: zero `Read a unhandled message with op 0x63` during a sustained
debuff fight, and a mob Slow that ends on its own.

---

## 10. Rollout

Ordering matters because WZ data is **ingested, not parsed per request**.

1. Deploy the code (atlas-data, atlas-monsters, atlas-maps, atlas-channel,
   atlas-buffs, atlas-configurations, atlas-ui).
2. **Re-ingest `Skill.wz`** for each tenant. `document.Storage.Add` writes
   through to the DB with an `OnConflict` clause (db_storage.go:144) and to
   the in-memory registry, so re-ingest is an upsert, not a duplicate error.
3. **Roll atlas-data.** `Storage.ByIdProvider` serves from the per-pod
   in-memory `document.Registry` first; replicas that did not perform the
   ingest keep stale mob-skill rows until restarted. Skipping this step is the
   most likely way for the fix to appear not to work.
4. Verify the data, not the deploy: `GET /data/mob-skills/126` must show
   `duration: 15000` for level 2. atlas-monsters holds no cache
   (`monster/mobskill/processor.go` fetches per use), so it picks up the new
   value immediately.
5. **Backfill live tenant socket configs** per `backfill.md`. Seed templates
   apply at tenant creation only; existing tenants never see the new handler
   otherwise, and atlas-channel does not hot-reload socket configuration.
   Model the doc on `task-153-corsair-battleship/backfill.md`: per-tenant
   `GET /tenants/{id}/configurations/{resource}`, insert the handler at its
   sorted opCode with `LoggedInValidator` and `"services": ["channel"]`,
   `PATCH` back.

---

## 11. Risks

| Risk | Mitigation |
|---|---|
| FR-1.1 lands without FR-1.2 ⇒ 1000× inflation silently masked by the 60 s mist clamp | Single commit; the analyzer fixture set includes the pre-fix forms; §8.2 makes the clamp's changed meaning explicit |
| Someone flips the contract a fourth time | D5 (one authority + pointers) and D6 (an analyzer that fails CI), plus the `11e07dfa7` reversal named in the code comment |
| Re-ingest skipped or atlas-data not rolled | §10 steps 2–4 are numbered and the verification is on the *data*, not the deploy |
| Throttle registry leaks entries | `ClearCharacter` in the socket destroyer, tested |
| A template entry lands without a validator ⇒ silently dropped handler | Every one of the ten entries specifies `LoggedInValidator`; the opcode-order guard runs in CI |
| Nine matrix cells is a wide verification surface | Each is a standard single-cell `/verify-packet` run against an empty body — the cheapest possible cell |

---

## 12. Explicitly out of scope

- `USER_CALC_DAMAGE_STAT_SET_REQUEST` (`0x6C`) — PRD §9.3; file as a follow-up
  at PR time (§7 residual).
- `MountBuffDuration = math.MaxInt32` and any client tick-overflow it causes.
- The benign `tSwallowBuffTime` over-send — PRD §9.6, resolved as not-a-bug.
- Creating `docs/packets/registry/gms_v92.yaml` — D7.
- Routing `CANCEL_DEBUFF` in `template_gms_12_1.json` — D8.
- Making the throttle window env-configurable — D2.
- Changing `libs/atlas-socket/server.go:282`'s Info-level unhandled-op log.
