package shops

import (
	shop "atlas-npc/kafka/message/shops"
	"atlas-npc/shops"
	"atlas-npc/test"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// TestHandleEnterCommand_PropagatesTransactionId asserts the enter command's
// transaction id survives the wire round trip. It is the only correlation key
// the saga orchestrator has (task-221 design delta D2).
func TestHandleEnterCommand_PropagatesTransactionId(t *testing.T) {
	txn := uuid.New()
	cmd := shop.Command[shop.CommandShopEnterBody]{
		TransactionId: txn,
		CharacterId:   1234,
		Type:          shop.CommandShopEnter,
		Body:          shop.CommandShopEnterBody{NpcTemplateId: 9090000},
	}

	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round shop.Command[shop.CommandShopEnterBody]
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.TransactionId != txn {
		t.Errorf("TransactionId = %s, want %s", round.TransactionId, txn)
	}
}

// TestStatusEvent_EnterErrorRoundTrips locks the ENTER_ERROR shape. It must be
// distinct from StatusEventTypeError: the channel writes NPCShopOperation for
// ERROR, and that packet with no outstanding request disconnects the client
// (producer.go:36-51).
func TestStatusEvent_EnterErrorRoundTrips(t *testing.T) {
	txn := uuid.New()
	e := shop.StatusEvent[shop.StatusEventEnterErrorBody]{
		TransactionId: txn,
		CharacterId:   1234,
		Type:          shop.StatusEventTypeEnterError,
		Body:          shop.StatusEventEnterErrorBody{NpcTemplateId: 9090000, Reason: shop.EnterErrorShopNotFound},
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round shop.StatusEvent[shop.StatusEventEnterErrorBody]
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.TransactionId != txn || round.Type != "ENTER_ERROR" || round.Body.Reason != "SHOP_NOT_FOUND" {
		t.Errorf("round-trip mismatch: %+v", round)
	}
}

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	shops.InitRegistry(client)
}

func TestHandleEnterCommand(t *testing.T) {
	t.Run("wrong command type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		// Create command with wrong type
		cmd := shop.Command[shop.CommandShopEnterBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopExit, // Wrong type
			Body: shop.CommandShopEnterBody{
				NpcTemplateId: 9001,
			},
		}

		// Handler should return early without error (no processor call)
		// We use nil db - if it tried to call processor, it would panic
		handler := handleEnterCommand(nil)
		handler(logger, ctx, cmd) // Should not panic
	})

	t.Run("correct command type processes enter", func(t *testing.T) {
		setupTestRegistry(t)
		// Create processor with test database
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		// Create command with correct type
		cmd := shop.Command[shop.CommandShopEnterBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopEnter,
			Body: shop.CommandShopEnterBody{
				NpcTemplateId: 9001,
			},
		}

		// Handler should process the command
		handler := handleEnterCommand(db)
		handler(logger, ctx, cmd) // Should not panic, may log errors if shop doesn't exist
	})
}

func TestHandleExitCommand(t *testing.T) {
	t.Run("wrong command type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		// Create command with wrong type
		cmd := shop.Command[shop.CommandShopExitBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopEnter, // Wrong type
			Body:        shop.CommandShopExitBody{},
		}

		handler := handleExitCommand(nil)
		handler(logger, ctx, cmd) // Should not panic
	})

	t.Run("correct command type processes exit", func(t *testing.T) {
		setupTestRegistry(t)
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		cmd := shop.Command[shop.CommandShopExitBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopExit,
			Body:        shop.CommandShopExitBody{},
		}

		handler := handleExitCommand(db)
		handler(logger, ctx, cmd) // Should not panic
	})
}

func TestHandleBuyCommand(t *testing.T) {
	t.Run("wrong command type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		cmd := shop.Command[shop.CommandShopBuyBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopSell, // Wrong type
			Body: shop.CommandShopBuyBody{
				Slot:           1,
				ItemTemplateId: 2000000,
				Quantity:       10,
				DiscountPrice:  100,
			},
		}

		handler := handleBuyCommand(nil)
		handler(logger, ctx, cmd) // Should not panic
	})

	t.Run("correct command type processes buy", func(t *testing.T) {
		setupTestRegistry(t)
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		cmd := shop.Command[shop.CommandShopBuyBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopBuy,
			Body: shop.CommandShopBuyBody{
				Slot:           1,
				ItemTemplateId: 2000000,
				Quantity:       10,
				DiscountPrice:  100,
			},
		}

		handler := handleBuyCommand(db)
		handler(logger, ctx, cmd) // Should not panic
	})
}

func TestHandleSellCommand(t *testing.T) {
	t.Run("wrong command type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		cmd := shop.Command[shop.CommandShopSellBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopBuy, // Wrong type
			Body: shop.CommandShopSellBody{
				Slot:           1,
				ItemTemplateId: 2000000,
				Quantity:       5,
			},
		}

		handler := handleSellCommand(nil)
		handler(logger, ctx, cmd) // Should not panic
	})

	t.Run("correct command type processes sell", func(t *testing.T) {
		setupTestRegistry(t)
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		cmd := shop.Command[shop.CommandShopSellBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopSell,
			Body: shop.CommandShopSellBody{
				Slot:           1,
				ItemTemplateId: 2000000,
				Quantity:       5,
			},
		}

		handler := handleSellCommand(db)
		handler(logger, ctx, cmd) // Should not panic
	})
}

func TestHandleRechargeCommand(t *testing.T) {
	t.Run("wrong command type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		cmd := shop.Command[shop.CommandShopRechargeBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopBuy, // Wrong type
			Body: shop.CommandShopRechargeBody{
				Slot: 1,
			},
		}

		handler := handleRechargeCommand(nil)
		handler(logger, ctx, cmd) // Should not panic
	})

	t.Run("correct command type processes recharge", func(t *testing.T) {
		setupTestRegistry(t)
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		cmd := shop.Command[shop.CommandShopRechargeBody]{
			CharacterId: 1234,
			Type:        shop.CommandShopRecharge,
			Body: shop.CommandShopRechargeBody{
				Slot: 1,
			},
		}

		handler := handleRechargeCommand(db)
		handler(logger, ctx, cmd) // Should not panic
	})
}
