package message

import (
	message3 "atlas-channel/kafka/message/message"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// This file is a regression guard for the whisper cross-channel delivery bug
// (docs/tasks/fix-whisper-cross-channel-delivery/bug-whisper-cross-channel-delivery.md):
// handleWhisperChat used to gate BOTH the sender's confirmation and the
// recipient's delivery on the sender's own channel, so a recipient logged
// into any other channel in the same world never received the whisper. Every
// scenario below simulates a handler bound to one specific channel — exactly
// as InitHandlers registers one handleWhisperChat closure per (tenant, world,
// channel) socket listener — receiving the same chat event.

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

func newServerModel(worldId world.Id, channelId channel.Id, tm tenant.Model) server.Model {
	ch := channel.NewModel(worldId, channelId)
	return server.NewProcessor(nullLogger(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// characterJSON renders a minimal JSON:API character document, mirroring
// kafka/consumer/map/consumer_test.go's characterJSON fixture.
func characterJSON(id uint32, name string, gm int) string {
	return fmt.Sprintf(`{
		"data": {
			"type": "characters",
			"id": "%d",
			"attributes": {
				"accountId": 1,
				"worldId": 0,
				"name": "%s",
				"level": 1,
				"gm": %d,
				"sp": "0,0,0,0,0,0,0,0,0,0"
			}
		}
	}`, id, name, gm)
}

// characterServer serves characterJSON for each id in characters, and 404 for
// anything else.
func characterServer(t *testing.T, characters map[uint32]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		for id, name := range characters {
			if r.URL.Path == fmt.Sprintf("/characters/%d", id) {
				_, _ = fmt.Fprint(w, characterJSON(id, name, 0))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// pipedSession registers a character-bound session on ch, backed by a real
// net.Conn (net.Pipe) so session.Announce's write reaches somewhere a test
// can observe. The Create handshake (session.Processor.Create writes a Hello
// packet synchronously) is drained here before returning, so callers only see
// the writes produced by the handler under test.
func pipedSession(t *testing.T, ctx context.Context, ch channel.Model, characterId uint32) (reads <-chan []byte, cleanup func()) {
	t.Helper()
	serverConn, clientConn := net.Pipe()

	readCh := make(chan []byte, 8)
	stop := make(chan struct{})
	go func() {
		for {
			buf := make([]byte, 4096)
			n, err := clientConn.Read(buf)
			if err != nil {
				return
			}
			b := make([]byte, n)
			copy(b, buf[:n])
			select {
			case readCh <- b:
			case <-stop:
				return
			}
		}
	}()

	sessionId := uuid.New()
	session.NewProcessor(nullLogger(), ctx).Create(ch, 0)(sessionId, serverConn)
	session.NewProcessor(nullLogger(), ctx).SetCharacterId(sessionId, characterId)

	// Drain the Hello packet written by Create before the caller starts
	// asserting on handler-produced writes.
	select {
	case <-readCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the session Hello handshake write")
	}

	cleanup = func() {
		close(stop)
		_ = clientConn.Close()
		_ = serverConn.Close()
	}
	return readCh, cleanup
}

func expectPacket(t *testing.T, reads <-chan []byte, label string) {
	t.Helper()
	select {
	case b := <-reads:
		if len(b) == 0 {
			t.Fatalf("%s: got a zero-length write, want a whisper packet", label)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("%s: timed out waiting for the expected whisper packet", label)
	}
}

func expectNoPacket(t *testing.T, reads <-chan []byte, label string) {
	t.Helper()
	select {
	case b := <-reads:
		t.Fatalf("%s: got an unexpected write (%d bytes), want none", label, len(b))
	case <-time.After(200 * time.Millisecond):
	}
}

// fakeWriterProducer resolves every writer name to a BodyFunc that ignores
// the encoder and returns fixed bytes — the assertions here are about WHICH
// session gets written to, not the packet's exact wire bytes.
func fakeWriterProducer() writer.Producer {
	return func(_ string) (socketwriter.BodyFunc, error) {
		return func(_ logrus.FieldLogger, _ context.Context) func(_ packet.Encode) []byte {
			return func(_ packet.Encode) []byte { return []byte{0xAB, 0xCD} }
		}, nil
	}
}

const (
	worldId         = world.Id(0)
	senderChannelId = channel.Id(0)
	recipChannelId  = channel.Id(1)
	senderCharId    = uint32(1)
	recipientCharId = uint32(2)
	senderName      = "Atlas"
	recipientName   = "Chronicle"
)

func whisperEvent() message3.ChatEvent[message3.WhisperChatBody] {
	return message3.ChatEvent[message3.WhisperChatBody]{
		WorldId:   worldId,
		ChannelId: senderChannelId, // set from the sender's field.Model by atlas-messages
		ActorId:   senderCharId,
		Message:   "yo",
		Type:      message3.ChatTypeWhisper,
		Body:      message3.WhisperChatBody{Recipient: recipientCharId},
	}
}

// TestHandleWhisperChat_RecipientChannel_DeliversReceiveOnly is scenario 1:
// the handler bound to the recipient's channel (not the sender's) must
// deliver WhisperReceive and must NOT emit the sender's WhisperSendResult.
func TestHandleWhisperChat_RecipientChannel_DeliversReceiveOnly(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	defer session.ClearRegistryForTenant(tm.Id())

	srv := characterServer(t, map[uint32]string{senderCharId: senderName})
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	recipCh := channel.NewModel(worldId, recipChannelId)
	recipReads, cleanup := pipedSession(t, ctx, recipCh, recipientCharId)
	defer cleanup()

	sc := newServerModel(worldId, recipChannelId, tm)
	h := handleWhisperChat(sc, fakeWriterProducer())
	h(nullLogger(), ctx, whisperEvent())

	expectPacket(t, recipReads, "recipient (bound channel)")
}

// TestHandleWhisperChat_SenderChannel_DeliversResultOnly is scenario 2: the
// handler bound to the sender's channel (not the recipient's, who is on a
// different channel) must emit WhisperSendResult to the sender and must NOT
// deliver WhisperReceive to the recipient (who is not present on this
// handler's channel).
func TestHandleWhisperChat_SenderChannel_DeliversResultOnly(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	defer session.ClearRegistryForTenant(tm.Id())

	srv := characterServer(t, map[uint32]string{senderCharId: senderName, recipientCharId: recipientName})
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	senderCh := channel.NewModel(worldId, senderChannelId)
	senderReads, senderCleanup := pipedSession(t, ctx, senderCh, senderCharId)
	defer senderCleanup()

	// The recipient really is logged in, just on a different channel than
	// this handler — reinforces that GetByCharacterId(sc.Channel()) is what
	// keeps them from receiving anything here, not their absence entirely.
	recipCh := channel.NewModel(worldId, recipChannelId)
	recipReads, recipCleanup := pipedSession(t, ctx, recipCh, recipientCharId)
	defer recipCleanup()

	sc := newServerModel(worldId, senderChannelId, tm)
	h := handleWhisperChat(sc, fakeWriterProducer())
	h(nullLogger(), ctx, whisperEvent())

	expectPacket(t, senderReads, "sender (bound channel)")
	expectNoPacket(t, recipReads, "recipient (different channel)")
}

// TestHandleWhisperChat_SameChannel_DeliversBothOnce is scenario 3: sender
// and recipient both on the handler's channel — both packets are emitted,
// exactly once each.
func TestHandleWhisperChat_SameChannel_DeliversBothOnce(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	defer session.ClearRegistryForTenant(tm.Id())

	srv := characterServer(t, map[uint32]string{senderCharId: senderName, recipientCharId: recipientName})
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	ch := channel.NewModel(worldId, senderChannelId)
	senderReads, senderCleanup := pipedSession(t, ctx, ch, senderCharId)
	defer senderCleanup()
	recipReads, recipCleanup := pipedSession(t, ctx, ch, recipientCharId)
	defer recipCleanup()

	sc := newServerModel(worldId, senderChannelId, tm)
	h := handleWhisperChat(sc, fakeWriterProducer())
	h(nullLogger(), ctx, whisperEvent())

	expectPacket(t, senderReads, "sender (same channel)")
	expectPacket(t, recipReads, "recipient (same channel)")
	expectNoPacket(t, senderReads, "sender (same channel, second packet)")
	expectNoPacket(t, recipReads, "recipient (same channel, second packet)")
}

// TestHandleWhisperChat_DifferentWorld_NoAnnounce is scenario 4: a handler
// bound to a different world than the event must not attempt either
// announcement at all — proven by never resolving a writer.
func TestHandleWhisperChat_DifferentWorld_NoAnnounce(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	defer session.ClearRegistryForTenant(tm.Id())

	otherWorld := world.Id(1)
	sc := newServerModel(otherWorld, senderChannelId, tm)

	wp := writer.Producer(func(name string) (socketwriter.BodyFunc, error) {
		t.Fatalf("writer %q resolved for an event outside this handler's world", name)
		return nil, nil
	})

	h := handleWhisperChat(sc, wp)
	h(nullLogger(), ctx, whisperEvent())
}
