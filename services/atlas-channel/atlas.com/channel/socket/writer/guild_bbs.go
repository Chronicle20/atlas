package writer

import (
	"atlas-channel/guild/thread"
	"context"

	guildpkt "github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func GuildBBSThreadsBody(ts []thread.Model, startIndex uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			if len(ts) == 0 {
				return guildpkt.NewBBSThreadList(nil, nil, startIndex).Encode(l, ctx)(options)
			}

			var notice *guildpkt.BBSThreadSummary
			var threads []guildpkt.BBSThreadSummary

			nt := ts[0]
			if nt.Notice() {
				n := guildpkt.BBSThreadSummary{
					Id:         nt.Id(),
					PosterId:   nt.PosterId(),
					Title:      nt.Title(),
					CreatedAt:  packetmodel.MsTime(nt.CreatedAt()),
					EmoticonId: nt.EmoticonId(),
					ReplyCount: uint32(len(nt.Replies())),
				}
				notice = &n
				for _, t := range ts[1:] {
					threads = append(threads, guildpkt.BBSThreadSummary{
						Id:         t.Id(),
						PosterId:   t.PosterId(),
						Title:      t.Title(),
						CreatedAt:  packetmodel.MsTime(t.CreatedAt()),
						EmoticonId: t.EmoticonId(),
						ReplyCount: uint32(len(t.Replies())),
					})
				}
			} else {
				for _, t := range ts {
					threads = append(threads, guildpkt.BBSThreadSummary{
						Id:         t.Id(),
						PosterId:   t.PosterId(),
						Title:      t.Title(),
						CreatedAt:  packetmodel.MsTime(t.CreatedAt()),
						EmoticonId: t.EmoticonId(),
						ReplyCount: uint32(len(t.Replies())),
					})
				}
			}
			return guildpkt.NewBBSThreadList(notice, threads, startIndex).Encode(l, ctx)(options)
		}
	}
}

func GuildBBSThreadBody(t thread.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			var replies []guildpkt.BBSReply
			for _, r := range t.Replies() {
				replies = append(replies, guildpkt.BBSReply{
					Id:        r.Id(),
					PosterId:  r.PosterId(),
					CreatedAt: packetmodel.MsTime(r.CreatedAt()),
					Message:   r.Message(),
				})
			}
			return guildpkt.NewBBSThread(t.Id(), t.PosterId(), packetmodel.MsTime(t.CreatedAt()), t.Title(), t.Message(), t.EmoticonId(), replies).Encode(l, ctx)(options)
		}
	}
}

