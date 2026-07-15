package quest

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Register(s *document.Storage[string, RestModel], quest RestModel) error
	RegisterQuest(path string) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "QUEST")
}

func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], quest RestModel) error {
	_, err := s.Add(p.ctx)(quest)()
	if err != nil {
		return err
	}
	return nil
}

// RegisterQuest registers all quests from the Quest.wz directory
// Quest data is spread across three files: QuestInfo.img.xml, Check.img.xml, and Act.img.xml
func (p *ProcessorImpl) RegisterQuest(path string) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		s := NewStorage(p.l, tx)

		// Step 1: Read QuestInfo.img.xml to get base quest data
		questInfoPath := filepath.Join(path, "QuestInfo.img.xml")
		quests := ReadQuestInfo(p.l)(xml.FromPathProvider(questInfoPath))
		p.l.Debugf("Read %d quests from QuestInfo.img.xml", len(quests))

		// Step 2: Read Check.img.xml to add requirements
		checkPath := filepath.Join(path, "Check.img.xml")
		quests = ReadQuestCheck(p.l)(xml.FromPathProvider(checkPath))(quests)
		p.l.Debugf("Merged Check.img.xml data")

		// Step 3: Read Act.img.xml to add actions
		actPath := filepath.Join(path, "Act.img.xml")
		quests = ReadQuestAct(p.l)(xml.FromPathProvider(actPath))(quests)
		p.l.Debugf("Merged Act.img.xml data")

		// Step 4: Register all quests
		count := 0
		for _, quest := range quests {
			if err := p.Register(s, quest); err != nil {
				p.l.WithError(err).Errorf("Failed to register quest %d", quest.Id)
				continue
			}
			count++
		}

		p.l.Infof("Registered %d quests", count)
		return nil
	})
}
