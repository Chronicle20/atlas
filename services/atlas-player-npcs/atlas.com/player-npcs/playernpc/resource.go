package playernpc

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	"atlas-player-npcs/inventory"
	"atlas-player-npcs/ranking"
	"encoding/json"
	"errors"
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
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// -- error responses --------------------------------------------------------

// writeError writes a JSON:API errors-array response carrying `code` --
// the design §8.3 failure code REST callers (and the GM command, design
// §9.2) branch on. It reuses api2go/jsonapi.Error's own field set (status,
// code, title, detail) rather than a private duplicate struct, since
// server.WriteErrorResponse/server.WriteBadRequest carry no `code` field
// for this package's domain-specific failure codes to ride on.
func writeError(l logrus.FieldLogger, w http.ResponseWriter, status int, code string, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	doc := struct {
		Errors []jsonapi.Error `json:"errors"`
	}{Errors: []jsonapi.Error{{Status: strconv.Itoa(status), Code: code, Title: http.StatusText(status), Detail: detail}}}
	if err := json.NewEncoder(w).Encode(doc); err != nil {
		l.WithError(err).Errorf("Writing error response.")
	}
}

// writeDeployError maps a Deploy/Redeploy/Remove failure to its PRD §5
// status via CodeFor (Task 23a, the classifier the Kafka command consumer
// also routes through so the two surfaces can never report different
// codes for the same failure): the four design §8.3 codes are 409, an
// unresolvable character or map (CodeUnresolvable) is 422, everything else
// (CodeInternal) is the generic error response.
func writeDeployError(l logrus.FieldLogger, w http.ResponseWriter, err error) {
	switch CodeFor(err) {
	case CodeUnresolvable:
		writeError(l, w, http.StatusUnprocessableEntity, CodeUnresolvable, err.Error())
	case CodeDuplicate:
		writeError(l, w, http.StatusConflict, CodeDuplicate, err.Error())
	case CodePoolExhausted:
		writeError(l, w, http.StatusConflict, CodePoolExhausted, err.Error())
	case CodeMapFull:
		writeError(l, w, http.StatusConflict, CodeMapFull, err.Error())
	case CodeIneligible:
		writeError(l, w, http.StatusConflict, CodeIneligible, err.Error())
	default:
		server.WriteErrorResponse(l)(w)(err)
	}
}

// -- routes ------------------------------------------------------------------

const idPathVar = "id"

// InitializeRoutes registers the PRD §5 surface under PathPrefix
// "/player-npcs". "/eligibility" is registered before "/{id}" so it is not
// shadowed by the id pattern -- the same ordering hazard
// services/atlas-tenants/atlas.com/tenants/configuration/resource.go:1552-1554
// documents. db is curried into every handler (services/atlas-tenants'
// configuration/resource.go handlers follow the same shape) since
// server.HandlerDependency carries only the per-request logger and
// context, not a database handle.
func InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := server.RegisterHandler(l)(si)
			registerDeploy := server.RegisterInputHandler[DeployRestModel](l)(si)
			registerRedeploy := server.RegisterInputHandler[RedeployRestModel](l)(si)

			r := router.PathPrefix("/player-npcs").Subrouter()
			r.HandleFunc("", registerGet("get_player_npcs", handleGetPlayerNpcs(db))).Methods(http.MethodGet)
			r.HandleFunc("", registerDeploy("deploy_player_npc", handleDeployPlayerNpc(db))).Methods(http.MethodPost)
			r.HandleFunc("", registerGet("delete_player_npcs_by_character", handleDeletePlayerNpcsByCharacter(db))).Methods(http.MethodDelete)

			// Must precede "/{id}" -- see the doc comment above.
			r.HandleFunc("/eligibility", registerGet("get_player_npc_eligibility", handleGetEligibility(db))).Methods(http.MethodGet)

			r.HandleFunc("/{"+idPathVar+"}", registerGet("get_player_npc", handleGetPlayerNpc(db))).Methods(http.MethodGet)
			r.HandleFunc("/{"+idPathVar+"}", registerRedeploy("redeploy_player_npc", handleRedeployPlayerNpc(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{"+idPathVar+"}", registerGet("delete_player_npc", handleDeletePlayerNpc(db))).Methods(http.MethodDelete)
		}
	}
}

// processorFor constructs a Processor with the real HTTP-backed read
// clients, per processor.go's NewProcessor doc: "Task 16/17/21's wiring
// constructs the real HTTP-backed processors with NewProcessor(l, ctx) from
// each package and passes them here." emit is nil (a no-op) here -- Task
// 17 wires the Kafka-backed EventEmitter.
func processorFor(db *gorm.DB, d *server.HandlerDependency) Processor {
	l, ctx := d.Logger(), d.Context()
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
func writePlayerNpc(d *server.HandlerDependency, c *server.HandlerContext, w http.ResponseWriter, r *http.Request, m Model, status int) {
	rm, err := Transform(m)
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

// handleGetPlayerNpcs handles GET /player-npcs -- the map-enter hot path
// (PRD §5), tenant-scoped and paginated.
func handleGetPlayerNpcs(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
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

			paged, err := processorFor(db, d).GetByMapPaged(worldId, mapId, page)
			if err != nil {
				d.Logger().WithError(err).Errorf("Listing player NPCs.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			rms, err := TransformSlice(paged.Items)
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
}

// handleGetPlayerNpc handles GET /player-npcs/{id}.
func handleGetPlayerNpc(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return server.ParseUUIDId(d.Logger(), idPathVar, func(id uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := processorFor(db, d).GetById(id)
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
}

// handleDeployPlayerNpc handles POST /player-npcs -- always the
// eligibility-checked path (PRD §5); the GM path
// (enforceEligibility: false, design §9.2) is only reached through the GM
// command, not REST.
func handleDeployPlayerNpc(db *gorm.DB) server.InputHandler[DeployRestModel] {
	return func(d *server.HandlerDependency, c *server.HandlerContext, input DeployRestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var explicit *Position
			if input.Position != nil {
				explicit = &Position{X: input.Position.X, Y: input.Position.Y}
			}

			m, err := processorFor(db, d).Deploy(input.CharacterId, world.Id(input.WorldId), _map.Id(input.MapId), true, explicit)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deploying player NPC for character [%d].", input.CharacterId)
				writeDeployError(d.Logger(), w, err)
				return
			}
			writePlayerNpc(d, c, w, r, m, http.StatusCreated)
		}
	}
}

// handleRedeployPlayerNpc handles PATCH /player-npcs/{id} -- refreshes
// appearance and current-standing ranks in place (design §6.2). The body
// (RedeployRestModel) carries no attributes and is ignored: script id,
// object id and position are immutable through this endpoint.
func handleRedeployPlayerNpc(db *gorm.DB) server.InputHandler[RedeployRestModel] {
	return func(d *server.HandlerDependency, c *server.HandlerContext, _ RedeployRestModel) http.HandlerFunc {
		return server.ParseUUIDId(d.Logger(), idPathVar, func(id uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := processorFor(db, d).Redeploy(id)
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
}

// handleDeletePlayerNpc handles DELETE /player-npcs/{id}.
func handleDeletePlayerNpc(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, _ *server.HandlerContext) http.HandlerFunc {
		return server.ParseUUIDId(d.Logger(), idPathVar, func(id uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				_, err := processorFor(db, d).RemoveById(id)
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
}

// handleDeletePlayerNpcsByCharacter handles
// DELETE /player-npcs?filter[characterId]=<id>[&filter[mapId]=<id>]
// (FR-8.2, PRD §5).
func handleDeletePlayerNpcsByCharacter(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, _ *server.HandlerContext) http.HandlerFunc {
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

			if _, err := processorFor(db, d).Remove(uint32(cid), mapIdPtr); err != nil {
				d.Logger().WithError(err).Errorf("Removing player NPCs for character [%d].", cid)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
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
// -- the FR-6.1 predicate (design §9.1). Evaluation itself lives in
// Processor.Eligibility (processor.go); this handler only parses the query
// string and maps the result to a response.
func handleGetEligibility(db *gorm.DB) server.GetHandler {
	return func(d *server.HandlerDependency, _ *server.HandlerContext) http.HandlerFunc {
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

			eligible, reason, err := processorFor(db, d).Eligibility(uint32(cid), worldId, uint32(mid))
			if err != nil {
				if errors.Is(err, requests.ErrNotFound) {
					writeError(d.Logger(), w, http.StatusUnprocessableEntity, CodeUnresolvable, err.Error())
					return
				}
				d.Logger().WithError(err).Errorf("Fetching character [%d] for eligibility check.", cid)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(eligibilityResponse{Eligible: eligible, Reason: reason}); err != nil {
				d.Logger().WithError(err).Errorf("Writing eligibility response.")
			}
		}
	}
}
