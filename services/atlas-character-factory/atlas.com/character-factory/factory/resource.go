package factory

import (
	"atlas-character-factory/character"
	"atlas-character-factory/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	CreateCharacter = "create_character"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/characters").Subrouter()
		r.HandleFunc("", rest.RegisterCreateHandler[RestModel](l)(si)(CreateCharacter, handleCreateCharacter)).Methods(http.MethodPost)
	}
}

func handleCreateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cm, err := Create(d.Logger(), d.Span(), c.Tenant())(input.AccountId, input.WorldId, input.Name, input.Gender, input.JobIndex, input.SubJobIndex, input.Face, input.Hair, input.HairColor, input.SkinColor, input.Top, input.Bottom, input.Shoes, input.Weapon)
		if err != nil {
			d.Logger().WithError(err).Error("Error creating character from seed.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		res, err := model.Transform(cm, character.Transform)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[character.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		w.WriteHeader(http.StatusCreated)
	}
}
