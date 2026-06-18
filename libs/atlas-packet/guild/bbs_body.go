package guild

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	"github.com/sirupsen/logrus"
)

// Guild-BBS clientbound body functions (CUIGuildBBS::OnGuildBBSPacket).
//
// Unlike the GuildOperation dispatcher, the BBS sub-dispatcher's mode bytes
// (6/7/8 = OnLoadListResult / OnViewEntryResult / OnEntryNotFound) are
// VERSION-STABLE across gms_v83/v84/v87/v95 and are NOT carried in any tenant
// `operations` table (guild_bbs.yaml: no seed template registers a GuildBBS
// operations map). They are therefore passed through as the fixed package consts
// clientbound.GuildBBSMode* rather than resolved via WithResolvedCode — wiring a
// config table the writer never reads would be a false config dependency. The
// constructors still take `mode byte` first (discrete-per-mode contract); the
// body function fixes the per-arm mode const, never a caller-supplied selector.

func GuildBBSThreadListBody(notice *clientbound.BBSThreadSummary, threads []clientbound.BBSThreadSummary, startIndex uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return clientbound.NewBBSThreadList(clientbound.GuildBBSModeThreadList, notice, threads, startIndex).Encode
}

func GuildBBSThreadBody(id uint32, posterId uint32, createdAt int64, title string, message string, emoticonId uint32, replies []clientbound.BBSReply) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return clientbound.NewBBSThread(clientbound.GuildBBSModeThread, id, posterId, createdAt, title, message, emoticonId, replies).Encode
}

func GuildBBSEntryNotFoundBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return clientbound.NewBBSEntryNotFound(clientbound.GuildBBSModeEntryNotFound).Encode
}
