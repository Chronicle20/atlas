package wz

import (
	"atlas-data/tenant"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

type fileCache struct {
	files map[string]map[string]map[string]FileEntry
}

var cache *fileCache
var once sync.Once

func GetFileCache() *fileCache {
	return cache
}

func (e *fileCache) Init(wzPath string) {
	once.Do(func() {
		dw, err := Read(wzPath)
		if err != nil {
			panic(err)
		}

		var files = make(map[string]map[string]map[string]FileEntry)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, d := range dw {
			wg.Add(1)
			go walkData(d, &wg, &mu, files)
		}

		wg.Wait()

		cache = &fileCache{files: files}
	})
}

func walkData(dw DataWalker, wg *sync.WaitGroup, mu *sync.Mutex, files map[string]map[string]map[string]FileEntry) {
	defer wg.Done()

	fe, err := dw()
	if err != nil {
		return
	}

	if fe == nil {
		return
	}

	var id string
	var folder string
	var results = make(map[string]FileEntry)
	for _, d := range fe {
		id = d.id
		folder = d.folder
		results[d.name] = d
	}

	mu.Lock()
	if _, ok := files[id]; !ok {
		files[id] = make(map[string]map[string]FileEntry)
	}

	if _, ok := files[id][folder]; !ok {
		files[id][folder] = make(map[string]FileEntry)
	}

	files[id][folder] = results
	mu.Unlock()
}

func (e *fileCache) GetFile(tenant tenant.Model, folder string, name string) (*FileEntry, error) {
	if tfc, ok := e.files[tenant.Id().String()]; ok {
		if ffc, ok := tfc[folder]; ok {
			if val, ok := ffc[name]; ok {
				return &val, nil
			}
		}
	}

	id := tenant.Region() + "-" + strconv.Itoa(int(tenant.MajorVersion())) + "." + strconv.Itoa(int(tenant.MinorVersion()))
	if fc, ok := e.files[id]; ok {
		if ffc, ok := fc[folder]; ok {
			if val, ok := ffc[name]; ok {
				return &val, nil
			}
		}
	}

	return nil, errors.New(fmt.Sprintf("file %s not found", name))
}
