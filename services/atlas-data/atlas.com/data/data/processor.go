package data

import (
	"archive/zip"
	"atlas-data/cash"
	"atlas-data/commodity"
	"atlas-data/consumable"
	"atlas-data/equipment"
	"atlas-data/etc"
	"atlas-data/kafka/producer"
	_map "atlas-data/map"
	"atlas-data/monster"
	"atlas-data/npc"
	"atlas-data/pet"
	"atlas-data/reactor"
	"atlas-data/setup"
	"atlas-data/skill"
	"context"
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
	WorkerCash      = "CASH"
	WorkerCommodity = "COMMODITY"
	WorkerEtc       = "ETC"
	WorkerSetup     = "SETUP"
)

var Workers = []string{WorkerMap, WorkerMonster, WorkerCharacter, WorkerReactor, WorkerSkill, WorkerPet, WorkerConsume, WorkerCash, WorkerCommodity, WorkerEtc, WorkerSetup}

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
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(name string, path string) error {
			return func(name string, path string) error {
				l.Infof("Starting worker [%s] at [%s].", name, path)
				if name == WorkerMap {
					_ = _map.InitString(t, filepath.Join(path, "String.wz", "Map.img.xml"))
					_ = npc.InitString(t, filepath.Join(path, "String.wz", "Npc.img.xml"))
					_ = RegisterAllData(l)(ctx)(path, filepath.Join("Map.wz", "Map"), _map.RegisterMap(db))()
					_ = _map.GetMapStringRegistry().Clear(t)
					_ = npc.GetNpcStringRegistry().Clear(t)
				} else if name == WorkerMonster {
					_ = monster.InitString(t, filepath.Join(path, "String.wz", "Mob.img.xml"))
					_ = monster.InitGauge(t, filepath.Join(path, "UI.wz", "UIWindow.img.xml"))
					_ = RegisterAllData(l)(ctx)(path, "Mob.wz", monster.RegisterMonster(db))()
					_ = monster.GetMonsterStringRegistry().Clear(t)
					_ = monster.GetMonsterGaugeRegistry().Clear(t)
				} else if name == WorkerCharacter {
					_ = RegisterAllData(l)(ctx)(path, "Character.wz", equipment.RegisterEquipment(db))()
				} else if name == WorkerReactor {
					_ = RegisterAllData(l)(ctx)(path, "Reactor.wz", reactor.RegisterReactor(db))()
				} else if name == WorkerSkill {
					_ = RegisterAllData(l)(ctx)(path, "Skill.wz", skill.RegisterSkill(db))()
				} else if name == WorkerPet {
					_ = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Pet"), pet.RegisterPet(db))()
				} else if name == WorkerConsume {
					_ = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Consume"), consumable.RegisterConsumable(db))()
				} else if name == WorkerCash {
					_ = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Cash"), cash.RegisterCash(db))()
				} else if name == WorkerCommodity {
					_ = RegisterFileData(l)(ctx)(path, filepath.Join("Etc.wz", "Commodity.img.xml"), commodity.RegisterCommodity(db))()
				} else if name == WorkerEtc {
					_ = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Etc"), etc.RegisterEtc(db))()
				} else if name == WorkerSetup {
					_ = RegisterAllData(l)(ctx)(path, filepath.Join("Item.wz", "Install"), setup.RegisterSetup(db))()
				}

				return nil
			}
		}
	}
}

type Worker func() error
type RegisterFunc func(l logrus.FieldLogger) func(ctx context.Context) func(filePath string)

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
