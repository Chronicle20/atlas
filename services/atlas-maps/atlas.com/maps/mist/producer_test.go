package mist

import (
	mistKafka "atlas-maps/kafka/message/mist"
	"encoding/json"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// TestCreatedEventCarriesSkillAndType verifies that createdEventProvider copies
// the source skill id/level and the mist type onto the MIST_CREATED event body.
func TestCreatedEventCarriesSkillAndType(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(0, 0, 100000000).Build()
	m := NewBuilder(uuid.New(), f).
		SetOwner("MONSTER", 42).
		SetSource(2121006, 20). // skill id + level
		SetType(1).             // mist/affected-area type
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDuration(10 * time.Second).
		Build()

	msgs, err := createdEventProvider(tn, m)()
	if err != nil {
		t.Fatal(err)
	}

	var ev mistKafka.Event[mistKafka.CreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Body.SourceSkillId != 2121006 || ev.Body.SourceSkillLevel != 20 || ev.Body.Type != 1 {
		t.Fatalf("body missing skill/type: %+v", ev.Body)
	}
}
