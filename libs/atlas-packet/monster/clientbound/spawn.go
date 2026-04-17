package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterSpawnWriter = "SpawnMonster"

type Spawn struct {
	uniqueId   uint32
	controlled bool
	monsterId  uint32
	monster    model.MonsterModel
}

func NewMonsterSpawn(uniqueId uint32, controlled bool, monsterId uint32, monster model.MonsterModel) Spawn {
	return Spawn{
		uniqueId:   uniqueId,
		controlled: controlled,
		monsterId:  monsterId,
		monster:    monster,
	}
}

func (m Spawn) UniqueId() uint32           { return m.uniqueId }
func (m Spawn) Controlled() bool           { return m.controlled }
func (m Spawn) MonsterId() uint32          { return m.monsterId }
func (m Spawn) Monster() model.MonsterModel { return m.monster }
func (m Spawn) Operation() string          { return MonsterSpawnWriter }
func (m Spawn) String() string {
	return fmt.Sprintf("uniqueId [%d], controlled [%t], monsterId [%d]", m.uniqueId, m.controlled, m.monsterId)
}

func (m Spawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			if m.controlled {
				w.WriteByte(1)
			} else {
				w.WriteByte(5)
			}
		}
		w.WriteInt(m.monsterId)
		w.WriteByteArray(m.monster.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Spawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			cb := r.ReadByte()
			m.controlled = cb == 1
		}
		m.monsterId = r.ReadUint32()
		m.monster.Decode(l, ctx)(r, options)
	}
}
