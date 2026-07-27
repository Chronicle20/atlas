package list

import (
	"atlas-buddies/buddy"
	list2 "atlas-buddies/kafka/message/list"
	list3 "atlas-buddies/kafka/producer/list"
	"atlas-buddies/rest"
	"errors"
	"net/http"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	GetBuddyList          = "get_buddy_list"
	CreateBuddyList       = "create_buddy_list"
	GetBuddiesInBuddyList = "get_buddies_in_buddy_list"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters/{characterId}/buddy-list").Subrouter()
			r.HandleFunc("", registerGet(GetBuddyList, handleGetBuddyList(db))).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)(CreateBuddyList, handleCreateBuddyList)).Methods(http.MethodPost)
			r.HandleFunc("/buddies", registerGet(GetBuddiesInBuddyList, handleGetBuddiesInBuddyList(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetBuddyList(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				bl, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterId(characterId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(bl))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
			}
		})
	}
}

func handleCreateBuddyList(d *rest.HandlerDependency, _ *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := producer.ProviderImpl(d.Logger())(d.Context())(list2.EnvCommandTopic)(list3.CreateCommandProvider(character.Id(characterId), i.Capacity))
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func handleGetBuddiesInBuddyList(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				bl, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterId(characterId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// GetByCharacterId preloads Buddies via GORM without an explicit
				// ORDER BY, so its row order is not guaranteed stable across
				// calls. Sort by CharacterId (unique within a single buddy list)
				// before slicing so pagination is deterministic (task-117
				// determinism requirement).
				buddies := make([]buddy.Model, len(bl.Buddies()))
				copy(buddies, bl.Buddies())
				sort.SliceStable(buddies, func(i, j int) bool {
					return buddies[i].CharacterId() < buddies[j].CharacterId()
				})

				paged := paginate.Slice(buddies, page)

				res, err := model.SliceMap(buddy.Transform)(model.FixedProvider(paged.Items))()()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				server.MarshalPaginatedResponse[[]buddy.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}
