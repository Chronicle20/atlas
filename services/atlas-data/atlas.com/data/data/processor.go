package data

import (
	"atlas-data/consumable"
	"atlas-data/equipment"
	_map "atlas-data/map"
	"atlas-data/monster"
	"atlas-data/npc"
	"atlas-data/pet"
	"atlas-data/reactor"
	"atlas-data/skill"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type Worker func() error

func RegisterData(l logrus.FieldLogger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		t := tenant.MustFromContext(ctx)
		l.Debugf("Attempting to ingest data for tenant.")

		dir, exists := os.LookupEnv("GAME_DATA_ROOT_DIR")
		if !exists {
			l.Errorf("Unable to retrieve [GAME_DATA_ROOT_DIR] configuration necessary to ingest data.")
			return errors.New("env not found")
		}

		stringWzMapPath := filepath.Join(dir, t.Id().String(), t.Region(), fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()), "String.wz", "Map.img.xml")
		_ = _map.GetMapStringRegistry().Init(t, stringWzMapPath)
		stringWzMapPath = filepath.Join(dir, t.Id().String(), t.Region(), fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()), "String.wz", "Npc.img.xml")
		_ = npc.GetNpcStringRegistry().Init(t, stringWzMapPath)
		stringWzMapPath = filepath.Join(dir, t.Id().String(), t.Region(), fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()), "String.wz", "Mob.img.xml")
		_ = monster.GetMonsterStringRegistry().Init(t, stringWzMapPath)
		uiWzMapPath := filepath.Join(dir, t.Id().String(), t.Region(), fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()), "UI.wz", "UIWindow.img.xml")
		_ = monster.GetMonsterGaugeRegistry().Init(t, uiWzMapPath)

		registers := make([]func() error, 0)
		registers = append(registers, RegisterAllData(l)(ctx)(dir, filepath.Join("Map.wz", "Map"), true, _map.RegisterMap))
		registers = append(registers, RegisterAllData(l)(ctx)(dir, "Mob.wz", false, monster.RegisterMonster))
		registers = append(registers, RegisterAllData(l)(ctx)(dir, "Character.wz", true, equipment.RegisterEquipment))
		registers = append(registers, RegisterAllData(l)(ctx)(dir, "Reactor.wz", true, reactor.RegisterReactor))
		registers = append(registers, RegisterAllData(l)(ctx)(dir, "Skill.wz", false, skill.RegisterSkill))
		registers = append(registers, RegisterAllData(l)(ctx)(dir, filepath.Join("Item.wz", "Pet"), false, pet.RegisterPet))
		registers = append(registers, RegisterAllData(l)(ctx)(dir, filepath.Join("Item.wz", "Consume"), false, consumable.RegisterConsumable))

		var wg sync.WaitGroup
		for _, register := range registers {
			wg.Add(1)
			go func() {
				_ = register()
				wg.Done()
			}()
		}
		wg.Wait()

		_ = _map.GetMapStringRegistry().Clear(t)
		_ = npc.GetNpcStringRegistry().Clear(t)
		_ = monster.GetMonsterStringRegistry().Clear(t)
		_ = monster.GetMonsterGaugeRegistry().Clear(t)

		return nil
	}
}

type RegisterFunc func(l logrus.FieldLogger) func(ctx context.Context) func(filePath string)

func RegisterAllData(l logrus.FieldLogger) func(ctx context.Context) func(rootDir string, wzFileName string, nested bool, rf RegisterFunc) Worker {
	return func(ctx context.Context) func(rootDir string, wzFileName string, nested bool, rf RegisterFunc) Worker {
		t := tenant.MustFromContext(ctx)
		return func(rootDir string, wzFileName string, nested bool, rf RegisterFunc) Worker {
			return func() error {
				baseDir := filepath.Join(rootDir, t.Id().String(), t.Region(), fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()), wzFileName)
				if _, err := os.Stat(baseDir); os.IsNotExist(err) {
					l.Debugf("Unable to locate directory. Expected [%s]", baseDir)
					return err
				}

				// Channel to collect file paths
				fileChan := make(chan string)
				var wg sync.WaitGroup

				// Start a worker pool for processing files
				const workerCount = 10 // Adjust based on your workload and system resources
				for i := 0; i < workerCount; i++ {
					wg.Add(1)
					go func() {
						for filePath := range fileChan {
							rf(l)(ctx)(filePath)
						}
						wg.Done()
					}()
				}

				err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return fmt.Errorf("error accessing path %s: %w", path, err)
					}

					if nested {
						if d.IsDir() {
							return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
								if err != nil {
									return fmt.Errorf("error accessing file %s: %w", path, err)
								}

								if d.IsDir() {
									return nil
								}

								fileChan <- path
								return nil
							})
						}
						return nil
					} else {
						if d.IsDir() {
							return nil
						}

						fileChan <- path
						return nil
					}
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
