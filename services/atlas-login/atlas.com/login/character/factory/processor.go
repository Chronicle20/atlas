package factory

import (
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	SeedCharacter(accountId uint32, worldId world.Id, name string, jobIndex uint32, subJobIndex uint16, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32, strength byte, dexterity byte, intelligence byte, luck byte) error
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

func (p *ProcessorImpl) SeedCharacter(accountId uint32, worldId world.Id, name string, jobIndex uint32, subJobIndex uint16,
	face uint32, hair uint32, color uint32, skinColor uint32, gender byte,
	top uint32, bottom uint32, shoes uint32, weapon uint32,
	strength byte, dexterity byte, intelligence byte, luck byte) error {
	p.l.Debugf("Create character [%s] with job [%d:%d] and gender [%d].", name, jobIndex, subJobIndex, gender)
	p.l.Debugf("Face [%d], Hair [%d], HairColor [%d] SkinColor [%d].", face, hair, color, skinColor)
	p.l.Debugf("Top [%d], Bottom [%d], Shoes [%d], Weapon [%d].", top, bottom, shoes, weapon)
	p.l.Debugf("Strength [%d], Dexterity [%d], Intelligence [%d], Luck [%d].", strength, dexterity, intelligence, luck)
	_, err := requestCreate(accountId, worldId, name, jobIndex, subJobIndex, face, hair, color, skinColor, gender, top, bottom, shoes, weapon, strength, dexterity, intelligence, luck)(p.l, p.ctx)
	return err
}
