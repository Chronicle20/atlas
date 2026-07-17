package handler

import (
	"context"
	"testing"

	"atlas-channel/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
	"io"
)

func mapTestLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func mkFullChar(id uint32, hp, maxHp, mp, maxMp uint16) character.Model {
	return character.NewModelBuilder().
		SetId(id).SetHp(hp).SetMaxHp(maxHp).SetMp(mp).SetMaxMp(maxMp).MustBuild()
}

func TestSelectAllCharactersInMap(t *testing.T) {
	prevInMap := inMapCharacterIdsFunc
	prevMember := loadPartyMemberFunc
	t.Cleanup(func() {
		inMapCharacterIdsFunc = prevInMap
		loadPartyMemberFunc = prevMember
	})

	inMapCharacterIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model) map[uint32]struct{} {
		return map[uint32]struct{}{1: {}, 2: {}, 3: {}}
	}
	members := map[uint32]character.Model{
		1: mkFullChar(1, 100, 500, 20, 200),
		2: mkFullChar(2, 0, 800, 0, 300), // HP 0 is NOT filtered by the map-wide selector
		// id 3 intentionally absent -> load error -> skipped
	}
	loadPartyMemberFunc = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		mc, ok := members[id]
		if !ok {
			return character.Model{}, errFakeNotFound
		}
		return mc, nil
	}

	got := SelectAllCharactersInMap(mapTestLogger(), context.Background(), field.NewBuilder(0, 0, 100000000).Build())

	ids := recipientIds(got) // helper already defined in recipients_test.go
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("recipient ids = %v, want [1 2] (id 3 skipped on load error, id 2 kept despite HP 0)", ids)
	}
	// Verify the MP snapshot flows through for a known recipient.
	for _, r := range got {
		if r.Id() == 1 {
			if r.Hp() != 100 || r.MaxHp() != 500 || r.Mp() != 20 || r.MaxMp() != 200 {
				t.Errorf("recipient 1 snapshot = hp %d/%d mp %d/%d, want 100/500 20/200",
					r.Hp(), r.MaxHp(), r.Mp(), r.MaxMp())
			}
		}
	}
}

var errFakeNotFound = &fakeErr{}

type fakeErr struct{}

func (*fakeErr) Error() string { return "not found" }
