package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/validation"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	WithCharacterProcessor(character.Processor) Handler
	WithCompartmentProcessor(compartment.Processor) Handler
	WithSkillProcessor(skill.Processor) Handler
	WithValidationProcessor(validation.Processor) Handler
	WithGuildProcessor(guild.Processor) Handler
	WithInviteProcessor(invite.Processor) Handler

	GetHandler(action Action) (ActionHandler, bool)
	
	logActionError(s Saga, st Step[any], err error, errorMsg string)
	handleAwardAsset(s Saga, st Step[any]) error
	handleAwardInventory(s Saga, st Step[any]) error
	handleWarpToRandomPortal(s Saga, st Step[any]) error
	handleWarpToPortal(s Saga, st Step[any]) error
	handleAwardExperience(s Saga, st Step[any]) error
	handleAwardLevel(s Saga, st Step[any]) error
	handleAwardMesos(s Saga, st Step[any]) error
	handleDestroyAsset(s Saga, st Step[any]) error
	handleEquipAsset(s Saga, st Step[any]) error
	handleUnequipAsset(s Saga, st Step[any]) error
	handleChangeJob(s Saga, st Step[any]) error
	handleCreateSkill(s Saga, st Step[any]) error
	handleUpdateSkill(s Saga, st Step[any]) error
	handleValidateCharacterState(s Saga, st Step[any]) error
	handleRequestGuildName(s Saga, st Step[any]) error
	handleRequestGuildEmblem(s Saga, st Step[any]) error
	handleRequestGuildDisband(s Saga, st Step[any]) error
	handleRequestGuildCapacityIncrease(s Saga, st Step[any]) error
	handleCreateInvite(s Saga, st Step[any]) error
	handleCreateCharacter(s Saga, st Step[any]) error
	handleCreateAndEquipAsset(s Saga, st Step[any]) error
}

type HandlerImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	t       tenant.Model
	charP   character.Processor
	compP   compartment.Processor
	skillP  skill.Processor
	validP  validation.Processor
	guildP  guild.Processor
	inviteP invite.Processor
}

func NewHandler(l logrus.FieldLogger, ctx context.Context) Handler {
	return &HandlerImpl{
		l:       l,
		ctx:     ctx,
		t:       tenant.MustFromContext(ctx),
		charP:   character.NewProcessor(l, ctx),
		compP:   compartment.NewProcessor(l, ctx),
		skillP:  skill.NewProcessor(l, ctx),
		validP:  validation.NewProcessor(l, ctx),
		guildP:  guild.NewProcessor(l, ctx),
		inviteP: invite.NewProcessor(l, ctx),
	}
}

func (h *HandlerImpl) WithCharacterProcessor(charP character.Processor) Handler {
	return &HandlerImpl{
		l:       h.l,
		ctx:     h.ctx,
		t:       h.t,
		charP:   charP,
		compP:   h.compP,
		skillP:  h.skillP,
		validP:  h.validP,
		guildP:  h.guildP,
		inviteP: h.inviteP,
	}
}

func (h *HandlerImpl) WithCompartmentProcessor(compP compartment.Processor) Handler {
	return &HandlerImpl{
		l:       h.l,
		ctx:     h.ctx,
		t:       h.t,
		charP:   h.charP,
		compP:   compP,
		skillP:  h.skillP,
		validP:  h.validP,
		guildP:  h.guildP,
		inviteP: h.inviteP,
	}
}

func (h *HandlerImpl) WithSkillProcessor(skillP skill.Processor) Handler {
	return &HandlerImpl{
		l:       h.l,
		ctx:     h.ctx,
		t:       h.t,
		charP:   h.charP,
		compP:   h.compP,
		skillP:  skillP,
		validP:  h.validP,
		guildP:  h.guildP,
		inviteP: h.inviteP,
	}
}

func (h *HandlerImpl) WithValidationProcessor(validP validation.Processor) Handler {
	return &HandlerImpl{
		l:       h.l,
		ctx:     h.ctx,
		t:       h.t,
		charP:   h.charP,
		compP:   h.compP,
		skillP:  h.skillP,
		validP:  validP,
		guildP:  h.guildP,
		inviteP: h.inviteP,
	}
}

func (h *HandlerImpl) WithGuildProcessor(guildP guild.Processor) Handler {
	return &HandlerImpl{
		l:       h.l,
		ctx:     h.ctx,
		t:       h.t,
		charP:   h.charP,
		compP:   h.compP,
		skillP:  h.skillP,
		validP:  h.validP,
		guildP:  guildP,
		inviteP: h.inviteP,
	}
}

func (h *HandlerImpl) WithInviteProcessor(inviteP invite.Processor) Handler {
	return &HandlerImpl{
		l:       h.l,
		ctx:     h.ctx,
		t:       h.t,
		charP:   h.charP,
		compP:   h.compP,
		skillP:  h.skillP,
		validP:  h.validP,
		guildP:  h.guildP,
		inviteP: inviteP,
	}
}

// ActionHandler is a function type for handling different saga action types
type ActionHandler func(s Saga, st Step[any]) error

func (h *HandlerImpl) GetHandler(action Action) (ActionHandler, bool) {
	switch action {
	case AwardInventory:
		return h.handleAwardInventory, true
	case AwardAsset:
		return h.handleAwardAsset, true
	case WarpToRandomPortal:
		return h.handleWarpToRandomPortal, true
	case WarpToPortal:
		return h.handleWarpToPortal, true
	case AwardExperience:
		return h.handleAwardExperience, true
	case AwardLevel:
		return h.handleAwardLevel, true
	case AwardMesos:
		return h.handleAwardMesos, true
	case DestroyAsset:
		return h.handleDestroyAsset, true
	case EquipAsset:
		return h.handleEquipAsset, true
	case UnequipAsset:
		return h.handleUnequipAsset, true
	case ChangeJob:
		return h.handleChangeJob, true
	case CreateSkill:
		return h.handleCreateSkill, true
	case UpdateSkill:
		return h.handleUpdateSkill, true
	case ValidateCharacterState:
		return h.handleValidateCharacterState, true
	case RequestGuildName:
		return h.handleRequestGuildName, true
	case RequestGuildEmblem:
		return h.handleRequestGuildEmblem, true
	case RequestGuildDisband:
		return h.handleRequestGuildDisband, true
	case RequestGuildCapacityIncrease:
		return h.handleRequestGuildCapacityIncrease, true
	case CreateInvite:
		return h.handleCreateInvite, true
	case CreateCharacter:
		return h.handleCreateCharacter, true
	case CreateAndEquipAsset:
		return h.handleCreateAndEquipAsset, true

	}
	return nil, false
}

// logActionError logs an error that occurred during action processing
func (h *HandlerImpl) logActionError(s Saga, st Step[any], err error, errorMsg string) {
	h.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        st.StepId,
		"tenant_id":      h.t.Id().String(),
	}).WithError(err).Error(errorMsg)
}

// handleAwardAsset handles the AwardAsset and AwardInventory actions
func (h *HandlerImpl) handleAwardAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardItemActionPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestCreateItem(s.TransactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award asset.")
		return err
	}

	return nil
}

// handleAwardInventory is a wrapper for handleAwardAsset for backward compatibility
// Deprecated: Use handleAwardAsset instead
func (h *HandlerImpl) handleAwardInventory(s Saga, st Step[any]) error {
	return h.handleAwardAsset(s, st)
}

// handleWarpToRandomPortal handles the WarpToRandomPortal action
func (h *HandlerImpl) handleWarpToRandomPortal(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(WarpToRandomPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	err := h.charP.WarpRandomAndEmit(s.TransactionId, payload.CharacterId, f)

	if err != nil {
		h.logActionError(s, st, err, "Unable to warp to random portal.")
		return err
	}

	return nil
}

// handleWarpToPortal handles the WarpToPortal action
func (h *HandlerImpl) handleWarpToPortal(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(WarpToPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	err := h.charP.WarpToPortalAndEmit(s.TransactionId, payload.CharacterId, f, model.FixedProvider(payload.PortalId))

	if err != nil {
		h.logActionError(s, st, err, "Unable to warp to specific portal.")
		return err
	}

	return nil
}

// handleAwardExperience handles the AwardExperience action
func (h *HandlerImpl) handleAwardExperience(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardExperiencePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	eds := TransformExperienceDistributions(payload.Distributions)
	err := h.charP.AwardExperienceAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, eds)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award experience.")
		return err
	}

	return nil
}

// handleAwardLevel handles the AwardLevel action
func (h *HandlerImpl) handleAwardLevel(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardLevelPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.AwardLevelAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.Amount)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award level.")
		return err
	}

	return nil
}

// handleAwardMesos handles the AwardMesos action
func (h *HandlerImpl) handleAwardMesos(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardMesosPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.AwardMesosAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.ActorId, payload.ActorType, payload.Amount)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award mesos.")
		return err
	}

	return nil
}

// handleDestroyAsset handles the DestroyAsset action
func (h *HandlerImpl) handleDestroyAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(DestroyAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestDestroyItem(s.TransactionId, payload.CharacterId, payload.TemplateId, payload.Quantity)

	if err != nil {
		h.logActionError(s, st, err, "Unable to destroy asset.")
		return err
	}

	return nil
}

// handleEquipAsset handles the EquipAsset action
func (h *HandlerImpl) handleEquipAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(EquipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestEquipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Source, payload.Destination)

	if err != nil {
		h.logActionError(s, st, err, "Unable to equip asset.")
		return err
	}

	return nil
}

// handleUnequipAsset handles the UnequipAsset action
func (h *HandlerImpl) handleUnequipAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(UnequipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestUnequipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Source, payload.Destination)

	if err != nil {
		h.logActionError(s, st, err, "Unable to unequip asset.")
		return err
	}

	return nil
}

// handleChangeJob handles the ChangeJob action
func (h *HandlerImpl) handleChangeJob(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(ChangeJobPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.ChangeJobAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.JobId)

	if err != nil {
		h.logActionError(s, st, err, "Unable to change job.")
		return err
	}

	return nil
}

// handleCreateSkill handles the CreateSkill action
func (h *HandlerImpl) handleCreateSkill(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(CreateSkillPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.skillP.RequestCreateAndEmit(s.TransactionId, payload.CharacterId, payload.SkillId, payload.Level, payload.MasterLevel, payload.Expiration)

	if err != nil {
		h.logActionError(s, st, err, "Unable to create skill.")
		return err
	}

	return nil
}

// handleUpdateSkill handles the UpdateSkill action
func (h *HandlerImpl) handleUpdateSkill(s Saga, st Step[any]) error {
	payload, ok := st.Payload.(UpdateSkillPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.skillP.RequestUpdateAndEmit(s.TransactionId, payload.CharacterId, payload.SkillId, payload.Level, payload.MasterLevel, payload.Expiration)

	if err != nil {
		h.logActionError(s, st, err, "Unable to update skill.")
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

// handleValidateCharacterState handles the ValidateCharacterState action
func (h *HandlerImpl) handleValidateCharacterState(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(ValidateCharacterStatePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the validation processor
	result, err := h.validP.ValidateCharacterState(payload.CharacterId, payload.Conditions)
	if err != nil {
		h.logActionError(s, st, err, "Unable to validate character state.")
		return err
	}

	// Check if validation passed
	if !result.Passed() {
		// If validation failed, mark the step as failed
		err := fmt.Errorf("character state validation failed: %v", result.Details())
		h.logActionError(s, st, err, "Character state validation failed.")
		return err
	}

	return nil
}

// handleRequestGuildName handles the RequestGuildName action
func (h *HandlerImpl) handleRequestGuildName(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildNamePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestName(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild name.")
		return err
	}

	return nil
}

// handleRequestGuildEmblem handles the RequestGuildEmblem action
func (h *HandlerImpl) handleRequestGuildEmblem(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildEmblemPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestEmblem(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild emblem.")
		return err
	}

	return nil
}

// handleRequestGuildDisband handles the RequestGuildDisband action
func (h *HandlerImpl) handleRequestGuildDisband(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildDisbandPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestDisband(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild disband.")
		return err
	}

	return nil
}

// handleRequestGuildCapacityIncrease handles the RequestGuildCapacityIncrease action
func (h *HandlerImpl) handleRequestGuildCapacityIncrease(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildCapacityIncreasePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestCapacityIncrease(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild capacity increase.")
		return err
	}

	return nil
}

// handleCreateInvite handles the CreateInvite action
func (h *HandlerImpl) handleCreateInvite(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(CreateInvitePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the invite processor
	err := h.inviteP.Create(s.TransactionId, payload.InviteType, payload.OriginatorId, payload.WorldId, payload.ReferenceId, payload.TargetId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to create invitation.")
		return err
	}

	return nil
}

// handleCreateCharacter handles the CreateCharacter action
func (h *HandlerImpl) handleCreateCharacter(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(CharacterCreatePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the character processor
	err := h.charP.RequestCreateCharacter(s.TransactionId, payload.AccountId, payload.WorldId, payload.Name, payload.Level, payload.Strength, payload.Dexterity, payload.Intelligence, payload.Luck, payload.Hp, payload.Mp, payload.JobId, payload.Gender, payload.Face, payload.Hair, payload.Skin, payload.MapId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to create character.")
		return err
	}

	return nil
}

// handleCreateAndEquipAsset handles the CreateAndEquipAsset action
// This is a compound action that first creates an asset (internally using award_asset semantics)
// and then dynamically creates an equip_asset step when the creation succeeds
func (h *HandlerImpl) handleCreateAndEquipAsset(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(CreateAndEquipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Step 1: Internal award_asset - Create the item using the same logic as handleAwardAsset
	// Convert saga payload to compartment payload to avoid import cycle
	compartmentPayload := compartment.CreateAndEquipAssetPayload{
		CharacterId: payload.CharacterId,
		Item: compartment.ItemPayload{
			TemplateId: payload.Item.TemplateId,
			Quantity:   payload.Item.Quantity,
		},
	}

	err := h.compP.RequestCreateAndEquipAsset(s.TransactionId, compartmentPayload)
	if err != nil {
		h.logActionError(s, st, err, "Unable to create asset for create_and_equip_asset.")
		return err
	}

	// Note: Step 2 (dynamic equip_asset step creation) will be handled by the compartment consumer
	// when it receives the StatusEventTypeCreated event from the compartment service.
	// The consumer will detect this is a CreateAndEquipAsset step and add the equip_asset step.

	return nil
}
