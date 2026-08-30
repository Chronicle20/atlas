package craft

import (
	"atlas-maker/character"
	"atlas-maker/compartment"
	"atlas-maker/crystalband"
	"atlas-maker/data/equipment"
	"atlas-maker/data/itemmake"
	"atlas-maker/quest"
	"atlas-maker/reagent"
	"atlas-maker/recipe"
	"atlas-maker/rest"
	"atlas-maker/skill"
	"errors"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// writeMethods are the methods the read-only recipe routes must refuse
// (FR-2.3).
var writeMethods = []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

func handleMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// processorFactory builds the (craft.Processor, recipe.Processor) pair a
// request handler needs from its per-request HandlerDependency.
// defaultProcessorFactory (production) builds every upstream HTTP-backed
// processor for real; resource_test.go substitutes one returning mock-backed
// processors, so a route's HTTP-level test never dials a real upstream
// service.
type processorFactory func(d *rest.HandlerDependency) (Processor, recipe.Processor)

// defaultProcessorFactory wires the full craft.Processor and its underlying
// recipe.Processor from d's logger, tenant context, and database: every
// upstream Task 15-23 built (character/skill/compartment/quest,
// reagent/crystalband, the equipment data client) plus the Kafka-backed
// SagaEmitter this task supplies. recipe.Processor is returned separately
// since ProcessorImpl's rp field is unexported and the GET handlers need
// GetAll/GetById directly.
func defaultProcessorFactory(d *rest.HandlerDependency) (Processor, recipe.Processor) {
	l := d.Logger()
	ctx := d.Context()

	cp := character.NewProcessor(l, ctx)
	sp := skill.NewProcessor(l, ctx)
	kp := compartment.NewProcessor(l, ctx)
	qp := quest.NewProcessor(l, ctx)
	im := itemmake.NewProcessor(l, ctx)
	rp := recipe.NewProcessor(l, ctx, im)
	rgp := reagent.NewProcessor(l, ctx, d.DB())
	cbp := crystalband.NewProcessor(l, ctx, d.DB())
	eqp := equipment.NewProcessor(l, ctx)
	em := NewKafkaEmitter(l, ctx)

	return NewProcessor(l, ctx, cp, sp, kp, qp, rp, rgp, cbp, eqp, em), rp
}

// InitResource registers the three PRD §5 atlas-maker routes: the two
// read-only recipe routes and the crafts POST. Every write method on the
// recipe routes is rejected explicitly (mirrors
// services/atlas-maker/atlas.com/maker/reagent/resource.go) rather than
// left to gorilla/mux's implicit method-mismatch handling.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return initResource(si, defaultProcessorFactory)
}

// initResource is InitResource's factory-parameterized body, split out so
// resource_test.go can substitute a mock-backed processorFactory while
// exercising the real route registration, JSON:API marshaling, and 405
// enforcement.
func initResource(si jsonapi.ServerInformation, pf processorFactory) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[CraftRequestRestModel](l)(db)(si)

			r := router.PathPrefix("/characters/{characterId}/maker").Subrouter()

			const recipesPath = "/recipes"
			const recipePath = "/recipes/{itemId:[0-9]+}"

			r.HandleFunc(recipesPath, registerGet("get_maker_recipes", handleListRecipes(pf))).Methods(http.MethodGet)
			r.HandleFunc(recipePath, registerGet("get_maker_recipe", handleGetRecipe(pf))).Methods(http.MethodGet)
			r.HandleFunc("/crafts", registerInput("create_maker_craft", handleCreateCraft(pf))).Methods(http.MethodPost)

			for _, m := range writeMethods {
				r.HandleFunc(recipesPath, handleMethodNotAllowed).Methods(m)
				r.HandleFunc(recipePath, handleMethodNotAllowed).Methods(m)
			}
		}
	}
}

// writeProcessorError maps err onto a response: a CraftError writes its own
// PRD §5 JSON:API code, anything else falls through to
// server.WriteErrorResponse (500, or 503 if the registered transient
// classifier recognizes it).
func writeProcessorError(l logrus.FieldLogger, w http.ResponseWriter, err error) {
	var ce CraftError
	if errors.As(err, &ce) {
		writeCraftError(l, w, ce)
		return
	}
	server.WriteErrorResponse(l)(w)(err)
}

// handleListRecipes returns only the recipes characterId currently
// qualifies for (design §4.2.5), sorted by item id for deterministic
// pagination, then paginated per the standard params.
func handleListRecipes(pf processorFactory) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				p, rp := pf(d)

				recipes, err := rp.GetAll()
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving maker recipes.")
					writeProcessorError(d.Logger(), w, err)
					return
				}

				snap, err := p.NewSnapshot(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Building snapshot for character [%d].", characterId)
					writeProcessorError(d.Logger(), w, err)
					return
				}

				eligible := make([]RecipeRestModel, 0, len(recipes))
				for _, rc := range recipes {
					elig, err := p.Evaluate(characterId, snap, rc)
					if err != nil {
						d.Logger().WithError(err).Errorf("Evaluating recipe [%d] for character [%d].", rc.Id(), characterId)
						writeProcessorError(d.Logger(), w, err)
						return
					}
					if elig.Eligible {
						eligible = append(eligible, TransformRecipe(rc, elig))
					}
				}
				sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].ItemId < eligible[j].ItemId })

				paged := paginate.Slice(eligible, page)
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RecipeRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

// handleGetRecipe returns one recipe with characterId's eligibility verdict
// and its computed material/meso cost (the recipe's own Materials/Meso --
// no separate cost model exists, since neither varies by request).
func handleGetRecipe(pf processorFactory) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseItemId(d.Logger(), func(itemId item.Id) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					p, rp := pf(d)

					rc, err := rp.GetById(itemId)
					if err != nil {
						if errors.Is(err, recipe.ErrNotFound) {
							writeCraftError(d.Logger(), w, ErrRecipeNotFound)
							return
						}
						d.Logger().WithError(err).Errorf("Retrieving recipe [%d].", itemId)
						writeProcessorError(d.Logger(), w, err)
						return
					}

					snap, err := p.NewSnapshot(characterId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Building snapshot for character [%d].", characterId)
						writeProcessorError(d.Logger(), w, err)
						return
					}

					elig, err := p.Evaluate(characterId, snap, rc)
					if err != nil {
						d.Logger().WithError(err).Errorf("Evaluating recipe [%d] for character [%d].", itemId, characterId)
						writeProcessorError(d.Logger(), w, err)
						return
					}

					rm := TransformRecipe(rc, elig)
					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RecipeRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// handleCreateCraft validates and, on acceptance, emits the craft saga
// (Processor.Create), returning its transaction id.
func handleCreateCraft(pf processorFactory) rest.InputHandler[CraftRequestRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input CraftRequestRestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p, _ := pf(d)

				txId, err := p.Create(characterId, input.ToRequest())
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating craft for character [%d].", characterId)
					writeProcessorError(d.Logger(), w, err)
					return
				}

				rm := CraftResponseRestModel{Id: txId.String(), TransactionId: txId.String()}
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[CraftResponseRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}
