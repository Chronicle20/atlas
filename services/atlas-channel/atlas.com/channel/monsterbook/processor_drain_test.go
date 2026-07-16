package monsterbook_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// cardDoc renders a JSON:API document for monster-book cards with CardId
// [from, to] belonging to a character. meta describes the current
// page/total so requests.DrainProvider can decide whether to keep paging.
func cardDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"monster-book-card","attributes":{"level":1,"isSpecial":false}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestCardsByCharacterIdProviderDrainsBeyondOnePage proves
// CardsByCharacterIdProvider (via requests.DrainProvider) fetches every
// page of a character's monster-book cards rather than stopping after the
// first response. atlas-monster-book's GET
// /characters/{characterId}/monster-book/cards is now paginated
// (task-117); the fixture server serves 300 cards across two pages of
// 250 -- only a genuine drain picks up cardId 300, which lives on page 2.
// This is the hot game path: MonsterBookDecorator (character/processor.go)
// attaches this full list to the character model sent on every channel
// spawn.
func TestCardsByCharacterIdProviderDrainsBeyondOnePage(t *testing.T) {
	characterId := character.Id(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(cardDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(cardDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	defer monsterbook.SetBaseURLForTest(srv.URL)()

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	cards, err := monsterbook.NewProcessor(l, ctx).GetCardsByCharacterId(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 300 {
		t.Fatalf("expected 300 cards (full drain), got %d; a single-fetch implementation would return 250", len(cards))
	}

	foundLast := false
	for _, c := range cards {
		if c.CardId() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("cardId 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
