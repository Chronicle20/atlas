package marriage

import (
	"context"
	"testing"
	"time"

	marriageMsg "atlas-marriages/kafka/message/marriage"
	marriageService "atlas-marriages/marriage"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockProcessor is a mock for the marriage processor
type MockProcessor struct {
	mock.Mock
	marriageService.Processor
}

func (m *MockProcessor) ProposeAndEmit(transactionId uuid.UUID, proposerId, targetId uint32) (marriageService.Proposal, error) {
	args := m.Called(transactionId, proposerId, targetId)
	return args.Get(0).(marriageService.Proposal), args.Error(1)
}

func (m *MockProcessor) AcceptProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (marriageService.Marriage, error) {
	args := m.Called(transactionId, proposalId)
	return args.Get(0).(marriageService.Marriage), args.Error(1)
}

func (m *MockProcessor) DeclineProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (marriageService.Proposal, error) {
	args := m.Called(transactionId, proposalId)
	return args.Get(0).(marriageService.Proposal), args.Error(1)
}

func (m *MockProcessor) CancelProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (marriageService.Proposal, error) {
	args := m.Called(transactionId, proposalId)
	return args.Get(0).(marriageService.Proposal), args.Error(1)
}

func (m *MockProcessor) ScheduleCeremonyAndEmit(transactionId uuid.UUID, marriageId uint32, scheduledAt time.Time, invitees []uint32) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, marriageId, scheduledAt, invitees)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) StartCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) CompleteCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) CancelCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, cancelledBy uint32, reason string) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId, cancelledBy, reason)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) PostponeCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, reason string) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId, reason)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) RescheduleCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, scheduledAt time.Time, rescheduledBy uint32) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId, scheduledAt, rescheduledBy)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) AddInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, inviteeId uint32, addedBy uint32) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId, inviteeId, addedBy)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) RemoveInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, inviteeId uint32, removedBy uint32) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId, inviteeId, removedBy)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func (m *MockProcessor) DivorceAndEmit(transactionId uuid.UUID, marriageId uint32, divorceInitiator uint32) (marriageService.Marriage, error) {
	args := m.Called(transactionId, marriageId, divorceInitiator)
	return args.Get(0).(marriageService.Marriage), args.Error(1)
}

func (m *MockProcessor) AdvanceCeremonyStateAndEmit(transactionId uuid.UUID, ceremonyId uint32, nextState string) (marriageService.Ceremony, error) {
	args := m.Called(transactionId, ceremonyId, nextState)
	return args.Get(0).(marriageService.Ceremony), args.Error(1)
}

func TestNewConfig(t *testing.T) {
	logger, _ := test.NewNullLogger()
	
	configFunc := NewConfig(logger)
	assert.NotNil(t, configFunc)
	
	nameFunc := configFunc("test-name")
	assert.NotNil(t, nameFunc)
	
	tokenFunc := nameFunc("test-token")
	assert.NotNil(t, tokenFunc)
	
	config := tokenFunc("test-group")
	assert.NotNil(t, config)
}

func TestInitHandlers(t *testing.T) {
	// Test that InitHandlers function exists and is callable
	// We don't actually call it to avoid context/database dependencies
	assert.NotNil(t, InitHandlers)
}

func TestInitConsumers(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	
	initFunc := InitConsumers(logger, ctx, &gorm.DB{})
	assert.NotNil(t, initFunc)
	
	// Test that it returns a function that expects consumer setup
	consumerSetupFunc := initFunc(func(config consumer.Config, decorators ...model.Decorator[consumer.Config]) {
		// Mock consumer setup function
	})
	assert.NotNil(t, consumerSetupFunc)
	
	// Test that the consumer setup function works
	consumerSetupFunc("test-group")
	// No assertion needed, just verifying it doesn't panic
}

func TestHandlePropose(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock proposal
	proposal, _ := marriageService.NewProposalBuilder(1, 2, uuid.New()).Build()
	mockProcessor.On("ProposeAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), uint32(2)).Return(proposal, nil)

	handler := handlePropose(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful proposal
	cmd := marriageMsg.Command[marriageMsg.ProposeBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandMarriagePropose,
		Body: marriageMsg.ProposeBody{
			TargetCharacterId: 2,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleAccept(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock marriage
	marriage, _ := marriageService.NewBuilder(1, 2, uuid.New()).Build()
	mockProcessor.On("AcceptProposalAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1)).Return(marriage, nil)

	handler := handleAccept(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful acceptance
	cmd := marriageMsg.Command[marriageMsg.AcceptBody]{
		CharacterId: 2,
		Type:        marriageMsg.CommandMarriageAccept,
		Body: marriageMsg.AcceptBody{
			ProposalId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleDecline(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock declined proposal
	proposal, _ := marriageService.NewProposalBuilder(1, 2, uuid.New()).Build()
	mockProcessor.On("DeclineProposalAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1)).Return(proposal, nil)

	handler := handleDecline(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful decline
	cmd := marriageMsg.Command[marriageMsg.DeclineBody]{
		CharacterId: 2,
		Type:        marriageMsg.CommandMarriageDecline,
		Body: marriageMsg.DeclineBody{
			ProposalId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleCancel(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock cancelled proposal
	proposal, _ := marriageService.NewProposalBuilder(1, 2, uuid.New()).Build()
	mockProcessor.On("CancelProposalAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1)).Return(proposal, nil)

	handler := handleCancel(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful cancel
	cmd := marriageMsg.Command[marriageMsg.CancelBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandMarriageCancel,
		Body: marriageMsg.CancelBody{
			ProposalId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleScheduleCeremony(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	scheduledAt := time.Now().Add(24 * time.Hour)
	invitees := []uint32{3, 4}
	
	mockProcessor.On("ScheduleCeremonyAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), scheduledAt, invitees).Return(ceremony, nil)

	handler := handleScheduleCeremony(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful ceremony scheduling
	cmd := marriageMsg.Command[marriageMsg.ScheduleCeremonyBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonySchedule,
		Body: marriageMsg.ScheduleCeremonyBody{
			MarriageId:  1,
			ScheduledAt: scheduledAt,
			Invitees:    invitees,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleStartCeremony(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	mockProcessor.On("StartCeremonyAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1)).Return(ceremony, nil)

	handler := handleStartCeremony(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful ceremony start
	cmd := marriageMsg.Command[marriageMsg.StartCeremonyBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyStart,
		Body: marriageMsg.StartCeremonyBody{
			CeremonyId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleCompleteCeremony(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	mockProcessor.On("CompleteCeremonyAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1)).Return(ceremony, nil)

	handler := handleCompleteCeremony(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful ceremony completion
	cmd := marriageMsg.Command[marriageMsg.CompleteCeremonyBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyComplete,
		Body: marriageMsg.CompleteCeremonyBody{
			CeremonyId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleCancelCeremony(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	mockProcessor.On("CancelCeremonyAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), uint32(1), "ceremony_cancelled").Return(ceremony, nil)

	handler := handleCancelCeremony(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful ceremony cancellation
	cmd := marriageMsg.Command[marriageMsg.CancelCeremonyBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyCancel,
		Body: marriageMsg.CancelCeremonyBody{
			CeremonyId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandlePostponeCeremony(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	mockProcessor.On("PostponeCeremonyAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), "ceremony_postponed").Return(ceremony, nil)

	handler := handlePostponeCeremony(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful ceremony postponement
	cmd := marriageMsg.Command[marriageMsg.PostponeCeremonyBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyPostpone,
		Body: marriageMsg.PostponeCeremonyBody{
			CeremonyId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleRescheduleCeremony(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	scheduledAt := time.Now().Add(48 * time.Hour)
	mockProcessor.On("RescheduleCeremonyAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), scheduledAt, uint32(1)).Return(ceremony, nil)

	handler := handleRescheduleCeremony(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful ceremony rescheduling
	cmd := marriageMsg.Command[marriageMsg.RescheduleCeremonyBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyReschedule,
		Body: marriageMsg.RescheduleCeremonyBody{
			CeremonyId:  1,
			ScheduledAt: scheduledAt,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleAddInvitee(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	mockProcessor.On("AddInviteeAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), uint32(3), uint32(1)).Return(ceremony, nil)

	handler := handleAddInvitee(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful invitee addition
	cmd := marriageMsg.Command[marriageMsg.AddInviteeBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyAddInvitee,
		Body: marriageMsg.AddInviteeBody{
			CeremonyId:  1,
			CharacterId: 3,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleRemoveInvitee(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	mockProcessor.On("RemoveInviteeAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), uint32(3), uint32(1)).Return(ceremony, nil)

	handler := handleRemoveInvitee(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful invitee removal
	cmd := marriageMsg.Command[marriageMsg.RemoveInviteeBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyRemoveInvitee,
		Body: marriageMsg.RemoveInviteeBody{
			CeremonyId:  1,
			CharacterId: 3,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleDivorce(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock marriage
	marriage, _ := marriageService.NewBuilder(1, 2, uuid.New()).Build()
	mockProcessor.On("DivorceAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), uint32(1)).Return(marriage, nil)

	handler := handleDivorce(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful divorce
	cmd := marriageMsg.Command[marriageMsg.DivorceBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandMarriageDivorce,
		Body: marriageMsg.DivorceBody{
			MarriageId: 1,
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}

func TestHandleAdvanceCeremonyState(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	mockProcessor := new(MockProcessor)

	// Create a mock ceremony
	ceremony, _ := marriageService.NewCeremonyBuilder(1, 1, 2, uuid.New()).Build()
	mockProcessor.On("AdvanceCeremonyStateAndEmit", mock.AnythingOfType("uuid.UUID"), uint32(1), "next_state").Return(ceremony, nil)

	handler := handleAdvanceCeremonyState(logger, ctx, mockProcessor)
	assert.NotNil(t, handler)

	// Test successful ceremony state advancement
	cmd := marriageMsg.Command[marriageMsg.AdvanceCeremonyStateBody]{
		CharacterId: 1,
		Type:        marriageMsg.CommandCeremonyAdvanceState,
		Body: marriageMsg.AdvanceCeremonyStateBody{
			CeremonyId: 1,
			NextState:  "next_state",
		},
	}

	handler(logger, ctx, cmd)
	mockProcessor.AssertExpectations(t)
}