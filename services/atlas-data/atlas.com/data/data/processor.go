package data

import (
	"archive/zip"
	"atlas-data/consumable"
	"atlas-data/equipment"
	"atlas-data/kafka/producer"
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
	"gorm.io/gorm"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
)

const (
	WorkerMap       = "MAP"
	WorkerMonster   = "MONSTER"
	WorkerCharacter = "CHARACTER"
	WorkerReactor   = "REACTOR"
	WorkerSkill     = "SKILL"
	WorkerPet       = "PET"
	WorkerConsume   = "CONSUME"
)

var Workers = []string{WorkerMap, WorkerMonster, WorkerCharacter, WorkerReactor, WorkerSkill, WorkerPet, WorkerConsume}

func ProcessZip(l logrus.FieldLogger) func(ctx context.Context) func(file multipart.File, handler *multipart.FileHeader) error {
	return func(ctx context.Context) func(file multipart.File, handler *multipart.FileHeader) error {
		t := tenant.MustFromContext(ctx)
		return func(file multipart.File, handler *multipart.FileHeader) error {
			uploadDir := os.Getenv("ZIP_DIR")

			// Save ZIP file to disk
			tenantDir := filepath.Join(uploadDir, t.Id().String(), t.Region())
			zipPath := filepath.Join(tenantDir, handler.Filename)

			if err := os.MkdirAll(tenantDir, os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.Create(zipPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			// Stream file contents to disk
			_, err = io.Copy(outFile, file)
			if err != nil {
				return err
			}

			err = unzip(zipPath, tenantDir)
			if err != nil {
				l.WithError(err).Errorf("Unable to unzip [%s].", zipPath)
				return err
			}

			l.Debugf("Unzipped to [%s].", zipPath)

			for _, wn := range Workers {
				err = InstructWorker(l)(ctx)(wn, filepath.Join(tenantDir, fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())))
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
}

func unzip(zipPath, dest string) error {
	// Open the ZIP file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}

	// Extract each file
	for _, file := range r.File {
		filePath := filepath.Join(dest, file.Name)

		// Ensure parent directories exist
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		} else {
			os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		}

		// Extract file contents
		destFile, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		srcFile, err := file.Open()
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			return err
		}
	}
	return nil
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
		return func(db *gorm.DB) func(name string, path string) error {
			return func(name string, path string) error {
				l.Debugf("Starting worker [%s] at [%s].", name, path)
				return nil
			}
		}
	}
}

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

		registers := make([]Worker, 0)
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
