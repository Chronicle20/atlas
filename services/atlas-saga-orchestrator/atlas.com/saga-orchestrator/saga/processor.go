package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for saga processing
type Processor interface {
	GetAll() ([]Saga, error)
	AllProvider() model.Provider[[]Saga]
	GetById(transactionId uuid.UUID) (Saga, error)
	ByIdProvider(transactionId uuid.UUID) model.Provider[Saga]
	Put(saga Saga) error
	MarkFurthestCompletedStepFailed(transactionId uuid.UUID) error
	MarkEarliestPendingStepCompleted(transactionId uuid.UUID) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	t     tenant.Model
	charP character.Processor
	compP compartment.Processor
}

// NewProcessor creates a new saga processor
func NewProcessor(logger logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l:     logger,
		ctx:   ctx,
		t:     tenant.MustFromContext(ctx),
		charP: character.NewProcessor(logger, ctx),
		compP: compartment.NewProcessor(logger, ctx),
	}
}

// GetAll returns all sagas for the current tenant
func (p *ProcessorImpl) GetAll() ([]Saga, error) {
	return p.AllProvider()()
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Saga] {
	return func() ([]Saga, error) {
		return GetCache().GetAll(p.t.Id()), nil
	}
}

// GetById returns a saga by its transaction ID for the current tenant
func (p *ProcessorImpl) GetById(transactionId uuid.UUID) (Saga, error) {
	return p.ByIdProvider(transactionId)()
}

func (p *ProcessorImpl) ByIdProvider(transactionId uuid.UUID) model.Provider[Saga] {
	return func() (Saga, error) {
		m, ok := GetCache().GetById(p.t.Id(), transactionId)
		if !ok {
			return Saga{}, errors.New("saga not found")
		}
		return m, nil
	}
}

// Put adds or updates a saga in the cache for the current tenant
func (p *ProcessorImpl) Put(saga Saga) error {
	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionId.String(),
		"saga_type":      saga.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Inserting saga into cache")

	GetCache().Put(p.t.Id(), saga)

	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionId.String(),
		"saga_type":      saga.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Saga inserted into cache")

	return p.Step(saga.TransactionId)
}

func (p *ProcessorImpl) StepCompleted(transactionId uuid.UUID, success bool) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		return nil
	}

	if s.Failing() {
		err = p.MarkFurthestCompletedStepFailed(transactionId)
		if err != nil {
			return err
		}
	} else {
		status := Failed
		if success {
			status = Completed
		}

		err = p.MarkEarliestPendingStep(transactionId, status)
		if err != nil {
			return err
		}
	}
	return p.Step(transactionId)
}

// MarkFurthestCompletedStepFailed marks the furthest completed step as failed
func (p *ProcessorImpl) MarkFurthestCompletedStepFailed(transactionId uuid.UUID) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Unable to locate saga for marking furthest completed step as failed.")
		return err
	}

	// Find the furthest completed step (last one with status "completed")
	furthestCompletedIndex := s.FindFurthestCompletedStepIndex()

	// If no completed step was found, return an error
	if furthestCompletedIndex == -1 {
		return nil
	}

	// Mark the step as failed
	s.SetStepStatus(furthestCompletedIndex, Failed)

	// Update the saga in the cache
	GetCache().Put(p.t.Id(), s)

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        s.Steps[furthestCompletedIndex].StepId,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Marked furthest completed step as failed.")

	return nil
}

// MarkEarliestPendingStep marks the earliest pending step
func (p *ProcessorImpl) MarkEarliestPendingStep(transactionId uuid.UUID, status Status) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debugf("Unable to locate saga for marking earliest pending step as [%s].", status)
		return err
	}

	// Find the earliest pending step (first one with status "pending")
	earliestPendingIndex := s.FindEarliestPendingStepIndex()

	// If no pending step was found, return an error
	if earliestPendingIndex == -1 {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debugf("No pending steps found to mark as [%s].", status)
		return errors.New("no pending steps found")
	}

	// Mark the step
	s.SetStepStatus(earliestPendingIndex, status)

	// Update the saga in the cache
	GetCache().Put(p.t.Id(), s)

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        s.Steps[earliestPendingIndex].StepId,
		"tenant_id":      p.t.Id().String(),
	}).Debugf("Marked earliest pending step as [%s].", status)

	return nil
}

// ActionHandler is a function type for handling different saga action types
type ActionHandler func(p *ProcessorImpl, s Saga, st Step[any]) error

// actionHandlers maps action types to their handler functions
var actionHandlers = map[Action]ActionHandler{
	AwardInventory:     handleAwardInventory,
	WarpToRandomPortal: handleWarpToRandomPortal,
	WarpToPortal:       handleWarpToPortal,
	AwardExperience:    handleAwardExperience,
	AwardLevel:         handleAwardLevel,
	AwardMesos:         handleAwardMesos,
	DestroyAsset:       handleDestroyAsset,
}

func (p *ProcessorImpl) Step(transactionId uuid.UUID) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Unable to locate saga being stepped.")
		return err
	}

	if s.Failing() {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("Reverting saga step.")
		return nil
	}

	st, ok := s.GetCurrentStep()
	if !ok {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("No steps remaining to progress.")
		GetCache().Remove(p.t.Id(), s.TransactionId)
		// TODO complete saga
		return nil
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debugf("Progressing saga step [%s].", st.StepId)

	// Get the handler for this action type
	handler, exists := actionHandlers[st.Action]
	if !exists {
		return fmt.Errorf("unknown action type: %s", st.Action)
	}

	// Execute the handler
	return handler(p, s, st)
}

// logActionError logs an error that occurred during action processing
func (p *ProcessorImpl) logActionError(s Saga, st Step[any], err error, errorMsg string) {
	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        st.StepId,
		"tenant_id":      p.t.Id().String(),
	}).WithError(err).Error(errorMsg)
}

// handleAwardInventory handles the AwardInventory action
func handleAwardInventory(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardItemActionPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.compP.RequestCreateItem(s.TransactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award item.")
		return err
	}

	return nil
}

// handleWarpToRandomPortal handles the WarpToRandomPortal action
func handleWarpToRandomPortal(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(WarpToRandomPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	err := p.charP.WarpRandomAndEmit(s.TransactionId, payload.CharacterId, f)

	if err != nil {
		p.logActionError(s, st, err, "Unable to warp to random portal.")
		return err
	}

	return nil
}

// handleWarpToPortal handles the WarpToPortal action
func handleWarpToPortal(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(WarpToPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	err := p.charP.WarpToPortalAndEmit(s.TransactionId, payload.CharacterId, f, model.FixedProvider(payload.PortalId))

	if err != nil {
		p.logActionError(s, st, err, "Unable to warp to specific portal.")
		return err
	}

	return nil
}

// handleAwardExperience handles the AwardExperience action
func handleAwardExperience(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardExperiencePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	eds := TransformExperienceDistributions(payload.Distributions)
	err := p.charP.AwardExperienceAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, eds)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award experience.")
		return err
	}

	return nil
}

// handleAwardLevel handles the AwardLevel action
func handleAwardLevel(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardLevelPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.charP.AwardLevelAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.Amount)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award level.")
		return err
	}

	return nil
}

// handleAwardMesos handles the AwardMesos action
func handleAwardMesos(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardMesosPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.charP.AwardMesosAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.ActorId, payload.ActorType, payload.Amount)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award mesos.")
		return err
	}

	return nil
}

// handleDestroyAsset handles the DestroyAsset action
func handleDestroyAsset(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(DestroyAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.compP.RequestDestroyItem(s.TransactionId, payload.CharacterId, payload.TemplateId, payload.Quantity)

	if err != nil {
		p.logActionError(s, st, err, "Unable to destroy asset.")
		return err
	}

	return nil
}

func TransformExperienceDistributions(source []ExperienceDistributions) []character2.ExperienceDistributions {
	target := make([]character2.ExperienceDistributions, len(source))

	for i, s := range source {
		target[i] = character2.ExperienceDistributions{
			ExperienceType: s.ExperienceType,
			Amount:         s.Amount,
			Attr1:          s.Attr1,
		}
	}

	return target
}
