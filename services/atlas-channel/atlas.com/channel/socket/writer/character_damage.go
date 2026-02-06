package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterDamage = "CharacterDamage"

func CharacterDamageBody(l logrus.FieldLogger) func(ctx context.Context) func(c character.Model, di model.DamageTakenInfo) BodyProducer {
	return func(ctx context.Context) func(c character.Model, di model.DamageTakenInfo) BodyProducer {
		t := tenant.MustFromContext(ctx)
		return func(c character.Model, di model.DamageTakenInfo) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				w.WriteInt(c.Id())
				w.WriteByte(byte(di.AttackIdx()))
				w.WriteInt32(di.Damage())
				if di.AttackIdx() == model.DamageTypePhysical || di.AttackIdx() == model.DamageTypeMagic {
					w.WriteInt(di.MonsterTemplateId())
					w.WriteBool(di.Left())

					stance := false
					w.WriteBool(stance)
					if stance {
						w.WriteBool(di.PowerGuard())
						w.WriteInt16(di.HitX())
						w.WriteInt16(di.HitY())
						powerGuard := false
						if powerGuard {
							w.WriteByte(0)  // hit action
							w.WriteInt16(0) // x
							w.WriteInt16(0) // y
						} else {
							w.WriteByte(0)
							w.WriteInt16(0)
							w.WriteInt16(0)
						}
					}
					if t.Region() == "GMS" && t.MajorVersion() >= 95 {
						w.WriteByte(0) // bGuard
					}
					w.WriteByte(0) // something that on &1 and &2 may relate to stance
				}
				w.WriteInt32(di.Damage())
				if di.Damage() == -1 {
					w.WriteInt(0) // misdirection skill 4120002
				}
				return w.Bytes()
			}
		}
	}
}
