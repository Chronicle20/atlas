package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// guildMemberLegacyNoAlliance reports whether this tenant serialises the legacy
// (pre-alliance) GUILDMEMBER: 33 bytes, WITHOUT the trailing AllianceTitle int.
// IDA-verified: GMS_v48 GUILDMEMBER::Decode@0x49c982 = DecodeBuffer(0x21=33);
// GMS_v61 GUILDMEMBER::Decode@0x4b54f6 = DecodeBuffer(37). Guild alliances (and
// the per-member AllianceTitle) were introduced at v61, so GMS < 61 omits it.
// (task-113 v48 close-I.) v28 is unverified-by-inference (no v28 IDB) — folded
// into the v48 legacy shape.
func guildMemberLegacyNoAlliance(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.IsRegion("GMS") && t.MajorVersion() < 61
}

type GuildMember struct {
	Name          string
	JobId         uint16
	Level         byte
	Title         byte
	Online        bool
	Signature     uint32
	AllianceTitle byte
}

func (b GuildMember) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	legacyNoAlliance := guildMemberLegacyNoAlliance(ctx)
	return func(options map[string]interface{}) []byte {
		WritePaddedString(w, b.Name, 13)
		w.WriteInt(uint32(b.JobId))
		w.WriteInt(uint32(b.Level))
		w.WriteInt(uint32(b.Title))
		if b.Online {
			w.WriteInt(1)
		} else {
			w.WriteInt(0)
		}
		w.WriteInt(b.Signature)
		if !legacyNoAlliance {
			w.WriteInt(uint32(b.AllianceTitle))
		}
		return w.Bytes()
	}
}
