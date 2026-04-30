package factory

import (
	"atlas-character-factory/rest"
	"errors"
	"net/http"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	CreateCharacter  = "create_character"
	CreateFromPreset = "create_from_preset"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		// Legacy path retained for back-compat with existing /api/characters/seed callers.
		r := router.PathPrefix("/characters/seed").Subrouter()
		r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)(CreateCharacter, handleCreateCharacter)).Methods(http.MethodPost)

		// New factory routes live under /factory/* so the gateway can route
		// /api/factory(/.*)? unambiguously without colliding with atlas-character.
		// Body is JSON:API encoded: {"data":{"type":"preset-create","attributes":{...}}}.
		fr := router.PathPrefix("/factory/characters").Subrouter()
		fr.HandleFunc("/from-preset", rest.RegisterInputHandler[PresetCreateRestModel](l)(si)(CreateFromPreset, handleCreateFromPreset)).Methods(http.MethodPost)
	}
}

func categorizePresetError(err error) int {
	var nie *NameInvalidError
	switch {
	case errors.Is(err, ErrInvalidPresetId):
		return http.StatusBadRequest
	case errors.Is(err, ErrPresetNotFound):
		return http.StatusNotFound
	case errors.As(err, &nie):
		return http.StatusBadRequest
	case errors.Is(err, ErrNameDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrAtlasDataUnreachable):
		return http.StatusBadGateway
	case errors.Is(err, ErrPresetValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// handleCreateFromPreset handles POST /characters/from-preset.
// The request body must be JSON:API encoded with type "preset-create".
func handleCreateFromPreset(d *rest.HandlerDependency, c *rest.HandlerContext, in PresetCreateRestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		processor := NewProcessor(d.Logger())
		transactionId, err := processor.CreateFromPreset(d.Context(), in)
		if err != nil {
			statusCode := categorizePresetError(err)
			w.WriteHeader(statusCode)
			return
		}

		response := CreateCharacterResponse{TransactionId: transactionId}
		w.WriteHeader(http.StatusAccepted)
		server.MarshalResponse[CreateCharacterResponse](d.Logger())(w)(c.ServerInformation())(map[string][]string{})(response)
	}
}

// categorizeError determines the appropriate HTTP status code for different error types
// used by handleCreateCharacter.
func categorizeError(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Validation errors (user input problems)
	validationErrors := []string{
		"character name must be between 1 and 12 characters and contain only valid characters",
		"gender must be 0 or 1",
		"must provide valid job index",
		"chosen face is not valid for job",
		"chosen hair is not valid for job",
		"chosen hair color is not valid for job",
		"chosen skin color is not valid for job",
		"chosen top is not valid for job",
		"chosen bottom is not valid for job",
		"chosen shoes is not valid for job",
		"chosen weapon is not valid for job",
	}

	errMsg := err.Error()
	for _, ve := range validationErrors {
		if strings.Contains(errMsg, ve) {
			return http.StatusBadRequest
		}
	}

	return http.StatusInternalServerError
}

func handleCreateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		processor := NewProcessor(d.Logger())
		transactionId, err := processor.Create(d.Context(), input)
		if err != nil {
			d.Logger().WithError(err).Error("Error creating character from seed.")
			w.WriteHeader(categorizeError(err))
			return
		}

		response := CreateCharacterResponse{
			TransactionId: transactionId,
		}

		w.WriteHeader(http.StatusAccepted)
		server.MarshalResponse[CreateCharacterResponse](d.Logger())(w)(c.ServerInformation())(map[string][]string{})(response)
	}
}
