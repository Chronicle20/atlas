package character

import (
	"errors"
	"net/http"
	"sort"
	"strconv"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	"atlas-monster-book/rest"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	GetMonsterBook   = "get_monster_book"
	PatchMonsterBook = "patch_monster_book"
	GetCards         = "get_monster_book_cards"
	GetCard          = "get_monster_book_card"
)

func InitResource(si jsonapi.ServerInformation) func(*gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			get := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("/{characterId}/monster-book", get(GetMonsterBook, handleGet(db))).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}/monster-book", rest.RegisterInputHandler[collection.PatchInput](l)(si)(PatchMonsterBook, handlePatch(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{characterId}/monster-book/cards", get(GetCards, handleListCards(db))).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}/monster-book/cards/{cardId}", get(GetCard, handleGetCard(db))).Methods(http.MethodGet)
		}
	}
}

func handleGet(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(rawId uint32) http.HandlerFunc {
			characterId := characterconst.Id(rawId)
			return func(w http.ResponseWriter, r *http.Request) {
				p := collection.NewProcessor(d.Logger(), d.Context(), db)
				m, err := p.GetByCharacterId(characterId)
				if err != nil {
					// GetByCharacterId synthesizes a default model on
					// ErrRecordNotFound, so any error here is a real DB
					// failure.
					d.Logger().WithError(err).Errorf("Failed to load monster-book collection for character %d.", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := collection.Transform(m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Failed to transform collection model for character %d.", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				server.MarshalResponse[collection.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(rm)
			}
		})
	}
}

func handlePatch(db *gorm.DB) rest.InputHandler[collection.PatchInput] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, in collection.PatchInput) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(rawId uint32) http.HandlerFunc {
			characterId := characterconst.Id(rawId)
			return func(w http.ResponseWriter, r *http.Request) {
				p := collection.NewProcessor(d.Logger(), d.Context(), db)
				if err := p.SetCoverAndEmit(uuid.New(), characterId, in.CoverCardId); err != nil {
					if errors.Is(err, collection.ErrCoverNotOwned) || errors.Is(err, collection.ErrCardIdOutOfRange) {
						d.Logger().WithError(err).Debugf("SetCover validation rejected for character %d cover %d.", characterId, in.CoverCardId)
						w.WriteHeader(http.StatusUnprocessableEntity)
						return
					}
					d.Logger().WithError(err).Errorf("SetCover failed for character %d cover %d.", characterId, in.CoverCardId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				m, err := p.GetByCharacterId(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Failed to reload collection after SetCover for character %d.", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := collection.Transform(m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Failed to transform collection model for character %d.", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				server.MarshalResponse[collection.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(rm)
			}
		})
	}
}

func handleListCards(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(rawId uint32) http.HandlerFunc {
			characterId := characterconst.Id(rawId)
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				cp := card.NewProcessor(d.Logger(), d.Context(), db)
				ms, err := cp.GetByCharacterId(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Failed to list cards for character %d.", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				if v := r.URL.Query().Get("filter[isSpecial]"); v != "" {
					want, perr := strconv.ParseBool(v)
					if perr == nil {
						filtered := ms[:0]
						for _, m := range ms {
							if m.IsSpecial() == want {
								filtered = append(filtered, m)
							}
						}
						ms = filtered
					}
				}

				// entity has a composite primary key (tenant_id, character_id,
				// card_id) with no auto-increment tiebreaker, so
				// database.PagedQuery cannot derive a stable ORDER BY (task-117
				// composite-PK fallback, see atlas-keys precedent).
				// GetByCharacterId's underlying SliceQuery has no explicit
				// ORDER BY either, so sort by CardId (unique within one
				// character's cards) before slicing for determinism.
				cards := make([]card.Model, len(ms))
				copy(cards, ms)
				sort.SliceStable(cards, func(i, j int) bool {
					return cards[i].CardId() < cards[j].CardId()
				})

				paged := paginate.Slice(cards, page)

				res, err := model.SliceMap(card.Transform)(model.FixedProvider(paged.Items))()()
				if err != nil {
					d.Logger().WithError(err).Errorf("Failed to transform cards for character %d.", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				server.MarshalPaginatedResponse[[]card.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

func handleGetCard(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(rawCharId uint32) http.HandlerFunc {
			characterId := characterconst.Id(rawCharId)
			return rest.ParseCardId(d.Logger(), func(rawCardId uint32) http.HandlerFunc {
				cardId := item.Id(rawCardId)
				return func(w http.ResponseWriter, r *http.Request) {
					cp := card.NewProcessor(d.Logger(), d.Context(), db)
					m, err := cp.GetByCharacterIdAndCardId(characterId, cardId)
					if err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							w.WriteHeader(http.StatusNotFound)
							return
						}
						d.Logger().WithError(err).Errorf("Failed to load card %d for character %d.", cardId, characterId)
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
					rm, err := card.Transform(m)
					if err != nil {
						d.Logger().WithError(err).Errorf("Failed to transform card %d for character %d.", cardId, characterId)
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
					server.MarshalResponse[card.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(rm)
				}
			})
		})
	}
}
