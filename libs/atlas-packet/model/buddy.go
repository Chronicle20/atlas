package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Buddy struct {
	FriendId    uint32
	FriendName  string
	Flag        byte
	ChannelId   channel.Id
	FriendGroup string
}

// BuddyHasFriendGroup reports whether the GW_Friend wire record carries the
// trailing 17-byte FriendGroup field.
//
// IDA-verified: the record is 22 bytes in GMS v61 (GW_Friend::Decode @0x4b54d8
// = DecodeBuffer(this, 22)) but 39 bytes in v72+ (GW_Friend::Decode @0x4d08d7 =
// DecodeBuffer(39)). The 17-byte delta is exactly the FriendGroup field, which
// buddy groups introduced after v61: v61's 22-byte layout is FriendId(4) +
// FriendName(13) + Flag(1) + ChannelId(4), with ChannelId at record offset 18
// (confirmed by OnFriendResult case 0x14 @0x858a20 reading `22*Index + 18`).
// Only GMS < 72 (i.e. v61) drops the field; all v72+ and JMS keep it, so v72+
// codec paths are unchanged. A nil ctx defaults to present (v72+ behavior).
func BuddyHasFriendGroup(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	t := tenant.MustFromContext(ctx)
	return t.Region() != "GMS" || t.MajorVersion() >= 72
}

func (b Buddy) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	hasGroup := BuddyHasFriendGroup(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(b.FriendId)
		WritePaddedString(w, b.FriendName, 13)
		w.WriteByte(b.Flag)
		w.WriteInt32(int32(b.ChannelId))
		if hasGroup {
			WritePaddedString(w, b.FriendGroup, 17)
		}
		return w.Bytes()
	}
}
