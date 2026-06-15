package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AdminCommandHandle = "AdminCommand"

// AdminCommand - CField::SendChatMsgSlash#AdminCommand (opcode varies per version).
// Sent by the /-command parser for the GM admin-command family. Every send-site
// leads with a single sub-command byte; the remaining payload is variable per
// sub-command (string/scalar combos), so only the stable leading sub-command
// byte is modeled here (decode-and-log).
// packet-audit:fname CField::SendChatMsgSlash#AdminCommand
type AdminCommand struct {
	subCommand byte
}

func NewAdminCommand(subCommand byte) AdminCommand {
	return AdminCommand{subCommand: subCommand}
}

func (m AdminCommand) SubCommand() byte { return m.subCommand }

func (m AdminCommand) Operation() string {
	return AdminCommandHandle
}

func (m AdminCommand) String() string {
	return fmt.Sprintf("subCommand [%d]", m.subCommand)
}

func (m AdminCommand) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.subCommand)
		return w.Bytes()
	}
}

func (m *AdminCommand) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.subCommand = r.ReadByte()
	}
}
