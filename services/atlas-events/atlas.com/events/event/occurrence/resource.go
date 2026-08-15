package occurrence

import (
	"atlas-events/event/transition"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := server.RegisterHandler(l)(si)

			router.HandleFunc("/events/occurrences", registerHandler("get_all_event_occurrences", getAllOccurrencesHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/events/occurrences/{occurrenceId}", registerHandler("get_event_occurrence", getOccurrenceHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/events/worlds/{worldId}/channels/{channelId}/maps/{mapId}/visuals", registerHandler("get_event_visuals_in_map", getVisualsInMapHandler(db))).Methods(http.MethodGet)
		}
	}
}

var errInvalidStartedAtFilter = errors.New("filter[startedAt][from]/filter[startedAt][to] must be RFC3339 timestamps")

// parseFilterInt parses a query-string filter value into an integer domain
// id type (world.Id, channel.Id, _map.Id). server.ParseIntId reads from the
// mux path, not the query string, so filter parsing needs its own thin
// strconv wrapper over the same IntegerId constraint.
func parseFilterInt[T server.IntegerId](raw string) (T, error) {
	v, err := strconv.Atoi(raw)
	if err != nil {
		return T(0), err
	}
	return T(v), nil
}

// parseListFilters resolves the FR-API6 collection filters from the request
// query. It never touches the database — a bad filter value is a 400, not a
// downstream query error.
func parseListFilters(query url.Values) (ListFilters, error) {
	var f ListFilters

	if raw := query.Get("filter[definitionId]"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return ListFilters{}, err
		}
		f.DefinitionId = id
	}

	f.Type = query.Get("filter[type]")
	f.State = query.Get("filter[state]")

	if raw := query.Get("filter[worldId]"); raw != "" {
		id, err := parseFilterInt[world.Id](raw)
		if err != nil {
			return ListFilters{}, err
		}
		f.WorldId = &id
	}

	if raw := query.Get("filter[channelId]"); raw != "" {
		id, err := parseFilterInt[channel.Id](raw)
		if err != nil {
			return ListFilters{}, err
		}
		f.ChannelId = &id
	}

	if raw := query.Get("filter[mapId]"); raw != "" {
		id, err := parseFilterInt[_map.Id](raw)
		if err != nil {
			return ListFilters{}, err
		}
		f.MapId = &id
	}

	if raw := query.Get("filter[voyageId]"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return ListFilters{}, err
		}
		f.VoyageId = id
	}

	if raw := query.Get("filter[startedAt][from]"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return ListFilters{}, errInvalidStartedAtFilter
		}
		f.StartedAtFrom = &t
	}

	if raw := query.Get("filter[startedAt][to]"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return ListFilters{}, errInvalidStartedAtFilter
		}
		f.StartedAtTo = &t
	}

	return f, nil
}

func getAllOccurrencesHandler(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			f, err := parseListFilters(r.URL.Query())
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid filter parameters")
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), db)
			paged, err := p.ListPaged(page, f)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving event occurrences.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rm, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm, paginate.EnvelopeFor(paged), r)
		}
	}
}

func getOccurrenceHandler(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return server.ParseUUIDId(d.Logger(), "occurrenceId", func(occurrenceId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := d.Context()
				scopedDB := db.WithContext(ctx)

				m, err := NewProcessor(d.Logger(), ctx, db).GetById(occurrenceId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving event occurrence [%s].", occurrenceId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// FR-API5: transitions come back as an included relationship.
				transEntities, err := transition.ByOccurrenceProvider(occurrenceId)(scopedDB)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving transitions for event occurrence [%s].", occurrenceId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				transModels, err := model.SliceMap(transition.Make)(model.FixedProvider(transEntities))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating transition model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				transRest, err := model.SliceMap(transition.Transform)(model.FixedProvider(transModels))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating transition REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformWithTransitions(m, transRest)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// getVisualsInMapHandler answers the channel's generic "what visuals are
// active in this map" question (FR-API8). It is a narrow, game-capped
// projection — the collection is bounded by how many events can concurrently
// occupy one map, not by table size — so it pages with paginate.MaxPageSize
// as both default and max, per docs/rest-pagination.md §3's game-capped class.
func getVisualsInMapHandler(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return server.ParseIntId[world.Id](d.Logger(), "worldId", func(worldId world.Id) http.HandlerFunc {
			return server.ParseIntId[channel.Id](d.Logger(), "channelId", func(channelId channel.Id) http.HandlerFunc {
				return server.ParseIntId[_map.Id](d.Logger(), "mapId", func(mapId _map.Id) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
						if err != nil {
							server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
							return
						}

						p := NewProcessor(d.Logger(), d.Context(), db)
						ms, err := p.VisualsInMap(worldId, channelId, mapId)
						if err != nil {
							d.Logger().WithError(err).Errorf("Retrieving event visuals in map [%d].", mapId)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						sorted := make([]Model, len(ms))
						copy(sorted, ms)
						sort.Slice(sorted, func(i, j int) bool { return sorted[i].Id().String() < sorted[j].Id().String() })
						paged := paginate.Slice(sorted, page)

						rm, err := model.SliceMap(TransformVisual)(model.FixedProvider(paged.Items))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalPaginatedResponse[[]VisualRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm, paginate.EnvelopeFor(paged), r)
					}
				})
			})
		})
	}
}
