package character_test

import (
	"atlas-asset-expiration/cashshop"
	"atlas-asset-expiration/character"
	"atlas-asset-expiration/inventory"
	"atlas-asset-expiration/storage"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// recorder captures every (topic, message) pair CheckAndExpire emits.
type recorder struct {
	mu     sync.Mutex
	topics []string
}

func (r *recorder) provider() producer.Provider {
	return func(token string) producer.MessageProducer {
		return func(p model.Provider[[]kafka.Message]) error {
			msgs, err := p()
			if err != nil {
				return err
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			for range msgs {
				r.topics = append(r.topics, token)
			}
			return nil
		}
	}
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.topics)
}

var past = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// withPageMeta injects a single-page JSON:API pagination envelope into a
// document produced by jsonapi.Marshal, matching what requests.PagedGetRequest
// expects alongside the "data" array.
func withPageMeta(t *testing.T, body []byte, total int) []byte {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("failed to inject page meta: %v", err)
	}
	m["meta"] = map[string]interface{}{
		"total": total,
		"page":  map[string]interface{}{"number": 1, "size": 250, "last": 1},
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal page meta envelope: %v", err)
	}
	return out
}

func mustMarshal(t *testing.T, data interface{}) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(data)
	if err != nil {
		t.Fatalf("jsonapi.Marshal failed: %v", err)
	}
	return b
}

const compartmentId = "11111111-1111-1111-1111-111111111111"

// stubs serves the four upstream reads CheckAndExpire performs, all via real
// JSON:API documents built from the same REST models the production clients
// decode, so a wiring mistake in the fixture (wrong relationship, wrong
// included type) fails the same way a real service response would. Every
// asset returned is already expired; templateId is the only variable under
// test.
func stubs(t *testing.T, characterId, accountId uint32, templateId uint32) {
	t.Helper()

	inventoryDoc := mustMarshal(t, inventory.RestModel{
		Id:          "1",
		CharacterId: characterId,
		Compartments: []inventory.CompartmentRestModel{
			{Id: compartmentId, Type: inventory.CompartmentTypeCash, Capacity: 24},
		},
	})
	compartmentAssetsDoc := withPageMeta(t, mustMarshal(t, []inventory.AssetRestModel{
		{Id: "101", TemplateId: templateId, Slot: 1, Expiration: past},
	}), 1)
	storageAssetsDoc := withPageMeta(t, mustMarshal(t, []storage.AssetRestModel{
		{Id: "201", TemplateId: templateId, Slot: 1, Expiration: past},
	}), 1)
	cashCompartmentsDoc := mustMarshal(t, []cashshop.CompartmentRestModel{
		{
			Id:        "2",
			AccountId: accountId,
			Type:      5,
			Capacity:  24,
			Assets: []cashshop.AssetRestModel{
				{
					Id:            "301",
					CompartmentId: "2",
					Item: cashshop.ItemRestModel{
						Id:          "301",
						CashId:      1,
						TemplateId:  templateId,
						CommodityId: 1,
						Quantity:    1,
						PurchasedBy: accountId,
						Expiration:  past,
						CreatedAt:   past,
					},
				},
			},
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		path := r.URL.Path
		switch {
		case strings.Contains(path, "/inventory/compartments/") && strings.HasSuffix(path, "/assets"):
			_, _ = w.Write(compartmentAssetsDoc)
		case strings.HasSuffix(path, "/inventory"):
			_, _ = w.Write(inventoryDoc)
		case strings.HasSuffix(path, "/storage/accounts/"+itoa(accountId)+"/assets"):
			_, _ = w.Write(storageAssetsDoc)
		case strings.HasSuffix(path, "/cash-shop/inventory/compartments"):
			_, _ = w.Write(cashCompartmentsDoc)
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	for _, env := range []string{"INVENTORY_SERVICE_URL", "STORAGE_SERVICE_URL", "CASHSHOP_SERVICE_URL", "DATA_SERVICE_URL"} {
		t.Setenv(env, srv.URL+"/")
	}
}

func itoa(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}

func TestCheckAndExpireEmitsForExpiredNonPet(t *testing.T) {
	stubs(t, 42, 7, 2000000)
	r := &recorder{}
	l, _ := test.NewNullLogger()
	character.NewProcessor(l, context.Background()).CheckAndExpire(r.provider())(42, 7, 0)
	if r.count() == 0 {
		t.Fatal("expected expire commands for an expired consumable, got none")
	}
}

// TestCheckAndExpireEmitsForExpiredNonPetRequiresStubs proves stubs actually
// drives the sweep: without it, every upstream read falls through to a
// connection error and the sweep finds nothing, which would make
// TestCheckAndExpireEmitsNothingForExpiredPet pass vacuously. If this test
// ever starts passing, the pet-exemption regression test above is no longer
// meaningful.
func TestCheckAndExpireEmitsForExpiredNonPetRequiresStubs(t *testing.T) {
	for _, env := range []string{"INVENTORY_SERVICE_URL", "STORAGE_SERVICE_URL", "CASHSHOP_SERVICE_URL", "DATA_SERVICE_URL"} {
		t.Setenv(env, "http://127.0.0.1:1/")
	}
	r := &recorder{}
	l, _ := test.NewNullLogger()
	character.NewProcessor(l, context.Background()).CheckAndExpire(r.provider())(42, 7, 0)
	if r.count() != 0 {
		t.Fatalf("expected no expire commands without stubs (unreachable upstream), got %d", r.count())
	}
}

func TestCheckAndExpireEmitsNothingForExpiredPet(t *testing.T) {
	stubs(t, 42, 7, 5000000)
	r := &recorder{}
	l, _ := test.NewNullLogger()
	character.NewProcessor(l, context.Background()).CheckAndExpire(r.provider())(42, 7, 0)
	if r.count() != 0 {
		t.Fatalf("expected no expire commands for an expired pet, got %d on topics %v", r.count(), r.topics)
	}
}
