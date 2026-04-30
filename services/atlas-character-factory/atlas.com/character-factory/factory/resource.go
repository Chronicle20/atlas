package factory

import (
	"atlas-character-factory/character"
	"atlas-character-factory/rest"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	CreateCharacter    = "create_character"
	CreateFromPreset   = "create_from_preset"
	GetNameValidity    = "get_name_validity"
)

// writeErrorResponse writes a JSON:API compliant error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"status": statusCode,
			"title":  http.StatusText(statusCode),
			"detail": message,
		},
	}

	_ = json.NewEncoder(w).Encode(errorResponse)
}

// categorizeError determines the appropriate HTTP status code for different error types
func categorizeError(err error) int {
	if err == nil {
		return http.StatusOK
	}

	errMsg := err.Error()

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

	for _, validationErr := range validationErrors {
		if strings.Contains(errMsg, validationErr) {
			return http.StatusBadRequest
		}
	}

	// Configuration/template errors (system issues)
	if strings.Contains(errMsg, "unable to find template validation configuration") ||
		strings.Contains(errMsg, "Unable to find template validation configuration") {
		return http.StatusInternalServerError
	}

	// Saga creation errors (internal service errors)
	if strings.Contains(errMsg, "unable to emit character creation saga") ||
		strings.Contains(errMsg, "Unable to emit character creation saga") {
		return http.StatusInternalServerError
	}

	// Default to internal server error for unknown errors
	return http.StatusInternalServerError
}

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/characters/seed").Subrouter()
		r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)(CreateCharacter, handleCreateCharacter)).Methods(http.MethodPost)

		cr := router.PathPrefix("/characters").Subrouter()
		cr.HandleFunc("/from-preset", rest.RegisterHandler(l)(si)(CreateFromPreset, handleCreateFromPreset)).Methods(http.MethodPost)
		cr.HandleFunc("/name-validity", rest.RegisterHandler(l)(si)(GetNameValidity, handleNameValidity)).Methods(http.MethodGet)
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

func handleCreateFromPreset(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in PresetCreateRestModel
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		defer r.Body.Close()

		if in.PresetId == "" || in.Name == "" {
			writeErrorResponse(w, http.StatusBadRequest, "presetId and name are required")
			return
		}

		processor := NewProcessor(d.Logger())
		transactionId, err := processor.CreateFromPreset(d.Context(), in)
		if err != nil {
			statusCode := categorizePresetError(err)
			writeErrorResponse(w, statusCode, err.Error())
			return
		}

		response := CreateCharacterResponse{TransactionId: transactionId}
		w.WriteHeader(http.StatusAccepted)
		server.MarshalResponse[CreateCharacterResponse](d.Logger())(w)(c.ServerInformation())(map[string][]string{})(response)
	}
}

func handleNameValidity(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		name := q.Get("name")
		widRaw := q.Get("worldId")
		if name == "" || widRaw == "" {
			writeErrorResponse(w, http.StatusBadRequest, "name and worldId are required")
			return
		}
		wid, err := strconv.ParseUint(widRaw, 10, 8)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "worldId must be a non-negative integer")
			return
		}

		client := character.NewNameValidityClient(d.Logger())
		res, err := client.Check(d.Context(), name, byte(wid))
		if err != nil {
			d.Logger().WithError(err).Error("name-validity passthrough failed")
			writeErrorResponse(w, http.StatusBadGateway, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}
}

func handleCreateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		processor := NewProcessor(d.Logger())
		transactionId, err := processor.Create(d.Context(), input)
		if err != nil {
			d.Logger().WithError(err).Error("Error creating character from seed.")

			// Determine appropriate HTTP status code based on error type
			statusCode := categorizeError(err)
			writeErrorResponse(w, statusCode, err.Error())
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
