package parcel

import (
	parcelmsg "atlas-channel/kafka/message/parcel"
	dueyparcel "atlas-channel/parcel"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// parcelEntryWidth is the fixed 234-byte PARCEL block plus the trailing
// hasItem bool — valid only when no fixture in this file attaches an item
// (none do; ToPacket's item field is exercised separately in
// parcel/model_test.go territory, not here).
const parcelEntryWidth = 234 + 1

func nullLogger() *logrus.Logger {
	l, _ := testlog.NewNullLogger()
	return l
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// newOldGmsTenant is an early GMS build, paired with newJmsTenant so the
// OPEN subtests prove the receiveOnly bool is FALSE on both — the NPC
// dialog is never opened receive-only, whatever the tenant.
func newOldGmsTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 61, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// newJmsTenant is a JMS v185 build — the column that used to take the
// "quick enabled" half of the old per-tenant gate, and so the one most
// likely to regress back to receiveOnly=true.
func newJmsTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "JMS", 185, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// newTestServer registers a server whose world/channel (0, 0) match
// session.NewSession's un-set default field, so IfPresentByCharacterId's
// world/channel filters actually match a directly-registered test session
// (mirrors kafka/consumer/report/consumer_test.go's identical helper).
func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 0)
	return server.NewProcessor(nullLogger(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

var operations = map[string]interface{}{
	parcelcb.ParcelOperationOpen:      float64(0x08),
	parcelcb.ParcelOperationOpenQuick: float64(0x1A),
}

// openCounts reads the raw OPEN body's mailbox/arrived counts. Layout
// (CTabReceive::SetParcel @0x6EF69C): mode byte, receiveOnly bool, count
// byte, count*PARCEL, newCount byte, newCount*PARCEL — every PARCEL here is
// exactly parcelEntryWidth bytes (no item attached).
func openCounts(t *testing.T, b []byte) (receiveOnly bool, mailbox, arrived int) {
	t.Helper()
	if len(b) < 3 {
		t.Fatalf("open body too short: % x", b)
	}
	receiveOnly = b[1] != 0
	mailbox = int(b[2])
	off := 3 + mailbox*parcelEntryWidth
	if len(b) <= off {
		t.Fatalf("open body too short for mailbox count %d: % x", mailbox, b)
	}
	arrived = int(b[off])
	return
}

// newRealSession registers a real session (net.Pipe-backed, matching
// kafka/consumer/report/consumer_test.go's pattern) for characterId on a
// (world 0, channel 0) session, so IfPresentByCharacterId actually finds it.
func newRealSession(tm tenant.Model, ctx context.Context, characterId uint32) (session.Model, func()) {
	serverConn, clientConn := net.Pipe()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tm, 0, serverConn)
	session.AddSessionToRegistry(tm.Id(), s)
	_ = session.NewProcessor(nullLogger(), ctx).SetCharacterId(sessionId, characterId)

	// net.Pipe is unbuffered — session.Announce's write to serverConn blocks
	// until something reads clientConn. Drain it in the background so the
	// handler under test never deadlocks on a write nothing is consuming.
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := clientConn.Read(buf); err != nil {
				return
			}
		}
	}()

	cleanup := func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
		session.ClearRegistryForTenant(tm.Id())
	}
	return s, cleanup
}

// wp resolves ParcelWriter against operations and captures the encoded bytes.
func wp(captured *[][]byte) writer.Producer {
	return func(name string) (socketwriter.BodyFunc, error) {
		if name != parcelcb.ParcelWriter {
			return nil, nil
		}
		return func(_ logrus.FieldLogger, _ context.Context) func(packet.Encode) []byte {
			return func(encode packet.Encode) []byte {
				b := encode(nullLogger(), context.Background())(map[string]interface{}{"operations": operations})
				*captured = append(*captured, b)
				return b
			}
		}, nil
	}
}

func receivableParcel(recipientId uint32, notified bool) dueyparcel.Model {
	var ln *time.Time
	if notified {
		n := time.Now().Add(-time.Hour)
		ln = &n
	}
	rm := dueyparcel.RestModel{
		Id:           uuid.New().String(),
		WorldId:      0,
		RecipientId:  recipientId,
		SenderName:   "Alice",
		MesoAmount:   1000,
		ReceivableAt: time.Now().Add(-time.Hour),
		LastNotified: ln,
	}
	m, _ := dueyparcel.Extract(rm)
	return m
}

func TestShowParcelCommand(t *testing.T) {
	worldId := world.Id(0)

	t.Run("open with mailbox", func(t *testing.T) {
		tm := newOldGmsTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()

		var captured [][]byte
		var notified []uuid.UUID
		deps := showParcelDeps{
			getMailbox: func(_ uint32, _ world.Id) ([]dueyparcel.Model, error) {
				return []dueyparcel.Model{receivableParcel(100, true), receivableParcel(100, true)}, nil
			},
			markNotified: func(id uuid.UUID) error { notified = append(notified, id); return nil },
		}

		e := parcelmsg.ShowParcelCommand{CharacterId: 100, WorldId: worldId, Quick: false}
		if err := showParcel(nullLogger(), ctx, wp(&captured), e, deps)(s); err != nil {
			t.Fatalf("showParcel: %v", err)
		}
		if len(captured) != 1 {
			t.Fatalf("announces = %d, want 1", len(captured))
		}
		receiveOnly, mailbox, arrived := openCounts(t, captured[0])
		if receiveOnly {
			t.Error("receiveOnly = true, want false — CParcelDlg(2) has no Send tab")
		}
		if mailbox != 2 {
			t.Errorf("mailbox = %d, want 2", mailbox)
		}
		if arrived != 0 {
			t.Errorf("arrived = %d, want 0", arrived)
		}
		if len(notified) != 0 {
			t.Errorf("notified = %v, want none", notified)
		}
	})

	t.Run("open with mailbox jms is not receive only", func(t *testing.T) {
		tm := newJmsTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()

		var captured [][]byte
		deps := showParcelDeps{
			getMailbox: func(_ uint32, _ world.Id) ([]dueyparcel.Model, error) {
				return []dueyparcel.Model{receivableParcel(100, true)}, nil
			},
			markNotified: func(_ uuid.UUID) error { return nil },
		}

		e := parcelmsg.ShowParcelCommand{CharacterId: 100, WorldId: worldId, Quick: false}
		if err := showParcel(nullLogger(), ctx, wp(&captured), e, deps)(s); err != nil {
			t.Fatalf("showParcel: %v", err)
		}
		receiveOnly, _, _ := openCounts(t, captured[0])
		if receiveOnly {
			t.Error("receiveOnly = true for a JMS v185 tenant, want false — the NPC dialog needs all three tabs")
		}
	})

	t.Run("open with new arrivals", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()

		newArrival := receivableParcel(100, false)
		var captured [][]byte
		var notified []uuid.UUID
		deps := showParcelDeps{
			getMailbox: func(_ uint32, _ world.Id) ([]dueyparcel.Model, error) {
				return []dueyparcel.Model{receivableParcel(100, true), newArrival}, nil
			},
			markNotified: func(id uuid.UUID) error { notified = append(notified, id); return nil },
		}

		e := parcelmsg.ShowParcelCommand{CharacterId: 100, WorldId: worldId, Quick: false}
		if err := showParcel(nullLogger(), ctx, wp(&captured), e, deps)(s); err != nil {
			t.Fatalf("showParcel: %v", err)
		}
		_, mailbox, arrived := openCounts(t, captured[0])
		if mailbox != 2 {
			t.Errorf("mailbox = %d, want 2", mailbox)
		}
		if arrived != 1 {
			t.Errorf("arrived = %d, want 1", arrived)
		}
		if len(notified) != 1 || notified[0] != newArrival.Id() {
			t.Errorf("notified = %v, want [%s]", notified, newArrival.Id())
		}
	})

	t.Run("open quick", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()

		var captured [][]byte
		mailboxFetched := false
		deps := showParcelDeps{
			getMailbox: func(_ uint32, _ world.Id) ([]dueyparcel.Model, error) {
				mailboxFetched = true
				return nil, nil
			},
			markNotified: func(_ uuid.UUID) error { return nil },
		}

		e := parcelmsg.ShowParcelCommand{CharacterId: 100, WorldId: worldId, Quick: true}
		if err := showParcel(nullLogger(), ctx, wp(&captured), e, deps)(s); err != nil {
			t.Fatalf("showParcel: %v", err)
		}
		if len(captured) != 1 {
			t.Fatalf("announces = %d, want 1", len(captured))
		}
		if mailboxFetched {
			t.Error("quick open must not fetch the mailbox")
		}
	})

	t.Run("not yet receivable excluded", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()

		notReceivable, _ := dueyparcel.Extract(dueyparcel.RestModel{
			Id:           uuid.New().String(),
			RecipientId:  100,
			ReceivableAt: time.Now().Add(time.Hour),
		})

		var captured [][]byte
		deps := showParcelDeps{
			getMailbox: func(_ uint32, _ world.Id) ([]dueyparcel.Model, error) {
				return []dueyparcel.Model{notReceivable}, nil
			},
			markNotified: func(_ uuid.UUID) error { return nil },
		}

		e := parcelmsg.ShowParcelCommand{CharacterId: 100, WorldId: worldId, Quick: false}
		if err := showParcel(nullLogger(), ctx, wp(&captured), e, deps)(s); err != nil {
			t.Fatalf("showParcel: %v", err)
		}
		_, mailbox, _ := openCounts(t, captured[0])
		if mailbox != 0 {
			t.Errorf("mailbox = %d, want 0 (FR-12)", mailbox)
		}
	})

	t.Run("wrong tenant", func(t *testing.T) {
		tm := newTestTenant(t)
		other := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), other)
		sc := newTestServer(t, tm)

		var captured [][]byte
		h := handleShowParcelCommand(sc, wp(&captured))
		h(nullLogger(), ctx, parcelmsg.ShowParcelCommand{Type: parcelmsg.CommandTypeShowParcel, CharacterId: 100, WorldId: sc.WorldId(), ChannelId: sc.ChannelId()})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})

	t.Run("recipient offline", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		sc := newTestServer(t, tm)

		var captured [][]byte
		h := handleShowParcelCommand(sc, wp(&captured))
		h(nullLogger(), ctx, parcelmsg.ShowParcelCommand{Type: parcelmsg.CommandTypeShowParcel, CharacterId: 999, WorldId: sc.WorldId(), ChannelId: sc.ChannelId()})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})
}
