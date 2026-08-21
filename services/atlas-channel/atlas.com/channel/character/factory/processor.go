package factory

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Processor issues the character-factory seed request. SeedCharacter returns
// the transactionId the factory assigns to the seed operation -- Task 14's
// consumer correlates the asynchronous seed-status event back to the pending
// maplelife dialog through exactly that value, so unlike atlas-login's
// SeedCharacter (services/atlas-login/atlas.com/login/character/factory/processor.go),
// this one must not discard it.
type Processor interface {
	SeedCharacter(accountId uint32, worldId world.Id, name string, jobIndex uint32, subJobIndex uint16, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32, strength byte, dexterity byte, intelligence byte, luck byte) (string, error)

	// CreateMapleLife creates a level-30 first-job character through the
	// factory's Maple Life path. Unlike SeedCharacter it forwards only what
	// the PLAYER chose -- the class ordinal, the four look values, the
	// gender and the SP level -- because the factory owns what a Maple Life
	// character of that class actually is (design.md §11 A5).
	CreateMapleLife(accountId uint32, worldId world.Id, name string, classOrdinal uint32, gender byte, face uint32, hair uint32, hairColor uint32, skinColor byte, sp byte) (string, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) SeedCharacter(accountId uint32, worldId world.Id, name string, jobIndex uint32, subJobIndex uint16,
	face uint32, hair uint32, color uint32, skinColor uint32, gender byte,
	top uint32, bottom uint32, shoes uint32, weapon uint32,
	strength byte, dexterity byte, intelligence byte, luck byte,
) (string, error) {
	p.l.Debugf("Create character [%s] with job [%d:%d] and gender [%d].", name, jobIndex, subJobIndex, gender)
	p.l.Debugf("Face [%d], Hair [%d], HairColor [%d] SkinColor [%d].", face, hair, color, skinColor)
	p.l.Debugf("Top [%d], Bottom [%d], Shoes [%d], Weapon [%d].", top, bottom, shoes, weapon)
	p.l.Debugf("Strength [%d], Dexterity [%d], Intelligence [%d], Luck [%d].", strength, dexterity, intelligence, luck)
	resp, err := requestCreate(p.ctx, accountId, worldId, name, jobIndex, subJobIndex, face, hair, color, skinColor, gender, top, bottom, shoes, weapon, strength, dexterity, intelligence, luck)(p.l, p.ctx)
	if err != nil {
		return "", err
	}
	return resp.TransactionId, nil
}

func (p *ProcessorImpl) CreateMapleLife(accountId uint32, worldId world.Id, name string, classOrdinal uint32, gender byte, face uint32, hair uint32, hairColor uint32, skinColor byte, sp byte) (string, error) {
	p.l.Debugf("Create maple life character [%s] with class ordinal [%d] and gender [%d].", name, classOrdinal, gender)
	p.l.Debugf("Face [%d], Hair [%d], HairColor [%d] SkinColor [%d], SP [%d].", face, hair, hairColor, skinColor, sp)
	resp, err := requestCreateMapleLife(p.ctx, accountId, worldId, name, classOrdinal, gender, face, hair, hairColor, skinColor, sp)(p.l, p.ctx)
	if err != nil {
		return "", err
	}
	return resp.TransactionId, nil
}
