package character

import (
	"atlas-keys/key"
	"atlas-keys/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
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
						w.WriteHeader(http.StatusInternalServerError)
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
					w.WriteHeader(http.StatusInternalServerError)
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
				processor := key.NewProcessor(d.Logger(), d.Context(), db)
				ks, err := processor.GetByCharacterId(characterId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(key.Transform)(model.FixedProvider(ks))()()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[[]key.RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	}
}
