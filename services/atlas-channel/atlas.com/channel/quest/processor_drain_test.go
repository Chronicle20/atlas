package quest_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/quest"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// questDoc renders a JSON:API document for quests [from, to] belonging to a
// character. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func questDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"quest-status","attributes":{"characterId":42,"questId":%d,"state":1,"startedAt":"2024-01-01T00:00:00Z","completedCount":0,"forfeitCount":0,"progress":[]}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestByCharacterIdProviderDrainsBeyondOnePage proves ByCharacterIdProvider
// (via requests.DrainProvider) fetches every page of a character's quests
// rather than stopping after the first response. atlas-quest's GET
// /characters/{characterId}/quests is now paginated (task-117); the
// fixture server serves 300 quests across two pages of 250 -- only a
// genuine drain picks up quest id 300, which lives on page 2. This is the
// hot game path: QuestModelDecorator (character/processor.go) attaches this
// full list to the character model sent on every channel spawn.
func TestByCharacterIdProviderDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(questDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(questDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := quest.NewProcessor(l, ctx).GetByCharacterId(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 quests (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("quest id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
