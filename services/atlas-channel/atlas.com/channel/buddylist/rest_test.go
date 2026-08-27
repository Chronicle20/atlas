package buddylist

import (
	"atlas-channel/buddylist/buddy"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads from RestModel (including the nested
// buddy list) survives a Transform -> Extract round trip.
func TestTransformRoundTrip(t *testing.T) {
	buddies := []buddy.RestModel{
		{
			CharacterId:   200,
			Group:         "Default",
			CharacterName: "Bob",
			ChannelId:     1,
			InShop:        true,
			Pending:       false,
		},
		{
			CharacterId:   300,
			Group:         "Friends",
			CharacterName: "Alice",
			ChannelId:     2,
			InShop:        false,
			Pending:       true,
		},
	}

	rmIn := RestModel{
		Id:          uuid.New(),
		TenantId:    uuid.New(),
		CharacterId: 100,
		Capacity:    20,
		Buddies:     buddies,
	}

	m, err := Extract(rmIn)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
