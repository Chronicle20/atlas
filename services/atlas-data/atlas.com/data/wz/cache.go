package wz

import (
	"atlas-data/tenant"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

type fileCache struct {
	files map[string]map[string]FileEntry
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

		var files = make(map[string]map[string]FileEntry)
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

func walkData(dw DataWalker, wg *sync.WaitGroup, mu *sync.Mutex, files map[string]map[string]FileEntry) {
	defer wg.Done()

	fe, err := dw()
	if err != nil {
		return
	}

	if fe == nil {
		return
	}

	var id string
	var results = make(map[string]FileEntry)
	for _, d := range fe {
		id = d.id
		results[d.name] = d
	}

	mu.Lock()
	files[id] = results
	mu.Unlock()
}

func (e *fileCache) GetFile(tenant tenant.Model, name string) (*FileEntry, error) {
	if fc, ok := e.files[tenant.Id().String()]; ok {
		if val, ok := fc[name]; ok {
			return &val, nil
		}
	}

	id := tenant.Region() + "-" + strconv.Itoa(int(tenant.MajorVersion())) + "." + strconv.Itoa(int(tenant.MinorVersion()))
	if fc, ok := e.files[id]; ok {
		if val, ok := fc[name]; ok {
			return &val, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("file %s not found", name))
}
