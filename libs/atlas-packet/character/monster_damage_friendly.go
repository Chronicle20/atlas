package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterDamageFriendlyHandle = "MonsterDamageFriendlyHandle"

// MonsterDamageFriendly - CMob::Update
type MonsterDamageFriendly struct {
	attackerId uint32
	observerId uint32
	attackedId uint32
}

func (m MonsterDamageFriendly) AttackerId() uint32 { return m.attackerId }
func (m MonsterDamageFriendly) ObserverId() uint32 { return m.observerId }
func (m MonsterDamageFriendly) AttackedId() uint32 { return m.attackedId }

func (m MonsterDamageFriendly) Operation() string {
	return MonsterDamageFriendlyHandle
}

func (m MonsterDamageFriendly) String() string {
	return fmt.Sprintf("attackerId [%d], observerId [%d], attackedId [%d]", m.attackerId, m.observerId, m.attackedId)
}

func (m MonsterDamageFriendly) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.attackerId)
		w.WriteInt(m.observerId)
		w.WriteInt(m.attackedId)
		return w.Bytes()
	}
}

func (m *MonsterDamageFriendly) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.attackerId = r.ReadUint32()
		m.observerId = r.ReadUint32()
		m.attackedId = r.ReadUint32()
	}
}
