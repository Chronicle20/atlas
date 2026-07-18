package character

import (
	"atlas-keys/key"
	"atlas-keys/rest"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	GetKeyMap   = "get_key_map"
	SetKey      = "set_key"
	ResetKeyMap = "reset_key_map"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("/{characterId}/keys", registerGet(GetKeyMap, handleGetKeyMap(db))).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}/keys", rest.RegisterHandler(l)(si)(ResetKeyMap, handleDeleteKeyMap(db))).Methods(http.MethodDelete)
			r.HandleFunc("/{characterId}/keys/{keyId}", rest.RegisterInputHandler[key.RestModel](l)(si)(SetKey, handleSetKey(db))).Methods(http.MethodPatch)
		}
	}
}

func handleSetKey(db *gorm.DB) rest.InputHandler[key.RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i key.RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseKeyId(d.Logger(), func(keyId int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := key.NewProcessor(d.Logger(), d.Context(), db)
					err := processor.ChangeKey(uuid.New(), characterId, keyId, i.Type, i.Action)
					if err != nil {
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			})
		})
	}
}

func handleDeleteKeyMap(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := key.NewProcessor(d.Logger(), d.Context(), db)
				err := processor.Reset(uuid.New(), characterId)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusOK)
			}
		})
	}
}

func handleGetKeyMap(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				processor := key.NewProcessor(d.Logger(), d.Context(), db)
				ks, err := processor.GetByCharacterId(characterId)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// The keys table has a composite primary key (character_id, key),
				// neither column auto-incrementing, so database.PagedQuery cannot
				// derive a single ORDER BY column (GORM's PrioritizedPrimaryField is
				// nil for a true composite key). The keymap is bounded (~90 rows,
				// far under the 250 cap) so it is cheap to materialize in full via
				// the existing GetByCharacterId, sort deterministically by Key
				// (unique within a character), then paginate.Slice in-process.
				sorted := make([]key.Model, len(ks))
				copy(sorted, ks)
				sort.SliceStable(sorted, func(i, j int) bool {
					return sorted[i].Key() < sorted[j].Key()
				})

				paged := paginate.Slice(sorted, page)

				res, err := model.SliceMap(key.Transform)(model.FixedProvider(paged.Items))()()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				server.MarshalPaginatedResponse[[]key.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}
