package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// AdminResultBody builds the ADMIN_RESULT (CField::OnAdminResult) clientbound
// packet — a GM-command result mode-demux. The codec + route exist so the admin
// feature can be switched on later without another packet-plumbing pass; the
// uncalled writer is a documented seam (IMPLEMENTING_A_PACKET D2), not dead code.
func AdminResultBody(mode byte, b []byte, s []string, mapId uint32) packet.Encode {
	return fieldcb.NewAdminResult(mode, b, s, mapId).Encode
}
