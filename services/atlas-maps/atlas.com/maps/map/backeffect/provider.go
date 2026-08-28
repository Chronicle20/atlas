package backeffect

func (r *Registry) Get(key FieldKey) []BackEffectEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	entries := r.entries[key]
	result := make([]BackEffectEntry, len(entries))
	copy(result, entries)
	return result
}
