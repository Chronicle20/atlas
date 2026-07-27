package note_test

import (
	"atlas-channel/note"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// noteDoc renders a JSON:API document for notes [from, to] belonging to a
// single character. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func noteDoc(characterId uint32, from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"notes","attributes":{"characterId":%d,"senderId":1,"message":"note-%d","flag":0,"timestamp":"2024-01-01T00:00:00Z"}}`,
			id, characterId, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestByCharacterProviderDrainsBeyondOnePage proves ByCharacterProvider
// (via requests.DrainProvider) fetches every page of a character's notes
// rather than stopping after the first response. atlas-notes' GET
// /characters/{characterId}/notes is now paginated (task-117); the fixture
// server serves 300 notes across two pages of 250 -- only a genuine drain
// picks up note id 300, which lives on page 2.
func TestByCharacterProviderDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(noteDoc(characterId, 251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(noteDoc(characterId, 1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("NOTES_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := note.NewProcessor(l, ctx).GetByCharacter(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 notes (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("note id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
