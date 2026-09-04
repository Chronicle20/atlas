package backeffect

func (r *Registry) Set(key FieldKey, entry BackEffectEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	entries := r.entries[key]
	for i, e := range entries {
		if e.PageId == entry.PageId {
			entries[i] = entry
			r.entries[key] = entries
			return
		}
	}
	r.entries[key] = append(entries, entry)
}

func (r *Registry) Clear(key FieldKey) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.entries[key]
	if !ok {
		return false
	}
	delete(r.entries, key)
	return true
}
