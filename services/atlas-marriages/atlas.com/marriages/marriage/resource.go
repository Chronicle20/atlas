package marriage

import (
	"atlas-marriages/rest"
	"encoding/json"
	"net/http"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitializeRoutes initializes marriage-related REST routes
func InitializeRoutes(db *gorm.DB) func(serverInfo jsonapi.ServerInformation) func(router *mux.Router, logger logrus.FieldLogger) {
	return func(serverInfo jsonapi.ServerInformation) func(router *mux.Router, logger logrus.FieldLogger) {
		return func(router *mux.Router, logger logrus.FieldLogger) {
			// GET /api/characters/{characterId}/marriage
			router.HandleFunc("/characters/{characterId}/marriage",
				rest.RegisterHandler(logger)(serverInfo)("get_character_marriage", getMarriageHandler(db))).
				Methods(http.MethodGet)

			// GET /api/characters/{characterId}/marriage/history
			router.HandleFunc("/characters/{characterId}/marriage/history",
				rest.RegisterHandler(logger)(serverInfo)("get_marriage_history", getMarriageHistoryHandler(db))).
				Methods(http.MethodGet)

			// GET /api/characters/{characterId}/marriage/proposals
			router.HandleFunc("/characters/{characterId}/marriage/proposals",
				rest.RegisterHandler(logger)(serverInfo)("get_character_proposals", getProposalsHandler(db))).
				Methods(http.MethodGet)
		}
	}
}

// getMarriageHandler returns the current marriage for a character
func getMarriageHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				marriage, err := processor.GetMarriageByCharacter(characterId)()
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, err.Error())
					return
				}

				// If no marriage found, return 404
				if marriage == nil {
					writeErrorResponse(w, http.StatusNotFound, "Character is not married")
					return
				}

				// Transform to REST model with partner information
				restMarriage, err := TransformMarriageWithPartner(*marriage, characterId)
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, "Failed to transform marriage data")
					return
				}

				// Get ceremony information if marriage is engaged or married
				if marriage.Status() == StatusEngaged || marriage.Status() == StatusMarried {
					ceremony, err := processor.GetCeremonyByMarriage(marriage.Id())()
					if err == nil && ceremony != nil {
						restMarriage, err = TransformMarriageComplete(*marriage, characterId, ceremony)
						if err != nil {
							d.Logger().WithError(err).Warn("Failed to include ceremony information")
						}
					}
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestMarriage](d.Logger())(w)(c.ServerInformation())(queryParams)(restMarriage)
			}
		})
	}
}

// getMarriageHistoryHandler returns marriage history for a character
func getMarriageHistoryHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				marriages, err := processor.GetMarriageHistory(characterId)()
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, err.Error())
					return
				}

				// Transform marriages to REST models
				restMarriages, err := TransformMarriages(marriages)
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, "Failed to transform marriage data")
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestMarriage](d.Logger())(w)(c.ServerInformation())(queryParams)(restMarriages)
			}
		})
	}
}

// getProposalsHandler returns pending proposals for a character
func getProposalsHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				proposals, err := processor.GetPendingProposalsByCharacter(characterId)()
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, err.Error())
					return
				}

				// Transform proposals to REST models
				restProposals, err := TransformProposals(proposals)
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, "Failed to transform proposal data")
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestProposal](d.Logger())(w)(c.ServerInformation())(queryParams)(restProposals)
			}
		})
	}
}

// writeErrorResponse writes a JSON error response
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
