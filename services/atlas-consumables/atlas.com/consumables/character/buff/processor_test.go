package buff_test

import (
	"atlas-consumables/character/buff"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// buffsDoc is a realistic JSON:API document for GET
// /characters/{characterId}/buffs -- one non-expiring buff carrying an
// UNDEAD stat change (the ZOMBIFY disease) and one ordinary, already-expired
// buff with no changes.
const buffsDoc = `{"data":[` +
	`{"id":"1","type":"buffs","attributes":{"sourceId":9001101,"level":1,"duration":0,"changes":[{"type":"UNDEAD","amount":1}],"createdAt":"2024-01-01T00:00:00Z","expiresAt":"2024-01-01T00:00:00Z","noExpiry":true}},` +
	`{"id":"2","type":"buffs","attributes":{"sourceId":2001002,"level":5,"duration":60000,"changes":[],"createdAt":"2024-01-01T00:00:00Z","expiresAt":"2024-01-01T01:00:00Z","noExpiry":false}}` +
	`],"meta":{"total":2,"page":{"number":1,"size":250,"last":1}}}`

// TestGetByCharacterIdDecodesPopulatedBuffDoc proves the real request path --
// requests.go's characterBuffsUrl through rest.go's Extract -- decodes a
// populated JSON:API buffs document into []Model with correct field mapping,
// and that IsZombified reads the decoded UNDEAD stat change correctly. This
// is the seam model_test.go's hand-built Models cannot exercise: the wire
// payload actually reaching the predicate that gates the ZOMBIFY feature.
func TestGetByCharacterIdDecodesPopulatedBuffDoc(t *testing.T) {
	characterId := uint32(3)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(buffsDoc))
	}))
	defer srv.Close()
	t.Setenv("BUFFS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Fatalf("expected 2 buffs, got %d", len(ms))
	}

	var undead, ordinary *buff.Model
	for i := range ms {
		m := ms[i]
		switch m.SourceId() {
		case 9001101:
			undead = &m
		case 2001002:
			ordinary = &m
		}
	}
	if undead == nil {
		t.Fatal("expected to find the UNDEAD buff (sourceId 9001101)")
	}
	if ordinary == nil {
		t.Fatal("expected to find the ordinary buff (sourceId 2001002)")
	}

	if undead.Level() != 1 {
		t.Fatalf("expected undead buff level 1, got %d", undead.Level())
	}
	if !undead.NoExpiry() {
		t.Fatal("expected undead buff to have noExpiry true")
	}
	if !undead.CreatedAt().Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected createdAt for undead buff: %v", undead.CreatedAt())
	}
	if len(undead.Changes()) != 1 || undead.Changes()[0].Type != "UNDEAD" || undead.Changes()[0].Amount != 1 {
		t.Fatalf("unexpected changes for undead buff: %+v", undead.Changes())
	}

	if ordinary.Level() != 5 {
		t.Fatalf("expected ordinary buff level 5, got %d", ordinary.Level())
	}
	if ordinary.NoExpiry() {
		t.Fatal("expected ordinary buff to have noExpiry false")
	}
	if !ordinary.ExpiresAt().Equal(time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected expiresAt for ordinary buff: %v", ordinary.ExpiresAt())
	}
	if len(ordinary.Changes()) != 0 {
		t.Fatalf("expected ordinary buff to have no changes, got %+v", ordinary.Changes())
	}

	if !buff.IsZombified(ms) {
		t.Fatal("expected IsZombified to be true with the UNDEAD buff present")
	}

	withoutUndead := make([]buff.Model, 0, len(ms))
	for _, m := range ms {
		if m.SourceId() != 9001101 {
			withoutUndead = append(withoutUndead, m)
		}
	}
	if buff.IsZombified(withoutUndead) {
		t.Fatal("expected IsZombified to be false once the UNDEAD buff is removed")
	}
}
