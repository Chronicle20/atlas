package character

import (
	"atlas-npc/kafka/message/character"
	"atlas-npc/shops"
	"atlas-npc/test"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	shops.InitRegistry(client)
}

func TestHandleStatusEventLogout(t *testing.T) {
	t.Run("wrong event type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		// Create event with wrong type
		event := character.StatusEvent[character.StatusEventLogoutBody]{
			WorldId:     0,
			CharacterId: 1234,
			Type:        character.StatusEventTypeMapChanged, // Wrong type
			Body: character.StatusEventLogoutBody{
				ChannelId: 1,
				MapId:     100000000,
			},
		}

		// Handler should return early without error (no processor call)
		handler := handleStatusEventLogout(nil)
		handler(logger, ctx, event) // Should not panic
	})

	t.Run("correct event type processes logout", func(t *testing.T) {
		setupTestRegistry(t)
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		event := character.StatusEvent[character.StatusEventLogoutBody]{
			WorldId:     0,
			CharacterId: 1234,
			Type:        character.StatusEventTypeLogout,
			Body: character.StatusEventLogoutBody{
				ChannelId: 1,
				MapId:     100000000,
			},
		}

		handler := handleStatusEventLogout(db)
		handler(logger, ctx, event) // Should not panic
	})
}

func TestHandleStatusEventMapChanged(t *testing.T) {
	t.Run("wrong event type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		event := character.StatusEvent[character.StatusEventMapChangedBody]{
			WorldId:     0,
			CharacterId: 1234,
			Type:        character.StatusEventTypeLogout, // Wrong type
			Body: character.StatusEventMapChangedBody{
				ChannelId:      1,
				OldMapId:       100000000,
				TargetMapId:    100000001,
				TargetPortalId: 0,
			},
		}

		handler := handleStatusEventMapChanged(nil)
		handler(logger, ctx, event) // Should not panic
	})

	t.Run("correct event type processes map change", func(t *testing.T) {
		setupTestRegistry(t)
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		event := character.StatusEvent[character.StatusEventMapChangedBody]{
			WorldId:     0,
			CharacterId: 1234,
			Type:        character.StatusEventTypeMapChanged,
			Body: character.StatusEventMapChangedBody{
				ChannelId:      1,
				OldMapId:       100000000,
				TargetMapId:    100000001,
				TargetPortalId: 0,
			},
		}

		handler := handleStatusEventMapChanged(db)
		handler(logger, ctx, event) // Should not panic
	})
}

func TestHandleStatusEventChannelChanged(t *testing.T) {
	t.Run("wrong event type does nothing", func(t *testing.T) {
		logger, _ := logtest.NewNullLogger()
		ctx := test.CreateTestContext()

		event := character.StatusEvent[character.ChangeChannelEventLoginBody]{
			WorldId:     0,
			CharacterId: 1234,
			Type:        character.StatusEventTypeLogout, // Wrong type
			Body: character.ChangeChannelEventLoginBody{
				ChannelId:    2,
				OldChannelId: 1,
				MapId:        100000000,
			},
		}

		handler := handleStatusEventChannelChanged(nil)
		handler(logger, ctx, event) // Should not panic
	})

	t.Run("correct event type processes channel change", func(t *testing.T) {
		setupTestRegistry(t)
		_, db, cleanup := test.CreateShopsProcessor(t)
		defer cleanup()

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		ctx := test.CreateTestContext()

		event := character.StatusEvent[character.ChangeChannelEventLoginBody]{
			WorldId:     0,
			CharacterId: 1234,
			Type:        character.StatusEventTypeChannelChanged,
			Body: character.ChangeChannelEventLoginBody{
				ChannelId:    2,
				OldChannelId: 1,
				MapId:        100000000,
			},
		}

		handler := handleStatusEventChannelChanged(db)
		handler(logger, ctx, event) // Should not panic
	})
}
