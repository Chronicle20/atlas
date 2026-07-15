package pet_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-messages/pet"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// petDoc renders a JSON:API document for pets [from, to] owned by a
// character. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func petDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"pets","attributes":{"cashId":%d,"templateId":5000017,"name":"Pet","level":10,"closeness":0,"fullness":100,"expiration":"2024-01-01T00:00:00Z","ownerId":42,"slot":-1,"x":0,"y":0,"stance":0,"fh":0,"flag":0,"purchaseBy":0}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetPetsDrainsBeyondOnePage proves GetPets (via requests.DrainProvider)
// fetches every page of a character's pets rather than stopping after the
// first response. atlas-pets' GET /characters/{characterId}/pets is now
// paginated (task-117); the fixture server serves 300 pets across two pages
// of 250 -- only a genuine drain picks up pet id 300, which lives on page 2
// (needed by GetPetIdsByName to find every matching pet by name).
func TestGetPetsDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(petDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(petDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("PETS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := pet.NewProcessor(l, ctx).GetPets(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 pets (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("pet id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
