package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuildBossHealerMoveWriter = "GuildBossHealerMove"

// packet-audit:fname CField_GuildBoss::OnHealerMove
type GuildBossHealerMove struct {
	moveAction uint16
}

func NewGuildBossHealerMove(moveAction uint16) GuildBossHealerMove {
	return GuildBossHealerMove{moveAction: moveAction}
}

func (m GuildBossHealerMove) MoveAction() uint16 { return m.moveAction }

func (m GuildBossHealerMove) Operation() string { return GuildBossHealerMoveWriter }
func (m GuildBossHealerMove) String() string {
	return fmt.Sprintf("moveAction [%d]", m.moveAction)
}

func (m GuildBossHealerMove) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.moveAction)
		return w.Bytes()
	}
}

func (m *GuildBossHealerMove) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.moveAction = r.ReadUint16()
	}
}
