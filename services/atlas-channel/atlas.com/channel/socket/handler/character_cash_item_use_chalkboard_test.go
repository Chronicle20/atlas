package handler

import (
	chalkboardMsg "atlas-channel/kafka/message/chalkboard"
	"encoding/json"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// chalkboardItemId (5370000) is classification 537 (item.ClassificationChalkboard),
// distinct from the Quick Delivery Ticket (5330000, classification 533) that
// collides with it on cash-slot type 32 under JMS (task-241 task-22).
const chalkboardItemId = uint32(5370000)

func decodeSetChalkboardCommand(t *testing.T, m kafka.Message) chalkboardMsg.Command[chalkboardMsg.SetCommandBody] {
	t.Helper()
	var cmd chalkboardMsg.Command[chalkboardMsg.SetCommandBody]
	if err := json.Unmarshal(m.Value, &cmd); err != nil {
		t.Fatalf("unmarshal SET chalkboard command: %v", err)
	}
	return cmd
}

// chalkboardSubBody encodes the ItemUseChalkboard sub-body (AsciiString
// message, then a trailing updateTime unless the tenant leads with it in the
// common ItemUse header) exactly as cashsb.ItemUseChalkboard.Encode does.
func chalkboardSubBody(updateTimeFirst bool, message string, updateTime uint32) []byte {
	w := response.NewWriter(logrus.New())
	w.WriteAsciiString(message)
	if !updateTimeFirst {
		w.WriteInt(updateTime)
	}
	return w.Bytes()
}

// TestHandleChalkboardCollision pins both sides of the cash-slot type-32
// collision the guard at character_cash_item_use.go:101-112 exists for:
// a genuine chalkboard item (classification 537) must still route to
// chalkboard.AttemptUse on GMS AND on JMS v185 (where GetCashSlotItemType
// also maps the Quick Delivery Ticket, classification 533, to the same
// enum 32), while the ticket itself must NOT reach the chalkboard path.
func TestHandleChalkboardCollision(t *testing.T) {
	const charId = uint32(444)
	const srcSlot = int16(3)

	t.Run("GMS chalkboard invokes AttemptUse", func(t *testing.T) {
		captured, restoreProducer := installCapturingProducer()
		defer restoreProducer()
		restoreSlot := installCashItemInSlotSeam(t, srcSlot, chalkboardItemId)
		defer restoreSlot()

		s, ctx, cleanup := newCashItemUseTestSession(t, charId)
		defer cleanup()

		ten := tenant.MustFromContext(ctx)
		if item.GetClassification(item.Id(chalkboardItemId)) != item.ClassificationChalkboard {
			t.Fatalf("fixture invalid: GetClassification(%d) = %d, want %d (chalkboard)", chalkboardItemId, item.GetClassification(item.Id(chalkboardItemId)), item.ClassificationChalkboard)
		}
		if got := GetCashSlotItemType(ten)(item.Id(chalkboardItemId)); got != CashSlotItemTypeChalkboard {
			t.Fatalf("fixture invalid: GetCashSlotItemType(%d) = %d, want %d (chalkboard)", chalkboardItemId, got, CashSlotItemTypeChalkboard)
		}

		updateTimeFirst := cashsb.UpdateTimeFirst(ten)
		raw := append(cashItemUsePrefix(srcSlot, chalkboardItemId), chalkboardSubBody(updateTimeFirst, "Hello!", 42)...)
		req := request.Request(raw)
		reader := request.NewRequestReader(&req, 0)
		CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

		msgs := (*captured)[chalkboardMsg.EnvCommandTopic]
		if len(msgs) != 1 {
			t.Fatalf("chalkboard SET messages produced = %d, want 1", len(msgs))
		}
		cmd := decodeSetChalkboardCommand(t, msgs[0])
		if cmd.Type != chalkboardMsg.CommandChalkboardSet {
			t.Errorf("command type = %q, want %q", cmd.Type, chalkboardMsg.CommandChalkboardSet)
		}
		if cmd.CharacterId != charId {
			t.Errorf("characterId = %d, want %d", cmd.CharacterId, charId)
		}
		if cmd.Body.Message != "Hello!" {
			t.Errorf("message = %q, want %q", cmd.Body.Message, "Hello!")
		}
	})

	t.Run("JMS chalkboard invokes AttemptUse", func(t *testing.T) {
		captured, restoreProducer := installCapturingProducer()
		defer restoreProducer()
		restoreSlot := installCashItemInSlotSeam(t, srcSlot, chalkboardItemId)
		defer restoreSlot()

		s, ctx, cleanup := newCashItemUseTestSessionForVersion(t, charId, "JMS", 185)
		defer cleanup()

		ten := tenant.MustFromContext(ctx)
		if got := GetCashSlotItemType(ten)(item.Id(chalkboardItemId)); got != CashSlotItemTypeChalkboard {
			t.Fatalf("fixture invalid: GetCashSlotItemType(%d) = %d, want %d (chalkboard)", chalkboardItemId, got, CashSlotItemTypeChalkboard)
		}
		// The Quick Delivery Ticket collides on this SAME enum under JMS; prove
		// the fixture is a genuine collision, not a distinct type.
		if got := GetCashSlotItemType(ten)(item.Id(item.QuickDeliveryTicketId)); got != CashSlotItemTypeChalkboard {
			t.Fatalf("fixture invalid: JMS Quick Delivery Ticket does not collide with chalkboard enum %d (got %d)", CashSlotItemTypeChalkboard, got)
		}

		updateTimeFirst := cashsb.UpdateTimeFirst(ten)
		// JMS leads with updateTime (cashsb.UpdateTimeFirst) before the common
		// source/itemId prefix — see character_cash_item_use_duey_test.go's
		// identical JMS raw-byte prepend.
		raw := append([]byte{0x2A, 0x00, 0x00, 0x00}, cashItemUsePrefix(srcSlot, chalkboardItemId)...)
		raw = append(raw, chalkboardSubBody(updateTimeFirst, "Hello!", 0)...)
		req := request.Request(raw)
		reader := request.NewRequestReader(&req, 0)
		CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

		msgs := (*captured)[chalkboardMsg.EnvCommandTopic]
		if len(msgs) != 1 {
			t.Fatalf("chalkboard SET messages produced = %d, want 1", len(msgs))
		}
		cmd := decodeSetChalkboardCommand(t, msgs[0])
		if cmd.Body.Message != "Hello!" {
			t.Errorf("message = %q, want %q", cmd.Body.Message, "Hello!")
		}
	})

	t.Run("JMS Quick Delivery Ticket does not invoke AttemptUse", func(t *testing.T) {
		captured, restoreProducer := installCapturingProducer()
		defer restoreProducer()
		restoreSlot := installCashItemInSlotSeam(t, srcSlot, item.QuickDeliveryTicketId)
		defer restoreSlot()
		sagas, restoreSaga := installDueyCouponSagaSeam(t)
		defer restoreSaga()

		s, ctx, cleanup := newCashItemUseTestSessionForVersion(t, charId, "JMS", 185)
		defer cleanup()

		raw := append([]byte{0x2A, 0x00, 0x00, 0x00}, cashItemUsePrefix(srcSlot, item.QuickDeliveryTicketId)...)
		req := request.Request(raw)
		reader := request.NewRequestReader(&req, 0)
		rec := &gaugeProducerRecorder{}
		CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

		msgs := (*captured)[chalkboardMsg.EnvCommandTopic]
		if len(msgs) != 0 {
			t.Errorf("chalkboard SET messages produced = %d, want 0 — the Quick Delivery Ticket must not reach chalkboard.AttemptUse", len(msgs))
		}
		if len(*sagas) != 1 {
			t.Fatalf("sagas created = %d, want 1 (must fall through to handleDueyCouponUse)", len(*sagas))
		}
	})
}
