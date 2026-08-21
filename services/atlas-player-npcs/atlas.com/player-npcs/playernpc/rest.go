package playernpc

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	"atlas-player-npcs/eligibility"
	"atlas-player-npcs/inventory"
	"atlas-player-npcs/ranking"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	mapdata "atlas-player-npcs/data/map"
	npcdata "atlas-player-npcs/data/npc"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// -- handler plumbing ------------------------------------------------------
//
// Ported from services/atlas-notes/atlas.com/notes/note/rest.go's
// neighbouring `rest` package: this service has no shared `rest` package of
// its own yet, so the boilerplate lives directly in this file rather than
// introducing a fourth new package for a single resource.

// HandlerDependency carries the per-request logger, database handle and
// tenant-scoped context into a handler.
type HandlerDependency struct {
	l   logrus.FieldLogger
	db  *gorm.DB
	ctx context.Context
}

func (h HandlerDependency) Logger() logrus.FieldLogger { return h.l }
func (h HandlerDependency) DB() *gorm.DB               { return h.db }
func (h HandlerDependency) Context() context.Context   { return h.ctx }

// HandlerContext carries the JSON:API server information a response needs.
type HandlerContext struct {
	si jsonapi.ServerInformation
}

func (h HandlerContext) ServerInformation() jsonapi.ServerInformation { return h.si }

type GetHandler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

type InputHandler[M any] func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc

func parseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m M

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() { _ = r.Body.Close() }()

		if err := jsonapi.Unmarshal(body, &m); err != nil {
			d.l.WithError(err).Errorln("Deserializing input.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(d, c, m)(w, r)
	}
}

func registerHandler(l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
			return func(handlerName string, handler GetHandler) http.HandlerFunc {
				return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
						return handler(&HandlerDependency{l: tl, db: db, ctx: tctx}, &HandlerContext{si: si})
					})
				})
			}
		}
	}
}

func registerInputHandler[M any](l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
				return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
						return parseInput[M](&HandlerDependency{l: tl, db: db, ctx: tctx}, &HandlerContext{si: si}, handler)
					})
				})
			}
		}
	}
}

// -- error responses --------------------------------------------------------

type errorObject struct {
	Status string `json:"status"`
	Code   string `json:"code,omitempty"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

// writeError writes a JSON:API errors-array response carrying `code` --
// the design §8.3 failure code REST callers (and the GM command, design
// §9.2) branch on.
func writeError(l logrus.FieldLogger, w http.ResponseWriter, status int, code string, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	doc := struct {
		Errors []errorObject `json:"errors"`
	}{Errors: []errorObject{{Status: strconv.Itoa(status), Code: code, Title: http.StatusText(status), Detail: detail}}}
	if err := json.NewEncoder(w).Encode(doc); err != nil {
		l.WithError(err).Errorf("Writing error response.")
	}
}

// writeDeployError maps a Deploy/Redeploy/Remove failure to its PRD §5
// status: the four design §8.3 codes are 409, an unresolvable character or
// map (requests.ErrNotFound) is 422, everything else is the generic error
// response.
func writeDeployError(d *HandlerDependency, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, requests.ErrNotFound):
		writeError(d.Logger(), w, http.StatusUnprocessableEntity, "unresolvable", err.Error())
	case errors.Is(err, ErrDuplicate):
		writeError(d.Logger(), w, http.StatusConflict, CodeDuplicate, err.Error())
	case errors.Is(err, ErrPoolExhausted):
		writeError(d.Logger(), w, http.StatusConflict, CodePoolExhausted, err.Error())
	case errors.Is(err, ErrMapFull):
		writeError(d.Logger(), w, http.StatusConflict, CodeMapFull, err.Error())
	case errors.Is(err, ErrIneligible):
		writeError(d.Logger(), w, http.StatusConflict, CodeIneligible, err.Error())
	default:
		server.WriteErrorResponse(d.Logger())(w)(err)
	}
}

// -- routes ------------------------------------------------------------------

const idPathVar = "id"

// InitializeRoutes registers the PRD §5 surface under PathPrefix
// "/player-npcs". "/eligibility" is registered before "/{id}" so it is not
// shadowed by the id pattern -- the same ordering hazard
// services/atlas-tenants/atlas.com/tenants/configuration/resource.go:1552-1554
// documents.
func InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := registerHandler(l)(db)(si)
			registerDeploy := registerInputHandler[DeployRestModel](l)(db)(si)

			r := router.PathPrefix("/player-npcs").Subrouter()
			r.HandleFunc("", registerGet("get_player_npcs", handleGetPlayerNpcs)).Methods(http.MethodGet)
			r.HandleFunc("", registerDeploy("deploy_player_npc", handleDeployPlayerNpc)).Methods(http.MethodPost)
			r.HandleFunc("", registerGet("delete_player_npcs_by_character", handleDeletePlayerNpcsByCharacter)).Methods(http.MethodDelete)

			// Must precede "/{id}" -- see the doc comment above.
			r.HandleFunc("/eligibility", registerGet("get_player_npc_eligibility", handleGetEligibility)).Methods(http.MethodGet)

			r.HandleFunc("/{"+idPathVar+"}", registerGet("get_player_npc", handleGetPlayerNpc)).Methods(http.MethodGet)
			r.HandleFunc("/{"+idPathVar+"}", registerGet("redeploy_player_npc", handleRedeployPlayerNpc)).Methods(http.MethodPatch)
			r.HandleFunc("/{"+idPathVar+"}", registerGet("delete_player_npc", handleDeletePlayerNpc)).Methods(http.MethodDelete)
		}
	}
}

// processorFor constructs a Processor with the real HTTP-backed read
// clients, per processor.go's NewProcessor doc: "Task 16/17/21's wiring
// constructs the real HTTP-backed processors with NewProcessor(l, ctx) from
// each package and passes them here." emit is nil (a no-op) here -- Task
// 17 wires the Kafka-backed EventEmitter.
func processorFor(d *HandlerDependency) Processor {
	l, ctx, db := d.Logger(), d.Context(), d.DB()
	return NewProcessor(l, ctx, db,
		character.NewProcessor(l, ctx),
		inventory.NewProcessor(l, ctx),
		ranking.NewProcessor(l, ctx),
		configuration.NewProcessor(l, ctx),
		npcdata.NewProcessor(l, ctx),
		mapdata.NewProcessor(l, ctx),
		nil,
	)
}

// writePlayerNpc transforms m and writes it as a JSON:API document with
// the given status code.
func writePlayerNpc(d *HandlerDependency, c *HandlerContext, w http.ResponseWriter, r *http.Request, m Model, status int) {
	rm, err := model.Map(Transform)(model.FixedProvider(m))()
	if err != nil {
		d.Logger().WithError(err).Errorf("Creating REST model.")
		server.WriteErrorResponse(d.Logger())(w)(err)
		return
	}
	query := r.URL.Query()
	queryParams := jsonapi.ParseQueryFields(&query)
	w.WriteHeader(status)
	server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
}

// parseListFilters reads filter[worldId]/filter[mapId] off the query
// string, PRD §5's list filters. Both default to 0 when absent, matching
// filter[worldId]=0's example in PRD §5 and covering a caller that only
// needs pagination.
func parseListFilters(r *http.Request) (world.Id, _map.Id, error) {
	q := r.URL.Query()

	worldId := world.Id(0)
	if v := q.Get("filter[worldId]"); v != "" {
		n, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return 0, 0, errors.New("filter[worldId] must be a valid world id")
		}
		worldId = world.Id(n)
	}

	mapId := _map.Id(0)
	if v := q.Get("filter[mapId]"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, 0, errors.New("filter[mapId] must be a valid map id")
		}
		mapId = _map.Id(n)
	}

	return worldId, mapId, nil
}

// pagedPlayerNpcsByMap is playerNpcsByMap (administrator.go), but keeping
// the provider's Total/Page so the list endpoint can build a real
// pagination envelope -- Processor.GetByMap only returns []Model (its
// signature is shared with Task 17/21/19 and is not REST's to reshape).
func pagedPlayerNpcsByMap(db *gorm.DB, worldId byte, mapId uint32, page model.Page) (model.Paged[Model], error) {
	entPaged, err := getByMapPagedProvider(worldId, mapId, page)(db)()
	if err != nil {
		return model.Paged[Model]{}, err
	}
	models := make([]Model, 0, len(entPaged.Items))
	for _, e := range entPaged.Items {
		m, err := getPlayerNpcModel(db, e.Id)
		if err != nil {
			return model.Paged[Model]{}, err
		}
		models = append(models, m)
	}
	return model.Paged[Model]{Items: models, Total: entPaged.Total, Page: entPaged.Page}, nil
}

// handleGetPlayerNpcs handles GET /player-npcs -- the map-enter hot path
// (PRD §5), tenant-scoped and paginated.
func handleGetPlayerNpcs(d *HandlerDependency, c *HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		worldId, mapId, err := parseListFilters(r)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, err.Error())
			return
		}
		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		paged, err := pagedPlayerNpcsByMap(d.DB().WithContext(d.Context()), byte(worldId), uint32(mapId), page)
		if err != nil {
			d.Logger().WithError(err).Errorf("Listing player NPCs.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		rms, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST models.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms, paginate.EnvelopeFor(paged), r)
	}
}

// handleGetPlayerNpc handles GET /player-npcs/{id}.
func handleGetPlayerNpc(d *HandlerDependency, c *HandlerContext) http.HandlerFunc {
	return server.ParseUUIDId(d.Logger(), idPathVar, func(id uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := getPlayerNpcModel(d.DB().WithContext(d.Context()), id)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Fetching player NPC [%s].", id)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			writePlayerNpc(d, c, w, r, m, http.StatusOK)
		}
	})
}

// handleDeployPlayerNpc handles POST /player-npcs -- always the
// eligibility-checked path (PRD §5); the GM path
// (enforceEligibility: false, design §9.2) is only reached through the GM
// command, not REST.
func handleDeployPlayerNpc(d *HandlerDependency, c *HandlerContext, input DeployRestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var explicit *Position
		if input.Position != nil {
			explicit = &Position{X: input.Position.X, Y: input.Position.Y}
		}

		m, err := processorFor(d).Deploy(input.CharacterId, world.Id(input.WorldId), _map.Id(input.MapId), true, explicit)
		if err != nil {
			d.Logger().WithError(err).Errorf("Deploying player NPC for character [%d].", input.CharacterId)
			writeDeployError(d, w, err)
			return
		}
		writePlayerNpc(d, c, w, r, m, http.StatusCreated)
	}
}

// handleRedeployPlayerNpc handles PATCH /player-npcs/{id} -- refreshes
// appearance and current-standing ranks in place (design §6.2). It takes
// no body: script id, object id and position are immutable through this
// endpoint.
func handleRedeployPlayerNpc(d *HandlerDependency, c *HandlerContext) http.HandlerFunc {
	return server.ParseUUIDId(d.Logger(), idPathVar, func(id uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := processorFor(d).Redeploy(id)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Re-deploying player NPC [%s].", id)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			writePlayerNpc(d, c, w, r, m, http.StatusOK)
		}
	})
}

// handleDeletePlayerNpc handles DELETE /player-npcs/{id}.
func handleDeletePlayerNpc(d *HandlerDependency, _ *HandlerContext) http.HandlerFunc {
	return server.ParseUUIDId(d.Logger(), idPathVar, func(id uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			_, err := processorFor(d).RemoveById(id)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Removing player NPC [%s].", id)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// handleDeletePlayerNpcsByCharacter handles
// DELETE /player-npcs?filter[characterId]=<id>[&filter[mapId]=<id>]
// (FR-8.2, PRD §5).
func handleDeletePlayerNpcsByCharacter(d *HandlerDependency, _ *HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		cidStr := q.Get("filter[characterId]")
		if cidStr == "" {
			server.WriteBadRequest(d.Logger(), w, "filter[characterId] is required")
			return
		}
		cid, err := strconv.ParseUint(cidStr, 10, 32)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "filter[characterId] must be a valid character id")
			return
		}

		var mapIdPtr *_map.Id
		if v := q.Get("filter[mapId]"); v != "" {
			mid, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "filter[mapId] must be a valid map id")
				return
			}
			id := _map.Id(mid)
			mapIdPtr = &id
		}

		if _, err := processorFor(d).Remove(uint32(cid), mapIdPtr); err != nil {
			d.Logger().WithError(err).Errorf("Removing player NPCs for character [%d].", cid)
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// eligibilityResponse is the plain (non-JSON:API) body design §9.1's
// canSpawnPlayerNpc condition consumes.
type eligibilityResponse struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
}

// handleGetEligibility handles
// GET /player-npcs/eligibility?characterId=<id>&mapId=<id>[&worldId=<id>]
// -- the FR-6.1 predicate (design §9.1), the single place the automatic
// deploy check (FR-1.1) and the conversation condition (FR-6.1) both
// evaluate through, via eligibility.Evaluate. worldId is not part of the
// design §9.1 call signature; it defaults to 0, matching the list
// endpoint's filter[worldId] default, and can be supplied when a caller
// needs a non-zero world.
func handleGetEligibility(d *HandlerDependency, _ *HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		cidStr := q.Get("characterId")
		if cidStr == "" {
			server.WriteBadRequest(d.Logger(), w, "characterId is required")
			return
		}
		cid, err := strconv.ParseUint(cidStr, 10, 32)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "characterId must be a valid character id")
			return
		}

		mapIdStr := q.Get("mapId")
		if mapIdStr == "" {
			server.WriteBadRequest(d.Logger(), w, "mapId is required")
			return
		}
		mid, err := strconv.ParseUint(mapIdStr, 10, 32)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "mapId must be a valid map id")
			return
		}

		worldId := byte(0)
		if v := q.Get("worldId"); v != "" {
			wid, err := strconv.ParseUint(v, 10, 8)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "worldId must be a valid world id")
				return
			}
			worldId = byte(wid)
		}

		l, ctx, db := d.Logger(), d.Context(), d.DB()
		c, err := character.NewProcessor(l, ctx).GetById(uint32(cid))
		if err != nil {
			if errors.Is(err, requests.ErrNotFound) {
				writeError(l, w, http.StatusUnprocessableEntity, "unresolvable", err.Error())
				return
			}
			l.WithError(err).Errorf("Fetching character [%d] for eligibility check.", cid)
			server.WriteErrorResponse(l)(w)(err)
			return
		}

		te := tenant.MustFromContext(ctx)
		cfg := configuration.NewProcessor(l, ctx).GetByTenantId(te.Id())

		existingCount, err := countByName(db.WithContext(ctx), worldId, uint32(mid), c.Name())
		if err != nil {
			l.WithError(err).Errorf("Counting existing player NPCs for eligibility check.")
			server.WriteErrorResponse(l)(w)(err)
			return
		}

		eligible, reason := eligibility.Evaluate(cfg, c, existingCount, true)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(eligibilityResponse{Eligible: eligible, Reason: reason}); err != nil {
			l.WithError(err).Errorf("Writing eligibility response.")
		}
	}
}
