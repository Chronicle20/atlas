package compartment

import (
	"atlas-inventory/rest"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// AccommodationInputRestModel is the POST body for
// /characters/{characterId}/inventory/accommodation: the set of items a caller
// wants to know it could grant. Each item is evaluated independently.
type AccommodationInputRestModel struct {
	Id    string                  `json:"-"`
	Items []AccommodationItemRest `json:"items"`
}

type AccommodationItemRest struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

func (AccommodationInputRestModel) GetName() string                          { return "inventoryAccommodations" }
func (m AccommodationInputRestModel) GetID() string                          { return m.Id }
func (m *AccommodationInputRestModel) SetID(id string) error                 { m.Id = id; return nil }
func (m *AccommodationInputRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (m *AccommodationInputRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// AccommodationOutputRestModel reports per-item verdicts plus an overall flag
// (true only when every requested item is independently accommodatable).
type AccommodationOutputRestModel struct {
	Id           string                    `json:"-"`
	Accommodated bool                      `json:"accommodated"`
	Results      []AccommodationResultRest `json:"results"`
}

type AccommodationResultRest struct {
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Accommodated bool   `json:"accommodated"`
}

func (AccommodationOutputRestModel) GetName() string                          { return "inventoryAccommodations" }
func (m AccommodationOutputRestModel) GetID() string                          { return m.Id }
func (m *AccommodationOutputRestModel) SetID(id string) error                 { m.Id = id; return nil }
func (m *AccommodationOutputRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (m *AccommodationOutputRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func handleCheckAccommodation(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, input AccommodationInputRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input AccommodationInputRestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			characterIdStr := mux.Vars(r)["characterId"]
			characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
			if err != nil {
				d.Logger().WithError(err).Errorf("Invalid characterId [%s].", characterIdStr)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			reqs := make([]AccommodationRequest, 0, len(input.Items))
			for _, it := range input.Items {
				reqs = append(reqs, AccommodationRequest{TemplateId: it.ItemId, Quantity: it.Quantity})
			}

			results, err := NewProcessor(d.Logger(), d.Context(), db).CanAccommodate(uint32(characterId), reqs)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to evaluate inventory accommodation for character [%d].", characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			all := true
			rr := make([]AccommodationResultRest, 0, len(results))
			for _, res := range results {
				if !res.Accommodated {
					all = false
				}
				rr = append(rr, AccommodationResultRest{ItemId: res.TemplateId, Quantity: res.Quantity, Accommodated: res.Accommodated})
			}
			out := AccommodationOutputRestModel{Id: characterIdStr, Accommodated: all, Results: rr}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[AccommodationOutputRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(out)
		}
	}
}
