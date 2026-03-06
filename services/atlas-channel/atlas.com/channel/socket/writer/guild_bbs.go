package writer

import (
	"atlas-channel/guild/thread"
	"context"
	"math"
	"time"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	GuildBBS = "GuildBBS"
)

func GuildBBSThreadsBody(ts []thread.Model, startIndex uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(0x06)
			if len(ts) == 0 {
				w.WriteByte(0)
				w.WriteInt(0)
				return w.Bytes()
			}
			nt := ts[0]

			var at []thread.Model
			if nt.Notice() {
				w.WriteByte(1)
				w.WriteInt(nt.Id())
				w.WriteInt(nt.PosterId())
				w.WriteAsciiString(nt.Title())
				w.WriteInt64(msTime(nt.CreatedAt()))
				w.WriteInt(nt.EmoticonId())
				w.WriteInt(uint32(len(nt.Replies())))
				at = append(ts[:0], ts[1:]...)
			} else {
				w.WriteByte(0)
				at = ts
			}
			w.WriteInt(uint32(len(at)))
			if len(at) > 0 {
				bound := uint32(math.Min(10, float64(len(at))-float64(startIndex)))
				w.WriteInt(bound)
				for i := startIndex; i < startIndex+bound; i++ {
					w.WriteInt(at[i].Id())
					w.WriteInt(at[i].PosterId())
					w.WriteAsciiString(at[i].Title())
					w.WriteInt64(msTime(at[i].CreatedAt()))
					w.WriteInt(at[i].EmoticonId())
					w.WriteInt(uint32(len(at[i].Replies())))
				}
			}
			return w.Bytes()
		}
	}
}

func GuildBBSThreadBody(t thread.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(0x07)
			w.WriteInt(t.Id())
			w.WriteInt(t.PosterId())
			w.WriteInt64(msTime(t.CreatedAt()))
			w.WriteAsciiString(t.Title())
			w.WriteAsciiString(t.Message())
			w.WriteInt(t.EmoticonId())
			w.WriteInt(uint32(len(t.Replies())))
			for _, r := range t.Replies() {
				w.WriteInt(r.Id())
				w.WriteInt(r.PosterId())
				w.WriteInt64(msTime(r.CreatedAt()))
				w.WriteAsciiString(r.Message())
			}
			return w.Bytes()
		}
	}
}

func msTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}
