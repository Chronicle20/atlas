package transport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// Error codes for transport failures
const (
	ErrorCodeCapacityFull      = "TRANSPORT_CAPACITY_FULL"
	ErrorCodeAlreadyInTransit  = "TRANSPORT_ALREADY_IN_TRANSIT"
	ErrorCodeRouteNotFound     = "TRANSPORT_ROUTE_NOT_FOUND"
	ErrorCodeServiceError      = "TRANSPORT_SERVICE_ERROR"
)

// TransportError represents an error from the transport service with an error code
type TransportError struct {
	Code    string
	Message string
}

func (e TransportError) Error() string {
	return e.Message
}

// Processor is the interface for transport operations
type Processor interface {
	// StartTransport starts an instance transport for a character
	// Returns a TransportError with specific error code on failure
	StartTransport(routeName string, characterId uint32, worldId world.Id, channelId channel.Id) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new transport processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// StartTransport starts an instance transport for a character
func (p *ProcessorImpl) StartTransport(routeName string, characterId uint32, worldId world.Id, channelId channel.Id) error {
	// Resolve route name to UUID
	route, err := GetRouteByName(p.l, p.ctx)(routeName)
	if err != nil {
		if strings.Contains(err.Error(), "route not found") {
			return TransportError{
				Code:    ErrorCodeRouteNotFound,
				Message: fmt.Sprintf("route not found: %s", routeName),
			}
		}
		return TransportError{
			Code:    ErrorCodeServiceError,
			Message: fmt.Sprintf("failed to resolve route: %s", err.Error()),
		}
	}

	p.l.WithFields(logrus.Fields{
		"route_name":   routeName,
		"route_id":     route.ID.String(),
		"character_id": characterId,
	}).Debug("Starting instance transport")

	// Call the transport service
	err = StartTransport(p.l, p.ctx)(route.ID, characterId, worldId, channelId)
	if err != nil {
		return p.mapTransportError(err)
	}

	return nil
}

// mapTransportError maps HTTP errors to transport-specific error codes
func (p *ProcessorImpl) mapTransportError(err error) error {
	errMsg := err.Error()

	// Check for specific error patterns from the transport service
	if strings.Contains(errMsg, "capacity") || strings.Contains(errMsg, "full") {
		return TransportError{
			Code:    ErrorCodeCapacityFull,
			Message: "transport is at capacity",
		}
	}

	if strings.Contains(errMsg, "already") || strings.Contains(errMsg, "in transit") {
		return TransportError{
			Code:    ErrorCodeAlreadyInTransit,
			Message: "character is already in a transport",
		}
	}

	if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") {
		return TransportError{
			Code:    ErrorCodeRouteNotFound,
			Message: "route not found",
		}
	}

	// Default to service error
	return TransportError{
		Code:    ErrorCodeServiceError,
		Message: fmt.Sprintf("transport service error: %s", errMsg),
	}
}

// GetErrorCode extracts the error code from a TransportError, or returns ServiceError for other errors
func GetErrorCode(err error) string {
	var transportErr TransportError
	if errors.As(err, &transportErr) {
		return transportErr.Code
	}
	return ErrorCodeServiceError
}
