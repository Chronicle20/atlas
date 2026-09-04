package handler

import (
	"atlas-channel/character"
	charmock "atlas-channel/character/mock"
	"atlas-channel/session"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// installRedeemStoredExperienceSeam swaps redeemStoredExperienceProcessorFunc
// for the test, returning a character.Processor backed by
// character/mock.MockProcessor (package-var injection precedent:
// installKarmaCharacterSeam in character_cash_item_use_karma_test.go).
func installRedeemStoredExperienceSeam(t *testing.T) (*[]struct {
	Field       field.Model
	CharacterId uint32
}, func(),
) {
	t.Helper()
	var got []struct {
		Field       field.Model
		CharacterId uint32
	}
	orig := redeemStoredExperienceProcessorFunc
	redeemStoredExperienceProcessorFunc = func(_ logrus.FieldLogger, _ context.Context) character.Processor {
		return &charmock.MockProcessor{
			RedeemStoredExperienceFunc: func(f field.Model, characterId uint32) error {
				got = append(got, struct {
					Field       field.Model
					CharacterId uint32
				}{Field: f, CharacterId: characterId})
				return nil
			},
		}
	}
	return &got, func() {
		redeemStoredExperienceProcessorFunc = orig
	}
}

// newStoredExperienceUseTestSession builds a session for character 1234 in
// world 1 / channel 2 (fixture from the brief).
func newStoredExperienceUseTestSession(t *testing.T, characterId uint32) (session.Model, context.Context, func()) {
	t.Helper()
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	sp := session.NewProcessor(logrus.New(), ctx)
	ch := channel.NewModel(world.Id(1), channel.Id(2))
	sp.Create(ch, 0)(sessionId, discardConn{})
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ctx, func() { session.ClearRegistryForTenant(ten.Id()) }
}

func TestCharacterUseStoredExperienceHandleFunc(t *testing.T) {
	calls, restore := installRedeemStoredExperienceSeam(t)
	defer restore()

	s, ctx, cleanup := newStoredExperienceUseTestSession(t, 1234)
	defer cleanup()

	// StoredExperienceUse's wire body is the tick and nothing else — four
	// bytes, no leading/trailing fields (Task 2 fixture).
	body := []byte{0x0D, 0x0C, 0x0B, 0x0A}
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)

	CharacterUseStoredExperienceHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("RedeemStoredExperience calls = %d, want 1", len(*calls))
	}
	got := (*calls)[0]
	if got.CharacterId != 1234 {
		t.Errorf("characterId = %d, want 1234", got.CharacterId)
	}
	if got.Field.WorldId() != world.Id(1) {
		t.Errorf("field.WorldId() = %d, want 1", got.Field.WorldId())
	}
	if got.Field.ChannelId() != channel.Id(2) {
		t.Errorf("field.ChannelId() = %d, want 2", got.Field.ChannelId())
	}
	if reader.Position() != 4 {
		t.Errorf("bytes consumed = %d, want 4", reader.Position())
	}
}
