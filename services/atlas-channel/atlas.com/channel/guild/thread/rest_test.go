package thread

import (
	"atlas-channel/guild/thread/reply"
	"reflect"
	"testing"
	"time"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. Model's tenantId and guildId are not carried by RestModel and are
// never populated by Extract (see model.go and rest.go), so they are not
// exercised here.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:         1,
		PosterId:   2,
		Title:      "Title",
		Message:    "Message",
		EmoticonId: 3,
		Notice:     true,
		Replies: []reply.RestModel{
			{Id: 4, PosterId: 5, Message: "Reply", CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		},
		CreatedAt: time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC),
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
