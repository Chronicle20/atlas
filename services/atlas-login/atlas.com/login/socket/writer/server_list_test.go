package writer

import (
	"atlas-login/socket/model"
	"atlas-login/world"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	world2 "github.com/Chronicle20/atlas/libs/atlas-constants/world"
	loginpkt "github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// The v83 client never formats a channel number off the wire — it indexes its
// channel-name array by the channel entry's packet position (bug:
// docs/tasks/task-238-whisper-find-location/bug-find-channel-always-zero.md).
// atlas-world returns channel loads in arbitrary order, so ServerListEntryBody
// must sort by ChannelId before handing entries to the encoder: packet
// position must equal channel id.
func TestServerListEntryBody_OrdersChannelsByIdRegardlessOfInputOrder(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()

	// Mirrors what atlas-world returns today: descending channel order.
	channelLoad := []model.Load{
		model.NewChannelLoad(channel.Id(1), 50),
		model.NewChannelLoad(channel.Id(0), 25),
	}

	body := ServerListEntryBody(world2.Id(0), "Scania", world.StateNormal, "", channelLoad)(l, ctx)(nil)

	req := request.Request(body)
	r := request.NewRequestReader(&req, 0)

	var entry loginpkt.ServerListEntry
	entry.Decode(l, ctx)(&r, nil)

	loads := entry.ChannelLoads()
	if len(loads) != 2 {
		t.Fatalf("expected 2 channel entries, got %d", len(loads))
	}
	for i, cl := range loads {
		if int(cl.ChannelId()) != i {
			t.Errorf("entry %d: got channel id %d, want %d", i, cl.ChannelId(), i)
		}
	}
}
