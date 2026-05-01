package characterimage

import "sync"

// metaCache memoizes per-templateId TemplateInfo across renders within one
// process. Entries are never evicted (Character.wz is small enough; bounded
// by extraction wipe). Thread-safe via sync.Map.
type metaCache struct {
	infos sync.Map // templateId -> TemplateInfo
}

func newMetaCache() *metaCache { return &metaCache{} }

func (c *metaCache) info(assetsRoot, templateId string) (TemplateInfo, error) {
	if v, ok := c.infos.Load(templateId); ok {
		return v.(TemplateInfo), nil
	}
	ti, err := LoadInfo(assetsRoot, templateId)
	if err != nil {
		return TemplateInfo{}, err
	}
	c.infos.Store(templateId, ti)
	return ti, nil
}
