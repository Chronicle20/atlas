package skill_test

import (
	"atlas-query-aggregator/skill"
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

// skillDoc renders a JSON:API document for skills with id [from, to]
// belonging to a character. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func skillDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"skills","attributes":{"level":1,"masterLevel":0}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetSkillsByCharacterDrainsBeyondOnePage proves GetSkillsByCharacter
// (via requests.DrainProvider) fetches every page of a character's skills
// rather than stopping after the first response. atlas-skills' GET
// /characters/{characterId}/skills is now paginated (task-117); this was a
// same-class gap to atlas-query-aggregator's quest drain conversion, caught
// by task-117's task-29 acceptance sweep (grep 3). The fixture server
// serves 300 skills across two pages of 250 -- only a genuine drain picks
// up skill id 300, which lives on page 2.
func TestGetSkillsByCharacterDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(skillDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(skillDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("SKILLS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := skill.NewProcessor(l, ctx).GetSkillsByCharacter(characterId)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 skills (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("skill id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
