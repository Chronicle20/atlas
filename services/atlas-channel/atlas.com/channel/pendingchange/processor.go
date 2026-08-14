package pendingchange

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	TypeNameChange    = "NAME_CHANGE"
	TypeWorldTransfer = "WORLD_TRANSFER"
)

// Processor defines the operations atlas-channel uses to create pending
// changes (name change / world transfer) in atlas-character. It is
// scaffolding for the BUY_NAME_CHANGE / BUY_TRANSFER_WORLD handlers (Task
// 25), which decode the real requested name / destination world from the
// ShopOperationBuy* ops.
type Processor interface {
	RequestNameChange(characterId uint32, requestedName string, assetId uint32) (RestModel, error)
	RequestWorldTransfer(characterId uint32, destinationWorldId world.Id, assetId uint32) (RestModel, error)
	CancelPendingChange(characterId uint32, changeType string) (RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) RequestNameChange(characterId uint32, requestedName string, assetId uint32) (RestModel, error) {
	input := CreateInputRestModel{
		Type:          TypeNameChange,
		RequestedName: requestedName,
		AssetId:       &assetId,
	}
	return p.request(characterId, input)
}

func (p *ProcessorImpl) RequestWorldTransfer(characterId uint32, destinationWorldId world.Id, assetId uint32) (RestModel, error) {
	input := CreateInputRestModel{
		Type:               TypeWorldTransfer,
		DestinationWorldId: destinationWorldId,
		AssetId:            &assetId,
	}
	return p.request(characterId, input)
}

// CancelPendingChange cancels the calling character's own pending record of
// changeType (task-227 client-cancel addendum: the wire arm is the coupon's
// item-use double-confirm dialog chain riding the generic cash item-use op,
// see docs/tasks/task-227-cash-name-change-world-transfer/cancel-entry-point.md
// and cancel-confirm-semantics.md). A 404 (nothing pending) is reported as a
// *RejectedError so the caller can distinguish it from an infrastructure
// failure -- it is not an error for the player, just nothing to cancel.
func (p *ProcessorImpl) CancelPendingChange(characterId uint32, changeType string) (RestModel, error) {
	input := CancelInputRestModel{Type: changeType}
	result, err := postCancel(p.l, p.ctx)(requestCancelUrl(characterId), input)
	if err == nil {
		return result, nil
	}

	if se, ok := asStatusError(err); ok {
		switch se.StatusCode {
		case http.StatusNotFound:
			reason := se.Detail
			if reason == "" {
				reason = "not_pending"
			}
			return result, &RejectedError{Status: http.StatusNotFound, Reason: reason}
		case http.StatusConflict:
			return result, &RejectedError{Status: http.StatusConflict, Reason: se.Detail}
		case http.StatusUnprocessableEntity:
			return result, &RejectedError{Status: http.StatusUnprocessableEntity, Reason: se.Detail}
		}
	}
	return result, err
}

func (p *ProcessorImpl) request(characterId uint32, input CreateInputRestModel) (RestModel, error) {
	result, err := postCreate(p.l, p.ctx)(requestCreateUrl(characterId), input)
	if err == nil {
		return result, nil
	}

	if se, ok := asStatusError(err); ok {
		switch se.StatusCode {
		case http.StatusConflict:
			return result, &RejectedError{Status: http.StatusConflict, Reason: se.Detail}
		case http.StatusUnprocessableEntity:
			return result, &RejectedError{Status: http.StatusUnprocessableEntity, Reason: se.Detail}
		case http.StatusNotFound:
			reason := se.Detail
			if reason == "" {
				reason = "unknown_character"
			}
			return result, &RejectedError{Status: http.StatusNotFound, Reason: reason}
		}
	}
	return result, err
}

// RejectedError is the typed, non-infrastructure rejection of a
// pending-change creation request, carrying the HTTP status atlas-character
// responded with and the design-level reason string (empty only when the
// status alone is the whole story).
type RejectedError struct {
	Status int
	Reason string
}

func (e *RejectedError) Error() string {
	if e.Reason != "" {
		return e.Reason
	}
	return http.StatusText(e.Status)
}
