package wz

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"os"
)

func Read(path string) ([]DataWalker, error) {
	return read(path)
}

func read(path string) ([]DataWalker, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return make([]DataWalker, 0), nil
	}

	fs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fes []DataWalker
	for _, cf := range fs {
		var tenantId uuid.UUID
		tenantId, err = uuid.Parse(cf.Name())
		if err == nil {
			// tenant override
			nfes, err := readTenantSpecific(tenantId, path+"/"+cf.Name())
			if err == nil {
				fes = append(fes, nfes...)
			}
			continue
		}

		// assume standard region - version
		region := cf.Name()
		nfes, err := readRegion(region, path+"/"+cf.Name())
		if err == nil {
			fes = append(fes, nfes...)
		}
	}
	return fes, nil
}

func readRegion(region string, path string) ([]DataWalker, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return nil, err
	}

	fs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fes []DataWalker
	for _, cf := range fs {
		ces, err := readVersionFiles(region, cf.Name(), path+"/"+cf.Name())
		if err != nil {
			return nil, err
		}
		fes = append(fes, ces...)
	}
	return fes, nil
}

func readVersionFiles(region string, name string, path string) ([]DataWalker, error) {
	_, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	return readFolders(RegionFileEntryDecorator(region, name))(path)
}

type DataWalker model.Provider[[]FileEntry]

func TenantFileEntryDecorator(id uuid.UUID) model.Decorator[FileEntry] {
	return func(entry FileEntry) FileEntry {
		return entry.SetId(id.String())
	}
}

func RegionFileEntryDecorator(region string, version string) model.Decorator[FileEntry] {
	return func(entry FileEntry) FileEntry {
		return entry.SetId(region + "-" + version)
	}
}

func FolderFileEntryDecorator(folder string) model.Decorator[FileEntry] {
	return func(entry FileEntry) FileEntry {
		return entry.SetFolder(folder)
	}
}

func readTenantSpecific(id uuid.UUID, path string) ([]DataWalker, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return nil, err
	}

	_, err = os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	return readFolders(TenantFileEntryDecorator(id))(path)
}

func readFolders(decorator model.Decorator[FileEntry]) func(path string) ([]DataWalker, error) {
	return func(path string) ([]DataWalker, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return nil, err
		}

		if stat.IsDir() {
			fs, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}

			var fes []DataWalker
			for _, cf := range fs {
				ces := readFiles([]model.Decorator[FileEntry]{decorator, FolderFileEntryDecorator(cf.Name())}...)(path + "/" + cf.Name())
				if err != nil {
					return nil, err
				}
				fes = append(fes, ces)
			}
			return fes, nil
		}
		return nil, nil
	}
}

func readFiles(decorator ...model.Decorator[FileEntry]) func(path string) DataWalker {
	return func(path string) DataWalker {
		return func() ([]FileEntry, error) {
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			stat, err := f.Stat()
			if err != nil {
				return nil, err
			}

			if stat.IsDir() {
				fs, err := os.ReadDir(path)
				if err != nil {
					return nil, err
				}

				var fes []FileEntry
				for _, cf := range fs {
					ces, err := readFiles(decorator...)(path + "/" + cf.Name())()
					if err != nil {
						return nil, err
					}
					fes = append(fes, ces...)
				}
				return fes, nil
			} else {
				fe := NewFileEntry(stat.Name(), f.Name())
				for _, d := range decorator {
					fe = d(fe)
				}
				return []FileEntry{fe}, nil
			}
		}
	}
}
