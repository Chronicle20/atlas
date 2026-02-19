package quest

import (
	database "github.com/Chronicle20/atlas-database"
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "QUEST")
}

func Register(s *document.Storage[string, RestModel]) func(ctx context.Context) func(quest RestModel) error {
	return func(ctx context.Context) func(quest RestModel) error {
		return func(quest RestModel) error {
			_, err := s.Add(ctx)(quest)()
			if err != nil {
				return err
			}
			return nil
		}
	}
}

// RegisterQuest registers all quests from the Quest.wz directory
// Quest data is spread across three files: QuestInfo.img.xml, Check.img.xml, and Act.img.xml
func RegisterQuest(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
		return func(ctx context.Context) func(path string) error {
			return func(path string) error {
				return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
					s := NewStorage(l, tx)

					// Step 1: Read QuestInfo.img.xml to get base quest data
					questInfoPath := filepath.Join(path, "QuestInfo.img.xml")
					quests := ReadQuestInfo(l)(xml.FromPathProvider(questInfoPath))
					l.Debugf("Read %d quests from QuestInfo.img.xml", len(quests))

					// Step 2: Read Check.img.xml to add requirements
					checkPath := filepath.Join(path, "Check.img.xml")
					quests = ReadQuestCheck(l)(xml.FromPathProvider(checkPath))(quests)
					l.Debugf("Merged Check.img.xml data")

					// Step 3: Read Act.img.xml to add actions
					actPath := filepath.Join(path, "Act.img.xml")
					quests = ReadQuestAct(l)(xml.FromPathProvider(actPath))(quests)
					l.Debugf("Merged Act.img.xml data")

					// Step 4: Register all quests
					count := 0
					for _, quest := range quests {
						if err := Register(s)(ctx)(quest); err != nil {
							l.WithError(err).Errorf("Failed to register quest %d", quest.Id)
							continue
						}
						count++
					}

					l.Infof("Registered %d quests", count)
					return nil
				})
			}
		}
	}
}
