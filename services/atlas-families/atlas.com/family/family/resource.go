package family

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RegisterRoutes registers all family-related REST endpoints
func RegisterRoutes(router *mux.Router, l logrus.FieldLogger, db *gorm.DB, processor Processor, administrator Administrator) {
	// Family management endpoints
	router.HandleFunc("/families/{characterId}/juniors", addJuniorHandler(l, processor)).Methods(http.MethodPost)
	router.HandleFunc("/families/links/{characterId}", breakLinkHandler(l, processor)).Methods(http.MethodDelete)
	router.HandleFunc("/families/tree/{characterId}", getFamilyTreeHandler(l, administrator)).Methods(http.MethodGet)
	
	// Reputation endpoints
	router.HandleFunc("/families/reputation/activities", processActivityHandler(l, administrator)).Methods(http.MethodPost)
	router.HandleFunc("/families/reputation/redeem", redeemRepHandler(l, processor)).Methods(http.MethodPost)
	router.HandleFunc("/families/reputation/{characterId}", getRepHandler(l, db)).Methods(http.MethodGet)
	
	// Location endpoint
	router.HandleFunc("/families/location/{characterId}", getLocationHandler(l, db)).Methods(http.MethodGet)
}

// addJuniorHandler handles POST /families/{characterId}/juniors
func addJuniorHandler(l logrus.FieldLogger, processor Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		seniorIdStr := vars["characterId"]
		
		seniorId, err := strconv.ParseUint(seniorIdStr, 10, 32)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid senior character ID")
			return
		}

		var req AddJuniorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate request
		if req.Data.Attributes.JuniorId == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "Junior ID is required")
			return
		}

		if uint32(seniorId) == req.Data.Attributes.JuniorId {
			writeErrorResponse(w, http.StatusBadRequest, "Cannot add self as junior")
			return
		}

		// Extract transaction ID from headers
		transactionId := r.Header.Get("X-Transaction-Id")
		if transactionId == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Transaction ID is required")
			return
		}

		// Process the request
		result, err := processor.AddJuniorAndEmit(transactionId, uint32(seniorId), req.Data.Attributes.JuniorId)()
		if err != nil {
			l.WithError(err).Error("Failed to add junior")
			
			// Map specific errors to HTTP status codes
			switch err {
			case ErrSeniorNotFound, ErrJuniorNotFound, ErrMemberNotFound:
				writeErrorResponse(w, http.StatusNotFound, err.Error())
			case ErrSeniorHasTooManyJuniors, ErrJuniorAlreadyLinked, ErrLevelDifferenceTooLarge, ErrNotOnSameMap:
				writeErrorResponse(w, http.StatusConflict, err.Error())
			case ErrSelfReference:
				writeErrorResponse(w, http.StatusBadRequest, err.Error())
			default:
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Transform to REST model
		restModel, err := Transform(result)
		if err != nil {
			l.WithError(err).Error("Failed to transform family member to REST model")
			writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		writeResponse(w, http.StatusCreated, restModel)
	}
}

// breakLinkHandler handles DELETE /families/links/{characterId}
func breakLinkHandler(l logrus.FieldLogger, processor Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		characterIdStr := vars["characterId"]
		
		characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid character ID")
			return
		}

		// Extract transaction ID and reason from headers/query params
		transactionId := r.Header.Get("X-Transaction-Id")
		if transactionId == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Transaction ID is required")
			return
		}
		
		reason := r.URL.Query().Get("reason")
		if reason == "" {
			reason = "Member requested link break"
		}

		// Process the request
		updatedMembers, err := processor.BreakLinkAndEmit(transactionId, uint32(characterId), reason)()
		if err != nil {
			l.WithError(err).Error("Failed to break family link")
			
			switch err {
			case ErrMemberNotFound:
				writeErrorResponse(w, http.StatusNotFound, err.Error())
			case ErrNoLinkToBreak:
				writeErrorResponse(w, http.StatusConflict, err.Error())
			default:
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Transform to REST models
		restModels := make([]RestFamilyMember, len(updatedMembers))
		for i, member := range updatedMembers {
			restModel, err := Transform(member)
			if err != nil {
				l.WithError(err).Error("Failed to transform family member to REST model")
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
				return
			}
			restModels[i] = restModel
		}

		writeResponse(w, http.StatusOK, map[string]interface{}{
			"data": restModels,
		})
	}
}

// getFamilyTreeHandler handles GET /families/tree/{characterId}
func getFamilyTreeHandler(l logrus.FieldLogger, administrator Administrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		characterIdStr := vars["characterId"]
		
		characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid character ID")
			return
		}

		// Get family tree
		familyTree, err := administrator.GetFamilyTree(uint32(characterId))()
		if err != nil {
			l.WithError(err).Error("Failed to get family tree")
			
			if err == ErrMemberNotFound {
				writeErrorResponse(w, http.StatusNotFound, err.Error())
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Transform to REST model
		restTree, err := TransformFamilyTree(familyTree)
		if err != nil {
			l.WithError(err).Error("Failed to transform family tree to REST model")
			writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		writeResponse(w, http.StatusOK, restTree)
	}
}

// processActivityHandler handles POST /families/reputation/activities
func processActivityHandler(l logrus.FieldLogger, administrator Administrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ActivityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate request
		if req.Data.Attributes.CharacterId == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "Character ID is required")
			return
		}
		
		if req.Data.Attributes.ActivityType == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Activity type is required")
			return
		}

		// Extract transaction ID from headers
		transactionId := r.Header.Get("X-Transaction-Id")
		if transactionId == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Transaction ID is required")
			return
		}

		// Process the activity
		result, err := administrator.ProcessRepActivity(
			transactionId,
			req.Data.Attributes.CharacterId,
			req.Data.Attributes.ActivityType,
			req.Data.Attributes.Amount,
		)()
		if err != nil {
			l.WithError(err).Error("Failed to process reputation activity")
			
			switch err {
			case ErrMemberNotFound:
				writeErrorResponse(w, http.StatusNotFound, err.Error())
			case ErrRepCapExceeded:
				writeErrorResponse(w, http.StatusConflict, err.Error())
			default:
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Transform to REST model
		restModel, err := Transform(result)
		if err != nil {
			l.WithError(err).Error("Failed to transform family member to REST model")
			writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		writeResponse(w, http.StatusOK, restModel)
	}
}

// redeemRepHandler handles POST /families/reputation/redeem
func redeemRepHandler(l logrus.FieldLogger, processor Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DeductRepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate request
		if req.Data.Attributes.CharacterId == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "Character ID is required")
			return
		}
		
		if req.Data.Attributes.Amount == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "Amount must be greater than 0")
			return
		}
		
		if req.Data.Attributes.Reason == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Reason is required")
			return
		}

		// Extract transaction ID from headers
		transactionId := r.Header.Get("X-Transaction-Id")
		if transactionId == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Transaction ID is required")
			return
		}

		// Process the request
		result, err := processor.DeductRepAndEmit(
			transactionId,
			req.Data.Attributes.CharacterId,
			req.Data.Attributes.Amount,
			req.Data.Attributes.Reason,
		)()
		if err != nil {
			l.WithError(err).Error("Failed to redeem reputation")
			
			switch err {
			case ErrMemberNotFound:
				writeErrorResponse(w, http.StatusNotFound, err.Error())
			case ErrInsufficientRep:
				writeErrorResponse(w, http.StatusConflict, err.Error())
			default:
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Transform to REST model
		restModel, err := Transform(result)
		if err != nil {
			l.WithError(err).Error("Failed to transform family member to REST model")
			writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		writeResponse(w, http.StatusOK, restModel)
	}
}

// getRepHandler handles GET /families/reputation/{characterId}
func getRepHandler(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		characterIdStr := vars["characterId"]
		
		characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid character ID")
			return
		}

		// Get family member
		member, err := GetByCharacterIdProvider(uint32(characterId))(db)()
		if err != nil {
			l.WithError(err).Error("Failed to get family member reputation")
			
			if err == ErrMemberNotFound {
				writeErrorResponse(w, http.StatusNotFound, err.Error())
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Create reputation response using TransformReputation helper
		repModel, err := TransformReputation(member)
		if err != nil {
			l.WithError(err).Error("Failed to transform reputation")
			writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		writeResponse(w, http.StatusOK, repModel)
	}
}

// getLocationHandler handles GET /families/location/{characterId}
func getLocationHandler(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		characterIdStr := vars["characterId"]
		
		characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid character ID")
			return
		}

		// Get family member location
		member, err := GetByCharacterIdProvider(uint32(characterId))(db)()
		if err != nil {
			l.WithError(err).Error("Failed to get family member location")
			
			if err == ErrMemberNotFound {
				writeErrorResponse(w, http.StatusNotFound, err.Error())
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		// Create location response
		// TODO: In a real implementation, this would query the character service 
		// for current location and online status. For now, we'll use defaults.
		var channel *byte = nil
		mapId := uint32(100000001) // Default map ID
		online := true // Default to online
		
		locationModel, err := TransformLocation(member, channel, mapId, online)
		if err != nil {
			l.WithError(err).Error("Failed to transform location")
			writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		writeResponse(w, http.StatusOK, locationModel)
	}
}

// Helper function to extract tenant from request context
func extractTenant(r *http.Request) string {
	// Extract tenant from headers or context
	tenant := r.Header.Get("X-Tenant-Id")
	if tenant == "" {
		// Default tenant for now - in production this should be properly handled
		tenant = "default"
	}
	return tenant
}

// Helper function to extract world from request
func extractWorld(r *http.Request) byte {
	worldStr := r.Header.Get("X-World-Id")
	if worldStr == "" {
		return 0 // Default world
	}
	
	world, err := strconv.ParseUint(worldStr, 10, 8)
	if err != nil {
		return 0
	}
	
	return byte(world)
}

// Helper function to validate and normalize reasons
func normalizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "No reason provided"
	}
	
	// Limit reason length
	if len(reason) > 100 {
		return reason[:100]
	}
	
	return reason
}

// Helper functions for HTTP responses

// writeErrorResponse writes an error response in JSON format
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
	
	json.NewEncoder(w).Encode(errorResponse)
}

// writeResponse writes a success response in JSON format
func writeResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]interface{}{
		"data": data,
	}
	
	json.NewEncoder(w).Encode(response)
}