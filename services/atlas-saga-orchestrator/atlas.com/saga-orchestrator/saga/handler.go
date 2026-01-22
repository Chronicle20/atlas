package saga

import (
	"atlas-saga-orchestrator/buddylist"
	"atlas-saga-orchestrator/cashshop"
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/consumable"
	"atlas-saga-orchestrator/data/foothold"
	"atlas-saga-orchestrator/data/portal"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	storage2 "atlas-saga-orchestrator/kafka/message/storage"
	"atlas-saga-orchestrator/monster"
	"atlas-saga-orchestrator/pet"
	portalBlocking "atlas-saga-orchestrator/portal"
	"atlas-saga-orchestrator/quest"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/storage"
	system_message "atlas-saga-orchestrator/system_message"
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
	WithBuddyListProcessor(buddylist.Processor) Handler
	WithPetProcessor(pet.Processor) Handler
	WithFootholdProcessor(foothold.Processor) Handler
	WithMonsterProcessor(monster.Processor) Handler
	WithConsumableProcessor(consumable.Processor) Handler
	WithPortalProcessor(portal.Processor) Handler
	WithPortalBlockingProcessor(portalBlocking.Processor) Handler
	WithCashshopProcessor(cashshop.Processor) Handler
	WithSystemMessageProcessor(system_message.Processor) Handler
	WithQuestProcessor(quest.Processor) Handler
	WithStorageProcessor(storage.Processor) Handler

	GetHandler(action Action) (ActionHandler, bool)

	logActionError(s Saga, st Step[any], err error, errorMsg string)
	handleAwardAsset(s Saga, st Step[any]) error
	handleAwardInventory(s Saga, st Step[any]) error
	handleWarpToRandomPortal(s Saga, st Step[any]) error
	handleWarpToPortal(s Saga, st Step[any]) error
	handleAwardExperience(s Saga, st Step[any]) error
	handleAwardLevel(s Saga, st Step[any]) error
	handleAwardMesos(s Saga, st Step[any]) error
	handleAwardCurrency(s Saga, st Step[any]) error
	handleDestroyAsset(s Saga, st Step[any]) error
	handleEquipAsset(s Saga, st Step[any]) error
	handleUnequipAsset(s Saga, st Step[any]) error
	handleChangeJob(s Saga, st Step[any]) error
	handleChangeHair(s Saga, st Step[any]) error
	handleChangeFace(s Saga, st Step[any]) error
	handleChangeSkin(s Saga, st Step[any]) error
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
	handleIncreaseBuddyCapacity(s Saga, st Step[any]) error
	handleGainCloseness(s Saga, st Step[any]) error
	handleSpawnMonster(s Saga, st Step[any]) error
	handleCompleteQuest(s Saga, st Step[any]) error
	handleStartQuest(s Saga, st Step[any]) error
	handleApplyConsumableEffect(s Saga, st Step[any]) error
	handleSendMessage(s Saga, st Step[any]) error
	handleDepositToStorage(s Saga, st Step[any]) error
	handleUpdateStorageMesos(s Saga, st Step[any]) error
	handleAwardFame(s Saga, st Step[any]) error
	handleShowStorage(s Saga, st Step[any]) error
	handleAcceptToStorage(s Saga, st Step[any]) error
	handleReleaseFromCharacter(s Saga, st Step[any]) error
	handleAcceptToCharacter(s Saga, st Step[any]) error
	handleReleaseFromStorage(s Saga, st Step[any]) error
	handleAcceptToCashShop(s Saga, st Step[any]) error
	handleReleaseFromCashShop(s Saga, st Step[any]) error
	handlePlayPortalSound(s Saga, st Step[any]) error
	handleShowInfo(s Saga, st Step[any]) error
	handleShowInfoText(s Saga, st Step[any]) error
	handleUpdateAreaInfo(s Saga, st Step[any]) error
	handleShowHint(s Saga, st Step[any]) error
	handleBlockPortal(s Saga, st Step[any]) error
	handleUnblockPortal(s Saga, st Step[any]) error
}

type HandlerImpl struct {
	l               logrus.FieldLogger
	ctx             context.Context
	t               tenant.Model
	charP           character.Processor
	compP           compartment.Processor
	skillP          skill.Processor
	validP          validation.Processor
	guildP          guild.Processor
	inviteP         invite.Processor
	buddyListP      buddylist.Processor
	petP            pet.Processor
	footholdP       foothold.Processor
	monsterP        monster.Processor
	consumableP     consumable.Processor
	portalP         portal.Processor
	portalBlockingP portalBlocking.Processor
	cashshopP       cashshop.Processor
	systemMessageP  system_message.Processor
	questP          quest.Processor
	storageP        storage.Processor
}

func NewHandler(l logrus.FieldLogger, ctx context.Context) Handler {
	return &HandlerImpl{
		l:               l,
		ctx:             ctx,
		t:               tenant.MustFromContext(ctx),
		charP:           character.NewProcessor(l, ctx),
		compP:           compartment.NewProcessor(l, ctx),
		skillP:          skill.NewProcessor(l, ctx),
		validP:          validation.NewProcessor(l, ctx),
		guildP:          guild.NewProcessor(l, ctx),
		inviteP:         invite.NewProcessor(l, ctx),
		buddyListP:      buddylist.NewProcessor(l, ctx),
		petP:            pet.NewProcessor(l, ctx),
		footholdP:       foothold.NewProcessor(l, ctx),
		monsterP:        monster.NewProcessor(l, ctx),
		consumableP:     consumable.NewProcessor(l, ctx),
		portalBlockingP: portalBlocking.NewProcessor(l, ctx),
		cashshopP:       cashshop.NewProcessor(l, ctx),
		systemMessageP:  system_message.NewProcessor(l, ctx),
		questP:          quest.NewProcessor(l, ctx),
		storageP:        storage.NewProcessor(l, ctx),
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
		l:          h.l,
		ctx:        h.ctx,
		t:          h.t,
		charP:      h.charP,
		compP:      h.compP,
		skillP:     h.skillP,
		validP:     h.validP,
		guildP:     h.guildP,
		inviteP:    inviteP,
		buddyListP: h.buddyListP,
	}
}

func (h *HandlerImpl) WithBuddyListProcessor(buddyListP buddylist.Processor) Handler {
	return &HandlerImpl{
		l:          h.l,
		ctx:        h.ctx,
		t:          h.t,
		charP:      h.charP,
		compP:      h.compP,
		skillP:     h.skillP,
		validP:     h.validP,
		guildP:     h.guildP,
		inviteP:    h.inviteP,
		buddyListP: buddyListP,
		petP:       h.petP,
	}
}

func (h *HandlerImpl) WithPetProcessor(petP pet.Processor) Handler {
	return &HandlerImpl{
		l:          h.l,
		ctx:        h.ctx,
		t:          h.t,
		charP:      h.charP,
		compP:      h.compP,
		skillP:     h.skillP,
		validP:     h.validP,
		guildP:     h.guildP,
		inviteP:    h.inviteP,
		buddyListP: h.buddyListP,
		petP:       petP,
		footholdP:  h.footholdP,
		monsterP:   h.monsterP,
	}
}

func (h *HandlerImpl) WithFootholdProcessor(footholdP foothold.Processor) Handler {
	return &HandlerImpl{
		l:          h.l,
		ctx:        h.ctx,
		t:          h.t,
		charP:      h.charP,
		compP:      h.compP,
		skillP:     h.skillP,
		validP:     h.validP,
		guildP:     h.guildP,
		inviteP:    h.inviteP,
		buddyListP: h.buddyListP,
		petP:       h.petP,
		footholdP:  footholdP,
		monsterP:   h.monsterP,
	}
}

func (h *HandlerImpl) WithMonsterProcessor(monsterP monster.Processor) Handler {
	return &HandlerImpl{
		l:           h.l,
		ctx:         h.ctx,
		t:           h.t,
		charP:       h.charP,
		compP:       h.compP,
		skillP:      h.skillP,
		validP:      h.validP,
		guildP:      h.guildP,
		inviteP:     h.inviteP,
		buddyListP:  h.buddyListP,
		petP:        h.petP,
		footholdP:   h.footholdP,
		monsterP:    monsterP,
		consumableP: h.consumableP,
		portalP:     h.portalP,
	}
}

func (h *HandlerImpl) WithConsumableProcessor(consumableP consumable.Processor) Handler {
	return &HandlerImpl{
		l:           h.l,
		ctx:         h.ctx,
		t:           h.t,
		charP:       h.charP,
		compP:       h.compP,
		skillP:      h.skillP,
		validP:      h.validP,
		guildP:      h.guildP,
		inviteP:     h.inviteP,
		buddyListP:  h.buddyListP,
		petP:        h.petP,
		footholdP:   h.footholdP,
		monsterP:    h.monsterP,
		consumableP: consumableP,
		portalP:     h.portalP,
	}
}

func (h *HandlerImpl) WithPortalProcessor(portalP portal.Processor) Handler {
	return &HandlerImpl{
		l:               h.l,
		ctx:             h.ctx,
		t:               h.t,
		charP:           h.charP,
		compP:           h.compP,
		skillP:          h.skillP,
		validP:          h.validP,
		guildP:          h.guildP,
		inviteP:         h.inviteP,
		buddyListP:      h.buddyListP,
		petP:            h.petP,
		footholdP:       h.footholdP,
		monsterP:        h.monsterP,
		consumableP:     h.consumableP,
		portalP:         portalP,
		portalBlockingP: h.portalBlockingP,
		cashshopP:       h.cashshopP,
	}
}

func (h *HandlerImpl) WithPortalBlockingProcessor(portalBlockingP portalBlocking.Processor) Handler {
	return &HandlerImpl{
		l:               h.l,
		ctx:             h.ctx,
		t:               h.t,
		charP:           h.charP,
		compP:           h.compP,
		skillP:          h.skillP,
		validP:          h.validP,
		guildP:          h.guildP,
		inviteP:         h.inviteP,
		buddyListP:      h.buddyListP,
		petP:            h.petP,
		footholdP:       h.footholdP,
		monsterP:        h.monsterP,
		consumableP:     h.consumableP,
		portalP:         h.portalP,
		portalBlockingP: portalBlockingP,
		cashshopP:       h.cashshopP,
	}
}

func (h *HandlerImpl) WithCashshopProcessor(cashshopP cashshop.Processor) Handler {
	return &HandlerImpl{
		l:              h.l,
		ctx:            h.ctx,
		t:              h.t,
		charP:          h.charP,
		compP:          h.compP,
		skillP:         h.skillP,
		validP:         h.validP,
		guildP:         h.guildP,
		inviteP:        h.inviteP,
		buddyListP:     h.buddyListP,
		petP:           h.petP,
		footholdP:      h.footholdP,
		monsterP:       h.monsterP,
		consumableP:    h.consumableP,
		portalP:        h.portalP,
		cashshopP:      cashshopP,
		systemMessageP: h.systemMessageP,
	}
}

func (h *HandlerImpl) WithSystemMessageProcessor(systemMessageP system_message.Processor) Handler {
	return &HandlerImpl{
		l:              h.l,
		ctx:            h.ctx,
		t:              h.t,
		charP:          h.charP,
		compP:          h.compP,
		skillP:         h.skillP,
		validP:         h.validP,
		guildP:         h.guildP,
		inviteP:        h.inviteP,
		buddyListP:     h.buddyListP,
		petP:           h.petP,
		footholdP:      h.footholdP,
		monsterP:       h.monsterP,
		consumableP:    h.consumableP,
		portalP:        h.portalP,
		cashshopP:      h.cashshopP,
		systemMessageP: systemMessageP,
	}
}

func (h *HandlerImpl) WithQuestProcessor(questP quest.Processor) Handler {
	return &HandlerImpl{
		l:              h.l,
		ctx:            h.ctx,
		t:              h.t,
		charP:          h.charP,
		compP:          h.compP,
		skillP:         h.skillP,
		validP:         h.validP,
		guildP:         h.guildP,
		inviteP:        h.inviteP,
		buddyListP:     h.buddyListP,
		petP:           h.petP,
		footholdP:      h.footholdP,
		monsterP:       h.monsterP,
		consumableP:    h.consumableP,
		portalP:        h.portalP,
		cashshopP:      h.cashshopP,
		systemMessageP: h.systemMessageP,
		questP:         questP,
		storageP:       h.storageP,
	}
}

func (h *HandlerImpl) WithStorageProcessor(storageP storage.Processor) Handler {
	return &HandlerImpl{
		l:              h.l,
		ctx:            h.ctx,
		t:              h.t,
		charP:          h.charP,
		compP:          h.compP,
		skillP:         h.skillP,
		validP:         h.validP,
		guildP:         h.guildP,
		inviteP:        h.inviteP,
		buddyListP:     h.buddyListP,
		petP:           h.petP,
		footholdP:      h.footholdP,
		monsterP:       h.monsterP,
		consumableP:    h.consumableP,
		portalP:        h.portalP,
		cashshopP:      h.cashshopP,
		systemMessageP: h.systemMessageP,
		questP:         h.questP,
		storageP:       storageP,
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
	case AwardCurrency:
		return h.handleAwardCurrency, true
	case DestroyAsset:
		return h.handleDestroyAsset, true
	case EquipAsset:
		return h.handleEquipAsset, true
	case UnequipAsset:
		return h.handleUnequipAsset, true
	case ChangeJob:
		return h.handleChangeJob, true
	case ChangeHair:
		return h.handleChangeHair, true
	case ChangeFace:
		return h.handleChangeFace, true
	case ChangeSkin:
		return h.handleChangeSkin, true
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
	case IncreaseBuddyCapacity:
		return h.handleIncreaseBuddyCapacity, true
	case GainCloseness:
		return h.handleGainCloseness, true
	case SpawnMonster:
		return h.handleSpawnMonster, true
	case CompleteQuest:
		return h.handleCompleteQuest, true
	case StartQuest:
		return h.handleStartQuest, true
	case ApplyConsumableEffect:
		return h.handleApplyConsumableEffect, true
	case SendMessage:
		return h.handleSendMessage, true
	case DepositToStorage:
		return h.handleDepositToStorage, true
	case UpdateStorageMesos:
		return h.handleUpdateStorageMesos, true
	case AwardFame:
		return h.handleAwardFame, true
	case ShowStorage:
		return h.handleShowStorage, true
	case AcceptToStorage:
		return h.handleAcceptToStorage, true
	case ReleaseFromCharacter:
		return h.handleReleaseFromCharacter, true
	case AcceptToCharacter:
		return h.handleAcceptToCharacter, true
	case ReleaseFromStorage:
		return h.handleReleaseFromStorage, true
	case AcceptToCashShop:
		return h.handleAcceptToCashShop, true
	case ReleaseFromCashShop:
		return h.handleReleaseFromCashShop, true
	case PlayPortalSound:
		return h.handlePlayPortalSound, true
	case ShowInfo:
		return h.handleShowInfo, true
	case ShowInfoText:
		return h.handleShowInfoText, true
	case UpdateAreaInfo:
		return h.handleUpdateAreaInfo, true
	case ShowHint:
		return h.handleShowHint, true
	case SetHP:
		return h.handleSetHP, true
	case BlockPortal:
		return h.handleBlockPortal, true
	case UnblockPortal:
		return h.handleUnblockPortal, true
	}
	return nil, false
}

// logActionError logs an error that occurred during action processing
func (h *HandlerImpl) logActionError(s Saga, st Step[any], err error, errorMsg string) {
	h.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        st.StepId(),
		"tenant_id":      h.t.Id().String(),
	}).WithError(err).Error(errorMsg)
}

// handleAwardAsset handles the AwardAsset and AwardInventory actions
func (h *HandlerImpl) handleAwardAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AwardItemActionPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, payload.Item.Expiration)

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
	payload, ok := st.Payload().(WarpToRandomPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	err := h.charP.WarpRandomAndEmit(s.TransactionId(), payload.CharacterId, f)

	if err != nil {
		h.logActionError(s, st, err, "Unable to warp to random portal.")
		return err
	}

	return nil
}

// handleWarpToPortal handles the WarpToPortal action
func (h *HandlerImpl) handleWarpToPortal(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(WarpToPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	// Determine portal provider: use name-based lookup if PortalName is provided, otherwise use PortalId
	var portalProvider model.Provider[uint32]
	if payload.PortalName != "" && h.portalP != nil {
		portalProvider = h.portalP.ByNameIdProvider(f.MapId(), payload.PortalName)
	} else {
		portalProvider = model.FixedProvider(payload.PortalId)
	}

	err := h.charP.WarpToPortalAndEmit(s.TransactionId(), payload.CharacterId, f, portalProvider)

	if err != nil {
		h.logActionError(s, st, err, "Unable to warp to specific portal.")
		return err
	}

	return nil
}

// handleAwardExperience handles the AwardExperience action
func (h *HandlerImpl) handleAwardExperience(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AwardExperiencePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	eds := TransformExperienceDistributions(payload.Distributions)
	err := h.charP.AwardExperienceAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, eds)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award experience.")
		return err
	}

	return nil
}

// handleAwardLevel handles the AwardLevel action
func (h *HandlerImpl) handleAwardLevel(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AwardLevelPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.AwardLevelAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.Amount)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award level.")
		return err
	}

	return nil
}

// handleAwardMesos handles the AwardMesos action
func (h *HandlerImpl) handleAwardMesos(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AwardMesosPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.AwardMesosAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.ActorId, payload.ActorType, payload.Amount)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award mesos.")
		return err
	}

	return nil
}

// handleAwardCurrency handles the AwardCurrency action for cash shop currency
func (h *HandlerImpl) handleAwardCurrency(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AwardCurrencyPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.cashshopP.AwardCurrencyAndEmit(s.TransactionId(), payload.AccountId, payload.CurrencyType, payload.Amount)

	if err != nil {
		h.logActionError(s, st, err, "Unable to award currency.")
		return err
	}

	return nil
}

// handleDestroyAsset handles the DestroyAsset action
func (h *HandlerImpl) handleDestroyAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(DestroyAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, payload.Quantity, payload.RemoveAll)

	if err != nil {
		h.logActionError(s, st, err, "Unable to destroy asset.")
		return err
	}

	return nil
}

// handleEquipAsset handles the EquipAsset action
func (h *HandlerImpl) handleEquipAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(EquipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestEquipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Source, payload.Destination)

	if err != nil {
		h.logActionError(s, st, err, "Unable to equip asset.")
		return err
	}

	return nil
}

// handleUnequipAsset handles the UnequipAsset action
func (h *HandlerImpl) handleUnequipAsset(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(UnequipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestUnequipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Source, payload.Destination)

	if err != nil {
		h.logActionError(s, st, err, "Unable to unequip asset.")
		return err
	}

	return nil
}

// handleChangeJob handles the ChangeJob action
func (h *HandlerImpl) handleChangeJob(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ChangeJobPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.ChangeJobAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.JobId)

	if err != nil {
		h.logActionError(s, st, err, "Unable to change job.")
		return err
	}

	return nil
}

// handleChangeHair handles the ChangeHair action
func (h *HandlerImpl) handleChangeHair(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ChangeHairPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.ChangeHairAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.StyleId)

	if err != nil {
		h.logActionError(s, st, err, "Unable to change hair.")
		return err
	}

	return nil
}

// handleChangeFace handles the ChangeFace action
func (h *HandlerImpl) handleChangeFace(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ChangeFacePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.ChangeFaceAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.StyleId)

	if err != nil {
		h.logActionError(s, st, err, "Unable to change face.")
		return err
	}

	return nil
}

// handleChangeSkin handles the ChangeSkin action
func (h *HandlerImpl) handleChangeSkin(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ChangeSkinPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.ChangeSkinAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.StyleId)

	if err != nil {
		h.logActionError(s, st, err, "Unable to change skin.")
		return err
	}

	return nil
}

// handleCreateSkill handles the CreateSkill action
func (h *HandlerImpl) handleCreateSkill(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(CreateSkillPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.skillP.RequestCreateAndEmit(s.TransactionId(), payload.CharacterId, payload.SkillId, payload.Level, payload.MasterLevel, payload.Expiration)

	if err != nil {
		h.logActionError(s, st, err, "Unable to create skill.")
		return err
	}

	return nil
}

// handleUpdateSkill handles the UpdateSkill action
func (h *HandlerImpl) handleUpdateSkill(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(UpdateSkillPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.skillP.RequestUpdateAndEmit(s.TransactionId(), payload.CharacterId, payload.SkillId, payload.Level, payload.MasterLevel, payload.Expiration)

	if err != nil {
		h.logActionError(s, st, err, "Unable to update skill.")
		return err
	}

	return nil
}

func (h *HandlerImpl) handleIncreaseBuddyCapacity(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(IncreaseBuddyCapacityPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.buddyListP.IncreaseCapacityAndEmit(s.TransactionId(), payload.CharacterId, payload.WorldId, payload.Amount)

	if err != nil {
		h.logActionError(s, st, err, "Unable to increase buddy capacity.")
		return err
	}

	return nil
}

func (h *HandlerImpl) handleGainCloseness(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(GainClosenessPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.petP.GainClosenessAndEmit(s.TransactionId(), payload.PetId, payload.Amount)

	if err != nil {
		h.logActionError(s, st, err, "Unable to gain pet closeness.")
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
	payload, ok := st.Payload().(ValidateCharacterStatePayload)
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
	payload, ok := st.Payload().(RequestGuildNamePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestName(s.TransactionId(), payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild name.")
		return err
	}

	return nil
}

// handleRequestGuildEmblem handles the RequestGuildEmblem action
func (h *HandlerImpl) handleRequestGuildEmblem(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload().(RequestGuildEmblemPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestEmblem(s.TransactionId(), payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild emblem.")
		return err
	}

	return nil
}

// handleRequestGuildDisband handles the RequestGuildDisband action
func (h *HandlerImpl) handleRequestGuildDisband(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload().(RequestGuildDisbandPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestDisband(s.TransactionId(), payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild disband.")
		return err
	}

	return nil
}

// handleRequestGuildCapacityIncrease handles the RequestGuildCapacityIncrease action
func (h *HandlerImpl) handleRequestGuildCapacityIncrease(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload().(RequestGuildCapacityIncreasePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := h.guildP.RequestCapacityIncrease(s.TransactionId(), payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request guild capacity increase.")
		return err
	}

	return nil
}

// handleCreateInvite handles the CreateInvite action
func (h *HandlerImpl) handleCreateInvite(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload().(CreateInvitePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the invite processor
	err := h.inviteP.Create(s.TransactionId(), payload.InviteType, payload.OriginatorId, payload.WorldId, payload.ReferenceId, payload.TargetId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to create invitation.")
		return err
	}

	return nil
}

// handleCreateCharacter handles the CreateCharacter action
func (h *HandlerImpl) handleCreateCharacter(s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload().(CharacterCreatePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the character processor
	err := h.charP.RequestCreateCharacter(s.TransactionId(), payload.AccountId, payload.WorldId, payload.Name, payload.Level, payload.Strength, payload.Dexterity, payload.Intelligence, payload.Luck, payload.Hp, payload.Mp, payload.JobId, payload.Gender, payload.Face, payload.Hair, payload.Skin, payload.MapId)
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
	payload, ok := st.Payload().(CreateAndEquipAssetPayload)
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

	err := h.compP.RequestCreateAndEquipAsset(s.TransactionId(), compartmentPayload)
	if err != nil {
		h.logActionError(s, st, err, "Unable to create asset for create_and_equip_asset.")
		return err
	}

	// Note: Step 2 (dynamic equip_asset step creation) will be handled by the compartment consumer
	// when it receives the StatusEventTypeCreated event from the compartment service.
	// The consumer will detect this is a CreateAndEquipAsset step and add the equip_asset step.

	return nil
}

// handleSpawnMonster handles the SpawnMonster action
func (h *HandlerImpl) handleSpawnMonster(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(SpawnMonsterPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Look up foothold from atlas-data
	fh, err := h.footholdP.GetFootholdBelow(payload.MapId, payload.X, payload.Y)
	if err != nil {
		h.l.WithError(err).Warnf("Failed to get foothold for map %d at (%d, %d), using fh=0", payload.MapId, payload.X, payload.Y)
		fh = 0
	}

	// Determine spawn count (default to 1 if not specified)
	count := payload.Count
	if count <= 0 {
		count = 1
	}

	// Spawn monsters
	for i := 0; i < count; i++ {
		err := h.monsterP.SpawnMonster(payload.WorldId, payload.ChannelId, payload.MapId, payload.MonsterId, payload.X, payload.Y, int16(fh), payload.Team)
		if err != nil {
			h.logActionError(s, st, err, fmt.Sprintf("Failed to spawn monster %d/%d", i+1, count))
			return err
		}
	}

	h.l.Debugf("Successfully spawned %d monsters (id=%d) at (%d, %d, fh=%d) in world %d, channel %d, map %d",
		count, payload.MonsterId, payload.X, payload.Y, fh, payload.WorldId, payload.ChannelId, payload.MapId)

	return nil
}

// handleCompleteQuest handles the CompleteQuest action
func (h *HandlerImpl) handleCompleteQuest(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(CompleteQuestPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Selection is not currently used in NPC conversations, default to 0
	err := h.questP.RequestCompleteQuest(byte(payload.WorldId), payload.CharacterId, payload.QuestId, payload.NpcId, 0, payload.Force)
	if err != nil {
		h.logActionError(s, st, err, "Unable to complete quest.")
		return err
	}

	return nil
}

// handleStartQuest handles the StartQuest action
func (h *HandlerImpl) handleStartQuest(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(StartQuestPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.questP.RequestStartQuest(byte(payload.WorldId), payload.CharacterId, payload.QuestId, payload.NpcId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to start quest.")
		return err
	}

	return nil
}

// handleApplyConsumableEffect handles the ApplyConsumableEffect action
// This applies consumable item effects to a character without consuming from inventory
func (h *HandlerImpl) handleApplyConsumableEffect(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ApplyConsumableEffectPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.consumableP.ApplyConsumableEffect(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId, payload.ItemId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to apply consumable effect.")
		return err
	}

	return nil
}

// handleSendMessage handles the SendMessage action
// This sends a system message to a character (e.g., "You have acquired a Dragon Egg.")
func (h *HandlerImpl) handleSendMessage(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(SendMessagePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.systemMessageP.SendMessage(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId, payload.MessageType, payload.Message)
	if err != nil {
		h.logActionError(s, st, err, "Unable to send message.")
		return err
	}

	return nil
}

// handleDepositToStorage handles the DepositToStorage action
// This deposits an item into account storage
func (h *HandlerImpl) handleDepositToStorage(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(DepositToStoragePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	refData := storage2.ReferenceData{
		Quantity: payload.Quantity,
		OwnerId:  payload.OwnerId,
		Flag:     payload.Flag,
	}

	err := h.storageP.DepositAndEmit(s.TransactionId(), payload.WorldId, payload.AccountId, payload.Slot, payload.TemplateId, payload.Expiration, payload.ReferenceId, payload.ReferenceType, refData)
	if err != nil {
		h.logActionError(s, st, err, "Unable to deposit to storage.")
		return err
	}

	return nil
}

// handleUpdateStorageMesos handles the UpdateStorageMesos action
// This updates the mesos in account storage
func (h *HandlerImpl) handleUpdateStorageMesos(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(UpdateStorageMesosPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.storageP.UpdateMesosAndEmit(s.TransactionId(), payload.WorldId, payload.AccountId, payload.Mesos, payload.Operation)
	if err != nil {
		h.logActionError(s, st, err, "Unable to update storage mesos.")
		return err
	}

	return nil
}

// handleAwardFame handles the AwardFame action
// This awards fame to a character (e.g., qm.gainFame() in quest scripts)
func (h *HandlerImpl) handleAwardFame(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AwardFamePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.AwardFameAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.Amount)
	if err != nil {
		h.logActionError(s, st, err, "Unable to award fame.")
		return err
	}

	return nil
}

// handleShowStorage handles the ShowStorage action
// This sends a command to the channel service to display the storage UI to the character
func (h *HandlerImpl) handleShowStorage(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ShowStoragePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.storageP.ShowStorageAndEmit(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId, payload.NpcId, payload.AccountId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to show storage.")
		return err
	}

	// ShowStorage is a synchronous command with no async response event
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}

// handleAcceptToStorage handles the AcceptToStorage action
// This sends an ACCEPT command to storage compartment with pre-populated asset data
func (h *HandlerImpl) handleAcceptToStorage(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AcceptToStoragePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.Debugf("Accepting asset template [%d] to storage for account [%d]", payload.TemplateId, payload.AccountId)

	// Send ACCEPT command to storage with pre-populated asset data
	err := h.storageP.AcceptAndEmit(
		payload.TransactionId,
		payload.WorldId,
		payload.AccountId,
		payload.CharacterId,
		-1, // Storage auto-assigns slot
		payload.TemplateId,
		payload.ReferenceId,
		payload.ReferenceType,
		payload.ReferenceData,
		payload.Quantity,
	)

	if err != nil {
		h.logActionError(s, st, err, "Unable to accept asset to storage.")
		return err
	}

	return nil
}

// handleReleaseFromCharacter handles the ReleaseFromCharacter action
// This sends a RELEASE command to character inventory compartment
func (h *HandlerImpl) handleReleaseFromCharacter(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ReleaseFromCharacterPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.compP.RequestReleaseAsset(
		payload.TransactionId,
		payload.CharacterId,
		payload.InventoryType,
		payload.AssetId,
		payload.Quantity,
	)

	if err != nil {
		h.logActionError(s, st, err, "Unable to release asset from character.")
		return err
	}

	return nil
}

// handleAcceptToCharacter handles the AcceptToCharacter action
// This sends an ACCEPT command to character inventory compartment with pre-populated asset data
func (h *HandlerImpl) handleAcceptToCharacter(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AcceptToCharacterPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.Debugf("Accepting asset template [%d] to character [%d] inventory", payload.TemplateId, payload.CharacterId)

	// Send ACCEPT command to character inventory with pre-populated asset data
	err := h.compP.RequestAcceptAsset(
		payload.TransactionId,
		payload.CharacterId,
		payload.InventoryType,
		payload.TemplateId,
		payload.ReferenceId,
		payload.ReferenceType,
		payload.ReferenceData,
		payload.Quantity,
	)

	if err != nil {
		h.logActionError(s, st, err, "Unable to accept asset to character.")
		return err
	}

	return nil
}

// handleReleaseFromStorage handles the ReleaseFromStorage action
// This sends a RELEASE command to storage compartment
func (h *HandlerImpl) handleReleaseFromStorage(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ReleaseFromStoragePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.storageP.ReleaseAndEmit(
		payload.TransactionId,
		payload.WorldId,
		payload.AccountId,
		payload.CharacterId,
		payload.AssetId,
		payload.Quantity,
	)

	if err != nil {
		h.logActionError(s, st, err, "Unable to release asset from storage.")
		return err
	}

	return nil
}

// handleAcceptToCashShop handles the AcceptToCashShop action
// This sends an ACCEPT command to cash shop compartment with pre-populated asset data
func (h *HandlerImpl) handleAcceptToCashShop(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(AcceptToCashShopPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.Debugf("Accepting asset template [%d] with cashId [%d] to cash shop compartment [%s] for account [%d] character [%d]",
		payload.TemplateId, payload.CashId, payload.CompartmentId, payload.AccountId, payload.CharacterId)

	// Send ACCEPT command to cash shop compartment with pre-populated asset data (including preserved cashId)
	err := h.cashshopP.AcceptAndEmit(
		payload.TransactionId,
		payload.CharacterId,
		payload.AccountId,
		payload.CompartmentId,
		payload.CompartmentType,
		payload.CashId,
		payload.TemplateId,
		payload.ReferenceId,
		payload.ReferenceType,
		payload.ReferenceData,
	)

	if err != nil {
		h.logActionError(s, st, err, "Unable to accept asset to cash shop.")
		return err
	}

	return nil
}

// handleReleaseFromCashShop handles the ReleaseFromCashShop action
// This sends a RELEASE command to cash shop compartment
func (h *HandlerImpl) handleReleaseFromCashShop(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ReleaseFromCashShopPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.Debugf("Releasing asset [%d] from cash shop compartment [%s] for account [%d] character [%d]",
		payload.AssetId, payload.CompartmentId, payload.AccountId, payload.CharacterId)

	err := h.cashshopP.ReleaseAndEmit(
		payload.TransactionId,
		payload.CharacterId,
		payload.AccountId,
		payload.CompartmentId,
		payload.CompartmentType,
		payload.AssetId,
		payload.CashId,
		payload.TemplateId,
	)

	if err != nil {
		h.logActionError(s, st, err, "Unable to release asset from cash shop.")
		return err
	}

	return nil
}

// handlePlayPortalSound handles the PlayPortalSound action
// This is a synchronous action - we send the command and immediately mark complete
func (h *HandlerImpl) handlePlayPortalSound(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(PlayPortalSoundPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.systemMessageP.PlayPortalSound(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to play portal sound.")
		return err
	}

	// PlayPortalSound is a synchronous command with no async response event
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}

// handleShowInfo handles the ShowInfo action
// This is a synchronous action - we send the command and immediately mark complete
func (h *HandlerImpl) handleShowInfo(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ShowInfoPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.systemMessageP.ShowInfo(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId, payload.Path)
	if err != nil {
		h.logActionError(s, st, err, "Unable to show info.")
		return err
	}

	// ShowInfo is a synchronous command with no async response event
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}

// handleShowInfoText handles the ShowInfoText action
// This is a synchronous action - we send the command and immediately mark complete
func (h *HandlerImpl) handleShowInfoText(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ShowInfoTextPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.systemMessageP.ShowInfoText(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId, payload.Text)
	if err != nil {
		h.logActionError(s, st, err, "Unable to show info text.")
		return err
	}

	// ShowInfoText is a synchronous command with no async response event
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}

// handleUpdateAreaInfo handles the UpdateAreaInfo action
// This is a synchronous action - we send the command and immediately mark complete
func (h *HandlerImpl) handleUpdateAreaInfo(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(UpdateAreaInfoPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.systemMessageP.UpdateAreaInfo(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId, payload.Area, payload.Info)
	if err != nil {
		h.logActionError(s, st, err, "Unable to update area info.")
		return err
	}

	// UpdateAreaInfo is a synchronous command with no async response event
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}

// handleShowHint handles the ShowHint action
// This is a synchronous action - we send the command and immediately mark complete
func (h *HandlerImpl) handleShowHint(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ShowHintPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.systemMessageP.ShowHint(s.TransactionId(), byte(payload.WorldId), byte(payload.ChannelId), payload.CharacterId, payload.Hint, payload.Width, payload.Height)
	if err != nil {
		h.logActionError(s, st, err, "Unable to show hint.")
		return err
	}

	// ShowHint is a synchronous command with no async response event
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}

// handleSetHP handles the SetHP action
// This is an asynchronous action - we send the command and wait for the status event
func (h *HandlerImpl) handleSetHP(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(SetHPPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.charP.SetHPAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.ChannelId, payload.Amount)
	if err != nil {
		h.logActionError(s, st, err, "Unable to set HP.")
		return err
	}

	// SetHP is an asynchronous command - the consumer will mark completion when the status event arrives
	return nil
}

// handleBlockPortal handles the BlockPortal action
// This is a synchronous action - we send the event and immediately mark complete
// The portal will remain blocked until character logout or explicit unblock
func (h *HandlerImpl) handleBlockPortal(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(BlockPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.portalBlockingP.BlockAndEmit(payload.CharacterId, payload.MapId, payload.PortalId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to block portal.")
		return err
	}

	// BlockPortal is a synchronous command with no async response
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}

// handleUnblockPortal handles the UnblockPortal action
// This is a synchronous action - we send the command and immediately mark complete
func (h *HandlerImpl) handleUnblockPortal(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(UnblockPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.portalBlockingP.UnblockAndEmit(payload.CharacterId, payload.MapId, payload.PortalId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to unblock portal.")
		return err
	}

	// UnblockPortal is a synchronous command with no async response
	// Mark the step as completed immediately after successfully sending the command
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)

	return nil
}
