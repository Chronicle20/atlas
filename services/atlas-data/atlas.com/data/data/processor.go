package data

import (
	"atlas-data/cash"
	"atlas-data/characters/templates"
	"atlas-data/commodity"
	"atlas-data/consumable"
	"atlas-data/cosmetic/face"
	"atlas-data/cosmetic/hair"
	"atlas-data/equipment"
	"atlas-data/etc"
	"atlas-data/item"
	"atlas-data/kafka/producer"
	_map "atlas-data/map"
	"atlas-data/mobskill"
	"atlas-data/monster"
	"atlas-data/npc"
	"atlas-data/pet"
	"atlas-data/quest"
	"atlas-data/reactor"
	"atlas-data/setup"
	"atlas-data/skill"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	WorkerMap               = "MAP"
	WorkerMonster           = "MONSTER"
	WorkerCharacter         = "CHARACTER"
	WorkerReactor           = "REACTOR"
	WorkerSkill             = "SKILL"
	WorkerPet               = "PET"
	WorkerConsume           = "CONSUME"
	WorkerCash              = "CASH"
	WorkerCommodity         = "COMMODITY"
	WorkerEtc               = "ETC"
	WorkerSetup             = "SETUP"
	WorkerCharacterCreation = "CHARACTER_CREATION"
	WorkerQuest             = "QUEST"
	WorkerNPC               = "NPC"
	WorkerFace              = "FACE"
	WorkerHair              = "HAIR"
	WorkerMobSkill          = "MOB_SKILL"
)

var Workers = []string{WorkerMap, WorkerMonster, WorkerCharacter, WorkerReactor, WorkerSkill, WorkerPet, WorkerConsume, WorkerCash, WorkerCommodity, WorkerEtc, WorkerSetup, WorkerCharacterCreation, WorkerQuest, WorkerNPC, WorkerFace, WorkerHair, WorkerMobSkill}

func ProcessData(l logrus.FieldLogger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		t := tenant.MustFromContext(ctx)
		dataDir := os.Getenv("ZIP_DIR")
		path := filepath.Join(dataDir, t.Id().String(), t.Region(), fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()))

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("data path does not exist: %s", path)
		}

		l.Infof("Processing data from [%s] for tenant [%s].", path, t.Id().String())

		for _, wn := range Workers {
			err := InstructWorker(l)(ctx)(wn, path)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func InstructWorker(l logrus.FieldLogger) func(ctx context.Context) func(workerName string, path string) error {
	return func(ctx context.Context) func(workerName string, path string) error {
		return func(workerName string, path string) error {
			l.Debugf("Sending notification to start worker [%s] at [%s].", workerName, path)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(startWorkerCommandProvider(workerName, path))
		}
	}
}

func StartWorker(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(name string, path string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(name string, path string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(name string, path string) error {
			return func(name string, path string) error {
				l.Infof("Starting worker [%s] at [%s].", name, path)
				var err error
				if name == WorkerMap {
					if err = _map.InitString(t, filepath.Join(path, "String.wz", "Map.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize map string registry.")
						return err
					}
					if err = npc.InitString(t, filepath.Join(path, "String.wz", "Npc.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize NPC string registry for map worker.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Map.wz", "Map"), _map.RegisterMap(db))()
					_ = _map.GetMapStringRegistry().Clear(t)
					// Note: Don't clear NPC registry here - WorkerNPC may run concurrently and needs it
				} else if name == WorkerMonster {
					if err = monster.InitString(t, filepath.Join(path, "String.wz", "Mob.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize monster string registry.")
						return err
					}
					if err = monster.InitGauge(t, filepath.Join(path, "UI.wz", "UIWindow.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize monster gauge registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, "Mob.wz", monster.RegisterMonster(db))()
					_ = monster.GetMonsterStringRegistry().Clear(t)
					_ = monster.GetMonsterGaugeRegistry().Clear(t)
				} else if name == WorkerCharacter {
					if err = item.InitStringNested(db)(l)(ctx)(filepath.Join(path, "String.wz", "Eqp.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize equipment item string registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, "Character.wz", equipment.RegisterEquipment(db))()
				} else if name == WorkerReactor {
					err = RegisterAllData(l)(ctx)(path, "Reactor.wz", reactor.RegisterReactor(db))()
				} else if name == WorkerSkill {
					if err = skill.InitString(t, filepath.Join(path, "String.wz", "Skill.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize skill string registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, "Skill.wz", skill.RegisterSkill(db))()
					_ = skill.GetSkillStringRegistry().Clear(t)
				} else if name == WorkerPet {
					if err = item.InitStringFlat(db)(l)(ctx)(filepath.Join(path, "String.wz", "Pet.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize pet item string registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Pet"), pet.RegisterPet(db))()
				} else if name == WorkerConsume {
					if err = item.InitStringFlat(db)(l)(ctx)(filepath.Join(path, "String.wz", "Consume.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize consumable item string registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Consume"), consumable.RegisterConsumable(db))()
				} else if name == WorkerCash {
					if err = item.InitStringFlat(db)(l)(ctx)(filepath.Join(path, "String.wz", "Cash.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize cash item string registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Cash"), cash.RegisterCash(db))()
				} else if name == WorkerCommodity {
					err = RegisterFileData(l)(ctx)(path, filepath.Join("Etc.wz", "Commodity.img.xml"), commodity.RegisterCommodity(db))()
				} else if name == WorkerEtc {
					if err = item.InitStringFlat(db)(l)(ctx)(filepath.Join(path, "String.wz", "Etc.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize etc item string registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Etc"), etc.RegisterEtc(db))()
				} else if name == WorkerSetup {
					if err = item.InitStringFlat(db)(l)(ctx)(filepath.Join(path, "String.wz", "Ins.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize setup item string registry.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Install"), setup.RegisterSetup(db))()
				} else if name == WorkerCharacterCreation {
					err = RegisterFileData(l)(ctx)(path, filepath.Join("Etc.wz", "MakeCharInfo.img.xml"), templates.RegisterCharacterTemplate(db))()
				} else if name == WorkerQuest {
					err = quest.RegisterQuest(db)(l)(ctx)(filepath.Join(path, "Quest.wz"))
				} else if name == WorkerNPC {
					if err = npc.InitString(t, filepath.Join(path, "String.wz", "Npc.img.xml")); err != nil {
						l.WithError(err).Errorf("Failed to initialize NPC string registry for NPC worker.")
						return err
					}
					err = RegisterAllData(l)(ctx)(path, "Npc.wz", npc.RegisterNpc(db))()
					// Note: Don't clear NPC registry - WorkerMap may run concurrently and needs it
				} else if name == WorkerFace {
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Character.wz", "Face"), face.RegisterFace(db))()
				} else if name == WorkerHair {
					err = RegisterAllData(l)(ctx)(path, filepath.Join("Character.wz", "Hair"), hair.RegisterHair(db))()
				} else if name == WorkerMobSkill {
					err = RegisterFileData(l)(ctx)(path, filepath.Join("Skill.wz", "MobSkill.img.xml"), mobskill.RegisterMobSkill(db))()
				}
				if err != nil {
					l.WithError(err).Errorf("Worker [%s] failed with error.", name)
					return err
				}
				l.Infof("Worker [%s] completed.", name)
				return nil
			}
		}
	}
}

type Worker func() error
type RegisterFunc func(l logrus.FieldLogger) func(ctx context.Context) func(filePath string) error

func RegisterAllData(l logrus.FieldLogger) func(ctx context.Context) func(rootDir string, wzFilePath string, rf RegisterFunc) Worker {
	return func(ctx context.Context) func(rootDir string, wzFileName string, rf RegisterFunc) Worker {
		return func(rootDir string, wzFileName string, rf RegisterFunc) Worker {
			return func() error {
				baseDir := filepath.Join(rootDir, wzFileName)
				if _, err := os.Stat(baseDir); os.IsNotExist(err) {
					l.Debugf("Unable to locate directory. Expected [%s]", baseDir)
					return err
				}

				// Channel to collect file paths
				fileChan := make(chan string)
				errChan := make(chan error)
				var wg sync.WaitGroup

				// Start a worker pool for processing files
				const workerCount = 10 // Adjust based on your workload and system resources
				for i := 0; i < workerCount; i++ {
					wg.Add(1)
					go func() {
						for filePath := range fileChan {
							if err := rf(l)(ctx)(filePath); err != nil {
								errChan <- fmt.Errorf("error processing %s: %w", filePath, err)
							}
						}
						wg.Done()
					}()
				}

				// Start error collector
				var errors []error
				go func() {
					for err := range errChan {
						errors = append(errors, err)
					}
				}()

				// Walk directory and send files
				err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return fmt.Errorf("error accessing path %s: %w", path, err)
					}

					if d.IsDir() {
						return nil
					}

					fileChan <- path
					return nil
				})

				// Close the file channel after walking the directory
				if err != nil {
					fmt.Printf("Error walking directory: %v\n", err)
				}
				close(fileChan)

				// Wait for all workers to finish
				wg.Wait()

				return nil

			}
		}
	}
}

func RegisterFileData(l logrus.FieldLogger) func(ctx context.Context) func(rootDir string, wzFileName string, rf RegisterFunc) Worker {
	return func(ctx context.Context) func(rootDir string, wzFileName string, rf RegisterFunc) Worker {
		return func(rootDir string, wzFileName string, rf RegisterFunc) Worker {
			return func() error {
				rf(l)(ctx)(filepath.Join(rootDir, wzFileName))
				return nil
			}
		}
	}
}
