package quest

import (
	"atlas-npc-conversations/rest"
	"net/http"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
)

type SeedStatusRestModel struct {
	Id                string  `json:"-"`
	ConversationCount int64   `json:"conversationCount"`
	UpdatedAt         *string `json:"updatedAt"`
}

func (r SeedStatusRestModel) GetName() string                                   { return "questConversationsSeedStatus" }
func (r SeedStatusRestModel) GetID() string                                     { return r.Id }
func (r *SeedStatusRestModel) SetID(id string) error                            { r.Id = id; return nil }
func (r SeedStatusRestModel) GetReferences() []jsonapi.Reference                { return []jsonapi.Reference{} }
func (r SeedStatusRestModel) GetReferencedIDs() []jsonapi.ReferenceID           { return []jsonapi.ReferenceID{} }
func (r SeedStatusRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier { return []jsonapi.MarshalIdentifier{} }
func (r *SeedStatusRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *SeedStatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r *SeedStatusRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// SeedStatusHandler handles GET /quests/conversations/seed/status
func SeedStatusHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := d.Logger()
		t := tenant.MustFromContext(d.Context())
		count, updated, err := NewProcessor(l, d.Context(), d.DB()).Count()
		if err != nil {
			l.WithError(err).Errorf("Unable to read quest conversations seed status.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		res := SeedStatusRestModel{
			Id:                t.Id().String(),
			ConversationCount: count,
		}
		if updated != nil {
			s := updated.UTC().Format(time.RFC3339)
			res.UpdatedAt = &s
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[SeedStatusRestModel](l)(w)(c.ServerInformation())(queryParams)(res)
	}
}
