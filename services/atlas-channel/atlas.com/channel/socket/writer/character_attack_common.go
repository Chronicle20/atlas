package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	"atlas-channel/socket/model"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

func WriteCommonAttackBody(t tenant.Model) func(c character.Model, ai model.AttackInfo) func(w *response.Writer) {
	return func(c character.Model, ai model.AttackInfo) func(w *response.Writer) {
		return func(w *response.Writer) {
			w.WriteInt(c.Id())
			w.WriteByte(byte(ai.Damage()<<4 | uint32(ai.Hits())))
			w.WriteByte(c.Level())
			if ai.SkillId() > 0 {
				// Cannot rely on ai.SkillLevel here because it is not provided GMS < 95
				var sk skill.Model
				for _, tsk := range c.Skills() {
					if tsk.Id() == ai.SkillId() {
						sk = tsk
					}
				}
				w.WriteByte(sk.Level())
				w.WriteInt(ai.SkillId())
			} else {
				w.WriteByte(0)
			}
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				// TODO identify what was introduced here
				if skill2.Is(skill2.Id(ai.SkillId()), skill2.SniperStrafeId) {
					w.WriteByte(0) // passive SLV
					if false {
						w.WriteInt(0) // passive skill id
					}
				}
			}

			// TODO better document what this value
			// interested in & 0x20 here
			w.WriteByte(ai.Option())
			left := 0
			if ai.Left() {
				left = 1
			}
			w.WriteInt16(int16((left << 15) | ai.AttackAction()))
			// TODO identify 0x110 constant introduced between GMS v83 and v95
			if ai.AttackAction() <= 0x110 {
				w.WriteByte(ai.ActionSpeed())
				w.WriteByte(0) // mastery

				var bulletItemId = ai.BulletItemId()
				if ai.CashBulletPosition() > 0 {
					for _, i := range c.Inventory().Cash().Items() {
						if uint16(i.Slot()) == ai.CashBulletPosition() {
							bulletItemId = i.ItemId()
							break
						}
					}
				} else if ai.ProperBulletPosition() > 0 {
					for _, i := range c.Inventory().Use().Items() {
						if uint16(i.Slot()) == ai.ProperBulletPosition() && (i.Bullet() || i.ThrowingStar()) {
							bulletItemId = i.ItemId()
						}
					}
				}
				w.WriteInt(bulletItemId)

				for _, di := range ai.DamageInfo() {
					w.WriteInt(di.MonsterId())
					if di.MonsterId() > 0 {
						w.WriteByte(di.HitAction())
						if skill2.Is(skill2.Id(ai.SkillId()), skill2.ChiefBanditMesoExplosionId) {
							w.WriteByte(byte(len(di.Damages())))
						}
						for _, d := range di.Damages() {
							w.WriteInt(d)
						}
					}
				}
			}

			if ai.AttackType() == model.AttackTypeRanged {
				w.WriteShort(ai.BulletX())
				w.WriteShort(ai.BulletY())
			}
			if skill2.Is(skill2.Id(ai.SkillId()), skill2.FirePoisonArchMagicianBigBangId, skill2.IceLightningArchMagicianBigBangId, skill2.BishopBigBangId, skill2.EvanStage4IceBreathId, skill2.EvanStage7FireBreathId) {
				w.WriteInt(ai.Keydown())
			}
			// TODO Wild Hunter swallow
		}
	}
}
