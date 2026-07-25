package npc

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// NpcController leading-flag keys (CNpcPool::OnNpcChangeController). Each
// resolves to the per-version flag byte via the tenant "operations" table
// (options.operations, writer SpawnNPCRequestController). The flag byte is
// per-tenant/version DATA — never a struct literal. Body functions fix the
// key; the constructor receives the RESOLVED flag (config-driven contract,
// like the guild/mts/storage dispatcher families) (DOM-25, task-176).
const (
	NpcControllerGrant  = "GRANT"
	NpcControllerRevoke = "REVOKE"
)

// NpcControllerGrantBody emits the grant arm (client takes local control of
// the NPC) with the leading flag byte resolved from the tenant "operations"
// table under NpcControllerGrant.
func NpcControllerGrantBody(id uint32, template uint32, x int16, cy int16, f int32, fh uint16, rx0 int16, rx1 int16, miniMap bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NpcControllerGrant, func(flag byte) packet.Encoder {
		return clientbound.NewNpcSpawnRequestController(flag, id, template, x, cy, f, fh, rx0, rx1, miniMap)
	})
}

// NpcControllerRevokeBody emits the revoke arm (client demotes the NPC to
// remote control) with the leading flag byte resolved from the tenant
// "operations" table under NpcControllerRevoke.
func NpcControllerRevokeBody(id uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NpcControllerRevoke, func(flag byte) packet.Encoder {
		return clientbound.NewNpcRemoveController(flag, id)
	})
}
