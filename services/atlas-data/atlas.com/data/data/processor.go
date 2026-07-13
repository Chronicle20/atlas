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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

type Processor interface {
	ProcessData() error
	InstructWorker(workerName string, path string) error
	StartWorker(name string, path string) error
	RegisterAllData(rootDir string, wzFileName string, rf RegisterFunc) Worker
	RegisterFileData(rootDir string, wzFileName string, rf RegisterFunc) Worker
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

func (p *ProcessorImpl) ProcessData() error {
	t := tenant.MustFromContext(p.ctx)
	dataDir := os.Getenv("ZIP_DIR")
	path := filepath.Join(dataDir, t.Id().String(), t.Region(), fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("data path does not exist: %s", path)
	}

	p.l.Infof("Processing data from [%s] for tenant [%s].", path, t.Id().String())

	for _, wn := range Workers {
		err := p.InstructWorker(wn, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) InstructWorker(workerName string, path string) error {
	p.l.Debugf("Sending notification to start worker [%s] at [%s].", workerName, path)
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(startWorkerCommandProvider(workerName, path))
}

func (p *ProcessorImpl) StartWorker(name string, path string) error {
	t := tenant.MustFromContext(p.ctx)
	p.l.Infof("Starting worker [%s] at [%s].", name, path)
	var err error
	if name == WorkerMap {
		if err = _map.InitString(t, filepath.Join(path, "String.wz", "Map.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize map string registry.")
			return err
		}
		if err = npc.InitString(t, filepath.Join(path, "String.wz", "Npc.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize NPC string registry for map worker.")
			return err
		}
		err = p.RegisterAllData(path, filepath.Join("Map.wz", "Map"), _map.NewProcessor(p.l, p.ctx, p.db).RegisterMap)()
		_ = _map.GetMapStringRegistry().Clear(t)
		// Note: Don't clear NPC registry here - WorkerNPC may run concurrently and needs it
	} else if name == WorkerMonster {
		if err = monster.InitString(t, filepath.Join(path, "String.wz", "Mob.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize monster string registry.")
			return err
		}
		if err = monster.InitGauge(t, filepath.Join(path, "UI.wz", "UIWindow.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize monster gauge registry.")
			return err
		}
		err = p.RegisterAllData(path, "Mob.wz", monster.NewProcessor(p.l, p.ctx, p.db).RegisterMonster)()
		_ = monster.GetMonsterStringRegistry().Clear(t)
		_ = monster.GetMonsterGaugeRegistry().Clear(t)
	} else if name == WorkerCharacter {
		if err = item.InitStringNested(p.db)(p.l)(p.ctx)(filepath.Join(path, "String.wz", "Eqp.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize equipment item string registry.")
			return err
		}
		err = p.RegisterAllData(path, "Character.wz", equipment.NewProcessor(p.l, p.ctx, p.db).RegisterEquipment)()
	} else if name == WorkerReactor {
		err = p.RegisterAllData(path, "Reactor.wz", reactor.NewProcessor(p.l, p.ctx, p.db).RegisterReactor)()
	} else if name == WorkerSkill {
		if err = skill.InitString(t, filepath.Join(path, "String.wz", "Skill.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize skill string registry.")
			return err
		}
		err = p.RegisterAllData(path, "Skill.wz", skill.NewProcessor(p.l, p.ctx, p.db).RegisterSkill)()
		_ = skill.GetSkillStringRegistry().Clear(t)
	} else if name == WorkerPet {
		if err = item.InitStringFlat(p.db)(p.l)(p.ctx)(filepath.Join(path, "String.wz", "Pet.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize pet item string registry.")
			return err
		}
		err = p.RegisterAllData(path, filepath.Join("Item.wz", "Pet"), pet.NewProcessor(p.l, p.ctx, p.db).RegisterPet)()
	} else if name == WorkerConsume {
		if err = item.InitStringFlat(p.db)(p.l)(p.ctx)(filepath.Join(path, "String.wz", "Consume.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize consumable item string registry.")
			return err
		}
		err = p.RegisterAllData(path, filepath.Join("Item.wz", "Consume"), consumable.NewProcessor(p.l, p.ctx, p.db).RegisterConsumable)()
	} else if name == WorkerCash {
		if err = item.InitStringFlat(p.db)(p.l)(p.ctx)(filepath.Join(path, "String.wz", "Cash.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize cash item string registry.")
			return err
		}
		err = p.RegisterAllData(path, filepath.Join("Item.wz", "Cash"), cash.NewProcessor(p.l, p.ctx, p.db).RegisterCash)()
	} else if name == WorkerCommodity {
		err = p.RegisterFileData(path, filepath.Join("Etc.wz", "Commodity.img.xml"), commodity.NewProcessor(p.l, p.ctx, p.db).RegisterCommodity)()
	} else if name == WorkerEtc {
		if err = item.InitStringFlat(p.db)(p.l)(p.ctx)(filepath.Join(path, "String.wz", "Etc.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize etc item string registry.")
			return err
		}
		err = p.RegisterAllData(path, filepath.Join("Item.wz", "Etc"), etc.NewProcessor(p.l, p.ctx, p.db).RegisterEtc)()
	} else if name == WorkerSetup {
		if err = item.InitStringFlat(p.db)(p.l)(p.ctx)(filepath.Join(path, "String.wz", "Ins.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize setup item string registry.")
			return err
		}
		err = p.RegisterAllData(path, filepath.Join("Item.wz", "Install"), setup.NewProcessor(p.l, p.ctx, p.db).RegisterSetup)()
	} else if name == WorkerCharacterCreation {
		err = p.RegisterFileData(path, filepath.Join("Etc.wz", "MakeCharInfo.img.xml"), templates.NewProcessor(p.l, p.ctx, p.db).RegisterCharacterTemplate)()
	} else if name == WorkerQuest {
		err = quest.NewProcessor(p.l, p.ctx, p.db).RegisterQuest(filepath.Join(path, "Quest.wz"))
	} else if name == WorkerNPC {
		if err = npc.InitString(t, filepath.Join(path, "String.wz", "Npc.img.xml")); err != nil {
			p.l.WithError(err).Errorf("Failed to initialize NPC string registry for NPC worker.")
			return err
		}
		err = p.RegisterAllData(path, "Npc.wz", npc.NewProcessor(p.l, p.ctx, p.db).RegisterNpc)()
		// Note: Don't clear NPC registry - WorkerMap may run concurrently and needs it
	} else if name == WorkerFace {
		err = p.RegisterAllData(path, filepath.Join("Character.wz", "Face"), face.NewProcessor(p.l, p.ctx, p.db).RegisterFace)()
	} else if name == WorkerHair {
		err = p.RegisterAllData(path, filepath.Join("Character.wz", "Hair"), hair.NewProcessor(p.l, p.ctx, p.db).RegisterHair)()
	} else if name == WorkerMobSkill {
		if err = mobskill.InitString(t, filepath.Join(path, "String.wz", "MobSkill.img.xml")); err != nil {
			p.l.WithError(err).Warnf("Failed to initialize mob skill string registry; names will be empty.")
		}
		err = p.RegisterFileData(path, filepath.Join("Skill.wz", "MobSkill.img.xml"), mobskill.NewProcessor(p.l, p.ctx, p.db).RegisterMobSkill)()
		_ = mobskill.GetMobSkillStringRegistry().Clear(t)
	}
	if err != nil {
		p.l.WithError(err).Errorf("Worker [%s] failed with error.", name)
		return err
	}
	p.l.Infof("Worker [%s] completed.", name)
	emitDataUpdated(p.l, p.ctx, t, name)
	return nil
}

type Worker func() error
type RegisterFunc func(filePath string) error

func (p *ProcessorImpl) RegisterAllData(rootDir string, wzFileName string, rf RegisterFunc) Worker {
	return func() error {
		baseDir := filepath.Join(rootDir, wzFileName)
		if _, err := os.Stat(baseDir); os.IsNotExist(err) {
			p.l.Debugf("Unable to locate directory. Expected [%s]", baseDir)
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
					if err := rf(filePath); err != nil {
						errChan <- fmt.Errorf("error processing %s: %w", filePath, err)
					}
				}
				wg.Done()
			}()
		}

		// Start error collector
		var collectWg sync.WaitGroup
		var collected []error
		collectWg.Add(1)
		go func() {
			defer collectWg.Done()
			for err := range errChan {
				collected = append(collected, err)
			}
		}()

		// Walk directory and send files
		walkErr := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("error accessing path %s: %w", path, err)
			}

			if d.IsDir() {
				return nil
			}

			fileChan <- path
			return nil
		})

		// No more files to enqueue; let workers drain and exit.
		close(fileChan)
		wg.Wait()

		// All producers are done: close the error channel so the collector
		// finishes, then it is safe to read the collected slice.
		close(errChan)
		collectWg.Wait()

		if walkErr != nil {
			collected = append(collected, fmt.Errorf("error walking directory %s: %w", baseDir, walkErr))
		}
		if len(collected) > 0 {
			err := errors.Join(collected...)
			p.l.WithError(err).Errorf("Registration under [%s] completed with %d error(s).", baseDir, len(collected))
			return err
		}
		return nil

	}
}

func (p *ProcessorImpl) RegisterFileData(rootDir string, wzFileName string, rf RegisterFunc) Worker {
	return func() error {
		rf(filepath.Join(rootDir, wzFileName))
		return nil
	}
}

func emitDataUpdated(l logrus.FieldLogger, ctx context.Context, t tenant.Model, worker string) {
	if !producerEnabled() {
		return
	}
	err := producer.ProviderImpl(l)(ctx)(EnvEventTopic)(
		dataUpdatedEventProvider(t.Id().String(), worker, time.Now()),
	)
	if err != nil {
		l.WithError(err).Warnf("Failed to emit DATA_UPDATED for tenant [%s] worker [%s]; cache invalidation will rely on TTL fallback.", t.Id(), worker)
		eventsEmitFailuresTotal.WithLabelValues(worker, EventTypeDataUpdated).Inc()
		return
	}
	eventsEmittedTotal.WithLabelValues(worker, EventTypeDataUpdated).Inc()
}

func producerEnabled() bool {
	v, ok := os.LookupEnv("DATA_EVENTS_PRODUCER_ENABLED")
	if !ok {
		return true
	}
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return enabled
}
