package progress_test

import (
	"atlas-npc-conversations/conversation/quest/progress"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// progressDoc renders a JSON:API "progress" document. entries are
// {id, infoNumber, progress} triples, in the order they should appear in the
// response body -- Extract must preserve that order into []Model. meta
// describes the current page/total so requests.DrainProvider can decide
// whether to keep paging.
func progressDoc(entries [][3]string, total, number, size, last int) string {
	var b strings.Builder
	for i, e := range entries {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"progress","attributes":{"infoNumber":%s,"progress":"%s"}}`,
			e[0], e[1], e[2],
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetByCharacterAndQuestRoundTrip drives a representative atlas-quest
// JSON:API payload through GetByCharacterAndQuest into a populated []Model,
// asserting that entries decode in response order and that Progress
// survives as the raw string it arrived as -- never coerced to a number.
func TestGetByCharacterAndQuestRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(progressDoc([][3]string{
			{"1", "0", "007"},
			{"2", "9300285", "Open Sesame"},
			{"3", "5", "1"},
		}, 3, 1, 250, 1)))
	}))
	defer srv.Close()
	t.Setenv("QUEST_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := progress.NewProcessor(l, ctx).GetByCharacterAndQuest(1, 100000)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 3 {
		t.Fatalf("expected 3 progress entries, got %d", len(ms))
	}

	wantInfoNumbers := []uint32{0, 9300285, 5}
	wantProgress := []string{"007", "Open Sesame", "1"}
	for i, m := range ms {
		if m.InfoNumber() != wantInfoNumbers[i] {
			t.Errorf("entry %d: InfoNumber mismatch: got %d want %d", i, m.InfoNumber(), wantInfoNumbers[i])
		}
		if m.Progress() != wantProgress[i] {
			t.Errorf("entry %d: Progress mismatch: got %q want %q", i, m.Progress(), wantProgress[i])
		}
	}
}

// TestGetByCharacterAndQuestNotFound proves a 404 from atlas-quest surfaces
// as progress.ErrNotFound, not the raw requests.ErrNotFound -- callers of
// this package should never need to know it is REST-backed.
func TestGetByCharacterAndQuestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("QUEST_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	_, err = progress.NewProcessor(l, ctx).GetByCharacterAndQuest(1, 100000)()
	if err == nil {
		t.Fatal("expected an error for a 404 response")
	}
	if !errors.Is(err, progress.ErrNotFound) {
		t.Fatalf("expected progress.ErrNotFound, got %v", err)
	}
}
