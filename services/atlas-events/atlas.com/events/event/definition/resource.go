package definition

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// EnabledOrchestrator, when non-nil, replaces the plain toggle-only
// Processor.SetEnabled for the PATCH handler's write, so a false->true
// transition can also schedule the FR-A2 TRIGGER_EVALUATION. event/definition
// cannot import event/scheduling (or a package that does) directly —
// event/scheduling already imports event/definition to resolve a claimed
// row's definition, so the reverse import would cycle (task-231 R33-3).
// main.go wires this to event/orchestration.SetEnabled at startup; leaving it
// nil (this package's own tests) exercises only the FR-D5 toggle.
var EnabledOrchestrator func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) func(id uuid.UUID, enabled bool) (Model, error)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := server.RegisterHandler(l)(si)
			registerInputHandler := server.RegisterInputHandler[PatchInput](l)(si)

			router.HandleFunc("/events/definitions", registerHandler("get_all_event_definitions", getAllDefinitionsHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/events/definitions/{definitionId}", registerHandler("get_event_definition", getDefinitionHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/events/definitions/{definitionId}", registerInputHandler("update_event_definition", updateDefinitionHandler(db))).Methods(http.MethodPatch)
		}
	}
}

// pagedDefinitions resolves the GET-collection query into a single
// model.Paged[Model], regardless of which filters were supplied. Unfiltered
// listings page in SQL via Processor.GetAllPaged; a filter[type] listing
// materializes that (naturally small — one row per registered event type in
// the common case) type's rows and pages them in Go with paginate.Slice, per
// docs/rest-pagination.md's adapter-choice rule for aggregations with no
// single scoped query. filter[enabled] narrows a filter[type] listing further
// and is rejected on its own — there is no unscoped "all enabled" listing.
func pagedDefinitions(p Processor, page model.Page, typeFilter string, enabledFilter string) (model.Paged[Model], error) {
	if typeFilter == "" {
		if enabledFilter != "" {
			return model.Paged[Model]{}, errFilterEnabledRequiresType
		}
		return p.GetAllPaged(page)
	}

	all, err := p.GetByType(typeFilter)
	if err != nil {
		return model.Paged[Model]{}, err
	}

	if enabledFilter != "" {
		want := enabledFilter == "true"
		filtered := make([]Model, 0, len(all))
		for _, m := range all {
			if m.Enabled() == want {
				filtered = append(filtered, m)
			}
		}
		all = filtered
	}

	return paginate.Slice(all, page), nil
}

var errFilterEnabledRequiresType = errors.New("filter[enabled] requires filter[type]")

func getAllDefinitionsHandler(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			query := r.URL.Query()
			p := NewProcessor(d.Logger(), d.Context(), db)

			paged, err := pagedDefinitions(p, page, query.Get("filter[type]"), query.Get("filter[enabled]"))
			if errors.Is(err, errFilterEnabledRequiresType) {
				server.WriteBadRequest(d.Logger(), w, errFilterEnabledRequiresType.Error())
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving event definitions.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rm, err := model.SliceMap(func(m Model) (RestModel, error) { return Transform(d.Context(), m) })(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm, paginate.EnvelopeFor(paged), r)
		}
	}
}

func getDefinitionHandler(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return server.ParseUUIDId(d.Logger(), "definitionId", func(definitionId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetById(definitionId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving event definition [%s].", definitionId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := Transform(d.Context(), m)
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

// updateDefinitionHandler enforces FR-API2 (only the enabled attribute may
// ever be changed through this route) via PatchInput.UnmarshalJSON rather
// than hand-parsing the JSON:API envelope — server.ParseInput decodes the
// body into PatchInput and rejects a malformed/disallowed attribute set with
// a 400 before this handler ever runs.
func updateDefinitionHandler(db *gorm.DB) server.InputHandler[PatchInput] {
	return func(d *server.HandlerDependency, c *server.HandlerContext, input PatchInput) http.HandlerFunc {
		return server.ParseUUIDId(d.Logger(), "definitionId", func(definitionId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				setEnabled := NewProcessor(d.Logger(), d.Context(), db).SetEnabled
				if EnabledOrchestrator != nil {
					setEnabled = EnabledOrchestrator(d.Logger(), d.Context(), db)
				}
				updated, err := setEnabled(definitionId, input.Enabled)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Updating event definition [%s].", definitionId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := Transform(d.Context(), updated)
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
