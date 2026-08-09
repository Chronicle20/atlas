package mist

import (
	mistKafka "atlas-maps/kafka/message/mist"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// TestCreatedEventProvider_CarriesRenderValues asserts MIST_CREATED carries
// the render values from the model rather than leaving the channel to
// hard-code them (task-200 FR-2.4). Both are 0 for every mist Atlas creates
// today; the plumbing exists so a future mist kind can differ without a
// contract change.
func TestCreatedEventProvider_CarriesRenderValues(t *testing.T) {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()

	// Non-zero values prove the event carries the MODEL's values. A test that
	// only ever asserted 0 would pass against the unchanged code.
	m := NewBuilder(uuid.New(), f).SetRender(7, 3).Build()

	msgs, err := createdEventProvider(tn, m)()
	if err != nil {
		t.Fatalf("createdEventProvider: %v", err)
	}
	var ev mistKafka.Event[mistKafka.CreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if ev.Body.ElemAttr != 7 {
		t.Fatalf("ev.Body.ElemAttr = %d, want 7", ev.Body.ElemAttr)
	}
	if ev.Body.SkillDelay != 3 {
		t.Fatalf("ev.Body.SkillDelay = %d, want 3", ev.Body.SkillDelay)
	}

	// And the default is 0 for every mist Atlas actually creates.
	zero := NewBuilder(uuid.New(), f).Build()
	if zero.ElemAttr() != 0 || zero.SkillDelay() != 0 {
		t.Fatalf("unset render values = (%d,%d), want (0,0)", zero.ElemAttr(), zero.SkillDelay())
	}
}
