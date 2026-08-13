package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/consumable"
	"atlas-channel/data/skill/effect"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math/rand"

	"github.com/sirupsen/logrus"

	charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// loadCasterFunc is the caster-load seam tests can replace. Production
// calls atlas-character via character.Processor.GetById(); tests inject a
// stub returning a deterministic character.Model so the orchestrator can
// exercise its mob-selection / status-apply logic offline.
var loadCasterFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById()(characterId)
}

// loadCasterWithInventoryFunc is the inventory-decorated caster-load seam the
// generic itemCon consume path uses (it needs an arbitrary compartment, not
// just the USE compartment loadCasterInventoryFunc returns). Tests replace it.
var loadCasterWithInventoryFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById(cp.InventoryDecorator)(characterId)
}

// requestItemConsumeFunc is the consumable-request seam tests can replace.
// Production delegates to consumable.Processor.RequestItemConsume.
var requestItemConsumeFunc = func(p consumable.Processor, f field.Model, characterId charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, updateTime uint32) error {
	return p.RequestItemConsume(f, characterId, itemId, source, quantity, updateTime)
}

// rectQueryFunc is the mob-selection seam tests can replace. Production
// calls atlas-monsters via monster.Processor.GetInMapRect; tests inject a
// stub returning a fixed slice.
var rectQueryFunc = func(p monster.Processor, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
	return p.GetInMapRect(f, x1, y1, x2, y2, limit)
}

// propRollFunc gates per-target apply/cancel by the skill's prop value.
// Production uses a uniform RNG; tests inject a deterministic implementation
// via a t.Cleanup-restored override.
var propRollFunc = func(prop float64) bool {
	if prop <= 0 {
		return false
	}
	if prop >= 1 {
		return true
	}
	return rand.Float64() <= prop
}

// reflectLookupFunc is the magic-reflect probe seam tests can replace.
var reflectLookupFunc = func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool) {
	return monster.GetStatusMirror().GetReflect(t, monsterId, kind)
}

// applyStatusFunc is the status-apply emit seam tests can replace.
var applyStatusFunc = func(p monster.Processor, f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	return p.ApplyStatus(f, monsterId, characterId, skillId, skillLevel, statuses, duration)
}

// cancelStatusFunc is the status-cancel emit seam tests can replace.
var cancelStatusFunc = func(p monster.Processor, f field.Model, monsterId uint32, statusTypes []string, sourceCharacterId, sourceSkillId uint32, sourceSkillClass string) error {
	return p.CancelStatus(f, monsterId, statusTypes, sourceCharacterId, sourceSkillId, sourceSkillClass)
}

// applyCooldownFunc is the cast-time cooldown emit seam tests can replace.
var applyCooldownFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, skillId skill2.Id, cooldown uint32, characterId uint32) error {
	return skill.NewProcessor(l, ctx).ApplyCooldown(f, skillId, cooldown)(characterId)
}

// shouldApplyCastCooldown gates the generic cast-time cooldown. Battleship
// (5221006) is exempt: its cooldown applies only when the ship breaks, never
// on cast (FR-2.3/FR-4.3) — the WZ cooltime would otherwise fire here.
func shouldApplyCastCooldown(cooldown uint32, skillId skill2.Id) bool {
	return cooldown > 0 && !skill2.IsBattleshipMountSkill(skillId)
}

func UseSkill(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
	return func(ctx context.Context) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
		return func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
			// Resolved once and reused for every version-sensitive wire-id
			// comparison in this closure (task-187): a raw wire-keyed compare
			// cannot tell a v0.48 SuperGM Hide cast (wire 5101004) apart from a
			// v0.62+ Brawler Corkscrew Blow cast (same wire).
			t := tenant.MustFromContext(ctx)
			set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
			castId, castIdOk := set.Skill.Resolve(skill2.Id(info.SkillId()))

			// Shadow Stars pre-flight (FR-5): validate the client-chosen star
			// and resolve the cast cost BEFORE any HP/MP/cooldown spend. A bogus
			// or unowned star aborts the whole cast — no MP, no cooldown, no buff,
			// no consume — so a crafted client cannot inject an id into the buff
			// or trigger consumption of an unintended item.
			statupsToApply := e.StatUps()
			var shadowStarDraws []StarDraw
			if castIdOk && skill2.IsIdentity(castId, skill2.NightLordShadowStars) {
				assets, invErr := loadCasterInventoryFunc(character.NewProcessor(l, ctx), characterId)
				if invErr != nil {
					l.WithError(invErr).Warnf("Character [%d] cast Shadow Stars [%d] but inventory load failed; aborting cast.", characterId, info.SkillId())
					return nil
				}
				rewritten, draws, shortfall, ok := resolveShadowStarsCast(assets, e.StatUps(), info.SpiritJavelinItemId(), int(e.BulletConsume()))
				if !ok {
					l.Warnf("Character [%d] cast Shadow Stars [%d] with invalid star [%d] (not a throwing star or not owned); aborting cast.", characterId, info.SkillId(), info.SpiritJavelinItemId())
					return nil
				}
				if shortfall {
					l.Warnf("Character [%d] cast Shadow Stars [%d]: insufficient star [%d] for cast cost [%d]; consuming what's available.", characterId, info.SkillId(), info.SpiritJavelinItemId(), e.BulletConsume())
				}
				statupsToApply = rewritten
				shadowStarDraws = draws
			}

			if e.HPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeHP(f, characterId, -int16(e.HPConsume()))
			}
			if e.MPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeMP(f, characterId, -int16(e.MPConsume()))
			}
			if itemId := e.ItemConsume(); itemId > 0 {
				cp := character.NewProcessor(l, ctx)
				if c, cErr := loadCasterWithInventoryFunc(cp, characterId); cErr == nil {
					if invType, typeOk := inventoryconst.TypeFromItemId(itemconst.Id(itemId)); typeOk {
						amount := int16(e.ItemConsumeAmount())
						if amount < 1 {
							// Absent itemConNo (reader default 0) means one item (FR-1).
							amount = 1
						}
						if a, found := c.Inventory().CompartmentByType(invType).FindFirstByItemIdWithQuantity(itemId, amount); found {
							_ = requestItemConsumeFunc(consumable.NewProcessor(l, ctx), f, charcon.Id(characterId), itemconst.Id(itemId), slot.Position(a.Slot()), amount, 0)
						} else {
							l.Warnf("Character [%d] cast skill [%d] requiring [%d]x item [%d] but no single slot holds enough; cast permitted (defense-in-depth gate only).", characterId, info.SkillId(), amount, itemId)
						}
					}
				} else {
					l.WithError(cErr).Warnf("Character [%d] cast skill [%d] requiring item [%d] but failed to load inventory; cast permitted.", characterId, info.SkillId(), itemId)
				}
			}
			skillId := skill2.Id(info.SkillId())
			if shouldApplyCastCooldown(e.Cooldown(), skillId) {
				_ = applyCooldownFunc(l, ctx, f, skillId, e.Cooldown(), characterId)
			}
			// Mount toggle (tamed + skill-only + battleship). Runs BEFORE the
			// generic buff apply and short-circuits it: mounts apply
			// MONSTER_RIDING with a MaxInt32 duration and a vehicle-id amount,
			// or cancel on re-cast.
			// Routed through the resolved Identity (task-187) rather than a raw
			// wire compare, per the defensive-consistency scope for mount sites.
			if castIdOk && (skill2.IsTamedMountSkillIdentity(castId) || isSkillOnlyMountIdentity(castId, info.SkillLevel()) || skill2.IsBattleshipMountSkillIdentity(castId)) {
				if err := HandleMount(l, f, characterId, info, e, castId, newMountDeps(l, ctx)); err != nil {
					l.WithError(err).Errorf("Mount toggle failed for character [%d] skill [%d].", characterId, info.SkillId())
				}
				return nil
			}

			if e.Duration() > 0 && len(statupsToApply) > 0 {
				applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), statupsToApply)
				_ = applyBuffFunc(characterId)
				recipients := applyToParty(l)(ctx)(f, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)
				announceSkillAffected(l, ctx, wp, f, characterId, recipients, info.SkillId(), info.SkillLevel())
			}

			// Shadow Stars cast cost (FR-4): charge bulletConsume (200 in WZ) of the
			// chosen star after the buff is applied. shadowStarDraws is empty for every
			// other skill.
			if len(shadowStarDraws) > 0 {
				if err := emitStarConsume(l, ctx, characterId, shadowStarDraws); err != nil {
					l.WithError(err).Errorf("Character [%d] Shadow Stars cast-cost consume failed.", characterId)
				}
			}

			// Handle mob-affecting buffs (crash, dispel, etc.)
			applyToMobs(l, ctx, f, characterId, info, e)

			// Per-skill dispatcher (Heal, Dispel, Cure, MPEater, Drain, ...),
			// routed on the Identity resolved above.
			if castIdOk {
				if h, ok := Lookup(castId); ok {
					if err := h(l)(ctx)(wp, f, characterId, info, e); err != nil {
						l.WithError(err).Errorf("Skill handler for [%d] failed for character [%d].", info.SkillId(), characterId)
					}
				}
			}

			return nil
		}
	}
}

func applyToMobs(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) {
	mobIds := info.AffectedMobIds()
	if len(mobIds) == 0 {
		return
	}

	sid := skill2.Id(info.SkillId())
	slvl := uint32(info.SkillLevel())
	mobCap := e.MobCount()

	// FR-4.3 — mobCount cap. Reject the entire cast if the client claims more
	// targets than the skill's WZ definition permits. This runs before any
	// atlas-monsters round-trip; an over-cap cast produces zero emit calls.
	if ExceedsMobCap(l, "monster_buff_anomaly_over_cap", characterId, sid, slvl, mobCap, mobIds) {
		return
	}

	mp := monster.NewProcessor(l, ctx)

	var (
		applied         []uint32
		anomaly         []uint32
		mobsInRectCount = -1
		rect            [4]int16 // x1, y1, x2, y2 — only meaningful when bbox present
	)

	if !hasEffectBbox(e.LT(), e.RB()) {
		// FR-4.2 — no rect contract in WZ data; trust the client unmodified
		// for the rect check. Cap (already done), prop, reflect still apply.
		l.WithFields(logrus.Fields{
			"skill_id":         uint32(sid),
			"skill_level":      slvl,
			"client_mob_count": len(mobIds),
		}).Debug("mob_buff_no_effect_bbox")
		applied = mobIds
	} else {
		// FR-4.1 — rect verification. Bail-on-error policy: any failure
		// drops the cast. See design §5.1.
		cp := character.NewProcessor(l, ctx)
		c, cErr := loadCasterFunc(cp, characterId)
		if cErr != nil {
			l.WithError(cErr).WithFields(logrus.Fields{
				"event":        "mob_buff_caster_load_failed",
				"character_id": characterId,
				"skill_id":     uint32(sid),
			}).Error("mob_buff_caster_load_failed")
			return
		}
		facingLeft := (c.Stance() & 1) == 1
		x1, y1, x2, y2 := calculateBoundingBox(c.X(), c.Y(), facingLeft, e.LT(), e.RB())
		rect = [4]int16{x1, y1, x2, y2}

		mobs, qErr := rectQueryFunc(mp, f, x1, y1, x2, y2, mobCap)
		if qErr != nil {
			l.WithError(qErr).WithFields(logrus.Fields{
				"event":        "mob_buff_rect_query_failed",
				"character_id": characterId,
				"skill_id":     uint32(sid),
				"rect":         rect,
			}).Error("mob_buff_rect_query_failed")
			return
		}
		serverMobIds := make([]uint32, 0, len(mobs))
		for _, m := range mobs {
			serverMobIds = append(serverMobIds, m.UniqueId())
		}
		mobsInRectCount = len(serverMobIds)

		applied, anomaly = IntersectMobIds(mobIds, serverMobIds)

		if len(anomaly) > 0 {
			l.WithFields(logrus.Fields{
				"event":           "monster_buff_anomaly_out_of_rect",
				"character_id":    characterId,
				"skill_id":        uint32(sid),
				"skill_level":     slvl,
				"rect":            map[string]int16{"x1": x1, "y1": y1, "x2": x2, "y2": y2},
				"mob_count_cap":   mobCap,
				"client_mob_ids":  mobIds,
				"server_mob_ids":  serverMobIds,
				"anomaly_mob_ids": anomaly,
			}).Warn("client_targeted_mob_outside_server_rect")
		}
	}

	t := tenant.MustFromContext(ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, idOk := set.Skill.Resolve(sid)

	monsterStatuses := make(map[string]int32, len(e.MonsterStatus()))
	for k, v := range e.MonsterStatus() {
		monsterStatuses[k] = int32(v)
	}

	isCancel := idOk && isCrashOrDispel(id)
	cancelClass := ""
	if isCancel {
		cancelClass = dispelSkillClass(id)
	}

	// Branch selection mirrors the FR-4.9 rule: a skill takes EITHER the
	// cancel branch (Crash family / Priest Dispel) OR the apply branch
	// (Doom and any future entry with non-empty MonsterStatus). Never both.
	branch := propBranchApply
	if isCancel {
		branch = propBranchCancel
	} else if len(monsterStatuses) == 0 {
		// Buff-classified skill with no MonsterStatus map — defensive: nothing
		// to apply. Should not occur for skills in isMobAffectingBuff today.
		l.WithFields(logrus.Fields{
			"skill_id": uint32(sid),
		}).Debug("mob_buff_no_emit_branch")
		l.WithFields(buildSummaryFields(characterId, sid, slvl, mobsInRectCount, len(mobIds), 0, 0, 0, len(anomaly))).Debug("mob_buff_apply_summary")
		return
	}

	appliedCount, reflectSkipped, propSkipped := 0, 0, 0
	for _, mobId := range applied {
		// FR-4.6 — kind-aware reflect skip.
		var kind string
		if isCancel {
			kind = cancelClass
		} else if idOk {
			kind = mobBuffApplyKind(id)
		}
		if kind == "" {
			l.WithFields(logrus.Fields{
				"event":    "mob_buff_unclassified_kind",
				"skill_id": uint32(sid),
				"mob_id":   mobId,
			}).Debug("mob_buff_unclassified_kind")
		} else if _, hasReflect := reflectLookupFunc(t, mobId, kind); hasReflect {
			l.WithFields(logrus.Fields{
				"skill_id": uint32(sid),
				"mob_id":   mobId,
				"kind":     kind,
			}).Debug("mob_buff_reflect_skip")
			reflectSkipped++
			continue
		}

		// FR-4.5 — prop roll, with per-skill carve-out support.
		if propAppliesTo(sid, branch) {
			if !propRollFunc(e.Prop()) {
				propSkipped++
				continue
			}
		}

		// FR-4.9 — branch emit.
		if isCancel {
			_ = cancelStatusFunc(mp, f, mobId, nil, characterId, uint32(sid), cancelClass)
		} else {
			_ = applyStatusFunc(mp, f, mobId, characterId, uint32(sid), slvl, monsterStatuses, uint32(e.Duration()))
		}
		appliedCount++
	}

	l.WithFields(buildSummaryFields(characterId, sid, slvl, mobsInRectCount, len(mobIds), appliedCount, reflectSkipped, propSkipped, len(anomaly))).Debug("mob_buff_apply_summary")
}

// buildSummaryFields packs the FR-4.8 per-cast summary fields.
func buildSummaryFields(characterId uint32, sid skill2.Id, slvl uint32, mobsInRect, clientMobCount, applied, reflectSkipped, propSkipped, outOfRectDropped int) logrus.Fields {
	return logrus.Fields{
		"caster":              characterId,
		"skill_id":            uint32(sid),
		"skill_level":         slvl,
		"mobs_in_rect":        mobsInRect,
		"client_mob_count":    clientMobCount,
		"applied":             applied,
		"reflect_skipped":     reflectSkipped,
		"prop_skipped":        propSkipped,
		"out_of_rect_dropped": outOfRectDropped,
	}
}

// isCrashOrDispel resolves the incoming wire id to its version-blind
// Identity before comparing (task-187): a raw wire-keyed skill2.Is compare
// against these canonical constants would silently misclassify a version
// where an unrelated skill happens to share the wire id.
func isCrashOrDispel(id skill2.Identity) bool {
	return skill2.IsIdentity(id,
		skill2.CrusaderArmorCrash,
		skill2.WhiteKnightMagicCrash,
		skill2.DragonKnightPowerCrash,
		skill2.PriestDispel,
	)
}

// dispelSkillClass classifies a crash/dispel skill by the attacker's hit
// class — warrior crashes are physical melee, Priest Dispel is magic. The
// returned string matches atlas-monsters' monster.ReflectKind* constants.
// Returns "" for unrecognized skills so the downstream guard falls through
// to normal cancel semantics. Identity-keyed (task-187) for the same reason
// as isCrashOrDispel.
func dispelSkillClass(id skill2.Identity) string {
	switch {
	case skill2.IsIdentity(id,
		skill2.CrusaderArmorCrash,
		skill2.WhiteKnightMagicCrash,
		skill2.DragonKnightPowerCrash):
		return monster2.ReflectKindPhysical
	case skill2.IsIdentity(id, skill2.PriestDispel):
		return monster2.ReflectKindMagical
	default:
		return ""
	}
}

// applyToParty applies idOperator to every party member selected for a party
// buff and returns the ids it applied to. Party buffs apply map-wide (no LT/RB
// rectangle), so member selection is driven by the client-sent affected-member
// bitmap via SelectPartyMembersInMap rather than the range-limited Heal
// selector. The returned ids feed announceSkillAffected — the recipients are
// resolved once here and reused rather than re-selected.
func applyToParty(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, casterId uint32, memberBitmap byte) func(idOperator model2.Operator[uint32]) []uint32 {
	return func(ctx context.Context) func(f field.Model, casterId uint32, memberBitmap byte) func(idOperator model2.Operator[uint32]) []uint32 {
		return func(f field.Model, casterId uint32, memberBitmap byte) func(idOperator model2.Operator[uint32]) []uint32 {
			return func(idOperator model2.Operator[uint32]) []uint32 {
				recipients := SelectPartyMembersInMap(l, ctx, f, casterId, memberBitmap)
				ids := make([]uint32, 0, len(recipients))
				for _, r := range recipients {
					_ = idOperator(r.Id())
					ids = append(ids, r.Id())
				}
				return ids
			}
		}
	}
}

// announceSkillAffected draws the buff-received animation over every party
// member who was buffed by SOMEONE ELSE's cast, and over the same player on
// every other client in the map.
//
// The caster is excluded: their own client already renders the cast locally
// (CUserLocal::DoActiveSkill_StatChange calls CUser::ShowSkillEffect right
// after SendSkillUseRequest) and the server additionally sends them SKILL_USE.
// SKILL_AFFECTED is the recipient-facing arm — gms_v83 CUser::OnEffect
// @0x9377d9 case 2 → CUser::ShowSkillAffected @0x93632a.
func announceSkillAffected(
	l logrus.FieldLogger, ctx context.Context, wp writer.Producer,
	f field.Model, casterId uint32, recipientIds []uint32,
	skillId uint32, skillLevel byte,
) {
	for _, id := range recipientIds {
		if id == casterId {
			continue
		}
		skillAffectedEmitFunc(l, ctx, wp, f, id, skillId, skillLevel)
	}
}

// skillAffectedEmitFunc is the per-recipient SKILL_AFFECTED emit seam tests
// can replace. Production writes the self-facing CharacterEffect to the
// recipient's own session and the CharacterEffectForeign to everyone else on
// their map.
var skillAffectedEmitFunc = func(
	l logrus.FieldLogger, ctx context.Context, wp writer.Producer,
	f field.Model, recipientId uint32, skillId uint32, skillLevel byte,
) {
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(f.Channel())(
		recipientId,
		session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(
			charpkt.CharacterSkillAffectedEffectBody(skillId, skillLevel)),
	)
	_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(
		f, recipientId,
		session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(
			charpkt.CharacterSkillAffectedEffectForeignBody(recipientId, skillId, skillLevel)),
	)
}
