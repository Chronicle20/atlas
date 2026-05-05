package character

import (
	"net/http"
	"strconv"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	"atlas-monster-book/rest"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
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
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rm, _ := collection.Transform(m)
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
					w.WriteHeader(http.StatusUnprocessableEntity)
					return
				}
				m, err := p.GetByCharacterId(characterId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rm, _ := collection.Transform(m)
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
				cp := card.NewProcessor(d.Logger(), d.Context(), db)
				ms, err := cp.GetByCharacterId(characterId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
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
				offset := parseUintQ(r.URL.Query().Get("page[offset]"), 0)
				limit := parseUintQ(r.URL.Query().Get("page[limit]"), 100)
				if limit > 200 {
					limit = 200
				}
				if int(offset) >= len(ms) {
					ms = nil
				} else {
					end := int(offset) + int(limit)
					if end > len(ms) {
						end = len(ms)
					}
					ms = ms[offset:end]
				}
				res, err := model.SliceMap(card.Transform)(model.FixedProvider(ms))()()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				server.MarshalResponse[[]card.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
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
						w.WriteHeader(http.StatusNotFound)
						return
					}
					rm, _ := card.Transform(m)
					server.MarshalResponse[card.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(rm)
				}
			})
		})
	}
}

func parseUintQ(s string, def uint32) uint32 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return def
	}
	return uint32(v)
}
