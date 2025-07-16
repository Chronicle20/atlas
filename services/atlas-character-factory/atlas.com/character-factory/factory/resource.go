package factory

import (
	"atlas-character-factory/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	CreateCharacter = "create_character"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/characters/seed").Subrouter()
		r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)(CreateCharacter, handleCreateCharacter)).Methods(http.MethodPost)
	}
}

func handleCreateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transactionId, err := Create(d.Logger())(d.Context())(input)
		if err != nil {
			d.Logger().WithError(err).Error("Error creating character from seed.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Create response with transaction ID
		response := CreateCharacterResponse{
			TransactionId: transactionId,
		}

		// Return 202 Accepted with transaction ID
		w.WriteHeader(http.StatusAccepted)
		server.MarshalResponse[CreateCharacterResponse](d.Logger())(w)(c.ServerInformation())(map[string][]string{})(response)
	}
}
