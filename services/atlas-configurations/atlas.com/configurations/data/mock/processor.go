package mock

import (
	"atlas-configurations/data"
	"context"
)

// FakeClient is an in-memory implementation of data.Client for use in unit tests.
type FakeClient struct {
	// Skills is a map of skill ID → SkillInfo returned by GetSkillsByIds.
	Skills map[uint32]data.SkillInfo
	// Items is a map of item ID → ItemInfo returned by GetItemById.
	Items map[uint32]data.ItemInfo
	// SkillsErr, if non-nil, is returned by every GetSkillsByIds call.
	SkillsErr error
	// ItemErrFor maps item IDs to errors returned by GetItemById for that ID.
	// Use data.ErrNotFound to simulate a missing template.
	ItemErrFor map[uint32]error
}

// GetSkillsByIds returns SkillInfo entries for each requested ID that exists in
// f.Skills. IDs not present in the map are silently skipped (mirrors real behaviour
// where atlas-data omits unknown IDs from the batch response).
func (f *FakeClient) GetSkillsByIds(_ context.Context, ids []uint32) ([]data.SkillInfo, error) {
	if f.SkillsErr != nil {
		return nil, f.SkillsErr
	}
	out := make([]data.SkillInfo, 0, len(ids))
	for _, id := range ids {
		if sk, ok := f.Skills[id]; ok {
			out = append(out, sk)
		}
	}
	return out, nil
}

// GetItemById looks up the item in f.Items. If f.ItemErrFor contains a non-nil
// error for the given ID, that error is returned. If the ID is absent from both
// maps, data.ErrNotFound is returned.
func (f *FakeClient) GetItemById(_ context.Context, id uint32) (data.ItemInfo, error) {
	if err, ok := f.ItemErrFor[id]; ok && err != nil {
		return data.ItemInfo{}, err
	}
	if it, ok := f.Items[id]; ok {
		return it, nil
	}
	return data.ItemInfo{}, data.ErrNotFound
}
