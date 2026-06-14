package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuildBossPulleyStateChangeWriter = "GuildBossPulleyStateChange"

type GuildBossPulleyStateChange struct {
	state byte
}

func NewGuildBossPulleyStateChange(state byte) GuildBossPulleyStateChange {
	return GuildBossPulleyStateChange{state: state}
}

func (m GuildBossPulleyStateChange) State() byte { return m.state }

func (m GuildBossPulleyStateChange) Operation() string { return GuildBossPulleyStateChangeWriter }
func (m GuildBossPulleyStateChange) String() string {
	return fmt.Sprintf("state [%d]", m.state)
}

func (m GuildBossPulleyStateChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.state)
		return w.Bytes()
	}
}

func (m *GuildBossPulleyStateChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = r.ReadByte()
	}
}
