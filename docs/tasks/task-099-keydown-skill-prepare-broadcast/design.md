# Keydown Skill Prepare/Cancel Broadcast — Design

Status: Draft
Created: 2026-06-15
PRD: ./prd.md
---

## 1. Summary

Implement the keydown-skill *prepare* (keydown) and *cancel* (keyup) packet family in atlas-channel so observers in the same map see the looping cast aura start and stop. The server is a **validated relay**: it decodes the serverbound prepare/cancel from the caster, validates skill ownership + keydown classification, and broadcasts a clientbound *foreign* prepare/cancel to the other sessions in the caster's map. No persistence, no Kafka, no cross-service calls — structurally identical to the existing attack broadcast (`processAttack` → `ForOtherSessionsInMap`).

This is a packet-wiring + codec task. Its risk is almost entirely in **per-version wire correctness** (opcodes + read/write order), not in service logic. Accordingly the design centers on: (a) a minimal, idiomatic channel handler/writer pair, and (b) a per-version IDB verification strategy that pins every byte before wiring.

## 2. Chosen architecture

### 2.1 Packet set (four packets, two of them foreign-only writers)

| Role | Kind | New artifact | Opcode source |
|---|---|---|---|
| serverbound prepare (keydown) | handler | `CharacterSkillPrepareHandle` | per-version IDB (`DoActiveSkill_Prepare`) |
| serverbound cancel (keyup) | handler | `CharacterSkillCancelHandle` | per-version IDB (`SendSkillCancelRequest`) |
| clientbound remote prepare | foreign writer | `CharacterSkillPrepareForeignWriter` | per-version IDB (`OnSkillPrepare`) |
| clientbound remote cancel | foreign writer | `CharacterSkillCancelForeignWriter` | per-version IDB (`OnSkillCancel`) |

**Decision D1 — foreign-only writers (no self writer).** Unlike `character_skill_use` (which announces to self *and* foreign via `CharacterEffect`/`CharacterEffectForeign`), the casting client renders its own keydown aura locally and never needs an echo. Only observers need the packet. So we add two **foreign** writers and no self writers — half the surface of the skill-use pattern.

**Decision D2 — dedicated writers, not a mode on `CharacterEffectForeign`.** The prepare/cancel packets are distinct client opcodes (`SKILL_EFFECT` / `CANCEL_SKILL_EFFECT`, e.g. v83 `0xBE`/`0xBF`), not the `CharacterEffect` opcode (`0xCE`-family). Multiplexing them onto the effect writer would emit the wrong opcode and wrong body. They get their own writer names + their own config opcode rows. (Alternative A in §5.)

### 2.2 Inbound handlers (atlas-channel `socket/handler/`)

New file `character_skill_prepare.go` with two handlers following the `character_skill_use.go` template exactly:

```
const CharacterSkillPrepareHandle = "CharacterSkillPrepareHandle"
const CharacterSkillCancelHandle  = "CharacterSkillCancelHandle"

func CharacterSkillPrepareHandleFunc(l, ctx, wp)(s, r, readerOptions) {
    info := &packetmodel.SkillPrepareInfo{}; info.Decode(l, ctx)(r, readerOptions)   // version-conditional decode
    c, err := character.NewProcessor(l, ctx).GetById(cp.SkillModelDecorator)(s.CharacterId())   // ownership lookup
    // validate: own the skill at a non-zero level  AND  skill.IsKeyDownSkill(info.SkillId())
    //   on miss -> log (debug) + return  (do NOT Destroy the session; see D3)
    _ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(),
        AnnounceForeignSkillPrepare(l)(ctx)(wp)(s.CharacterId(), info))
}
```

`CharacterSkillCancelHandleFunc` is the same shape with a smaller body (`skillId` only) and `AnnounceForeignSkillCancel`.

**Decision D3 — drop-on-mismatch, do not Destroy.** `character_skill_use` calls `session.Destroy` when the caster doesn't own the skill (anti-cheat on a state-changing action). Prepare/cancel are low-stakes *visual* packets; a benign client/skill desync should not disconnect the player. On ownership/keydown-classification miss we log at debug and return. (Ownership validation still happens — FR-1.3 — it just gates the broadcast rather than punishing.)

**Decision D4 — gate on `skill.IsKeyDownSkill`.** The 14-id canonical classifier in `libs/atlas-constants/skill` is the single source of truth. The handler broadcasts only for those skills; everything else is dropped. (Confirms FR-1.4/3.3; the constant already covers Hurricane, Monster Magnet ×3, Rapid Fire, Piercing Arrow, Poison Bomb, BigBang ×3, WindArcher Hurricane, ThunderBreaker Corkscrew, Evan Ice/Fire Breath.)

### 2.3 Codecs (libs/atlas-packet)

- Inbound: `model.SkillPrepareInfo` with `Decode(l, ctx)(r, options)`, version-conditional via `tenant.MustFromContext(ctx)` (mirrors `model.AttackInfo`). Working read order (v95): `skillId` u32, `level` u8, `action` u16 (bit15 = move-action, low15 = action), `actionSpeed` u8. Cancel has no dedicated model — it's a single `skillId` u32 read inline (too small to warrant a struct), or a tiny `SkillCancelInfo` for symmetry; executor's call.
- Outbound: `character/clientbound/skill_prepare_foreign.go` (`CharacterSkillPrepareForeign`) and `skill_cancel_foreign.go` (`CharacterSkillCancelForeign`), each a struct + `Encode(l, ctx)(options)` writing `characterId` first then the body, version-conditional like `effect_skill_use.go`.

**Decision D5 — one version-conditional codec per packet, read order pinned per IDB.** Rather than a codec per version, use a single codec with `if t.Region()=="GMS" && t.MajorVersion() >= N` branches (the established `AttackInfo`/`EffectSkillUse` idiom). The exact branch set is filled in from the per-version IDB read orders (OQ-3), not assumed equal.

### 2.4 Broadcast helpers (`socket/handler/effects.go`)

Add `AnnounceForeignSkillPrepare` and `AnnounceForeignSkillCancel` mirroring `AnnounceForeignSkillUse` — curried `(l)(ctx)(wp)(characterId, …) Operator[session.Model]` returning `session.Announce(l)(ctx)(wp)(<ForeignWriter>)(<body>)`.

### 2.5 Registration & config

- `main.go produceHandlers()`: `handlerMap[handler.CharacterSkillPrepareHandle] = handler.CharacterSkillPrepareHandleFunc` (+ cancel).
- `main.go produceWriters()`: add the two foreign writer name constants.
- Seed templates `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`: add `socket.handlers[]` rows `{opCode, validator:"LoggedInValidator", handler}` and `socket.writers[]` rows `{opCode, writer}` for every supported version.
- **Live tenant config patch** (FR-7.2): handlers/writers don't hot-reload from the config projection, so the live channel config for each existing tenant must be patched and the channel restarted. Every handler row MUST carry a validator or `BuildHandlerMap` silently drops it.

## 3. Per-version verification strategy (the core risk)

Opcodes and read/write orders are **derived from each version's client IDB**, not the registry CSV or Cosmic. Verification is a fan-out of `packet-verifier`-style passes (one per packet × version) following `docs/packets/audits/VERIFYING_A_PACKET.md`, producing byte-fixture tests and promoting the matrix cell.

IDB instances: v83 `:13342`, v84 `:13337`, v87 `:13341`, v95 `:13340`, JMS185 `:13339`.

Per version, pin from the IDB:
1. serverbound prepare opcode + read order (`CUserLocal::DoActiveSkill_Prepare`).
2. serverbound cancel opcode + read order (`CUserLocal::SendSkillCancelRequest` / keyup path) — **OQ-1**, do not trust Cosmic's `CANCEL_BUFF` overload.
3. clientbound remote-prepare opcode + read order (`CUserRemote::OnSkillPrepare`).
4. clientbound remote-cancel opcode + read order (`CUserRemote::OnSkillCancel`).
5. which in-scope skills route through `OnMovingShootAttackPrepare` (nType 216) vs `OnSkillPrepare` (nType 215) — **OQ-5/D8**.

Each (packet, version) gets a byte-fixture test (`packet-audit:verify` marker) and a STATUS.md promotion. The matrix going green for all four ops × supported versions is an acceptance gate.

## 4. Resolved open questions / decisions

- **D6 — termination (OQ-4): relay the keyup cancel; rely on avatar removal for map-leave/disconnect; no new server-side keydown state.** The primary teardown is broadcasting the remote-cancel when the caster's keyup arrives. When a caster leaves the map or disconnects, the observer client removes the avatar entirely (aura goes with it) — so no synthesized cancel is needed for those paths, and we avoid introducing per-session "active keydown" state. Death/stun: expected to make the caster's client send a cancel (keydown interrupt); verify empirically/IDB during execution, and only add server-side synthesis if a stuck aura is actually observed. This keeps the server stateless for this feature.
- **D7 — v92 has no IDB (OQ-2): port from the structurally-nearest verified version + registry opcode, banner UNVERIFIED.** v92 cannot be byte-verified. Implement it by reusing the nearest verified read order (v95 or v87, whichever the other GMS versions agree with) and the registry opcode, and banner the v92 matrix cells UNVERIFIED (precedent: parked v92 mount-food). This preserves all-version parity for a low-stakes visual packet at accepted risk. **Flagged for the user** — the alternative is deferring v92 wiring until a v92 IDB exists (Alternative D, §5).
- **D8 — MovingShootAttackPrepare (OQ-5): include iff verification shows an in-scope skill uses it.** Structure the codec/handler so a moving-shoot-prepare packet is an incremental addition. If the IDB shows e.g. Rapid Fire / Hurricane-while-moving dispatch through nType 216, add `CharacterMovingShootPrepare*` as a fifth/sixth artifact; otherwise document it as out of scope and move on.
- **D9 — leave the attack writer alone.** `socket/writer/character_attack_common.go:isKeydownSkill` (the narrow BigBang/breath list) stays untouched; IDA proved the remote attack `tKeyDown` is BigBang-only and broadening it would corrupt the shoot-attack packet. This is a regression guard, not a change.

## 5. Alternatives considered

- **Alt A — reuse `CharacterEffectForeign` with a new mode** instead of dedicated writers. Rejected: prepare/cancel are distinct client opcodes with their own bodies; the effect writer would emit the wrong opcode. (See D2.)
- **Alt B — server-side keydown state machine** (track active keydown per character; synthesize cancels on any interruption: stun, debuff, death, leave). Rejected for MVP: adds state + lifecycle hooks for a purely cosmetic packet; avatar removal + client keyup-cancel cover the real cases. Revisit only if stuck auras are observed.
- **Alt C — broaden the attack-packet `isKeydownSkill`** (the originally tempting one-liner). Rejected: IDA-proven incorrect (remote attack `tKeyDown` is read only for the BigBang trio + Evan magic skills, never for shoot skills like Hurricane); it would write a field the observer never reads and corrupt the packet.
- **Alt D — defer v92** until a v92 IDB exists, wiring only the five IDB-backed versions. Viable and lower-risk-of-wrongness than D7; the tradeoff is v92 players keep the bug. Recommend D7 (port + UNVERIFIED banner) unless the user prefers strict IDB-only.

## 6. Affected files (anticipated)

- `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go` (new — both handlers)
- `services/atlas-channel/atlas.com/channel/socket/handler/effects.go` (add two foreign announce helpers)
- `services/atlas-channel/atlas.com/channel/main.go` (register 2 handlers + 2 writers)
- `libs/atlas-packet/model/skill_prepare_info.go` (new — inbound decode; + cancel)
- `libs/atlas-packet/character/clientbound/skill_prepare_foreign.go`, `skill_cancel_foreign.go` (new — writers + writer-name consts)
- `libs/atlas-packet/...` byte-fixture tests (per version)
- `services/atlas-configurations/seed-data/templates/template_*_*.json` (handler + writer rows per version)
- `docs/packets/audits/STATUS.md` + per-version audit docs (matrix promotion)
- Live tenant config patch (operational, per existing tenant)

No `go.mod` changes anticipated → docker-bake gate N/A; standard `go test -race`/`vet`/`build` for atlas-channel + libs/atlas-packet, plus `tools/redis-key-guard.sh` (no redis here, but it runs repo-wide).

## 7. Risks

- **Wrong per-version wire format** → client desync/crash. Mitigated by IDB verification + byte-fixture tests before wiring; never trust the registry CSV or Cosmic field labels.
- **v92 unverifiable** (D7) → accepted risk, bannered; visual-only so a wrong opcode degrades to "no aura"/"unhandled op" rather than a crash, but confirm the v92 opcode isn't aliased to a state-changing handler before wiring.
- **Live-config drift** → existing tenants don't get the behavior, or a validator-less handler row is silently dropped. Mitigated by the explicit live-patch + validator checklist (FR-7.2).
- **Stuck aura** if a termination path is missed → mitigated by D6; add server synthesis only if observed.
