package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
)

const LoginAuth = "LoginAuth"

func LoginAuthBody(screen string) packet.Encode {
	return loginpkt.NewLoginAuth(screen).Encode
}
