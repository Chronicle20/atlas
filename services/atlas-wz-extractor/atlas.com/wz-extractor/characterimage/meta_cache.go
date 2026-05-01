package characterimage

import "sync"

// metaCacheKey is the composite key used to isolate cached TemplateInfo by
// both assetsRoot (tenant) and templateId.
type metaCacheKey struct {
	AssetsRoot string
	TemplateId string
}

// metaCache memoizes per-(assetsRoot, templateId) TemplateInfo across renders
// within one process. Entries are never evicted (Character.wz is small enough;
// bounded by extraction wipe). Thread-safe via sync.Map.
type metaCache struct {
	infos sync.Map // metaCacheKey -> TemplateInfo
}

func newMetaCache() *metaCache { return &metaCache{} }

func (c *metaCache) info(assetsRoot, templateId string) (TemplateInfo, error) {
	key := metaCacheKey{AssetsRoot: assetsRoot, TemplateId: templateId}
	if v, ok := c.infos.Load(key); ok {
		return v.(TemplateInfo), nil
	}
	ti, err := LoadInfo(assetsRoot, templateId)
	if err != nil {
		return TemplateInfo{}, err
	}
	c.infos.Store(key, ti)
	return ti, nil
}
