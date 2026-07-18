package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const GuildBossHandle = "GuildBoss"

// GuildBoss - CField_GuildBoss::BasicActionAttack
// Sent after CPulley::Hit in the guild boss minigame. Empty body (header only).
// packet-audit:fname CField_GuildBoss::BasicActionAttack#GuildBoss
type GuildBoss struct{}

func NewGuildBoss() GuildBoss {
	return GuildBoss{}
}

func (m GuildBoss) Operation() string {
	return GuildBossHandle
}

func (m GuildBoss) String() string {
	return "empty"
}

func (m GuildBoss) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *GuildBoss) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
