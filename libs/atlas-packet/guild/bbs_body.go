package guild

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// Guild-BBS result-mode keys (CUIGuildBBS::OnGuildBBSPacket). Like the
// GuildOperation dispatcher, each BBS arm's MODE byte is resolved at emit time
// from the tenant "operations" table (docs/packets/dispatchers/guild_bbs.yaml) —
// never a struct literal. The 6/7/8 mode bytes are version-stable across
// gms_v83/v84/v87/v95, but they are config-resolved for pattern uniformity with
// the rest of the dispatcher families (jms-absent: no GuildBBS writer). Body
// functions fix the key; the constructor receives the RESOLVED mode.
const (
	GuildBBSOperationThreadList    = "BBS_THREAD_LIST"
	GuildBBSOperationThread        = "BBS_THREAD"
	GuildBBSOperationEntryNotFound = "BBS_ENTRY_NOT_FOUND"
)

func GuildBBSThreadListBody(notice *clientbound.BBSThreadSummary, threads []clientbound.BBSThreadSummary, startIndex uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildBBSOperationThreadList, func(mode byte) packet.Encoder {
		return clientbound.NewBBSThreadList(mode, notice, threads, startIndex)
	})
}

func GuildBBSThreadBody(id uint32, posterId uint32, createdAt int64, title string, message string, emoticonId uint32, replies []clientbound.BBSReply) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildBBSOperationThread, func(mode byte) packet.Encoder {
		return clientbound.NewBBSThread(mode, id, posterId, createdAt, title, message, emoticonId, replies)
	})
}

func GuildBBSEntryNotFoundBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildBBSOperationEntryNotFound, func(mode byte) packet.Encoder {
		return clientbound.NewBBSEntryNotFound(mode)
	})
}
