package recipe

import (
	"atlas-npc-conversations/conversation"
	"context"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SkippedRecipe captures a craftAction state that the rebuild could not
// materialise, e.g. because itemId did not parse or materials/quantities
// lengths disagreed.
type SkippedRecipe struct {
	NpcId   uint32 `json:"npcId"`
	StateId string `json:"stateId"`
	Reason  string `json:"reason"`
}

// RebuildResult summarises one RebuildForConversation call.
type RebuildResult struct {
	Inserted       int             `json:"inserted"`
	Skipped        int             `json:"skipped"`
	SkippedDetails []SkippedRecipe `json:"skippedDetails,omitempty"`
}

// ReindexResult summarises one ReindexAllRecipes call.
type ReindexResult struct {
	DeletedCount         int64           `json:"deletedCount"`
	InsertedCount        int             `json:"insertedCount"`
	SkippedCount         int             `json:"skippedCount"`
	SkippedDetails       []SkippedRecipe `json:"skippedDetails,omitempty"`
	ConversationsScanned int             `json:"conversationsScanned"`
}

// Processor exposes recipe operations to other packages.
type Processor interface {
	ByItemIdProvider(itemId uint32) model.Provider[[]Model]
	ByNpcIdProvider(npcId uint32) model.Provider[[]Model]
	RebuildForConversation(tx *gorm.DB) func(npcId uint32, conversationId uuid.UUID, states []conversation.StateModel) (RebuildResult, error)
	ClearForTenant(tx *gorm.DB) (int64, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		db:  db,
	}
}

func (p *ProcessorImpl) ByItemIdProvider(itemId uint32) model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getByItemIdProvider(itemId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getByNpcIdProvider(npcId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// ClearForTenant hard-deletes every recipe row for the active tenant. Returns
// the number of rows removed.
func (p *ProcessorImpl) ClearForTenant(tx *gorm.DB) (int64, error) {
	return deleteAllRecipes(tx.WithContext(p.ctx))
}

// RebuildForConversation clear-and-rebuilds the recipe rows derived from the
// given conversation states. Bad states are logged and recorded in
// RebuildResult.SkippedDetails rather than aborting the whole rebuild — but if
// the underlying database operations fail, the error is returned so the
// surrounding tx can roll back.
func (p *ProcessorImpl) RebuildForConversation(tx *gorm.DB) func(npcId uint32, conversationId uuid.UUID, states []conversation.StateModel) (RebuildResult, error) {
	return func(npcId uint32, conversationId uuid.UUID, states []conversation.StateModel) (RebuildResult, error) {
		txCtx := tx.WithContext(p.ctx)

		if _, err := deleteRecipesByConversation(txCtx)(conversationId); err != nil {
			return RebuildResult{}, err
		}

		var result RebuildResult
		for _, state := range states {
			if state.Type() != conversation.CraftActionType {
				continue
			}
			ca := state.CraftAction()
			if ca == nil {
				result.Skipped++
				result.SkippedDetails = append(result.SkippedDetails, SkippedRecipe{NpcId: npcId, StateId: state.Id(), Reason: "craftAction is nil"})
				p.l.WithField("npcId", npcId).WithField("stateId", state.Id()).Warn("Skipping recipe: craftAction is nil")
				continue
			}

			itemIdU64, err := strconv.ParseUint(ca.ItemId(), 10, 32)
			if err != nil {
				result.Skipped++
				result.SkippedDetails = append(result.SkippedDetails, SkippedRecipe{NpcId: npcId, StateId: state.Id(), Reason: "itemId not parseable as uint32"})
				p.l.WithField("npcId", npcId).WithField("stateId", state.Id()).WithField("itemId", ca.ItemId()).Warn("Skipping recipe: unparseable itemId")
				continue
			}

			mats := ca.Materials()
			qtys := ca.Quantities()
			if len(mats) != len(qtys) {
				result.Skipped++
				result.SkippedDetails = append(result.SkippedDetails, SkippedRecipe{NpcId: npcId, StateId: state.Id(), Reason: "materials/quantities length mismatch"})
				p.l.WithField("npcId", npcId).WithField("stateId", state.Id()).Warnf("Skipping recipe: materials(%d) vs quantities(%d) length mismatch", len(mats), len(qtys))
				continue
			}

			materials := make([]Material, 0, len(mats))
			for i := range mats {
				materials = append(materials, Material{ItemId: mats[i], Quantity: qtys[i]})
			}

			m, err := NewBuilder().
				SetTenantId(p.t.Id()).
				SetConversationId(conversationId).
				SetNpcId(npcId).
				SetStateId(state.Id()).
				SetItemId(uint32(itemIdU64)).
				SetMaterials(materials).
				SetMesoCost(ca.MesoCost()).
				SetStimulatorId(ca.StimulatorId()).
				SetStimulatorFailChance(ca.StimulatorFailChance()).
				Build()
			if err != nil {
				return result, err
			}

			if _, err := createRecipe(txCtx)(p.t.Id())(m); err != nil {
				return result, err
			}
			result.Inserted++
		}

		return result, nil
	}
}
