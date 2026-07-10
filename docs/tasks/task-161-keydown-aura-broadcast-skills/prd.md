# Keydown Cast-Aura Broadcast for Four Missing v83 Skills — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

When a player channels a "keydown" skill (one where the client holds the attack key and plays a
looping wind-up/cast animation — e.g. Bowmaster's Hurricane), the v83 client sends a serverbound
`CUserLocal::DoActiveSkill_Prepare` packet. atlas-channel relays this to the other sessions in the
map so they see the caster's wind-up aura, and relays the matching cancel (on key-up) so the aura
stops. The gate for both relays is `skill.IsKeyDownSkill(skillId)` in
`libs/atlas-constants/skill/model.go`.

That gate is currently missing four skills that are keydown-channeled in v83:

- **FP Mage Explosion** — `2111002` (`FirePoisonMagicianExplosionId`)
- **Chief Bandit Chakra** — `4211001` (`ChiefBanditChakraId`)
- **Brawler Corkscrew Blow** — `5101004` (`BrawlerCorkscrewBlowId`)
- **Gunslinger Grenade** — `5201002` (`GunslingerGrenadeId`)

Because they are absent from the predicate, a caster of any of these four produces **no wind-up /
cast aura for observers** in the map. The gap was surfaced by comparison against Cosmic's
`SkillEffectHandler.java:58-77`, which broadcasts these four in addition to the set Atlas already
covers.

The seemingly one-line fix carries a real hazard: **`IsKeyDownSkill` is a single predicate doing
double duty.** Beyond the two display relays, it also gates a wire-format field in the shared
attack-packet codec (`libs/atlas-packet/model/attack_info.go`) that decides whether an extra
`keyDown` uint32 is read (serverbound) and written (clientbound). Adding a skill that the v83
client does *not* actually carry a `keyDown` field for would misalign attack-packet parsing —
corrupting damage / crashing. Project memory explicitly warns "do not broaden the writer's narrow
`isKeydownSkill`" and flags Cosmic's list as unfaithful to v83 wire behavior. This PRD therefore
requires the four skills be **IDA-verified against the v83 client** before landing, and separates
the display concern from the wire concern so the attack codec can never be broadened by accident.

## 2. Goals

Primary goals:
- Casts of the four named skills broadcast a wind-up/cast aura to other players in the map, and stop
  it on key-up, matching the behavior Atlas already gives Hurricane et al.
- The attack-packet `keyDown` field alignment is provably unchanged for every skill unless IDA
  confirms a skill genuinely carries that field in the v83 attack sender.
- Byte-level confirmation that the observer receives the aura for each verified skill.

Non-goals:
- Reworking the `DoActiveSkill_Prepare` / cancel packet handlers themselves — they already exist and
  work for the covered skills.
- Adding any skill beyond the four named (no speculative expansion of the keydown set).
- Extending or auditing keydown behavior for non-v83 tenant versions (v87 / v95 / jms), except as
  incidentally required by the shared codec — see §7.
- Implementing skills that are entirely unimplemented in atlas-channel (this is a broadcast-visibility
  fix, not a new-skill feature).

## 3. User Stories

- As a player standing near an FP Mage / Chief Bandit / Brawler / Gunslinger who is channeling
  Explosion / Chakra / Corkscrew Blow / Grenade, I want to see their cast wind-up aura, so the world
  looks consistent with every other keydown skill.
- As a maintainer, I want the attack-packet wire format to remain byte-identical for all existing
  skills after this change, so I can trust that a display fix did not silently break combat parsing.
- As a reviewer, I want each of the four skill IDs backed by an IDA citation proving it is keydown in
  the v83 client, so no Cosmic-sourced guess lands in the predicate.

## 4. Functional Requirements

### FR-1 — IDA verification (gating, per §Answers)
For each of the four skill IDs, the design phase MUST establish against the v83 client:
- **FR-1.1** Whether the skill is in the client's `is_keydown_skill` set (the property gating
  `OnSkillPrepare`).
- **FR-1.2** Whether the v83 client actually sends `DoActiveSkill_Prepare` (serverbound) when the
  skill is cast — because if it does not, there is nothing for atlas-channel to relay and the skill
  must be dropped from scope with that finding recorded.
- **FR-1.3** Whether the v83 attack sender writes a `keyDown` (`tKeyDown`) uint32 for the skill —
  this determines whether the skill belongs in the *attack-wire* predicate (see FR-3).
- **FR-1.4** Any skill that fails FR-1.1/FR-1.2 verification is removed from scope; the negative
  finding is documented in `design.md`. Cosmic's `SkillEffectHandler.java` is corroborating context
  only, never the authority.

### FR-2 — Cast-aura broadcast (display)
- **FR-2.1** For each verified skill, a cast (prepare) by the owner is relayed to all other sessions
  in the caster's map via the existing `AnnounceForeignSkillPrepare` path.
- **FR-2.2** For each verified skill, the key-up cancel is relayed via the existing
  `AnnounceForeignSkillCancel` path so the observer's aura stops.
- **FR-2.3** The existing ownership/level guard (`shouldBroadcastKeydown`: skill present in the
  caster's skill book at level > 0) continues to apply.

### FR-3 — Separation of display and wire concerns (decision from Q2)
The single predicate is split so the two concerns cannot be conflated:
- **FR-3.1** A **display predicate** (e.g. `BroadcastsCastAura`, name TBD in design) gates the
  prepare/cancel relays in `character_skill_prepare.go` and `character_buff_cancel.go`. The four
  verified skills are added here.
- **FR-3.2** `IsKeyDownSkill` (or its successor) retains its role as the **attack-wire** gate in
  `attack_info.go` and admits a skill ONLY when FR-1.3 confirms the v83 attack sender carries the
  `keyDown` field for it.
- **FR-3.3** If FR-1.3 shows all four already-verified skills DO carry the attack `keyDown` field
  (i.e. the two sets are identical), design MAY keep a single predicate rather than splitting — but
  must state that finding explicitly. The default, absent that proof, is the split. Either way the
  attack-wire membership must equal the IDA-confirmed `keyDown`-carrying set exactly.
- **FR-3.4** The existing display set (Big Bang ×3, Monster Magnet ×3, Hurricane, Piercing Arrow,
  Rapid Fire, WA Hurricane, NW Poison Bomb, TB Corkscrew, Evan breaths) must be preserved in the
  display predicate with no regressions.

### FR-4 — Attack-codec regression safety
- **FR-4.1** The set of skills for which `attack_info.go` reads/writes the `keyDown` field MUST NOT
  change unless FR-1.3 justifies it per skill.
- **FR-4.2** A test asserts the attack-packet encode/decode byte layout is unchanged for a
  representative non-keydown skill and for an existing keydown skill (e.g. Hurricane), guarding
  against accidental broadening.

## 5. API Surface

No REST/HTTP surface. The affected surface is the MapleStory socket protocol, all internal to
atlas-channel and its packet library:

- Serverbound: `CUserLocal::DoActiveSkill_Prepare` (already decoded) — relay condition broadened.
- Clientbound: `OnSkillPrepare` (foreign prepare relay) and the keydown cancel relay — emitted for
  the four verified skills to other map sessions.
- Serverbound/clientbound: attack packet (`attack_info.go`) — `keyDown` field presence governed per
  FR-3.2/FR-4.1. No new fields; only the membership set that triggers the existing field.

No new Kafka topics or messages.

## 6. Data Model

No database schema changes, no new entities, no migrations. The only "data" changed is the static
skill-ID membership of one or two predicate functions in `libs/atlas-constants/skill/model.go`.

## 7. Service Impact

- **`libs/atlas-constants`** (`skill/model.go`) — add the four verified IDs to the display predicate;
  introduce the split predicate per FR-3 (or document the single-predicate justification). This lib
  is consumed by both atlas-channel and atlas-packet, so a change here recompiles both.
- **`libs/atlas-packet`** (`model/attack_info.go`) — update the wire gate to reference the
  attack-wire predicate (FR-3.2). Behavior must be byte-unchanged unless FR-1.3 dictates otherwise.
- **`services/atlas-channel`** (`socket/handler/character_skill_prepare.go`,
  `socket/handler/character_buff_cancel.go`) — repoint the relay gates to the display predicate.
- **Version scope:** the four skill IDs are v83-class ids (job ranges 21xx/42xx/51xx/52xx) shared
  across GMS versions. Because the predicate is version-agnostic Go (not tenant-config driven), the
  membership applies to all versions that run this codec. Design must confirm this does not
  mis-gate the attack `keyDown` field for v87/v95/jms (the attack-wire predicate is the risk surface,
  not the display one). If a version divergence is found, it is recorded as a follow-up, not silently
  broadened.

## 8. Non-Functional Requirements

- **Correctness / safety:** attack-packet parsing must remain byte-correct for all skills (FR-4).
  This is the dominant NFR — a display regression is cosmetic, a wire regression corrupts combat.
- **Grounding:** every skill added is backed by an IDA citation (address/function) in `design.md`;
  no Cosmic-sourced value lands unverified (project "verify, don't invent" rule).
- **Multi-tenancy:** predicate is pure Go, tenant-independent; no per-tenant config. The relay path
  already runs under the caster's tenant/field context — unchanged.
- **Observability:** existing debug logs in the two handlers ("not a keydown skill or not owned,
  skipping broadcast") continue to cover the miss path; no new logging required, but design should
  confirm a verified skill now takes the broadcast path rather than the skip path.
- **Performance:** membership test is an O(n) switch over a tiny constant list; negligible.

## 9. Open Questions

- **OQ-1 (resolved by verification):** Do all four skills actually cause the v83 client to send
  `DoActiveSkill_Prepare`? If any do not, that skill leaves scope (FR-1.4). To be answered in design
  via IDA, not assumed.
- **OQ-2 (resolved by verification):** Does the v83 attack sender carry a `keyDown` field for these
  four? Determines single-predicate vs. split (FR-3.3).
- **OQ-3:** Is the naming `BroadcastsCastAura` vs. keeping `IsKeyDownSkill` as the display name (and
  renaming the wire predicate) the least-churn option given the four call sites? Design decides.
- **OQ-4:** For versions beyond v83, do these four ids also carry the attack `keyDown` field? If IDA
  access is limited to v83, note the assumption and scope the wire predicate to what is verified.

## 10. Acceptance Criteria

- [ ] Each of the four skill IDs has an IDA citation in `design.md` establishing its v83
      `is_keydown_skill` status and whether the client sends `DoActiveSkill_Prepare` (FR-1).
- [ ] Any skill that fails verification is explicitly dropped with the finding recorded (FR-1.4).
- [ ] Casting each verified skill relays a prepare packet to other map sessions; key-up relays the
      cancel (FR-2.1, FR-2.2).
- [ ] The display gate and the attack-wire gate are separated per FR-3 (or a single predicate is
      retained with an explicit FR-3.3 justification).
- [ ] The set of skills triggering the attack `keyDown` field is unchanged except where FR-1.3
      justifies it; a byte-layout regression test guards this (FR-4).
- [ ] **Byte-level confirmation** (per the answered scope question): a byte-fixtured test demonstrates
      the observer receives the correct `OnSkillPrepare` bytes for each verified skill, and the
      key-up cancel bytes.
- [ ] `go test -race ./...`, `go vet ./...`, and `go build ./...` clean in every changed module
      (`libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel`).
- [ ] `docker buildx bake atlas-channel` succeeds from the worktree root.
- [ ] `tools/redis-key-guard.sh` clean.
