package character_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-login/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// characterDoc renders a JSON:API document for characters [from, to]
// belonging to one account/world. Each character's name is unique so the
// test can assert presence of a page-2-only item.
func characterDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"characters","attributes":{"accountId":7,"worldId":0,"name":"char%d","level":10}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestByAccountAndWorldProviderDrainsBeyondOnePage proves
// character.Processor.ByAccountAndWorldProvider (via requests.DrainProvider)
// fetches every page of an account's characters in a world rather than
// stopping after the first response. atlas-character's GET
// /characters?accountId=&worldId= is now paginated (task-117); the
// character-select screen is a genuine consumer that must see every
// character, not just page 1. The fixture server serves 260 characters
// across two pages of 250 -- only a genuine drain picks up the character
// that lives on page 2.
func TestByAccountAndWorldProviderDrainsBeyondOnePage(t *testing.T) {
	accountId := uint32(7)
	worldId := world.Id(0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(characterDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(characterDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	cs, err := character.NewProcessor(l, ctx).GetForWorld()(accountId, worldId)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 260 {
		t.Fatalf("expected 260 characters (full drain), got %d; a single-fetch implementation would return 250", len(cs))
	}

	foundLast := false
	for _, c := range cs {
		if c.Name() == "char260" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("character \"char260\" (page 2) must be present; single-fetch impl would miss it")
	}
}
